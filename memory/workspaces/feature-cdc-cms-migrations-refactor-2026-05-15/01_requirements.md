# 01 — Requirements

> **Source**: User directive 2026-05-15 (admin@homeproxy.vn).

## REQ-1 — Refactor layout migrations/ phải "trực quan, chuyên nghiệp"

**Acceptance**:
- Cấu trúc folder phản ánh **trục chức năng** (schema vs seed vs cluster vs
  archive), KHÔNG trộn DDL với seed business data trong cùng file.
- Tên folder + filename phải self-documenting (đọc filename là biết purpose).
- Top-level `migrations/README.md` mô tả layout, lifecycle, runbook.
- Mỗi sub-folder có `README.md` liệt kê file + mục đích + depends_on.

## REQ-2 — "Gọn gàng, clear, đầy đủ, rõ ràng cho từng nhóm chức năng"

**Acceptance**:
- Mỗi file SQL có **header comment chuẩn** gồm:
  - `Purpose`: 1 dòng mô tả mục tiêu.
  - `Schema target`: schema mà file ghi vào (public / cdc_internal / cdc_system).
  - `Idempotent`: cách re-run an toàn (IF NOT EXISTS / ON CONFLICT).
  - `Depends on`: file phụ thuộc (nếu có).
  - `Env scope`: schema-only / seed (dev-only) / cluster (DBA manual).
- README drift hiện tại được sửa (cdc_system_model/README.md, cluster/README.md).
- `.archive/` phải có README giải thích lý do từng file bị archive +
  replacement (nếu có).

## REQ-3 — "Đáp ứng được chạy trên production với config riêng"

**Acceptance**:
- Có cơ chế **tách seed data ra khỏi schema migration**:
  - Folder `seed/` (mới) chứa các INSERT seed configurable (default schedule,
    enum types, legacy infrastructure connections).
  - Folder `schema/` chứa DDL-only (CREATE/ALTER + CREATE FUNCTION).
- Production config (`config-production.yml`) có toggle `migration.skipSeeds:
  true` để KHÔNG apply seed files.
- Local config (`config-local.yml`) toggle `migration.skipSeeds: false`
  (default) để dev có data demo.
- `migrate.Run()` đọc `cfg.Migration.SkipSeeds` qua signature mới
  `Run(db, cfg, logger)` (hoặc tương đương).
- ENV bind: `CMS_MIGRATION_SKIP_SEEDS=true` override yml.

## REQ-4 — "Theo hướng core systems, không cheat DB"

**Acceptance**:
- Refactor KHÔNG DROP/RESET DB.
- KHÔNG mutate tracker `cdc_system.schema_migrations` qua manual SQL.
- Local DB đã apply 28 file → giữ tracker compat: KHÔNG renumber, KHÔNG rename
  basename. Tracker version sẽ vẫn match sau refactor.
- Trên DB đã apply: re-run migrator là no-op (tracker skip).
- Trên fresh DB (production cold-boot): chạy DDL-only nếu `skipSeeds=true`,
  chạy DDL + seed nếu false.

## REQ-5 — "Report dựa trên kết quả tính toán thực tế"

**Acceptance**:
- Có file `report_migrations_refactor_2026-05-15.md` ghi:
  - Số file thay đổi (đếm thực tế qua `find`).
  - Diff line per file (đếm thực tế qua `git diff --stat` hoặc tương đương).
  - Verify build: `go build ./...` exit code thực tế.
  - Verify vet: `go vet ./...` exit code.
  - Verify test: `go test ./internal/migrate/...` (nếu có test).
  - Verify migrator no-op trên local DB: log thực tế từ `make run` boot.
- KHÔNG báo "DONE" nếu chưa run được service local sau refactor.

## REQ-6 — "Kết thúc luôn kiểm tra service work mới báo done"

**Acceptance**:
- Trước khi báo done:
  1. `go build ./...` PASS.
  2. `go vet ./...` PASS.
  3. Build binary, start local server, verify `migrate.Run` log "migrations
     done" + `applied_now=0` (no-op trên DB đã có 28 records).
  4. `curl localhost:8083/health` hoặc tương đương → 200.
  5. Verify 1 endpoint CRUD list (vd: `/api/v1/source-objects`) → không 500.
  6. Ghi result thực tế vào `06_validation.md`.

## REQ-7 — "Luôn có file report_*.md ghi lại"

**Acceptance**:
- File: `cdc-cms-service/migrations/report_refactor_2026-05-15.md` (trong repo)
  HOẶC `agent/memory/workspaces/.../report_migrations_refactor_2026-05-15.md`
  (trong agent memory).
- Nội dung gồm: before/after layout, list file thay đổi, diff summary,
  verification result.

## REQ-8 — Workspace governance (theo CLAUDE.md)

**Acceptance** (từ §7):
- Workspace `feature-cdc-cms-migrations-refactor-2026-05-15` được khởi tạo.
- Tài liệu bắt buộc: 00_context, 01_requirements, 02_plan, 03_implementation,
  05_progress (append-only), 06_validation, 08_tasks, 09_tasks_solution.
- Mọi quyết định kiến trúc → 04_decisions.md.
- Mọi thay đổi file → append `05_progress.md` cùng turn.

## Non-requirements (out of scope, ghi rõ để tránh nhầm)

- KHÔNG migrate sang công cụ ngoài (golang-migrate, atlas, flyway).
- KHÔNG implement down/rollback migrations.
- KHÔNG add CLI tool dry-run.
- KHÔNG fix V1 vs V2 dual coexistence.
- KHÔNG fix partition rotation TIME BOMB.
- KHÔNG đụng services khác trong monorepo.
