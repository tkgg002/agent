# 06 Test Cases & Verification Scenarios

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Phạm vi kiểm thử**: CDC Ingest Pipeline, SMT Routing, Mongo Atomic Operations, Backfill Snapshot

---

## Danh mục Kịch bản Kiểm thử

| Mã TC | Tên Kịch bản | Mô tả Thực hiện | Kết quả Kỳ vọng |
|:---|:---|:---|:---|
| **TC-01** | **Strict In-Order Execution** | Gửi liên tiếp 50 thay đổi xen kẽ giữa `orders`, `order_items`, `order_payments` của cùng 1 `order_id` qua Kafka. | Mọi thao tác ghi nhận trên MongoDB theo đúng tuần tự WAL commit, không xảy ra race condition giữa các luồng. |
| **TC-02** | **Idempotent Pull-then-Push** | Replay cùng một sự kiện INSERT/UPDATE của `order_items` 5 lần liên tiếp. | Mảng `items` trong MongoDB không bị duplicate phần tử, chỉ có đúng 1 item duy nhất với `_v` mới nhất. |
| **TC-03** | **Zombie Document Prevention** | Gửi sự kiện DELETE `orders` (Soft-delete), sau đó gửi sự kiện UPDATE của một item con `order_items`. | Document `orders` có `is_deleted: true`; sự kiện item con bị reject do điều kiện filter `is_deleted != true`, không tạo document mồ côi. |
| **TC-04** | **Unwrap Key on Delete** | Thực thi `DELETE FROM order_items WHERE id = 10;` trên PostgreSQL. | Debezium SMT unwrap payload cũ, Kafka message key vẫn là `order_id` (không bị null), item bị xóa khỏi mảng trong Mongo. |
| **TC-05** | **Element-level Version Isolation** | Cập nhật Item A lúc $T_2$, Item B lúc $T_1$ ($T_1 < T_2$). Replay sự kiện $T_1$ của Item B. | Item B cập nhật bình thường, không bị drop bởi version của Item A. |
| **TC-06** | **Cartesian-Free Snapshot** | Kích hoạt SnapshotRunner đọc 10,000 đơn hàng có trung bình 5 items và 2 payments. | Query `LEFT JOIN LATERAL` chạy < 100ms/batch, không phình RAM/CPU, đạt throughput > 5,000 docs/s sang Mongo. |
| **TC-07** | **Data Integrity Recon Check** | Chạy Recon Job so sánh hash giữa PostgreSQL và MongoDB cho 1,000 orders bất kỳ. | Drift = 0. Khi giả lập sửa trực tiếp 1 item trên Mongo, Recon phát hiện chính xác drift và tự động Heal. |
