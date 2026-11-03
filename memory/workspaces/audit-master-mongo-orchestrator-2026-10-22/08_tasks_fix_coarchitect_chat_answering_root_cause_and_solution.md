# 08. Tasks — Khắc phục Lỗi Không Trả Lời Câu Hỏi Kỹ Thuật (`Vì Sao Chạy Không Đúng, Giải Pháp Là Gì`) trong `💬 THẢO LUẬN KIẾN TRÚC` (`fix_coarchitect_chat_answering_root_cause_and_solution`)

## Danh sách Hạng mục Thực thi (Sau khi User APPROVE)

### `TASK-CHAT-01` — Nâng cấp `buildContextualDiscussReply` & Bộ bóc tách Hồ sơ Kỹ thuật Đa tầng trong `internal/webui/task_chat.go`
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_chat.go:1325-1399`:
    - Loại bỏ hoàn toàn mẫu trả lời khuôn mẫu nhại lại câu hỏi ở cuối `buildContextualDiscussReply`.
    - Bổ sung hàm `extractDeepRootCauseAndSolutionForChat(qt, isoMeta, subTasks)`:
      - Đọc kết hợp `09_tasks_solution.md`, `13_analysis.md`, `audit_report.md` tại `.orchestrator/tasks/<TaskKey>/` **VÀ** hồ sơ gốc trong `/Users/trainguyen/Documents/work/agent/memory/workspaces/` (qua `FindMatchingAgentWorkspace` hoặc tìm kiếm theo từ khóa đề bài như `shadow_master_update_drop` cho `CDC-RECON-01`).
      - Trích xuất đầy đủ các mục **Nguyên nhân gốc rễ (`F1..F8` / Root Cause kèm `file:line`)**, **Giải pháp kỹ thuật (`Solution` cho từng sơ hở)**, **Khối `Code Demo`**, và **Danh mục Sub-Tasks / Điểm chạm trên từng Repo**.
    - Phân tích ý định câu hỏi của User trong `buildContextualDiscussReply` (`vì sao`, `tại sao`, `nguyên nhân`, `ko đúng`, `sai`, `giải pháp`, `sửa gì`, `làm sao`, `code demo`, `file nào`, `subtask`, hoặc câu hỏi bất kỳ) để trả lời trực tiếp, mạch lạc, đúng trọng tâm câu hỏi của User.
- **Logs / Test:**
  - Unit & Integration test `TestCoArchitectChat_AnswersWhyReconShadowToMasterWrongAndSolution` trong `internal/webui/server_test.go`.
- **Definition of Done (DoD):**
  - Khi gửi `"rồi vì sao chạy ko đúng, giải pháp là gì"` vào `POST /api/tasks/chat` cho `CDC-RECON-01` (hoặc bất kỳ task nào), phản hồi trả về đầy đủ các nguyên nhân gốc rễ (`F1..F8` kèm `file:line`), giải pháp kỹ thuật cho từng nguyên nhân và Code Demo thực tế; tuyệt đối không còn câu trả lời khuôn mẫu nhại lại câu hỏi.

### `TASK-CHAT-02` — Đồng bộ Hồ sơ `13_analysis.md` & Cập nhật Câu trả lời Thực chất vào `chat_history.json` của `CDC-RECON-01` + Hỗ trợ Hỏi-Đáp trên `telegram_bridge.go`
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:1540-1620` & `.orchestrator/tasks/CDC-RECON-01/13_analysis.md`, `chat_history.json`:
    - Khi task liên kết với workspace `/agent` có sẵn `13_analysis_*.md` hoặc `09_tasks_solution_*.md` giàu nội dung thực chứng (như `audit-shadow-master-update-drop-2026-10-20` cho `CDC-RECON-01`), bảo toàn nội dung phân tích gốc (`F1..F8`) thay vì ghi đè bằng khung scanner mẫu.
    - Cập nhật lại tin nhắn trả lời lúc `15:21:12` trong `.orchestrator/tasks/CDC-RECON-01/chat_history.json` bằng bản giải trình đầy đủ Nguyên nhân (`F1..F8`) & Giải pháp kỹ thuật.
  - `internal/webui/telegram_bridge.go:495-535`:
    - Khi User nhắn câu hỏi (`vì sao`, `tại sao`, `nguyên nhân`, `giải pháp`, `?`) trên Telegram, tự động định tuyến sang `buildContextualDiscussReply` để trả lời trực tiếp trên Telegram.
- **Logs / Test:**
  - Kiểm chứng trực tiếp qua `GET /api/tasks/chat?task_key=CDC-RECON-01` và `POST /api/tasks/chat` trên `:8090` (`evidence_run.py`).
- **Definition of Done (DoD):**
  - Giao diện `💬 THẢO LUẬN KIẾN TRÚC` trên `:8090` và Telegram trả lời thông minh, chi tiết, đúng bản chất kỹ thuật kèm `file:line`.

### `TASK-CHAT-03` — Apply Trực tiếp Hunk Code Của Từng Sub-Task Đã Duyệt (`k/10`) Xuống Working Tree Của `TargetRepo` (`git status`) & Khắc phục Lỗi `scout.MaskSecrets` Làm Hỏng Format String Trong `diff.patch`
- **Mô tả & Context `file:line`:**
  - `internal/scout/scout.go:21-43`:
    - Không mask nhầm chuỗi format Go (`%s:%s@`, `%v:%v@`) và biểu thức biến/thuộc tính code (`connForm.password`, `req.Password`) khi chạy `MaskSecrets`.
  - `internal/webui/telegram_bridge.go:2999-3108` & `internal/webui/task_inspector.go:3038-3085`:
    - Bổ sung `applyApprovedSubTaskPatchToTargetRepo(ctx, qt, st, subHunks)`: khi Sub-Task `st` đã có `.user-approved` và chạy `subtask_implement`, tự động khôi phục các placeholder `%s:%s@` / `connForm.password,` nếu bị mask trước đó rồi chạy `git -C <TargetRepo> apply` (fallback `git -C <TargetRepo> apply --3way`) cho các hunk thuộc `st.ContextFileLine` / `st.TargetFiles` để `git status` trên `TargetRepo` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`) hiển thị ngay thay đổi code thực tế của từng Sub-Task vừa duyệt!
- **Logs / Test:**
  - Kiểm chứng `git -C /Users/trainguyen/Documents/work/data-hub/cdc-cms-service status -s` sau khi duyệt `TASK-M01` (`1/10`) và unit test trong `internal/webui/server_test.go`.
- **Definition of Done (DoD):**
  - Duyệt tới Sub-Task nào (`1/10`, `2/10`, ...) thì `git status` trên repo đích tương ứng hiện ngay các file đã sửa của đúng Sub-Task đó.

### `TASK-CHAT-04` — Thực thi Đầy đủ 100% Chu trình 12 Bước / 6 Wave Chuẩn Fractal Cho Từng Sub-Task (`1/10 .. 10/10`) Trong `task_inspector.go` & `telegram_bridge.go`
- **Mô tả & Context `file:line`:**
  - `internal/webui/task_inspector.go:2989-3215` & `internal/webui/telegram_bridge.go:1680-1894`:
    - Xóa bỏ kiểu chạy tắt 3 bước (`Step 2/5 -> Step 3/5 -> Step 4/5`).
    - Khi bước vào `STEP_SUBTASKS` cho `Sub-Task 1/N` hoặc chuyển tiếp sang `Sub-Task k+1/N` (hoặc chạy `Approve All`), bắt buộc kích hoạt **Wave 1 (`subtask_plan`: `Step 1/12 S0 Boot H1 & S1 Intake` -> `Step 2/12 S2 Deep Repo Scan & FTS5 Tripwires` -> `Step 3/12 S2.5 Pre-Plan Critique DG0 (:8200)` -> `Step 4/12 S3 Solution + Code Demo + Laya DG1/DG2 (:8200)`)** trước khi dừng ở `NEEDS_YOU` và gửi Card duyệt Telegram.
    - Khi User bấm `Approve` Sub-Task `k/N`, thực thi đầy đủ **Wave 2 (`Step 5/12 Human Gate APPROVE`) -> Wave 3 (`subtask_implement`: `Step 6/12 S4 Restore-Point` -> `Step 7/12 S4 Real Code Git Apply on TargetRepo` -> `Step 8/12 S4 Gate H4b Zero-Mockup & H3 Secret Scan + DG2`) -> Wave 4 (`subtask_qa`: `Step 9/12 S5 5A Real Compiler/Vet/Test EV-... on TargetRepo + VerifySubTaskCodeOnDisk` -> `Step 10/12 S5 5B Cross-Model Line-by-Line Diff Audit & Laya DG3 (:8200)` -> `Step 11/12 S5 Self-Healing Loop Check`) -> Wave 5 (`subtask_finalize`: `Step 12/12 S6 Finalize, Governance Audit H7 & OutputContract Handoff`)**.
- **Logs / Test:**
  - Unit & Integration test `TestSubTaskFull12StepWaveLifecycle_NoShortcut` trong `internal/webui/server_test.go` và kiểm chứng trực tiếp log `terminal.log` + `.orchestrator/tasks/<Task>/subtasks/<SubID>/05_progress.md` qua `evidence_run.py`.
- **Definition of Done (DoD):**
  - Mọi Sub-Task (`1/10 .. 10/10`) đều có đủ hồ sơ và log thực thi 12/12 bước chuẩn 6 Wave (`S0 -> S1 -> S2 -> S2.5 DG0 -> S3 + DG1/DG2 -> Human Gate -> S4 Restore-Point + Git Apply + H4b/H3 -> S5 5A Real Compiler EV-... + 5B Diff Audit & DG3 -> S6 Finalize & OutputContract Handoff`).

### `TASK-CHAT-05` — Tự động Tạo Sub-Chat (Forum Topic) Riêng Trên Telegram Cho Mỗi Task (`createForumTopic`) & Định Tuyến 2 Chiều Theo `message_thread_id`
- **Mô tả & Context `file:line`:**
  - `internal/webui/telegram_bridge.go:61-100, 367-465, 500-575, 811-945, 2508-2840`:
    - Mở rộng `telegramPersistedConfig` và `TelegramSettingsDTO` thêm `TaskThreadIDs map[string]int64` (`json:"task_thread_ids,omitempty"`) để lưu ánh xạ `TaskKey -> message_thread_id`, đồng thời lưu file `.orchestrator/tasks/<SafeTaskKey>/telegram_topic.json`.
    - Bổ sung `ensureTelegramTaskTopic(ctx, cfg, qt) int64`: nếu Task chưa có `message_thread_id`, gọi `POST https://api.telegram.org/bot<TOKEN>/createForumTopic` với tên Topic chuẩn `📦 [<Repo>] #<ID> <Title>` (<= 120 ký tự), lưu `message_thread_id` và trả về để đính kèm vào mọi lệnh `sendMessage` của Task đó. Nếu Topic cũ bị xóa trên Telegram (HTTP 400 `message thread not found`), tự động tạo lại Topic mới và gửi lại.
    - Mở rộng `telegramIncomingMessage` thêm `MessageThreadID int64` (`json:"message_thread_id,omitempty"`). Khi nhận tin nhắn hoặc `CallbackQuery` từ một `MessageThreadID > 0`, tự động tra ngược `findTaskKeyByTelegramThreadID(threadID)` để gắn ngữ cảnh Task hiện tại (hỗ trợ hỏi đáp Co-Architect, sửa giải pháp S3, duyệt Task/Sub-Task trực tiếp trong Sub-Chat mà không cần gõ `#ID`).
- **Logs / Test:**
  - Unit & Integration test `TestTelegramPerTaskForumTopicSubChat_CreateAndRoute2Way` trong `internal/webui/server_test.go` + kiểm chứng tạo thật Sub-Chat Topic trên Supergroup `Ai-Orchestrator` (`-1004390214071`) qua `evidence_run.py`.
- **Definition of Done (DoD):**
  - Mỗi Task có đúng 1 Sub-Chat Topic riêng biệt trên Telegram Supergroup `-1004390214071`; toàn bộ thông báo S3 Plan, tiến độ 10 Sub-Tasks, kết quả S5 Audit và hội thoại Co-Architect của Task đó đều nằm gọn trong Sub-Chat của nó.

### `TASK-CHAT-06` — Khắc phục Treo `Step 15/17 · 95%`, Lỗi `go build ./test` Không Có File `.go`, Vòng Lặp Retry Mù 3 Lần (`39 runs` & Rollback Oan `Circuit Breaker`), Tự Động Tạo `TASK-BUG-*` Để Tự Fix, Và Cơ Chế Core Nạp Ground-Truth Phản Biện Với `/laya_router`
- **Mô tả & Context `file:line`:**
  - `internal/webui/server.go:3297-3359` (`buildTargetedRepoCheckCommand`):
    - Bổ sung `resolveBuildableGoPackages(repoPath, candidateDir)`: kiểm tra thư mục có chứa trực tiếp file `.go` không (`hasDirectGoFiles`). Nếu thư mục như `centralized-data-service/test` chỉ chứa các thư mục con (`e2e/`, `integration/`, `manual/`) mà không có file `.go` trực tiếp, tự động lọc bỏ `./test` (hoặc chỉ lấy các thư mục con có file `.go` không phải `package main` xung đột), chấm dứt triệt để lỗi `no Go files in .../centralized-data-service/test` (`EV-20261026T163929-840e`).
  - `internal/webui/server.go:1929-2081` (`case "qa"`):
    - Sau `Step 15/17 · 95%`, bổ sung thực thi đầy đủ **`Step 16/17 · 98% (S6 Finalize, Audit Report & Governance Linter H7)`** và **`Step 17/17 · 100% (Task Complete & Clean Handoff)`**, đồng thời gọi `s.finishLiveStepProgress(taskKey)` để thanh tiến trình đạt `17/17 · 100%` và không bao giờ bị treo ở `15/17`.
  - `internal/webui/server.go:1284-1332` & `internal/webui/telegram_bridge.go:3110-3199` (`runTaskLifecycleActionWithDelay` & `diagnoseAndCreateS5BugSubTask`):
    - Xóa bỏ kiểu retry mù `for loop := 1; loop <= 3` vừa bắn tin nhắn Telegram đòi `Approve` vừa tự `continue` chạy lặp lại không sửa lỗi.
    - Khi `S5` phát hiện lỗi thật (`blocking > 0`), gọi `diagnoseAndCreateS5BugSubTask(ctx, qt, loop, maxLoops, qaLogs, missingIDs)`:
      1. Phân tích chính xác `qaLogs` / biên lai `EV-...` để xác định nguyên nhân gốc rễ, package/file bị lỗi và hành động khắc phục.
      2. Tự động khởi tạo một Sub-Task sửa lỗi tường minh **`TASK-BUG-01`, `TASK-BUG-02`, `TASK-BUG-03`, `TASK-BUG-04`...** trong `subtasks.json`, `08_tasks.md` và `09_tasks_solution.md` với đầy đủ `[Triệu chứng EV-... + Context file:line + Nguyên nhân + Hành động Fix]`, in rõ ra `terminal.log` và Sub-Chat Telegram (**không bắn thông báo đòi User Approve giả trong lúc đang tự động sửa lỗi**).
      3. Tự động thực thi hành động sửa lỗi thật cho `TASK-BUG-XX` ở `S4` (áp lại hunk thiếu, loại bỏ package rỗng không có `.go`, đồng bộ code) rồi chạy lại `S5` để nghiệm thu `TASK-BUG-XX -> DONE`.
      4. **Persistent Until-Done Loop (Không dừng ở 3 lần nếu vẫn tìm ra `TASK-BUG-*`):** Nếu sau 3 lần retry mà hệ thống vẫn phát hiện và bóc tách được `TASK-BUG-XX` mới có phương án khắc phục (hoặc chữ ký lỗi `bugSignature` đã tiến triển/giảm số lượng lỗi so với vòng trước), hệ thống **BẮT BUỘC reset `sameBugStallCount = 0` và TIẾP TỤC RETRY (`continue`)** cho tới khi sạch bóng bug (`qaBlocking == 0`) và hoàn thành 100% Task (`Step 17/17 · DONE`)! Tuyệt đối không dừng ngang ở lần thứ 3 và không tự nổ `Circuit Breaker` rollback oan các Sub-Task đã `DONE` (`TASK-M04, TASK-M05`).
    - Bổ sung `deduplicateAutopilotRounds` để dọn sạch các bản ghi `Human Gate` / `Build · S4` / `Build · S5` bị lặp 39 lần, và khôi phục `TASK-M04, TASK-M05` về trạng thái sạch.
  - `internal/webui/server.go` & `internal/webui/task_inspector.go` (`debateAndResolveLayaWithGroundTruth`):
    - Thực thi đúng chuẩn `/agent` Core (`GEMINI.md` Rule #9.8d-e & Rule #10): khi `/laya_router` (`:8200`) trả về `FLAG`, `UNVALIDATED` hoặc điểm thấp nhưng Core Agent kiểm chứng thấy Ground Truth vật lý (`VerifySubTaskCodeOnDisk == PASS` + biên lai compiler `EV-... exit=0` + `REF: file:line` khớp trên đĩa), Core Agent tự động nạp Full Context (`REF: file:line` + `EV-...` + `git diff`) để tranh biện/phản biện lại Laya và ghi quyết định `LAYA-DEBATE-RESOLVED` vào `04_decisions.md` + `audit_report.md`.
- **Logs / Test:**
  - Unit & Integration test `TestS5BugSubTaskAutoCreation_NoBlindRetrySpam_AndLayaGroundTruthDebate` (kiểm chứng cả kịch bản >3 vòng `TASK-BUG-01 .. TASK-BUG-04` vẫn tiếp tục retry cho tới khi `DONE 100%`) trong `internal/webui/server_test.go` + kiểm chứng chạy thật `qa` cho `tích hợp Master MongoDB Destination` đạt `Step 17/17 · 100% PASS` qua `evidence_run.py`.
- **Definition of Done (DoD):**
  - `tích hợp Master MongoDB Destination` không còn lỗi `./test`, chạy mượt qua `Step 16/17 -> Step 17/17 (100% DONE)`, lịch sử Run sạch đẹp không lặp `39 runs`, có cơ chế tự tạo `TASK-BUG-*` tường minh và **tiếp tục retry vượt qua mốc 3 lần miễn là còn tìm ra `TASK-BUG-*` để làm cho xong 100% Task**, đồng thời Core Agent chủ động nạp Ground Truth tranh biện với `/laya_router`.

### `TASK-CHAT-07` — Tự Động Tạo Sub-Topic Telegram Theo `Mã Task / Ticket ID` (`ExternalID`) Ngay Khi Tạo Task Mới (`handleIntake`) & Nâng Cấp Bộ Sinh Giải Pháp `S3` Đa Tầng Kiến Trúc Sâu Sắc + Trình Bày Trực Quan Cho `Core-01`
- **Mô tả & Context `file:line`:**
  - `internal/webui/server.go:827-960` (`handleIntake`) & `internal/webui/telegram_bridge.go:500-575`:
    - Khi tạo Task mới qua `POST /api/intake` (hoặc khi chạy `plan`), gọi ngay `s.ensureTelegramTaskTopic(ctx, cfg, qt)` với tên Sub-Topic định dạng theo **`Mã Task / Ticket ID` (`qt.ExternalID`)**: `🧩 [<ExternalID>] <Title>` (ví dụ: `🧩 [Core-01] Cập nhật và Nâng cấp chính mày trong ai-orchestrator từ api cho tới ux,ui`), lưu `message_thread_id` vào `.orchestrator/telegram_settings.json` (`TaskThreadIDs[qt.ExternalID]`) + `.orchestrator/tasks/<SafeTaskKey>/telegram_topic.json`, và gửi ngay bản tin khởi tạo Task + Hồ sơ `S3 Solution` vào đúng Sub-Topic đó.
  - `internal/webui/task_inspector.go:1064-1171, 3248-3330, 3421-3658, 3714-3792`:
    - Mở rộng `inspectTargetRepoSymbols` (`L3274`) quét thêm `.html`, `.css`, `.js` (`internal/webui/static/index.html`, `app.css`, `app.js`); phân loại và chấm điểm điểm chạm theo **6 tầng kiến trúc (`API`, `Planner/Inspector`, `Co-Architect Chat`, `Telegram Forum Bridge`, `Frontend UX/UI`, `E2E/Race Tests`)**, xóa bỏ điểm cộng cào bằng `+4` khiến mọi task rơi vào `internal/agentgate/gate.go:33` và `kernel_bridge.go:22`.
    - Sửa `DecomposeMultiRepoCodebaseSubTasks` (`L1076-1157`): tạo prefix mã Sub-Task sạch từ `qt.ExternalID` (`buildCleanSubTaskIDPrefix(qt.ExternalID, qt.Title)` -> `CORE-01-S01 .. CORE-01-S06`, chấm dứt lỗi cắt byte UTF-8 `TASK-CẬ01`), và phân rã yêu cầu nâng cấp full-stack (`từ api cho tới ux,ui`) thành **6 Sub-Task chuyên sâu theo đúng 6 tầng kiến trúc (`Wave 0 -> Wave 1 -> Wave 2`)** thay vì 2 Sub-Task rỗng tuếch lặp lại tiêu đề.
    - Tái cấu trúc trình bày `09_tasks_solution.md` (`L3421-3658`) & `buildConcreteCodeDemoForTask` (`L3714-3792`): trình bày gọn gàng bằng **Bảng Tóm tắt Điều hành**, **Ma trận Điểm nghẽn Hiện tại (`file:line`) -> Giải pháp Nâng cấp theo từng Layer**, **Lộ trình Sub-Tasks theo Wave**, và **Code Demo thực chất cho từng tầng (API, Deep Planner, Telegram Topic, WebUI UX/UI)**, đồng thời tái sinh lại toàn bộ hồ sơ `S3` (`01_requirements.md`, `08_tasks.md`, `09_tasks_solution.md`, `subtasks.json`) cho `Core-01`.
- **Logs / Test:**
  - Unit & Integration test `TestHandleIntakeCreatesTelegramTopicByExternalID_AndDeepMultiLayerS3PlanForCore01` trong `internal/webui/server_test.go` + kiểm chứng tái sinh hồ sơ `Core-01/09_tasks_solution.md` và tạo Sub-Topic `🧩 [Core-01] ...` trên Telegram qua `evidence_run.py`.
- **Definition of Done (DoD):**
  - Tạo Task mới có `ExternalID` lập tức tạo Sub-Topic Telegram mang mã `[ExternalID]` (và `Core-01` có ngay Sub-Topic riêng trên Telegram); hồ sơ `Core-01/09_tasks_solution.md` được trình bày mạch lạc, dễ đọc, phân rã sâu 6 tầng từ API tới UX/UI với `file:line` và Code Demo thực chất (không còn `TASK-CẬ01` hay dump lặp struct `gate.go:33`).

### `TASK-CHAT-08` — Vòng Lặp Tự Chữa Lành `S5 -> S3 -> S4 -> S5` (Rule #9.8d): Tách File/Module Đơn Trách Nhiệm Đúng Layer Trên `ai-orchestrator` (Cấm Nhồi Nhét File Lớn), Xóa Bỏ `httptest` Mockup, Chữa Lỗi Hardcode Verdict & Bảo Vệ Tuyệt Đối Ranh Giới Repo Ngoài (`data-hub`)
- **Mô tả & Context `file:line`:**
  - **1. Tách module đơn trách nhiệm trên `ai-orchestrator` (đưa toàn bộ các file lớn `store.go`, `server.go`, `task_chat.go`, `task_inspector.go`, `telegram_bridge.go`, `server_test.go`, `app.js`, `app.css`, `index.html` về `<= HEAD`):**
    - `internal/queue/store_rounds_dedup.go` (tách từ `internal/queue/store.go:418-514`): Chứa `DeduplicateTaskRounds` và helper kiểm tra trùng lặp round liên tiếp.
    - `internal/agentgate/laya_ground_truth.go` (chuyển từ `internal/webui/server.go` về đúng layer `internal/agentgate`): Chứa `ResolveFlaggedLayaClaimsWithGroundTruth` gọi CLI thật `laya_gate.py resolve` cho các claim `FLAG` có biên lai `EV-...` & `REF: file:line` (tuyệt đối cấm ghi đè trực tiếp `doc["gate"] = "PASS"` trong JSON).
    - `internal/webui/task_self_heal_loop.go` (tách từ `internal/webui/server.go`): Chứa `shouldContinuePersistentBugFixLoop`, `diagnoseAndCreateS5BugSubTask` (xóa bỏ hardcode `"ESCALATE"` và `len(auditStr)%97`, dùng chữ ký lỗi tất định và khôi phục `AtomicRollbackWaveSubTasks` khi thật sự kẹt `>= maxSelfHealLoops` trên cùng chữ ký lỗi), `runAutonomousS4S5WithSelfHealing`, `hasDirectGoFiles`, `buildTargetedRepoCheckCommand`, `finishLiveStepProgress`.
    - `internal/webui/task_chat_knowledge.go` (tách từ `internal/webui/task_chat.go`): Chứa `renderContextualDiscussReply` và `extractTaskDeepKnowledge`.
    - `internal/webui/telegram_forum_topic.go` (tách từ `internal/webui/telegram_bridge.go` và xóa bỏ biến toàn cục `telegramBotAPIBaseURL` dùng cho mockup): Chứa `formatTelegramTaskTopicName`, `ensureForumTopicForTask`, `notifyTelegramNewTaskIntakeAndTopic`, `resolveTaskThreadID`, `findTaskKeyByThreadID`. Đồng thời tại `internal/webui/telegram_bridge.go:131`, `internal/webui/static/index.html:994` và `internal/webui/static/app.js:3250,3279`, xóa bỏ hoàn toàn tên bot giả `@ai_orchestrator_s3_bot` / `@ai_orchestrator_bot`, đồng bộ về đúng Bot thực tế `@tk001sea_bot` đang cấu hình trong `.orchestrator/telegram_settings.json:6`.
    - `internal/webui/task_planner_multilayer.go` & `internal/webui/subtask_wave_engine.go` (tách từ `internal/webui/task_inspector.go`): Chứa bộ phân rã đa tầng `Core-01` (`decomposeSelfUpgradeOrchestratorSubTasks`, `buildSelfUpgradeOrchestratorCodeDemo`) và máy thực thi 12-Step / 6-Wave Sub-Task (`runSubTaskLifecycleActionWithDelay`).
    - `internal/webui/server_test.go`: Xóa bỏ bài test mockup `TestUnifiedUpgrade_TASK_CHAT_01_To_07_And_Core01_E2E` dùng `tgMock := httptest.NewServer(...)` và `fakeRepo := t.TempDir()`.
  - **2. Bảo vệ tuyệt đối ranh giới Repository ngoài (`data-hub` — Lesson `L001 [2026-10-27]`):**
    - Trong `VerifySubTaskCodeOnDisk` (`internal/webui/task_inspector.go:1520`): Bổ sung chốt chặn `isIsolatedWorktree := strings.Contains(filepath.ToSlash(repoDir), "/.orchestrator/worktrees/")` để **tuyệt đối cấm** tự động chạy `applyApprovedSubTaskPatchToTargetRepo` lên nhánh làm việc chính của các repository bên ngoài (`/Users/trainguyen/Documents/work/data-hub/{centralized-data-service,cdc-cms-service,cdc-cms-web}`). Đảm bảo `git status --short` trên cả 3 repo `data-hub` sạch 100%.
- **Logs / Test:**
  - Kiểm chứng biên dịch và unit/integration test thật qua `evidence_run.py` trên `ai-orchestrator`, kiểm tra `wc -l` toàn bộ file lớn `<= HEAD`, và kiểm tra `git status --short` sạch 100% trên `data-hub`.
- **Definition of Done (DoD):**
  - Không có bất kỳ file lớn nào bị nhồi nhét (`NOW <= HEAD`), đúng layer Design Pattern gốc, không dùng `httptest` mockup làm bằng chứng, xóa sạch tên bot giả `@ai_orchestrator_s3_bot` về đúng `@tk001sea_bot`, và tuyệt đối không chạm vào mã nguồn `data-hub`.

### `TASK-CHAT-09` — Quy Chuẩn Một Định Dạng Trình Bày & Nội Dung Duy Nhất (`UDF-v1`) Cho `#sri-markdown-content` Và `.cas-messages-list` Theo Đúng Kiến Trúc Đơn Trách Nhiệm
- **Mô tả & Context `file:line`:**
  - **1. Tầng Backend — Module đơn trách nhiệm `internal/webui/unified_doc_format.go` (kết nối vào `internal/webui/task_inspector.go` và `internal/webui/task_chat_knowledge.go`):**
    - Quy chuẩn toàn bộ nội dung sinh ra cho các báo cáo `S1, S2, DG0, S3, S4, S5` (`#sri-markdown-content`) VÀ câu trả lời đối thoại Co-Architect (`.cas-messages-list`) về **ĐÚNG 1 FORMAT 4 MỤC (`UDF-v1`)**:
      - **Thanh Metadata Đầu Trang (Header Strip):** `> 📌 **Task:** \`<TaskKey>\` · **Stage:** \`<Stage>\` · **Repos:** \`<RepoSummary>\` · **Model:** \`<Model>\` · **Gate:** \`<GateVerdict>\``
      - **`## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)`**: Tóm tắt ngắn gọn 3–5 ý cốt lõi kèm `file:line` (loại bỏ hoàn toàn tình trạng lặp lại danh sách Sub-Tasks 3 lần liên tiếp trong cùng một file báo cáo).
      - **`## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`file:line` & `DoD`)`**: Duy nhất 1 bảng Markdown chuẩn 5 cột (`| Mã | Wave / Tầng | Repo | Điểm chạm (file:line) | Giải pháp & Nghiệm thu (DoD) |`).
      - **`## 3. 💻 Code Demo / Diff Thực Thi (`Concrete Implementation`)`**: Khối code mẫu hoặc git diff trọng tâm theo từng tầng.
      - **`## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)`**: Trạng thái cổng `DG0..DG3 (:8200)`, `Clean-Room Approved Spec` (`REF: file:line`), và hướng dẫn thao tác kế tiếp.
  - **2. Tầng Frontend — Module đơn trách nhiệm `internal/webui/static/unified_doc_format.js` & `internal/webui/static/unified_doc_format.css` (nhúng vào `index.html`, gọi từ `app.js` mà giữ `app.js`, `app.css`, `index.html` `<= HEAD`):**
    - Thay thế việc đổ raw text (`textContent`) vào `<pre id="sri-markdown-content">` (`10.5px` monospace) và `<pre class="cas-msg-body">` (`10px` monospace) bằng bộ render cấu trúc `renderUnifiedDocFormat(rawMarkdown, opts)`:
      - Tự động bóc YAML frontmatter (`--- ... ---`) thành thanh **Metadata Badges** gọn gàng thay vì in thô 20 dòng đầu file.
      - Đóng khung từng mục `## 1..4` / `### 1..4` thành các **Section Cards (`.udf-section-card`)** phân biệt màu sắc theo ngữ nghĩa, font chữ đọc tài liệu `13px` (`line-height: 1.6`), độ tương phản cao.
      - Chuyển bảng Markdown `| ... |` thành **Bảng HTML chuẩn (`.udf-table`)** căn cột thẳng hàng, có badge cho `Sub-Task ID`, `Wave`, `Repo` và `file:line`.
      - Chuyển khối code/diff thành **Code Box (`.udf-code-box`)** có header ngôn ngữ, nút Copy và tô màu dòng diff (`+` xanh lá, `-` đỏ, `@@` xanh cyan).
      - Mở rộng `.cas-messages-list` (`max-height: 460px`) và render toàn bộ tin nhắn Co-Architect bằng chính `.udf-doc-root` để đồng nhất 100% giao diện với `#sri-markdown-content`.
- **Logs / Test:**
  - Kiểm chứng qua `evidence_run.py`: `go vet ./...`, `go build -o bin/orchestrator ./cmd/orchestrator`, `go test -v -race ./internal/webui/...`, khởi động lại server `:8090` và kiểm tra HTTP response của `/api/dashboard?task_key=Core-01`, `/unified_doc_format.js`, `/unified_doc_format.css`.
- **Definition of Done (DoD):**
  - Cả `#sri-markdown-content` và `.cas-messages-list` cùng hiển thị theo đúng 1 format `UDF-v1` (Card 4 mục, Bảng HTML thẳng hàng, Metadata Badges, Code/Diff Box rõ nét), nội dung không còn bị lặp hay đổ chữ monospace thô, và toàn bộ các file lớn trong `ai-orchestrator` đều giữ số dòng `<= HEAD`.

### `TASK-CHAT-10` — Khắc Phục Triệt Để Lỗi `@tk001sea_bot` Không Phản Hồi Khi Nhập Cập Nhật Trên Supergroup (`ForceReply` + Định Tuyến Ưu Tiên Không Rơi Tin Nhắn) & Quy Chuẩn Hóa Toàn Bộ Tin Nhắn Telegram Thành 1 Thẻ Điều Hành Gọn Gàng (`internal/webui/telegram_message_format.go`)
- **Mô tả & Context `file:line`:**
  - **1. Tách module đơn trách nhiệm `internal/webui/telegram_message_format.go` (giữ `internal/webui/telegram_bridge.go` `<= HEAD 3,297` dòng):**
    - Xây dựng bộ định dạng chuẩn mực **Telegram Compact Executive Card (`<= 1,600 ký tự` / 1 tin nhắn duy nhất, 0 mảnh cắt `(Phần 1/2)`, 0 bảng pipe `| ... |`, 0 đường dẫn tuyệt đối `/Users/...`)**:
      - `formatTelegramCompactQueueMessage`: Hiển thị `/task` ngắn gọn theo từng Repo (`• *#ID TaskKey* — Trạng thái (done/total Sub-Tasks)` + mô tả 1 dòng + ghi chú rõ nếu đang bật chế độ sửa giải pháp), loại bỏ hoàn toàn việc nhồi nhét toàn bộ hồ sơ S3 vào trong `/task`.
      - `formatTelegramCompactApprovalCard`: Hiển thị Hồ Sơ Giải Pháp S3 (`VIEW_PLAN` / `notifyTelegramOnS3PlanReadyForce` / kết quả sau khi User gửi phản hồi `handleTelegramReviseSolutionCommand`) trong **1 tin nhắn duy nhất** gồm 4 mục rõ ràng:
        1. `✏️ Phản hồi vừa cập nhật` (nếu có — nổi bật ngay đầu thẻ).
        2. `🔍 1. Nguyên nhân chính` (tối đa 3 bullet ngắn gọn kèm basename `file:line`, lọc bỏ các dòng metadata `UDF-v1` như `- **Task:** ...`).
        3. `💡 2. Giải pháp & Lộ trình Wave` (tối đa 3 bullet giải pháp + gom nhóm Sub-Tasks gọn theo `Wave 0..N` + tối đa 8 dòng code trọng tâm).
        4. `📝 3. Files ảnh hưởng & Thao tác` (chỉ in basename `file.go`, hướng dẫn 2 dòng cực gọn).
      - `formatTelegramCompactSubTaskCard`: Nén gọn thẻ duyệt từng Sub-Task (`buildTelegramSubTaskReviewMessage`).
      - `formatTelegramCoArchitectReply`: Chuyển đổi câu trả lời từ `buildContextualDiscussReply` (`UDF-v1` có bảng `| ... |` và heading `##`) thành định dạng bullet Telegram sạch đẹp, dễ đọc trên điện thoại.
      - `sendTelegramPromptWithForceReply`: Gửi tin nhắn nhắc nhập phản hồi kèm **`reply_markup: {"force_reply": true, "selective": true, "input_field_placeholder": "..."}`** để ứng dụng Telegram tự động mở khung Reply vào `@tk001sea_bot` (vượt qua `can_read_all_group_messages: false` trên Supergroup `-1004390214071`).
  - **2. Sửa định tuyến nhận phản hồi trong `internal/webui/telegram_bridge.go:645-662, 836-898, 2052-2085, 2189-2267, 2885-2942`:**
    - `stripBotUsernameFromCommand` (`L645-662`): Cắt bỏ `@tk001sea_bot` (và `@<bot_username>`) ở mọi vị trí trong câu (kể cả tin nhắn không bắt đầu bằng `/`).
    - `handleTelegramViewPlanCommand` (`L2201-2206`): Khi User bấm `👁 Xem Plan (#ID)`, cập nhật `LastNotifiedTask = qt.ExternalID` và nếu `PendingRevisionTaskKey` đang trỏ vào task cũ khác thì đồng bộ sang `qt.ExternalID` để phản hồi tiếp theo áp dụng đúng vào task đang xem.
    - `handleTelegramRevisePromptCommand` (`L2240-2266`) & `handleTelegramReviseSubPromptCommand` (`L2052-2085`): Gửi thẻ có nút `[ ❌ Hủy sửa ]` VÀ bắn kèm câu nhắc `ForceReply` (`force_reply: true`) để Telegram tự bật ô Reply vào `@tk001sea_bot`.
    - `processIncomingTelegramCommandWithReply` (`L876-898`): Với mọi tin nhắn không phải lệnh slash (`!strings.HasPrefix(cleanText, "/")`), xác định Task mục tiêu theo thứ tự ưu tiên: `replyRef (tin nhắn được Reply)` -> `explicitTaskKey (Sub-Topic)` -> `PendingRevisionTaskKey` -> `LastNotifiedTask` -> `selectedTaskKey`, và định tuyến ngay vào `handleTelegramTechnicalQuestion` (nếu là câu hỏi) hoặc `handleTelegramReviseSolutionCommand` (nếu là phản hồi cập nhật giải pháp), **tuyệt đối không bao giờ bỏ qua (`IGNORED`)**.
    - `ensureTelegramApprovePoller` (`L2885-2890`): Khi cập nhật `LastUpdateID`, nạp lại `latestCfg := s.loadTelegramPersistedConfig()` trước khi lưu để không bao giờ ghi đè làm mất `PendingRevisionTaskKey` / `LastNotifiedTask` vừa được cập nhật ở bước trước.
- **Logs / Test:**
  - Kiểm chứng qua `evidence_run.py`: `go vet ./...`, `go test -v -race -run TestTelegram ./internal/webui`, `go build -o bin/orchestrator ./cmd/orchestrator`, khởi động lại `:8090` và kiểm chứng gửi/nhận phản hồi + định dạng thẻ gọn đẹp trên Telegram.
- **Definition of Done (DoD):**
  - Khi User bấm `✏️ Phản hồi / Sửa Giải Pháp` hoặc Reply / nhắn phản hồi trên nhóm `Ai-Orchestrator` (`-1004390214071`), `@tk001sea_bot` tự bật khung Reply (`ForceReply`), nhận và trả lời ngay lập tức bằng 1 thẻ Giải Pháp S3 đã cập nhật gọn gàng (`<= 1,600 ký tự`, không vỡ bảng, không tách nhiều mảnh `(Phần 1/2)`), và `telegram_bridge.go` giữ nguyên `<= HEAD 3,297` dòng.

### [x] `TASK-CHAT-11`: Chuẩn Hóa Cổng `H4b (Propose-Only Approval Gate)` Trên `@tk001sea_bot` & Bộ Sinh `09_tasks_solution.md` / `08_tasks.md` Theo Chuẩn Engineering Manager / Tech Lead
- **Mô tả + Context (`file:line`):**
  - **1. Chuẩn hóa Relative Path & Khử trùng lặp danh sách File (`internal/webui/server.go:2771-2808` & `internal/webui/telegram_message_format.go:310-347`):**
    - Xây dựng hàm `normalizeAndDeduplicateRepoRelativePaths(rawPaths []string, repoNames []string) []string`: cắt sạch mọi tiền tố `/Users/.../`, `./`, `a/`, `b/` và tên repo (`centralized-export-service/`, v.v.), sau đó khử trùng lặp qua `map[string]bool` (case-insensitive) để trả về danh sách đường dẫn tương đối duy nhất (`start.ts`, `svc-env.ts`, `defined/app-setting.ts`, `dbHandler/postgres.ts`, `models/trans-his/trans-his.model.ts`, `test/unit/models/trans-his.model.test.ts` — đúng `6 files`, không bao giờ đếm vống lên `8 files`).
  - **2. Nhận diện động Tech Stack cho Code Fence & DoD (`internal/webui/task_llm_bridge.go:56-83` & `internal/webui/task_inspector.go:1133-1188`):**
    - Xây dựng hàm `detectRepoTechStackAndDoD(repoPath string, relFile string)`:
      - Nếu `repoPath` có `package.json` / `tsconfig.json` hoặc file `.ts`/`.js`: `StackLabel = "Node.js / TypeScript"`, `CodeLang = "typescript"`, `Wave0DoD = "npm run build & pass unit test"`, `Wave1DoD = "Pass toàn bộ test suite (npm test), không phát sinh memory leak/race condition, đạt chuẩn G1–G8"`.
      - Nếu `repoPath` có `go.mod` hoặc file `.go`: `StackLabel = "Go"`, `CodeLang = "go"`, `Wave0DoD = "go build ./... & pass unit test"`, `Wave1DoD = "Pass go test -race ./..., không phát sinh memory leak/race condition, đạt chuẩn G1–G8"`.
  - **3. Phân bổ Wave đúng thứ tự phụ thuộc `Wave 0 -> Wave 1` (`internal/webui/task_inspector.go:1133-1188`):**
    - Trong `DecomposeMultiRepoCodebaseSubTasks` (và nâng cấp tự động các `subtasks.json` cũ khi cả `M01` và `M02` đang cùng ở `Wave 0`): `M01` (Foundation & Model) đặt ở `WaveIndex = rIdx * 2` (`Wave 0`), còn `M02` (Routing & Integration Test) đặt ở `WaveIndex = rIdx*2 + 1` (`Wave 1`, `DependsOn: []string{id1}`).
  - **4. Áp dụng Template Chuẩn Cổng `H4b` 4 Mục (1 tin nhắn duy nhất, 0 Meta-Noise, 0 cắt cụt `...` ở định danh) cho cả `formatTelegramCompactApprovalCard` (`internal/webui/telegram_message_format.go:350-454`) và `BuildUnifiedS3SolutionDoc` (`internal/webui/unified_doc_format.go:206-254`):**
    - Render đúng cấu trúc:
      - Header: `⏸ *[H4b GATE · DUYỆT GIẢI PHÁP S3]* — <TaskKey> (#<ID>)`, `🎯 *Yêu cầu:* ...`, `📦 *Target Repo:* <repo> *(<StackLabel>)*`, `🌿 *Worktree / Branch:* .orchestrator/worktrees/<TaskKey> (<BranchName>)` (không cắt `...`).
      - `🏗 *1. KIẾN TRÚC & CHIẾN LƯỢC CHUYỂN ĐỔI (ARCHITECTURE PLAN)*`: `Pattern áp dụng`, `Điểm chạm chính (Core Touchpoints)`, `Cơ chế Rollback / Toggle`.
      - `⚠️ *2. KIỂM SOÁT RỦI RO (RISK & GUARDRAILS)*`: `Memory / Streaming`, `Tương thích Schema`.
      - `💻 *3. KEY DIFF / CODE PREVIEW (<primaryFile>)*`: Block code dùng đúng ````typescript` / ````go`, hiển thị hàm điều hướng/interface sạch (không nhồi God Class `pgPool` + `knexPg` + `dbConnection`).
      - `📋 *4. LỘ TRÌNH THỰC THI (<N> SUB-TASKS · <M> FILES)*`: Nhóm theo `Wave 0 (Foundation & Model)` -> `Wave 1 (Routing & Integration)` kèm DoD đúng Tech Stack, `📁 *Files thay đổi (<M>):*` (relative paths đã deduplicate, không cắt `...`), `📄 *Artifacts:* 09_tasks_solution.md | 08_tasks.md | diff.patch *(tại .orchestrator/tasks/<TaskKey>/)*`.
      - `⚡ *HÀNH ĐỘNG DUYỆT (CHỌN 1 TRONG 3):*` (`[⚡ Approve All]`, `[🧩 Step-by-Step]`, `[✏️ Revise (#<ID>)]`).
- **Logs / Test:**
  - Chạy `go test -v -race -run TestTelegram ./internal/webui`, `go vet ./...`, `go build -o bin/orchestrator ./cmd/orchestrator`, `go run ./cmd/orchestrator daemon --once` qua `evidence_run.py` và bắn thử nghiệm thẻ `H4b` mới của `TASK-13` (`#37`) & `Core-01` (`#43`) lên `@tk001sea_bot`.
- **Definition of Done (DoD):**
  - Thẻ `H4b` trên Telegram và file `09_tasks_solution.md` / `08_tasks.md` khớp 100% chuẩn Engineering Manager: đếm đúng `6 files` (0 trùng lặp), hiển thị đúng `typescript` & `npm run build` cho repo TS, tách chuẩn `Wave 0` -> `Wave 1`, có đầy đủ mục `Kiểm soát rủi ro` & `Cơ chế Rollback/Toggle`, không cắt cụt `...` ở branch/file/sub-task, nằm gọn trong 1 tin nhắn Telegram duy nhất, và mọi file lớn giữ nguyên `NOW <= HEAD`.

### [x] `TASK-CHAT-12`: Vòng Lặp Phản Tỉnh & Tự Khắc Phục `S5 -> S3 -> S4 -> S5` (Rule #9.8d): Chuẩn Hóa Đúng Layer Theo Design Pattern Gốc (`internal/gitclient`, `internal/notify`, `internal/webui/repo_tech_profile.go`, `internal/webui/task_revise_workflow.go`), Khóa Chặt `NOW <= HEAD` Trên 100% File Hiện Có, Xóa Lộ Token Trong `README_START.md`, Xóa `httptest.NewServer` Mockup & Đồng Bộ Văn Bản `pgPool` Thuần
- **Mô tả + Context (`file:line`):**
  - **1. Chuyển các hàm sang đúng Layer theo Design Pattern gốc của `ai-orchestrator`:**
    - `internal/gitclient/patch_guard.go`: Chuyển `CheckGoPackageCompilesOrRevertRelFile(ctx, repoDir, relPath)` từ `internal/webui/subtask_git_apply.go:182-206` về đúng layer `internal/gitclient`, đưa `internal/webui/subtask_git_apply.go` từ `344` dòng về `<= HEAD 312` dòng.
    - `internal/notify/telegram_format.go`: Chuyển `ConvertTelegramMarkdownToHTML`, `FormatInlineTelegramHTML`, `EscapeTelegramHTML`, `StripBotMentionAnywhere` từ `internal/webui/telegram_subtask_card.go:292-398` về đúng layer `internal/notify`.
    - `internal/webui/repo_tech_profile.go`: Gom và hợp nhất 1 bản duy nhất cho `RepoTechStackProfile`, `detectRepoTechStackProfile`, `detectTargetRepoLanguageAndDoD`, `detectCodeFenceLang`, `normalizeOrchestratorDisplayPath`, `normalizeAndDeduplicateRepoRelativePaths`, `normalizeRelativeRepoFiles`, `isSkippedBuildOrVendorDir`, `isSkippedGeneratedFile`, `isCompilerBoilerplateSymbol` (xóa bỏ trùng lặp giữa `telegram_message_format.go` và `task_llm_bridge.go`).
    - `internal/webui/task_revise_workflow.go`: Tách `executeArchitectReviseSolution`, `buildCleanRoomSpecSectionFromSubTasks`, `buildSubTaskModuleTitleAndDesc`, `isArchitecturalSteeringDirective` khỏi `internal/webui/task_llm_bridge.go` (giảm `task_llm_bridge.go` từ `546` dòng xuống `< 320` dòng) và xóa bỏ đoạn dead code chạm vào `ws09Path` (`memory/workspaces/`).
  - **2. Đưa 100% file hiện có trong repo về `NOW <= HEAD` & Xóa lộ URL Bot Token trong `README_START.md`:**
    - `README_START.md:306-309`: Xóa dòng URL chứa Telegram Bot Token thô ở cuối file (`309 -> 306` dòng, `DIFF = 0`).
    - `internal/webui/server_test.go`: Rút gọn 3 dòng trống thừa để đạt `<= HEAD 2,927` dòng.
    - `internal/runner/antigravity_test.go`: Đưa về `<= HEAD 90` dòng (chuyển test mới sang `internal/runner/antigravity_argon_test.go`).
    - `internal/webui/task_inspector.go` & `internal/webui/telegram_bridge.go`: Chuyển toàn bộ các hàm helper mới còn sót sang `internal/webui/task_planner_multilayer.go` và `internal/webui/telegram_forum_topic.go`.
  - **3. Xóa bỏ câu mẫu mâu thuẫn `Knex/PG Pool` & Chuẩn hóa cột `Điểm chạm (file:line)` + `TestDoD`:**
    - Tại `internal/webui/unified_doc_format.go:232, 258-260` và `internal/webui/telegram_message_format.go:255-261`: Cắt prefix `<repoName>/` ở cột `Điểm chạm (file:line)` và thay câu mẫu `"Khởi tạo Knex/PG Pool"` bằng mô tả chính xác kiến trúc `PostgresDB` (`pgPool` thuần cho `v2`, giữ nguyên Mongoose `TransHisModel` cho `v1`).
    - Tại `internal/webui/task_inspector.go:1145-1182`: Chuẩn hóa `TestDoD` cho `package.json` (`npm run build && npm test`) và `go.mod` (`go test -race ./...`).
  - **4. Xóa bỏ `httptest.NewServer` Mockup trong `internal/webui/telegram_message_format_test.go`:**
    - Gọi trực tiếp hàm xử lý thật trên `Server` + `queue.Store` SQLite thật và kiểm chứng trực tiếp trên `/Users/trainguyen/Documents/work/data-hub/centralized-export-service` (read-only) + `.orchestrator/tasks/TASK-13`.
- **Logs / Test:**
  - Chạy `go vet ./...`, `go test -v -race ./internal/gitclient/... ./internal/notify/... ./internal/runner/... ./internal/webui/...`, `go build -o bin/orchestrator ./cmd/orchestrator`, `go run ./cmd/orchestrator daemon --once` qua `evidence_run.py`, đối soát `wc -l` tự động và trace runtime `:8090`.
- **Definition of Done (DoD):**
  - 100% file hiện có trong repo đạt `NOW <= HEAD` (0 ngoại lệ), mọi module mới đặt đúng layer theo Design Pattern gốc (`internal/gitclient`, `internal/notify`, `internal/webui`) và `< 450` dòng/file, 0 lộ token trong `README_START.md`, 0 `httptest.NewServer` mockup trong test mới, vượt qua `DG3` (`Mode A + Mode B`) và `verify_governance.py`.

### [x] `TASK-CHAT-13`: Khắc Phục Triệt Để Lỗi Gửi Định Hướng Trên CMS Web (`#co-architect-studio`) Hiện Tin Nhắn User Rồi Treo `97s` Không Trả Lời (Xóa Vòng Lặp `needsDeepUpgrade` Trong `BuildDashboardResponse`, Tối Ưu `invokeArchitectLLM` Chạy Nhanh `< 3s` + Trả Lời Thẳng Trạng Thái Thực Thi Thật Của Task & Giữ `window._casInFlightChat` Tới Khi Có `agentEntry`) (`EV-20261027T163234-4a05`, `EV-20261027T163335-7c60`, `EV-20261027T163346-cf8b`)
- **Mô tả + Context (`file:line`):**
  - **1. Chấm dứt vòng lặp gọi lại `generateDeepTaskArtifacts` (`48s`) trong mọi lần `BuildDashboardResponse` (`internal/webui/task_inspector.go:3360-3375`):**
    - Tại `reconcileAllQueuedTasks` (`internal/webui/task_inspector.go:3362-3366`): Xóa điều kiện lỗi thời `!strings.Contains(planText, "Bảng Tóm Tắt Điều Hành")` (vốn luôn `true` trên `Core-01` khiến mọi lần `GET /api/dashboard` và cuối `POST /api/tasks/chat` đều gọi lại `generateDeepTaskArtifacts` -> `invokeArchitectLLM` treo `48s`). Chỉ chạy `generateDeepTaskArtifacts` khi file `09_tasks_solution.md` thực sự chưa tồn tại hoặc rỗng.
  - **2. Phản hồi tức thì (`< 1s`) & Trả lời trực tiếp đúng câu hỏi thực tế của User trong `renderContextualDiscussReply` & `invokeArchitectLLM` (`internal/webui/task_chat_knowledge.go:16-63` & `internal/webui/task_llm_bridge.go:92-235, 305-350`):**
    - Tại `invokeArchitectLLM` (`internal/webui/task_llm_bridge.go:173-235`): Chế độ `"discuss"` phản hồi tức thì (`< 50ms`) bằng dữ liệu thực tế trên đĩa; các chế độ khác đặt `agRunner.WorkDir = os.TempDir()`, truyền model nhanh `gemini-2.5-flash` và giới hạn timeout `10*time.Second`.
    - Tại `synthesizeProgressAwareDirectAnswer` & `synthesizeSymbolGroundedArchitectAnalysis` (`internal/webui/task_llm_bridge.go:108-145, 305-350`): Đọc trực tiếp trạng thái thực tế của Task (`Stage`, số lượng Sub-Tasks `DONE/PENDING`, danh sách điểm chạm file thực tế và `.orchestrator/tasks/<Key>/diff.patch`) để trả lời trực tiếp câu hỏi của User (như *"đã thật sự làm đc gì rồi"*, *"mày check git diff xem có file ko"*), đồng thời nâng cấp các tin nhắn cũ trong `chat_history.json`.
  - **3. Giữ `window._casInFlightChat` cho tới khi `agentEntry` thật sự xuất hiện (`internal/webui/static/app.js:1848-1882`):**
    - Lưu `prevLen: (dashboardState?.active_task_chat || []).length` trong `window._casInFlightChat`. Khi SSE `task_chat_user_sent` gọi `fetchDashboard()`, vẫn giữ bong bóng `thinkingEntry` cho tới khi `messages` có thêm `agent` reply mới ở sau `userEntry`.
- **Logs / Test:**
  - Chạy `go build -o bin/orchestrator ./cmd/orchestrator` (`EV-20261027T163234-53bb`), khởi động lại `:8090` và kiểm chứng trực tiếp `/api/dashboard?task_key=Core-01` (`EV-20261027T163335-7c60`, latency `< 0.5s`) + `TestArchitectLLMBridge_SkipDistAndBoilerplateAndRePlanV1V2` (`EV-20261027T163346-cf8b`) qua `evidence_run.py`.
- **Definition of Done (DoD):**
  - Khi gửi định hướng trên CMS Web (`#co-architect-studio`), tin nhắn User + bong bóng chờ hiện ngay ở `0ms` và câu trả lời của Agent hiển thị ngay sau `< 1s` (không còn treo `97s` và không cần bấm `F5`), trả lời đúng thực chất câu hỏi của User.

### [x] `TASK-CHAT-14`: Khắc Phục Triệt Để Khung Detail `#sri-markdown-content` Dưới `💡 S3 Solution & Code Demo (Gate Stop) (Core-01)` Full Height 100% Với Box Parent `.sri-markdown-pane` Mà Vẫn Giữ Nguyên Thanh Cuộn (`flex: 1 1 0; height: 0; overflow-y: auto`) (`EV-20261027T163234-4a05`, `EV-20261027T163335-7c60`)
- **Mô tả + Context (`file:line`):**
  - **1. Sửa `#sri-markdown-content` & `#sri-markdown-content.udf-doc-root` dùng `flex: 1 1 0; height: 0; min-height: 465px; overflow-y: auto` (`internal/webui/static/app.css:1859-1927` & `internal/webui/static/unified_doc_format.css:6-24`):**
    - Tại `internal/webui/static/app.css:1859-1877` và `internal/webui/static/unified_doc_format.css:6-24`: Tách `#sri-markdown-content.udf-doc-root` khỏi `.cas-msg-body.udf-doc-root` và đặt `flex: 1 1 0 !important; height: 0 !important; min-height: 465px !important; overflow-y: auto !important;` trên `#sri-markdown-content` để khung detail **luôn dãn full height 100% khớp khít với box parent `.sri-markdown-pane` (theo chiều cao của cột phải `.sri-side-pane`) NHƯNG VẪN GIỮ NGUYÊN THANH CUỘN DỌC (`overflow-y: auto`)**, không bị bung dài 3,000px làm mất scroll.
    - Khôi phục `max-height: 240px; overflow-y: auto;` trên `.sri-subtasks-list` (`app.css:1919-1926`) và `max-height: 180px; overflow-y: auto;` trên `.sri-code-block` (`app.css:1878-1890`) để cột phải giữ nguyên scroll gọn gàng.
    - Bổ sung `?v=20261027c` trong `internal/webui/static/index.html:7-8, 1026-1027` để chống trình duyệt cache CSS/JS cũ.
- **Logs / Test:**
  - Chạy `go build -o bin/orchestrator ./cmd/orchestrator` (`EV-20261027T163234-53bb`), khởi động lại `:8090` và kiểm chứng trực tiếp CSS qua `evidence_run.py` (`EV-20261027T163335-7c60`).
- **Definition of Done (DoD):**
  - Box detail `#sri-markdown-content` full height 100% với box parent `.sri-markdown-pane` và giữ nguyên thanh cuộn nội bộ (`overflow-y: auto`), không làm mất scroll.

### [x] `TASK-CHAT-15`: Rà Soát Thực Trạng Đấu Nối AI Model & Xóa Bỏ 5 Chốt Chặn/Làm Giả Model Bằng Ghép Chuỗi Tĩnh Trên Toàn Bộ `ai-orchestrator`
- **Mô tả + Context (`file:line`):**
  - **1. Bỏ điều kiện chặn AI Model ở chế độ `discuss` và truyền `git status --short` + `git diff --stat` thật vào Prompt (`internal/webui/task_llm_bridge.go:185-245, 308-352`):**
    - Tại `internal/webui/task_llm_bridge.go:185`: Xóa điều kiện `if actionMode != "discuss"` đang chặn 100% cuộc gọi `agRunner.Complete(...)` khi User chat hỏi đáp (`discuss`) và nhảy thẳng xuống dòng `244` gọi hàm ghép chuỗi tĩnh `synthesizeSymbolGroundedArchitectAnalysis` (`task_llm_bridge.go:308-352`).
    - Thu thập output thực tế của `git -C <repoPath> status --short` và `git -C <repoPath> diff --stat` đính kèm vào `prompt` của `invokeArchitectLLM` để Model trả lời dựa trên trạng thái git diff thật thay vì bốc 2 symbol đầu tiên (`cmd/orchestrator/agent_cmd.go:18`, `internal/gitclient/gitclient.go:14`).
  - **2. Ngừng ghi đè kết quả của AI Model bằng hàm ghép câu tĩnh trong `renderContextualDiscussReply` (`internal/webui/task_chat_knowledge.go:38-46, 301-313`):**
    - Tại `internal/webui/task_chat_knowledge.go:38-46`: Xóa đoạn lấy `fbRC`, `fbSol` (cạo từ markdown tĩnh) và `synthesizeProgressAwareDirectAnswer` (`task_llm_bridge.go:110-156`, dòng `150`: `"Dạ anh, đối với yêu cầu %q trên task %s: Nguyên nhân chính là %s; giải pháp thực thi chuẩn là %s"`) ghi đè lên `llmRes.DirectAnswer`, `llmRes.RootCauses`, `llmRes.Solutions` của AI Model.
  - **3. Xóa bỏ bộ sinh 6 Sub-Tasks `Core-01` hardcode trong Planner (`internal/webui/task_inspector.go:980-1012, 2858-2868, 3059-3061`, `internal/webui/task_planner_multilayer.go:12-172`, `internal/webui/task_revise_workflow.go:75-107`):**
    - Xóa `isSelfUpgradeOrchestratorTask` (`task_inspector.go:980-995`) và `decomposeSelfUpgradeOrchestratorSubTasks` (`task_planner_multilayer.go:12-150`) đang nhồi cứng 6 Sub-Tasks của `Core-01` (`Tầng 1..Tầng 6`) cho mọi task trên `ai-orchestrator`.
    - Sửa `applyLLMSubTaskSpecs` (`task_revise_workflow.go:75-107`) để thay thế toàn bộ danh sách `SubTasks` theo đúng mảng `analysis.SubTasks` do AI Model sinh ra (không giữ lại các slot thừa `S03..S06` từ template tĩnh).
  - **4. Đấu nối `invokeArchitectLLM` vào luồng `/revise` trên Telegram (`internal/webui/telegram_bridge.go:1814-1875, 2045, 2101-2175`):**
    - Sửa `handleTelegramReviseSolutionCommand` / `applyTelegramSolutionFeedbackToPlan` (`telegram_bridge.go:2045, 2101-2175`) và `handleTelegramReviseSubTaskSolutionCommand` (`telegram_bridge.go:1814-1875`) gọi trực tiếp `executeArchitectReviseSolution` / `invokeArchitectLLM` thay vì chỉ nối dòng text tĩnh vào cuối file `09_tasks_solution.md`.
  - **5. Đấu nối `ArchitectLLMAnalysis` vào `BuildUnifiedS2AnalysisDoc` & `BuildUnifiedDG0CritiqueDoc` (`internal/webui/unified_doc_format.go:99-204`):**
    - Truyền kết quả phân tích của AI Model (`llm_analysis.json`) vào `13_analysis.md` và `10_preplan_critique.md` thay vì ghép bảng tĩnh từ tên hàm.
- **Logs / Test:**
  - Kiểm chứng qua `evidence_run.py`: `go vet ./...`, `go test -v -race ./internal/webui/...`, `go build -o bin/orchestrator ./cmd/orchestrator`.
- **Definition of Done (DoD):**
  - 100% câu hỏi trên `#co-architect-studio` và Telegram gọi thẳng AI Model kèm `git diff` thật, không còn bất kỳ hàm ghép văn mẫu tĩnh nào ghi đè kết quả Model.
