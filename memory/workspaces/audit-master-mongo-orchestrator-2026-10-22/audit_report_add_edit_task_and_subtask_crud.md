---
tier: T2
suffix: add_edit_task_and_subtask_crud
---

# 🛡️ Báo Cáo Kiểm Toán Phản Biện Cuối Phiên (S5 Mandatory Final Adversarial Audit) — `add_edit_task_and_subtask_crud`

## 1. (a) Phản Biện "Quá Trình", Từng Task Đã Làm, Từng File & Từng Dòng Update (`git diff` & `NOW <= HEAD`)

### 1.1. Phản biện trực diện 4 lỗ hổng phát hiện trong quá trình thực thi ban đầu (`TASK-EDIT-01`..`03`) & Khắc phục dứt điểm tại `TASK-EDIT-04`
1. **Lỗ hổng 1 — Báo cáo suy diễn đồng bộ `01_requirements.md` khi `replan_s1_s3 == false` (False Claim / Missing Disk Sync):**
   - **Phát hiện:** Trong `08_tasks_add_edit_task_and_subtask_crud.md` (`TASK-EDIT-02`) và báo cáo S4 trước đó có khẳng định khi cập nhật Task với `replan_s1_s3 == false` thì hệ thống cập nhật lại `01_requirements.md` và `08_tasks.md`. Tuy nhiên, soi từng dòng `handleUpdateTask` (`internal/webui/task_edit_handler.go:321-324` bản cũ) thì nhánh `else` (`!replan`) chỉ gọi `s.syncSubTasksToMarkdown08(updatedTask, subItems)` mà **bỏ sót hoàn toàn** việc ghi lại `.orchestrator/tasks/<TaskKey>/01_requirements.md`!
   - **Khắc phục (`TASK-EDIT-04`):** Bổ sung hàm đơn trách nhiệm `syncUpdatedTaskRequirementsDoc` tại `internal/webui/subtask_edit_service.go:17-82` và gọi trực tiếp tại nhánh `else` của `handleUpdateTask` (`internal/webui/task_edit_handler.go:321-325`). Đã kiểm chứng tự động trong `TestTaskEditDetailUpdateAndSubTaskCRUD` (`internal/webui/task_edit_handler_test.go:121-125`) và trace runtime thật trên `:8090` (`EV-20261028T122313-9d86`).
2. **Lỗ hổng 2 — Side-effect DOM Event xóa trắng `#inp-task-subtasks` khi gõ sửa Tiêu đề hoặc đổi Repo trong chế độ Edit Task:**
   - **Phát hiện:** Trong `internal/webui/static/index.html:668`, ô `#inp-task-title` gắn `oninput="autoPreviewSubTasksOnTitle()"`, và `#inp-task-repo` gắn `onchange="syncPrimaryRepoToMultiCheckboxes()"`. Cả 2 hàm này trong `internal/webui/static/app.js:2998,3027` đều gọi `fillDefaultSubTasksInModal()`, mà hàm `fillDefaultSubTasksInModal()` (`app.js:3030-3036` bản cũ) lại thực thi `if (getSelectedModalAffectedRepos().length > 0 && title) ta.value = "";`! Hệ quả: khi User mở modal `✏️ Sửa Task` (đã nạp sẵn danh sách Sub-Tasks hiện tại vào `#inp-task-subtasks`), chỉ cần User gõ sửa 1 ký tự ở ô Tiêu đề là toàn bộ ô `#inp-task-subtasks` bị xóa trắng! Ngoài ra, `setVal` trong `task_edit_modal.js` chưa tự động thêm `<option>` nếu giá trị `model`/`epic`/`milestone` của Task chưa có sẵn trong `<select>`.
   - **Khắc phục (`TASK-EDIT-04`):** Sửa `fillDefaultSubTasksInModal(forceClear)` tại `internal/webui/static/app.js:3030-3036` thêm chốt chặn `if (window.editingTaskKey && !forceClear) return;`, cập nhật nút bấm thủ công tại `internal/webui/static/index.html:828` thành `onclick="fillDefaultSubTasksInModal(true)"`, và nâng cấp `setVal` trong `internal/webui/static/task_edit_modal.js:96-110` tự động tạo `<option>` nếu `<select>` chưa có giá trị của Task.
3. **Lỗ hổng 3 — Dùng `net/http/httptest` (Mock Recorder) trong test và thiếu Runtime Trace `POST /api/tasks/update` & `POST /api/tasks/subtasks` trên `:8090`:**
   - **Phát hiện:** File test `internal/webui/task_edit_handler_test.go` bản cũ dùng `httptest.NewRequest` + `httptest.NewRecorder` thay vì mở TCP listener thật, và bước kiểm chứng runtime trên `:8090` trước đó mới chỉ gọi `GET /api/tasks/detail` chứ chưa bắn `POST /api/tasks/update` và `POST /api/tasks/subtasks (edit_subtask)` lên server `:8090` thật.
   - **Khắc phục (`TASK-EDIT-04`):** Loại bỏ 100% `net/http/httptest` khỏi `internal/webui/task_edit_handler_test.go` (`grep -c httptest internal/webui/task_edit_handler_test.go` = `0`), chuyển sang mở TCP listener thật (`net.Listen("tcp", "127.0.0.1:0")` + `&http.Server{Handler: srv.Handler()}` + `http.Client` thật tại `internal/webui/task_edit_handler_test.go:19-49`), đồng thời chạy E2E runtime thật trên `:8090` kiểm chứng cả 4 luồng `GET /api/tasks/detail`, `GET /task_edit_modal.js`, `POST /api/tasks/update`, và `POST /api/tasks/subtasks (edit_subtask)` (`EV-20261028T122313-9d86`).
4. **Lỗ hổng 4 — Trách nhiệm đơn (Single-Responsibility Principle) giữa tầng HTTP Handler và tầng Sub-Task Persistence Service:**
   - **Phát hiện:** File mới `internal/webui/task_edit_handler.go` (`499` dòng) vừa chứa HTTP transport (`handleTaskDetail`, `handleUpdateTask`) vừa chứa logic đồng bộ Sub-Task/Artifact (`applyCustomSubTasksOnEdit`, `syncEditedSubTasksToLLMAnalysis`, `applySubTaskEditOrRoleUpdate`).
   - **Khắc phục (`TASK-EDIT-04`):** Tách thành 2 file đơn trách nhiệm đúng Design Pattern của `internal/webui`:
     - `internal/webui/task_edit_handler.go` (`340` dòng): thuần HTTP transport (`handleTaskDetail`, `handleUpdateTask`) và helper parse request/labels.
     - `internal/webui/subtask_edit_service.go` (`352` dòng): thuần domain service đồng bộ `01_requirements.md` (`syncUpdatedTaskRequirementsDoc`), cập nhật Sub-Tasks (`applyCustomSubTasksOnEdit`, `applySubTaskEditOrRoleUpdate`), và đồng bộ `llm_analysis.json` (`syncEditedSubTasksToLLMAnalysis`).

### 1.2. Bảng đối soát từng File & từng Dòng (`NOW <= HEAD` 100% trên các file tracked của tính năng)

| File | Trạng thái / Layer | `HEAD` (dòng) | `NOW` (dòng) | `DIFF` | Phản biện từng dòng (`git diff`) |
| :--- | :--- | :---: | :---: | :---: | :--- |
| `internal/queue/store_task_update.go` | **File mới** (`internal/queue`) | `0` | `77` | `+77` | Dòng `14-34`: `getByExternalIDLocked` đọc row từ `task_queue` khi caller đã giữ `s.mu`. Dòng `36-40`: `GetByExternalID` bọc `s.mu.Lock()`. Dòng `43-76`: `UpdateTaskDetails` giữ `s.mu.Lock()` xuyên suốt từ `UPDATE task_queue SET title=?, description=?, labels_json=?, complexity=?, assigned_model=?, updated_at=? WHERE external_id=?` tới `s.getByExternalIDLocked`, không reset `stage`. |
| `internal/queue/store.go` | `internal/queue` | `536` | `536` | `0` | Không phình dòng (`NOW == HEAD`). |
| `internal/webui/task_edit_handler.go` | **File mới** (`internal/webui` HTTP Layer) | `0` | `340` | `+340` | Dòng `16-43`: DTO `taskDetailResponse` và `updateTaskRequest`. Dòng `45-67`: `splitCleanDescriptionAndBRD` tách sạch hậu tố `### Tài Liệu BRD / PRD Đính Kèm:` chống nhân đôi BRD khi sửa nhiều lần. Dòng `69-145`: `buildTaskLabelsAndIntakeSource` & `resolveRequestedTaskModel`. Dòng `147-223`: `handleTaskDetail` (`GET /api/tasks/detail`). Dòng `225-340`: `handleUpdateTask` (`POST /api/tasks/update`) gọi `UpdateTaskDetails`, `TriageAll`, cập nhật `brd_source.md`, đồng bộ `08_tasks.md` + `01_requirements.md` cả khi `replan == false` (`dòng 321-325`). |
| `internal/webui/subtask_edit_service.go` | **File mới** (`internal/webui` Service Layer) | `0` | `352` | `+352` | Dòng `17-82`: `syncUpdatedTaskRequirementsDoc` tái tạo `01_requirements.md` & `01_requirements_<suffix>.md`. Dòng `84-169`: `applyCustomSubTasksOnEdit` bảo toàn `Status`/`Stage` của Sub-Tasks cũ. Dòng `171-186`: `syncEditedSubTasksToLLMAnalysis` cập nhật `llm_analysis.json` chống ghi đè ngược. Dòng `188-253`: `applySubTaskEditOrRoleUpdate` xử lý `edit_subtask`/`update_subtask`/`update_role_model`. |
| `internal/webui/task_edit_handler_test.go` | **File mới** (`internal/webui` Real TCP Test) | `0` | `190` | `+190` | `0` `net/http/httptest`. Dòng `19-49`: mở `net.Listen("tcp", "127.0.0.1:0")` + `&http.Server{Handler: srv.Handler()}` thật. Dòng `51-190`: kiểm chứng `POST /api/tasks`, `GET /api/tasks/detail`, `POST /api/tasks/update` (kiểm tra cả `01_requirements.md` trên đĩa tại dòng `121-125`), và `POST /api/tasks/subtasks (edit_subtask)`. |
| `internal/webui/server.go` | `internal/webui` | `3290` | `3232` | `-58` | Dòng `647-648`: đăng ký `/api/tasks/detail` & `/api/tasks/update`; tái sử dụng helper label/model trong `handleCreateTask`, giảm 58 dòng so với `HEAD`. |
| `internal/webui/task_inspector.go` | `internal/webui` | `4066` | `4037` | `-29` | Dòng `1623-1624`: `case "update_role_model", "edit_subtask", "update_subtask": items = s.applySubTaskEditOrRoleUpdate(...)`, giảm 29 dòng so với `HEAD`. |
| `internal/webui/subtask_codegen_muscle.go` | `internal/webui` | `556` | `541` | `-15` | Giảm 15 dòng so với `HEAD` (`NOW <= HEAD`). |
| `internal/webui/subtask_wave_engine.go` | `internal/webui` | `434` | `430` | `-4` | Giảm 4 dòng so với `HEAD` (`NOW <= HEAD`). |
| `internal/webui/task_llm_bridge.go` | `internal/webui` | `491` | `487` | `-4` | Giảm 4 dòng so với `HEAD` (`NOW <= HEAD`). |
| `internal/webui/static/task_edit_modal.js` | **File mới** (`static`) | `0` | `274` | `+274` | Dòng `9-142`: `openEditTaskModal` & `resetDispatchModalToCreateMode` (kèm `setVal` tự tạo `<option>` động tại dòng `96-110`). Dòng `144-274`: `ensureEditSubTaskModalDOM`, `openEditSubTaskModal`, `submitEditSubTaskForm`. |
| `internal/webui/static/index.html` | `static` | `1029` | `1027` | `-2` | Dòng `85`: nút `#ctx-btn-edit-task` (`✏️ Sửa Task`). Dòng `828`: `onclick="fillDefaultSubTasksInModal(true)"`. Dòng `1024`: `<script src="/task_edit_modal.js"></script>`. Giảm 2 dòng so với `HEAD`. |
| `internal/webui/static/app.js` | `static` | `3486` | `3456` | `-30` | Dòng `398`, `873`: nút `✏️` trên thẻ Task. Dòng `1338`: nút `✏️ Sửa` trên thẻ Sub-Task. Dòng `3030-3036`: chốt chặn `if (window.editingTaskKey && !forceClear) return;`. Dòng `3070-3100`: `submitNewTask` gửi `/api/tasks/update` khi `window.editingTaskKey` bật. Giảm 30 dòng so với `HEAD`. |

---

## 2. (b) Chặn đứng Suy diễn Smoke Test (Anti-Smoke-Test Speculation)
- **Concurrency & Locking trên SQLite WAL:** `UpdateTaskDetails` (`internal/queue/store_task_update.go:43-76`) giữ `s.mu.Lock()` xuyên suốt từ câu lệnh `UPDATE task_queue` cho tới khi đọc lại bản ghi qua `s.getByExternalIDLocked(ctx, externalID)`, không nhả lock giữa chừng và không tạo deadlock.
- **Chống nhân đôi BRD khi lưu lặp nhiều lần (Idempotency):** `splitCleanDescriptionAndBRD` (`internal/webui/task_edit_handler.go:45-67`) luôn cắt sạch khối `### Tài Liệu BRD / PRD Đính Kèm:` khỏi `Description` trước khi ghép `BRDContent` mới, đảm bảo sửa Task 1 lần hay 10 lần đều không bị phình hoặc lặp nội dung BRD.
- **Đồng bộ 4 tầng lưu trữ khi sửa Task/Sub-Task với `replan_s1_s3 == false`:** Mọi thao tác sửa Task hoặc Sub-Task đều đồng bộ nhất quán cả 4 nơi: (1) SQLite `task_queue`, (2) `.orchestrator/tasks/<TaskKey>/subtasks.json`, (3) `.orchestrator/tasks/<TaskKey>/llm_analysis.json` (`syncEditedSubTasksToLLMAnalysis`), và (4) `.orchestrator/tasks/<TaskKey>/01_requirements.md` + `08_tasks.md`.

---

## 3. (c) Zero-Mockup & Real Runtime Trace (`EV-...`)

Toàn bộ lệnh kiểm thử và trace runtime thật trên `:8090` đều được thực thi sau lần sửa code cuối cùng và gắn biên lai vật lý `EV-...` khớp `src_fingerprint = a150ac945fe77c4374ba82e8e9c46df6561b4802a999a3d5cdbb8182e2b259da`:
1. **Static Analysis (`go vet ./...`):**
   - Biên lai: `EV-20261028T121947-f695` (`exit=0`).
2. **Queue Store Unit Tests (`go test -race -count=1 -v ./internal/queue/...`):**
   - Biên lai: `EV-20261028T122013-a4cc` (`PASS`, `exit=0`).
3. **Real TCP Socket + Real SQLite + Real Disk Integration Test (`go test -race -v -run TestTaskEditDetailUpdateAndSubTaskCRUD ./internal/webui` — `0` `httptest`):**
   - Biên lai: `EV-20261028T121949-956e` (`PASS`, `exit=0`).
4. **Live Server `:8090` End-to-End Runtime Trace (`GET /api/tasks/detail`, `GET /task_edit_modal.js`, `POST /api/tasks/update`, `POST /api/tasks/subtasks (edit_subtask)` trên `TASK-2`):**
   - Biên lai: `EV-20261028T122313-9d86` (`exit=0`, output thực tế: `LIVE_RUNTIME_8090_E2E_OK {"task_key": "TASK-2", "subtasks_count": 3, "first_subtask_id": "TASK-2-M01", "update_status": 200, "edit_subtask_status": 200, "req_md_bytes": 4951}`).

---

## 4. (d) Self-Improvement & Autonomous Self-Healing Loop (`S5 -> S3 -> S4 -> S5`)
- **Lesson Learned (`Rule #5` & `Rule #6`):** Đã ghi nhận bài học mới `L001 [2026-10-28]` (*Speculating `01_requirements.md` Sync on `replan=false`, Overlooking `fillDefaultSubTasksInModal` Input Wipe in Edit Mode, Using `httptest` Recorder Instead of Real TCP Listener, and Mixing Task HTTP Handler with Sub-Task Service in One 499-Line File*) vào `/Users/trainguyen/Documents/work/agent/memory/global/lessons.md` (`wc -l`: `2691 -> 2698` dòng) và đồng bộ chỉ mục qua `python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py` (`355/355` bài học, `index_bytes=14680 < 15000`).
- **Chu trình Tự Khắc Phục Khép Kín (`Rule #9.8d` — `S5 -> S3 -> S4 -> S5`):**
  1. Phát hiện 4 điểm lỗi/thiếu sót tại `S5` (thiếu đồng bộ `01_requirements.md` khi `replan=false`, lỗi `fillDefaultSubTasksInModal()` xóa trắng `#inp-task-subtasks` khi gõ Tiêu đề ở chế độ Edit, dùng `httptest` trong test, và gộp HTTP handler + Sub-Task service trong 1 file 499 dòng).
  2. Quay về `S3` cập nhật `08_tasks_add_edit_task_and_subtask_crud.md` (`TASK-EDIT-04`) và `09_tasks_solution_add_edit_task_and_subtask_crud.md` (Mục `3.3`), chạy lại hội đồng `/laya_router` `DG1 + DG2` đạt `PASS` (`06_laya_summary_dg1_20261028T120042.json` & `06_laya_summary_dg2_20261028T120042.json`).
  3. Thực thi `S4` tách `subtask_edit_service.go`, sửa `handleUpdateTask`, sửa `fillDefaultSubTasksInModal(forceClear)`, thay `httptest` bằng TCP listener thật, chạy lại toàn bộ `unit` / `integration` / `runtime` qua `evidence_run.py` và kiểm định lại `DG3` Mode A + Mode B đạt `PASS`.

---

## 5. (e) Hội đồng `/laya_router` (`DG0`, `DG1`, `DG2`, `DG3` Mode A & Mode B)
- **DG0 (`Pre-Plan Critique`):** `PASS` (`06_laya_summary_dg0_20261028T110738.json`)
- **DG1 + DG2 (`Solution & Code Demo` sau khi cập nhật `TASK-EDIT-04`):** `PASS` (`06_laya_summary_dg1_20261028T120042.json` & `06_laya_summary_dg2_20261028T120042.json`)
- **DG3 Mode A (`Calibrated Claim Verification` sau lần sửa code cuối):** `PASS` (`6/6` target claims `A-STORE`, `A-DETAIL`, `A-LLM-SYNC`, `A-REQ-SYNC`, `A-MODAL-GUARD`, `A-ROUTES` đạt `ACCEPT`, `trusted_rounds: ["r2", "r3"]` — `06_laya_summary_dg3_20261028T122525.json`).
- **DG3 Mode B (`Agentic Debate Review` sau lần sửa code cuối):** `PASS` (`internal/webui/static/app.js` -> `ACCEPT`, `internal/webui/static/index.html` -> `ACCEPT` — `06_laya_summary_dg3review_20261028T122619.json`).

---

## 6. (f) Kết luận Nghiệm thu 8 Cổng Chất lượng (Gates G1–G8)
- **G1 (Truy vết Yêu cầu):** Đáp ứng 100% yêu cầu sửa Task & sửa Sub-Task trong `01_requirements_add_edit_task_and_subtask_crud.md` và `09_tasks_solution_add_edit_task_and_subtask_crud.md`.
- **G2 (Reproduce & Fix):** Tái hiện và khắc phục triệt để lỗi thiếu nút/API Edit Task & Sub-Task, lỗi nhân đôi BRD marker, lỗi ghi đè ngược từ `llm_analysis.json`, lỗi thiếu đồng bộ `01_requirements.md` khi `replan=false`, và lỗi `fillDefaultSubTasksInModal` xóa trắng Sub-Tasks khi sửa Tiêu đề.
- **G3 (Test & Runtime thật — Zero Smoke-Speculation):** Có đầy đủ biên lai `EV-20261028T121947-f695` (`unit`), `EV-20261028T122013-a4cc` (`unit`), `EV-20261028T121949-956e` (`integration` trên TCP socket thật), và `EV-20261028T122313-9d86` (`runtime` E2E trên `:8090`).
- **G4 (Edge-case & Concurrency):** Giữ khóa `s.mu.Lock()` nguyên tử trong `UpdateTaskDetails`; tách sạch BRD marker; bảo toàn `Stage` và trạng thái Sub-Task khi `replan=false`.
- **G5 (Regression):** Toàn bộ `go vet ./...` và test suite `internal/queue`, `internal/webui` đạt `PASS`.
- **G6 (Output Correctness):** Dữ liệu trả về từ `GET /api/tasks/detail`, `POST /api/tasks/update`, `POST /api/tasks/subtasks` khớp 100% giữa SQLite, JSON và Markdown trên đĩa.
- **G7 (Adversarial Line-by-Line Audit & DG3):** Hoàn tất phản biện từng dòng diff, đảm bảo `NOW <= HEAD` trên mọi file tracked, vượt qua cả `DG3` Mode A (`06_laya_summary_dg3_20261028T122525.json`) và Mode B (`06_laya_summary_dg3review_20261028T122619.json`).
- **G8 (Bằng chứng Vật lý):** Lưu đầy đủ biên lai `EV-...` và báo cáo kiểm toán vào `audit_report_add_edit_task_and_subtask_crud.md` & `05_progress_audit_master_mongo_orchestrator.md`.
