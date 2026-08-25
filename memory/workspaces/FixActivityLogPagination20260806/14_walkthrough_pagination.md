# Walkthrough: Sửa Lỗi Phân Trang API Activity Log

## Các công việc đã hoàn thành
1. **Phân tích Root Cause**:
   - Viết script `test_db.go` để chạy SQL query trực tiếp trên database. Phát hiện ra join `cdc_system.master_binding` bị nhân bản dòng vì một shadow table có thể map đến nhiều master table (1-N).
2. **Sửa đổi mã nguồn**:
   - File: `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`
   - Đổi join của `master_binding` thành `LEFT JOIN LATERAL` với `LIMIT 1` để đảm bảo không bị nhân bản dòng.
3. **Xác thực kết quả**:
   - Chạy script kiểm thử `test_db.go`. Kết quả:
     - Page 1: Đúng 30 dòng (mục logs mới nhất).
     - Page 2: Đúng 30 dòng (mục logs tiếp theo).
     - Phân trang hoạt động hoàn toàn chính xác.
