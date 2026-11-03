# 08 Tasks: Danh mục Sub-Task — TASK-12 (//127.0.0.1:8090: LIVE-PROCESS-HUD-01, ROLE-MODEL-ATTRIBUTION-01, CO-ARCHITECT-CHAT-01, TASK-INTAKE-DUAL-MODE-01)

> **Affected Repos**: `agent`
> **Tier**: T2
> **Suffix**: `task_12`

### Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [ ] **TASK-12-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)** (`Wave 0` · Repo: `agent` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-12-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)** (`Wave 0` · Repo: `agent` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-12-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`)** (`Wave 1` · Repo: `agent` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-12-M01, TASK-12-M02`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-12-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)** (`Wave 1` · Repo: `agent` · Assignee: `Tester` · Model: `convaiinnovations/laya (local :8200)` · DependsOn: `TASK-12-M01, TASK-12-M02`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (`def norm_ws(text: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
