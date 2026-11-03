---
tier: T2
parent_task: TASK-3
sub_task_id: TASK-3-M01
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-3/task-3-m01
---
# 01 Requirements: TASK-3-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)
- **Parent Task:** `TASK-3`
- **Wave:** `Wave 0`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** ``
- **Downstream Consumers:** `TASK-3-M03, TASK-3-M04`
- **Test / DoD:** go test -race ./...
