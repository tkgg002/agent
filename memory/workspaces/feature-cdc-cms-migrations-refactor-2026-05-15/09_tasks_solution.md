# 09 — Tasks Solution (Code Snippets)

> Solution snippet cho mỗi task trong `08_tasks.md`. Format: copy-paste ready.
> Reference: 03_implementation.md để hiểu context.

---

## T-03 — Folder skeleton

### T-03.1 → T-03.6 — Move folders

```bash
set -euo pipefail
cd cdc-cms-service/migrations

# Pre-flight
[ ! -d schema ] || { echo "ERR: schema/ exists"; exit 1; }
for d in core ids partitioning registry worker recon_dlq audit_security cdc_system_model ops; do
  [ -d "$d" ] || { echo "ERR: source folder missing: $d"; exit 1; }
done

mkdir -p schema seed archive

for d in core ids partitioning registry worker recon_dlq audit_security cdc_system_model ops; do
  git mv "$d" "schema/$d"
done

if [ -d ".archive" ]; then
  for f in .archive/*.sql; do
    [ -e "$f" ] && git mv "$f" "archive/$(basename "$f")"
  done
  if [ -f ".archive/README.md" ]; then
    git mv ".archive/README.md" "archive/README.md"
  fi
  rmdir .archive
fi

git status --short
```

Expected output (line-prefix R = rename):
```
R  core/001_init_schema.sql -> schema/core/001_init_schema.sql
R  core/002_standardize_schema.sql -> schema/core/002_standardize_schema.sql
... (28 schema renames)
R  .archive/003_add_mapping_rule_status.sql -> archive/003_add_mapping_rule_status.sql
... (4 archive renames)
```

---

## T-04 — Split seed blocks

### T-04.1 — `schema/worker/007_worker_schedule.sql`

```sql
-- ==========================================================================
-- File:        007_worker_schedule.sql
-- Purpose:     Tạo bảng cdc_worker_schedule (config-driven runner schedule).
-- Schema:      public (will be moved to cdc_system by migration 037).
-- Idempotent:  CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS.
-- Depends on:  001_init_schema, 002_standardize_schema.
-- Env scope:   schema-only. Default 5 schedule rows moved to
--              seed/100_worker_schedules.sql (apply when skipSeeds=false).
-- ==========================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS cdc_worker_schedule (
  job_name          VARCHAR(64) PRIMARY KEY,
  interval_seconds  INTEGER NOT NULL,
  next_run_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_run_at       TIMESTAMPTZ,
  is_paused         BOOLEAN NOT NULL DEFAULT FALSE,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_worker_schedule_next_run
  ON cdc_worker_schedule(next_run_at)
  WHERE is_paused = FALSE;

COMMIT;
```
(Nội dung DDL exact phải đọc file 007 thực tế. Snippet trên là khung — Muscle
phải copy DDL gốc, KHÔNG re-design.)

### T-04.2 — `schema/cdc_system_model/029_v2_connection_registry.sql`

Bỏ lines 51-99 (3 INSERT blocks). Header mới:

```sql
-- ==========================================================================
-- File:        029_v2_connection_registry.sql
-- Purpose:     V2 control plane — cdc_system.connection_registry catalog.
-- Schema:      cdc_system (created here if missing).
-- Idempotent:  CREATE SCHEMA IF NOT EXISTS / CREATE TABLE IF NOT EXISTS.
-- Depends on:  none (foundation of V2 schema set).
-- Env scope:   schema-only. Legacy infra seed (legacy_system_db /
--              legacy_shadow_default / legacy_master_default) moved to
--              seed/101_v2_legacy_connections.sql.
--              Production deploys MUST rely on bootstrap.EnsureDefaultShadowConnection
--              (server.go) thay vì SQL seed.
-- ==========================================================================
-- SQUASH HISTORY:
--   2026-05-14 — Absorbed 3 INSERT seeds từ migration 035 (now archived).
--   2026-05-15 — Split INSERT seeds → seed/101 (production safety).
-- ==========================================================================

BEGIN;

CREATE SCHEMA IF NOT EXISTS cdc_system;

CREATE TABLE IF NOT EXISTS cdc_system.connection_registry (
  id                BIGSERIAL PRIMARY KEY,
  connection_code   VARCHAR(100) NOT NULL UNIQUE,
  display_name      VARCHAR(200) NOT NULL,
  role_type         VARCHAR(32) NOT NULL
    CHECK (role_type IN ('source','shadow','master','system','mixed')),
  engine_type       VARCHAR(32) NOT NULL
    CHECK (engine_type IN ('postgresql','mariadb','mysql','mongodb','clickhouse')),
  host              VARCHAR(255),
  port              INTEGER,
  default_database  VARCHAR(255),
  default_schema    VARCHAR(255),
  secret_ref        VARCHAR(255) NOT NULL,
  options_json      JSONB NOT NULL DEFAULT '{}'::jsonb,
  capabilities_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  status            VARCHAR(32) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','paused','failed','retired')),
  created_by        VARCHAR(100),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_v2_connection_role
  ON cdc_system.connection_registry(role_type);

CREATE INDEX IF NOT EXISTS idx_v2_connection_engine
  ON cdc_system.connection_registry(engine_type);

CREATE INDEX IF NOT EXISTS idx_v2_connection_status
  ON cdc_system.connection_registry(status);

COMMENT ON SCHEMA cdc_system IS
  'V2 control plane metadata schema. Stores registry, bindings, and runtime state. Does not store shadow/master payload tables.';

COMMENT ON TABLE cdc_system.connection_registry IS
  'Physical connection catalog used by source, shadow, master, and system layers.';

COMMIT;
```

### T-04.3 — `schema/registry/020_mapping_rule_jsonpath.sql`

Chỉ thêm header. Giữ nguyên INSERT enum_types.

```sql
-- ==========================================================================
-- File:        020_mapping_rule_jsonpath.sql
-- Purpose:     ALTER cdc_mapping_rules để thêm jsonpath fields + CREATE
--              cdc_enum_types + seed 3 domain enums (payment_state,
--              api_type, currency_iso).
-- Schema:      public.
-- Idempotent:  ALTER ... ADD COLUMN IF NOT EXISTS / INSERT ON CONFLICT
--              DO NOTHING.
-- Depends on:  001_init_schema (cdc_mapping_rules base table).
-- Env scope:   schema-only — enum_types là domain config bất biến, áp
--              dụng MỌI env (kể cả production).
-- ==========================================================================

-- (giữ nguyên content gốc từ đây xuống)
```

---

## T-05 — New seed files

### T-05.1 — `migrations/seed/100_worker_schedules.sql`

```sql
-- ==========================================================================
-- File:        100_worker_schedules.sql
-- Purpose:     Seed 5 default worker schedule rows.
-- Schema:      cdc_system (re-located by 037).
-- Idempotent:  INSERT ... ON CONFLICT (job_name) DO NOTHING.
-- Depends on:  schema/worker/007_worker_schedule.sql,
--              schema/core/037_move_system_tables_to_cdc_system.sql.
-- Env scope:   seed (dev/staging). Skipped when CMS_MIGRATION_SKIP_SEEDS=true.
-- Source:      Split from schema/worker/007_worker_schedule.sql (2026-05-15).
-- ==========================================================================

BEGIN;

INSERT INTO cdc_system.cdc_worker_schedule (job_name, interval_seconds, next_run_at)
VALUES
  ('bridge',           60,  NOW()),
  ('transform',        90,  NOW()),
  ('field-scan',       120, NOW()),
  ('partition-check',  300, NOW()),
  ('airbyte-sync',     180, NOW())
ON CONFLICT (job_name) DO NOTHING;

COMMIT;
```

### T-05.2 — `migrations/seed/101_v2_legacy_connections.sql`

```sql
-- ==========================================================================
-- File:        101_v2_legacy_connections.sql
-- Purpose:     Seed 3 legacy infrastructure connections (system/shadow/master)
--              for V2 control plane.
-- Schema:      cdc_system.
-- Idempotent:  INSERT ... WHERE NOT EXISTS.
-- Depends on:  schema/cdc_system_model/029_v2_connection_registry.sql.
-- Env scope:   seed (dev/staging). Skipped when CMS_MIGRATION_SKIP_SEEDS=true.
-- Source:      Split from schema/cdc_system_model/029_v2_connection_registry.sql
--              (2026-05-15). Production cold-boot dùng
--              bootstrap.EnsureDefaultShadowConnection (Go) thay vì SQL.
-- ==========================================================================

BEGIN;

INSERT INTO cdc_system.connection_registry (
  connection_code, display_name, role_type, engine_type,
  default_database, default_schema, secret_ref,
  options_json, capabilities_json, status, created_by
)
SELECT
  'legacy_system_db', 'Legacy System DB', 'system', 'postgresql',
  current_database(), 'public', 'env:DB_SINK_URL',
  '{}'::jsonb,
  '{"supports_schema": true, "supports_upsert": true, "supports_jsonb": true}'::jsonb,
  'active', 'seed_101'
WHERE NOT EXISTS (
  SELECT 1 FROM cdc_system.connection_registry WHERE connection_code = 'legacy_system_db'
);

INSERT INTO cdc_system.connection_registry (
  connection_code, display_name, role_type, engine_type,
  default_database, default_schema, secret_ref,
  options_json, capabilities_json, status, created_by
)
SELECT
  'legacy_shadow_default', 'Legacy Shadow Default', 'shadow', 'postgresql',
  current_database(), 'cdc_internal', 'env:DB_SINK_URL',
  '{}'::jsonb,
  '{"supports_schema": true, "supports_upsert": true, "supports_jsonb": true}'::jsonb,
  'active', 'seed_101'
WHERE NOT EXISTS (
  SELECT 1 FROM cdc_system.connection_registry WHERE connection_code = 'legacy_shadow_default'
);

INSERT INTO cdc_system.connection_registry (
  connection_code, display_name, role_type, engine_type,
  default_database, default_schema, secret_ref,
  options_json, capabilities_json, status, created_by
)
SELECT
  'legacy_master_default', 'Legacy Master Default', 'master', 'postgresql',
  current_database(), 'public', 'env:DB_SINK_URL',
  '{}'::jsonb,
  '{"supports_schema": true, "supports_upsert": true, "supports_jsonb": true}'::jsonb,
  'active', 'seed_101'
WHERE NOT EXISTS (
  SELECT 1 FROM cdc_system.connection_registry WHERE connection_code = 'legacy_master_default'
);

COMMIT;
```

---

## T-07 — `embed.go`

```go
package migrations

import "embed"

// SchemaFiles bundles all DDL-only SQL files (schema/**).
// Applied on every boot regardless of environment.
//
//go:embed all:schema
var SchemaFiles embed.FS

// SeedFiles bundles env-specific seed SQL files (seed/*.sql).
// Skipped when cfg.Migration.SkipSeeds=true (production default).
//
//go:embed all:seed
var SeedFiles embed.FS
```

---

## T-08 — `config/config.go`

```go
// MigrationConfig kiểm soát behavior của runtime migrator.
type MigrationConfig struct {
    // SkipSeeds=true → runner KHÔNG apply file trong migrations/seed/.
    // Production yml mặc định true (chỉ apply DDL).
    // Local yml mặc định false (apply cả DDL + seed).
    // ENV override: CMS_MIGRATION_SKIP_SEEDS.
    SkipSeeds bool `mapstructure:"skipSeeds"`
}

type AppConfig struct {
    Server    ServerConfig    `mapstructure:"server"`
    DB        DBConfig        `mapstructure:"db"`
    ShadowDB  ShadowDBConfig  `mapstructure:"shadowDB"`
    Nats      NatsConfig      `mapstructure:"nats"`
    Redis     RedisConfig     `mapstructure:"redis"`
    JWT       JWTConfig       `mapstructure:"jwt"`
    System    SystemConfig    `mapstructure:"system"`
    Otel      OtelConfig      `mapstructure:"otel"`
    Migration MigrationConfig `mapstructure:"migration"` // NEW
}
```

Trong `Load()` (chỗ `v.SetEnvKeyReplacer` + `BindEnv`):
```go
v.SetDefault("migration.skipSeeds", false)
_ = v.BindEnv("migration.skipSeeds", "CMS_MIGRATION_SKIP_SEEDS")
```

---

## T-09 — yml updates

### T-09.1 `config-production.yml` (append cuối file)

```yaml
migration:
  # Production: chỉ apply DDL (schema/**), KHÔNG apply seed/.
  # Legacy infra rows: bootstrap.EnsureDefaultShadowConnection (Go) handle.
  # Default schedules: operator config qua CMS UI sau cold-boot.
  skipSeeds: true
```

### T-09.2 `config-local.yml` (append cuối file)

```yaml
migration:
  # Local dev: apply cả schema + seed.
  skipSeeds: false
```

### T-09.3 `config-sample.yml` (append cuối file)

```yaml
migration:
  # Toggle áp dụng file trong migrations/seed/.
  # - true:  production-style, chỉ DDL.
  # - false: dev-style, kèm 5 default schedules + 3 legacy connections.
  # ENV override: CMS_MIGRATION_SKIP_SEEDS=true|false.
  skipSeeds: false
```

---

## T-10 — `internal/migrate/runner.go`

### T-10.1 — Signature

```go
// Run executes pending migrations.
//
// includeSeeds: false → chỉ apply file từ migrations.SchemaFiles.
//               true  → apply schema rồi tới seed (sort by basename).
//
// Idempotency: tracker cdc_system.schema_migrations skip applied files;
// advisory lock pg_advisory_lock(0x4344444D49475282042) prevents concurrent runs.
func Run(gdb *gorm.DB, includeSeeds bool, logger *zap.Logger) error {
    // ... existing body, gọi listMigrationFiles(includeSeeds)
}
```

### T-10.2 — `walkEmbed`

```go
import (
    "embed"
    "io/fs"
    "strings"
)

func walkEmbed(efs embed.FS, root string) ([]migrationFile, error) {
    var out []migrationFile
    err := fs.WalkDir(efs, root, func(p string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            return nil
        }
        if !strings.HasSuffix(p, ".sql") {
            return nil
        }
        content, rerr := efs.ReadFile(p)
        if rerr != nil {
            return rerr
        }
        out = append(out, migrationFile{path: p, content: content})
        return nil
    })
    return out, err
}
```

### T-10.3 — `listMigrationFiles`

```go
import "path"
import "sort"

func listMigrationFiles(includeSeeds bool) ([]migrationFile, error) {
    schemaList, err := walkEmbed(migrations.SchemaFiles, "schema")
    if err != nil {
        return nil, fmt.Errorf("walk schema embed: %w", err)
    }

    var combined []migrationFile
    combined = append(combined, schemaList...)

    if includeSeeds {
        seedList, err := walkEmbed(migrations.SeedFiles, "seed")
        if err != nil {
            return nil, fmt.Errorf("walk seed embed: %w", err)
        }
        combined = append(combined, seedList...)
    }

    sort.Slice(combined, func(i, j int) bool {
        return path.Base(combined[i].path) < path.Base(combined[j].path)
    })

    return combined, nil
}
```

---

## T-11 — `internal/server/server.go`

Line 63 area:

```go
// BEFORE
if err := migrate.Run(db, logger); err != nil {
    return fmt.Errorf("migrate: %w", err)
}

// AFTER
if err := migrate.Run(db, !cfg.Migration.SkipSeeds, logger); err != nil {
    return fmt.Errorf("migrate: %w", err)
}
```

---

## T-12 — README files

### T-12.1 — `migrations/README.md`

```markdown
# CDC CMS Service — Migrations

Layout:

```
migrations/
├── schema/          # DDL-only. Apply on EVERY boot regardless of env.
│   ├── core/        # Foundation tables (cdc_table_registry, mapping_rules).
│   ├── ids/         # Sonyflake ID generation (worker_registry, sequences).
│   ├── partitioning/ # Partitioned failed_sync_logs + cdc_activity_log.
│   ├── registry/    # V1 control plane (table_registry, mapping_rules, ...).
│   ├── worker/      # cdc_worker_schedule + transmute_schedule.
│   ├── recon_dlq/   # Reconciliation reports + DLQ.
│   ├── audit_security/ # admin_actions + cdc_alerts.
│   ├── cdc_system_model/ # V2 control plane (connection_registry, ...).
│   └── ops/         # Operational helpers (provisioning_log, cdc_jobs).
│
├── seed/            # Env-specific fixtures. Skipped when skipSeeds=true.
│   ├── 100_worker_schedules.sql       # 5 default schedule rows.
│   └── 101_v2_legacy_connections.sql  # 3 legacy_*_default connections.
│
├── cluster/         # NOT embedded. DBA runs manually with superuser.
│   ├── 001_roles.sql
│   └── 002_search_path.sql
│
└── archive/         # Frozen files. NOT embedded. Reference only.
    ├── 003_add_mapping_rule_status.sql  # → squashed into registry/013.
    ├── 004_bridge_columns.sql            # → squashed into registry/013.
    ├── 005_admin_actions.sql             # → audit_security/040.
    └── 013_alerts.sql                    # → audit_security/041.
```

## Lifecycle

1. **Service boot** (server.go) gọi `migrate.Run(db, includeSeeds, logger)`.
2. Runner pin `pg_advisory_lock(0x4344444D49475282042)` trên dedicated conn.
3. Runner walk `SchemaFiles` (embed `all:schema`) → sort by basename.
4. Nếu `includeSeeds=true` → walk thêm `SeedFiles` (embed `all:seed`).
5. Apply file chưa có trong tracker `cdc_system.schema_migrations`.
6. Re-run = no-op (tracker skip applied files).

## Config

```yaml
migration:
  skipSeeds: true   # production: true. local: false.
```

ENV override: `CMS_MIGRATION_SKIP_SEEDS=true|false`.

## Add new migration

1. Đặt tên: `NNN_<descriptor>.sql` (NNN = số tiếp theo trong chuỗi 053+, hoặc
   100+ nếu là seed).
2. Bỏ vào folder phù hợp dưới `schema/<group>/` hoặc `seed/`.
3. Header chuẩn (6 fields): xem template trong `schema/README.md`.
4. Idempotent patterns:
   - `CREATE TABLE IF NOT EXISTS`
   - `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`
   - `INSERT ... ON CONFLICT DO NOTHING` (hoặc `WHERE NOT EXISTS`).
5. Build local: `go build ./...` → verify embed pickup file mới.
6. Boot service local: log "applied_now=1" với basename file mới.

## Cluster (DBA manual)

```bash
psql -h <host> -U postgres -d cdc_dw < cluster/001_roles.sql
psql -h <host> -U postgres -d cdc_dw < cluster/002_search_path.sql
```

## Archive policy

File trong `archive/` không apply. Lý do archive + replacement ghi trong
`archive/README.md`.
```

### T-12.2 — `migrations/schema/README.md`

```markdown
# Schema — DDL-only migrations

Apply on every boot regardless of environment.

## Sub-folders

| Folder | Purpose |
|---|---|
| `core/` | Foundation tables (V1 `cdc_table_registry`, schema relocations). |
| `ids/` | Sonyflake ID generation (`cdc_internal.*`, `cdc_system.gen_sonyflake_id`). |
| `partitioning/` | Partitioned `failed_sync_logs` + `cdc_activity_log`. |
| `registry/` | V1 control plane (table_registry, mapping_rules, systematic_sources). |
| `worker/` | `cdc_worker_schedule` + `transmute_schedule`. |
| `recon_dlq/` | Reconciliation reports + DLQ tables. |
| `audit_security/` | `admin_actions` (partitioned) + `cdc_alerts`. |
| `cdc_system_model/` | V2 control plane (`connection_registry`, bindings, runtime state). |
| `ops/` | Operational helpers (`provisioning_log`, `cdc_jobs`). |

## Numbering history

Chuỗi 001-052 có 24 lỗ thủng do squash operations. Detail trong
`migrations/archive/README.md`.

Files đã được squash/move:
- 004-006, 009, 012, 014-017, 021, 024, 026, 028, 035, 039, 042-043, 045-047,
  049-051, 053.

## Dependency graph

```
core/001 (V1 base)
  ├→ core/002 (helper)
  ├→ ids/003 (V1.12)
  ├→ registry/013 (alter base)
  └→ worker/007 → seed/100

ids/018 (V1.25 foundation)
  ├→ registry/019, 020, 023, 025, 027
  └→ worker/022

audit_security/040, 041 (cdc_system layer)

cdc_system_model/029 (V2 base) → seed/101
  ├→ 030, 031, 032, 033, 034, 036

core/037, 038 (schema relocation public → cdc_system)
core/044 (cleanup)

ops/048, 052 (cross-cutting)
```

## Header convention

Mọi file `.sql` trong `schema/` PHẢI có header:

```sql
-- ==========================================================================
-- File:        <basename>.sql
-- Purpose:     <1 dòng mô tả>.
-- Schema:      <public | cdc_internal | cdc_system>.
-- Idempotent:  <IF NOT EXISTS / ON CONFLICT / WHERE NOT EXISTS>.
-- Depends on:  <comma-separated basenames, hoặc "none">.
-- Env scope:   schema-only.
-- ==========================================================================
```
```

### T-12.3 — `migrations/seed/README.md`

```markdown
# Seed — Env-specific fixtures

> **WARNING**: Files in this folder are SKIPPED on production deploys
> (`migration.skipSeeds=true`). Use only for dev/local/staging.

## Files

| File | Source | Purpose |
|---|---|---|
| `100_worker_schedules.sql` | Split from `schema/worker/007` (2026-05-15) | 5 default worker schedule rows. |
| `101_v2_legacy_connections.sql` | Split from `schema/cdc_system_model/029` (2026-05-15) | 3 legacy_*_default connection rows. |

## Numbering convention

- Schema files: `001-052` (current), reserve `053-099` for growth.
- Seed files: `100+` to avoid basename collision with schema tracker.

## How seed apply works

1. Service boot reads `cfg.Migration.SkipSeeds`.
2. Runner `migrate.Run(db, !cfg.Migration.SkipSeeds, logger)`.
3. If `includeSeeds=true`: runner walks `seed/*.sql` AFTER `schema/**/*.sql`,
   sorted by basename → `100_*` applies after `052_*`.
4. Tracker `cdc_system.schema_migrations` records `version='100_worker_schedules'`
   etc. after success.

## Re-enable seed on staging

```bash
CMS_MIGRATION_SKIP_SEEDS=false ./bin/cms-service
# Hoặc trong config-staging.yml:
migration:
  skipSeeds: false
```

## Why split out from schema?

- Schedule rows: runtime config, operator should configure via CMS UI.
- Legacy connections: bootstrap.EnsureDefaultShadowConnection (Go) handle
  production case using `cfg.ShadowDB` instead of SQL hardcoded values.

## Why enum_types NOT in seed?

`schema/registry/020_mapping_rule_jsonpath.sql` keeps INSERT enum_types because:
- Enums (payment_state, api_type, currency_iso) là **domain config bất biến**.
- Mapping rules reference các enum này; production cần.
- Phân loại: schema = DDL + domain config; seed = env-specific fixture.
```

### T-12.4 — `migrations/archive/README.md`

```markdown
# Archive — Frozen migrations

Files in this folder are **NOT embedded** and **NOT applied** by the runtime
migrator. They exist for historical reference only.

## Files

| File | Archived | Reason | Replacement |
|---|---|---|---|
| `003_add_mapping_rule_status.sql` | 2026-03 | Squashed | `schema/registry/013_table_registry_expected_fields.sql` |
| `004_bridge_columns.sql` | 2026-03 | Squashed | `schema/registry/013_table_registry_expected_fields.sql` |
| `005_admin_actions.sql` | 2026-04 | Schema layer change | `schema/audit_security/040_admin_actions_in_cdc_system.sql` |
| `013_alerts.sql` | 2026-04 | Schema layer change | `schema/audit_security/041_cdc_alerts_in_cdc_system.sql` |

## Other archived numbers (file not preserved)

| Number | Reason |
|---|---|
| 006 | Squashed into registry/013. |
| 009 | Reserved, never created. |
| 012 | Squashed into partitioning/010. |
| 014, 015, 016, 017 | Squashed into registry/013. |
| 021 | Squashed into registry/020. |
| 024 | Squashed into registry/019. |
| 026 | Reserved, never created. |
| 028 | Squashed into ids/018. |
| 035 | Squashed into cdc_system_model/029 → then seed split → `seed/101`. |
| 039 | Reserved, never created. |
| 042, 043 | Squashed into audit_security/040. |
| 045 | Squashed into partitioning/010. |
| 046 | Squashed into registry/013 + 020. |
| 047 | Squashed into cdc_system_model/030. |
| 049, 050, 051, 053 | Reserved, never created. |

## Policy

- Files archived = frozen. Do NOT modify.
- For new schema changes, create a new file with the next available number
  (currently `053+`) in `schema/<group>/`.
- Never delete archive files; they document migration history.
```

### T-12.5 — `migrations/cluster/README.md` (MOD: bỏ 005_pg_users)

Đọc file hiện tại, tìm reference `005_pg_users.sql`, xoá dòng đó.

### T-12.6 — `migrations/schema/cdc_system_model/README.md` (MOD)

Đọc file hiện tại, tìm:
- `028_sonyflake_fallback_fn.sql` → xoá dòng đó.
- `035_v2_backfill_legacy_registry.sql` → thay bằng note:
  > "(035 squashed into 029, then seed extracted to seed/101_v2_legacy_connections.sql)"

---

## T-13 — Verify

### T-13.1 — Build

```bash
cd cdc-cms-service
go build ./... && echo "build OK" || echo "build FAIL"
```

### T-13.2 — Vet

```bash
go vet ./... && echo "vet OK" || echo "vet FAIL"
```

### T-13.3 — Test

```bash
go test ./internal/migrate/... -v -count=1
```

### T-13.4 — Run local (skipSeeds=false)

```bash
make run  # hoặc: go run ./cmd/cms-service
```

Trong log lookup:
- `total_files=30`
- `applied_now=0` (DB local đã có 28 schema; cần thêm 100, 101 → applied_now=2 lần đầu).

### T-13.5 — Run với CMS_MIGRATION_SKIP_SEEDS=true

```bash
CMS_MIGRATION_SKIP_SEEDS=true go run ./cmd/cms-service
```

Lookup:
- `total_files=28`
- `applied_now=0`.

### T-13.6 — Health check

```bash
curl -fsS http://localhost:8083/health
```

### T-13.7 — Endpoint smoke

```bash
curl -fsS http://localhost:8083/api/v1/source-objects | jq .
```

---

## T-14 — Report

### T-14.2 — `migrations/report_refactor_2026-05-15.md`

Skeleton:

```markdown
# Migrations Refactor Report — 2026-05-15

## Summary
- Workspace: feature-cdc-cms-migrations-refactor-2026-05-15.
- Goal: layout chuyên nghiệp + production skipSeeds toggle.
- Status: DONE (post-verification).

## Before/After

### Before (2026-05-14)
```
migrations/
├── core/, ids/, partitioning/, registry/, worker/, recon_dlq/,
│   audit_security/, cdc_system_model/, ops/  (28 SQL files)
├── cluster/  (2 SQL, DBA manual)
├── .archive/  (4 SQL, frozen)
└── embed.go  (single `var Files`)
```

### After (2026-05-15)
```
migrations/
├── README.md          [NEW]
├── embed.go           [MOD: SchemaFiles + SeedFiles]
├── schema/            [9 group folders, 28 SQL]
├── seed/              [NEW: 100 + 101]
├── cluster/           [unchanged]
└── archive/           [renamed from .archive]
```

## Files changed (exact count)

(Run: `git diff --stat | wc -l`)

| Category | Count |
|---|---|
| Renamed | 32 |
| Modified | 8 |
| Created | 9 |
| Deleted | 0 |

## Line diff

(Run: `git diff --stat`)

```
<paste actual git output>
```

## Verification

| Step | Exit code | Notes |
|---|---|---|
| `go build ./...` | 0 | OK |
| `go vet ./...` | 0 | OK |
| `go test ./internal/migrate/...` | 0 | N/A or PASS |
| `make run` log | "applied_now=2 total=30" | DB ghi 100, 101 |
| Re-run log | "applied_now=0 total=30" | idempotent OK |
| `CMS_MIGRATION_SKIP_SEEDS=true` log | "applied_now=0 total=28" | seed skipped |
| `curl :8083/health` | 200 | OK |
| `curl :8083/api/v1/source-objects` | 200 | OK |
```

---

## T-15 — Security gate

```bash
# CLAUDE.md §8 mandate
/security-agent --scope=migrations
```
