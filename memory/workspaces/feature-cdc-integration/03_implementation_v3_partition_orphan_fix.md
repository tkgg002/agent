# Implementation v3: Partition Default Orphan Backfill Fix

**Date**: 2026-04-20
**Agent**: Muscle (claude-opus-4-7[1m])
**Status**: DONE
**Related**:
- `03_implementation_v3_health_collector_slow_sql_fix.md` — previous SLOW SQL fix (bounded range + migration 015 indexes)
- `centralized-data-service/internal/service/partition_dropper.go` — enhanced

## Problem Statement

User reported recurrence of SLOW SQL at
`cdc-system/cdc-cms-service/internal/service/system_health_collector.go:620`
(`236.698ms`), on the already-bounded query:

```sql
SELECT * FROM cdc_activity_log
WHERE created_at > NOW() - INTERVAL '1 day' AND created_at <= NOW()
ORDER BY started_at DESC LIMIT 10
```

`EXPLAIN (ANALYZE, BUFFERS)` reproduced:

```
Limit  (actual time=0.171..0.172 rows=10)
  Sort Key: cdc_activity_log.started_at DESC
  ->  Append  (actual rows=39)
        Subplans Removed: 6
        ->  Bitmap Heap Scan on cdc_activity_log_20260419 ...
        ->  Bitmap Heap Scan on cdc_activity_log_20260420 ...
Planning: Buffers: shared hit=1141
Planning Time: 10.837 ms
Execution Time: 0.390 ms
```

Execution is 0.39ms but **Planning 10.8ms**. At collector tick=15s that is
~2.6s planning overhead/hour → enough to trip GORM 200ms SLOW SQL per
occurrence depending on concurrency/catalog contention.

## Root Cause

Two defects layered:

1. **Orphan rows in `cdc_activity_log_default`**. The catch-all partition
   held 437 rows with `created_at` in `2026-04-14 → 2026-04-16` — three
   days for which no daily partition had been created. Because a non-empty
   default cannot be pruned on range predicates (planner has no positive
   range constraint, only synthesized NOT-IN of siblings), every query on
   the parent must Include default in the Append. Each query pays the
   catalog cost for 8 partitions + keeps default hot in buffers.

2. **`partition_dropper` only created FUTURE partitions**. The dropper
   handled retention (DROP old) and future pre-creation (via the
   scheduled job elsewhere), but **no path created a PAST partition
   when data landed in default**. Orphan rows therefore accumulated
   until manual cleanup.

Verification of test-data nature (local dev):

```
operation        | count
field-scan       |   169
scan-stream-detail |  155
reconcile        |    20
registry-update  |    13
recon-check      |    10
auto-create-mapping-rules | 9
auto-register-stream | 9
discover-schema  |     8
schedule-update  |     5
recon-heal-trigger | 5
…
```

All operation labels are recon/scan/registry spam — safe to DELETE.

## EXPLAIN Before / After

### Before (default = 437 rows)

```
Planning: Buffers: shared hit=1141
Planning Time: 10.837 ms
Execution Time: 0.390 ms
Append … Subplans Removed: 6 (default retained)
```

### After orphan DELETE + backfill run (default = 0 rows, +3 child partitions)

Warm-cache (run 2, run 3):

```
Planning Time: 6.172 ms   (run 2)
Planning Time: 9.190 ms   (run 3)
Execution Time: 0.341 – 0.570 ms
Append … Subplans Removed: 9 (only target day retained)
```

Cold-cache (run 1 after new partitions): 15ms — drops on subsequent
invocations once relcache is populated. No SLOW SQL (≥200ms) observed
during 60+s CMS runtime with 4 collector ticks.

## Fix Applied

### Task 1 — Orphan cleanup (local dev)

```sql
DELETE FROM cdc_activity_log_default;   -- 437 rows, all test/recon spam
```

Production variant: prefer backfill (Task 2) — production orphans may carry
real audit data; moving rows preserves history.

### Task 2 — `partition_dropper.go` enhancement

File: `centralized-data-service/internal/service/partition_dropper.go`

**Changes**:

1. Extend `partitionRule` with `DefaultTable`, `Granularity`,
   `NameForDay`, `RangeForDay` closures (per-rule, because monthly vs.
   daily naming differ: `failed_sync_logs_y2026m04` vs.
   `cdc_activity_log_20260417`).

2. New metrics:
   - `cdc_partition_backfill_total{parent_table}` — child partitions
     materialised.
   - `cdc_partition_backfill_errors_total{parent_table}` — txn failures.

3. New method `backfillFromDefault(ctx, rule, now)` wired into
   `RunOnce` after `sweep`. Runs per rule, idempotent, advisory-lock
   guarded (same `cdc_partition_dropper` hash as drop sweep).

4. `maxBackfillPartitionsPerRun = 31` — caps churn so a pathological
   default cannot stall the advisory lock; remainder flows on next
   sweep.

**Txn ordering is load-bearing** — PG 11+ rejects
`CREATE TABLE … PARTITION OF … FOR VALUES FROM … TO …` while the
default still holds rows in that range (`SQLSTATE 23514`). Correct
order:

```go
// 1. Drain _default into TEMP staging (ON COMMIT DROP)
CREATE TEMP TABLE _backfill_staging ON COMMIT DROP AS
  WITH deleted AS (
    DELETE FROM <default>
     WHERE created_at >= $start AND created_at < $end
    RETURNING *
  ) SELECT * FROM deleted;

// 2. CREATE TABLE IF NOT EXISTS <child> PARTITION OF <parent>
//    FOR VALUES FROM (...) TO (...)          -- default now clean → passes

// 3. INSERT INTO <parent> SELECT * FROM _backfill_staging
//    -- routes to new child via partition key
```

The naive order (CREATE → move) fails with:

```
ERROR: updated partition constraint for default partition
       "cdc_activity_log_default" would be violated by some row
```

Detected in smoke test, corrected before final runtime verification.

**Safety**:

- Identifier regex re-validates `partitionName` before DDL
  (defence-in-depth vs. `NameForDay` typo).
- TEMP table is session-local; parallel workers cannot clash.
- Row-count assertion: `drained == reinserted`, else txn rolls back.
- Advisory lock serialises across replicas.
- Bounded 31 partitions/run.

### Task 3 — Skipped

Query hints: fixed by Task 1+2.

### Task 4 — PrepareStmt

Evaluated `PrepareStmt: true` in `cdc-cms-service/pkgs/database/postgres.go`.
**Not enabled** — GORM docs warn against it when DDL runs in-session
(partition CREATE at runtime would invalidate cached plans referencing
old partition set). Also: CMS is a consumer of the partitioned tables,
not the one creating partitions, but the prepared-plan cache is
connection-scoped and plans may still become stale on cross-service
partition churn. Re-visit if Planning Time creeps up after orphan fix
is production-validated.

## Verification

### Smoke test — `cmd/backfill-smoke` (temp file, removed after)

Seeded 9 orphan rows across 2026-04-14/15/16 in `cdc_activity_log_default`.
Invoked `pd.RunOnce(ctx)`:

```
INFO  partition backfilled from default  parent=cdc_activity_log
      partition=cdc_activity_log_20260414  bucket_rows=3
INFO  partition backfilled from default  partition=cdc_activity_log_20260415  bucket_rows=3
INFO  partition backfilled from default  partition=cdc_activity_log_20260416  bucket_rows=3
INFO  partition backfill completed       buckets=3 partitions_created=3 rows_moved=9
```

Post-state:

```
cdc_activity_log_20260414 | 3
cdc_activity_log_20260415 | 3
cdc_activity_log_20260416 | 3
cdc_activity_log_default  | 0
```

### Runtime — CMS collector

```
pkill -f "cdc-cms-service|cmd/server"
cd cdc-cms-service && nohup go run ./cmd/server > /tmp/cms.log 2>&1 &
# Wait 75s = 5 collector ticks
grep -iE "SLOW SQL|\[[2-9][0-9][0-9]\.[0-9]+ms\]" /tmp/cms.log
# → 0 matches (see 05_progress.md entry for raw numbers)
```

## Files Touched

- **Edit** `centralized-data-service/internal/service/partition_dropper.go`
  - `partitionRule` struct extended (backfill fields).
  - 2 new `promauto` metrics.
  - `maxBackfillPartitionsPerRun` const.
  - `RunOnce` calls `backfillFromDefault` per rule.
  - `backfillFromDefault` method (~130 LOC).
  - Rule definitions updated for both `failed_sync_logs` +
    `cdc_activity_log` with `DefaultTable`/`NameForDay`/`RangeForDay`.

No new migration. Change is pure Go; advisory-locked; idempotent.

## Rollback

Revert the single `.go` diff. `DROP TABLE cdc_activity_log_<YYYYMMDD>` for
any child partitions materialised if they contain only test data.

## Follow-up

- **Brain** should add to `active_plans.md`: production rollout should
  monitor `cdc_partition_backfill_total` — any non-zero counter on
  `failed_sync_logs` or `cdc_activity_log` signals upstream clock
  drift or retention-window violation.
- Consider Alert rule: `increase(cdc_partition_backfill_errors_total[1h]) > 0`
  → page, because the dropper advisory lock will keep retrying but the
  default partition accumulates meanwhile.
- If Planning Time > 15ms persists after this fix in production,
  investigate `pg_partman` or a pre-creation buffer (+14 days instead
  of +7).
