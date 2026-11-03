# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-4-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`migration_fix_tz_worker_schedule.sql`)
- **Parent Task:** `TASK-4` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-4/task-4-m01`
- [x] **TASK-4-M01 (agent):**
  - **Context (`file:line`):** `agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql` (`-- Migration: fix Bug B — TZ freeze schedule`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-4-M03, TASK-4-M04` (Biên lai `EV-SUB-TASK-4-M01-PASS`)
  - **Definition of Done:** go test -race ./...
