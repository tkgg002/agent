---
tier: T2
suffix: fix_hallucinated_laya_and_security_models
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T15:14:00+07:00
status: DONE
---

# Báo Cáo Kiểm Định & Phản Biện Cuối Cùng (`audit_report_fix_hallucinated_laya_and_security_models.md`)

## 1. (a) Line-by-Line Diff Critique — Phản Biện Toàn Bộ Quá Trình, Từng Task & Từng Dòng Update (`git diff`)

### 1.1. Phản biện quá trình thực hiện & 4 Task (`TASK-MDL-01 .. TASK-MDL-04`)
- **Sai lầm gốc rễ ban đầu (đã tự thú và ghi nhận vào `lessons.md` `L002`, `L003`):**
  1. Ở task trước (`pipeline_resilience_and_cross_model_gates`), khi đề bài có câu ví dụ *"Cross-Model Auditing (Gemini code -> Claude/Codex review)"*, em đã **cắm đầu làm mù quáng (Zero Adversarial Pushback)** mà không "la làng lên" rằng:
     - **`Laya Gate (Slot H)`** thực chất là **Local Neural Decision Engine `convaiinnovations/laya` (~800MB weights)** chạy tại `http://127.0.0.1:8200/v1/predict` (`~33ms/pass`, **0 LLM token**), tuyệt đối không phải model LLM `claude-sonnet-4-6 (antigravity)`.
     - **`codex-5.3 (antigravity)`** hoàn toàn **không tồn tại** trên Antigravity lẫn trên máy (`which codex opencode` đều `not found`), trong khi cơ chế Security thật trên máy là **`agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)`**.
  2. Hơn nữa, trước đây cả `/agent` lẫn `ai-orchestrator` đều nhảy thẳng từ `S2 Investigate` sang `S3 Solution (Plan)` mà thiếu hẳn **Wave Phản Biện Yêu Cầu Trước Khi Lên Plan (`S2.5 Pre-Plan Adversarial Critique — DG0`)**.
- **Giải pháp khắc phục triệt để trong đợt thực thi này (`TASK-MDL-01 .. TASK-MDL-04`):**
  - **`TASK-MDL-01` & `TASK-MDL-02` (Dọn sạch 100% cấu hình model/runner bịa đặt):** Trả `LayaGateModel` về đúng **`convaiinnovations/laya (local :8200)`** và `SecurityModel` về đúng **`agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)`**; xóa sạch toàn bộ chuỗi `codex-5.3`, `codex`, `opencode`, `gpt-5-codex`, `cursor`, `laya-rl` trong `task_inspector.go`, `seed_data.go`, `index.html`, `app.js`, `server_test.go`.
  - **`TASK-MDL-03` (Nâng cấp `/Users/trainguyen/Documents/work/agent`):** Bổ sung bước bắt buộc **`4. S2.5 PRE-PLAN ADVERSARIAL CRITIQUE (DG0 — Wave Phản biện trước khi lên Plan)`** vào `GEMINI.md:71`, nâng cấp `tooling/laya_gate.py` hỗ trợ `--stage dg0` (`C-REQ`, `C-REALITY`, `C-CONTRA`), và đồng bộ `skills/brain-delegate/SKILL.md` + `skills/laya-router/SKILL.md`.
  - **`TASK-MDL-04` (Nâng cấp `/Users/trainguyen/Documents/work/ai-orchestrator`):** Chèn chặng **Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique · Laya DG0 :8200 · 0 Token`)** vào trước `Plan (S3)` trên cả 4 mặt trận:
    1. Thanh Autopilot Pipeline chính (`server.go:2895-2902` stage `ID: "critique"` trước `ID: "plan"`, nâng tổng số stage lên `7`).
    2. Thanh Sub-Task Fractal Pipeline (`task_inspector.go:1954-1961` thêm card `S2.5 Wave Phản Biện Trước Plan (DG0)` trước `Plan & Code Demo riêng`).
    3. Cột Workflow Steps (`task_inspector.go:4047-4061` Step 4 `Wave Phản Biện Trước Plan (DG0)` trước Step 5 `Giải pháp & Code Demo (S3)`, nâng tổng số step lên `17`) & Step Report Inspector (`task_inspector.go:3816-3842` + `index.html:277` thêm tab `0. Phản Biện (DG0)`, nâng tổng số Step Reports lên `6`).
    4. Bộ máy thực thi (`server.go:554-563, 608-613`, `task_inspector.go:2058-2166, 3600-3695`, `kernel_bridge.go:691-730`): tự động sinh biên bản `10_preplan_critique.md` và gọi `/laya_router` local (`:8200`, 0 LLM token) xuyên suốt cả 4 cổng `dg0`, `dg1`, `dg2`, `dg3`.

### 1.2. Phản biện từng file và từng dòng `git diff`
1. **`/Users/trainguyen/Documents/work/agent/GEMINI.md` (`L68-L88`):**
   - Bổ sung mục `4. S2.5 PRE-PLAN ADVERSARIAL CRITIQUE (DG0 — Wave Phản biện trước khi lên Plan)` ngay trước `5. S3 SOLUTION`.
   - Cập nhật Rule #10 chỉ rõ bản chất vật lý của `/laya_router` (`convaiinnovations/laya` chạy local `:8200`, 0 LLM token, `~33ms/pass`) và 4 cổng bắt buộc `DG0`, `DG1`, `DG2`, `DG3`.
   - Kiểm tra dung lượng `GEMINI.md`: `18,563 bytes` (`131` dòng), nằm gọn dưới ngưỡng `< 24KB`.
2. **`/Users/trainguyen/Documents/work/agent/tooling/laya_gate.py` (`L327, L634-L645`):**
   - Mở rộng `--stage` trong cả `run` và `resolve` thành `choices=["dg0", "dg1", "dg2", "dg12", "dg3", "dg3review"]`.
   - Kiểm tra tương thích ngược: không ảnh hưởng tới `dg1`, `dg2`, `dg12`, `dg3`, `dg3review` (`49/49` unit tests trong `tooling/tests` đều `OK`, biên lai `EV-20261023T151340-110a`).
3. **`/Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/kernel_bridge.go` (`L691-L735`):**
   - Trong `BuildAndRunCalibratedLayaGate`, bổ sung `case "dg0":` tạo 3 claim chuẩn hóa `C-REQ-01`, `C-REALITY-01`, `C-CONTRA-01` kèm `K1-FALSE`, `K2-FALSE`, `K3-TRUE` với trích dẫn `REF: <absRef>:<line>-<line>` thật từ file trên đĩa.
   - Bổ sung nhánh `case "dg2":` riêng biệt (`D-BEHAV-01`, `D-EDGE-01`, `D-API-01`) để khi chạy `S4 Implement` gọi `dg2` độc lập trên `:8200`.
4. **`/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go`:**
   - `L637-657` (`extractModelFamilyPrefix`): nhận diện `"convaiinnovations/laya"` -> `"laya-local-8200"` và `"agy_guard"` + `"claude"` -> `"agy-guard+claude"`, đảm bảo `EvaluateCrossModelDiversity` xác nhận `MUSCLE[gemini] ≠ QA[claude] ≠ Laya[laya-local-8200] ≠ Sec[agy-guard+claude]`.
   - `L688-753` (`loadCLISettings`): đặt mặc định `LayaGateModel: "convaiinnovations/laya (local :8200)"`, `SecurityModel: "agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)"`, đồng thời tự động sanitize mọi giá trị cũ trong `.orchestrator/cli_settings.json` có chứa `codex`, `opencode`, hoặc gán nhầm `claude-`/`gemini-` vào `LayaGateModel`.
   - `L1874-1968` & `L2055-2170`: sinh `10_preplan_critique.md` cho từng Sub-Task, bổ sung card `S2.5 Wave Phản Biện Trước Plan (DG0)` và gọi `BuildAndRunCalibratedLayaGate` ở cả `dg0`, `dg1`, `dg2`, `dg3`.
   - `L3600-3695`, `L3816-3842`, `L4047-4061`, `L4192-4206`: sinh `10_preplan_critique.md` cho Task cha, thêm Step Report `DG0`, thêm Workflow Step 4 `Wave Phản Biện Trước Plan (DG0)`, và cập nhật `Slot H` / `Slot I` hiển thị đúng engine/model thật.
5. **`/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go` (`L554-563, L608-613, L2895-2910`):**
   - Trong `runSingleStageWithDelay`: `case "critique", "plan":` chạy Wave Phản Biện `DG0` trước khi chạy `S3 Plan (DG1/DG2)`; `case "implement":` gọi `BuildAndRunCalibratedLayaGate("dg2")` sau khi verify code trên đĩa.
   - Trong `buildAutopilotView`: thêm stage `ID: "critique"` (`Wave Phản Biện Trước Plan · 0 LLM Token`) ngay trước stage `ID: "plan"`.
6. **`/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html` & `app.js`:**
   - Thêm nút Step Report `0. Phản Biện (DG0)` (`index.html:277`), thêm `DG0` vào `order = ['DG0', 'S1', 'S2', 'S3', 'S4', 'S5']` (`app.js:1229`).
   - Xóa sạch mọi `<option>` chứa `codex-5.3`, `codex`, `opencode`, `gpt-5-codex` trong `index.html:720-735, 875-945` và `app.js:1831-1834, 2491, 3040-3078`.

---

## 2. (b) Anti-Smoke-Test Speculation — Phản Biện Tương Tác Runtime Thực Tế & Concurrency

- **Không suy diễn từ Unit Test đơn lẻ:**
  - Kiểm tra đồng thời (`go test -v -race`) trên toàn bộ 11 test suites của `internal/webui` và `internal/agentgate` (`EV-20261023T150108-a955`, `exit=0`).
  - Phản biện trường hợp file `.orchestrator/cli_settings.json` trên đĩa đã lưu giá trị cũ từ phiên trước (`"laya_gate_model": "claude-sonnet-4-6 (antigravity)"`, `"security_model": "codex-5.3 (antigravity)"`): Nếu chỉ sửa struct default mà không có bước sanitize khi đọc file, server thực tế `:8090` vẫn sẽ trả về giá trị cũ. Do đó, `loadCLISettings()` (`task_inspector.go:731-744`) đã được thiết kế để tự động phát hiện và ghi đè các giá trị bịa đặt/lệch bản chất về đúng `"convaiinnovations/laya (local :8200)"` và `"agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)"`.
  - Phản biện tính nhất quán giữa UI và API khi mở rộng Autopilot Stages từ `6 -> 7` (thêm `critique`) và Step Reports từ `5 -> 6` (thêm `DG0`): Đã kiểm chứng cả trên DOM render (`app.js:1229`) lẫn API `/api/dashboard` trên tiến trình live `:8090` (`EV-20261023T150417-b7dd`).

---

## 3. (c) Zero-Mockup & Real Runtime Trace (`EV-...` Receipts)

Toàn bộ kết quả kiểm chứng đều được thực thi trên tiến trình thật và lưu biên lai vật lý trong `06_evidence/`:
1. **Kiểm tra tĩnh `go vet ./...`:**
   - Biên lai: `EV-20261023T150007-a026` (`exit=0`) & `EV-20261023T152839-9a51` (`exit=0`) — PASS sạch 100%.
2. **Kiểm tra toàn bộ Test Suites với Race Detector (`go test -v -race ./internal/webui/... ./internal/orchestrator/... ./internal/agentgate/...`):**
   - Biên lai: `EV-20261023T150108-a955` (`exit=0`) & `EV-20261023T152842-dd34` (`exit=0`) — PASS toàn bộ test suites của `webui`, `orchestrator` (`TaskRunner` + `runPrePlanCritiqueDG0`) và `agentgate`.
3. **Kiểm tra Unit Tests & Surface Validation của `/agent` (`tooling/tests`, `hooks/agy/tests`, `harness2_engine.py validate-surface`):**
   - Biên lai: `EV-20261023T151340-110a` (`exit=0`, `49/49` tests OK), `EV-20261023T151347-b34a` (`exit=0`, `62/62` tests OK) & `EV-20261023T152901-ae51` (`exit=0`, `"status": "VALIDATED"`, `always_on_utf8_bytes=22663 <= 24000`).
4. **Kiểm tra cổng `/laya_router` `DG0` & `DG1/DG2` (`laya_gate.py run` trên `:8200`):**
   - Biên lai: `EV-20261023T150201-8170` (`exit=0`, `06_laya_summary_dg0_20261023T150203.json` = `PASS`) & `EV-20261023T152517-3102` (`exit=0`, `06_laya_summary_dg1_20261023T152519.json` & `06_laya_summary_dg2_20261023T152519.json` = `PASS`).
5. **Build binary `./bin/orchestrator` & chạy `go run ./cmd/orchestrator daemon --once`:**
   - Biên lai: `EV-20261023T153035-421a` (`exit=0`, `go build -o bin/orchestrator ./cmd/orchestrator`) & `EV-20261023T152855-096a` (`exit=0`, `go run ./cmd/orchestrator daemon --once`).
6. **Trace Runtime thật trên Server Live `http://127.0.0.1:8090/api/dashboard` & `/app.js`:**
   - Biên lai: `EV-20261023T150417-b7dd` (`exit=0`), `EV-20261023T153116-1270` (`exit=0`, `/app.js`) & `EV-20261023T153118-5097` (`exit=0`, `/api/dashboard`):
     - `laya_gate_model = "convaiinnovations/laya (local :8200)"`
     - `security_model = "agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)"`
     - `cross_model_verified = true` (`Cross-Model Verified: MUSCLE[gemini] ≠ QA[claude] ≠ Laya[laya-local-8200] ≠ Sec[agy-guard+claude]`)
     - `autopilot_stages = ["start", "critique", "plan", "you", "implement", "qa", "pr"]` (`7` stages, trong đó `critique` có `round_badge = "Wave Phản Biện Trước Plan · 0 LLM Token"`)
     - `step_reports = ["DG0", "S1", "S2", "S3", "S4", "S5"]` (`6` step reports)
     - `workflow_step_4 = "Wave Phản Biện Trước Plan (DG0)"` (`17` workflow steps)
     - Quét `codex-5.3` trên toàn bộ payload JSON `/api/dashboard`: **`0` kết quả**.

---

## 4. (d) Self-Improvement & Autonomous Self-Healing Loop (`lessons.md` & `S5 -> S3 -> S4 -> S5`)

- **Thực thi vòng lặp tự chữa lành khép kín `S5 -> S3 -> S4 -> S5` (`TASK-MDL-05`):**
  - Khi rà soát phản biện từng dòng `git diff` tại `S5 Stop-Audit`, em phát hiện 2 điểm cần khép kín:
    1. Trong `internal/webui/static/app.js`: hàm `mapStepNumToReportTab(stepNum)` (`L473-480`) và `mapAutopilotBoxIndexToReportTab(idx)` (`L1808-1822`) vẫn dùng chỉ số cũ của 6 stages / 16 steps nên khi click vào stage `critique` (`idx=1`) hoặc Step 4 (`stepNum=4`) chưa mở đúng tab `DG0`.
    2. Trong `internal/orchestrator/task_runner.go` (`L447-452, L844-891`): bộ chạy CLI `TaskRunner` nhảy từ `Research` sang `Decompose` mà chưa gọi Wave Phản Biện `runPrePlanCritiqueDG0` (`BuildAndRunCalibratedLayaGate("dg0")` trên `:8200`).
  - Đã quay về `S3` cập nhật `09_tasks_solution_fix_hallucinated_laya_and_security_models.md` + `08_tasks_fix_hallucinated_laya_and_security_models.md`, chạy `DG1/DG2` (`EV-20261023T152517-3102` = `PASS`), tự sửa dứt điểm `app.js` và `task_runner.go`, rebuild & restart `:8090`, và chạy lại toàn bộ test + `DG3`.
- **Đã ghi nhận 3 bài học phản biện mới vào `/Users/trainguyen/Documents/work/agent/memory/global/lessons.md` và đồng bộ `lessons/INDEX.md` (`297` bài học):**
  - `L001 [2026-10-23] Chèn thêm chặng mới (S2.5 Pre-Plan Critique / DG0) vào mảng AutopilotStages (6 -> 7) và WorkflowSteps (16 -> 17) trên Backend nhưng bỏ sót hàm ánh xạ chỉ số click trên Frontend (app.js) và luồng CLI TaskRunner (task_runner.go)`
  - `L003 [2026-10-23] Missing Pre-Plan Adversarial Critique Wave (DG0) & Underutilizing Zero-Token Local Laya Router (:8200)`
  - `L004 [2026-10-23] Cắm đầu làm mù quáng theo ví dụ trong Prompt (Zero Adversarial Pushback) & Bịa đặt tên Model không tồn tại trên máy`

---

## 5. (e) Hội Đồng `/laya_router` (`DG3`) Sau Lần Sửa Code Cuối

- Đã chạy `laya_gate.py run --stage dg3` trên engine neural local `convaiinnovations/laya` (`:8200`, 0 LLM token) với bộ claims `06_laya_claims_dg3_fix_hallucinated_laya_and_security_models.json` sau lần sửa code cuối (`TASK-MDL-05`):
  - Biên lai: `EV-20261023T152948-79d4` (`exit=0`)
  - File tổng hợp: `06_laya_summary_dg3_20261023T152953.json`
  - Kết quả: **`"gate": "PASS"`** (`5/5` claims `A-DIFF-01 .. A-DIFF-05` đều đạt **`ACCEPT`**).

---

## 6. (f) Tổng Hợp Trạng Thái 8 Cổng Chất Lượng (`G1–G8`)

| Gate | Nội dung kiểm định | Bằng chứng vật lý (`EV-...` / File) | Kết quả |
| :--- | :--- | :--- | :---: |
| **G1** | Truy vết yêu cầu `01_requirements_fix_hallucinated_laya_and_security_models.md` | Đủ 5/5 hạng mục `TASK-MDL-01..05` | **PASS** |
| **G2** | Reproduce & Verify Red->Green | `EV-20261023T152842-dd34`, `EV-20261023T153118-5097` | **PASS** |
| **G3** | Test & Runtime thật trên `:8090`, `:8200`, `daemon --once`, `validate-surface` | `EV-20261023T152855-096a`, `EV-20261023T152901-ae51`, `EV-20261023T153118-5097` | **PASS** |
| **G4** | Edge-case & Sanitize cấu hình cũ trên đĩa | `task_inspector.go:731-744` (`loadCLISettings`), `app.js:1808-1822` | **PASS** |
| **G5** | Regression Test (`go test -race` + `unittest`) | `EV-20261023T152842-dd34`, `EV-20261023T151340-110a`, `EV-20261023T151347-b34a` | **PASS** |
| **G6** | Output Correctness (0 chuỗi `codex-5.3`, 7 Autopilot stages, 6 Step reports) | `EV-20261023T153118-5097`, `EV-20261023T153116-1270` | **PASS** |
| **G7** | Adversarial Line-by-Line Audit & `/laya_router` `DG0`/`DG1`/`DG2`/`DG3` | `06_laya_summary_dg0_20261023T150203.json`, `06_laya_summary_dg3_20261023T152953.json` (`EV-20261023T152948-79d4`) | **PASS** |
| **G8** | Bằng chứng vật lý đầy đủ trong Workspace | `audit_report_fix_hallucinated_laya_and_security_models.md`, `06_validation_fix_hallucinated_laya_and_security_models.md` | **PASS** |

