# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-6-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-6` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-6/task-6-m02`
- [x] **TASK-6-M02 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-6-M03, TASK-6-M04` (Biên lai `EV-SUB-TASK-6-M02-PASS`)
  - **Definition of Done:** go test -race ./...
