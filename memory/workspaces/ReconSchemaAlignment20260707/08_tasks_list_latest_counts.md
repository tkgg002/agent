# Danh sách Task - Sửa lỗi sai lệch Count hiển thị trên Dashboard (ListLatest)

- [x] Cập nhật SQL query `listLatestPrimary` trong `recon_read_repo_gorm.go` để lấy các cột active/total counts từ `cdc_recon_smoke_result` thông qua LEFT JOIN LATERAL.
- [x] Chạy build dự án đảm bảo compile thành công.
- [x] Chạy unit test suites của queries.
- [x] Xác minh kết quả trả về của API `/api/reconciliation/report` (ListLatest) chứa counts từ smoke check thay vì full search window-based.
