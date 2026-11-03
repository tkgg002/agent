---
tier: T2
suffix: add_edit_task_and_subtask_crud
---

# ⚡ 10 Pre-Plan Adversarial Critique (`DG0`) — `add_edit_task_and_subtask_crud`

## 1. Phản Biện Giả Định & Kiểm Chứng Thực Tế (`C-REQ`, `C-REALITY`, `C-CONTRA`)
1. **`C-REQ` (Kiểm chứng nhu cầu thực tế của User):**
   - Tại `internal/webui/static/index.html:68-91` (`#cross-screen-context-bar`), `internal/webui/static/app.js:869-911` (`renderSidebarTaskQueue`), và `internal/webui/static/app.js:2239-2248` (`renderWaveBoard`), hệ thống chỉ có nút Chạy (`▶`), Dừng (`⏹`), và Xóa (`🗑`), hoàn toàn không có nút `✏️ Sửa Task`.
   - Tại `internal/webui/server.go:639-663`, router HTTP chỉ đăng ký `/api/tasks` (`handleCreateTask`), `/api/tasks/delete` (`handleDeleteTask`), `/api/tasks/stop` (`handleStopTask`), chưa có `/api/tasks/detail` hay `/api/tasks/update`.
2. **`C-REALITY` (Kiểm chứng tầng SQLite `internal/queue/store.go:183-193`):**
   - Trong `EnqueueRaw` (`internal/queue/store.go:183-193`), khi `external_id` đã tồn tại, câu lệnh `UPDATE task_queue` chỉ cập nhật `source, title, description, severity, labels_json, complexity, updated_at` mà **không cập nhật `assigned_model`**, đồng thời `handleCreateTask` (`server.go:741-743`) nối trực tiếp `\n\n### Tài Liệu BRD / PRD Đính Kèm:\n` vào `req.Description`. Nếu tái sử dụng mù quáng `handleCreateTask` để Edit Task, mỗi lần bấm Lưu sẽ bị nhân đôi khối `### Tài Liệu BRD / PRD Đính Kèm:` và luôn ép `Stage` về `READY` (`server.go:860`), làm mất trạng thái `NEEDS_YOU` hoặc `DONE` của Task khi User chỉ muốn sửa thông tin mà không chạy lại từ đầu.
3. **`C-CONTRA` (Bác bỏ phương án nhồi nhét vào `server.go` / `app.js`):**
   - Không được nhồi thêm hàng trăm dòng code xử lý Edit Task vào `internal/webui/server.go` hay `internal/queue/store.go`. Bắt buộc tách thành `internal/queue/store_task_update.go`, `internal/webui/task_edit_handler.go`, và giữ `NOW <= HEAD` trên các file hiện có.
