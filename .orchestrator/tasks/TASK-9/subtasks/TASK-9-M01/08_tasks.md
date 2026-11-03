# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-9-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)
- **Parent Task:** `TASK-9` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-9/task-9-m01`
- [ ] **TASK-9-M01 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py` (`VALUE = 2`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-9-M03, TASK-9-M04` (Biên lai `EV-SUB-TASK-9-M01-PENDING`)
  - **Definition of Done:** go test -race ./...
