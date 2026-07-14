# Walkthrough - Tăng cường Tracing & Sửa lỗi Reconciliation History & Đồng bộ Mapping Rules

Chúng tôi đã hoàn thành triển khai các cải tiến về Observability, sửa lỗi schema database cho tiến trình đối soát và khắc phục triệt để lỗi đồng bộ mapping rules khi phê duyệt Master Table.

## Thay đổi đã thực hiện

### 1. Database Schema & API Reconciliation
- Cột `healed_mismatched_at`, `healed_missing_src_at`, và `healed_missing_dest_at` đã tồn tại trong cấu trúc database hiện tại, giải quyết triệt để lỗi `500 Internal Server Error` trên endpoint `/api/reconciliation/report/schedule_histories`.

### 2. Observability & Tracing
- **`pkgs/observability/trace_helpers.go`**: Bổ sung helper `ContextWithoutSkipTrace(ctx)` giúp khôi phục lại cấu hình trace (bỏ qua cờ bypass trace) trên các window/bucket bị lệch (drifted).
- **`internal/service/recon/recon_tier_a.go`**: Cấu hình lại loop `RunHashWindowCheck` để khi gặp window bị drifted sẽ khôi phục trace chi tiết trước khi drill down.
- **`internal/service/recon/recon_tier_b.go`**: Tương tự, cấu hình lại loop `RunHashWindowCheckB` và `RunDeepCheckB` để khôi phục trace chi tiết khi bucket bị drifted.

### 3. Đồng bộ Trạng thái Rule (Rule Sync Fix)
- **`internal/infra/persistence/master/master_repo_gorm.go`**: Sửa đổi logic clone rules trong cả hai hàm `ApproveSchemaTx` và `CloneMappingRules` để kế thừa status từ `v2.status` thay vì gán cứng `'pending'`.
- **`test/internal/app/commands/approve_master_test.go`**: Sửa lỗi interface mismatch bằng repository wrapper `persisMaster.NewMasterRepo(db)`. Sửa đổi assertion kết quả thành `"approved"` để phù hợp logic.
- **`test/internal/app/commands/approve_schema_proposal_integration_test.go`**: Khắc phục lỗi compile do interface `ports.SchemaProposalRepo` thay đổi và import sai package `commands`.

---

## Kết quả kiểm thử

### 1. Centralized Data Service
Toàn bộ unit test cho package `recon` đã vượt qua thành công:
```bash
go test -v ./internal/service/recon/...
```
Kết quả:
```text
ok  	centralized-data-service/internal/service/recon	0.629s
```

### 2. CDC CMS Service
Các test query lịch sử đối soát và toàn bộ integration tests trong package `commands` đều hoạt động ổn định:
```bash
go test -v ./test/internal/app/queries/ -run TestGetTableHistory
go test -v -tags=integration ./test/internal/app/commands/...
```
Kết quả:
```text
PASS
ok  	cdc-cms-service/test/internal/app/commands	16.172s
```
