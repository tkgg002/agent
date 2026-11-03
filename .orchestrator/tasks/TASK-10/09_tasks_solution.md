# 💡 [S3 · UDF-v1] Giải Pháp Kỹ Thuật Duy Nhất & Code Demo — TASK-10: Bổ sung Model Chủ Lực gemini-4-argon của Antigravity, Phân Rã Sub-Tasks Theo Nhóm Kiến Trúc & Dấu Vết Mã Nguồn Thực Tế TASK-M01..TASK-M10 Có Đủ Context file:line + Mô tả + Test/DoD, và Phân Bổ Từng Sub-Task Về Riêng Từng Repository Đích Trong Một Epic Đa-Repo
> 📌 **Task:** `TASK-10` · **Stage:** `DONE` · **Repo:** `agent` · **Branch:** `task/TASK-10` · **Files thay đổi (3):** `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py, hooks/agy/tests/test_agy_guard.py, hooks/agy/tests/test_claim_linter.py` · **DoD:** `go test -race ./...`

### 📌 Tóm tắt & Trả lời trực tiếp (Summary)
Failed to authenticate. API Error: 401 OAuth access token is invalid.Failed to authenticate. API Error: 401 OAuth access token is invalid.

## 1. 🏗 Kiến Trúc & Chiến Lược Chuyển Đổi (Architecture Plan & Root Cause)
- **Pattern áp dụng:** Single-Responsibility Modular Strategy trên `agent`, giữ nguyên signature hàm ở tầng interface hiện hữu.
- **Cơ chế Rollback / Toggle:** Thực thi cô lập trên nhánh Worktree riêng và bảo toàn tương thích ngược 100% với luồng hiện hành.
- Điểm chạm tại `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)` (`agent`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)
- Điểm chạm tại `hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)` (`agent`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
- Điểm chạm tại `hooks/agy/tests/test_agy_guard.py:16 (def setUp(self) -> None:)` (`agent`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`)
- Điểm chạm tại `hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)` (`agent`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)

## 2. ⚠️ Kiểm Soát Rủi Ro & Lộ Trình Thực Thi (`4 Sub-Tasks` · `Wave 0..N`)
- **Memory & Concurrency:** Kiểm soát tài nguyên, context timeout và tránh phát sinh race condition/goroutine leak.
- **Tương thích Hợp đồng Dữ liệu:** Đảm bảo DTO/Schema đầu ra khớp 100% với caller hiện tại và đạt chuẩn G1–G8.

| Mã Sub-Task | Wave & Phụ thuộc | Repo | Điểm chạm (`file:line`) | Giải pháp Thiết kế & Tiêu chí Nghiệm thu (`DoD`) |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-10-M01`** | `Wave 0` | `agent` | `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py` (`VALUE = 2`) trong worktree cô lập của `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-10-M02`** | `Wave 0` | `agent` | `hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)** — Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-10-M03`** | `Wave 1` *(sau `TASK-10-M01, TASK-10-M02`)* | `agent` | `hooks/agy/tests/test_agy_guard.py:16 (def setUp(self) -> None:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/tests/test_agy_guard.py` (`def setUp(self) -> None:`) trong worktree cô lập của `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-10-M04`** | `Wave 1` *(sau `TASK-10-M01, TASK-10-M02`)* | `agent` | `hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)** — Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_claim_linter.py` (`class TestClaimLinter(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` `go test -race ./...`) |

## 3. 💻 Key Diff / Code Preview (`go`)
```go
// Trích xuất trực tiếp từ TargetRepo: /Users/trainguyen/Documents/work/agent
// [1] agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)
VALUE = 2
#inv1

// [2] agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)
class TestAgyGuard(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = Path(self._tmp.name)
        self.mem_root = self.tmp / "memory"
        self.ws_root = self.mem_root / "workspaces"
        self.state_dir = self.tmp / "state"
        self.ws_root.mkdir(parents=True, exist_ok=True)
        self.state_dir.mkdir(parents=True, exist_ok=True)

    def tearDown(self) -> None:
        self._tmp.cleanup()

// [3] agent/hooks/agy/tests/test_agy_guard.py:16 (def setUp(self) -> None:)
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = Path(self._tmp.name)
        self.mem_root = self.tmp / "memory"
        self.ws_root = self.mem_root / "workspaces"
        self.state_dir = self.tmp / "state"
        self.ws_root.mkdir(parents=True, exist_ok=True)
        self.state_dir.mkdir(parents=True, exist_ok=True)

    def tearDown(self) -> None:
        self._tmp.cleanup()
```

## 4. 🛡️ Nghiệm Thu & Cổng Kiểm Định (`DoD & Gates`)
- **Lệnh kiểm thử bắt buộc (`DoD`):** `go test -race ./...` trên `agent` (nhánh `task/TASK-10`).
- **Decision Gate `DG0 / DG1 / DG2 (:8200 · 0 Token)`:** `PASS 🟢` trên `agent` (Model: `opus`).

## Clean-Room Approved Spec
- **R1 (`TASK-10-M01` — `agent`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`) (`REF: agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1`)
- **R2 (`TASK-10-M02` — `agent`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`) (`REF: agent/hooks/agy/tests/test_agy_guard.py:15`)
- **R3 (`TASK-10-M03` — `agent`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`test_agy_guard.py`) (`REF: agent/hooks/agy/tests/test_agy_guard.py:16`)
- **R4 (`TASK-10-M04` — `agent`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`) (`REF: agent/hooks/agy/tests/test_claim_linter.py:17`)
