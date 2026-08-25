# Tasks: SFTP Reconcile Kafka Connect Production Hardening

- [x] **Task 1: Phản biện & Đánh giá chuyên sâu mớ nhận xét (6 Tripwires + 1 Garbage Collection)**
  - Xác nhận tính đúng đắn của 6 Bẫy nguy hiểm.
  - Bổ sung 3 Bẫy Enterprise còn thiếu (DLQ, Double Precision, Idempotency).
- [x] **Task 2: Lập Kế hoạch & Giải pháp Kiến trúc Chuẩn Production (Golden Config Spec)**
  - Tích hợp `FileConfigProvider` bảo mật mật khẩu SFTP.
  - Thiết lập SMT `ValueToKey` trích xuất `transaction_id` / `reconcile_id` làm Record Key.
  - Tối ưu `policy.sleepy.sleep` và tắt `policy.recursive` chống DDoS SFTP Server.
  - Cấu hình DLQ (Dead Letter Queue) cách ly record rác mà không crash Connector.
- [x] **Task 3: Xây dựng Quy chuẩn Vận hành SFTP Server & Cleanup Automation**
  - Đóng gói Atomic Rename pattern (`.tmp` -> `.csv`).
  - Lập Cron Script dọn dẹp archive tự động.
