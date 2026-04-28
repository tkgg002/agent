# Implementation Phase 1

## Code changes applied

### 1. Added V2 migrations

- `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/029_v2_connection_registry.sql`
- `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/030_v2_source_object_registry.sql`
- `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/031_v2_shadow_binding.sql`
- `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/032_v2_master_binding.sql`
- `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/033_v2_mapping_rule.sql`
- `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/034_v2_sync_runtime_state.sql`
- `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/035_v2_backfill_legacy_registry.sql`

### 2. Added V2 models

- `internal/model/connection_registry.go`
- `internal/model/source_object_registry.go`
- `internal/model/shadow_binding.go`
- `internal/model/master_binding.go`
- `internal/model/mapping_rule_v2.go`
- `internal/model/sync_runtime_state.go`

### 3. Added V2 repositories

- `internal/repository/connection_registry_repo.go`
- `internal/repository/source_object_registry_repo.go`
- `internal/repository/shadow_binding_repo.go`
- `internal/repository/master_binding_repo.go`
- `internal/repository/mapping_rule_v2_repo.go`
- `internal/repository/sync_runtime_state_repo.go`

### 4. Extended config for single-system + multi-shadow/master DB targets

- `config/config.go`
  - added `systemDb.url`
  - added `shadowDb.defaultKey`, `shadowDb.urls`
  - added `masterDb.defaultKey`, `masterDb.urls`
  - added env parsing:
    - `CDC_SYSTEM_DB_URL`
    - `CDC_SHADOW_DB_URL`
    - `CDC_SHADOW_DB_URLS`
    - `CDC_SHADOW_DB_DEFAULT_KEY`
    - `CDC_MASTER_DB_URL`
    - `CDC_MASTER_DB_URLS`
    - `CDC_MASTER_DB_DEFAULT_KEY`
  - added fallback so all new roles still point to the legacy single DB when not explicitly configured
  - fixed actual use of `DB_SINK_URL` by storing it into `cfg.DB.URL`
- `pkgs/database/postgres.go`
  - now opens GORM using `cfg.DB.DSN()`
  - added `NewPostgresConnectionByDSN`
- `pkgs/database/pgx_pool.go`
  - now opens pgx pool using `cfg.DB.PgxDSN()`
  - added `NewPgxPoolByDSN`
- `config/config-local.yml`
  - added example sections for `systemDb`, `shadowDb`, `masterDb`

### 5. Added connection manager scaffold

- `internal/service/connection_manager.go`
  - caches one system DB connection
  - caches named shadow/master DB connections by key
  - reads from:
    - `cfg.SystemDBURL()`
    - `cfg.ShadowDBURLs()`
    - `cfg.MasterDBURLs()`
  - uses the new `NewPostgresConnectionByDSN` helper

## Implementation notes

1. Migrations dùng `IF NOT EXISTS` để tránh blast radius trong môi trường đang có dữ liệu.
2. `035_v2_backfill_legacy_registry.sql` chỉ bootstrap metadata V2 từ các bảng legacy hiện hữu, chưa đổi runtime path.
3. Models bind trực tiếp vào `cdc_system.*` để chuẩn bị cho phase refactor service.
4. Repositories mới giữ mức tối thiểu đủ dùng cho phase kế tiếp:
   - get by id/code
   - list active/by source
   - create/update
