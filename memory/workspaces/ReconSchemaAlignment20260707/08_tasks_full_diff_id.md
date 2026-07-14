# Danh sách Task - Khắc phục Hiển thị Dữ liệu ID Diff

- [x] Cập nhật SELECT list trong hàm `GetTableHistory` tại `recon_read_repo_gorm.go` để lấy đầy đủ các cột ID diff và các trường heal
- [x] Chạy build dự án đảm bảo compile thành công
- [x] Chạy unit test suites của queries
- [x] Xác minh kết quả trả về của API `/api/reconciliation/report/export_jobs` chứa trường dữ liệu diff mong muốn
