# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-5-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-5` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-5/task-5-m04`
- [x] **TASK-5-M04 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-5)` (Biên lai `EV-SUB-TASK-5-M04-PASS`)
  - **Definition of Done:** go test -race ./...
