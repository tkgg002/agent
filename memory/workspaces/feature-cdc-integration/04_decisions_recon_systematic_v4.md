# ADR — Reconciliation Systematic v4 (Stability + Scale 200+ tables)

> **Date**: 2026-04-20
> **Author**: Brain (claude-opus-4-7)
> **Context**: Session 04-20 phát hiện recon bất ổn + UX error vô nghĩa. User feedback: "với quy mô 200 table, fix từng cái à, ngu đần. cái cần là giải pháp thông minh, không phải tình thế".
> **Decision**: Bỏ per-table band-aid. Implement systematic auto-detect + fallback + UX contract.

---

## 1. Problem statement

### 1.1 Observed bugs
- `payment_bills src=0 dst=0 Lỗi nguồn` — Mongo 2 docs với `createdAt`, registry `timestamp_field='updated_at'` → query return 0 → Mongo transient error masks as 0.
- `export_jobs src=2 dst=11 -450%` — Drift formula SIGNED + base wrong → phép tính vô nghĩa.
- `refund_requests src=0 dst=3422 Lệch` — Source fail silent, dest count window-based (lookback 7d match 3422 PG rows).
- `"Lỗi nguồn"` generic — không phân biệt timeout/connection/field_missing/circuit_open.
- SLOW SQL recurring — GORM `PrepareStmt` không set.

### 1.2 Root pattern
Per-table config (`timestamp_field`) require admin set manually per table × 200 tables = O(N) human error. Source error không propagate structured → `source_count=0` fallback ngụy trang query failure as "no data".

---

## 2. Design Decisions

### 2.1 Registry extension: timestamp config per-table with auto-detect

**Schema change** (migration 017):
```sql
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS timestamp_field_candidates JSONB 
    DEFAULT '["updated_at","updatedAt","created_at","createdAt"]'::jsonb;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS timestamp_field_detected_at TIMESTAMPTZ;
ALTER TABLE cdc_table_registry ADD COLUMN IF NOT EXISTS timestamp_field_source TEXT 
    DEFAULT 'auto';  -- auto | admin_override
```

**Auto-detect logic** (Worker service `internal/service/timestamp_detector.go` NEW):
```
DetectTimestampField(ctx, registry_entry):
    candidates = registry_entry.timestamp_field_candidates (default chain)
    samples = mongo_coll.Find({}).Limit(100)
    field_coverage = {}
    for doc in samples:
        for candidate in candidates:
            if hasField(doc, candidate):
                field_coverage[candidate]++
    
    // Rank by coverage DESC, then by chain order
    sorted = rank_by_coverage(field_coverage)
    
    if top.coverage >= 80%:
        update registry.timestamp_field = top
        log "auto-detected {field} for {table} with {coverage}% confidence"
    else:
        log warn "low confidence detection, admin should override"
        // Fall back to _id.getTimestamp() (ObjectID embedded)
        set timestamp_field = '_id'
        special_handler = extractFromObjectId
```

**Trigger points**:
1. On `POST /api/registry` register → auto-detect immediately.
2. On recon source query returns 0 docs for K=3 consecutive runs → auto-redetect + fallback.
3. Manual button on FE `Registry.tsx` → "Re-detect timestamp field" per table.

**Admin override**:
- FE form `timestamp_field_source='admin_override'` → Worker không auto-overwrite.
- Reset to auto: toggle back to 'auto' → next detect cycle overwrites.

### 2.2 Drift % formula correction

**Before**: `diff = src - dst; drift_pct = diff / src * 100` → signed, unbounded, division-by-zero khi src=0.

**After**:
```
drift_pct = ABS(src - dst) / GREATEST(src, dst, 1) * 100
Status:
    src == dst == 0: 'ok (empty)'
    src == dst > 0:  'ok'
    src > 0 AND dst == 0: 'dest_missing' (drift 100%)
    src == 0 AND dst > 0: 'source_missing_or_stale' (drift 100%)
    |drift_pct| < 0.5%: 'ok'
    |drift_pct| < 5%: 'warning'
    |drift_pct| >= 5%: 'drift'
```

### 2.3 Error message categorization

**Schema change** (migration 017):
```sql
ALTER TABLE cdc_reconciliation_report ADD COLUMN IF NOT EXISTS error_code TEXT;
```

**Codes** (enum):
```
SRC_TIMEOUT        — Mongo query timeout >5s
SRC_CONNECTION     — Mongo connection error (EOF, incomplete read)
SRC_FIELD_MISSING  — timestamp_field không tồn tại trong collection
SRC_EMPTY          — Query OK, 0 results (không phải error)
DST_MISSING_COLUMN — PG thiếu _source_ts
DST_TIMEOUT        — PG query timeout
CIRCUIT_OPEN       — Circuit breaker open, retry sau N seconds
AUTH_ERROR         — DB connection auth fail
UNKNOWN            — Uncategorized
```

**FE translation table** (tiếng Việt):
```typescript
const ERROR_MESSAGES_VI = {
  SRC_TIMEOUT: 'Nguồn phản hồi chậm (>5s) — Mongo có thể đang overload',
  SRC_CONNECTION: 'Kết nối nguồn Mongo bị ngắt — sẽ retry tự động',
  SRC_FIELD_MISSING: 'Field timestamp không tồn tại ở nguồn — chạy re-detect',
  SRC_EMPTY: 'Nguồn trống trong window 7 ngày — bình thường cho data cũ',
  ...
}
```

### 2.4 Source query error propagation

**Current bug**: Source query fail → returned 0 → report `source_count=0, status=drift` (misleading).

**Fix**:
```go
type WindowResult struct {
    Count     int64
    XorHash   uint64
    Error     error   // nil if success
    ErrorCode string  // SRC_TIMEOUT / SRC_CONNECTION / ...
}

// In recon_core:
if result.Error != nil {
    report.Status = "error"
    report.ErrorCode = result.ErrorCode
    report.SourceCount = nil  // NULL, không 0 (distinguish failure vs real 0)
    log warn "recon failed for {table} reason={code}"
    return  // Không tính drift khi source fail
}
```

**Schema**: `cdc_reconciliation_report.source_count` đổi `BIGINT NULL` (hiện NOT NULL default 0).

### 2.5 Mongo retry + transient error handling

Wrap Mongo query calls với retry helper (đã design trong Muscle brief trước, keep):
- Max 3 attempts, backoff 500ms × attempt#
- Match transient patterns: `incomplete read`, `EOF`, `connection reset`, CommandError codes [6,7,89,91,189]
- Non-transient (auth error, bad query) → fail-fast

### 2.6 CMS stability (SLOW SQL)

**Fix** `pkgs/database/postgres.go`:
```go
gorm.Config{
    PrepareStmt: true,  // Cache statements per connection
    Logger:      ...,
}
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxIdleTime(30*time.Minute)
sqlDB.SetConnMaxLifetime(1*time.Hour)

// Pre-warm pool
for i := 0; i < 5; i++ { sqlDB.Ping() }
```

### 2.7 Dest count window semantic fix (refund_requests dst=3422 vs Mongo 1719)

**Root cause**: Dest agent filter `_source_ts >= windowLo_ms` (ms epoch). `refund_requests` có 3422 rows trong window 7 ngày. BUT Mongo source `updated_at` max=17/4 → chỉ 4 docs in window. Window count asymmetric because:
- Worker ghi PG `_source_ts` = `message.source.ts_ms` = Debezium snapshot time OR change time.
- Snapshot reads all 1719 Mongo docs at time T → writes 1719 rows với `_source_ts=T`. 
- Nếu snapshot chạy recent → PG 1719 rows có `_source_ts` gần đây → window match.
- Mongo doc `updated_at` = last business update → có thể cũ.
- → Source query filter by `updated_at` (recent < 4 docs), Dest query filter by `_source_ts` (snapshot time, all 1719) → asymmetric.

**Fix**: **Both sides filter by SAME semantic field**.
- Source Mongo: filter by `timestamp_field` (updated_at / createdAt).
- Dest PG: filter by `_source_ts` THEN convert to **doc timestamp space** using same field (e.g., PG table store Debezium's source.ts_ms which = Mongo oplog timestamp — usually ≈ Mongo updated_at).

Actually cleaner: **both filter by docTs**, where:
- Mongo: `doc.{timestamp_field}` as BSON DateTime.
- PG: `_source_ts BIGINT ms` from Debezium metadata = equivalent of oplog clusterTime ≈ Mongo updated_at event time.

Verify: inspect 1 PG row's `_source_ts` vs Mongo same `_id`'s `updated_at` — should match within ms.

If ASYMMETRIC (snapshot backfill timestamps differ from oplog event time), then:
- Prefer dest to use stored oplog ts from Debezium (already should be source.ts_ms).
- Mongo side filter by `ts_ms` extracted from oplog (NOT `updated_at` doc field).

For now: registry has `timestamp_field` for Mongo side; **Dest always uses `_source_ts`**. Assume source.ts_ms ≈ updated_at within seconds. Drift of 3422 vs 4 suggests snapshot loaded full collection BUT Mongo data updated_at static → mismatch.

**Actual clean fix**: During snapshot, Debezium uses source.ts_ms = snapshot start time (NOT doc's own updated_at). So PG `_source_ts` ≠ Mongo `updated_at`.

**Solution**: Add another PG column `_event_ts_ms BIGINT` = equivalent of `updated_at` (extract from payload.after.updated_at if present). Then dest filter `WHERE _event_ts_ms >= windowLo`.

Complex. **Compromise for v4**: 
- Document clearly: recon window compare is "recent event" not "full table". Drift reports don't mean data loss, just activity mismatch in window.
- Add NEW column: `total_source_count` + `total_dest_count` — **full unfiltered** count (once per day, cache).
- Drift status derived from full count match primarily, window count secondary.

### 2.8 UX — FE error + error code rendering

FE `DataIntegrity.tsx` thêm cột:
- Render `error_code` với icon + VI label lookup.
- Tooltip trên drift % giải thích formula.
- Tooltip trên src/dst giải thích "window-based" vs full count.

---

## 3. Implementation Plan (Muscle tasks)

### Phase A — Stability hardening (1-2h)
- A1: GORM PrepareStmt + pool warmup in `cdc-system/cdc-cms-service/pkgs/database/postgres.go`.
- A2: Mongo retry helper `cdc-system/centralized-data-service/internal/service/mongo_retry.go`.

### Phase B — Systematic timestamp detection (3-4h)
- B1: Migration 017 — schema changes (candidates, detected_at, source field).
- B2: `internal/service/timestamp_detector.go` — auto-detect logic.
- B3: Integrate vào register flow + recon failure fallback.
- B4: FE button "Re-detect" + admin override form.

### Phase C — Correct drift formula + error codes (2h)
- C1: Update `recon_core.go` status computation.
- C2: Migration `cdc_reconciliation_report.error_code TEXT`, `source_count BIGINT NULL`.
- C3: FE error translation + drift tooltip.

### Phase D — Full count dashboard (2h)
- D1: Worker daily aggregate job → PG `cdc_table_registry.full_source_count, full_dest_count, counted_at`.
- D2: API response include full counts.
- D3: FE new columns "Total Source / Total Dest".

---

## 4. Success Criteria

- [ ] 0 SLOW SQL warnings in CMS after restart (PrepareStmt effective).
- [ ] Register new table → timestamp_field auto-detected correctly (test: seed 3 Mongo collections with different field patterns).
- [ ] payment_bills / export_jobs / refund_requests — all show correct source count (no silent 0 when Mongo has data).
- [ ] Drift % bounded 0-100%, signed fixed.
- [ ] Error code structured, FE shows VI label not "Lỗi nguồn" generic.
- [ ] Total Source / Dest full count column visible.
- [ ] Recon stable over 10 cycles (no circuit open, no transient error uncovered by retry).

---

## 5. Anti-patterns rejected

- ❌ Manual UPDATE registry SET timestamp_field='X' WHERE table='Y' per table (tình thế, không scale).
- ❌ Cap SLOW SQL threshold higher (hide symptom).
- ❌ Return 0 when query fails (mask error as "empty").
- ❌ Assume Mongo source always has `updated_at` (schema-less, false assumption).

---

## 6. Reference lessons

- Lesson #60: ADR passive không enforce → repeat violation
- Lesson #62: Partitioned table default orphan → auto-backfill
- Lesson #63: Silent-skip scheduled jobs mask init failure
- Lesson #64: Hard-coded field name cross-store schema drift
- Lesson #65: Per-entity band-aid không scale N entities (session này)

---

## 7. Out of scope (future iteration)

- Merkle tree full-table compare (expensive O(N), defer)
- Multi-source validation (Airbyte + Debezium concurrent)
- ML-based anomaly detection
- Automated heal trigger on drift > threshold
