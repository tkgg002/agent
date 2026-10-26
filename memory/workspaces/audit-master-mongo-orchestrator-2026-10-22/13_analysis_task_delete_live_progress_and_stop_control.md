---
tier: T2
suffix: task_delete_live_progress_and_stop_control
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T11:35:25+07:00
---

# 13. Analysis — Điều tra Nguyên nhân Gốc rễ (`file:line`) cho Xóa Task, Tiến Trình Đang Chạy (`RUNNING...`) & Nút Dừng Tiến Trình

## 1. Bảng phân loại Bằng chứng (`VERIFIED` vs `HYPOTHESIS`)

| Mã | Phân loại | Điểm chạm `file:line` (`REF:`) | Phát hiện thực tế trên mã nguồn |
| :--- | :---: | :--- | :--- |
| `F-CTRL-01` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/queue/store.go:322-354` | `queue.Store` chỉ có `ListAll`, `EnqueueRaw`, `UpdateStage`, `PurgeMockSeedData` — hoàn toàn **chưa có** hàm `DeleteTask(ctx, externalID)` hay `DeleteDoneTasks(ctx, keepKey)` để xóa task do người dùng chọn khỏi bảng `task_queue` và `autopilot_rounds`. |
| `F-CTRL-02` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/seed_data.go:220-304` | Hàm `SyncRealWorkspaceState` tự động đọc `08_tasks_*.md` và gọi `EnqueueRaw` cho mọi task chưa có trong `existingByKey`. Nếu không lưu danh sách `.orchestrator/deleted_tasks.json` (tombstone), bất kỳ task nào từ workspace sau khi xóa sẽ bị tái tạo lại ngay khi khởi động lại server. |
| `F-CTRL-03` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:1156-1170` | Trong `renderLiveTelemetryHUD`, câu lệnh `if (st === "NEEDS_YOU") ... else if (st === "DONE") ... else { indEl.textContent = "⚡ LIVE RUNNING"; indEl.className = "hud-live-badge running"; }` khiến mọi task ở trạng thái tĩnh `READY` hoặc `BLOCKED` (dù `le.is_running == false` và không có tiến trình nào chạy) vẫn hiện chữ `"⚡ LIVE RUNNING"`, làm người dùng không phân biệt được lúc nào đang chạy thật, lúc nào đang đứng yên! |
| `F-CTRL-04` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:439-480` | Danh sách Task ở Sidebar (`renderSidebarTaskQueue`), thẻ Task ở Wave Board (`app.js:1620-1654`), và thẻ Sub-Task (`app.js:990-1028`) chưa có nút `🗑 Xóa`, chưa có hiệu ứng viền chuyển động (`is-actively-running` / `⟳ RUNNING...` spinner + thanh mini progress `%`) khi Task hoặc Sub-Task đó đang chạy, và chưa có nút `⏹ Dừng` trực tiếp. |
| `F-CTRL-05` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:242-261` | Struct `Server` chỉ lưu `liveStepProgress LiveStepProgressState` mà **chưa có** `activeExecCancel context.CancelFunc`, `activeExecTaskKey string`, `activeExecSubTaskID string` hay endpoint `POST /api/tasks/stop` để hủy lập tức tiến trình đang chạy (`advancePacedWorkflowStep` / `executeCommandInDir`) khi người dùng bấm nút dừng. |
| `F-CTRL-06` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:608-629` | Hàm `runSubTaskLifecycle` khi bấm chạy 1 Sub-Task (`subtask_plan`, `subtask_implement`, `subtask_qa`, `subtask_autopilot`) không bật cờ `isLifecycleActionRunning`, không có đồng hồ polling / progress bar, và ở backend (`task_inspector.go:2748`) chưa cập nhật `liveStepProgress` có `SubTaskID` nên người dùng bấm chạy Sub-Task mà không thấy tiến trình nhảy bước. |
