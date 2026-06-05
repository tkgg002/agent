-- =====================================================================
-- Migration: RENAME _gpay_source_id -> source_id, _gpay_deleted -> _deleted
-- Date     : 2026-05-28
-- Workspace: cleanup-gpay-cols-2026-05-28
-- Target   : Shadow tables + Master tables across all shadow_* and master_*
--            schemas in centralized-data-service PG cluster.
-- =====================================================================
-- Apply order (see 09_tasks_solution_cleanup.md "Deploy order"):
--   1) Drain sinkworker + snapshot runner (no in-flight upsert).
--   2) Apply THIS migration.
--   3) Deploy code in centralized-data-service + cdc-cms-service + cdc-cms-web.
--   4) Restart workers (schema_manager.go + master_ddl_generator.go will
--      self-heal indexes via CREATE UNIQUE INDEX IF NOT EXISTS).
--   5) Smoke verify (\d shadow.<t> + grep zero-residue).
--
-- Idempotent: safe to run multiple times.
-- Reversibility: reverse migration in migration_rename_gpay_cols_reverse.sql.
-- =====================================================================

BEGIN;

DO $$
DECLARE
    r RECORD;
BEGIN
    -----------------------------------------------------------------
    -- 1) RENAME _gpay_source_id -> source_id
    -----------------------------------------------------------------
    FOR r IN
        SELECT table_schema, table_name
          FROM information_schema.columns
         WHERE column_name = '_gpay_source_id'
           AND table_schema LIKE ANY (ARRAY['shadow%', 'master%', 'dw_%', 'cdc_%'])
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
             WHERE table_schema = r.table_schema
               AND table_name  = r.table_name
               AND column_name = 'source_id'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I.%I RENAME COLUMN _gpay_source_id TO source_id',
                r.table_schema, r.table_name);
            RAISE NOTICE 'RENAMED %.% _gpay_source_id -> source_id', r.table_schema, r.table_name;
        ELSE
            -- source_id đã tồn tại → drop _gpay_source_id (duplicate).
            EXECUTE format(
                'ALTER TABLE %I.%I DROP COLUMN _gpay_source_id',
                r.table_schema, r.table_name);
            RAISE NOTICE 'DROPPED duplicate %.%._gpay_source_id (source_id existed)', r.table_schema, r.table_name;
        END IF;
    END LOOP;

    -----------------------------------------------------------------
    -- 2) RENAME _gpay_deleted -> _deleted
    -----------------------------------------------------------------
    FOR r IN
        SELECT table_schema, table_name
          FROM information_schema.columns
         WHERE column_name = '_gpay_deleted'
           AND table_schema LIKE ANY (ARRAY['shadow%', 'master%', 'dw_%', 'cdc_%'])
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
             WHERE table_schema = r.table_schema
               AND table_name  = r.table_name
               AND column_name = '_deleted'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I.%I RENAME COLUMN _gpay_deleted TO _deleted',
                r.table_schema, r.table_name);
            RAISE NOTICE 'RENAMED %.% _gpay_deleted -> _deleted', r.table_schema, r.table_name;
        ELSE
            -- Path B (handler-created tables yesterday) đã có cả 2 → drop _gpay_deleted.
            EXECUTE format(
                'ALTER TABLE %I.%I DROP COLUMN _gpay_deleted',
                r.table_schema, r.table_name);
            RAISE NOTICE 'DROPPED duplicate %.%._gpay_deleted (_deleted existed)', r.table_schema, r.table_name;
        END IF;
    END LOOP;

    -----------------------------------------------------------------
    -- 3) DROP partial UNIQUE indexes referencing legacy column names.
    --    Code (sinkworker/schema_manager.go + master_ddl_generator.go)
    --    sẽ tự CREATE UNIQUE INDEX IF NOT EXISTS với target cột mới khi
    --    service restart.
    -----------------------------------------------------------------
    FOR r IN
        SELECT schemaname, indexname
          FROM pg_indexes
         WHERE indexdef ILIKE '%_gpay_source_id%'
            OR indexdef ILIKE '%_gpay_deleted%'
    LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I.%I', r.schemaname, r.indexname);
        RAISE NOTICE 'DROPPED stale index %.%', r.schemaname, r.indexname;
    END LOOP;

    -----------------------------------------------------------------
    -- 4) DROP named UNIQUE constraint cũ ở Path B (handler-created):
    --    uq_<table>_gpay_source_id. Code command_handler.go sẽ ADD
    --    constraint mới với tên uq_<table>_source_id ở lần create-default-
    --    columns kế tiếp.
    -----------------------------------------------------------------
    FOR r IN
        SELECT n.nspname AS schema_name,
               c.relname AS table_name,
               con.conname AS constraint_name
          FROM pg_constraint con
          JOIN pg_class c ON c.oid = con.conrelid
          JOIN pg_namespace n ON n.oid = c.relnamespace
         WHERE con.conname LIKE 'uq_%_gpay_source_id'
    LOOP
        EXECUTE format(
            'ALTER TABLE %I.%I DROP CONSTRAINT IF EXISTS %I',
            r.schema_name, r.table_name, r.constraint_name);
        RAISE NOTICE 'DROPPED constraint %.%.%', r.schema_name, r.table_name, r.constraint_name;
    END LOOP;
END $$;

COMMIT;

-- =====================================================================
-- Post-apply verify (run AFTER service restart so code recreates indexes):
-- =====================================================================
-- SELECT table_schema, table_name, column_name
--   FROM information_schema.columns
--  WHERE column_name IN ('_gpay_source_id', '_gpay_deleted');
-- Expect: 0 rows.
--
-- SELECT indexname, indexdef FROM pg_indexes
--  WHERE indexdef ILIKE '%source_id%' AND schemaname LIKE 'shadow%' OR schemaname LIKE 'master%';
-- Expect: partial UNIQUE index ON (source_id) WHERE NOT _deleted.
-- =====================================================================
