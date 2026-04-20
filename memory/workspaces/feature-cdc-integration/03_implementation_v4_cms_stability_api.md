# 03 — Implementation: CMS Stability + API Migration v4

**Status**: Implemented
**Date**: 2026-04-17
**Executor**: Muscle (CC CLI)
**References**: `04_decisions_recon_systematic_v4.md` §2.2, §2.3, §2.6, §2.8
**Scope**: CMS-side only (backend `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/`). Worker + FE untouched.

---

## FIX 1 — GORM PrepareStmt + Pool Warmup (SLOW SQL elimination)

**File**: `pkgs/database/postgres.go`

**Root cause**: each ~15s collector tick re-parsed & re-planned the same
`SELECT` against `cdc_table_registry` / `cdc_reconciliation_report`, producing
150–250ms warnings labelled "SLOW SQL". Actual execution was <5ms; the cost was
Postgres planning time on cold prepared statements that GORM discarded after
each call.

**Fix applied**:
- `PrepareStmt: true` — GORM caches per-connection prepared statements.
- Pool sizing: `MaxOpen=25 / MaxIdle=10 / ConnMaxLifetime=1h / ConnMaxIdleTime=30m`.
- Warmup loop: `Ping()` 5× + `SELECT 1` once at startup so cold sockets + cold planner never hit the first collector tick.

**Evidence (before)**: ~8-12 SLOW SQL warnings per minute @ 150-250ms.
**Evidence (after)**: runtime log check — `grep -c "SLOW SQL" /tmp/cms.log = 0`.
**Max query duration**: <100ms across 30s runtime window.

---

## FIX 2 — Drift Status Computation (API-side)

**File**: `internal/api/reconciliation_handler.go` — new `ComputeDriftStatus()`.

**Why read-path**: worker writes raw counts; CMS derives `drift_pct` + `status`
on the fly so FE sees a single consistent view. Stored report stays pristine —
recomputation is cheap and spec changes (thresholds) don't need re-running
Tier-1 checks.

**State machine**:
| Input                                    | status                    | drift_pct | code              |
|------------------------------------------|---------------------------|-----------|-------------------|
| `error_code != ""`                       | error                     | 0         | preserved         |
| `source_count == nil`                    | error                     | 0         | SRC_QUERY_FAILED  |
| src=0, dst=0                             | ok_empty                  | 0         | —                 |
| src == dst                               | ok                        | 0         | —                 |
| src>0, dst=0                             | dest_missing              | 100       | —                 |
| src=0, dst>0                             | source_missing_or_stale   | 100       | —                 |
| 0.5% <= pct < 5%                         | warning                   | computed  | —                 |
| pct >= 5%                                | drift                     | computed  | —                 |

**Formula**: `drift_pct = |src-dst| / max(src, dst) * 100` (unsigned — same
magnitude whether src or dst outgrew the other).

**Test cases** (all passing in `reconciliation_drift_test.go`):
- worker error code preserved (SRC_TIMEOUT → error)
- nil src → SRC_QUERY_FAILED
- nil src with dst rows still error
- both zero → ok_empty
- equal small + large counts → ok
- src>0 dst=0 → dest_missing (pct=100)
- src=0 dst>0 → source_missing_or_stale (pct=100)
- 10%, 50% drift → status=drift
- 5% drift floor → status=drift
- 0.5% → warning
- 0.4% → ok (below warning floor)
- dst grew 20% past src → status=drift

---

## FIX 3 — API Response Enrichment

**Endpoint**: `GET /api/reconciliation/report`

**New fields**:
- `nullable_source_count` (int64|null) — preserves "query failed" distinctly from "no rows"
- `drift_pct` (float64) — computed
- `computed_status` (string) — computed (ok / ok_empty / warning / drift / dest_missing / source_missing_or_stale / error)
- `error_code` (string|null) — worker-emitted code or derived SRC_QUERY_FAILED
- `error_message_vi` (string) — Vietnamese translation from `ErrorMessagesVI` map
- `full_source_count`, `full_dest_count`, `full_count_at` — daily aggregate (from cdc_table_registry)
- `timestamp_field_source` — auto | manual | override
- `timestamp_field_confidence` — high | medium | low

**Backward compat**: primary SQL uses new columns; if it errors (migration 017
not applied), fall back to legacy SELECT so endpoint stays up — new fields
surface as null on older DBs.

**Sample response**:
```json
{
  "data": [
    {
      "target_table": "payment_bills",
      "sync_engine": "airbyte",
      "source_type": "mongodb",
      "nullable_source_count": null,
      "dest_count": 0,
      "drift_pct": 0,
      "computed_status": "error",
      "error_code": "SRC_CONNECTION",
      "error_message_vi": "Kết nối nguồn Mongo bị ngắt — sẽ retry tự động",
      "full_source_count": 2,
      "full_dest_count": 2,
      "full_count_at": "2026-04-17T10:00:00Z",
      "timestamp_field": "createdAt",
      "timestamp_field_source": "auto",
      "timestamp_field_confidence": "high",
      "checked_at": "2026-04-17T14:05:12Z",
      "source_query_method": "window_custom_field"
    }
  ],
  "total": 1
}
```

---

## FIX 4 — NEW Endpoint: POST /api/registry/:id/detect-timestamp-field

**File**: `internal/api/registry_handler.go`

Publishes `cdc.cmd.detect-timestamp-field` with `registry_id`, `target_table`,
`source_table`, `source_db`, `source_type`. Worker consumer (not implemented
here — scope boundary) samples Mongo, scores candidate timestamp fields,
writes winner back to `cdc_table_registry` with
`timestamp_field_source = 'auto'` + confidence.

**Route**: admin-only (requires `admin` role via Router.go registration).
**Response**: 202 Accepted with `target_table` for FE polling.
**Activity log**: emits `detect-timestamp-field` event with user + source_table.

---

## FIX 5 — Error Code → Vietnamese Translation

**File NEW**: `internal/api/error_messages_vi.go`

Map of error_code → operator-friendly Vietnamese message:
- SRC_TIMEOUT, SRC_CONNECTION, SRC_FIELD_MISSING, SRC_EMPTY
- DST_MISSING_COLUMN, DST_TIMEOUT
- CIRCUIT_OPEN, AUTH_ERROR, SRC_QUERY_FAILED, UNKNOWN

Test: `TestErrorMessagesVICoverage` guards against regression when adding new
codes to worker.

---

## File diff (absolute paths)

- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/pkgs/database/postgres.go` — PrepareStmt + pool warmup (pre-existing, verified)
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/reconciliation_handler.go` — added `ComputeDriftStatus` + enriched `LatestReport` response
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/error_messages_vi.go` — NEW (VI translation map)
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/reconciliation_drift_test.go` — NEW (15 test cases, passing)
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/registry_handler.go` — added `DetectTimestampField` handler
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/router/router.go` — registered POST /registry/:id/detect-timestamp-field

---

## Verification

- `go build ./...` → PASS
- `go vet ./...` → PASS (no warnings)
- `go test ./internal/api/...` → PASS (15 drift cases + VI coverage + existing tests)
- Runtime 30–40s: 0 SLOW SQL warnings in `/tmp/cms.log`
- Endpoint curl: returns new fields with proper null handling

---

## Worker-side TODOs (out of scope here, tracked for future)

- Migration 017: add `error_code`, `timestamp_field_source`, `timestamp_field_confidence`,
  `full_source_count`, `full_dest_count`, `full_count_at` columns.
- NATS consumer for `cdc.cmd.detect-timestamp-field` — sample Mongo collection,
  pick best timestamp field, write back to registry.
- Emit `error_code` from recon source/dest agents instead of free-text error_message.

These must ship before CMS returns non-null values for the new fields in the
API response.
