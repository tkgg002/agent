---
tier: T2
parent_task: TASK-6
sub_task_id: TASK-6-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-6/task-6-m04
---
# 01 Requirements: TASK-6-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)
- **Parent Task:** `TASK-6`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** `TASK-6-M01, TASK-6-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-6)`
- **Test / DoD:** go test -race ./...
