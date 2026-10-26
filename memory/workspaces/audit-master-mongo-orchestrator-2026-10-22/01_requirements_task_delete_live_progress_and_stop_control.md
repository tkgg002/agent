---
tier: T2
suffix: task_delete_live_progress_and_stop_control
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T11:34:00+07:00
---

# 01. Requirements — Xóa Task/Sub-Task (`Delete Task`), Hiển thị Tiến Trình Đang Chạy Trực Quan (`Live Running Progress UI`) & Nút Dừng Tiến Trình (`Stop Running Process`)

## 1. Bối cảnh & Yêu cầu của User
User chỉ đạo trực tiếp 3 yêu cầu cốt lõi để giải quyết triệt để sự rối mắt và mù trạng thái khi vận hành `ai-orchestrator`:
> *"thêm chức năng xoá các task, dể nhìn bớt rôí. rồi có 1 vấn đề tao đã nói đi nói lại là tao đang ko biết cái nào đang chạy, phải có progess cho tao (có hiệu ứng running ... hay chỉnh ui cho nó dễ nhìn hơn), và thêm nút dừng tiến trình. mày hiểu ko"*

## 2. Phân rã Yêu cầu Chức năng & Kỹ thuật (`R1 – R3`)

### `R1` — Chức năng Xóa Task, Xóa Sub-Task & Dọn dẹp Nhanh (`Delete Task / Sub-Task & Bulk Cleanup`)
1. **Backend SQLite & Disk Cleanup (`internal/queue/store.go` & `internal/webui/server.go`):**
   - Bổ sung `Store.DeleteTask(ctx context.Context, externalID string) error` và `Store.DeleteDoneTasks(ctx context.Context, keepTaskKey string) ([]string, error)` trong `internal/queue/store.go` để xóa bản ghi khỏi `task_queue` và `autopilot_rounds`.
   - Lưu danh sách `deleted_tasks.json` tại `.orchestrator/deleted_tasks.json` để `SyncRealWorkspaceState` (`internal/webui/seed_data.go:239-303`) và `restoreSmokeCompletedUserTasks` (`internal/webui/task_inspector.go:3701`) **TUYỆT ĐỐI KHÔNG** tự động nạp lại các task mà người dùng đã chủ động xóa.
   - Bổ sung endpoint `POST /api/tasks/delete` (và hỗ trợ `action: "delete"` trong `/api/tasks/action` + `action: "delete_subtask"` trong `/api/tasks/subtasks`):
     - Nếu task đang chạy, tự động gọi `StopTaskExecution` hủy tiến trình trước khi xóa.
     - Dọn sạch thư mục `.orchestrator/tasks/<safeKey>` và `.orchestrator/worktrees/<safeKey>`.
     - Tự động chuyển `selectedTaskKey` sang task còn lại tiếp theo.
2. **Frontend UI (`index.html`, `app.js`, `app.css`):**
   - Nút **`🗑 Xóa`** trên từng dòng Task ở Sidebar bên trái (`#sidebar-task-queue`), trên từng thẻ Task ở màn hình Wave (`#wave-tier-tasks`), và nút **`🗑 Xóa Task đang chọn`** trên thanh Action Bar (`#cross-screen-context-bar`).
   - Nút **`🧹 Xóa bớt Task DONE`** ngay trên đầu cột `Hàng Đợi Task` ở Sidebar để dọn nhanh các task đã xong chỉ bằng 1 cú nhấp chuột.
   - Nút **`🗑 Xóa Sub-Task`** trên từng thẻ Sub-Task (`#sri-subtasks-list` & `#subtask-fractal-inspector`).

### `R2` — Hiển thị Tiến Trình Đang Chạy Trực Quan (`Live Running Progress Banner, Animated Progress Bar %, Spinner & Highlighted Active Item`)
1. **Khắc phục lỗi hiển thị sai trạng thái `LIVE RUNNING` khi đứng yên (`app.js:1158-1168`):**
   - Sửa `renderLiveTelemetryHUD` để phân biệt chính xác giữa trạng thái **Đang chạy thực sự (`le.is_running === true` hoặc `isLifecycleActionRunning === true`)** với trạng thái **Đang chờ lệnh (`READY` / `IDLE`)**, **Đang chờ duyệt (`NEEDS_YOU`)**, **Đã dừng (`STOPPED`)**, và **Đã hoàn tất (`DONE`)**.
2. **Bổ sung Paced Progress cho cả Task lẫn Sub-Task (`task_inspector.go:2748` & `app.js:608`):**
   - Khi chạy bất kỳ hành động nào của Sub-Task (`subtask_plan`, `subtask_implement`, `subtask_qa`, `subtask_autopilot`), cập nhật `liveStepProgress` (`TaskKey`, `SubTaskID`, `StageLabel`, `StepIndex`, `TotalSteps`, `Percent`, `CurrentAction`, `ActiveRoleSlot`) và phát SSE `step_progress` để giao diện hiển thị tiến độ trực quan theo thời gian thực.
3. **Thanh `Sticky Live Running Progress Banner` + Hiệu ứng `RUNNING...` nổi bật trên toàn UI (`index.html`, `app.js`, `app.css`):**
   - Bổ sung thanh **`#global-running-progress-banner`** nổi bật ngay dưới thanh công cụ (tự động bật sáng với hiệu ứng chuyển động shimmer + spinner xoay `⟳ RUNNING...` khi có tiến trình chạy), hiển thị rõ:
     - **Đang chạy Task / Sub-Task nào:** `🎯 TASK-11 › 🧩 TASK-M03`
     - **Đang ở bước nào & Ai đang làm:** `Step 10/17 · S4 Multi-Repo Implement · 🛠️ MUSCLE (Slot B · gemini-4-argon)`
     - **Thanh Progress Bar `%` chuyển động mượt + Đồng hồ đếm giây thực tế (`00:06s · 70%`)**
     - **Nút `⏹ Dừng Tiến Trình` nằm ngay bên phải thanh Progress**.
   - Làm nổi bật rõ ràng thẻ **Task đang chạy** (Sidebar & Wave Board), **Sub-Task đang chạy** (`#sri-subtasks-list`), và **Ô Stage đang chạy** (`#autopilot-flow` & `#sfi-pipeline-row`) bằng viền sáng chuyển động (`running-glow-card`), huy hiệu `⟳ RUNNING...` nhấp nháy, và thanh mini-progress bar ngay bên trong thẻ.

### `R3` — Nút Dừng Tiến Trình Khẩn Cấp (`Stop / Cancel Running Process`)
1. **Backend Cancellable Execution Context (`internal/webui/server.go`, `internal/webui/telegram_bridge.go`, `internal/webui/task_inspector.go`):**
   - Bổ sung bộ quản lý tiến trình đang chạy trong `Server`: `activeCancel context.CancelFunc`, `activeExecTaskKey string`, `activeExecSubTaskID string` (được bảo vệ bởi `s.mu`).
   - Khi `runTaskLifecycleActionWithDelay` hoặc `runSubTaskLifecycleAction` chạy, khởi tạo `execCtx, cancel := s.startCancelableExecution(ctx, taskKey, subTaskID, action)` và kiểm tra `execCtx.Err()` tại mỗi bước `advancePacedWorkflowStep` cũng như trước/trong `executeCommandInDir`.
   - Bổ sung endpoint `POST /api/tasks/stop` (và hỗ trợ `action: "stop"` trong `/api/tasks/action` & `/api/tasks/subtasks`):
     - Gọi `cancel()` ngắt ngay tiến trình đang chạy (`< 100ms`).
     - Chuyển trạng thái Task / Sub-Task đang chạy về `NEEDS_YOU` (kèm cờ `StoppedByUser: true` và log `[⏹ Dừng Tiến Trình] Đã dừng tiến trình theo lệnh người dùng`).
2. **Frontend Nút `⏹ Dừng Tiến Trình` (`index.html`, `app.js`, `app.css`):**
   - Nút **`⏹ Dừng Tiến Trình`** xuất hiện trực diện trên:
     1. Thanh **Global Running Progress Banner** (`#global-running-progress-banner`)
     2. Thanh **Context Action Bar** (`#ctx-btn-stop-task`) & **Live Telemetry HUD** (`#hud-btn-stop-task`)
     3. Ngay trên **thẻ Task ở Sidebar / Wave** và **thẻ Sub-Task** đang chạy
   - Kết hợp `AbortController` ở client + gọi `POST /api/tasks/stop` để dừng tức thì cả vòng polling lẫn tiến trình trên server.

## 3. Ràng buộc Kỷ luật `/agent`
- Tuân thủ nghiêm ngặt `S1 -> S2 -> S3 -> DG1/DG2 -> Dừng chờ APPROVE` (Rule #9, Rule #13).
- Không chạm vào các repo `/Users/trainguyen/Documents/work/data-hub/*`.
