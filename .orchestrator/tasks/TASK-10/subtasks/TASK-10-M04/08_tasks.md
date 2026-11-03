# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-10-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)
- **Parent Task:** `TASK-10` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-10/task-10-m04`
- [ ] **TASK-10-M04 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)`
  - **Input Contract (`UpstreamReady=false`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_claim_linter.py` (`class TestClaimLinter(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-10)` (Biên lai `EV-SUB-TASK-10-M04-PENDING`)
  - **Definition of Done:** go test -race ./...
