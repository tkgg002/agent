# Task Solution — B11 _gpay_source_id ingest population
**Date**: 2026-05-04
**Author**: Brain (Antigravity)
**Owner (execute)**: Muscle (CC CLI)
**Type**: Code fix (permanent) + already applied DATA backfill

---

## 1. Vấn đề
Shadow ingest path (Debezium → cdc-worker → shadow_*) KHÔNG populate `_gpay_source_id`. Master `orders_fact` UNIQUE constraint trên `_gpay_source_id` → mọi row có `_gpay_source_id=''` collide → ON CONFLICT DO UPDATE giữ 1 row, các row khác bị overwrite/lost.

Verified evidence:
```sql
-- Trước backfill: shadow rows 56-63 đều _gpay_source_id NULL/empty
SELECT id, _gpay_source_id FROM shadow_goopay_source.orders;
 id | _gpay_source_id 
----+-----------------
 51 | 51    -- pre-V2 path đã set
 ...
 56 |       -- V2 fan-out path KHÔNG set
 ...
```

## 2. Root cause (3-layer trace)
| Layer | File | Vấn đề |
|-------|------|--------|
| A. Event handler | `internal/handler/event_handler.go:113-126` | Build `UpsertRecord` với `MappedData=mapped.Columns` — không đưa `_gpay_source_id` vào |
| B. Batch buffer | `internal/handler/batch_buffer.go:158-162` | Truyền `MappedData` raw vào `BuildUpsertSQLInSchema` |
| C. Schema adapter | `internal/service/schema_adapter.go:404-520` | Loop `mappedData` + meta cols (`_raw_data, _source, _synced_at, ...`) nhưng **không có branch cho `_gpay_source_id`** |

Code path tham chiếu (sinkworker đường Mongo):
- `internal/sinkworker/sinkworker.go:146` — đường Mongo có `record["_gpay_source_id"] = sourceID` đúng
- Đường Debezium thiếu equivalent

## 3. Fix proposal (permanent code fix)

### Option A — Schema adapter (RECOMMENDED, minimal)
**File**: `internal/service/schema_adapter.go`
**Function**: `BuildUpsertSQLInSchema` (line 404-520)
**Change**: Sau block ghi `_hash` (line 452), thêm branch:
```go
// V2 anchor key — chỉ ghi nếu shadow/master có cột _gpay_source_id.
// Source PK value đảm bảo distinct → master ON CONFLICT (_gpay_source_id) chạy đúng.
if _, ok := schema.Columns["_gpay_source_id"]; ok {
    allCols = append(allCols, `"_gpay_source_id"`)
    allPlaceholders = append(allPlaceholders, "?")
    finalValues = append(finalValues, fmt.Sprintf("%v", pkValue))
}
```
+ thêm vào ON CONFLICT UPDATE branch (line 487):
```go
if _, ok := schema.Columns["_gpay_source_id"]; ok {
    updateSets = append(updateSets, `"_gpay_source_id" = EXCLUDED."_gpay_source_id"`)
}
```

**Trade-off chấp nhận**: pkValue cast `interface{}` → string qua `%v`. Đảm bảo TEXT cột nhận được giá trị xác định, không empty.

### Option B — Event handler (less surgical)
Thêm `_gpay_source_id` vào `mapped.Columns` ở `event_handler.go` ngay trước Add. **Nhược điểm**: phải lặp ở cả `processEvent` lẫn `handleDelete`, dễ miss.

→ Chọn **Option A**.

## 4. DoD (Definition of Done)
- `go build ./...` PASS
- `go test ./internal/service/... ./internal/handler/...` PASS (đặc biệt schema_adapter_test, batch_buffer_test)
- Thêm 1 unit test `TestBuildUpsertSQL_PopulatesGpaySourceID`:
  - Schema có cột `_gpay_source_id`, pkValue=`"42"` → SQL chứa `"_gpay_source_id"` trong INSERT cols + EXCLUDED trong UPDATE
  - Schema không có `_gpay_source_id` → SQL không chứa cột đó (backward compat V1 tables)
- Live smoke (sau rebuild + restart):
  1. INSERT 1 row source mới
  2. Query shadow: `_gpay_source_id` của row đó = source PK value (không NULL/empty)
  3. Wait 1 cron tick → master `dw_orders.orders_fact` count tăng 1 với `_gpay_source_id` = source PK

## 5. DATA backfill (đã apply bởi Brain — 2026-05-04 04:30)
```sql
-- Đã chạy ở gpay-postgres-cdc / cdc_dw:
UPDATE shadow_goopay_source.orders 
   SET _gpay_source_id = id 
 WHERE _gpay_source_id IS NULL OR _gpay_source_id = '';
-- UPDATE 8 (rows 56-63)

-- Cleanup master residue:
DELETE FROM dw_orders.orders_fact WHERE _gpay_source_id = '';
-- DELETE 1
```

Sau backfill:
- shadow rows 51-63 đều có `_gpay_source_id = id` distinct
- master `orders_fact` 33 rows, 33 distinct `_gpay_source_id`

## 6. Idempotency note
- Sau code fix, ingest mới cho row đã backfill sẽ UPDATE `_gpay_source_id = pkValue` (cùng giá trị) → no-op delta. An toàn.
- Backfill SQL chạy lại lần 2 = WHERE filter rỗng = 0 rows updated. Idempotent.

## 7. Risks
- **Risk**: Shadow tables LEGACY V1 (rows ingested trước fix) có `_gpay_source_id` đã bằng id rồi. Code fix sẽ ghi đè cùng giá trị → an toàn.
- **Risk**: Một số shadow tables KHÔNG có column `_gpay_source_id` (V1 only). Branch `if _, ok := schema.Columns[...]` đã guard.

## 8. Verification commands (cho Muscle khi xong)
```bash
# 1. Build + test
go build ./...
go test ./internal/service/... ./internal/handler/... -count=1

# 2. Rebuild + restart
docker compose up -d --build cdc-worker

# 3. Insert source row
docker exec gpay-postgres-source psql -U src_user -d goopay_source -c \
  "INSERT INTO public.orders (user_id, amount, status, notes) VALUES (8888, 77.77, 'pending', 'b11-permanent-fix-smoke') RETURNING id;"

# 4. Verify shadow populated (without manual backfill)
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT id, _gpay_source_id FROM shadow_goopay_source.orders ORDER BY id::int DESC LIMIT 1;"
# Expect: id=N, _gpay_source_id='N' (NOT NULL/empty)

# 5. Wait 60s + check master
sleep 65 && docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest -c \
  "SELECT count(*), count(DISTINCT _gpay_source_id) FROM dw_orders.orders_fact;"
# Expect: count = 34 (was 33), distinct = 34
```

## 9. Files to modify (Muscle)
| File | Lines | Action |
|------|-------|--------|
| `internal/service/schema_adapter.go` | 452 (after `_hash` block) | INSERT — add `_gpay_source_id` to allCols/Placeholders/Values |
| `internal/service/schema_adapter.go` | 487 (after `_hash` UPDATE block) | INSERT — add `_gpay_source_id` to updateSets |
| `internal/service/schema_adapter_test.go` | NEW test | Add `TestBuildUpsertSQL_PopulatesGpaySourceID` |

## 10. Skills/Agents required
- Go code editing (Muscle direct)
- `go build ./...`
- `go test ./internal/...`
- Docker compose rebuild
- Live smoke testing với psql
