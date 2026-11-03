# 08 Tasks: Danh mục Sub-Task — TASK-2 (Nâng cấp internal/agent/pm.go, internal/gitclient & internal/orchestrator/task_runner.go để hỗ trợ Dynamic Model Routing (Opus vs Sonnet) và Chạy Song Song Đa Coder theo Wave 0..N trên git worktree)

> **Affected Repos**: `agent`
> **Tier**: T2

## Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [ ] **TASK-2-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`__init__.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""")`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py` (`"""Orchestrator & Harness2 Recursive Evolution Package for /agent."""`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-2-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`harness2_engine.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _extract_yaml_frontmatter(src: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-2-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _has_skill_trigger_token(body: str, sname: str) -> bool:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-2-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`proposer_axes.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py` (`def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
