# 08 Tasks: Danh mục Sub-Task — TASK-8 (S3 Re-Plan — Hợp nhất 1 Khung Ứng Dụng Duy Nhất Single Unified IDE Shell, Xây dựng Bộ Máy Thực Thi Tự Động Per-Task Multi-Stage Autopilot Core Engine & Cô lập Vật lý 4 Lớp Git Worktree + Task Dir + Terminal + Scoped Autopilot)

> **Affected Repos**: `agent`
> **Tier**: T2
> **Suffix**: `task_8`

### Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [ ] **TASK-8-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)** (`Wave 0` · Repo: `agent` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _extract_yaml_frontmatter(src: str) -> str:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-8-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`analyze_dependencies.py`)** (`Wave 0` · Repo: `agent` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (`import os`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-8-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`analyze_dependencies.py`)** (`Wave 1` · Repo: `agent` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-8-M01, TASK-8-M02`)
  - **Context (`file:line`)**: `agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (`import os`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-8-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)** (`Wave 1` · Repo: `agent` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-8-M01, TASK-8-M02`)
  - **Context (`file:line`)**: `agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `tooling/evidence_run.py` (`def _git(repo: Path, *args: str) -> Optional[str]:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
