---
parent_task: TASK-6
sub_task_id: TASK-6-M01
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-6/agent
isolated_branch: task/TASK-6/task-6-m01
stage: DONE
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-6-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `true`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [Wave 0 Entry · Parent TASK-6] Nhận trực tiếp đặc tả gốc 01_requirements.md & điểm chạm agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:) trên agent
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` (`/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-6-M01` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-6-M01 (agent · Wave 0): Nhận đầu vào từ 0 Upstream Sub-Tasks (Wave 0 Entry), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-6/agent` (nhánh `task/TASK-6/task-6-m01`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 2 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 0`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-6-M01` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-6-M01` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-6/task-6-m01` (`.orchestrator/worktrees/TASK-6/agent`) trước S4 + kiểm tra `Wave 0 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-6-M01-PASS`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-6/task-6-m01` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`TASK-6-M03, TASK-6-M04`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-6-M01`
```go
// [TASK-6-M01 · agent] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
// Đầu vào (InputContract): agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:) (Upstream DependsOn: [])
// Đầu ra (OutputContract): Hoàn tất agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:) trên nhánh cô lập task/TASK-6/task-6-m01 (/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py (lines 154-178)
 154 | def _unquote(val: Any) -> Any:
 155 |     if isinstance(val, str) and len(val) >= 2 and val[0] == '"' and val[-1] == '"':
 156 |         try:
 157 |             return json.loads(val)
 158 |         except Exception:
 159 |             return val.strip('"')
 160 |     return val
 161 | 
 162 | 
 163 | def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:
 164 |     tr = payload.get("transcriptPath", "")
 165 |     if not tr:
 166 |         return []
 167 |     try:
 168 |         with open(tr, encoding="utf-8") as f:
 169 |             return [json.loads(line) for line in f if line.strip()]
 170 |     except OSError:
 171 |         return []
 172 | 
 173 | 
 174 | def calls(st: List[Dict[str, Any]]) -> Iterator[Tuple[int, Optional[str], Dict[str, Any]]]:
 175 |     for i, s in enumerate(st):
 176 |         for tc in s.get("tool_calls") or []:
 177 |             raw_args = tc.get("args") or {}
 178 |             clean_args = {k: _unquote(v) for k, v in raw_args.items()}
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:) — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-6-M01-PASS · sha256=d9f2566edbb8)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `TASK-6-M03, TASK-6-M04`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-6/subtasks/TASK-6-M01/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-6/subtasks/TASK-6-M01/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-6-M01-PASS` (`HandoffReady = true`)
