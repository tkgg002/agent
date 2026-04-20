# 03_implementation_v3_heal_audit_fix.md

**Task**: EMERGENCY FIX — `ReconHealer.HealWindow` ghi 1 audit row PER record, scale prod 50M = disaster.
**Date**: 2026-04-17
**Author**: Muscle (claude-opus-4-7[1m])
**Related plan**: `02_plan_data_integrity_v3.md` §6 — "Batch audit log 100 heal actions/insert" (Muscle Phase 2/3 implement FAIL, regression fixed here)
**Files**:
- `centralized-data-service/internal/service/recon_heal.go` — refactored
- `centralized-data-service/internal/service/recon_heal_audit_integration_test.go` — NEW (build tag `integration`)

---

## 1. Root Cause

File `internal/service/recon_heal.go` trước fix:

```go
auditBuf := make([]model.ActivityLog, 0, rh.cfg.AuditFlushSize)
flushAudit := func() { ... CreateInBatches(&auditBuf, rh.cfg.AuditFlushSize) }

for _, chunk := range chunkStrings(ids, rh.cfg.BatchSize) {
    ...
    for _, doc := range docs {
        action, err := rh.applyOne(...)
        ...
        auditBuf = append(auditBuf, rh.buildAuditRow(..., action, ..., doc))
        if len(auditBuf) >= rh.cfg.AuditFlushSize { flushAudit() }
    }
    for _, s := range chunk {
        if !fetched[s] { res.Skipped++ /* no audit — only metric counter */ }
    }
}
```

**Bug thực tế**: mỗi 1 doc fetched từ Mongo → 1 `auditBuf = append(...)` → 1 row vào `cdc_activity_log`. Batching ở đây chỉ giảm số round-trip INSERT (N/100) nhưng KHÔNG giảm số row (vẫn N). Plan v3 §6 intent là "log 100 heal actions/insert" nhưng implement đã hiểu sai thành "flush buffer mỗi 100 insert" — vẫn 1 row per action.

**Scale impact**:
- Prod 50M records heal → 50M audit rows. Partition `cdc_activity_log` daily giữ 30 day (migration 010) → vẫn overflow storage + index + backup.
- Local damage confirm: 1712 real records → 5136 audit rows (1712 upsert + 3424 skip). Skip gấp đôi vì vừa log trong loop vừa log trong "IDs requested but not returned" branch.

**Skip = no-op semantically**: record exists với `_source_ts` mới hơn → OCC rejected → DB không change. Log per-record skip không có diagnostic value.

---

## 2. Fix Pattern — Sampling + Aggregate Summary

NEW `healAuditBatcher` struct trong `recon_heal.go`:

| Action | Behavior | Row Count per Run |
|--------|----------|-------------------|
| `skip` | counter-only, in-memory | **0** |
| `upsert` | sample ≤ 100, extra counted-only | **≤ 100** |
| `error` | always persist (rare + actionable) | **= errorCount** |
| `run_started` | 1 row at Begin() | **1** |
| `run_completed` | 1 row at End() với aggregate counters | **1** |

**Total per run** = `1 + min(upserts, 100) + errorCount + 1`

Với run healthy typical (no error): **≤ 102 rows**.
Với 10K error run (catastrophic): **1 + 100 + 10000 + 1 = 10102 rows** — vẫn reasonable vì errors đáng xem.

---

## 3. Implementation Highlights

### 3.1 `healAuditBatcher.Begin(ctx, runID, table)`
Insert ngay 1 row `{action: "run_started", run_id: X}` — head marker, không batch (rẻ, 1 row/run).

### 3.2 `healAuditBatcher.Record(ctx, action, recordID, srcTsMs, errMsg)`
- Switch theo action: skip → `skipCount++; return`. Upsert → `upsertCount++; if upsertLogged >= maxSampleUpsert return`. Error → luôn append.
- `append(&buf, row)`; if `len(buf) >= maxBatch` → `flushLocked` — single multi-row INSERT qua `CreateInBatches`.
- Mutex-guarded: `ReconHealer` sẽ multi-goroutine khi Phase A signal + Phase B direct chạy interleaved.

### 3.3 `healAuditBatcher.End(ctx, status, err, usedSignal, signalID)`
- `Flush()` drain buffer còn sót.
- Insert 1 row `run_completed` với full counters JSON:
  ```json
  {"action":"run_completed","run_id":"heal-20260417T092101.588-b4c840",
   "status":"success","duration_ms":12,"used_signal":false,
   "audit_flushes":2,"upserted_count":10000,"skipped_count":10000,
   "errored_count":50}
  ```

### 3.4 `HealWindow` owns ONE batcher cho cả Phase A + Phase B
- Trước: `HealMissingIDs` tự allocate local `auditBuf` → mỗi call tự flush riêng.
- Sau: `HealWindow` Begin batcher → pass vào `healMissingIDsWithBatcher(...)` inner → End batcher trong `defer`.
- Kết quả: 1 HealWindow invocation = 1 run_started + 1 run_completed regardless phases.

### 3.5 `HealMissingIDs` standalone (không qua `HealWindow`)
Vẫn tự wrap Begin/End riêng để callsite trực tiếp (tests, CMS ad-hoc) cũng có summary rows.

### 3.6 `newHealRunID()`
`heal-20060102T150405.000-XXXXXX` — timestamp với ms precision + 6 hex suffix từ UnixNano low bits. Dep-free, sortable, collision-safe trong 1 process.

---

## 4. Before/After Evidence

### 4.1 PG row count — BEFORE cleanup
```sql
SELECT operation, details->>'action' AS action, COUNT(*)
FROM cdc_activity_log
WHERE operation='recon-heal' AND triggered_by='recon-healer'
GROUP BY operation, details->>'action' ORDER BY 3 DESC;
```
```
 operation  | action | count
------------+--------+-------
 recon-heal | skip   |  3424
 recon-heal | upsert |  1712
(2 rows)
```
Total: **5136 spam rows** cho 2 heal runs (mỗi run ~1712 records).

### 4.2 Cleanup
```sql
DELETE FROM cdc_activity_log
WHERE operation='recon-heal' AND triggered_by='recon-healer'
AND details->>'action' IN ('skip','upsert');
```
Result: `DELETE 5136`. Post-check: 0 rows.

### 4.3 Scale integration test — AFTER fix
```bash
$ go test -tags=integration -run TestHealAuditBatcher_ScaleCap -count=1 -v ./internal/service/...
=== RUN   TestHealAuditBatcher_ScaleCap
    recon_heal_audit_integration_test.go:123: scale test summary: 152 total rows for 10000 upserts + 10000 skips + 50 errors (cap 152)
    recon_heal_audit_integration_test.go:139: run_completed details: {"action": "run_completed", "run_id": "heal-20260417T092101.588-b4c840", "status": "success", "duration_ms": 12, "used_signal": false, "audit_flushes": 2, "errored_count": 50, "skipped_count": 10000, "upserted_count": 10000}
--- PASS: TestHealAuditBatcher_ScaleCap (0.05s)
PASS
```

**Input**: 10,000 upsert + 10,000 skip + 50 error = 20,050 actions.
**Before fix**: 20,050 rows.
**After fix**: 152 rows = 1 start + 100 upsert sample + 50 error + 1 completed.
**Reduction**: **132x** (20050/152).

### 4.4 Scale projection — prod 50M records
| Scenario | Before | After |
|----------|--------|-------|
| 50M all upsert | 50,000,000 rows | 102 rows |
| 50M all skip (re-heal no-op) | 50,000,000 rows | 2 rows |
| 50M with 1% error (500K) | 50,000,000 rows | 500,102 rows (errors persist) |
| 50M with 0.01% error (5K) | 50,000,000 rows | 5,102 rows |

**At-rest prod impact**: disk + index + replication + backup all reduced by same factor.

---

## 5. Verify Checklist

- [x] `go build ./...` PASS
- [x] `go vet ./...` clean
- [x] `go test ./internal/service/... -count=1` PASS (existing heal tests still green)
- [x] `go test -tags=integration -run TestHealAuditBatcher_ScaleCap` PASS — live PG, 10K upsert + 10K skip + 50 error → 152 rows
- [x] `run_completed` row carries accurate aggregates (verified in test log)
- [x] `run_started` + `run_completed` always exactly 1 each per HealWindow
- [x] skip rows = 0 (in-memory count only)
- [x] upsert rows ≤ 100 per run (sampling works)
- [x] error rows = errorCount (never sampled)
- [x] Existing 5136 spam rows cleaned from `gpay-postgres`

---

## 6. Global Pattern Lesson Candidate

**Pattern**: `[A logs B per record of X] → Result [audit table overflow O(N) at scale]`.

**Correct flow**:
1. Skip/no-op decisions → aggregate counter only, single summary row per run.
2. Rare errors → persist each (actionable).
3. Happy-path events → sample (first 100 representative) + aggregate counter.
4. Emit 1 `run_started` + 1 `run_completed` with counters per run for operator observability.
5. Batching writes (multi-row INSERT) reduces round-trips but does NOT reduce N. Must combine with sampling/aggregation to cap N itself.

**Rule of thumb**: if N is unbounded by business scale, audit must be O(1) + error tail, not O(N).

**Applicable**: CDC reconciliation, ETL validators, bulk processors, any hot-path decision log, event sourcing projection retries.

---

## 7. Files Changed

```
M centralized-data-service/internal/service/recon_heal.go          (+280, -80)
A centralized-data-service/internal/service/recon_heal_audit_integration_test.go  (NEW, build tag `integration`)
```

No migration required — reuses existing `cdc_activity_log` table schema (partitioned daily per migration 010).
