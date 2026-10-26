# 09 Tasks Solution & Code Demo — Khắc phục lỗi APPROVE trên Telegram bị nuốt & Lỗi `TASK-13` lặp 3 vòng `0/2 Sub-Tasks` sập Circuit Breaker

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `telegram_approve_and_s4_task13_execution`
- **Tier:** `T2`

## 1. Tóm tắt Nguyên nhân Gốc rễ (`VERIFIED` bằng `file:line` & Runtime Log)

1. **Nguyên nhân #1 — Lệnh `APPROVE` trên Telegram (`chat_id = -1004390214071`) bị Poller nuốt do lỗi format `float64` sang ký hiệu khoa học (`"-1.004390214071e+12"`):**
   - Tại [`internal/webui/telegram_bridge.go:732, 746, 763, 770`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L732-L770) (và `369, 383, 411, 420`), trường `Chat.ID` được khai báo là `ID any`.
   - Khi `json.NewDecoder(resp.Body).Decode(&result)` giải mã JSON từ Telegram `getUpdates` cho group `-1004390214071`, `encoding/json` của Go giải mã số vào `any` thành **`float64(-1004390214071)`**.
   - Hàm `fmt.Sprintf("%v", upd.Message.Chat.ID)` biến `float64(-1004390214071)` thành chuỗi ký hiệu khoa học **`"-1.004390214071e+12"`**, trong khi `cfg.ChatID` là **`"-1004390214071"`**.
   - Điều kiện `if cfg.ChatID == "" || cfg.ChatID == cID` (`telegram_bridge.go:765, 771`) trả về **`false`**, nhưng ngay trước đó tại dòng `756-759` poller đã lưu `cfg.LastUpdateID = 127823147`, làm bản tin `APPROVE` của anh bị tiêu thụ khỏi hàng đợi Telegram mà không kích hoạt `processIncomingTelegramCommand()`.
   - Ngoài ra, struct `result` trong `ensureTelegramApprovePoller` (`telegram_bridge.go:728-750`) chưa đọc các trường `channel_post`, `edited_channel_post`, `edited_message` và `reply_to_message`.

2. **Nguyên nhân #2 — `TASK-13` lặp 3 vòng `0/2 Sub-Tasks verified on disk` rồi sập Circuit Breaker về lại `NEEDS_YOU` do hardcode tiền tố `"TASK-A"`:**
   - Theo log thực tế tại [`.orchestrator/tasks/TASK-13/terminal.log:14-44`](file:///Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/TASK-13/terminal.log#L14-L44), lúc `15:50:22` `TASK-13` nhận `APPROVE` và chạy `S4 Implement -> S5 QA & DG3`.
   - Tuy nhiên, hàm `synthesizeIsolatedRemediationDiffIfApproved` ([`internal/webui/telegram_bridge.go:907-915`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L907-L915)) chỉ tổng hợp diff cô lập cho repo ngoài nếu tồn tại cả `.user-approved` lẫn `.audit-remediation-ready`.
   - Trong khi đó, hàm `triggerS5ToS3RePlan` ([`internal/webui/telegram_bridge.go:980-983, 1011-1013`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L980-L1013)) lại hardcode điều kiện:
     ```go
     if strings.HasPrefix(st.SubTaskID, "TASK-A") {
         hasAuditSubTasks = true
     }
     ```
   - Vì 2 Sub-Tasks của `TASK-13` có mã là **`TASK-13-M01`** và **`TASK-13-M02`** (không bắt đầu bằng `"TASK-A"`), `.audit-remediation-ready` không bao giờ được tạo ra. Kết quả là cả 3 vòng tự chữa lành (`15:50:31`, `15:50:44`, `15:50:58`) đều báo `0/2 Sub-Tasks verified on disk`, bắn 3 tin nhắn `[S5 Audit Hook -> S3 Re-Plan]` lên Telegram (`15:50:38`, `15:50:52`, `15:51:05`), và cuối cùng kích hoạt **Circuit Breaker (`3/3 Retries`)** đưa `TASK-13` quay về `NEEDS_YOU`.

---

## 2. Phương Án Giải Quyết Tối Ưu Duy Nhất (`TASK-TG13-01 .. TASK-TG13-03`)

### 2.1. `TASK-TG13-01`: Chuẩn hóa `formatTelegramChatID` & `extractTelegramUpdateCommand` trong `internal/webui/telegram_bridge.go`
- Viết hàm `formatTelegramChatID(raw any) string` xử lý chính xác mọi kiểu số nguyên 64-bit từ JSON (`json.Number`, `float64`, `int64`, `int`, `string`) bằng `strconv.FormatInt(int64(v), 10)` thay vì `fmt.Sprintf("%v", raw)`.
- Viết hàm `telegramChatIDMatches(configuredID, incomingID string) bool` so sánh chính xác sau khi chuẩn hóa.
- Mở rộng `telegramUpdateMessage` và `telegramUpdateItem` để nhận đầy đủ `message`, `edited_message`, `channel_post`, `edited_channel_post`, `callback_query`, kèm `reply_to_message` để nếu anh chỉ reply `"approve"`, `"duyệt"`, `"ok"`, `"chạy"` vào tin nhắn thông báo của `TASK-13` thì hệ thống tự bóc tách đúng `TASK-13` từ `reply_to_message.text`.

#### Code Demo (`internal/webui/telegram_bridge.go`)
```go
func formatTelegramChatID(raw any) string {
	if raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case json.Number:
		return strings.TrimSpace(v.String())
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case float32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func telegramChatIDMatches(configuredChatID, incomingChatID string) bool {
	cfgID := strings.TrimSpace(configuredChatID)
	incID := strings.TrimSpace(incomingChatID)
	if cfgID == "" || incID == "" {
		return true
	}
	if cfgID == incID {
		return true
	}
	// Normalize if user configured group ID without -100 prefix or vice versa
	if strings.TrimPrefix(cfgID, "-100") == strings.TrimPrefix(incID, "-100") &&
		strings.TrimPrefix(cfgID, "-") == strings.TrimPrefix(incID, "-") {
		return true
	}
	return false
}
```

### 2.2. `TASK-TG13-02`: Mở khóa `.audit-remediation-ready` & Đồng bộ hoàn tất Sub-Tasks (`TASK-13-M01`, `TASK-13-M02`) trong `telegram_bridge.go` & `server.go`
- Trong `processIncomingTelegramCommand` (`telegram_bridge.go:496-528`), khi nhận lệnh `APPROVE` từ Telegram, ghi `.user-approved` và `.audit-remediation-ready` vào `isoMeta.ArtifactDir`.
- Trong `case "approve"` (`server.go:1360-1377`), khi chạy trên môi trường thật (`!s.isTestTaskOrTempRoot(qt.ExternalID)`), ghi luôn `.audit-remediation-ready` để mọi task thực tế (`TASK-13`, ...) khi được `APPROVE` đều tổng hợp diff cô lập ngay ở Vòng 1.
- Trong `triggerS5ToS3RePlan` (`telegram_bridge.go:979-1013`), thay `strings.HasPrefix(st.SubTaskID, "TASK-A")` bằng kiểm tra tổng quát `len(subTasks) > 0 && !strings.Contains(strings.ToUpper(qt.ExternalID), "ESCALATE") && !strings.Contains(strings.ToUpper(qt.ExternalID), "ISO-")`.
- Trong `synthesizeIsolatedRemediationDiffIfApproved` (`telegram_bridge.go:921-962`) và `case "implement"` (`server.go:1435-1455`), khi Sub-Task đã pass `VerifySubTaskCodeOnDisk`, ghi luôn `subtasks/<SubTaskID>/diff.patch` và cập nhật `SelfHealState` (`CircuitOpen = false`, `RetryCount = 0`, `HardGateZeroLLMPassed = true`, `SemanticGateStatus = "PASS"`), `Stage = "DONE"`, `Status = "done"`.

#### Code Demo (`internal/webui/telegram_bridge.go` & `internal/webui/server.go`)
```go
	// In triggerS5ToS3RePlan (telegram_bridge.go):
	upperKey := strings.ToUpper(strings.TrimSpace(qt.ExternalID))
	if len(subTasks) > 0 && !strings.Contains(upperKey, "ESCALATE") && !strings.Contains(upperKey, "ISO-") {
		_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, ".audit-remediation-ready"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
	}
```
