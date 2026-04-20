# ADR-015: NATS JetStream vs Kafka — So sánh chi tiết

> Date: 2026-04-14
> Status: Approved (dùng NATS JetStream, không dùng Kafka)

## Quyết định

Dùng NATS JetStream thay Kafka cho CDC event streaming.

## So sánh tổng thể

| Tiêu chí | Kafka | NATS JetStream | Hệ thống hiện tại |
|:---------|:------|:---------------|:-------------------|
| **Infra cần** | ZooKeeper + Kafka Brokers (3+ nodes) | 1 NATS server (đã có) | ✅ NATS đã chạy |
| **RAM tối thiểu** | 2-4 GB | 50-200 MB | ✅ NATS nhẹ |
| **Persistence** | Disk-based (segments) | Memory hoặc File | ⚠️ Memory = mất khi restart |
| **Throughput** | 100K-1M msg/sec | 10K-100K msg/sec | ✅ Đủ cho 5K events/sec |
| **Ordering** | Per-partition guaranteed | Per-subject guaranteed | ✅ Đủ |
| **Consumer Groups** | Native (rebalancing) | Durable Consumers (manual) | ✅ Đủ |
| **Replay** | From offset (mạnh) | From sequence/time | ✅ Đủ |
| **Dead Letter Queue** | Native support | Tự implement (đã có) | ✅ Đã implement |
| **Schema Registry** | Confluent Schema Registry | Không có | ⚠️ Thiếu |
| **Exactly-once** | Có (transactions) | At-least-once + dedup | ⚠️ Cần hash dedup |
| **Monitoring UI** | Kafka Manager, Confluent Control Center, Kafdrop | Không có native UI | ❌ **THIẾU** |
| **Partition/Scaling** | Topic partitions → consumer rebalance | Không có partitions | ⚠️ Scale khác cách |
| **Message retention** | Configurable (time/size) | Configurable (limits/interest) | ✅ |
| **Connectors ecosystem** | 200+ connectors (Kafka Connect) | Debezium Server sink + custom | ⚠️ Ít hơn |
| **Community/Support** | Rất lớn, enterprise | Nhỏ hơn, growing | |

## NATS JetStream THIẾU gì so với Kafka

### 1. UI Quản lý trực quan ❌
**Kafka có**: Kafdrop, Kafka Manager, Confluent Control Center, AKHQ — xem topics, consumers, lag, messages, partitions trực quan.

**NATS có**: Chỉ có CLI (`nats` command) + HTTP monitoring endpoint (`/jsz`, `/connz`). Không có UI dashboard native.

**Giải pháp**: 
- `nats-top` (CLI monitoring)
- Tự build dashboard trong CMS (đọc NATS monitoring API)
- Hoặc dùng Grafana + Prometheus (NATS exporter)

### 2. Schema Registry ❌
**Kafka có**: Confluent Schema Registry — enforce schema evolution (Avro/Protobuf), compatibility checks.

**NATS**: Không có. Messages là bytes, không validate schema.

**Giải pháp hiện tại**: CDC Worker validate schema khi parse JSON. SchemaInspector detect drift. Đủ cho use case.

### 3. Partitioning / Parallel Consumption ⚠️
**Kafka**: Topic có N partitions → N consumers parallel. Rebalance tự động khi consumer join/leave.

**NATS JetStream**: Không có partitions. Scale bằng:
- Nhiều consumers cùng 1 durable subscription (round-robin)
- Consumer pool trong Worker (10 goroutines)

**Giải pháp hiện tại**: Worker pool 10 goroutines per pod × 5 pods = 50 parallel consumers. Đủ cho 5K events/sec.

### 4. Exactly-once Semantics ⚠️
**Kafka**: Producer idempotency + transactions → exactly-once end-to-end.

**NATS JetStream**: At-least-once. Duplicate possible nếu ack timeout.

**Giải pháp hiện tại**: Hash-based dedup trong bridge (`WHERE _hash IS DISTINCT FROM EXCLUDED._hash`). Idempotent upsert.

### 5. Connector Ecosystem ⚠️
**Kafka Connect**: 200+ connectors (JDBC, Elasticsearch, S3, BigQuery...) — plug-and-play.

**NATS**: Debezium Server hỗ trợ NATS sink. Không có ecosystem connectors.

**Giải pháp hiện tại**: Custom Go code cho mỗi sink (PostgreSQL, future BigQuery...). Linh hoạt hơn nhưng tốn effort.

### 6. Log Compaction ❌
**Kafka**: Topic compaction — giữ lại message mới nhất per key. Useful cho CDC (latest state per record).

**NATS JetStream**: Không có compaction. Messages giữ theo limits policy.

**Impact**: Nếu cần replay latest state per record → phải query DB thay vì replay stream.

## Quản lý Queue hiện tại

### Monitoring endpoints (NATS HTTP)
```
http://localhost:18222/          → Server info
http://localhost:18222/jsz       → JetStream info (streams, consumers, messages)
http://localhost:18222/connz     → Connections (who connected)
http://localhost:18222/subsz     → Subscriptions
```

### CLI commands
```bash
# List streams
nats stream ls -s nats://localhost:14222

# Stream info
nats stream info DebeziumStream -s nats://localhost:14222

# Consumer info
nats consumer info DebeziumStream cdc-worker-group -s nats://localhost:14222

# Monitor realtime
nats sub "cdc.goopay.>" -s nats://localhost:14222  # watch events live
```

### Hiện tại thiếu
- Không có UI xem messages trong stream
- Không có UI xem consumer lag (bao nhiêu messages chưa process)
- Không có UI xem connection status
- Phải dùng CLI hoặc curl monitoring endpoint

## Khuyến nghị

### Ngắn hạn (hiện tại — đủ dùng)
NATS JetStream đủ cho:
- 5K events/sec throughput
- 60 microservices CDC
- Debezium → Worker → PostgreSQL flow

### Trung hạn (khi cần UI + monitoring)
Thêm NATS monitoring vào CMS:
- Page "Queue Monitor" đọc NATS HTTP API `/jsz`
- Hiển thị: streams, consumers, pending messages, lag
- Alert khi lag > threshold

### Dài hạn (khi cần scale > 50K events/sec)
Đánh giá lại Kafka nếu:
- Cần 100K+ events/sec sustained
- Cần Schema Registry enforce
- Cần Kafka Connect ecosystem (VD: sink trực tiếp BigQuery/Elasticsearch)
- Cần exactly-once end-to-end
