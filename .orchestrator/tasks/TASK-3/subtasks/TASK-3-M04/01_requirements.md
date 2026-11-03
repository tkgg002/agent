---
tier: T2
parent_task: TASK-3
sub_task_id: TASK-3-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-3/task-3-m04
---
# 01 Requirements: TASK-3-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)
- **Parent Task:** `TASK-3`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** `TASK-3-M01, TASK-3-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-3)`
- **Test / DoD:** go test -race ./...
