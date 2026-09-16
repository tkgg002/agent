# 01 Requirements: CDC PostgreSQL → MongoDB & Multi-Table Aggregation

> **Feature**: `feat-cdc-postgres-to-mongo-3tables`  
> **Target Systems**: `centralized-data-service`, `cdc-cms-service`, Kafka Connect Debezium

---

## 1. Yêu cầu Chức năng (Functional Requirements)

### FR-01: Capture CDC từ PostgreSQL
- Bắt tất cả sự kiện `INSERT` (`c`), `UPDATE` (`u`), `DELETE` (`d`), `SNAPSHOT` (`r`) từ PostgreSQL logical replication stream.
- Đảm bảo trích xuất đầy đủ `before` image đối với lệnh `DELETE` thông qua cấu hình `REPLICA IDENTITY FULL` trên PostgreSQL.

### FR-02: Định tuyến Stream Hợp nhất (Unified Topic Routing)
- Với bài toán gom 3 bảng (`orders`, `order_items`, `order_payments`), connector BẮT BUỘC định tuyến tất cả event về **1 Topic Kafka duy nhất** (`cdc.pg.unified.orders`).
- Tất cả message phải mang **Message Key = `order_id`** để Kafka phân vùng cố định vào 1 Partition, đảm bảo thứ tự tuần tự tuyệt đối (Strict FIFO).

### FR-03: Bảo toàn Kiểu dữ liệu (PostgreSQL → BSON Conversion)
- `UUID`: Chuyển đổi sang canonical text `8-4-4-4-12`.
- `NUMERIC / DECIMAL`: BẮT BUỘC chuyển đổi sang BSON `primitive.Decimal128`, không làm tròn dạng float.
- `TIMESTAMPTZ / TIMESTAMP`: Chuyển đổi sang UTC `primitive.DateTime` (`ISODate`).
- `JSONB`: Unmarshal thành nested BSON Document (`bson.M`) hoặc Array (`bson.A`).

### FR-04: Gom 3 Bảng về 1 Document (Multi-Table Aggregation)
- Bảng cha `orders`: Cập nhật root fields của document `orders` trong MongoDB.
- Bảng con 1:N `order_items`: Cập nhật embedded array `items: [ { id, product_id, quantity, price, _v } ]` theo cơ chế **Pull-then-Push** lũy thừa.
- Bảng con 1:1 `order_payments`: Cập nhật embedded subdocument `payment: { id, payment_method, amount, status, _v }`.

### FR-05: Chống Zombie Document khi Xóa
- Khi bảng cha `orders` bị xóa: Sử dụng **Soft-Delete** (`is_deleted: true`, `deleted_at: timestamp`).
- Bảng con (`order_items`, `order_payments`) **tuyệt đối KHÔNG dùng `SetUpsert(true)`** — chỉ cập nhật khi document cha đang tồn tại (`is_deleted != true`).

### FR-06: Backfill Lịch sử Hiệu năng Cao (Snapshot Runner)
- Chạy batch stream SQL `LEFT JOIN LATERAL` đọc dữ liệu lịch sử từ Postgres, gom thành JSONB và ghi bulk sang Mongo mà không gây Cartesian Product.

### FR-07: Đối soát Tính Toàn vẹn (Reconciliation & Healing)
- So sánh MD5 Checksum giữa PostgreSQL (3 tables) và MongoDB (Collection `orders`) cho các record `is_deleted != true`.

---

## 2. Yêu cầu Phi chức năng (Non-Functional Requirements)

- **NFR-01 (Stateless Architecture)**: Worker hoàn toàn stateless, không dùng in-memory debounce window. Xử lý micro-batch từ Kafka và commit offset sau khi ghi MongoDB thành công.
- **NFR-02 (High Throughput)**: Hỗ trợ tốc độ xử lý streaming $\ge 3,000$ msg/s và snapshot backfill $\ge 8,000$ docs/s.
- **NFR-03 (Idempotency)**: Bất kỳ message nào bị replay/retry nhiều lần đều không làm thay đổi trạng thái cuối cùng hoặc nhân đôi phần tử mảng trong MongoDB.
