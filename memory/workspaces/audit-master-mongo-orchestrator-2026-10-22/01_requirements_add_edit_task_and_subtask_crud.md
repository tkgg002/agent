---
tier: T2
suffix: add_edit_task_and_subtask_crud
mode: standard
---

# 📋 01 Requirements — Bổ sung Tính năng Chỉnh sửa Task (`Edit Task` — `✏️ Sửa Task`) & Sub-Task trên WebUI và Backend (`ai-orchestrator`)

## 1. Bối cảnh & Yêu cầu của User
- **User phản ánh trực tiếp:** `"chưa có edit task"`
- **Hiện trạng thực tế trong `/Users/trainguyen/Documents/work/ai-orchestrator` (`VERIFIED`):**
  1. Trên Thanh ngữ cảnh (`#cross-screen-context-bar` tại `internal/webui/static/index.html:68-91`), Hàng đợi Sidebar (`renderSidebarTaskQueue` tại `internal/webui/static/app.js:869-911`) và Bảng Wave (`renderWaveBoard` tại `internal/webui/static/app.js:2239-2248`), hiện chỉ có các nút `▶ Chạy`, `⏹ Dừng` và `🗑 Xóa Task`, **hoàn toàn chưa có nút `✏️ Sửa Task` (`Edit Task`)**.
  2. Modal `#dispatch-modal` (`internal/webui/static/index.html:635-859` & `internal/webui/static/app.js:2744-3153`) hiện chỉ có chế độ Tạo mới (`openDispatchModal` / `submitNewTask`), chưa có chế độ nạp dữ liệu của một Task đã tồn tại (`openEditTaskModal`) để chỉnh sửa `Title`, `Description`, `BRDContent`, `TargetRepo`, `AffectedRepos`, `Runner`, `Model`, `Complexity`, `Milestone`, `Epic`, `Assignee`, `DependsOn`, `Critical` và danh sách `Sub-Tasks`.
  3. Tầng Backend (`internal/webui/server.go:640-663`, `internal/queue/store.go:155-208`) chưa có API `GET /api/tasks/detail` và `POST /api/tasks/update` cũng như phương thức `Store.UpdateTaskDetails` để cập nhật đồng bộ SQLite `task_queue` (`title`, `description`, `labels_json`, `complexity`, `assigned_model`) và hồ sơ `.orchestrator/tasks/<TaskKey>/` (`brd_source.md`, `01_requirements.md`, `08_tasks.md`, `09_tasks_solution.md`, `subtasks.json`).

## 2. Ràng buộc Kỹ thuật từ Bài học (`/agent` Lessons-as-Constraints — `L001`, `L005`, `L007`, `L015`, `L022`)
1. **Single-Responsibility & Zero Bloat (`NOW <= HEAD` trên các file lớn hiện có):**
   - Bắt buộc tách module đơn trách nhiệm mới:
     - `internal/queue/store_task_update.go`: Chứa `UpdateTaskDetails` và `GetByExternalID` cho tầng SQLite `queue.Store`.
     - `internal/webui/task_edit_handler.go`: Chứa `handleTaskDetail` (`GET /api/tasks/detail`) và `handleUpdateTask` (`POST /api/tasks/update` + hỗ trợ `edit_subtask`).
   - Không nhồi nhét logic xử lý Edit Task vào `internal/webui/server.go` hay `internal/webui/task_inspector.go`.
2. **Bảo toàn 100% dữ liệu BRD & Không nhân đôi chuỗi `### Tài Liệu BRD / PRD Đính Kèm:`:**
   - Khi mở Modal `Edit Task`, phải tách riêng phần `clean_description` và phần `brd_content` (đọc từ `.orchestrator/tasks/<TaskKey>/brd_source.md` hoặc bóc tách phần sau `### Tài Liệu BRD / PRD Đính Kèm:`) để khi User bấm Lưu nhiều lần không bao giờ bị lặp chuỗi BRD trong `Description`.
3. **Đồng bộ AI Model thật khi tái sinh Plan (`L005` / `L016`):**
   - Khi User chỉnh sửa Task và chọn tùy chọn tái sinh Phase 1 (`S1–S3`), backend bắt buộc xóa cache `llm_analysis.json` cũ và gọi `generateDeepTaskArtifacts` (`invokeArchitectLLM`) để cập nhật lại `01_requirements.md`, `08_tasks.md`, `09_tasks_solution.md`, `subtasks.json`.

## 3. Tiêu chí Nghiệm thu (Acceptance Criteria — Gates G1..G8)
- **REQ-EDIT-01 (UI Entry Points & Dual-Mode Modal):** Có nút `✏️ Sửa Task` trên `#cross-screen-context-bar`, nút `✏️` trên từng thẻ Task ở Sidebar `#sidebar-task-queue` và trên bảng Wave `#wave-tier-tasks`. Bấm vào mở `#dispatch-modal` ở chế độ `Edit Task` nạp sẵn đầy đủ 100% thông tin hiện tại của Task.
- **REQ-EDIT-02 (Backend SQLite & Artifact Sync):** Endpoint `GET /api/tasks/detail` và `POST /api/tasks/update` (cùng `Store.UpdateTaskDetails`) cập nhật chính xác SQLite `task_queue`, `brd_source.md`, `01_requirements.md`, `subtasks.json` / `08_tasks.md`, và hỗ trợ tái chạy Phase 1 `S1–S3` khi được chọn.
- **REQ-EDIT-03 (Inline Sub-Task Edit):** Hỗ trợ `action: "edit_subtask"` trong `/api/tasks/subtasks` để sửa trực tiếp thông tin Sub-Task (`title`, `target_repo`, `context_file_line`, `description`, `test_dod`, `wave_index`, `assignee`, `model`).
