# Danh sách Task: Tối ưu hóa SQL cdc_activity_log

- [ ] Tối ưu hóa logic đếm trong `activity_log_read_repo_gorm.go`
  - [ ] Kiểm tra các điều kiện lọc để quyết định xem có cần thực hiện JOIN khi COUNT hay không.
  - [ ] Thực hiện đếm trực tiếp (`SELECT COUNT(*) FROM cdc_activity_log`) nếu các trường lọc của bảng liên kết đều rỗng.
- [ ] Xác minh kết quả & Kiểm thử
  - [ ] Chạy unit/integration test suite của service để đảm bảo tính đúng đắn.
  - [ ] Viết benchmark script đo thời gian thực thi của count query mới.
  - [ ] Tạo walkthrough báo cáo kết quả.
