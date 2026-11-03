---
parent_task: TASK-7
sub_task_id: TASK-7-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-7/agent
isolated_branch: task/TASK-7/task-7-m04
stage: DONE
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-7-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`proposer_axes.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-7-M01 · agent · HANDOFF_READY ✓] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:))
  - [TASK-7-M02 · agent · PENDING] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`__init__.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent."""))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py` (`def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:`) và viết integration test xác minh đầu-cuối trên `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:)` (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-7-M04` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-7-M04 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-7-M01, TASK-7-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-7/agent` (nhánh `task/TASK-7/task-7-m04`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-7-M04` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-7-M04` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-7/task-7-m04` (`.orchestrator/worktrees/TASK-7/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-7-M04-PASS`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-7/task-7-m04` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-7)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-7-M04`
```go
// [TASK-7-M04 · agent] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`proposer_axes.py`)
// Đầu vào (InputContract): agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:) (Upstream DependsOn: [TASK-7-M01 TASK-7-M02])
// Đầu ra (OutputContract): Hoàn tất agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:) trên nhánh cô lập task/TASK-7/task-7-m04 (/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py (lines 36-60)
  36 | def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:
  37 |     """Đóng AF-01: Hợp nhất 3 chiều bảo toàn thay đổi và phép xóa của cả 2 trục trực giao INV & DOM."""
  38 |     merged: Tree = dict(base_tree)
  39 |     for branch in (inv_tree, dom_tree):
  40 |         deleted = set(base_tree) - set(branch)
  41 |         for k in deleted:
  42 |             merged.pop(k, None)
  43 |         for k, v in branch.items():
  44 |             if k not in base_tree or branch[k] != base_tree[k]:
  45 |                 merged[k] = v
  46 |     return merged
  47 | 
  48 | 
  49 | def validate_orthogonal_axis_edit(
  50 |     dim: str,
  51 |     before: Tree,
  52 |     after: Tree,
  53 |     surface: Optional[Any] = None,
  54 | ) -> Tuple[bool, str]:
  55 |     """Đóng F2, AF-04, AF-06: Kiểm tra rào chắn trực giao INV vs DOM, KEEP_BASE và trần UTF-8 bytes."""
  56 |     always_on_files: Tuple[str, ...] = getattr(
  57 |         surface, "always_on", ("GEMINI.md", "OPERATOR_MAP.md")
  58 |     )
  59 |     always_on_max: int = int(getattr(surface, "always_on_max", 24_000))
  60 | 
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`proposer_axes.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-7-M04-PASS · sha256=a881ac273bbd)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-7)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-7/subtasks/TASK-7-M04/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-7/subtasks/TASK-7-M04/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-7-M04-PASS` (`HandoffReady = true`)
