# 02 — Plan

> **Mode**: Plan-only (chưa execute). Đợi user approve trước khi code.

## A. Nguyên tắc dẫn đường

1. **Backward compat = ưu tiên #1**. Tracker `cdc_system.schema_migrations`
   trên DB local đã ghi 28 row theo basename. Mọi giải pháp KHÔNG được phá compat.
2. **Tách trục chức năng > Renumber**. User muốn "rõ ràng cho từng nhóm chức
   năng" → reorganize theo trục **mức độ idempotent + môi trường** (DDL-only
   chạy production / seed chạy dev-only / cluster chạy DBA manual /
   archive đóng băng), KHÔNG renumber.
3. **DDL ≠ seed**. Mọi `INSERT INTO ... VALUES (...)` cần audit có phải config
   bắt buộc cho production hay không. Nếu **dev demo** hoặc **dataset có thể
   miss trên prod** → tách ra `seed/`.
4. **Config-driven**. Toggle `migration.skipSeeds` đọc từ yml + env. Production
   yml mặc định `true`. Local yml mặc định `false`.
5. **No data destruction**. KHÔNG xoá tracker row, KHÔNG TRUNCATE bảng.
6. **Verify-before-done**. Build + vet + start service + check log + curl endpoint.

## B. Phân loại 3 INSERT seed (current state)

| Vị trí | Loại data | Dev cần? | Prod cần? | Quyết định |
|---|---|---|---|---|
| worker/007 — 5 default schedule rows (bridge/transform/field-scan/partition-check/airbyte-sync) | Schedule config | YES — dev cần để smoke pipeline | DEBATABLE — prod operator có thể tự config qua CMS UI sau cold-boot | **Tách → seed/**, prod skip mặc định, operator manual insert |
| registry/020 — 3 enum_types (payment_state, api_type, currency_iso) | Domain enum | YES — mapping rules reference các enum này | YES — cùng business domain | **GIỮ trong schema** (vì là config-like bắt buộc) |
| cdc_system_model/029 — 3 legacy_*_default connections (system/shadow/master) | Infrastructure config | YES — V2SyncCommand cần ≥1 row mỗi role_type | **NO** — prod đã có Go `bootstrap.EnsureDefaultShadowConnection` tự inject từ `cfg.ShadowDB` | **Tách → seed/**, prod skip (Go bootstrap thay thế) |

Logic phân loại:
- **Enum types** = domain knowledge bất biến → schema.
- **Schedule rows** = runtime config có thể khác giữa env → seed.
- **Legacy connections** = dev/local fixture → seed; prod dùng env-based bootstrap.

## C. Layout TARGET (proposed)

```
cdc-cms-service/migrations/
├── README.md                    [NEW] top-level: layout + lifecycle + runbook
├── embed.go                     [MOD] tách SchemaFS + SeedFS
│
├── schema/                      [NEW folder — MOVE from 9 sub-folders]
│   ├── README.md                [NEW] danh sách file, depends_on graph
│   ├── core/                    [MOVE từ migrations/core/]
│   │   ├── 001_init_schema.sql              [MOD bỏ seed block /* ... */ disabled]
│   │   ├── 002_standardize_schema.sql       [no change]
│   │   ├── 037_move_system_tables_to_cdc_system.sql  [no change]
│   │   ├── 038_finalize_cdc_system_namespace.sql     [no change]
│   │   └── 044_cleanup_public_residue.sql            [no change]
│   ├── ids/
│   │   ├── 003_sonyflake_schema.sql                  [no change]
│   │   └── 018_sonyflake_v125_foundation.sql         [no change]
│   ├── partitioning/
│   │   └── 010_partitioning.sql                      [no change]
│   ├── registry/
│   │   ├── 013_table_registry_expected_fields.sql    [no change]
│   │   ├── 019_system_registry.sql                   [no change]
│   │   ├── 020_mapping_rule_jsonpath.sql             [MOD: chuyển INSERT enum_types → seed]
│   │   ├── 023_master_table_registry.sql             [no change]
│   │   ├── 025_schema_proposal.sql                   [no change]
│   │   └── 027_systematic_sources.sql                [no change]
│   ├── worker/
│   │   ├── 007_worker_schedule.sql                   [MOD: chuyển INSERT schedule → seed]
│   │   └── 022_transmute_schedule.sql                [no change]
│   ├── recon_dlq/
│   │   ├── 008_reconciliation.sql                    [no change]
│   │   └── 011_recon_runs.sql                        [no change]
│   ├── audit_security/
│   │   ├── 040_admin_actions_in_cdc_system.sql       [no change]
│   │   └── 041_cdc_alerts_in_cdc_system.sql          [no change]
│   ├── cdc_system_model/
│   │   ├── 029_v2_connection_registry.sql            [MOD: chuyển INSERT 3 legacy_* → seed]
│   │   ├── 030_v2_source_object_registry.sql         [no change]
│   │   ├── 031_v2_shadow_binding.sql                 [no change]
│   │   ├── 032_v2_master_binding.sql                 [no change]
│   │   ├── 033_v2_mapping_rule.sql                   [no change]
│   │   ├── 034_v2_sync_runtime_state.sql             [no change]
│   │   └── 036_v2_transmute_schedule.sql             [no change — INSERT...SELECT migration data]
│   └── ops/
│       ├── 048_provisioning_log_cap_helper.sql       [no change]
│       └── 052_create_cdc_jobs.sql                   [no change]
│
├── seed/                        [NEW] dev-only data, prod skip via config
│   ├── README.md                [NEW] env-toggle explanation
│   ├── 100_worker_schedules.sql           [NEW from worker/007 INSERT block]
│   ├── 101_v2_legacy_connections.sql      [NEW from cdc_system_model/029 INSERT block]
│   └── (note: enum_types GIỮ trong schema/registry/020 vì là config-like)
│
├── cluster/                     [KEEP unchanged]
│   ├── README.md                [MOD: bỏ ref tới 005_pg_users.sql không tồn tại]
│   ├── 001_roles.sql            [no change]
│   └── 002_search_path.sql      [no change]
│
└── archive/                     [RENAME từ .archive/]
    ├── README.md                [NEW: explain lý do từng file bị archive + replacement]
    ├── 003_add_mapping_rule_status.sql    [moved]
    ├── 004_bridge_columns.sql              [moved]
    ├── 005_admin_actions.sql               [moved — replaced by audit_security/040]
    └── 013_alerts.sql                      [moved — replaced by audit_security/041]
```

### Note về numbering của seed files:
- Dùng prefix **100+** để (a) không trùng basename với bất kỳ schema file
  nào (cao nhất hiện tại = 052), (b) signal rõ "ngoài chuỗi schema chính".
- Tracker version = `100_worker_schedules` / `101_v2_legacy_connections` →
  unique, không collide.
- Khi `skipSeeds=true`, runtime sẽ KHÔNG walk vào seed/ → tracker không
  record 100/101 → next deploy nếu set lại `false` vẫn apply được.

### Note về tracker compat:
- Cho file MOVE (vd: `core/001_init_schema.sql` → `schema/core/001_init_schema.sql`):
  basename giữ nguyên = `001_init_schema`. Tracker đã có record này → skip.
- Cho file MOD (vd: bỏ INSERT block khỏi 007/020/029): basename giữ nguyên,
  tracker đã có → skip. **Hệ quả**: DB local đã chứa seed data, KHÔNG bị
  re-INSERT (idempotent IF NOT EXISTS dù sao cũng safe).
- Cho file NEW (100/101): basename mới, tracker chưa có → apply nếu
  `skipSeeds=false`. DB local đã có data sẵn → INSERT WHERE NOT EXISTS no-op.

## D. Config schema target

Thêm vào `config/config.go`:

```go
type MigrationConfig struct {
    SkipSeeds bool `mapstructure:"skipSeeds"`
}

type AppConfig struct {
    // ... existing fields
    Migration MigrationConfig `mapstructure:"migration"`
}
```

ENV bind: `CMS_MIGRATION_SKIP_SEEDS`.

`config-production.yml` THÊM:
```yaml
migration:
  skipSeeds: true   # production: DDL-only, no demo data
```

`config-local.yml` THÊM:
```yaml
migration:
  skipSeeds: false  # local: include seeds for smoke test
```

`config-sample.yml` THÊM (default false, doc-leaning):
```yaml
migration:
  # When true, runtime skips files in migrations/seed/.
  # Use true for production deploys; false for dev/staging where you want
  # default schedules and demo data.
  skipSeeds: false
```

## E. Runner contract change

Hiện tại: `migrate.Run(db *gorm.DB, logger *zap.Logger) error`.

Sau refactor: `migrate.Run(db *gorm.DB, opts Options, logger *zap.Logger) error`
với `Options{SkipSeeds bool}` — backward-compatible nếu caller pass
zero-value Options.

Hoặc đơn giản hơn (chosen): `migrate.Run(db, skipSeeds, logger)`.

Implementation:
- `embed.go` expose 2 FS: `SchemaFiles` (embed `schema/**/*.sql`) và
  `SeedFiles` (embed `seed/*.sql`).
- `listMigrationFiles(includeSeeds bool)` walk SchemaFiles luôn, walk
  SeedFiles nếu `includeSeeds`.
- Sort tổng hợp theo `path.Base()` — vì seed dùng prefix 100+, chúng sort
  sau schema 001-052. Hành vi: schema apply trước, seed apply sau (đúng
  thứ tự logic — seed reference schema tables).

## F. Embed pattern

Hiện tại:
```go
//go:embed core/*.sql ids/*.sql partitioning/*.sql registry/*.sql worker/*.sql recon_dlq/*.sql audit_security/*.sql cdc_system_model/*.sql ops/*.sql
var Files embed.FS
```

Sau refactor:
```go
//go:embed schema/core/*.sql schema/ids/*.sql schema/partitioning/*.sql schema/registry/*.sql schema/worker/*.sql schema/recon_dlq/*.sql schema/audit_security/*.sql schema/cdc_system_model/*.sql schema/ops/*.sql
var SchemaFiles embed.FS

//go:embed seed/*.sql
var SeedFiles embed.FS
```

Trade-off: vẫn phải maintain manually pattern selective vì go:embed không
support `**`. Có thể đơn giản hơn:
```go
//go:embed all:schema
var SchemaFiles embed.FS

//go:embed all:seed
var SeedFiles embed.FS
```
`all:` modifier kéo cả sub-folder. Cần lọc trong code: `if !strings.HasSuffix(p, ".sql")`.

→ **Chọn `all:` modifier** vì gọn và auto-include sub-folder mới (vd:
nếu thêm `schema/auth/` trong tương lai, không phải sửa embed.go).

## G. Phases & Steps

### Phase 1 — Prep (no code change)
- [x] Audit hiện tại (đã làm).
- [x] Khởi tạo workspace (đã làm).
- [x] Viết 00_context.md, 01_requirements.md, 02_plan.md (đang làm).

### Phase 2 — Document plan + decisions (no code change)
- [ ] 03_implementation.md: chi tiết kỹ thuật từng bước, SQL diff.
- [ ] 04_decisions.md: ADR cho mỗi quyết định (folder layout, seed split,
      config toggle, embed pattern).
- [ ] 08_tasks.md: checklist tasks executable.
- [ ] 09_tasks_solution.md: solution snippet cho từng task.
- [ ] **TRÌNH BÀY plan này cho user, đợi approve.**

### Phase 3 — Execute (sau khi user approve)
Mỗi bước = 1 commit logical, append `05_progress.md`.

#### Step 3.1 — Create folder skeleton
```bash
cd cdc-cms-service/migrations
mkdir -p schema seed archive
mv core ids partitioning registry worker recon_dlq audit_security cdc_system_model ops schema/
mv .archive/* archive/
rmdir .archive
```

#### Step 3.2 — Split seed blocks
- `schema/worker/007_worker_schedule.sql`: xoá block INSERT (line 30-36),
  giữ DDL + indexes + COMMIT.
- `schema/registry/020_mapping_rule_jsonpath.sql`: GIỮ INSERT enum_types
  (config-like), KHÔNG move.
- `schema/cdc_system_model/029_v2_connection_registry.sql`: xoá 3 INSERT
  legacy_*_default block (lines 51-99), giữ CREATE TABLE + indexes + COMMENT.

#### Step 3.3 — Create new seed files
- `seed/100_worker_schedules.sql`: chứa INSERT block từ 007 (5 row).
- `seed/101_v2_legacy_connections.sql`: chứa 3 INSERT block từ 029.

#### Step 3.4 — Update embed.go
- Tách SchemaFiles + SeedFiles.

#### Step 3.5 — Update config.go + yml
- Add `MigrationConfig` struct.
- Add ENV bind `CMS_MIGRATION_SKIP_SEEDS`.
- Update 3 yml (production: true, local: false, sample: false + comment).

#### Step 3.6 — Update runner.go
- Signature mới: `Run(db, skipSeeds, logger)`.
- Walk SchemaFiles luôn, walk SeedFiles nếu includeSeeds.

#### Step 3.7 — Update server.go
- Pass `cfg.Migration.SkipSeeds` vào `migrate.Run`.

#### Step 3.8 — Write READMEs
- Top-level `migrations/README.md`.
- `schema/README.md`.
- `seed/README.md`.
- `archive/README.md`.
- Update `cluster/README.md` (bỏ ref 005_pg_users).
- Update `schema/cdc_system_model/README.md` (bỏ ref 028, 035).

#### Step 3.9 — Verify
- `go build ./...` → exit 0.
- `go vet ./...` → exit 0.
- `go test ./internal/migrate/...` → PASS (nếu có).
- Start service local với `config-local.yml` (`skipSeeds=false`):
  - Log: `migrations done total_files=X applied_now=0` (X = 28 schema + 2 seed = 30).
  - Tracker: SELECT version FROM cdc_system.schema_migrations COUNT phải tăng 2
    (100, 101). Nếu fresh DB thì 30.
- Start service với env override `CMS_MIGRATION_SKIP_SEEDS=true`:
  - Log: total_files=28, applied_now=0 (đã apply trước đó).
  - Trên fresh DB: applied_now=28.
- curl health endpoint → 200.

#### Step 3.10 — Write report
- `migrations/report_refactor_2026-05-15.md` ghi before/after.
- Workspace report copy.

## H. Risk register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Move folder → embed.go path sai → build fail | Medium | High | Test build trước khi commit; có thể rollback bằng `git mv` ngược |
| Tracker collision basename (vd: 100_*.sql trùng cũ) | Low | High | Đã chọn prefix 100+ không trùng với 001-052 hiện tại |
| Production fresh deploy bị missing legacy connections | Medium | Medium | Go bootstrap.EnsureDefaultShadowConnection đã có sẵn (server.go:94) — verify run trước khi remove SQL seed |
| User config-local.yml không update → local quên seed | Low | Low | Doc + ENV override |
| Renaming `.archive/` → `archive/` break .gitignore patterns | Low | Low | Check .gitignore cdc-cms-service trước khi rename |
| Test suite reference path cũ | Low | Medium | grep + update test path nếu có |

## I. Decision points (chờ user)

1. **Có cần CLI tool dry-run (`make migrate-status`) không?** → Default: KHÔNG
   (out of scope user nói "chỉ làm đúng những gì được yêu cầu").
2. **Renumber file để fill 24 lỗ thủng?** → KHÔNG (sẽ phá tracker compat).
3. **Move enum_types ra seed?** → KHÔNG (là config domain bất biến, không
   phải dev demo).
4. **Add partition rotation job?** → KHÔNG (out of scope, defer).

## J. Definition of Done

Plan này được coi là **completed** khi:
1. User approve hoặc reject với feedback cụ thể.
2. Nếu approve → Muscle execute Phase 3, hoàn thành 06_validation + report.
3. Nếu reject → ghi feedback vào 04_decisions, re-plan.
