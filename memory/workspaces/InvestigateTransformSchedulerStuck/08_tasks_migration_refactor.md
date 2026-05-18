# 08 — Tasks: Migration System Refactor

**Phase**: `migration_refactor`
**Workspace**: `InvestigateTransformSchedulerStuck`
**Date**: 2026-05-14 15:32 (Asia/Ho_Chi_Minh)
**Status**: Draft — chờ duyệt cùng `02_plan_migration_refactor.md`

> Convention: mỗi task = 1 hàng checklist. Status: ⬜ TODO / 🟡 IN-PROGRESS / ✅ DONE / ❌ BLOCKED.

---

## Phase 1 — Unblock prod (NO CODE CHANGE)

### Pre-condition
- User trả lời Q1 (DBA có superuser).
- User chỉ định DBA window thực thi.

### Tasks

| # | Task | Owner | Status | Acceptance |
|---|---|---|---|---|
| 1.1 | Viết `cdc-cms-service/scripts/bootstrap_cms_db.sql` với 3 block (schema+tracker / INSERT 3 version / GRANT min) | Muscle | ⬜ | File pass `psql --dry-run` syntax check |
| 1.2 | Viết `cdc-cms-service/scripts/bootstrap_cms_db.md` runbook | Muscle | ⬜ | Có Pre-condition / Run command / Verify query / Rollback |
| 1.3 | Tự test script trên DB throwaway (vd tạo `cdc_cms_test` ở `gpay-postgres-cdc` local) | Muscle | ⬜ | `psql -v ON_ERROR_STOP=1 -1 -f bootstrap_cms_db.sql` exit 0; verify query hiện 3 row tracker |
| 1.4 | Đẩy script cho DBA (commit trên branch riêng, không merge ngay) | Muscle | ⬜ | Branch `migration-bootstrap-script` có commit duy nhất với script + runbook |
| 1.5 | DBA chạy bootstrap trên prod `cdc_cms_database` | DBA | ⬜ | psql exit 0; tracker có 3 row |
| 1.6 | User restart CMS prod | User | ⬜ | Log: `migrations done applied_now=N` không có fatal |
| 1.7 | Smoke test CMS prod `/health` `/ready` | Muscle | ⬜ | HTTP 200 |
| 1.8 | APPEND `05_progress.md` evidence | Muscle | ⬜ | Có timestamp + service uptime + tracker count |

### Rollback path
- `DELETE FROM cdc_system.schema_migrations WHERE applied_at >= '<bootstrap_time>'` (chỉ row do bootstrap insert).
- DBA REVOKE quyền vừa cấp nếu intent thay đổi.

---

## Phase 2 — Code refactor

### Pre-condition
- Phase 1 DONE.
- User trả lời Q3 (deploy channel).

### Tasks

| # | Task | Owner | Status | Acceptance |
|---|---|---|---|---|
| 2.1 | Tạo `cdc-cms-service/internal/migrate/skip_list.go` với `var ClusterBootstrap = map[string]bool{...}` cho 3 version | Muscle | ⬜ | `go vet` clean |
| 2.2 | Sửa `runner.go`: load skip-list, branch xử lý trong `Run` loop, log warn 1 lần / startup | Muscle | ⬜ | Diff < 40 dòng; không refactor unrelated code (CLAUDE.md §6 minimal impact) |
| 2.3 | Viết unit test `runner_test.go`: case skip-list hit, miss, đã applied, chưa applied | Muscle | ⬜ | 4 test pass; coverage `runner.go` ≥ 85% |
| 2.4 | Tạo migration mới `054_tracker_applied_by.sql` (L3 portable, `ADD COLUMN IF NOT EXISTS applied_by TEXT`) | Muscle | ⬜ | Apply ok trên dev `cdc_dw` + giả lập `cdc_cms_test` |
| 2.5 | Tạo thư mục `cdc-cms-service/migrations/cluster_bootstrap/` + 2 file SQL parameterize + README | Muscle | ⬜ | Test: `psql -v worker_password=test123 -f 001_pg_roles.sql` chạy được trên DB sạch |
| 2.6 | Update `migrations/embed.go` đảm bảo KHÔNG embed `cluster_bootstrap/` (kiểm tra: `go run -tags=embedcheck`) | Muscle | ⬜ | Build artifact không chứa byte của 2 file mới |
| 2.7 | Build binary CMS local → restart → log `applied_now=0` đối với 3 version skip | Muscle | ⬜ | Worker `cdc_dw` không thấy thay đổi |
| 2.8 | Deploy staging (hoặc env giả prod) → verify | Muscle + User | ⬜ | Như Phase 1.6-1.7 nhưng KHÔNG cần Phase 1 SQL thủ công |
| 2.9 | Deploy prod canary | User | ⬜ | 10' soak không tăng error rate |
| 2.10 | Deploy prod 100% | User | ⬜ | CMS healthy 30' continuous |
| 2.11 | APPEND `05_progress.md` | Muscle | ⬜ | Mỗi deploy step 1 dòng evidence |

### Rollback path
- `git revert <commit-2.1..2.6>`.
- `DELETE FROM cdc_system.schema_migrations WHERE version = '054_tracker_applied_by'` (nếu chưa code ref dùng column → drop column an toàn).

---

## Phase 3 — Secret hygiene + docs

### Pre-condition
- User trả lời Q4 (cho rotate password) + Q5 (secret manager).

### Tasks

| # | Task | Owner | Status | Acceptance |
|---|---|---|---|---|
| 3.1 | Generate 3 password mới (cdc_worker, cms_service, cdc_readonly) qua secret manager | DBA | ⬜ | 3 secret entry mới active |
| 3.2 | DBA `ALTER ROLE cdc_worker WITH PASSWORD '<new>'` cho 3 role trên `cdc_dw` | DBA | ⬜ | `\du` xác nhận, kiểm chứng qua `pg_authid` |
| 3.3 | Update worker config (`centralized-data-service/config-local.yml` hoặc env var deploy) | Muscle | ⬜ | Worker rolling restart không drop connection |
| 3.4 | Update CMS config | Muscle | ⬜ | CMS rolling restart không drop connection |
| 3.5 | Thêm comment `-- DEPRECATED: see migrations/cluster_bootstrap/001_pg_roles.sql` vào dòng 1 của `005_pg_users.sql` cũ. **KHÔNG sửa body.** | Muscle | ⬜ | Tracker checksum giữ nguyên (nếu engine kiểm checksum); body sau dòng comment không đổi byte |
| 3.6 | Cùng comment cho `039_set_search_path.sql` + `042_search_path_with_auth.sql` | Muscle | ⬜ | Như 3.5 |
| 3.7 | Viết `cdc-cms-service/docs/migrations.md` (3 layer model, runbook, rotate flow, skip-list rationale) | Muscle | ⬜ | Review pass (Brain hoặc Staff Engineer) |
| 3.8 | APPEND lesson vào `agent/memory/global/lessons.md` (Global Pattern theo CLAUDE.md §13) | Muscle | ⬜ | Format `Global Pattern [A does B to X] → Result Y. Đúng: ...` |
| 3.9 | Update `agent/memory/global/active_plans.md` + `project_context.md` | Muscle | ⬜ | Có entry cho phase mới |

### Rollback path
- Revert password = ALTER ROLE WITH PASSWORD `<old>` (DBA giữ old password trong escrow 7 ngày).
- Revert comment changes = `git revert`.

---

## Phase 4 — Security + final verification

### Tasks

| # | Task | Owner | Status | Acceptance |
|---|---|---|---|---|
| 4.1 | Chạy `/security-agent` rà soát code change Phase 2 + 3 | Muscle | ⬜ | Output: 0 critical / 0 high |
| 4.2 | Smoke test CMS prod 4 endpoint chính (`/health`, `/ready`, source-objects/registry, jobs/:id) | Muscle | ⬜ | 4/4 HTTP 200 |
| 4.3 | Smoke test worker (`/health`, `/ready`, `/api/v1/internal/stats`) + scheduler gap | Muscle | ⬜ | Gap 60s ± 50ms qua 5 tick (xem report 1121) |
| 4.4 | Tracker integrity query: `SELECT COUNT(*) WHERE applied_by='cluster-bootstrap' = 3 trên cdc_cms_database, 0 trên cdc_dw` | Muscle | ⬜ | Đúng số liệu kỳ vọng |
| 4.5 | Viết `07_status_report_migration_refactor.md` final | Muscle | ⬜ | Có evidence runtime + tracker count + endpoint check |
| 4.6 | Tạo `report_*.md` tóm tắt session cuối + APPEND `05_progress.md` | Muscle | ⬜ | Có timestamp ICT + Owner + skill list |
| 4.7 | Update `active_plans.md` chuyển workspace status sang ✅ Done | Muscle | ⬜ | Status row update |

---

## Cross-checklist (CLAUDE.md compliance)

- [ ] §3 Plan & Verify: mỗi phase có acceptance + verification thực tế (không suy diễn).
- [ ] §6 Simplicity First: skip-list 3 entries thay vì viết DSL phức tạp.
- [ ] §7 Doc Set: 01_req + 02_plan + 08_tasks + 09_tasks_solution + 03_implementation + 05_progress (append).
- [ ] §10 Security Auto-Check: Phase 4 gate.
- [ ] §11 Memory Append-only: 05_progress + lessons APPEND.
- [ ] §12 Brain Code Prohibition: Brain chỉ document; Muscle thực thi sau approve.
- [ ] §13 Lesson Global Pattern: dùng biến A/B/X/Y.
- [ ] §14 Governance Pre-flight: trước close phase chạy lại quét rules.

---

## Mapping task ↔ AC (Acceptance Criteria của `01_requirements`)

| AC | Task verify |
|---|---|
| AC1 CMS prod start ok | 1.6, 1.7, 2.9, 2.10 |
| AC2 Tracker có 3 version skip với marker | 1.5, 2.4, 4.4 |
| AC3 Dev DB không re-apply | 2.7 |
| AC4 Worker không regression | 2.7, 4.3 |
| AC5 Password không còn raw | 3.5, 3.6 + grep verify |
| AC6 README runbook đầy đủ | 3.7 |
