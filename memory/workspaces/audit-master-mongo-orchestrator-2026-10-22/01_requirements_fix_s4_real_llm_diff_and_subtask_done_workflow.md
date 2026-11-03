---
tier: T2
suffix: fix_s4_real_llm_diff_and_subtask_done_workflow
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-27T15:27:10+07:00
---

# 01. Requirements — Khắc phục lỗi Task `EX-002` báo `DONE` nhưng không có Code Diff thật và Sub-Tasks không chạy Wave / không báo `DONE` (`ai-orchestrator`)

## 1. Yêu cầu gốc từ User
> `"tao chạy task 5 ex-002. vì sao báo done nhưng tao ko thấy code diff, và sub task ko thấy báo done. mày lại phá gì nữa. có cái quy gtrinh là fix hoài vậy, mày chỉ vận hành theo workflow tao đưa ra thôi mà, khó vậy sao"`

## 2. Phạm vi & Ràng buộc kiến trúc (Scope & Constraints)
- **Phạm vi tuyệt đối (Lesson `L009`):** Chỉ làm việc trên `/Users/trainguyen/Documents/work/ai-orchestrator` và `/Users/trainguyen/Documents/work/agent`.
- **Agnostic Core (`ADR-07` & Lesson `L001`):** Core `ai-orchestrator` phải vận hành tổng quát 100% theo Master Workflow (`S0 -> S1 -> S2 -> S2.5 DG0 -> S3 + DG1/DG2 -> [APPROVE] -> S4 Sub-Task Wave Engine -> S5 QA/Audit + DG3 -> S6 DONE`), tuyệt đối không hardcode tên task, từ khóa công nghệ hay sinh hunk comment giả `"// Isolated Remediation Hunk"`.
- **Kỷ luật Propose-Only (Rule #9 & Rule #13):** Lập đầy đủ hồ sơ `S1 -> S2 -> S2.5 (DG0) -> S3 -> DG1/DG2` (`PASS`), trả lời thẳng câu hỏi của User với khối **Direct Summary** ở đầu (Lesson `L008`), trình bày nguyên nhân gốc rễ (`file:line`) + 1 phương án tối ưu duy nhất kèm Code Demo và dừng chờ `APPROVE` trước khi sửa source code `.go`/`.js`.

## 3. Các yêu cầu chức năng cần giải quyết (`REQ-S4-01` .. `REQ-S4-04`)
- **`REQ-S4-01` (Thực thi đầy đủ Sub-Task Wave Engine khi bấm `APPROVE` / `RUN_TASK`):**
  - Khi User bấm `✓ APPROVE Giải Pháp S3 & Chạy S4–S5` trên WebUI (`action = "approve_and_run"`) hoặc bấm `RUN_TASK` trên Telegram (`action = "approve_and_run"`), nếu Task có `len(subTasks) > 0`, hệ thống **BẮT BUỘC** phải chạy `executeAllSubTasksWaveByWave` tuần tự từng Sub-Task qua đủ 6 Wave (`S1 -> S2 -> S3 -> S4 -> S5 -> S6`) thay vì chỉ chạy khi `action == "approve_all"` (`internal/webui/server.go:1209`).
- **`REQ-S4-02` (Sinh Code thật bằng LLM Coder ở `S4` & Chặn đứng Hunk giả / Nhiễm bẩn Diff chéo giữa các Task):**
  - Xóa bỏ hoàn toàn đoạn sinh 4 dòng comment giả `"// Isolated Remediation Hunk for ... External repo untouched per L003"` tại `internal/webui/telegram_bridge.go:2918-2932`.
  - Tắt `includeAllWhenAnyMatch` (`false`) tại `internal/webui/telegram_bridge.go:2850, 2878` để một Task (như `EX-002` ClickHouse) chỉ thu thập diff của đúng các file ứng viên thuộc Task/Sub-Task đó, tuyệt đối không gom nhầm các file uncommitted của Task khác (như 6 file PostgreSQL của `TASK-13`).
  - Tại `S4 Implement` (`executeAllSubTasksWaveByWave` / `synthesizeIsolatedRemediationDiffIfApproved`), gắn trực tiếp LLM Coder (`invokeMuscleSubTaskCodeGen` qua `AntigravityRunner`) để đọc nội dung file hiện tại trên `TargetRepo` + đặc tả Sub-Task (`09_tasks_solution.md`), sinh code thực thi thật ghi xuống `TargetRepo`, thu thập `git diff` thật (bao gồm cả file mới tạo như `src/infrastructure/database/dbHandler/clickhouse.ts`) vào `subtasks/<SubID>/diff.patch` và `diff.patch` của Task cha.
- **`REQ-S4-03` (Cổng kiểm tra `VerifySubTaskCodeOnDisk` & `5A Hard Gate` trung thực trên mọi Tech Stack):**
  - `VerifySubTaskCodeOnDisk` (`internal/webui/task_inspector.go:1420-1473`) phải bác bỏ mọi hunk comment giả, yêu cầu file mục tiêu tồn tại vật lý trên đĩa (`os.Stat`) VÀ có thay đổi thật trong `git status --porcelain` / `diff.patch`.
  - `S5 QA` (`internal/webui/server.go:1788-1815` & `internal/webui/subtask_wave_engine.go:280-296`) phải chạy lệnh build/typecheck thật cho cả repo Node.js/TypeScript (`package.json`) lẫn Go (`go.mod`).
- **`REQ-S4-04` (Đồng bộ trạng thái `DONE` của từng Sub-Task và hiển thị Code Diff `S4` trên WebUI & Telegram):**
  - Cập nhật `st.WorkflowStep = fmt.Sprintf("Wave %d · S6 Hoàn Tất (DONE)", st.WaveIndex)` ngay khi Sub-Task hoàn tất (`server.go:1260, 1506, 1847`, `subtask_wave_engine.go:392`, `task_inspector.go:1994`) để thẻ Sub-Task trên WebUI không còn bị treo ở `"Wave 0 · S3 Chờ Duyệt"` / `"Wave 1 · S3 Chờ Duyệt"`.
  - Cập nhật trạng thái `✅ DONE` cho từng Sub-Task trong bảng Sub-Tasks của `09_tasks_solution.md` và gửi thông báo hoàn tất từng Sub-Task (`sendTelegramWaveExecutionLedger` + danh sách Sub-Tasks `DONE` kèm danh sách file diff thật trong `notifyTelegramOnTaskDone`) lên Topic Telegram.
  - Trên WebUI (`internal/webui/static/app.js:1366-1378`), khi mở tab `S4` (hoặc sau khi `approve_and_run` hoàn tất), khung code bên phải (`#sri-code-demo-content`) phải hiển thị `diff.patch` thật của `S4` thay vì vẫn hiển thị `s3Report.code_demo`.
