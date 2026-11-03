# 08 Tasks: Danh mục Sub-Task — TASK-5 (S5 -> S3 Re-Plan — Khắc phục triệt để 6 Sơ hở AF-1 $\rightarrow$ AF-6 phát hiện qua Adversarial Re-Audit Commit 54e36e4, thực thi tại Commit fa6b410)

> **Affected Repos**: `agent`
> **Tier**: T2

## Nhóm 1: Thực thi & Kiểm thử trên `agent` (45 files)
- [x] **TASK-5-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`analyze_dependencies.py`)**
  - **Context (`file:line`)**: `agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (`import os`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-5-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)**
  - **Context (`file:line`)**: `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-5-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)**
  - **Context (`file:line`)**: `agent/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/agy_guard.py` (`def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:`) trong worktree cô lập của `agent`.
  - **Test / DoD**: go test -race ./...
- [x] **TASK-5-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)**
  - **Context (`file:line`)**: `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)`
  - **Role / Slot / Model**: `🧪 Cross-Model QA & E2E Auditor` · `Slot K` · `claude-opus-4-6 (antigravity)` (`/qa-agent`)
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.
  - **Test / DoD**: go test -race ./...
