# Implementation Plan: Fix Critical Recon Issues

Kế hoạch này mô tả các bước để khắc phục 3 lỗi Critical P0 trong module Recon nhằm nâng cao tính bảo mật, cấu hình và chuẩn hóa code.

## Proposed Changes

### Component: Handler Recon

#### [MODIFY] [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
- Sửa SQL Injection tại hàm `mapGpayToSourceIDs` bằng cách bọc qualified table reference thông qua helper `quoteRelation`.
- Sửa context key string `"manual_lookback"` và `"cold_lookback"` thành accessors an toàn `servicerecon.WithManualLookback` và `servicerecon.WithColdLookback`.

#### [MODIFY] [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
- Thay đổi `context.WithValue` dùng string key sang dùng accessors từ `servicerecon`.

#### [MODIFY] [recon_check_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_heal_handler.go)
- Thay đổi `context.WithValue` dùng string key sang dùng accessors từ `servicerecon`.

#### [MODIFY] [recon_base_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_base_handler.go)
- Thêm helper function `quoteRelation`.
- Import `"centralized-data-service/internal/naming"` và cập nhật hằng số/biến `ShadowPrefix` dùng `naming.ShadowSchemaPrefix()`.

### Component: Service Recon

#### [MODIFY] [recon_models.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_models.go)
- Khai báo unexported struct key types và các hàm accessors:
  - `WithManualLookback` / `GetManualLookback`
  - `WithColdLookback` / `GetColdLookback`

#### [MODIFY] [recon_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go)
- Cập nhật hàm `effectiveLookback` dùng `GetColdLookback(ctx)`.

#### [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
- Cập nhật đọc context dùng `GetManualLookback(ctx)`.

---

## Verification Plan

### Automated Tests
- Chạy unit tests cho module handler và service:
  ```bash
  go test ./internal/handler/recon/...
  go test ./internal/service/recon/...
  ```
- Chạy Process Linter:
  ```bash
  python3 agent/tooling/verify_governance.py --workspace ReconCodebaseAudit20260707
  ```
