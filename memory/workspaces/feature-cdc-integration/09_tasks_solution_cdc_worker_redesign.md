# Solution: CDC Worker Redesign — Phase A + B + OpenTelemetry

> Date: 2026-04-15
> 3 Tasks liên tục: Phase A (fix bugs) → Phase B (redesign) → OpenTelemetry

---

## Task 1: Phase A — Fix bugs

### A1: export_jobs PK mismatch
- Table `export_jobs` có column `_id`, không có `id`
- Registry `primary_key_field = id` → EventHandler normalize `_id` → `id` → column not found
- **Fix**: EventHandler KHÔNG normalize PK khi table đã có `_id` column. Hoặc: fix registry `primary_key_field = _id`
- **Quick fix**: UPDATE registry SET primary_key_field = '_id' WHERE source_table = 'export-jobs'
- **Proper fix**: EventHandler check target table column exists trước khi normalize

### A2: MongoDB ObjectId chưa extract
- `_id = 'map[$oid:69819...]'` → Go fmt.Sprintf của map, không phải ObjectId string
- DynamicMapper `convertType` cần handle MongoDB `{"$oid": "..."}` → extract string
- **Fix**: Trong MapData, khi gặp `_id` field với value `map[$oid:xxx]` → extract `xxx`

### A3: Debezium `after` chứa MongoDB date format
- `createdAt = 'map[$date:1.77e+12]'` → cần extract timestamp
- **Fix**: convertType TIMESTAMP handle `{"$date": epoch_ms}` (đã có, cần verify)

---

## Task 2: Phase B — Clean Architecture Redesign

### Cấu trúc mới
```
internal/
├── transport/           — Transport layer
│   ├── nats_commands.go
│   ├── nats_consumer.go (legacy)
│   ├── kafka_consumer.go
│   └── http_server.go
├── handler/             — Thin dispatch
│   ├── cdc_event_handler.go
│   └── command_dispatcher.go
├── service/             — Business logic
│   ├── bridge_service.go
│   ├── transform_service.go
│   ├── scan_service.go
│   ├── schema_service.go
│   ├── dynamic_mapper.go (giữ)
│   ├── enrichment_service.go (giữ)
│   ├── registry_service.go (giữ)
│   └── activity_logger.go (giữ)
├── repository/ (giữ)
├── model/ (giữ)
└── server/worker_server.go (slim)
```

### Split command_handler.go 900 lines → 4 services
- `bridge_service.go`: HandleAirbyteBridge, bridgeInPlace, ensureCDCColumns
- `transform_service.go`: HandleBatchTransform, buildCastExpr
- `scan_service.go`: HandleScanRawData, HandlePeriodicScan
- `schema_service.go`: HandleStandardize, HandleDiscover, HandleCreateDefaultColumns, HandleDropGINIndex, HandleBackfill

### Move kafka_consumer.go → transport/
- + Avro helpers → pkgs/kafka/avro.go

---

## Task 3: OpenTelemetry + SigNoz

### Pattern từ centralized-export-service
- SigNoz endpoint: `http://localhost:4318` (OTLP HTTP)
- 3 signals: Traces + Metrics + Logs
- Go equivalent libraries:
  - `go.opentelemetry.io/otel` — core
  - `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` — trace exporter
  - `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp` — metric exporter
  - `go.opentelemetry.io/otel/sdk/trace` — trace provider
  - `go.opentelemetry.io/otel/sdk/metric` — metric provider

### Instrumentation points
- Kafka consumer: span per message consumed
- EventHandler: span per event processed
- DynamicMapper: span per MapData call
- BatchBuffer: span per batch upsert
- Bridge/Transform: span per operation
- DB queries: auto-instrument GORM

### Config
```yaml
otel:
  enabled: true
  serviceName: cdc-worker
  endpoint: http://localhost:4318
  sampleRatio: 1.0
```
