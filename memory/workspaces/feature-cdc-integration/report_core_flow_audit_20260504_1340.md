# Core-Flow Audit (read-only) — Source → Kafka → Worker → Shadow → Master

**Ngày**: 2026-05-04 13:40 (+07)
**Scope**: User refocus — "đừng quan tâm dữ liệu hiện tại, chỉ xem xét core flow chạy đúng sai. vì chúng ta đang build hệ thống. mọi dữ liệu hiện tại đều là demo".
**Loại**: Audit code-path. KHÔNG sửa code (tôi là Brain — CLAUDE.md §12).
**Trigger ngữ cảnh**: /loop "verify V2 bridge end-to-end after cron tick" — cron tick 13:33+00 (UTC = 06:33+00 stamp ở DB) tất cả 6 schedule `last_status='success'`, master/shadow/transmute close-loop OK ở runtime hiện tại.

---

## TL;DR

- Cron tick + master upsert: **OK** ở runtime hiện tại (6/6 schedule success).
- B11 `_gpay_source_id` fix verify-live OK (shadow id=64 → master 34/34 distinct).
- **Đã phát hiện 6 gap kiến trúc** ảnh hưởng "build hệ thống" — không phải bug runtime, mà giới hạn capability cho runtime-registration / multi-engine / delete completeness. Severity ranked.

---

## Gap matrix

| # | Severity | File:Line | Issue | Tác động |
|---|----------|-----------|-------|----------|
| G1 | **HIGH** | `internal/handler/kafka_consumer.go:110-151` | `discoverTopics` chỉ chạy initial + retry rồi `kafka.NewReader` chốt `GroupTopics` cố định | Topic mới (PUT include list runtime) **KHÔNG** được pick up cho đến khi worker restart. A2 `payment_bills_addtest` đã trúng case này. |
| G2 | MED | `internal/handler/event_handler.go:186-198` `extractSourceAndTable` | Parse `subject` cứng theo `cdc.{prefix}.{db}.{table}` → 4-segment | Postgres Debezium 5-segment (`cdc.goopay.{db}.{schema}.{table}`) hoặc service-layer prefix Mongo (`cdc.goopay.payment-bill-service.{coll}`) → `parts[2]` không phải DB thật, mà là service-name. Hiện tại routeCache lookup vẫn match được vì index theo `source_object_name` đơn lẻ, nhưng lookup `db|name` bị lệch — fragile khi 2 collection cùng tên ở 2 service khác nhau. |
| G3 | MED | `internal/handler/event_handler.go:145-184` `handleDelete` | Delete UPDATE chỉ stamp `_deleted=TRUE, _updated_at=NOW()` — **KHÔNG** set `_gpay_source_id` | B11 INSERT/UPDATE branch fix `_gpay_source_id` ở `schema_adapter.go::BuildUpsertSQLInSchema`. DELETE đi đường khác (raw SQL), không qua adapter → thiếu anchor key cho row delete-first (chưa từng INSERT). |
| G4 | MED | `internal/handler/event_handler.go:164-166` `handleDelete` | Hard-rename `pkField="_id"` → `pgPKField="id"` bằng `if` | INSERT branch dùng pkField as-is (registry quyết định). DELETE inconsistent — nếu binding chọn shadow column name `"_id"` thì DELETE WHERE `id=?` sẽ no-op silent. |
| G5 | LOW | `internal/service/metadata_registry_service.go:269-307` `ResolveSourceRoutes` | Fan-out `cloneRoutes[masterID]` map giả định 1 source → 1 master + N clones | Không xử lý case 2+ first-class masters cùng đọc 1 source. Hiện tại không có use-case đó nhưng cần document. |
| G6 | **HIGH** | (architectural) admin endpoint không có | PUT Debezium include list KHÔNG trigger registry validate / FK check | Admin tạo orphan topic dễ dàng (đã verify với A2 payment_bills_addtest). Không có symmetric API "register source + provision topic + validate" → operator phải làm 3 bước thủ công, sai 1 → silent stuck. |

---

## Detailed findings

### G1 — Topic discovery không refresh runtime

**Code**:
```go
// kafka_consumer.go:110
topics, err := kc.discoverTopics(ctx)
// retry once if err
// retry on tick if 0 topics
// then:
reader := kafka.NewReader(kafka.ReaderConfig{
    GroupTopics: topics,  // ← cố định, không re-set
    ...
})
```

**Tác động cụ thể (verify-live)**: A2 attempt 12:45 → PUT Mongo include list thêm `payment_bills_addtest` → Kafka topic `cdc.goopay.payment-bill-service.payment_bills_addtest` có 6 messages → shadow=0 vì worker đã chốt subscription từ startup. Buộc phải rollback A2 (`report_phase_c_cleanup_20260504_1300.md`).

**Đề xuất (cho architect quyết)**:
1. **Option A**: Periodic re-discover (e.g., 60s ticker) → so sánh topics list → nếu khác → close+rebuild reader. Đơn giản nhưng tạo connection churn.
2. **Option B**: Use kafka-go's regex topic matching (`GroupTopics` không support, nhưng `Topic + ConsumerGroup` ở `kafka.Conn` level có). Cần migrate khỏi `Reader{GroupTopics}` API.
3. **Option C**: Admin endpoint POST /v2/topics/refresh → trigger reload trong worker qua NATS signal. Đỡ churn nhưng không hoàn toàn dynamic.
4. **Option D (recommended)**: Kết hợp B (regex subscribe) + admin trigger NATS broadcast để force rebalance khi PUT include list thành công.

### G2 — `extractSourceAndTable` parse cứng 4-segment

**Code**:
```go
// event_handler.go:186
func extractSourceAndTable(subject, source string) (string, string) {
    parts := strings.Split(subject, ".")
    if len(parts) >= 4 {
        return parts[2], parts[3]   // db = parts[2], table = parts[3]
    }
    ...
}
```

**Verify-live observation**:
- Postgres topic chuẩn: `cdc.goopay_source.public.orders` → parts[2]="public", parts[3]="orders" → db="public" (sai, đúng phải "goopay_source").
- Mongo topic chuẩn: `cdc.goopay.payment-bill-service.payment_bills` → parts[2]="payment-bill-service", parts[3]="payment_bills" → db="payment-bill-service" (sai, đúng phải "goopay").

**Tại sao runtime vẫn work?** `MetadataRegistryService.buildSourceLookupKeys` (lines 355-375) index source_object_name ĐƠN LẺ ở key đầu tiên (`[sourceTable]`), nên lookup chỉ cần `sourceTable` match — `sourceDB` chỉ làm fallback key (`db|name`). Đến khi 2 source cùng tên `orders` ở 2 db khác nhau cùng được register, runtime sẽ collide silently.

**Đề xuất**: Parse theo SourceType (Postgres 5-seg, Mongo 4-seg, MariaDB 4-seg) + lấy DB từ event.Source struct (Debezium luôn populate `source.db`/`source.connector` block).

### G3 — handleDelete không stamp `_gpay_source_id`

**Code**:
```go
// event_handler.go:175
sql := fmt.Sprintf(`UPDATE %s SET _deleted = TRUE, _updated_at = NOW() WHERE %s = ?`,
    qualifiedShadowTable(route), quoteEventIdent(pgPKField))
```

**Tác động**: Nếu source publish DELETE cho row chưa từng INSERT vào shadow (rare nhưng valid trong replay scenario), shadow KHÔNG có row → UPDATE no-op silent → master DW không biết delete. Master query `WHERE _deleted=TRUE` không thấy gì.

**Đề xuất**: Đổi sang INSERT ON CONFLICT:
```sql
INSERT INTO shadow.tbl (id, _gpay_source_id, _deleted, _updated_at)
VALUES (?, ?::text, TRUE, NOW())
ON CONFLICT (id) DO UPDATE SET _deleted=TRUE, _updated_at=NOW()
```
Parallel với INSERT branch — tombstone-first write semantics.

### G4 — `_id` → `id` hard-rename ở DELETE branch

**Code**:
```go
// event_handler.go:164
if pkField == "_id" {
    pgPKField = "id"
}
```

**Tác động**: INSERT branch (line 111) dùng `pgPKField := pkField` (as-is). DELETE branch override → asymmetry. Nếu binding chọn shadow column name `"_id"` (Mongo native), DELETE WHERE `id=?` không trúng index.

**Đề xuất**: Xóa cái `if` này. Để registry/binding quyết column name. Nếu cần backward-compat cho legacy V1 `_id`→`id`, đặt vào layer normalize sớm hơn.

### G5 — Fan-out ngầm giả định 1:1:N

`cloneRoutes[masterID]` là `map[int64][]*ResolvedSourceRoute`. Mỗi source chỉ trỏ về 1 masterID qua `logical_clone_of`. Không cản trở runtime hiện tại nhưng nếu sau này có "multi-master tap" (2 master read cùng 1 source) phải refactor schema (`logical_clone_of` thành mảng / bảng nối).

### G6 — Không có endpoint "register source + provision topic"

**Hiện trạng**: 3 bước rời (cdc team kinh nghiệm B-series fix nhiều lần):
1. INSERT registry rows (source_object_registry + shadow_binding + master_binding + transmute_schedule).
2. PUT Debezium connector include list.
3. Pre-empt Schema Registry per-subject compat = NONE.

Sai 1 → silent stuck (không validate, không reconcile). A2 case `payment_bills_addtest` đã trúng combo G1+G6: bước 2 thành công, bước 1 lệch (registry id=31 trỏ collection cũ), worker restart-required (G1) → 0 row.

**Đề xuất**: Wrapper admin POST /v2/sources/register nhận `{source_engine, locator, target_master, mapping_rules?}`, làm cả 3 bước transactionally + verify-loop (5s wait → check 1 message landed) → trả về status.

---

## Verification methodology

- **Read-only audit** — chỉ đọc code + query DB + curl Connect API. Không edit code (CLAUDE.md §12).
- **Live evidence** đính theo finding khi cần (G1 dùng A2 attempt; G2 derive từ topic listing; G3-G5 derive từ code path).
- **Cross-check**: cron tick `last_status='success'` xác nhận hot-path (PG source → Mongo source `payment_bills` cũ → master) chạy đúng. Các gap ở trên là cold-path / runtime-add scenarios.

---

## Đề xuất phân ưu tiên

| Priority | Gap | Lý do |
|----------|-----|-------|
| P0 (next sprint) | G1 + G6 | Trực tiếp block "build hệ thống" — runtime registration là core capability, không thể restart-and-pray. |
| P1 | G3 | Tombstone correctness — nhỏ nhưng silent corruption nếu trúng. |
| P2 | G2 | Documentation + parse-by-engine refactor — không cấp bách nhưng code-smell rõ. |
| P3 | G4 | Cleanup. |
| P4 | G5 | YAGNI — chỉ cần khi multi-master use-case xuất hiện. |

---

## Status

- File này: NEW (read-only audit, không có code change).
- Không append `05_progress.md` vì đây là audit chứ không phải task complete (CLAUDE.md §11 — append cho "đã làm", chứ không phải "đã đọc").
  - Sẽ append khi user duyệt → giao Muscle thực thi P0 fixes.
- Ngoài scope: các P plan ở `/Users/trainguyen/.claude/plans/curried-waddling-spindle.md` (Track D Hardening P2/P3/P4) thuộc workspace `feature-multi-pg-isolation-e2e` — KHÔNG cùng workspace.

## Next step (chờ user duyệt)

**Câu hỏi cho user**:
1. Confirm scope ưu tiên: P0 (G1+G6) trước? hay G3 (correctness) trước?
2. G1 nghiêng option D (regex + admin signal) hay option A (60s poll)? Architect quyết định.
3. G6 design wrapper API ở service nào? (cdc-admin-api? cdc-worker control plane?)
4. Trigger Muscle `/muscle-execute` cho P0 sau khi chốt scope không?

Skills đã dùng: Read, Grep, Bash (psql / docker exec), Write, ScheduleWakeup (sẽ schedule sau).
