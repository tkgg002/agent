---
tier: T2
parent_task: TASK-1
sub_task_id: TASK-1-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-1/task-1-m03
---
# 01 Requirements: TASK-1-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-1`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
- **Input Contract Ready:** `false`
- **Upstream Dependencies:** `TASK-1-M01, TASK-1-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-1)`
- **Test / DoD:** go test -race ./...
