---
tier: T2
parent_task: TASK-6
sub_task_id: TASK-6-M02
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-6/task-6-m02
---
# 01 Requirements: TASK-6-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-6`
- **Wave:** `Wave 0`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** ``
- **Downstream Consumers:** `TASK-6-M03, TASK-6-M04`
- **Test / DoD:** go test -race ./...
