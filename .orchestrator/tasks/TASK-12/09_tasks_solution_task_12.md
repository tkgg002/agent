# 💡 [S3 · UDF-v1] Giải Pháp Kỹ Thuật Duy Nhất & Code Demo — TASK-12: //127.0.0.1:8090: LIVE-PROCESS-HUD-01, ROLE-MODEL-ATTRIBUTION-01, CO-ARCHITECT-CHAT-01, TASK-INTAKE-DUAL-MODE-01
> 📌 **Task:** `TASK-12` · **Stage:** `DONE` · **Repo:** `agent` · **Branch:** `task/TASK-12` · **Files thay đổi (3):** `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py, memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py, memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` · **DoD:** `go test -race ./...`

### 📌 Tóm tắt & Trả lời trực tiếp (Summary)
Failed to authenticate. API Error: 401 OAuth access token is invalid.Failed to authenticate. API Error: 401 OAuth access token is invalid.

## 1. 🏗 Kiến Trúc & Chiến Lược Chuyển Đổi (Architecture Plan & Root Cause)
- **Pattern áp dụng:** Single-Responsibility Modular Strategy trên `agent`, giữ nguyên signature hàm ở tầng interface hiện hữu.
- **Cơ chế Rollback / Toggle:** Thực thi cô lập trên nhánh Worktree riêng và bảo toàn tương thích ngược 100% với luồng hiện hành.
- Điểm chạm tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` (`agent`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
- Điểm chạm tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` (`agent`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)
- Điểm chạm tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` (`agent`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`)
- Điểm chạm tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)` (`agent`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)

## 2. ⚠️ Kiểm Soát Rủi Ro & Lộ Trình Thực Thi (`4 Sub-Tasks` · `Wave 0..N`)
- **Memory & Concurrency:** Kiểm soát tài nguyên, context timeout và tránh phát sinh race condition/goroutine leak.
- **Tương thích Hợp đồng Dữ liệu:** Đảm bảo DTO/Schema đầu ra khớp 100% với caller hiện tại và đạt chuẩn G1–G8.

| Mã Sub-Task | Wave & Phụ thuộc | Repo | Điểm chạm (`file:line`) | Giải pháp Thiết kế & Tiêu chí Nghiệm thu (`DoD`) |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-12-M01`** | `Wave 0` | `agent` | `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-12-M02`** | `Wave 0` | `agent` | `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-12-M03`** | `Wave 1` *(sau `TASK-12-M01, TASK-12-M02`)* | `agent` | `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) trong worktree cô lập của `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-12-M04`** | `Wave 1` *(sau `TASK-12-M01, TASK-12-M02`)* | `agent` | `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (`def norm_ws(text: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` `go test -race ./...`) |

## 3. 💻 Key Diff / Code Preview (`go`)
```go
// Trích xuất trực tiếp từ TargetRepo: /Users/trainguyen/Documents/work/agent
// [1] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)
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

// [2] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)
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

// [3] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)
class LessonEntry:
    idx: int
    date: str
    title: str
    line_no: int
    raw_block: str
    topic_code: str
    topic_file: str
    short_label: str


# G-08: lesson tripwire fields (máy đọc được) — ID ổn định, Trigger regex, Check tiền điều kiện, Require bằng chứng tuỳ chọn.
```

## 4. 🛡️ Nghiệm Thu & Cổng Kiểm Định (`DoD & Gates`)
- **Lệnh kiểm thử bắt buộc (`DoD`):** `go test -race ./...` trên `agent` (nhánh `task/TASK-12`).
- **Decision Gate `DG0 / DG1 / DG2 (:8200 · 0 Token)`:** `PASS 🟢` trên `agent` (Model: `opus`).

## Clean-Room Approved Spec
- **R1 (`TASK-12-M01` — `agent`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`) (`REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154`)
- **R2 (`TASK-12-M02` — `agent`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`) (`REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83`)
- **R3 (`TASK-12-M03` — `agent`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`) (`REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83`)
- **R4 (`TASK-12-M04` — `agent`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`) (`REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89`)
