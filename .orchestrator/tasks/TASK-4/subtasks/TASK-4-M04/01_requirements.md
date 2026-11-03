---
tier: T2
parent_task: TASK-4
sub_task_id: TASK-4-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-4/task-4-m04
---
# 01 Requirements: TASK-4-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`mutation_gate.py`)
- **Parent Task:** `TASK-4`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:)`
- **Input Contract Ready:** `true`
- **Upstream Dependencies:** `TASK-4-M01, TASK-4-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-4)`
- **Test / DoD:** go test -race ./...
