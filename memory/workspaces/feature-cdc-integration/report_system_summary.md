# Báo cáo Tổng kết Hệ thống CDC

**Ngày**: 2026-04-30 23:34 (Asia/Ho_Chi_Minh)
**Phạm vi**: Toàn bộ hệ thống CDC — kiến trúc, cdc-worker, cms-service, mọi API, mọi luồng
**Phương pháp**: Khảo sát thực tế codebase + DB + container (3 Explore subagents song song) — KHÔNG đoán mò
**Trạng thái services tại thời điểm report**: CMS PID 13653, Worker PID 20450 (cả hai chạy từ Wed04PM), 11/11 Docker container `Up`

---

## 0. Mục lục

- [1. Kiến trúc tổng thể](#1-kiến-trúc-tổng-thể)
- [2. cdc-worker (centralized-data-service)](#2-cdc-worker-centralized-data-service)
  - 2.1 Entry point + boot wiring
  - 2.2 NATS subscriptions (in / out)
  - 2.3 Kafka consumer
  - 2.4 Handlers
  - 2.5 Services
  - 2.6 Provisioning state machine
  - 2.7 REST endpoints
  - 2.8 Config + env override
- [3. cdc-cms-service (port 8083)](#3-cdc-cms-service)
  - 3.1 Entry + middleware chain
  - 3.2 API routes (đầy đủ)
  - 3.3 Handlers
  - 3.4 Services
  - 3.5 NATS publishes
  - 3.6 Auth/RBAC/Idempotency/Audit
  - 3.7 DB models
- [4. Tầng dữ liệu](#4-tầng-dữ-liệu)
  - 4.1 cdc_system.* (control plane — 13 bảng chính)
  - 4.2 Shadow tables (cdc_dw / shadow_*)
  - 4.3 Master DW tables (goopay_dest / dw_*)
  - 4.4 Source DBs (PG / MariaDB / MongoDB)
  - 4.5 Debezium connectors
  - 4.6 Kafka topics
  - 4.7 Schema Registry
- [5. Frontend cdc-cms-web](#5-frontend-cdc-cms-web)
- [6. Các luồng end-to-end](#6-các-luồng-end-to-end)
- [7. Xác minh thực tế](#7-xác-minh-thực-tế)
- [8. Khoảng trống đã biết & rủi ro](#8-khoảng-trống-đã-biết--rủi-ro)
- [9. Skills sử dụng](#9-skills-sử-dụng)

---

## 1. Kiến trúc tổng thể

### 1.1 Sơ đồ luồng dữ liệu

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ goopay_source    │  │ gpay-mariadb     │  │ gpay-mongo       │
│ (PG :5435)       │  │ (MariaDB :3307)  │  │ (Mongo :27018)   │
│ public.orders    │  │ legacy_orders    │  │ payment-bills... │
│ public.users     │  │                  │  │ refund-requests  │
│ public.payments  │  │                  │  │ ...              │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         │ pgoutput            │ binlog ROW          │ change-streams
         │                     │                     │
   ┌─────▼──────┐        ┌─────▼──────┐        ┌─────▼──────┐
   │ cdc-pg     │        │ cdc-mariadb│        │ goopay-    │
   │ -source    │        │ -source    │        │ mongodb-cdc│
   └─────┬──────┘        └─────┬──────┘        └─────┬──────┘
         │   gpay-kafka-connect:8084 (Debezium)      │
         └───────────────────┬─────────────────────────┘
                             │ Avro (Schema Registry :18081)
                  ┌──────────▼──────────┐
                  │ Kafka :19092 (KRaft)│
                  │ cdc.gpay.*          │
                  │ cdc.goopay.*        │
                  │ cdc.mariadb.*       │
                  └──────────┬──────────┘
                             │
                  ┌──────────▼──────────────────────────────┐
                  │ cdc-worker :8082 (Go/Fiber)             │
                  │  - KafkaConsumer (group cdc-worker-grp) │
                  │  - SchemaValidator (drift gate)         │
                  │  - DLQ → failed_sync_logs               │
                  │  - SinkWorker → BatchBuffer (upsert)    │
                  │  - TransmuteScheduler (cron 60s)        │
                  │  - JobMonitor (close-loop)              │
                  │  - ProvisioningOrchestrator (gated env) │
                  │  - ReconCore (Tier1/2/3)                │
                  │  - PartitionDropper                     │
                  └──┬──────────────────────┬───────────────┘
                     │ shadow upsert        │ transmute (Shadow→Master)
                     ▼                      ▼
        ┌────────────────────┐    ┌─────────────────────┐
        │ cdc_dw :5433       │    │ goopay_dest :5434   │
        │ shadow_<conn>.*    │    │ dw_<binding>.*      │
        │ cdc_system.*       │    │ master.* (optional) │
        │ (control plane)    │    │ (master tables)     │
        └────────────────────┘    └─────────────────────┘
                     │                      │
                     │  NATS :14222 (cdc.cmd.* / cdc.evt.* / cdc.result.*)
                     │
                  ┌──┴──────────────────────────────────────┐
                  │ cdc-cms-service :8083 (Go/Fiber)        │
                  │  - JWT(HS256) + RBAC + Idempotency-Key  │
                  │  - Audit reason ≥ 10 chars              │
                  │  - HealthCollector (Redis snapshot 15s) │
                  │  - AlertManager (firing/acked/silenced) │
                  │  - ProvisioningOrchestrator (CMS-side)  │
                  └────────────────┬────────────────────────┘
                                   │ HTTP REST
                    ┌──────────────▼──────────────┐
                    │ cdc-cms-web (Vite + React + │
                    │   Antd, port 5173)          │
                    └─────────────────────────────┘
```

### 1.2 Service inventory

| Service | Loại | Port | Tech stack | Chức năng |
|---|---|---|---|---|
| **cdc-worker** | Go binary | 8082 (HTTP), 9090 (Prom) | Go + Fiber + GORM + zap + nats.go + kafka-go | Data plane: ingest Kafka → shadow → master, DLQ replay, recon, provisioning step handlers, scheduler |
| **cdc-cms-service** | Go binary | 8083 | Go + Fiber + GORM + JWT + Redis + NATS | Control plane: REST API cho FE, audit, idempotency, alert manager, provisioning state |
| **cdc-cms-web** | Frontend | 5173 (dev) | Vite + React 18 + Antd + TanStack Query + axios | UI quản trị 13 page chính |
| **cdc-auth-service** | Go binary | 8081 | Go + JWT | Auth (login/refresh) — không phân tích sâu trong report này |
| **gpay-postgres-cdc** | PG container | 5433 | PostgreSQL 15 | Control plane DB (`cdc_dw` + schema `cdc_system`) + shadow tables (`shadow_*`) |
| **gpay-postgres-source** | PG container | 5435 | PostgreSQL 15 | Source DB cho test (`goopay_source.public.*`) |
| **gpay-postgres-dest** | PG container | 5434 | PostgreSQL 15 | Destination DW (`goopay_dest`) — chứa master tables |
| **gpay-postgres** | PG container | 5436 | PostgreSQL 15 | Đa mục đích (auth/cms metadata) |
| **gpay-mariadb** | MariaDB container | 3307 | MariaDB 10.11 + binlog ROW | Source MariaDB (`goopay_legacy_maria.*`) |
| **gpay-mongo** | Mongo replica | 27018 | MongoDB rs0 | Source Mongo (`payment-bill-service.*`) |
| **gpay-kafka** | Kafka KRaft | 19092 | Confluent 7.6 | Topic broker |
| **gpay-kafka-connect** | Connect | 8084 | Debezium 2.5.4 | Connector runtime (PG + Mongo plugin sẵn; **MySQL plugin còn thiếu** — xem §8) |
| **gpay-schema-registry** | Schema Registry | 18081 | Confluent 7.6 | Avro schema persistence |
| **gpay-nats** | NATS + JetStream | 14222 | NATS 2.10 | Pub/sub bus toàn worker ↔ CMS |
| **gpay-redis** | Redis | 16379 | Redis 7 | Idempotency cache + health snapshot + recon leader-election |

---

## 2. cdc-worker (centralized-data-service)

Đường dẫn: `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service`

### 2.1 Entry point + boot wiring

**`cmd/worker/main.go`** (105 dòng):
1. `config.NewConfig()` → load `config/config-local.yml` (env `CFG_PATH` override)
2. `zap.NewProduction()` (debug mode → development)
3. `idgen.Init(logger)` — Sonyflake generator
4. `observability.InitOtel(otelCfg, logger)` — OTel traces+logs → SigNoz, severity-aware sampler
5. `server.NewWorkerServer(cfg, logger)` — full DI wiring (xem 2.1.1)
6. `metrics.StartMetricsServer(ctx, 9090, logger)` — Prom HTTP độc lập (`net/http`, không Fiber)
7. `srv.Start()` — Fiber listen `cfg.Server.Port` (`:8082`)
8. SIGINT/SIGTERM → `srv.Shutdown()`

**`internal/server/worker_server.go::NewWorkerServer()`** (lines 57–496) — DI chain:

| Bước | Module | Lines |
|---|---|---|
| 1 | `database.NewRegistry` (multi-PG control-plane pool) | 63–73 |
| 1b | optional `database.NewPostgresReadReplica` | 79–91 |
| 2 | NATS connect + `EnsureStreams()` (3 JetStream stream — xem 2.2) | 107–117 |
| 3 | Redis | 119–122 |
| 4 | Repos: Registry, MappingRule, PendingField, ConnectionRegistry, SourceObjectRegistry, ShadowBinding, MappingRuleV2, SyncRuntimeState | 125–132 |
| 5 | Services: MetadataRegistry, MaskingService, ConnectionManager, SchemaInspector | 135–143 |
| 6 | SchemaAdapter + BatchBuffer | 145–148 |
| 6b | Mongo client + ReconSourceAgent + ReconDestAgent + ReconCore (Redis leader-election) | 154–178 |
| 6c | SchemaValidator | 183–185 |
| 6d | DLQHandler + DLQStateMachine | 188–191 |
| 7 | DynamicMapper + EventHandler | 192–194 |
| 8 | **ConsumerPool** (JetStream pull `cdc.goopay.>`, group `cdc-worker-group`) | 197–207 |
| 9 | NATS sub `schema.config.reload` | 210–231 |
| 10b | **CommandHandler** + 12 NATS subs | 241–261 |
| 11 | Transmuter module + 2 subs | 266–274 |
| — | TransmuteScheduler `go ... Start(ctx)` | 281 |
| — | JobMonitor `cdc.evt.transmute.completed` | 289–292 |
| — | MasterDDLHandler `cdc.cmd.master-create` | 298–300 |
| — | Provisioning subs (gated `PROVISIONING_ORCHESTRATOR_ENABLED=1`) | 306–346 |
| 10d | ReconHandler + 7 subs (chỉ khi MongoDB cấu hình) | 384–412 |
| 11 | Fiber HTTP wiring | 438–463 |
| 10e | PartitionDropper | 469–471 |

### 2.2 NATS subjects

#### 2.2.1 JetStream streams (tạo ở `EnsureStreams`)

| Stream | Subjects | Retention | MaxAge |
|---|---|---|---|
| `CDC_EVENTS` | `cdc.goopay.>` | Limits | 7d |
| `SCHEMA_DRIFT` | `schema.drift.detected` | Limits | 7d |
| `SCHEMA_CONFIG` | `schema.config.reload` | Limits | 7d |

#### 2.2.2 NATS Inbound (worker subscribes)

Toàn bộ là `nats.Conn.Subscribe` (core NATS), trừ `cdc.goopay.>` qua JetStream pull.

| Subject | Handler | File:Line |
|---|---|---|
| `schema.config.reload` | anonymous → `registrySvc.ReloadAll + redis.DeletePattern("schema:*")` | worker_server.go:210 |
| `cdc.cmd.standardize` | `CommandHandler.HandleStandardize` | command_handler.go:168 |
| `cdc.cmd.discover` | `CommandHandler.HandleDiscover` | command_handler.go:348 |
| `cdc.cmd.backfill` | `CommandHandler.HandleBackfill` | command_handler.go:610 |
| `cdc.cmd.scan-raw-data` | `CommandHandler.HandleScanRawData` | command_handler.go:771 |
| `cdc.cmd.batch-transform` | `CommandHandler.HandleBatchTransform` | command_handler.go:679 |
| `cdc.cmd.periodic-scan` | `CommandHandler.HandlePeriodicScan` | command_handler.go:857 |
| `cdc.cmd.drop-gin-index` | `CommandHandler.HandleDropGINIndex` | command_handler.go:973 |
| `cdc.cmd.create-default-columns` | `CommandHandler.HandleCreateDefaultColumns` | command_handler.go:214 |
| `cdc.cmd.scan-fields` | `CommandHandler.HandleScanFields` | command_handler.go:1161 |
| `cdc.cmd.sync-register` | `CommandHandler.HandleSyncRegister` | command_handler.go:1221 |
| `cdc.cmd.sync-state` | `CommandHandler.HandleSyncState` | command_handler.go:1280 |
| `cdc.cmd.restart-debezium` | `CommandHandler.HandleRestartDebezium` | command_handler.go:1343 |
| `cdc.cmd.alter-column` | `CommandHandler.HandleAlterColumn` | command_handler.go:1382 |
| `cdc.cmd.transmute` | `TransmuteHandler.HandleTransmute` | transmute_handler.go:153 |
| `cdc.cmd.transmute-shadow` | `TransmuteHandler.HandleTransmuteShadow` | transmute_handler.go:43 |
| `cdc.cmd.master-create` | `MasterDDLHandler.HandleMasterCreate` | master_ddl_handler.go:53 |
| `cdc.cmd.master.bind` | `MasterDDLHandler.HandleMasterCreate` (alias provisioning) | worker_server.go:331 |
| `cdc.evt.transmute.completed` | `JobMonitor.HandleCompleted` | job_monitor.go:69 |
| `cdc.evt.provisioning.step_completed` | `ProvisioningHandler.HandleStepCompleted` (gated) | provisioning_handler.go:40 |
| `cdc.cmd.shadow.bind` | `ProvisioningStepHandler.HandleShadowBind` (gated) | provisioning_step_handlers.go:102 |
| `cdc.cmd.schedule.enable` | `ProvisioningStepHandler.HandleScheduleEnable` (gated) | provisioning_step_handlers.go:350 |
| `cdc.cmd.recon-check` | `ReconHandler.HandleReconCheck` | recon_handler.go:81 |
| `cdc.cmd.recon-heal` | `ReconHandler.HandleReconHeal` | recon_handler.go:140 |
| `cdc.cmd.retry-failed` | `ReconHandler.HandleRetryFailed` | recon_handler.go:219 |
| `cdc.cmd.debezium-signal` | `ReconHandler.HandleDebeziumSignal` | recon_handler.go:270 |
| `cdc.cmd.debezium-snapshot` | `ReconHandler.HandleDebeziumSignal` (alias) | worker_server.go:409 |
| `cdc.cmd.recon-backfill-source-ts` | `ReconHandler.HandleBackfillSourceTs` | recon_handler.go:326 |
| `cdc.cmd.detect-timestamp-field` | `ReconHandler.HandleDetectTimestampField` | recon_handler.go:406 |
| `cdc.goopay.>` (JetStream pull) | `EventHandler.Handle` (ConsumerPool) | worker_server.go:197–207 |

#### 2.2.3 NATS Outbound (worker publishes)

| Subject | Publisher | File:Line | Mục đích |
|---|---|---|---|
| `cdc.cmd.transmute` | `TransmuteHandler.HandleTransmuteShadow` | transmute_handler.go:122 | Fan-out per master từ shadow hook |
| `cdc.cmd.transmute` | `TransmuteScheduler.tick` | transmute_scheduler.go:153 | Cron-driven |
| `cdc.cmd.transmute` | `WorkerServer.runTransformCycle` | worker_server.go:679 | Schedule loop |
| `cdc.cmd.scan-raw-data` | field-scan schedule | worker_server.go:628 | Per-table |
| `cdc.cmd.periodic-scan` | field-scan schedule | worker_server.go:631 | All-tables |
| `cdc.cmd.batch-transform` | `runTransformCycle` | worker_server.go:679 | Per-table |
| `cdc.cmd.master-create` | `CommandHandler.HandleDiscover` (post-discover) | command_handler.go:600 | Auto-trigger sau discover |
| `cdc.evt.transmute.completed` | `TransmuteHandler.publishCompleted` | transmute_handler.go:226 | Close-loop tới JobMonitor |
| `cdc.evt.provisioning.step_completed` | `emitStepCompleted` (provisioning_emit.go) | nhiều chỗ | Drive state machine |
| `cdc.evt.provisioning.step_completed` | `JobMonitor.bridgeScheduleEnable` | job_monitor.go:161 | Bridge transmute success → schedule_enable |
| `cdc.result.transmute` | reply | transmute_handler.go:241 | Async result |
| `cdc.result.master-create` | reply | master_ddl_handler.go:118 | Async result |
| `cdc.result.scan-fields` | reply | command_handler.go:1209 | Async |
| `cdc.result.sync-register` | reply | command_handler.go:1258 | Async |
| `cdc.result.sync-state` | reply | command_handler.go:1337 | Async |
| `cdc.result.restart-debezium` | reply | command_handler.go:1373 | Async |
| `cdc.result.alter-column` | reply | command_handler.go:1432 | Async |
| `cdc.result.recon-backfill-source-ts` | reply | recon_handler.go:373 | Async |
| `schema.drift.detected` | `SchemaInspector` | schema_inspector.go | Alert drift |
| `cdc.cmd.transmute-shadow` | SinkWorker post-ingest hook | sinkworker/sinkworker.go:217 | Post-ingest fan-out |

### 2.3 Kafka consumer

**File**: `internal/handler/kafka_consumer.go`

- **Brokers**: `[localhost:19092]` (config `kafka.brokers`)
- **Group**: `cdc-worker-group`
- **Topic prefixes** (union): `cdc.gpay`, `cdc.goopay`, `cdc.mariadb` (config `kafka.topicPrefix`)
- **Schema Registry**: `http://localhost:18081`
- **Format detection**: byte đầu `0x00` → Avro (Confluent 5-byte header), khác → JSON (line 282)
- **Avro decoder**: cache schema theo id (line 533); sanitize tên `-` → `_` (line 588)
- **Topic discovery** (lines 435–530): `ReadPartitions()` → filter theo prefix → check tên cuối có trong `MetadataRegistry.GetDebeziumTables()`
- **Schema validation** (Phase A, lines 351–358): `SchemaValidator.ValidatePayloadWithCase` → `ErrSchemaDrift` / `ErrMissingRequired`
- **CDC envelope**: bọc thành `{"source":"/kafka/<topic>","data":{"op","before","after","source_ts_ms"}}` (line 363–374)
- **Dispatch**: `eventHandler.HandleRaw(ctx, subject, cdcJSON)` (line 385)
- **DLQ write-before-ACK**: nếu xử lý fail, INSERT `failed_sync_logs` TRƯỚC khi commit Kafka offset; nếu DLQ insert fail → offset KHÔNG commit → Kafka redeliver (lines 249–264)
- **Stats**: flush mỗi 100 messages hoặc 5s vào `cdc_activity_log` (lines 159–161, 239)

**DLQ status classification**:
- error chứa `schema_drift` / `missing_required_field` → `status='pending'` (cần ops action)
- còn lại → `status='failed'` (DLQStateMachine retry)

### 2.4 Handlers (`internal/handler/`)

| File | Struct | Chức năng chính |
|---|---|---|
| `command_handler.go` | `CommandHandler` | 13 NATS handlers — toàn bộ legacy V1 lifecycle (standardize/discover/backfill/scan/batch-transform/alter-column/sync-state/restart-debezium...) |
| `transmute_handler.go` | `TransmuteHandler` | `cdc.cmd.transmute` (chạy `TransmuterModule.Run`) + `cdc.cmd.transmute-shadow` (fan-out per master); publish `cdc.evt.transmute.completed` |
| `master_ddl_handler.go` | `MasterDDLHandler` | `cdc.cmd.master-create` + `cdc.cmd.master.bind` (alias) → `MasterDDLGenerator.Apply` (CREATE TABLE + indexes + RLS); emit `step_completed` khi `provisioning=true` |
| `recon_handler.go` | `ReconHandler` | 7 NATS subs cho recon Tier1/2/3 + heal + retry DLQ + Debezium signal + backfill `_source_ts` + detect-timestamp-field |
| `provisioning_handler.go` | `ProvisioningHandler` | `cdc.evt.provisioning.step_completed` → `Orchestrator.HandleStepCompleted` |
| `provisioning_step_handlers.go` | `ProvisioningStepHandler` | `cdc.cmd.shadow.bind` (CREATE shadow + biz cols) + `cdc.cmd.schedule.enable` (UPSERT `transmute_schedule`) |
| `batch_buffer.go` | `BatchBuffer` | In-memory accumulator (default 500 rows / 2s flush) cho upserts từ EventHandler |
| `dlq_handler.go` | `DLQHandler` | Manual replay tool, không subscribe ở boot |
| `dlq_state_machine.go` | `DLQStateMachine` | Background poll `failed_sync_logs` mỗi 5 phút, replay (max 3 retries) qua original Kafka subject |
| `event_handler.go` | `EventHandler` | Core CDC processor; nhận từ ConsumerPool + KafkaConsumer; route qua MetadataRegistry → DynamicMapper → BatchBuffer |
| `consumer_pool.go` | `ConsumerPool` | JetStream PullSubscribe `cdc.goopay.>`, AckWait=30s, MaxDeliver=5 |
| `provisioning_emit.go` | helper | `emitStepCompleted` publish `cdc.evt.provisioning.step_completed` |
| `event_bridge.go` | `EventBridge` | Legacy PG-trigger polling — KHÔNG wire ở boot (giữ làm reserve) |

### 2.5 Services (`internal/service/`)

| Service | File | Mục đích chính |
|---|---|---|
| `ProvisioningOrchestrator` | `provisioning_orchestrator.go` | Drive state machine: `Advance/Pause/Resume/Retry/SetMode` + `RecoveryLoop` (1 phút quét `*_pending` quá TTL 10p) |
| `ProvisioningStateMachine` | `provisioning_state_machine.go` | Pure data: 13 states + 4 transitions |
| `TransmuterModule` | `transmuter.go` | Cursor batch shadow → master, OCC `_source_ts` guard, ON CONFLICT `_gpay_id` UPDATE |
| `TransmuteScheduler` | `transmute_scheduler.go` | Cron 60s tick, `SELECT ... FOR UPDATE SKIP LOCKED`, plant fencing GUC, publish `cdc.cmd.transmute` |
| `SchemaAdapter` | `schema_adapter.go` | Read `information_schema.columns`, prepare CDC cols, build UPSERT SQL với OCC guard |
| `MasterDDLGenerator` | `master_ddl_generator.go` | Generate CREATE TABLE + ALTER ADD COLUMN từ `mapping_rule_v2`; financial col GIN index; RLS policy |
| `MetadataRegistryService` | `metadata_registry_service.go` | V2-aware in-memory cache cho table configs + mapping rules + source routing |
| `DynamicMapper` | `dynamic_mapper.go` | Apply `cdc_mapping_rules` (transform_fn, source_path) → `MappedData{Columns, EnrichedData, RawJSON}` |
| `ReconCore` | `recon_core.go` | Tier1 (count) / Tier2 (ID set diff windowed) / Tier3 (hash) — Redis leader election, 15 phút sliding window |
| `ReconHealer` | `recon_heal.go` | v3 batched heal — fetch missing IDs từ Mongo (batch 500), upsert PG với OCC guard, prefer Debezium signal |
| `SchemaInspector` | `schema_inspector.go` | Detect schema drift, cache Redis (`schema:*`), publish `schema.drift.detected` |
| `SchemaValidator` | `schema_validator.go` | Phase A pre-flight: compare payload fields vs `cdc_table_registry.expected_fields` (fail-open bootstrap) |
| `MaskingService` | `masking_service.go` | PII masking — sensitive keywords default `phone/email/secret/password/token/balance/otp/pin/card/account/address/ssn` |
| `JobMonitor` | `job_monitor.go` | Subscribe `cdc.evt.transmute.completed` → UPDATE `transmute_schedule.last_status` + bridge first success → `step_completed` |
| `ActivityLogger` | `activity_logger.go` | Write `cdc_activity_log` (Start/Complete/Fail/Quick) |
| `PartitionDropper` | `partition_dropper.go` | Daily advisory-locked drop partitions (`failed_sync_logs_*` 90d, `cdc_activity_log_*` 30d) |
| `FullCountAggregator` | `full_count_aggregator.go` | 03:00 UTC daily — Mongo `EstimatedDocumentCount` + PG `COUNT(*)` (fast-path `pg_class.reltuples` >10M) |
| `BridgeService` | `bridge_service.go` | Legacy probe — chỉ còn `TableExists` + `HasColumn` (Sprint 4 4A.1 retire) |
| `DebeziumSignalClient` | `debezium_signal.go` | Insert incremental snapshot signal vào Mongo `debezium_signal` |
| `ConnectionManager` | `connection_manager.go` | Multi-destination PG pool registry |
| `TypeResolver` | `type_resolver.go` | BSON / MySQL / PG type → canonical PG type cho master DDL |
| `SourceRouter` | `source_router.go` | Topic → source DB+table qua `MetadataRegistry.ResolveSourceRoute` |
| `BackfillSourceTsService` | `backfill_source_ts.go` | Tier 4 — read MongoDB `_id` timestamp → backfill PG `_source_ts` |
| `TimestampDetector` | `timestamp_detector.go` | Migration 017 — sample Mongo, probe candidate timestamp fields |

### 2.6 Provisioning state machine

**Gating**: `PROVISIONING_ORCHESTRATOR_ENABLED=1` (env). Khi `0`, các sub liên quan KHÔNG được register.

**Bảng transition** (`provisioning_state_machine.go:53–57`):

| From | Step | NATS subject | Pending state | Final state |
|---|---|---|---|---|
| `draft` | `shadow_bind` | `cdc.cmd.shadow.bind` | `shadow_pending` | `shadow_active` |
| `shadow_active` | `master_bind` | `cdc.cmd.master.bind` | `master_pending` | `master_active` |
| `master_active` | `discover` | `cdc.cmd.discover` | `mapping_pending` | `mapping_ready` |
| `mapping_ready` | `schedule_enable` | `cdc.cmd.schedule.enable` | `schedule_pending` | `running` |

**Special**: `schedule_enable` KHÔNG emit `step_completed` ở handler. `JobMonitor.HandleCompleted` bridge first `cdc.evt.transmute.completed{status=success}` → publish `step_completed` cho mọi source ở `schedule_pending` (job_monitor.go:106–173).

**Recovery**: `RecoveryLoop` mỗi 1 phút quét row `*_pending` với `updated_at < NOW() - 10m` → CAS `failed` với error `TIMEOUT_EXCEEDED`. TTL chỉnh qua `PROVISIONING_PENDING_TTL_MIN`.

**Step log**: `cdc_system.source_object_registry.provisioning_step_log` JSONB array, max 50 entries (qua PG fn `cdc_system.append_step_log_capped`, D7).

### 2.7 REST endpoints (port `:8082`)

| Method | Path | File:Line |
|---|---|---|
| GET | `/health` | worker_server.go:441 |
| GET | `/ready` | worker_server.go:444 (PG ping; 503 nếu down) |
| GET | `/metrics` | worker_server.go:451 (Prom adaptor) |
| GET | `/api/v1/internal/stats` | worker_server.go:454 (`consumerPool.GetStats() + batchBuffer.GetStatus()`) |

**Port `:9090`** — standalone Prom server (`net/http promhttp.Handler()`), tách khỏi Fiber để metrics luôn reachable.

**Không có JWT middleware ở worker HTTP** — tất cả endpoint internal-only, không expose public.

### 2.8 Config + env override

**File**: `config/config-local.yml` — các block chính: `server`, `db`, `systemDb`, `shadowDb`, `masterDb`, `controlPlane`, `sources`, `nats`, `kafka`, `otel`, `redis`, `worker`, `jwt`.

**Env override** (config.go:273–351) — đầy đủ:

| Env | Field |
|---|---|
| `CFG_PATH` | config file path |
| `DB_SINK_URL` | `cfg.DB.URL` |
| `DB_READ_REPLICA_DSN` | `cfg.DB.ReadReplicaDSN` |
| `CDC_SYSTEM_DB_URL` | `cfg.SystemDB.URL` |
| `CDC_CONTROL_PLANE_URL` | `cfg.ControlPlane.URL` |
| `CDC_DESTINATION_URL` / `CDC_MASTER_DB_URL` | `cfg.MasterDB.URLs["default"]` |
| `CDC_SHADOW_DB_URL` | `cfg.ShadowDB.URLs["default"]` |
| `CDC_SHADOW_DB_URLS` / `CDC_MASTER_DB_URLS` | full map JSON or `;`-sep |
| `CDC_*_DB_DEFAULT_KEY` | DefaultKey |
| `NATS_URL`, `REDIS_URL`, `JWT_SECRET` | self-explanatory |
| `KAFKA_CONNECT_URL` | `cfg.Debezium.KafkaConnectURL` |
| `DEBEZIUM_CONNECTOR_NAME` | override connector name |
| `PROVISIONING_ORCHESTRATOR_ENABLED` | gate provisioning subs |
| `PROVISIONING_PENDING_TTL_MIN` | recovery TTL |
| `PROVISIONING_STEP_LOG_MAX` | step log cap |
| `PROVISIONING_DEFAULT_SHADOW_CONNECTION_CODE` | shadow lookup |
| `PROVISIONING_DEFAULT_CRON_EXPR` | default cron `*/1 * * * *` |
| `SOURCE_DSN_<connection_id>` | per-conn source DSN |
| `SOURCE_PG_DSN`, `SOURCE_MYSQL_DSN` | engine fallback |
| `MONGODB_URL` | bridge `sources.mongodb_primary` |

**Scheduled background jobs** (worker_server.go:Start):

| Operation | Interval | Trigger |
|---|---|---|
| `transform` | 5m | publish `cdc.cmd.batch-transform` per active table |
| `field-scan` | 60m | publish `cdc.cmd.periodic-scan` / `scan-raw-data` |
| `partition-check` | 24h | `SELECT ensure_cdc_partition(...)` |
| `reconcile` | 30m | `reconCore.CheckAll(ctx)` |
| `bridge` | — | no-op (legacy) |

---

## 3. cdc-cms-service

Đường dẫn: `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service`. Port `:8083`.

### 3.1 Entry + middleware chain

**`cmd/server/main.go`**:
1. `config.NewConfig()` — Viper, env `CFG_PATH`
2. `observability.InitOtel(...)` (optional)
3. `server.New(cfg, logger)` — full DI
4. `srv.Start()` — Fiber listen `cfg.Server.Port`

**`internal/server/server.go:162–171`** — Fiber config:
```go
app := fiber.New(fiber.Config{DisableStartupMessage: true})
app.Use(fiberlogger.New())   // request log
app.Use(cors.New())          // permissive CORS
app.Get("/swagger/*", swagger.HandlerDefault)
```

**Background goroutines** (`server.go:183–213`):
- `reconSvc.Start(ctx)` — no-op sau Airbyte retire
- `healthCollector.Run(ctx)` — Redis snapshot mỗi 15s
- `auditLogger.Run(ctx)` — drain audit channel → batch INSERT `admin_actions` mỗi 2s
- `alertMgr.RunBackgroundResolver(ctx)` — 1 phút: reopen expired silences, auto-resolve stale firing alerts

### 3.2 API routes (đầy đủ)

`apiGroup.Use(middleware.JWTAuth(cfg))` — toàn bộ `/api/*` cần JWT. Public exceptions: `/health`, `/ready`, `/api/system/health`, `/swagger/*`.

**Vai trò**:
- `OpsAdmin` = `RequireOpsAdmin()` (accept role `ops-admin` HOẶC legacy `admin`)
- `Admin` = `RequireRole("admin")`
- `Shared` = `RequireRole("admin","operator")`
- **Destructive chain** = OpsAdmin + Idempotency-Key (Redis 1h) + Audit (`reason ≥ 10` chars)
- **Restart chain** = OpsAdmin + RateLimit(3/h) + Idempotency-Key + Audit

#### 3.2.1 Public/Health

| Method | Path | Handler |
|---|---|---|
| GET | `/health` | `HealthHandler.Health` |
| GET | `/ready` | `HealthHandler.Ready` |
| GET | `/api/system/health` | `SystemHealthHandler.Health` (Redis snapshot) |
| GET | `/swagger/*` | swagger.HandlerDefault |

#### 3.2.2 Destructive (OpsAdmin + Idempotency + Audit)

| Method | Path | Handler:line | Mục đích |
|---|---|---|---|
| POST | `/api/reconciliation/check` | `ReconciliationHandler.TriggerCheckAll`:450 | Tier-1 recon all/one |
| POST | `/api/reconciliation/check/:table` | `ReconciliationHandler.TriggerCheck`:392 | Tier1 recon 1 bảng |
| POST | `/api/reconciliation/heal` | `ReconciliationHandler.TriggerHeal`:488 | Heal |
| POST | `/api/reconciliation/heal/:table` | `ReconciliationHandler.TriggerHeal`:488 | Heal 1 bảng |
| POST | `/api/failed-sync-logs/:id/retry` | `ReconciliationHandler.RetryFailedLog`:642 | Retry 1 DLQ record |
| POST | `/api/tools/reset-debezium-offset` | `ReconciliationHandler.ResetDebeziumOffset`:723 | Publish `cdc.cmd.debezium-signal` |
| POST | `/api/tools/trigger-snapshot/:table` | `ReconciliationHandler.TriggerSnapshot`:889 | Publish `cdc.cmd.debezium-snapshot` |
| POST | `/api/tools/restart-debezium` | `SystemHealthHandler.RestartDebezium`:119 | Restart connector (rate-limit 3/h) |
| POST | `/api/recon/backfill-source-ts` | `ReconciliationHandler.TriggerBackfillSourceTs`:750 | Dispatch backfill |
| POST | `/api/v1/system/connectors` | `SystemConnectorsHandler.Create`:184 | Create Debezium connector + persist Source fingerprint |
| POST | `/api/v1/system/connectors/:name/restart` | `SystemConnectorsHandler.Restart`:148 | Restart connector + tasks |
| POST | `/api/v1/system/connectors/:name/tasks/:taskId/restart` | `SystemConnectorsHandler.RestartTask`:162 | Restart 1 task |
| POST | `/api/v1/system/connectors/:name/pause` | `SystemConnectorsHandler.Pause`:261 | Pause |
| POST | `/api/v1/system/connectors/:name/resume` | `SystemConnectorsHandler.Resume`:265 | Resume |
| DELETE | `/api/v1/system/connectors/:name` | `SystemConnectorsHandler.Delete`:237 | Delete + soft-delete fingerprint |
| POST | `/api/v1/masters` | `MasterRegistryHandler.Create`:309 | Create `master_binding` (status=pending_review) |
| POST | `/api/v1/masters/:name/approve` | `MasterRegistryHandler.Approve`:446 | Publish `cdc.cmd.master-create` |
| POST | `/api/v1/masters/:name/reject` | `MasterRegistryHandler.Reject`:517 | Mark rejected |
| POST | `/api/v1/masters/:name/toggle-active` | `MasterRegistryHandler.ToggleActive`:575 | Toggle is_active |
| POST | `/api/v1/masters/:name/swap` | `MasterRegistryHandler.Swap`:630 | Atomic RENAME swap |
| POST | `/api/v1/wizard/sessions/:id/execute` | `WizardHandler.Execute`:129 | Flip wizard → running |
| POST | `/api/v1/schema-proposals/:id/approve` | `SchemaProposalHandler.Approve` | ALTER TABLE + create rule |
| POST | `/api/v1/schema-proposals/:id/reject` | `SchemaProposalHandler.Reject` | Mark rejected |
| POST | `/api/v1/schedules` | `TransmuteScheduleHandler.Create`:90 | Create/upsert schedule |
| POST | `/api/v1/schedules/:id/run-now` | `TransmuteScheduleHandler.RunNow`:171 | Publish `cdc.cmd.transmute` |
| PATCH | `/api/v1/schedules/:id` | `TransmuteScheduleHandler.Toggle`:147 | Toggle enabled |
| POST | `/api/v1/mapping-rules/preview` | `MappingPreviewHandler.Preview` | JsonPath preview vs live shadow |
| POST | `/api/alerts/:fingerprint/ack` | `AlertsHandler.Ack`:109 | Ack alert |
| POST | `/api/alerts/:fingerprint/silence` | `AlertsHandler.Silence`:143 | Silence alert |
| GET | `/api/v1/cms/sources/:id/provisioning` | `ProvisioningHandler.GetState`:100 | Read snapshot |
| POST | `/api/v1/cms/sources/:id/provisioning/advance` | `ProvisioningHandler.Advance`:113 | Advance 1 step |
| POST | `/api/v1/cms/sources/:id/provisioning/pause` | `ProvisioningHandler.Pause`:131 | running → paused |
| POST | `/api/v1/cms/sources/:id/provisioning/resume` | `ProvisioningHandler.Resume`:144 | paused → running |
| POST | `/api/v1/cms/sources/:id/provisioning/retry` | `ProvisioningHandler.Retry`:157 | failed → rollback to last from_state → Advance |
| POST | `/api/v1/cms/sources/:id/provisioning/archive` | `ProvisioningHandler.Archive`:172 | any → archived |
| POST | `/api/v1/cms/sources/:id/provisioning/mode` | `ProvisioningHandler.SetMode`:190 | Toggle auto/manual |

#### 3.2.3 Shared (admin + operator) — read-mostly

| Method | Path | Handler:line |
|---|---|---|
| GET | `/api/schema-changes/pending` | `SchemaChangeHandler.GetPending`:42 |
| GET | `/api/schema-changes/history` | `SchemaChangeHandler.GetHistory`:148 |
| GET | `/api/sync/health` | `RegistryHandler.SyncHealth`:422 |
| GET | `/api/activity-log` | `ActivityLogHandler.List`:105 |
| GET | `/api/activity-log/stats` | `ActivityLogHandler.Stats`:205 |
| GET | `/api/worker-schedule` | `ScheduleHandler.List`:315 |
| GET | `/api/v1/source-objects/stats` | `SourceObjectsHandler.GetStats`:141 |
| GET | `/api/v1/source-objects` | `SourceObjectsHandler.List`:251 |
| GET | `/api/v1/source-objects/registry/:registry_id` | `SourceObjectsHandler.GetMappingContext`:517 |
| GET | `/api/v1/shadow-bindings` | `SourceObjectsHandler.ListShadowBindings`:405 |
| GET | `/api/v1/source-objects/:id/dispatch-status` | `SourceObjectActionsHandler.DispatchStatusV2`:487 |
| GET | `/api/v1/source-objects/:id/transform-status` | `SourceObjectActionsHandler.TransformStatusV2`:636 |
| GET | `/api/v1/source-objects/registry/:id/dispatch-status` | `SourceObjectActionsHandler.DispatchStatus`:468 |
| GET | `/api/v1/source-objects/registry/:id/transform-status` | `SourceObjectActionsHandler.TransformStatus`:619 |
| GET | `/api/mapping-rules` | `MappingRuleHandler.List`:244 |
| GET | `/api/introspection/scan/:table` | `IntrospectionHandler.Scan`:33 (NATS Request 10s) |
| GET | `/api/introspection/scan-raw/:table` | `IntrospectionHandler.ScanRawData`:78 |
| GET | `/api/v1/system/connectors` | `SystemConnectorsHandler.List`:53 |
| GET | `/api/v1/system/connectors/:name` | `SystemConnectorsHandler.Get`:111 |
| GET | `/api/v1/system/connector-plugins` | `SystemConnectorsHandler.Plugins`:136 |
| GET | `/api/v1/sources` | `SourcesHandler.List`:26 |
| GET | `/api/v1/sources/:id` | `SourcesHandler.Get`:38 |
| GET | `/api/v1/wizard/sessions/:id` | `WizardHandler.Get`:61 |
| GET | `/api/v1/wizard/sessions/:id/progress` | `WizardHandler.Progress`:154 |
| GET | `/api/v1/masters` | `MasterRegistryHandler.List`:97 |
| GET | `/api/v1/schema-proposals` | `SchemaProposalHandler.List` |
| GET | `/api/v1/schema-proposals/:id` | `SchemaProposalHandler.Get` |
| GET | `/api/v1/schedules` | `TransmuteScheduleHandler.List`:54 |
| GET | `/api/reconciliation/report` | `ReconciliationHandler.LatestReport`:175 |
| GET | `/api/reconciliation/report/:table` | `ReconciliationHandler.TableHistory`:357 |
| GET | `/api/failed-sync-logs` | `ReconciliationHandler.ListFailedLogs`:538 |
| GET | `/api/recon/backfill-source-ts/status` | `ReconciliationHandler.BackfillSourceTsStatus`:804 |
| GET | `/api/alerts/active` | `AlertsHandler.Active`:38 |
| GET | `/api/alerts/silenced` | `AlertsHandler.Silenced`:52 |
| GET | `/api/alerts/history` | `AlertsHandler.History`:68 |

#### 3.2.4 Admin-only (write)

| Method | Path | Handler:line |
|---|---|---|
| POST | `/api/schema-changes/:id/approve` | `SchemaChangeHandler.Approve`:78 |
| POST | `/api/schema-changes/:id/reject` | `SchemaChangeHandler.Reject`:114 |
| POST | `/api/v1/source-objects/register` | `SourceObjectActionsHandler.Register`:87 → `RegistryHandler.Register` |
| PATCH | `/api/v1/source-objects/:id` | `SourceObjectActionsHandler.UpdateV2`:123 |
| POST | `/api/v1/source-objects/:id/create-default-columns` | `SourceObjectActionsHandler.CreateDefaultColumnsV2`:246 |
| POST | `/api/v1/source-objects/:id/scan-fields` | `SourceObjectActionsHandler.ScanFieldsV2`:394 |
| POST | `/api/v1/source-objects/:id/standardize` | `SourceObjectActionsHandler.StandardizeV2`:324 |
| PATCH | `/api/v1/source-objects/registry/:id` | `SourceObjectActionsHandler.UpdateBridge`:105 |
| POST | `/api/v1/source-objects/register-batch` | `SourceObjectActionsHandler.BulkRegister`:212 |
| POST | `/api/v1/source-objects/registry/:id/standardize` | `SourceObjectActionsHandler.Standardize`:306 |
| POST | `/api/v1/source-objects/registry/:id/scan-fields` | `SourceObjectActionsHandler.ScanFields`:376 |
| POST | `/api/v1/source-objects/registry/:id/transform` | `SourceObjectActionsHandler.Transform`:451 |
| POST | `/api/v1/source-objects/:id/detect-timestamp-field` | `SourceObjectActionsHandler.DetectTimestampFieldV2`:564 |
| POST | `/api/v1/source-objects/registry/:id/detect-timestamp-field` | `SourceObjectActionsHandler.DetectTimestampField`:546 |
| POST | `/api/v1/source-objects/registry/:id/create-default-columns` | `SourceObjectActionsHandler.CreateDefaultColumns`:228 |
| POST | `/api/mapping-rules` | `MappingRuleHandler.Create`:373 |
| PATCH | `/api/mapping-rules/batch` | `MappingRuleHandler.BatchUpdate`:615 (publish `alter-column` per approved) |
| PATCH | `/api/mapping-rules/:id` | `MappingRuleHandler.UpdateStatus`:535 |
| POST | `/api/mapping-rules/reload` | `MappingRuleHandler.Reload`:479 (publish `schema.config.reload`) |
| POST | `/api/mapping-rules/:id/backfill` | `MappingRuleHandler.Backfill`:584 |
| PATCH | `/api/worker-schedule/:id` | `ScheduleHandler.Update`:337 |
| POST | `/api/worker-schedule` | `ScheduleHandler.Create`:410 |
| POST | `/api/v1/wizard/sessions` | `WizardHandler.Create`:36 |
| PATCH | `/api/v1/wizard/sessions/:id` | `WizardHandler.Patch`:82 |

### 3.3 Handlers (`internal/api/`)

| File | Struct | Public methods chính |
|---|---|---|
| `activity_log_handler.go` | `ActivityLogHandler` | `List`, `Stats` |
| `alerts_handler.go` | `AlertsHandler` | `Active`, `Silenced`, `History`, `Ack`, `Silence` |
| `health_handler.go` | `HealthHandler` | `Health`, `Ready` |
| `introspection_handler.go` | `IntrospectionHandler` | `Scan`, `ScanRawData` (NATS Request 10s) |
| `mapping_preview_handler.go` | `MappingPreviewHandler` | `Preview` (JsonPath vs live shadow `_raw_data`) |
| `mapping_rule_handler.go` | `MappingRuleHandler` | `List`, `Create`, `UpdateStatus`, `Reload`, `Backfill`, `BatchUpdate` |
| `master_registry_handler.go` | `MasterRegistryHandler` | `List`, `Create`, `Approve`, `Reject`, `ToggleActive`, `Swap` |
| `provisioning_handler.go` | `ProvisioningHandler` | `GetState`, `Advance`, `Pause`, `Resume`, `Retry`, `Archive`, `SetMode` |
| `reconciliation_handler.go` | `ReconciliationHandler` | 11 methods (recon + DLQ + Debezium signal + backfill) |
| `registry_handler.go` | `RegistryHandler` | 13 methods — legacy bridge; `Register` gọi `ShadowAutomator` |
| `schedule_handler.go` | `ScheduleHandler` | `List`, `Update`, `Create` |
| `schema_change_handler.go` | `SchemaChangeHandler` | `GetPending`, `Approve` (ALTER + rule trong TX), `Reject`, `GetHistory` |
| `schema_proposal_handler.go` | `SchemaProposalHandler` | `List`, `Get`, `Approve`, `Reject` (Sprint 5) |
| `source_object_actions_handler.go` | `SourceObjectActionsHandler` | V2 namespace wrapper + V2 resolves `source_object_id` → active `shadow_binding` |
| `source_objects_handler.go` | `SourceObjectsHandler` | `GetStats`, `List`, `ListShadowBindings`, `GetMappingContext` |
| `sources_handler.go` | `SourcesHandler` | `List`, `Get` (`cdc_system.sources`) |
| `system_connectors_handler.go` | `SystemConnectorsHandler` | HTTP proxy → Kafka Connect REST; strip credential fields |
| `system_health_handler.go` | `SystemHealthHandler` | `Health` (Redis snapshot), `RestartDebezium` |
| `transmute_schedule_handler.go` | `TransmuteScheduleHandler` | `List`, `Create`, `Toggle`, `RunNow` |
| `wizard_handler.go` | `WizardHandler` | `Create`, `Get`, `Patch`, `Execute`, `Progress` |

### 3.4 Services (`internal/service/`)

| Service | Mục đích |
|---|---|
| `ApprovalService` | Schema change approval — ALTER TABLE + create rule + log trong 1 TX, sau publish `schema.config.reload` |
| `AlertManager` | State machine (firing→ack→silenced→resolved); fingerprint = `sha256(name+sorted labels)`; Redis dedup 5p; `RunBackgroundResolver` 1p |
| `MasterSwap` | Atomic `RENAME` master tables — TX `SET LOCAL lock_timeout='3s'`, current → `<name>_old_<ts>`, new → current |
| `ProvisioningOrchestrator` | CMS-side state machine — `Advance/Pause/Resume/Retry/SetMode/Archive`, `seedMasterBindingForAdvance` UPSERT `master_binding` |
| `ReconciliationService` | No-op sau Airbyte retire |
| `ShadowAutomator` | Sync DDL ở Register: schema + table + 8 cols + indexes + sonyflake trigger (idempotent) |
| `SourceObjectV2SyncService` | Sync `source_object_registry` + `shadow_binding` từ legacy `cdc_table_registry` |
| `Collector` (system_health_collector.go) | Poll Kafka Connect + NATS monitor + Prom + worker /metrics → JSON snapshot vào Redis mỗi 15s |
| `PromClient` | Query Prometheus histogram_quantile, fallback scrape worker /metrics |

### 3.5 NATS publishes từ CMS

CMS là **publisher thuần** — không subscribe subject nào. Tất cả fire-and-forget hoặc sync Request (introspection 10s).

| Subject | Publisher (file:line) | Trigger |
|---|---|---|
| `cdc.cmd.create-default-columns` | registry_handler.go:132,310,533 / source_object_actions_handler.go:273 | Register / BulkRegister / V2 |
| `cdc.cmd.standardize` | registry_handler.go:363 / source_object_actions_handler.go:347 | Standardize |
| `cdc.cmd.scan-fields` | registry_handler.go:402 / source_object_actions_handler.go:419 | ScanFields |
| `cdc.cmd.batch-transform` | registry_handler.go:462 | Transform |
| `cdc.cmd.detect-timestamp-field` | registry_handler.go:625 / source_object_actions_handler.go:589 | DetectTimestampField |
| `cdc.cmd.backfill` | mapping_rule_handler.go:600 | Backfill rule |
| `cdc.cmd.alter-column` | mapping_rule_handler.go:658 | Approve rule batch |
| `cdc.cmd.master-create` | master_registry_handler.go:494 | Approve master |
| `cdc.cmd.recon-check` | reconciliation_handler.go:417,460,467 | Trigger recon |
| `cdc.cmd.recon-heal` | reconciliation_handler.go:507 | Trigger heal |
| `cdc.cmd.retry-failed` | reconciliation_handler.go:697 | RetryFailedLog |
| `cdc.cmd.debezium-signal` | reconciliation_handler.go:737 | ResetDebeziumOffset |
| `cdc.cmd.debezium-snapshot` | reconciliation_handler.go:892 | TriggerSnapshot |
| `cdc.cmd.recon-backfill-source-ts` | reconciliation_handler.go:771 | TriggerBackfillSourceTs |
| `cdc.cmd.restart-debezium` | system_health_handler.go:128 | RestartDebezium |
| `cdc.cmd.transmute` | transmute_schedule_handler.go:203 | RunNow |
| `cdc.cmd.shadow.bind` / `master.bind` / `discover` / `schedule.enable` | provisioning_orchestrator.go:250 | Advance |
| `schema.config.reload` | natsconn/nats_client.go:67 (`PublishReload`) | Register/Update/Approve mapping/etc. |
| `cdc.cmd.scan-raw-data` (NATS Request) | introspection_handler.go:36 | Scan (sync 10s) |
| `cdc.cmd.introspect` (NATS Request) | introspection_handler.go:49 | Scan fallback |

### 3.6 Auth/RBAC/Idempotency/Audit

#### JWT (`internal/middleware/jwt.go:12–47`)
- Header `Authorization: Bearer <token>`
- HMAC only (reject non-HMAC)
- Secret = `cfg.JWT.Secret` (env `JWT_SECRET`, dev: `change-me-in-production`)
- Claims → `Locals`: `username`, `role`, `roles`
- Public: `/health`, `/ready`, `/api/system/health`, `/swagger/*`

#### RBAC (`internal/middleware/rbac.go`)
3 shape resolution:
1. Legacy single string `role`
2. Multi-role `roles []string` / `[]interface{}`
3. `ADMIN_USERS` env (CSV usernames được cấp `ops-admin`)

Roles: `ops-admin` / `admin` / `operator`. `RequireOpsAdmin()` = `RequireAnyRole("ops-admin","admin")` (backward-compat IdP).

#### Idempotency-Key (`internal/middleware/idempotency.go`)
- Required header trên mọi destructive POST/PATCH
- Format: `[A-Za-z0-9\-_\.]{8-128}`
- Redis keys: `idem:{route}:{key}:lock` (TTL 30s), `idem:{route}:{key}:response` (TTL 1h)
- Missing → `400 missing Idempotency-Key header`
- Concurrent → `409 in progress, retry_after: 30`
- Cached success → `200` + header `X-Idempotent-Replay: true`
- Chỉ cache success (<400)

#### Audit (`internal/middleware/audit.go`)
- Trigger qua `AuditLogger.Middleware()` trên destructive routes
- **Reason validation**: field `reason` body JSON, **≥10 non-whitespace chars**, missing/short → `400 missing or too-short 'reason', min_length: 10`
- Async queue (buffered 100); drops oldest nếu full; batch INSERT `cdc_system.admin_actions` mỗi 2s hoặc batch≥16
- Fields: `user_id, action, target, payload (cap 64KiB), reason, result, idempotency_key, ip_address, user_agent, created_at`
- Restart rate-limit: 3/h/user (Redis scope `restart`)

### 3.7 DB Models (`internal/model/`)

| Model | File | Table | Key cols |
|---|---|---|---|
| `Source` | source.go | `cdc_system.sources` | `connector_name` UNIQUE, `source_type`, `connector_class`, `topic_prefix`, `status`, `created_by` |
| `TableRegistry` | table_registry.go | `cdc_table_registry` | `source_db`, `source_table`, `target_table`, `sync_engine`, `priority`, `is_active`, `is_table_created`, `sync_status`, `last_recon_at`, `recon_drift`, `is_partitioned`, `timestamp_field` |
| `MappingRule` | mapping_rule.go | `cdc_mapping_rules` | `source_table`, `source_field`, `target_column`, `data_type`, `status`, `rule_type` (system/discovered/mapping) |
| `ActivityLog` | activity_log.go | `cdc_activity_log` | partitioned, composite PK `(created_at,id)`, `operation`, `target_table`, `status`, `rows_affected`, `duration_ms`, `details` |
| `Alert` | alert.go | `cdc_system.cdc_alerts` | UUID, `fingerprint` UNIQUE, `severity`, `labels`, `status`, `occurrence_count` |
| `ReconciliationReport` | reconciliation_report.go | `cdc_reconciliation_report` | `target_table`, `source_count`, `dest_count`, `diff`, `missing_count`, `missing_ids`, `tier`, `healed_count` |
| `FailedSyncLog` | failed_sync_log.go | `failed_sync_logs` | partitioned, `target_table`, `record_id`, `raw_json`, `kafka_topic/partition/offset`, `retry_count`, `status` |
| `WorkerSchedule` | worker_schedule.go | `cdc_worker_schedule` | `operation`, `target_table`, `interval_minutes`, `is_enabled`, `next_run_at` |
| `WizardSession` | wizard_session.go | `cdc_system.cdc_wizard_sessions` | UUID, `current_step`, `status`, `step_payload`, `progress_log` |
| `PendingField` | pending_field.go | `pending_fields` | `table_name`, `field_name`, `suggested_type`, `status`, `target_column_name` |
| `SchemaChangeLog` | schema_change_log.go | `schema_changes_log` | `table_name`, `change_type`, `field_name`, `sql_executed`, `rollback_sql` |

**V2 tables (raw SQL only)**: `cdc_system.source_object_registry`, `shadow_binding`, `master_binding`, `mapping_rule_v2`, `connection_registry`, `transmute_schedule`, `admin_actions`, `sync_runtime_state`.

---

## 4. Tầng dữ liệu

### 4.1 `cdc_system.*` — Control plane (cdc_dw, schema cdc_system)

#### 4.1.1 `connection_registry` (mig 029)
| Cột | Kiểu | Default | Ghi chú |
|---|---|---|---|
| id BIGSERIAL PK | | | |
| connection_code VARCHAR(100) | | | UNIQUE |
| display_name VARCHAR(200) | | | |
| role_type VARCHAR(32) | | | CHECK IN ('source','shadow','master','system','mixed') |
| engine_type VARCHAR(32) | | | CHECK IN ('postgresql','mariadb','mysql','mongodb','clickhouse') |
| host, port, default_database, default_schema | | | |
| secret_ref VARCHAR(255) | | | e.g. `env:DB_SINK_URL` |
| options_json, capabilities_json JSONB | `'{}'` | | |
| status VARCHAR(32) | `'active'` | | CHECK IN ('active','paused','failed','retired') |

Indexes: `idx_v2_connection_role/engine/status`. Seeded codes: `legacy_system_db`, `legacy_shadow_default`, `legacy_master_default`, `mariadb_legacy_default`.

#### 4.1.2 `source_object_registry` (mig 030 + 047)
| Cột | Kiểu | Default | Ghi chú |
|---|---|---|---|
| id BIGSERIAL PK | | | |
| object_code VARCHAR(150) UNIQUE | | | e.g. `mariadb_legacy_orders_v1` |
| source_connection_id BIGINT FK→connection_registry | | | |
| source_engine_type VARCHAR(32) | | | CHECK postgresql/mariadb/mysql/mongodb/clickhouse |
| source_database/schema/namespace/object_name VARCHAR(255) | | | |
| source_object_type VARCHAR(32) | | | CHECK table/collection/view |
| source_locator_json JSONB | `'{}'` | | chứa `kafka_topic, topic_prefix` |
| normalized_source_key VARCHAR(500) UNIQUE | | | format `engine:db:object` |
| primary_key_field VARCHAR(255) | `'id'` | | |
| primary_key_type VARCHAR(100) | | | |
| timestamp_field VARCHAR(255) | | | |
| timestamp_candidates_json JSONB | `'[]'` | | |
| cdc_mode VARCHAR(32) | `'incremental'` | | CHECK snapshot/incremental/full_refresh/hybrid |
| sync_engine VARCHAR(32) | `'debezium'` | | CHECK debezium/airbyte/both/custom |
| is_active BOOLEAN | TRUE | | |
| profile_status VARCHAR(32) | `'draft'` | | CHECK draft/pending_data/syncing/active/failed/paused |
| **provisioning_mode VARCHAR(20)** | `'manual'` | mig 047 | CHECK auto/manual |
| **provisioning_state VARCHAR(40)** | `'draft'` | mig 047 | state machine |
| **provisioning_step_log JSONB** | `'[]'` | mig 047 | cap 50 entries |
| **last_step_error TEXT** | | mig 047 | |

#### 4.1.3 `shadow_binding` (mig 031 + 043)
| Cột | Default | Ghi chú |
|---|---|---|
| id BIGSERIAL PK | | |
| binding_code UNIQUE | | |
| source_object_id FK→source_object_registry CASCADE | | |
| shadow_connection_id FK→connection_registry | | |
| shadow_database, shadow_schema, shadow_table | | normalized 043: `shadow_<source_database>` |
| physical_table_fqn VARCHAR(600) | | `shadow_<src>.<table>` |
| namespace_strategy | `'preserve'` | CHECK preserve/prefix/flatten/custom |
| write_mode | `'upsert'` | CHECK upsert/append/replace |
| ddl_status | `'pending'` | CHECK pending/created/failed/drifted |
| is_active BOOLEAN | TRUE | |

UNIQUE `(source_object_id, shadow_connection_id, shadow_schema, shadow_table)`.

#### 4.1.4 `master_binding` (mig 032)
| Cột | Default | Ghi chú |
|---|---|---|
| id BIGSERIAL PK | | |
| binding_code UNIQUE | | |
| source_object_id FK CASCADE | | |
| shadow_binding_id FK SET NULL | | |
| master_connection_id FK | | |
| master_database/schema/table | | |
| physical_table_fqn VARCHAR(600) | | |
| transform_type VARCHAR(32) | | CHECK copy_1_to_1/filter/aggregate/group_by/join/custom_sql |
| transform_spec JSONB | `'{}'` | |
| schema_status | `'pending_review'` | CHECK pending_review/approved/rejected/failed/drifted |
| is_active BOOLEAN | FALSE | CHECK: `is_active=FALSE OR schema_status='approved'` |

UNIQUE `(master_connection_id, master_schema, master_table)`.

#### 4.1.5 `mapping_rule_v2` (mig 033)
| Cột | Default | Ghi chú |
|---|---|---|
| id BIGSERIAL PK | | |
| source_object_id FK CASCADE | | |
| master_binding_id FK CASCADE | | |
| source_field, source_path | | gjson/JSONPath |
| target_column, data_type | | |
| source_format | `'raw'` | CHECK raw/jsonpath/expression |
| transform_fn | | |
| is_nullable BOOLEAN | TRUE | |
| default_value TEXT | | |
| is_active BOOLEAN | TRUE | |
| status | `'pending'` | CHECK pending/approved/rejected |

UNIQUE index `ux_v2_mapping_rule_identity ON (source_object_id, COALESCE(master_binding_id,0), target_column)`.

#### 4.1.6 `cdc_mapping_rules` V1 (mig 001 + 020/046/037)
Schema legacy — vẫn dùng cho shadow ingest (xem §6.1). Cột chính: `source_table, source_field, target_column, data_type, status, source_format ('debezium_after'/'debezium_full_envelope'/'airbyte_flat'/'raw_jsonb'), jsonpath, transform_fn, version, master_table, rule_type ('system'/'discovered'/'mapping')`.

UNIQUE `ux_mapping_rules_identity ON (source_table, COALESCE(master_table,''), target_column)`.

#### 4.1.7 `transmute_schedule` (mig 036)
| Cột | Default | Ghi chú |
|---|---|---|
| id BIGSERIAL PK | | |
| master_binding_id FK CASCADE | | |
| mode TEXT | | CHECK immediate/cron/post_ingest |
| cron_expr TEXT | | required khi mode='cron' |
| last_run_at, next_run_at TIMESTAMPTZ | | |
| last_status TEXT | | CHECK success/failed/running/skipped |
| last_error TEXT | | |
| last_stats JSONB | | |
| is_enabled BOOLEAN | FALSE | |

UNIQUE `(master_binding_id, mode)`. Partial index `idx_v2_schedule_due` WHERE `is_enabled=true AND mode='cron'`.

#### 4.1.8 `failed_sync_logs` (mig 010 + 045)
**Partitioned** by RANGE `created_at` (monthly). PK `(id, created_at)`.

Cột chính: `target_table, source_table, source_db, record_id, operation, raw_json JSONB, error_message, error_type, kafka_topic/partition/offset, retry_count, max_retries (default 3), status (pending/failed/retrying/resolved/dead_letter), next_retry_at, last_error`.

#### 4.1.9 `cdc_activity_log` (mig 010)
**Partitioned** by RANGE `created_at` (daily). PK `(id, created_at)`.

Cột: `operation, target_table, status (running/success/error/skipped), rows_affected, duration_ms, details JSONB, error_message, triggered_by (scheduler/manual/nats-command), started_at, completed_at`.

#### 4.1.10 `sync_runtime_state` (mig 034)
Cột: `source_object_id, shadow_binding_id, master_binding_id`, `runtime_scope (source/shadow/master)`, `last_success_at, last_error_at, last_error_message, last_cursor_json, last_source_ts, last_recon_at, recon_drift_count, ddl_status, stats_json`.

CHECK `v2_runtime_requires_target_ref` — mỗi scope cần ref FK tương ứng.

#### 4.1.11 `admin_actions` (mig 040)
**Partitioned** by month. PK `(created_at, id)`. Partitions: `admin_actions_2026_04..06 + _default`.

Cột: `user_id, action, target, payload, reason (mandatory), result, idempotency_key, ip_address, user_agent`.

#### 4.1.12 `cdc_alerts` (mig 041)
UUID PK. Cột: `fingerprint UNIQUE, name, severity, labels JSONB, description, status (firing/resolved), fired_at, resolved_at, ack_by, ack_at, silenced_by, silenced_until, silence_reason, occurrence_count, last_fired_at`.

#### 4.1.13 Bảng phụ
- `cdc_table_registry` — V1 source registry
- `cdc_reconciliation_report` — per-run recon results
- `recon_runs` — UUID-keyed runs, partial UNIQUE 1-running-per-table
- `cdc_worker_schedule` — worker cron (bridge/transform/field-scan/partition-check)
- `schema_proposal` — Sprint 5 schema approval
- `worker_registry` — Sonyflake machine_id heartbeat + fencing_token
- `sources` — Debezium connector fingerprints
- `cdc_wizard_sessions` — wizard state machine UUID
- `enum_types` — named enum catalogs
- `table_registry_legacy`, `master_table_registry_legacy` — old profile tracker
- Helper functions: `gen_sonyflake_id()`, `claim_machine_id`, `heartbeat_machine_id`, `tg_fencing_guard`, `tg_sonyflake_fallback`, `enable_master_rls`, `append_step_log_capped`

### 4.2 Shadow tables (cdc_dw / `shadow_*`)

**Convention** (mig 043): schema = `shadow_<source_database>` (lowercase, non-alnum→`_`); table = source object name.

**Standard cols** (từ `create_cdc_table()` mig 001 + 018):
- `<pk_field>` (Mongo `_id` → `id`)
- `_raw_data JSONB NOT NULL` — full Debezium event
- `_source VARCHAR(20) NOT NULL DEFAULT 'debezium'`
- `_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `_version BIGINT NOT NULL DEFAULT 1` — OCC counter
- `_hash VARCHAR(64)`
- `_deleted BOOLEAN DEFAULT FALSE`
- `_created_at, _updated_at TIMESTAMPTZ`
- `_gpay_id BIGINT` (Sonyflake trigger)
- `_source_ts BIGINT` (epoch ms, v1.25)

Indexes per shadow: `idx_<table>_synced ON _synced_at`, `idx_<table>_source ON _source`, `idx_<table>_raw GIN`.

Shadow schemas hiện có:
- `shadow_goopay_source.*` (orders/users/payments/orders_addtest)
- `shadow_payment_bill_service.*` (payment_bills..., payment_bills_addtest)
- `shadow_goopay_legacy_maria.*` (legacy_orders, legacy_orders_addtest)

### 4.3 Master DW tables (`goopay_dest`)

| Schema | Tạo bởi | Mục đích |
|---|---|---|
| `public` | convention | Empty (Phase 39 invariant: 0 tables) |
| `master` | mig 001_dest_init | Optional grouping |
| `dw_<binding_code>` | runtime `MasterDDLGenerator.Apply()` | Per-binding DW schemas |

**Standard cols** master table:
```sql
"_gpay_id"        BIGINT PRIMARY KEY        -- Sonyflake
"_gpay_source_id" TEXT NOT NULL             -- source row identity
"_raw_data"       JSONB                     -- full Debezium envelope
"_source"         TEXT NOT NULL             -- source connection_code
"_source_ts"      BIGINT                    -- epoch ms source
"_synced_at"      TIMESTAMPTZ NOT NULL
"_version"        BIGINT NOT NULL DEFAULT 1
"_hash"           TEXT NOT NULL
"_gpay_deleted"   BOOLEAN NOT NULL DEFAULT FALSE
"_created_at"     TIMESTAMPTZ DEFAULT NOW()
"_updated_at"     TIMESTAMPTZ DEFAULT NOW()
-- + user-defined typed columns từ mapping_rule_v2
```

**Auto indexes**:
- `UNIQUE ux_<table>_source_id ON (_gpay_source_id)`
- `INDEX ix_<table>_created_at`, `ix_<table>_updated_at`
- Financial cols (regex `^(amount|fee|balance|total|price|refund|subtotal|discount|tax|cost)...|_amount$|_fee$|_balance$|_price$`) → individual index

**Re-apply safe**: `AlterSQL` list emit `ADD COLUMN IF NOT EXISTS` — idempotent rerun.

### 4.4 Source DBs

#### `goopay_source.public` (PG :5435)
| Bảng | Cấu trúc |
|---|---|
| `orders` | `id BIGSERIAL, user_id, amount NUMERIC(15,2), status VARCHAR(32), notes, created_at, updated_at` |
| `users` | `id BIGSERIAL, username, email, full_name, is_active, created_at, updated_at` |
| `payments` | `id BIGSERIAL, order_id, method, amount, status, transaction_id, paid_at, created_at, updated_at` |

`REPLICA IDENTITY FULL` cả 3 bảng. Seed 10 rows mỗi bảng.

#### `goopay_legacy_maria.*` (MariaDB :3307)
| Bảng | Cấu trúc |
|---|---|
| `legacy_orders` | `id BIGINT AUTO_INCREMENT PK, order_code VARCHAR(64), user_id, amount INT, status, created_at, updated_at` (InnoDB, binlog ROW+FULL) |

Seed 5 rows. CDC user grant `REPLICATION SLAVE, REPLICATION CLIENT, RELOAD`.

#### `payment-bill-service.*` (Mongo :27018, rs0)
Collections (theo `collection.include.list`):
- `payment-bills`, `refund-requests`, `payment-bill-histories`, `payment-bill-codes`, `payment-bill-events`, `payment-bill-holdings`, `identitycounters`, `refund-requests-histories`
- `centralized-export-service.export-jobs` (qua cùng connector)

### 4.5 Debezium connectors (deployments/debezium/)

#### `cdc-pg-source` (PostgresConnector)
- Topic prefix: `cdc.gpay`
- `database.dbname=goopay_source`, `plugin.name=pgoutput`, slot `cdc_gpay_pg_source`, publication `cdc_gpay_pub`
- `table.include.list=public.orders,public.users,public.payments`
- `snapshot.mode=initial`, heartbeat 10s
- Avro converters → schema-registry:8081
- **Topics produce**: `cdc.gpay.public.{orders,users,payments}`

#### `goopay-mongodb-cdc` (MongoDbConnector)
- `mongodb.connection.string=mongodb://gpay-mongo:27017/?replicaSet=rs0`
- Topic prefix `cdc.goopay`
- `database.include.list=payment-bill-service,centralized-export-service`
- `collection.include.list=payment-bill-service.{payment-bills,refund-requests,...}` (8 collections + export-jobs)
- `capture.mode=change_streams_update_full`
- `signal.data.collection=goopay.debezium_signals`
- **Topics produce**: `cdc.goopay.payment-bill-service.<coll>` + `cdc.goopay.centralized-export-service.export-jobs`

#### `cdc-mariadb-source` (MySqlConnector)
- Topic prefix `cdc.mariadb`
- `database.server.id=1010`, `database.server.name=gpay-mariadb`
- `table.include.list=goopay_legacy_maria.legacy_orders`
- `snapshot.mode=initial`, `snapshot.locking.mode=minimal`
- Schema history: Kafka topic `cdc.mariadb.schema-history`
- **Topic produce**: `cdc.mariadb.goopay_legacy_maria.legacy_orders`

### 4.6 Kafka topics

KRaft mode, `KAFKA_AUTO_CREATE_TOPICS_ENABLE=true`. Topic dự kiến (theo connector configs):

```
cdc.gpay.public.orders
cdc.gpay.public.users
cdc.gpay.public.payments
cdc.goopay.payment-bill-service.payment-bills
cdc.goopay.payment-bill-service.refund-requests
cdc.goopay.payment-bill-service.payment-bill-histories
cdc.goopay.payment-bill-service.payment-bill-codes
cdc.goopay.payment-bill-service.payment-bill-events
cdc.goopay.payment-bill-service.payment-bill-holdings
cdc.goopay.payment-bill-service.identitycounters
cdc.goopay.payment-bill-service.refund-requests-histories
cdc.goopay.centralized-export-service.export-jobs
cdc.mariadb.goopay_legacy_maria.legacy_orders
cdc.mariadb.schema-history
_connect-configs / _connect-offsets / _connect-status (internal)
```

**Kafka Exporter** (port 9308) scrape `^cdc\..*`. **Redpanda Console** (port 18088) UI.

### 4.7 Schema Registry

Confluent Platform 7.6.0, `gpay-schema-registry`, internal 8081 / external **18081**, backed by Kafka topic `_schemas`.

Subjects naming: `<topic>-key` + `<topic>-value` (default Confluent strategy).

---

## 5. Frontend cdc-cms-web

**Location**: `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/`

**Stack**: Vite + React 18 + TypeScript + Antd + TanStack Query v5 + axios + react-router-dom v6

**API instances** (`src/services/api.ts`):
| Instance | Base URL (env) | Auth |
|---|---|---|
| `authApi` | `VITE_AUTH_API_URL` \| `http://localhost:8081` | none |
| `cmsApi` | `VITE_CMS_API_URL` \| `http://localhost:8083` | `Authorization: Bearer <access_token>` |
| `workerApi` | `VITE_WORKER_API_URL` \| `http://localhost:8082` | none |

JWT lưu `localStorage` (`access_token`, `refresh_token`, `user`). Interceptor `cmsApi` redirect `/login` khi 401.

**Pages**:
| Route | Component | Mục đích |
|---|---|---|
| `/login` | `Login` | JWT auth |
| `/` | `Dashboard` | Stats cards + Sync Health |
| `/source-to-master` | `SourceToMasterWizard` | 10-step Wizard |
| `/sources` | `SourceConnectors` | Quản lý Debezium connector (live state, tasks, config; create/pause/delete) |
| `/registry` | `TableRegistry` | V2 source object registry; toggle Auto/Manual; scan-fields; xem shadow bindings |
| `/registry/:id/mappings` | `MappingFieldsPage` | Editor mapping rules per source |
| `/masters` | `MasterRegistry` | Master binding (approve/reject/DDL apply) |
| `/schema-proposals` | `SchemaProposals` | Pending field proposals từ SinkWorker |
| `/schedules` | `TransmuteSchedules` | Edit `transmute_schedule` rows |
| `/activity-log` | `ActivityLog` | filterable cdc_activity_log |
| `/data-integrity` | `DataIntegrity` | Recon reports + manual recon trigger |
| `/system-health` | `SystemHealth` | Live snapshot; Restart Connector |
| `/schema-changes` | `SchemaChanges` | V1 schema changes log |
| `/activity-manager` | `ActivityManager` | Worker schedule (enable/disable/trigger) |

**Async Dispatch pattern** (`useAsyncDispatch`):
1. POST → 202 Accepted
2. Poll `GET .../dispatch-status?subject=<op>&since=<dispatchedAt>` mỗi 3s
3. State: `idle → dispatching → accepted → running → success|error|timeout`
4. Auto-stop terminal; hard timeout 5 phút
5. Headers: `Idempotency-Key` (UUID) + `X-Action-Reason`

**Provisioning Mode toggle** (`useProvisioningMode`):
- POST `/api/v1/cms/sources/:id/provisioning/mode` body `{ mode, reason }` + `Idempotency-Key`
- Lỗi 409 (CAS conflict) hoặc 422 (invalid transition) → propagate operator (KHÔNG auto-retry)

---

## 6. Các luồng end-to-end

### 6.1 Ingest CDC: Source → Shadow

```
Source DB (PG/Mongo/MariaDB)
   │ [Debezium: pgoutput / change-streams / binlog]
   ▼
Kafka topic cdc.<prefix>.<db>.<obj>  (Avro encoded, schema-registry id)
   │
   ▼
cdc-worker.KafkaConsumer.consumeOne()  [kafka_consumer.go:270–386]
   │ 1. detect Avro vs JSON (magic byte 0x00)
   │ 2. Avro decode + cache schema
   │ 3. SchemaValidator.ValidatePayloadWithCase(tbl, after)
   │      → ErrSchemaDrift / ErrMissingRequired → DLQ status='pending'
   │ 4. wrap into CDC envelope
   │ 5. eventHandler.HandleRaw(ctx, subject, cdcJSON)
   │
   ▼
EventHandler  (event_handler.go)
   │ MetadataRegistry.ResolveSourceRoute(topic) → tableConfig
   │ DynamicMapper.MapData(ctx, targetTable, raw) → MappedData
   │ MaskingService.MaskTableData(data) (PII)
   │
   ▼
BatchBuffer.Add(record)  (batch_buffer.go)
   │ flush khi maxSize=500 hoặc timeout 2s
   │
   ▼
SchemaAdapter.BuildUpsertSQL(...) + ConnectionManager → shadow PG
   │ INSERT … ON CONFLICT (pk) DO UPDATE WHERE _source_ts IS NULL OR EXCLUDED._source_ts > _source_ts (OCC)
   ▼
shadow_<connection>.<table> row landed
   │
   │ post-ingest hook (sinkworker.go:217)
   ▼
NATS publish cdc.cmd.transmute-shadow{shadow_table=<schema.table>}
```

**DLQ path** (kafka_consumer.go:249–264): nếu `procErr != nil`:
1. INSERT `failed_sync_logs` (raw_json + error + topic/partition/offset)
2. Nếu insert OK → commit Kafka offset (advance past message)
3. Nếu insert FAIL → KHÔNG commit → Kafka redeliver

### 6.2 Transmute: Shadow → Master

```
NATS cdc.cmd.transmute-shadow{shadow_table}
   │
   ▼
TransmuteHandler.HandleTransmuteShadow  (transmute_handler.go:43)
   │ Lookup mọi master_binding active có shadow_binding_id = ?
   │ for each master: NATS publish cdc.cmd.transmute{master_table, schedule_id?}
   ▼
TransmuteHandler.HandleTransmute  (transmute_handler.go:153)
   │ TransmuterModule.Run(ctx, masterName, sourceIDs)
   │   - loadMaster(masterName) → schema, table, connection
   │   - loadRules(masterBindingID) → mapping_rule_v2 rows
   │   - cursor loop fetchShadowBatch (LIMIT N, ORDER BY _gpay_id, ...) ← OCC: skip rows where dest._source_ts >= src._source_ts
   │   - processBatch → buildMasterRow → upsertMaster (ON CONFLICT _gpay_id DO UPDATE)
   │   - return TransmuteResult{Scanned, Inserted, Updated, Skipped, TypeErrors, RuleMisses, ActiveGate, DurationMs}
   │
   ▼
NATS publish cdc.evt.transmute.completed{schedule_id, status, stats, error}
   │
   ▼
JobMonitor.HandleCompleted  (job_monitor.go:69)
   │ UPDATE cdc_system.transmute_schedule SET last_status, last_stats, last_error, last_run_at WHERE id=?
   │ Nếu source ở schedule_pending → publish cdc.evt.provisioning.step_completed (bridge schedule_enable success)
   ▼
ProvisioningHandler.HandleStepCompleted → orchestrator: schedule_pending → running
```

**Cron path** (transmute_scheduler.go:93–): mỗi 60s tick → `SELECT * FROM transmute_schedule WHERE is_enabled AND mode='cron' AND next_run_at <= NOW() FOR UPDATE SKIP LOCKED LIMIT N` → publish `cdc.cmd.transmute` per row.

### 6.3 Provisioning cascade (state machine)

```
CMS UI: TableRegistry → toggle Auto
   │
   ▼ POST /api/v1/cms/sources/:id/provisioning/mode {mode:'auto', reason}
ProvisioningHandler.SetMode (CMS) → orchestrator.SetMode (CMS-side)
   │ UPDATE source_object_registry SET provisioning_mode='auto'
   │ if mode='auto': call Advance immediately
   │
   ▼
orchestrator.Advance(source_id):
   │ read current state, lookup ProvisioningTransitions[cur]
   │ seedMasterBindingForAdvance (UPSERT master_binding khi step='master_bind')
   │ CAS UPDATE provisioning_state: cur → cur_pending (only if WHERE state=cur)
   │ NATS publish cdc.cmd.<step>{source_id, correlation_id, ..., trace_id}
   │
   ▼ NATS cdc.cmd.shadow.bind
ProvisioningStepHandler.HandleShadowBind (worker)
   │ 1. Resolve shadow target từ source_object_registry locator
   │ 2. Mongo pre-flight (nếu Mongo source)
   │ 3. inferSourceColumns (introspect source DB)
   │ 4. SchemaAdapter.PrepareForCDCInsertWithBusinessCols(schema, table, pk, businessCols)
   │ 5. UPSERT cdc_system.shadow_binding với ddl_status='created'
   │ defer: emit cdc.evt.provisioning.step_completed{step:'shadow_bind', status:success/fail}
   │
   ▼ NATS cdc.evt.provisioning.step_completed
ProvisioningHandler.HandleStepCompleted (worker)
   │ orchestrator.HandleStepCompleted: shadow_pending → shadow_active (success) | failed (error)
   │
   ▼ orchestrator auto-Advance khi mode='auto'
NATS cdc.cmd.master.bind (alias) → MasterDDLHandler.HandleMasterCreate
   │ MasterDDLGenerator.Apply(masterName) → CREATE TABLE + indexes + RLS
   │ defer: emit step_completed
   │
   ▼ master_pending → master_active → Advance
NATS cdc.cmd.discover → CommandHandler.HandleDiscover
   │ Read information_schema.columns → INSERT cdc_mapping_rules cho cols thiếu
   │ bridgeMappingRulesToV2 → INSERT mapping_rule_v2 từ V1
   │ Optionally publish cdc.cmd.master-create (ALTER pass với mapping_rule_v2 đầy đủ)
   │ emit step_completed
   │
   ▼ mapping_pending → mapping_ready → Advance
NATS cdc.cmd.schedule.enable → ProvisioningStepHandler.HandleScheduleEnable
   │ Lookup master_binding_id cho source
   │ UPSERT cdc_system.transmute_schedule {master_binding_id, mode='cron', cron_expr=*/1 * * * *, is_enabled=true}
   │ KHÔNG emit step_completed ngay! (chờ JobMonitor bridge)
   │
   ▼ TransmuteScheduler tick (sau ~60s)
NATS cdc.cmd.transmute → TransmuteHandler.HandleTransmute → publish cdc.evt.transmute.completed{status=success}
   │
   ▼
JobMonitor.HandleCompleted bridge → publish cdc.evt.provisioning.step_completed{step:'schedule_enable'}
   │
   ▼ schedule_pending → running ✓ (cascade complete)
```

### 6.4 Reconciliation (Tier 1/2/3)

```
Worker scheduler (mỗi 30 phút) hoặc CMS POST /api/reconciliation/check
   │
   ▼
ReconCore.AcquireLeader (Redis SET NX) — nếu thất bại, exit
   │
   ▼
ReconCore.RunTier1 (count check):
   │ Mongo: db.coll.estimatedDocumentCount()
   │ PG: SELECT pg_class.reltuples khi >10M, else COUNT(*)
   │ INSERT cdc_reconciliation_report {tier=1, source_count, dest_count, diff}
   │
   ▼
RunTier2 (ID-based diff windowed):
   │ 15 phút sliding window, 7 ngày lookback, 5 phút freeze margin
   │ Mongo: db.coll.find({ts_field:{$gte:lo,$lt:hi}}, {_id:1})
   │ PG: SELECT _gpay_source_id WHERE _source_ts BETWEEN lo AND hi
   │ Compute set diff: missing_ids[], stale_ids[]
   │ INSERT report tier=2
   │
   ▼ (nếu Tier2 phát hiện missing và CMS gọi POST /api/reconciliation/heal)
ReconHealer.HealWindow:
   │ Prefer Debezium signal nếu connector healthy → MongoCollection insert vào debezium_signal
   │ Else: db.coll.find({_id:{$in: missing_ids[]}}) batch 500 → SchemaAdapter.BuildUpsertSQL → upsert PG
   │ Audit batched (max 100 sample rows + tất cả error)
   │
   ▼ Tier3 (heavy, optional): hash-based MD5 full diff
```

### 6.5 DLQ replay

```
Kafka message fail → INSERT failed_sync_logs (status='failed', retry_count=0, max_retries=3)
   │
   ▼ DLQStateMachine 5-phút tick
SELECT * FROM failed_sync_logs WHERE next_retry_at <= NOW() AND status='failed' AND retry_count < max_retries LIMIT 100
   │ for each: NATS publish back to original Kafka subject (event flow goes through normal path)
   │ if attempt fails: retry_count++, set next_retry_at = NOW() + exp_backoff
   │ if retry_count >= max_retries: status='dead_letter'
```

### 6.6 Wizard end-to-end (Source → Master)

```
CMS UI: /source-to-master 10-step Wizard
   │
   ▼ Step 1: pick connector (cdc.gpay.public.orders, etc.)
   │ Step 2: register shadow → POST /api/v1/source-objects/register
   │   ShadowAutomator.EnsureShadowTable (sync DDL)
   │   POST /api/v1/source-objects/:id/create-default-columns (publish create-default-columns NATS)
   │
   ▼ Step 3-4: snapshot wait + ingest verify (live count via /api/v1/source-objects)
   │
   ▼ Step 5: SchemaProposals từ SinkWorker drift detection
   │   GET /api/v1/schema-proposals
   │
   ▼ Step 6: Approve proposals (per field)
   │   POST /api/v1/schema-proposals/:id/approve → ALTER TABLE shadow + create cdc_mapping_rules row
   │
   ▼ Step 7: Map fields → master_binding
   │   POST /api/mapping-rules + mapping_rule_v2
   │
   ▼ Step 8: Create master binding
   │   POST /api/v1/masters {source_object_id, master_table, transform_type, ...}
   │
   ▼ Step 9: Approve master
   │   POST /api/v1/masters/:name/approve → publish cdc.cmd.master-create → MasterDDLGenerator.Apply
   │
   ▼ Step 10: Activate
   │   POST /api/v1/wizard/sessions/:id/execute → flip status=running
   │   (transmute schedule có thể tạo manual qua /api/v1/schedules hoặc thông qua provisioning cascade)
```

---

## 7. Xác minh thực tế

Verify lúc 23:34 ngày 2026-04-30:

### 7.1 Process

```
$ ps -ef | grep -E "cdc-worker|cms-server" | grep -v grep
501 13653 1  0 Wed04PM  /tmp/cms-server  (uptime ~52h, 0:24 CPU)
501 20450 1  0 Wed04PM  /tmp/cdc-worker  (uptime ~52h, 0:25 CPU)
```

### 7.2 Containers

```
$ docker ps --format "table {{.Names}}\t{{.Status}}"
gpay-mariadb           Up 33h (healthy)
gpay-redis             Up 2d
gpay-kafka-connect     Up 2d (healthy)
gpay-schema-registry   Up 2d
gpay-mongo             Up 2d (healthy)
gpay-kafka             Up 2d
gpay-nats              Up 2d
gpay-postgres-cdc      Up 2d (healthy)
gpay-postgres-source   Up 2d (healthy)
gpay-postgres          Up 2d (healthy)
gpay-postgres-dest     Up 2d (healthy)
```

11/11 container `Up`. CMS + Worker process còn sống. Không phải báo cáo láo.

---

## 8. Khoảng trống đã biết & rủi ro

Tham chiếu `10_gap_analysis_track_e.md` — Track E (data plane addtest) còn 6 blocker mở:

| # | Blocker | Tác động | Trạng thái |
|---|---|---|---|
| B1 | `profile_status='draft'` chặn transmute | Scanner skip 0 rows | **FIXED** session trước |
| B2 | `shadow_binding.ddl_status='pending'` chặn ingest | BatchBuffer skip | **FIXED** session trước |
| B3 | `source_locator_json` không trỏ tới physical addtest | Logical-clone design ambiguity Mongo | OPEN — chờ architect |
| B4 | SchemaValidator drift toàn bộ `cdc.gpay.public.orders` | Tất cả ingest fail (KHÔNG chỉ addtest) | **CRITICAL** OPEN |
| B5 | DLQ `failed_sync_logs.raw_json` reject UTF8 0x00 | Avro bytes có 0x00 → PG `SQLSTATE 22021` → infinite redeliver | **CRITICAL** OPEN |
| B6 | Transmute scanner query `_gpay_id` từ shadow tables | Shadow chưa có `_gpay_id` (master col) | OPEN |
| B7 | `dw_orders.orders_fact` PK `_gpay_id` collision | Re-run transmute fail không ON CONFLICT | OPEN (pre-existing) |
| B8 | Debezium MySQL plugin chưa cài | KHÔNG tạo được connector cho MariaDB live | OPEN — cần infra |

**Test data residue**: `goopay_source.public.orders` còn 3 rows ids 56-58 với `notes LIKE 'track-e-test-%'` đang stuck DLQ redelivery loop (do B5).

**Triage options** (xem `10_gap_analysis_track_e.md`):
- A: PG addtest only (~2-4h fix B4+B5+B6)
- B: Full Track E (~1d, gồm install MySQL plugin)
- C: Defer + architect decision

---

## 9. Skills sử dụng

Tuân thủ CLAUDE.md:
- **§3 Plan & Verify** — gather data trước, không đoán
- **§7 Memory Retention** — đã đọc `lessons.md` đầu phiên (qua summary), workspace context, gap analysis trước khi tổng hợp
- **§9 Workspace-First** — file lưu trong `agent/memory/workspaces/feature-cdc-integration/`
- **§11 No Overwrite** — file mới `report_system_summary.md`, KHÔNG sửa file existing
- **§12 Brain Code Prohibition** — báo cáo này chỉ research, không sửa source

**Tools/agents**:
- `Bash` — verify ps/docker/timestamp
- `Agent (Explore × 3 song song)` — research worker / cms / data plane
- `Read` — context files (gap analysis, /tmp scripts)
- `Write` — tạo file report duy nhất

**Verification trước "done"**:
- Process check: PID 13653 + 20450 còn sống ✓
- Container check: 11/11 Up ✓
- Report file viết xong tại `agent/memory/workspaces/feature-cdc-integration/report_system_summary.md`

---

## 10. Kết luận

Hệ thống CDC gồm **2 binary Go** (cdc-worker :8082, cdc-cms-service :8083), **1 frontend React** (:5173), và **11 container hạ tầng** (Postgres ×4, MariaDB, Mongo, Kafka, Connect, Schema Registry, NATS, Redis).

- **Data plane**: Debezium (PG/Mongo/MariaDB) → Kafka → cdc-worker → shadow tables (`shadow_*` ở cdc_dw) → master tables (`dw_*` ở goopay_dest)
- **Control plane**: 13 bảng `cdc_system.*` + 4 bảng partitioned (`failed_sync_logs`, `cdc_activity_log`, `admin_actions`, …)
- **NATS bus**: ~30 subjects (cdc.cmd.* / cdc.evt.* / cdc.result.*) — toàn bộ CMS publish, worker subscribe
- **State machine** (provisioning): 4 transitions (`shadow_bind → master_bind → discover → schedule_enable`), event-driven qua `cdc.evt.provisioning.step_completed`
- **API**: 70+ endpoints CMS chia 4 lớp permission (Public / Shared / OpsAdmin destructive / Admin write); JWT HS256 + RBAC + Idempotency-Key + Audit reason ≥10
- **Reconciliation**: Tier 1 (count) / Tier 2 (windowed ID diff) / Tier 3 (hash) — 30 phút schedule, Redis leader election
- **DLQ**: write-before-ACK, 3 retries với exponential backoff, dead_letter terminal

Ingest live đang gặp 6 blocker (B3-B8) — đã document rõ trong gap analysis. Control plane (provisioning + V1/V2 bridge + Master DDL ALTER pass) đã verify end-to-end.
