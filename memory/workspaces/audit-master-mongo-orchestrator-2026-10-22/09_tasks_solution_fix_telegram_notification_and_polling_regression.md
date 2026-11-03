---
tier: T2
suffix: fix_telegram_notification_and_polling_regression
project: ai-orchestrator
created_at: 2026-10-28T13:34:00+07:00
status: PROPOSE_ONLY
---

# 09. Technical Solution — Khôi phục & Chống Hồi Quy Luồng Telegram Bridge, Forum Sub-Topic & Long-Polling (`fix_telegram_notification_and_polling_regression`)

## 1. Tổng quan Nguyên nhân Gốc rễ (`VERIFIED` với `file:line`)

| Mã | Mức độ | Vị trí `file:line` | Nguyên nhân gốc rễ (Root Cause) | Tác động thực tế |
|---|---|---|---|---|
| **RC-1** | **P0** | `internal/webui/telegram_bridge.go:138-177` | `loadTelegramPersistedConfig()` và `saveTelegramPersistedConfig(cfg)` đọc/ghi `.orchestrator/telegram_settings.json` không có `sync.Mutex`, dùng `os.WriteFile` (`O_TRUNC`) trực tiếp lên file đích và bỏ qua lỗi `json.Unmarshal` (`_ = json.Unmarshal(raw, &cfg)` tại dòng `148`). Khi 2 goroutine ghi sự kiện sát nhau (`2026-10-27 17:33:15.534` và `.535`), goroutine đọc trúng thời điểm file bị `O_TRUNC` (`0 byte`) nhận về `BotTokenEncrypted: ""` & `ChatID: ""` rồi ghi đè mất sạch thông tin kết nối trên đĩa. | Mọi bản tin gửi đi bị rơi vào nhánh `QUEUED_LOCAL_BRIDGE` (`telegram_bridge.go:2450-2454`); goroutine Long-Polling `ensureTelegramApprovePoller` (`telegram_bridge.go:2559`) thoát ngay lập tức (`return`). Toàn bộ luồng Telegram "im re". |
| **RC-2** | **P0** | `internal/webui/task_inspector.go:3250-3252, 3269-3288` | `restoreSmokeCompletedUserTasks` có bộ lọc `if strings.HasPrefix(qt.ExternalID, "TASK-") && len(qt.ExternalID) == 6 && qt.ExternalID[5] >= '1' && qt.ExternalID[5] <= '7' { continue }` chặn cứng `TASK-1`..`TASK-7`, và chỉ gọi `ensureTelegramTaskTopic` bên trong nhánh chuyển trạng thái `done` của smoke task. | Task thật `TASK-2` (`id=48`, đang ở `NEEDS_YOU`) của User bị bỏ qua hoàn toàn, chưa từng có `.orchestrator/tasks/TASK-2/telegram_topic.json` và không được gửi bản tin lên Forum `-1004390214071`. |
| **RC-3** | **P1** | `internal/webui/task_chat.go:832-929` & `internal/webui/task_edit_handler.go:321-325` | Khi User chat với Co-Architect trên WebUI (`discuss`, `revise_solution`, `update_subtasks`) hoặc bấm `💾 Lưu Thay Đổi Task` (`replan == false`), hệ thống chỉ cập nhật giao diện WebUI mà không bắn bản tin đồng bộ sang Forum Sub-Topic của Task trên Telegram. | User thao tác trên WebUI nhưng Sub-Topic của Task trên Telegram không nhận được thông báo cập nhật Plan hay phản hồi của Co-Architect. |
| **RC-4** | **P1** | `internal/webui/subtask_wave_engine.go:152-430` | Khi Sub-Task hoàn thành các pha `subtask_plan` (`NEEDS_YOU`), `subtask_implement`, hoặc `subtask_qa` (`DONE`) từ WebUI, hệ thống không gửi thông báo tiến độ về Forum Sub-Topic của Task cha. | Sub-Topic trên Telegram không hiển thị tiến trình hoàn thành của từng Sub-Task. |

---

## 2. Phương án Kỹ thuật Duy Nhất & Tối Ưu Nhất (Single Optimal Architecture)

### 2.1. Tách Module Đơn Trách Nhiệm `internal/webui/telegram_config_persistence.go` (Giữ `100%` file hiện hữu `NOW <= HEAD`)
Tuân thủ nghiêm ngặt **Single Responsibility Principle (SRP)** và kỷ luật không nhồi nhét vào file lớn:
- Di chuyển các hàm quản lý lưu trữ cấu hình (`telegramSettingsPath`, `loadTelegramPersistedConfig`, `saveTelegramPersistedConfig`, `appendTelegramEvent`) khỏi `internal/webui/telegram_bridge.go:132-204` sang file mới `internal/webui/telegram_config_persistence.go` (giúp `telegram_bridge.go` giảm ~70 dòng so với `HEAD = 2920`).
- Thiết kế cơ chế **3 lớp bảo vệ chống mất cấu hình vĩnh viễn**:
  1. **Mutex Serialization (`telegramCfgMu sync.Mutex`):** Khóa toàn bộ thao tác đọc/ghi `.orchestrator/telegram_settings.json` và `.orchestrator/telegram_credentials.bak.json`.
  2. **Atomic Temp-File Write + Rename:** Trong `saveTelegramPersistedConfigLocked`, luôn ghi ra file tạm `.orchestrator/telegram_settings.json.tmp.<nano>` rồi gọi `os.Rename(tmpPath, p)` để file đích không bao giờ rơi vào trạng thái `0 byte` (`O_TRUNC`) hay JSON ghi dở.
  3. **Credential Preservation Guard & Backup Vault (`.orchestrator/telegram_credentials.bak.json`):**
     - Mỗi khi lưu `cfg` có `BotTokenEncrypted != ""` và `ChatID != ""`, tự động lưu bản backup vào `.orchestrator/telegram_credentials.bak.json`.
     - Khi một goroutine gọi `saveTelegramPersistedConfig(cfg)` mà `cfg.BotTokenEncrypted == ""` hoặc `cfg.ChatID == ""` (ví dụ goroutine chỉ append event hoặc cập nhật offset), hệ thống tự động đọc lại thông tin xác thực đang có trên đĩa hoặc từ `.orchestrator/telegram_credentials.bak.json` để **bảo toàn `BotTokenEncrypted` và `ChatID`**, tuyệt đối không bao giờ ghi đè chuỗi rỗng!
     - Trong `loadTelegramPersistedConfig`, nếu `cfg.BotTokenEncrypted == ""` hoặc `cfg.ChatID == ""`, tự động khôi phục từ `.orchestrator/telegram_credentials.bak.json` (hoặc biến môi trường `ORCH_TELEGRAM_BOT_TOKEN` / `ORCH_TELEGRAM_CHAT_ID`) và tự kích hoạt lại `ensureTelegramApprovePoller()`.

### 2.2. Mở khóa `TASK-2` trong `restoreSmokeCompletedUserTasks` (`internal/webui/task_inspector.go`)
- Xóa bỏ điều kiện skip `TASK-1`..`TASK-7` tại `internal/webui/task_inspector.go:3250-3252`.
- Bổ sung lời gọi helper `s.ensureActiveTaskTelegramTopicAndSync(qt, subTasks)` (đặt trong `telegram_config_persistence.go`) để mọi Task thật trong `queue.db` (bao gồm `TASK-2` đang ở `NEEDS_YOU`) đều được đảm bảo khởi tạo Forum Sub-Topic (`telegram_topic.json`) và gửi bản tin S1/S3 kèm inline keyboard nếu Sub-Topic chưa từng được khởi tạo.

### 2.3. Đồng bộ 2 chiều từ WebUI sang Telegram Sub-Topic (`task_chat.go`, `task_edit_handler.go`, `subtask_wave_engine.go`)
- Trong `internal/webui/telegram_config_persistence.go`, định nghĩa 3 hàm đồng bộ gọn sạch:
  1. `notifyTelegramOnWebUIChat(qt queue.QueuedTask, actionMode, roleLabel, model, userMsg, agentReply string, appliedToPlan bool)`: Gọi từ `task_chat.go` khi User chat trên WebUI (`discuss`, `revise_solution`, `update_subtasks`) để bắn tóm tắt câu hỏi + phản hồi của Co-Architect (và trạng thái cập nhật Plan) vào đúng Forum Sub-Topic của Task.
  2. `notifyTelegramOnTaskUpdated(qt queue.QueuedTask, subTaskCount int)`: Gọi từ `task_edit_handler.go` khi User lưu chỉnh sửa Task (`replan == false`) để thông báo cập nhật thông tin Task và số lượng Sub-Tasks vào đúng Forum Sub-Topic của Task.
  3. `notifyTelegramOnSubTaskWaveMilestone(qt queue.QueuedTask, st SubTaskDTO, phase, summary string)`: Gọi từ `subtask_wave_engine.go` khi Sub-Task hoàn thành `subtask_plan`, `subtask_implement`, hoặc `subtask_qa` để báo tiến độ kèm bàn phím điều khiển Sub-Task vào đúng Forum Sub-Topic của Task cha.

---

## 3. Code Demo Chi Tiết

### 3.1. File mới `internal/webui/telegram_config_persistence.go`
```go
package webui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/trainguyen/ai-orchestrator/internal/queue"
)

var telegramCfgMu sync.Mutex

type telegramCredentialsBackup struct {
	BotTokenEncrypted string `json:"bot_token_stored"`
	ChatID            string `json:"chat_id"`
	BotUsername       string `json:"bot_username"`
	UpdatedAt         string `json:"updated_at"`
}

func (s *Server) telegramSettingsPath() string {
	root := s.Root
	if root == "" {
		root = "."
	}
	return filepath.Join(root, ".orchestrator", "telegram_settings.json")
}

func (s *Server) telegramCredentialsBackupPath() string {
	root := s.Root
	if root == "" {
		root = "."
	}
	return filepath.Join(root, ".orchestrator", "telegram_credentials.bak.json")
}

func (s *Server) loadTelegramCredentialsBackupLocked() telegramCredentialsBackup {
	var bak telegramCredentialsBackup
	if raw, err := os.ReadFile(s.telegramCredentialsBackupPath()); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &bak)
	}
	if strings.TrimSpace(bak.BotTokenEncrypted) == "" {
		bak.BotTokenEncrypted = strings.TrimSpace(os.Getenv("ORCH_TELEGRAM_BOT_TOKEN"))
	}
	if strings.TrimSpace(bak.ChatID) == "" {
		bak.ChatID = strings.TrimSpace(os.Getenv("ORCH_TELEGRAM_CHAT_ID"))
	}
	return bak
}

func (s *Server) saveTelegramCredentialsBackupLocked(token, chatID, username string) {
	token = strings.TrimSpace(token)
	chatID = strings.TrimSpace(chatID)
	if token == "" || chatID == "" {
		return
	}
	bakPath := s.telegramCredentialsBackupPath()
	_ = os.MkdirAll(filepath.Dir(bakPath), 0o755)
	bak := telegramCredentialsBackup{
		BotTokenEncrypted: token,
		ChatID:            chatID,
		BotUsername:       strings.TrimSpace(username),
		UpdatedAt:         time.Now().Format(time.RFC3339),
	}
	if raw, err := json.MarshalIndent(bak, "", "  "); err == nil {
		tmp := fmt.Sprintf("%s.tmp.%d", bakPath, time.Now().UnixNano())
		if os.WriteFile(tmp, raw, 0o600) == nil {
			_ = os.Rename(tmp, bakPath)
		}
	}
}

func (s *Server) loadTelegramPersistedConfig() telegramPersistedConfig {
	telegramCfgMu.Lock()
	defer telegramCfgMu.Unlock()
	return s.loadTelegramPersistedConfigLocked()
}

func (s *Server) loadTelegramPersistedConfigLocked() telegramPersistedConfig {
	p := s.telegramSettingsPath()
	cfg := telegramPersistedConfig{
		Enabled:          true,
		NotifyOnS3:       true,
		AutoRunOnApprove: true,
		BotUsername:      "@tk001sea_bot",
		TaskThreadIDs:    make(map[string]int64),
	}
	if raw, err := os.ReadFile(p); err == nil && len(raw) > 0 {
		var parsed telegramPersistedConfig
		if err := json.Unmarshal(raw, &parsed); err == nil {
			cfg = parsed
		}
	}
	if cfg.TaskThreadIDs == nil {
		cfg.TaskThreadIDs = make(map[string]int64)
	}
	bak := s.loadTelegramCredentialsBackupLocked()
	healed := false
	if strings.TrimSpace(cfg.BotTokenEncrypted) == "" && bak.BotTokenEncrypted != "" {
		cfg.BotTokenEncrypted = bak.BotTokenEncrypted
		healed = true
	}
	if strings.TrimSpace(cfg.ChatID) == "" && bak.ChatID != "" {
		cfg.ChatID = bak.ChatID
		healed = true
	}
	if strings.TrimSpace(cfg.BotUsername) == "" && bak.BotUsername != "" {
		cfg.BotUsername = bak.BotUsername
	}
	if cfg.Enabled && strings.TrimSpace(cfg.BotTokenEncrypted) != "" && !cfg.NotifyOnS3 {
		cfg.NotifyOnS3 = true
	}
	cfg.MaskedBotToken = maskTelegramBotToken(cfg.BotTokenEncrypted)
	if healed {
		_ = s.saveTelegramPersistedConfigLocked(cfg)
	} else if strings.TrimSpace(cfg.BotTokenEncrypted) != "" && strings.TrimSpace(cfg.ChatID) != "" {
		s.saveTelegramCredentialsBackupLocked(cfg.BotTokenEncrypted, cfg.ChatID, cfg.BotUsername)
	}
	return cfg
}

func (s *Server) saveTelegramPersistedConfig(cfg telegramPersistedConfig) error {
	telegramCfgMu.Lock()
	defer telegramCfgMu.Unlock()
	return s.saveTelegramPersistedConfigLocked(cfg)
}

func (s *Server) saveTelegramPersistedConfigLocked(cfg telegramPersistedConfig) error {
	p := s.telegramSettingsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	var existing telegramPersistedConfig
	if raw, err := os.ReadFile(p); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &existing)
	}
	bak := s.loadTelegramCredentialsBackupLocked()
	if strings.TrimSpace(cfg.BotTokenEncrypted) == "" {
		if strings.TrimSpace(existing.BotTokenEncrypted) != "" {
			cfg.BotTokenEncrypted = existing.BotTokenEncrypted
		} else if bak.BotTokenEncrypted != "" {
			cfg.BotTokenEncrypted = bak.BotTokenEncrypted
		}
	}
	if strings.TrimSpace(cfg.ChatID) == "" {
		if strings.TrimSpace(existing.ChatID) != "" {
			cfg.ChatID = existing.ChatID
		} else if bak.ChatID != "" {
			cfg.ChatID = bak.ChatID
		}
	}
	if cfg.TaskThreadIDs == nil {
		cfg.TaskThreadIDs = make(map[string]int64)
	}
	for k, v := range existing.TaskThreadIDs {
		if cfg.TaskThreadIDs[k] == 0 && v > 0 {
			cfg.TaskThreadIDs[k] = v
		}
	}
	if len(cfg.RecentEvents) > 20 {
		cfg.RecentEvents = cfg.RecentEvents[len(cfg.RecentEvents)-20:]
	}
	cfg.MaskedBotToken = maskTelegramBotToken(cfg.BotTokenEncrypted)
	if strings.TrimSpace(cfg.BotTokenEncrypted) != "" && strings.TrimSpace(cfg.ChatID) != "" {
		s.saveTelegramCredentialsBackupLocked(cfg.BotTokenEncrypted, cfg.ChatID, cfg.BotUsername)
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", p, time.Now().UnixNano())
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
```

### 3.2. Cập nhật `internal/webui/task_inspector.go`, `internal/webui/task_chat.go`, `internal/webui/task_edit_handler.go`, `internal/webui/subtask_wave_engine.go`
- Trong `internal/webui/task_inspector.go:3250-3288`: Xóa 3 dòng skip `TASK-1..TASK-7` và gọi `s.ensureActiveTaskTelegramTopicAndSync(qt, subTasks)` để `TASK-2` được khởi tạo Forum Sub-Topic riêng (`telegram_topic.json`) và nhận bản tin S3 Plan kèm nút `APPROVE` ngay trên Telegram.
- Trong `internal/webui/task_chat.go:922`: Gọi `go s.notifyTelegramOnWebUIChat(*qt, actionMode, agentRoleLabel, selectedModel, cleanMsg, replyText, appliedToPlan)` khi `actionMode != "ingest_audit_remediation"`.
- Trong `internal/webui/task_edit_handler.go:323`: Gọi `go s.notifyTelegramOnTaskUpdated(*updatedTask, len(updatedSubTasks))` trong nhánh `else` (`replan == false`).
- Trong `internal/webui/subtask_wave_engine.go`: Gọi `go s.notifyTelegramOnSubTaskWaveMilestone(*qt, target, phase, finalMsg)` khi Sub-Task hoàn thành `subtask_plan`, `subtask_implement`, hoặc `subtask_qa`.

---

## 4. Bổ sung Thiết kế Tự Chữa Lành `S5 -> S3 -> S4 -> S5` (`TASK-TELE-04` — Khắc phục `SA-TELE-01 .. SA-TELE-04`)

1. **Tách Module Đơn Trách Nhiệm `internal/webui/telegram_webui_sync.go` (`SA-TELE-01`):**
   - Giữ `internal/webui/telegram_config_persistence.go` chuyên biệt 100% cho **Config Persistence, Atomic Mutation & Backup Vault**.
   - Tách toàn bộ các hàm đồng bộ sự kiện từ WebUI (`ensureActiveTaskTelegramTopicAndSync`, `notifyTelegramOnWebUIChat`, `notifyTelegramOnTaskUpdated`, `notifyTelegramOnSubTaskWaveMilestone`) sang file đơn trách nhiệm `internal/webui/telegram_webui_sync.go`.
2. **Đóng kín Chu trình `Read-Modify-Write` trong 1 Khóa `telegramCfgMu` (`SA-TELE-02`):**
   ```go
   func (s *Server) updateTelegramPersistedConfig(mutator func(cfg *telegramPersistedConfig)) telegramPersistedConfig {
       telegramCfgMu.Lock()
       defer telegramCfgMu.Unlock()
       cfg := s.loadTelegramPersistedConfigLocked()
       if mutator != nil {
           mutator(&cfg)
       }
       _ = s.saveTelegramPersistedConfigLocked(cfg)
       return cfg
   }

   func (s *Server) recordTelegramUpdateOffset(updateID int64) {
       s.updateTelegramPersistedConfig(func(cfg *telegramPersistedConfig) {
           if updateID > cfg.LastUpdateID {
               cfg.LastUpdateID = updateID
           }
       })
   }

   func (s *Server) recordTelegramTaskThreadID(taskKey string, threadID int64) {
       if strings.TrimSpace(taskKey) == "" || threadID <= 0 {
           return
       }
       s.updateTelegramPersistedConfig(func(cfg *telegramPersistedConfig) {
           if cfg.TaskThreadIDs == nil {
               cfg.TaskThreadIDs = make(map[string]int64)
           }
           cfg.TaskThreadIDs[taskKey] = threadID
       })
   }
   ```
   - Thay thế các đoạn `load -> modify -> save` ngoài Mutex tại `ensureTelegramApprovePoller` (`telegram_bridge.go:2478-2483`) và `ensureForumTopicForTask` (`telegram_forum_topic.go:95-100, 146-151`) bằng `s.recordTelegramUpdateOffset(upd.UpdateID)` và `s.recordTelegramTaskThreadID(taskKey, tid)`.
3. **Đảm bảo Mọi Chunk Tin Nhắn Dài Đều Giữ Ngữ Cảnh `taskKey` (`SA-TELE-03`):**
   - Trong `sendTelegramBotMessageWithKeyboard` (`telegram_bridge.go:2187-2220`), truyền `taskKey` vào header của từng chunk `*(Phần %d/%d · [%s])*` khi chia nhiều phần để mọi chunk đều được định tuyến đúng `message_thread_id` của `taskKey`.
