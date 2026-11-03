---
tier: T2
suffix: audit_core_workflow_ex002_and_activity_log
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-27T14:17:30+07:00
status: IN_PROGRESS
---

# 01. Requirements — Audit Toàn Diện Core Workflow (`S0 -> S6`), Gốc Rễ Lệch Plan `EX-002` (`ClickHouse` -> `Postgres`), Lý Do Phản Biện `DG0`/`Laya Gate` Bị Qua Mặt & Khôi Phục `Activity-Log Progress`

## 1. Yêu cầu gốc từ User
> *"có khá nhiều nhầm lẫn khi chạy các bước, ở task ex002, tao kêi làm clickhouese, nhưng plan là làm postgress, dù đã có rất nhiều bước như phản biện, laya quyết định. mày audit lại vì sao các bước này ko được thực hiện. activity-log progress hiện tại đang ở đâu. sao ko tháy thông báo. mày audit lại toàn bộ cái core workflow của tao xem coi có chỗ nào làm láo."*

## 2. Danh sách Yêu cầu Cụ thể (`REQ-AUDIT-CW-01 .. REQ-AUDIT-CW-04`)

### `REQ-AUDIT-CW-01`: Giải phẫu chính xác vì sao Task `EX-002` (ClickHouse V3) bị lập Plan làm PostgreSQL V1/V2 & vì sao các bước Phản biện (`DG0`), Laya Decision Gate (`DG1/DG2/DG3`) không chặn lại
- Chỉ rõ bằng `file:line` trong `/Users/trainguyen/Documents/work/ai-orchestrator` và dữ liệu vật lý trong `.orchestrator/tasks/EX-002/`:
  1. Tại sao `DecomposeMultiRepoCodebaseSubTasks` (`internal/webui/task_inspector.go:1089-1198`) và `generateDeepTaskArtifacts` (`internal/webui/task_inspector.go:3078-3294`) hoàn toàn không gọi Model (`invokeArchitectLLM`) mà lại bị bẫy từ khóa `isDualVersionV1V2Directive` (`internal/webui/task_llm_bridge.go:189-195`) và `isPostgresOrDualSourceMigrationTask` (`internal/webui/telegram_message_format.go:135-142`) biến `EX-002` thành `TASK-TRAN01/02 (PostgreSQL)` + Code Demo `PostgresDB`.
  2. Tại sao bước phản biện `S2.5 (DG0)` (`BuildUnifiedDG0CritiqueDoc` tại `internal/webui/unified_doc_format.go:169-201`) chỉ in bảng tĩnh `"VERIFIED"` / `"PASS"` mà không hề kiểm tra xem các thực thể công nghệ trong đề bài (`ClickHouse`, `V3`) có mặt trong Plan hay không.
  3. Tại sao cổng quyết định Laya (`BuildAndRunCalibratedLayaGate` tại `internal/agentgate/kernel_bridge.go:637-923`) lại bị làm giả ("làm láo") bằng cách tự đọc lại chính dòng tiêu đề đầu tiên của file Markdown vừa sinh (`pickLine(0)[:80]`) để gán `Claim = q1`, `Evidence = q1`, `MustContain = [q1]` (tự trích dẫn chính mình), dẫn đến `06_laya_claims_dg0_ex_002.json` luôn `PASS` 100% dù Plan sai lệch hoàn toàn.

### `REQ-AUDIT-CW-02`: Truy vết vị trí hiện tại của `activity-log progress` (WebUI & Telegram) và lý do không thấy thông báo tiến trình chạy từng bước
- Chỉ rõ bằng `file:line`:
  1. `activity-log progress` hiện đang nằm ở đâu trên đĩa (`.orchestrator/tasks/EX-002/terminal.log`), trong bộ nhớ (`s.liveStepProgress` tại `internal/webui/server.go:154-166`), trên WebUI (`#global-running-progress-banner` tại `internal/webui/static/index.html:99` và `#live-task-terminal-pre` tại `internal/webui/static/index.html:462-472`), và trên Telegram (`internal/webui/telegram_bridge.go:2674-2771`).
  2. Tại sao User không thấy tiến trình chạy từng bước trên WebUI: `handleCreateTask` (`server.go:932-968`) chạy `generateDeepTaskArtifacts` không bật `liveStepProgress`, sau đó gọi `runTaskLifecycleAction` (`server.go:1122-1167`) với `stepDelay = 0` (`server.go:1123`) khiến toàn bộ các bước `Step 2/18 -> 7/18` chạy xong trong `< 1ms` (`13:56:34`) rồi bị `defer s.clearPacedWorkflowStep()` (`server.go:1167`) xóa sạch về `Active: false` trước khi trình duyệt kịp nhận phản hồi HTTP! Đồng thời khung `#live-task-terminal-pre` bị nhét tận dưới đáy màn hình (`index.html:464`).
  3. Tại sao User không thấy thông báo tiến trình (`activity-log progress`) trên Telegram: `advancePacedWorkflowStep` (`telegram_bridge.go:2674`) và `advanceSubTaskStepProgress` (`telegram_bridge.go:2723`) chỉ ghi vào RAM và file `terminal.log` cục bộ, **hoàn toàn không gửi hoặc `editMessageText` một thẻ tiến trình trực tiếp (`Live Activity-Log Progress`) vào Forum Sub-Topic của Task trên Telegram**.

### `REQ-AUDIT-CW-03`: Audit toàn bộ Core Workflow (`S0 -> S1 -> S2 -> S2.5 DG0 -> S3 + DG1/DG2 -> S4 -> S5 + DG3 -> S6`) để vạch trần mọi chỗ "làm láo"
- Kiểm tra không sót bất kỳ mắt xích nào từ `S0` tới `S6`, chỉ rõ từng hàm làm giả/hardcode:
  - `S0 Boot`: Không tự động khởi chạy Laya `:8200` (`ensure_laya_up.sh`) khi `laya_health_s0: false` mà rơi thẳng vào fallback offline (`kernel_bridge.go:900-922`).
  - `S1 Intake & Sub-Task Decomposition`: `DecomposeMultiRepoCodebaseSubTasks` (`task_inspector.go:1089`) không gọi LLM mà quét file có sẵn trên đĩa (`postgres.ts` được cộng điểm ưu tiên ở `task_inspector.go:2984-2989`) rồi gắn nhãn bằng hàm hardcode `buildSubTaskModuleTitleAndDesc` (`task_llm_bridge.go:197`).
  - `S2 Investigate & S2.5 DG0 Critique`: `BuildUnifiedDG0CritiqueDoc` (`unified_doc_format.go:169`) in bảng tĩnh `"VERIFIED"`; `BuildAndRunCalibratedLayaGate` (`kernel_bridge.go:649-723`) tự lấy dòng của chính file vừa ghi làm `Claim == Evidence == MustContain`.
  - `S3 Solution & Co-Architect Chat`: `generateDeepTaskArtifacts` (`task_inspector.go:3078`) không gọi `invokeArchitectLLM`; `BuildUnifiedS3SolutionDoc` (`unified_doc_format.go:253`) và `BuildUnifiedCoArchitectReply` (`unified_doc_format.go:344`) gọi `synthesizeDirectAnswerSummary` đè mất `llmRes.DirectAnswer` bằng văn mẫu `PostgresDB`; `invokeArchitectLLM` (`task_llm_bridge.go:279`) nhét luật cứng của `TASK-13 (PostgresDB)` vào `sysPrompt` chung.
  - `S4 Implement`: `server.go:1467-1663` và `subtask_wave_engine.go:245-276` hoàn toàn **không gọi LLM Runner (`AntigravityRunner` / `ClaudeRunner`) để viết code thật** cho task mới mà chỉ gọi `synthesizeIsolatedRemediationDiffIfApproved` (`telegram_bridge.go:2821`) chụp lại `git diff` cũ.
  - `S5 Audit & DG3`: `server.go:1788-1830` chỉ chạy `RunEvidenceCommand` nếu có `go.mod` (bỏ qua repo Node/TS như `centralized-export-service`), rồi chạy lệnh giả `test -f 01_requirements.md` (`server.go:1826`) để lấy biên lai `EV-...` giả! Tệ hơn, `server.go:1939` hardcode câu `"Multi-Worker & OCC Concurrency: Verified intra-batch _source_ts deduplication... thread-safe masterMongoPool"` vào `audit_report.md` của mọi task!

### `REQ-AUDIT-CW-04`: Thiết kế Giải pháp Duy nhất (`S3`) & Code Demo cụ thể để xóa sạch mọi chỗ "làm láo" và khôi phục `Activity-Log Progress`
- Tuân thủ nghiêm ngặt quy tắc **Single-Responsibility Module Split (`NOW <= HEAD`)** và thực thi khép kín `S5 -> S3 -> S4 -> S5` theo Rule #9.8d.

### `REQ-AUDIT-CW-05`: Giải phẫu & Khắc phục triệt để lỗi Task `core-003` (`audit lại`) bị trộn lẫn 4 Sub-Tasks hardcode của `Core-01` (`CORE-003-S03..S06`) và ảo giác `internal/agentgate/gate.go` (`CORE-003-S01..S02`)
- **Yêu cầu trực tiếp từ User:**
  > *"con chó này, mẹ mày . tao kêu làm cái task core repo ai-orchestrator. sao cái plan của mày là [S3 · UDF-v1] Giải Pháp Kỹ Thuật Duy Nhất & Code Demo — core-003: audit lại ... mẹ mày. mày có làm nãy giờ ko, sao bị lộn qua task này task kia, hồi hard code tùm lum vậy"*
- **Phạm vi bắt buộc:**
  1. Giải thích thẳng và trung thực 3 nguyên nhân gốc rễ (`file:line`) khiến `core-003` bị lai ghép giữa 2 Sub-Task ảo giác `gate.go` (`S01..S02`) và 4 Sub-Task hardcode của `Core-01` (`S03..S06`).
  2. Xóa bỏ hoàn toàn `isSelfUpgradeOrchestratorTask` (`internal/webui/task_inspector.go:980-1012`), `decomposeSelfUpgradeOrchestratorSubTasks` và `buildSelfUpgradeOrchestratorCodeDemo` (`internal/webui/task_planner_multilayer.go:12-172`).
  3. Sửa `applyLLMSubTaskSpecs` (`internal/webui/task_revise_workflow.go:75-107`) để xây dựng danh sách Sub-Tasks trực tiếp từ `analysis.SubTasks` (kèm `TestDoD` động theo `detectRepoTechStackProfile`), chấm dứt lỗi ghi đè một phần mảng template cũ.
  4. Sửa `invokeArchitectLLM` (`internal/webui/task_llm_bridge.go:171-206`) để bóc tách chỉ thị thực của User (sau dấu `=>` hoặc cuối đoạn code mẫu), không bị đánh lừa bởi đoạn snippet ví dụ User dán trong đề bài, và không giới hạn cứng `2 Sub-Tasks`.
  5. Xóa bỏ toàn bộ các ổ cheat code và lấy nhầm dữ liệu liên task còn sót lại (`readTaskOrSiblingDiffPatch` / `byTitle[normTitle]` trong `task_inspector.go:1334-1375, 3291-3345`; hardcode `"TASK-A04"` / `"AF-ORCH-01..06"` trong `task_chat.go:1034-1058, 1111, 1186`; hardcode `"feat-master-mongo-destination-2026-10-20"` trong `task_chat_knowledge.go:81-89`; hunk comment giả trong `subtask_codegen_muscle.go:434-451`).
  6. Tái sinh sạch sẽ toàn bộ hồ sơ của `core-003` (`.orchestrator/tasks/core-003/09_tasks_solution.md`, `08_tasks.md`, `subtasks.json`, `llm_analysis.json`, `chat_history.json`) trên đĩa và trên server `:8090`.

