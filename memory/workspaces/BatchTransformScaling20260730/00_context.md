# 00_context.md: BatchTransformScaling20260730

## Scope
Nâng cấp `BatchTransformHandler` để xử lý 50M - 500M records mà không timeout, với progress tracking và Pause/Cancel từ UI.

## Components bị ảnh hưởng
- **Worker BE**: `centralized-data-service/internal/handler/shadow/batch_transform_handler.go`
- **CMS BE**: `cdc-cms-service` (endpoint trigger + job status API)
- **CMS FE**: `cdc-cms-web/src/pages/TableRegistry.tsx` (Progress Bar UI)
- **DB**: `cdc_system.recon_jobs` (job tracking table - cần kiểm tra xem đã có chưa)

## Trạng thái hiện tại
- Code sync, blocking, maxIterations=100000 (thiếu cho 500M), chunk=1000 cứng
- Không có job tracking, không có progress, không có Pause/Cancel
