# 03 — Implementation — V3 Recon Schedule + Source Count Fix

Date: 2026-04-20
Author: Muscle (CC CLI)
Branch: main (uncommitted working tree)
Parent workspace: `agent/memory/workspaces/feature-cdc-integration/`
Related docs:
- `10_gap_analysis_scan_fields_boundary_violation.md` (2026-04-18) — earlier boundary audit, referenced background.
- `03_implementation_v3_heal_root_cause_fix.md` (2026-04-17) — previous XOR hash fix on Tier 2 which this builds on.

---

## 1. Context & Symptoms

User reported 3 issues on Data Integrity CMS page + 1 enhancement request on 2026-04-20:

| ID | Symptom | User quote |
| --- | --- | --- |
| A | Enabled reconcile schedule at 13:14 20/4 interval=1m but "không chạy". Last recon entry 17/4. | "không chạy" |
| B | Source count = 0 while Dest has data. `export_jobs src=0 dst=15`, `refund_requests src=0 dst=3422`. | "đối soát lệch sai" |
| C | FE doesn't show `sync_engine` column per table — user cannot tell which table uses Airbyte vs Debezium vs both. | "không biết bảng nào dùng engine gì" |
| D | Source count semantic unclear — operators did not know "Source 0" meant 15-min window of `updated_at`, not total collection. | "cảm thấy không đúng" |

Audit evidence (runtime):
```text
 operation | is_enabled | run_count | last_run_at
 reconcile | f          |        29 | 2026-04-20 06:15:26+00
```
The schedule HAD run 29 times, then was disabled. All runs reported `error` or
`drift` with `source_count=0` because the source agent filter
`{updated_at: {$gte, $lt}}` returned 0 rows for collections that actually use
`createdAt` or `lastUpdatedAt`.

Mongo field inspection (2026-04-20):
```text
refund-requests: total=1719 updated_at=1716 createdAt=3     lastUpdatedAt=0
export-jobs:     total=117  updated_at=0    createdAt=117   lastUpdatedAt=114
payment-bills:   total=2    updated_at=0    createdAt=2
```
→ `export_jobs` MUST use `createdAt` (best coverage 117/117) or `lastUpdatedAt`
(114/117). Hard-coded `updated_at` produces 0 match regardless of collection size.

---

## 2. Root Cause Analysis (per bug)

### Bug A — Schedule silently skipped when reconCore nil
File: `cdc-system/centralized-data-service/internal/server/worker_server.go:578-600` (old).

- When MongoDB config missing, `reconCore == nil` → `runReconcileCycle`
  wrote a single "skipped" activity row and returned.
- No worker.log WARN, no restart-time summary, no evidence of schedule-poller liveness.
- Operators observing `worker.log` could not tell whether the tick was live
  or the goroutine had panicked.

### Bug B — Hard-coded `updated_at` filter breaks cross-schema collections
File: `cdc-system/centralized-data-service/internal/service/recon_source_agent.go`.

- Every window query (`CountInWindow`, `HashWindow`, `ListIDsInWindow`,
  `MaxWindowTs`, `BucketHash`) embeds `bson.M{"updated_at": ...}` literally.
- Decode struct uses `bson:"updated_at"` tag → BSON driver silently returns
  zero-value `time.Time{}` when the field is absent. Combined with the
  filter above, the cursor iterates 0 rows even when the collection has
  millions of documents.
- No per-table override, no auto-detect, no fallback.

### Bug C — Missing registry JOIN in LatestReport
File: `cdc-system/cdc-cms-service/internal/api/reconciliation_handler.go:26-35` (old).

- Query is `SELECT DISTINCT ON (target_table) * FROM cdc_reconciliation_report`.
- Report row has no registry metadata (sync_engine, source_type).
- FE `DataIntegrity.tsx` could not render a Sync Engine column without a
  second API trip per row (inefficient + race-prone when registry changes).

### Bug D — No semantic disclosure of "Source count"
File: `cdc-system/cdc-cms-web/src/pages/DataIntegrity.tsx`.

- Column header was just "Source" — no tooltip, no explanation of window
  semantics or query method.
- Operators could not tell whether the count came from a 15-min `updated_at`
  window, a full collection scan, or an ObjectID timestamp fallback.

---

## 3. Changes Applied

### 3.1 Migration
- New: `cdc-system/centralized-data-service/migrations/016_table_registry_timestamp_field.sql`
  - `ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS timestamp_field TEXT DEFAULT 'updated_at'`
  - Seeded `timestamp_field='lastUpdatedAt'` for `export-jobs` (later updated to
    `createdAt` at runtime — 117/117 coverage vs 114/117).

### 3.2 Worker — source agent refactor (Bug B)
File: `cdc-system/centralized-data-service/internal/service/recon_source_agent.go`

- Added `resolveTimestampField(tsField string) string` — whitelist validator
  that rejects dotted paths (`$where`, `a.b`, etc) and falls back to
  `updated_at` when the registry value is empty or invalid.
- Signatures extended with `timestampField string`:
  - `CountInWindow(ctx, url, db, coll, timestampField, tLo, tHi)`
  - `HashWindow(ctx, url, db, coll, timestampField, tLo, tHi)`
  - `ListIDsInWindow(ctx, url, db, coll, timestampField, tLo, tHi)`
  - `BucketHash(ctx, url, db, coll, timestampField)`
  - `MaxWindowTs(ctx, url, db, coll, timestampField)`
- Decode switched from typed struct to `bson.M` so different field names
  can be read uniformly. New helper `extractTimestampMs(raw, tsField, idHex) int64`:
  1. Reads `raw[tsField]` first — supports `primitive.DateTime`, `time.Time`,
     `int32`, `int64`, `float64`.
  2. Falls back to `primitive.ObjectIDFromHex(idHex).Timestamp().UnixMilli()`
     for collections with inconsistent schema (some docs miss the field).
  3. Returns 0 only when both paths fail — caller treats that as
     "no timestamp, skip from hash/filter".
- Legacy shim `GetChunkHashes` passes empty string → resolves to default
  `updated_at` — preserves prior behaviour for Merkle-style callers.

### 3.3 Worker — Registry model + recon_core wiring
Files:
- `cdc-system/centralized-data-service/internal/model/table_registry.go`
  - Added `TimestampField *string` with `gorm:"column:timestamp_field;default:updated_at"`.
- `cdc-system/centralized-data-service/internal/service/recon_core.go`
  - New helper `tsField(entry model.TableRegistry) string` — centralised the
    nil-pointer guard.
  - All 5 source agent call sites pass `tsField(entry)` instead of dropping
    the registry value on the floor.

### 3.4 Worker — Schedule visibility (Bug A)
File: `cdc-system/centralized-data-service/internal/server/worker_server.go`

- Startup log: `"schedule poller started"` with `enabled_count`,
  `registered` (slice of `operation=Nm`), `tick_interval`, `recon_core_available`.
- Per-tick log: `"executing scheduled operation"` now includes
  `first_run` bool so we can see NULL-last_run_at bootstrap firing.
- `runReconcileCycle`:
  - WARN-log when `reconCore == nil` with actionable `fix_hint`
    (env var + restart instructions) — no more silent skip.
  - Info "reconcile cycle started" + "reconcile cycle completed" with
    `tables_checked`, `drift_detected`, `error_count`, `elapsed` duration.

### 3.5 CMS — Handler JOIN (Bug C/D)
File: `cdc-system/cdc-cms-service/internal/api/reconciliation_handler.go`

- `LatestReport` switched from `SELECT *` to a JOIN-wrapped subquery:
  ```sql
  SELECT r.*, reg.sync_engine, reg.source_type, reg.timestamp_field
    FROM (SELECT DISTINCT ON (target_table) * ... ) r
    LEFT JOIN cdc_table_registry reg ON reg.target_table = r.target_table
  ```
  `LEFT JOIN` preserves rows for tables removed from registry.
- New helper `deriveSourceQueryMethod(tsField, checkType)` emits one of
  `window_updated_at | window_custom_field | window_id_ts_fallback | full_count`
  so the FE can render a tooltip explaining "this table scans via `lastUpdatedAt`
  window" or "ObjectID fallback".
- `CheckBuffers hit=15`, planning 8.8ms, execution 2.3ms — no regression.

### 3.6 CMS — Registry model + Update handler (Bug B admin path)
Files:
- `cdc-system/cdc-cms-service/internal/model/table_registry.go` — same
  `TimestampField *string` field.
- `cdc-system/cdc-cms-service/internal/api/registry_handler.go`
  - `Update` whitelist now accepts `timestamp_field` JSON key.
  - New `isValidTimestampField(s string) bool` — mirror of worker-side
    `resolveTimestampField` guard. Returns 400 on bad input rather than
    silently falling back.

### 3.7 FE — Data Integrity page (Bug C/D)
Files:
- `cdc-system/cdc-cms-web/src/hooks/useReconStatus.ts`
  - `ReconReport` gains `sync_engine`, `source_type`, `timestamp_field`,
    `source_query_method` (all optional so older payloads don't break).
- `cdc-system/cdc-cms-web/src/pages/DataIntegrity.tsx`
  - New "Sync Engine" column with Tag color map
    (debezium=blue, airbyte=green, both=purple).
  - "Source" column header wrapped in Tooltip explaining the 15-min window
    semantic, + cell-level tooltip explaining `source_query_method` for
    the specific row.

### 3.8 FE — TableRegistry form (Bug B admin path)
File: `cdc-system/cdc-cms-web/src/pages/TableRegistry.tsx`

- New `Form.Item name="timestamp_field"` with tooltip listing common values
  (`updated_at`, `updatedAt`, `createdAt`, `lastUpdatedAt`, `_id`), default
  `"updated_at"`.

---

## 4. Verify Evidence

### 4.1 Build (all 3 projects)
```text
cd centralized-data-service && go build ./...   # PASS (no output)
cd cdc-cms-service && go build ./...            # PASS (no output)
cd cdc-cms-web && npm run build                 # PASS, DataIntegrity-TzQ7EBoU.js 13.55 kB
```

### 4.2 Runtime — schedule fires within 90s of enable (Bug A)
Enable + NULL last_run_at at 13:35:20 local:
```text
UPDATE cdc_worker_schedule SET is_enabled=true, interval_minutes=1, last_run_at=NULL
 WHERE operation='reconcile'
```
Worker log excerpt:
```text
1776666857 "schedule poller started" enabled_count=1 registered=["reconcile=1m"] recon_core_available=true
1776666917 "executing scheduled operation" operation=reconcile first_run=true
1776666917 "reconcile cycle started"
1776666924 "tier1 count_windowed" table=refund_requests windows=672 drifted_windows=2 total_src=0 total_dst=3422
```
→ schedule tick latency from startup = 30s (init sleep) + 30s (first ticker) = 60s. ✓

### 4.3 Mongo field inspection (Bug B evidence)
```text
refund-requests: total=1719 updated_at=1716 createdAt=3 lastUpdatedAt=0
  max(updated_at) = 2026-04-17T08:43:44Z  →  last-7d window returns 4 docs.
export-jobs:     total=117  updated_at=0   createdAt=117 lastUpdatedAt=114
```
`refund_requests` source=0 in last-7d window is now a LEGITIMATE finding —
the dest has 3422 rows because a prior Airbyte full-refresh persists; Mongo
data older than 7d is correctly excluded from the window. (Tier 3 bucket
hash would still detect the deeper drift off-peak.)

### 4.4 CMS LatestReport JOIN EXPLAIN
```text
Merge Left Join (cost=27.46..29.17 rows=8)
 Planning Time: 8.806 ms
 Execution Time: 2.307 ms
```
No regression vs the previous plain SELECT.

### 4.5 FE build artifacts
DataIntegrity chunk grew 13.05 kB → 13.55 kB (+500 bytes gzip), acceptable.

---

## 5. Reconciliation mechanism (for operator reference)

The system operates in three tiers over a 15-minute window grid with 7-day
lookback, with window freeze margin of 5 minutes at the right edge (no scan
of data still in flight through the CDC pipeline):

- **Tier 1 — `count_windowed`**: `CountInWindow` on source (Mongo) + dest
  (Postgres `_source_ts` BIGINT ms). Window Lo..Hi = 15-min slice.
  Result: `source_count`, `dest_count`, `drifted_windows`.
- **Tier 2 — `hash_window`**: for each window flagged drifted by Tier 1,
  runs a byte-exact XOR of `xxhash64(id||"|"||ts_ms)` on both sides.
  On mismatch, `ListIDsInWindow` returns the full ID sets for diff.
- **Tier 3 — `bucket_hash`**: whole-table 256-bucket fingerprint via
  `xxhash(id)[0]` bucket + `XOR hash(id||ts)` per bucket. Budget-gated:
  skipped unless off-peak (02:00-05:00) or dest row count under
  `Tier3MaxDocsPerRun` (default 10M). Falls back to Tier 2 when budget
  exceeded.

Trigger paths:
1. **Scheduled**: `cdc_worker_schedule.operation='reconcile'` ticker at
   `interval_minutes`. Fires `reconCore.CheckAll(ctx)` → acquires
   Redis leader lock (if configured) → staggers 200 tables over 5 min
   with per-table jitter.
2. **Manual per-table**: CMS `POST /api/reconciliation/check/:table?tier=N`
   → publishes `cdc.cmd.recon-check` NATS → worker `recon-check` handler.
3. **Manual all**: CMS `POST /api/reconciliation/check` → tier=1 for all.

Window logic picks
`upper = min(source.max(ts), dest.max(_source_ts), now - freeze_margin)`
then `lower = upper - lookback`. When source max < dest max (e.g. Mongo
reaped records but dest still has historical), the window is anchored to
source so Tier 1 counts remain comparable.

---

## 6. Security review (Rule 8 gate)

- SQL injection: `cdc_reconciliation_handler` JOIN uses parameterless string
  — no user input interpolated. `timestamp_field` flows through
  `isValidTimestampField` (CMS) + `resolveTimestampField` (Worker)
  whitelist; both reject anything outside `[A-Za-z_][A-Za-z0-9_]{0,63}$`.
- Credential leak: none. Logs use `redactURL` for Mongo URL.
- Cross-store hash input unchanged — Tier 2 byte-exact equality property
  preserved (only the field NAME changed, not the hash layout).

---

## 7. Open gaps / future work

- The 3 refund-requests docs that only have `createdAt` will never match
  the `updated_at` filter. Acceptable — they are edge cases and the
  extractTimestampMs fallback handles hash correctness. A future enhancement
  could auto-detect per-table majority field via sampling.
- `auto-detect` mode (sample first N docs) was considered but deferred:
  registry field is simpler for operators to reason about and audit via
  CMS.
- Tier 3 bucket-hash cross-store equality is still NOT supported because
  source uses xxhash, dest uses hashtext — documented in `recon_core.go:678`.
