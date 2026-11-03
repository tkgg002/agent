# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-2-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`harness2_engine.py`)
- **Parent Task:** `TASK-2` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-2/task-2-m02`
- [x] **TASK-2-M02 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _extract_yaml_frontmatter(src: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-2-M03, TASK-2-M04` (Biên lai `EV-SUB-TASK-2-M02-PASS`)
  - **Definition of Done:** go test -race ./...
