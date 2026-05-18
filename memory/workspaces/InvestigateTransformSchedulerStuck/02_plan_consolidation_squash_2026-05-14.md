# Plan — Migration Consolidation (Squash)

**Date**: 2026-05-14
**Goal**: 54 file SQL → ~25 file (gom theo bảng/chức năng, bỏ ALTER thừa).
**Constraint**: KHÔNG break cdc_dw prod (53 tracker rows hiện có). KHÔNG cheat config.

## Strategy

### Backward compat: idempotent + tracker-key by basename
- Tracker `cdc_system.schema_migrations` lưu version = filename (basename, không .sql).
- File giữ tên gốc khi anchor (file CREATE TABLE chính) → cdc_dw tracker row khớp → SKIP.
- File bị DELETE → tracker row của nó vẫn còn trong cdc_dw nhưng inert (không có file matching → runtime bỏ qua).
- Modified anchor file: content thay đổi nhưng version basename giữ nguyên → cdc_dw `applied[version]=true` → SKIP (không re-run).
- Fresh DB (cdc_cms_database): anchor file chạy 1 lần, content cuối cùng tạo ra final schema.

### Idempotency layer (defense in depth)
- Mọi anchor file dùng `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, `CREATE OR REPLACE FUNCTION`, `ADD COLUMN IF NOT EXISTS`.
- Trigger: `DROP TRIGGER IF EXISTS ... CREATE TRIGGER ...`.
- Constraints: wrap trong DO block check `pg_constraint`.
- Seed: `INSERT ... ON CONFLICT DO NOTHING`.

## Target Layout (25 file, từ 50)

### Files DELETE (pure ALTER/redefine/transition — merged hoặc dead)
| # | File | Lý do | Merge target |
|---|---|---|---|
| 1 | `core/002_standardize_schema.sql` | Function redefine | core/001 |
| 2 | `core/044_cleanup_public_residue.sql` | Transition cleanup, no-op trên fresh DB | — (drop) |
| 3 | `ids/003_sonyflake_schema.sql` | Function redefine | core/001 |
| 4 | `partitioning/004_partitioning.sql` | V1 partitioning, dead (replaced by 010) | — (drop) |
| 5 | `registry/009_source_ts.sql` | Data table loop ALTER (template handles) | — (drop) |
| 6 | `registry/014_sensitive_fields.sql` | ALTER ADD COLUMN | registry/013 |
| 7 | `registry/016_table_registry_timestamp_field.sql` | ALTER ADD COLUMN | registry/013 |
| 8 | `registry/017_timestamp_detection.sql` | ALTER ADD COLUMN | registry/013 + recon_dlq/008 |
| 9 | `registry/024_shadow_is_active.sql` | ALTER ADD COLUMN | registry/019 |
| 10 | `recon_dlq/012_dlq_state_machine.sql` | ALTER on V1 failed_sync_logs (dead) | partitioning/010 |
| 11 | `recon_dlq/045_dlq_columns_in_cdc_system.sql` | ALTER on cdc_system.failed_sync_logs | partitioning/010 |
| 12 | `worker/053_fix_tz_worker_schedule.sql` | TZ fix ALTER | worker/007 |
| 13 | `audit_security/006_activity_log.sql` | V1 public.cdc_activity_log (replaced 010) | — (drop) |
| 14 | `audit_security/026_master_rls_helper.sql` | Function moved to cdc_system (038) | — (drop, 038 keep) |
| 15 | `cdc_system_model/028_sonyflake_fallback_fn.sql` | Functions duplicate of 038 | — (drop) |
| 16 | `ops/015_slow_sql_indexes.sql` | CREATE INDEX trên V1 dead tables | — (drop) |
| 17 | `ops/021_airbyte_deprecation_comments.sql` | COMMENT thuần | — (drop) |
| 18 | `ops/043_normalize_shadow_binding_schema.sql` | Data UPDATE | — (drop) |
| 19 | `ops/046_model_drift_patches.sql` | ALTER ADD COLUMN | registry/013 + registry/020 |
| 20 | `ops/047_source_provisioning_state.sql` | ALTER ADD COLUMN | cdc_system_model/030 |
| 21 | `ops/049_mariadb_seed_legacy_orders.sql` | All comment-out, dead | — (drop) |
| 22 | `ops/050_logical_clone_locator_keys.sql` | Specific row UPDATE | — (drop) |
| 23 | `ops/051_prune_legacy_v1.sql` | Specific row UPDATE | — (drop) |
| 24 | `core/037_move_system_tables_to_cdc_system.sql` | Transition (cdc_dw applied; fresh = no-op) | — (drop, safe) |

**Total DELETE**: 24 files

### Files KEEP (modified to absorb merged content) — anchor files
| # | File | Modifications |
|---|---|---|
| 1 | `core/001_init_schema.sql` | Absorb 002 (column_exists, standardize_cdc_table, updated create_cdc_table) + 003 (BIGINT PK + per-table SEQUENCE). KHÔNG absorb registry ALTERs (giữ tách bạch). |
| 2 | `core/038_finalize_cdc_system_namespace.sql` | Keep (sequences + functions in cdc_system). Drop the `DROP cdc_internal` part — chỉ keep CREATE. |
| 3 | `ids/018_sonyflake_v125_foundation.sql` | Keep as-is (creates cdc_internal — 038 moves to cdc_system; idempotent on fresh DB) |
| 4 | `partitioning/010_partitioning.sql` | Absorb 012 + 045 (next_retry_at, last_error cols on failed_sync_logs) + 015 (slow_sql indexes) |
| 5 | `registry/013_table_registry_expected_fields.sql` | Absorb 014 + 016 (col only, not UPDATE seed) + 017 + 046 (cdc_table_registry portion) |
| 6 | `registry/019_system_registry.sql` | Absorb 024 (is_active col) |
| 7 | `registry/020_mapping_rule_jsonpath.sql` | Absorb 046 (cdc_mapping_rules portion: rule_type col) |
| 8 | `registry/023_master_table_registry.sql` | Keep |
| 9 | `registry/025_schema_proposal.sql` | Keep |
| 10 | `registry/027_systematic_sources.sql` | Keep |
| 11 | `worker/007_worker_schedule.sql` | Rewrite: TIMESTAMPTZ from the start (absorb 053) |
| 12 | `worker/022_transmute_schedule.sql` | Keep |
| 13 | `recon_dlq/008_reconciliation.sql` | Absorb 017 portion (error_code col, source_count NULL) |
| 14 | `recon_dlq/011_recon_runs.sql` | Keep |
| 15 | `audit_security/040_admin_actions_in_cdc_system.sql` | Keep |
| 16 | `audit_security/041_cdc_alerts_in_cdc_system.sql` | Keep |
| 17 | `cdc_system_model/029_v2_connection_registry.sql` | Keep |
| 18 | `cdc_system_model/030_v2_source_object_registry.sql` | Absorb 047 (provisioning state cols + idx) |
| 19 | `cdc_system_model/031_v2_shadow_binding.sql` | Keep |
| 20 | `cdc_system_model/032_v2_master_binding.sql` | Keep |
| 21 | `cdc_system_model/033_v2_mapping_rule.sql` | Keep |
| 22 | `cdc_system_model/034_v2_sync_runtime_state.sql` | Keep |
| 23 | `cdc_system_model/035_v2_backfill_legacy_registry.sql` | Keep (3 infra seed rows) |
| 24 | `cdc_system_model/036_v2_transmute_schedule.sql` | Keep |
| 25 | `ops/048_provisioning_log_cap_helper.sql` | Keep (function) |
| 26 | `ops/052_create_cdc_jobs.sql` | Keep |

**Total KEEP**: 26 files (đáng ra 25, nhưng 037 còn dùng để move 7 bảng. Let me re-check — phía trên đã liệt 037 trong DELETE.)

**Recount**: 50 - 24 = 26 files. Sau khi rename "v2" → "cdc_system_model" và move file đã làm xong. Net consolidation: **50 → 26 files**.

### Files NOT in embed (cluster bootstrap, DBA-only)
- `cluster/001_roles.sql` (KEEP)
- `cluster/002_search_path.sql` (KEEP)

## Verification

### Test 1 — cdc_dw (53 legacy tracker rows + full schema)
Expectation:
- Tracker rows cho 24 file DELETE: tồn tại nhưng inert (file không có) → bỏ qua
- Tracker rows cho 26 file KEEP: tồn tại → `applied[version]=true` → SKIP toàn bộ
- `migrations done total_files=26 applied_now=0 already_applied=26`
- /health=200 /ready=200
- Schema không thay đổi

### Test 2 — cdc_cms_database fresh
- DROP database; CREATE OWNER cdc-cms-user
- Start CMS với env override
- Expectation: `applied_now=26`, tất cả 26 file chạy, schema final đầy đủ (cdc_table_registry có sensitive_fields/timestamp_field/source_url/sync_status/...; failed_sync_logs có next_retry_at/last_error)
- /health=200 /ready=200
- COUNT(*) cdc_system.schema_migrations = 26

### Test 3 — schema diff
- `pg_dump -s cdc_dw` vs `pg_dump -s cdc_cms_database` (sau test 2)
- Expectation: zero functional drift (chỉ khác: cdc_dw có 1 số legacy artifact như public.* orphans nếu có)

## Skills used
- Layer separation (L1/L2/L3 đã có từ refactor trước)
- Tracker-stable rename (basename unchanged → backward compat)
- Idempotent SQL patterns (IF NOT EXISTS, OR REPLACE)
- Content-grounded audit (Agent đọc body từng file, không suy từ tên)
- Squash with anchor preservation (giữ file CREATE TABLE, drop ALTER thừa)
