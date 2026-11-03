# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-12-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)
- **Parent Task:** `TASK-12` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-12/task-12-m04`
- [ ] **TASK-12-M04 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)`
  - **Input Contract (`UpstreamReady=false`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (`def norm_ws(text: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-12)` (Biên lai `EV-SUB-TASK-12-M04-PENDING`)
  - **Definition of Done:** go test -race ./...
