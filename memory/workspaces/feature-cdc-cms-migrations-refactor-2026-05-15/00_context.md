# 00 — Context & Scope

> **Workspace**: `feature-cdc-cms-migrations-refactor-2026-05-15`
> **Owner (Muscle)**: Claude Code CLI / claude-opus-4-7
> **Repo**: `data-hub/cdc-cms-service` (Go 1.26.1, Fiber + GORM, control plane CDC)
> **Target dir**: `cdc-cms-service/migrations/`
> **Started**: 2026-05-15

## 1. Bối cảnh

`cdc-cms-service` là control plane của hệ thống CDC (Change Data Capture). Mỗi
khi service boot, runtime migrator `internal/migrate/runner.go` walk
`migrations/` (qua go:embed), so với tracker `cdc_system.schema_migrations` và
apply những file chưa được record. Tracker key = `path.Base(file)` (KHÔNG
gồm subfolder).

Hiện trạng: 28 file `.sql` được embed, chia 9 sub-folder chức năng. 2 file
cluster-level (`cluster/*.sql`) KHÔNG embed, DBA chạy manual. 4 file
`.archive/*.sql` đã đóng băng (không embed).

## 2. Mục tiêu user

> "check lại `cdc-cms-service/migrations`, lên kế hoạch refactor cho nó trực
> quan, chuyên nghiệp. Mục tiêu là gọn gàng, clear, đầy đủ, rõ ràng cho từng
> nhóm chức năng, đáp ứng được chạy trên production với config riêng."

Diễn giải:
- **Gọn gàng**: layout dễ đọc, mỗi file 1 trách nhiệm rõ ràng.
- **Trực quan**: nhìn folder + filename là biết file thuộc nhóm nào.
- **Đầy đủ**: README/header doc giải thích purpose, depends_on, idempotency.
- **Rõ ràng cho từng nhóm chức năng**: tách nhóm theo trục **chức năng**
  (schema/seed/cluster/archive) thay vì trộn DDL với seed.
- **Production với config riêng**: production deploy phải có **toggle** để
  KHÔNG seed dữ liệu demo/sample vào DB thật.

## 3. Constraints (User-stated)

1. Đọc lesson trước tất cả → đã đọc `lessons.md` phần migration.
2. Đọc `agent/GEMINI.md` để hiểu role/skill → đã đọc.
3. Chỉ làm đúng những gì được yêu cầu → KHÔNG tự fix bug khác.
4. Theo hướng core systems → KHÔNG cheat DB, KHÔNG đổi config để qua mặt.
5. Report dựa trên kết quả tính toán thực tế → KHÔNG báo láo.
6. Kết thúc → verify service work mới báo done.
7. Luôn có `report_*.md` ghi lại thay đổi.

## 4. Out of scope

- KHÔNG sửa schema (CREATE/ALTER) ngoài việc tách INSERT-seed ra file riêng.
- KHÔNG fix V1/V2 dual coexistence (lesson đã ghi nhận, là design choice).
- KHÔNG fix partition rotation TIME BOMB của 010/040 (defer phase khác).
- KHÔNG renumber file (giữ basename = giữ tracker compat với DB local đã apply).
- KHÔNG sửa code Go ngoài: `config/config.go`, `internal/migrate/runner.go`,
  `internal/server/server.go` (chỉ để add `skipSeeds` toggle).
- KHÔNG đụng vào `cdc-auth-service` hoặc `centralized-data-service` migrations.

## 5. Inventory hiện tại

### 5.1 File embed (28 file, sort theo basename):

| # | Path | Group | Purpose |
|---|---|---|---|
| 1 | core/001_init_schema.sql | core | V1 cdc_table_registry + mapping_rules + pending_fields + schema_changes_log (seed block disabled) |
| 2 | core/002_standardize_schema.sql | core | V1 standardize_cdc_table helper |
| 3 | ids/003_sonyflake_schema.sql | ids | V1.12 create_cdc_table BIGINT + per-table seq |
| 4 | worker/007_worker_schedule.sql | worker | cdc_worker_schedule + **5 INSERT seed** |
| 5 | recon_dlq/008_reconciliation.sql | recon_dlq | cdc_reconciliation_report + failed_sync_logs (legacy non-partitioned) |
| 6 | partitioning/010_partitioning.sql | partitioning | Partitioned cdc_system.failed_sync_logs + cdc_activity_log (squash 012, 045) |
| 7 | recon_dlq/011_recon_runs.sql | recon_dlq | recon_runs state table |
| 8 | registry/013_table_registry_expected_fields.sql | registry | ALTER cdc_table_registry (squash 6 file 013/014/016/017/046) |
| 9 | ids/018_sonyflake_v125_foundation.sql | ids | cdc_internal schema + worker_registry + sequences + fencing |
| 10 | registry/019_system_registry.sql | registry | cdc_internal.table_registry (V1.25 lifecycle, squash 024) |
| 11 | registry/020_mapping_rule_jsonpath.sql | registry | ALTER cdc_mapping_rules + **enum_types + 3 INSERT seed enums** (squash 046) |
| 12 | worker/022_transmute_schedule.sql | worker | cdc_internal.transmute_schedule |
| 13 | registry/023_master_table_registry.sql | registry | cdc_internal.master_table_registry |
| 14 | registry/025_schema_proposal.sql | registry | cdc_internal.schema_proposal |
| 15 | registry/027_systematic_sources.sql | registry | cdc_internal.sources + wizard_sessions |
| 16 | cdc_system_model/029_v2_connection_registry.sql | cdc_system_model | V2 connection_registry + **3 INSERT seed legacy_* connections** (squash 035) |
| 17 | cdc_system_model/030_v2_source_object_registry.sql | cdc_system_model | V2 source_object_registry (squash 047) |
| 18 | cdc_system_model/031_v2_shadow_binding.sql | cdc_system_model | V2 shadow_binding |
| 19 | cdc_system_model/032_v2_master_binding.sql | cdc_system_model | V2 master_binding |
| 20 | cdc_system_model/033_v2_mapping_rule.sql | cdc_system_model | V2 mapping_rule_v2 |
| 21 | cdc_system_model/034_v2_sync_runtime_state.sql | cdc_system_model | V2 sync_runtime_state |
| 22 | cdc_system_model/036_v2_transmute_schedule.sql | cdc_system_model | V2 transmute_schedule + **INSERT...SELECT** từ cdc_internal |
| 23 | core/037_move_system_tables_to_cdc_system.sql | core | SET SCHEMA public.* / cdc_internal.* → cdc_system |
| 24 | core/038_finalize_cdc_system_namespace.sql | core | Move sequences/functions, DROP cdc_internal |
| 25 | audit_security/040_admin_actions_in_cdc_system.sql | audit_security | cdc_system.admin_actions partitioned (April-June 2026) |
| 26 | audit_security/041_cdc_alerts_in_cdc_system.sql | audit_security | cdc_system.cdc_alerts |
| 27 | core/044_cleanup_public_residue.sql | core | DROP orphan partitions + verify public empty |
| 28 | ops/048_provisioning_log_cap_helper.sql | ops | append_step_log_capped function |
| 29 | ops/052_create_cdc_jobs.sql | ops | cdc_system.cdc_jobs tracker |

### 5.2 File KHÔNG embed:

- `cluster/001_roles.sql`, `cluster/002_search_path.sql` + `cluster/README.md`
  → DBA chạy manual với superuser.
- `.archive/003_add_mapping_rule_status.sql`, `.archive/004_bridge_columns.sql`,
  `.archive/005_admin_actions.sql`, `.archive/013_alerts.sql` → đã đóng băng.

### 5.3 Tổng: 28 embedded + 2 cluster + 4 archived = 34 file SQL.

## 6. Pain points đã identify

1. **Numbering có 24 lỗ thủng** (do squash/archive): 004-006, 009, 012, 014-017,
   021, 024, 026, 028, 035, 039, 042-043, 045-047, 049-051, 053. Không có doc
   liệt kê lý do từng số bị bỏ.
2. **Seed data INSERT trong cùng file DDL** (3 vị trí: 007, 020, 029):
   - `worker/007` seed 5 default schedule rows.
   - `registry/020` seed 3 enum_types (payment_state, api_type, currency_iso).
   - `cdc_system_model/029` seed 3 legacy_*_default connections.
   → Production cold-boot phải gồng dataset này. Bootstrap Go code đã có
     `EnsureDefaultShadowConnection` (server.go:94) làm duplicate logic.
3. **README drift**:
   - `cdc_system_model/README.md` mention `028_sonyflake_fallback_fn.sql` và
     `035_v2_backfill_legacy_registry.sql` — KHÔNG TỒN TẠI (squashed/moved).
   - `cluster/README.md` mention `005_pg_users.sql` legacy — KHÔNG TỒN TẠI.
4. **Top-level migrations/ KHÔNG có README** giải thích layout.
5. **`.archive/` chấm dấu chấm KHÔNG có README** giải thích lý do từng file
   bị archive + có replacement nào.
6. **`audit_security/040` cố định partitions April-June 2026** — time-bomb sau
   2026-06 (default partition sẽ phình).
7. **Production config thiếu toggle** để skip seed:
   - `config-production.yml` không có `migration.*` section.
   - `config.go` không có `MigrationConfig` struct.
   - migrator chạy 100% files bất kể env.
8. **Tracker version dùng basename** → nếu 2 file cùng basename ở folder khác
   nhau sẽ collide (hiện tại không xảy ra nhưng là time-bomb).
9. **search_path swap hack** trong `runner.go:189-196`: SET LOCAL search_path
   TO public, "$user" cho mỗi tx vì legacy migration author dưới default
   search_path. File V2 đều schema-qualify — hack hiện vẫn cần cho compat
   nhưng không clear cho người mới đọc.
10. **embed.go pattern selective**: `core/*.sql ids/*.sql partitioning/*.sql
    registry/*.sql worker/*.sql recon_dlq/*.sql audit_security/*.sql
    cdc_system_model/*.sql ops/*.sql` — phải maintain manually mỗi khi thêm
    folder mới.

## 7. Giao tiếp với hệ thống

- **DB target**: PostgreSQL `cdc_dw` (port 5433 local, controlled by
  `config.DB`). Server.go:63 gọi `migrate.Run(db, logger)` ngay sau khi connect.
- **Tracker**: `cdc_system.schema_migrations` (version VARCHAR PK, applied_at).
- **Lock**: `pg_advisory_lock(0x4344444D49475282042)` pin trên dedicated conn.
- **Idempotency**: re-run = no-op (tracker check + advisory lock).

## 8. Liên quan

- Lesson `L-multi-engine-2`: Migration draft phải align với `\d <table>` thực tế.
- Lesson 2026-05-11 `Production migration seeds demo data`: schema migrations
  chỉ chứa DDL + config-like seed (enum, schedule); business data KHÔNG seed.
- Lesson 2026-05-11 `Audit table usage on both consumer services BEFORE
  adding migration`: pre-merge check `grep -r tablename worker/ cms/`.
- Workspace cũ liên quan: `feature-cdc-system-recreate-2026-05-11` (đã làm
  một số việc disable seed).
