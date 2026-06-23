# 01_structure_review.md — CDS Project Structure (Reviewed 2026-06-18)

## Thống kê chính xác (sau review)

| Metric | Giá trị |
|---|---|
| Total Go files | **191** |
| Production Go files (non-test) | **141** |
| Test Go files (test/) | **50** |
| External entrypoints (cmd/) | **3** |
| Build artifacts (root: worker, admin-api) | **2 binary files** |

---

## Cây thư mục đầy đủ (verified)

```
centralized-data-service/
├── Makefile                          — Build, test, migrate, infra targets
├── README.md
├── docker-compose.yml
├── go.mod / go.sum
├── test_output.log                   — ⚠️ Log file checked-in (should gitignore)
│
├── admin-api                         — ⚠️ Compiled binary (root level)
├── worker                            — ⚠️ Compiled binary (root level)
│
├── bin/                              — Build output directory
│
├── cmd/                              — Entrypoints
│   ├── worker/main.go                — CDC Worker (Kafka + NATS + cron)
│   ├── sinkworker/main.go            — Kafka Sink Worker
│   └── admin-api/main.go             — Admin REST API
│
├── config/                           — Config layer
│   ├── config.go                     — AppConfig struct + NewConfig() (viper)
│   ├── config-local.yml
│   ├── config-production.yml
│   └── config-sample.yml
│
├── internal/                         — Private application logic
│   ├── activity/
│   │   └── taxonomy.go               — Activity type enums
│   ├── admin/                        — Admin HTTP server (Gin)
│   │   ├── server.go
│   │   ├── helpers.go
│   │   ├── source_register.go
│   │   └── types.go
│   ├── handler/                      — NATS handlers + Kafka consumer (24 files)
│   │   ├── command_handler.go        — ⭐ Core NATS command handler (3437 lines)
│   │   ├── kafka_consumer.go         — Kafka consumer + adaptive batching
│   │   ├── recon_handler.go          — Reconciliation handler
│   │   ├── recon_heal_v4.go          — Healing V4 logic
│   │   ├── snapshot_runner_handler.go — MongoDB snapshot runner
│   │   ├── provisioning_step_handlers.go — Provisioning wizard steps
│   │   ├── provisioning_handler.go
│   │   ├── provisioning_emit.go
│   │   ├── transmute_handler.go
│   │   ├── master_ddl_handler.go
│   │   ├── dlq_handler.go
│   │   ├── dlq_state_machine.go
│   │   ├── dlq_circuit_breaker.go
│   │   ├── event_handler.go
│   │   ├── event_bridge.go
│   │   ├── batch_buffer.go
│   │   ├── consumer_pool.go
│   │   ├── action_trace.go
│   │   └── [5 test files: *_test.go]
│   ├── model/                        — GORM entities (18 files)
│   │   ├── table_registry.go         — ⭐ Main registry model
│   │   ├── shadow_binding.go
│   │   ├── master_binding.go
│   │   ├── mapping_rule.go + mapping_rule_v2.go
│   │   ├── source_object_registry.go
│   │   ├── reconciliation_report.go
│   │   ├── failed_sync_log.go
│   │   ├── pending_field.go
│   │   ├── activity_log.go
│   │   ├── sensitive_field.go
│   │   ├── schema_change_log.go
│   │   ├── sync_runtime_state.go
│   │   ├── transmute_schedule.go
│   │   ├── worker_schedule.go
│   │   ├── connection_registry.go
│   │   ├── cdc_event.go
│   │   └── snapshot_dlq.go
│   ├── naming/
│   │   └── naming.go                 — Table/connector naming conventions
│   ├── repository/                   — Data access layer (11 files)
│   │   ├── registry_repo.go
│   │   ├── mapping_rule_repo.go + mapping_rule_v2_repo.go
│   │   ├── shadow_binding_repo.go
│   │   ├── master_binding_repo.go
│   │   ├── source_object_registry_repo.go
│   │   ├── pending_field_repo.go
│   │   ├── connection_registry_repo.go
│   │   ├── sync_runtime_state_repo.go
│   │   ├── schema_log_repo.go
│   │   └── transmute_schedule_repo.go
│   ├── server/
│   │   └── worker_server.go          — ⭐ DI root: wire all deps (1245 lines)
│   ├── service/                      — Business logic (48 files)
│   │   ├── recon_core.go             — ⭐ Reconciliation engine (1900 lines)
│   │   ├── transmuter.go             — Shadow → Master transform (31K)
│   │   ├── metadata_registry_service.go — In-memory registry cache (31K)
│   │   ├── provisioning_orchestrator.go — Multi-step provisioning (30K)
│   │   ├── master_ddl_generator.go   — DDL generation with type safety
│   │   ├── masking_service.go        — Data masking (4 strategies)
│   │   ├── recon_source_agent.go     — Source data reader
│   │   ├── recon_dest_agent.go       — Dest data reader
│   │   ├── recon_heal.go             — Heal logic
│   │   ├── schema_adapter.go         — Event payload coercion
│   │   ├── schema_validator.go       — Schema validation
│   │   ├── schema_inspector.go       — Live schema drift detection
│   │   ├── debezium_signal.go        — Debezium signal API client
│   │   ├── dynamic_mapper.go         — Field mapping engine
│   │   ├── backfill_source_ts.go     — Timestamp backfill
│   │   ├── partition_dropper.go      — Old partition cleanup
│   │   ├── wal_monitor.go            — WAL replication slot monitor
│   │   ├── timestamp_detector.go     — MongoDB timestamp field detection
│   │   ├── transform_registry.go     — Named transform functions
│   │   ├── type_resolver.go          — SQL type validation/enum resolver
│   │   ├── transmute_scheduler.go    — Cron scheduler for transmute
│   │   ├── mongo_introspection.go    — MongoDB schema discovery
│   │   ├── child_explode.go          — Nested JSON → flat columns
│   │   ├── activity_logger.go
│   │   ├── dlq_worker.go
│   │   ├── full_count_aggregator.go
│   │   ├── job_monitor.go
│   │   ├── text_sanitizer.go
│   │   ├── source_router.go
│   │   ├── enrichment_service.go
│   │   ├── connection_manager.go
│   │   ├── bridge_service.go
│   │   └── transmute/                — Sub-package: pure transmute strategies
│   │       ├── strategy.go           — Strategy interface + registry
│   │       ├── copy_1_to_1.go        — 1:1 field copy strategy
│   │       ├── flatten.go            — JSON flatten strategy
│   │       └── strategy_test.go
│   └── sinkworker/                   — Kafka sink processor (4 files)
│       ├── sinkworker.go             — Main consumer loop
│       ├── schema_manager.go         — Shadow schema management
│       ├── envelope.go               — Debezium envelope parser
│       └── upsert.go                 — Shadow table UPSERT logic
│
├── pkgs/                             — Shared infrastructure packages
│   ├── crypto/aes.go                 — AES encryption
│   ├── database/                     — DB connection factory (7 files)
│   ├── idgen/sonyflake.go            — Distributed ID generator
│   ├── kafka/avro.go                 — Avro encoder/decoder
│   ├── metrics/                      — Prometheus metrics (http.go + prometheus.go)
│   ├── mongodb/client.go             — MongoDB client factory
│   ├── natsconn/nats_client.go       — NATS connection factory
│   ├── observability/                — OTel (traces+logs+metrics) — 3 files
│   ├── rediscache/redis_client.go    — Redis client factory
│   └── utils/                        — hash.go + type_inference.go
│
├── migrations/
│   └── dest/001_dest_init.sql        — Destination DB init schema
│
├── test/                             — External test suite (50 files)
│   ├── config/config_test.go
│   ├── internal/
│   │   ├── handler/                  — 11 test files (unit + integration)
│   │   ├── service/                  — 19 test files
│   │   ├── admin/                    — 2 test files
│   │   └── sinkworker/               — 2 test files
│   └── pkgs/                         — 7 test files
│
├── deployments/                      — Infrastructure configs
│   ├── debezium/                     — Debezium connector configs
│   ├── docker/                       — Dockerfiles
│   ├── k8s/                          — Kubernetes manifests
│   ├── kafka/                        — Kafka configs
│   ├── mariadb/                      — MariaDB configs
│   ├── nats/                         — NATS configs
│   ├── prometheus/                   — Prometheus scrape config
│   ├── redpanda/                     — Redpanda (Kafka alternative) configs
│   ├── runbooks/                     — Operational runbooks
│   ├── sql/                          — SQL scripts
│   └── otel-collector-config.yml     — OTel Collector pipeline config
│
├── scripts/                          — Test & chaos scripts
│   ├── load_test.js                  — k6 load test
│   ├── load_test_cdc.js              — k6 CDC-specific load test
│   ├── soak_test.sh                  — Long-running soak test
│   ├── smoke_failover.sh             — Failover smoke test
│   └── chaos_network.sh              — Network chaos injection
│
├── scratch/                          — ⚠️ Debug scripts (should gitignore)
│   ├── debug_registry/debug_registry.go
│   ├── query_db/
│   └── test_api_response/
│
└── docs/
    └── runbooks/                     — Documentation runbooks
```

---

## ⚠️ Phát hiện trong Review

| # | Issue | Severity | Ghi chú |
|---|---|---|---|
| 1 | `admin-api` và `worker` — compiled binaries ở **root dir** | Medium | Nên ở `bin/` hoặc gitignore |
| 2 | `test_output.log` checked-in vào repo | Low | Nên gitignore |
| 3 | `scratch/` checked-in vào repo | Low | Debug scripts — nên gitignore |
| 4 | Tests được tổ chức tách biệt trong `test/` (không co-located) | Note | Convention tốt cho integration tests nhưng khác Go idiom |
| 5 | `command_handler.go` = 3,437 dòng — God object | High | Cần xem xét tách nhỏ theo domain |
| 6 | `internal/service/transmute/` — sub-package nhỏ (4 files) | Note | Strategy pattern tốt |
| 7 | Chỉ có **1 migration file** cho dest DB | Note | Schema được quản lý ở đâu? |
| 8 | `mapping_rule.go` + `mapping_rule_v2.go` song song | Medium | V1 có được deprecated chưa? |
