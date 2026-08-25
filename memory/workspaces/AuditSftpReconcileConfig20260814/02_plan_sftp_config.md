# 02_plan: Lộ Trình Triển Khai Cao Tầng (High-Level Master Roadmap)

## Phase 0: Audit & Đánh Giá Rủi Ro (Hoàn thành)
- [x] Phân tích 6 Bẫy nguy hiểm (Tripwires) + 1 Lưu ý dọn rác SFTP.
- [x] Rà soát 3 Cổng An toàn Enterprise bị thiếu (DLQ, Decimal Precision, Key Partitioning).
- [x] Khởi tạo Workspace Documentation chuẩn `GEMINI.md`.

## Phase 1: Chuẩn Hóa Cấu Hình Connector & Bảo Mật Secrets
- [ ] Xây dựng file secrets `/etc/kafka-connect/secrets/sftp-credentials.properties` trên Kafka Connect Nodes.
- [ ] Bật `FileConfigProvider` trên Kafka Connect Worker config.
- [ ] Soạn thảo Golden JSON Connector Config sử dụng `${file:...}` biến bảo mật.

## Phase 2: Chuẩn Hóa Quy Trình SFTP Server & Atomic Protocol
- [ ] Thiết lập thư mục `/home/gp-reconcile-admin/goopay/reconcile/` và `/home/gp-reconcile-admin/goopay/reconcile_archive/`.
- [ ] Ban hành Atomic Upload Standard cho đối tác/hệ thống nguồn: Upload `.tmp` -> Rename `.csv` khi hoàn tất.
- [ ] Cài đặt Cronjob Archive tự động di chuyển file > 3 ngày và xóa file > 90 ngày.

## Phase 3: Xây Dựng SMT Key Extraction & DLQ Pipeline
- [ ] Thiết lập Kafka Topic `cdc.sftplocal.reconcile.transactions` với 3 Partition và Factor 3.
- [ ] Cấu hình Dead Letter Queue Topic `dlq.cdc.sftplocal.reconcile.transactions`.
- [ ] Cấu hình SMT `ValueToKey` + `ExtractField` trích xuất `transaction_id` làm Record Key.

## Phase 4: Verification, Load Test & Monitoring Integration
- [ ] Test đọc file CSV mẫu 100,000 dòng.
- [ ] Test trường hợp lỗi (Negative Path): File CSV chứa dòng hỏng format -> Verify dòng hỏng vào DLQ, Connector giữ trạng thái RUNNING.
- [ ] Tích hợp Prometheus / SigNoz theo dõi Connector status và Consumer lag.
