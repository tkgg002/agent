# Phase 39 — Requirements: Schema Consolidation (cdc_system + cdc_auth_service)

## Mục tiêu

Đưa toàn bộ application system tables vào schema có chủ — không còn 1 table system nào nằm trong `public`. Sau Phase 39:

- `cdc_system` — CDC control plane (≥23 tables hiện có + `admin_actions`, `cdc_alerts` move từ public)
- `cdc_auth_service` — Auth service tables (`auth_users` + tương lai)
- `shadow_<src>` — CDC ingest output (per source DB, do worker tạo runtime)
- `dw_<binding>` — Master output (per binding, do transmute pipeline tạo runtime)
- `public` — RỖNG (verify: 0 rows trong `pg_tables WHERE schemaname='public'`)
- `cdc_internal` — DROP (đã deprecated từ phase 37)

## Rule absolute từ user

> "toàn bộ table của hệ thống ở cdc_system, ko 1 table nào nằm ngoài schema này"
> Đính chính: auth-service vẫn riêng → `cdc_auth_service` schema riêng. Các bảng shadow/master vẫn nằm ở `shadow_<src>`/`dw_<binding>` đúng vị trí của nó.
> "public xoá mẹ đi" → public phải rỗng hoàn toàn sau cutover.

## Scope

### IN scope

1. **Move CDC system tables** đang ở public → cdc_system:
   - `admin_actions` (partitioned by month) + 4 child partitions (2026_04/05/06/default)
   - `cdc_alerts`
2. **Tạo schema mới `cdc_auth_service`** + move `auth_users` (kèm seed admin/admin123) sang đó
3. **Drop toàn bộ residue trong public**:
   - Partition orphan: `cdc_activity_log_2026042{6..30}`, `cdc_activity_log_2026050{1,2}`, `cdc_activity_log_default`, `cdc_activity_log_legacy`, `failed_sync_logs_y2026m04..m07`, `failed_sync_logs_default`, `failed_sync_logs_legacy`
   - Shadow ingest data sai vị trí: `payments`, `users`, `merchants`, `orders`, `order_items`, `refunds`, `wallets`, `wallet_transactions`, `payment_bills`, `payment_bill_codes`, `payment_bill_events`, `payment_bill_histories`, `payment_bill_holdings`, `refund_requests`, `export_jobs`, `identitycounters`, `legacy_payments`, `legacy_refunds`
   - Master data sai vị trí: `export_jobs_master`, `refund_requests_master`
   - Test residue: `test_table_refactor_202604`
4. **Drop schema `cdc_internal`** (verify rỗng trước)
5. **Update search_path role**: `cdc_system, cdc_auth_service, public`
6. **Update Go code qualify schema**:
   - `cdc-auth-service/internal/model/user.go:17` — `TableName() = "cdc_auth_service.auth_users"`
   - `cdc-cms-service/internal/model/alert.go:37` — `TableName() = "cdc_system.cdc_alerts"`
   - `cdc-cms-service/internal/middleware/audit.go:166` — `INSERT INTO cdc_system.admin_actions`
7. **Migration mới**:
   - `centralized-data-service/migrations/040_admin_actions_in_cdc_system.sql`
   - `centralized-data-service/migrations/041_cdc_alerts_in_cdc_system.sql`
   - `centralized-data-service/migrations/042_search_path_with_auth.sql`
   - `cdc-auth-service/migrations/001_auth_users.sql` REWRITE (CREATE SCHEMA cdc_auth_service + table trong schema mới)
8. **Migration prune**:
   - `cdc-cms-service/migrations/{003,004,005,013}.sql` — document để xoá (đã được 040/041 thay thế; chưa có Makefile runner cho thư mục này)

### OUT of scope

- Không refactor cdc-auth-service business logic, chỉ qualify schema
- Không thay đổi shadow_automator / transmute pipeline behavior
- Không thay đổi Debezium connector config
- Không backup/restore data thực — wipe toàn bộ runtime + bootstrap fresh (môi trường local dev)

## Definition of Done

| # | Criteria | Verify |
|---|---|---|
| 1 | `SELECT count(*) FROM pg_tables WHERE schemaname='public'` = 0 | psql one-liner |
| 2 | `\dn` chỉ thấy `cdc_system`, `cdc_auth_service`, `public` (rỗng), pg_* defaults | psql `\dn` |
| 3 | `\dt cdc_system.admin_actions` exist + partitioned + 4 partitions | psql `\d+` |
| 4 | `\dt cdc_system.cdc_alerts` exist | psql |
| 5 | `\dt cdc_auth_service.auth_users` exist + seed admin/admin123 | psql + `SELECT username FROM cdc_auth_service.auth_users;` |
| 6 | Schema `cdc_internal` không tồn tại | `SELECT count(*) FROM pg_namespace WHERE nspname='cdc_internal';` = 0 |
| 7 | `SHOW search_path;` = `cdc_system, cdc_auth_service, public` | psql |
| 8 | Login `admin/admin123` qua `:8081` trả JWT 200 | curl `/api/auth/login` |
| 9 | 11 endpoints CMS = 200 (theo Phase 38 verify pack) | bash loop |
| 10 | Debezium connector RUNNING + ≥4 topics | curl `:18083/connectors/.../status` |
| 11 | Worker `discoverTopics` loop healthy không panic | `tail /tmp/cdc-worker.log` |
| 12 | `go build ./...` PASS cả 4 service | shell |

## ✅ Architect Decision — Option A (Approved 2026-04-28)

**Lý do chọn**: Option B vi phạm Separation of Concerns (trộn shadow data vào control plane). Option C đẩy debt → user phải đi check lại sau. Option A là con đường duy nhất tuân Runbook V2.

**Scope mở rộng** (so với version trước):
- shadow_automator + mapping_preview_handler + schema_proposal_handler → binding-aware (`ShadowBinding.shadow_schema = shadow_<source_db>`)
- shadow_automator drop helper `ensure_shadow_sonyflake_trigger(p_table TEXT)` — thay bằng version 2-arg `(p_schema TEXT, p_table TEXT)` để target schema động
- Sonyflake function ở `cdc_system.gen_sonyflake_id()` — dùng chung global, không recreate trong shadow schema

**Migration mới (so với version trước)**:
- Migration 028 REWRITE → tạo function trong `cdc_system`, helper signature `(p_schema, p_table)`
- Migration 043 NEW → normalize legacy `cdc_system.shadow_binding.shadow_schema = 'cdc_internal'` → `shadow_<source_db>` per source_object linkage
- Wipe script: thêm `DROP SCHEMA shadow_*` loop (đã có) + `DROP SCHEMA cdc_internal CASCADE` (đã có)

**Code patches Go (6 file)**:
| # | File | Hành vi |
|---|---|---|
| 1 | `cdc-auth-service/internal/model/user.go:17` | `TableName() = "cdc_auth_service.auth_users"` |
| 2 | `cdc-cms-service/internal/model/alert.go:37` | `TableName() = "cdc_system.cdc_alerts"` |
| 3 | `cdc-cms-service/internal/middleware/audit.go:166` | Raw SQL `INSERT INTO cdc_system.admin_actions` |
| 4 | `cdc-cms-service/internal/service/shadow_automator.go` | Refactor: nhận `shadowSchema string` từ caller; xóa block CREATE SCHEMA cdc_internal; xóa function bootstrap inline (function đã ở `cdc_system` qua migration); attachTrigger dùng `cdc_system.ensure_shadow_sonyflake_trigger(p_schema, p_table)` |
| 5 | `cdc-cms-service/internal/api/mapping_preview_handler.go` | Lookup `shadow_schema` từ `cdc_system.shadow_binding` theo `req.ShadowTable` (1 query) trước khi build SELECT |
| 6 | `cdc-cms-service/internal/api/schema_proposal_handler.go` | Tương tự — lookup `shadow_schema` cho `row.TableName` (TableLayer="shadow") |

**Caller upstream của EnsureShadowTable** (`cdc-cms-service/internal/api/registry_handler.go:113`): phải resolve `shadowSchema` từ binding LOOKUP trước khi gọi. Pragma: `shadow_<source_db_normalized>` (lower_snake_case, dấu `-` → `_`).

**Đã ghi nhận debt giải quyết**: D-38.A từ Phase 38.

## DoD Criteria B Hardened (2-step verify)

| Step | Khi nào | Verify | Expected |
|---|---|---|---|
| B1 | Ngay sau migrate | `SELECT count(*) FROM pg_namespace WHERE nspname='cdc_internal'` | `0` |
| B2 | **Sau worker chạy ≥5 phút + register 1 source object qua Wizard** | Same query | `0` (KHÔNG được tự về) |

Nếu B1 = 0 nhưng B2 > 0 → có code path nào còn hardcode `cdc_internal` em chưa thấy → re-grep + fix → re-execute Phase 39.

## Risks

| Risk | Mitigation |
|---|---|
| Wipe public CASCADE phá data thật | Local dev only; dump backup ra `/tmp/phase39_backup_<ts>.sql` trước wipe |
| Auth service connection pool cache search_path cũ | Restart cdc-auth-service sau khi ALTER ROLE |
| GORM `TableName()` ghi `"cdc_auth_service.auth_users"` không work với 1 số driver | Postgres GORM hỗ trợ qualified table name; verify bằng smoke test login |
| Migration 005/013 cũ của cdc-cms-service tự apply lại lúc service restart | Đã verify: cdc-cms-service không có migration runner cho thư mục `migrations/` (no Makefile target). Tuy nhiên để chắc, mark obsolete bằng comment header trong file. |
| `DROP SCHEMA public CASCADE` ảnh hưởng extension default | Sau drop, recreate `CREATE SCHEMA public;` rỗng để extension như pgcrypto vẫn hoạt động (extension đã được pin schema riêng) |

## Dependencies

- Runbook: `cdc-system/centralized-data-service/deployments/runbooks/wipe_bootstrap_v2.md`
- Wipe script base: `cdc-system/centralized-data-service/deployments/sql/wipe_cdc_runtime_v2.sql`
- Bootstrap seed: `cdc-system/centralized-data-service/deployments/sql/bootstrap_cdc_system_v2_local.sql`
- Phase 38 lesson về search_path coupling (đã append `agent/memory/global/lessons.md`)
