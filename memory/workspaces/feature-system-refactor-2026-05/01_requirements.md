# 01 — Requirements (Phase B1 + B2)

**Date**: 2026-05-04 17:10+07
**Phase**: B1 (Quick Wins) + B2 (Tooling & Hygiene)
**User chốt scope**: B1 + B2 (Recommended), full kill/restart OK, ưu tiên 4 flow (Operator UI / Auth / Admin-API / CDC pipeline).

---

## Functional Requirements

### B1 — Quick Wins (nửa ngày, low risk)

- **FR-B1-1**: Commit 5 file Go pending (admin-api hardening Phase F1 + helpers.go fix Phase F3) với commit message rõ — không lẫn vào commit khác.
- **FR-B1-2**: `cdc-system/architecture.md` không còn nói Airbyte như hybrid path đang hoạt động (commit `8ef7d71` đã gỡ Airbyte).
- **FR-B1-3**: `agent/memory/global/project_context.md` + `tech_stack.md` được fill với thực tế disk (4 service, ngôn ngữ, infra real).
- **FR-B1-4**: `agent/memory/global/active_plans.md` đánh dấu `feature-cdc-integration` Phase F = Done, thêm `feature-system-refactor-2026-05` = Active.
- **FR-B1-5**: APPEND lesson `L-input-fallback-pattern` vào `lessons.md` theo format Global Pattern §13.

### B2 — Tooling & Hygiene (1-2 ngày, low risk)

- **FR-B2-1**: `cdc-auth-service` chạy được local, `/healthz` OK, login endpoint trả token.
- **FR-B2-2**: `cdc-cms-web` chạy `npm run dev`, browser load `/` không crash, login page render đúng.
- **FR-B2-3**: Smoke E2E operator path:
  1. Login qua FE.
  2. List source registry (V1 + V2).
  3. Create source mới (V2 register qua admin-api hoặc CMS API).
  4. Approve 1 schema_proposal nếu có pending.
  5. Create master qua wizard.
  6. Trigger transmute (cron tick hoặc post-ingest).
  7. Verify row landed shadow + master.
- **FR-B2-4**: Có 1 script `dev-up.sh` (hoặc Makefile target) để start cả 4 service local + check healthy.

## Non-Functional Requirements

- **NFR-1** (CLAUDE.md §1): Brain TUYỆT ĐỐI không edit `.go/.ts/.js/.py/.sql`. Mọi code edit → Muscle delegate.
- **NFR-2** (Lessons): mọi PASS phải exercise-driven (smoke endpoint), không health-only. "Service listening ≠ healthy".
- **NFR-3** (Lessons): mọi commit phải verify build + test PASS trước, không "build pass = done".
- **NFR-4** (CLAUDE.md §11): mọi memory file APPEND, không overwrite.
- **NFR-5** (CLAUDE.md §7): Full Doc Set per phase. Phase B1+B2 dùng các file `01_requirements.md`, `02_plan.md`, `03_implementation.md`, `05_progress.md` (APPEND), `08_tasks.md`, `09_tasks_solution.md`.
- **NFR-6** (Lessons "Brain plan dựa state tưởng tượng"): mọi step phải verify thực tế disk/runtime trước khi mark done.

## Acceptance Criteria

| AC | Pass evidence |
|---|---|
| AC-B1-1 | `git log --oneline` xuất hiện 1 commit mới với 5 file Phase F1/F3 |
| AC-B1-2 | `grep -i airbyte cdc-system/architecture.md` trả về 0 kết quả hoặc nói rõ "đã gỡ" |
| AC-B1-3 | `project_context.md` + `tech_stack.md` không còn `[DATE]` `[Project Name]` placeholder |
| AC-B1-4 | `active_plans.md` có entry `feature-system-refactor-2026-05` Active |
| AC-B1-5 | `lessons.md` có heading mới `L-input-fallback-pattern` với format Global Pattern A/B/X/Y |
| AC-B2-1 | `curl http://localhost:<auth-port>/healthz` → 200 |
| AC-B2-2 | `curl http://localhost:5173/` → 200 hoặc browser thấy app render |
| AC-B2-3 | log smoke E2E ghi rõ 7 step pass + row landed shadow + master |
| AC-B2-4 | `bash scripts/dev-up.sh` → 4 service báo healthy |
| AC-Final | `report_phase_b_*.md` ghi đầy đủ exercise + log evidence |

## Risk

| ID | Risk | Mitigation |
|---|---|---|
| R-1 | Auth service config thiếu (DB? JWT secret?) | Đọc `cdc-auth-service/config/` + Makefile. Nếu thiếu → ghi rõ vào report, không bịa |
| R-2 | FE không build được do dep version | `npm install` lại nếu cần. Vite cache xóa nếu lỗi |
| R-3 | Smoke E2E chạm dữ liệu live → có thể nhiễm | Dùng prefix `b2_smoke_*` cho test data, không dùng id production-like |
| R-4 | Restart cms-server PID 13653 (5d uptime) có thể mất state in-memory | User OK, full restart cho phép. Backup log trước khi kill |
| R-5 | Brain quên tracking, hoặc miss step | Task list đầy đủ + APPEND progress sau MỖI step |

## Definition of Done

1. Tất cả AC PASS với exercise evidence.
2. `report_phase_b_*.md` viết bằng kết quả thực tế (lệnh + output snippet).
3. `05_progress.md` APPEND đầy đủ timeline.
4. 4 service local đều chạy + verified.
5. Brain pre-flight CLAUDE.md §14 PASS.
