# 09_tasks_solution — Delete /shadow Row (Solution dossier)

> Nối dài 03_implementation.md — focus vào edge case, branching logic, và verify checklist mà 03 chưa diễn giải.

## S1 — Command handler edge cases (T1)

### S1.1 — Source object không tồn tại
- Path: SELECT identity → `src.ID == 0`.
- Return `ErrSourceObjectNotFound` (đã có ở `update_source_object_v2.go:35`).
- HTTP layer map → 404.
- **Lý do**: tránh phá vỡ idempotency — replay sau khi đã xoá nên trả 404 thay vì silent OK.

### S1.2 — Source object có 0 binding (v2_source_only)
- `len(bindings) == 0` → skip TX block "DELETE legacy theo shadow_table list" (guard `if len > 0`).
- Vẫn DELETE `cdc_mapping_rules` theo `source_table` (legacy V1 có thể tồn tại trước khi register binding).
- Vẫn DELETE `source_object_registry` row.
- Loop DROP rỗng → `dropped_tables=[]`, `skipped_drops=[]`.
- Output 202 vẫn hợp lệ.

### S1.3 — Source object có N binding cùng `(shadow_schema, shadow_table)` — multi-binding share
- Sau DELETE registry (CASCADE), `shadow_binding` của source này đã bị xoá.
- Loop DROP: `SELECT count(*) FROM shadow_binding WHERE schema=? AND table=?`.
  - Nếu = 0 → DROP TABLE.
  - Nếu > 0 → binding của source_object KHÁC còn ref → skip + record `skipped_drops` reason `multi_binding_share`.
- **Quan trọng**: query count phải chạy SAU TX commit, nếu không sẽ count cả binding vừa bị xoá.

### S1.4 — Postgres DDL fail (table không tồn tại, lock, permission)
- `DROP TABLE IF EXISTS` đã handle "không tồn tại" → không lỗi.
- Lock contention: worker đang ghi → `DROP` chờ (PG mặc định không timeout). Risk: hang.
  - Mitigation: trước DROP, set `SET LOCAL lock_timeout = '5s'` (nếu cần — optional). Để Phase 4 quyết, mặc định không set.
- Permission denied: log warn, append `skipped_drops` reason `drop_failed: <err>`.
- Operator nhìn `skipped_drops` trong response → tự `psql` DROP thủ công.

### S1.5 — Mongo source — KHÔNG có shadow physical table?
- Mongo cũng được route qua shadow Postgres table (binding ghi schema `shadow_<db>`). Cùng pattern.
- Không có gì khác biệt — `shadow_table` luôn tồn tại nếu DDL đã apply.

### S1.6 — `parent_binding_id` chain (explode binding tree)
- File `071_add_explode_to_shadow_binding.sql` thêm self-FK `parent_binding_id ON DELETE CASCADE`.
- Khi xoá `source_object_id`, CASCADE xoá ROOT binding → CASCADE tiếp child explode binding.
- Loop DROP của ta SELECT TRƯỚC TX → đã capture cả parent + child. Mỗi shadow_table sẽ được drop tự nhiên.

## S2 — HTTP handler edge cases (T3)

### S2.1 — `:id` không phải số
- `strconv.ParseInt` fail → 400 `{error:"invalid_source_object_id"}`.

### S2.2 — `:id` âm hoặc 0
- Sau ParseInt nhưng `id <= 0` → 400. Defense-in-depth (Validate cũng check lại).

### S2.3 — Body empty hoặc malformed JSON
- `BodyParser` error → `_ = err` (ignore). Middleware Audit đã ép buộc body có `reason ≥ 10` TRƯỚC khi đi vào handler → nếu lọt qua đến đây nghĩa là reason valid trong middleware buffer.
- Handler chỉ dùng `req.Reason` để inject vào command audit log entry — middleware cũng đã ghi audit_action row riêng rồi. KHÔNG critical.

### S2.4 — `bus == nil`
- 503 `{error:"command bus not ready"}`. Mirror `system_connectors_handler.go:289`.

### S2.5 — Map error ngoài `ErrSourceObjectNotFound`
- TX fail → 502 `{error:"delete_failed", detail: ...}`.
- FE `humanizeApiError` sẽ Vietnamize SQLSTATE nếu có (e.g. 23503 FK violation → "Bản ghi đang được tham chiếu...").

## S3 — Router mount edge cases (T5)

### S3.1 — Cần dual-mount `/v1` alias?
- DELETE đã ở `/v1/source-objects/:id` (đã `/v1` rồi). KHÔNG cần alias.
- Pattern y hệt block `/v1/system/connectors/:name` (chỉ mount 1 lần).

### S3.2 — Middleware leak?
- Mount thủ công bằng append slice + `apiGroup.Delete(...)` không dùng `Group.Use()` → middleware chỉ apply cho route này. An toàn.

### S3.3 — JWT auth?
- `apiGroup` đã có JWT middleware ở level cao hơn (xem router setup phần đầu). DELETE thừa hưởng.

## S4 — FE edge cases (T6)

### S4.1 — User click "Xoá" trên row Mongo vs PostgreSQL
- Cùng flow. `record.id` là `source_object_registry.id` (BIGSERIAL) đồng nhất.

### S4.2 — User click "Xoá" trên row có binding khác share table
- BE trả 202 + `skipped_drops: [{schema, table, reason:"multi_binding_share"}]`.
- FE message.success vẫn fire (metadata đã xoá). Optional: nếu `skipped_drops.length > 0`, dùng `Modal.warning` show user biết physical table chưa drop. **MVP**: chỉ log via `console.warn`, không hiển thị (giữ minimal). Có thể thêm sau nếu user muốn.

### S4.3 — User click "Xoá" rồi mất mạng giữa chừng
- `cmsApi.delete` timeout → `humanizeApiError` → "Server phản hồi quá lâu".
- Idempotency-Key `delete-source-object-{id}-{Date.now()}` đã giữ slot Redis 1h → retry trong 1h sẽ idempotent (cùng key = same response).
- Trade-off: cùng `id` + retry sau khi reload page → `Date.now()` mới → key mới → nếu lần đầu chưa kịp commit, lần 2 sẽ chạy lại. Acceptable vì BE `ErrSourceObjectNotFound` cho lần 2 (đã xoá).

### S4.4 — Concurrent user xoá cùng 1 id
- 2 request đến cùng lúc → 1 thắng TX → trả 202. 1 thua → BE select ra 0 row → 404.
- FE 2: `humanizeApiError(err, 'Xoá source-object thất bại')` → "Không tìm thấy..." Bad UX nhưng acceptable race window <100ms.

### S4.5 — Modal đóng giữa lúc gửi request
- `ConfirmDestructiveModal` có `disabled={isBusy}` cho cancel button + `mask={{closable: !isBusy}}`. User không thể đóng khi đang `loading=true`. Safe.

### S4.6 — `targetName` rỗng?
- Guard `deletePending ? ... : ''`. Khi modal `open=false`, không render anyway (destroyOnHidden=true).

## S5 — Smoke test scenarios (T7)

| ID | Scenario | Expected |
|----|---------|----------|
| TC1 | Register fresh source → xoá ngay → re-register cùng key | 202 + 200 OK lần 2 |
| TC2 | Register source → tạo binding → drift recon → xoá | recon_report row biến mất, table dropped |
| TC3 | Register source A + B share cùng shadow_table (multi-binding) → xoá A | A metadata xoá, table KHÔNG drop (skipped_drops), B vẫn ghi được |
| TC4 | Xoá id không tồn tại | 404 |
| TC5 | Xoá thiếu Idempotency-Key | 400 (middleware) |
| TC6 | Xoá reason < 10 char | 400 (middleware) |
| TC7 | Xoá khi worker đang snapshot vào table | DROP có thể chậm (5-10s). Worker tx fail, restart, log warn "table not exists" — OK |
| TC8 | Xoá 1 source có 5 explode child binding (parent_binding tree) | Tất cả 5 shadow_table drop hết, không sót row |

## S6 — Rollback strategy (nếu cần revert)

Nếu sau merge bị bug nặng, revert chỉ cần:
1. Revert commit FE — nút "Xoá" biến mất, user không click được.
2. Optional: revert BE — handler vẫn còn nhưng FE không gọi → no-op. Có thể giữ BE cho ops thao tác qua curl.
3. KHÔNG có data destructive nào tự động chạy — chỉ trigger qua user click. An toàn revert.

## S7 — Verification commands (cho T7 report)

```bash
# BE build
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go build ./...

# FE type-check
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-web && npx tsc --noEmit -p tsconfig.app.json

# Smoke: SQL verify post-delete
psql -h localhost -U cdc -d cdc_db -c "SELECT * FROM cdc_system.source_object_registry WHERE id = <TEST_ID>"
psql -h localhost -U cdc -d cdc_db -c "\dt shadow_<db>.<table>"
psql -h localhost -U cdc -d cdc_db -c "SELECT * FROM cdc_system.cdc_reconciliation_report WHERE target_table = '<shadow_table>'"
psql -h localhost -U cdc -d cdc_db -c "SELECT id, action_type, reason, actor FROM cdc_system.admin_actions ORDER BY id DESC LIMIT 1"

# Re-register sanity (A2)
curl -X POST http://localhost:8080/api/v1/source-objects/register \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"source_connection_code":"...", "source_db":"...", "source_table":"...", ...}'
```

## S8 — Lessons để ghi (sau T7 hoặc nếu phát hiện sai pattern)

- Nếu Postgres DDL trong transaction gây surprise (vd. `DROP TABLE` không thấy effect ngay sau COMMIT) → ghi lesson "Postgres DDL pattern with FK CASCADE — separate metadata TX from DDL".
- Nếu phát hiện `cdc_worker_schedule` không có cột `target_table` → ghi lesson + sửa migration filter.
- Nếu user click nhanh 2 lần → FE chưa disable button đủ nhanh → ghi lesson "Disable destructive button on first click, not on modal open".
