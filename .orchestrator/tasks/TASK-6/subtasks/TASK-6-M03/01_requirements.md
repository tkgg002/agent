---
tier: T2
parent_task: TASK-6
sub_task_id: TASK-6-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-6/task-6-m03
---
# 01 Requirements: TASK-6-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-6`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** `TASK-6-M01, TASK-6-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-6)`
- **Test / DoD:** go test -race ./...
