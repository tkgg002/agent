# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-3-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)
- **Parent Task:** `TASK-3` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-3/task-3-m04`
- [x] **TASK-3-M04 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
  - **Input Contract (`UpstreamReady=true`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-3)` (Biên lai `EV-SUB-TASK-3-M04-PASS`)
  - **Definition of Done:** go test -race ./...
