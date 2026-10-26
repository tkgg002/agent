---
tier: T2
suffix: subtask_fractal_lifecycle
workspace: audit-master-mongo-orchestrator-2026-10-22
status: PASS
---
# 🛡️ Báo Cáo Kiểm Định Phản Biện Cuối Cùng (S5 Mandatory Final Adversarial Audit & Autonomous Self-Healing Report — `subtask_fractal_lifecycle`)

- **Task Suffix:** `subtask_fractal_lifecycle` (`Tier: T2`)
- **Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22`
- **Target Project:** `/Users/trainguyen/Documents/work/ai-orchestrator`
- **Biên lai thực thi (`EV-...`) khớp mã nguồn cuối (`src_fingerprint`):**
  - `EV-20261023T095428-9c30` (`go vet ./... && go test -v -race -run TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch ./internal/webui/...` — `exit=0`, `1/1 PASS` `[EV-20261023T095428-9c30]`)
  - `EV-20261023T095631-254d` (`go test -v -race ./internal/webui/...` — `exit=0`, `9/9 PASS` `[EV-20261023T095631-254d]`)
  - `EV-20261023T095441-a34f` (`go run ./cmd/orchestrator daemon --once && go build -o ./bin/orchestrator ./cmd/orchestrator` — `exit=0`, `runtime OK` `[EV-20261023T095441-a34f]`)
  - `EV-20261023T095550-3ea0` (Kiểm chứng `runtime` thật trên cổng `http://127.0.0.1:8090/api/tasks/action` với `TASK-11` — `LIVE_SUBTASKS_COUNT= 14`, biên lai vật lý `.orchestrator/tasks/TASK-11/subtasks/TASK-M01/06_evidence/EV-SUB-TASK-M01-PASS.json` có `src_fingerprint=138188c2ef9a06b2`, `A04_REAL_SNIPPET= True`, `TASKS08_EXISTS= True` `[EV-20261023T095550-3ea0]`)
  - `EV-20261023T095618-46d6` (Kiểm chứng `runtime` âm tính trên cổng `http://127.0.0.1:8090/api/tasks/action` với `TASK-13-M02` — chặn gọi sớm `subtask_implement` giữ ở `STAGE_AFTER_PREMATURE_IMPLEMENT= NEEDS_YOU`, `CODE_DEMO_HAS_REAL_SOURCE= True` `[EV-20261023T095618-46d6]`)

---

## (a) Line-by-Line Diff Critique (Phản Biện Từng Task, Từng File & Từng Dòng `git diff`)

Qua rà soát phản biện từng dòng (`git diff`) trên toàn bộ quá trình thực hiện `TASK-SUB-01..03`, chúng ta đã phát hiện **5 lỗ hổng kỹ thuật & tính trung thực runtime (`GAP-FRAC-AUDIT-01` .. `GAP-FRAC-AUDIT-05`)** trong bản cài đặt đầu tiên và đã tự chữa lành dứt điểm trong `TASK-SUB-04`:

1. **Phản biện `internal/webui/task_inspector.go` (`:44-102`, `:1109-1175`, `:1446-1538`, `:1614-2470`):**
   - **Phát hiện #1 (`GAP-FRAC-AUDIT-01` tại `runSubTaskLifecycleAction` dòng `2163-2204` cũ — Lọt cổng `UpstreamReady`, `H4b Approve`, và `VerifySubTaskCodeOnDisk`):**
     - *Trước khi sửa:* Hàm `runSubTaskLifecycleAction` khi nhận `subtask_implement`, `subtask_qa`, hoặc `subtask_autopilot` (Phase 2) chỉ gán biến `st.Stage = "IN_REVIEW"` hoặc `"DONE"` mà **không hề kiểm tra** `st.InputContract.UpstreamReady`. Hệ quả là một Sub-Task ở `Wave 1` (`TASK-M04`) có thể bấm chạy `Implement` / `QA` và nhảy sang `DONE` ngay cả khi các Sub-Task phụ thuộc ở `Wave 0` (`TASK-M01..M03`) chưa hoàn thành! Đồng thời, gọi `subtask_implement` khi chưa có `.user-approved` cũng không bị `H4b` chặn.
     - *Đã khắc phục tại dòng `2330-2470`:* Bổ sung 2 hàm chặn cứng `checkUpstreamWaveGate` (nếu `st.WaveIndex > 0 && !st.InputContract.UpstreamReady` -> giữ nguyên `st.Stage = "NEEDS_YOU"`, `st.Status = "waiting"`, ghi log `[Wave Dependency Gate BLOCK]`) và `checkApproveGateH4b` (nếu chưa có `.user-approved` -> chặn ở `NEEDS_YOU` kèm log `[Gate H4b BLOCK]`), đồng thời gọi `VerifySubTaskCodeOnDisk` + `writeSubTaskPhysicalEvidenceReceipt` khi chạy `subtask_qa` / `subtask_autopilot` Phase 2 (`9/9 PASS` `[EV-20261023T095631-254d]`).
   - **Phát hiện #2 (`GAP-FRAC-AUDIT-02` tại dòng `1708-1714` & `1871-1880` cũ — Bịa chuỗi tĩnh `"EV-SUB-<ID>-PASS"` không có file thật & tự chế 4 dòng comment `diff --git` giả):**
     - *Trước khi sửa:* `evReceipt` chỉ là một chuỗi ghép tên `"EV-SUB-" + safeSubID + "-PASS"` hoàn toàn không có file JSON biên lai vật lý trên đĩa; và `extractSubTaskDiffFromParentPatch` khi không tìm thấy hunk trong `parentDiff` lại tự chế ra một khối `diff --git a/... b/... +// [TASK-M01] Isolated Branch...` bằng 4 dòng comment giả tạo.
     - *Đã khắc phục tại dòng `1707-1715`, `1925-1980`, `2036-2066`:* Xây dựng hàm `writeSubTaskPhysicalEvidenceReceipt` băm SHA-256 trực tiếp trên file nguồn thật trong `st.TargetRepo` và ghi biên lai vật lý `.orchestrator/tasks/<safeParent>/subtasks/<safeSubID>/06_evidence/EV-SUB-<safeSubID>-PASS|PENDING.json` có trường `src_fingerprint` thật (`M01_SRC_FP= 138188c2ef9a06b2` `[EV-20261023T095550-3ea0]`). Xóa bỏ hoàn toàn việc sinh `diff --git` comment giả trong `extractSubTaskDiffFromParentPatch`, thay bằng báo cáo trung thực `# [NO UNCOMMITTED GIT DIFF HUNK IN WORKTREE — VERIFIED ON-DISK SOURCE AT TargetRepo]` kèm đoạn mã nguồn thật đọc từ đĩa.
   - **Phát hiện #3 (`GAP-FRAC-AUDIT-03` tại `buildDedicatedSubTaskCodeDemo` dòng `1995-2008` cũ — Rơi về hàm stub rỗng `ExecuteSubTask_... return nil` cho các Sub-Task ngoài `TASK-M01..M10`):**
     - *Trước khi sửa:* Bất kỳ Sub-Task nào có mã ngoài `TASK-M01..M10` (như `TASK-A04..A07` của `TASK-11` hoặc `TASK-13-M01..M04` của `TASK-13`) đều rơi vào nhánh `default:` sinh ra một hàm giả `func ExecuteSubTask_...(ctx context.Context) error { return nil }`. Hơn nữa, khi trace live lần 1 trên `:8090`, phát hiện `TASK-A04` trỏ tới file mới tách `transmuter_mongo.go` nên cần đọc cả trong worktree lẫn file gốc cùng package (`transmuter.go`).
     - *Đã khắc phục tại dòng `1982-2213`:* Viết hàm `readSubTaskRealSourceSnippet(st SubTaskDTO)` đọc trực tiếp các dòng code thật tại `st.ContextFileLine` từ `WorktreePath`, `TargetRepo`, và package directory trên đĩa (`A04_REAL_SNIPPET= True` và `CODE_DEMO_HAS_REAL_SOURCE= True` `[EV-20261023T095550-3ea0]` `[EV-20261023T095618-46d6]`).
   - **Phát hiện #4 (`GAP-FRAC-AUDIT-04` tại dòng `1814-1832` cũ — Thiếu `08_tasks.md` và cấu trúc chuẩn `/agent` Tier T2 trong thư mục Sub-Task):**
     - *Đã khắc phục tại dòng `1835-1910`:* Mỗi thư mục `.orchestrator/tasks/<safeParent>/subtasks/<safeSubID>/` nay được ghi đầy đủ `01_requirements.md` (có YAML frontmatter `tier: T2`), `08_tasks.md`, `13_analysis.md` (bảng `VERIFIED` + snippet code thật), `09_tasks_solution.md`, `diff.patch`, `audit_report.md` (đủ các mục S5 Audit), `05_progress.md` và `06_evidence/EV-SUB-*.json` (`TASKS08_EXISTS= True` `[EV-20261023T095550-3ea0]`).
2. **Phản biện `internal/webui/server.go` (`:960-992`, `:1502-1518`, `:1544-1565`):**
   - **Phát hiện #5 (`GAP-FRAC-AUDIT-05` tại nhánh `case "qa"` dòng `1544-1559` cũ — Ghi đè `subtasks.json` làm mất `Stage` và hợp đồng Fractal):**
     - *Trước khi sửa:* Khi Task cha chạy `case "qa"`, vòng lặp gán `subTasks[i].Status = "done"` rồi gọi `saveTaskSubTasks` ngay mà không cập nhật `subTasks[i].Stage = "DONE"`, không ghi `.subtask-stage`, và không gọi `populateSubTaskFractalContractsAndArtifacts`.
     - *Đã khắc phục tại dòng `1544-1565`:* Đồng bộ đầy đủ `subTasks[i].Stage`, `.subtask-stage`, `enrichSubTasksWithRoleAndModel` và `populateSubTaskFractalContractsAndArtifacts` trước khi lưu `subtasks.json` (`9/9 PASS` `[EV-20261023T095631-254d]`).
3. **Phản biện `internal/webui/static/index.html` (`:313-315`, `:525-535`), `internal/webui/static/app.js` (`:15-17`, `:580-852`, `:955-995`, `:1638-1812`), `internal/webui/static/app.css` (`:2944-3204`):**
   - Tất cả chuỗi hiển thị trên `#subtask-fractal-inspector` (View 1) và `#wave-subtask-fractal-inspector` (View 2) đều đi qua `escHtml()`, đồng bộ `selectedSubTaskByTask[taskKey]` giữa 2 màn hình, và phân bổ đúng `st.wave_index ?? st.wave ?? 0` trên bảng `Wave 0 .. Wave 4` (`9/9 PASS` `[EV-20261023T095631-254d]`).
4. **Phản biện `internal/webui/server_test.go` (`:1380-1625`):**
   - Mở rộng `TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch` bổ sung kiểm tra vật lý `08_tasks.md` + `06_evidence/EV-SUB-*.json` (có `src_fingerprint`) và **2 bài test âm tính (`Negative Tests`)**: (1) gọi sớm `subtask_implement` trên `Wave 1` `TASK-M04` khi `Wave 0` chưa xong bị chặn tại `NEEDS_YOU`; (2) gọi `subtask_implement` trên `Wave 0` `TASK-M01` khi chưa `APPROVE` bị cổng `H4b` chặn tại `NEEDS_YOU` (`1/1 PASS` `[EV-20261023T095428-9c30]`, `9/9 PASS` `[EV-20261023T095631-254d]`).

---

## (b) Chặn Đứng Suy Diễn Smoke Test (Anti-Smoke-Test Speculation)

- **Kiểm chứng cả chiều Thuận (`Happy-Path Wave 0 -> Wave 1`) lẫn chiều Nghịch (`Negative Gate Blocking`):**
  - Không suy diễn từ việc hiển thị cờ `UpstreamReady: false` trên JSON rằng hệ thống đã chặn: chúng ta đã gửi trực tiếp HTTP POST `{"task_key":"TASK-FRACTAL-10","sub_task_id":"TASK-M04","action":"subtask_implement"}` khi `Wave 0` chưa xong và kiểm chứng thực tế `runSubTaskLifecycleAction` chặn đứng yêu cầu tại `Stage = "NEEDS_YOU"` (`1/1 PASS` `[EV-20261023T095428-9c30]`).
  - Không suy diễn từ `TASK-M01..M10` rằng mọi Sub-Task khác đều có Code Demo thật: chúng ta đã trace trực tiếp trên cổng `:8090` với `TASK-A04` (`TASK-11`) và `TASK-13-M02` (`TASK-13`) để chứng minh `readSubTaskRealSourceSnippet` đọc dòng code thật trên đĩa (`A04_REAL_SNIPPET= True`, `CODE_DEMO_HAS_REAL_SOURCE= True` `[EV-20261023T095550-3ea0]` `[EV-20261023T095618-46d6]`).
  - Chạy toàn bộ 9 test suites tích hợp với `-race` sạch 100% data race (`9/9 PASS` `[EV-20261023T095631-254d]`).

---

## (c) Zero-Mockup & Real Runtime Trace (`EV-...`)

1. **Kiểm tra tĩnh & Test mục tiêu (`go vet ./... && go test -v -race -run TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch ./internal/webui/...`) — `1/1 PASS` `[EV-20261023T095428-9c30]`:**
   - Kết quả: `EVIDENCE EV-20261023T095428-9c30 exit=0` (`1/1 PASS` `[EV-20261023T095428-9c30]`).
2. **Toàn bộ Test Suite tích hợp kèm Race Detector (`go test -v -race ./internal/webui/...`) — `9/9 PASS` `[EV-20261023T095631-254d]`:**
   - Kết quả: Toàn bộ `9/9` test suites đạt `PASS` (`52.777s`, `exit=0` `[EV-20261023T095631-254d]`).
3. **Kiểm chứng CLI Daemon (`daemon --once`) & Build Binary (`./bin/orchestrator`) — `runtime OK` `[EV-20261023T095441-a34f]`:**
   - Kết quả: `EVIDENCE EV-20261023T095441-a34f exit=0` (`runtime OK` `[EV-20261023T095441-a34f]`).
4. **Trace Runtime HTTP thật trên cổng `:8090` (`TASK-11` & `TASK-13`) — `runtime OK` `[EV-20261023T095550-3ea0]` `[EV-20261023T095618-46d6]`:**
   - Trace 1 (`TASK-11` trên `http://127.0.0.1:8090/api/tasks/action`):
     `LIVE_SUBTASKS_COUNT= 14 M01_EV_FILE= .orchestrator/tasks/TASK-11/subtasks/TASK-M01/06_evidence/EV-SUB-TASK-M01-PASS.json M01_SRC_FP= 138188c2ef9a06b2 A04_REAL_SNIPPET= True TASKS08_EXISTS= True` (`runtime OK` `[EV-20261023T095550-3ea0]`).
   - Trace 2 (`TASK-13-M02` trên `http://127.0.0.1:8090/api/tasks/action`):
     `TASK_13_M02_WAVE= 0 UPSTREAM_READY= True STAGE_AFTER_PREMATURE_IMPLEMENT= NEEDS_YOU CODE_DEMO_HAS_REAL_SOURCE= True` (`runtime OK` `[EV-20261023T095618-46d6]`).

---

## (d) Self-Improvement & Autonomous Self-Healing Loop (`S5 → S3 → S4 → S5`)

- **Ghi nhận bài học mới vào `memory/global/lessons.md` (`Rule #5 & #6`):**
  - Đã chèn bài học `[2026-10-23] Unenforced Sub-Task Wave/Approve/On-Disk Verification Gates, Fabricated Static EV-SUB-*-PASS String & Fake Comment Diff, and Hollow Fallback Code Demo Stubs` lên đầu `lessons.md` (`wc -l` tăng từ `2205` lên `2212` dòng) và chạy `build_lessons_index.py` (`entries=286/286/286`).
- **Chu trình Tự Chữa Lành Khép Kín (`S5 -> S3 -> S4 -> S5` không đẩy lại cho User):**
  1. Từ kết quả Audit `S5`, quay về `S3` cập nhật `08_tasks_subtask_fractal_lifecycle.md` (`TASK-SUB-04`) và `09_tasks_solution_subtask_fractal_lifecycle.md` (Mục 6), chạy `laya_gate.py run --stage dg1` (`06_laya_summary_dg1_20261023T094028.json` = `PASS`).
  2. Thực thi `S4` sửa dứt điểm `GAP-FRAC-AUDIT-01..05` trong `internal/webui/task_inspector.go`, `internal/webui/server.go`, `internal/webui/server_test.go`.
  3. Khi trace runtime thật trên `:8090` (`EV-20261023T095141-bda5`) phát hiện `TASK-A04` (`transmuter_mongo.go`) cần đọc cả file cùng package (`transmuter.go`) khi file mới tách chưa ghi vào repo chính, đã tinh chỉnh ngay `readSubTaskRealSourceSnippet`, rebuild binary, khởi động lại `:8090` và trace lại đạt `A04_REAL_SNIPPET= True` (`runtime OK` `[EV-20261023T095550-3ea0]`).

---

## (e) Hội Đồng `/laya_router` (`DG3`) Sau Lần Sửa Code Cuối

- **Lệnh thực thi:**
  `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg3 --claims /Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_laya_claims_dg3_subtask_fractal_lifecycle.json --workspace audit-master-mongo-orchestrator-2026-10-22`
- **File kết quả vật lý:** `06_laya_summary_dg3_20261023T095823.json` & `06_laya_raw_dg3_20261023T095823.json`.
- **Kết quả Calibration & Verdict:**
  - `trusted_rounds`: `["r1", "r2"]` (bắt chính xác cả 2 biến thể `control_false` `K1`, `K2` và `control_true` `K3`).
  - `A-SUB-01` (`SubTaskInputContractDTO` & `SubTaskOutputContractDTO` tại `task_inspector.go:44-61`): `ACCEPT` (`r1=0.8952`, `r2=0.7046`, `r3=0.8728`)
  - `A-SUB-02` (`checkUpstreamWaveGate` & `checkApproveGateH4b` tại `task_inspector.go:2376-2404`): `ACCEPT` (`r1=0.5381`, `r2=0.5904`, `r3=0.8632`)
  - `A-SUB-03` (`handleTaskAction` nhánh `subtask_*` tại `server.go:977-989`): `ACCEPT` (`r1=0.5802`, `r2=0.4647`, `r3=0.9073`)
  - `A-SUB-04` (`TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch` tại `server_test.go:1380-1396`): `ACCEPT` (`r1=0.9052`, `r2=0.5920`, `r3=0.8664`)
  - **Cổng `DG3`:** `"gate": "PASS"`.

---

## (f) Đối Chiếu 8 Cổng Chất Lượng DoD (`Gates G1–G8`)

| Gate | Tiêu chí kiểm định | Trạng thái | Bằng chứng vật lý |
| :--- | :--- | :---: | :--- |
| **G1 (Traceability)** | Khớp 100% yêu cầu `01_requirements_subtask_fractal_lifecycle.md` & `09_tasks_solution_subtask_fractal_lifecycle.md` | **PASS** | `TASK-SUB-01..04` hoàn tất `[x]` `[EV-20261023T095631-254d]` |
| **G2 (Reproduce Red→Green)** | Tái hiện & đóng kín 5 lỗ hổng `GAP-FRAC-AUDIT-01..05` | **PASS** | `TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch` `[EV-20261023T095428-9c30]` |
| **G3 (Real Test & Runtime)** | `go vet`, `go test -race` (`9/9`), `daemon --once`, và live `:8090` HTTP trace | **PASS** | `[EV-20261023T095428-9c30]`, `[EV-20261023T095631-254d]`, `[EV-20261023T095441-a34f]`, `[EV-20261023T095550-3ea0]` |
| **G4 (Edge-case & Concurrency)** | Chặn Wave N+1 chạy sớm khi thiếu phụ thuộc Wave N; chặn chạy S4 khi thiếu `.user-approved` (`H4b`); không có data race | **PASS** | `[EV-20261023T095428-9c30]`, `[EV-20261023T095618-46d6]`, `[EV-20261023T095631-254d]` |
| **G5 (Regression)** | Toàn bộ 9/9 test suites trong `internal/webui` đều PASS | **PASS** | `[EV-20261023T095631-254d]` |
| **G6 (Output Correctness)** | Đủ bộ hồ sơ Tier T2 (`01`, `08`, `13`, `09`, `diff.patch`, `audit_report`, `05_progress`, `06_evidence/EV-SUB-*.json`) cho từng Sub-Task | **PASS** | `[EV-20261023T095550-3ea0]`, `[EV-20261023T095631-254d]` |
| **G7 (Adversarial Audit & DG3)** | Phản biện từng dòng diff + `/security-agent` + `DG3` sau lần sửa cuối đạt `gate: "PASS"` | **PASS** | `06_laya_summary_dg3_20261023T095823.json` `[EV-20261023T095631-254d]` |
| **G8 (Physical Evidence)** | Lưu đầy đủ biên lai `EV-...` trong `06_evidence/` và báo cáo hợp nhất `audit_report_subtask_fractal_lifecycle.md` | **PASS** | `[EV-20261023T095428-9c30]`, `[EV-20261023T095631-254d]`, `[EV-20261023T095550-3ea0]` |
