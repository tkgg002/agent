# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-4-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)
- **Parent Task:** `TASK-4` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-4/task-4-m02`
- [x] **TASK-4-M02 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (`def norm_ws(text: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-4-M03, TASK-4-M04` (Biên lai `EV-SUB-TASK-4-M02-PASS`)
  - **Definition of Done:** go test -race ./...
