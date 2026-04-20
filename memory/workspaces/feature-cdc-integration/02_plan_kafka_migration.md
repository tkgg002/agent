# Plan: Kafka Migration

> Date: 2026-04-15
> Phase: kafka_migration
> Decisions: KRaft + Redpanda Console + Schema Registry (Avro) + Kafka Connect Distributed

## Architecture mới

```
MongoDB (60 sources) + MySQL
    ↓ Change Streams / Binlog
Debezium Kafka Connectors (Distributed Mode)
    ↓ Avro + Schema Registry
Kafka (KRaft mode, single broker dev / 3 brokers prod)
    ↓ Topics: cdc.goopay.{db}.{collection}
    ↓
    ├─→ CDC Worker (Go, Kafka consumer group)
    │       ↓ DynamicMapper → BatchBuffer
    │       ↓ PostgreSQL (destination)
    │
    ├─→ Redpanda Console (monitoring UI)
    │
    └─→ Schema Registry (Avro schema evolution)

NATS (giữ lại cho internal commands)
    ↓ cdc.cmd.*, schema.config.reload
    ↓ CMS ↔ Worker communication
```

## Infra mới (docker-compose)

| Container | Image | RAM estimate | Port |
|:----------|:------|:-------------|:-----|
| Kafka (KRaft) | confluentinc/cp-kafka:7.6 | 1-2 GB | 9092 |
| Schema Registry | confluentinc/cp-schema-registry:7.6 | 256-512 MB | 8081 |
| Kafka Connect | confluentinc/cp-kafka-connect:7.6 + Debezium | 1-2 GB | 8083 |
| Redpanda Console | redpandadata/console:latest | 128-256 MB | 8080 |
| **Tổng thêm** | | **~3-4 GB** | |

⚠️ RAM hiện tại 5.24/7.65 GB — cần nâng RAM host lên ít nhất 12 GB.

## Tasks

### T1: Infra — docker-compose thêm Kafka stack
- Kafka KRaft (single broker, dev mode)
- Schema Registry
- Kafka Connect + Debezium MongoDB connector
- Redpanda Console

### T2: Debezium — chuyển từ Server mode sang Kafka Connect mode
- Debezium MongoDB Source Connector config (JSON)
- Avro serialization + Schema Registry URL
- Topics: auto-create từ Debezium

### T3: Worker — thêm Kafka consumer
- Go kafka consumer library (segmentio/kafka-go hoặc confluent-kafka-go)
- Consumer group: `cdc-worker-group`
- Consume → parse Avro → EventHandler (giữ nguyên DynamicMapper)
- Parallel: 1 consumer per partition

### T4: Worker — giữ NATS cho commands
- NATS vẫn dùng cho: cdc.cmd.*, schema.config.reload, CMS ↔ Worker
- Kafka chỉ cho CDC event streaming

### T5: Redpanda Console config
- Connect tới Kafka broker + Schema Registry
- Hiện: topics, messages, consumer groups, lag

### T6: Test E2E
- MongoDB insert → Debezium → Kafka → Worker → PostgreSQL
- Verify trong Redpanda Console: message visible, consumer lag = 0

### T7: Cleanup
- Bỏ Debezium Server container (thay bằng Kafka Connect)
- Bỏ NATS JetStream streams cho CDC (CDC_EVENTS, DebeziumStream)
- Giữ NATS cho commands

## Execution order

```
T1 (infra) → T2 (debezium) → T5 (console) → T3 (worker) → T6 (test) → T7 (cleanup)
T4 chạy song song (NATS giữ nguyên)
```

## RAM concern

Hiện tại Docker dùng 5.24/7.65 GB. Thêm Kafka stack ~3-4 GB → cần 12+ GB.

Options:
- A) Nâng RAM Docker Desktop lên 12-16 GB
- B) Tắt bớt containers không cần (mysql, some-mongo, some-nats, some-redis)
- C) Dev mode: Kafka single broker + connect + registry minimal RAM

## Definition of Done
- [ ] MongoDB insert → Kafka topic visible trong Redpanda Console
- [ ] Consumer group `cdc-worker-group` active, lag = 0
- [ ] PostgreSQL có record mới
- [ ] Schema Registry có Avro schema cho mỗi collection
- [ ] Redpanda Console accessible, hiện đầy đủ topics/consumers/messages
