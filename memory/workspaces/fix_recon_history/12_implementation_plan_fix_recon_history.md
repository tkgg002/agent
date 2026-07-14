# Kế hoạch triển khai: Sửa lỗi 500 Endpoint Lịch sử Đối soát (schedule_histories)

## 1. Phát hiện lỗi & Phân tích (Root Cause Analysis)
Khi gọi API `GET /api/reconciliation/report/schedule_histories`, repository chạy hàm `GetTableHistory` thực hiện query SELECT các cột `healed_mismatched_at`, `healed_missing_src_at`, `healed_missing_dest_at` từ bảng `cdc_system.cdc_reconciliation_report`.
Tuy nhiên, trong database table này chưa được ALTER để thêm 3 cột này (nhánh `recon-heal` mới chỉ bổ sung fields vào Go model và query code mà chưa tạo file migration). Do đó Postgres ném lỗi:
`ERROR: column "healed_mismatched_at" does not exist (SQLSTATE 42703)`

## 2. Giải pháp đề xuất
1. **Thêm file migration mới**: Tạo file `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/093_recon_heal_timestamps.sql`.
   Nội dung file sẽ thực hiện `ALTER TABLE cdc_system.cdc_reconciliation_report` để thêm 3 cột:
   - `healed_mismatched_at` (TIMESTAMP)
   - `healed_missing_src_at` (TIMESTAMP)
   - `healed_missing_dest_at` (TIMESTAMP)
2. **Khởi động lại service**: Khi cdc-cms-service khởi động, migration runner của nó (`internal/migrate/runner.go`) sẽ tự động phát hiện file SQL mới và thực thi migration vào database local.

## 3. Kế hoạch xác minh (Verification Plan)
- **Integration Test**: Chạy lại integration test `TestGetTableHistory_RealDB` trong `recon_read_repo_gorm_real_test.go` để xác nhận query kết nối DB thành công và trả về dữ liệu mà không bị lỗi SQL.
- **Xóa file test sau khi chạy xong**: Để giữ repository sạch sẽ, file `recon_read_repo_gorm_real_test.go` sẽ được xóa sau khi xác minh xong (hoặc git checkout revert lại file test đó).
