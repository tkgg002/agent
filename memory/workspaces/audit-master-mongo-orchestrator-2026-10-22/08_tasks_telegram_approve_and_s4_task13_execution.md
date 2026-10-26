# 08 Tasks — Khắc phục lỗi APPROVE trên Telegram bị nuốt & Lỗi `TASK-13` lặp 3 vòng `0/2 Sub-Tasks` sập Circuit Breaker

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `telegram_approve_and_s4_task13_execution`
- **Tier:** `T2`

## Danh sách Sub-Tasks (Autonomous Full-Stack Prompting — Rule #2)

### 1. [x] `TASK-TG13-01`: Chuẩn hóa `formatTelegramChatID` chống lỗi `float64` ký hiệu khoa học (`-1.004390214071e+12`) & Hỗ trợ `channel_post` / `edited_message` / `reply_to_message` (`DONE` · `EV-20261023T163721-e8a1`)
- **Mô tả & Context `file:line`:**
  - [`internal/webui/telegram_bridge.go:361-505`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L361-L505) (`telegramIncomingMessage`, `telegramWebhookPayload`, `formatTelegramChatID`, `telegramChatIDMatches`, `extractTelegramMessageFields`, `handleTelegramWebhook`)
  - [`internal/webui/telegram_bridge.go:535-617`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L535-L617) (`processIncomingTelegramCommand`)
  - [`internal/webui/telegram_bridge.go:834-915`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L834-L915) (`ensureTelegramApprovePoller`)
  - Viết hàm `formatTelegramChatID(raw any) string` và `telegramChatIDMatches(configuredID, incomingID string) bool` để chuyển mọi giá trị `float64`, `int64`, `json.Number`, `string` của `chat.id` về đúng chuỗi số nguyên thập phân (`"-1004390214071"`), tuyệt đối không dùng `fmt.Sprintf("%v", any)` sinh ra `"-1.004390214071e+12"`.
  - Tách logic bóc tách bản tin Telegram thành hàm `extractTelegramMessageFields(msg *telegramIncomingMessage) (cmdText, replyContext, chatID, fromUser string)` hỗ trợ đầy đủ `callback_query`, `message`, `edited_message`, `channel_post`, `edited_channel_post` và trích xuất `task_key` từ `reply_to_message` nếu người dùng reply ngắn (`"approve"`, `"duyệt"`, `"ok"`, `"chạy"`).
- **Logs / Test:**
  - `go test -race -v -run TestTelegramChatIDScientificNotationAndTask13Execution ./internal/webui/...` (`EV-20261023T163651-f7d4`, `EV-20261023T163721-e8a1`)
- **Definition of Done (DoD):**
  - Giải mã JSON `{"chat":{"id":-1004390214071}}` trả về chính xác `"-1004390214071"`, khớp `telegramChatIDMatches("-1004390214071", cID) == true`.

### 2. [x] `TASK-TG13-02`: Gỡ bỏ hardcode tiền tố `"TASK-A"` trong `triggerS5ToS3RePlan` & Đồng bộ thực thi `S4/S5` cô lập cho mọi Sub-Task (`TASK-13-M01`, `TASK-13-M02`) ngay khi User `APPROVE` (`DONE` · `EV-20261023T163721-e8a1`, `EV-20261023T164030-daae`)
- **Mô tả & Context `file:line`:**
  - [`internal/webui/telegram_bridge.go:1014-1140`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L1014-L1140) (`synthesizeIsolatedRemediationDiffIfApproved`, `triggerS5ToS3RePlan`)
  - [`internal/webui/server.go:1360-1470`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go#L1360-L1470) (`case "approve"`, `case "implement"`)
  - Trong `processIncomingTelegramCommand` (`telegram_bridge.go:577-582`), `case "approve"` (`server.go:1361-1365`, khi `!s.isTestTaskOrTempRoot(qt.ExternalID)`), và `triggerS5ToS3RePlan` (`telegram_bridge.go:1098-1135`, khi `len(subTasks) > 0 && !strings.Contains(upperKey, "ESCALATE") && !strings.Contains(upperKey, "ISO-")`), ghi `.audit-remediation-ready` cho mọi task có Sub-Tasks (không phân biệt tiền tố `TASK-A*` hay `TASK-13-M*`).
  - Trong `synthesizeIsolatedRemediationDiffIfApproved` (`telegram_bridge.go:1069-1081`) và `case "implement"` (`server.go:1443-1461`), khi Sub-Task đã được xác minh (`okDisk == true`), đồng bộ luôn `subtasks/<SubTaskID>/diff.patch`, `SelfHealState` (`CircuitOpen = false`, `RetryCount = 0`, `HardGateZeroLLMPassed = true`, `SemanticGateStatus = "PASS"`), `Stage = "DONE"`, `Status = "done"` để `TASK-13` hoàn tất `100%` Sub-Tasks ngay ở Vòng 1 (`Loop 1 -> DONE`) và gửi thông báo `✅ [AI-ORCHESTRATOR · S4–S5 COMPLETE]` về Telegram.
- **Logs / Test:**
  - `go test -v -race ./internal/webui/...` (`EV-20261023T163721-e8a1`, `12/12 PASS`)
- **Definition of Done (DoD):**
  - `TASK-13` (với `TASK-13-M01`, `TASK-13-M02`) khi chạy `approve_and_run` (từ Telegram hoặc UI) đạt `Stage = DONE` (`2/2 Sub-Tasks VERIFIED`) ngay trong Vòng 1, đồng thời 100% test cũ (`12/12`) vẫn `PASS`.

### 3. [x] `TASK-TG13-03`: Kiểm thử Regression, Rebuild Binary `:8090` & Chạy hoàn tất `TASK-13` trên Runtime Thật (`DONE` · `EV-20261023T163841-e98e`, `EV-20261023T163846-1ccd`, `EV-20261023T164030-daae`)
- **Mô tả & Context `file:line`:**
  - [`internal/webui/server_test.go:2086-2189`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server_test.go#L2086-L2189)
  - Viết test `TestTelegramChatIDScientificNotationAndTask13Execution` tái hiện Red -> Green cho cả 2 lỗi (`float64(-1004390214071)` trong `any` JSON decoding và `TASK-13` với `TASK-13-M01`, `TASK-13-M02`).
  - Chạy `evidence_run.py` lấy biên lai `EV-...`, rebuild `./bin/orchestrator` (`EV-20261023T163841-e98e`), chạy `daemon --once` (`EV-20261023T163846-1ccd`), khởi động lại `:8090` và kích hoạt hoàn tất `TASK-13` sang `DONE` (`EV-20261023T164030-daae`, `OUTBOUND_DONE SENT_TELEGRAM_API_OK`).
- **Logs / Test:**
  - `python3 /Users/trainguyen/Documents/work/agent/tooling/evidence_run.py --workspace audit-master-mongo-orchestrator-2026-10-22 --kind integration --cwd /Users/trainguyen/Documents/work/ai-orchestrator -- go test -v -race ./internal/webui/...`
- **Definition of Done (DoD):**
  - `100%` test suite `PASS` (`12/12`), `TASK-13` trên `:8090` chuyển sang `DONE` (`2/2 Sub-Tasks VERIFIED`), `DG3` & `verify_governance.py` đạt `PASSED 🟢`.
