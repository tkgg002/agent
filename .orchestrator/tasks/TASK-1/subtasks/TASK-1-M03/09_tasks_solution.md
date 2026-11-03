---
parent_task: TASK-1
sub_task_id: TASK-1-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-1/agent
isolated_branch: task/TASK-1/task-1-m03
stage: DONE
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-1-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-1-M01 · agent · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`query_db.go`) (agent/memory/workspaces/ReconHealStaleReport copy/query_db.go:12 (type MappingRuleV2 struct {))
  - [TASK-1-M02 · agent · HANDOFF_READY ✓] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`) (agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) trong worktree cô lập của `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` (`/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-1-M03` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-1-M03 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-1-M01, TASK-1-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-1/agent` (nhánh `task/TASK-1/task-1-m03`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-1-M03` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-1-M03` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-1/task-1-m03` (`.orchestrator/worktrees/TASK-1/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-1-M03-PASS`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-1/task-1-m03` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-1)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-1-M03`
```go
// [TASK-1-M03 · agent] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)
// Đầu vào (InputContract): agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:) (Upstream DependsOn: [TASK-1-M01 TASK-1-M02])
// Đầu ra (OutputContract): Hoàn tất agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:) trên nhánh cô lập task/TASK-1/task-1-m03 (/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py)
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
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-1-M03-PASS · sha256=d9f2566edbb8)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-1)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-1/subtasks/TASK-1-M03/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-1/subtasks/TASK-1-M03/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-1-M03-PASS` (`HandoffReady = true`)
