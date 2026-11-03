---
tier: T2
suffix: fix_orchestrator_real_llm_bridge_and_scanner
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-27T09:58:00+07:00
---

# 01. Requirements — Kiểm định & Khắc phục Toàn diện Hệ thống `ai-orchestrator`: Đấu nối LLM (`gemini-4-argon` / `AntigravityRunner`), Nhận diện Chỉ đạo Kiến trúc (Re-Plan V1/V2), Chặn Quét `dist/` & Rò rỉ Template

## 1. Bối cảnh & Yêu cầu của User
User chỉ rõ 5 lỗi chí mạng của hệ thống `ai-orchestrator` (`/Users/trainguyen/Documents/work/ai-orchestrator`) khi xử lý câu chỉ đạo trong khung `💬 THẢO LUẬN KIẾN TRÚC` (`🧠 BRAIN · Architect · Slot A · /brain-delegate · gemini-4-argon (antigravity)`):
> *"khi tao hỏi : hiện tại mày đang làm là bỏ cái export type cũ đi, đè lên nó. giải pháp của anh là tạo 1 cái type v2 qua postgrest, còn v1 vẫn là cái cũ ... con chó này, tao kêu mày nối qua con 4 argon trả lời, mà mày nối đi đâu vậy."*
> *"ko đào sâu vào export, cái mày cần làm là kiểm tra lại hệ thống ai-orchestrator"*

### Ranh giới Phạm vi (Scope Boundaries — Lesson `L001`, `L010`)
- **Trong phạm vi (In-Scope 100%):** Chỉ kiểm tra và khắc phục hệ thống `ai-orchestrator` (`/Users/trainguyen/Documents/work/ai-orchestrator`).
- **Ngoài phạm vi (Out-of-Scope):** Tuyệt đối không đào sâu hay sửa sang mã nguồn của `centralized-export-service` hoặc `data-hub`.

## 2. Danh sách Yêu cầu Kỹ thuật (`REQ-ORCH-01` -> `REQ-ORCH-05`)

- **`REQ-ORCH-01` (Đấu nối thật `AntigravityRunner` / `gemini-4-argon` vào `internal/webui/`):**
  - Hiện tại toàn bộ `internal/webui/` (`task_chat.go`, `task_chat_knowledge.go`, `telegram_forum_topic.go`, `task_inspector.go`) có `0` lời gọi tới `runner.AntigravityRunner`, mà chỉ dùng `fmt.Sprintf` cắt ghép các dòng tĩnh.
  - Ngoài ra, `internal/runner/antigravity.go:113-155` thiếu `--skip-trust` và biến môi trường `GEMINI_CLI_TRUST_WORKSPACE=true`, `COLORTERM=truecolor`, `TERM=xterm-256color` (khiến `gemini -p` thoát với `exit code 55` ở thư mục chưa trust), chưa fallback tìm binary `gemini` trong `~/.nvm/versions/node/*/bin/gemini`, và chưa strip hậu tố `" (antigravity)"` khỏi tên model UI.
  - Phải nối thật `handleTaskPlanningChat`, `handleTelegramTechnicalQuestion`, và `revise_solution` / `S3` qua `AntigravityRunner.Complete(ctx, req)` kèm context thực tế của Task (yêu cầu mới của User + code thật của `TargetRepo` + `09_tasks_solution.md` + `subtasks.json`).

- **`REQ-ORCH-02` (Loại bỏ Râu ông nọ cắm cằm bà kia — Cross-Workspace Contamination & Regex Hallucination):**
  - `FindMatchingAgentWorkspace` (`internal/webui/task_inspector.go:868-935`) đang khớp từ khóa lỏng lẻo (`mongo`, `postgres`, `trans`) mà không khóa theo `TargetRepo`, dẫn tới lấy nhầm `09_tasks_solution.md` của workspace khác (`BSON document size limit > 16MB -> NATS/Kafka DLQ`).
  - `extractTaskDeepKnowledge` (`internal/webui/task_chat_knowledge.go:160-235`) gán nhầm biến trả về thứ 4 (`req01File` = `01_requirements_*.md`) vào `ws13` và dùng `strings.Contains(lowerLn, "lỗi")` bốc nhầm mục `Gate G2: Red -> Green Reproduction` (`VerifySubTaskCodeOnDisk`, `evidence_run.py`) làm "Nguyên nhân gốc rễ".

- **`REQ-ORCH-03` (Chặn tuyệt đối quét thư mục build `dist/`, `build/`, boilerplate `__awaiter` và sửa lỗi bọc cứng ````go`):**
  - `inspectTargetRepoSymbols` (`internal/webui/task_inspector.go:2955-3055`) thiếu `dist`, `build`, `out`, `.next`, `coverage`, `target` trong danh sách skip directory, khiến `dist/models/trans-his/trans-his.model.js:3` đứng đầu danh sách và hàm polyfill TypeScript `function adopt(value) { ... }` bị trích xuất làm điểm chạm kiêm Code Demo.
  - `task_chat_knowledge.go:103` và `task_inspector.go:2635, 3235, 3364` bọc cứng mọi đoạn code trong tag ````go` bất kể file `.ts`, `.js`, `.py`, `.sql`.

- **`REQ-ORCH-04` (Loại bỏ String Interpolation lặp từ máy móc khi phân rã Sub-Tasks):**
  - `DecomposeMultiRepoCodebaseSubTasks` (`internal/webui/task_inspector.go:1172-1199`) đang ghép chuỗi tĩnh `fmt.Sprintf("Triển khai kiến trúc & nghiệp vụ cốt lõi cho %s trên %s", qt.Title, repoName)`. Phải sinh mô tả Sub-Task dựa trên yêu cầu kiến trúc thực tế và symbol nguồn sạch (`.ts`/`.go`, không bao giờ trỏ vào `dist/`).

- **`REQ-ORCH-05` (Nhận diện Chỉ đạo Đổi hướng Kiến trúc & Re-Open / Re-Plan thật sự thay vì báo `DONE` ảo):**
  - `classifyChatActionMode` (`internal/webui/task_chat.go:1019-1041`) đang bỏ lọt các câu chỉ đạo kiến trúc tự nhiên của User (ví dụ: *"hiện tại mày đang làm là bỏ cái export type cũ đi, đè lên nó. giải pháp của anh là tạo 1 cái type v2 qua postgrest, còn v1 vẫn là cái cũ"*), xếp nhầm vào `"discuss"` và in trạng thái `Stage DONE | 2/2 DONE (100% DONE)`.
  - Khi User chỉ đạo đổi hướng giải pháp/kiến trúc, hệ thống phải phân loại vào `"revise_solution"`, tự động **re-open Task từ `DONE` về `NEEDS_YOU`**, xóa `.user-approved`, reset Sub-Tasks khỏi trạng thái `DONE` ảo, gọi LLM (`gemini-4-argon` qua `AntigravityRunner`) để tái thiết kế `09_tasks_solution.md` & `subtasks.json` theo đúng kiến trúc mới của User, và hiển thị trạng thái chờ duyệt (`NEEDS_YOU`) trung thực.

- **`REQ-ORCH-06` (Chấm dứt "Cross-Task Contamination" từ `agent/memory/workspaces/` sang `.orchestrator/tasks/<task_id>` như `cdc-01`):**
  - Trong `cdc-01` (*"kiểm tra cho tao vì sao transmute bị thiếu"*), User dán bảng trạng thái có chữ `[mongodb] traitestpbs` và `master_payment_bill_service`. Do `ResolveTaskAffectedRepos` (`task_inspector.go:569`) và `FindMatchingAgentWorkspace` (`task_inspector.go:877`) kiểm tra `strings.Contains(lower, "mongo") && strings.Contains(lower, "master")`, hệ thống đã bốc nguyên `08_tasks_master_mongo.md` (`TASK-M01..M10`) và `09_tasks_solution_master_mongo.md` từ `feat-master-mongo-destination-2026-10-20` đè thẳng vào `.orchestrator/tasks/cdc-01/`!
  - **Yêu cầu:** Cách ly tuyệt đối `.orchestrator/tasks/<task_id>` khỏi `agent/memory/workspaces/` đối với mọi Task mới (`cdc-01`, v.v.) — cấm copy/seed `08_tasks_*.md` / `09_tasks_solution_*.md` từ `memory/workspaces/` sang `.orchestrator/tasks/<task_id>`.

- **`REQ-ORCH-07` (Xóa bỏ Regex String-Prefixing `"- **Phương án khắc phục:** Workspace**: ..."` trong `/brain-delegate` & Trả lời đúng 5 điểm nghẽn Transmute cho `cdc-01`):**
  - Trong `renderContextualDiscussReply` (`task_chat_knowledge.go:70-88`), hệ thống chỉ gọi `invokeArchitectLLM` khi `len(rootCauses) == 0` hoặc khớp `isArchitecturalSteeringDirective`, dẫn tới khi User hỏi/chửi trên `cdc-01`, nó không gọi LLM mà lấy các dòng cắt từ `09_tasks_solution.md` (`Workspace**: feat-master-mongo-destination-2026-10-20`, `Suffix**: master_mongo`) rồi ghép cứng `"- **Phương án khắc phục:** "` tại `unified_doc_format.go:308`!
  - **Yêu cầu:** Xóa bỏ hoàn toàn việc cắt dòng metadata gắn chữ `"Phương án khắc phục:"`, luôn gọi `invokeArchitectLLM` cho mọi tin nhắn trong `/brain-delegate`, và làm sạch hồ sơ + lịch sử chat của `cdc-01` với đúng 5 điểm nghẽn làm hụt bản ghi khi Transmute từ Shadow sang Master (`transmuter.go` & `transmute_handler.go`).

- **`REQ-ORCH-08` (Khóa chặt `selectedChainKey` / `task_key` để khi đang làm `cdc-01` không nhảy sang Plan của `TASK-13`):**
  - Trong `static/app.js:52-66` (`fetchDashboard`), khi SSE nhận sự kiện `update` (`app.js:90-92`), `fetchDashboard()` gọi `/api/dashboard` không kèm `?task_key=${selectedChainKey}`, dẫn tới lấy `s.selectedTaskKey` toàn cục trên server (đang bị đổi sang `TASK-13`) đè lên màn hình của User. Phải luôn giữ `selectedChainKey` của client khi refresh dashboard.
- **`REQ-ORCH-08` — Bắt buộc có 1 đoạn Summary ngắn trả lời trực tiếp câu hỏi của User ở ngay đầu phản hồi trước khi vào nội dung chính (`Direct-Answer Summary First`):**
  - Khi User đặt câu hỏi và khi Agent / Co-Architect Bot (`/brain-delegate` trong `ai-orchestrator`) phản hồi, **luôn luôn mở đầu bằng 1 đoạn Summary ngắn (2–4 câu) trả lời trực tiếp câu hỏi của User**, xong mới tới nội dung phân tích chính (Nguyên nhân gốc rễ, Giải pháp, Code Demo, Kiểm định cổng).
  - Trong `ai-orchestrator`: nạp trường `DirectAnswer` (`json:"direct_answer"`) từ `ArchitectLLMAnalysis` (`internal/webui/task_llm_bridge.go:23`) vào đầu `BuildUnifiedCoArchitectReply` (`internal/webui/unified_doc_format.go:293-304`) và đầu `formatTelegramCoArchitectReply` (`internal/webui/telegram_message_format.go:530-570`).
- **`REQ-ORCH-09` — Chuẩn hóa 4 điểm trong Bộ Formatter & Prompt Sinh Nội Dung của `ai-orchestrator`:**
  1. **Deduplicate & Relative Path cho danh sách File:** Cắt bỏ prefix tên repo (`centralized-export-service/`) giữ lại đường dẫn tương đối (`models/trans-his/trans-his.model.ts`) và đưa vào `Set` để loại bỏ 100% file trùng lặp.
  2. **Dynamic Language Detection cho DoD & Code Block:** Kiểm tra sự tồn tại của `package.json` vs `go.mod` trong target repo (`package.json` -> ````typescript` + DoD `npm run build && npm test`; `go.mod` -> ````go` + DoD `go test -race ./...`).
  3. **Xóa bỏ câu chữ giải thích cơ chế hệ thống (Meta-System Noise):** Cắt bỏ hoàn toàn các câu *"Đã quét trực tiếp ... file mã nguồn..."*, *"Mọi thay đổi ở bước S4 được cô lập trong Git Worktree theo Rule #12, #13, #19..."* và hướng dẫn bấm nút dài dòng để 100% tin nhắn nằm gọn trong 1 message duy nhất (`< 4096` ký tự).
  4. **Cấm cắt chuỗi bằng `...` (`truncate`) ở các trường định danh:** Không cắt cụt tên nhánh git (`branch`), tên file, lệnh `DoD` và nội dung tóm tắt Sub-task ở mục 3. Đồng thời làm sạch Code Demo của `TASK-13` (tách riêng handler PostgreSQL ra khỏi Mongoose `TransHisModel`, không trộn lẫn `pgPool`, `knexPg` và `dbConnection` Mongo).
- **`REQ-ORCH-10` — Xóa bỏ 100% Hardcode `Postgres/V1-V2` & Đấu nối Model (`invokeArchitectLLM` / `AntigravityRunner`) Toàn trình vào Lập Plan Tổng (`09_tasks_solution.md`, `08_tasks.md`), Khung Chat và Luồng Phản hồi (`WebUI & Telegram`):**
  1. **Xóa sổ hoàn toàn các hàm/nhánh hardcode theo bài toán Postgres V1/V2:** Xóa `isDualVersionV1V2Directive`, `isPostgresOrDualSourceMigrationTask`, câu nhắc `TASK-13 PostgresDB` trong `sysPrompt` của `invokeArchitectLLM`, và các đoạn văn mẫu/Code Demo `PostgresDB` cứng trong `task_llm_bridge.go`, `unified_doc_format.go`, `telegram_message_format.go`, `task_inspector.go`.
  2. **Giữ nguyên `llmRes.DirectAnswer` & Truyền đủ `qt.Description` + `chat_history.json` vào `invokeArchitectLLM`:** Không bao giờ dùng hàm tĩnh ghi đè lên `llmRes.DirectAnswer` mà Model đã sinh ra; truyền lịch sử chat và toàn bộ `qt.Description` vào prompt để Model nắm trọn mạch hội thoại.
  3. **Gắn Model (`invokeArchitectLLM`) trực tiếp vào Bộ sinh Plan tổng (`generateDeepTaskArtifacts`) và Luồng Cập nhật Plan tổng khi Chat/Phản hồi (`handleTaskPlanningChat`, `executeArchitectReviseSolution`, `handleTelegramReviseSolutionCommand`):**
     - Khi tạo Task mới (ví dụ `EX-002` làm ClickHouse V3), `generateDeepTaskArtifacts` gọi `invokeArchitectLLM` để sinh đúng `Sub-Tasks`, `Architecture Plan`, `Risks`, `ChangedFiles` và `Code Demo` (ClickHouse `dbHandler/clickhouse.ts` + Router V1/V2/V3) ngay từ đầu.
     - Khi User nhắn phản hồi, chỉ đạo kiến trúc, hoặc sửa lưng/chỉ ra nhầm lẫn trong khung Chat (WebUI) hoặc qua `/revise` (Telegram), hệ thống bắt buộc gọi `invokeArchitectLLM` để tái lập và **cập nhật đồng bộ toàn bộ Plan tổng (`09_tasks_solution.md`, `08_tasks.md`, `subtasks.json`, `01_requirements.md`, và thẻ Telegram H4b)**.
