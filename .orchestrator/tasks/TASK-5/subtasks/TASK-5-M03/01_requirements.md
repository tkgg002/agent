---
tier: T2
parent_task: TASK-5
sub_task_id: TASK-5-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-5/task-5-m03
---
# 01 Requirements: TASK-5-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-5`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** `TASK-5-M01, TASK-5-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-5)`
- **Test / DoD:** go test -race ./...
