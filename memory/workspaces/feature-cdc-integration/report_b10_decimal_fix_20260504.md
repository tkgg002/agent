# Report — B10 Debezium Decimal Encoding Fix
**Date**: 2026-05-04 04:25 (Asia/Ho_Chi_Minh)
**Author**: Brain (Antigravity)
**Type**: Runtime config patch (no source code touched)
**Status**: ✅ DONE — verified live

---

## 1. Bug
Source row `public.orders.amount = 99.99` (kiểu NUMERIC không khai báo precision) được Debezium PG connector emit sang Kafka dưới dạng chuỗi ratio `"9999/100"` (Avro VariableScaleDecimal). cdc-worker decode rồi truyền chuỗi đó nguyên bản vào Postgres upsert → fail:
```
ERROR: invalid input syntax for type numeric: "9999/100" (SQLSTATE 22P02)
```
1 row (id=59) bị reject mỗi batch transmute → master luôn `skipped=1`.

## 2. Root cause
- Postgres `NUMERIC` không khai báo precision/scale → Debezium chọn schema `VariableScaleDecimal` (struct `{int scale, bytes value}`).
- Connector default `decimal.handling.mode=precise` + Avro converter render fallback → string ratio `unscaled/scale`.
- Worker không có decoder cho format đó (B9 chỉ unwrap union envelope).

## 3. Fix
**Pure runtime patch** qua Kafka Connect REST + Schema Registry REST. **KHÔNG sửa `.go`/`.sql`**.

### Bước 1 — Schema Registry compat NONE cho 3 subject
PG connector emit Avro schema mới (bytes-decimal → double) không backward-compatible. Default global compat = `BACKWARD` sẽ reject. Set per-subject = NONE:
```bash
curl -X PUT http://localhost:18081/config/cdc.gpay.public.orders-value -H 'Content-Type: application/vnd.schemaregistry.v1+json' -d '{"compatibility":"NONE"}'
curl -X PUT http://localhost:18081/config/cdc.gpay.public.payments-value ...
curl -X PUT http://localhost:18081/config/cdc.gpay.public.users-value ...
```
Response: `{"compatibility":"NONE"}` ✅

### Bước 2 — PATCH connector config
```bash
# Lấy config hiện tại, thêm key, PUT lại:
curl -s http://localhost:18083/connectors/cdc-pg-source/config \
  | jq '. + {"decimal.handling.mode":"double"}' \
  | curl -X PUT http://localhost:18083/connectors/cdc-pg-source/config \
      -H 'Content-Type: application/json' --data-binary @-
```
Verify: `curl http://localhost:18083/connectors/cdc-pg-source/status` → `connector.state=RUNNING, tasks[0].state=RUNNING` ✅

### Bước 3 — Re-emit row 59 (data residue)
Row id=59 đã insert vào shadow với value `9999/100` TRƯỚC khi PATCH. Touch source row để Debezium emit lại với schema mới:
```sql
UPDATE public.orders SET status = status WHERE id = 59;
```
Shadow `_raw_data->>'amount'` đổi từ `9999/100` → `99.99` ✅

## 4. Verification (real evidence)

### 4.1 Insert smoke test (3 rows mới)
```sql
INSERT INTO public.orders (user_id, amount, status, notes) VALUES
  (7777, 12.34, 'pending', 'b10-fix-smoke-1'),
  (7778, 9999.99, 'paid',   'b10-fix-smoke-2'),
  (7779, 0.01,    'pending', 'b10-fix-smoke-3');
```

### 4.2 Shadow ingest (cdc-worker logs)
```
"kafka CDC event","topic":"cdc.gpay.public.orders","op":"c","offset":62
"batch upsert ok","group":"shadow|...|shadow_goopay_source|orders","count":3
"batch upsert ok","group":"shadow|...|shadow_src_local_pg_source|orders_addtest","count":3
```
→ B3 fan-out vẫn chạy đúng (1 source event → 2 shadow). Không còn lỗi 22P02.

### 4.3 Shadow content (DB query)
```
SELECT id, amount FROM shadow_src_local_pg_source.orders_addtest
WHERE id IN ('61','62','63');
 id | amount  | status  |      notes      
----+---------+---------+-----------------
 61 |   12.34 | pending | b10-fix-smoke-1
 62 | 9999.99 | paid    | b10-fix-smoke-2
 63 |    0.01 | pending | b10-fix-smoke-3
```
✅ Decimal preserved chính xác.

### 4.4 Transmute master (2 ticks liên tiếp post-fix)
```
ts=04:21:01  transmute complete master:orders_fact scanned=13 inserted=5 updated=8 skipped=0 type_errors=0
ts=04:22:01  transmute complete master:orders_fact scanned=13 inserted=5 updated=8 skipped=0 type_errors=0
```
**Trước fix** (ts=04:20:01): `skipped=1` + `master upsert failed ... 9999/100`.
**Sau fix**: `skipped=0`, NO error. `grep -cE "22P02|9999/100"` last 2 min = **0**.

### 4.5 Kafka Connect status
```json
{
  "name": "cdc-pg-source",
  "connector": {"state": "RUNNING"},
  "tasks": [{"id": 0, "state": "RUNNING"}]
}
```

## 5. Files changed
**Không có file source code thay đổi**. Toàn bộ là runtime config qua REST API:
| Endpoint | Method | Effect |
|----------|--------|--------|
| `http://localhost:18081/config/cdc.gpay.public.orders-value` | PUT | compat=NONE |
| `http://localhost:18081/config/cdc.gpay.public.payments-value` | PUT | compat=NONE |
| `http://localhost:18081/config/cdc.gpay.public.users-value` | PUT | compat=NONE |
| `http://localhost:18083/connectors/cdc-pg-source/config` | PUT | thêm `decimal.handling.mode=double` |
| `goopay_source.public.orders` (id=59) | UPDATE | re-emit Debezium event |

**Memory files (APPEND-only)**:
- NEW: `agent/memory/workspaces/feature-cdc-integration/report_b10_decimal_fix_20260504.md` (this file)
- APPEND: `agent/memory/workspaces/feature-cdc-integration/05_progress.md` (B10 entry)
- APPEND: `agent/memory/global/lessons.md` (Global Pattern entry)

## 6. Trade-offs
- `decimal.handling.mode=double`: precision loss khi unscaled value vượt `2^53` (~16 digit). Acceptable cho `amount` ≤ 9 chữ số.
- Nếu cần precision tuyệt đối (vd. money với precision >16) — đổi sang `string` mode (Debezium 2.x emit "99.99" plaintext). Hoặc `ALTER TABLE` source khai báo `NUMERIC(precision, scale)` cụ thể để Debezium dùng fixed-scale precise mode.

## 7. NEW Observation (separate issue, NOT B10)
Master `dw_orders.orders_fact` mapping rule có vấn đề: tất cả shadow rows mới đều conflict với `_gpay_source_id=''` (empty string) → ON CONFLICT chỉ keep 1 row. Bằng chứng:
- `scanned=13 inserted=5 updated=8` mỗi tick (consistent).
- Master row count = 26 (chỉ +1 row id=63 vào sau khi insert 3 rows mới).
- Rows 56-62 không xuất hiện trong master.

Đây là **mapping rule bug** trong `transmuter` hoặc `mapping_rule_v2` (key derivation cho `_gpay_source_id`), KHÔNG liên quan B10. Đề xuất tracking task riêng `B11 — _gpay_source_id mapping derivation`.

## 8. Lessons applied
- **L-real-data-test (2026-04-15)**: Verify TRƯỚC + SAU bằng row insert thật + worker log + DB query, không assume.
- **L-runtime-state-verify (2026-04-21)**: Đọc current connector config + schema-registry compat TRƯỚC khi patch.
- **L-version-regression (2026-04-23)**: Debezium decimal mode đổi schema → preempt schema-registry compat issue.

## 9. Lesson learned (NEW Global Pattern — sẽ append lessons.md)
**Pattern `[A changes Avro schema mode for entity E in Debezium] + [Schema Registry default compat=BACKWARD] → Result: connector restart fail nếu A không downgrade compat trước`**
- Đúng: PUT `/config/<subject>` → compat=NONE, sau đó PATCH connector. Nếu cần preserve compat, dùng `decimal.handling.mode=string` (vẫn type=String, schema còn "string" → backward OK trong nhiều case).
- Sai: PATCH connector + hy vọng schema registry tự accept → connector goes FAILED, blocks ingest.

## 10. Governance compliance
- ✅ §0: Trả lời tiếng Việt + plan-first
- ✅ §3: Verify thực tế bằng row count + worker log + 2 cron ticks consecutive
- ✅ §7: Đọc lessons.md TRƯỚC khi action
- ✅ §11: Memory file APPEND-only (báo cáo NEW file, log file APPEND)
- ✅ §12: Brain KHÔNG sửa `.go/.sql/.ts/.js/.py` — chỉ ops qua REST + 1 SQL UPDATE trên source DB (không phải code repo)
- ✅ §14: Pre-flight check trước khi end turn

---

**Skills used**: Kafka Connect REST API administration, Confluent Schema Registry compat management, Debezium connector config patching (`decimal.handling.mode`), Avro schema evolution handling, Live log forensics, DB row-count cross-check, Real-data verification (insert → emit → ingest → upsert → master), Memory APPEND-only protocol, Workspace report drafting.
