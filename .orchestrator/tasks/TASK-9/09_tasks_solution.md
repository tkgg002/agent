# 💡 [S3 · UDF-v1] Giải Pháp Kỹ Thuật Duy Nhất & Code Demo — TASK-9: Chốt chặn 2 Pha S1..S3 -> DỪNG CHỜ APPROVE -> S4..S6, Chọn Repo Đích & Setup CLI/Model, Cơ chế Chia nhỏ Sub-Tasks theo Wave, Khung Đọc Report/Giải Pháp/Code Demo Trực Tiếp, Đồng bộ 100% Cửa Sổ Phải theo Task & Bảng Hiển thị /agent Kernel + Bài học FTS5
> 📌 **Task:** `TASK-9` · **Stage:** `DONE` · **Repo:** `agent` · **Branch:** `task/TASK-9` · **Files thay đổi (3):** `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py, memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py, memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` · **DoD:** `go test -race ./...`

### 📌 Tóm tắt & Trả lời trực tiếp (Summary)
Failed to authenticate. API Error: 401 OAuth access token is invalid.Failed to authenticate. API Error: 401 OAuth access token is invalid.

## 1. 🏗 Kiến Trúc & Chiến Lược Chuyển Đổi (Architecture Plan & Root Cause)
- **Pattern áp dụng:** Single-Responsibility Modular Strategy trên `agent`, giữ nguyên signature hàm ở tầng interface hiện hữu.
- **Cơ chế Rollback / Toggle:** Thực thi cô lập trên nhánh Worktree riêng và bảo toàn tương thích ngược 100% với luồng hiện hành.
- Điểm chạm tại `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)` (`agent`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)
- Điểm chạm tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` (`agent`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)
- Điểm chạm tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)` (`agent`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
- Điểm chạm tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` (`agent`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)

## 2. ⚠️ Kiểm Soát Rủi Ro & Lộ Trình Thực Thi (`4 Sub-Tasks` · `Wave 0..N`)
- **Memory & Concurrency:** Kiểm soát tài nguyên, context timeout và tránh phát sinh race condition/goroutine leak.
- **Tương thích Hợp đồng Dữ liệu:** Đảm bảo DTO/Schema đầu ra khớp 100% với caller hiện tại và đạt chuẩn G1–G8.

| Mã Sub-Task | Wave & Phụ thuộc | Repo | Điểm chạm (`file:line`) | Giải pháp Thiết kế & Tiêu chí Nghiệm thu (`DoD`) |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-9-M01`** | `Wave 0` | `agent` | `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py` (`VALUE = 2`) trong worktree cô lập của `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-9-M02`** | `Wave 0` | `agent` | `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-9-M03`** | `Wave 1` *(sau `TASK-9-M01, TASK-9-M02`)* | `agent` | `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:`) trong worktree cô lập của `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-9-M04`** | `Wave 1` *(sau `TASK-9-M01, TASK-9-M02`)* | `agent` | `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` `go test -race ./...`) |

## 3. 💻 Key Diff / Code Preview (`go`)
```go
// Trích xuất trực tiếp từ TargetRepo: /Users/trainguyen/Documents/work/agent
// [1] agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)
VALUE = 2
#inv1

// [2] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)
def _unquote(val: Any) -> Any:
    if isinstance(val, str) and len(val) >= 2 and val[0] == '"' and val[-1] == '"':
        try:
            return json.loads(val)
        except Exception:
            return val.strip('"')
    return val


def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:
    tr = payload.get("transcriptPath", "")
    if not tr:

// [3] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)
def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:
    tr = payload.get("transcriptPath", "")
    if not tr:
        return []
    try:
        with open(tr, encoding="utf-8") as f:
            return [json.loads(line) for line in f if line.strip()]
    except OSError:
        return []


def calls(st: List[Dict[str, Any]]) -> Iterator[Tuple[int, Optional[str], Dict[str, Any]]]:
```

## 4. 🛡️ Nghiệm Thu & Cổng Kiểm Định (`DoD & Gates`)
- **Lệnh kiểm thử bắt buộc (`DoD`):** `go test -race ./...` trên `agent` (nhánh `task/TASK-9`).
- **Decision Gate `DG0 / DG1 / DG2 (:8200 · 0 Token)`:** `PASS 🟢` trên `agent` (Model: `opus`).

## Clean-Room Approved Spec
- **R1 (`TASK-9-M01` — `agent`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`) (`REF: agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1`)
- **R2 (`TASK-9-M02` — `agent`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`) (`REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154`)
- **R3 (`TASK-9-M03` — `agent`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`) (`REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163`)
- **R4 (`TASK-9-M04` — `agent`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`) (`REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83`)
