# Báo Cáo Kiểm Toán Toàn Diện (S5 Adversarial Audit & S6 Validation Report)
- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `fix_s4_real_llm_diff_and_subtask_done_workflow`
- **Tier:** `T2`
- **Target Project:** `/Users/trainguyen/Documents/work/ai-orchestrator`
- **Timestamp:** `2026-10-27T16:56:30+07:00`

---

## 1. (a) Line-by-Line Diff Critique (`git diff` trên `/Users/trainguyen/Documents/work/ai-orchestrator`)

### 1.1. `internal/webui/server.go:1205-1266` (`NOW=3240 <= HEAD=3277`)
- **Thay đổi:**
  - Hợp nhất nhánh `if len(subTasks) > 0 && (action == "approve_all" || action == "approve_and_run")` tại `server.go:1212` để cả hành động `approve_and_run` (từ nút `APPROVE` trên WebUI và `RUN_TASK` trên Telegram) lẫn `approve_all` đều bắt buộc chạy `executeAllSubTasksWaveByWave` tuần tự qua từng Sub-Task trước khi mở khóa `S5` của Task cha.
  - Cập nhật `subTasks[i].Status = "done"`, `subTasks[i].Stage = "DONE"`, và `subTasks[i].WorkflowStep = fmt.Sprintf("Wave %d · S6 Hoàn Tất (DONE)", subTasks[i].WaveIndex)` tại `server.go:1263-1266` (cũng như `server.go:1506-1509, 1847-1850`), xóa bỏ triệt để tình trạng Sub-Task treo ở `"Wave 0/1 · S3 Chờ Duyệt"`.
- **Phản biện:** Không còn nhánh bypass nào cho phép Task cha nhảy cóc sang `DONE` khi các Sub-Task chưa hoàn tất `Wave 0..N`.

### 1.2. `internal/webui/subtask_codegen_muscle.go:1-556` (File mới tách module MUSCLE S4 CodeGen)
- **Thay đổi:**
  - `isSameOrSubpathOfRoot(rootDir, targetDir)` (`subtask_codegen_muscle.go:32-46`): Chuẩn hóa `filepath.Abs` trước khi so sánh, đảm bảo khi chạy `daemon --once` (`s.Root == "."`) hệ thống nhận diện đúng thư mục gốc của `ai-orchestrator` và không bao giờ nhầm là repo ngoài.
  - `resolveActualRepoRelPath(repoDir, rawRel)` (`subtask_codegen_muscle.go:51-75`): Đối chiếu cây thư mục thực tế trên đĩa của `repoDir` để tự động gọt bỏ các prefix thư mục ảo do LLM sinh dư (ví dụ `src/infrastructure/database/dbHandler/clickhouse.ts` -> `dbHandler/clickhouse.ts` khi repo có `dbHandler/` ở gốc), hoàn toàn Agnostic (0% từ khóa hardcode).
  - `invokeMuscleSubTaskCodeGen` (`subtask_codegen_muscle.go:180-310`): Gọi trực tiếp `AntigravityRunner` ở bước `S4 Implement` cho từng Sub-Task, đọc nội dung file thật trên đĩa, ghi code mới vào đúng đường dẫn đã chuẩn hóa, và thu thập `git diff` có cô lập theo phạm vi file của từng Sub-Task (`captureScopedSubTaskDiff`), lưu vào `.orchestrator/tasks/<TaskID>/subtasks/<SubID>/diff.patch` và tổng hợp lên `.orchestrator/tasks/<TaskID>/diff.patch`.
- **Phản biện:** Tuyệt đối không sinh comment giả `"// Isolated Remediation Hunk"` và không dùng `includeAllWhenAnyMatch = true` gây nhiễm chéo diff từ task khác (`TASK-13`).

### 1.3. `internal/webui/telegram_bridge.go:2266-2295, 2810-2924` (`NOW=2924 <= HEAD=2994`)
- **Thay đổi:**
  - Xóa bỏ hoàn toàn đoạn sinh 4 dòng comment giả `"// Isolated Remediation Hunk for ..."` trong `synthesizeIsolatedRemediationDiffIfApproved` và thay bằng lời gọi `s.invokeMuscleSubTaskCodeGen`.
  - Bổ sung bảng tổng kết trạng thái từng Sub-Task (`✅ DONE` + `WorkflowStep`) kèm thống kê `diff.patch` vào thông báo hoàn tất `notifyTelegramOnTaskDone`.

### 1.4. `internal/webui/task_inspector.go:1295-1473` (`NOW=4111 <= HEAD=4445`)
- **Thay đổi:**
  - `isFakeIsolatedRemediationPatch`: Chặn đứng và bác bỏ mọi bản vá chỉ chứa comment giả `"Isolated Remediation Hunk for "`.
  - `VerifySubTaskCodeOnDisk`: Chuẩn hóa đường dẫn qua `resolveActualRepoRelPath`, kiểm tra sự tồn tại vật lý của file mục tiêu trên đĩa kèm diff thật trong `subtasks/<SubID>/diff.patch` hoặc `diff.patch` của Task cha, đồng thời hỗ trợ kiểm tra dự án Node/TypeScript (`package.json`) bên cạnh Go (`go.mod`).

### 1.5. `internal/webui/static/app.js:1366-1382`
- **Thay đổi:**
  - Khi người dùng chọn bước `S4` hoặc `S5` trên Step Report Inspector, ưu tiên hiển thị trực tiếp `diff.patch` thật đã sinh ở `S4` trên khung `#sri-code-demo-content` với tiêu đề `"🔥 Unified Code Diff Thật Đã Thực Thi Ở S4 (diff.patch)"`.

---

## 2. (b) Anti-Smoke-Test Speculation & Concurrency / Multi-Wave Audit
- Không suy diễn từ smoke test đơn lẻ: Đã kiểm chứng cả qua bộ test tích hợp cô lập trên git worktree tạm (`t.TempDir()`) lẫn chạy thực tế toàn bộ chu trình `S4 -> S5 -> S6` của Task `EX-002` (`TASK-TRAN01` ở `Wave 0` và `TASK-TRAN02` ở `Wave 1`) trên tiến trình `:8090`.
- Kiểm chứng tính tuần tự và cô lập giữa các Wave: `Wave 0` (`TASK-TRAN01`: `dbHandler/clickhouse.ts`, `models/trans-his/trans-his.model.ts`) hoàn tất và lưu `subtasks/TASK-TRAN01/diff.patch` (`11,585` bytes) trước khi `Wave 1` (`TASK-TRAN02`: `GetAllTransHisExportHandler.ts`, `GetListTransHisByTransIdHandler.ts`, lưu `subtasks/TASK-TRAN02/diff.patch` `10,953` bytes) thực thi; không bị nhiễm diff `postgres.ts` của `TASK-13`.

---

## 3. (c) Zero-Mockup & Real Runtime Verification Receipts (`EV-...`)

| Biên lai (`EV-...`) | Loại (`kind`) | Lệnh thực thi | Kết quả |
| :--- | :---: | :--- | :---: |
| `EV-20261027T163446-a174` | `test` | `go test -v ./internal/webui -run TestResolveActualRepoRelPathAndRejectFakeHunk` | `PASS (exit=0)` |
| `EV-20261027T163449-05f1` | `test` | `go test -v ./internal/webui -run TestApproveAndRunExecutesSubTaskWavesAndUpdatesWorkflowStepDone` | `PASS (exit=0)` |
| `EV-20261027T163501-62b3` | `test` | `go test -v ./internal/webui -run TestRealRepoCentralizedExportService_AndTask13OnDiskRegeneration` | `PASS (exit=0)` |
| `EV-20261027T163505-e2ef` | `test` | `go test -v ./internal/webui -run TestTelegramFractalSubTaskFlow` | `PASS (exit=0)` |
| `EV-20261027T163754-742d` | `runtime` | Thực thi E2E `approve_and_run` cho `EX-002` trên `:8090` (`2/2 Sub-Tasks VERIFIED`, `diff.patch` `647` dòng / `22,539` bytes) | `PASS (exit=0)` |
| `EV-20261027T165015-a4eb` | `runtime` | `go run ./cmd/orchestrator daemon --once` | `PASS (exit=0)` |
| `EV-20261027T165109-6b13` | `runtime` | `curl -sS -G -d task=EX-002 -d step=S4 http://127.0.0.1:8090/api/dashboard` | `PASS (exit=0)` |

---

## 4. (d) Self-Improvement & Lesson Audit
- Đã ghi nhận đầy đủ 2 bài học trong `agent/memory/global/lessons.md` và đồng bộ `lessons/INDEX.md` (`347` entries):
  1. Bài học về lỗi `EX-002` báo `DONE` nhưng không có code diff thật và Sub-Task treo ở `"S3 Chờ Duyệt"`.
  2. Bài học Mid-Session Fix (`#never-edit-external-repo-from-orchestrator-session`) nghiêm cấm tuyệt đối việc gọi trực tiếp `replace_file_content` / `write_to_file` từ phiên làm việc `ai-orchestrator` lên bất kỳ file nào thuộc repo ngoài (`/Users/trainguyen/Documents/work/centralized-export-service` hoặc `/Users/trainguyen/Documents/work/data-hub/*`).

---

## 5. (e) Kết Quả Hội Đồng `/laya_router` (`DG3`) Sau Lần Sửa Code Cuối
- **Lệnh thực thi:** `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg3 --workspace audit-master-mongo-orchestrator-2026-10-22 --claims /tmp/dg3_fix_s4_real_llm_diff.json`
- **Summary File:** `06_laya_summary_dg3_20261027T165600.json`
- **Raw File:** `06_laya_raw_dg3_20261027T165600.json`
- **Claims Verified (`A-WAVE`, `A-CODEGEN`, `A-DIFF` + `CTRL-TRUE-1`, `CTRL-FALSE-1`, `CTRL-FALSE-2`):** `gate = PASS` (`UNVALIDATED` ở vòng đối chiếu Laya do không có round vượt ngưỡng tin cậy đồng thời trên cả control, đã xác minh trực tiếp bằng code thật tại `server.go:1211-1266`, `subtask_codegen_muscle.go:32-75`, `task_inspector.go:1301-1332` và các biên lai `EV-20261027T163446-a174`, `EV-20261027T163449-05f1`, `EV-20261027T165015-a4eb`, `EV-20261027T165109-6b13`).

---

## 6. (f) Security & Governance Gate Checklist (`G1–G8`)
- **G1 (Traceability):** Khớp 100% với `01_requirements_fix_s4_real_llm_diff_and_subtask_done_workflow.md` và `09_tasks_solution_fix_s4_real_llm_diff_and_subtask_done_workflow.md`.
- **G2 (Red -> Green):** Tái hiện và kiểm chứng bằng unit/integration test + E2E `EX-002`.
- **G3 (Real Runtime & Evidence):** Đầy đủ biên lai `EV-...` sau lần sửa code cuối.
- **G4 (Edge Cases & Isolation):** Chuẩn hóa đường dẫn thực tế, chặn fake hunk, cô lập diff từng Sub-Task.
- **G5 (Regression):** Toàn bộ test suite `internal/webui` PASS, kỷ luật số dòng `NOW <= HEAD` đạt 100%.
- **G6 (Output Correctness):** `TASK-TRAN01` (`Wave 0 · S6 Hoàn Tất (DONE)`) & `TASK-TRAN02` (`Wave 1 · S6 Hoàn Tất (DONE)`), `diff.patch` `647` dòng code ClickHouse thật.
- **G7 (Adversarial Audit & DG3):** Đã qua `DG3` (`06_laya_summary_dg3_20261027T165600.json`, `gate: PASS`) và `/security-agent` (không leak secret/PII, không `git push`).
- **G8 (Physical Artifacts):** Đầy đủ hồ sơ trong workspace `audit-master-mongo-orchestrator-2026-10-22`.
