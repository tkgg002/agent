# 02_plan — Delete /shadow Row (Mức C)

> Plan-only. KHÔNG đụng source code cho tới khi user duyệt.
> Tham chiếu: `01_requirements.md`, `00_context.md`.

## Nguyên tắc cốt lõi
- §6 Simplicity First + minimal impact: tái sử dụng `destructiveChain` (JWT → OpsAdmin → Idempotency → Audit), `ConfirmDestructiveModal`, `humanizeApiError`. Không thêm middleware mới.
- §12 Brain Code Prohibition: tài liệu mô tả chi tiết Go/TS, Muscle mới viết code.
- Theo pattern CQRS hiện có: `commands.DeleteXHandler` + thin handler trong `api/` + mount thủ công ở `router.go` (vì `registerDestructive` chỉ wrap POST).
- DDL (DROP TABLE) **best-effort sau khi metadata commit** — không đặt DROP trong cùng TX để tránh rollback metadata khi DDL fail (R1).

## Roadmap (5 phase tuần tự)

### Phase 0 — Audit dependency (đã làm 1 phần)
- ✅ Xác nhận FK cascade từ `source_object_registry`:
  - `shadow_binding (source_object_id) ON DELETE CASCADE`
  - `master_binding (source_object_id) ON DELETE CASCADE`
  - `mapping_rule_v2 (source_object_id) ON DELETE CASCADE`
  - `sync_runtime_state (source_object_id) ON DELETE CASCADE`
- ✅ FK con cấp 2 từ `shadow_binding`:
  - `mapping_rule_v2.shadow_binding_id ON DELETE CASCADE` (file 067)
  - `sync_runtime_state.shadow_binding_id ON DELETE CASCADE` (file 034)
  - `master_binding.shadow_binding_id ON DELETE SET NULL` (chỉ null, không xoá row master)
  - `shadow_binding.parent_binding_id ON DELETE CASCADE` (self-ref, child explode binding bị kéo theo)
- ✅ Bảng legacy KHÔNG có FK (cần manual cleanup):
  - `cdc_system.cdc_reconciliation_report` — filter `target_table = shadow_table`
  - `cdc_system.cdc_worker_schedule` — filter `target_table = shadow_table` (xác nhận lại schema cột khi implement)
  - `cdc_system.cdc_mapping_rules` — filter `source_table = source_object_name AND COALESCE(target_table, master_table) ...`

### Phase 1 — BE Command + Handler (Muscle)
1. **NEW `internal/app/commands/delete_source_object_v2.go`** (~180 LOC):
   - Struct `DeleteSourceObjectV2Command { ID, DeletedBy, Reason }`.
   - `Validate()` — ID > 0, Reason ≥ 10 (defense-in-depth, mặc dù middleware Audit đã check).
   - Handler `DeleteSourceObjectV2Handler.Handle(ctx, c)`:
     1. SELECT `(source_database, source_object_name, source_connection_id, object_code)` của `source_object_registry.id = $1` → 404 nếu không có.
     2. SELECT mọi `shadow_binding` rows (id, shadow_schema, shadow_table, physical_table_fqn).
     3. **TX#1 (metadata)** — `BEGIN; ... COMMIT`:
        - Manual cleanup legacy (3 bảng) — DELETE filter theo `(source_database, source_object_name, shadow_table)`. Log số row.
        - `DELETE FROM cdc_system.source_object_registry WHERE id = $1` → FK CASCADE tự kéo `shadow_binding`, `mapping_rule_v2`, `master_binding`, `sync_runtime_state`.
     4. **Best-effort DDL (ngoài TX#1)** — với mỗi binding đã collect:
        - Check `SELECT count(*) FROM cdc_system.shadow_binding WHERE shadow_schema=$1 AND shadow_table=$2`. Nếu > 0 (binding khác còn ref) → append vào `skipped_drops` với reason `multi_binding_share`.
        - Ngược lại: `DROP TABLE IF EXISTS "<shadow_schema>"."<shadow_table>"`. Capture lỗi → append `skipped_drops` với reason.
     5. Return JSON `{status, source_object_id, dropped_tables[], skipped_drops[]}`.
2. **Wire vào CQRS bus** (`internal/server/server.go:230` adjacent):
   - `cmdBus.RegisterSync("source.delete-v2", commands.NewDeleteSourceObjectV2Handler(db, logger))`.

### Phase 2 — BE HTTP handler + route (Muscle)
1. **EDIT `internal/api/source_objects_handler.go`** (~+45 LOC):
   - Thêm method `(*SourceObjectsHandler).Delete(c *fiber.Ctx) error`.
   - Parse `:id` → int64; reject ≤ 0.
   - Đọc body `{reason}` — chỉ để FE biết, audit middleware đã đọc trước.
   - Build `commands.DeleteSourceObjectV2Command{ID, DeletedBy: middleware.GetUsername(c), Reason}`.
   - `bus.Execute(ctx, cmd)` qua `messaging.WithMetadata`.
   - Map error:
     - `ErrSourceObjectNotFound` → 404 `{error:"source_object_not_found"}`.
     - Khác → 502 `{error:"delete_failed", detail: err}`.
   - Success → 202 + body JSON từ handler (chứa `dropped_tables`, `skipped_drops`).
2. **EDIT `internal/api/source_objects_handler.go` constructor** — `NewSourceObjectsHandler` thêm tham số `bus messaging.CommandBus` (giống `system_connectors_handler`).
3. **EDIT `internal/server/server.go:267`** — pass `cmdBus` vào `NewSourceObjectsHandler(...)`.
4. **EDIT `internal/router/router.go` (block sau dòng 322)** — mount thủ công vì `registerDestructive` chỉ wrap POST:
   ```go
   // Destructive DELETE — full hard wipe + DROP shadow table (Level C).
   // registerDestructive only wraps POST → mount manually.
   {
       deleteHandlers := append([]fiber.Handler{}, destructiveChain...)
       deleteHandlers = append(deleteHandlers, sourceObjectsHandler.Delete)
       apiGroup.Delete("/v1/source-objects/:id", deleteHandlers...)
   }
   ```

### Phase 3 — FE button + modal (Muscle)
1. **EDIT `cdc-cms-web/src/pages/TableRegistry.tsx`** (~+60 LOC):
   - Import `DeleteOutlined` (đã có `EditOutlined`, thêm vào dòng 3).
   - State mới: `const [deletePending, setDeletePending] = useState<TRegistry | null>(null)`; `const [deleting, setDeleting] = useState(false)`.
   - Hàm `handleDelete = async (reason: string) => { ... cmsApi.delete(...) }` — payload `{ data: { reason }, headers: { 'Idempotency-Key': \`delete-source-object-\${id}-\${Date.now()}\` } }`.
   - Trong block `Thao tác` cell (dòng 856–895): thêm `<Button danger size="small" icon={<DeleteOutlined />} onClick={(e) => { e.stopPropagation(); setDeletePending(record); }}>Xoá</Button>`.
   - Cuối file (trước `</div>` của return): render 1 `ConfirmDestructiveModal` với props:
     - `open={!!deletePending}`, `danger`, `title="Xoá vĩnh viễn shadow object"`,
     - `targetName={deletePending ? `${deletePending.source_db}.${deletePending.source_table}` : ''}`,
     - `description={<>...</>}` — cảnh báo "Sẽ DROP table shadow vật lý + xoá metadata. KHÔNG reversible. User có thể re-register lại."
     - `actionLabel="Xoá vĩnh viễn"`, `loading={deleting}`,
     - `onConfirm={handleDelete}`, `onCancel={() => setDeletePending(null)}`.
   - Sau xoá thành công: `message.success`, `setDeletePending(null)`, `fetchData()` + `fetchShadowBindings()`.
   - Error: `message.error(humanizeApiError(err, 'Xoá thất bại'))`.

### Phase 4 — Verify + report
1. **Build**:
   - BE: `cd data-hub/cdc-cms-service && go build ./...` → EXIT=0.
   - FE: `cd data-hub/cdc-cms-web && npx tsc --noEmit -p tsconfig.app.json` → EXIT=0.
2. **Smoke test** (local docker stack chạy sẵn):
   - Tạo source_object test bằng /api/v1/source-objects/register.
   - Click "Xoá" → nhập reason 10+ ký tự → confirm.
   - Kiểm tra: `SELECT * FROM cdc_system.source_object_registry WHERE id=...` → 0 row.
   - Kiểm tra: `\dt shadow_<db>.<table>` rỗng.
   - Kiểm tra: `SELECT * FROM cdc_system.cdc_reconciliation_report WHERE target_table='<shadow_table>'` → 0 row.
   - Kiểm tra: `SELECT * FROM cdc_system.admin_actions ORDER BY id DESC LIMIT 1` → row mới với `action_type='delete_source_object'`, có reason.
   - Re-register cùng `(connection, source_db, source_table)` → 200 OK, không 23505.
3. **Worker** — quan sát log worker sau xoá 30s: chỉ có WARN "table not exists", không panic/restart. Acceptable per R3.
4. **Tạo `report_2026-06-02_delete-shadow-row.md`** — list từng file thay đổi + LOC delta, kết quả test, các file `skipped_drops` nếu có.

## Phân chia task (chi tiết tại `08_tasks.md`)

| # | Phase | Title | Owner | Phụ thuộc |
|---|-------|-------|-------|-----------|
| T1 | 1 | NEW `delete_source_object_v2.go` | Muscle | Phase 0 |
| T2 | 1 | Wire `cmdBus.RegisterSync("source.delete-v2", ...)` | Muscle | T1 |
| T3 | 2 | EDIT `source_objects_handler.go` (constructor + `Delete`) | Muscle | T2 |
| T4 | 2 | EDIT `server.go` (pass `cmdBus` vào handler) | Muscle | T3 |
| T5 | 2 | EDIT `router.go` (mount thủ công DELETE) | Muscle | T3 |
| T6 | 3 | EDIT `TableRegistry.tsx` (button + modal + handler) | Muscle | T5 |
| T7 | 4 | `go build` + `tsc` + smoke test + viết `report_*.md` | Muscle | T6 |

## Quyết định kỹ thuật (tham chiếu `04_decisions.md` khi cần)

- **D1**: Đặt 1 `DeleteSourceObjectV2Command` (entrypoint duy nhất) thay vì 2 command (delete metadata + drop table) — vì khái niệm "level C" là atomic toàn phần. Nếu sau này tách level B, có thể thêm `DeleteSourceObjectMetadataOnlyCommand` ngang hàng.
- **D2**: DROP TABLE **ngoài** TX#1 — tránh rollback metadata khi DDL fail. Trade-off: nếu DROP fail, metadata đã xoá → operator phải DROP tay (skipped_drops trả về để biết). Đúng với intent "best-effort" trong requirements.
- **D3**: KHÔNG dùng `*pgx.Tx.Exec(DDL)` trong cùng GORM transaction. Dùng `db.WithContext(ctx).Exec(...)` riêng cho từng DROP, sau khi TX commit.
- **D4**: KHÔNG drop schema `shadow_<db>` — chứa nhiều binding khác. User tự dọn nếu muốn.
- **D5**: KHÔNG động Kafka topic / Debezium offset. Re-register sẽ tạo binding mới + worker apply DDL lại từ event stream (worker đã idempotent với "table not exists").

## Risk tracking (xem chi tiết §Risk trong 01_requirements.md)
- R1 DDL implicit-commit → mitigated bởi D2/D3.
- R2 multi-binding share → check count trước DROP, append vào `skipped_drops`.
- R3 worker đang ghi → acceptable, replay-safe.
- R4 user lỡ tay → modal warning + reason ≥ 10 + audit log.

## Out-of-scope (xác nhận lại)
- Soft delete (mức A) — đã có toggle `is_active`, không build thêm.
- Bulk delete — chỉ 1 row/lần để limit blast radius.
- Auto-purge Kafka offset/topic — out-of-scope per requirements.
- Migration schema rename `shadow_<connector>_<db>` → `shadow_<db>` — đã abandon, không liên quan task này.
