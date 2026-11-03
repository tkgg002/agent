# 08 Tasks: Danh mục Sub-Task — TASK-10 (Bổ sung Model Chủ Lực gemini-4-argon của Antigravity, Phân Rã Sub-Tasks Theo Nhóm Kiến Trúc & Dấu Vết Mã Nguồn Thực Tế TASK-M01..TASK-M10 Có Đủ Context file:line + Mô tả + Test/DoD, và Phân Bổ Từng Sub-Task Về Riêng Từng Repository Đích Trong Một Epic Đa-Repo)

> **Affected Repos**: `agent`
> **Tier**: T2
> **Suffix**: `task_10`

### Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [ ] **TASK-10-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)** (`Wave 0` · Repo: `agent` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py` (`VALUE = 2`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-10-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)** (`Wave 0` · Repo: `agent` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-10-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`)** (`Wave 1` · Repo: `agent` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-10-M01, TASK-10-M02`)
  - **Context (`file:line`)**: `agent/hooks/agy/tests/test_agy_guard.py:16 (def setUp(self) -> None:)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/tests/test_agy_guard.py` (`def setUp(self) -> None:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-10-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)** (`Wave 1` · Repo: `agent` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-10-M01, TASK-10-M02`)
  - **Context (`file:line`)**: `agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_claim_linter.py` (`class TestClaimLinter(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
