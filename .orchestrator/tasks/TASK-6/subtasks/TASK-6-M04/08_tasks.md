# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-6-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)
- **Parent Task:** `TASK-6` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-6/task-6-m04`
- [x] **TASK-6-M04 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_claim_linter.py` (`class TestClaimLinter(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-6)` (Biên lai `EV-SUB-TASK-6-M04-PASS`)
  - **Definition of Done:** go test -race ./...
