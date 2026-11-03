# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-6-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-6` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-6/task-6-m01`
- [x] **TASK-6-M01 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-6-M03, TASK-6-M04` (Biên lai `EV-SUB-TASK-6-M01-PASS`)
  - **Definition of Done:** go test -race ./...
