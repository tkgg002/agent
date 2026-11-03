# 08 Tasks: Danh mục Sub-Task — TASK-3 (Xây dựng internal/agentgate (Trạm Kiểm Soát /agent) & Sổ Cái Chi Phí Autopilot Cost ($) Ledger)

> **Affected Repos**: `agent`
> **Tier**: T2

## Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [x] **TASK-3-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py` (`VALUE = 2`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-3-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-3-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-3-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)**
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
