# Plan: CDC Worker Redesign — Clean Architecture + Kafka Fix

> Date: 2026-04-15
> Role: Brain
> Status: PLANNING — chờ duyệt trước khi Muscle code

## Vấn đề hiện tại

### 1. Code structure lộn xộn
```
internal/handler/
├── batch_buffer.go        — DB batch write
├── bridge_batch.go        — pgx bridge (nên ở service layer)
├── command_handler.go     — 900+ lines, god object
├── consumer_pool.go       — NATS consumer (legacy)
├── dlq_handler.go         — DLQ retry logic
├── event_bridge.go        — PG triggers → NATS (nên ở service)
├── event_handler.go       — CDC event processing
├── kafka_consumer.go      — Kafka consumer (mới, chưa test)
```
- `command_handler.go` 900+ lines = god object, chứa bridge/transform/scan/discover/standardize/backfill
- `handler/` chứa cả transport layer (NATS/Kafka) lẫn business logic
- Không tách rõ transport ↔ service ↔ repository

### 2. Bugs cần fix NGAY
- Avro schema name `centralized-export-service` chứa dấu `-` → goavro reject
- EventHandler `CDCEvent.source` expect string nhưng Debezium Kafka gửi object
- Data cũ trong Kafka (từ snapshot) chưa consume được → data đích thiếu
- Mapping rules chưa có cho collection `export-jobs` (registry source_table mismatch?)

### 3. worker_server.go quá tải
- 300+ lines
- Chứa: init infra + NATS subscribe + Kafka consumer + schedule executor + partition check + HTTP server
- Khó maintain, khó test

## Cấu trúc đề xuất (Clean Architecture)

```
centralized-data-service/
├── cmd/worker/main.go                        — Entry point (nhẹ, chỉ init + start)
├── config/config.go                          — Config loading
│
├── internal/
│   ├── transport/                            — Transport layer (NATS, Kafka, HTTP)
│   │   ├── nats_commands.go                  — NATS command subscribers (cdc.cmd.*)
│   │   ├── nats_consumer.go                  — NATS CDC consumer (legacy, phase out)
│   │   ├── kafka_consumer.go                 — Kafka CDC consumer + Avro deserialize
│   │   └── http_server.go                    — Health + metrics endpoints
│   │
│   ├── handler/                              — Event processing (thin, delegate to service)
│   │   ├── cdc_event_handler.go              — Parse CDC event → call service
│   │   └── command_dispatcher.go             — Route NATS commands → service methods
│   │
│   ├── service/                              — Business logic (core)
│   │   ├── bridge_service.go                 — Bridge Airbyte → CDC (từ command_handler)
│   │   ├── transform_service.go              — Transform _raw_data → typed columns
│   │   ├── scan_service.go                   — Field scan + periodic scan
│   │   ├── schema_service.go                 — Standardize + Discover + Schema Inspector
│   │   ├── dynamic_mapper.go                 — Field mapping logic (giữ nguyên)
│   │   ├── enrichment_service.go             — Computed fields (giữ nguyên)
│   │   ├── registry_service.go               — In-memory cache (giữ nguyên)
│   │   └── activity_logger.go                — Activity log (giữ nguyên)
│   │
│   ├── repository/                           — Data access (giữ nguyên)
│   │   ├── mapping_rule_repo.go
│   │   ├── registry_repo.go
│   │   └── pending_field_repo.go
│   │
│   ├── model/                                — Domain models (giữ nguyên)
│   │   ├── cdc_event.go
│   │   ├── table_registry.go
│   │   ├── mapping_rule.go
│   │   └── ...
│   │
│   └── server/
│       └── worker_server.go                  — Init + wire dependencies + start (slim)
│
├── pkgs/                                     — Shared packages (giữ nguyên)
│   ├── database/, natsconn/, rediscache/
│   ├── idgen/, metrics/, utils/
│   └── kafka/                                — NEW: Kafka client + Avro helpers
│       ├── consumer.go                       — Kafka consumer wrapper
│       ├── avro.go                           — Schema Registry client + Avro decode
│       └── config.go                         — Kafka config
│
└── test/                                     — Tests
```

### Lợi ích
1. **Tách transport ↔ logic**: Kafka/NATS là transport, không chứa business logic
2. **Tách command_handler 900 dòng** → 4 services (bridge, transform, scan, schema)
3. **Testable**: Service layer test được mà không cần NATS/Kafka running
4. **Avro xử lý riêng**: `pkgs/kafka/avro.go` — schema name sanitize + decode

## Bugs cần fix TRƯỚC khi redesign

### Bug 1: Avro schema name chứa dấu `-`
```
centralized-export-service → Avro reject
```
**Fix**: Sanitize schema name — replace `-` với `_` khi create Avro codec

### Bug 2: CDCEvent.source expect string
```
cannot unmarshal object into Go struct field CDCEvent.source of type string
```
**Fix**: Đổi `CDCEvent.Source` từ `string` sang `interface{}` hoặc `json.RawMessage`

### Bug 3: Registry source_table mismatch cho export-jobs
**Check**: `cdc_table_registry` có entry cho `export-jobs`? `source_table` = gì? `sync_engine` = gì?

### Bug 4: Data cũ (Debezium snapshot) chưa consume
**Check**: Kafka consumer `StartOffset: kafka.FirstOffset` — nên consume từ đầu. Nhưng nếu Avro fail → skip → data mất.

## Execution order

```
Phase A: Fix bugs NGAY (không redesign)
  A1: Fix Avro schema name sanitize
  A2: Fix CDCEvent.source type
  A3: Verify registry export-jobs entry
  A4: Test: MongoDB insert → Kafka → Worker → Postgres
  A5: Verify data cũ (snapshot) consumed

Phase B: Redesign (sau khi data flow OK)
  B1: Tạo internal/transport/ — move NATS/Kafka code
  B2: Tạo internal/service/ — split command_handler
  B3: Tạo pkgs/kafka/ — Avro + consumer wrapper
  B4: Slim worker_server.go
  B5: Tests
```

## Definition of Done

### Phase A (bugs)
- [ ] MongoDB insert → Postgres trong < 5 giây
- [ ] Data cũ (snapshot) đã consume hết
- [ ] Activity Log ghi mỗi Kafka event processed
- [ ] Redpanda Console: consumer lag = 0

### Phase B (redesign)
- [ ] command_handler.go < 100 lines (chỉ dispatch)
- [ ] worker_server.go < 100 lines (chỉ init + wire)
- [ ] Mỗi service < 200 lines
- [ ] Unit tests cho từng service
- [ ] All builds pass
