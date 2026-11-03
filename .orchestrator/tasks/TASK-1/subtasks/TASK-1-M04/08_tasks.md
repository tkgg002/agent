# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-1-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-1` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-1/task-1-m04`
- [x] **TASK-1-M04 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Input Contract (`UpstreamReady=false`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-1)` (Biên lai `EV-SUB-TASK-1-M04-PASS`)
  - **Definition of Done:** go test -race ./...
