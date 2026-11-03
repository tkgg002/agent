---
tier: T2
parent_task: TASK-2
sub_task_id: TASK-2-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-2/task-2-m03
---
# 01 Requirements: TASK-2-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)
- **Parent Task:** `TASK-2`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:)`
- **Input Contract Ready:** `false`
- **Upstream Dependencies:** `TASK-2-M01, TASK-2-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-2)`
- **Test / DoD:** go test -race ./...
