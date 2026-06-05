# Report — 1 source → multi shadow target (V1 UNIQUE relax)

**Phase**: fe-api-worker-action-tracer-2026-05-18 / 1source_multi_shadow
**Author**: Claude Code (Muscle, claude-opus-4-7)
**Date**: 2026-05-19

## Vấn đề

```
failed to register table: ERROR: duplicate key value violates unique
constraint "cdc_table_registry_source_db_source_table_key"
(SQLSTATE 23505)
```

User muốn 1 source → N shadow target. FE đã hỗ trợ; backend V2 model
(`source_object_registry` + `shadow_binding`) đã 1→N. Block đến từ V1
legacy `cdc_table_registry` với UNIQUE 2-cột.

## Root cause

Migration `001_init_schema.sql` tạo:
```sql
UNIQUE (source_db, source_table)
```
tại `cdc_system.cdc_table_registry`. Constraint này restriction quá tay
— V1 chỉ là legacy bridge, V2 mới là authoritative routing.

## Fix (schema-only, không đổi Go code)

### File mới
- `cdc-cms-service/migrations/schema/core/053_relax_table_registry_unique.sql`
  - DROP `cdc_table_registry_source_db_source_table_key`
  - ADD `cdc_table_registry_source_db_source_table_target_key` UNIQUE
    (source_db, source_table, target_table)
  - Comment + backout block.

### Lý do KHÔNG đổi Go code

| Layer | Vì sao đã 1→N tolerant |
|---|---|
| V2 source_object_registry | `ON CONFLICT (normalized_source_key) DO UPDATE` — 1 row/source by design |
| V2 shadow_binding | `ON CONFLICT (source_object_id, shadow_connection_id, shadow_schema, shadow_table)` — N rows/source |
| CMS RegisterRegistryCommand | Chỉ V1 INSERT bị chặn — migration unblocks |
| CMS V2 syncer (`source_object_v2_sync.go`) | ON CONFLICT idempotent |
| CMS bootstrap mirror (`registry_mirror.go`) | ON CONFLICT idempotent |
| Worker sourceCache | First-wins defensively (`if _, exists := ... !exists`) |
| Worker targetCache/idCache/routeBySourceID | Keyed by target/id — full 1→N precision |

## Audit kết quả (Caller-Resolver Wiring Verification)

Đã enumerate tất cả write sites V1 + V2 + downstream reads:

| File:line | Path | Status |
|---|---|---|
| `cdc-cms-service/.../register_registry.go:88` | V1 INSERT `tx.Create(&entry)` | Unblocked ✓ |
| `cdc-cms-service/.../bulk_register_registry.go` | V1 bulk INSERT | Unblocked ✓ |
| `cdc-cms-service/.../update_registry.go:110` | V1 UPDATE Where id=? | Không đụng UNIQUE ✓ |
| `cdc-cms-service/.../source_object_v2_sync.go:132,193` | V2 ON CONFLICT cả 2 inserts | 1→N OK ✓ |
| `cdc-cms-service/.../registry_mirror.go:122,149` | Bootstrap V2 ON CONFLICT | 1→N OK ✓ |
| `cds/.../metadata_registry_service.go:192-195` | sourceCache first-wins | 1→N OK ✓ |
| `cds/.../metadata_registry_service.go:187-189` | targetCache/idCache keyed by target/id | 1→N OK ✓ |

## Verify (kết quả thực tế)

| Bước | Action | Kết quả |
|---|---|---|
| Migration syntax | `psql -c` parse | Pending user apply |
| V2 idempotency audit | Read source_object_v2_sync.go | Confirmed ON CONFLICT đúng |
| Bootstrap mirror audit | Read registry_mirror.go | Confirmed ON CONFLICT đúng |
| Worker sourceCache audit | Read metadata_registry_service.go | First-wins defensively — OK |
| CMS RegisterRegistryCommand audit | Read register_registry.go | tx.Create blocked by V1 UNIQUE only |

## Hành động user cần làm

1. **Apply migration**:
   ```bash
   psql "$CDC_SYSTEM_DB_URL" \
     -f cdc-cms-service/migrations/schema/core/053_relax_table_registry_unique.sql
   ```
   Hoặc qua migration runner của CMS service nếu có.

2. **Verify constraint mới**:
   ```sql
   SELECT conname, pg_get_constraintdef(oid)
   FROM pg_constraint
   WHERE conrelid = 'cdc_system.cdc_table_registry'::regclass
     AND contype = 'u';
   ```
   Expected output: chỉ còn UNIQUE 3-cột.

3. **Retry register cùng source + target khác từ FE TableRegistry**:
   - `goopay.export-jobs` → `sd_export_jobs_main` (đã có)
   - `goopay.export-jobs` → `sd_export_jobs_analytics` (mới)
   - Expected: 202 accepted, không còn SQLSTATE 23505.

4. **Verify DB consistency**:
   ```sql
   SELECT id, source_db, source_table, target_table
   FROM cdc_system.cdc_table_registry
   WHERE source_db='goopay' AND source_table='export-jobs';
   ```
   Expected: 2 rows.

   ```sql
   SELECT so.object_code, COUNT(sb.id) AS bindings
   FROM cdc_system.source_object_registry so
   JOIN cdc_system.shadow_binding sb ON sb.source_object_id = so.id
   WHERE so.source_database='goopay' AND so.source_object_name='export-jobs'
   GROUP BY so.object_code;
   ```
   Expected: bindings >= 2.

## Out of scope

- Drop V1 table (V2 cutover là phase riêng).
- FE multi-target UX (đã ngầm hỗ trợ — user nhập target_table khác).
- Worker code thay đổi (sourceCache first-wins đã graceful; route precision đi qua targetCache).

## Risk + Mitigation

| Risk | Mitigation |
|---|---|
| Existing duplicate `(source_db, source_table, target_table)` rows | UNIQUE cũ block từ tier-1 → không thể có dup. ADD constraint sẽ thành công |
| Worker startup load sourceCache map nhiều rows về 1 target | First-wins logic xử lý gracefully; warning log nếu cần debug |
| Cấp quyền migration runner | User confirm DB admin có quyền ALTER TABLE |

## Skills sử dụng

- Bash + grep → enumerate V1 mirror writes, V2 sync points, worker reads
- Read tool → audit source_object_v2_sync.go, registry_mirror.go, register_registry.go, metadata_registry_service.go
- Write tool → tạo migration 053 + workspace docs
- TaskUpdate/TaskCreate → 3 task tracking (#49/#50/#51)
- Workspace governance → Full Doc Set (01/02/08/09/report) + APPEND 05_progress.md + global lesson

## Files changed checklist (Pre-flight Governance)

- [x] `cdc-cms-service/migrations/schema/core/053_relax_table_registry_unique.sql` (NEW)
- [x] Workspace docs: `01_requirements_1source_multi_shadow.md`, `02_plan_1source_multi_shadow.md`, `08_tasks_1source_multi_shadow.md`, `09_tasks_solution_1source_multi_shadow.md`
- [x] `05_progress.md` APPEND
- [x] `report_1source_multi_shadow.md` (this file)
- [x] Global lesson APPEND (`agent/memory/global/lessons.md`)
