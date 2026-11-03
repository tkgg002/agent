---
tier: T2
suffix: fix_telegram_notification_and_polling_regression
project: ai-orchestrator
created_at: 2026-10-28T13:30:20+07:00
status: DONE
---

# 08. Tasks — Khôi phục & Chống Hồi Quy Luồng Telegram Bridge, Forum Sub-Topic & Long-Polling (`fix_telegram_notification_and_polling_regression`)

## Danh sách Sub-Tasks Thực thi (Sau khi có User `APPROVE`)

### `TASK-TELE-01`: Tách module đơn trách nhiệm `internal/webui/telegram_config_persistence.go` (Mutex + Atomic Write + Credential Preservation Guard + Auto-Recovery)
- **Mô tả & Context `file:line`:**
  - Tách hàm `telegramConfigPath`, `loadTelegramPersistedConfig`, `saveTelegramPersistedConfig`, `appendTelegramEvent` từ `internal/webui/telegram_bridge.go:132-204` sang file đơn trách nhiệm mới `internal/webui/telegram_config_persistence.go` (giúp giảm kích thước `telegram_bridge.go` xuống dưới `HEAD = 2920` dòng).
  - Bổ sung `telegramCfgMu sync.Mutex` bảo vệ toàn bộ thao tác đọc/ghi `.orchestrator/telegram_settings.json` và `.orchestrator/telegram_credentials.bak.json`.
  - Sử dụng Atomic Write (`os.WriteFile(tmpPath, ...)` + `os.Rename(tmpPath, path)`).
  - Kiểm tra lỗi `json.Unmarshal`: nếu file đang đọc bị lỗi parse JSON thì thử đọc từ `.orchestrator/telegram_credentials.bak.json` thay vì trả về struct rỗng.
  - Bổ sung **Credential Preservation Guard**: trong `saveTelegramPersistedConfig`, nếu `cfg.BotTokenEncrypted == ""` hoặc `cfg.ChatID == ""` thì tự động giữ lại `BotTokenEncrypted`/`ChatID` từ đĩa hoặc từ `.orchestrator/telegram_credentials.bak.json` (trừ khi có cờ xóa chủ động), đồng thời tự động sao lưu vào `.orchestrator/telegram_credentials.bak.json` mỗi khi có `BotTokenEncrypted` và `ChatID` hợp lệ.
  - Bổ sung helper đồng bộ thông báo từ WebUI sang Sub-Topic Telegram (`notifyTelegramOnWebUIChatOrEdit` và `notifyTelegramOnSubTaskWaveMilestone`).
- **Logs / Test:**
  - Chạy `go test -v -race ./internal/webui -run TestTelegramConfigPersistenceAndNotificationSync`.
- **Definition of Done (DoD):**
  - Không còn race condition `O_TRUNC` làm mất `bot_token_stored` và `chat_id` khi nhiều goroutine ghi event/offset/topic cùng lúc; `telegram_bridge.go` giảm dòng (`NOW <= HEAD`).

### `TASK-TELE-02`: Mở khóa `TASK-2` trong `restoreSmokeCompletedUserTasks` & Đấu nối thông báo WebUI Chat / Edit Task / Sub-Task Wave sang Telegram Sub-Topic
- **Mô tả & Context `file:line`:**
  - Tại `internal/webui/task_inspector.go:3250-3252`: Xóa bộ lọc chặn `TASK-1`..`TASK-7` trong `restoreSmokeCompletedUserTasks` và gọi đảm bảo khởi tạo Forum Sub-Topic + thông báo cho mọi Task đang active/pending/needs_you (bao gồm `TASK-2`) ngay cả khi trạng thái khác `done`.
  - Tại `internal/webui/task_chat.go:920-929`: Gọi `go s.notifyTelegramOnWebUIChatOrEdit(*qt, actionMode, agentRoleLabel, selectedModel, cleanMsg, replyText, appliedToPlan)` sau khi Co-Architect trả lời/cập nhật Plan trên WebUI (cho các mode `discuss`, `revise_solution`, `update_subtasks`).
  - Tại `internal/webui/task_edit_handler.go:321-325`: Gọi `go s.notifyTelegramOnTaskUpdated(*updatedTask, len(updatedSubTasks), false)` khi User lưu chỉnh sửa Task trên WebUI (`replan == false`).
  - Tại `internal/webui/subtask_wave_engine.go:200-425`: Gọi `go s.notifyTelegramOnSubTaskWaveMilestone(*qt, target, phase, finalMsg)` khi Sub-Task hoàn thành `subtask_plan` (`NEEDS_YOU`), `subtask_implement`, hoặc `subtask_qa` (`DONE`).
  - Đảm bảo `100%` file hiện hữu (`telegram_bridge.go`, `task_inspector.go`, `task_chat.go`, `task_edit_handler.go`, `subtask_wave_engine.go`) giữ nguyên hoặc giảm số dòng so với `HEAD` (`NOW <= HEAD`).
- **Logs / Test:**
  - Kiểm tra `git diff --numstat HEAD` đảm bảo không có file hiện hữu nào vượt `HEAD`.
- **Definition of Done (DoD):**
  - `TASK-2` được tự động tạo Forum Sub-Topic riêng (`telegram_topic.json`) trên Group Forum `-1004390214071`; mọi thao tác Chat/Revise Plan, Edit Task và Sub-Task Wave trên WebUI đều bắn bản tin đồng bộ vào đúng Sub-Topic Telegram của Task.

### `TASK-TELE-03`: Kiểm chứng Unit/Concurrency Test (`-race`, 0 `httptest`), Khôi phục Runtime `:8090` thật & Trace E2E trên Telegram Bot API thật + Audit S5 (`DG3`)
- **Mô tả & Context `file:line`:**
  - Tạo file test đơn trách nhiệm `internal/webui/telegram_config_persistence_test.go` (không dùng `net/http/httptest`, dùng TCP listener thật `net.Listen("tcp", "127.0.0.1:0")`) kiểm chứng:
    1. 50 goroutine gọi đồng thời `appendTelegramEvent`, `saveTelegramPersistedConfig` và `loadTelegramPersistedConfig` dưới cờ `-race` bảo toàn `100%` `BotTokenEncrypted` và `ChatID`.
    2. Tự động phục hồi `BotTokenEncrypted` và `ChatID` từ `.orchestrator/telegram_credentials.bak.json` nếu `.orchestrator/telegram_settings.json` bị cắt cụt (`0 byte`).
    3. Đồng bộ thông báo `notifyTelegramOnWebUIChatOrEdit`, `notifyTelegramOnTaskUpdated`, `notifyTelegramOnSubTaskWaveMilestone` vào đúng `message_thread_id` của Task.
  - Build và khởi động lại binary `orchestrator` trên cổng `:8090`, khôi phục cấu hình Telegram Bot `@tk001sea_bot` (`539329***BU0`) + Forum `-1004390214071`, xác minh `TASK-2` tự động tạo Forum Sub-Topic thật (`message_thread_id > 0`) và gửi bản tin thật lên Telegram (`HTTP 200 OK` từ `api.telegram.org`, không còn `QUEUED_LOCAL_BRIDGE`).
  - Thực hiện đầy đủ 6 bước `S5 Mandatory Final Adversarial Audit` (`DG3` Mode A + Mode B) và xuất `audit_report_fix_telegram_notification_and_polling_regression.md`.
- **Logs / Test:**
  - Chạy qua `evidence_run.py` lấy biên lai `EV-...` cho `go vet`, `go test -race`, và E2E runtime trace trên `:8090` + `api.telegram.org`.
- **Definition of Done (DoD):**
  - Toàn bộ test PASS với `-race`, server `:8090` gửi tin thật lên Telegram Forum `-1004390214071` (`SENT_TELEGRAM_API ...`), Long-Poller chạy ổn định, `DG3` PASS và `verify_governance.py` đạt `PASSED 🟢`.

### `TASK-TELE-04`: Vòng lặp Tự Chữa Lành `S5 -> S3 -> S4 -> S5` (`SA-TELE-01 .. SA-TELE-04`)
- **Mô tả & Context `file:line`:**
  1. **`SA-TELE-01` (Tách bạch Single Responsibility Principle):** Tách 4 hàm đồng bộ sự kiện từ WebUI (`ensureActiveTaskTelegramTopicAndSync`, `notifyTelegramOnWebUIChat`, `notifyTelegramOnTaskUpdated`, `notifyTelegramOnSubTaskWaveMilestone`) khỏi `internal/webui/telegram_config_persistence.go` sang module đơn trách nhiệm riêng `internal/webui/telegram_webui_sync.go`.
  2. **`SA-TELE-02` (Đóng kín chu trình `Read-Modify-Write` bên trong `telegramCfgMu` cho Poller & Forum Topic):** Bổ sung `updateTelegramPersistedConfig(mutator func(*telegramPersistedConfig))`, `recordTelegramUpdateOffset(updateID int64)`, và `recordTelegramTaskThreadID(taskKey string, threadID int64)` trong `internal/webui/telegram_config_persistence.go`; thay thế các đoạn `load -> modify -> save` bên ngoài Mutex tại `internal/webui/telegram_bridge.go:2478-2483` (`ensureTelegramApprovePoller`) và `internal/webui/telegram_forum_topic.go:95-100, 146-151` (`ensureForumTopicForTask`) để không bao giờ ghi đè làm mất `RecentEvents` hay `PendingRevisionTaskKey` khi chạy đồng thời.
  3. **`SA-TELE-03` (Giữ ngữ cảnh `taskKey` trên mọi chunk của tin nhắn dài):** Trong `sendTelegramChunkedMessageWithCustomKeyboard` / `sendTelegramBotMessageWithKeyboard` (`internal/webui/telegram_bridge.go:2200-2280`), đảm bảo mọi chunk (`Phần i/N`) đều giữ ngữ cảnh `taskKey` để `inferTaskForOutboundTelegramMessage` luôn định tuyến đúng 100% vào `message_thread_id` của Task đó.
  4. **`SA-TELE-04` (Trace Runtime E2E thật cho luồng `OUTBOUND_EDIT` trên `:8090`):** Khởi động lại `:8090` với binary mới, gọi trực tiếp `POST http://127.0.0.1:8090/api/tasks/update` trên `TASK-2` (`replan_s1_s3: false`) qua `evidence_run.py --kind runtime` và xác nhận `OUTBOUND_EDIT` đạt `SENT_TELEGRAM_API_OK` trên Sub-Topic `320` của `TASK-2`.
- **Logs / Test:**
  - Chạy `go vet ./...`, `go test -v -race ./internal/webui -run TestTelegramConfigPersistenceAndNotificationSync`, và trace runtime thật trên `:8090` qua `evidence_run.py`.
- **Definition of Done (DoD):**
  - 100% file hiện hữu trong `git diff --name-only HEAD` đạt `NOW <= HEAD`; `SA-TELE-01 .. SA-TELE-04` được khắc phục triệt để; `/laya_router` `DG3` (Mode A + Mode B) đạt `PASS`.
