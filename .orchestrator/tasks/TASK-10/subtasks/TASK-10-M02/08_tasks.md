# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-10-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-10` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-10/task-10-m02`
- [ ] **TASK-10-M02 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-10-M03, TASK-10-M04` (Biên lai `EV-SUB-TASK-10-M02-PENDING`)
  - **Definition of Done:** go test -race ./...
