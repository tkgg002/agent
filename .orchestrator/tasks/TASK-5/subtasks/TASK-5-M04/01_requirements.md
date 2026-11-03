---
tier: T2
parent_task: TASK-5
sub_task_id: TASK-5-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-5/task-5-m04
---
# 01 Requirements: TASK-5-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-5`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** `TASK-5-M01, TASK-5-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-5)`
- **Test / DoD:** go test -race ./...
