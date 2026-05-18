# 08 — Tasks (Executable Checklist)

> Granular task list. Mỗi task binary (done/not-done). Status update kèm
> entry trong `05_progress.md` cùng turn.

## Pre-flight (Phase 1 — done)

- [x] T-00.1 Đọc `agent/memory/global/lessons.md`.
- [x] T-00.2 Đọc `agent/GEMINI.md` + `CLAUDE.md`.
- [x] T-00.3 Audit hiện trạng `cdc-cms-service/migrations/`.
- [x] T-00.4 Khởi tạo workspace folder.
- [x] T-00.5 Viết `00_context.md`.
- [x] T-00.6 Viết `01_requirements.md`.

## Plan (Phase 2 — done)

- [x] T-01.1 Viết `02_plan.md` (layout target + phases).
- [x] T-01.2 Viết `03_implementation.md` (SQL diff + Go diff).
- [x] T-01.3 Viết `04_decisions.md` (12 ADR).
- [x] T-01.4 Viết `08_tasks.md` (file này).
- [x] T-01.5 Viết `09_tasks_solution.md`.
- [x] T-01.6 Append `05_progress.md` entry "Phase 2 done".

## Gate (sync point)

- [ ] T-02.1 **TRÌNH BÀY plan cho user (Vietnamese summary + key risks).**
- [ ] T-02.2 **Chờ user approve / reject.**
  - Approve → tiến T-03+.
  - Reject với feedback → re-plan, update 04_decisions.

## Execute (Phase 3 — pending user approval)

### Step 3.1 — Folder skeleton

- [ ] T-03.1 `mkdir -p schema seed archive` trong `cdc-cms-service/migrations`.
- [ ] T-03.2 `git mv` 9 folder vào `schema/`.
- [ ] T-03.3 `git mv .archive/*.sql archive/`.
- [ ] T-03.4 `git mv .archive/README.md archive/README.md` (nếu có).
- [ ] T-03.5 `rmdir .archive` (sau khi rỗng).
- [ ] T-03.6 `git status` verify rename detected.

### Step 3.2 — Split seed blocks

- [ ] T-04.1 `schema/worker/007_worker_schedule.sql`:
  - Xoá block `INSERT INTO cdc_worker_schedule (...) VALUES ... ON CONFLICT`.
  - Thêm header chuẩn 6-field.
- [ ] T-04.2 `schema/cdc_system_model/029_v2_connection_registry.sql`:
  - Xoá 3 block `INSERT INTO cdc_system.connection_registry (...) SELECT ... WHERE NOT EXISTS`.
  - Thêm header chuẩn 6-field.
- [ ] T-04.3 `schema/registry/020_mapping_rule_jsonpath.sql`:
  - GIỮ INSERT enum_types (không split).
  - Thêm header chuẩn 6-field.

### Step 3.3 — New seed files

- [ ] T-05.1 Tạo `migrations/seed/100_worker_schedules.sql` (5 INSERT từ 007).
- [ ] T-05.2 Tạo `migrations/seed/101_v2_legacy_connections.sql` (3 INSERT từ 029).
- [ ] T-05.3 Verify created_by tag = `'seed_100'` / `'seed_101'`.

### Step 3.4 — Add header chuẩn cho schema files còn lại

- [ ] T-06.1 Thêm header 6-field cho 25 SQL còn lại chưa có header chuẩn:
  - core/001, 002, 037, 038, 044
  - ids/003, 018
  - partitioning/010
  - registry/013, 019, 023, 025, 027
  - worker/022
  - recon_dlq/008, 011
  - audit_security/040, 041
  - cdc_system_model/030, 031, 032, 033, 034, 036
  - ops/048, 052

### Step 3.5 — Update `embed.go`

- [ ] T-07.1 Mở `cdc-cms-service/migrations/embed.go`.
- [ ] T-07.2 Replace single `Files embed.FS` thành 2 var:
  - `SchemaFiles embed.FS` với `//go:embed all:schema`.
  - `SeedFiles embed.FS` với `//go:embed all:seed`.

### Step 3.6 — Update `config/config.go`

- [ ] T-08.1 Thêm struct `MigrationConfig{ SkipSeeds bool }`.
- [ ] T-08.2 Thêm field `Migration MigrationConfig` vào `AppConfig`.
- [ ] T-08.3 Bind ENV `CMS_MIGRATION_SKIP_SEEDS`.
- [ ] T-08.4 Default `migration.skipSeeds = false`.

### Step 3.7 — Update yml configs

- [ ] T-09.1 `config-production.yml`: thêm `migration: { skipSeeds: true }`.
- [ ] T-09.2 `config-local.yml`: thêm `migration: { skipSeeds: false }`.
- [ ] T-09.3 `config-sample.yml`: thêm `migration: { skipSeeds: false }` + comment.

### Step 3.8 — Update `internal/migrate/runner.go`

- [ ] T-10.1 Đổi signature `Run(db, logger)` → `Run(db, includeSeeds, logger)`.
- [ ] T-10.2 Tạo function `walkEmbed(fs embed.FS, root string)`.
- [ ] T-10.3 Đổi `listMigrationFiles()` → `listMigrationFiles(includeSeeds bool)`.
- [ ] T-10.4 Sort by `path.Base()` để giữ thứ tự 001 < 052 < 100 < 101.

### Step 3.9 — Update `internal/server/server.go`

- [ ] T-11.1 Line 63: đổi `migrate.Run(db, logger)` → `migrate.Run(db, !cfg.Migration.SkipSeeds, logger)`.

### Step 3.10 — Write READMEs

- [ ] T-12.1 `migrations/README.md` (top-level NEW).
- [ ] T-12.2 `migrations/schema/README.md` (NEW).
- [ ] T-12.3 `migrations/seed/README.md` (NEW).
- [ ] T-12.4 `migrations/archive/README.md` (NEW).
- [ ] T-12.5 `migrations/cluster/README.md` (MOD: bỏ 005_pg_users ref).
- [ ] T-12.6 `migrations/schema/cdc_system_model/README.md` (MOD: bỏ 028/035 ref).

## Verify (Phase 4)

- [ ] T-13.1 `cd cdc-cms-service && go build ./...` → exit 0.
- [ ] T-13.2 `go vet ./...` → exit 0.
- [ ] T-13.3 `go test ./internal/migrate/...` → PASS (nếu test tồn tại).
- [ ] T-13.4 `make run` với `config-local.yml`:
  - Log "migrate.done applied_now=2 total=30" (lần đầu).
  - Log "migrate.done applied_now=0 total=30" (lần thứ 2).
- [ ] T-13.5 `CMS_MIGRATION_SKIP_SEEDS=true ./bin/cms-service`:
  - Log "migrate.done applied_now=0 total=28".
- [ ] T-13.6 `curl localhost:8083/health` → 200.
- [ ] T-13.7 `curl localhost:8083/api/v1/source-objects` → 200 (không 500).

## Report (Phase 5)

- [ ] T-14.1 Viết `06_validation.md` với exit codes thực tế + log snippet.
- [ ] T-14.2 Viết `migrations/report_refactor_2026-05-15.md`:
  - Before/after layout (tree -L 2 output thực).
  - List file thay đổi (git diff --stat).
  - Diff line count (thực tế).
  - Verify result (exit codes thực).
- [ ] T-14.3 Update `agent/memory/global/active_plans.md` (đánh dấu workspace
  complete).
- [ ] T-14.4 Append `05_progress.md` entry "Phase 5 done — service verified".

## Final gate

- [ ] T-15.1 Chạy `/security-agent` review (CLAUDE.md §8).
- [ ] T-15.2 Báo cáo user: kèm path file report + key metrics + exit codes.

## Total: 60 tasks (10 done + 50 pending)

**Estimated effort**: 2-4 giờ cho Phase 3 + Phase 4 (giả sử không gặp build
error hoặc test reference path cũ).

**Critical path**:
T-02 (approval) → T-03 → T-04 → T-05 → T-07 → T-08 → T-09 → T-10 → T-11
→ T-13 (verify) → T-14 (report).
