---
tier: T2
suffix: fix_telegram_notification_and_polling_regression
project: ai-orchestrator
created_at: 2026-10-28T13:30:10+07:00
status: IN_PROGRESS
---

# 01. Requirements — Khôi phục & Chống Hồi Quy Luồng Telegram Bridge, Forum Sub-Topic & Long-Polling (`fix_telegram_notification_and_polling_regression`)

## 1. Yêu cầu từ User
- **Nguyên văn yêu cầu:**
  > *"luồng tele sao im re rồi. kiểm tra lại xem. làm mà ko để ý các chức năng cũ à"*

## 2. Kết quả Điều tra Hiện trạng (S2 Investigation — Ground Truth `file:line`)
Qua kiểm tra trực tiếp trạng thái runtime trên `http://127.0.0.1:8090/api/settings/telegram`, file cấu hình `.orchestrator/telegram_settings.json`, cơ sở dữ liệu `.orchestrator/queue.db`, thư mục `.orchestrator/tasks/TASK-2/` và mã nguồn `internal/webui/`, có **4 nguyên nhân gốc rễ (`VERIFIED`)** khiến toàn bộ luồng Telegram bị im lặng ("im re"):

1. **RC-1 (P0 — Nguyên nhân trực tiếp làm liệt toàn bộ chiều gửi và chiều nhận Telegram): Race Condition `Read-Modify-Write` không có Mutex & Ghi đè `O_TRUNC` làm xóa trắng `bot_token_stored` và `chat_id` trong `.orchestrator/telegram_settings.json` (`internal/webui/telegram_bridge.go:138-177`)**:
   - Tại `internal/webui/telegram_bridge.go:138-159` (`loadTelegramPersistedConfig`) và `161-177` (`saveTelegramPersistedConfig`), việc đọc (`os.ReadFile`) và ghi (`os.WriteFile`) lên `.orchestrator/telegram_settings.json` **không có Mutex bảo vệ** và **không dùng Atomic Write (`write temp + os.Rename`)**.
   - Nhiều goroutine (`appendTelegramEvent` tại dòng `179`, `ensureForumTopicForTask` tại `telegram_forum_topic.go:169`, `ensureTelegramApprovePoller` tại dòng `2589`) đồng thời đọc file, sửa `Events`/`TaskTopics`/`LastUpdateOffset` rồi ghi đè lại file.
   - Khi Goroutine A đang gọi `os.WriteFile` (mở file với cờ `O_TRUNC` cắt file về `0 byte`), Goroutine B gọi `loadTelegramPersistedConfig()` đọc phải `0 byte` (hoặc JSON đang ghi dở), bỏ qua lỗi `json.Unmarshal` (`_ = json.Unmarshal(raw, &cfg)` tại `telegram_bridge.go:148`), nhận về struct mặc định với `BotTokenEncrypted: ""` và `ChatID: ""`, rồi gọi `saveTelegramPersistedConfig(cfg)` **ghi đè `bot_token_stored: ""` và `chat_id: ""` xuống đĩa**!
   - **Bằng chứng vật lý trong `.orchestrator/telegram_settings.json`**:
     - Lúc `2026-10-27 17:33:15` (`evt-1761561195534452000`), hệ thống vừa gọi Telegram API thật tạo thành công Forum Sub-Topic `thread_id=184` cho Task `#32` trên Group Forum `Ai-Orchestrator` (`chat_id: "-1004390214071"`, Bot `@tk001sea_bot` `539329***BU0`).
     - Đúng `0.86ms` sau (`evt-1761561195535318000`), race condition xảy ra làm `bot_token_stored` và `chat_id` trong `.orchestrator/telegram_settings.json` bị ghi đè thành chuỗi rỗng `""`!
     - Từ thời điểm đó, mọi lệnh gửi tin tại `sendTelegramBotMessageWithCustomKeyboard` (`telegram_bridge.go:2450-2454`) đều bị chặn lại ở nhánh:
       `QUEUED_LOCAL_BRIDGE (Chưa nhập BotToken/ChatID thật — Sẵn sàng nhận Webhook/Test APPROVE)`
       và vòng lặp Long-Polling `ensureTelegramApprovePoller` (`telegram_bridge.go:2559`) thoát ngay lập tức (`return`) vì `cfg.BotTokenEncrypted == ""`!

2. **RC-2 (P0 — Hàm `restoreSmokeCompletedUserTasks` chặn cứng `TASK-1`..`TASK-7` làm bỏ qua luôn Task thật `TASK-2` của User tại `internal/webui/task_inspector.go:3250-3252`)**:
   - Tại `internal/webui/task_inspector.go:3250-3252`:
     ```go
     if strings.HasPrefix(qt.ExternalID, "TASK-") && len(qt.ExternalID) == 6 && qt.ExternalID[5] >= '1' && qt.ExternalID[5] <= '7' {
         continue
     }
     ```
   - Task thật của User trong `queue.db` (`id=48`) có `ExternalID = "TASK-2"`. Do bộ lọc này bỏ qua mọi `TASK-1..TASK-7`, `TASK-2` không bao giờ được gọi `ensureTelegramTaskTopic` và `notifyTelegramNewTaskIntakeAndTopic` (`task_inspector.go:3277-3288`), dẫn đến `.orchestrator/tasks/TASK-2/telegram_topic.json` chưa từng được tạo!

3. **RC-3 (P1 — Tương tác trên WebUI như Chat Co-Architect / Revise Plan / Edit Task không đồng bộ sang Sub-Topic Telegram của Task)**:
   - Tại `internal/webui/task_chat.go:832-929` (`handleTaskPlanningChat`), khi User chat với Co-Architect trên WebUI (`discuss`, `revise_solution`, `update_subtasks`), hệ thống chỉ cập nhật giao diện WebUI mà không gửi bản tin đồng bộ sang Sub-Topic Telegram của Task (chỉ có nhánh `ingest_audit_remediation` ở dòng `1182` là có gọi `notifyTelegramOnS3PlanReady`).
   - Tại `internal/webui/task_edit_handler.go:321-325` (`handleUpdateTask` khi `replan == false`), khi User chỉnh sửa thông tin Task hoặc danh sách Sub-Tasks trên WebUI, hệ thống cũng không thông báo cập nhật sang Sub-Topic Telegram của Task.

4. **RC-4 (P1 — Tiến trình chạy Sub-Task Wave `executeSubTask12StepWaveLifecycle` không gửi thông báo tiến độ về Telegram Sub-Topic)**:
   - Tại `internal/webui/subtask_wave_engine.go:152-430`, khi Sub-Task hoàn thành các mốc `subtask_plan` (`NEEDS_YOU`), `subtask_implement` hoặc `subtask_qa` (`DONE`) được kích hoạt từ WebUI, hệ thống không gửi thông báo về Sub-Topic Telegram của Task cha.

## 3. Mục tiêu & Phạm vi Giải pháp (Acceptance Criteria / DoD)
1. **AC-1 (Tách Module Đơn Trách Nhiệm `internal/webui/telegram_config_persistence.go` & Chống Mất Cấu Hình Vĩnh Viễn):**
   - Tách toàn bộ logic đọc/ghi `.orchestrator/telegram_settings.json` sang file đơn trách nhiệm `internal/webui/telegram_config_persistence.go` (giữ `100%` file hiện hữu trong repo đạt `NOW <= HEAD`, không nhồi nhét vào `telegram_bridge.go`).
   - Bảo vệ đọc/ghi bằng `sync.Mutex` (`telegramCfgMu`), sử dụng Atomic Write (`write temp file + os.Rename`), kiểm tra lỗi `json.Unmarshal` nghiêm ngặt (không bao giờ dùng struct rỗng ghi đè lên đĩa khi unmarshal lỗi).
   - Bổ sung **Credential Preservation Guard & Backup Vault** (`.orchestrator/telegram_credentials.bak.json`): Khi các hàm nội bộ (`appendTelegramEvent`, `ensureForumTopicForTask`, `ensureTelegramApprovePoller`) gọi `saveTelegramPersistedConfig`, nếu `cfg.BotTokenEncrypted` hoặc `cfg.ChatID` truyền vào bị rỗng thì tự động giữ nguyên `BotTokenEncrypted` / `ChatID` đang có trên đĩa hoặc phục hồi từ `.orchestrator/telegram_credentials.bak.json`.
   - Khôi phục kết nối Bot `@tk001sea_bot` (`539329***BU0`) và Supergroup Forum `Ai-Orchestrator` (`chat_id: "-1004390214071"`) cùng Long-Poller hoạt động thông suốt.
2. **AC-2 (Khởi tạo Forum Sub-Topic & Đồng bộ Thông báo cho `TASK-2` và mọi Task trong `queue.db`):**
   - Xóa bộ lọc skip `TASK-1..TASK-7` tại `internal/webui/task_inspector.go:3250-3252` để mọi Task thật trong `queue.db` (bao gồm `TASK-2`) đều được tự động khởi tạo Forum Sub-Topic (`telegram_topic.json`) và gửi bản tin S1/S3 kèm bàn phím inline lên đúng Sub-Topic trên Telegram.
3. **AC-3 (Đồng bộ 2 chiều WebUI ↔ Telegram Sub-Topic cho Chat, Edit Task & Sub-Task Wave):**
   - Khi User chat/revise plan/update subtasks trên WebUI (`task_chat.go`), chỉnh sửa Task (`task_edit_handler.go`), hoặc khi Sub-Task hoàn thành pha thực thi trong `subtask_wave_engine.go`, tự động gửi thông báo tóm tắt kèm inline keyboard vào đúng Forum Sub-Topic của Task trên Telegram.
4. **AC-4 (Kiểm chứng Không-Mockup trên Telegram API thật & `:8090` thật):**
   - Viết unit/concurrency test kiểm chứng 50 goroutine gọi `appendTelegramEvent` + `saveTelegramPersistedConfig` song song không bao giờ làm mất `BotTokenEncrypted`/`ChatID` (0 `httptest`, dùng TCP listener thật).
   - Trace runtime thật trên `:8090` và Telegram API (`getMe`, `createForumTopic` / `sendMessage` vào Forum `-1004390214071`) xác nhận `TASK-2` có `message_thread_id` thật và nhận được tin nhắn thật trên Telegram.
