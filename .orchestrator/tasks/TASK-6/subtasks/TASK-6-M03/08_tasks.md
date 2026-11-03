# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-6-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`)
- **Parent Task:** `TASK-6` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-6/task-6-m03`
- [x] **TASK-6-M03 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-6)` (Biên lai `EV-SUB-TASK-6-M03-PASS`)
  - **Definition of Done:** go test -race ./...
