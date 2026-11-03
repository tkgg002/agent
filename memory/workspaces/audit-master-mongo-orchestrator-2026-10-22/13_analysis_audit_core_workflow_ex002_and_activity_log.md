---
tier: T2
suffix: audit_core_workflow_ex002_and_activity_log
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-27T14:18:50+07:00
status: VERIFIED
---

# 13. Analysis — Báo Cáo Audit Toàn Diện Core Workflow (`S0 -> S6`) Của `ai-orchestrator`: Vì Sao `EX-002` (ClickHouse) Bị Lập Plan `PostgreSQL`, Vì Sao `DG0`/`Laya Gate` Bị Qua Mặt & Vị Trí Của `Activity-Log Progress`

---

## PHẦN I. GIẢI PHẪU VÌ SAO `EX-002` (KÊU LÀM CLICKHOUSE V3) LẠI RA PLAN LÀM POSTGRESQL V1/V2 & VÌ SAO PHẢN BIỆN (`DG0`) + `LAYA GATE` KHÔNG CHẶN ĐƯỢC

### 1. Bẫy từ khóa `isDualVersionV1V2Directive` & `isPostgresOrDualSourceMigrationTask` cướp quyền lập Plan (`S1` & `S3`)
- **Dữ liệu đầu vào của `EX-002`** (`.orchestrator/tasks/EX-002/01_requirements.md:10-22`):
  - Tiêu đề: `"centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn làm thêm qua clickhouse"`
  - Mô tả:
    - *"Giữ nguyên Export Type V1: hiện đang kết nối và lấy dữ liệu từ mongo"*
    - *"Giữ nguyên Export Type V2: hiện đang kết nối và lấy dữ liệu từ postgres"*
    - *"Triển khai Export Type V3: hiện đang kết nối và lấy dữ liệu từ clickhouse"*
- **Thủ phạm #1 — Khớp từ khóa mù quáng (`internal/webui/task_llm_bridge.go:189-195` & `internal/webui/telegram_message_format.go:135-142`):**
  - Tại `internal/webui/task_llm_bridge.go:189-195`:
    ```go
    func isDualVersionV1V2Directive(text string) bool {
        lower := strings.ToLower(text)
        hasV1 := strings.Contains(lower, "v1") || strings.Contains(lower, "version 1") || strings.Contains(lower, "export_type=v1") || strings.Contains(lower, "export type v1")
        hasV2 := strings.Contains(lower, "v2") || strings.Contains(lower, "version 2") || strings.Contains(lower, "export_type=v2") || strings.Contains(lower, "export type v2")
        return hasV1 && hasV2
    }
    ```
  - Vì đề bài của `EX-002` nhắc tới cả `"Export Type V1"`, `"Export Type V2"` và `"Export Type V3 (clickhouse)"`, hàm `isDualVersionV1V2Directive` lập tức trả về `true`!
  - Kéo theo `isPostgresOrDualSourceMigrationTask` (`internal/webui/telegram_message_format.go:135-142`) cũng trả về `true`.
- **Thủ phạm #2 — `DecomposeMultiRepoCodebaseSubTasks` không hề gọi Model (`invokeArchitectLLM`) mà quét file sẵn có trên đĩa (`internal/webui/task_inspector.go:1089-1198` & `2940-3074`):**
  - Trong `inspectTargetRepoSymbols` (`internal/webui/task_inspector.go:2984-2989`), khi quét repo `centralized-export-service`, vì file `src/common/adapters/database/dbHandler/postgres.ts` đã tồn tại trên đĩa và đề bài có chữ `"postgres"`, nó được cộng điểm `score += 12` + `score += 5`, trong khi `dbHandler/clickhouse.ts` là file cần tạo mới (chưa có trên đĩa) nên không có mặt trong danh sách quét!
  - Tiếp đó, `DecomposeMultiRepoCodebaseSubTasks` (`task_inspector.go:1126`) gọi `buildSubTaskModuleTitleAndDesc` (`internal/webui/task_llm_bridge.go:197-218`). Tại dòng `203-210`, vì `isDualVersionV1V2Directive(combined)` bằng `true`, nó **hardcode cứng**:
    - Sub-Task 1 (`TASK-TRAN01`): *"Giữ nguyên luồng V1 (MongoDB) & Xây dựng Model/Repository V2 (PostgreSQL)..."*
    - Sub-Task 2 (`TASK-TRAN02`): *"Triển khai Strategy/Router định tuyến song song `export_type` V1 (Mongo) vs V2 (Postgres)..."*
- **Thủ phạm #3 — `generateDeepTaskArtifacts` & `BuildUnifiedS3SolutionDoc` không gọi Model mà ghi đè bằng văn mẫu `PostgresDB` (`internal/webui/task_inspector.go:3078-3294` & `internal/webui/unified_doc_format.go:253-266`):**
  - `generateDeepTaskArtifacts` (`task_inspector.go:3078-3258`) **không gọi `invokeArchitectLLM` một lần nào**! Nó gọi `buildConcreteCodeDemoForTask` (`task_inspector.go:3288-3294`) -> gọi `buildCleanKeyDiffPreview` (`telegram_message_format.go:144-161`) trả về đoạn code cứng `export class PostgresDB { ... }`.
  - Tại `BuildUnifiedS3SolutionDoc` (`unified_doc_format.go:253`) và `BuildUnifiedCoArchitectReply` (`unified_doc_format.go:344`), hệ thống gọi `synthesizeDirectAnswerSummary` (`task_llm_bridge.go:220-239`) — hàm này lại kiểm tra `isDualVersionV1V2Directive` và in ra câu văn mẫu cứng:
    > *"Dạ anh, đối với task `EX-002` (...), giải pháp chuẩn là giữ nguyên 100% luồng `v1` (Mongoose `TransHisModel` trên MongoDB) và tách độc lập luồng `v2` qua `PostgresDB`..."*

---

### 2. Vì sao bước Phản biện `S2.5 (DG0)` và `Laya Decision Gate (DG0/DG1/DG2/DG3)` không chặn được mà vẫn báo `PASS` ("Làm láo" ở tầng Gatekeeper)?

1. **`BuildUnifiedDG0CritiqueDoc` (`internal/webui/unified_doc_format.go:169-201`) là hàm ghép bảng Markdown tĩnh 100%:**
   - Khi bước `Step 4/18: S2.5 Pre-Plan Critique (DG0)` chạy (`internal/webui/server.go:1357` -> `ensureTaskStageArtifacts` tại `internal/webui/task_inspector.go:3637`), nó chỉ gọi `BuildUnifiedDG0CritiqueDoc`.
   - Tại `unified_doc_format.go:187-192`, hàm này hoàn toàn **không kiểm tra xem từ khóa công nghệ trong đề bài (`clickhouse`, `v3`) có xuất hiện trong `subTasks` hay không**, mà chỉ nối chuỗi tĩnh:
     ```go
     b.WriteString(fmt.Sprintf("| `C-REQ` | Đối chiếu yêu cầu gốc của task `%s` với cấu trúc repo `%s` | `VERIFIED` | Khớp với phạm vi file mục tiêu đã quét trên đĩa. |\n", taskID, targetRepo))
     b.WriteString(fmt.Sprintf("| `C-REALITY` | Kiểm tra sự tồn tại vật lý của các điểm chạm mã nguồn (`%s`) | `VERIFIED` | Tồn tại vật lý trên nhánh làm việc hiện tại. |\n", firstRef))
     b.WriteString("| `C-CONTRA` | Rà soát mâu thuẫn giả định kiến trúc / model / CLI trước khi lập Plan S3 | `PASS` | Không phát hiện giả định mâu thuẫn với mã nguồn thực tế. |\n\n")
     ```
   - Kết quả: File `.orchestrator/tasks/EX-002/10_preplan_critique.md` đóng dấu `VERIFIED` / `PASS` giả tạo dù đề bài kêu làm `ClickHouse` mà Sub-Tasks toàn là `PostgreSQL`!

2. **`BuildAndRunCalibratedLayaGate` (`internal/agentgate/kernel_bridge.go:637-923`) "làm láo" bằng cách tự trích dẫn dòng đầu của chính file vừa sinh làm `Claim == Evidence == MustContain`:**
   - Tại `internal/agentgate/kernel_bridge.go:649-723`:
     - Hàm đọc chính file vừa tạo (`10_preplan_critique.md` ở `DG0`, hoặc `09_tasks_solution.md` ở `DG1/DG2`).
     - Hàm `pickLine(idx int)` (`L653-674`) lấy dòng thứ 0, dòng thứ 2, dòng thứ 4 của file đó, cắt thô 80 byte `trimmed[:80]` (`L663`, làm đứt đôi byte UTF-8 tiếng Việt tạo ký tự lỗi `\ufffd`), rồi gán:
       ```go
       q1 := pickLine(0)
       q2 := pickLine(2)
       q3 := pickLine(4)
       ...
       Claim:       q1,
       Evidence:    q1,
       Reference:   fmt.Sprintf("REF: %s:%d-%d", relRefPath, 1, maxLine),
       ControlType: "target",
       MustContain: []string{q1},
       ```
   - Bằng chứng vật lý trong `.orchestrator/tasks/EX-002/06_laya_claims_dg0_ex_002.json`:
     - `C-REQ`: `claim = "# ⚡ [S2.5 · UDF-v1] Wave Phản Biện Trước Khi Lên Plan (DG0) — EX-002"`, `evidence = "# ⚡ [S2.5 · UDF-v1] Wave Phản Biện Trước Khi Lên Plan (DG0) — EX-002"`, `must_contain = ["# ⚡ [S2.5 · UDF-v1] Wave Phản Biện Trước Khi Lên Plan (DG0) — EX-002"]`.
     - `C-REALITY`: `claim = "## 1. Nguyên Nhân Gốc Rễ & Điểm Chạm Thực Tế (Root Cause & Verified Touchpoints"`, `must_contain = ["## 1. Nguyên Nhân Gốc Rễ & Điểm Chạm Thực Tế (Root Cause & Verified Touchpoints"]`.
     - `C-CONTRA`: `claim = "### 1.2 Điểm chạm mã nguồn đã xác minh (Verified Touchpoints)"`.
   - Vì lấy chính tiêu đề Markdown của file vừa sinh để kiểm tra xem tiêu đề đó có nằm trong file đó không (`Claim == Evidence == MustContain`), cộng thêm khi `laya_health_s0: false` thì `kernel_bridge.go:900-922` chạy nhánh `OFFLINE_CALIBRATED_FALLBACK` chỉ kiểm tra chuỗi `MustContain` có nằm trong file hay không -> **100% lúc nào cũng `PASS` một cách vô nghĩa**, hoàn toàn không kiểm tra xem yêu cầu của User (`clickhouse`, `v3`) có được giải quyết trong Plan hay không!

---

## PHẦN II. `ACTIVITY-LOG PROGRESS` HIỆN TẠI ĐANG Ở ĐÂU & VÌ SAO KHÔNG THẤY THÔNG BÁO?

### 1. Vị trí hiện tại của `Activity-Log Progress`
1. **Trên đĩa vật lý:** Lưu tại `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/<TaskID>/terminal.log` (ví dụ `.orchestrator/tasks/EX-002/terminal.log`, hiện có 8 dòng log `[13:56:34]`).
2. **Trong RAM & API `/api/dashboard`:**
   - Struct `s.liveStepProgress` (`internal/webui/server.go:154-166`) được trả ra ở trường JSON `live_step_progress` và nội dung file `terminal.log` được trả ra ở trường `live_task_terminal_log` (`internal/webui/server.go:2589-2603`).
3. **Trên giao diện WebUI (`internal/webui/static/index.html` & `app.js`):**
   - **Thanh tiến trình đang chạy (`#global-running-progress-banner`):** Nằm ở `internal/webui/static/index.html:99-120`, được điều khiển bởi hàm `renderRunningStepBanner(data)` tại `internal/webui/static/app.js:879-951`.
   - **Khung Activity Log (`#live-task-terminal-pre` — `📟 Live Task Execution Log (.orchestrator/tasks/<TaskKey>/terminal.log)`):** Nằm tuốt ở **đáy màn hình `screen-autopilot`** tại `internal/webui/static/index.html:462-472` (bên dưới 5 khối lớn: Active Task Context, Sub-Tasks Progress, S3 Review & Co-Architect Chat, Pipeline Stepper, và Queue Table).

### 2. Vì sao không thấy tiến trình chạy từng bước trên WebUI và không thấy thông báo trên Telegram?
1. **Nguyên nhân #1 (WebUI): Chạy đồng bộ với `stepDelay = 0` rồi xóa trắng `s.liveStepProgress` ngay trong `< 1ms` (`internal/webui/server.go:932-968, 1122-1167`):**
   - Khi tạo task `EX-002` tại `handleCreateTask` (`server.go:932`), hàm `generateDeepTaskArtifacts` chạy đồng bộ mà không gọi `advancePacedWorkflowStep`.
   - Ngay sau đó, `req.AutoAutopilot` gọi `s.runTaskLifecycleAction(createdTask.ExternalID, "autopilot")` (`server.go:967`).
   - Tại `server.go:1122-1123`, `runTaskLifecycleAction` truyền `stepDelay = 0` vào `runTaskLifecycleActionWithPacing`!
   - Toàn bộ các bước `Step 2/18 (S1) -> Step 4/18 (S2.5 DG0) -> Step 5/18 (S3) -> Step 7/18 (DG1/DG2)` tại `server.go:1351-1364` chạy vèo qua trong chưa tới `1ms` (tất cả đều mang cùng timestamp `[13:56:34]` trong `.orchestrator/tasks/EX-002/terminal.log`), rồi khi hàm kết thúc ở dòng `1166-1167`:
     ```go
     defer s.endTrackExecution(qt.ExternalID)
     defer s.clearPacedWorkflowStep()
     ```
     Nó xóa sạch `s.liveStepProgress` về `Active: false` **trước cả khi HTTP response của `POST /api/tasks/create` trả về cho trình duyệt**! Do đó, khi trình duyệt gọi `fetchDashboard()`, `live_step_progress.active` đã là `false`, khiến `#global-running-progress-banner` luôn ẩn (`display: none`).
2. **Nguyên nhân #2 (Telegram): `advancePacedWorkflowStep` không hề gửi hay cập nhật tin nhắn tiến trình vào Topic Telegram (`internal/webui/telegram_bridge.go:2674-2771`):**
   - Hàm `advancePacedWorkflowStep` (`telegram_bridge.go:2674-2721`) và `advanceSubTaskStepProgress` (`telegram_bridge.go:2723-2771`) chỉ cập nhật biến RAM `s.liveStepProgress` và ghi vào file `terminal.log` trên ổ cứng, **hoàn toàn không gọi Telegram API (`sendMessage` / `editMessageText`)** để báo tiến trình từng bước (`S0 -> S1 -> S2 -> S2.5 DG0 -> S3`) vào Sub-Topic của Task trên Telegram!
   - Hơn nữa, trong `handleCreateTask` (`server.go:937`), hệ thống gọi `notifyTelegramNewTaskIntakeAndTopic` ngay khi `createdTask.Stage` còn là `READY`, rồi `< 1s` sau `runTaskLifecycleAction` lại gọi `notifyTelegramOnS3PlanReady` (`server.go:1403`) bị trùng khóa chống lặp hoặc chỉ bắn thẻ S3 cuối cùng mà không có bất kỳ thông báo tiến trình (`activity-log progress`) nào ở giữa.

---

## PHẦN III. BẢNG TỔNG HỢP TOÀN BỘ CÁC CHỖ "LÀM LÁO" TRONG CORE WORKFLOW (`S0 -> S6`)

| Giai đoạn | Vị trí `file:line` | Hành vi "Làm láo" (Fake / Hardcode / Tự trích dẫn) | Hậu quả thực tế |
| :--- | :--- | :--- | :--- |
| **`S0 Boot`** | `internal/agentgate/kernel_bridge.go:535-545, 900-922` | Khi `CheckLayaHealth` trả về `false` (`laya_health_s0: false`), không gọi `hooks/agy/ensure_laya_up.sh` để bật Laya `:8200` mà nhảy thẳng vào nhánh `OFFLINE_CALIBRATED_FALLBACK` tự cho `PASS`. | Trong `.orchestrator/tasks/EX-002/06_governance_BootS0.json`, `laya_health_s0: false` nhưng vẫn đóng dấu `PASSED`. |
| **`S1 Intake & Sub-Task Decomposition`** | `internal/webui/task_inspector.go:1089-1198, 2984-2989` & `internal/webui/task_llm_bridge.go:189-218` | `DecomposeMultiRepoCodebaseSubTasks` không gọi LLM; cộng điểm thiên vị `postgres.ts` có sẵn trên đĩa (`L2984-2989`); dùng `isDualVersionV1V2Directive` (`L189`) hardcode tiêu đề Sub-Task `TASK-TRAN01/02` thành `PostgreSQL V2`. | Task `EX-002` yêu cầu làm `ClickHouse V3` bị biến thành 2 Sub-Task làm `PostgreSQL V2`. |
| **`S2.5 Pre-Plan Critique (DG0)`** | `internal/webui/unified_doc_format.go:169-201` | `BuildUnifiedDG0CritiqueDoc` không đối chiếu yêu cầu gốc của User với Sub-Tasks mà chỉ in bảng Markdown tĩnh với chữ `"VERIFIED"` và `"PASS"` cố định (`L189-191`). | `10_preplan_critique.md` của `EX-002` báo `PASS` 100% dù đề bài `ClickHouse` bị lập Plan `Postgres`. |
| **`Laya Gate (DG0 / DG1 / DG2 / DG3)`** | `internal/agentgate/kernel_bridge.go:649-723` | `BuildAndRunCalibratedLayaGate` đọc chính file Markdown vừa ghi, lấy dòng 0, 2, 4 (`pickLine`), cắt thô 80 byte `trimmed[:80]` (vỡ UTF-8 `\ufffd`), rồi gán `Claim = q1`, `Evidence = q1`, `MustContain = [q1]`. | Tự lấy tiêu đề file để kiểm tra chính file đó -> `laya_gate.py` luôn `PASS` giả tạo 100%, vô hiệu hóa hoàn toàn vai trò kiểm định của Laya. |
| **`S3 Solution & Co-Architect Chat`** | `internal/webui/task_inspector.go:3078-3294`, `unified_doc_format.go:253-266, 344`, `task_llm_bridge.go:220-239, 279`, `telegram_message_format.go:135-161` | (1) `generateDeepTaskArtifacts` không gọi `invokeArchitectLLM`; (2) `buildCleanKeyDiffPreview` hardcode `export class PostgresDB`; (3) `synthesizeDirectAnswerSummary` đè mất `llmRes.DirectAnswer` bằng câu mẫu `PostgresDB`; (4) `sysPrompt` (`L279`) nhét luật riêng của `TASK-13 (PostgresDB)` vào mọi task. | Cả `09_tasks_solution.md`, thẻ Telegram lẫn câu mở đầu khung Chat của `EX-002` đều in văn mẫu và Code Demo `PostgresDB`. |
| **`S4 Implement`** | `internal/webui/server.go:1467-1663`, `subtask_wave_engine.go:245-276`, `telegram_bridge.go:2821-2995` | `S4` hoàn toàn **không gọi `AntigravityRunner` / `ClaudeRunner` để viết code thật** cho các Sub-Task của task mới! Chỉ gọi `synthesizeIsolatedRemediationDiffIfApproved` copy `git diff` cũ trong repo hoặc tạo bản vá giả. | Khi bấm `APPROVE` sang `S4`, hệ thống không thực sự code tính năng mới (như `ClickHouse V3`) mà chỉ kiểm tra file cũ trên đĩa. |
| **`S5 Audit & Evidence Receipt`** | `internal/webui/server.go:1788-1830, 1939` | (1) Chỉ chạy `RunEvidenceCommand` nếu có `go.mod` (`L1788`); với repo Node/TS (`centralized-export-service`), vì `evidenceReceiptIDs` rỗng nên chạy lệnh giả `test -f 01_requirements.md` (`L1826`) để lấy biên lai `EV-...` khống! (2) Hardcode câu `"Verified intra-batch _source_ts deduplication... thread-safe masterMongoPool"` (`L1939`) vào mọi `audit_report.md`. | Đúc biên lai `EV-...` giả bằng lệnh `test -f` và chèn báo cáo `masterMongoPool` của dự án CDC vào mọi dự án khác. |
| **`Activity-Log Progress`** | `internal/webui/server.go:932-968, 1123, 1166-1167`, `telegram_bridge.go:2674-2771`, `static/index.html:464` | Chạy `stepDelay = 0` rồi `defer clearPacedWorkflowStep()` xóa trắng tiến trình trong `< 1ms`; giấu `terminal.log` ở cuối trang WebUI; không gửi/live-edit tiến trình vào Topic Telegram. | User không thấy thanh tiến trình chạy qua các bước trên WebUI và không nhận được thông báo tiến trình trên Telegram. |
