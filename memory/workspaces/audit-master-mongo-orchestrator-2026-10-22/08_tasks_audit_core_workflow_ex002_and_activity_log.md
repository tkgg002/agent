---
tier: T2
suffix: audit_core_workflow_ex002_and_activity_log
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-27T14:17:45+07:00
status: PROPOSED
---

# 08. Tasks — Khắc Phục Toàn Bộ Điểm "Làm Láo" Trong Core Workflow (`S0 -> S6`) & Khôi Phục `Activity-Log Progress` (WebUI + Telegram)

## Danh mục Task Thực thi (Sau khi User `APPROVE`)

### `TASK-CW-01`: Xóa sổ 100% Cheat Code (`TASK-13 / TASK-11 / Postgres / Mongo / V1-V2`) & Đấu nối `invokeArchitectLLM` toàn trình (`S1/S2/S3`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_llm_bridge.go:24-210`: Xóa sạch mọi nhánh hardcode `isDualVersionV1V2Directive`, `isPostgresOrDualSourceMigrationTask`, `"TASK-13"`, `"PostgresDB"`, `"TransHisModel"`, `"pgPool"`, `"knexPg"`. Mở rộng `ArchitectLLMAnalysis` thêm `SubTasks []ArchitectSubTaskSpec` để Model trực tiếp phân rã Sub-Tasks.
  - `internal/webui/task_revise_workflow.go:30-81`: Xóa `isMasterMongoDestinationTask`, `isDualVersionV1V2Directive`, và nhánh hardcode `PostgreSQL V2` trong `buildSubTaskModuleTitleAndDesc`.
  - `internal/webui/telegram_message_format.go:13-186` & `telegram_subtask_card.go:337` & `unified_doc_format.go:256-265, 453`: Xóa `isPostgresOrDualSourceMigrationTask`, xóa template cứng `export class PostgresDB` trong `buildCleanKeyDiffPreview` và mọi nhánh `if isPostgresOrDualSourceMigrationTask`.
  - `internal/webui/task_planner_multilayer.go:12-177`: Xóa bỏ 6 Sub-Tasks hardcode `CORE-01-S01..S06` và `buildSelfUpgradeOrchestratorCodeDemo`.
  - `internal/webui/task_inspector.go:512-655, 1089-1285, 2597-2708, 2978-3005, 3282-3347`: Xóa `buildDedicatedSubTaskCodeDemo` (`case "TASK-M01".."TASK-M10"`), xóa các nhánh hardcode trong `buildConcreteCodeDemoForTask`, xóa điểm cộng từ khóa thiên vị `"trans-his"`/`"postgres"`/`"recon"` trong `inspectTargetRepoSymbols`, khớp động repo từ `DiscoverWorkspaceRepos`, và gọi `invokeArchitectLLM` trong `DecomposeMultiRepoCodebaseSubTasks` + `generateDeepTaskArtifacts`.
  - `internal/webui/task_chat.go:180-195, 1018-1038, 1495-1580` & `task_chat_knowledge.go:40-68`: Xóa các nhánh hardcode tên repo `centralized-data-service` / `cdc-cms-service` và văn mẫu trả lời tĩnh.
- **Logs / Test:** `go test -v ./internal/webui/...`
- **Definition of Done (DoD):** `grep -rnIE --exclude="*_test.go" "TASK-13|TASK-11|PostgresDB|TransHisModel|trans_his|pgPool|knexPg|masterMongoPool|_source_ts|isDualVersion|isPostgresOrDual|isMasterMongo" internal/webui internal/agentgate` trả về **0 kết quả**!

### `TASK-CW-02`: Khử "Làm Láo" ở Cổng Phản Biện `S2.5 (DG0)` & `BuildAndRunCalibratedLayaGate` theo chuẩn 100% Agnostic Core (`internal/agentgate/kernel_bridge.go` & `internal/webui/unified_doc_format.go`)
- **Mô tả & Context `file:line`:**
  - `internal/agentgate/kernel_bridge.go:637-923` (`BuildAndRunCalibratedLayaGate`): Chấm dứt trò tự lấy `pickLine(0)[:80]` của chính file vừa ghi làm `Claim == Evidence == MustContain`, đồng thời **TUYỆT ĐỐI KHÔNG dùng mảng từ khóa công nghệ hardcode (`"clickhouse"`, `"postgres"`, ...)**. Thay bằng trích xuất các mệnh đề giải pháp thực chất (`extractAgnosticDocumentClaims`) đối chiếu với `01_requirements.md` và mã nguồn thật qua Neural Engine `convaiinnovations/laya (:8200 /v1/predict)` + cắt chuỗi theo `[]rune` (`truncateUTF8Runes`).
  - Tại `S0` (`kernel_bridge.go:150-175`), nếu `checkLayaHealthHTTP` trả về `false` thì tự động gọi `hooks/agy/ensure_laya_up.sh`.
- **Logs / Test:** `go test -v ./internal/agentgate/...`
- **Definition of Done (DoD):** Không còn `pickLine(0)` tự gán `Claim == Evidence == MustContain`, không có bất kỳ danh sách từ khóa công nghệ hardcode nào trong `internal/agentgate/`.

### `TASK-CW-03`: Khôi phục `Activity-Log Progress` Thời gian thực trên cả WebUI lẫn Telegram Forum Sub-Topic (`internal/webui/server.go`, `internal/webui/telegram_bridge.go`, `internal/webui/static/index.html`, `internal/webui/static/app.js`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/server.go:932-968, 1122-1167` & `internal/webui/telegram_bridge.go:2674-2771`: Giữ lịch sử các bước thực thi trong `s.liveStepProgress` thay vì xóa trắng ngay lập tức; đưa bảng **📟 Live Activity-Log & Step Progress Stream** lên đầu màn hình `screen-autopilot` (`index.html`); gửi/live-edit tiến trình vào Topic Telegram.
- **Logs / Test:** `go test -v ./internal/webui -run TestLiveActivityLogProgress`
- **Definition of Done (DoD):** Cả trên đầu trang WebUI (`:8090`) lẫn trong Topic của Task trên Telegram đều hiển thị rõ tiến trình từng bước (`S0 -> S3`) và `terminal.log`.

### `TASK-CW-04`: Khử "Làm Láo" ở `S4 Implement` & `S5 Audit` (`internal/webui/server.go`, `internal/webui/task_inspector.go`, `internal/webui/subtask_git_apply.go`) — `DONE`
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:4184-4318`: Xóa bỏ hoàn toàn đoạn sinh hunk giả `// Isolated Remediation Hunk for ... CoerceToBSONValue, Ordered BulkWrite OCC (_source_ts <= incomingTs), masterMongoPool`.
  - `internal/webui/subtask_git_apply.go:205-208`: Bổ sung chốt chặn `if repoDir == "" || (s.Root != "" && filepath.Clean(repoDir) == filepath.Clean(s.Root)) { return 0 }` để ngăn server `:8090` tự động `git apply` các hunk `diff.patch` cũ lên chính cây thư mục nguồn `s.Root` (`ai-orchestrator`).
  - `internal/webui/server.go:1788-1860, 1966`: Chạy lệnh kiểm chứng thật (`RunEvidenceCommand`) theo `detectRepoTechStackProfile` cho cả Node.js/TypeScript (`package.json`) và Go (`go.mod`); **xóa bỏ lệnh giả `test -f 01_requirements.md`** và **xóa bỏ dòng hardcode `masterMongoPool / _source_ts`**.
- **Logs / Test:** `go test -v ./internal/webui/...` (`EV-20261027T162655-e2dd`, `EV-20261027T162707-3318`)
- **Definition of Done (DoD):** Không còn diff giả, không còn lệnh `test -f 01_requirements.md` đúc biên lai `EV-...` giả, không còn chữ `masterMongoPool` / `_source_ts` trong `server.go` hay `telegram_bridge.go` (`TestZeroCheatCodeAndNoHardcodedTechOrDomainStringsInProduction` PASS).

### `TASK-CW-05`: Xóa Sổ 100% Các Chốt Chặn Giấu AI Model, Xóa Template 6-Tầng `Core-01` (`decomposeSelfUpgradeOrchestratorSubTasks`), Sửa `applyLLMSubTaskSpecs`, Đấu Nối AI Model Toàn Trình & Tái Sinh Sạch Sẽ Task `core-003` (`S5 -> S3 -> S4 -> S5` Rule #9.8d)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_chat_knowledge.go:31-95` & `internal/webui/task_llm_bridge.go:185-250`: Xóa bỏ hoàn toàn nhánh `if isProgressOrStatusQuestion(cleanMsg)` trả về template `strings.Builder` tĩnh mà không gọi AI Model; thay vào đó đưa trực tiếp trạng thái thật `git status --short`, `git diff --stat`, kích thước `.orchestrator/tasks/<key>/diff.patch` và tiến độ Sub-Tasks vào prompt của `invokeArchitectLLM` để **chính AI Model trả lời trực tiếp**, không bao giờ giấu Model sau văn mẫu tĩnh.
  - `internal/webui/task_planner_multilayer.go:12-172` & `internal/webui/task_inspector.go:980-1001, 2985, 3059-3061, 3360`: Xóa sổ vĩnh viễn `isSelfUpgradeOrchestratorTask`, `decomposeSelfUpgradeOrchestratorSubTasks` (6 Sub-Tasks hardcode của `Core-01`), và `buildSelfUpgradeOrchestratorCodeDemo`.
  - `internal/webui/task_revise_workflow.go:75-110` (`applyLLMSubTaskSpecs`): Thay thế 100% danh sách `subTasks` bằng đúng danh sách `analysis.SubTasks` do AI Model trả về (hỗ trợ mọi số lượng `1..N` Sub-Tasks, sinh mã `<PREFIX>01..0N`, `WaveIndex`, `DependsOn`, và `TestDoD` động theo `detectRepoTechStackProfile`), xóa bỏ ngoại lệ hardcode `"Core-01"`.
  - `internal/webui/task_llm_bridge.go:185-245` & `internal/webui/task_inspector.go:2722-2776`: Tách chỉ thị thực của User (`extractUserDirectiveFromDescription`, ưu tiên nội dung sau dấu `=>`) khỏi đoạn code ví dụ dán trong đề bài; bỏ giới hạn cứng `"2 Sub-Tasks"` trong prompt để AI Model tự phân rã `2..6 Sub-Tasks` khớp với các file thật cần sửa trên đĩa.
  - `internal/webui/telegram_bridge.go:1814-1875, 2045`: Đấu nối `handleTelegramReviseSolutionCommand` và `handleTelegramReviseSubTaskSolutionCommand` gọi trực tiếp `executeArchitectReviseSolution` / `invokeArchitectLLM` thay vì chỉ nối dòng chữ tĩnh vào file markdown.
  - `internal/webui/unified_doc_format.go:99-204` & `internal/webui/task_inspector.go:2942-2960`: Đưa kết quả phân tích thật của AI Model (`llmRes.DirectAnswer`, `llmRes.RootCauses`, `llmRes.Risks`) vào `BuildUnifiedS2AnalysisDoc` (`13_analysis.md`) và `BuildUnifiedDG0CritiqueDoc` (`10_preplan_critique.md`).
  - `internal/webui/task_inspector.go:1154-1165, 1334-1375, 1408-1416, 2295, 3291-3345`, `internal/webui/task_chat.go:1034-1058, 1111, 1186`, `internal/webui/subtask_codegen_muscle.go:434-451`: Xóa bỏ hoàn toàn cơ chế copy lén `diff.patch`/`audit_report.md` giữa các task trùng `Title` (`byTitle[normTitle]` & sibling scan), xóa chuỗi hardcode `"TASK-A04"`, `"AF-ORCH-01..06"`, `"TASK-M0"`, `"feat-master-mongo-destination-2026-10-20"`, và xóa hunk comment giả `remediation_impl.go`.
  - Tái sinh sạch sẽ toàn bộ hồ sơ `.orchestrator/tasks/core-003/` (`09_tasks_solution.md`, `08_tasks.md`, `01_requirements.md`, `13_analysis.md`, `10_preplan_critique.md`, `subtasks.json`, `llm_analysis.json`, `chat_history.json`) bằng AI Model thật và cập nhật trực tiếp trên `:8090`.
- **Logs / Test:** `go test -race ./internal/agentgate/... ./internal/webui/...`
- **Definition of Done (DoD):** 0% hàm giấu AI Model, 0% template Sub-Tasks `Core-01` cứng, `core-003` trên đĩa và trên `:8090` sạch 100% không còn `S03..S06` của `Core-01` hay ảo giác `gate.go`, 100% file `NOW <= HEAD`.

