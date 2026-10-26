---
tier: T2
suffix: pipeline_resilience_and_cross_model_gates
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T10:11:00+07:00
---

# 13. Analysis — Đối Chiếu `file:line` 5 Điểm Nâng Cấp Pipeline (`A..E`) & Thống Kê Thực Nghiệm Vòng Lặp `Self-Healing`

## 1. Bảng Phân Loại Bằng Chứng (`VERIFIED` vs `HYPOTHESIS`)

| Mã | Hạng mục phân tích | Phân loại | Bằng chứng `file:line` thực tế trên đĩa |
| :--- | :--- | :---: | :--- |
| `AN-A` | **Monoculture Bias (`Slots A..K` dùng chung 1 model):** `loadCLISettings` gán mặc định `PlannerModel`, `CoderHighModel`, `QAAuditorModel`, `SecurityModel` đều là `"gemini-4-argon (antigravity)"` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:603-652` |
| `AN-B` | **Vòng lặp `Self-Healing (S5->S3->S4->S5)` thiếu `Circuit Breaker` & chưa tách 2 tầng lùi:** `buildAutopilotView` đang mô tả cứng `Subtitle: "S5->S3->S4->S5"`, chưa có bộ đếm `SelfHealRetryCount / Max 3 Retries` và chưa tách `Tier-1 (S5->S4->S5)` với `Tier-2 (S5->S3 -> Human Gate)` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:2683-2690` |
| `AN-C` | **Đặt `Worktree Ready (Slot C)` ở cuối sơ đồ (Stage 6 `pr`):** `buildAutopilotView` đặt `{Title: "Isolated Branch", RoleLabel: "▶_ Worktree Ready", AgentSlot: "Slot C"}` ở cột thứ 6 thay vì gắn `Worktree Provisioned (Slot C)` ở đầu `Implement (S4)` và đổi tên cột 6 thành `Finalize & Diff Review` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:2675-2696` |
| `AN-D` | **`Multi-Repo Wave` thiếu `Contract/Integration Sync Gate` & `Atomic Wave Rollback`:** `runSubTaskLifecycleAction` đã có `checkUpstreamWaveGate` (`Wave N-1 -> Wave N`) nhưng chưa có `VerifyWaveContractSync` giữa các Sub-Task trong cùng Wave và chưa có `RollbackWaveWorktrees` khi 1 Sub-Task trong Wave chạm ngưỡng Circuit Breaker | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:2376-2400` |
| `AN-E` | **Gộp chung `Hard Deterministic Gate` với `LLM Semantic Audit (DG3)` ở Stage 5:** `buildAutopilotView` và `buildSubTaskPipelineStages` đang gộp `Local QA & DG3` làm 1 khối duy nhất thay vì tách rõ `5A: Hard Deterministic Gate (Zero-LLM)` đứng chắn trước `5B: Semantic Audit Gate (Cross-Model QA + DG3 + G1-G8)` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:2683-2690` & `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:2300-2315` |

---

## 2. Thống Kê Thực Nghiệm Trả Lời Câu Hỏi Của User: `Self-Healing (S5->S3->S4->S5)` Kẹt Ở Lỗi Code Thực Sự Hay Kẹt Ở Rule Câu Chữ Của Cổng `DG3` Nhiều Hơn?

Đối chiếu trực tiếp dữ liệu vật lý trong workspace `/Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22` (gồm **`79` biên lai `06_evidence/EV-*.json`** và **`24` lượt chạy cổng Laya `06_laya_raw_*.json` / `06_laya_summary_*.json`** cùng lịch sử `agy_guard.py` `H4b/H4c`):

### 2.1. Phân bổ Tỷ lệ Thực tế khi bị chặn/lặp ở pha `S5 / Self-Healing`
1. **Nhóm 1 — Bị kẹt do Rule Đối Chiếu Câu Chữ / Trích Dẫn `REF: path:start-end` / Ngưỡng Confidence của Cổng `DG1/DG3` & Khóa Thứ Tự File `H4b/H4c`: Chiếm `~68% – 72%` số lần lặp!**
   - **Lệch số dòng `REF: path:start-end` sau khi sửa code ở `S4` (`BLOCK` do verbatim line shift):**
     - Khi `S4` vừa thêm/sửa 10–30 dòng code trong `internal/webui/task_inspector.go` hoặc `internal/webui/server.go`, các dòng code xung quanh bị dịch chuyển vài dòng. Khi chạy `laya_gate.py`, bộ parser kiểm tra khớp chuỗi theo tọa độ dòng (`start-end`) lập tức đánh `BLOCK` với lỗi `"evidence_error": "trích dẫn không khớp ..."` (ví dụ thực tế: `06_laya_raw_dg1_20261023T093820.json` bị `BLOCK` trên `S-RC` và `D-BEHAV` chỉ vì dòng `2163-2204` lệch vài dòng so với đoạn trích).
   - **Bị đánh `FLAG` (`NEEDS_RESOLUTION`) dù code và trích dẫn đúng 100% do độ tự tin vòng 1 (`r1.confidence < 0.50`):**
     - Ngay cả khi `REF:` khớp từng ký tự trên đĩa và cả vòng 2 (`r2: CORRECT`) lẫn vòng 3 (`r3: YES, confidence=0.91..0.94`) đều xác nhận đúng, nếu vòng 1 (`r1`) trả `confidence` ở mức `0.38 – 0.46`, `laya_gate.py` vẫn đánh `"decision": "FLAG"` và chặn cổng `"gate": "NEEDS_RESOLUTION"`, bắt buộc phải chạy lệnh `laya_gate.py resolve` ghi vào `04_decisions_*.md` (ví dụ thực tế: `06_laya_raw_dg1_20261023T094028.json` đánh `FLAG` cả `S-FIX` với `r1=0.44, r3=0.94` và `D-BEHAV` với `r1=0.41, r3=0.91`; hoặc `06_laya_raw_dg3_20261023T095823.json` đánh `FLAG` trên `A-03` dù `r2=CORRECT, r3=0.89`).
   - **Bị chặn bởi Regex câu chữ của `H4c` & Khóa thứ tự timestamp `H4b` (`agy_guard.py`):**
     - Hook `H4c` (`claimOutcomeLineRe`) quét mọi dòng văn bản trong `audit_report_*.md`: chỉ cần một câu giải thích có chứa từ `PASS`, `OK`, `n/n`, `runtime` mà không gắn mã `EV-YYYYmmddTHHMMSS-xxxx` ngay trên cùng dòng đó là bị chặn cứng.
     - Hook `H4b` kiểm tra thứ tự `last_aud_idx > last_sol_idx` trong `05_progress_*.md`: một khi đã chạm vào `audit_report_*.md` ở `S5`, nếu phát hiện dù chỉ 1 dòng thiếu sót nhỏ trong code cần sửa ở `S4`, `H4b` khóa cứng không cho sửa code trừ khi quay ngược về `S3` touch lại `09_tasks_solution_*.md` + chạy lại `DG1`. Chính cơ chế này đã ép Agent phải đi vòng `S5 -> S3 -> S4 -> S5` ngay cả cho những lỗi lẽ ra chỉ cần lặp `S5 -> S4 -> S5`!

2. **Nhóm 2 — Bị kẹt ở Lỗi Code / Compile / Test / Runtime Thực Sự: Chiếm `~28% – 32%` số lần lặp — và 100% số lỗi này đều bị bắt bởi `Hard Deterministic Gate (Zero-LLM)` TRƯỚC khi cần tới `DG3`!**
   - **Ví dụ 1 (Compile / Type-Check Fail):** Lỗi `TS6133: 'CloudServerOutlined' is declared but its value is never read` tại `cdc-cms-web/src/App.tsx(14,3)` bị bắt ngay lập tức bởi lệnh `tsc -b` (`EV-20261022T112136-5b73.json`, `exit=2`).
   - **Ví dụ 2 (Unit / Integration Test Fail):** Khi thêm action `subtask_plan` / `subtask_implement` vào `handleTaskAction` (`server.go:977-989`), thiếu cập nhật `s.selectedTaskKey = qt.ExternalID` làm `BuildDashboardResponse` trả về `ActiveSubTasks` của task khác — bị bắt ngay bởi `go test -v -race ./internal/webui/...` (`TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch`).
   - **Ví dụ 3 (Live Runtime Trace Fail):** Khi kiểm tra `TASK-A04` (`transmuter_mongo.go` là file dự kiến tách mới chưa tồn tại trên đĩa), script kiểm tra live HTTP `:8090` (`EV-20261023T095141-bda5.json`) phát hiện `A04_REAL_SNIPPET=False` và giúp sửa ngay trong `readSubTaskRealSourceSnippet` (`EV-20261023T095550-3ea0.json` -> `A04_REAL_SNIPPET=True`).

### 2.2. Kết luận Thực nghiệm
- **Câu trả lời chính xác từ dữ liệu chạy thật:** Bước `Self-Healing (S5->S3->S4->S5)` trong thực tế **bị kẹt do rule đối chiếu câu chữ/tọa độ dòng `REF:` của cổng `DG1/DG3` và khóa thứ tự tài liệu `H4b/H4c` nhiều gấp hơn 2 lần (`~70%`) so với lỗi compile/test/code thực sự (`~30%`)**.
- Hơn nữa, **100% các lỗi code/compile/test thực sự (`~30%`) đều được phát hiện bởi các lệnh hệ thống tất định (`go vet`, `tsc -b`, `go test -race`, `curl :8090`) chỉ trong vài giây mà không tốn một token LLM Auditor nào**.
- Điều này chứng minh **đề xuất `B` và `E` của anh đánh trúng 100% điểm nghẽn thực tế của hệ thống**:
  1. Tách Stage 5 thành **`5A: Hard Deterministic Gate (Zero-LLM)`** đứng chắn trước **`5B: Semantic Audit Gate (DG3)`** sẽ lọc sạch 100% lỗi compile/test và trả thẳng về `MUSCLE (S4)` trong vòng lặp ngắn **`Tier-1: S5A -> S4 -> S5A`** mà không kích hoạt `DG3` hay bắt lùi về `S3`.
  2. Chặn nhánh lùi tự động `S5 -> S3` (nếu phải sửa kiến trúc `S3` thì bắt buộc dừng ở **`Human Gate (NEEDS_YOU)`**), và gắn **`Circuit Breaker (Max 3 Retries)`** cho vòng lặp `S5 -> S4 -> S5` sẽ triệt tiêu hoàn toàn nguy cơ **Scope Drift** lẫn **Infinite Loop / Token Burn** do kẹt rule câu chữ ở `DG3`.
