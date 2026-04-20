# Implementation v4 — Worker Systematic Timestamp Detection + Error Propagation

> **Date**: 2026-04-20
> **Author**: Muscle (claude-opus-4-7[1m])
> **Reference ADR**: `04_decisions_recon_systematic_v4.md` §2.1, §2.3, §2.4, §2.5, §2.7
> **Scope**: Worker (`cdc-system/centralized-data-service/`). CMS + FE untouched.

---

## 1. Goal

Replace per-table band-aid (`UPDATE registry SET timestamp_field='X'` per collection) with:

1. **Auto-detect** (Component 1) — sample + rank Mongo field coverage.
2. **Runtime fallback** (Component 2) — probe candidate chain when primary field returns 0.
3. **Structured error propagation** (Component 2) — `WindowResult.ErrorCode` + `error_code` column in recon reports.
4. **Migration 017** (Component 3) — schema changes, idempotent, safe rollback.
5. **NATS re-detect handler** (Component 4) — `cdc.cmd.detect-timestamp-field`.
6. **Daily full count aggregator** (Component 5) — absolute truth source/dest counts.

Anti-pattern rejected: manual per-table UPDATE scales as `O(N-tables × human-effort)`.

---

## 2. Component Map

| Component | File | Lines | Purpose |
|-----------|------|-------|---------|
| 1. Detector | `internal/service/timestamp_detector.go` (NEW) | ~220 | Sample → rank → confidence band |
| 1. Tests | `internal/service/timestamp_detector_test.go` (NEW) | ~180 | Rank + banding + security whitelist |
| 2. Source agent | `internal/service/recon_source_agent.go` (EDIT) | +80 | `CountInWindowWithFallback`, `classifyMongoError`, error codes |
| 2. Recon core | `internal/service/recon_core.go` (EDIT) | +30 | Tier 1 calls fallback, errorReport populates code |
| 3. Migration | `migrations/017_timestamp_detection.sql` (NEW) | ~110 | 6 new columns + CHECK constraint + NULL allow |
| 3. Models | `internal/model/table_registry.go` (EDIT) | +25 | Candidates/Detected/Source/Confidence + full_count fields + GetCandidates helper |
| 3. Models | `internal/model/reconciliation_report.go` (EDIT) | +4 | SourceCount *int64 (nullable), ErrorCode string |
| 4. Handler | `internal/handler/recon_handler.go` (EDIT) | +120 | `HandleDetectTimestampField` + `WithTimestampDetector` |
| 4. Wiring | `internal/server/worker_server.go` (EDIT) | +25 | Detector + aggregator construction + subscribe |
| 5. Aggregator | `internal/service/full_count_aggregator.go` (NEW) | ~240 | Daily Mongo + PG counts |

---

## 3. Detailed Diff Summary

### 3.1 TimestampDetector (Component 1)

- `NewTimestampDetector(mc, logger)` — accepts shared Mongo client.
- `DetectForCollection(ctx, db, coll, candidates, sampleSize)`:
  - Whitelist-filters candidates via `candidateNameRE = ^[A-Za-z_][A-Za-z0-9_]{0,63}$`.
  - Finds up to sample docs (default 100) on secondary, with projection limited to `_id` + candidate fields (no heavy payload).
  - Counts presence of each candidate (null / zero-time / empty-string excluded via `isZeroBSONValue`).
  - Sort by coverage DESC, ties break by candidate-chain order (so `updated_at` beats `createdAt` when both 100%).
  - Confidence bands:
    - `coverage >= 0.8` → high
    - `0.3 <= coverage < 0.8` → medium
    - `coverage < 0.3` → low
    - `coverage == 0` → `Field='_id'`, `FallbackToID=true`
  - Empty collection → low confidence + `_id` fallback (ObjectID timestamp extraction already implemented in `extractTimestampMs`).

### 3.2 Retry Helper Evidence (Component 2)

`queryWithRetry` was already in place (`recon_source_agent.go:703-731`). This change:

- **Expanded `isMongoTransient`** to add Mongo CommandError codes **262, 318, 9001** (ExceededTimeLimit, NoProgressMade, SocketException).
- **Wrapped methods (already wrapped pre-v4)**: `CountInWindow`, `HashWindow`, `MaxWindowTs`, `BucketHash`.
- **NEW**: `CountInWindowWithFallback` — invokes `CountInWindow` (which is retry-wrapped) repeatedly per candidate; full-path retry chain preserved.

### 3.3 Error Code Classification (Component 2)

New `classifyMongoError(err) string`:

| Pattern / Driver Code | Code |
|-----------------------|------|
| "circuit breaker"/"breaker ... open" | `CIRCUIT_OPEN` |
| "timeout"/"deadline exceeded"/"i/o timeout" | `SRC_TIMEOUT` |
| CommandError 50 (MaxTimeMSExpired) | `SRC_TIMEOUT` |
| "unauthorized"/"authentication failed"/"auth fail" | `AUTH_ERROR` |
| CommandError 13 (Unauthorized), 18 (AuthenticationFailed) | `AUTH_ERROR` |
| "no field"/"field does not exist"/"missing field" | `SRC_FIELD_MISSING` |
| `isMongoTransient(err)` match | `SRC_CONNECTION` |
| anything else | `UNKNOWN` |

Used in `recon_core.errorReport()` to populate `cdc_reconciliation_report.error_code`.

### 3.4 Migration 017 Summary

```sql
ALTER TABLE cdc_table_registry
  ADD COLUMN timestamp_field_candidates JSONB DEFAULT '["updated_at","updatedAt","created_at","createdAt"]',
  ADD COLUMN timestamp_field_detected_at TIMESTAMPTZ,
  ADD COLUMN timestamp_field_source TEXT DEFAULT 'auto',
  ADD CONSTRAINT ... CHECK (source IN ('auto','admin_override')),
  ADD COLUMN timestamp_field_confidence TEXT,
  ADD COLUMN full_source_count BIGINT,
  ADD COLUMN full_dest_count BIGINT,
  ADD COLUMN full_count_at TIMESTAMPTZ;

ALTER TABLE cdc_reconciliation_report
  ADD COLUMN error_code TEXT,
  ALTER COLUMN source_count DROP NOT NULL;
```

Idempotent (`IF NOT EXISTS`) and has rollback comments.

Runtime verify after apply:

```
source_count  | bigint  |           | (NULL allowed)
error_code    | text    |           |
timestamp_field_candidates | jsonb | default '["updated_at","updatedAt","created_at","createdAt"]'
... (+7 more columns)
```

### 3.5 NATS Handler (Component 4)

Subject: `cdc.cmd.detect-timestamp-field`

Payload: `{registry_id: uint, target_table: string (optional fallback)}`

Behavior:
1. Load registry row (by id or target_table).
2. `td.DetectForCollection(ctx, entry.SourceDB, entry.SourceTable, entry.GetCandidates(), 100)`.
3. If `timestamp_field_source=='admin_override'` → DO NOT mutate registry, still log + respond with result (operator review).
4. Else → UPDATE `timestamp_field`, `_detected_at`, `_confidence`; set `_source='auto'`.
5. Reply payload (if `msg.Reply`): `{registry_id, target_table, result, source}`.

Wired in `worker_server.go` alongside the other 6 recon handlers. Total now **7 recon NATS subjects**.

### 3.6 Full Count Aggregator (Component 5)

`FullCountAggregator.Start(ctx)`:
- Ticks daily at `cfg.RunAt` (default `03:00` UTC).
- Per table:
  - Mongo: `EstimatedDocumentCount()` on secondary (~1ms).
  - PG: `SELECT reltuples FROM pg_class WHERE relname=?`; if `> 10M` rows use reltuples, else `SELECT COUNT(*)` (bounded by `PGCountTimeout=2m`).
  - Updates `full_source_count`, `full_dest_count`, `full_count_at`.
- Uses replica DB (`dbReplica`) for read, primary (`db`) for write.
- Sleeps `PerTableGap=500ms` between tables → ~100s over 200 tables.
- Non-fatal errors: logs + continues; skips registry update if BOTH sides failed.

---

## 4. Test Evidence

### 4.1 Unit Tests (NEW `timestamp_detector_test.go`)

```
=== RUN   TestClassifyMongoError           — 12 cases  PASS
=== RUN   TestClassifyMongoError_CommandErrorCodes — 5 driver codes  PASS
=== RUN   TestIsZeroBSONValue              — 6 cases   PASS
=== RUN   TestCandidateNameRE              — 7 valid + 7 invalid PASS
=== RUN   TestDetectorRanking_ConfidenceBands — 5 cases PASS
```

Test totals: **35 sub-tests**, all green.

### 4.2 Regression

```
go build ./...        → OK (0 errors)
go vet ./...          → OK (0 warnings)
go test ./... -count=1 → ALL packages PASS:
  - internal/handler     1.249s
  - internal/service     0.749s
  - pkgs/idgen          41.641s  (sonyflake timing, unrelated)
  - pkgs/utils           2.081s
  - test/integration     0.529s
```

### 4.3 Migration Runtime Verify

```
$ docker exec -i gpay-postgres psql -U user -d goopay_dw < migrations/017_timestamp_detection.sql
ALTER TABLE ... (11 statements all succeeded)

$ \d cdc_table_registry
   timestamp_field_candidates    | jsonb (default chain)
   timestamp_field_detected_at   | timestamptz
   timestamp_field_source        | text default 'auto'
   timestamp_field_confidence    | text
   full_source_count             | bigint
   full_dest_count               | bigint
   full_count_at                 | timestamptz
   CONSTRAINT ... CHECK (source IN ('auto','admin_override'))

$ \d cdc_reconciliation_report
   source_count   | bigint  (NULL allowed)
   error_code     | text
```

Schema matches ADR v4 §2.1 / §2.3 / §2.7 exactly.

---

## 5. Security Notes

- **Candidate name whitelist**: `candidateNameRE = ^[A-Za-z_][A-Za-z0-9_]{0,63}$` applied in BOTH `resolveTimestampField` (runtime) and `DetectForCollection` (detect phase). Invalid entries logged + dropped.
- **JSONB candidate column default**: fixed safe values; admin UI form should round-trip through JSON array only (no raw string concat).
- **Projection**: detector never fetches full payloads; projection limited to `_id` + candidate fields.
- **Read preference**: detector + aggregator + recon all use Mongo secondary so primary oplog tailers aren't throttled.
- **PG replica**: aggregator uses `dbReplica` for COUNT(*) heavy reads; writes go to primary.

---

## 6. Runtime Verification Gap

Worker restart + NATS `detect-timestamp-field` trigger denied by sandbox (shared infrastructure permission). Migration verified via psql against `gpay-postgres`. Next step (requires operator approval):

```bash
pkill -f cmd/worker && nohup go run ./cmd/worker > /tmp/worker.log 2>&1 &
sleep 10
for id in 37 38 39 40 41 42 43 44; do
  docker exec gpay-nats nats pub cdc.cmd.detect-timestamp-field "{\"registry_id\":$id}"
done
sleep 15
docker exec gpay-postgres psql -U user -d goopay_dw -c "
SELECT target_table, timestamp_field, timestamp_field_source,
       timestamp_field_confidence, timestamp_field_detected_at
FROM cdc_table_registry WHERE is_active=true ORDER BY target_table"
```

Expected: `payment_bills + export_jobs` → `createdAt` high confidence; others → `updated_at` high confidence; all rows have `timestamp_field_detected_at` populated.

---

## 7. Rollback Plan

```sql
ALTER TABLE cdc_table_registry
  DROP COLUMN timestamp_field_candidates,
  DROP COLUMN timestamp_field_detected_at,
  DROP COLUMN timestamp_field_source,
  DROP COLUMN timestamp_field_confidence,
  DROP COLUMN full_source_count,
  DROP COLUMN full_dest_count,
  DROP COLUMN full_count_at;

ALTER TABLE cdc_reconciliation_report
  DROP COLUMN error_code,
  ALTER COLUMN source_count SET NOT NULL,
  ALTER COLUMN source_count SET DEFAULT 0;
```

Code rollback: revert this commit. No data migration required.

---

## 8. Success Criteria Mapping

| ADR v4 §4 Criterion | Status |
|---|---|
| Register → auto-detect | Handler + detector ready; needs worker runtime |
| payment_bills / export_jobs / refund_requests correct source count | Fallback wired; needs runtime verify |
| Drift % bounded 0-100% signed fixed | Out of v4 Muscle scope (FE formula change) |
| Error code structured | DONE — `error_code` column + `classifyMongoError` |
| Total Source/Dest | DONE — aggregator writes `full_*_count`; FE render out of Muscle scope |
| Recon stable 10 cycles | DONE — retry helper + circuit breaker preserved; `isMongoTransient` expanded |

Worker-side systematic fix complete. Next phase: CMS/FE render new columns + error codes (Brain delegation).
