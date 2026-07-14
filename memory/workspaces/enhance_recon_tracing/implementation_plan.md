# Kế hoạch triển khai: Sửa lỗi Reconciliation History & Tăng cường Observability cho Reconciliation Pipeline

Kế hoạch này giải quyết hai vấn đề:
1. Khắc phục lỗi `500 Internal Server Error` trên endpoint `/api/reconciliation/report/schedule_histories` do thiếu các cột timestamp chữa lành trong cơ sở dữ liệu.
2. Tăng cường khả năng quan sát (observability) của tiến trình đối soát bằng cách tích hợp OpenTelemetry child spans, sử dụng cơ chế smart tracing để giảm thiểu noise của các window sạch.

## User Review Required

> [!IMPORTANT]
> - Chúng ta cần chạy một migration SQL để thêm các cột `healed_mismatched_at`, `healed_missing_src_at`, và `healed_missing_dest_at` vào bảng `cdc_system.cdc_reconciliation_report`.
> - Việc tích hợp tracing sẽ sử dụng một helper mới `ContextWithoutSkipTrace` để phục hồi trace trên các window bị lệch (drifted), giúp SigNoz ghi lại chi tiết các query database con.

## Proposed Changes

### 1. Database Migration (Sửa lỗi 500)

#### [NEW] [093_recon_heal_timestamps.sql](file:///Users/trainguyen/Documents/work-db/cdc-cms-service/migrations/schema/recon_dlq/093_recon_heal_timestamps.sql)

```sql
BEGIN;

ALTER TABLE cdc_system.cdc_reconciliation_report
  ADD COLUMN IF NOT EXISTS healed_mismatched_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS healed_missing_src_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS healed_missing_dest_at TIMESTAMP;

COMMIT;
```

### 2. Observability & Tracing (Centralized Data Service)

#### [MODIFY] [trace_helpers.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/observability/trace_helpers.go)
- Thêm `ContextWithoutSkipTrace(ctx context.Context) context.Context` để xóa cờ bypass trace khi phát hiện window bị lệch.

#### [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
- Cập nhật loop trong `RunHashWindowCheck` để kích hoạt trace chi tiết bằng `ContextWithoutSkipTrace` cho window bị drifted trước khi gọi `ListIDTsInWindow`.

#### [MODIFY] [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
- Cập nhật loop trong `RunHashWindowCheckB` và `RunDeepCheckB` để kích hoạt trace chi tiết bằng `ContextWithoutSkipTrace` cho bucket bị drifted trước khi gọi `ListIDTsInWindow`.

---

## Verification Plan

### Automated Tests
- Chạy test migration và repository query:
  ```bash
  CFG_PATH=/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/config/config-local.yml go test -v -run TestGetTableHistory_RealDB ./internal/infra/persistence/recon/...
  ```
- Chạy toàn bộ test suites đối soát để đảm bảo không bị regression:
  ```bash
  go test -v ./internal/service/recon/...
  ```
