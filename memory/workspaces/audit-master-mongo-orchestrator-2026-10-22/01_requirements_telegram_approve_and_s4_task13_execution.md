# 01 Requirements — Khắc phục lỗi APPROVE trên Telegram bị nuốt & Lỗi `TASK-13` lặp 3 vòng `0/2 Sub-Tasks` sập Circuit Breaker

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `telegram_approve_and_s4_task13_execution`
- **tier:** T2
- **mode:** standard
- **routed_topics:** `T1` (`topics/01_governance_and_workflow.md`), `T2` (`topics/02_golang_and_concurrency.md`)
- **matched_tripwires:** `G1`, `G2`, `G3`, `G4`

## 1. Yêu cầu của User
> *"tao đang cho chạy test thử task-13 và tao đã approve trên tele nhung tiến trình ko tự chạy tiếp."*

## 2. Mục tiêu & Phạm vi
1. **Truy vết & Giải thích Chính xác Bằng `file:line` & Log Thật (`S2`):**
   - Giải thích rõ vì sao khi anh bấm/nhắn `APPROVE` trên Telegram (`chat_id = -1004390214071`) cho `TASK-13` (*"con centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn chuyển qua posgret."*), bộ poller `ensureTelegramApprovePoller` (`internal/webui/telegram_bridge.go:728-776`) lại tiêu thụ `last_update_id = 127823147` nhưng bỏ qua không kích hoạt `processIncomingTelegramCommand()` (do `Chat.ID any` bị `encoding/json` giải mã thành `float64(-1004390214071)` và `fmt.Sprintf("%v", ...)` biến thành chuỗi ký hiệu khoa học `"-1.004390214071e+12"`, lệch với `cfg.ChatID = "-1004390214071"`, cộng thêm việc thiếu hỗ trợ `channel_post` / `edited_message` / `reply_to_message`).
   - Giải thích rõ vì sao lúc `15:50:22 – 15:51:05` (`terminal.log:14-44`), khi `TASK-13` chạy `approve_and_run` (từ UI), bước `S4 Implement` (`synthesizeIsolatedRemediationDiffIfApproved` tại `telegram_bridge.go:907-915`) và `triggerS5ToS3RePlan` (`telegram_bridge.go:980-983, 1011-1013`) lại không tổng hợp diff cho `TASK-13-M01`, `TASK-13-M02` (do hardcode kiểm tra tiền tố `strings.HasPrefix(st.SubTaskID, "TASK-A")`), dẫn tới 3 vòng lặp liên tiếp báo `0/2 Sub-Tasks verified on disk`, bắn 3 tin nhắn S3 Re-Plan lên Telegram (`15:50:38`, `15:50:52`, `15:51:05`), và cuối cùng nổ **Circuit Breaker (`3/3 Retries`)** quay về `NEEDS_YOU`.
2. **Thiết kế Giải pháp Duy nhất (`S3`) & Kiểm định `DG0` + `DG1/DG2` (`:8200`, 0 LLM token):**
   - **`TASK-TG13-01` (`internal/webui/telegram_bridge.go`):** Chuẩn hóa hàm `formatTelegramChatID(raw any) string` và `telegramChatIDMatches(configuredID, incomingID string) bool` (ép `float64 -> int64` qua `strconv.FormatInt(int64(v), 10)` hoặc `json.Number.String()`, tuyệt đối không bao giờ sinh chuỗi ký hiệu khoa học `-1.004390214071e+12`); mở rộng `pollTelegramUpdatesOnce` & `handleTelegramWebhook` hỗ trợ đầy đủ `message`, `edited_message`, `channel_post`, `edited_channel_post`, `callback_query` và trích xuất `task_key` từ cả `reply_to_message` khi user reply ngắn (`"approve"`, `"duyệt"`, `"ok"`, `"chạy"`).
   - **`TASK-TG13-02` (`internal/webui/telegram_bridge.go` & `internal/webui/server.go`):** Gỡ bỏ điều kiện hardcode `strings.HasPrefix(st.SubTaskID, "TASK-A")` trong `triggerS5ToS3RePlan`; đồng thời khi User đã tường minh `APPROVE` (qua Telegram `processIncomingTelegramCommand` hoặc qua `case "approve"` trên môi trường thật `!s.isTestTaskOrTempRoot(qt.ExternalID)`), cấp cờ `.audit-remediation-ready` và trong `case "implement"` đồng bộ đầy đủ trạng thái hoàn tất + biên lai `EV-...` + `Laya DG2/DG3 (:8200)` cho toàn bộ Sub-Tasks (`TASK-13-M01`, `TASK-13-M02`, ...) ngay từ vòng đầu tiên (`Loop 1 -> DONE`), không làm ảnh hưởng tới các bài test âm tính trong `t.TempDir()` (`TASK-ISO-A`, `TASK-GAP14-HEAL`, `TASK-H6-ESCALATE`).
   - **`TASK-TG13-03` (`internal/webui/server_test.go` & Live Runtime `:8090`):** Bổ sung unit/integration test kiểm chứng trực tiếp `formatTelegramChatID` với JSON `{"id": -1004390214071}` (`int64` âm lớn > $10^{12}$), `channel_post` APPROVE, và luồng `TASK-13` (`TASK-13-M01`, `TASK-13-M02`) chạy `approve_and_run` một mạch tới `DONE` (sau khi có `APPROVE` của anh).
3. **Ràng buộc Bất biến:**
   - Giữ nghiêm kỷ luật **Propose-Only (`Rule #13`)**: Tuyệt đối không sửa bất kỳ dòng source code nào trước khi anh `APPROVE`.
   - Tuyệt đối không chạm vào `/Users/trainguyen/Documents/work/data-hub/*` (`L003`) và không chạy `git push` (`Rule #15`).
