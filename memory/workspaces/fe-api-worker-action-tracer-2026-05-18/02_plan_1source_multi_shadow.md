# Plan — 1 source → multi shadow target

**Phase**: 1source_multi_shadow
**Date**: 2026-05-19

## Strategy

Schema-only fix tại V1 legacy table. Bao trùm bằng migration 053 (CMS
service quản lý migration), KHÔNG đụng Go code (worker + CMS đã 1→N
tolerant).

## Steps

1. **Migration 053**: DROP `cdc_table_registry_source_db_source_table_key`,
   ADD `cdc_table_registry_source_db_source_table_target_key` UNIQUE
   (source_db, source_table, target_table). Comment + backout block.

2. **Verify V2 idempotency** (`source_object_v2_sync.go`):
   - `source_object_registry` ON CONFLICT (normalized_source_key) — 1 row/source. OK.
   - `shadow_binding` ON CONFLICT (source_object_id, shadow_connection_id, shadow_schema, shadow_table) — N rows/source. OK.

3. **Verify CMS API paths**:
   - `RegisterRegistryCommand.Handle` (`register_registry.go:88`): `tx.Create(&entry)` — chỉ bị V1 UNIQUE chặn. Migration unblocks. ✓
   - `BulkRegisterRegistryCommand`: cùng pattern, bulk INSERT — unblocked. ✓
   - `UpdateRegistryCommand` (`update_registry.go:110-113`): Where id=? — không đụng UNIQUE. ✓
   - `bootstrap.SyncLegacyToV2Bootstrap` (`registry_mirror.go`): chỉ đọc V1 + ghi V2 (ON CONFLICT). ✓

4. **Verify worker (centralized-data-service) reads**:
   - `MetadataRegistryService.sourceCache` (`metadata_registry_service.go:192-195`): defensive first-wins (`if _, exists := ... !exists`). Multiple V1 rows / target không crash. ✓
   - `targetCache` + `idCache` keyed by target_table/id — fully 1→N safe. ✓
   - `routeCache` cũng first-wins. ✓
   - `GetTableConfigBySource(sourceTable)` returns 1 ptr — degrades to first match. Acceptable: callers fall back to target-keyed lookup khi cần precision.

5. **User apply migration + verify end-to-end**:
   - `psql ... -f migrations/schema/core/053_relax_table_registry_unique.sql`.
   - Click Register cùng source + target khác từ FE → expect 202.

## Files changed

| File | Action | Lý do |
|---|---|---|
| `cdc-cms-service/migrations/schema/core/053_relax_table_registry_unique.sql` | NEW | DROP + ADD constraint |
| Workspace `01/02/08/09/report_1source_multi_shadow.md` | NEW | Governance Full Doc Set |
| `agent/memory/workspaces/fe-api-worker-action-tracer-2026-05-18/05_progress.md` | APPEND | Audit log |
| `agent/memory/global/lessons.md` | APPEND | Global pattern |

## Risk + Mitigation

| Risk | Mitigation |
|---|---|
| Existing rows duplicate `(source_db, source_table, target_table)` | `ADD CONSTRAINT` sẽ fail nếu dup tồn tại. User phải de-dup trước. Hiện tại chưa có vì UNIQUE cũ block ngay từ tier-1 → an toàn |
| Migration runner không chạy lại | Backout block đã document — manual SQL apply OK |
| Worker `sourceCache` first-wins chọn target "không mong muốn" | sourceCache là helper lookup; route precision dùng targetCache/idCache. Caller `recon_handler.go:689` đã pass `targetTable` (target precision đảm bảo) |

## Verification gates

- [x] V1 INSERT path audit (RegisterRegistryCommand)
- [x] V2 sync path audit (SyncFromLegacyTx)
- [x] Bootstrap mirror audit (SyncLegacyToV2Bootstrap)
- [x] Worker reads audit (MetadataRegistryService)
- [ ] User apply migration
- [ ] User verify retry register success
