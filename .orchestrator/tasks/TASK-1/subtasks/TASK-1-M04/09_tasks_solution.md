---
parent_task: TASK-1
sub_task_id: TASK-1-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-1/agent
isolated_branch: task/TASK-1/task-1-m04
stage: DONE
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-1-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `false`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-1-M01 · agent · PENDING] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`query_db.go`) (agent/memory/workspaces/ReconHealStaleReport copy/query_db.go:12 (type MappingRuleV2 struct {))
  - [TASK-1-M02 · agent · HANDOFF_READY ✓] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`) (agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_agy_guard.py` (`class TestAgyGuard(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)` (`/Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_agy_guard.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-1-M04` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-1-M04 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-1-M01, TASK-1-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-1/agent` (nhánh `task/TASK-1/task-1-m04`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-1-M04` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-1-M04` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-1/task-1-m04` (`.orchestrator/worktrees/TASK-1/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-1-M04-PASS`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-1/task-1-m04` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-1)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-1-M04`
```go
// [TASK-1-M04 · agent] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
// Đầu vào (InputContract): agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):) (Upstream DependsOn: [TASK-1-M01 TASK-1-M02])
// Đầu ra (OutputContract): Hoàn tất agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):) trên nhánh cô lập task/TASK-1/task-1-m04 (/Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_agy_guard.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_agy_guard.py (lines 15-39)
  15 | class TestAgyGuard(unittest.TestCase):
  16 |     def setUp(self) -> None:
  17 |         self._tmp = tempfile.TemporaryDirectory()
  18 |         self.tmp = Path(self._tmp.name)
  19 |         self.mem_root = self.tmp / "memory"
  20 |         self.ws_root = self.mem_root / "workspaces"
  21 |         self.state_dir = self.tmp / "state"
  22 |         self.ws_root.mkdir(parents=True, exist_ok=True)
  23 |         self.state_dir.mkdir(parents=True, exist_ok=True)
  24 | 
  25 |     def tearDown(self) -> None:
  26 |         self._tmp.cleanup()
  27 | 
  28 |     def _write_transcript(self, lines: list[dict]) -> str:
  29 |         tr = self.tmp / "transcript.jsonl"
  30 |         tr.write_text("\n".join(json.dumps(x) for x in lines) + "\n", encoding="utf-8")
  31 |         return str(tr)
  32 | 
  33 |     # 1. H2: Overwrite existing memory file -> deny
  34 |     def test_01_h2_overwrite_existing_memory_denied(self) -> None:
  35 |         lessons = self.mem_root / "global" / "lessons.md"
  36 |         lessons.parent.mkdir(parents=True, exist_ok=True)
  37 |         lessons.write_text("# Lessons\n", encoding="utf-8")
  38 |         res = agy_guard.handle_pre_write(
  39 |             {
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-1-M04-PASS · sha256=be01a8ae8438)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-1)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-1/subtasks/TASK-1-M04/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-1/subtasks/TASK-1-M04/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-1-M04-PASS` (`HandoffReady = true`)
