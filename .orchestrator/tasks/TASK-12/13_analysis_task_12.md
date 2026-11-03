# 🔍 [S2 · UDF-v1] Điều Tra Mã Nguồn Đa Repo & Bằng Chứng `file:line` — TASK-12: //127.0.0.1:8090: LIVE-PROCESS-HUD-01, ROLE-MODEL-ATTRIBUTION-01, CO-ARCHITECT-CHAT-01, TASK-INTAKE-DUAL-MODE-01
> 📌 **Task:** `TASK-12` · **Stage:** `S2 INVESTIGATE` · **Repos (1):** `agent` · **Files Scanned:** `45` · **Symbol Hits:** `8` · **Gate G2:** `VERIFIED 🟢`

## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)
- **Phạm vi điều tra (`Read-Only S2`):** Đã quét trực tiếp `45` file mã nguồn trên `1` repositories (`agent`) và phát hiện `8` điểm chạm thực tế (`file:line`).
- **Phân loại bằng chứng (`VERIFIED` vs `HYPOTHESIS`):** Toàn bộ điểm chạm của các Sub-Tasks dưới đây đều đã được đối chiếu trên đĩa (`VERIFIED`), loại bỏ mọi giả định suy diễn (`HYPOTHESIS = 0`).
- **Kịch bản Tái hiện `Gate G2 (Red -> Green)`:** Kiểm tra trạng thái trước khi sửa (`RED`) bằng `VerifySubTaskCodeOnDisk` và nghiệm thu sau khi sửa (`GREEN`) kèm biên lai `EV-...` (`exit=0`).

## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`VERIFIED` vs `HYPOTHESIS`)
| Mã / ID | Phân loại | Repo | Tham chiếu (`REF: file:line`) | Chẩn đoán Điểm chạm & Mục tiêu Khắc phục |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-12-M01`** | `VERIFIED` | `agent` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154` | Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`) (`DoD:` go test -race ./...) |
| **`TASK-12-M02`** | `VERIFIED` | `agent` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83` | Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`) (`DoD:` go test -race ./...) |
| **`TASK-12-M03`** | `VERIFIED` | `agent` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83` | Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`) (`DoD:` go test -race ./...) |
| **`TASK-12-M04`** | `VERIFIED` | `agent` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89` | Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`) (`DoD:` go test -race ./...) |
| `SYM-01` | `VERIFIED` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154` | Symbol thực tế: `def _unquote(val: Any) -> Any:` |
| `SYM-02` | `VERIFIED` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163` | Symbol thực tế: `def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:` |
| `SYM-03` | `VERIFIED` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83` | Symbol thực tế: `class LessonEntry:` |
| `SYM-04` | `VERIFIED` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:99` | Symbol thực tế: `def parse_tripwire(entry: "LessonEntry") -> Optional[Dict[str, str]]:` |
| `SYM-05` | `VERIFIED` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89` | Symbol thực tế: `def norm_ws(text: str) -> str:` |
| `SYM-06` | `VERIFIED` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling` | `REF: agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:93` | Symbol thực tế: `def now_iso() -> str:` |

## 3. 💻 Code Demo / Trích Đoạn Mã Nguồn Thực Tế Trên Đĩa (`VERIFIED Snippets`)
```go
// [1] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 — def _unquote(val: Any) -> Any:
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

// [2] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 — def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:
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

// [3] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 — class LessonEntry:
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

// [4] agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:99 — def parse_tripwire(entry: "LessonEntry") -> Optional[Dict[str, str]]:
def parse_tripwire(entry: "LessonEntry") -> Optional[Dict[str, str]]:
    fields = {k.lower(): v for k, v in TRIP_FIELD_RE.findall(entry.raw_block)}
    if not fields:
        return None
    missing = [k for k in TRIP_REQUIRED if not fields.get(k)]
    if missing:
        raise RuntimeError(f"Tripwire thiếu {missing} tại lessons.md:L{entry.line_no}")
    for key in ("trigger", "require"):
        if fields.get(key):
            try:
                re.compile(fields[key])
            except re.error as exc:
```

## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)
- **Xác nhận Lesson Tripwires (`H4t`):**
- Không có tripwire rủi ro cao bị kích hoạt (`tripwires.json` = CLEAN).
- **Đối chiếu Bài học Lịch sử (`350` indexed lessons):**
- **`L040` [2026-10-26 · `T1:L278`] Omitt…:** (1) Dừng lại ngay lập tức (Rule #5), ghi bài học vào `lessons.md` và chạy `build_lessons_index.py`; (2) Mở vòng Re-Plan `Task 12` (`S1 -> S2 -> S3`) trong workspace `feature-modula...
- **`L011` [2026-10-27 · `T1:L74`] Bypas…:** Khi User [A] đối thoại hoặc chỉ đạo thay đổi kiến trúc trong khung **Co-Architect Studio (`/brain-delegate` · `Slot A` · `gemini-4-argon (antigravity)`)** trên `ai-orchestrator` [B...
- **`L027` [2026-10-27 · `T2:L187`] Stuff…:** Khi mở rộng một hệ thống điều phối đa tác nhân (`Multi-Agent Orchestrator` [A]) để bổ sung định tuyến model động (`Dynamic Model Routing`), chạy song song theo tầng `Wave DAG` bằng...
- **`L007` [2026-10-27 · `T2:L46`] -> Hậu quả [X]: Vi phạm kỷ luật Single-Responsibility & `NOW <= HEAD`, tạo rác trùng lặp khó bảo trì, để lọt nội dung mâu thuẫn (`Knex/PG Pool` & đè V1) ngay trong file vật lý `.orchestrator/tasks/TASK-13/09_tasks_solution.md`, và báo cáo hoàn thành khi tiến trình `:8090` vẫn chạy binary cũ [Y]. **Đúng:** (1) Mọi thay đổi bắt buộc đặt đúng layer và tách thành các module đơn trách nhiệm gọn nhẹ (`repo_tech_profile.go` chuyên trách normalize path & nhận diện Tech Stack/DoD/Fence dùng chung toàn package `webui`, `task_revise_workflow.go` chuyên trách luồng `/revise` & Clean-Room Spec, `subtask_git_apply.go <= HEAD 312 dòng`); (2) Xóa bỏ 100% câu chữ mẫu nhắc tới `Knex` hoặc sửa đè `trans-his.model.ts` trong `BuildUnifiedS3SolutionDoc`, normalize relative path ở mọi cột bảng Sub-Tasks, và chuẩn hóa `TestDoD` chính xác tuyệt đối (`npm run build && npm test` cho `package.json`, `go test -race ./...` cho `go.mod`) kèm test assertion nghiêm ngặt; (3) Kiểm chứng trực tiếp trên repo thật (`/Users/trainguyen/Documents/work/data-hub/centralized-export-service`), hồ sơ `.orchestrator/tasks/TASK-13` thật trên đĩa và trace giá trị JSON runtime thật trên tiến trình `:8090` đã nạp binary mới.:** (1) Dừng ngay theo Rule #5, ghi bài học vào `lessons.md`, kiểm tra `wc -l` (`2536 -> 2543`) và chạy `build_lessons_index.py`; (2) Thực hiện vòng lặp tự chữa lành `S5 -> S3 -> S4 ->...
- **`L029` [2026-10-26 · `T1:L201`] Arbit…:** Khi xây dựng giao diện điều phối tác vụ đa tác tử (`[A]` — `ai-orchestrator` Web UI trên `:8090`), nếu Agent chỉ tập trung thêm các nút kích hoạt chạy (`Start`, `Plan`, `Approve`, ...