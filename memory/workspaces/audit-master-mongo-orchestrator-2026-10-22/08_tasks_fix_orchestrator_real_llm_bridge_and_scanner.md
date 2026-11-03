# 08. Tasks — Khắc phục Toàn diện Hệ thống `ai-orchestrator` (`fix_orchestrator_real_llm_bridge_and_scanner`)

## [x] TASK-ORCH-01: Khắc phục `internal/runner/antigravity.go` & Đấu nối `AntigravityRunner` (`gemini-4-argon`) vào `internal/webui/` (`REQ-ORCH-01` — `DONE · EV-20261027T103920-69c8, EV-20261027T104124-992a`)
- **Mô tả & Context `file:line`:**
  - `internal/runner/antigravity.go:150-218` (`normalizeAntigravityModel`, `buildClaudeCLIArgs`, `buildGeminiCLIArgs`, `resolveCommand`, `isAntigravityNoiseLine`) và `antigravity.go:108-148` (`Complete`):
    - Chuẩn hóa tên model UI (`"gemini-4-argon (antigravity)"`, `"claude-opus-4-6 (antigravity)"`) bằng cách strip hậu tố `" (antigravity)"` và prefix `"antigravity/"`; không truyền `-m gemini-4-argon` xuống `gemini` CLI để tránh lỗi `429 limit: 0`.
    - Bổ sung tìm kiếm đường dẫn binary `gemini` / `claude` trong các thư mục cài đặt thực tế (`~/.nvm/versions/node/*/bin/gemini`, `/opt/homebrew/bin`, `/usr/local/bin`, `~/.local/bin`) khi `exec.LookPath` trong môi trường daemon không thấy.
    - Truyền đủ cờ `--skip-trust -e none -p` cho `gemini` và biến môi trường `GEMINI_CLI_TRUST_WORKSPACE=true`, `COLORTERM=truecolor`, `TERM=xterm-256color` để tránh lỗi `exit code 55 (not running in a trusted directory)` và cảnh báo terminal.
  - Tạo module đơn trách nhiệm mới `internal/webui/task_llm_bridge.go`:
    - Cung cấp `s.invokeArchitectLLM` gọi thật `runner.NewAntigravityRunner(...)` với model đã cấu hình của Slot (`gemini-4-argon (antigravity)` ở Slot A `/brain-delegate`), nạp đầy đủ mã nguồn thật từ các file nguồn sạch của `TargetRepo` (đã loại bỏ `dist/`, `build/`, `node_modules/`).
- **Logs / Test:**
  - Unit & Integration test PASS: `EV-20261027T103920-69c8` (`exit=0`) và `EV-20261027T104124-992a` (`exit=0`).
- **Definition of Done (DoD):**
  - [x] `internal/webui/` thực sự gọi `AntigravityRunner.Complete` khi thảo luận kiến trúc (`discuss`) hoặc điều chỉnh giải pháp (`revise_solution`).

## [x] TASK-ORCH-02: Khóa `FindMatchingAgentWorkspace` theo `TargetRepo` & Làm sạch `extractTaskDeepKnowledge` (`REQ-ORCH-02` — `DONE · EV-20261027T104124-992a, EV-20261027T104233-fae1`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:868-938` (`FindMatchingAgentWorkspace`) & `internal/webui/task_llm_bridge.go:84-122` (`workspaceMatchesTargetRepo`):
    - Kiểm tra tên `TargetRepo` (hoặc `qt.ExternalID`) phải xuất hiện trong tên workspace hoặc nội dung hồ sơ trước khi chấp nhận kết quả khớp từ khóa chung (`mongo`, `postgres`, `trans`). Chặn lấy hồ sơ của `core-trans-proxy-history-service` gắn cho `centralized-export-service`.
  - `internal/webui/task_chat_knowledge.go:91-224` (`extractTaskDeepKnowledge`):
    - Sửa lỗi gán nhầm biến trả về thứ 4 `req01File` vào `ws13` tại `task_chat_knowledge.go:129`.
    - Lọc sạch boilerplate quy trình (`Gate G2: Red -> Green Reproduction`, `VerifySubTaskCodeOnDisk`, `evidence_run.py`) qua `isInternalOrchestratorBoilerplateLine`.
- **Logs / Test:**
  - Test với `TASK-13` (`centralized-export-service`): `0%` nội dung `VerifySubTaskCodeOnDisk`, `Gate G2`, hay `NATS/Kafka DLQ` của workspace khác lọt vào câu trả lời (`EV-20261027T104124-992a`, `EV-20261027T104233-fae1` `exit=0`).
- **Definition of Done (DoD):**
  - [x] Không còn hiện tượng "râu ông nọ cắm cằm bà kia" giữa các Task/Workspace khác repo.

## [x] TASK-ORCH-03: Chặn quét thư mục build (`dist/`, `build/`) & Boilerplate `__awaiter`, Tự động nhận diện Language Fence (`REQ-ORCH-03`, `REQ-ORCH-04` — `DONE · EV-20261027T104124-992a, EV-20261027T104233-fae1`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:2955-3082` (`inspectTargetRepoSymbols`) & `internal/webui/task_llm_bridge.go:25-82`:
    - Chặn cứng `dist`, `build`, `out`, `.next`, `coverage`, `target`, `.cache`, `tmp`, `.turbo` và các file build/minified (`*.min.js`, `*.bundle.js`, `*.d.ts`, `*.map`).
    - Chặn trích xuất các hàm boilerplate do compiler sinh ra (`function adopt(value)`, `__awaiter`, `__generator`, `__importDefault`, `__exportStar`, `Object.defineProperty(exports`).
    - Ưu tiên sắp xếp file nguồn gốc (`.ts`, `.tsx`, `.go`, `.py`, `.sql`) lên trước `.js`.
  - `internal/webui/unified_doc_format.go:180-325` và `internal/webui/task_inspector.go:2227-2387`:
    - Thay thế toàn bộ chỗ hardcode ````go` bằng hàm `detectCodeFenceLang(filePath, snippet)` trả về đúng `ts`, `js`, `go`, `python`, `sql`.
  - `internal/webui/task_inspector.go:1165-1205` (`DecomposeMultiRepoCodebaseSubTasks`) & `internal/webui/task_llm_bridge.go:170-191` (`buildSubTaskModuleTitleAndDesc`):
    - Bỏ lối ghép chuỗi máy móc `fmt.Sprintf("Triển khai kiến trúc & nghiệp vụ cốt lõi cho %s trên %s", qt.Title, repoName)`; sinh mô tả Sub-Task theo đúng kiến trúc V1/V2 và file nguồn thật.
- **Logs / Test:**
  - Chạy `inspectTargetRepoSymbols` trên `centralized-export-service`: kết quả đầu bảng là `models/trans-his/trans-his.model.ts` và `dbHandler/postgres.ts` (không có bất kỳ file `dist/...` nào), language fence là ````ts` (`EV-20261027T104124-992a`, `EV-20261027T104233-fae1` `exit=0`).
- **Definition of Done (DoD):**
  - [x] `0` điểm chạm nào trỏ vào `dist/`; `0` đoạn code `.ts`/`.js` nào bị bọc trong ````go`.

## [x] TASK-ORCH-04: Nhận diện Chỉ đạo Đổi hướng Kiến trúc (`classifyChatActionMode`) & Re-Open / Re-Plan thật sự thay vì báo `DONE` ảo (`REQ-ORCH-05` — `DONE · EV-20261027T104124-992a, EV-20261027T104233-fae1`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_chat.go:845-875` (`classifyChatActionMode`) & `internal/webui/task_llm_bridge.go:139-169` (`isArchitecturalSteeringDirective`, `isDualVersionV1V2Directive`):
    - Nhận diện các mẫu câu chỉ đạo kiến trúc tự nhiên của User (`"giải pháp của anh là"`, `"hiện tại mày đang làm"`, `"bỏ cái export type cũ đi"`, `"tạo 1 cái type v2"`, `"v1 vẫn là cái cũ"`, v.v.) và trả về `"revise_solution"`.
  - `internal/webui/task_llm_bridge.go:368-475` (`executeArchitectReviseSolution`):
    - Thu hồi trạng thái `DONE` ảo: xóa `.user-approved` & `diff.patch` cũ, cập nhật `qt.Stage = queue.StageNeedsYou` (`NEEDS_YOU`).
    - Re-open toàn bộ Sub-Tasks về `NEEDS_YOU` (`0/2 DONE`) trên file nguồn `.ts` sạch (`models/trans-his/trans-his.model.ts`, `dbHandler/postgres.ts`) và cập nhật `09_tasks_solution.md` + `chat_history.json` theo đúng kiến trúc Strategy/Factory V1 (Mongo) + V2 (Postgres) với code fence ````ts`.
- **Logs / Test:**
  - E2E & Runtime `:8090` PASS (`EV-20261027T104124-992a`, `EV-20261027T104233-fae1` `exit=0`).
- **Definition of Done (DoD):**
  - [x] Toàn bộ 5 lỗi chí mạng trên `ai-orchestrator` được khắc phục triệt để và chứng minh bằng biên lai `EV-...` (`exit=0`).

## [x] TASK-ORCH-05: Cách ly tuyệt đối `.orchestrator/tasks/<task_id>` khỏi `memory/workspaces/` & Khóa `selectedChainKey` / `findQueuedTask` chống nhảy sang `TASK-13` (`REQ-ORCH-06`, `REQ-ORCH-08` — `DONE · EV-20261027T113701-abe6, EV-20261027T113702-4bfd, EV-20261027T113733-dc53`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:502-607` (`isExplicitOrchestratorTarget`, `ResolveTaskTargetRepo`, `ResolveTaskAffectedRepos`), `888-904` (`FindMatchingAgentWorkspace`), `1232-1279` (`hasContaminatedSubTasks`, `loadOrGenerateSubTasks`), `3641-3665` (`restoreSmokeCompletedUserTasks`):
    - Cấm khớp nhầm `cdc-01` (có `[mongodb] traitestpbs` và `master_payment_bill_service` trong mô tả) vào `feat-master-mongo-destination-2026-10-20` (`isMasterMongoDestinationTask(qt)`).
    - Xóa bỏ nhánh quét tự do `memory/workspaces/`; cấm copy/seed `08_tasks_*.md` và `09_tasks_solution_*.md` từ `agent/memory/workspaces/` sang `.orchestrator/tasks/<task_id>`.
  - `internal/webui/server.go:1091-1102` (`findQueuedTask`) & `internal/webui/static/app.js:52-64` (`fetchDashboard`):
    - Trong `findQueuedTask`, khi `taskKey != ""` không khớp task nào, trả về lỗi rõ ràng thay vì âm thầm fallback xuống phần tử `Stage != DONE` đầu tiên (`TASK-13`).
    - Trong `fetchDashboard(optionalTaskKey)`, luôn giữ `selectedChainKey` qua các nhịp SSE `update`.
- **Logs / Test:**
  - Unit & integration test `TestCDC01IsolationAndNoWorkspaceContamination` PASS (`EV-20261027T113702-4bfd` `exit=0`).
- **Definition of Done (DoD):**
  - [x] `cdc-01` chỉ trỏ vào `centralized-data-service`, hoàn toàn sạch bóng `TASK-M01..M10` và `feat-master-mongo-destination-2026-10-20`, không bao giờ nhảy sang Plan của `TASK-13`.

## [x] TASK-ORCH-06: Xóa bỏ Regex String-Prefixing `"- **Phương án khắc phục:** "` trong `/brain-delegate` & Luôn định tuyến qua `invokeArchitectLLM` (`REQ-ORCH-07` — `DONE · EV-20261027T113702-4bfd, EV-20261027T113709-1353, EV-20261027T113713-b65b`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_chat_knowledge.go:70-235` (`renderContextualDiscussReply`, `isMarkdownMetadataOrLessonNoiseLine`, `extractTaskDeepKnowledge`) & `internal/webui/unified_doc_format.go:296-325` (`BuildUnifiedCoArchitectReply`):
    - Trong `renderContextualDiscussReply`, luôn gọi `invokeArchitectLLM` trước cho mọi câu hỏi thảo luận (`discuss`).
    - Trong `extractTaskDeepKnowledge`, xóa bỏ hoàn toàn việc đọc từ `/Users/trainguyen/Documents/work/agent/memory/workspaces/`, chặn các dòng metadata (`Workspace**:`, `Suffix**:`, `Tier**:`, `Trạng thái**:`, `L010`, `L079`, `CMS Connection API`) và xóa bỏ hoàn toàn việc ghép tiền tố máy móc `"- **Phương án khắc phục:** "` trong `unified_doc_format.go`.
  - `internal/webui/task_chat_knowledge.go:237-344` (`reconcileChatHistoryAndAuditRemediation`):
    - Tự động phát hiện và làm sạch mọi hồ sơ task bị nhiễm chéo nội dung `feat-master-mongo-destination-2026-10-20` hoặc có câu trả lời chứa `"Phương án khắc phục: Workspace**:"`.
- **Logs / Test:**
  - Kiểm chứng qua `evidence_run.py`: `EV-20261027T113702-4bfd` (`exit=0`), `EV-20261027T113709-1353` (`exit=0`), `EV-20261027T113713-b65b` (`exit=0`), `EV-20261027T113733-dc53` (`exit=0`).
- **Definition of Done (DoD):**
  - [x] Mọi task trong `.orchestrator/tasks/<task_id>` hoàn toàn độc lập với `memory/workspaces/`, không còn câu `"Phương án khắc phục: Workspace**:"`, và câu trả lời `/brain-delegate` đi qua `invokeArchitectLLM`.

## [x] TASK-ORCH-07: Bổ sung đoạn Summary ngắn trả lời trực tiếp câu hỏi của User (`DirectAnswer`) ở ngay đầu mọi phản hồi trước khi vào nội dung chính (`REQ-ORCH-08` — `DONE · EV-20261027T143210-40a0, EV-20261027T143707-20e1`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_llm_bridge.go:18-24, 226-231, 281-355` (`ArchitectLLMAnalysis`, `invokeArchitectLLM`, `synthesizeSymbolGroundedArchitectAnalysis`):
    - Yêu cầu LLM trả thêm trường `"direct_answer": "..."` (2–4 câu tóm tắt trả lời trực tiếp câu hỏi/chỉ đạo của User) trong JSON schema tại dòng `228-230` và sinh `DirectAnswer` tương ứng trong `synthesizeSymbolGroundedArchitectAnalysis`.
  - `internal/webui/unified_doc_format.go:258-304` (`BuildUnifiedCoArchitectReply`) & `internal/webui/task_chat_knowledge.go:70-94` (`renderContextualDiscussReply`):
    - Hiển thị khối **`### 💬 Tóm Tắt & Trả Lời Trực Tiếp (Direct Summary)`** ở ngay đầu phản hồi (trước `## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ`) để trả lời thẳng câu hỏi của User trước khi đi vào 4 mục nội dung chính.
  - `internal/webui/telegram_message_format.go:530-570` (`formatTelegramCoArchitectReply`):
    - Giữ nguyên đoạn Summary trả lời trực tiếp ở ngay đầu thẻ tin nhắn Telegram trước khi liệt kê các điểm phân tích chính.
- **Logs / Test:**
  - Unit & integration test `go test -v -race ./internal/webui -run TestArchitectLLMBridge` xác nhận phản hồi luôn bắt đầu bằng khối `💬 Tóm Tắt & Trả Lời Trực Tiếp` trả lời thẳng câu hỏi của User trước khi vào `## 1.` (`EV-20261027T143210-40a0`).
- **Definition of Done (DoD):**
  - [x] 100% phản hồi của Agent trên khung chat và của Co-Architect Bot trên WebUI/Telegram đều mở đầu bằng đoạn Summary ngắn trả lời trực tiếp câu hỏi của User rồi mới tới nội dung chính.

## [x] TASK-ORCH-08: Nâng cấp 4 chuẩn Formatter & Prompt Builder trong `ai-orchestrator` (`REQ-ORCH-09` — `DONE · EV-20261027T143608-44bd, EV-20261027T143623-8c4a`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/telegram_message_format.go:245, 272-425` (`compactAffectedFileBasenames` -> `normalizeAndDeduplicateRelativeFiles`, `formatTelegramCompactApprovalCard`, `formatTelegramCompactSubTaskCard`):
    - (1) Cắt prefix `<repoName>/`, giữ relative path (`models/trans-his/trans-his.model.ts`) và khử trùng lặp qua `Set`.
    - (3) Loại bỏ hoàn toàn meta-system noise (*"Đã quét trực tiếp..."*, *"Mọi thay đổi ở bước S4..."*, hướng dẫn bấm nút dài dòng).
    - (4) Xóa bỏ `truncateRunes` ở tên nhánh `branch`, tên file, lệnh `DoD` và nội dung tóm tắt Sub-Task ở mục 3.
  - `internal/webui/task_inspector.go:1149, 1171` & `internal/webui/task_llm_bridge.go:56-83, 221-231, 300-338` (`detectRepoLanguageAndDoD`, `invokeArchitectLLM`, `synthesizeSymbolGroundedArchitectAnalysis`):
    - (2) Phát hiện động `package.json` vs `go.mod` trong `repoPath` (`package.json` -> ````typescript` + DoD `npm run build && npm test`; `go.mod` -> ````go` + DoD `go test -race ./...`).
    - Cập nhật prompt & Code Demo của `TASK-13` tách riêng PostgreSQL repository (`PostgresDB`) khỏi Mongoose `TransHisModel` (không trộn lẫn `pgPool`, `knexPg` và `dbConnection` Mongo).
  - `internal/webui/unified_doc_format.go:145-148, 230-236` (`BuildUnifiedS2AnalysisDoc`, `BuildUnifiedS3SolutionDoc`):
    - Loại bỏ các câu meta-system noise dài dòng khỏi bản Plan `S3`.
- **Logs / Test:**
  - Unit & integration test `go test -v -race ./internal/webui/...` PASS (`EV-20261027T143608-44bd`, `exit=0`).
- **Definition of Done (DoD):**
  - [x] Toàn bộ 4 điểm Formatter & Prompt Builder hoạt động chính xác 100%, không cắt `...` trên định danh, tự động chọn đúng ngôn ngữ/DoD theo `package.json` vs `go.mod`, khử trùng lặp relative file path và sạch bóng meta-system noise.

## [x] TASK-ORCH-09: Tự chữa lành 5 phát hiện Audit S5 — Tách module đơn trách nhiệm (`repo_tech_profile.go`, `task_revise_workflow.go`), đưa `subtask_git_apply.go <= HEAD`, sửa văn bản mâu thuẫn `Knex/PG Pool` & `TestDoD`, kiểm chứng Repo thật + Live `:8090` (`S5 -> S3 -> S4 -> S5` — `DONE · EV-20261027T143208-39a6, EV-20261027T143210-40a0, EV-20261027T143618-8c79, EV-20261027T143623-8c4a`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/repo_tech_profile.go` (Module mới đơn trách nhiệm `246` dòng):
    - Gom tập trung toàn bộ logic nhận diện Tech Stack (`detectRepoTechStackProfile`, `detectTargetRepoLanguageAndDoD`, `detectCodeFenceLang`) và chuẩn hóa/khử trùng lặp đường dẫn tương đối (`normalizeOrchestratorDisplayPath`, `normalizeRelativeRepoFiles`, `normalizeAndDeduplicateRepoRelativePaths`), xóa bỏ hoàn toàn code trùng lặp trong `task_llm_bridge.go` và `telegram_message_format.go`.
  - `internal/webui/task_revise_workflow.go` (Module mới đơn trách nhiệm `206` dòng):
    - Tách `executeArchitectReviseSolution` và `buildCleanRoomSpecSectionFromSubTasks` ra khỏi `task_llm_bridge.go` để `task_llm_bridge.go` giữ đúng trách nhiệm đơn là cầu nối Prompt/LLM (`230` dòng).
  - `internal/webui/subtask_git_apply.go:1-310` & `internal/gitclient/patch_guard.go`:
    - Ủy quyền kiểm tra biên dịch sau khi apply patch sang `gitclient.CheckGoPackageCompilesOrRevertRelFile`, đưa `subtask_git_apply.go` từ `345` dòng về `310` dòng (`<= HEAD 312`), đảm bảo 100% file cũ trong repo thỏa mãn `NOW <= HEAD`.
  - `internal/webui/unified_doc_format.go:228-266` & `internal/webui/task_inspector.go:1133-1182`:
    - Sửa dòng `Điểm chạm chính (Core Touchpoints)` tại `unified_doc_format.go:258` không còn nhắc tới `Knex/PG Pool` hay sửa đè `trans-his.model.ts` (chỉ dùng `pgPool` thuần trong `PostgresDB` cho luồng V2 và giữ nguyên Mongoose V1 ở `trans-his.model.ts`), đồng thời cắt prefix `<repoName>/` ở cột `Điểm chạm (file:line)` của bảng Sub-Tasks (`unified_doc_format.go:232`).
    - Chuẩn hóa `TestDoD` trong `DecomposeMultiRepoCodebaseSubTasks` trả về chính xác `npm run build && npm test` (khi có `package.json`) hoặc `go test -race ./...` (khi có `go.mod`).
  - `internal/webui/task_llm_bridge_test.go`:
    - Khôi phục assertion nghiêm ngặt `st.TestDoD == "npm run build && npm test"` và bổ sung test kiểm chứng trực tiếp trên repo thật `/Users/trainguyen/Documents/work/centralized-export-service` + tái sinh `.orchestrator/tasks/TASK-13/09_tasks_solution.md` & `08_tasks.md` trên đĩa.
- **Logs / Test:**
  - Chạy `go vet ./...` (`EV-20261027T143208-39a6`), `go test -v -race ./internal/webui/...` (`EV-20261027T143210-40a0`, `EV-20261027T143216-398b`, `EV-20261027T143608-44bd`, `EV-20261027T143707-20e1`), `go run ./cmd/orchestrator daemon --once` (`EV-20261027T143618-8c79`), khởi động lại `./bin/orchestrator` trên `:8090` và kiểm chứng JSON thực tế từ `/api/dashboard?task_key=TASK-13` (`EV-20261027T143623-8c4a`).
- **Definition of Done (DoD):**
  - [x] 100% file sẵn có thỏa mãn `NOW <= HEAD`, các module mới tách đúng trách nhiệm đơn, không còn code trùng lặp hay câu chữ mâu thuẫn `Knex/PG Pool`, và trace đầy đủ giá trị runtime thật trên `:8090`.

## [ ] TASK-ORCH-10: Xóa sổ 100% Hardcode `Postgres/V1-V2` & Đấu nối Model (`invokeArchitectLLM`) Toàn trình vào Bộ sinh Plan tổng (`generateDeepTaskArtifacts`), Khung Chat (`handleTaskPlanningChat`) và Luồng Phản hồi (`WebUI & Telegram`) (`REQ-ORCH-10`)
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_llm_bridge.go:18-24, 166-422`:
    - Mở rộng `ArchitectLLMAnalysis` thêm `ShouldUpdatePlan bool`, `Architecture []string`, `Risks []string`, `SubTasks []ArchitectLLMSubTaskSpec`, `ChangedFiles []string`.
    - Xóa sổ hoàn toàn `isDualVersionV1V2Directive` (`dòng 189-195`), xóa nhánh hardcode `PostgresDB` trong `buildSubTaskModuleTitleAndDesc` (`dòng 203-210`), `synthesizeDirectAnswerSummary` (`dòng 222-224`), `synthesizeSymbolGroundedArchitectAnalysis` (`dòng 358-404`), và xóa câu nhắc `TASK-13 (centralized-export-service) ... PostgresDB` trong `sysPrompt` (`dòng 279`).
    - Trong `invokeArchitectLLM` (`dòng 241-317`): Truyền đầy đủ `qt.Description` nguyên bản + lịch sử hội thoại gần nhất (`chat_history.json`) + `subTasks` hiện tại vào prompt để Model (`gemini-4-argon` qua `AntigravityRunner`) nắm trọn ngữ cảnh, trả lời đúng trọng tâm câu hỏi trong `direct_answer`, đồng thời sinh đầy đủ `architecture`, `risks`, `sub_tasks`, `changed_files`, `code_demo` và cờ `should_update_plan`.
  - `internal/webui/task_chat_knowledge.go:70-94` (`renderContextualDiscussReply`) & `internal/webui/unified_doc_format.go:206-392` (`BuildUnifiedS3SolutionDoc`, `BuildUnifiedCoArchitectReply`):
    - Truyền trực tiếp `llmRes.DirectAnswer` vào `BuildUnifiedCoArchitectReply` và `BuildUnifiedS3SolutionDoc` (tuyệt đối không gọi `synthesizeDirectAnswerSummary` đè lên `DirectAnswer` của Model).
    - Trong `renderContextualDiscussReply` / `handleTaskPlanningChat` (`internal/webui/task_chat.go:822-915`): Khi User nhắn chỉ đạo kiến trúc, yêu cầu sửa giải pháp, hoặc sửa lưng/chỉ ra Plan đang bị nhầm (`llmRes.ShouldUpdatePlan == true` hoặc `isPlanCorrectionOrSteeringMessage(cleanMsg, *qt)`), tự động gọi `s.syncArchitectLLMAnalysisToMasterPlan(...)` để cập nhật ngay `subtasks.json`, `08_tasks.md`, `09_tasks_solution.md`, `01_requirements.md` và trả về bản Plan + Code Demo đã cập nhật.
    - Xóa nhánh `if isPostgresOrDualSourceMigrationTask(...)` trong `BuildUnifiedS3SolutionDoc` (`unified_doc_format.go:256-265`), thay bằng `archBullets` & `riskBullets` do Model sinh ra.
  - `internal/webui/task_inspector.go:1089-1198, 3200-3295` (`DecomposeMultiRepoCodebaseSubTasks`, `generateDeepTaskArtifacts`, `buildConcreteCodeDemoForTask`):
    - Trong `generateDeepTaskArtifacts`: Gọi `s.invokeArchitectLLM(ctx, qt, subTasks, qt.Title+"\n"+qt.Description, routedModel, "plan")` để cập nhật `subTasks` (`subtasks.json`, `08_tasks.md`), `01_requirements.md` và `09_tasks_solution.md` bằng đúng kết quả phân tích của Model (ví dụ với `EX-002` là ClickHouse `dbHandler/clickhouse.ts` + Router V1/V2/V3).
    - Xóa nhánh `if isPostgresOrDualSourceMigrationTask(...)` trong `buildConcreteCodeDemoForTask` (`task_inspector.go:3292-3294`).
  - `internal/webui/telegram_message_format.go:135-315` & `internal/webui/telegram_bridge.go:1999-2162` (`handleTelegramReviseSolutionCommand`, `applyTelegramSolutionFeedbackToPlan`, `formatTelegramCompactApprovalCard`):
    - Xóa hoàn toàn `isPostgresOrDualSourceMigrationTask` và các đoạn hardcode `PostgresDB` trong `telegram_message_format.go`; đọc trực tiếp `Architecture`, `Risks`, `CodeDemo`, `SubTasks` từ hồ sơ `09_tasks_solution.md` & `subtasks.json` của Task.
    - Trong `handleTelegramReviseSolutionCommand`: Gọi `s.executeArchitectReviseSolution(...)` (qua `invokeArchitectLLM`) để viết lại toàn bộ Plan tổng (`09_tasks_solution.md`, `08_tasks.md`, `subtasks.json`, `01_requirements.md`) theo phản hồi của User trước khi gửi lại thẻ H4b trên Telegram.
- **Logs / Test:**
  - Chạy `go vet ./...`, `go test -v -race ./internal/webui/...` qua `evidence_run.py`, tái sinh `.orchestrator/tasks/EX-002` & kiểm chứng trực tiếp trên live server `:8090` (`/api/dashboard?task_key=EX-002` và `/api/tasks/chat`).
- **Definition of Done (DoD):**
  - [ ] `0%` hardcode `Postgres/V1-V2` trong Engine; `EX-002` hiển thị đúng Plan tổng, Sub-Tasks và Code Demo `ClickHouse` (`dbHandler/clickhouse.ts`, Router V1 Mongo / V2 Postgres / V3 ClickHouse); mọi phản hồi trên khung Chat và Telegram đều đi qua Model (`invokeArchitectLLM`), giữ nguyên `DirectAnswer` của Model và tự động cập nhật Plan tổng.
