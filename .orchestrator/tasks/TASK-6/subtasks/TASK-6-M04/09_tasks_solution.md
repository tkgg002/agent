---
parent_task: TASK-6
sub_task_id: TASK-6-M04
wave_index: 1
target_repo: /Users/trainguyen/Documents/work/agent
worktree_path: .orchestrator/worktrees/TASK-6/agent
isolated_branch: task/TASK-6/task-6-m04
stage: DONE
role: 🧪 Cross-Model QA & E2E Auditor (Slot K · /qa-agent)
model: claude-opus-4-6 (antigravity)
qa_auditor_model: claude-opus-4-6 (antigravity)
laya_gate_model: convaiinnovations/laya (local :8200)
max_self_heal_retries: 3
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: TASK-6-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)

## 1. 📥 Hợp Đồng Đầu Vào (`InputContract`)
- **Điểm chạm mã nguồn (`SourceContext`):** `agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)` (`agent`)
- **Trạng thái Đầu vào từ Wave trước (`UpstreamReady`):** `true`
- **Sản phẩm bàn giao từ Wave trước (`UpstreamDeliverables`):**
  - [TASK-6-M01 · agent · HANDOFF_READY ✓] Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`) (agent/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:))
  - [TASK-6-M02 · agent · HANDOFF_READY ✓] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_agy_guard.py`) (agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):))
- **Đặc tả nghiệp vụ đầu vào (`InputSpec`):** Kết nối dây dẫn nghiệp vụ tại `hooks/agy/tests/test_claim_linter.py` (`class TestClaimLinter(unittest.TestCase):`) và viết integration test xác minh đầu-cuối trên `agent`.

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`)
- **Kiểm định thực tế trên đĩa (`C-REQ / C-REALITY / C-CONTRA`):** Đã phản biện ngược điểm chạm `agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)` (`/Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_claim_linter.py`) và kiểm chứng với Local Neural Gate `/laya_router` (`convaiinnovations/laya (local :8200)`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `TASK-6-M04` (`S3 Single Optimal Solution`)
- **Tóm tắt giải pháp:** Giải pháp chuyên biệt cho TASK-6-M04 (agent · Wave 1): Nhận đầu vào từ 2 Upstream Sub-Tasks (TASK-6-M01, TASK-6-M02), gắn Worktree Slot C tại `.orchestrator/worktrees/TASK-6/agent` (nhánh `task/TASK-6/task-6-m04`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (go test -race ./...) và bàn giao đầu ra cho 0 Sub-Tasks ở Wave kế tiếp.
- **Luồng thực thi cô lập (`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`):**
  1. **Start (`S0/S1`):** Khóa hợp đồng đầu vào `InputContract` và kiểm tra sẵn sàng từ `Wave 1`.
  2. **S2.5 Pre-Plan Critique (`DG0 :8200`) -> Plan & DG1/DG2 (`S2/S3`):** Phản biện điểm chạm `agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):)` bằng `DG0` rồi kiểm định Code Demo của riêng `TASK-6-M04` bằng Local Laya Gate (`convaiinnovations/laya (local :8200)`, 0 LLM token).
  3. **You (`Human Gate`):** Cổng duyệt phương án cho `TASK-6-M04` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`Slot C Worktree + S4`):** Khởi tạo Git Worktree `task/TASK-6/task-6-m04` (`.orchestrator/worktrees/TASK-6/agent`) trước S4 + kiểm tra `Wave 1 Contract Sync` + `Laya DG2 (:8200)`.
  5. **Local QA & DG3 (`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`):** Chạy `5A Hard Gate` (`go test -race ./...`, 0 LLM calls nếu lỗi) trước khi gọi `5B Cross-Model QA` (`claude-opus-4-6 (antigravity)`, `Max 3 Retries`, biên lai `EV-SUB-TASK-6-M04-PASS`).
  6. **Finalize & Diff Review (`S6`):** Đóng gói nhánh `task/TASK-6/task-6-m04` (`No-Push H5`) và mở khóa đầu vào cho Wave kế tiếp (`Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-6)`).

## 3. 💻 Concrete Code Demo Cho Riêng `TASK-6-M04`
```go
// [TASK-6-M04 · agent] Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)
// Đầu vào (InputContract): agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):) (Upstream DependsOn: [TASK-6-M01 TASK-6-M02])
// Đầu ra (OutputContract): Hoàn tất agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):) trên nhánh cô lập task/TASK-6/task-6-m04 (/Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_claim_linter.py)
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_claim_linter.py (lines 17-41)
  17 | class TestClaimLinter(unittest.TestCase):
  18 |     def setUp(self) -> None:
  19 |         self._t = tempfile.TemporaryDirectory()
  20 |         self.ws = Path(self._t.name) / "ws"
  21 |         (self.ws / "06_evidence").mkdir(parents=True)
  22 |         self.st = [{"tool_calls": [{"name": "run_command", "args": {"CommandLine": "python3 tooling/evidence_run.py --workspace w --kind unit -- python3 -m unittest"}}]}]
  23 | 
  24 |     def tearDown(self) -> None:
  25 |         self._t.cleanup()
  26 | 
  27 |     def _ev(self, **kw) -> None:
  28 |         rec = {"id": EV, "cmd": "python3 -m unittest", "kind": "unit", "exit": 0, "stdout_tail": "Ran 49 tests\nOK", "src_fingerprint": "fp1"}
  29 |         rec.update(kw)
  30 |         (self.ws / "06_evidence" / f"{EV}.json").write_text(json.dumps(rec), encoding="utf-8")
  31 | 
  32 |     def test_claim_without_ev_rejected(self) -> None:
  33 |         self.assertTrue(agy_guard.claim_issues("- hooks 49/49 PASS", self.ws, self.st, "fp1"))
  34 | 
  35 |     def test_valid_ev_passes(self) -> None:
  36 |         self._ev()
  37 |         self.assertEqual(agy_guard.claim_issues(f"- hooks 49/49 PASS ({EV})", self.ws, self.st, "fp1"), [])
  38 | 
  39 |     def test_number_not_in_output_rejected(self) -> None:
  40 |         self._ev()
  41 |         out = agy_guard.claim_issues(f"- hooks 50/50 PASS ({EV})", self.ws, self.st, "fp1")
```

## 4. 📤 Hợp Đồng Đầu Ra (`OutputContract`) & DoD
- **Sản phẩm bàn giao (`Deliverables`):**
  - [agent] Cập nhật & kiểm chứng agent/hooks/agy/tests/test_claim_linter.py:17 (class TestClaimLinter(unittest.TestCase):) — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`test_claim_linter.py`)
  - [DoD Gate] go test -race ./... (Biên lai vật lý: EV-SUB-TASK-6-M04-PASS · sha256=d9c50bbfbcfc)
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`DownstreamSubTaskIDs`):** `Wave Cuối (Bàn giao kết quả tổng cho Task cha TASK-6)`
- **File Diff cô lập:** `.orchestrator/tasks/TASK-6/subtasks/TASK-6-M04/diff.patch`
- **File Audit S5 & DG3:** `.orchestrator/tasks/TASK-6/subtasks/TASK-6-M04/audit_report.md`
- **Biên lai xác thực (`EvidenceReceiptID`):** `EV-SUB-TASK-6-M04-PASS` (`HandoffReady = true`)
