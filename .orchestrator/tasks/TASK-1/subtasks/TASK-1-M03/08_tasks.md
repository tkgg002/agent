# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-1-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-1` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-1/task-1-m03`
- [x] **TASK-1-M03 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Input Contract (`UpstreamReady=false`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-1)` (Biên lai `EV-SUB-TASK-1-M03-PASS`)
  - **Definition of Done:** go test -race ./...
