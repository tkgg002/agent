---
parent_task: TASK-2
sub_task_id: TASK-2-M03
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-2/agent
isolated_branch: task/TASK-2/task-2-m03
stage: DONE
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-2-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-2-M01 · agent · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`__init__.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/__init__.py:1 ("""Orchestrator & Harness2 Recursive Evolution Package for /agent."""))
  - [TASK-2-M02 · agent · HANDOFF_READY ✓] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`harness2_engine.py`) (agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (`def _has_skill_trigger_token(body: str, sname: str) -> bool:`) trong worktree cô lập của `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:)` (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-2-M03` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-2-M03 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-2-M01, TASK-2-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-2/agent` (nhánh `task/TASK-2/task-2-m03`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-2-M03` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-2-M03` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-2/task-2-m03` (`.orchestrator/worktrees/TASK-2/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-2-M03-PASS`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-2/task-2-m03` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-2)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-2-M03`
```go
// [TASK-2-M03 · agent] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)
// Đầu vào (InputContract): agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:) (Upstream DependsOn: [TASK-2-M01 TASK-2-M02])
// Đầu ra (OutputContract): Hoàn tất agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:) trên nhánh cô lập task/TASK-2/task-2-m03 (/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py (lines 90-114)
  90 | def _has_skill_trigger_token(body: str, sname: str) -> bool:
  91 |     """Đóng AF-04: Kiểm tra tên skill theo ranh giới từ/token, tránh dương tính giả do chuỗi con."""
  92 |     return bool(re.search(rf"(?<![\w\-]){re.escape(sname)}(?![\w\-])", body))
  93 | 
  94 | 
  95 | # ============================================================================
  96 | # Giải pháp xử lý Khoảng trống F3 (Task 3 & AF-04):
  97 | # Bộ Kiểm định Bề mặt Cấu hình /agent (AgentSurfaceValidator & Auto-Trigger)
  98 | # ============================================================================
  99 | @dataclass(frozen=True)
 100 | class AgentSurfaceSpec:
 101 |     name: str = "agy_agent"
 102 |     always_on: Tuple[str, ...] = ("GEMINI.md", "OPERATOR_MAP.md")
 103 |     always_on_max: int = 24_000
 104 |     skill_dir: str = "skills"
 105 |     autotrigger_file: str = "OPERATOR_MAP.md"
 106 | 
 107 | 
 108 | def load_agent_surface_tree(agent_root: Path = AGENT_ROOT) -> Tree:
 109 |     """Đọc cây cấu hình /agent thực tế từ đĩa phục vụ kiểm định bề mặt F3."""
 110 |     tree: Tree = {}
 111 |     for rel in ("GEMINI.md", "OPERATOR_MAP.md"):
 112 |         p = agent_root / rel
 113 |         if p.is_file():
 114 |             tree[rel] = p.read_text(encoding="utf-8")
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:) — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`harness2_engine.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-2-M03-PASS · sha256=126be8edaed1)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-2)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-2/subtasks/TASK-2-M03/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-2/subtasks/TASK-2-M03/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-2-M03-PASS` (`HandoffReady = true`)
