---
parent_task: TASK-12
sub_task_id: TASK-12-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-12/agent
isolated_branch: task/TASK-12/task-12-m04
stage: READY
role: ⚖️ Cross-Model Laya Gate Reviewer (Slot H · /laya-router)
model: convaiinnovations/laya (local :8200)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-12-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-12-M01 · agent · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:))
  - [TASK-12-M02 · agent · PENDING] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (`def norm_ws(text: str) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)` (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-12-M04` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-12-M04 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-12-M01, TASK-12-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-12/agent` (nhánh `task/TASK-12/task-12-m04`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-12-M04` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-12-M04` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-12/task-12-m04` (`.orchestrator/worktrees/TASK-12/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-12-M04-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-12/task-12-m04` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-12)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-12-M04`
```go
// [TASK-12-M04 · agent] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)
// Đầu vào (InputContract): agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:) (Upstream DependsOn: [TASK-12-M01 TASK-12-M02])
// Đầu ra (OutputContract): Hoàn tất agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:) trên nhánh cô lập task/TASK-12/task-12-m04 (/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py (lines 89-113)
  89 | def norm_ws(text: str) -> str:
  90 |     return re.sub(r"\s+", " ", text).strip()
  91 | 
  92 | 
  93 | def now_iso() -> str:
  94 |     return dt.datetime.now(dt.timezone(dt.timedelta(hours=7))).isoformat(timespec="seconds")
  95 | 
  96 | 
  97 | def resolve_ref_path(raw_path: str, base_dirs: List[Path]) -> Optional[Path]:
  98 |     p = Path(raw_path)
  99 |     if p.is_absolute():
 100 |         return p if p.is_file() else None
 101 |     for base in base_dirs:
 102 |         cand = (base / p).resolve()
 103 |         if cand.is_file():
 104 |             return cand
 105 |     return None
 106 | 
 107 | 
 108 | def verify_evidence(claim_obj: Dict[str, Any], base_dirs: Optional[List[Path]] = None) -> Tuple[bool, str]:
 109 |     """Verify that every REF: path:start-end exists and its quoted lines match the actual file."""
 110 |     bases = base_dirs or [Path.cwd(), AGENT_ROOT, AGENT_ROOT.parent]
 111 |     evidence = claim_obj.get("evidence", "") or ""
 112 |     blocks = re.split(r"(?m)^(?=REF:|DEMO:)", evidence)
 113 |     refs = [b for b in blocks if b.startswith("REF:")]
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-12-M04-PENDING · sha256=2dd4ef3e87b1)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-12)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-12/subtasks/TASK-12-M04/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-12/subtasks/TASK-12-M04/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-12-M04-PENDING` (`HandoffReady = false`)
