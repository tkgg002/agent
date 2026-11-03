# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-5-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-5` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-5/task-5-m03`
- [x] **TASK-5-M03 (agent):**
  - **Context (`file:line`):** `agent/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/agy_guard.py` (`def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-5)` (Biên lai `EV-SUB-TASK-5-M03-PASS`)
  - **Definition of Done:** go test -race ./...
