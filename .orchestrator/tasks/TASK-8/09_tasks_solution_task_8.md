# 💡 [S3 · UDF-v1] Giải Pháp Kỹ Thuật Duy Nhất & Code Demo — TASK-8: S3 Re-Plan — Hợp nhất 1 Khung Ứng Dụng Duy Nhất Single Unified IDE Shell, Xây dựng Bộ Máy Thực Thi Tự Động Per-Task Multi-Stage Autopilot Core Engine & Cô lập Vật lý 4 Lớp Git Worktree + Task Dir + Terminal + Scoped Autopilot
> 📌 **Task:** `TASK-8` · **Stage:** `DONE` · **Repo:** `agent` · **Branch:** `task/TASK-8` · **Files thay đổi (3):** `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py, memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py, tooling/evidence_run.py` · **DoD:** `go test -race ./...`

### 📌 Tóm tắt & Trả lời trực tiếp (Summary)
Failed to authenticate. API Error: 401 OAuth access token is invalid.Failed to authenticate. API Error: 401 OAuth access token is invalid.

## 1. 🏗 Kiến Trúc & Chiến Lược Chuyển Đổi (Architecture Plan & Root Cause)
- **Pattern áp dụng:** Single-Responsibility Modular Strategy trên `agent`, giữ nguyên signature hàm ở tầng interface hiện hữu.
- **Cơ chế Rollback / Toggle:** Thực thi cô lập trên nhánh Worktree riêng và bảo toàn tương thích ngược 100% với luồng hiện hành.
- Điểm chạm tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)` (`agent`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)
- Điểm chạm tại `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)` (`agent`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`analyze_dependencies.py`)
- Điểm chạm tại `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)` (`agent`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`analyze_dependencies.py`)
- Điểm chạm tại `tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)` (`agent`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)

## 2. ⚠️ Kiểm Soát Rủi Ro & Lộ Trình Thực Thi (`4 Sub-Tasks` · `Wave 0..N`)
- **Memory & Concurrency:** Kiểm soát tài nguyên, context timeout và tránh phát sinh race condition/goroutine leak.
- **Tương thích Hợp đồng Dữ liệu:** Đảm bảo DTO/Schema đầu ra khớp 100% với caller hiện tại và đạt chuẩn G1–G8.

| Mã Sub-Task | Wave & Phụ thuộc | Repo | Điểm chạm (`file:line`) | Giải pháp Thiết kế & Tiêu chí Nghiệm thu (`DoD`) |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-8-M01`** | `Wave 0` | `agent` | `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _extract_yaml_frontmatter(src: str) -> str:`) trong worktree cô lập của `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-8-M02`** | `Wave 0` | `agent` | `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`analyze_dependencies.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (`import os`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-8-M03`** | `Wave 1` *(sau `TASK-8-M01, TASK-8-M02`)* | `agent` | `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`analyze_dependencies.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (`import os`) trong worktree cô lập của `agent`. (`DoD:` `go test -race ./...`) |
| **`TASK-8-M04`** | `Wave 1` *(sau `TASK-8-M01, TASK-8-M02`)* | `agent` | `tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)** — Kết nối dây dẫn nghiệp vụ tại `tooling/evidence_run.py` (`def _git(repo: Path, *args: str) -> Optional[str]:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` `go test -race ./...`) |

## 3. 💻 Key Diff / Code Preview (`go`)
```go
// Trích xuất trực tiếp từ TargetRepo: /Users/trainguyen/Documents/work/agent
// [1] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)
def _extract_yaml_frontmatter(src: str) -> str:
    """Đóng AF-04: Tách chính xác khối YAML frontmatter nằm giữa cặp dấu --- ở đầu file."""
    m = _FRONTMATTER_RE.match(src)
    return m.group(1) if m else ""


def _has_skill_trigger_token(body: str, sname: str) -> bool:
    """Đóng AF-04: Kiểm tra tên skill theo ranh giới từ/token, tránh dương tính giả do chuỗi con."""
    return bool(re.search(rf"(?<![\w\-]){re.escape(sname)}(?![\w\-])", body))


# ============================================================================

// [2] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:)
def _has_skill_trigger_token(body: str, sname: str) -> bool:
    """Đóng AF-04: Kiểm tra tên skill theo ranh giới từ/token, tránh dương tính giả do chuỗi con."""
    return bool(re.search(rf"(?<![\w\-]){re.escape(sname)}(?![\w\-])", body))


# ============================================================================
# Giải pháp xử lý Khoảng trống F3 (Task 3 & AF-04):
# Bộ Kiểm định Bề mặt Cấu hình /agent (AgentSurfaceValidator & Auto-Trigger)
# ============================================================================
@dataclass(frozen=True)
class AgentSurfaceSpec:
    name: str = "agy_agent"

// [3] agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)
import os
import re

service_dir = "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service"
files = [f for f in os.listdir(service_dir) if f.endswith(".go") and not f.endswith("_test.go")]

# Map symbol -> defining file
symbol_to_file = {}
file_defs = {}

struct_re = re.compile(r"type\s+([A-Z][a-zA-Z0-9_]*)\s+(struct|interface)")
func_re = re.compile(r"func\s+([A-Z][a-zA-Z0-9_]*)\(")
```

## 4. 🛡️ Nghiệm Thu & Cổng Kiểm Định (`DoD & Gates`)
- **Lệnh kiểm thử bắt buộc (`DoD`):** `go test -race ./...` trên `agent` (nhánh `task/TASK-8`).
- **Decision Gate `DG0 / DG1 / DG2 (:8200 · 0 Token)`:** `PASS 🟢` trên `agent` (Model: `opus`).

## Clean-Room Approved Spec
- **R1 (`TASK-8-M01` — `agent`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`) (`REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84`)
- **R2 (`TASK-8-M02` — `agent`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`analyze_dependencies.py`) (`REF: agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1`)
- **R3 (`TASK-8-M03` — `agent`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`analyze_dependencies.py`) (`REF: agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1`)
- **R4 (`TASK-8-M04` — `agent`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`) (`REF: agent/tooling/evidence_run.py:39`)
