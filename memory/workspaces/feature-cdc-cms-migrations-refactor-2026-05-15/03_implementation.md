# 03 — Implementation Details

> **Mode**: Spec-only. Mô tả chi tiết kỹ thuật, KHÔNG phải execution log.
> Execution log nằm ở `05_progress.md`. Sẵn sàng cho Muscle implement sau khi
> user approve plan (02_plan.md §J).

## 1. Filesystem operations

### 1.1 Folder creation + move

```bash
cd cdc-cms-service/migrations

# 1. Tạo skeleton mới (idempotent)
mkdir -p schema seed archive

# 2. Move 9 functional groups vào schema/
for d in core ids partitioning registry worker recon_dlq audit_security cdc_system_model ops; do
  [ -d "$d" ] && git mv "$d" "schema/$d"
done

# 3. Rename .archive → archive (dot-prefix gây embed.go ignore)
if [ -d ".archive" ]; then
  for f in .archive/*.sql; do
    [ -e "$f" ] && git mv "$f" "archive/$(basename "$f")"
  done
  [ -f ".archive/README.md" ] && git mv ".archive/README.md" "archive/README.md"
  rmdir .archive
fi
```

Note: dùng `git mv` để Git track rename → diff hiển thị "renamed", không phải "deleted + added".

### 1.2 Pre-flight checks trước khi move

```bash
# Bảo đảm chưa có folder schema/ tồn tại từ trước:
[ ! -d schema ] || { echo "schema/ already exists"; exit 1; }

# Bảo đảm tất cả 9 folders nguồn tồn tại:
for d in core ids partitioning registry worker recon_dlq audit_security cdc_system_model ops; do
  [ -d "$d" ] || { echo "missing: $d"; exit 1; }
done
```

## 2. SQL file modifications

### 2.1 `schema/worker/007_worker_schedule.sql`

**Trước (line 30-end)**: chứa DDL `CREATE TABLE cdc_worker_schedule` + 5 INSERT.

**Sau**: bỏ block `INSERT`. Còn lại DDL + indexes + COMMENT.

Header chuẩn (REQ-2):
```sql
-- ==========================================================================
-- File:        007_worker_schedule.sql
-- Purpose:     Tạo bảng cdc_worker_schedule (config-driven runner schedule).
-- Schema:      public (will be moved to cdc_system by 037).
-- Idempotent:  CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS.
-- Depends on:  001_init_schema, 002_standardize_schema.
-- Env scope:   schema-only. Default 5 schedule rows moved to
--              seed/100_worker_schedules.sql (apply when skipSeeds=false).
-- ==========================================================================
```

Block bị bỏ (chuyển sang seed/100):
```sql
INSERT INTO cdc_worker_schedule (job_name, interval_seconds, next_run_at)
VALUES
  ('bridge', 60, NOW()),
  ('transform', 90, NOW()),
  ('field-scan', 120, NOW()),
  ('partition-check', 300, NOW()),
  ('airbyte-sync', 180, NOW())
ON CONFLICT (job_name) DO NOTHING;
```

### 2.2 `schema/cdc_system_model/029_v2_connection_registry.sql`

**Trước**: DDL + 3 INSERT `WHERE NOT EXISTS` cho `legacy_system_db`,
`legacy_shadow_default`, `legacy_master_default` (lines 51-99).

**Sau**: chỉ DDL. Header chuẩn:
```sql
-- ==========================================================================
-- File:        029_v2_connection_registry.sql
-- Purpose:     V2 control plane — cdc_system.connection_registry catalog.
-- Schema:      cdc_system (created here if missing).
-- Idempotent:  CREATE SCHEMA IF NOT EXISTS / CREATE TABLE IF NOT EXISTS.
-- Depends on:  none (foundation of V2 schema set).
-- Env scope:   schema-only. Legacy infra seed moved to
--              seed/101_v2_legacy_connections.sql.
--              Production deploys MUST rely on bootstrap.EnsureDefaultShadowConnection
--              (server.go) thay vì SQL seed.
-- ==========================================================================
```

Squash note (DỜI từ phần seed đã bỏ vào header comment):
```sql
-- SQUASH HISTORY:
--   2026-05-14 — Absorbed 3 INSERT seeds từ migration 035 (now archived).
--   2026-05-15 — Split INSERT seeds → seed/101 vì production không nên depend
--                on SQL-level seed. bootstrap.EnsureDefaultShadowConnection (Go)
--                thay thế cho `legacy_shadow_default` trên production.
```

### 2.3 `schema/registry/020_mapping_rule_jsonpath.sql`

**KHÔNG đổi**: enum_types INSERT (payment_state / api_type / currency_iso) là
domain config bất biến, GIỮ trong schema. Chỉ thêm header chuẩn.

```sql
-- ==========================================================================
-- File:        020_mapping_rule_jsonpath.sql
-- Purpose:     ALTER cdc_mapping_rules + CREATE TABLE cdc_enum_types + seed 3
--              domain enums (payment_state, api_type, currency_iso).
-- Schema:      public.
-- Idempotent:  ALTER ... ADD COLUMN IF NOT EXISTS / INSERT ON CONFLICT.
-- Depends on:  001_init_schema (cdc_mapping_rules base table).
-- Env scope:   schema-only — enums là domain config, áp dụng MỌI env.
-- ==========================================================================
```

### 2.4 Header chuẩn cho 25 file còn lại

Pattern áp dụng tất cả file SQL trong `schema/`:

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

**Header này CHỈ thêm vào file nào chưa có**. Không touch file đã có header
detail (ví dụ 010, 037, 038).

## 3. New seed files

### 3.1 `seed/100_worker_schedules.sql`

```sql
-- ==========================================================================
-- File:        100_worker_schedules.sql
-- Purpose:     Seed 5 default worker schedule rows (bridge/transform/
--              field-scan/partition-check/airbyte-sync).
-- Schema:      cdc_system (re-located by 037).
-- Idempotent:  INSERT ... ON CONFLICT (job_name) DO NOTHING.
-- Depends on:  schema/worker/007_worker_schedule.sql,
--              schema/core/037_move_system_tables_to_cdc_system.sql.
-- Env scope:   seed (dev/staging). Skipped when CMS_MIGRATION_SKIP_SEEDS=true.
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

**Note schema-qualify**: vì migration 037 đã SET SCHEMA `cdc_worker_schedule` từ
`public` sang `cdc_system`. File seed này chạy SAU 037 → table nằm trong
`cdc_system`. Sort-by-basename: `100_*` > `037_*` → đúng thứ tự.

### 3.2 `seed/101_v2_legacy_connections.sql`

```sql
-- ==========================================================================
-- File:        101_v2_legacy_connections.sql
-- Purpose:     Seed 3 legacy infrastructure connections (system/shadow/master)
--              for V2 control plane. Local-only fixture; production uses
--              bootstrap.EnsureDefaultShadowConnection (Go).
-- Schema:      cdc_system.
-- Idempotent:  INSERT ... WHERE NOT EXISTS.
-- Depends on:  schema/cdc_system_model/029_v2_connection_registry.sql.
-- Env scope:   seed (dev/staging). Skipped when CMS_MIGRATION_SKIP_SEEDS=true.
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

Note: `created_by` đổi từ `'migration_029'` → `'seed_101'` để audit trail rõ
source (file seed/101 sinh row, không phải migration 029).

## 4. Go code changes

### 4.1 `migrations/embed.go`

Trước (1 var):
```go
package migrations

import "embed"

//go:embed core/*.sql ids/*.sql partitioning/*.sql registry/*.sql worker/*.sql recon_dlq/*.sql audit_security/*.sql cdc_system_model/*.sql ops/*.sql
var Files embed.FS
```

Sau (2 var):
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

`all:` prefix tells go:embed to include sub-folders, including files with
leading `_` or `.`. Cluster/ và archive/ KHÔNG embed (vì không nằm trong
`schema/` hay `seed/`).

### 4.2 `config/config.go`

Thêm struct + field:
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
    // ... existing fields
    Migration MigrationConfig `mapstructure:"migration"`
}
```

Trong `Load()` hoặc `bindEnv()`:
```go
v.BindEnv("migration.skipSeeds", "CMS_MIGRATION_SKIP_SEEDS")
v.SetDefault("migration.skipSeeds", false)  // local-friendly default
```

### 4.3 `config/config-production.yml`

Thêm cuối file:
```yaml
migration:
  # Production: chỉ apply DDL, KHÔNG apply seed dev/local fixtures.
  # Legacy infrastructure rows được seed bởi bootstrap.EnsureDefaultShadowConnection (Go).
  skipSeeds: true
```

### 4.4 `config/config-local.yml`

Thêm cuối file:
```yaml
migration:
  # Local dev: apply cả schema + seed để có default worker schedules
  # và legacy connection fixtures.
  skipSeeds: false
```

### 4.5 `config/config-sample.yml`

Thêm cuối file:
```yaml
migration:
  # Toggle áp dụng file trong migrations/seed/.
  # - true:  production-style, chỉ DDL.
  # - false: dev-style, kèm 5 default schedules + 3 legacy connections.
  # ENV override: CMS_MIGRATION_SKIP_SEEDS=true|false.
  skipSeeds: false
```

### 4.6 `internal/migrate/runner.go`

Signature đổi:
```go
// Run executes pending migrations.
//
// includeSeeds: false → chỉ apply file từ migrations.SchemaFiles.
//               true  → apply schema rồi tới seed (sort by basename).
//
// Idempotency: tracker cdc_system.schema_migrations skip applied files;
// advisory lock prevents concurrent runs.
func Run(gdb *gorm.DB, includeSeeds bool, logger *zap.Logger) error {
    // ... existing body
}
```

Hàm `listMigrationFiles` đổi:
```go
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

func walkEmbed(fs embed.FS, root string) ([]migrationFile, error) {
    var out []migrationFile
    err := fs.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            return nil
        }
        if !strings.HasSuffix(p, ".sql") {
            return nil
        }
        content, rerr := fs.ReadFile(p)
        if rerr != nil {
            return rerr
        }
        out = append(out, migrationFile{path: p, content: content})
        return nil
    })
    return out, err
}
```

### 4.7 `internal/server/server.go` — line 63

Trước:
```go
if err := migrate.Run(db, logger); err != nil {
    return fmt.Errorf("migrate: %w", err)
}
```

Sau:
```go
if err := migrate.Run(db, !cfg.Migration.SkipSeeds, logger); err != nil {
    return fmt.Errorf("migrate: %w", err)
}
```

Note: `!cfg.Migration.SkipSeeds` chuyển từ "skip" → "include" cho rõ contract
nội bộ. Hoặc đổi tên cho khớp:
```go
if err := migrate.Run(db, cfg.Migration.SkipSeeds, logger); err != nil {
```
và đổi tham số runner thành `skipSeeds bool`. Chọn 1 trong 2 — đề xuất giữ
`includeSeeds` để runner internal đọc dương (positive) thay vì negate.

## 5. README files

### 5.1 `migrations/README.md` (top-level — NEW)

Outline:
- Mục đích folder.
- Layout (`schema/`, `seed/`, `cluster/`, `archive/`).
- Lifecycle: runtime migrator chạy `schema/**` (always), `seed/*` (toggle).
- Cách re-run idempotent (advisory lock + tracker).
- Cách thêm migration mới (numbering convention, header comment, idempotent
  patterns).
- Cluster manual run (DBA superuser command).
- Archive policy.
- Toggle `skipSeeds`: local=false, prod=true.

### 5.2 `migrations/schema/README.md` (NEW)

Outline:
- Tóm tắt 9 sub-folder (mỗi folder 1 dòng).
- Dependency graph (text-based, vd: `core/001 → ids/003 → registry/013 → ...`).
- Numbering history note: số 004-006, 009, ... bị skip do squash; tra
  `archive/README.md` để biết file gốc.

### 5.3 `migrations/seed/README.md` (NEW)

Outline:
- Cảnh báo: file ở đây KHÔNG chạy trên production (skipSeeds=true).
- Numbering convention: 100+ để tách bạch với schema (001-052).
- Mỗi file 1 dòng giải thích nguồn gốc (vd: split từ worker/007).
- Cách re-enable cho staging: ENV `CMS_MIGRATION_SKIP_SEEDS=false`.

### 5.4 `migrations/archive/README.md` (NEW)

Outline (table):
| File | Archived date | Reason | Replacement |
|---|---|---|---|
| 003_add_mapping_rule_status.sql | <date> | Squashed | registry/013_table_registry_expected_fields.sql |
| 004_bridge_columns.sql | <date> | Squashed | registry/013 |
| 005_admin_actions.sql | <date> | Replaced | audit_security/040_admin_actions_in_cdc_system.sql |
| 013_alerts.sql | <date> | Replaced | audit_security/041_cdc_alerts_in_cdc_system.sql |

### 5.5 `migrations/cluster/README.md` (MOD)

Fix: bỏ reference tới `005_pg_users.sql` (file không tồn tại).

### 5.6 `migrations/schema/cdc_system_model/README.md` (MOD)

Fix: bỏ reference tới `028_sonyflake_fallback_fn.sql` và
`035_v2_backfill_legacy_registry.sql` (squashed). Note rằng 035's seed đã
được move sang `seed/101_v2_legacy_connections.sql`.

## 6. Test plan

### 6.1 Static checks

```bash
cd cdc-cms-service
go build ./...    # expect exit 0
go vet ./...      # expect exit 0
```

### 6.2 Unit tests (nếu có)

```bash
go test ./internal/migrate/... -v -count=1
```

Cần verify nếu tests reference old path. Nếu có:
- Update path từ `migrations/core/001_*.sql` → `migrations/schema/core/001_*.sql`.

### 6.3 Integration — local DB (28 records sẵn)

Trước khi chạy:
```bash
docker compose ps  # postgres cdc_dw đang up
psql -h localhost -p 5433 -U postgres -d cdc_dw \
  -c "SELECT version FROM cdc_system.schema_migrations ORDER BY version;"
```
Phải thấy 28 records: 001, 002, 003, 007, 008, 010, 011, 013, 018, 019, 020,
022, 023, 025, 027, 029, 030, 031, 032, 033, 034, 036, 037, 038, 040, 041,
044, 048, 052.

Run service local:
```bash
make run  # hoặc: go run ./cmd/cms-service
```

Expected log (skipSeeds=false default):
```
INFO migrate.start total_files=30
INFO migrate.tracker_existing count=28
INFO migrate.apply file=seed/100_worker_schedules.sql
INFO migrate.apply file=seed/101_v2_legacy_connections.sql
INFO migrate.done applied_now=2 total=30
```

Sau khi 100/101 đã apply 1 lần, re-run:
```
INFO migrate.done applied_now=0 total=30
```

### 6.4 Integration — production-like (skipSeeds=true)

```bash
CMS_MIGRATION_SKIP_SEEDS=true ./bin/cms-service
```

Expected log:
```
INFO migrate.start total_files=28
INFO migrate.done applied_now=0 total=28
```

KHÔNG được apply 100/101.

### 6.5 Endpoint smoke

```bash
curl -fsS http://localhost:8083/health
# Expected: {"status":"ok"} hoặc 200 với JSON body.

curl -fsS http://localhost:8083/api/v1/source-objects
# Expected: 200 với JSON array (có thể empty).
```

## 7. Rollback plan

Nếu Phase 3 fail bất kỳ bước:

1. **Git rollback**: `git restore --staged --worktree .` + clean folder mới.
2. **DB rollback**: KHÔNG cần. Tracker đã ghi 28 row + idempotent SQL đảm
   bảo re-apply old layout là no-op.
3. **Phụ lục: nếu lỡ apply seed 100/101 nhưng muốn revert**:
   ```sql
   DELETE FROM cdc_system.schema_migrations
   WHERE version IN ('100_worker_schedules', '101_v2_legacy_connections');
   -- Data rows trong cdc_worker_schedule / connection_registry GIỮ NGUYÊN
   -- (idempotent INSERT ON CONFLICT nên next apply là no-op).
   ```

## 8. Diff size estimate

| File category | Files affected | Lines changed (est) |
|---|---|---|
| Folder rename (git mv) | 28 schema + 4 archive = 32 | rename-only (0 content change tracked) |
| SQL header chuẩn (add) | ~25 files | +8 lines each = +200 |
| SQL split seed | 3 files (007, 029, [020 keep]) | -20 / -55 / 0 |
| New seed SQL | 2 files (100, 101) | +20 / +75 |
| README new | 4 files (top, schema, seed, archive) | +50 / +40 / +30 / +20 |
| README fix | 2 files (cluster, cdc_system_model) | ±5 each |
| embed.go | 1 | -1 line, +6 lines |
| config.go | 1 | +10 lines |
| yml config | 3 (prod/local/sample) | +3-6 lines each |
| runner.go | 1 | +20 lines (split walk function) |
| server.go | 1 | +1 line (arg pass) |
| **Total est** | **~50 files** | **~+450 / -75 = +375 net** |

## 9. Acceptance gates

Per `01_requirements.md`:

- [ ] REQ-1 layout chuyên nghiệp → schema/seed/cluster/archive folders + READMEs.
- [ ] REQ-2 header chuẩn → 25 SQL files có 6-field header.
- [ ] REQ-3 production config → MigrationConfig + 3 yml update + ENV bind.
- [ ] REQ-4 không cheat DB → tracker compat verify, basename preserve.
- [ ] REQ-5 report thực tế → `report_*.md` có exit codes + log snippets thực.
- [ ] REQ-6 verify service work → build/vet/start/curl PASS.
- [ ] REQ-7 file report tồn tại → physical file.
- [ ] REQ-8 workspace governance → đủ bộ 00→09 + 05_progress append.
