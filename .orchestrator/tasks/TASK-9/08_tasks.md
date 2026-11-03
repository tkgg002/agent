# 08 Tasks: Danh mục Sub-Task — TASK-9 (Chốt chặn 2 Pha S1..S3 -> DỪNG CHỜ APPROVE -> S4..S6, Chọn Repo Đích & Setup CLI/Model, Cơ chế Chia nhỏ Sub-Tasks theo Wave, Khung Đọc Report/Giải Pháp/Code Demo Trực Tiếp, Đồng bộ 100% Cửa Sổ Phải theo Task & Bảng Hiển thị /agent Kernel + Bài học FTS5)

> **Affected Repos**: `agent`
> **Tier**: T2
> **Suffix**: `task_9`

### Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [ ] **TASK-9-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)** (`Wave 0` · Repo: `agent` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py` (`VALUE = 2`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-9-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)** (`Wave 0` · Repo: `agent` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-9-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)** (`Wave 1` · Repo: `agent` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-9-M01, TASK-9-M02`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-9-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)** (`Wave 1` · Repo: `agent` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-9-M01, TASK-9-M02`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
