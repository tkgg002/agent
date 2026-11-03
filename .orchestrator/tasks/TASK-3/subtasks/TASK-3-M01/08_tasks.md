# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-3-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)
- **Parent Task:** `TASK-3` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-3/task-3-m01`
- [x] **TASK-3-M01 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py` (`VALUE = 2`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-3-M03, TASK-3-M04` (Biên lai `EV-SUB-TASK-3-M01-PASS`)
  - **Definition of Done:** go test -race ./...
