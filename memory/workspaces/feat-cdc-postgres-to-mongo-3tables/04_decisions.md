# 04 Architectural Decision Records (ADRs)

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Chủ quản**: Brain & Staff Engineer

---

## ADR-01: Định tuyến 1 Topic duy nhất thay vì 3 Topics riêng biệt
- **Bối cảnh**: 3 bảng `orders`, `order_items`, `order_payments` có quan hệ 1:N và 1:1. Trong Kafka, thứ tự FIFO chỉ được bảo đảm trên 1 partition của 1 topic. Nếu xuất ra 3 topics, các consumer pods sẽ đọc độc lập và gây race condition nghiêm trọng.
- **Quyết định**: Sử dụng Debezium SMT `ByLogicalTableRouter` để gom cả 3 bảng về chung topic `cdc.pg.unified.orders` với Kafka Key là `order_id`.
- **Hệ quả**: Tất cả thay đổi của cùng 1 đơn hàng rơi vào cùng 1 Kafka Partition, bảo toàn 100% thứ tự commit từ PostgreSQL WAL.

---

## ADR-02: Cơ chế Unwrap khi Delete để bảo toàn Key
- **Bối cảnh**: Mặc định Debezium event `DELETE` có `after = null`. SMT `ValueToKey` thông thường sẽ sinh ra message key `null`, khiến Kafka phân phối ngẫu nhiên (round-robin) làm vỡ trật tự partition.
- **Quyết định**: Sử dụng `ExtractNewRecordState` với cấu hình `delete.handling.mode: rewrite` để bảo toàn trường `order_id` khi xóa.

---

## ADR-03: Triệt tiêu Zombie Document bằng No-Upsert cho Bảng Con
- **Bối cảnh**: Nếu bảng cha đã bị xóa mà một event đến trễ từ bảng con được ghi với `Upsert: true`, MongoDB sẽ tạo ra một document rác mồ côi.
- **Quyết định**: Bảng cha khi xóa dùng Soft-Delete (`is_deleted: true`). Bảng con tuyệt đối không dùng `Upsert: true` và lọc điều kiện `is_deleted != true`.

---

## ADR-04: Stateless Consumer thay vì Stateful In-memory Buffer
- **Bối cảnh**: Ý tưởng buffer 200ms in-memory vi phạm Simplicity First, biến worker thành stateful và có nguy cơ mất mát dữ liệu hoặc lệch offset khi pod restart.
- **Quyết định**: Worker hoàn toàn stateless. Tận dụng cơ chế natural batching của Kafka (`reader.FetchMessage` batch 500 items), build `WriteModel` trong RAM cục bộ và gọi 1 lệnh `BulkWrite(SetOrdered(true))` duy nhất, sau đó commit offset.

---

## ADR-05: Subquery Lateral cho Snapshot Backfill
- **Bối cảnh**: SQL multi `LEFT JOIN` tạo tích Đề-các ($N \times M$ dòng), làm tràn RAM và nghẽn CPU Postgres khi scan dữ liệu lớn.
- **Quyết định**: Sử dụng `LEFT JOIN LATERAL` độc lập cho từng bảng con kết hợp `jsonb_agg` và `LIMIT 1`.
