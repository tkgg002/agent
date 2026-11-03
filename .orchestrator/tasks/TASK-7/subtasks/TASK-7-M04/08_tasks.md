# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-7-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`proposer_axes.py`)
- **Parent Task:** `TASK-7` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-7/task-7-m04`
- [x] **TASK-7-M04 (agent):**
  - **Context (`file:line`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:)`
  - **Input Contract (`UpstreamReady=false`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py` (`def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Output Contract (`HandoffReady=true`):** Bàn giao cho `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-7)` (Biên lai `EV-SUB-TASK-7-M04-PASS`)
  - **Definition of Done:** go test -race ./...
