# 13. Root Cause Analysis — Vì sao Task #5 (`EX-002`) báo `DONE` nhưng không có Code Diff thật và Sub-Tasks không báo `DONE` (`fix_s4_real_llm_diff_and_subtask_done_workflow`)

## I. Tóm tắt điều tra vật lý trên `.orchestrator/tasks/EX-002` và Repo `centralized-export-service`

Khi kiểm tra trực tiếp các artifact vật lý của `EX-002` tại `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/EX-002/` và trạng thái git của `/Users/trainguyen/Documents/work/centralized-export-service`, toàn bộ chuỗi sự kiện diễn ra lúc `15:02:54 -> 15:03:15` (trong `.orchestrator/tasks/EX-002/terminal.log:17-30`) đã lộ rõ 2 nhóm nguyên nhân gốc rễ.

---

## II. Nguyên nhân 1: Vì sao `EX-002` báo `DONE` nhưng KHÔNG có Code Diff thật (và anh không thấy code diff ClickHouse)?

1. **Khâu `S4 Implement` không gọi Model (`AntigravityRunner`) để viết code cho Sub-Task mà tự sinh 4 dòng comment giả `"// Isolated Remediation Hunk"` (`internal/webui/telegram_bridge.go:2918-2932`):**
   - Trong `EX-002`, Plan `S3` đã phân rã đúng 2 Sub-Task cho ClickHouse:
     - `TASK-TRAN01`: `src/infrastructure/database/dbHandler/clickhouse.ts`, `src/models/trans-his/trans-his.model.ts`
     - `TASK-TRAN02`: `src/domain/handlers/trans-his/GetAllTransHisExportHandler.ts`, `src/domain/handlers/trans-his/GetListTransHisByTransIdHandler.ts`
   - Tuy nhiên, khi chạy `S4` (`internal/webui/server.go:1467-1663` -> `synthesizeIsolatedRemediationDiffIfApproved` tại `internal/webui/telegram_bridge.go:2821-2962`), hàm này **hoàn toàn không gọi `AntigravityRunner` (MUSCLE)** để viết code `clickhouse.ts` hay sửa `GetAllTransHisExportHandler.ts` / `GetListTransHisByTransIdHandler.ts`.
   - Thay vào đó, khi thấy các file của `EX-002` chưa có diff trên đĩa, đoạn code tại `internal/webui/telegram_bridge.go:2918-2925` đã tự bịa ra một hunk comment giả 4 dòng:
     ```go
     b.WriteString(fmt.Sprintf("+    // Isolated Remediation Hunk for %s (%s)\n", qt.Key, filepath.Base(rel)))
     b.WriteString(fmt.Sprintf("+    // Synthesized inside .orchestrator/tasks/%s/diff.patch (External repo untouched per L003)\n", qt.Key))
     ```
   - Tiếp theo, hàm `tryGitApplySectionToRepo` (`internal/webui/subtask_git_apply.go:179` & `234`) lại có lệnh chặn:
     ```go
     if strings.Contains(section, "Isolated Remediation Hunk for ") {
         return false
     }
     ```
   - **Hệ quả vật lý:** Trong `.orchestrator/tasks/EX-002/diff.patch:659-697` và `.orchestrator/tasks/EX-002/subtasks/TASK-TRAN02/diff.patch:1-30` chỉ chứa đúng 4 dòng comment giả đó; còn trên đĩa `/Users/trainguyen/Documents/work/centralized-export-service`, file `src/infrastructure/database/dbHandler/clickhouse.ts` **không hề tồn tại**, và 2 file handler `GetAllTransHisExportHandler.ts` / `GetListTransHisByTransIdHandler.ts` **không có bất kỳ dòng code nào được sửa** (`git status --short` không có mặt).

2. **Nhiễm bẩn diff chéo từ `TASK-13` (PostgreSQL) sang `EX-002` (ClickHouse) do cờ `includeAllWhenAnyMatch = true` (`internal/webui/telegram_bridge.go:2813-2815, 2850, 2878`):**
   - Trong `extractMatchingCandidateDiffSections` (`telegram_bridge.go:2813-2815`):
     ```go
     if len(matched) > 0 && includeAllWhenAnyMatch {
         return allSections
     }
     ```
   - Vì `centralized-export-service` đang có sẵn 6 file sửa dở (uncommitted) của `TASK-13` (PostgreSQL: `dbHandler/postgres.ts`, `models/trans-his/trans-his.model.ts`, `defined/app-setting.ts`, `start.ts`, `svc-env.ts`, `test/unit/models/trans-his.model.test.ts`), khi `trans-his.model.ts` trùng tên file ứng viên, cờ `includeAllWhenAnyMatch = true` đã quét sạch cả 6 file PostgreSQL của `TASK-13` nhét vào `.orchestrator/tasks/EX-002/diff.patch:1-657`!

3. **`VerifySubTaskCodeOnDisk` (`internal/webui/task_inspector.go:1423-1436`) chấm `VERIFIED` giả tạo dựa trên hunk comment giả:**
   - Tại `task_inspector.go:1425-1428`:
     ```go
     if strings.Contains(combinedPatch, " b/"+cleanRel) || strings.Contains(combinedPatch, "\t"+cleanRel) {
         verified = append(verified, cleanRel+" (in diff.patch)")
         continue
     }
     ```
   - Vì `diff.patch` có dòng `+++ b/src/domain/handlers/trans-his/GetAllTransHisExportHandler.ts` (thuộc 4 dòng comment giả), `VerifySubTaskCodeOnDisk` lập tức báo `PASS` (`VERIFIED (in diff.patch)`) dù file trên repo chưa hề được sửa!

4. **Giao diện WebUI (`internal/webui/static/app.js:1366-1378`) và tin nhắn `DONE` trên Telegram (`internal/webui/telegram_bridge.go:2276-2284`) không hiển thị `diff.patch` của `S4`:**
   - Trên WebUI (`app.js:1372-1377`), khi bấm sang tab `S4`, ô code bên phải (`#sri-code-demo-content`) vẫn luôn render `s3Report.code_demo` (Code Demo của bước `S3`) chứ không hiển thị nội dung `diff.patch` của bước `S4`.
   - Trên Telegram (`telegram_bridge.go:2276-2284`), hàm `notifyTelegramOnTaskDone` chỉ in đường dẫn chuỗi `.orchestrator/tasks/EX-002/diff.patch` mà không hiển thị danh sách file đã sửa hay trích đoạn diff thực tế.

---

## III. Nguyên nhân 2: Vì sao các Sub-Tasks (`TASK-TRAN01`, `TASK-TRAN02`) KHÔNG chạy Wave và KHÔNG thấy báo `DONE`?

1. **Nút `APPROVE` trên WebUI và `RUN_TASK` trên Telegram gửi `action = "approve_and_run"`, nhưng `server.go:1209` chỉ chạy Sub-Task Wave Engine khi `action == "approve_all"`:**
   - Tại `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1209`:
     ```go
     if len(subTasks) > 0 && action == "approve_all" {
         ...
         waveErr := s.executeAllSubTasksWaveByWave(r.Context(), &targetTask, subTasks)
         ...
         s.sendTelegramWaveExecutionLedger(&targetTask, subTasks)
     }
     ```
   - Trong khi đó, nút bấm chính trên WebUI (`✓ APPROVE Giải Pháp S3 & Chạy S4–S5` tại `static/app.js:131, 601, 1389, 2009, 2221`) và nút `RUN_TASK #5` trên Telegram (`telegram_bridge.go:2200`) đều gửi `action = "approve_and_run"`.
   - Vì `action != "approve_all"`, toàn bộ nhánh `executeAllSubTasksWaveByWave` và `sendTelegramWaveExecutionLedger` bị **bỏ qua 100%**! Bằng chứng trong `.orchestrator/tasks/EX-002/terminal.log:17-30`: từ `Step 8 (Human Gate APPROVED)` lúc `15:02:54`, hệ thống nhảy thẳng sang `Step 10 -> Step 17 (DONE)` lúc `15:03:15` mà không hề chạy bất kỳ Wave nào của `TASK-TRAN01` hay `TASK-TRAN02`, và trên Topic Telegram (`thread_id=245`) không có bất kỳ thông báo `OUTBOUND_WAVE_LEDGER` nào!

2. **Trường `workflow_step` của từng Sub-Task bị đóng băng ở `"Wave 0 · S3 Chờ Duyệt"` và `"Wave 1 · S3 Chờ Duyệt"` (`internal/webui/server.go:1260-1263, 1506-1508, 1847-1850` & `internal/webui/task_inspector.go:1988-2000`):**
   - Ở cuối `server.go:1506-1508` và `1847-1850`, code chỉ gán:
     ```go
     subTasks[i].Stage = "DONE"
     subTasks[i].Status = "done"
     ```
     nhưng **quên không cập nhật `subTasks[i].WorkflowStep`**.
   - Ngoài ra, mỗi khi `populateSubTaskFractalContractsAndArtifacts` (`task_inspector.go:1994-1996`) được gọi, nó chỉ gán `WorkflowStep = "Wave X · S3 Chờ Duyệt"` nếu rỗng chứ không đồng bộ theo `Stage == "DONE"`.
   - **Bằng chứng vật lý trong `.orchestrator/tasks/EX-002/subtasks.json:7, 227` và `/api/dashboard`:**
     - `TASK-TRAN01`: `"stage": "DONE", "status": "done", "workflow_step": "Wave 0 · S3 Chờ Duyệt"`
     - `TASK-TRAN02`: `"stage": "DONE", "status": "done", "workflow_step": "Wave 1 · S3 Chờ Duyệt"`
   - Trên giao diện WebUI (`internal/webui/static/app.js:1446, 1476, 1529`), nhãn badge chính trên từng thẻ Sub-Task (`<span class="sri-wave-badge">${escapeHtml(stStep)}</span>`) và tiêu đề nhóm đều render biến `st.workflow_step`, nên anh nhìn vào WebUI vẫn thấy cả 2 Sub-Task hiện chữ **`Wave 0 · S3 Chờ Duyệt`** và **`Wave 1 · S3 Chờ Duyệt`**!

3. **File `EX-002/09_tasks_solution.md` và thông báo `DONE` trên Telegram không cập nhật trạng thái Sub-Tasks:**
   - Dòng 2 của `.orchestrator/tasks/EX-002/09_tasks_solution.md` vẫn giữ nguyên `> 📌 **Task:** EX-002 · **Stage:** NEEDS_YOU`, bảng Sub-Tasks ở dòng 21-22 không có cột trạng thái `✅ DONE`, và tin nhắn Telegram `notifyTelegramOnTaskDone` (`telegram_bridge.go:2276-2284`) không liệt kê trạng thái `✅ DONE` của từng Sub-Task.
   - Thêm vào đó, `populateSubTaskFractalContractsAndArtifacts` (`task_inspector.go:2078-2079`) còn cắt 25 dòng đầu của file `DbConnectorFactory.ts` có sẵn trên đĩa để ghi đè vào mục Code Demo của `subtasks/TASK-TRAN01/09_tasks_solution.md`.
