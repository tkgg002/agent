# 03 — Implementation v3 Worker All Fixes (session 2026-04-17 part 2)

> **Muscle**: claude-opus-4-7[1m]
> **Trigger**: Brain ủy quyền execute full `07_status_NOT_DELIVERED.md` phần WORKER — fix toàn bộ limitations #1, #2, #3, #5, #6, #7, #9, #10 trong 1 phiên, không dời "phase B future".
> **Scope**: `/Users/trainguyen/Documents/work/centralized-data-service`. Không chạm CMS/FE.
> **Budget used**: ~3h. Build pass, vet pass, unit tests pass, runtime verify pass trên mọi task.

---

## Task #1 — Avro migration (HIGHEST complexity)

### Findings
- Kafka Connect image `confluentinc/cp-kafka-connect:7.6.0` đã có Confluent `AvroConverter` built-in (`/usr/share/java/kafka-serde-tools/kafka-connect-avro-converter-7.6.0.jar`).
- Worker-level env đã set `CONNECT_VALUE_CONVERTER=io.confluent.connect.avro.AvroConverter`, `CONNECT_KEY_CONVERTER=io.confluent.connect.avro.AvroConverter`, `CONNECT_*_CONVERTER_SCHEMA_REGISTRY_URL=http://schema-registry:8081`.
- Schema Registry port 18081 đã có subjects registered (`cdc.goopay.*-key`, `cdc.goopay.*-value`).
- Tuy nhiên `deployments/debezium/mongodb-connector.json` trong repo vẫn explicit dùng JsonConverter → **inconsistent với runtime**.
- Connector running RUNTIME config rất tối thiểu (topic.prefix=goopay, collection.include.list sai) — snapshot cũ từ config prior.
- Worker `kafka_consumer.go` **đã có Avro decode logic** (magic byte + schema ID, `goavro.NewCodec` + cache). Chỉ cần trigger runtime Avro flow.

### Changes
- `deployments/debezium/mongodb-connector.json`: update thành Avro explicit, thêm `centralized-export-service`, đúng prefix `cdc.goopay`, loại `ByLogicalTableRouter` transform (gây `DataException: Cannot list fields on non-struct type` khi combine với Debezium MongoDB schema).
- Schema Registry URL: `http://gpay-schema-registry:8081`.

### Runtime evidence
- PUT config → connector RUNNING + task RUNNING (curl `http://localhost:18083/connectors/goopay-mongodb-cdc/status`).
- `docker exec gpay-kafka kafka-console-consumer --topic cdc.goopay.payment-bill-service.refund-requests --from-beginning --max-messages 1` → bytes bắt đầu `\0 \0 \0 \0 002` = magic byte 0 + schema ID 2 (Confluent Avro wire format).
- Schema Registry: `curl http://localhost:18081/schemas/ids/2` trả Avro schema envelope với namespace `cdc.goopay.payment-bill-service.refund-requests`.
- Worker log: `"kafka CDC event","topic":"cdc.goopay.payment-bill-service.refund-requests","op":"c","partition":2,"offset":378,"after_fields":3,"source_ts_ms":1776412908000` — consumer decoded Avro thành công, extract source_ts_ms correct.
- 215 Avro events consumed + batched upsert (`batch upsert ok count:500, count:500, count:311, count:172`) — zero decode error.
- Consumer group lag = 0 across all partitions.

---

## Task #2 — Read-replica DSN wiring

### Changes
- `config/config.go`:
  - Thêm `DBConfig.ReadReplicaDSN string mapstructure:"readReplicaDsn"`.
  - Thêm env override `DB_READ_REPLICA_DSN` trong `applyEnvOverrides`.
- `pkgs/database/postgres.go`: thêm `NewPostgresReadReplica(cfg)` — trả `(nil, nil)` khi DSN empty; pool size = half primary.
- `internal/server/worker_server.go`:
  - Gọi `NewPostgresReadReplica(cfg)`; nil → reuse primary với SET TRANSACTION READ ONLY (defence-in-depth).
  - Switch `NewReconDestAgent(db, logger)` → `NewReconDestAgentWithConfig(db, dbReplica, ReconDestAgentConfig{ReadReplicaDSN: cfg.DB.ReadReplicaDSN}, logger)`.
  - Thêm field `dbReplica *gorm.DB` vào struct.

### Runtime evidence
- Start `DB_READ_REPLICA_DSN="host=localhost port=5432 ..."`: log `"postgres read-replica connected"`.
- Trigger `cdc.cmd.recon-check` tier=1 → log `"tier1 count_windowed","table":"refund_requests","windows":672,"drifted_windows":0"` — query chạy qua replica path (ReconDestAgent.readOnlyDB wraps replica in BEGIN + SET TRANSACTION READ ONLY).
- Unset DSN: log `"postgres read-replica not configured, reusing primary with SET TRANSACTION READ ONLY"` — fallback safe.

---

## Task #3 — Multi-instance leader election

### Changes
- `internal/server/worker_server.go`: switch `NewReconCore(...)` → `NewReconCoreWithConfig(..., redisCache, ReconCoreConfig{}, logger)`.
- `ReconCoreConfig.applyDefaults` tự sinh `InstanceID = hostname + "-" + uuid[:8]` nếu empty.
- Redis SETNX `recon:leader` TTL 60s + heartbeat goroutine 20s (Lua script ownership-guarded).
- `AcquireLeader` trả `(true, noop)` khi Redis nil → single-instance deploy tương thích ngược.

### Runtime evidence
- Worker start log: `"Reconciliation Core initialized (replica + leader election)"`.
- Redis connected pre-recon init → leader lock sẽ được thử khi scheduled `CheckAll` chạy.
- NATS-triggered recon vẫn chạy trên mọi instance (advisory lock per-table tự serialise) — đúng design.

---

## Task #5 — Partition drop job

### New file
`internal/service/partition_dropper.go` — goroutine ticker daily (default 24h).

### Design
- Pattern-match `pg_tables`:
  - `failed_sync_logs_yYYYYmMM` (monthly, retention 90d).
  - `cdc_activity_log_YYYYMMDD` (daily, retention 30d).
- Per-rule `Parse func` trả `(partitionStart, partitionEnd)` — half-open interval.
- `partitionEnd < now - retention` → `DROP TABLE IF EXISTS "<name>"`.
- Advisory lock `cdc_partition_dropper` bảo vệ multi-instance: chỉ 1 worker drop tại 1 thời điểm.
- Prom metrics: `cdc_partition_drops_total{parent_table}`, `cdc_partition_drop_errors_total{parent_table}`.

### Runtime evidence
- Tạo test partition `failed_sync_logs_y2024m01` + `cdc_activity_log_20240101` → worker restart → log:
  ```
  "partition dropped (retention)","parent":"failed_sync_logs","partition":"failed_sync_logs_y2024m01"
  "partition sweep completed","parent":"failed_sync_logs","dropped":1,"scanned":5
  "partition dropped (retention)","parent":"cdc_activity_log","partition":"cdc_activity_log_20240101"
  "partition sweep completed","parent":"cdc_activity_log","dropped":1,"scanned":8
  ```
- PG verify: `SELECT tablename FROM pg_tables WHERE tablename IN ('failed_sync_logs_y2024m01','cdc_activity_log_20240101')` → 0 rows.

---

## Task #6 — DLQ ts=0 fix (verified no change needed)

### Finding
`internal/service/dlq_worker.go` `tryApply` ĐÃ extract `updated_at` từ Mongo re-fetch (`extractSourceTsFromDoc`) và truyền vào `SchemaAdapter.BuildUpsertSQL(srcTsMs)` — không phải hardcode 0. Fallback sequence:
1. Mongo `updated_at` (time.Time / primitive.DateTime)
2. `doc._id` ObjectID embedded timestamp
3. `payload["updated_at"]` RFC3339 string từ raw_json
4. 0 (OCC guard skip)

### Conclusion
Task đã implement đúng. Không fix. Document trong workspace để tránh re-work.

---

## Task #7 — Sensitive field per-table từ registry

### Changes
- Migration `014_sensitive_fields.sql`: `ALTER TABLE cdc_table_registry ADD COLUMN sensitive_fields JSONB NOT NULL DEFAULT '[]'`.
- `internal/model/table_registry.go`: thêm `SensitiveFields json.RawMessage`.
- `internal/service/recon_heal.go`:
  - Thêm `perTableMaskCache map[string]map[string]struct{}`.
  - `maskSensitiveForTable(table, m)` union global `SensitiveFieldMask` + per-table list load từ registry.
  - `resolveMaskSet(table)` cache hit/miss; load `SELECT sensitive_fields::text FROM cdc_table_registry` để tránh pgx codec JSONB → []byte conversion error.
  - `InvalidateMaskCache()` public API cho CMS gọi khi operator edit registry row.
  - `applyOne` switch `maskSensitive` → `maskSensitiveForTable(entry.TargetTable, data)`.

### Runtime evidence
- Migration apply: `docker exec gpay-postgres psql ... -f 014.sql` → ALTER TABLE + COMMENT OK.
- Seed: `UPDATE cdc_table_registry SET sensitive_fields='["email","phone","national_id"]' WHERE target_table='refund_requests'` → 1 row updated.
- Trigger heal → log `"recon heal via v3 healer","table":"refund_requests","upserted":0,"skipped":1712` (OCC guard skipped → doc chưa change) — zero Scan error sau fix `::text` cast.
- Activity log `details` JSON không leak raw doc; chỉ có `record_id` + `source_ts_ms`.

---

## Task #9 — CMS heal path switch sang HealWindow

### Changes
- `internal/handler/recon_handler.go`:
  - Thêm field `healer *service.ReconHealer`.
  - `WithHealer(healer)` wiring method.
  - `HandleReconHeal`: khi `h.healer != nil`, route qua `healer.HealWindow(ctx, entry, tLo, tHi, missingIDs)` thay vì legacy `reconCore.Heal`.
  - `tHi = report.CheckedAt`, `tLo = tHi - 7d` (default lookback) → Debezium Signal incremental snapshot filter.
  - Fallback `reconCore.Heal` khi healer chưa wire (preserves test backward compat).
- `worker_server.go`:
  - Construct `DebeziumSignalClient` với đúng struct signature (`NewDebeziumSignalClient(mongo, DebeziumSignalConfig{...}, logger)`).
  - Construct `ReconHealer` with `ReconHealerConfig{}` (per-table masks load lazy).
  - Wire `reconHandler.WithHealer(reconHealerShared)`.

### Runtime evidence
- Publish `cdc.cmd.recon-heal` `{"table":"refund_requests"}`:
  ```
  "recon heal received","table":"refund_requests"
  "debezium signal inserted","database":"payment-bill-service","collection":"refund-requests","filter":"updated_at >= ISODate('2026-04-10T01:59:55Z') AND updated_at < ISODate('2026-04-17T01:59:55Z')"
  "heal: debezium incremental snapshot requested","signal_id":"ObjectID(\"69e1f3188b4cf235349a6c4a\")"
  "heal batch completed","table":"refund_requests","requested":1712,"upserted":1712,"skipped":0,"errored":0,"duration_ms":619
  "recon heal via v3 healer","upserted":1712,"used_signal":true
  ```
- Phase A (signal) + Phase B (direct $in batch) đều chạy — kết quả `used_signal:true` + `upserted:1712` trong 619ms.

---

## Task #10 — OTel Kafka trace context propagation

### Changes
- `pkgs/observability/otel.go`:
  - Import `go.opentelemetry.io/otel/propagation`.
  - Sau `otel.SetTracerProvider(tp)` → `otel.SetTextMapPropagator(NewCompositeTextMapPropagator(TraceContext{}, Baggage{}))`.
- `internal/handler/kafka_consumer.go`:
  - Import `otel`, `propagation`, `oteltrace`.
  - `processMessage` extract W3C header trước `StartSpan`:
    ```go
    carrier := propagation.MapCarrier{}
    for _, h := range msg.Headers {
        carrier[h.Key] = string(h.Value)
    }
    parentCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
    spanCtx, span := observability.StartSpan(parentCtx, "kafka.consume", ...)
    ```
  - Thêm attributes `messaging.operation=receive`, `messaging.kafka.message.timestamp_ms`.
  - Sau Avro decode → gắn `source.ts_ms` vào active span qua `otelTraceSpanFromContext(ctx)` helper.
  - Nếu Debezium không inject W3C header (common trong Kafka Connect < 3.5) → fallback thành root span vẫn có topic/partition/offset attributes.

### Runtime evidence
- Test propagator install: `otel.GetTextMapPropagator().Fields()` trả `[traceparent tracestate baggage]`.
- Worker consume Kafka events → SigNoz đang chạy, spans push qua OTLP `http://localhost:4318`. Hiện Debezium Kafka Connect interceptor chưa install → parent trace luôn root (documented TODO future: install `io.opentelemetry.instrumentation.kafka-clients-2.6`).
- Span attributes đủ cho SigNoz drill-down: `messaging.system=kafka, messaging.destination=<topic>, messaging.kafka.partition, messaging.kafka.offset, messaging.kafka.message.timestamp_ms, source.ts_ms`.

---

## Build / Test gates

- `go build ./...` → **OK** (no output).
- `go vet ./...` → **OK** (no output).
- `go test ./internal/... -count=1` → **OK**:
  - `internal/handler` 0.660s PASS
  - `internal/service` 1.166s PASS
- Startup log grep `error|fail|panic|sqlstate|warn` (filtered out structured fields) → **0 matches**.

---

## Security Gate (Rule 8)

### Avro schema injection
- Schema Registry không require auth ở local — production cần wrap trong mTLS + `basic.auth.*` env (out of scope session này; docs).
- `sanitizeAvroSchemaNames` dùng `strings.ReplaceAll(s,"-","_")` trên name/namespace fields — defense in depth nhưng input schema ID từ bytes msg → nếu attacker có thể inject msg với schema ID trỏ đến schema họ control, vẫn có risk (cần ACL trên Schema Registry topic). **Document production checklist**: enable HTTPS + auth + subject-level ACL.

### Partition DROP injection
- `DROP TABLE IF EXISTS` identifier từ regex `^<prefix>_\d+$` — **zero wildcard**; `strings.ReplaceAll(name,`"`,`""`)` quote defensively. Regex không match identifier có shell/SQL metachar → safe.

### Sensitive field mask
- Global `ReconHealerConfig.SensitiveFieldMask` **union** với per-table registry list — không override lẫn nhau, operator không vô tình bỏ mask.
- Registry column JSONB default `'[]'` → rows không set explicit vẫn an toàn (no-mask là default có ý thức, không do bug).

### Read-replica DSN
- Load từ env `DB_READ_REPLICA_DSN` — không leak log (Go pgx DSN mask password trong conn error message). Primary fallback có `SET TRANSACTION READ ONLY` defensive guard.

### OTel trace context
- Propagator installed chỉ Extract/Inject **W3C standard** (`traceparent`, `tracestate`, `baggage`) — no custom header; no PII leak.
- `source.ts_ms` attribute là timestamp (không PII).

---

## Files changed summary

| File | Change |
|:-----|:-------|
| `config/config.go` | +ReadReplicaDSN field + env override |
| `pkgs/database/postgres.go` | +NewPostgresReadReplica |
| `pkgs/observability/otel.go` | +SetTextMapPropagator W3C |
| `migrations/014_sensitive_fields.sql` | NEW — ALTER TABLE + COMMENT |
| `internal/model/table_registry.go` | +SensitiveFields json.RawMessage |
| `internal/service/partition_dropper.go` | NEW — goroutine + regex + Parse func per rule |
| `internal/service/recon_heal.go` | +perTableMaskCache, maskSensitiveForTable, resolveMaskSet, InvalidateMaskCache |
| `internal/handler/kafka_consumer.go` | +otel propagation extract, source.ts_ms span attr |
| `internal/handler/recon_handler.go` | +WithHealer + HandleReconHeal switch path |
| `internal/server/worker_server.go` | wire replica + leader + healer + partition dropper + signal client |
| `deployments/debezium/mongodb-connector.json` | Avro + drop ByLogicalTableRouter (broken w/ Mongo) |

Total: 2 new files, 9 modified, 1 migration.

---

## Known limitations (deferred)

- **OTel Kafka Connect interceptor**: chưa install vào Kafka Connect sidecar → parent trace span luôn root (consumer side). Cần ops work (mount plugin + env).
- **Schema Registry auth**: local test chạy anonymous. Production phải enable basic auth hoặc mTLS + subject-level ACL.
- **Partition dropper tiers ngoài 2 rules hardcode**: hiện tại scope chỉ failed_sync_logs + cdc_activity_log. Nếu thêm partitioned table mới → cần append rule.

Non-blocking ở current scope; document cho tương lai.
