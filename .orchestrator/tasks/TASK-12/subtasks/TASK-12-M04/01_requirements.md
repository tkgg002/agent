---
tier: T2
parent_task: TASK-12
sub_task_id: TASK-12-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-12/task-12-m04
---
# 01 Requirements: TASK-12-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)
- **Parent Task:** `TASK-12`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)`
- **Input Contract Ready:** `false`
- **Upstream Dependencies:** `TASK-12-M01, TASK-12-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-12)`
- **Test / DoD:** go test -race ./...
