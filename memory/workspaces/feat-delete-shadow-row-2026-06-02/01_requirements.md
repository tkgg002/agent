# 01_requirements — Delete /shadow Row (Level C)

## Mục tiêu
User mở `/shadow` (TableRegistry), bấm "Xoá" trên 1 row → backend xoá sạch metadata + DROP table vật lý → user có thể re-register lại không bị duplicate key.

## Scope IN
1. **BE endpoint** `DELETE /api/v1/source-objects/:id`
   - Auth: JWT + RequireOpsAdmin.
   - Middleware: Idempotency (Redis TTL 1h) + Audit (admin_actions row với `reason ≥ 10 chars`).
   - Body: `{ "reason": "<≥10 chars>" }`.
   - Header: `Idempotency-Key`.
   - Transaction PG cascade:
     - SELECT các `shadow_binding` để biết `shadow_schema.shadow_table` cần DROP.
     - Manual cleanup legacy: `cdc_reconciliation_report`, `cdc_worker_schedule`, `cdc_mapping_rules` (filter theo source_db+source_table+shadow_table).
     - `DELETE FROM cdc_system.source_object_registry WHERE id = $1` → FK ON DELETE CASCADE kéo `shadow_binding`, `mapping_rule_v2`, `master_binding`, `sync_runtime_state`.
     - DDL DROP TABLE (best-effort, log warn nếu fail): mỗi `shadow_schema."<shadow_table>"`.
     - Optional DROP SCHEMA nếu rỗng (skip default → user tự dọn nếu cần).
   - Return: 202 `{ "status": "deleted", "id": N, "dropped_tables": [...], "skipped_drops": [...] }`.
2. **FE button + modal** trong `TableRegistry.tsx`:
   - Cột "Action" thêm nút `Delete` (icon `DeleteOutlined`, danger).
   - Dùng `ConfirmDestructiveModal` (đã có) — body: cảnh báo "Xoá vĩnh viễn metadata + DROP TABLE shadow vật lý. Có thể re-register lại sau."
   - Form: Input lý do ≥10 ký tự.
   - Gọi `cmsApi.delete('/api/v1/source-objects/:id', { data: { reason }, headers: { 'Idempotency-Key': ... }})`.
   - `humanizeApiError` cho error path (đã có).

## Scope OUT (best-effort, không tự xoá)
- **Không** xoá Kafka topic / reset Debezium offset. Worker sẽ replay events vào "table không tồn tại" và log warning — user re-register sẽ tự apply DDL lại.
- **Không** drop `shadow_<db>` schema (chứa table khác có thể vẫn dùng).
- **Không** xoá connection_registry row (1 connection nhiều source_object).
- **Không** xoá master_table vật lý — `master_binding` cascade chỉ xoá row binding, table master vật lý giữ nguyên.

## Acceptance
- A1: Click "Xoá" + nhập reason ≥10 → 1 source_object biến mất khỏi list `/shadow`.
- A2: Sau khi xoá, register lại CÙNG `(connection, source_db, source_table)` thành công (không 23505).
- A3: Postgres shadow schema không còn table tương ứng (`\dt shadow_<db>.<table>` rỗng).
- A4: `cdc_reconciliation_report`, `cdc_worker_schedule`, `cdc_mapping_rules` không còn row liên quan.
- A5: Audit log có 1 row `delete_source_object` với reason và actor.
- A6: Worker không crash sau xoá (log warning, tự skip event).
- A7: tsc EXIT=0, go build EXIT=0.

## Anti-acceptance (sẽ KHÔNG đạt)
- N1: KHÔNG rollback nếu user lỡ tay (DROP TABLE không reversible — đó là intent mức C).
- N2: KHÔNG dọn Kafka offset (out-of-scope).

## Risk
- R1: DROP TABLE trong transaction PG cùng với DELETE row — DDL implicit-commit. Mitigation: DROP sau khi DELETE row commit; nếu DROP fail vẫn trả 202 + log skipped_drops.
- R2: 2 source_object share `shadow_table` (multi-binding) — chỉ DROP nếu binding khác không còn ref. Mitigation: trước DROP, query `SELECT count(*) FROM shadow_binding WHERE shadow_schema=? AND shadow_table=?`. Nếu count > 0 (binding khác còn ref) → skip.
- R3: Worker đang ghi vào shadow_table khi DROP → 1 transaction worker fail. Acceptable (replay).
- R4: User dùng nhầm xoá row đang active → mitigation: modal warning + reason gate.
