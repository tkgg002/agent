---
tier: T2
suffix: add_edit_task_and_subtask_crud
---

# 📋 08 Tasks — Bổ sung Tính năng Chỉnh sửa Task (`Edit Task` — `✏️ Sửa Task`) & Sub-Task (`add_edit_task_and_subtask_crud`)

### Task 1 (`TASK-EDIT-01`): Xây dựng tầng lưu trữ `UpdateTaskDetails` & `GetByExternalID` trong `internal/queue/store_task_update.go`
- **Mô tả & Context (`file:line`):**
  - Tham chiếu `internal/queue/store.go:43-60, 155-208`.
  - Tạo file đơn trách nhiệm `internal/queue/store_task_update.go` cung cấp `GetByExternalID(ctx, externalID)` và `UpdateTaskDetails(ctx, externalID, title, description, complexity, assignedModel, stage string, labels []string) (QueuedTask, error)`.
- **Logs / Test:**
  - Unit test `TestStore_UpdateTaskDetails` trong `internal/queue/store_task_update_test.go` kiểm chứng cập nhật đầy đủ các cột trên SQLite thật.
- **Definition of Done (DoD):**
  - `go test -v -race ./internal/queue/...` PASS (`exit=0`), `internal/queue/store.go` giữ nguyên số dòng `<= HEAD`.

### Task 2 (`TASK-EDIT-02`): Xây dựng API `GET /api/tasks/detail` & `POST /api/tasks/update` trong `internal/webui/task_edit_handler.go`
- **Mô tả & Context (`file:line`):**
  - Tham chiếu `internal/webui/server.go:639-663, 689-980` và `internal/webui/task_inspector.go:1543-1690`.
  - Tạo file đơn trách nhiệm `internal/webui/task_edit_handler.go` xử lý:
    1. `GET /api/tasks/detail?task_key=<key>`: Trả về đầy đủ thông tin Task hiện tại (`external_id`, `title`, `clean_description` đã tách phần BRD đính kèm, `brd_content` đọc từ `brd_source.md`, `target_repo`, `affected_repos`, `runner`, `model`, `complexity`, `milestone`, `epic`, `assignee`, `depends_on`, `critical`, `sub_tasks`, `sub_tasks_markdown`).
    2. `POST /api/tasks/update`: Cập nhật Task trong SQLite (`UpdateTaskDetails`), cập nhật `brd_source.md`, `01_requirements.md`, cập nhật `subtasks.json` & `08_tasks.md` đồng thời đồng bộ `llm_analysis.json` (`syncEditedSubTasksToLLMAnalysis` — chống `hasLLMSubTaskMismatch` tại `task_inspector.go:1176-1184` ghi đè lại Sub-Task cũ khi `replan_s1_s3: false`), hoặc tái chạy Phase 1 `S1–S3` nếu `replan_s1_s3: true`, ghi log sự kiện và broadcast SSE `task_updated`.
    3. Hỗ trợ `action: "edit_subtask"` để sửa chi tiết từng Sub-Task và đồng bộ cả `subtasks.json`, `08_tasks.md` lẫn `llm_analysis.json`.
- **Logs / Test:**
  - Integration test `TestTaskEdit_DetailAndUpdateAndSubTaskEdit` trong `internal/webui/task_edit_handler_test.go` (Zero `httptest.NewServer` — gọi trực tiếp trên `Server` + SQLite thật).
- **Definition of Done (DoD):**
  - `go test -v -run TestTaskEdit ./internal/webui/...` PASS (`exit=0`).

### Task 3 (`TASK-EDIT-03`): Tích hợp nút `✏️ Sửa Task` & Chế độ Edit trên `#dispatch-modal` (`internal/webui/static/index.html` & `internal/webui/static/app.js`)
- **Mô tả & Context (`file:line`):**
  - Tham chiếu `internal/webui/static/index.html:68-91, 635-859` và `internal/webui/static/app.js:869-911, 2239-2248, 2744-3153`.
  - Bổ sung nút `✏️ Sửa Task` trên `#cross-screen-context-bar`, nút `✏️` trên từng thẻ Task ở `#sidebar-task-queue` và trên bảng Wave `#wave-tier-tasks`.
  - Nâng cấp `#dispatch-modal` hỗ trợ cả 2 chế độ: **Create Task** (`openDispatchModal()`) và **Edit Task** (`openEditTaskModal(taskKey, ev)`), tự động nạp đầy đủ dữ liệu từ `/api/tasks/detail` và gửi `POST /api/tasks/update` khi lưu.
- **Logs / Test:**
  - Kiểm chứng trực tiếp trên trình duyệt / API `:8090` (`curl -sS http://127.0.0.1:8090/api/tasks/detail?task_key=...` & `POST /api/tasks/update`).
- **Definition of Done (DoD):**
  - Toàn bộ file chỉnh sửa đạt `NOW <= HEAD`, build `./bin/orchestrator` & restart `:8090` thành công, vượt qua `/laya_router` `DG1/DG2/DG3` (`PASS`) và `verify_governance.py` (`PASSED` 🟢).

### Task 4 (`TASK-EDIT-04`): Vòng lặp Phản tỉnh & Tự khắc phục `S5 -> S3 -> S4 -> S5` (Rule #9.8d — 5 Hạng mục Kiểm toán Phản biện)
- **Mô tả & Context (`file:line`):**
  1. **Tách Module Đơn Trách Nhiệm (`internal/webui/task_edit_handler.go` & `internal/webui/subtask_edit_service.go`):** Tách phần Sub-Task CRUD & đồng bộ (`applyCustomSubTasksOnEdit`, `syncEditedSubTasksToLLMAnalysis`, `applySubTaskEditOrRoleUpdate`, `syncUpdatedTaskRequirementsDoc`) sang `internal/webui/subtask_edit_service.go`, giữ `internal/webui/task_edit_handler.go` (`~265` dòng) thuần HTTP Transport.
  2. **Khắc phục lỗi không cập nhật `01_requirements.md` khi `replan_s1_s3 == false` (`internal/webui/task_edit_handler.go:321-324`):** Gọi `s.syncUpdatedTaskRequirementsDoc(updatedTask, isoMeta, subItems)` ngay trong nhánh `replan == false` để `.orchestrator/tasks/<TaskKey>/01_requirements.md` luôn đồng bộ tức thì với `Title`, `Description`, `BRDContent`, `TargetRepo`, `AffectedRepos`, `Model` và `SubTasks` mới.
  3. **Khắc phục lỗi `autoPreviewSubTasksOnTitle()` xóa trắng `#inp-task-subtasks` khi sửa Title ở Edit Mode (`internal/webui/static/app.js:3016-3037`, `internal/webui/static/index.html:828`):** Chặn `fillDefaultSubTasksInModal()` tự động xóa trắng `#inp-task-subtasks` khi `window.editingTaskKey` đang bật (trừ khi User chủ động bấm nút `⚡ Tự chia Sub-Tasks`).
  4. **Xóa bỏ 100% `net/http/httptest` trong `internal/webui/task_edit_handler_test.go` & Trace Runtime thật trên `:8090`:** Dùng TCP socket thật (`net.Listen("tcp", "127.0.0.1:0")` + `&http.Server{}` + `http.Client`) và trace trực tiếp `GET /api/tasks/detail`, `POST /api/tasks/update`, `POST /api/tasks/subtasks (edit_subtask)` lên server `:8090` đang chạy.
  5. **Đưa 100% file được Git theo dõi trong toàn bộ repo `ai-orchestrator` về `NOW <= HEAD` (0 ngoại lệ):** Tách các hàm mở rộng ở `internal/webui/task_llm_bridge.go`, `internal/webui/subtask_wave_engine.go`, `internal/webui/subtask_codegen_muscle.go`, `internal/runner/runner.go` sang `internal/webui/task_wave_llm_hooks.go` để mọi file cũ trong `git diff --name-only HEAD` đều thỏa mãn `NOW <= HEAD`.
- **Logs / Test:**
  - `TestTaskEditDetailUpdateAndSubTaskCRUD` (Real TCP Socket + Real SQLite + Real Disk `01_requirements.md` & `08_tasks.md`) + Live HTTP `POST /api/tasks/update` & `POST /api/tasks/subtasks` trên `:8090`.
- **Definition of Done (DoD):**
  - 100% file trong `git diff --name-only HEAD` đạt `NOW <= HEAD`, 0 import `net/http/httptest` trong `task_edit_handler_test.go`, `01_requirements.md` cập nhật ngay cả khi `replan_s1_s3 == false`, qua `/laya_router` `DG1/DG2/DG3` (Mode A & Mode B) `PASS`.
