# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-12-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)
- **Parent Task:** `TASK-12` | **Wave:** `Wave 0` | **Isolated Branch:** `task/TASK-12/task-12-m02`
- [ ] **TASK-12-M02 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-12-M03, TASK-12-M04` (Biên lai `EV-SUB-TASK-12-M02-PENDING`)
  - **Definition of Done:** go test -race ./...
