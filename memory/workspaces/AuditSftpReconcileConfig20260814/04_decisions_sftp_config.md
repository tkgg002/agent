# 04_decisions: Nhật Ký Quyết Định Kiến Trúc (Architectural Decision Records)

## ADR-001: Sử dụng FileConfigProvider cho SFTP Credentials
- **Bối cảnh:** Cấu hình ban đầu hardcode mật khẩu plain-text `sftp_password` trên URI `fs.uris`.
- **Quyết định:** Sử dụng `FileConfigProvider` của Kafka Connect để bọc credentials vào file biến bảo mật `/etc/kafka-connect/secrets/sftp-credentials.properties`.
- **Hệ quả:** Loại bỏ hoàn toàn rủi ro lộ secret qua REST API `GET /connectors` và startup logs.

## ADR-002: Áp dụng SMT ValueToKey + ExtractField theo transaction_id
- **Bối cảnh:** Mặc định Key = Null khiến Kafka dùng Round-Robin đẩy record vào các partition ngẫu nhiên, làm xáo trộn thứ tự dòng CSV.
- **Quyết định:** Sử dụng SMT `ValueToKey` kết hợp `ExtractField` để trích xuất `transaction_id` làm Record Key.
- **Hệ quả:** Các record của cùng 1 giao dịch luôn được định tuyến vào đúng 1 Partition duy nhất, đảm bảo strict partition ordering.

## ADR-003: Quy chuẩn Atomic Upload Protocol (.tmp -> .csv)
- **Bối cảnh:** `SleepyPolicy` quét mỗi 3s/60s có thể chộp phải file CSV đang trong quá trình upload (Partial Read).
- **Quyết định:** Bắt buộc đối tác/hệ thống nguồn ghi file tạm `.tmp` và đổi tên sang `.csv` khi upload xong 100%. Regex `^reconcile_.*\\.csv$` chỉ khớp file đã hoàn tất.
- **Hệ quả:** Triệt tiêu hoàn toàn rủi ro mất 90% dữ liệu đối soát do đọc file dở dang.

## ADR-004: Bật Dead Letter Queue (DLQ) & Error Tolerance
- **Bối cảnh:** Cấu hình ban đầu thiếu xử lý lỗi, 1 dòng CSV rác sẽ làm sập toàn bộ Connector (FAILED state).
- **Quyết định:** Bật `errors.tolerance=all` và định tuyến record lỗi vào `dlq.cdc.sftplocal.reconcile.transactions`.
- **Hệ quả:** Connector duy trì trạng thái RUNNING liên tục 24/7, dữ liệu lỗi được lưu vết đầy đủ ở DLQ để audit.
