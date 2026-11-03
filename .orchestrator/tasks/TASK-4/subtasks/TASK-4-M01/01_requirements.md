---
tier: T2
parent_task: TASK-4
sub_task_id: TASK-4-M01
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-4/task-4-m01
---
# 01 Requirements: TASK-4-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`migration_fix_tz_worker_schedule.sql`)
- **Parent Task:** `TASK-4`
- **Wave:** `Wave 0`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** ``
- **Downstream Consumers:** `TASK-4-M03, TASK-4-M04`
- **Test / DoD:** go test -race ./...
