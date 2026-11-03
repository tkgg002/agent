---
parent_task: TASK-12
sub_task_id: TASK-12-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-12/agent
isolated_branch: task/TASK-12/task-12-m03
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-12-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-12-M01 · agent · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:))
  - [TASK-12-M02 · agent · PENDING] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) trong worktree cô lập của `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-12-M03` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-12-M03 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-12-M01, TASK-12-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-12/agent` (nhánh `task/TASK-12/task-12-m03`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-12-M03` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-12-M03` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-12/task-12-m03` (`.orchestrator/worktrees/TASK-12/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-12-M03-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-12/task-12-m03` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-12)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-12-M03`
```go
// [TASK-12-M03 · agent] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`)
// Đầu vào (InputContract): agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:) (Upstream DependsOn: [TASK-12-M01 TASK-12-M02])
// Đầu ra (OutputContract): Hoàn tất agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:) trên nhánh cô lập task/TASK-12/task-12-m03 (/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py (lines 83-107)
  83 | class LessonEntry:
  84 |     idx: int
  85 |     date: str
  86 |     title: str
  87 |     line_no: int
  88 |     raw_block: str
  89 |     topic_code: str
  90 |     topic_file: str
  91 |     short_label: str
  92 | 
  93 | 
  94 | # G-08: lesson tripwire fields (máy đọc được) — ID ổn định, Trigger regex, Check tiền điều kiện, Require bằng chứng tuỳ chọn.
  95 | TRIP_FIELD_RE = re.compile(r"^- \*\*(ID|Trigger|Check|Require):\*\*\s*(.+?)\s*$", re.M)
  96 | TRIP_REQUIRED = ("id", "trigger", "check")
  97 | 
  98 | 
  99 | def parse_tripwire(entry: "LessonEntry") -> Optional[Dict[str, str]]:
 100 |     fields = {k.lower(): v for k, v in TRIP_FIELD_RE.findall(entry.raw_block)}
 101 |     if not fields:
 102 |         return None
 103 |     missing = [k for k in TRIP_REQUIRED if not fields.get(k)]
 104 |     if missing:
 105 |         raise RuntimeError(f"Tripwire thiếu {missing} tại lessons.md:L{entry.line_no}")
 106 |     for key in ("trigger", "require"):
 107 |         if fields.get(key):
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:) — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`build_lessons_index.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-12-M03-PENDING · sha256=c637a6ff032e)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-12)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-12/subtasks/TASK-12-M03/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-12/subtasks/TASK-12-M03/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-12-M03-PENDING` (`HandoffReady = false`)
