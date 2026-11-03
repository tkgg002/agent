---
tier: T2
parent_task: TASK-10
sub_task_id: TASK-10-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-10/task-10-m03
---
# 01 Requirements: TASK-10-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-10`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:16 (def setUp(self) -> None:)`
- **Input Contract Ready:** `false`
- **Upstream Dependencies:** `TASK-10-M01, TASK-10-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-10)`
- **Test / DoD:** go test -race ./...
