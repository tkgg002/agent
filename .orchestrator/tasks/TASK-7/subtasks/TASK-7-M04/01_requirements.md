---
tier: T2
parent_task: TASK-7
sub_task_id: TASK-7-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
isolated_branch: task/TASK-7/task-7-m04
---
# 01 Requirements: TASK-7-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`proposer_axes.py`)
- **Parent Task:** `TASK-7`
- **Wave:** `Wave 1`
- **Target Repo:** `/Users/trainguyen/Documents/work/agent`
- **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:)`
- **Input Contract Ready:** `false`
- **Upstream Dependencies:** `TASK-7-M01, TASK-7-M02`
- **Downstream Consumers:** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-7)`
- **Test / DoD:** go test -race ./...
