# 08 Tasks: Danh mục Sub-Task — TASK-4 (Bổ sung 3 Tab Hiển Thị (Wave 0..N DAG, Autopilot Cost & Gate Ledger, Multi-Worker Grid) vào Bubble Tea TUI có sẵn)

> **Affected Repos**: `agent`
> **Tier**: T2

## Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [x] **TASK-4-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`migration_fix_tz_worker_schedule.sql`)**
  - **Context (`file:line`)**: `agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql` (`-- Migration: fix Bug B — TZ freeze schedule`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-4-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)`
  - **Role / Slot / Model**: `⚖️ Cross-Model Laya Gate Reviewer` · `Slot H` · `convaiinnovations/laya (local :8200)` (`/laya-router`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (`def norm_ws(text: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-4-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`laya_gate.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:93 (def now_iso() -> str:)`
  - **Role / Slot / Model**: `⚖️ Cross-Model Laya Gate Reviewer` · `Slot H` · `convaiinnovations/laya (local :8200)` (`/laya-router`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (`def now_iso() -> str:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-4-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`mutation_gate.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py` (`def sha256_bytes(data: bytes) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
