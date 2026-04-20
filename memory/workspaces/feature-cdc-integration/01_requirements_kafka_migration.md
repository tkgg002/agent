# Requirements: Kafka Migration — Thay NATS JetStream bằng Kafka

> Date: 2026-04-15
> Phase: kafka_migration
> Triggered by: NATS thiếu monitoring nghiêm trọng, stream conflict khi restart, không có UI quản lý queue

## Lý do chuyển

### NATS đã gặp vấn đề thực tế
1. Stream conflict (`subjects overlap`) khi Debezium + Worker cùng tạo stream → phải xoá thủ công
2. Không có UI xem messages trong stream, consumer lag, throughput
3. Mỗi lần NATS restart → Debezium connection closed → crash
4. ACL phải config `$JS.API.>` + `_INBOX.>` cho mỗi user — dễ miss
5. Debug pipeline = đọc log từng container, không có cái nhìn tổng thể
6. Với 5M-100M records, không thể debug thủ công mỗi lần lỗi

### Kafka giải quyết được
1. **Kafka UI** (Kafdrop/AKHQ/Redpanda Console): xem topics, messages, consumer groups, lag — realtime
2. **Consumer groups native**: rebalance tự động, lag tracking built-in
3. **Schema Registry**: validate message format, compatibility checks
4. **Debezium native**: Debezium sinh ra cho Kafka, không cần workaround
5. **Connector ecosystem**: Kafka Connect cho Postgres sink (không cần custom code)
6. **Production proven**: dùng rộng rãi cho CDC ở scale lớn

## Yêu cầu

### R1: Thay NATS JetStream bằng Kafka cho CDC event streaming
- Debezium → Kafka topics (native, không cần Debezium Server)
- CDC Worker consume từ Kafka thay NATS

### R2: Giữ NATS cho internal commands
- `cdc.cmd.*` (bridge, transform, scan) — vẫn dùng NATS (nhẹ, đơn giản)
- `schema.config.reload` — vẫn dùng NATS
- Chỉ thay tầng CDC event streaming

### R3: Kafka monitoring UI
- Kafdrop hoặc AKHQ — xem topics, messages, consumer groups, lag
- Truy cập qua browser

### R4: Debezium chuyển về Kafka Connect mode
- Dùng Debezium Kafka Connector (native) thay Debezium Server
- Cần Kafka Connect cluster

### R5: Worker consumer chuyển từ NATS pull sang Kafka consumer group

## Scope thay đổi

### Infra mới cần thêm
- Kafka broker (có thể dùng KRaft mode — không cần ZooKeeper từ Kafka 3.3+)
- Kafka Connect (cho Debezium connector)
- Kafka UI (Kafdrop hoặc AKHQ)
- Schema Registry (optional, recommended)

### Code cần sửa
- Worker: thay NATS consumer → Kafka consumer
- Worker: EventHandler giữ nguyên (chỉ đổi transport layer)
- docker-compose: thêm Kafka + Kafka Connect + UI containers

### Code KHÔNG đổi
- DynamicMapper, BatchBuffer, SchemaInspector — giữ nguyên
- CMS Service — giữ NATS cho commands
- FE — giữ nguyên
- PostgreSQL — giữ nguyên

## Câu hỏi cho User

### Q1: Kafka mode nào?
- A) KRaft (Kafka 3.3+, không cần ZooKeeper) — recommended cho mới
- B) Classic (ZooKeeper + Kafka) — legacy nhưng stable

### Q2: Kafka UI nào?
- A) Kafdrop — đơn giản, lightweight
- B) AKHQ — nhiều feature hơn (topic management, consumer groups, schema registry)
- C) Redpanda Console — modern UI, free

### Q3: Schema Registry?
- A) Có — Confluent Schema Registry (validate Debezium events)
- B) Không — giữ JSON schemaless như hiện tại
