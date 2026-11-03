# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-12-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`)
- **Parent Task:** `TASK-12` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-12/task-12-m03`
- [ ] **TASK-12-M03 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
  - **Input Contract (`UpstreamReady=false`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) trong worktree cô lập của `agent`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-12)` (Biên lai `EV-SUB-TASK-12-M03-PENDING`)
  - **Definition of Done:** go test -race ./...
