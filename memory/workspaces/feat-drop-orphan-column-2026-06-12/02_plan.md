# 02_plan — Drop orphan column cho rule rejected (per-row + drop-all + confirm)

> User approve: rule `status='rejected'` → hiện nút "drop field"; thêm nút "drop all field reject"; bấm → modal xác nhận yes/no (thao tác xoá rủi ro).

## Bối cảnh (đã verify code + data)
- Reject chỉ set `status='rejected'`, KHÔNG drop cột (master DDL ADD-only) → cột "orphan" còn vật lý. (gap_analysis_master_rule_reject_orphan_column.md)
- Master DDL chạy trên **dest 5434** qua `cdc.cmd.master-alter-column` → `HandleMasterAlterColumn` → `ReconcileColumn` (rename/alter_type, CHƯA có drop). `cdc.cmd.alter-column` là SHADOW plane (5436) — KHÔNG dùng.
- Fixture thật: binding 18 (export_jobs_testid1) = 76 rule rejected, **cả 76 cột tồn tại vật lý** (orphan). Unique `(master_binding_id,target_column)` → cột rejected ≠ cột approved.

## Giải pháp (reuse plane master, minimal-impact)
**Worker** (`centralized-data-service`):
1. `master_ddl_generator.go`: thêm `DropColumn(ctx, masterName, column)` — mirror `ReconcileColumn` (loadBinding→GetMasterDB→quoteDDLQualified→tx lock/timeout→`ALTER TABLE … DROP COLUMN IF EXISTS`→cacheInvalidator). **Guard: từ chối cột `_*` meta**.
2. `master_ddl_handler.go`: payload `masterAlterColumnRequest` +`Action`; `HandleMasterAlterColumn`: `action=="drop"` → `gen.DropColumn`, else giữ nguyên ReconcileColumn. Reply ok/error về reply_to.

**CMS** (`cdc-cms-service`):
3. `master_mapping_rule_handler.go`: `publishMasterDropColumn` (mirror publishMasterReconcile, action=drop, chờ reply 60s).
4. Handler `DropColumn` — `POST /v1/master-mapping-rules/:id/drop-column`. **Guards server-side**: rule phải `status='rejected'`; cột không `_*`; KHÔNG có rule approved+active dùng cùng target_column.
5. Handler `DropRejectedColumns` — `POST /v1/master-mapping-rules/drop-rejected-columns?binding_id=X`: lấy cột của mọi rule rejected (loại `_*`, loại cột approved dùng), drop từng cột, trả {dropped, failed, errors}.
6. `router.go`: registerDestructive 2 route trên (static `drop-rejected-columns` đặt cạnh `:id` — Fiber ưu tiên static).

**FE** (`cdc-cms-web/MasterMappingFieldsPage.tsx`):
7. Action column: `record.status==='rejected'` → nút "Drop field" (danger) → `Modal.confirm` yes/no → POST `/:id/drop-column`.
8. Toolbar: nút "Drop all rejected" (danger) → `Modal.confirm` (cảnh báo N cột) → POST `/drop-rejected-columns?binding_id`.
9. Sau drop → `fetchRules()` (in_master cột đã drop → false/biến mất).

## An toàn (G4)
- Server guard 3 lớp: status='rejected' + không `_*` + không bị approved dùng. Worker guard `_*`. `DROP COLUMN IF EXISTS` idempotent. Confirm modal FE. Request-reply (biết drop thành công thật).

## Verify (red→green) — fixture binding 18
- Chọn 1 rule rejected (vd id 1073 fileUrl) → drop-column → cột `fileUrl` BIẾN MẤT khỏi `master_centrallized_export_service.export_jobs_testid1` (information_schema). Re-drop → ok idempotent (IF EXISTS).
- `go build` (CMS+worker), `tsc` FE; deploy + trigger thật + đối soát cột vật lý before/after.

## File thay đổi (dự kiến)
- worker: master_ddl_generator.go (+DropColumn), master_ddl_handler.go (+Action+drop branch)
- CMS: master_mapping_rule_handler.go (+publishMasterDropColumn +2 handler), router.go (+2 route)
- FE: MasterMappingFieldsPage.tsx (2 nút + confirm)
