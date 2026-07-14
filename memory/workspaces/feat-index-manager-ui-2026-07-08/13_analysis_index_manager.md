# Báo cáo Phân tích Kỹ thuật: Index Management & Lock Contention

## 1. Phân tích Hiện tượng Lock Contention (SQLSTATE 55P03)
Khi một truy vấn đọc/ghi lớn đang chạy trên PostgreSQL, nó sẽ giữ một lock tương ứng (`AccessShareLock` hoặc `RowExclusiveLock`).
Khi câu lệnh DDL thông thường (`CREATE INDEX` hoặc `DROP INDEX`) được kích hoạt, PostgreSQL sẽ cố gắng lấy `AccessExclusiveLock`. Nếu bảng đang bận, DDL sẽ xếp hàng chờ trong hàng đợi Lock Queue. Do bản chất ưu tiên của lock độc quyền, PostgreSQL sẽ **chặn tất cả các kết nối tiếp theo** (kể cả SELECT/INSERT nhẹ) tạo nên hiện tượng treo hệ thống.

## 2. Giải pháp Kỹ thuật
- **Sử dụng CONCURRENTLY**: Từ khóa `CONCURRENTLY` trong `CREATE INDEX CONCURRENTLY` và `DROP INDEX CONCURRENTLY` cho phép PostgreSQL chạy DDL mà không cần khóa ghi/đọc bảng chính.
- **Thực thi Ngoài GORM Transaction**: `CONCURRENTLY` bắt buộc phải chạy ngoài khối TRANSACTION trong Go. Do đó, `IndexManager` sử dụng trực tiếp connection pool `sql.DB` và chạy `ExecContext` để đảm bảo thực thi độc lập.
- **Ngăn chặn SQL Injection**: Do tên bảng, schema, và cột là tham số động không hỗ trợ binding variable của PostgreSQL trong DDL, chúng tôi đã sử dụng thư viện `pkgs/sqlutil.QuoteIdent` để sanitize hoàn toàn các định danh trước khi ghép chuỗi SQL.
- **Bảo vệ System Indexes**: Chặn việc drop các index hệ thống có tiền tố `pk_` (Primary Key) hoặc `ux_` (Unique Key) từ Frontend để tránh sai sót vận hành.
