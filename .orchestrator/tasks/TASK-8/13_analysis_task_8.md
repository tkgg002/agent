# 🔍 [S2 · UDF-v1] Điều Tra Mã Nguồn Đa Repo & Bằng Chứng `file:line` — TASK-8: S3 Re-Plan — Hợp nhất 1 Khung Ứng Dụng Duy Nhất Single Unified IDE Shell, Xây dựng Bộ Máy Thực Thi Tự Động Per-Task Multi-Stage Autopilot Core Engine & Cô lập Vật lý 4 Lớp Git Worktree + Task Dir + Terminal + Scoped Autopilot
> 📌 **Task:** `TASK-8` · **Stage:** `S2 INVESTIGATE` · **Repos (1):** `agent` · **Files Scanned:** `45` · **Symbol Hits:** `8` · **Gate G2:** `VERIFIED 🟢`

## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)
- **Phạm vi điều tra (`Read-Only S2`):** Đã quét trực tiếp `45` file mã nguồn trên `1` repositories (`agent`) và phát hiện `8` điểm chạm thực tế (`file:line`).
- **Phân loại bằng chứng (`VERIFIED` vs `HYPOTHESIS`):** Toàn bộ điểm chạm của các Sub-Tasks dưới đây đều đã được đối chiếu trên đĩa (`VERIFIED`), loại bỏ mọi giả định suy diễn (`HYPOTHESIS = 0`).
- **Kịch bản Tái hiện `Gate G2 (Red -> Green)`:** Kiểm tra trạng thái trước khi sửa (`RED`) bằng `VerifySubTaskCodeOnDisk` và nghiệm thu sau khi sửa (`GREEN`) kèm biên lai `EV-...` (`exit=0`).

## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`VERIFIED` vs `HYPOTHESIS`)
| Mã / ID | Phân loại | Repo | Tham chiếu (`REF: file:line`) | Chẩn đoán Điểm chạm & Mục tiêu Khắc phục |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-8-M01`** | `VERIFIED` | `agent` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84` | Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`) (`DoD:` go test -race ./...) |
| **`TASK-8-M02`** | `VERIFIED` | `agent` | `REF: agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1` | Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`analyze_dependencies.py`) (`DoD:` go test -race ./...) |
| **`TASK-8-M03`** | `VERIFIED` | `agent` | `REF: agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1` | Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`analyze_dependencies.py`) (`DoD:` go test -race ./...) |
| **`TASK-8-M04`** | `VERIFIED` | `agent` | `REF: agent/tooling/evidence_run.py:39` | Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`) (`DoD:` go test -race ./...) |
| `SYM-01` | `VERIFIED` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84` | Symbol thực tế: `def _extract_yaml_frontmatter(src: str) -> str:` |
| `SYM-02` | `VERIFIED` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90` | Symbol thực tế: `def _has_skill_trigger_token(body: str, sname: str) -> bool:` |
| `SYM-03` | `VERIFIED` | `agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan` | `REF: agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1` | Symbol thực tế: `import os` |
| `SYM-04` | `VERIFIED` | `agent/tooling` | `REF: agent/tooling/evidence_run.py:39` | Symbol thực tế: `def _git(repo: Path, *args: str) -> Optional[str]:` |
| `SYM-05` | `VERIFIED` | `agent/tooling` | `REF: agent/tooling/evidence_run.py:47` | Symbol thực tế: `def source_fingerprint(cwd: Path) -> str:` |
| `SYM-06` | `VERIFIED` | `agent/tooling/orchestrator` | `REF: agent/tooling/orchestrator/harness2_engine.py:84` | Symbol thực tế: `def _extract_yaml_frontmatter(src: str) -> str:` |

## 3. 💻 Code Demo / Trích Đoạn Mã Nguồn Thực Tế Trên Đĩa (`VERIFIED Snippets`)
```go
// [1] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 — def _extract_yaml_frontmatter(src: str) -> str:
def _extract_yaml_frontmatter(src: str) -> str:
    """Đóng AF-04: Tách chính xác khối YAML frontmatter nằm giữa cặp dấu --- ở đầu file."""
    m = _FRONTMATTER_RE.match(src)
    return m.group(1) if m else ""


def _has_skill_trigger_token(body: str, sname: str) -> bool:
    """Đóng AF-04: Kiểm tra tên skill theo ranh giới từ/token, tránh dương tính giả do chuỗi con."""
    return bool(re.search(rf"(?<![\w\-]){re.escape(sname)}(?![\w\-])", body))


# ============================================================================

// [2] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 — def _has_skill_trigger_token(body: str, sname: str) -> bool:
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

// [3] agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 — import os
import os
import re

service_dir = "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service"
files = [f for f in os.listdir(service_dir) if f.endswith(".go") and not f.endswith("_test.go")]

# Map symbol -> defining file
symbol_to_file = {}
file_defs = {}

struct_re = re.compile(r"type\s+([A-Z][a-zA-Z0-9_]*)\s+(struct|interface)")
func_re = re.compile(r"func\s+([A-Z][a-zA-Z0-9_]*)\(")

// [4] agent/tooling/evidence_run.py:39 — def _git(repo: Path, *args: str) -> Optional[str]:
def _git(repo: Path, *args: str) -> Optional[str]:
    try:
        r = subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=True, timeout=20)
    except (OSError, subprocess.TimeoutExpired):
        return None
    return r.stdout if r.returncode == 0 else None


def source_fingerprint(cwd: Path) -> str:
    """sha256 nội dung file source của repo chứa cwd (tracked + untracked, bỏ memory/). Commit không đổi giá trị."""
    root_out = _git(cwd, "rev-parse", "--show-toplevel")
    root = Path(root_out.strip()) if root_out else cwd.resolve()
```

## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)
- **Xác nhận Lesson Tripwires (`H4t`):**
- Không có tripwire rủi ro cao bị kích hoạt (`tripwires.json` = CLEAN).
- **Đối chiếu Bài học Lịch sử (`350` indexed lessons):**
- **`L053` [2026-10-23 · `T1:L369`] Viết…:** Khi thiết kế và trình bày một hệ thống nền tảng điều phối đa tác nhân chạy ngầm theo hàng đợi (`Event/Queue-Driven Multi-Agent Orchestrator` [A]) có tích hợp bộ quy tắc quản trị hi...
- **`L020` [2026-10-27 · `T1:L137`] S5:** Khi xây dựng luồng thực thi tự động (`S1 -> S2 -> DG0 -> S3 -> APPROVE -> S4 -> S5 -> DG3 -> DONE`) cho hệ thống Orchestrator [A] quản lý nhiều loại Task (Single-Repo, Multi-Repo, ...
- **`L027` [2026-10-27 · `T2:L187`] Stuff…:** Khi mở rộng một hệ thống điều phối đa tác nhân (`Multi-Agent Orchestrator` [A]) để bổ sung định tuyến model động (`Dynamic Model Routing`), chạy song song theo tầng `Wave DAG` bằng...
- **`L029` [2026-10-26 · `T1:L201`] Arbit…:** Khi xây dựng giao diện điều phối tác vụ đa tác tử (`[A]` — `ai-orchestrator` Web UI trên `:8090`), nếu Agent chỉ tập trung thêm các nút kích hoạt chạy (`Start`, `Plan`, `Approve`, ...
- **`L044` [2026-10-26 · `T1:L306`] Stuff…:** Khi hiện thực hóa quy trình xây dựng lại phòng sạch (`Clean-Room Rebuild Protocol` kiểu REA — *"Model thấy code gốc thì không bao giờ được viết code, và model viết code thì không b...