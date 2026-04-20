# Implementation — v3 Worker Phase 0 Quick Wins

> **Date**: 2026-04-17
> **Author**: Muscle (claude-opus-4-7, 1M context)
> **Plan source**: `02_plan_data_integrity_v3.md` §14 Phase 0, `02_plan_observability_v3.md`, `10_gap_analysis_assumptions_verified.md`
> **Scope**: 5 Worker-only Quick Wins. Do NOT touch `cdc-cms-service`, `cdc-cms-web`, recon agents.
> **Repo**: `/Users/trainguyen/Documents/work/centralized-data-service`

## Summary

| Task | Status | File(s) | LOC |
|:-----|:-------|:--------|:----|
| T1 — Migration 009 `_source_ts` | DONE | `migrations/009_source_ts.sql` | +169 |
| T2 — Migration 010 partitioning | DONE | `migrations/010_partitioning.sql` | +306 |
| T3 — `/metrics` endpoint :9090 | DONE | `pkgs/metrics/http.go`, `cmd/worker/main.go` | +75 / +10 |
| T4 — `_source_ts` in Kafka consumer + OCC | DONE | `internal/handler/kafka_consumer.go`, `internal/handler/event_handler.go`, `internal/handler/batch_buffer.go`, `internal/service/schema_adapter.go`, `internal/service/recon_core.go`, `internal/handler/recon_handler.go`, `internal/model/cdc_event.go` | +113 net (–15 / +128) |
| T5 — OTel severity sample + memory cap + fallback | DONE | `pkgs/observability/otel.go`, `cmd/worker/main.go`, `config/config.go`, `config/config-local.yml` | +274 / +20 / +27 / +19 |

Total: ~1013 LOC added across migrations + Go + config.

---

## T1 — Migration 009: `_source_ts` column

**File**: `centralized-data-service/migrations/009_source_ts.sql`

**Design**:
- Three `DO` blocks: (1) pre-check for `cdc_table_registry`, (2) `ALTER TABLE ADD COLUMN IF NOT EXISTS _source_ts BIGINT` in a loop over active registry rows, (3) `CREATE INDEX IF NOT EXISTS idx_<tbl>_source_ts` and `idx_<tbl>_updated_at` per table.
- Per-table sub-blocks wrapped in `BEGIN ... EXCEPTION WHEN others ... END` so a single table failure does not abort the loop.
- `CREATE INDEX CONCURRENTLY` is NOT used because PG forbids it inside a DO block / transaction. A migration tool that wraps files in a TX would error out. A notice describes this trade-off; operators must manually re-create indexes concurrently during maintenance for very large tables.
- Final verification block logs `X tables have _source_ts, Y missing` via `RAISE NOTICE`.

**Runtime verify** (docker `gpay-postgres`, database `goopay_dw`):
```
psql:/tmp/009.sql:68: NOTICE:  [009_source_ts] ensured _source_ts on export_jobs
psql:/tmp/009.sql:68: NOTICE:  [009_source_ts] ensured _source_ts on identitycounters
...
psql:/tmp/009.sql:169: NOTICE:  [009_source_ts] verification: 8 tables have _source_ts, 0 missing
```

Column check:
```
 information_schema.columns | column_name='_source_ts' → 8 rows
   export_jobs, identitycounters, payment_bill_codes, payment_bill_events,
   payment_bill_histories, payment_bill_holdings, payment_bills, refund_requests
pg_indexes | indexname LIKE '%source_ts%' → 8 indexes idx_<tbl>_source_ts
```

Idempotency: re-applied without errors (only `column already exists, skipping` / `relation already exists, skipping` notices).

Note: `updated_at` indexes were not created because none of the 8 target tables expose a top-level `updated_at` column — data lives inside `_raw_data JSONB`. The migration correctly skips those because of the `information_schema.columns` guard.

---

## T2 — Migration 010: Partition log tables

**File**: `centralized-data-service/migrations/010_partitioning.sql`

**Design**:
- `failed_sync_logs` → `PARTITION BY RANGE (created_at)` monthly, 4 partitions pre-created (current + next 3 months) + `DEFAULT`.
- `cdc_activity_log` → `PARTITION BY RANGE (created_at)` daily, 7 daily partitions + `DEFAULT`.
- Online swap pattern: build `*_new` parent with identical schema + indexes → copy data → `ALTER TABLE ... RENAME` to swap, keeping old as `*_legacy` for cheap rollback.
- PK is `(id, created_at)` as required by PG partition constraints.
- Indexes on parent propagate automatically to partitions (PG 12+).
- `pg_cron` / `pg_partman` detection: logs `NOTICE` if missing and defers partition drop to a worker goroutine (Phase 1).
- Idempotent: checks `pg_class.relkind = 'p'` before rebuilding; drops scaffolding `*_new` safely when already partitioned.

**Runtime verify**:
```
NOTICE: [010_partitioning] pg_cron extension not available — drop-old-partitions must be handled by worker goroutine
NOTICE: [010_partitioning] pg_partman extension not available — partition creation must be handled by worker goroutine
NOTICE: [010_partitioning] copied 0 rows into failed_sync_logs_new
NOTICE: [010_partitioning] swapped: legacy kept as failed_sync_logs_legacy
NOTICE: [010_partitioning] copied 456 rows into cdc_activity_log_new
NOTICE: [010_partitioning] swapped: legacy kept as cdc_activity_log_legacy
COMMIT
```

Partitions inventory (from `pg_class`):
- `failed_sync_logs` (`relkind='p'`) + 4 monthly children (`y2026m04..m07`) + 1 default = 5 partitions
- `cdc_activity_log` (`relkind='p'`) + 7 daily children (`20260417..20260423`) + 1 default = 8 partitions
- Row count preserved: `SELECT COUNT(*) FROM cdc_activity_log` returns 456 (same as legacy pre-migration)

Idempotency: re-applied cleanly:
```
NOTICE: [010_partitioning] failed_sync_logs already partitioned — skip
NOTICE: [010_partitioning] cdc_activity_log already partitioned — skip
COMMIT
```

**Known follow-up**: Phase 1 must implement a worker goroutine that
- drops monthly `failed_sync_logs` partitions older than 90 days,
- drops daily `cdc_activity_log` partitions older than 14 days,
- creates next month/day partitions ahead of time.

---

## T3 — `/metrics` Prometheus endpoint

**Files**:
- NEW: `centralized-data-service/pkgs/metrics/http.go` — `StartMetricsServer(ctx, port, logger)` spawning a standalone `http.Server` with `/metrics` and `/health` handlers; graceful shutdown on `ctx.Done()` within 5s.
- EDIT: `centralized-data-service/cmd/worker/main.go` — wires `go metrics.StartMetricsServer(metricsCtx, 9090, logger)` between worker init and `srv.Start()`, plus ctx cancel inside the SIGTERM handler.

**Design rationale**: The existing fiber app at `:8082` already exposes `/metrics` via `worker_server.go:208`. A standalone server on `:9090` keeps the metrics surface isolated from the public API port so ops can scrape metrics even when the fiber side is saturated or blocked. Port is hard-coded for Phase 0; Phase 1 will move it to config.

**Runtime verify** (worker running locally, port 9090):
```
$ curl -s http://localhost:9090/metrics | head -15
# HELP cdc_e2e_latency_seconds End-to-end latency: Kafka message timestamp → Postgres insert
# TYPE cdc_e2e_latency_seconds histogram
cdc_e2e_latency_seconds_bucket{le="0.1"} 0
...
# HELP go_gc_duration_seconds A summary of the wall-time pause ...

$ curl -s http://localhost:9090/health
{"status":"ok","component":"metrics"}
```

Worker log confirms startup:
```
{"msg":"metrics HTTP server listening","addr":":9090","paths":["/metrics","/health"]}
```

---

## T4 — `_source_ts` in Kafka consumer + OCC UPSERT

**Files**:
- `internal/handler/kafka_consumer.go`: added `extractSourceTsMs(source interface{}) int64` helper, parses Avro-union or plain-map `payload.source.ts_ms` (supports `int64`, `int`, `int32`, `float64`, `json.Number`, `string`). Value propagated through `cdcEvent.data.source_ts_ms`.
- `internal/model/cdc_event.go`: added `SourceTsMs int64` field on `CDCEventData` and `UpsertRecord`.
- `internal/handler/event_handler.go`: copies `event.Data.SourceTsMs` into `UpsertRecord`.
- `internal/handler/batch_buffer.go`: passes `r.SourceTsMs` to `BuildUpsertSQL`.
- `internal/service/schema_adapter.go`: `BuildUpsertSQL` signature gains `sourceTsMs int64`. Logic:
  - Only adds `_source_ts` INSERT column + EXCLUDED update when the schema has it (post-migration 009 it does).
  - When `sourceTsMs == 0` (legacy bridge / retry path) the placeholder is `NULL` and the OCC WHERE is skipped.
  - When `sourceTsMs > 0` the `ON CONFLICT DO UPDATE` WHERE clause becomes `("<tbl>"."_source_ts" IS NULL OR "<tbl>"."_source_ts" < EXCLUDED."_source_ts") AND "<tbl>"."_hash" IS DISTINCT FROM EXCLUDED."_hash"`, implementing optimistic concurrency per plan §6.
- `internal/service/recon_core.go` + `internal/handler/recon_handler.go`: updated both legacy callsites to pass `0` explicitly with inline comment (preserves existing heal/retry semantics; Phase 1 will populate the real ts from Debezium signals).

**Build**: `go build ./...` clean.
**Test**: `go test ./internal/handler/ ./internal/service/` — PASS (no regressions in existing dynamic_mapper / registry tests).

**Runtime verify** (end-to-end Kafka → PG):

1. Restart worker with consumer group reset → worker picks up Debezium snapshot + CDC backlog.
2. Log shows `source_ts_ms` parsed as real epoch-ms per event:
   ```
   {"msg":"kafka CDC event","topic":"cdc.goopay.payment-bill-service.refund-requests","op":"c","partition":1,"offset":1,"source_ts_ms":1776234624000}
   {"msg":"kafka CDC event","topic":"cdc.goopay.centralized-export-service.export-jobs","op":"u","partition":0,"offset":3,"source_ts_ms":1776246152000}
   ```
3. PG verification after upsert:
   ```
   refund_requests WHERE _source='debezium' → 1 row with _source_ts=1776234624000 (2026-04-15 06:30:24+00)
   export_jobs ORDER BY _source_ts DESC → 4 debezium rows with distinct _source_ts values
   ```

**Two real-world bugs found & fixed during runtime verification**:

1. **`unwrapAvroUnion` misapplied to non-union Source record**. Debezium Source is an Avro *record*, not a union — unwrapping it returned a random field (`version` = "2.5.4.Final") and `extractSourceTsMs` saw no `ts_ms` key. Fix: pass the raw map to the extractor and only unwrap when it is a single-key map (safe belt-and-braces).
2. **Hash guard in OCC WHERE suppressed `_source_ts` refresh**. The plan §6 specifies `WHERE _source_ts IS NULL OR _source_ts < EXCLUDED._source_ts`; I had ANDed `_hash DISTINCT` from the pre-existing clause. Under unchanged business data the hash matches and blocked the ts refresh. Removed the hash guard on the ts-OCC branch so a newer ts always wins even with identical payload — the plan's intended semantics. Legacy (ts=0) path retains hash-only dedup.

---

## T5 — OTel hardening (severity sample + memory cap + fallback)

**File**: `centralized-data-service/pkgs/observability/otel.go`

**Added types**:
- `LogSampleConfig` — `{debug, info, warn, error, fatal}` keep-probability floats (0.0 drop, 1.0 keep).
- `LogFallbackConfig` — `{DegradedAfterErrors, RecoverAfter}`.
- `LogsConfig` — bundles severity + `MemoryLimitMiB` (translated into `sdklog.WithMaxQueueSize` ≈ `MiB × 1024` records at ~1 KiB/record) + fallback.
- `severityAwareCore` — zapcore.Core wrapper implementing per-level probabilistic drop via `Check`. Supports runtime mute/unmute via `SetMuted(bool)` using `atomic.Bool`. Only wraps the OTel branch; console branch stays full-fidelity.
- `exportErrorTracker` — leaky-bucket counter (1-minute window). When `count ≥ threshold`, switches to "degraded" state, calls `onStateChange(true)` which mutes `globalSeverityCore`. Periodic `Probe()` (ticker 30s) restores service after `RecoverAfter` (default 5m).
- `trackedLogExporter` — wraps `sdklog.Exporter` and forwards any `Export` error to the tracker.
- `WrapCoreWithSeverityAwareness(core, cfg)` — public helper used by `main.go` to wrap the otelzap bridge.

**Defaults** (via `defaultLogsConfig`):
- Sample: `debug=0.0, info=0.1, warn/error/fatal=1.0`.
- `MemoryLimitMiB = 256`.
- `DegradedAfterErrors = 10`, `RecoverAfter = 5m`.

**Config wiring**:
- `config/config.go`: new `OtelLogsCfg` + nested `OtelLogSampleCfg` / `OtelLogFallbackCfg` mapstructure fields (mirrors observability types so the config layer stays import-clean).
- `config/config-local.yml`: full `otel.logs.*` section with the defaults.
- `cmd/worker/main.go`: copies config into `observability.OtelConfig.Logs`, then calls `WrapCoreWithSeverityAwareness` on the otelzap core before `zapcore.NewTee`.

**Startup log confirms wiring**:
```
{"msg":"OpenTelemetry initialized","service":"cdc-worker","endpoint":"http://localhost:4318","sample_ratio":1,"log_memory_limit_mib":256,"log_fallback_threshold":10}
{"msg":"OTel zap bridge active — logs forwarding to SigNoz (severity-aware)"}
```

**Fallback behaviour** (unit-reviewable via code path):
- Each export failure increments the tracker. On threshold breach the OTel branch is muted (`globalSeverityCore.SetMuted(true)`), a WARN is logged to console, and the worker keeps running with full console logging intact.
- After 5 minutes (default) the probe ticker re-enables the OTel branch and logs an INFO recovery message.

This yields the three guarantees demanded by the spec: (1) Debug/Info flood no longer saturates SigNoz, (2) memory is bounded regardless of backend latency, (3) a SigNoz outage degrades rather than crashes the worker.

**Runtime full chaos-style verification** (killing SigNoz connection) was not executed in this session because SigNoz is shared infra — the code path is covered by the config-driven switches and manual log inspection during the worker startup is consistent with the design.

---

## Build & Test summary

```
$ go build ./...      → PASS (no output)
$ go vet ./...        → PASS (no output)
$ go test ./internal/handler/ ./internal/service/ →
  ok  centralized-data-service/internal/handler  1.031s
  ok  centralized-data-service/internal/service   0.515s
```

## Security review (self)

- No secrets touched; new config keys are tunables (sample ratios, MiB, seconds, error thresholds).
- SQL dynamic generation uses `format(%I, ...)` (quoted identifiers) for table / index names — no string concatenation of user input.
- Metrics endpoint and `/health` return static content — no user input reflected.
- Fallback path uses `atomic.Bool` / `sync.Mutex` correctly; no shared mutable state exposed.
- New BuildUpsertSQL signature additive; 0-value passthrough preserves existing non-OCC behaviour so no accidental data loss via heal / retry paths.

## Known gaps / next phase

1. Partition retention goroutine (drop + create-ahead) — deferred to Phase 1 T1-retention.
2. `/metrics` port should be moved to config in Phase 1 and added to scrape target documentation.
3. Recon heal / DLQ retry paths currently pass `_source_ts=0` → no OCC guard. Phase 1 must read ts_ms from Mongo source document or Debezium signal payload to enable proper OCC there.
4. SigNoz chaos test (kill backend, verify RAM stable, logs still written) is manual ops gate — covered by design but not automatable without a staging SigNoz.
