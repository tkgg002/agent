# Walkthrough: Sửa lỗi 500 Endpoint Lịch sử Đối soát (schedule_histories)

## 1. Thay đổi đã thực hiện
Chúng ta đã tạo và áp dụng file migration SQL mới:
- `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/093_recon_heal_timestamps.sql`

Nội dung migration thực hiện ALTER TABLE để thêm 3 cột:
- `healed_mismatched_at` (TIMESTAMP)
- `healed_missing_src_at` (TIMESTAMP)
- `healed_missing_dest_at` (TIMESTAMP)

## 2. Kết quả kiểm tra (Validation Results)
- Chạy integration test kết nối trực tiếp database thực tế local:
  ```bash
  CFG_PATH=/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/config/config-local.yml go test -v -run TestGetTableHistory_RealDB ./internal/infra/persistence/recon/...
  ```
- Kết quả test: **PASS** (Đã chạy thành công, database tự động áp dụng file migration 093 mới và câu query `GetTableHistory` thực thi không còn báo lỗi thiếu cột).
- Đã dọn dẹp file test tạm và cập nhật đầy đủ logs.
