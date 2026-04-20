# Technical Architecture Review — CDC Integration System

> Date: 2026-04-14
> Role: Brain (Architecture Review)
> Đọc toàn bộ workspace docs + verify infra thực tế

---

## 1. Hệ thống này là gì

**GooPay CDC Platform** — nền tảng đồng bộ dữ liệu cho hệ thống thanh toán GooPay (~60 microservices). Thu thập thay đổi từ nhiều source databases → tập trung về PostgreSQL làm Single Source of Truth cho reporting, analytics, và downstream services.

---

## 2. Quy mô thực tế (đã verify)

### Sources
| Source | Loại | Port | Collections/Tables | Status |
|:-------|:-----|:-----|:-------------------|:-------|
| payment-bill-service | MongoDB ReplicaSet | 17017 | 8 collections (6 active, 2 non-active) | ✅ Connected |
| (chưa kết nối) | MySQL 8.0 Binlog | 13306 | Chưa xác định | Config sẵn |

### Data volume
- `refund_requests`: 1,712 rows (verified)
- `payment-bills`: 1,000,001 rows (bridge verified)
- Tổng ước tính: 10-50 triệu records, 500GB projected

### Destinations
- PostgreSQL 15 (goopay_dw) — 1 destination hiện tại
- Multi-destination ready (model có `airbyte_destination_id`)

### Streams đang active trong Airbyte
```
payment-bills              → payment_bills          (selected, 36 fields)
refund-requests            → refund_requests        (selected, 30 fields)
payment-bill-histories     → payment_bill_histories (selected, 0 fields*)
payment-bill-codes         → payment_bill_codes     (selected, 0 fields*)
payment-bill-events        → payment_bill_events    (selected, 0 fields*)
payment-bill-holdings      → payment_bill_holdings  (selected, 0 fields*)
identitycounters           → identitycounters       (selected, 0 fields*)
refund-requests-histories  → (non-selected)
```
*0 fields = Airbyte DiscoverSchema trả properties rỗng cho streams này

---

## 3. Technology Stack

```
Sources:     MongoDB ReplicaSet + MySQL 8.0
CDC:         Debezium Server 2.5 (MongoDB Change Stream → NATS)
Batch Sync:  Airbyte OSS (cron 24h, propagate_columns)
Broker:      NATS JetStream 2.9 (KHÔNG dùng Kafka)
Worker:      Go 1.26.1 (10 goroutines/pod, 500-record batching)
Database:    PostgreSQL 15 (Single Source of Truth)
Cache:       Redis 7
ID:          Sonyflake (64-bit BigInt)
JSON Parse:  gjson
CMS:         Go Fiber + React + Ant Design
Auth:        JWT + RBAC (admin/operator)
Container:   Docker Compose (dev), K8s ready (deployment yaml)
```

---

## 4. NATS JetStream — có cần Kafka không?

### Hiện trạng NATS
- 1 stream: `DebeziumStream`
- Subjects: `cdc.goopay.>`
- Storage: Memory
- 4 users ACL
- Throughput verified: 5,640 rows/sec

### NATS thiếu gì so với Kafka

| Thiếu | Impact | Cần ngay? |
|:------|:-------|:----------|
| **UI monitoring** | Không thấy queue lag, pending messages | ⚠️ Trung hạn — thêm vào CMS |
| **Schema Registry** | Không validate schema evolution | Không — SchemaInspector đủ |
| **Partitioning** | Scale limited nếu > 50K events/sec | Không — 5K đủ hiện tại |
| **Exactly-once** | Possible duplicates | Không — hash dedup xử lý |
| **Connector ecosystem** | Ít connectors hơn Kafka Connect | Không — custom Go code linh hoạt hơn |
| **Log compaction** | Không giữ latest state per key | Không — query DB thay vì replay |

### Kết luận: NATS ĐỦ cho quy mô hiện tại

- 2 source databases, 8 collections, 10-50M records
- 5K events/sec throughput đã verified
- Kafka chỉ cần nếu: > 50K events/sec sustained, hoặc cần Schema Registry enforce, hoặc cần 200+ connector ecosystem

### Khi nào cần Kafka
- Scale > 30 source databases
- Throughput > 50K events/sec sustained
- Cần multi-datacenter replication
- Cần strict schema governance (regulated industry)

---

## 5. Monitoring NATS hiện tại

### Có
- HTTP: `http://localhost:18222/jsz` (streams info), `/connz` (connections)
- CLI: `nats stream info`, `nats consumer info`
- Activity Log trong CMS (ghi tất cả operations)
- Worker Schedule Manager (bật/tắt/interval per operation)

### Thiếu
- **Không có UI xem messages trong stream** — phải dùng CLI `nats sub`
- **Không có consumer lag dashboard** — bao nhiêu messages chờ xử lý
- **Không có throughput graph** — messages/sec theo thời gian

### Khuyến nghị
Thêm page "Queue Monitor" trong CMS:
- Gọi NATS HTTP `/jsz` → hiện streams, consumers, pending
- Gọi NATS HTTP `/connz` → hiện connections
- Tính lag = stream messages - consumer acked
- Grafana + Prometheus NATS exporter cho production

---

## 6. Vấn đề P0 chưa giải

**`_raw_data` không chứa đầy đủ data từ source.**

- Airbyte ghi typed columns, bridge pack `to_jsonb(*)` = chỉ destination columns
- Field mới từ source miss trong khoảng chờ Airbyte sync (24h)
- Debezium Change Stream gửi `fullDocument` JSON gốc — giải quyết P0 NẾU Worker ghi `fullDocument` vào `_raw_data`

**Debezium đã streaming** — cần verify Worker nhận event + ghi `_raw_data` = `fullDocument`.

---

## 7. 4 Services hiện tại

| Service | Project | Port | Status |
|:--------|:--------|:-----|:-------|
| CDC Worker | centralized-data-service | 8082 | ✅ Running |
| CMS API | cdc-cms-service | 8080 | ✅ Running |
| CMS Frontend | cdc-cms-web | 5173 | ✅ Running |
| Auth Service | cdc-auth-service | 8081 | ✅ Running |
| Debezium Server | gpay-debezium | 18083 | ✅ Streaming |

---

## 8. Đánh giá: Cần Kafka không?

**KHÔNG CẦN** ở giai đoạn hiện tại.

Lý do:
1. 2 source databases, 8 collections — NATS thừa sức
2. 5K events/sec — NATS handle 10-100K
3. Debezium Server native NATS sink — không cần Kafka adapter
4. Kafka thêm ZooKeeper + 3 brokers = 4-8 GB RAM + ops overhead
5. Team size nhỏ — ops cho Kafka tốn effort

**Khi nào re-evaluate**: Scale > 30 DBs hoặc > 50K events/sec hoặc cần Schema Registry.
