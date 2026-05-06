# Phase 39 — Implementation Log (template — Muscle fill sau exec)

## Files changed

### 1) Migration mới (centralized-data-service)
- `migrations/040_admin_actions_in_cdc_system.sql` — NEW (xem 09_tasks_solution T-39.8a)
- `migrations/041_cdc_alerts_in_cdc_system.sql` — NEW (xem 09 T-39.8b)
- `migrations/042_search_path_with_auth.sql` — NEW (xem 09 T-39.8c)
- `migrations/043_normalize_shadow_binding_schema.sql` — NEW Option A (xem 09 T-39.8e)

### 1b) Migration rewrite (centralized-data-service) — Option A
- `migrations/028_sonyflake_fallback_fn.sql` — REWRITE: `cdc_internal.*` → `cdc_system.*`, helper 2-arg `(p_schema, p_table)` (xem 09 T-39.8d)

### 2) Migration rewrite (cdc-auth-service)
- `migrations/001_auth_users.sql` — REWRITE: `CREATE SCHEMA cdc_auth_service` + table trong schema mới (xem 09 T-39.9)

### 3) Code patches (Go) — schema-qualify (3 file)
- `cdc-auth-service/internal/model/user.go:17` — `TableName()` qualify `cdc_auth_service.auth_users` (T-39.10a)
- `cdc-cms-service/internal/model/alert.go:37` — `TableName()` qualify `cdc_system.cdc_alerts` (T-39.10b)
- `cdc-cms-service/internal/middleware/audit.go:166` — `INSERT INTO cdc_system.admin_actions` (T-39.10c)

### 3b) Code patches (Go) — binding-aware refactor (4 file) — Option A
- `cdc-cms-service/internal/service/shadow_automator.go` — signature `EnsureShadowTable(ctx, reg, shadowSchema)`, bỏ `ensureSonyflakeFunction`, bỏ `CREATE SCHEMA cdc_internal`, helper 2-arg (T-39.10d)
- `cdc-cms-service/internal/api/registry_handler.go:113` — caller resolve `shadowSchema = "shadow_" + normalizeShadowIdent(entry.SourceDB)` + thêm helper `normalizeShadowIdent` (T-39.10e)
- `cdc-cms-service/internal/api/mapping_preview_handler.go:47` — binding lookup từ `cdc_system.shadow_binding`, dynamic schema, miss → 404 binding_not_found (T-39.10f)
- `cdc-cms-service/internal/api/schema_proposal_handler.go:131` — case "shadow" lookup binding, ALTER TABLE schema động (T-39.10g)

### 4) Wipe script
- `centralized-data-service/deployments/sql/wipe_cdc_runtime_v2.sql` — replace public residue section bằng `DROP SCHEMA public CASCADE; CREATE SCHEMA public;` + dynamic shadow_*/dw_* drop loop (xem 09 T-39.11)

### 5) Migration orphan (document để xoá)
- `cdc-cms-service/migrations/003_add_mapping_rule_status.sql`
- `cdc-cms-service/migrations/004_bridge_columns.sql`
- `cdc-cms-service/migrations/005_admin_actions.sql` ← replaced by `040_admin_actions_in_cdc_system.sql`
- `cdc-cms-service/migrations/013_alerts.sql` ← replaced by `041_cdc_alerts_in_cdc_system.sql`

(Đã verify: cdc-cms-service không có Makefile target chạy thư mục `migrations/` → orphan 100%, an toàn xoá. Trên thực tế đã apply manual lúc nào đó vì admin_actions / cdc_alerts đang tồn tại trong public.)

## Build matrix
- `cdc-auth-service`: `go build ./...` → ⏳
- `cdc-cms-service`: `go build ./...` → ⏳
- `centralized-data-service`: `go build ./...` → ⏳

## Backup artifacts
- `/tmp/phase39_schema_<ts>.sql` — schema-only dump trước wipe
- `/tmp/phase39_cdc_system_<ts>.sql` — cdc_system data dump
- `/tmp/phase39_auth_users_<ts>.sql` — auth_users dump (rollback safety)

## Restart sequence
1. `cdc-auth-service` (8081)
2. `cdc-cms-service` (8083)
3. `centralized-data-service` worker (8084)
4. `centralized-data-service` transmute (8085)

## Auth note
Sau Phase 39, route + body login giữ nguyên Phase 38:
```bash
curl -s -X POST http://localhost:8081/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
```
Bảng đích đổi từ `public.auth_users` → `cdc_auth_service.auth_users`. GORM model qualify trong code, không phụ thuộc search_path (mặc dù search_path cũng đã include `cdc_auth_service` qua migration 042 — defense in depth).
