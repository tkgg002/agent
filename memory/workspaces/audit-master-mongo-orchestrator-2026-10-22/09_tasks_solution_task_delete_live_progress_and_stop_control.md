---
tier: T2
suffix: task_delete_live_progress_and_stop_control
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T11:35:40+07:00
---

# 09. Tasks Solution — Thiết kế Giải pháp & Code Demo cho Xóa Task/Sub-Task, Tiến Trình Đang Chạy (`RUNNING...`) & Nút Dừng Tiến Trình

> **Tuân thủ Rule #9.4:** Trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** (Single-Best Solution) kèm Code Demo cụ thể cho cả 3 hạng mục `TASK-CTRL-01 .. TASK-CTRL-03`.

---

## 1. Kiến trúc Giải pháp Tối ưu Duy nhất (Single-Best Architecture)

### 1.1. `TASK-CTRL-01` — Xóa Task, Xóa Sub-Task & Dọn nhanh Task `DONE` (Chống Tái tạo Tombstone)
- **Tầng Lưu trữ SQLite (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/queue/store.go:322-354`):**
  - Bổ sung `Store.DeleteTask(ctx context.Context, externalID string) error`: xóa nguyên tử trong transaction cả `DELETE FROM task_queue WHERE external_id = ?` và `DELETE FROM autopilot_rounds WHERE task_key = ?`.
  - Bổ sung `Store.DeleteDoneTasks(ctx context.Context, keepTaskKey string) ([]string, error)`: tìm toàn bộ task có `stage = 'DONE'` (trừ `keepTaskKey` nếu được chỉ định), xóa khỏi `task_queue` + `autopilot_rounds`, và trả về danh sách `deletedKeys`.
- **Tầng Chống Tái tạo & Dọn Đĩa (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/seed_data.go:220-304` & `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:621-643`):**
  - Lưu file `.orchestrator/deleted_tasks.json` (map các `taskKey` đã bị người dùng xóa).
  - Trong `SyncRealWorkspaceState` (`seed_data.go:239`), nếu `s.isTaskDeleted(pt.Key)` trả về `true` thì bỏ qua (`continue`), tuyệt đối không nạp lại task đã xóa. Khi người dùng chủ động tạo lại task trùng tên qua `handleCreateTask`, gọi `s.unmarkTaskDeleted(req.ExternalID)`.
  - Đăng ký endpoint `POST /api/tasks/delete` (`handleDeleteTask`) và hỗ trợ `action: "delete"` / `"delete_done"` trong `handleTaskAction` + `action: "delete_subtask"` trong `handleTaskSubTasks`:
    - Nếu task đang chạy, gọi `s.StopTaskExecution(ctx, taskKey, "Stopped before task deletion")` trước.
    - Xóa khỏi SQLite, ghi tombstone vào `.orchestrator/deleted_tasks.json`, xóa thư mục `.orchestrator/tasks/<safeKey>` và `.orchestrator/worktrees/<safeKey>`.
    - Tự động chọn lại `s.selectedTaskKey` sang task đầu tiên còn lại.
- **Tầng Giao diện (`index.html`, `app.js`, `app.css`):**
  - Thêm nút **`🗑`** xóa trực tiếp trên từng thẻ Task ở Sidebar (`#sidebar-task-queue`), trên thẻ Task ở màn hình Wave (`#wave-tier-tasks`), nút **`🗑 Xóa Task`** trên thanh Context Action Bar (`#cross-screen-context-bar`), nút **`🧹 Dọn Task DONE`** ở tiêu đề Sidebar, và nút **`🗑 Xóa Sub-Task`** trên từng dòng Sub-Task.

### 1.2. `TASK-CTRL-02` — Hệ thống Hiển thị Tiến Trình Đang Chạy Trực Quan (`Live Running Progress Banner + Animated Spinner + Progress Bar % + Highlighted Active Item`)
- **Mở rộng Telemetry Đang Chạy (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:29-42` & `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:36-68`):**
  - Bổ sung vào `LiveStepProgressState`:
    - `SubTaskID string`, `SubTaskTitle string`, `ActionName string`, `Stopped bool`, `StopReason string`.
  - Bổ sung vào `LiveExecutionTelemetryDTO`:
    - `ActiveAction string`, `ElapsedSec int`, `CanStop bool`, `WasStopped bool`, `StopReason string`.
  - Trong `runSubTaskLifecycleAction` (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:2748-2885`), cập nhật `liveStepProgress` có gắn `SubTaskID` & `SubTaskTitle` ở từng pha (`subtask_plan`, `subtask_implement`, `subtask_qa`, `subtask_autopilot`) để khi chạy Sub-Task thì thanh Progress vẫn hiển thị rõ đang chạy `Sub-Task` nào.
- **Thanh `Sticky Global Running Progress Banner` (`#global-running-progress-banner` trong `index.html` & `app.js`):**
  - Đặt ngay dưới thanh Context Bar, luôn hiển thị nổi bật:
    - **Khi đang chạy (`is_running == true` hoặc `isLifecycleActionRunning == true`):** Bật nền xanh chuyển động (`shimmer`), icon xoay `⟳ ĐANG CHẠY TIẾN TRÌNH (RUNNING...)`, hiển thị rõ:
      - **Task & Sub-Task đang chạy:** `🎯 TASK-11 › 🧩 TASK-M03`
      - **Bước & Tác vụ hiện tại:** `Step 10/17 · S4 Multi-Repo Worktree Implementation (🛠️ MUSCLE · Slot B · gemini-4-argon)`
      - **Thanh Progress Bar `%` chạy mượt + Đồng hồ đếm giây thực tế (`00:05s · 70%`)**
      - **Nút đỏ nổi bật `⏹ Dừng Tiến Trình` (`Stop Process`)** nằm ngay bên phải.
    - **Khi vừa bấm Dừng (`was_stopped == true`):** Hiển thị rõ `⏹ ĐÃ DỪNG TIẾN TRÌNH THEO LỆNH CỦA BẠN (STOPPED -> NEEDS_YOU)`.
    - **Khi đứng yên (`is_running == false`):** Sửa lỗi `renderLiveTelemetryHUD` (`app.js:1158-1168`) — hiển thị rõ `⏸ IDLE · Không có tiến trình nào đang chạy (Sẵn sàng)` thay vì báo giả `"⚡ LIVE RUNNING"`.
- **Hiệu ứng `RUNNING...` trực tiếp trên thẻ Task, Sub-Task và Ô Stage (`app.js` & `app.css`):**
  - Thẻ Task đang chạy ở Sidebar (`#sidebar-task-queue`) và Wave Board (`#wave-tier-tasks`), thẻ Sub-Task đang chạy (`#sri-subtasks-list`), và ô Stage đang chạy (`#autopilot-flow`) được gắn class `.is-actively-running` (viền sáng xanh chuyển động, huy hiệu `⟳ RUNNING...` nhấp nháy, thanh mini progress bar `%` ngay trong thẻ, và nút `⏹ Dừng` tại chỗ).

### 1.3. `TASK-CTRL-03` — Nút Dừng Tiến Trình Khẩn Cấp (`⏹ Dừng Tiến Trình` / `Stop Running Process`)
- **Backend Cancellable Context Registry (`REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:242-261, 1056-1165`):**
  - Thêm vào `Server`:
    - `activeExecCancel context.CancelFunc`
    - `activeExecTaskKey string`
    - `activeExecSubTaskID string`
    - `activeExecAction string`
    - `lastStoppedTaskKey string`
    - `lastStopReason string`
  - Hàm `s.beginCancelableTaskExecution(ctx, taskKey, subTaskID, action)` tạo `execCtx, cancel := context.WithCancel(ctx)` (hủy tiến trình cũ nếu đang chạy trùng) và trả về `execCtx, finishFunc`.
  - Hàm `s.StopTaskExecution(ctx, taskKey, reason)`:
    - Gọi `cancel()` để đánh thức ngay lập tức `select { case <-time.After(stepDelay): case <-ctx.Done(): }` trong `advancePacedWorkflowStep` (`telegram_bridge.go:811-815`) và hủy mọi lệnh `exec.CommandContext(runCtx, ...)` đang chạy trong `executeCommandInDir` (`server.go:375-377`).
    - Cập nhật `Stage` của Task trong SQLite về `NEEDS_YOU` và mọi Sub-Task đang `RUNNING` về `NEEDS_YOU`.
    - Lưu trạng thái `lastStoppedTaskKey = taskKey`, `lastStopReason = reason`, ghi log `[⏹ Dừng Tiến Trình] Đã dừng tiến trình đang chạy theo lệnh người dùng` vào `terminal.log` và phát SSE `"task_stopped"`.
  - Trong `runSingleStageWithDelay`, `runAutonomousS4S5WithSelfHealing`, và `runSubTaskLifecycleAction`: kiểm tra `if ctx.Err() != nil { return nil }` sau mỗi bước để thoát sạch ngay lập tức mà không chạy sang bước kế tiếp.

---

## 2. Concrete Code Demo (`DG2` Grounded Implementation)

### 2.1. Backend: `DeleteTask`, `StopTaskExecution` & `LiveStepProgressState` (`internal/queue/store.go` & `internal/webui/server.go`)
```go
// internal/queue/store.go
func (s *Store) DeleteTask(ctx context.Context, externalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return errors.New("queue: external_id is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM autopilot_rounds WHERE task_key = ?`, externalID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_queue WHERE external_id = ?`, externalID); err != nil {
		return err
	}
	return tx.Commit()
}

// internal/webui/server.go
func (s *Server) StopTaskExecution(ctx context.Context, taskKey, reason string) (string, error) {
	s.mu.Lock()
	targetKey := strings.TrimSpace(taskKey)
	if targetKey == "" {
		targetKey = s.activeExecTaskKey
	}
	if targetKey == "" {
		targetKey = s.liveStepProgress.TaskKey
	}
	if targetKey == "" {
		targetKey = s.selectedTaskKey
	}
	cancelFn := s.activeExecCancel
	activeSubID := s.activeExecSubTaskID
	s.activeExecCancel = nil
	s.activeExecTaskKey = ""
	s.activeExecSubTaskID = ""
	s.activeExecAction = ""
	if strings.TrimSpace(reason) == "" {
		reason = "Đã dừng tiến trình theo lệnh người dùng (Stopped by User)"
	}
	s.lastStoppedTaskKey = targetKey
	s.lastStopReason = reason
	s.liveStepProgress = LiveStepProgressState{
		Active:        false,
		TaskKey:       targetKey,
		SubTaskID:     activeSubID,
		StageLabel:    "⏹ Đã dừng tiến trình (STOPPED BY USER)",
		CurrentAction: reason,
		Stopped:       true,
		StopReason:    reason,
		StartedAt:     time.Now(),
	}
	s.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}
	if qt, err := s.findQueuedTask(ctx, targetKey); err == nil && qt != nil {
		isoMeta := s.buildTaskIsolationMeta(qt.ExternalID)
		_ = s.QueueStore.UpdateStage(ctx, qt.ID, queue.StageNeedsYou, qt.AssignedModel, isoMeta.WorktreePath)
		items := s.loadOrGenerateSubTasks(*qt)
		for i := range items {
			if strings.EqualFold(items[i].Stage, "RUNNING") || strings.EqualFold(items[i].Status, "running") ||
				(activeSubID != "" && strings.EqualFold(items[i].SubTaskID, activeSubID)) {
				items[i].Stage = "NEEDS_YOU"
				items[i].Status = "waiting"
			}
		}
		_ = s.saveTaskSubTasks(qt.ExternalID, items)
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[⏹ Dừng Tiến Trình] %s (Task=%s, SubTask=%s) -> Chuyển về NEEDS_YOU", reason, qt.ExternalID, activeSubID))
	}
	s.broadcast("task_stopped")
	return targetKey, nil
}
```

### 2.2. Frontend: `stopActiveTaskExecution`, `deleteTaskItem`, & `renderGlobalRunningBanner` (`internal/webui/static/app.js`)
```javascript
let activeActionAbortController = null;
let clientRunningState = {
  active: false,
  taskKey: "",
  subTaskID: "",
  action: "",
  startedAtMs: 0,
};

async function stopActiveTaskExecution(taskKey, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const targetKey =
    taskKey ||
    clientRunningState.taskKey ||
    dashboardState?.live_execution?.active_task_key ||
    dashboardState?.active_task_key ||
    selectedChainKey ||
    "";
  if (activeActionAbortController) {
    try {
      activeActionAbortController.abort();
    } catch (_) {}
    activeActionAbortController = null;
  }
  clientRunningState.active = false;
  isLifecycleActionRunning = false;

  const res = await fetch("/api/tasks/stop", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      task_key: targetKey,
      reason: "Người dùng bấm nút ⏹ Dừng Tiến Trình trên UI",
    }),
  });
  if (res.ok) {
    dashboardState = await res.json();
    renderAll();
  }
}

async function deleteTaskItem(taskKey, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const targetKey = taskKey || dashboardState?.active_task_key || selectedChainKey || "";
  if (!targetKey) return;
  const res = await fetch("/api/tasks/delete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ task_key: targetKey }),
  });
  if (!res.ok) return;
  dashboardState = await res.json();
  selectedChainKey = dashboardState.active_task_key || "";
  renderAll();
}
```

---

## 3. [S5 -> S3 Self-Healing Update] Khắc phục 3 Lỗ Hổng Phản Biện Từng Dòng (`GAP-CTRL-AUDIT-01 .. GAP-CTRL-AUDIT-03`)

### 3.1. `GAP-CTRL-AUDIT-01` — Đồng bộ tên trường JSON `live_step_progress` giữa Backend và Frontend (`internal/webui/static/app.js` & `internal/webui/telegram_bridge.go`)
- Trong `internal/webui/static/app.js`:
  - Tại `renderGlobalRunningProgressBanner` (`app.js:652-660`): đọc `lsp.step_index || lsp.step_num || le.active_step_num || 1`, `lsp.total_steps || le.total_steps || 17`, `[lsp.stage_label, lsp.current_action].filter(Boolean).join(" — ") || lsp.detail || le.active_step_title`, và `lsp.active_role_slot || lsp.role_label || le.active_role_label`.
  - Tại `renderSidebarTaskQueue` (`app.js:735-738`): đọc `lsp.step_index || lsp.step_num || 1` và `lsp.stage_label || lsp.current_action || lsp.title || "Đang xử lý..."`.
  - Tại `renderLiveTelemetryHUD` (`app.js:1598-1622`): khi `isRunning` là `true`, ưu tiên hiển thị `lsp.step_index`, `lsp.stage_label`, `lsp.current_action`, `lsp.current_where`, `lsp.active_role_slot` theo thời gian thực để HUD khớp 100% với Global Running Progress Banner.

### 3.2. `GAP-CTRL-AUDIT-02` — Chốt chặn `if ctx.Err() != nil { return nil }` sau vòng lặp lệnh trong `runSingleStageWithDelay` (`internal/webui/server.go`)
- Trong `runSingleStageWithDelay` (`internal/webui/server.go:1462-1604, 1629-1855`):
  - Ngay sau vòng lặp `for _, repoName := range affectedRepos` ở cả `case "implement"` và `case "qa"`, và ngay sau `BuildAndRunCalibratedLayaGate` ở `case "qa"`, bổ sung chốt chặn:
```go
if ctx.Err() != nil {
	return nil
}
```
  - Ngăn chặn tuyệt đối goroutine đã bị `StopTaskExecution` hủy (`ctx.Err() == context.Canceled`) tiếp tục chạy xuống `s.QueueStore.UpdateStage(...)`, `s.QueueStore.RecordRound(...)`, hoặc `s.triggerS5ToS3RePlan(...)`.

### 3.3. `GAP-CTRL-AUDIT-03` — Dọn sạch thư mục `.orchestrator/worktrees/<safeKey>` khi xóa Task (`internal/webui/task_inspector.go`)
- Trong `DeleteSingleTask` (`task_inspector.go:4366-4372`) và `DeleteDoneTasksBulk` (`task_inspector.go:4410-4418`), xóa đồng bộ cả `.orchestrator/tasks/<safeKey>` lẫn `.orchestrator/worktrees/<safeKey>`:
```go
taskDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeKey)
_ = os.RemoveAll(taskDir)
worktreeDir := filepath.Join(s.Root, ".orchestrator", "worktrees", safeKey)
_ = os.RemoveAll(worktreeDir)
```

