# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-10-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-10` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-10/task-10-m03`
- [ ] **TASK-10-M03 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:16 (def setUp(self) -> None:)`
  - **Input Contract (`UpstreamReady=false`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/tests/test_agy_guard.py` (`def setUp(self) -> None:`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-10)` (Biên lai `EV-SUB-TASK-10-M03-PENDING`)
  - **Definition of Done:** go test -race ./...
