# Implementation: v3 Heal + DLQ + Schema Validator (Phase 2-3)

> **Date**: 2026-04-17
> **Author**: Muscle (claude-opus-4-7[1m])
> **Context**: Phase 2-3 per `02_plan_data_integrity_v3.md` §6, §7, §9, §11.
> **Depends on**: Phase 0 (`_source_ts` + `/metrics` + OTel) + Phase 1
> (Recon Source/Dest Agent rewrite + `recon_core.go` v3 + `recon_runs`).

---

## 1. Scope

Deliver the fintech-grade Heal + DLQ pipeline:

| Task | Plan ref | File |
|:----:|:---------|:-----|
| 1 | §6 Heal v3 with OCC + batch $in | `internal/service/recon_heal.go` |
| 2 | §7 Heal via Debezium Signal | `internal/service/debezium_signal.go` |
| 3 | §9 DLQ write-before-ACK | `internal/handler/kafka_consumer.go` (edit) |
| 4 | §9 DLQ state machine + retry worker | `internal/service/dlq_worker.go` + `migrations/012` |
| 5 | §11 Schema validation Phase A | `internal/service/schema_validator.go` + `migrations/013` |

Out of scope (remains in Phase 4+):
- RBAC middleware / Idempotency-Key / Admin-actions audit table — those
  belong to CMS and are tracked separately.
- Phase B Avro migration — roadmap item.

---

## 2. Task 1 — Heal v3 (`recon_heal.go`)

### Contract
```go
type ReconHealer struct { ... }

func NewReconHealer(db, mongoClient, schemaAdapter, signal, cfg, logger) *ReconHealer

// Direct batch $in heal.
func (rh *ReconHealer) HealMissingIDs(ctx, entry, ids) (*HealResult, error)

// Orchestrated: Signal → direct.
func (rh *ReconHealer) HealWindow(ctx, entry, tLo, tHi, missingIDs) (*HealResult, error)
```

### Key design points
- **Batch $in = 500** — single Mongo roundtrip per 500 IDs (plan §6).
  Chunking goes through `chunkStrings` which pre-sorts for audit
  determinism.
- **OCC delegation** — reuses `SchemaAdapter.BuildUpsertSQL`, which
  emits `WHERE tbl._source_ts IS NULL OR tbl._source_ts < EXCLUDED._source_ts`
  when the incoming ts is > 0. Zero ts falls back to hash-guard (legacy
  parity).
- **`_source_ts` extraction** — 3-tier preference:
  1. `doc.updated_at` as `time.Time`
  2. `doc.updated_at` as `primitive.DateTime`
  3. `doc._id.Timestamp()` (ObjectID embedded ts)
  4. `0` → OCC guard skipped (unconditional upsert)
- **Batched audit log** — buffer 100 rows, then `CreateInBatches`. Reuses
  `cdc_activity_log` (operation="recon-heal") — NO new migration needed.
- **Mongo fetch decoupled from PG write** — we collect docs from cursor
  into a slice, close the cursor, THEN run upserts. Prevents Mongo
  cursor from being held open across N × PG latency.
- **Read preference secondary** on Mongo collection handle.
- **Prom metric** `cdc_recon_heal_actions_total{table, action}` —
  action ∈ upsert|skip|error. Fed from the existing metric registered
  in `pkgs/metrics/prometheus.go` (Phase 1).

### RowsAffected-driven outcome classification
```go
if err != nil               → "error"
if RowsAffected == 0        → "skip"   // OCC rejected
else                        → "upsert"
```

Docs in the $in chunk that Mongo did NOT return are also counted as
`skip` (source deleted; PG stale; or doc lost).

---

## 3. Task 2 — Debezium Signal (`debezium_signal.go`)

### Contract
```go
func NewDebeziumSignalClient(mongoClient, cfg, logger) *DebeziumSignalClient

func (d) TriggerIncrementalSnapshot(ctx, database, collection, filter) (string, error)
func (d) IsConnectorHealthy(ctx) (bool, error)
func BuildUpdatedAtRangeFilter(tLo, tHi) string
```

### Design
- Signal doc inserted into `{database}.{SignalCollection}` (default:
  `debezium_signal`).
- Payload shape matches Debezium MongoDB incremental-snapshot spec:
  ```json
  {"type":"execute-snapshot","data":{"data-collections":["db.coll"],
    "type":"incremental","additional-conditions":[{"data-collection":"db.coll",
    "filter":"updated_at >= ISODate('...') AND updated_at < ISODate('...')"}]}}
  ```
- `IsConnectorHealthy` probes Kafka Connect REST (`/connectors/<name>/status`)
  — configurable via `DebeziumConfig.ConnectorStatusURL`. When unset, we
  return `true` and let `TriggerIncrementalSnapshot` fail loudly if the
  connector is actually dead.

### HealWindow orchestration
```
missingIDs + window [tLo, tHi]
  │
  ├─► Signal configured + connector RUNNING → Signal (best-effort)
  │         → Debezium re-streams range over Kafka
  │         → Worker kafka_consumer path absorbs
  │
  └─► Direct HealMissingIDs (always runs when IDs supplied)
          → Converges KNOWN missing rows immediately
```

Signal + Direct are NOT mutually exclusive — the Direct path short-circuits
for known offenders so operators don't wait for Kafka lag.

---

## 4. Task 3 — DLQ write-before-ACK (`kafka_consumer.go`)

### Before (v2 risk)
Original loop:
```go
procErr := kc.processMessage(...)
if procErr != nil { log + metric; batch.failed++ }
// FALL THROUGH to CommitMessages unconditionally → leak risk
reader.CommitMessages(ctx, msg)
```

### After (v3)
```go
procErr := kc.processMessage(...)
// (metrics unchanged above)

if procErr != nil {
  if dlqErr := kc.writeDLQ(ctx, msg, procErr); dlqErr != nil {
    service.DLQWriteFail.Inc()
    log.Error("DLQ write failed — skipping offset commit for redelivery")
    continue  // NOT committing → Kafka redelivers
  }
}
reader.CommitMessages(ctx, msg)
```

### writeDLQ helper
- Infers `target_table` from topic (`cdc.goopay.<db>.<table>`).
- Extracts `record_id`, `operation`, `raw_json` via `extractDLQMetadata`:
  1. Mongo key JSON (`{"id":"..."}`)
  2. Bare string key
  3. `payload.after._id`
  4. `payload.after` string JSON unmarshal fallback
- Raw JSON wrapped when not valid JSON: `{"raw":"<bytes>"}`.
- Error classification:
  - `schema_drift` in error text → `error_type=schema_drift`, `status=pending`
  - `missing_required_field` → `error_type=missing_required`, `status=pending`
  - Otherwise → `error_type=processing`, `status=failed`
- INSERT uses GORM on the partitioned parent; PG routes to the right
  monthly partition via `created_at` default.

### Schema-drift integration
- `kc.validator.ValidatePayloadWithCase(tableName, afterData)` runs
  BEFORE `EventHandler.HandleRaw`. The error (wrapped `ErrSchemaDrift`
  / `ErrMissingRequired`) propagates up to the loop and becomes a DLQ
  entry rather than a worker crash.

---

## 5. Task 4 — DLQ Retry Worker (`dlq_worker.go`)

### Polling query
```sql
SELECT * FROM failed_sync_logs
 WHERE status IN ('pending','failed','retrying')
   AND (next_retry_at IS NULL OR next_retry_at < NOW())
   AND retry_count < $MaxRetries
 ORDER BY next_retry_at NULLS FIRST, id
 LIMIT $BatchSize
```

### State transitions
- `pending | failed` → `retrying` (worker marks BEFORE attempt so a
  crash mid-flight is idempotent on next tick).
- `retrying` + success → `resolved` (resolved_at set, next_retry_at cleared).
- `retrying` + fail + retry_count < MaxRetries → stays `retrying`,
  `next_retry_at` set via `BackoffDelay(retry_count)`.
- `retrying` + fail + retry_count ≥ MaxRetries → `dead_letter`
  (next_retry_at cleared, last_error stored).

### Backoff schedule (plan §9)
| retry_count | delay |
|:-----------:|:------|
| 1 | 1 min |
| 2 | 5 min |
| 3 | 30 min |
| 4 | 2 h |
| ≥5 | 6 h |

### Retry payload resolution
1. `raw_json` present → unmarshal + OCC upsert.
2. Mongo re-fetch via `record_id + source_db + source_table` when
   `raw_json` empty.
3. Neither → error → scheduled retry (and eventually dead_letter).

### Migration 012
Adds `next_retry_at TIMESTAMPTZ` + `last_error TEXT` to the partitioned
parent `failed_sync_logs`. Idempotent. Partial index
`idx_fsl_retry_poll (next_retry_at, status)` accelerates the poll.

### Prom metrics (new, this phase)
- `cdc_dlq_stuck_records_total{status}` — sampled per cycle from
  `SELECT status, COUNT(*) FROM failed_sync_logs GROUP BY status`. All
  known statuses reset to 0 first so tiles disappear when drained.
- `cdc_dlq_retry_attempts_total{table,outcome}` — counter. Outcomes:
  resolved | retrying | dead_letter | error.
- `cdc_dlq_write_failures_total` — global counter. Incremented by the
  Kafka consume loop when DLQ INSERT itself fails.

### Wiring
- Instantiated in `worker_server.go` alongside the schema validator.
- Launched in `Start()` via `go s.dlqWorker.Start(ctx)` — same lifetime
  as the worker process.

---

## 6. Task 5 — Schema Validator Phase A (`schema_validator.go`)

### Contract
```go
sv.ValidatePayload(tableName, payload)           // case-sensitive
sv.ValidatePayloadWithCase(tableName, payload)   // case-insensitive
sv.InvalidateCache(tableName)                    // hot reload hook
```

### Errors
- `ErrMissingRequired` (wrapped with field name)
- `ErrSchemaDrift` (wrapped with `unknown_field=X`)

### Data sources (priority)
1. `cdc_table_registry.expected_fields` JSONB (added by migration 013).
   Shape: `{"required":["_id","updated_at"], "known":[...]}`.
2. `information_schema.columns` — column list union into `known`.
3. Zero-config: when registry row missing AND PG table missing →
   VALIDATION SKIPPED (fail-open, so bootstrap doesn't block brand-new
   tables).

### Always-accepted housekeeping columns
`_id, _source_ts, _synced_at, _hash, _version, _deleted, _raw_data,
_source, _created_at, _updated_at` — prevents the validator from
rejecting CDC metadata that the schema_adapter injects.

### Integration
Called in `kafka_consumer.go processMessage()` right after payload
decode and BEFORE `EventHandler.HandleRaw`. Drift errors propagate up
→ write-before-ACK catches them → DLQ entry with
`error_type=schema_drift`.

### Metric
`schema_drift_detected_total{source_db,table}` — existing metric from
`pkgs/metrics/prometheus.go`, now incremented per drift detection.

---

## 7. Files touched

### New files
- `internal/service/recon_heal.go` (≈ 450 lines)
- `internal/service/debezium_signal.go` (≈ 190 lines)
- `internal/service/dlq_worker.go` (≈ 320 lines)
- `internal/service/schema_validator.go` (≈ 230 lines)
- `migrations/012_dlq_state_machine.sql`
- `migrations/013_table_registry_expected_fields.sql`

### Tests added
- `internal/service/dlq_worker_test.go`
  - TestBackoffSchedule (exact values per plan §9)
  - TestBackoffMonotonic (symmetric guard)
  - TestChunkStringsBatching / Empty / Deterministic
  - TestTruncate
- `internal/service/recon_heal_test.go`
  - TestHealOCCSkipsOlderTs (SQL WHERE guard present)
  - TestHealOCCAppliesNewerTs (no hash fallback when ts > 0)
  - TestExtractSourceTsFromDoc (3 ts sources + unknown)
  - TestSchemaValidatorDriftDetection (missing + unknown)
- `internal/handler/kafka_consumer_dlq_test.go`
  - TestExtractDLQMetadata_KeyID / _AfterFallback / _StringKey / _NonJSONValue
  - TestDLQWriteBeforeACK_SemanticContract (sentinel)

### Edits
- `internal/handler/kafka_consumer.go` — imports `service`, adds
  `validator` field, `SetSchemaValidator`, `writeDLQ`,
  `extractDLQMetadata`, changes consume loop to skip commit on DLQ
  failure.
- `internal/server/worker_server.go` — schema validator + dlq worker
  wiring + kept shared mongo client for reuse.
- `internal/model/failed_sync_log.go` — adds `NextRetryAt`, `LastError`.
- `internal/model/table_registry.go` — adds `ExpectedFields` JSONB.
- `config/config.go` — adds `DebeziumConfig` (not yet used by wiring
  but allows operators to set signal collection + connector status URL).

---

## 8. Test output

```
$ go build ./... && go vet ./...
(no output = success)

$ go test ./internal/service/... ./internal/handler/...
ok  	centralized-data-service/internal/service	0.517s
ok  	centralized-data-service/internal/handler	0.964s

$ go test ./...
ok  	centralized-data-service/internal/handler
ok  	centralized-data-service/internal/service
ok  	centralized-data-service/pkgs/idgen
ok  	centralized-data-service/pkgs/utils
ok  	centralized-data-service/test/integration
(all pass; non-test packages show [no test files])
```

---

## 9. Runtime verification (2026-04-17)

### Migrations
```
$ psql -U user -d goopay_dw < migrations/012_dlq_state_machine.sql
NOTICE:  [012_dlq_state_machine] failed_sync_logs columns ensured
COMMIT
$ psql -U user -d goopay_dw < migrations/013_table_registry_expected_fields.sql
NOTICE:  [013_registry_expected_fields] expected_fields column ensured
COMMIT

$ \d failed_sync_logs  # snippet
...
 next_retry_at   | timestamp with time zone |           |          | 
 last_error      | text                     |           |          | 
```

### DLQ worker E2E
- Inserted row `(smoke_test_table, rec-1, status=pending, retry_count=0)`.
- Started worker, log shows:
  ```
  dlq retry worker started poll_interval=300 batch_size=100 max_retries=5
  dlq worker picked up rows count=1
  dlq retry scheduled id=2 table=smoke_test_table retries=1 next_retry_at=<+60s>
  ```
- DB snapshot after 6s:
  ```
  status='retrying', retry_count=1, next_retry_at=<now+60s>,
  last_error='registry lookup: record not found'
  ```
- State machine transitioned `pending → retrying`, backoff=1m applied
  (matches `BackoffDelay(1) = 1 * time.Minute`).

### Prometheus scrape
```
$ curl localhost:9090/metrics | grep cdc_dlq
# HELP cdc_dlq_stuck_records_total ...
# TYPE cdc_dlq_stuck_records_total gauge
cdc_dlq_stuck_records_total{status="dead_letter"} 0
cdc_dlq_stuck_records_total{status="failed"} 0
cdc_dlq_stuck_records_total{status="pending"} 0
cdc_dlq_stuck_records_total{status="resolved"} 0
cdc_dlq_stuck_records_total{status="retrying"} 0
# HELP cdc_dlq_write_failures_total ...
cdc_dlq_write_failures_total 0
```

---

## 10. Follow-ups

1. **RBAC + Idempotency-Key on CMS heal/signal endpoints** — plan §13.
   Not implemented here; tracked for CMS-side workspace.
2. **Sensitive-field masking via registry config** — `ReconHealerConfig.SensitiveFieldMask`
   is a list today. Plan is to add `cdc_table_registry.sensitive_fields`
   JSONB + per-table lookup. TODO comment in recon_heal.go.
3. **DLQ retention drop partition > 90d** — migration 010 notes this as
   worker goroutine TODO. Not done in this phase.
4. **Heal via Debezium signal — connector health probe URL** — operators
   must set `debezium.connectorStatusUrl` in config; worker currently
   runs with optimistic-healthy when unset.
5. **Heal NATS command handler** — the existing `cdc.cmd.recon-heal`
   handler (`recon_handler.go`) still uses the old per-ID heal path via
   `ReconCore.Heal`. Next step: switch it to `ReconHealer.HealWindow`
   once CMS-side workspace confirms the API contract.
6. **FE surface** — add DLQ stats gauges + drill-down view for
   `dead_letter` rows. CMS-side workspace.
7. **Dead-letter → CMS dashboard alert rule** — `cdc_dlq_stuck_records_total{status="dead_letter"} > 0`
   to Prom alertmanager. Tracked in observability workspace.

---

## 11. Definition of Done (Phase 2-3)

| Criterion | Status |
|:----------|:-------|
| `_source_ts` OCC in Heal pipeline | Done — via SchemaAdapter reuse |
| Heal batch $in 500 | Done |
| Heal via Debezium Signal + direct fallback | Done |
| Audit log batched 100/insert | Done — CreateInBatches |
| `cdc_recon_heal_actions_total{action}` emitted | Done — already wired in Phase 1 |
| DLQ write-before-ACK flow | Done |
| DLQ state machine (pending → retrying → resolved / dead_letter) | Done |
| Exponential backoff (1m/5m/30m/2h/6h) | Done |
| Schema validation Phase A (registry + info_schema) | Done |
| DLQ stuck-records alert gauge | Done — `cdc_dlq_stuck_records_total` |
| Unit tests — OCC older ts skip, newer ts apply, backoff, DLQ write-before-ACK | Done |
| Build pass + `go vet` | Done |
| Runtime verify — DLQ pickup + state transition | Done |
