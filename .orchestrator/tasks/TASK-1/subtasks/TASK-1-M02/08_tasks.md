# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-1-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-1` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-1/task-1-m02`
- [x] **TASK-1-M02 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-1-M03, TASK-1-M04` (Biên lai `EV-SUB-TASK-1-M02-PASS`)
  - **Definition of Done:** go test -race ./...
