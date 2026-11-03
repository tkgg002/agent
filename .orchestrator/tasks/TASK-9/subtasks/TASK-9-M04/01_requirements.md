---
tier: T2
parent_task: TASK-9
sub_task_id: TASK-9-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-9/task-9-m04
---
# 01 Requirements: TASK-9-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)
- **Parent Task:** `TASK-9`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
- **Input Contract Ready:** `false`
- **Upstream Dependencies:** `TASK-9-M01, TASK-9-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-9)`
- **Test / DoD:** go test -race ./...
