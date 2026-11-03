# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-9-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-9` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-9/task-9-m02`
- [ ] **TASK-9-M02 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-9-M03, TASK-9-M04` (Biên lai `EV-SUB-TASK-9-M02-PENDING`)
  - **Definition of Done:** go test -race ./...
