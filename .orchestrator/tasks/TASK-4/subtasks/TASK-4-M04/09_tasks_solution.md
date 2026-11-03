---
parent_task: TASK-4
sub_task_id: TASK-4-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-4/agent
isolated_branch: task/TASK-4/task-4-m04
stage: DONE
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-4-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`mutation_gate.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `true`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-4-M01 · agent · HANDOFF_READY ✓] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`migration_fix_tz_worker_schedule.sql`) (agent/memory/workspaces/InvestigateTransformSchedulerStuck/migration_fix_tz_worker_schedule.sql:1 (-- Migration: fix Bug B — TZ freeze schedule))
  - [TASK-4-M02 · agent · HANDOFF_READY ✓] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`laya_gate.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py` (`def sha256_bytes(data: bytes) -> str:`) và viết integration test xác minh đầu-cuối trên `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:)` (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-4-M04` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-4-M04 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-4-M01, TASK-4-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-4/agent` (nhánh `task/TASK-4/task-4-m04`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-4-M04` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-4-M04` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-4/task-4-m04` (`.orchestrator/worktrees/TASK-4/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-4-M04-PASS`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-4/task-4-m04` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-4)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-4-M04`
```go
// [TASK-4-M04 · agent] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`mutation_gate.py`)
// Đầu vào (InputContract): agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:) (Upstream DependsOn: [TASK-4-M01 TASK-4-M02])
// Đầu ra (OutputContract): Hoàn tất agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:) trên nhánh cô lập task/TASK-4/task-4-m04 (/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py (lines 32-56)
  32 | def sha256_bytes(data: bytes) -> str:
  33 |     return hashlib.sha256(data).hexdigest()
  34 | 
  35 | 
  36 | def _invalidate_pyc(fpath: Path) -> None:
  37 |     if fpath.suffix != ".py":
  38 |         return
  39 |     pycache = fpath.parent / "__pycache__"
  40 |     if pycache.is_dir():
  41 |         for pyc in pycache.glob(f"{fpath.stem}.*.pyc"):
  42 |             try:
  43 |                 pyc.unlink()
  44 |             except OSError:
  45 |                 pass
  46 | 
  47 | 
  48 | def resolve_target_file(raw_path: str, mutations_file: Path) -> Optional[Path]:
  49 |     p = Path(raw_path.strip())
  50 |     if p.is_absolute() and p.is_file():
  51 |         return p.resolve()
  52 |     for base in (Path.cwd(), AGENT_ROOT, mutations_file.resolve().parent):
  53 |         cand = (base / p).resolve()
  54 |         if cand.is_file():
  55 |             return cand
  56 |     return None
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`mutation_gate.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-4-M04-PASS · sha256=9bcc5ae3bc10)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-4)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-4/subtasks/TASK-4-M04/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-4/subtasks/TASK-4-M04/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-4-M04-PASS` (`HandoffReady = true`)
