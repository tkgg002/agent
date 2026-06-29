# Context: Runtime Check for DB and API Gateway

## Goal
Thực hiện kiểm tra runtime đối với cơ sở dữ liệu và API Gateway của `cdc-cms-service`:
1. Tìm cấu hình HTTP port của API Gateway trong `cdc-cms-service`.
2. Đếm số lượng bản ghi của bảng `cdc_system.cdc_recon_smoke_result` trực tiếp trong DB (qua python/go script hoặc CLI).
3. Nếu bảng trống, trigger chạy thử đối soát khói O(1) CheckAllUnified bằng cách gửi `POST /api/reconciliation/check` hoặc NATS command.
4. Chạy `GET http://localhost:<port>/api/reconciliation/report` để kiểm tra kết quả thực tế.

## Key Metadata
- **Workspace**: [feat-recon-runtime-check-2026-06-29](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feat-recon-runtime-check-2026-06-29)
- **Service**: cdc-cms-service
- **DB Table**: `cdc_system.cdc_recon_smoke_result`
- **Target Port**: Cần xác định từ config
