---
tier: T2
suffix: fix_telegram_notification_and_polling_regression
project: ai-orchestrator
created_at: 2026-10-28T13:31:00+07:00
stage: DG0
---

# 10. Pre-Plan Adversarial Critique (S2.5 — DG0) — `fix_telegram_notification_and_polling_regression`

## 1. Phản biện Yêu cầu & Giả định so với Ground Truth (`file:line` & Runtime)

### 1.1. Kiểm định `C-REQ`: Tại sao User phản ánh *"luồng tele sao im re rồi. kiểm tra lại xem. làm mà ko để ý các chức năng cũ à"*?
- **Sự thật trên Runtime (`http://127.0.0.1:8090/api/settings/telegram` & `.orchestrator/telegram_settings.json`):**
  - Trong file `.orchestrator/telegram_settings.json`, hai trường `bot_token_stored` và `chat_id` hiện đang bị rỗng (`""`), mặc dù trước đó lúc `2026-10-27 17:33:15` (`evt-1761561195534452000`) hệ thống vừa gọi Telegram Bot API thật tạo thành công Forum Sub-Topic `thread_id=184` cho Task `#32` trên Supergroup Forum `Ai-Orchestrator` (`chat_id: "-1004390214071"`).
  - Do `bot_token_stored` và `chat_id` bị ghi đè thành `""` đúng `0.86ms` sau (`evt-1761561195535318000`), toàn bộ các bản tin tiếp theo (`evt-1761561195535318000` trở đi) đều bị hàm `sendTelegramBotMessageWithCustomKeyboard` (`internal/webui/telegram_bridge.go:2450-2454`) chặn lại với trạng thái `"QUEUED_LOCAL_BRIDGE (Chưa nhập BotToken/ChatID thật — Sẵn sàng nhận Webhook/Test APPROVE)"` thay vì gửi lên `api.telegram.org`.
  - Cùng lúc đó, goroutine Long-Polling `ensureTelegramApprovePoller` (`internal/webui/telegram_bridge.go:2559`) kiểm tra `strings.TrimSpace(cfg.BotTokenEncrypted) == ""` nên thoát ngay lập tức (`return`), khiến mọi tương tác từ phía Telegram cũng không còn được đọc về.

### 1.2. Kiểm định `C-REALITY`: Nguyên nhân vật lý trong code gây mất `bot_token_stored`/`chat_id` và bỏ sót `TASK-2`
1. **Race Condition `Read-Modify-Write` + `O_TRUNC` tại `internal/webui/telegram_bridge.go:138-177`:**
   - `loadTelegramPersistedConfig()` (`telegram_bridge.go:138-159`) gọi `os.ReadFile(path)` và bỏ qua lỗi parse JSON (`_ = json.Unmarshal(raw, &cfg)` tại dòng `148`).
   - `saveTelegramPersistedConfig(cfg)` (`telegram_bridge.go:161-177`) gọi trực tiếp `os.WriteFile(path, raw, 0o600)` mà không có `sync.Mutex` và không ghi qua file tạm (`tmp` + `os.Rename`).
   - Khi nhiều goroutine (`appendTelegramEvent`, `ensureForumTopicForTask`, `ensureTelegramApprovePoller`) gọi đồng thời, `os.WriteFile` cắt file về `0 byte` (`O_TRUNC`), làm goroutine đọc song song nhận `0 byte`, unmarshal lỗi (bị nuốt), trả về struct mặc định (`BotTokenEncrypted: ""`, `ChatID: ""`) rồi ghi đè mất sạch thông tin xác thực trên đĩa!
2. **Bộ lọc chặn cứng `TASK-1`..`TASK-7` tại `internal/webui/task_inspector.go:3250-3252`:**
   - Trong `restoreSmokeCompletedUserTasks` (`task_inspector.go:3250-3252`), điều kiện `if strings.HasPrefix(qt.ExternalID, "TASK-") && len(qt.ExternalID) == 6 && qt.ExternalID[5] >= '1' && qt.ExternalID[5] <= '7' { continue }` chặn luôn Task thật `TASK-2` (`id=48`) của User. Ngoài ra dòng `3269` chỉ gọi `ensureTelegramTaskTopic` bên trong nhánh cập nhật `qt.State != "done"` của smoke task mà chưa đảm bảo khởi tạo Sub-Topic và đồng bộ thông báo S3 cho mọi active task như `TASK-2`.
3. **Thiếu đồng bộ thông báo từ WebUI sang Telegram Sub-Topic (`task_chat.go:832-929`, `task_edit_handler.go:321-325`, `subtask_wave_engine.go:152-430`):**
   - Khi User chat/revise plan trên WebUI (`task_chat.go:841-902`), chỉnh sửa Task (`task_edit_handler.go:321-325`), hoặc chạy Sub-Task Wave (`subtask_wave_engine.go:152-430`), chưa có lệnh gửi thông báo cập nhật sang Sub-Topic Telegram của Task.

### 1.3. Kiểm định `C-CONTRA`: Phản biện rủi ro nếu chỉ nạp lại Token thủ công mà không sửa code
- Nếu chỉ nạp lại `bot_token` và `chat_id` vào `.orchestrator/telegram_settings.json` mà không khóa Mutex + Atomic Write + Credential Preservation Guard và không gỡ bộ lọc `TASK-1..TASK-7`:
  1. Ngay khi có 2 sự kiện Telegram xảy ra sát nhau (<1ms), race condition `O_TRUNC` tại `telegram_bridge.go:138-177` sẽ lại xóa trắng `bot_token_stored` và `chat_id` một lần nữa!
  2. `TASK-2` vẫn sẽ bị `task_inspector.go:3250-3252` bỏ qua và không có Sub-Topic riêng.
  3. Khi User chat/revise plan hoặc sửa Task trên WebUI, Sub-Topic Telegram vẫn sẽ im lặng.
- Do đó, bắt buộc phải giải quyết tận gốc cả 4 nguyên nhân theo kiến trúc tách module đơn trách nhiệm (`internal/webui/telegram_config_persistence.go`) và giữ `100%` file hiện hữu `NOW <= HEAD`.
