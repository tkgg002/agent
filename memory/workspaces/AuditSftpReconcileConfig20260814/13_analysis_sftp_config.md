# Deep Analysis: Review & Expansion of 6 Tripwires SFTP Kafka Connect

## 1. Đánh giá Mớ Nhận Xét (Review of User Request Text)

| Stt | Tripwire | Đánh giá của AI | Chi tiết Kỹ thuật / Bổ sung sâu |
|---|---|---|---|
| 1 | **Bẫy Bảo Mật (Plain-text Password)** | **ĐÚNG 100%** | Hardcode mật khẩu trong `fs.uris` làm rò rỉ secret qua Kafka Connect REST API `GET /connectors/<name>/config` và startup logs. Bắt buộc dùng `FileConfigProvider` hoặc `EnvVarConfigProvider`. |
| 2 | **Bẫy Môi Trường (`host.docker.internal`)** | **ĐÚNG 100%** | `host.docker.internal` chỉ tồn tại trên Docker Desktop loopback, sẽ crash lập tức với `UnknownHostException` trên Linux bare-metal/K8s. |
| 3 | **Bẫy DDoS SFTP (`sleep=3000`, `recursive=true`)** | **ĐÚNG 100%** | Poll 3 giây/lần kèm đệ quy tạo bão lệnh SFTP `readdir` qua SSH connection. Làm quá tải SFTP CPU & dễ bị Fail2ban/Firewall chốt IP. |
| 4 | **Bẫy Đọc Dở Dang (Partial Read)** | **ĐÚNG 100%** | Nghiêm trọng nhất đối với Reconcile. Connector chộp file khi đang upload 10% -> mất data 90% còn lại. Phải bắt buộc dùng Atomic Rename Protocol (`.tmp` -> `.csv`). |
| 5 | **Bẫy Cấu Hình Rác (Shotgun Config Header)** | **ĐÚNG 100%** | Khai báo 3 thuộc tính header đè nhau (`file_reader.delimited.settings.header`, `file_reader.delimited.header`, `file_reader.csv.header`). Làm header rơi vào row data -> làm sập Consumer/DB downstream. |
| 6 | **Bẫy Mất Thứ Tự (Null Key Ordering)** | **ĐÚNG 100%** | Key = Null khiến Kafka dùng Round-Robin đẩy record lung tung vào các partition. Phải dùng SMT `ValueToKey` trích xuất `transaction_id` làm Key. |
| 7 | **Bổ Sung: Nghẽn Directory Listing** | **ĐÚNG 100%** | Không dọn file cũ làm `ls` trên SFTP mất hàng chục giây. Phải có Cron Script Archive file cũ +90 ngày. |

## 2. Các Lỗ Hổng Enterprise Bị Thiếu Trong Nhận Xét Trút Ra (What Was Missed)

1. **Thiếu Dead Letter Queue (DLQ) & Error Handling:**
   - Nếu bài viết chỉ tập trung vào 6 tripwires mà không bật DLQ (`errors.tolerance=all`, `errors.deadletterqueue.topic.name`), khi gặp 1 dòng CSV rác (sai mã hóa UTF-8, thừa cột), toàn bộ Kafka Connect Task sẽ chuyển trạng thái `FAILED` và dừng ingest lập tức.
2. **Double / Floating Point Truncation trên JSON Converter:**
   - `value.converter.schemas.enable=false` biến CSV numeric thành JSON Number. Với số tiền lớn (VD 500,000,000,000 VND), Javascript/JSON Parser ở Consumer downstream có thể bị đứt độ chính xác nếu dùng float. Cần lưu ý schema coercion hoặc parse string ở Consumer.
3. **Partition Strategy & Key Extraction:**
   - Nếu topic có N partitions, việc trích xuất `transaction_id` làm Key đảm bảo mọi event của cùng 1 giao dịch đi đúng partition. Nếu muốn strict file-level sequence, topic Reconcile nên duy trì 1 partition hoặc dùng Key theo `transaction_id`.
