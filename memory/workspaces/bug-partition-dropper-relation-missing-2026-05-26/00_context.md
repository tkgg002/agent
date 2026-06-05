# Bối cảnh & Phạm vi (Scope & Context)

- **Vấn đề**: Hệ thống CDC Worker ghi nhận lỗi `ERROR: relation "failed_sync_logs_default" does not exist (SQLSTATE 42P01)` và `ERROR: relation "cdc_activity_log_default" does not exist` tại tệp `partition_dropper.go:265` khi cố gắng truy vấn dữ liệu từ các phân vùng mặc định (`_default`).
- **Phạm vi xử lý**:
  1. Xác định lý do tại sao `partition_dropper` cố gắng truy vấn từ `failed_sync_logs_default` và `cdc_activity_log_default`.
  2. Kiểm tra xem các bảng mặc định (`_default`) này có thực sự cần tồn tại hay không theo cơ chế phân vùng (partitioning) của hệ thống.
  3. Đưa ra giải pháp sửa đổi mã nguồn hoặc migration để tạo các bảng mặc định nếu thiếu, hoặc bắt lỗi `SQLSTATE 42P01` (giống như đã làm với trùng khóa/deadlock) hoặc sửa logic truy vấn kiểm tra sự tồn tại của phân vùng trước khi thực hiện SELECT.
