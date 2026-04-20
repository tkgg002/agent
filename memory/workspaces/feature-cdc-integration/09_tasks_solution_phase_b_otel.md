# Solution: Phase B (Redesign) + OpenTelemetry — Continuation Guide

> Date: 2026-04-15
> Status: Phase A bugs fixed (A1-A3). Phase B + OTel pending.

## Phase A Results (DONE)
- A1: Registry export-jobs PK fixed (`_id`). EventHandler no longer normalizes PK.
- A2: CDCEvent.Source → interface{}. extractSourceAndTable handles both string/object.
- A3: `unwrapMongoTypes()` in DynamicMapper — handles `{$oid}` → string, `{$date}` → time.Time
- Avro: schema name sanitize + union unwrap done.
- Build OK. Cần restart Worker + test.

## Phase B: Redesign (PENDING)
Directories created: `internal/transport/`, `pkgs/kafka/`

### Steps
1. Move `kafka_consumer.go` → `internal/transport/kafka_consumer.go`
2. Move Avro helpers → `pkgs/kafka/avro.go`
3. Move `consumer_pool.go` → `internal/transport/nats_consumer.go`
4. Split `command_handler.go` (900 lines) → 4 services:
   - `service/bridge_service.go`
   - `service/transform_service.go`
   - `service/scan_service.go`
   - `service/schema_service.go`
5. Create `handler/command_dispatcher.go` — thin NATS → service routing
6. Slim `server/worker_server.go` < 100 lines

### Key: EventHandler stays in handler/, services in service/
Transport → Handler → Service → Repository

## OpenTelemetry (PENDING)
### Pattern from centralized-export-service
- SigNoz endpoint: `http://localhost:4318` (OTLP HTTP)
- Traces + Metrics + Logs

### Go libraries needed
```
go.opentelemetry.io/otel
go.opentelemetry.io/otel/sdk/trace
go.opentelemetry.io/otel/sdk/metric
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp
go.opentelemetry.io/contrib/instrumentation/gorm.io/gorm/otelgorm
```

### Instrumentation points
- Kafka consumer: span per message
- EventHandler: span per event
- BatchBuffer: span per upsert
- GORM: auto-instrument
- Bridge/Transform: span per operation

### Config
```yaml
otel:
  enabled: true
  serviceName: cdc-worker
  endpoint: http://localhost:4318
  sampleRatio: 1.0
```

### File to create: `pkgs/observability/otel.go`
Init trace provider + metric provider + OTLP exporters. Call from main.go.
