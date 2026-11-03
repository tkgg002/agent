# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-4-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`mutation_gate.py`)
- **Parent Task:** `TASK-4` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-4/task-4-m04`
- [x] **TASK-4-M04 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py` (`def sha256_bytes(data: bytes) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-4)` (Biên lai `EV-SUB-TASK-4-M04-PASS`)
  - **Definition of Done:** go test -race ./...
