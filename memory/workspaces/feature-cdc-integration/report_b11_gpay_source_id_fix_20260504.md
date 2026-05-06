# Report — B11 _gpay_source_id Ingest Population Fix
**Date**: 2026-05-04 (Asia/Ho_Chi_Minh)
**Author**: Brain (Antigravity) — plan + DATA backfill + governance closure
**Executor**: Muscle (CC sub-agent, sonnet-4-6)
**Type**: Permanent code fix (`.go`) + already-applied DATA backfill
**Status**: ✅ DONE — verified live (shadow + master + cron tick + worker logs)

---

## 1. Bug

Master `dw_orders.orders_fact` UNIQUE constraint on `_gpay_source_id`. Shadow ingest path (Debezium → cdc-worker → `shadow_*`) **không populate `_gpay_source_id`** trên đường V2 (`schema_adapter.BuildUpsertSQLInSchema`). Mọi shadow row mới có `_gpay_source_id = ''` (empty string) → khi transmute đẩy lên master, ON CONFLICT (`_gpay_source_id`) collide → 1 row giữ lại, các row khác bị overwrite.

**Triệu chứng quan sát được trước fix**:
- shadow `_gpay_source_id` NULL/empty cho rows 56-63 (ingest qua V2 path).
- master `dw_orders.orders_fact` count = 26 (chỉ chứa rows pre-V2 + 1 row `_gpay_source_id=''`).
- Mỗi cron tick: `transmute complete scanned=13 inserted=5 updated=8 skipped=0` — count master không tăng theo shadow.

## 2. Root cause (3-layer trace, theo lesson L-three-layer-trust 2026-04-29)

| Layer | File | Vấn đề |
|-------|------|--------|
| A. Event handler | `internal/handler/event_handler.go:113-126` | Build `UpsertRecord` với `MappedData = mapped.Columns` — không inject `_gpay_source_id` |
| B. Batch buffer | `internal/handler/batch_buffer.go:158-162` | Truyền `MappedData` raw vào `BuildUpsertSQLInSchema` |
| C. Schema adapter | `internal/service/schema_adapter.go:404-520` | Loop business cols + meta cols (`_raw_data, _source, _synced_at, _hash, _deleted, _created_at, _updated_at, _version`) — **không có branch ghi `_gpay_source_id`** |

V1 path (cũ) ghi `_gpay_source_id = id` trong DB-side default hoặc trigger. V2 logical-clone path mới (B3) routes qua `BuildUpsertSQLInSchema` nhưng quên port branch này.

So sánh: Mongo path (`internal/sinkworker/sinkworker.go:146`) có `record["_gpay_source_id"] = sourceID` đúng. Postgres/Debezium path thiếu equivalent.

## 3. Fix — Option A (chosen, minimal blast radius)

Đặt branch ghi `_gpay_source_id` ngay trong `BuildUpsertSQLInSchema` — guard bằng schema reflection để backward-compat với V1 tables không có cột.

**File**: `internal/service/schema_adapter.go`
**Function**: `BuildUpsertSQLInSchema` (line 404-520)

### Change 1 — INSERT branch (sau block ghi `_hash`, ~line 452)
```go
// V2 anchor key — chỉ ghi nếu shadow/master có cột _gpay_source_id.
// Source PK value đảm bảo distinct → master ON CONFLICT (_gpay_source_id) chạy đúng.
if _, ok := schema.Columns["_gpay_source_id"]; ok {
    allCols = append(allCols, `"_gpay_source_id"`)
    allPlaceholders = append(allPlaceholders, "?")
    finalValues = append(finalValues, fmt.Sprintf("%v", pkValue))
}
```

### Change 2 — ON CONFLICT UPDATE branch (~line 487)
```go
if _, ok := schema.Columns["_gpay_source_id"]; ok {
    updateSets = append(updateSets, `"_gpay_source_id" = EXCLUDED."_gpay_source_id"`)
}
```

### Trade-off
- `pkValue interface{}` → cast string qua `%v`. Đảm bảo TEXT col luôn có giá trị xác định, không empty.
- Schema reflection (`schema.Columns["_gpay_source_id"]`) cho V1 tables (không có cột) → nguyên si V1 behavior, no regression.

---

## 4. DoD verification (Muscle thực thi)

| Item | Result |
|------|--------|
| `go build ./...` | ✅ PASS |
| `go test ./internal/service/...` | ✅ PASS (trừ 2 pre-existing failures `TestSchemaValidatorDriftDetection` + `TestExtractDLQMetadata_NonJSONValue` — verified via `git stash` không phải B11 regression) |
| `go test ./internal/handler/...` | ✅ PASS |
| Unit test `TestBuildUpsertSQL_PopulatesGpaySourceID` | ✅ ADD + PASS — 2 cases: V2 schema có cột → SQL chứa `"_gpay_source_id"` + EXCLUDED; V1 schema không có cột → SQL không chứa cột (backward-compat) |
| Docker rebuild + restart `gpay-cdc-worker` | ✅ DONE |

## 5. Live smoke test (real evidence post-fix)

### 5.1 INSERT new source row id=64
```sql
INSERT INTO public.orders (user_id, amount, status, notes)
VALUES (8888, 77.77, 'pending', 'b11-permanent-fix-smoke') RETURNING id;
-- id=64
```

### 5.2 Shadow auto-populated (NO manual backfill)
```
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
 "SELECT id, _gpay_source_id FROM shadow_goopay_source.orders ORDER BY id::int DESC LIMIT 5;"

 id | _gpay_source_id
----+-----------------
 64 | 64
 63 | 63
 62 | 62
 61 | 61
 60 | 60
```
✅ Row 64 mới có `_gpay_source_id='64'` AUTO populated bởi code fix.

### 5.3 Master post-cron-tick
```
docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest -c \
 "SELECT count(*) AS total, count(DISTINCT _gpay_source_id) AS distinct_ids FROM dw_orders.orders_fact;"

 total | distinct_ids
-------+--------------
    34 |           34
```
✅ count = 34 (was 33 before INSERT id=64), distinct = 34 (no collision).

### 5.4 Worker logs (transmute scheduler chạy đều)
```
transmute complete master=orders_fact scanned=14 inserted=14 updated=0 skipped=0 type_errors=0 rule_misses=0 duration_ms=23
job monitor: schedule closed schedule_id=1 status=success master=orders_fact
```
✅ Cron tick chạy success, JobMonitor (P4) đóng loop schedule.

---

## 6. DATA backfill (Brain đã apply trước code fix — 2026-05-04 04:30)

```sql
-- Backfill shadow rows 56-63 (V2 path đã ingest mà thiếu _gpay_source_id):
UPDATE shadow_goopay_source.orders
   SET _gpay_source_id = id
 WHERE _gpay_source_id IS NULL OR _gpay_source_id = '';
-- UPDATE 8

-- Cleanup master residue row stuck với _gpay_source_id='':
DELETE FROM dw_orders.orders_fact WHERE _gpay_source_id = '';
-- DELETE 1
```

**Idempotency check**: chạy lại lần 2 = WHERE filter rỗng = 0 rows updated. An toàn.

## 7. Files changed

| Path | Type | Owner | Action |
|------|------|-------|--------|
| `internal/service/schema_adapter.go` | code (Go) | Muscle | EDIT — INSERT branch + UPDATE branch for `_gpay_source_id` |
| `internal/service/schema_adapter_test.go` | code (Go test) | Muscle | NEW — `TestBuildUpsertSQL_PopulatesGpaySourceID` |
| `shadow_goopay_source.orders` (DB rows) | DATA | Brain | UPDATE 8 rows backfill |
| `dw_orders.orders_fact` (DB rows) | DATA | Brain | DELETE 1 residue row |

**Brain KHÔNG sửa file `.go/.sql/.ts/.js/.py`** (CLAUDE.md §12). Brain chỉ làm DATA backfill SQL trên DB live + plan + governance closure.

---

## 8. Idempotency analysis

- **Code fix**: ingest mới cho row đã backfill sẽ UPDATE `_gpay_source_id = pkValue` (cùng giá trị) → no-op delta. An toàn.
- **DATA backfill SQL**: chạy lại = WHERE filter rỗng → 0 rows updated.
- **Master DELETE**: residue row `_gpay_source_id=''` đã bị xóa, không tái tạo (code fix luôn ghi pkValue ≠ '').

## 9. Risks (post-deploy)

- **V1 legacy tables không có cột `_gpay_source_id`**: branch `if _, ok := schema.Columns[...]` đã guard. ✅ Tested in unit test case 2.
- **pkValue NULL/empty**: nếu source PK NULL, `fmt.Sprintf("%v", nil) = "<nil>"` — sẽ insert string `"<nil>"`. **Acceptable** vì source schema enforces NOT NULL trên PK; nếu xảy ra = source corruption upstream.
- **Master mapping rule có target cho `_gpay_source_id`**: hiện tại mapping_rule_v2 không bind cột này (master derive từ ON CONFLICT), nên không conflict.

## 10. Lessons applied

- ✅ **L-three-layer-trust (2026-04-29)**: Trace từ master upsert SQL → shadow data → ingest code path → identify missing write at schema_adapter layer.
- ✅ **L-real-data-test (2026-04-15)**: Verify TRƯỚC + SAU bằng INSERT row thật + worker log + DB query, không assume.
- ✅ **L-runtime-state-verify (2026-04-21)**: Đọc shadow + master + worker log live state TRƯỚC khi propose fix.

## 11. NEW lesson (sẽ append `agent/memory/global/lessons.md`)

**Global Pattern [V1→V2 path migration: schema-evolution path G writes upsert SQL but forgets to populate constraint-keyed anchor column C that exists only in V2 schema] → Result: V2 master ON CONFLICT (C) collapses N distinct rows into 1**

**Correct Pattern**: Khi migrate ingest path V1 → V2, nếu V2 schema thêm UNIQUE/anchor column C derived từ source PK, phải:
1. Audit MỌI write path (event_handler, batch_buffer, schema_adapter, sinkworker, …) cho đường V2.
2. Ở generator SQL (schema_adapter.BuildUpsertSQL): branch `if schema.Columns[C] exists → write pkValue`.
3. Mirror branch trong ON CONFLICT UPDATE để re-emit cùng giá trị.
4. Unit test 2 cases: schema có C (V2) + schema không có C (V1 backward-compat).
5. Live smoke: INSERT row mới → assert shadow.C = pkValue (NOT NULL/empty) WITHOUT manual backfill.

Tags: #cdc #schema-migration #v1-v2 #anchor-key #on-conflict #ingest-path #three-layer-trust

## 12. Governance compliance (CLAUDE.md pre-flight §14)

- ✅ §0: Trả lời tiếng Việt + plan-first (`09_tasks_solution_b11_gpay_source_id_20260504.md`).
- ✅ §1: Brain chỉ Chairman (plan + DATA SQL); Muscle là Chief Engineer (code edit).
- ✅ §2: Lệnh delegate gồm [Mô tả lỗi 3-layer] + [Logs/DB query] + [DoD].
- ✅ §3: Verify thực tế bằng row count + distinct count + worker log + cron tick consecutive.
- ✅ §7: Đọc `lessons.md` TRƯỚC khi action; sẽ APPEND new lesson sau.
- ✅ §11: Memory file APPEND-only — file này NEW, `05_progress.md` APPEND.
- ✅ §12: Brain KHÔNG sửa `.go/.sql/.ts/.js/.py` — chỉ DATA UPDATE/DELETE trên live DB (không phải code repo).
- ✅ §13: NEW lesson abstract thành Global Pattern với biến A/B/X/Y, generalize được sang 3+ scenarios.
- ✅ §14: Pre-flight scan trước khi end turn.

## 13. Skills used
- Go code editing (delegate Muscle)
- Unit test design (table-driven 2 cases)
- `go build ./...` / `go test ./internal/...`
- Docker compose rebuild + restart
- psql DB forensics (shadow + master cross-check)
- Worker log live tail (cron tick assertion)
- Three-layer trust trace methodology
- Memory APPEND-only protocol
- Global Pattern lesson abstraction (CLAUDE.md §13)
- Plan-first + delegate-execute split (CLAUDE.md §1, §12)

---

**Summary**: B11 mapping derivation bug đã được fix permanent (CODE) + retroactive (DATA backfill). Live verified: shadow row 64 auto-populate `_gpay_source_id='64'`, master count 33→34 distinct=34, no collision. Ingest path V2 giờ ngang feature parity với V1 cho anchor key `_gpay_source_id`.
