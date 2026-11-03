# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-12-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-12` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-12/task-12-m01`
- [ ] **TASK-12-M01 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-12-M03, TASK-12-M04` (Biên lai `EV-SUB-TASK-12-M01-PENDING`)
  - **Definition of Done:** go test -race ./...
