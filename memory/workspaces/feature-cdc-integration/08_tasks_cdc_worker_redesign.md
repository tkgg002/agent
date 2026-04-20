# Tasks: CDC Worker Redesign

> Date: 2026-04-15

## Phase A: Fix bugs (NGAY)
- [ ] A1: Avro schema name sanitize (replace `-` → `_`)
- [ ] A2: CDCEvent.source type fix (string → interface{})
- [ ] A3: Verify registry export-jobs entry
- [ ] A4: E2E test: MongoDB insert → Kafka → Worker → Postgres
- [ ] A5: Verify snapshot data consumed

## Phase B: Redesign
- [ ] B1: internal/transport/ — move NATS/Kafka
- [ ] B2: internal/service/ — split command_handler
- [ ] B3: pkgs/kafka/ — Avro + consumer
- [ ] B4: Slim worker_server.go
- [ ] B5: Tests
