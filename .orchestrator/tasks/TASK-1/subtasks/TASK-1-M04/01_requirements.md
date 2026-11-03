---
tier: T2
parent_task: TASK-1
sub_task_id: TASK-1-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-1/task-1-m04
---
# 01 Requirements: TASK-1-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-1`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
- **Input Contract Ready:** `false`
- **Upstream Dependencies:** `TASK-1-M01, TASK-1-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-1)`
- **Test / DoD:** go test -race ./...
