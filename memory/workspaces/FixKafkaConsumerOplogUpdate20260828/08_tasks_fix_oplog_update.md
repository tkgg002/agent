# Tasks Checklist: Fix Kafka Consumer Oplog UPDATE Event Processing

- [x] **Task 1: Root Cause Analysis & Log Audit**
  - [x] Trace `kafka_consumer.go` drop condition (`if afterData == nil && opStr != "d"`).
  - [x] Trace `cdc_system.sources` Debezium configuration (`"capture.mode": "change_streams"`).
  - [x] Trace `event_handler.go` `after` validation.

- [ ] **Task 2: Update Debezium Mongo Connector Configuration**
  - [ ] Set `"capture.mode": "change_streams_with_full_update"` in `cdc_system.sources` records.
  - [ ] Update running Debezium Mongo connectors via Debezium REST API / CMS connector recover.

- [ ] **Task 3: Code Enhancement (Propose to User)**
  - [ ] Enhancement in `internal/handler/shadow/kafka_consumer.go` to handle `op == "u"` gracefully.
  - [ ] Enhancement in `internal/handler/shadow/event_handler.go` for update fallback.

- [ ] **Task 4: Verification & E2E Testing**
  - [ ] Verify unit tests pass in `centralized-data-service`.
  - [ ] Audit DB Shadow (`cdc_shadow`) to verify UPDATE events are properly ingested.
