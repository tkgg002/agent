# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-3-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)
- **Parent Task:** `TASK-3` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-3/task-3-m02`
- [x] **TASK-3-M02 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `TASK-3-M03, TASK-3-M04` (Biên lai `EV-SUB-TASK-3-M02-PASS`)
  - **Definition of Done:** go test -race ./...
