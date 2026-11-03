---
parent_task: TASK-8
sub_task_id: TASK-8-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-8/agent
isolated_branch: task/TASK-8/task-8-m04
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-8-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-8-M01 · agent · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:))
  - [TASK-8-M02 · agent · PENDING] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`analyze_dependencies.py`) (agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `tooling/evidence_run.py` (`def _git(repo: Path, *args: str) -> Optional[str]:`) và viết integration test xác minh đầu-cuối trên `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)` (`/Users/trainguyen/Documents/work/agent/tooling/evidence_run.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-8-M04` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-8-M04 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-8-M01, TASK-8-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-8/agent` (nhánh `task/TASK-8/task-8-m04`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-8-M04` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-8-M04` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-8/task-8-m04` (`.orchestrator/worktrees/TASK-8/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-8-M04-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-8/task-8-m04` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-8)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-8-M04`
```go
// [TASK-8-M04 · agent] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)
// Đầu vào (InputContract): agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:) (Upstream DependsOn: [TASK-8-M01 TASK-8-M02])
// Đầu ra (OutputContract): Hoàn tất agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:) trên nhánh cô lập task/TASK-8/task-8-m04 (/Users/trainguyen/Documents/work/agent/tooling/evidence_run.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/tooling/evidence_run.py (lines 39-63)
  39 | def _git(repo: Path, *args: str) -> Optional[str]:
  40 |     try:
  41 |         r = subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=True, timeout=20)
  42 |     except (OSError, subprocess.TimeoutExpired):
  43 |         return None
  44 |     return r.stdout if r.returncode == 0 else None
  45 | 
  46 | 
  47 | def source_fingerprint(cwd: Path) -> str:
  48 |     """sha256 nội dung file source của repo chứa cwd (tracked + untracked, bỏ memory/). Commit không đổi giá trị."""
  49 |     root_out = _git(cwd, "rev-parse", "--show-toplevel")
  50 |     root = Path(root_out.strip()) if root_out else cwd.resolve()
  51 |     listing = _git(root, "ls-files", "-co", "--exclude-standard")
  52 |     files = sorted(
  53 |         f for f in (listing.splitlines() if listing is not None else [])
  54 |         if f.endswith(SRC_SUFFIXES) and not f.startswith("memory/") and "/__pycache__/" not in f
  55 |     )
  56 |     h = hashlib.sha256()
  57 |     for rel in files:
  58 |         p = root / rel
  59 |         try:
  60 |             data = p.read_bytes()
  61 |         except OSError:
  62 |             continue
  63 |         h.update(rel.encode("utf-8") + b"\0" + hashlib.sha256(data).digest())
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/tooling/evidence_run.py:39 (def _git(repo: Path, *args: str) -> Optional[str]:) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`evidence_run.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-8-M04-PENDING · sha256=1fddb12db8d2)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-8)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-8/subtasks/TASK-8-M04/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-8/subtasks/TASK-8-M04/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-8-M04-PENDING` (`HandoffReady = false`)
