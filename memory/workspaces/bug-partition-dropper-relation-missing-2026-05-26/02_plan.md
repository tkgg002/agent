# Kế hoạch Giải quyết (High-level Plan)

1. **Khảo sát mã nguồn (Research)**:
   - Đọc tệp `internal/service/partition_dropper.go` xung quanh dòng 265 để hiểu logic đang thực hiện truy vấn SELECT trên bảng mặc định (`_default`).
   - Xác định xem các bảng này có phải là phân vùng mặc định của cơ chế PostgreSQL partitioning đối với `failed_sync_logs` và `cdc_activity_log` hay không.
2. **Xác định Giải pháp**:
   - Nếu các bảng mặc định (`failed_sync_logs_default`, `cdc_activity_log_default`) bắt buộc phải tồn tại trong thiết kế partitioning của Postgres nhưng chưa được tạo qua migration -> Cần tạo file migration bổ sung hoặc khởi tạo chúng.
   - Nếu đây là logic kiểm tra động của `partition_dropper` trên các phân vùng (partiton) hiện có -> Cập nhật truy vấn để chỉ chạy SELECT trên những phân vùng thực sự tồn tại trong hệ thống (kiểm tra metadata của Postgres bằng cách query `pg_class` hoặc `information_schema.tables`), tránh query mù dẫn tới lỗi `relation does not exist`.
3. **Thực thi sửa đổi (Execution)**:
   - Sửa đổi mã nguồn hoặc file cấu hình cần thiết.
   - Biên dịch và chạy thử nghiệm.
4. **Kiểm thử & Báo cáo (Verification & Reporting)**:
   - Chạy các unit test liên quan.
   - Tạo file báo cáo kết quả.
