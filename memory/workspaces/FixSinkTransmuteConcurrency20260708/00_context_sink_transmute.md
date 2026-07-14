# Phạm vi & Thành phần (Context): Concurrency & Batching Optimization

Tài liệu này xác định ngữ cảnh hệ thống và phạm vi ảnh hưởng của thay đổi tối ưu hóa.

## 1. Thành phần ảnh hưởng (System Components)
*   **Shadow Sink (`BatchBuffer`):** Quản lý bộ đệm gom lô ghi xuống Postgres Shadow DB.
*   **Transmute Worker (`TransmuteHandler`):** Đăng ký nhận tin nhắn trigger từ NATS, thực hiện transmuting từ Shadow sang Master DB.
*   **NATS JetStream:** Hàng đợi tin nhắn điều phối (Orchestrator) kết nối giữa chặng Sink và chặng Transmute.

## 2. Ranh giới hệ thống (System Boundaries)
*   **Kafka:** Khách hàng (Consumer) của Kafka không bị thay đổi logic đọc, chỉ được hưởng lợi từ việc giảm thời gian chặn (blocking) do Flush song song.
*   **Postgres:** Giữ nguyên schema, chỉ thay đổi luồng và lưu lượng truy vấn đồng thời (Upsert).
