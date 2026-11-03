# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-7-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`__init__.py`)
- **Parent Task:** `TASK-7` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-7/task-7-m02`
- [ ] **TASK-7-M02 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""")`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py` (`"""Orchestrator & Harness2 Recursive Evolution Package for /agent."""`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-7-M03, TASK-7-M04` (Biên lai `EV-SUB-TASK-7-M02-PENDING`)
  - **Definition of Done:** go test -race ./...
