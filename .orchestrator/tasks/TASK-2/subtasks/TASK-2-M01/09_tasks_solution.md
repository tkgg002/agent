---
parent_task: TASK-2
sub_task_id: TASK-2-M01
wave_index: 0
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-2/agent
isolated_branch: task/TASK-2/task-2-m01
stage: READY
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-2-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`__init__.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""")` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `true`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [Wave 0 Entry · Parent TASK-2] Nhận trực tiếp đặc tả gốc 01_requirements.md & điểm chạm agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""") trên agent
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py` (`"""Orchestrator & Harness2 Recursive Evolution Package for /agent."""`) trong worktree cô lập của `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""")` (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-2-M01` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-2-M01 (agent · Wave 0): Nhận đầu vào từ 0 Upstream Sub-Tasks (Wave 0 Entry), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-2/agent` (nhánh `task/TASK-2/task-2-m01`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 2 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 0`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""")` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-2-M01` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-2-M01` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-2/task-2-m01` (`.orchestrator/worktrees/TASK-2/agent`) trước S4 + kiểm tra `Wave 0 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-2-M01-PENDING`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-2/task-2-m01` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`TASK-2-M03, TASK-2-M04`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-2-M01`
```go
// [TASK-2-M01 · agent] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`__init__.py`)
// Đầu vào (InputContract): agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""") (Upstream DependsOn: [])
// Đầu ra (OutputContract): Hoàn tất agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""") trên nhánh cô lập task/TASK-2/task-2-m01 (/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py (lines 1-13)
   1 | """Orchestrator & Harness2 Recursive Evolution Package for /agent."""
   2 | from tooling.orchestrator.proposer_axes import (
   3 |     CONTEXT_HIERARCHY_ORDER,
   4 |     three_way_merge_trees,
   5 |     validate_orthogonal_axis_edit,
   6 | )
   7 | 
   8 | __all__ = [
   9 |     "CONTEXT_HIERARCHY_ORDER",
  10 |     "three_way_merge_trees",
  11 |     "validate_orthogonal_axis_edit",
  12 | ]
  13 | 
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent.""") — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`__init__.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-2-M01-PENDING · sha256=392349326599)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `TASK-2-M03, TASK-2-M04`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-2/subtasks/TASK-2-M01/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-2/subtasks/TASK-2-M01/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-2-M01-PENDING` (`HandoffReady = false`)
