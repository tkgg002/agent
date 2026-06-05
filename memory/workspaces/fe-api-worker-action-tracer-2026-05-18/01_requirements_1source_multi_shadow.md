# Requirements — 1 source → multi shadow target (relax V1 UNIQUE)

**Phase**: fe-api-worker-action-tracer-2026-05-18 / 1source_multi_shadow
**Author**: Claude Code (Muscle, claude-opus-4-7)
**Date**: 2026-05-19

## User report (verbatim)

```
failed to register table: ERROR: duplicate key value violates unique
constraint "cdc_table_registry_source_db_source_table_key"
(SQLSTATE 23505)
mở khoá vụ này. 1 source -> multi shadow
```

Trigger: FE TableRegistry → `Register` cùng cặp `(source_db, source_table)`
nhưng `target_table` mới — 502/409 từ CMS API.

## Mục tiêu

Cho phép cùng 1 source object (e.g. `goopay.export-jobs`) được register
vào N target table khác nhau:
- `goopay.export-jobs` → `sd_export_jobs_main`
- `goopay.export-jobs` → `sd_export_jobs_analytics`
- `goopay.export-jobs` → `sd_export_jobs_archive`
...

Vẫn block cặp `(source_db, source_table, target_table)` duplicate
(data integrity — đó là double-register cùng route).

## Constraint

- KHÔNG đổi V2 schema (`source_object_registry` + `shadow_binding` đã 1→N).
- KHÔNG đổi FE/CMS API contract.
- KHÔNG đổi worker code (chỉ schema-level fix).
- Migration phải có backout block + comment lý do.

## Root cause

`cdc_table_registry` (V1 legacy) tạo từ migration 001 với:
```sql
UNIQUE (source_db, source_table)
```
Block insert thứ 2 cùng source → khác target.

V2 model đã đúng:
- `source_object_registry.normalized_source_key` UNIQUE — 1 hàng/source.
- `shadow_binding (source_object_id, shadow_connection_id, shadow_schema, shadow_table)` UNIQUE — N hàng/source, 1 hàng/target.

V1 mirror chỉ là legacy bridge cho worker chưa migrate hết V2 reads;
constraint cũ là restriction quá tay.

## Out of scope

- Drop V1 table hoàn toàn (V2 cutover là phase riêng).
- Đổi worker `sourceCache` (đã first-wins defensively — OK cho 1→N).
- FE multi-target UX (đã hỗ trợ ngầm — user nhập target_table khác).

## Definition of Done

- [x] Migration 053 tạo (DROP cũ + ADD mới UNIQUE 3-cột).
- [x] Audit V1 INSERT path: `RegisterRegistryCommand.Handle` → `tx.Create(&entry)`.
- [x] Audit V2 sync path 1→N tolerant (ON CONFLICT đã đúng).
- [x] Audit bootstrap mirror (`registry_mirror.go`) — ON CONFLICT đã đúng.
- [x] Audit downstream reads worker (`sourceCache` first-wins, `targetCache` keyed by target — OK).
- [ ] User chạy migration 053 trên DB cdc_dw.
- [ ] User retry register cùng source + target khác → expect 202 success.
