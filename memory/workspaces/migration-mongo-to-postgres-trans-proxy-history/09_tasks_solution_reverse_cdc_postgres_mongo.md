# 📐 Hồ sơ Giải pháp Kỹ thuật — CDC Pipeline Chiều Ngược (PostgreSQL → MongoDB)
## Target Service: `core-trans-proxy-history-service` | Workspace: `migration-mongo-to-postgres-trans-proxy-history`

> **Mục tiêu**: Xây dựng tuyến đồng bộ dữ liệu CDC chiều ngược từ PostgreSQL (v2 Write Target mới) về MongoDB (v1 Legacy / Backup), phục vụ cơ chế Safety Net cho giai đoạn Switch Write — cho phép Rollback về MongoDB bất kỳ lúc nào mà KHÔNG mất dữ liệu phát sinh.

---

## 1. Tổng quan Kiến trúc Toàn trình (End-to-End Flow)

```
[PostgreSQL DB (v2)]
       │ (WAL - Write Ahead Log / wal_level=logical)
       ▼
[Debezium Postgres Connector / pgoutput]
       │ (Publish CDC Event JSON)
       ▼
[Event Bus (Kafka / NATS JetStream)]
       │ (Subscribe Event Topic: cdc.postgres.trans_history)
       ▼
[Reverse Transform Worker (Go / Node.js)]
       ├── 1. Circuit Breaker & Origin Check (Chống Loop CDC)
       ├── 2. Data Type & Schema Reverse Mapping (Row → BSON Document)
       └── 3. JSONB Unnesting & Struct Reconstruction
       │
       ▼
[MongoDB Writer Service]
       ├── Bulk Upsert (updateOne + upsert: true) theo _id
       └── Dead Letter Queue (DLQ) & Retry Mechanism
       │
       ▼
[MongoDB (Legacy / Backup Target)]
```

---

## 2. Các Thành phần Cốt lõi & Yêu cầu Kỹ thuật

### 🔹 Thành phần 1: PostgreSQL CDC Source (Logical Replication)
- **Cấu hình DB Postgres**:
  - `wal_level = logical`
  - `max_replication_slots >= 4`
  - `max_worker_processes >= 8`
- **Replication Slot & Publication**:
  - Tạo publication: `CREATE PUBLICATION cdc_reverse_pub FOR TABLE trans_proxy_history;`
  - Debezium Postgres Connector kết nối qua plugin `pgoutput` (hoặc `decoderbufs`).
- **Nội dung Event Capture**:
  - `op: 'c'` (Create): Payload chứa full record mới (`after`).
  - `op: 'u'` (Update): Payload chứa record trước (`before`) và sau (`after`).
  - `op: 'd'` (Delete): Payload chứa identifier của record bị xóa (`before`).

---

### 🔹 Thành phần 2: Reverse Transformer Engine (Row → BSON Converter)
Chuyển đổi kiểu dữ liệu quan hệ (Postgres) thành BSON Document (Mongo):

| Field Type Postgres | Conversion Logic | Target BSON / Mongo Field |
| :--- | :--- | :--- |
| `id` (UUID / TEXT / BIGINT) | Ep kiểu sang String hoặc `ObjectId(hex)` tùy schema cũ | `_id` |
| `TIMESTAMPTZ` / `TIMESTAMP` | Convert sang `ISODate` (`time.Time` trong Go) | Date field (e.g. `createdAt`, `updatedAt`) |
| `NUMERIC` / `DECIMAL` | Convert sang `primitive.Decimal128` | Currency / Amount fields |
| `JSONB` (e.g., `payload_json`, `metadata_json`) | `json.Unmarshal` ngược lại map/array trong Mongo | Embedded Sub-document / Array |
| `VARCHAR` / `TEXT` / `INT` | Passthrough direct mapping | String / Int32 / Int64 |

---

### 🔹 Thành phần 3: Chống Vòng Lặp CDC (CDC Infinite Evasion)
⚠️ **Nguy cơ**: Nếu CDC chiều thuận (Mongo→Postgres) và chiều ngược (Postgres→Mongo) vô tình bật đồng thời, 1 record sẽ bị sync vòng tròn vô tận gây sập hệ thống.

**Giải pháp Chống Loop (Strict Origin Identification)**:
1. Khi Worker ghi vào MongoDB, gắn thêm system metadata field:
   `"_cdc_metadata": { "origin": "pg_reverse", "synced_at": ISODate() }`
2. CDC chiều Mongo→Postgres phải cấu hình Filter Rule: **Bỏ qua tất cả Change Stream Event có `fullDocument._cdc_metadata.origin == 'pg_reverse'`**.
3. Ngược lại, CDC chiều Postgres→Mongo phải có Header Filter: **Bỏ qua event nếu trigger bởi CDC worker cũ**.

---

### 🔹 Thành phần 4: MongoDB Idempotent Writer (Bulk Write & Fail-safe)
- **Cơ chế Write**:
  - Không dùng `insertOne` / `updateStatement` đơn thuần.
  - Sử dụng `db.collection.bulkWrite()` với batch size 500 - 1000 items / batch.
  - Pattern Upsert:
    ```javascript
    db.collection.bulkWrite([
      {
        updateOne: {
          filter: { _id: doc._id },
          update: { $set: doc },
          upsert: true
        }
      }
    ]);
    ```
- **Xử lý Error & Retry (DLQ)**:
  - Nếu gặp transient error (network timeout, Mongo primary election): Retry 3 lần với Exponential Backoff.
  - Nếu gặp permanent error (BSON document size limit > 16MB, type mismatch): Đẩy vào **NATS/Kafka Dead Letter Queue (DLQ)** để CMS báo động và operator can thiệp manual.

---

## 3. Checklist Thực thi & Ma trận Điều kiện Go/No-Go

### ✅ Chuẩn bị Hạ tầng & Code (Pre-switch)
- [ ] Bật `wal_level=logical` trên Postgres DB instance.
- [ ] Implement Go Reverse Transformer module trong `centralized-data-service` (hoặc CDC worker).
- [ ] Unit Test: Verify 100% trường hợp transform Row Postgres → BSON Mongo document.
- [ ] Test E2E trên Staging: Insert/Update/Delete 10,000 bản ghi trên Postgres → kiểm tra Mongo backup khớp 100%.

### 🚨 Kịch bản Rollback khẩn cấp (Emergency Rollback Protocol)
1. Tắt Write Path trên v2 Service.
2. Đợi Reverse CDC Worker xả hết lag trên Kafka/NATS Queue về 0 (Drain Queue).
3. Đổi DNS / Route Traffic Write quay lại MongoDB Service v1.
4. **Kết quả**: Không mất bất kỳ giao dịch nào đã phát sinh trong thời gian chạy v2 trên Postgres.
