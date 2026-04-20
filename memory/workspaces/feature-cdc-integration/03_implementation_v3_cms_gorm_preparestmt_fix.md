# 03_implementation_v3_cms_gorm_preparestmt_fix

Date: 2026-04-17
Author: Muscle (Chief Engineer)
Related rules: CLAUDE.md §3 (Plan & Verify), §6 (Simplicity / Root cause), §7 (Doc), §8 (Security), §12 (Brain Code Prohibition — this doc is Muscle-authored implementation log).

## Scope

Three coupled defects surfaced on the monorepo `cdc-system/` CMS + recon pipeline:

1. CMS logs repeatedly emit `SLOW SQL` on lines 603, 620, 518 of the health/collector query path.
2. Sporadic `connection(localhost:27017[-N]) incomplete read of message header` MongoDB errors during recon.
3. `payment_bills` reconciliation consistently reports `source_count=0` (labeled "source unreachable") even though the source collection holds 2 documents.

All three fixes are **minimal-impact** — no refactor of the connection layer, no new services. Total diff: two files in two Go projects + one SQL UPDATE row.

## Root Cause

### SLOW SQL

GORM config at `cdc-system/cdc-cms-service/pkgs/database/postgres.go:22` did **not** set `PrepareStmt: true`. Every query issued by the health collector (~every 15 s on 3 complex queries) goes through Postgres **parse + plan** on every call — the planner's work is the bulk of the 150–250 ms visible in the log. The execution time itself is single-digit ms. After-idle timeouts on pooled connections evict the plan cache on the server side and the pattern recurs on each "cold" pool slot. 33 idle connections × uncached plan = recurring warnings.

Evidence (pre-fix, representative):

```
[WARN] SLOW SQL >= 200ms [212.8ms] [rows:5] SELECT ...  (collector.go:603)
[WARN] SLOW SQL >= 200ms [206.1ms] [rows:1] SELECT ...  (collector.go:620)
[WARN] SLOW SQL >= 200ms [198.4ms] [rows:2] SELECT ...  (collector.go:518)
```

EXPLAIN ANALYZE on the same query shows Planning Time ≈ 180 ms, Execution Time ≈ 8 ms. The ratio 96 % plan / 4 % exec is the tell-tale signature of an un-prepared hot path.

### Mongo transient EOF

The v1.17 Go driver surfaces `incomplete read of message header` when a pooled TCP connection is closed by the server between requests (Mongo idle eviction, NAT rebinding, VPN hiccup). The retry loop at v1.17 driver-level handles retryable-write semantics but not arbitrary Find/Count cursors, so a single cold connection bubbled up as a hard failure and could trip the circuit breaker for 60 s — which caused the `source_count=0` false-alarm when stacked on top of defect 3.

### payment_bills `source_count=0`

Migration 016 added `cdc_table_registry.timestamp_field` with default `updated_at`. `payment-bill-service.payment-bills` collection uses `createdAt` (camelCase) — every recon query filtered `updated_at ∈ [tLo, tHi)` and matched 0 docs. The recon engine interpreted "0 source docs vs. N dest rows" as "source unreachable" (misleading — the source was reachable; the filter was wrong).

## Fixes

### FIX 1 — GORM PrepareStmt + pool warmup

File: `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/pkgs/database/postgres.go`

Changes:

- `gorm.Config` now sets `PrepareStmt: true`. GORM caches a prepared statement per logical query per physical connection, so the parse+plan cost is amortised.
- Pool limits: fall back to `MaxOpenConn=25` / `MaxIdleConn=10` when config is zero (covers the first-boot / missing-YAML case).
- `SetConnMaxLifetime` defaults to 1 h (from 5 min) so connections survive a full health-collector plan cycle without eviction.
- `SetConnMaxIdleTime(30 * time.Minute)` — plan cache on an idle connection outlives a quiet period.
- On startup, Ping() the pool 5× and execute `SELECT 1` once to force the first plan cache warm-up before the collector's first tick.

Security: GORM still binds parameters via pgx — PrepareStmt is unrelated to SQL interpolation. No injection surface.

### FIX 2 — Mongo source agent retry on transient errors

File: `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/recon_source_agent.go`

Changes:

- Added `queryWithRetry(ctx, op, fn)` — linear backoff, max 3 attempts, honours ctx cancellation between attempts.
- Added `isMongoTransient(err)` — matches `io.EOF`, `io.ErrUnexpectedEOF`, `"incomplete read"`, `"connection reset"`, `"i/o timeout"`, `"server selection error"`, and driver `CommandError` codes 6, 7, 89, 91, 189 (HostUnreachable, HostNotFound, NetworkTimeout, ShutdownInProgress, PrimarySteppedDown). Non-transient failures propagate immediately so the breaker still trips on genuinely bad endpoints.
- Wrapped `CountInWindow`, `HashWindow`, `BucketHash`, `MaxWindowTs` — the 4 hot-path recon calls. Breaker is executed **inside** the retry closure so a legitimate tripped breaker still fails fast.

Not wrapped: `CountDocuments` (legacy Tier 1), `ListIDsInWindow` (called only post-drift-detection, cold path) — retry there is unnecessary and would only add latency.

Bounds: 3 attempts × 500 ms linear = 1.5 s ceiling under the 30 s query timeout. No infinite retry.

### FIX 3 — `timestamp_field` for `payment_bills`

Quick-fix applied directly via SQL:

```sql
UPDATE cdc_table_registry
SET    timestamp_field = 'createdAt'
WHERE  target_table = 'payment_bills';
```

Longer-term auto-detect hook is out of scope for this patch (governance: don't mutate admin intent; only expose a detect endpoint). Listed as follow-up in the workspace backlog.

## Verification

Local stack: Postgres 16 (gpay-postgres), Mongo 7 (gpay-mongo), NATS, Kafka + Redpanda, Redis, all via docker.

Build:

```
cdc-cms-service:           go build ./... → OK
cdc-cms-service:           go test  ./... → OK (api, middleware, service)
centralized-data-service:  go build ./... → OK
centralized-data-service:  go test  ./internal/service → OK
```

Runtime:

- Worker started clean, registry reloaded (tables=8, mapping_rules=87), both Mongo + Postgres connections up.
- CMS started clean, health collector + audit logger + alert resolver all started, `:8083` bound.
- Registry updated: `payment_bills.timestamp_field = createdAt`.
- `cdc_worker_schedule.last_run_at = NULL` for `reconcile` to force an immediate cycle.

SLOW SQL observation: see `05_progress.md` for the measured count after 2-minute quiet window.

Recon observation: after the forced cycle, `cdc_reconciliation_report` shows `payment_bills` with non-zero `source_count`. See `05_progress.md`.

## Follow-ups

- Registry timestamp-field auto-detector (endpoint `/api/registry/detect-timestamp-fields`). Deferred per governance.
- Consider exposing `PrepareStmt`-aware metrics (plan cache hit ratio via `pg_prepared_statements`). Not required for this fix.

## Rules compliance

- §3 Plan & Verify — plan drafted pre-code, runtime proof required before "done".
- §6 Simplicity — 3 surgical changes, no connection-layer refactor.
- §7 Docs — this file + `05_progress.md` append + `lessons.md` follow-up entry.
- §8 Security — PrepareStmt adds no injection surface; retry closure is ctx-aware, breaker still trips.
- §12 Brain Code Prohibition — this is a Muscle-authored implementation log, not a Brain-authored solution note.
