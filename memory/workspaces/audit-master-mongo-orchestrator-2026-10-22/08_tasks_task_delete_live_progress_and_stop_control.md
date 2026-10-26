---
tier: T2
suffix: task_delete_live_progress_and_stop_control
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T11:34:10+07:00
---

# 08. Tasks — Xóa Task/Sub-Task, Hiển thị Tiến Trình Đang Chạy (`RUNNING...`) & Nút Dừng Tiến Trình (`TASK-CTRL-01 .. TASK-CTRL-03`)

## Danh sách Task Thực thi (Sau khi có `APPROVE`)

- [x] `TASK-CTRL-01` — Chức năng Xóa Task (`Delete Task`), Xóa Sub-Task (`Delete Sub-Task`) & Dọn nhanh Task DONE (`Bulk Cleanup`)
- [x] `TASK-CTRL-02` — Hệ thống Hiển thị Tiến Trình Đang Chạy Trực Quan (`Global Live Running Progress Banner`, `Animated Progress %`, `⟳ RUNNING...` Glow trên Sidebar / Wave / Pipeline / Sub-Task)
- [x] `TASK-CTRL-03` — Cơ chế Dừng Tiến Trình Khẩn Cấp (`⏹ Dừng Tiến Trình` / `Stop Task Execution` với `context.CancelFunc` & `AbortController`)
- [x] `TASK-CTRL-04` — [S5 Self-Healing Loop] Khắc phục 3 lỗ hổng phản biện từng dòng (`GAP-CTRL-AUDIT-01`: lệch tên trường JSON `live_step_progress` giữa Backend `step_index/stage_label/current_action/active_role_slot` và Frontend `step_num/title/detail/role_label`; `GAP-CTRL-AUDIT-02`: thiếu chốt chặn `if ctx.Err() != nil { return nil }` sau vòng lặp lệnh trong `runSingleStageWithDelay`; `GAP-CTRL-AUDIT-03`: bỏ sót dọn thư mục `.orchestrator/worktrees/<safeKey>` khi xóa Task)

---

### `TASK-CTRL-01` — Chức năng Xóa Task (`Delete Task`), Xóa Sub-Task (`Delete Sub-Task`) & Dọn nhanh Task DONE (`Bulk Cleanup`)
- **Mô tả & Context `file:line`:**
  - `internal/queue/store.go:322-354`: Bổ sung `DeleteTask(ctx context.Context, externalID string) error` và `DeleteDoneTasks(ctx context.Context, keepTaskKey string) ([]string, error)` xóa dữ liệu khỏi bảng `task_queue` và `autopilot_rounds`.
  - `internal/webui/seed_data.go:220-304` & `internal/webui/task_inspector.go:3701-3760`: Lưu danh sách `deleted_tasks.json` tại `.orchestrator/deleted_tasks.json` (qua helper `s.isTaskDeleted(taskKey)` / `s.markTaskDeleted(taskKey)` / `s.unmarkTaskDeleted(taskKey)`) để các task đã xóa không bị `SyncRealWorkspaceState` tự động khôi phục lại.
  - `internal/webui/server.go:627-643` & `internal/webui/server.go:953-1010`: Đăng ký route `POST /api/tasks/delete` (`handleDeleteTask`) và hỗ trợ `action: "delete"` / `"delete_done"` trong `handleTaskAction` + `action: "delete_subtask"` trong `handleTaskSubTasks` (`internal/webui/task_inspector.go:1582-1648`). Dọn sạch thư mục `.orchestrator/tasks/<safeKey>` và `.orchestrator/worktrees/<safeKey>`.
  - `internal/webui/static/index.html:64-106` & `internal/webui/static/app.js:439-480, 721-731, 990-1028, 1620-1654`: Thêm nút `🗑 Xóa` trên từng Task ở Sidebar, trên thẻ Task ở Wave Board, nút `🗑 Xóa Task` trên thanh Context Action Bar, nút `🧹 Xóa bớt DONE` ở đầu Sidebar, và nút `🗑` xóa từng Sub-Task.
- **Logs / Test:**
  - `go test -v -race -run TestWebUIServer_TaskDeleteLiveProgressAndStopControl ./internal/webui/...`
- **Definition of Done (DoD):**
  - Xóa 1 Task bất kỳ hoặc xóa hàng loạt Task `DONE` hoặc xóa 1 Sub-Task thành công, dọn sạch SQLite + thư mục `.orchestrator/tasks/<key>`, không bị tái tạo khi reload trang.

### `TASK-CTRL-02` — Hệ thống Hiển thị Tiến Trình Đang Chạy Trực Quan (`Global Live Running Progress Banner`, `Animated Progress %`, `⟳ RUNNING...` Glow trên Sidebar / Wave / Pipeline / Sub-Task)
- **Mô tả & Context `file:line`:**
  - `internal/webui/telegram_bridge.go:29-42, 783-825` & `internal/webui/task_chat.go:36-68, 509-573`: Mở rộng `LiveStepProgressState` và `LiveExecutionTelemetryDTO` bổ sung `SubTaskID`, `SubTaskTitle`, `ActionName`, `ElapsedSec`, `CanStop`, `WasStopped`, `StopReason`; cập nhật `runSubTaskLifecycleAction` (`internal/webui/task_inspector.go:2748-2885`) phát tiến độ từng bước (`advancePacedWorkflowStep`) khi chạy Sub-Task.
  - `internal/webui/static/index.html:61-95, 148-190`: Thêm thanh **`#global-running-progress-banner`** cố định ngay dưới thanh công cụ hiển thị trực diện tiến trình đang chạy (`Task`, `Sub-Task`, `Step X/17`, `Role/Slot/Model`, `Progress Bar %` chuyển động shimmer, đồng hồ `00:05s`, và nút `⏹ Dừng Tiến Trình`).
  - `internal/webui/static/app.js:1135-1215`: Sửa dứt điểm lỗi hiển thị `"⚡ LIVE RUNNING"` giả khi task đang ở trạng thái tĩnh `READY`/`IDLE`; bổ sung đồng hồ đếm giây thực tế và làm nổi bật thẻ Task/Sub-Task/Stage đang chạy bằng class `.is-actively-running` (viền phát sáng xanh dương + huy hiệu `⟳ RUNNING...` chuyển động + thanh mini progress bar `%`).
- **Logs / Test:**
  - `go test -v -race -run TestWebUIServer_TaskDeleteLiveProgressAndStopControl ./internal/webui/...`
- **Definition of Done (DoD):**
  - Khi không có gì chạy, UI hiển thị rõ `⏸ IDLE / READY (Không có tiến trình nào đang chạy)`. Khi bấm chạy bất kỳ Task hoặc Sub-Task nào, người dùng nhìn thấy ngay lập tức cái nào đang chạy, ở bước mấy, bao nhiêu `%`, đã chạy bao nhiêu giây trên cả Banner trên cùng, Sidebar, Wave, Pipeline Stage và Sub-Task card.

### `TASK-CTRL-03` — Cơ chế Dừng Tiến Trình Khẩn Cấp (`⏹ Dừng Tiến Trình` / `Stop Task Execution` với `context.CancelFunc` & `AbortController`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/server.go:242-261, 627-643, 1056-1165`: Bổ sung `activeExecCancel context.CancelFunc`, `activeExecTaskKey string`, `activeExecSubTaskID string` trong `Server`; đăng ký endpoint `POST /api/tasks/stop` (`handleStopTaskExecution`) và hỗ trợ `action: "stop"` trong `/api/tasks/action` & `/api/tasks/subtasks`.
  - Khi gọi `StopTaskExecution(ctx, taskKey)`: gọi `cancel()` ngắt ngay `advancePacedWorkflowStep` và `executeCommandInDir`, chuyển trạng thái Task (và Sub-Task đang `RUNNING`) về `NEEDS_YOU` (`StoppedByUser = true`), ghi log `[⏹ Dừng Tiến Trình]` vào `terminal.log` + `chat_history.json`, và phát SSE `"task_stopped"`.
  - `internal/webui/static/app.js:99-168, 608-629` & `internal/webui/static/index.html`: Tích hợp `AbortController` + hàm `stopActiveTaskExecution(taskKey, ev)` gắn vào nút **`⏹ Dừng Tiến Trình`** trên Global Running Banner, Context Action Bar, Live Telemetry HUD, Sidebar Task item và Sub-Task card.
- **Logs / Test:**
  - Kiểm chứng khởi chạy 1 tiến trình có `step_delay_ms: 3000`, gọi `POST /api/tasks/stop` giữa chừng -> tiến trình dừng tức thì (`< 200ms`), `live_execution.is_running == false`, `live_execution.was_stopped == true`, và `active_task_stage == "NEEDS_YOU"`.
- **Definition of Done (DoD):**
  - Nút `⏹ Dừng Tiến Trình` hiển thị rõ ràng và ngắt ngay lập tức tiến trình đang chạy cả ở Backend lẫn Frontend.

### `TASK-CTRL-04` — [S5 Self-Healing Loop] Khắc phục 3 lỗ hổng phản biện từng dòng (`GAP-CTRL-AUDIT-01..03`)
- **Mô tả & Context `file:line`:**
  - **`GAP-CTRL-AUDIT-01` (`internal/webui/static/app.js:652-659, 735-736, 1598-1619` & `internal/webui/telegram_bridge.go:30-47`):** Đồng bộ tên trường JSON của `live_step_progress` (`lsp.step_index`, `lsp.stage_label`, `lsp.current_action`, `lsp.current_where`, `lsp.active_role_slot`) trong `renderGlobalRunningProgressBanner`, `renderSidebarTaskQueue`, `renderLiveTelemetryHUD` để thanh Global Running Progress Banner, Sidebar và HUD hiển thị chính xác bước động thực tế từ `LiveStepProgressState` thay vì bị rơi về `undefined` / `le.active_step_num`.
  - **`GAP-CTRL-AUDIT-02` (`internal/webui/server.go:1462-1604, 1629-1855`):** Đặt chốt chặn `if ctx.Err() != nil { return nil }` ngay sau vòng lặp `for _, repoName := range affectedRepos` và sau `BuildAndRunCalibratedLayaGate` (trước mọi lời gọi `UpdateStage`, `RecordRound`, `triggerS5ToS3RePlan`) trong `runSingleStageWithDelay` để ngăn goroutine đã bị `StopTaskExecution` hủy tiếp tục ghi đè trạng thái `STOPPED / NEEDS_YOU`.
  - **`GAP-CTRL-AUDIT-03` (`internal/webui/task_inspector.go:4366-4372, 4410-4418`):** Bổ sung `os.RemoveAll(filepath.Join(s.Root, ".orchestrator", "worktrees", safeKey))` trong cả `DeleteSingleTask` và `DeleteDoneTasksBulk` để dọn sạch thư mục worktree cô lập khi xóa Task.
- **Logs / Test:**
  - `go test -v -race -run TestWebUIServer_TaskDelete_LiveRunningProgress_And_StopProcessControl ./internal/webui/...`
- **Definition of Done (DoD):**
  - Cả 3 lỗ hổng `GAP-CTRL-AUDIT-01..03` được sửa dứt điểm, kiểm chứng bằng unit/integration test và live runtime `:8090` trace.

