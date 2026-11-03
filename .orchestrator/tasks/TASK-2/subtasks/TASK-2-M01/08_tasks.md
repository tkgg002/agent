# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-2-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`__init__.py`)
- **Parent Task:** `TASK-2` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-2/task-2-m01`
- [ ] **TASK-2-M01 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""")`
  - **Input Contract (`UpstreamReady=true`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py` (`"""Orchestrator & Harness2 Recursive Evolution Package for /agent."""`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-2-M03, TASK-2-M04` (Biên lai `EV-SUB-TASK-2-M01-PENDING`)
  - **Definition of Done:** go test -race ./...
