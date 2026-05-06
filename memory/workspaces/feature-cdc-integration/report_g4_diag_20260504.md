# G4 Diagnostic Report — `orders_addtest` master = 0 dù shadow = 11

**Date**: 2026-05-04 (16:55+07)
**Scope**: Phase E task E4 — diagnostic only, không auto-fix.
**Conclusion**: **Class E root cause** — V2 anchor port miss (pattern L-v1-v2-anchor-key-port). 8/11 shadow rows có `_gpay_source_id=NULL/empty` → transmuter skip silently → cursor advance → `scanned=0` mỗi tick.

---

## 1. Snapshot 5 layer (verified 2026-05-04)

### Layer A — `master_binding`

```
mb_id | binding_code | mb_active | master_table   | schema_status | sor_id | object_code         | src_active | provisioning_state
   28 | auto_src_29  | t         | orders_addtest | approved      | 29     | addtest_pg_orders   | t          | running
```
✅ binding active + approved + provisioning state running.

### Layer B — `transmute_schedule`

```
id | master_binding_id | mode | is_enabled | last_status | last_run_at                  | last_error
13 | 28                | cron | t          | success     | 2026-05-04 08:55:13.748 UTC  | (null)
```
✅ schedule enabled + recent + no error.

### Layer C — `last_stats` (smoking gun)

```json
{
  "scanned": 0,
  "skipped": 0,
  "updated": 0,
  "inserted": 0,
  "duration_ms": 2,
  "rule_misses": 0,
  "type_errors": 0
}
```
🔴 transmuter scan 0 rows — không phải skip, không phải type error, **không thấy data**.

### Layer D — Shadow data (where the smoke shows up)

| `_gpay_source_id` | `_gpay_id` | count |
|---|---|---|
| `NULL/empty`      | `NULL`     | **8** |
| `'64'`            | `NULL`     | 1     |
| `'88888'`         | `NULL`     | 1     |
| `'99999'`         | `NULL`     | 1     |

🔴 8/11 rows thiếu **cả 2** anchor cols. 3 rows có `_gpay_source_id` nhưng `_gpay_id` vẫn NULL — đây là rows từ P1.1 smoke (FIRST-TOUCH delete + INSERT-then-DELETE).

### Layer E — Shadow `orders` (working comparison)

```
_gpay_source_id | count
56              | 1
55              | 1
52              | 1
88888           | 1
54              | 1
...
```
✅ Mỗi row có `_gpay_source_id` distinct = source pk → V2 anchor đầy đủ. Đây là path ingest đang hoạt động tốt → master `orders_fact` = 35 rows.

### Layer F — Master DDL

```
dw_src_local_pg_source.orders_addtest  EXISTS
```
✅ DDL đã tạo (cascade master_bind step success).

---

## 2. Root cause

**Pattern**: L-v1-v2-anchor-key-port (lesson 2026-05-04, abstracted earlier this session).

**Mechanism cụ thể cho `addtest_pg_orders`**:

1. Source PG `public.orders_addtest` ingest qua Debezium → kafka topic.
2. cdc-worker consume → `BuildUpsertSQLInSchema` generate SQL → INSERT vào shadow.
3. Generator V2 cho ingest path này **không** explicit write `_gpay_source_id`. 8/11 rows landing với cột NULL/empty.
4. Transmuter mỗi 60s tick:
   - SELECT `WHERE _gpay_source_id IS NOT NULL AND _gpay_source_id <> '' AND <cursor condition>`.
   - 8 rows fail filter → bypass.
   - 3 rows P1.1 (`'64'`, `'88888'`, `'99999'`) đã được scan trong tick trước → cursor đã advance → tick hiện tại scan 0.
5. Master `dw_src_local_pg_source.orders_addtest` = 0.

**Đối chứng**: Shadow `goopay_source.orders` (qua V2 path khác — `BatchBuffer.batchUpsert` đường `_gpay_source_id` đã fix B11 trong session 2026-05-04 trưa) hoạt động bình thường. Source `addtest_pg_orders` (qua admin-api 2026-05-02 hoặc qua provisioning auto cascade) đi đường khác → fix B11 chưa cover.

---

## 3. Recommendation (KHÔNG fix trong Phase E)

### Option 1 (RECOMMENDED) — Trace ingest path của `addtest_pg_orders`
- File entry: `internal/handler/event_handler.go::processEvent` → `internal/service/dynamic_mapper.go::MapData` → `internal/service/schema_adapter.go::BuildUpsertSQLInSchema`.
- Verify branch: route `auto_src_29` có dùng `BatchBuffer` hay đường nào khác? Có gọi `MappedData["_gpay_source_id"] = pkValue` không?
- Nếu miss → port logic B11 sang generator này → unit test.

### Option 2 (workaround) — Backfill rows + reset cursor
```sql
-- 1. Backfill _gpay_source_id cho 8 rows orphan
UPDATE shadow_src_local_pg_source.orders_addtest
   SET _gpay_source_id = id
 WHERE _gpay_source_id IS NULL OR _gpay_source_id = '';

-- 2. Bump _synced_at để transmuter rescan
UPDATE shadow_src_local_pg_source.orders_addtest
   SET _synced_at = NOW()
 WHERE _gpay_source_id <> '';

-- 3. Wait 60s → verify master count
```
⚠️ Workaround — **không** fix root cause. Lần sau ingest tiếp sẽ lại miss.

### Option 3 — Archive nếu không cần Track E
Set `is_active=false` cho source `addtest_pg_orders` + disable schedule.

---

## 4. Defer reasoning

Phase E scope chốt 5 task: G5/G4/G7/G2/G8. Bug G4 thực ra là **code-level fix** trong `dynamic_mapper.go` hoặc generator V2 — blast radius lớn hơn dự kiến (>30 phút Muscle, có thể kéo theo các path khác đang hoạt động).

→ Phase E giữ E4 ở mức **diagnostic + report**. Fix code đề xuất tách thành **Phase F2** sau khi Phase E hoàn thành.

---

## 5. Skills used

Bash (psql, docker exec), Read (lessons), Write (file này).

**Lessons applied**:
- L-v1-v2-anchor-key-port — pattern khớp 100%, dùng làm root cause classifier.
- L-three-layer-trust — diagnose 5 layer riêng (binding → schedule → stats → data → DDL) thay vì nhảy thẳng "transmuter bug".
- L-real-data-test — query shadow data thực, không tin claim "schedule success = pipeline OK".
