# Solution — 1 source → multi shadow target

**Phase**: 1source_multi_shadow
**Date**: 2026-05-19

## Diagnosis

```
SQLSTATE 23505
duplicate key value violates unique constraint
"cdc_table_registry_source_db_source_table_key"
```

Source: migration 001 tạo `cdc_table_registry` với
```sql
UNIQUE (source_db, source_table)
```
Block insert thứ 2 với cùng `(source_db, source_table)` dù `target_table`
khác.

## Layer audit

| Layer | Concern | Status |
|---|---|---|
| V1 legacy table | UNIQUE quá rộng | **BUG** — fix migration 053 |
| V2 source_object_registry | UNIQUE (normalized_source_key) | OK — 1 row/source by design |
| V2 shadow_binding | UNIQUE (source_object_id, shadow_connection_id, shadow_schema, shadow_table) | OK — N rows/source |
| CMS RegisterRegistryCommand | INSERT V1 → conflict | Unblocked sau migration |
| CMS V2 syncer | ON CONFLICT cả 2 inserts | OK |
| CMS Bootstrap mirror | ON CONFLICT cả 2 inserts | OK |
| Worker sourceCache | First-wins (`if !exists`) | OK — degrade graceful |
| Worker targetCache/idCache | Keyed by target/id | OK — full precision |

## Fix

### Migration `cdc-cms-service/migrations/schema/core/053_relax_table_registry_unique.sql`

```sql
BEGIN;
ALTER TABLE cdc_system.cdc_table_registry
  DROP CONSTRAINT IF EXISTS cdc_table_registry_source_db_source_table_key;
ALTER TABLE cdc_system.cdc_table_registry
  ADD CONSTRAINT cdc_table_registry_source_db_source_table_target_key
  UNIQUE (source_db, source_table, target_table);
COMMENT ON CONSTRAINT cdc_table_registry_source_db_source_table_target_key
  ON cdc_system.cdc_table_registry IS
  'Replaces UNIQUE(source_db, source_table) from migration 001. Allows '
  '1 source → multi target_table while still blocking duplicate '
  'same-source-same-target. Authoritative routing lives in V2 '
  'source_object_registry + shadow_binding.';
COMMIT;
```

Backout block (in comments) — DROP 3-col, re-ADD 2-col sau khi de-dup.

## Why not modify Go code

- V2 model **đã** 1→N tolerant (đó là design intent).
- V1 mirror chỉ là legacy bridge — fix tại schema level, code không cần đổi.
- Worker `sourceCache` defensively first-wins → không crash với multiple V1 rows.
- Caller precision dùng `targetCache`/`idCache`/`targetRouteMap` (target-keyed).

## User actions (verify)

1. Apply migration:
   ```bash
   psql "$CDC_SYSTEM_DB_URL" -f cdc-cms-service/migrations/schema/core/053_relax_table_registry_unique.sql
   ```
   (hoặc qua migration runner của CMS service nếu có).

2. Verify constraint mới:
   ```sql
   SELECT conname, pg_get_constraintdef(oid)
   FROM pg_constraint
   WHERE conrelid = 'cdc_system.cdc_table_registry'::regclass
     AND contype = 'u';
   ```
   Expected: chỉ thấy `cdc_table_registry_source_db_source_table_target_key`
   với `(source_db, source_table, target_table)`.

3. Retry register từ FE TableRegistry:
   - Source: `goopay.export-jobs` → Target: `sd_export_jobs_v2` (mới).
   - Expected: 202 accepted.
   - Verify DB:
     ```sql
     SELECT id, source_db, source_table, target_table
     FROM cdc_system.cdc_table_registry
     WHERE source_db='goopay' AND source_table='export-jobs';
     ```
     Expected: 2 rows (target_table cũ + mới).

4. Verify V2 cả 2 mapping tồn tại:
   ```sql
   SELECT so.object_code, sb.shadow_table, sb.binding_code
   FROM cdc_system.source_object_registry so
   JOIN cdc_system.shadow_binding sb ON sb.source_object_id = so.id
   WHERE so.source_database='goopay' AND so.source_object_name='export-jobs';
   ```
   Expected: 1 source_object, 2 shadow_binding rows.
