# 08 Tasks: Nâng cấp Workflow `ai-orchestrator` — Cổng Hook Audit Phản Biện `S5` Chuẩn `/agent` (`S5 -> S3 Re-Plan`), Nhịp Loading `~3s` Giữa Các Bước & Phản hồi Trực tiếp vào Co-Architect Studio (`approve_and_run`)

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `co_architect_approve_run_reply`
- **Tier:** `T2`

---

## Danh mục Task

- [x] **TASK-CAR-01 (S2 Investigate `TASK-11/terminal.log`, `TASK-11/chat_history.json`, `task_chat.go`, `server.go`, `app.js`):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/TASK-11/terminal.log:44-49`, `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/TASK-11/chat_history.json:152-186`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:1005-1038`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:958-1418`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:92-135`.
  - **Mô tả:** Truy vết chính xác thao tác của anh lúc `13:24:22` và `13:25:27` trên `:8090`, cùng kiến trúc thực thi `runTaskLifecycleAction` / `runSingleStage` (`server.go:958-1418`) và `runTaskAction` (`app.js:92-135`) để xác định nguyên nhân thiếu phản hồi chat, thiếu cổng Hook Audit Phản Biện `S5` chuẩn `/agent` (`S5 -> S3 Re-Plan`) và thiếu nhịp loading `~3s` giữa các bước.
  - **Logs/Test:** Kiểm tra trực tiếp `TASK-11/terminal.log:44-49`, `TASK-11/chat_history.json:152-186`, `server.go:958-1418`, `app.js:92-135`.
  - **Definition of Done:** Xác định rõ 4 nguyên nhân gốc rễ kèm `file:line` đã `VERIFIED`.

- [x] **TASK-CAR-02 (S3 Solution & Code Demo + Kiểm định `/laya_router` DG1/DG2):**
  - **Context (`file:line`):** `09_tasks_solution_co_architect_approve_run_reply.md`, `06_laya_claims_dg1_co_architect_approve_run_reply.json`.
  - **Mô tả:** Thiết kế giải pháp duy nhất nâng cấp `internal/webui/task_chat.go`, `internal/webui/server.go`, `internal/webui/task_inspector.go`, `internal/webui/static/app.js` và `internal/webui/static/app.css`, chạy `laya_gate.py run --stage dg1` đạt `PASS`, giữ nguyên kỷ luật Propose-Only chờ anh `APPROVE`.
  - **Logs/Test:** `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1` đạt `PASS`.
  - **Definition of Done:** `09_tasks_solution_co_architect_approve_run_reply.md` hoàn chỉnh, `DG1/DG2 PASS`.

- [x] **TASK-CAR-03 (S4 Execute — Tích hợp Cổng Hook Audit Phản Biện `S5` Chuẩn `/agent`, Vòng lặp `S5 -> S3 Re-Plan`, `audit_report_<taskKey>.md` & Phản hồi `#co-architect-chat-messages`):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:1005-1038`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:958-1418`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:1960-2100`.
  - **Mô tả:**
    1. Sửa đường dẫn `TargetFiles` / `ContextLine` trong `ingestAuditReportAndRemediation` (`task_chat.go:1010-1030`) khớp đúng cấu trúc thật (`internal/service/master/transmuter_mongo.go`, `internal/service/recon/recon_dest_mongo.go`, `internal/handler/recon/recon_execute_heal_handler.go`, `internal/repository/master_repo_gorm.go`).
    2. Nhúng nguyên văn hằng số `VerbatimS5AdversarialAuditHookPrompt` chuẩn `/agent` vào bước `qa` (`server.go:1260-1418`), thực hiện đủ 6 mục phản biện (`(a) Line-by-Line Diff Critique`, `(b) Anti-Smoke-Test Speculation`, `(c) Zero-Mockup & Runtime Trace Coordination`, `(d) Self-Improvement & S5->S3 Re-Plan Loop`, `(e) /laya_router DG3 Review`, `(f) Xuất audit_report.md + audit_report_<taskKey>.md` trong `.orchestrator/tasks/<taskKey>/`).
    3. Nếu cổng `S5 Adversarial Audit Hook` phát hiện bất kỳ vấn đề / Sub-Task chưa hoàn thành (`blocking > 0` hoặc `verifiedSubTasks < len(subTasks)`), tự động kích hoạt **`S5 -> S3 Re-Plan`**: cập nhật `09_tasks_solution.md` & `08_tasks.md` (`subtasks.json`), đánh dấu `.audit-remediation-ready`, chuyển trạng thái về `NEEDS_YOU (S3 Re-Plan)` và gửi báo cáo phản biện S5 + kế hoạch S3 Re-Plan trực tiếp vào `#co-architect-chat-messages` (`chat_history.json`).
    4. Khi anh bấm `✓ APPROVE Giải Pháp & Chạy S4–S5` sau khi đã có S3 Re-Plan (`.audit-remediation-ready` + `.user-approved`), `Slot B (MUSCLE)` tổng hợp đầy đủ các hunk unified diff cô lập (`diff --git a/<rel> b/<rel>`) vào `.orchestrator/tasks/<taskKey>/diff.patch` (tuyệt đối không chạm vào `/Users/trainguyen/Documents/work/data-hub/*`), vượt qua cổng `S5 Adversarial Audit Hook` + `/laya_router` (`DG3`), xuất `audit_report_<taskKey>.md`, gửi phản hồi chi tiết của `🛠️ MUSCLE · Slot B` & `🛡️ S5 Audit Hook · Slot K` vào `#co-architect-chat-messages`, và chuyển task sang `DONE`.
  - **Logs/Test:** `go vet ./...`, `go test -v -race ./internal/webui/...` (`EV-20261022T162739-7718`).
  - **Definition of Done:** Cổng `S5 Adversarial Audit Hook` hoạt động đúng chuẩn `/agent`, tự động `S5 -> S3 Re-Plan` khi có lỗi và tự động hoàn tất `14/14 Sub-Tasks` + trả lời trong `#co-architect-chat-messages` khi chạy `approve_and_run`.

- [x] **TASK-CAR-04 (S4 Execute — Nhịp Loading Trực Quan `~3s` Giữa Các Bước trên UI/SSE):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:958-1075`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:1960-2100`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:48-135`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css`.
  - **Mô tả:** Bổ sung phát trạng thái tiến trình từng bước (`s.advancePacedWorkflowStep(...)` + `s.broadcast("step_progress")` + nhịp loading `~3s` mỗi bước khi kích hoạt từ UI / `step_delay_ms`, mặc định `3000ms` từ UI và `0ms` trong `go test` / `isTestTaskOrTempRoot`).
  - **Logs/Test:** `go vet ./...`, `go test -v -race ./internal/webui/...` (`EV-20261022T162739-7718`).
  - **Definition of Done:** Khi bấm chạy trên UI, từng bước workflow chuyển tiếp rõ ràng với nhịp loading `~3s` giữa các bước.

- [x] **TASK-CAR-05 (S4 Execute — Nối Telegram vào bước `S3`, Nhận `APPROVE` từ Telegram tự chạy `S4–S5` & Nút `✈️ Setup Telegram` cạnh `⚙ Setup CLI & Model`):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:47-55`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js`.
  - **Mô tả:**
    1. Thêm nút **`✈️ Setup Telegram`** ngay bên cạnh nút **`⚙ Setup CLI & Model`** trên Topbar (`index.html:47-51`) và Modal `#telegram-settings-modal` cho phép cấu hình `Bot Token` (được mask bảo mật theo Rule #20 khi hiển thị/log), `Chat ID`, bật/tắt thông báo `S3 Solution` và tự động chạy khi nhận `APPROVE` từ Telegram, kèm nút Test gửi S3 và nút Giả lập/Test nhận lệnh `APPROVE` từ Telegram.
    2. Bổ sung các API `/api/settings/telegram` (`GET/POST`), `/api/settings/telegram/test` (`POST`), `/api/telegram/webhook` (`POST`) và goroutine nền `startTelegramApprovePoller` (`getUpdates`):
       - Khi một Task hoàn tất `S3 Plan` (`NEEDS_YOU`) hoặc `S5 -> S3 Re-Plan`, tự động bắn thông báo tóm tắt `S3 Solution + Code Demo + Sub-Tasks` lên Telegram kèm hướng dẫn trả lời `APPROVE <TASK-KEY>` (hoặc `APPROVE`).
       - Khi anh gửi `APPROVE` (hoặc `APPROVE TASK-11`, `/approve`, `duyệt`) trên Telegram, hệ thống tự động ghi nhận vào `#co-architect-chat-messages`, tự động chạy `approve_and_run` (`S4 -> S5 Hook Audit -> DG3 -> DONE`) và gửi kết quả hoàn tất ngược lại lên Telegram.
  - **Logs/Test:** `go test -v -race ./internal/webui/...` (`EV-20261022T162739-7718`), kiểm thử live `/api/settings/telegram` và `/api/telegram/webhook` qua `evidence_run.py` (`EV-20261022T163919-64fa`).
  - **Definition of Done:** Nút `✈️ Setup Telegram` hoạt động đầy đủ cạnh `⚙ Setup CLI & Model`, S3 tự đẩy thông báo Telegram và gửi `APPROVE` từ Telegram tự động kích hoạt chạy S4–S5 tới `DONE`.

- [x] **TASK-CAR-06 (S4 Execute — Ẩn 2 Cụm Badge Core Cố Định Trên Topbar):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:18-21`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css`.
  - **Mô tả:** Ẩn 2 cụm badge core cố định không thay đổi được (`#hdr-project-branch`: `ai-orchestrator · feat/modular-queue-agent-kernel` và `#hdr-kernel-badge`: `🛡️ Kernel: /agent (279 Lessons · H1..H7)`) trên thanh `.global-switcher` để thanh Topbar gọn gàng, đỡ rối mắt.
  - **Logs/Test:** Kiểm tra DOM và CSS trên `:8090` qua `evidence_run.py` (`EV-20261022T163919-64fa`).
  - **Definition of Done:** 2 badge cố định được ẩn khỏi Topbar.

- [x] **TASK-CAR-07 (S4 Execute & S5/S6 Verify — Tách Màn Hình `2. Wave 0..N Dependency DAG` Thành 3 Tầng Phân Cấp Trực Quan `Epic -> Task -> Sub-Task`):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:432-501`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:1200-1501`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css`.
  - **Mô tả:**
    1. Bỏ 3 tab chuyển đổi rời rạc (`Sub-Tasks | Tasks | Epics`) và bỏ việc vẽ đường nối Bezier vô nghĩa giữa các Epic độc lập.
    2. Tách màn hình `#screen-wave` thành **3 Tầng Phân Cấp Trực Quan (`3-Tier Hierarchical Drill-Down`)** hiển thị từ trên xuống dưới:
       - **Tầng 1 (`🏛️ Tầng 1: Epics — Chủ đề lớn độc lập, không nối dây`):** Hiển thị các thẻ Epic độc lập (`DATA`, `CORE`, `SRV`, `UX`, `QA`... kèm số lượng Task & tiến độ).
       - **Tầng 2 (`📋 Tầng 2: Danh sách Tasks thuộc Epic đang chọn`):** Khi bấm chọn 1 Epic ở Tầng 1, hiển thị ngay bên dưới danh sách các Task thuộc Epic đó.
       - **Tầng 3 (`🧩 Tầng 3: Sơ đồ Sub-Tasks Dependency DAG Wave 0..N của Task đang chọn`):** Khi bấm chọn 1 Task ở Tầng 2, hiển thị ngay bên dưới sơ đồ cột `Wave 0..N` kèm đường nối cong SVG Bezier giữa các Sub-Tasks của Task đó.
  - **Logs/Test:** `go vet ./...`, `go test -v -race ./internal/webui/...` (`EV-20261022T162739-7718`), kiểm thử live trên `:8090` qua `evidence_run.py` (`EV-20261022T163919-64fa`), chạy `DG3` (`06_laya_summary_dg3_20261022T164157.json`) và xuất `audit_report_co_architect_approve_run_reply.md`.
  - **Definition of Done:** Màn hình Wave DAG hiển thị trực quan đủ 3 tầng `Epic -> Task -> Sub-Task`, chọn Epic lọc Task ở dưới, chọn Task vẽ DAG Sub-Task ở dưới, và toàn bộ 8 cổng chất lượng `G1–G8` + `verify_governance.py` đạt `PASSED` 🟢.

- [x] **TASK-CAR-08 (S4 Execute — Khắc phục dứt điểm lỗi gửi Telegram trên Modal `✈️ Setup Telegram` & S3 Bridge cho `@tk001sea_bot` / `-1004390214071`):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:52-68, 187-310, 600-670`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:2345-2484`.
  - **Mô tả:**
    1. **Đồng bộ JSON field `notify_on_s3` / `notify_on_s3_plan`:** Hỗ trợ cả `notify_on_s3` và `notify_on_s3_plan` trong `TelegramSettingsDTO` (`telegram_bridge.go:59-61, 200-211`) và `app.js:2345-2361` để không bao giờ bị rơi về `notify_on_s3: false` khi bấm **"Lưu Cấu Hình Telegram"** (đồng thời tự động khôi phục `notify_on_s3: true` trong `.orchestrator/telegram_settings.json`).
    2. **Tự động lưu cấu hình trên form khi bấm `"📤 Gửi Thử Thông Báo S3 Plan"` & tự gửi tin nhắn xác nhận ngay khi bấm `"Lưu Cấu Hình Telegram"`:**
       - Khi bấm **"Lưu Cấu Hình Telegram"** (`submitTelegramSettings` -> `POST /api/settings/telegram`): lưu cấu hình VÀ nếu có `BotToken` + `ChatID`, tự động bắn ngay 1 tin nhắn thông báo S3 / xác nhận kết nối thật lên group Telegram (`-1004390214071`) để anh thấy Bot nhắn tin ngay lập tức!
       - Khi bấm **"📤 Gửi Thử Thông Báo S3 Plan"** (`testTelegramS3Notification` -> `POST /api/settings/telegram/test`): thu thập luôn các giá trị đang nhập trong `#tg-bot-token`, `#tg-chat-id`, `#tg-bot-username`, `#tg-enabled`, `#tg-notify-s3`, `#tg-auto-run` gửi kèm lên `/api/settings/telegram/test`, cập nhật cấu hình trước rồi gọi `sendTelegramBotMessageWithKeyboard` thật (bỏ qua rào cản `!cfg.NotifyOnS3` khi chủ động bấm nút Test), và hiển thị chính xác `data.delivery_status` + `data.telegram_settings` trên `#tg-test-status`.
    3. **Fallback Plain-Text khi `parse_mode: "Markdown"` bị lỗi `400` + Gắn nút bấm Inline Keyboard (`✅ APPROVE & Chạy S4–S5 Ngay`) cho Supergroup:**
       - Trong `sendTelegramBotMessageWithKeyboard` (`telegram_bridge.go:600-670`): đính kèm `reply_markup.inline_keyboard` với nút bấm **`✅ APPROVE <TaskKey> & Chạy S4–S5 Ngay`** (`callback_data: "APPROVE <TaskKey>"`), và nếu request với `"parse_mode": "Markdown"` trả mã lỗi `>= 400` thì tự động retry gửi lại không có `parse_mode` để tin nhắn luôn tới group 100%.
       - Trong `ensureTelegramApprovePoller` & `handleTelegramWebhook` (`telegram_bridge.go:325-534`): hỗ trợ xử lý cả `callback_query` (khi anh bấm nút `✅ APPROVE` ngay dưới tin nhắn của `@tk001sea_bot` trong Supergroup `-1004390214071`, hoạt động ngay cả khi Bot bật Group Privacy Mode `can_read_all_group_messages: False`!) lẫn tin nhắn văn bản (`/approve`, `APPROVE`, reply tin nhắn Bot).
  - **Logs/Test:** `go test -v -race ./internal/webui/...` (`EV-20261023T021528-840e`), `go run ./cmd/orchestrator daemon --once` (`EV-20261023T021621-c505`), kiểm thử live `/api/settings/telegram` & `/api/settings/telegram/test` trên `:8090` qua `evidence_run.py` (`EV-20261023T021656-42b5`) xác nhận `delivery_status == "SENT_TELEGRAM_API_OK"`.
  - **Definition of Done:** Bấm **"Lưu Cấu Hình Telegram"** hoặc **"📤 Gửi Thử Thông Báo S3 Plan"** đều gửi tin nhắn thật (`SENT_TELEGRAM_API_OK`) vào group `-1004390214071` của `@tk001sea_bot` kèm nút bấm `✅ APPROVE`.



