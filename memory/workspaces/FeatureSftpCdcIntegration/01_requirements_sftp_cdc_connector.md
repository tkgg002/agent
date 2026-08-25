# Yêu cầu: Tích hợp SFTP Source Connector vào hệ thống CDC

## 1. Bối cảnh & Mô hình đối soát đề xuất
Hệ thống đối soát tài chính phối hợp giữa **n8n** và **cdc-worker** theo mô hình:
- **n8n (Orchestrator & Business Logic):**
  - Collect dữ liệu giao dịch nội bộ (Internal) đẩy vào DB (nội bộ).
  - Collect dữ liệu từ phía ngân hàng đối tác (External) ➔ Ghi vào file template final ➔ Upload lên SFTP Server.
  - So sánh dữ liệu đối soát: Đối chiếu `external` (dữ liệu bank do cdc-worker import) === `internal` (dữ liệu hệ thống) để ra kết quả Khớp/Lệch.
- **cdc-worker (CDC Engine & Data Pipe):**
  - Đóng vai trò ghi nhận dữ liệu file final của ngân hàng vào DB (bảng Shadow của Postgres/Mongo).
  - Lắng nghe Kafka topic được SFTP Source Connector sinh ra khi n8n upload file lên SFTP, tự động parse và insert/upsert dữ liệu một cách tối ưu.

## 2. Mục tiêu kỹ thuật
- Tích hợp **SFTP Source Connector** trên Kafka Connect để lắng nghe và chuyển đổi dữ liệu file final từ SFTP thành event stream trong Kafka topic.
- Tự động gọi Kafka Connect API từ `cdc-cms-service` khi người dùng submit cấu hình nguồn trên CMS UI.
- Tận dụng tối đa bộ máy xử lý event CDC hiện có của `cdc-worker` (BatchBuffer, Transmuter, Schema Evolution) để lưu dữ liệu vào Shadow Table Postgres mà không phải viết lại driver SFTP hay db query.

## 3. Ràng buộc & Điểm lưu ý
- n8n không kết nối ghi trực tiếp vào DB hệ thống nhằm bảo đảm data integrity. Việc ghi dữ liệu bank vào DB hoàn toàn do `cdc-worker` thực thi.
- Cơ chế trigger luồng đối soát của n8n: n8n sẽ so sánh sau khi `cdc-worker` hoàn tất việc ghi. Cần dùng Cron trigger hoặc cdc-worker bắn Webhook/NATS event thông báo đã import xong.
- Tránh lỗi đọc dở dang (Partial Read) của Kafka Connect: n8n khi đẩy file lên SFTP bắt buộc phải upload dưới dạng file tạm (ví dụ `*.csv.tmp`) rồi mới Rename sang `.csv`.
