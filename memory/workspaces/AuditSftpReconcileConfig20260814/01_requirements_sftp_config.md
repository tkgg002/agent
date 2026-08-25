# Requirements: Chuẩn hóa & Tháo gỡ 6 Tripwires SFTP Kafka Connect Reconcile

## 1. Bối cảnh & Mục tiêu
- **Môi trường:** Đưa cấu hình `kafka-connect-fs` (plugin `mmolimar`) cho SFTP CSV ingestion lên môi trường Staging/Production chuẩn Enterprise.
- **Nghiệp vụ:** Hệ thống Đối soát Giao dịch (Reconciliation) Goopay.
- **Mục tiêu:** 
  1. Đánh giá chuyên sâu 6 Tripwires + 1 Bổ sung từ nhận xét của User.
  2. Cung cấp 1 Cấu hình Chuẩn Production-Ready (Golden Template Config) tháo gỡ 100% rủi ro bảo mật, hiệu năng, toàn vẹn dữ liệu.
  3. Lập Kế hoạch Triển khai Chi tiết (Architectural & Operational Roadmap) cho đội ngũ Engineering.

## 2. Tiêu chí Chấp nhận (Definition of Done)
- [x] Tháo gỡ hoàn toàn 6 Bẫy chết người (Plain-text credential, host.docker.internal, DDoS SFTP poll, Partial Read, Shotgun Config Header, Null Key Ordering).
- [x] Bổ sung 3 Cổng An toàn Enterprise bị thiếu (DLQ Error Handling, Double/Float Precision Protection, Idempotent Deduplication).
- [x] Cung cấp JSON Config chuẩn duy nhất sẵn sàng nộp lên Kafka Connect REST API.
- [x] Lập quy trình vận hành SFTP Server (Atomic Rename Policy + Retention/Archive Cleanup Script).
