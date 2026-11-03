---
tier: T2
parent_task: TASK-5
sub_task_id: TASK-5-M02
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-5/task-5-m02
---
# 01 Requirements: TASK-5-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-5`
- **Wave:** `Wave 0`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** ``
- **Downstream Consumers:** `TASK-5-M03, TASK-5-M04`
- **Test / DoD:** go test -race ./...
