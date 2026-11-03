# 08 Tasks: Danh mục Sub-Task — TASK-6 (1 cả 3 tấm hình mẫu của User)

> **Affected Repos**: `agent`
> **Tier**: T2

## Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [x] **TASK-6-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)**
  - **Context (`file:line`)**: `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-6-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)**
  - **Context (`file:line`)**: `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-6-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`)**
  - **Context (`file:line`)**: `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-6-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)**
  - **Context (`file:line`)**: `agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_claim_linter.py` (`class TestClaimLinter(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
