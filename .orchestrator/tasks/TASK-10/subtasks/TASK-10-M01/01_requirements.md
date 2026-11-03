---
tier: T2
parent_task: TASK-10
sub_task_id: TASK-10-M01
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-10/task-10-m01
---
# 01 Requirements: TASK-10-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)
- **Parent Task:** `TASK-10`
- **Wave:** `Wave 0`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** ``
- **Downstream Consumers:** `TASK-10-M03, TASK-10-M04`
- **Test / DoD:** go test -race ./...
