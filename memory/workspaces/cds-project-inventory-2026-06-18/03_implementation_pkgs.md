# 03_implementation_pkgs.md — pkgs/ (Shared Packages)

---

## pkgs/crypto/
- Encryption utilities (AES, HMAC)

## pkgs/database/ (7 files)
| File | Mô tả |
|---|---|
| `postgres.go` | `NewPostgresConnection(cfg)` — khởi tạo GORM + pgx pool |
| `pgx_pool.go` | `NewPgxPool(dsn)` — raw pgx pool (không qua GORM) |
| `multi.go` | `MultiDB` — manage nhiều DB connections đồng thời |
| `per_source_pool.go` | Pool riêng cho từng source DB |
| `gorm_logger.go` | Custom GORM logger (zap integration) |
| `metrics_callback.go` | Prometheus metrics cho DB operations |
| `metrics_callback_test.go` | Tests |

## pkgs/idgen/ (1 file)
- `Init(logger)` — init Sonyflake với auto machine ID
- `InitWithMachineID(id)` — init với explicit machine ID (sinkworker)
- `NewID()` — generate unique distributed ID (uint64)
- Phòng ID collision qua fencing token mechanism

## pkgs/kafka/ (1 file)
| File | Mô tả |
|---|---|
| `avro.go` | Avro encoder/decoder cho Kafka Schema Registry |

## pkgs/metrics/ (inferred)
- `StartMetricsServer(ctx, port, logger)` — expose Prometheus metrics trên port 9090

## pkgs/mongodb/ (inferred)
- MongoDB client factory

## pkgs/natsconn/ (inferred)
- `NewNatsClient(cfg, logger)` — NATS connection với auto-reconnect
- Returns wrapper struct với `.Conn *nats.Conn`

## pkgs/observability/ (3 files)
| File | Functions | Mô tả |
|---|---|---|
| `otel.go` | `InitOtel(cfg, logger)` | Init OpenTelemetry (traces + metrics + logs) → SigNoz |
| `otel.go` | `LogProvider()` | Get OTel log provider |
| `otel.go` | `NewOTelBridgeCore(...)` | Tee zap logs → OTel (severity-aware sampler) |
| `trace_helpers.go` | `StartSpan(ctx, name)`, `SetError(span, err)` | Span helpers |
| `log_template.go` | Log format templates | - |

## pkgs/rediscache/ (inferred)
- Redis client factory (go-redis/v9)

## pkgs/utils/
- Common utilities

---

## internal/admin/ — Admin HTTP Server (4 files)

| File | Mô tả |
|---|---|
| `server.go` | `NewServer(deps)` + `Run(ctx, addr)` — Gin HTTP server với Bearer token auth |
| `helpers.go` | Helper functions cho admin API responses |
| `source_register.go` | Routes cho quản lý source connections |
| `types.go` | Request/Response types |

**Deps inject vào server:**
- `DB *gorm.DB`
- `NATS *nats.Conn`
- `DebeziumBaseURL string`
- `SchemaRegistryURL string`
- `AuthToken string`
- `Logger *zap.Logger`

---

## internal/server/worker_server.go (57,505 bytes — 1245 dòng)

**Struct**: `WorkerServer` — core wiring của toàn bộ worker service

| Function | Mô tả |
|---|---|
| `NewWorkerServer(cfg, logger)` | Khởi tạo: DB, NATS, Kafka, tất cả services và handlers |
| `Start()` | Start Fiber HTTP server + subscribe tất cả NATS subjects + cron jobs |
| `Shutdown()` | Graceful shutdown: drain NATS, close DB, stop cron |
| `runBridgeCycle(t, targetTable)` | Cron: chạy bridge cycle |
| `runTransformCycle(now, targetTable)` | Cron: chạy transform cycle |
| `runPartitionCheck(now)` | Cron: check/drop old partitions |
| `runReconcileCycle(now)` | Cron: trigger reconciliation |
| `collectPoolStats(pools)` | Aggregate consumer pool metrics |
| `redactDSN(dsn)` | Redact password từ DSN string |

**NATS Subjects subscribed (WorkerServer):**
- `cdc.command.standardize`
- `cdc.command.discover`
- `cdc.command.backfill`
- `cdc.command.master_swap`
- `cdc.command.create_default_columns`
- `cdc.command.batch_transform`
- `cdc.command.scan_raw_data`
- `cdc.command.scan_array_fields`
- `cdc.command.periodic_scan`
- `cdc.command.scan_fields`
- `cdc.command.sync_register`
- `cdc.command.sync_state`
- `cdc.command.restart_debezium`
- `cdc.command.alter_column`
- `cdc.command.drop_gin_index`
- `cdc.command.discover_mongo_databases`
- `cdc.command.discover_mongo_collections`
- `cdc.recon.check`
- `cdc.recon.heal`
- `cdc.recon.retry_failed`
- `cdc.recon.debezium_signal`
- `cdc.recon.backfill_source_ts`
- `cdc.recon.detect_timestamp`
- `cdc.transmute.shadow`
- `cdc.transmute.execute`
- `cdc.provisioning.step_completed`
- `cdc.provisioning.shadow_bind`
- `cdc.provisioning.schedule_enable`
- `cdc.master.alter_column`
- `cdc.master.create`
- `cdc.snapshot.run`

---

## internal/sinkworker/ (4 files)

| File | Mô tả |
|---|---|
| `sinkworker.go` | `SinkWorker` — nhận Kafka message, parse, gọi schema manager, upsert |
| `schema_manager.go` | `SchemaManager` — manage shadow table schema cho sink path |
| `envelope.go` | Parse Debezium envelope format (before/after/op/source) |
| `upsert.go` | UPSERT logic vào shadow table |

---

## internal/naming/ (1 file)
- `naming.go` — conventions cho naming: table names, column names, connector names

## internal/activity/ (1 file)
- `taxonomy.go` — taxonomy/enum cho activity types (DISCOVER, STANDARDIZE, BACKFILL, ...)

## migrations/dest/ (1 file)
- `001_dest_init.sql` — Init migration cho destination DB (tạo cdc_system schema)

## config/ (4 files)
| File | Mô tả |
|---|---|
| `config.go` | `AppConfig` struct + `NewConfig()` — load từ YAML + env vars |
| `config-local.yml` | Local development config |
| `config-production.yml` | Production config template |
| `config-sample.yml` | Sample config |

**AppConfig top-level fields:**
- `Server`, `DBPool`, `SystemDB`, `ShadowDB`, `MasterDB`, `ReadReplica`
- `MasterKey`, `MaskingHMACKey`, `MaskingAESKey`
- `Nats`, `Redis`, `Worker`, `JWT`, `Kafka`, `Otel`, `MongoDB`, `Debezium`
- `ConnectionOverrides map[string]string`
- `Debug`
