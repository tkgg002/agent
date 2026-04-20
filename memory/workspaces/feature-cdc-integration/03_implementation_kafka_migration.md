# Implementation: Kafka Migration

> Date: 2026-04-15
> Phase: kafka_migration
> Status: T1 docker-compose done, T2-T7 pending

## T1: Docker Compose — Kafka Stack (DONE)

### Containers thêm mới
| Container | Image | Port | RAM config | Purpose |
|:----------|:------|:-----|:-----------|:--------|
| gpay-kafka | confluentinc/cp-kafka:7.6.0 | 19092 | 512MB max | KRaft broker (no ZooKeeper) |
| gpay-schema-registry | confluentinc/cp-schema-registry:7.6.0 | 18081 | 256MB max | Avro schema validation |
| gpay-kafka-connect | confluentinc/cp-kafka-connect:7.6.0 | 18083 | 512MB max | Debezium connector runtime |
| gpay-redpanda-console | redpandadata/console:latest | 18088 | ~128MB | Monitoring UI |

### Kafka KRaft config
- Single broker (dev mode)
- `KAFKA_NODE_ID: 1`
- `KAFKA_PROCESS_ROLES: broker,controller`
- Auto-create topics enabled
- Log retention: 168h (7 days)
- Replication factor: 1 (dev, tăng lên 3 cho prod)

### Schema Registry config
- Bootstrap: `kafka:9092`
- Listener: `http://0.0.0.0:8081`

### Kafka Connect config
- Bootstrap: `kafka:9092`
- Converter: Avro (key + value)
- Schema Registry URL: `http://schema-registry:8081`
- Plugin: `debezium/debezium-connector-mongodb:2.5.4` (installed at startup)
- Group: `cdc-connect-group`

### Redpanda Console config
- Kafka brokers: `kafka:9092`
- Schema Registry: `http://schema-registry:8081`
- Kafka Connect: `http://kafka-connect:8083`

## T2: Debezium MongoDB Connector

### Connector config (deploy via REST API)
```json
POST http://localhost:18083/connectors
{
  "name": "goopay-mongodb-cdc",
  "config": {
    "connector.class": "io.debezium.connector.mongodb.MongoDbConnector",
    "mongodb.connection.string": "mongodb://mongodb:27017/?replicaSet=rs0",
    "topic.prefix": "cdc.goopay",
    "database.include.list": "payment-bill-service",
    "capture.mode": "change_streams_update_full",
    "snapshot.mode": "initial",
    "key.converter": "io.confluent.connect.avro.AvroConverter",
    "key.converter.schema.registry.url": "http://schema-registry:8081",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "http://schema-registry:8081",
    "mongodb.poll.interval.ms": 1000,
    "max.batch.size": 2048,
    "max.queue.size": 8192
  }
}
```

### Topics tự tạo bởi Debezium
- `cdc.goopay.payment-bill-service.payment-bills`
- `cdc.goopay.payment-bill-service.refund-requests`
- `cdc.goopay.payment-bill-service.payment-bill-histories`
- ... (1 topic per collection)

## T3: Worker Kafka Consumer

### Library: segmentio/kafka-go
- Lightweight, pure Go
- Consumer group support
- No CGO dependency (unlike confluent-kafka-go)

### Implementation plan
File: `internal/handler/kafka_consumer.go`

```go
type KafkaConsumer struct {
    reader  *kafka.Reader
    handler *EventHandler
    logger  *zap.Logger
}

// Reader config:
// - Brokers: ["kafka:9092"]
// - GroupID: "cdc-worker-group"
// - Topic pattern: "cdc.goopay.*"
// - MinBytes: 10KB, MaxBytes: 10MB
// - CommitInterval: 1s
```

### Avro deserialization
- Dùng Schema Registry client để fetch schema
- Deserialize Avro → map[string]interface{} → EventHandler.Handle()
- Library: `github.com/linkedin/goavro/v2`

### Config thêm
```yaml
# config-local.yml
kafka:
  brokers: ["localhost:19092"]
  groupId: cdc-worker-group
  topicPrefix: cdc.goopay
  schemaRegistryUrl: http://localhost:18081
```

## T4: NATS giữ commands
- Không thay đổi code NATS
- Xoá NATS JetStream streams CDC: `nats stream rm CDC_EVENTS`
- Giữ: cdc.cmd.*, schema.config.reload

## T5: Redpanda Console
- URL: http://localhost:18088
- Features: topics list, message browser, consumer groups, lag, schema registry viewer

## T6: E2E Test
1. Start Kafka stack: `docker compose up -d kafka schema-registry kafka-connect redpanda-console`
2. Wait Kafka Connect ready: `curl http://localhost:18083/connectors`
3. Deploy Debezium connector: `curl -X POST http://localhost:18083/connectors -d @connector.json`
4. Insert MongoDB: `mongosh --eval 'db.collection.insertOne({...})'`
5. Check Redpanda Console: topic has message
6. Start Worker: messages consumed, PostgreSQL has record
7. Check consumer lag = 0

## T7: Cleanup
- Stop gpay-debezium container
- Remove NATS CDC streams
- Update CLAUDE.md, 00_current.md, 07_technical_architecture_review.md
