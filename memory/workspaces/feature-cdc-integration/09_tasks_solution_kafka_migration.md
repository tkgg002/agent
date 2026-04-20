# Solution: Kafka Migration — Technical Details

> Date: 2026-04-15
> Phase: kafka_migration

## Files mới cần tạo

| File | Purpose |
|:-----|:--------|
| `deployments/kafka/connector-mongodb.json` | Debezium MongoDB connector config |
| `internal/handler/kafka_consumer.go` | Go Kafka consumer (replace NATS CDC consumer) |
| `internal/handler/avro_deserializer.go` | Avro → map[string]interface{} |

## Files cần sửa

| File | Thay đổi |
|:-----|:---------|
| `docker-compose.yml` | Thêm 4 containers Kafka stack (DONE) |
| `config/config.go` | Thêm KafkaConfig struct |
| `config/config-local.yml` | Thêm kafka section |
| `internal/server/worker_server.go` | Start KafkaConsumer thay vì NATS ConsumerPool cho CDC events |
| `go.mod` | Thêm kafka-go, goavro dependencies |

## Files KHÔNG đổi

| File | Lý do |
|:-----|:------|
| `internal/handler/event_handler.go` | EventHandler nhận map[string]interface{} — transport agnostic |
| `internal/service/dynamic_mapper.go` | Mapping logic không phụ thuộc transport |
| `internal/handler/batch_buffer.go` | Batch upsert PostgreSQL giữ nguyên |
| `internal/handler/command_handler.go` | NATS commands giữ nguyên |
| `internal/service/registry_service.go` | Cache giữ nguyên |

## Debezium connector JSON

```json
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
    "mongodb.poll.interval.ms": 1000
  }
}
```

## Worker startup flow (sau migration)

```go
// worker_server.go Start()
func (s *WorkerServer) Start() error {
    // 1. NATS commands (giữ nguyên)
    s.natsSubscribeCommands()
    
    // 2. Kafka CDC consumer (MỚI — thay NATS ConsumerPool)
    kafkaConsumer := handler.NewKafkaConsumer(s.cfg.Kafka, s.eventHandler, s.logger)
    go kafkaConsumer.Start(context.Background())
    
    // 3. Schedule-driven executor (giữ nguyên)
    go s.startScheduler()
    
    // 4. HTTP server (giữ nguyên)
    return s.app.Listen(s.cfg.Server.Port)
}
```
