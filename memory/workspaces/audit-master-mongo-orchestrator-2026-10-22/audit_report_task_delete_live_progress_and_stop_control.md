---
tier: T2
suffix: task_delete_live_progress_and_stop_control
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T12:41:35+07:00
---

# Báo Cáo Kiểm Định Phản Biện Cuối Cùng (S5 Mandatory Final Adversarial Audit Report) — `task_delete_live_progress_and_stop_control`

> **Khẩu lệnh Thực thi Nguyên văn của User (Rule #9.7 & Hook H7):**
> *"Audit lại 'quá trình' vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update. Kiểm tra các xem có suy diễn, báo cáo láo không. Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop). Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng Lưu toàn bộ vào file audit_report_*.md trong workspace của task. Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value."*

---

## (a) Line-by-Line Diff Critique (Phản biện từng Task, từng File và từng Dòng `git diff`)

Đã chạy `git status -sb` và `git diff` thật trên `/Users/trainguyen/Documents/work/ai-orchestrator` để rà soát từng dòng code vừa cập nhật:

1. **[`internal/queue/store.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/queue/store.go#L355-L408) (`+55 dòng` — `TASK-CTRL-01`):**
   - `DeleteTask(ctx, externalID)` (dòng `356-371`): Khóa `s.mu.Lock()`, xóa bản ghi khỏi `task_queue` và `autopilot_rounds` theo `external_id`, trả về `n > 0`.
   - `DeleteDoneTasks(ctx, keepExternalID)` (dòng `374-408`): Khóa `s.mu.Lock()`, truy vấn danh sách `external_id` có `stage = 'DONE'` (trừ `keepExternalID` nếu đang được bảo vệ), đóng `rows.Close()` trước khi thực thi `DELETE FROM task_queue` và `DELETE FROM autopilot_rounds` để tránh SQLite busy lock trên cùng connection.
2. **[`internal/webui/seed_data.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/seed_data.go#L231-L246) (`+7 dòng` — `TASK-CTRL-01`):**
   - Trong `SyncRealWorkspaceState`: nạp `deletedSet := s.loadDeletedTasksSet()` và bỏ qua (`continue`) mọi task có key nằm trong `.orchestrator/deleted_tasks.json`, chặn đứng lỗi task đã xóa bị hồi sinh sau khi F5 / reload.
3. **[`internal/webui/telegram_bridge.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L30-L50) & [`dòng 790-902`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L790-L902) (`TASK-CTRL-02` & `TASK-CTRL-03`):**
   - Mở rộng `LiveStepProgressState` bổ sung `SubTaskID`, `SubTaskTitle`, `ActionName`, `ElapsedSec`, `CanStop`, `WasStopped`, `StopReason`.
   - Trong `advancePacedWorkflowStep` (dòng `791-837`) và `advanceSubTaskStepProgress` (dòng `840-888`): kiểm tra `if ctx.Err() != nil { return }` ở đầu hàm và dùng `select { case <-time.After(stepDelay): case <-ctx.Done(): }` ở cuối hàm để khi người dùng bấm **`⏹ Dừng Tiến Trình`**, lệnh sleep bị hủy tức thì (`0ms` chờ thừa).
   - Trong `clearPacedWorkflowStep` (dòng `890-902`): bảo toàn `WasStopped = true` và `StopReason` khi tiến trình vừa bị người dùng dừng để UI hiển thị rõ thông báo `⏹ ĐÃ DỪNG TIẾN TRÌNH` thay vì xóa trắng trạng thái.
4. **[`internal/webui/task_chat.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go#L36-L70) & [`dòng 509-605`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go#L509-L605) (`TASK-CTRL-02`):**
   - Mở rộng `LiveExecutionTelemetryDTO` bổ sung `IsRunning`, `CanStop`, `WasStopped`, `StopReason`, `ActiveSubTaskID`, `ActiveSubTaskTitle`, `StepIndex`, `TotalSteps`, `Percent`, `ElapsedSec`.
   - Trong `buildLiveExecutionTelemetry`: tính `IsRunning` trung thực từ `s.liveStepProgress.Active` (hoặc `stage == RUNNING / PLANNING / IN_REVIEW`), chấm dứt hoàn toàn việc task đang `READY` (chưa chạy) vẫn báo đang chạy.
5. **[`internal/webui/task_inspector.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go#L4155-L4566) (`TASK-CTRL-01`, `TASK-CTRL-02`, `TASK-CTRL-03`, `TASK-CTRL-04`):**
   - Quản lý Tombstone `.orchestrator/deleted_tasks.json` (`loadDeletedTasksSet`, `markTaskDeletedTombstone`, `unmarkTaskDeletedTombstone`).
   - `beginTrackExecution` (dòng `4203-4238`): khởi tạo `context.WithCancel`, hủy tiến trình cũ nếu còn treo, bật ngay `s.liveStepProgress` (`Active: true, CanStop: true, Percent: 12, StartedAt: time.Now()`) và phát SSE `"step_progress"`.
   - `StopTaskExecution` (dòng `4252-4339`): gọi `cancel()` ngắt `context.Context` đang chạy, cập nhật `s.liveStepProgress` sang `Active: false, WasStopped: true, CanStop: false`, chuyển Task và Sub-Task đang chạy về `NEEDS_YOU`, ghi log `[⏹ Dừng Tiến Trình]` vào `05_progress.md` + `terminal.log`, và phát SSE `"task_stopped"`.
   - `DeleteSingleTask`, `DeleteDoneTasksBulk`, `DeleteSingleSubTask`, `handleDeleteTask`, `handleStopTask` (dòng `4340-4566`): xử lý xóa đơn lẻ, xóa hàng loạt `DONE`, xóa Sub-Task kèm dọn sạch đồng bộ cả thư mục `.orchestrator/tasks/<safeKey>` lẫn `.orchestrator/worktrees/<safeKey>` (`GAP-CTRL-AUDIT-03`) và tự động chọn Task kế tiếp.
6. **[`internal/webui/server.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go#L628-L645) & [`dòng 1290-1890`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go#L1290-L1890) (`TASK-CTRL-01 .. 04`):**
   - Đăng ký route `/api/tasks/delete` và `/api/tasks/stop`, đồng thời hỗ trợ `action: "stop"`, `"delete"`, `"delete_done"` ngay trên `/api/tasks/action`.
   - Khi tạo lại một Task bằng `POST /api/tasks`, gọi `s.unmarkTaskDeletedTombstone(req.ExternalID)` để cho phép người dùng chủ động tạo lại mã Task đã xóa trước đó.
   - Bọc `beginTrackExecution` và đặt chốt chặn `if ctx.Err() != nil { return nil }` xuyên suốt `runTaskLifecycleActionWithDelay`, `runAutonomousS4S5WithSelfHealing`, và ngay sau mỗi vòng lặp `affectedRepos` / `BuildAndRunCalibratedLayaGate` trước mọi lệnh `UpdateStage` / `RecordRound` trong `runSingleStageWithDelay` (`GAP-CTRL-AUDIT-02`).
7. **[`internal/webui/static/index.html`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html#L88-L152), [`internal/webui/static/app.js`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js#L635-L1630) & [`internal/webui/static/app.css`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L3250-L3535) (`TASK-CTRL-01 .. 04`):**
   - Thêm `#global-running-progress-banner` cố định đầu màn hình (spinner `⟳`, tên Task/Sub-Task đang chạy, `Step X/Y · Z%`, thanh progress shimmer, đồng hồ đếm giây thực tế `00:05s`, và nút **`⏹ Dừng Tiến Trình`**).
   - Đồng bộ tuyệt đối tên trường JSON của `live_step_progress` (`lsp.step_index`, `lsp.stage_label`, `lsp.current_action`, `lsp.current_where`, `lsp.active_role_slot`) trong `renderGlobalRunningProgressBanner`, `renderSidebarTaskQueue`, và `renderLiveTelemetryHUD` (`GAP-CTRL-AUDIT-01`).
   - Thêm nút **`⏹ Dừng Tiến Trình`** (`#ctx-btn-stop-task`, `#hud-btn-stop-task`), nút **`🗑 Xóa Task`** (`#ctx-btn-delete-task`), nút **`🧹 Dọn DONE`** (`#btn-cleanup-done-tasks`), và nút `⏹` / `🗑` trên từng dòng Sidebar Task, Wave Card, và Sub-Task Card.
   - Sửa `renderLiveTelemetryHUD` trong `app.js`: khi Task đang ở trạng thái tĩnh (`READY`, `isRunning == false`), hiển thị rõ `⏸ SẴN SÀNG CHẠY (IDLE — CHƯA CHẠY)` hoặc `⏹ ĐÃ DỪNG TIẾN TRÌNH` (nếu vừa bấm Stop) thay vì báo ảo `"⚡ LIVE RUNNING"`.

---

## (b) Chặn đứng Suy diễn Smoke Test (Anti-Smoke-Test Speculation & Concurrency Analysis)

- **Phản biện Concurrency & Race Condition khi bấm Stop giữa chừng (`go test -race`):**
  - Khi một goroutine đang thực thi `runTaskLifecycleActionWithDelay` (hoặc `runSubTaskLifecycleActionWithDelay`) và một HTTP request khác gọi `POST /api/tasks/stop` (hoặc `POST /api/tasks/delete`), cả 2 goroutine cùng truy cập `s.activeExecCancel` và `s.liveStepProgress`.
  - Toàn bộ thao tác đọc/ghi `s.activeExecCancel`, `s.activeExecTaskKey`, `s.activeExecSubTaskID`, `s.liveStepProgress` đều được bảo vệ dưới `s.mu.Lock() / s.mu.Unlock()`. Hàm `cancel()` được gọi **ngoài** `s.mu.Lock()` (`internal/webui/task_inspector.go:4290-4295`) để tránh deadlock.
  - Trong `runSingleStageWithDelay` (`internal/webui/server.go:1309-1865`), các chốt chặn `if ctx.Err() != nil { return nil }` đặt ngay sau vòng lặp `affectedRepos` và sau `BuildAndRunCalibratedLayaGate` chặn đứng hoàn toàn nguy cơ goroutine đã bị hủy tiếp tục gọi `s.QueueStore.UpdateStage` hoặc `s.QueueStore.RecordRound` ghi đè trạng thái `STOPPED / NEEDS_YOU` (`GAP-CTRL-AUDIT-02`).
  - Khi goroutine thực thi thoát ra và gọi `defer s.clearPacedWorkflowStep(qt.ExternalID)`, hàm `clearPacedWorkflowStep` (`internal/webui/telegram_bridge.go:893-895`) kiểm tra `if s.liveStepProgress.WasStopped` để chỉ tắt `Active = false, CanStop = false` mà **không** ghi đè mất thông tin `WasStopped = true` và `StopReason` mà `StopTaskExecution` vừa ghi nhận.
  - Đã kiểm chứng thực tế bằng `go test -v -race ./internal/webui/... ./internal/queue/...` (`EV-20261023T133800-d2f7`): `0` data race, `11/11` test suites PASS.

---

## (c) Zero-Mockup & Real Runtime Trace (`EV-...` Receipts)

- **Zero-Mockup Gate (`H4b`):** Không sử dụng bất kỳ closure giả (`*Fn`) nào trong struct production của `internal/webui` hay `internal/queue`.
- **Biên lai Thực thi Thật (`EV-...`):**
  - `EV-20261023T134000-3dd4`: `go vet ./...` (`exit=0`)
  - `EV-20261023T133716-8723` & `EV-20261023T134001-ba47`: `go test -v -race -run TestWebUIServer_TaskDelete_LiveRunningProgress_And_StopProcessControl ./internal/webui/...` (`PASS`, `3.95s`, `exit=0`)
  - `EV-20261023T133800-d2f7`: `go test -v -race ./internal/webui/... ./internal/queue/...` (`11/11 PASS`, `63.799s`, `exit=0`)
  - `EV-20261023T133755-b32b`: `go run ./cmd/orchestrator daemon --once` (`exit=0`)
  - `EV-20261023T133759-baee`: `go build -o bin/orchestrator ./cmd/orchestrator` (`exit=0`)
  - `EV-20261023T134002-507a`: Khởi chạy lại daemon `./bin/orchestrator ui --addr 127.0.0.1:8090` (`task-7118`) và chạy kịch bản kiểm chứng live HTTP `:8090` (`verify_live_8090_ctrl_audit.py`):
    `LIVE_8090_VERIFY_OK {"observed_step_index": 2, "observed_stage_label": "S0 Boot H1 & S1 Intake Scope", "observed_role_slot": "🧠 BRAIN · Slot A", "stopped_stage": "NEEDS_YOU", "worktree_removed": true, "remaining_active_task": "tích hợp Master MongoDB Destination"}` (`exit=0`).

---

## (d) Self-Improvement & Autonomous Self-Healing Loop (`S5 -> S3 -> S4 -> S5`)

1. **Bài học đã ghi nhận vào `lessons.md` (`wc -l`: `2247 -> 2254`, `292/292` entries trong `INDEX.md`):**
   - `L001` (`[2026-10-23] Frontend-Backend live_step_progress JSON Field Mismatch, Missing Post-Command ctx.Err() Guard Overwriting Stopped Stage, and Orphaned Worktree Directory on Task Deletion`).
   - `L004` (`[2026-10-23] Missing Task Deletion Cleanup, Opaque Live Running Progress Indicator, and Missing Stop/Cancel Process Control on Orchestrator UI`).
2. **Phát hiện 3 lỗ hổng khi rà soát phản biện từng dòng ở `S5` và Tự chữa lành khép kín (`S5 -> S3 -> S4 -> S5` — `TASK-CTRL-04`):**
   - **`GAP-CTRL-AUDIT-01` (Lệch tên trường JSON `live_step_progress` giữa `telegram_bridge.go` và `app.js`):**
     - *Phát hiện:* `LiveStepProgressState` (`telegram_bridge.go:30-48`) xuất `step_index`, `stage_label`, `current_action`, `active_role_slot`, nhưng `renderGlobalRunningProgressBanner` và `renderSidebarTaskQueue` (`app.js:652-659, 735-736`) lại đọc `lsp.step_num`, `lsp.detail`, `lsp.title`, `lsp.role_label`, `lsp.agent_slot` khiến UI bị rơi về giá trị fallback tĩnh `le.active_step_num`.
     - *Khắc phục:* Đã đồng bộ `renderGlobalRunningProgressBanner`, `renderSidebarTaskQueue` và `renderLiveTelemetryHUD` trong `internal/webui/static/app.js:652-664, 731-737, 1603-1623` đọc trực tiếp `lsp.step_index`, `lsp.stage_label`, `lsp.current_action`, `lsp.current_where`, `lsp.active_role_slot`.
   - **`GAP-CTRL-AUDIT-02` (Thiếu chốt chặn `if ctx.Err() != nil { return nil }` sau vòng lặp lệnh trong `runSingleStageWithDelay`):**
     - *Phát hiện:* Trong `internal/webui/server.go`, nếu người dùng bấm `⏹ Dừng Tiến Trình` đúng lúc đang chạy vòng lặp `for _, repoPath := range affectedRepos` hoặc `BuildAndRunCalibratedLayaGate` ở `case "plan"`, `case "approve"`, `case "implement"`, hoặc `case "qa"`, hàm vẫn chạy tiếp xuống `s.QueueStore.UpdateStage` và `s.QueueStore.RecordRound`.
     - *Khắc phục:* Đã bổ sung đầy đủ chốt chặn `if ctx.Err() != nil { return nil }` tại `internal/webui/server.go:1310-1318, 1365-1367, 1460-1480, 1528-1530, 1558-1560, 1610-1612, 1622-1670, 1813-1829`.
   - **`GAP-CTRL-AUDIT-03` (Bỏ sót dọn thư mục `.orchestrator/worktrees/<safeKey>` khi xóa Task):**
     - *Phát hiện:* `DeleteSingleTask` và `DeleteDoneTasksBulk` (`internal/webui/task_inspector.go:4366-4369, 4410-4415`) mới chỉ xóa `.orchestrator/tasks/<safeKey>` mà quên xóa `.orchestrator/worktrees/<safeKey>`.
     - *Khắc phục:* Đã bổ sung `_ = os.RemoveAll(filepath.Join(s.Root, ".orchestrator", "worktrees", safeKey))` ở cả `DeleteSingleTask` (`task_inspector.go:4369`) và `DeleteDoneTasksBulk` (`task_inspector.go:4413`), kiểm chứng bằng assertion trong `TestWebUIServer_TaskDelete_LiveRunningProgress_And_StopProcessControl` (`EV-20261023T133716-8723`) và trên live `:8090` (`EV-20261023T134002-507a`).

---

## (e) Hội đồng `/laya_router` (`DG1/DG2` & `DG3`) Sau Lần Sửa Code Cuối

- **Cổng `DG1/DG2` (cho cập nhật `S3` `TASK-CTRL-04`):**
  - [`06_laya_summary_dg1_20261023T133002.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_laya_summary_dg1_20261023T133002.json) & [`06_laya_summary_dg2_20261023T133002.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_laya_summary_dg2_20261023T133002.json): **`gate: PASS`** (`trusted_rounds: ["r2"]`).
- **Cổng `DG3` (sau lần sửa code cuối):**
  - **Lệnh thực thi:**
    `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg3 --workspace audit-master-mongo-orchestrator-2026-10-22 --claims /Users/trainguyen/.gemini/antigravity/brain/029e438b-df78-4af7-bdf5-559bbd0375ab/scratch/dg3_claims_ctrl_audit.json`
  - **File kết quả:** [`06_laya_summary_dg3_20261023T134149.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_laya_summary_dg3_20261023T134149.json)
  - **Verdict:** **`PASS`** (`trusted_rounds: ["r3"]`, `1 control_true` + `2 control_false` calibrated):
    - `AUDIT-GAP-01`: `ACCEPT` (`r3=YES 0.9219`)
    - `AUDIT-GAP-02`: `FLAG -> RESOLVED` với `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1604-1615`
    - `AUDIT-GAP-03`: `FLAG -> RESOLVED` với `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:4366-4418`
    - `AUDIT-TEST-01`: `ACCEPT` (`r3=YES 0.8955`)

---

## (f) Tổng kết Bàn giao (`Gates G1–G8 = PASS`)

| Gate | Hạng mục Kiểm định | Trạng thái | Bằng chứng Vật lý |
| :---: | :--- | :---: | :--- |
| **G1** | Truy vết Yêu cầu (`TASK-CTRL-01 .. 04`) | **PASS** | `01_requirements_task_delete_live_progress_and_stop_control.md`, `08_tasks_task_delete_live_progress_and_stop_control.md` |
| **G2** | Reproduce & Fix (`READY` báo ảo `LIVE RUNNING`, thiếu Stop/Delete, `GAP-CTRL-AUDIT-01..03`) | **PASS** | `TestWebUIServer_TaskDelete_LiveRunningProgress_And_StopProcessControl` (`EV-20261023T133716-8723`, `EV-20261023T134001-ba47`) |
| **G3** | Test & Runtime thật (`:8090` + `daemon --once`) | **PASS** | `EV-20261023T133755-b32b`, `EV-20261023T134002-507a` |
| **G4** | Edge-case & Concurrency (`-race`, Stop mid-flight `ctx.Err()` guard, Tombstone + Worktree cleanup) | **PASS** | `EV-20261023T133800-d2f7` (`0` race conditions) |
| **G5** | Regression (`11/11` toàn bộ test suites cũ + mới) | **PASS** | `EV-20261023T133800-d2f7` |
| **G6** | Output Correctness (Xóa sạch SQLite + `.orchestrator/tasks/<key>` + `.orchestrator/worktrees/<key>`) | **PASS** | `EV-20261023T133716-8723`, `EV-20261023T134002-507a` |
| **G7** | Adversarial Line-by-Line Diff Audit & `/laya_router` `DG3` | **PASS** | `06_laya_summary_dg3_20261023T134149.json` (`gate: PASS`) |
| **G8** | Hồ sơ Vật lý (`06_validation_*`, `audit_report_*`, `05_progress_*`) | **PASS** | Lưu đầy đủ tại workspace `audit-master-mongo-orchestrator-2026-10-22` |

