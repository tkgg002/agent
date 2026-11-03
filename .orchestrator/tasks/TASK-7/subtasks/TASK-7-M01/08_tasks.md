# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-7-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)
- **Parent Task:** `TASK-7` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-7/task-7-m01`
- [x] **TASK-7-M01 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _extract_yaml_frontmatter(src: str) -> str:`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-7-M03, TASK-7-M04` (Biên lai `EV-SUB-TASK-7-M01-PASS`)
  - **Definition of Done:** go test -race ./...
