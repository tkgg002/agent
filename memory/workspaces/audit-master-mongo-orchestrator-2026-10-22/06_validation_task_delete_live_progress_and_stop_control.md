---
tier: T2
suffix: task_delete_live_progress_and_stop_control
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T12:41:30+07:00
---

# 06. Validation Report — Xóa Task/Sub-Task, Hiển thị Tiến Trình Đang Chạy (`RUNNING...`) & Nút Dừng Tiến Trình (`TASK-CTRL-01 .. TASK-CTRL-04`)

## 1. Danh mục Biên lai Thực thi Vật lý (`EV-...` từ `evidence_run.py`)

| Biên lai (`EV-...`) | Phân loại | Lệnh kiểm định thực tế | Kết quả |
| :--- | :---: | :--- | :---: |
| `EV-20261023T134000-3dd4` | `unit` | `go vet ./...` | `PASS` (`exit=0`) |
| `EV-20261023T133716-8723` | `integration` | `go test -v -race -run TestWebUIServer_TaskDelete_LiveRunningProgress_And_StopProcessControl ./internal/webui/...` (`3.95s`) | `PASS` (`exit=0`) |
| `EV-20261023T134001-ba47` | `integration` | `go test -v -race -run TestWebUIServer_TaskDelete_LiveRunningProgress_And_StopProcessControl ./internal/webui/...` | `PASS` (`exit=0`) |
| `EV-20261023T133800-d2f7` | `integration` | `go test -v -race ./internal/webui/... ./internal/queue/...` (`11/11` test suites PASS, `63.799s`) | `PASS` (`exit=0`) |
| `EV-20261023T133755-b32b` | `runtime` | `go run ./cmd/orchestrator daemon --once` | `PASS` (`exit=0`) |
| `EV-20261023T133759-baee` | `unit` | `go build -o bin/orchestrator ./cmd/orchestrator` | `PASS` (`exit=0`) |
| `EV-20261023T134002-507a` | `runtime` | `python3 verify_live_8090_ctrl_audit.py` (Live HTTP `:8090` trace: `observed_step_index=2`, `observed_stage_label="S0 Boot H1 & S1 Intake Scope"`, `observed_role_slot="🧠 BRAIN · Slot A"`, `stopped_stage="NEEDS_YOU"`, `worktree_removed=true`) | `PASS` (`exit=0`) |

## 2. Kết quả Kiểm định E2E & Vòng lặp Tự chữa lành `S5 -> S3 -> S4 -> S5` (`TASK-CTRL-01 .. TASK-CTRL-04`)

- **`TASK-CTRL-02` & `GAP-CTRL-AUDIT-01` (Chống báo ảo `⚡ LIVE RUNNING` khi `READY` & Đồng bộ trường JSON `live_step_progress`):**
  - Khi `TASK-CTRL-TEST` vừa khởi tạo ở trạng thái `READY` (chưa bấm chạy), `dashIdle.LiveExecution.IsRunning == false` và `dashIdle.LiveStepProgress.Active == false`.
  - Khi kích hoạt `/api/tasks/action` (`action: "plan"`, `step_delay_ms: 1500`), `LiveStepProgress` phát trực tiếp `step_index > 0`, `stage_label`, `current_action`, `active_role_slot`, `percent > 0`, `can_stop = true`, và `app.js` (`renderGlobalRunningProgressBanner`, `renderSidebarTaskQueue`, `renderLiveTelemetryHUD`) đọc khớp 100% các trường JSON này.
- **`TASK-CTRL-03` & `GAP-CTRL-AUDIT-02` (Dừng Tiến Trình Khẩn Cấp `POST /api/tasks/stop` & Chốt chặn `if ctx.Err() != nil`):**
  - Gọi `POST /api/tasks/stop` giữa lúc tiến trình đang chạy -> `StopTaskExecution` (`internal/webui/task_inspector.go:4274-4339`) gọi `cancel()` ngắt ngay `context.Context`, hạ `LiveStepProgress.Active = false`, đặt `WasStopped = true`, `CanStop = false`, chuyển trạng thái Task về `NEEDS_YOU`, và các chốt chặn `if ctx.Err() != nil { return nil }` trong `runSingleStageWithDelay` (`internal/webui/server.go`) bảo đảm goroutine bị hủy không ghi đè trạng thái sau khi dừng.
- **`TASK-CTRL-01` & `GAP-CTRL-AUDIT-03` (Xóa Sub-Task, Xóa Task Đơn Lẻ, Dọn Nhanh Task DONE kèm Tombstone & Dọn sạch `.orchestrator/worktrees/<safeKey>`):**
  - `POST /api/tasks/delete` (`mode: "subtask"`) xóa Sub-Task khỏi danh sách, lọc sạch `DependsOn` của các Sub-Task còn lại, xóa thư mục `.orchestrator/tasks/<taskKey>/subtasks/<subID>`, và đồng bộ lại `08_tasks.md`.
  - `POST /api/tasks/delete` (`mode: "done"` và `mode: "single"`) xóa dứt điểm Task khỏi SQLite (`task_queue` + `autopilot_rounds`), xóa cả `.orchestrator/tasks/<safeKey>` lẫn `.orchestrator/worktrees/<safeKey>`, và ghi Tombstone vào `.orchestrator/deleted_tasks.json` để `SyncRealWorkspaceState` (`internal/webui/seed_data.go:234-243`) không bao giờ tự khôi phục lại Task đã xóa.

