# 08. Tasks — Khắc phục lỗi Task `EX-002` báo `DONE` nhưng không có Code Diff thật và Sub-Tasks không chạy Wave / không báo `DONE` (`fix_s4_real_llm_diff_and_subtask_done_workflow`)

## Danh sách Sub-Tasks (Autonomous Full-Stack Prompting — Rule #2)

### 1. `TASK-S4-01`: Hợp nhất `approve_and_run` & `approve_all` để luôn chạy `executeAllSubTasksWaveByWave` và đồng bộ trạng thái `DONE` của từng Sub-Task
- **Mô tả & Context `file:line`:**
  - Tại `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1209`, điều kiện `if len(subTasks) > 0 && action == "approve_all"` khiến hành động mặc định `action == "approve_and_run"` (khi User bấm nút `✓ APPROVE Giải Pháp S3 & Chạy S4–S5` tại `internal/webui/static/app.js:1389` hoặc bấm `RUN_TASK` trên Telegram tại `internal/webui/telegram_bridge.go:2200`) **bỏ qua 100%** hàm `executeAllSubTasksWaveByWave` và `sendTelegramWaveExecutionLedger`.
  - Tại `internal/webui/server.go:1260-1263, 1506-1508, 1847-1850` và `internal/webui/task_inspector.go:1988-2000`, khi Sub-Task chuyển sang `Stage = "DONE"` / `Status = "done"`, trường `st.WorkflowStep` vẫn giữ nguyên giá trị cũ `"Wave 0 · S3 Chờ Duyệt"` và `"Wave 1 · S3 Chờ Duyệt"` (đã xác minh trong `.orchestrator/tasks/EX-002/subtasks.json:7, 227`), làm giao diện WebUI (`app.js:1446, 1476, 1529`) vẫn hiển thị các Sub-Task đang `"S3 Chờ Duyệt"`.
  - Tại `internal/webui/telegram_bridge.go:2266-2295` (`notifyTelegramOnTaskDone`), tin nhắn `OUTBOUND_DONE` không in bảng trạng thái hoàn tất của các Sub-Tasks (`TASK-TRAN01`, `TASK-TRAN02`).
- **Logs / Test:**
  - `go test -v ./internal/webui -run TestApproveAndRunExecutesSubTaskWavesAndUpdatesWorkflowStep`
- **Definition of Done (DoD):**
  - Khi `len(subTasks) > 0` và `action == "approve_and_run" || action == "approve_all"`, hệ thống luôn thực thi `executeAllSubTasksWaveByWave`, cập nhật `st.WorkflowStep = fmt.Sprintf("Wave %d · S6 Hoàn Tất (DONE)", st.WaveIndex)` cho mọi Sub-Task `DONE`, cập nhật bảng Sub-Tasks trong `09_tasks_solution.md`, và gửi báo cáo hoàn tất từng Sub-Task lên WebUI & Topic Telegram.

### 2. `TASK-S4-02`: Xóa bỏ Hunk comment giả `"// Isolated Remediation Hunk"`, chặn nhiễm bẩn diff chéo (`includeAllWhenAnyMatch = false`) và gắn LLM Coder (`invokeMuscleSubTaskCodeGen`) ở `S4 Implement`
- **Mô tả & Context `file:line`:**
  - Tại `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:2918-2932`, khi file mục tiêu của Sub-Task chưa có diff trên đĩa, hàm `synthesizeIsolatedRemediationDiffIfApproved` tự tạo ra 4 dòng comment giả `"// Isolated Remediation Hunk for EX-002 ... External repo untouched per L003"` rồi bị `tryGitApplySectionToRepo` (`internal/webui/subtask_git_apply.go:179, 234`) từ chối apply, dẫn đến file `src/infrastructure/database/dbHandler/clickhouse.ts` không tồn tại trên đĩa và `GetAllTransHisExportHandler.ts` / `GetListTransHisByTransIdHandler.ts` có `0` dòng thay đổi.
  - Tại `internal/webui/telegram_bridge.go:2813-2815, 2850, 2878`, biến `includeAllWhenAnyMatch = isSingleRepoTask = true` làm `EX-002` (ClickHouse) kéo toàn bộ 6 file PostgreSQL chưa commit của `TASK-13` vào `.orchestrator/tasks/EX-002/diff.patch:1-657`.
- **Logs / Test:**
  - `go test -v ./internal/webui -run TestS4MuscleSubTaskCodeGenAndIsolatedDiff`
- **Definition of Done (DoD):**
  - Xóa sạch 100% logic sinh `"// Isolated Remediation Hunk"`.
  - Đặt `includeAllWhenAnyMatch = false` để chỉ thu thập diff của đúng các file thuộc Task/Sub-Task hiện tại.
  - Bổ sung `invokeMuscleSubTaskCodeGen` (gọi `AntigravityRunner` ở `S4` theo từng Sub-Task với schema `{"files":[{"file_path":"...","code":"..."}]}` tránh xung đột `extractStreamJSONText` trên khóa `"content"`) để đọc file hiện tại trên `TargetRepo` + đặc tả Sub-Task (`09_tasks_solution.md`), sinh code thực thi thật ghi vào `TargetRepo`, lấy `git diff` thật lưu vào `subtasks/<SubID>/diff.patch` và `diff.patch` của Task cha (kể cả khi file mục tiêu đã có thay đổi từ Task trước đó, và tái sử dụng `subtasks/<SubID>/diff.patch` khi đã sinh xong).

### 3. `TASK-S4-03`: Siết chặt `VerifySubTaskCodeOnDisk` (chặn false-positive từ comment giả) & hỗ trợ kiểm tra TypeScript/Node.js (`package.json`) tại `S5 QA` + hiển thị `diff.patch` thật trên tab `S4` WebUI
- **Mô tả & Context `file:line`:**
  - Tại `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:1423-1436`, `VerifySubTaskCodeOnDisk` thấy chuỗi `" b/"+cleanRel` trong `diff.patch` (dù là 4 dòng comment giả và file chưa hề tồn tại trên đĩa) vẫn trả về `VERIFIED (in diff.patch)`.
  - Tại `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1788-1815`, `S5 QA` chỉ kiểm tra `go.mod`, bỏ qua các repo Node.js/TypeScript có `package.json`.
  - Tại `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:1366-1378`, khi User bấm vào tab `S4` trong `#stage-report-inspector`, khung `#sri-code-demo-content` vẫn hiển thị `s3Report.code_demo` thay vì `diff.patch` thực tế của `S4`.
- **Logs / Test:**
  - `go test -v ./internal/webui -run TestVerifySubTaskCodeOnDiskRejectsFakeHunk`
- **Definition of Done (DoD):**
  - `VerifySubTaskCodeOnDisk` bác bỏ mọi hunk giả, bắt buộc file tồn tại vật lý trên đĩa (`os.Stat`) kèm diff thực tế.
  - `S5 QA` hỗ trợ kiểm tra cả `go.mod` và `package.json`.
  - Khung `#sri-code-demo-content` trên WebUI hiển thị trực tiếp `diff.patch` thật khi chọn tab `S4` (và tự động mở tab `S4` sau khi `approve_and_run` hoàn tất).
