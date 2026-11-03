# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-2-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)
- **Parent Task:** `TASK-2` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-2/task-2-m03`
- [x] **TASK-2-M03 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:)`
  - **Input Contract (`UpstreamReady=false`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _has_skill_trigger_token(body: str, sname: str) -> bool:`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-2)` (Biên lai `EV-SUB-TASK-2-M03-PASS`)
  - **Definition of Done:** go test -race ./...
