# Yêu cầu: Tối ưu hóa xử lý bất đồng bộ cho Create/Drop Index

## 1. Bối cảnh
Khi người dùng thực hiện tạo hoặc xóa index trên UI (ví dụ tạo partial index `_deleted` cho `CountDeletedRows`), câu lệnh `CREATE INDEX CONCURRENTLY` hoặc `DROP INDEX CONCURRENTLY` bị chặn chờ các active transaction khác kết thúc. 
Do xử lý hiện tại trong `IndexHandler` là đồng bộ (synchronous), luồng xử lý NATS bị block quá lâu và vượt quá timeout của API Gateway / UI Client, dẫn đến thông báo lỗi *"server phản hồi quá lâu vui lòng thử lại sau"*.

## 2. Mục tiêu
- Chuyển đổi cơ chế xử lý của `HandleCreateIndex` và `HandleDropIndex` trong `IndexHandler` sang **bất đồng bộ (asynchronous)**.
- Khi nhận được tin nhắn NATS, worker sẽ gửi phản hồi `CommandResult` với `Status: "success"` ngay lập tức để giải phóng NATS client và tránh lỗi timeout trên UI.
- Thực thi câu lệnh tạo/xóa index (`CreateIndexConcurrently` và `DropIndexConcurrently`) dưới nền trong một goroutine riêng.
- Sử dụng detached context hoặc `context.Background()` kế thừa span/header cho goroutine chạy ngầm để tránh bị cancel khi request gốc kết thúc.
- Ghi log rõ ràng khi tiến trình chạy ngầm thành công hoặc thất bại.

## 3. Ràng buộc & Tiêu chuẩn Gates (DoD)
- Không chỉnh sửa trực tiếp source code bằng Brain, phải delegate qua Muscle/Sub-agent.
- Cập nhật tài liệu workspace đầy đủ.
- Chạy test suite để kiểm tra tính đúng đắn và biên dịch thành công.
- Không tự ý git commit.
