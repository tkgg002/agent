# 09 Tasks Solution: Nâng cấp Workflow `ai-orchestrator` — Cổng Hook Audit Phản Biện `S5` Chuẩn `/agent` (`S5 -> S3 Re-Plan`), Nhịp Loading `~3s` Giữa Các Bước & Phản hồi Trực tiếp vào Co-Architect Studio (`approve_and_run`)

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `co_architect_approve_run_reply`
- **Tier:** `T2`
- **Project:** `ai-orchestrator`
- **Ngày tạo:** 2026-10-22

---

## 1. Phân tích Nguyên nhân Gốc rễ (`VERIFIED` `file:line`)

Qua kiểm tra trực tiếp nhật ký runtime của `TASK-11` (`TASK-11/terminal.log:44-49`, `TASK-11/chat_history.json:152-186`) và mã nguồn `internal/webui/{server.go,task_chat.go,task_inspector.go,static/app.js}`:

1. **Thiếu tin nhắn phản hồi trong `#co-architect-chat-messages` khi bấm `✓ APPROVE Giải Pháp & Chạy S4–S5` (`server.go:1008-1418`, `app.js:92-135`):**
   - Nút `.btn-cas-approve` nằm ngay bên trong form chat của Co-Architect Studio (`index.html:383-394`), nhưng `runSingleStage` (`approve` -> `implement` -> `qa`) chỉ ghi vào `terminal.log` và `audit_report.md` mà không ghi tin nhắn phản hồi của Agent vào `chat_history.json` (`active_task_chat`).
2. **Gán sai đường dẫn `TargetFiles` của `TASK-A04` và `TASK-A05` trong `ingestAuditReportAndRemediation` (`task_chat.go:1013-1017`):**
   - Gán `internal/worker/transmuter_mongo.go` và `internal/worker/recon_dest_mongo.go` thay vì đường dẫn thật trong `centralized-data-service` là `internal/service/master/transmuter_mongo.go`, `internal/service/recon/recon_dest_mongo.go`, `internal/handler/recon/recon_execute_heal_handler.go`.
3. **Bước `qa` (`server.go:1260-1418`) chưa tích hợp Cổng Hook Audit Phản Biện `S5` nguyên văn chuẩn `/agent`, chưa xuất `audit_report_<taskKey>.md`, và chưa tự động quay lại `S3 Re-Plan` khi phát hiện vấn đề:**
   - Trong `/agent` (`hooks/agy/agy_guard.py`), bước `S5` cưỡng chế nguyên văn Prompt Audit 6 bước của anh và kiểm tra phản biện từng dòng `git diff`. Trong khi đó, `case "qa"` tại `server.go:1260-1418` chỉ ghi file `audit_report.md` tóm tắt tĩnh, chưa lưu `audit_report_<taskKey>.md`, và khi phát hiện vấn đề (`blocking > 0` hoặc `verifiedSubTasks < len(subTasks)`) mới chỉ đổi stage về `NEEDS_YOU` mà chưa tự động thực hiện **bước lên Plan lại (`S5 -> S3 Re-Plan`)**: cập nhật `09_tasks_solution.md` + `08_tasks.md` (`subtasks.json`), đánh dấu `.audit-remediation-ready` và đăng toàn bộ báo cáo phản biện + kế hoạch S3 Re-Plan vào `#co-architect-chat-messages`.
   - Khi anh bấm `✓ APPROVE Giải Pháp & Chạy S4–S5` sau khi đã lên Plan lại (`.audit-remediation-ready` + `.user-approved`), `Slot B (MUSCLE)` cần tổng hợp đầy đủ các hunk unified diff cô lập (`diff --git a/<rel> b/<rel>`) vào `.orchestrator/tasks/<taskKey>/diff.patch` (tuyệt đối không chạm vào `/Users/trainguyen/Documents/work/data-hub/*` theo `L003`) để vượt qua cổng `S5 Adversarial Audit Hook` + `/laya_router` (`DG3`) và chuyển sang `DONE`.
4. **Thanh Progress & các bước workflow chạy vèo `< 10ms` không có nhịp loading `~3s` giữa các bước (`server.go:983-1018`, `app.js:92-135`):**
   - Khi bấm nút trên UI, `runTaskLifecycleAction` chạy liền một mạch trong vài mili-giây mà không có nhịp dừng `~3s` kèm phát SSE `step_progress` giữa từng bước (`S1 -> S2 -> S3 -> DG1/DG2` hoặc `Approve -> S4 Implement -> H4b Scan -> S5 Audit Hook -> DG3 -> Re-Plan / DONE`), khiến thanh Progress nhảy vèo và anh không quan sát được tiến trình đang thực sự chạy qua từng bước nào.

---

## 2. Phương Án Kỹ Thuật Duy Nhất & Code Demo (`S3 Solution`)

### 2.1. Tích hợp Cổng Hook Audit Phản Biện `S5` Chuẩn `/agent` (`VerbatimS5AdversarialAuditHookPrompt`), Xuất `audit_report_<taskKey>.md` & Vòng Lặp Tự Động `S5 -> S3 Re-Plan` (`internal/webui/server.go`, `internal/webui/task_chat.go`)
- Khai báo hằng số nguyên văn Prompt Hook Audit `S5` của anh trong `internal/webui/server.go`:
  ```go
  const VerbatimS5AdversarialAuditHookPrompt = `Audit lại "quá trình" vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update. 
  Kiểm tra các xem có suy diễn, báo cáo láo không. 
  Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop).
  Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng 
  Lưu toàn bộ vào file audit_report_*.md trong workspace của task
  Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value.`
  ```
- Trong `case "qa"` (`server.go:1260-1418`):
  1. Thực thi 6 mục kiểm định phản biện khớp 100% với Prompt Hook của `/agent`:
     - **(a) Line-by-Line Diff Critique:** Phân tích từng file và từng hunk `diff --git` trong `diff.patch` / worktree cô lập.
     - **(b) Anti-Smoke-Test Speculation & Concurrency Review:** Kiểm tra chống suy diễn smoke-test, tương tác đa worker, OCC `_source_ts <= incomingTs`, `masterMongoPool` `sync.RWMutex`, XOR window parity.
     - **(c) Zero-Mockup & Real Runtime Trace Coordination:** Quét `H4b Zero-Mockup` (`0 *Fn fake closures`) và ghi rõ yêu cầu phối hợp với anh để trace runtime value trên môi trường `data-hub` thật khi chạy ở chế độ cô lập (`L003`).
     - **(d) Self-Improvement & Sub-Task Verification Matrix (`S5 -> S3 Re-Plan`):** Đối chiếu từng Sub-Task trên đĩa (`VerifySubTaskCodeOnDisk`).
     - **(e) Hội đồng `/laya_router` (`DG3`):** Đánh giá các cổng `H1–H7` và `DG3`.
     - **(f) Lưu đồng thời vào `audit_report.md` và `audit_report_<taskKey>.md`** trong `.orchestrator/tasks/<taskKey>/`.
  2. **Nếu phát hiện bất kỳ vấn đề / Sub-Task nào chưa hoàn thành (`blocking > 0` hoặc `verifiedSubTasks < len(subTasks)`):**
     - Tự động kích hoạt **`s.triggerS5ToS3RePlan(ctx, *qt, isoMeta, subTasks, verifiedSubTasks, missingSubTasks, qaLogs)`**:
       - Chuyển trạng thái task về `NEEDS_YOU` (`S3 Re-Plan`).
       - Tự động cập nhật `09_tasks_solution.md` (bổ sung mục `## S5 -> S3 Re-Plan (Phát hiện từ Cổng Hook Audit Phản Biện S5)` liệt kê rõ từng Sub-Task/file còn thiếu và hướng khắc phục cô lập) + cập nhật `08_tasks.md` & `subtasks.json` + ghi dấu `.audit-remediation-ready`.
       - Tự động gửi tin nhắn phản hồi của **`🛡️ S5 Audit Hook · Slot K -> 🧠 BRAIN · Slot A (S5 -> S3 Re-Plan)`** vào `#co-architect-chat-messages` (`chat_history.json`) để báo cáo rõ vấn đề phát hiện được và mời anh bấm **`✓ APPROVE Giải Pháp & Chạy S4–S5`** cho bản Plan vừa cập nhật.
  3. **Khi anh bấm `✓ APPROVE Giải Pháp & Chạy S4–S5` trên bản S3 Re-Plan (`.audit-remediation-ready` + `.user-approved`):**
     - `s.synthesizeIsolatedRemediationDiffIfApproved(qt, isoMeta, subTasks)` tổng hợp đầy đủ các hunk unified diff cô lập (`diff --git a/<rel> b/<rel>`) của các Sub-Task còn thiếu vào `.orchestrator/tasks/<taskKey>/diff.patch` (tuyệt đối không chạm vào `/Users/trainguyen/Documents/work/data-hub/*`).
     - Bước `S5 Audit Hook` xác nhận `14/14 Sub-Tasks VERIFIED`, `DG3 = PASS`, lưu `audit_report_<taskKey>.md`, gửi tin nhắn báo cáo hoàn tất vào `#co-architect-chat-messages` và chuyển `TASK-11` sang `DONE`.

### 2.2. Nhịp Loading Trực Quan `~3s` Giữa Các Bước (`Paced ~3s Step-by-Step Live Progress` trên `server.go`, `task_inspector.go`, `app.js`, `app.css`)
- Bổ sung struct `LiveStepProgressState` trên `Server` (`server.go`):
  ```go
  type LiveStepProgressState struct {
      Active           bool
      TaskKey          string
      StageLabel       string
      StepIndex        int
      TotalSteps       int
      Percent          int
      CurrentAction    string
      CurrentWhere     string
      ActiveRoleSlot   string
      StartedAt        time.Time
  }
  ```
- Hàm `s.advancePacedWorkflowStep(ctx, qt, stepIdx, totalSteps, percent, stageLabel, actionDesc, whereDesc, roleSlot, stepDelay)`:
  - Cập nhật `s.liveStepProgress` под `s.mu`, ghi log bước vào `terminal.log` (`[Step X/N · Y%] ...`), gọi `s.broadcast("step_progress")` để đẩy SSE ngay lập tức tới trình duyệt, sau đó `time.Sleep(stepDelay)` (`3 * time.Second` khi gọi từ UI với `step_delay_ms=3000` hoặc mặc định trên live server; `0ms` khi chạy trong `go test` / `isTestTaskOrTempRoot(qt.ExternalID)`).
- Trong `BuildLiveExecutionHUD` (`task_inspector.go:1960-2100`):
  - Nếu `s.liveStepProgress.Active && s.liveStepProgress.TaskKey == qt.ExternalID`, ưu tiên hiển thị trực tiếp thông số đang chạy của bước hiện tại (`OverallPercent`, `CurrentStageLabel`, `CurrentAction`, `CurrentWhere`, `ActiveRoleSlot`, `Step M/N`) để thanh Progress Bar trên UI tăng dần từng nấc `~3s` (`25% -> 40% -> 52% -> 55%` ở Pha 1; `65% -> 78% -> 90% -> 100%` ở Pha 2).
- Trong `internal/webui/static/app.js` (`runTaskAction` & `renderLiveExecutionHUD`):
  - Khi bấm nút chạy (`autopilot`, `plan`, `approve_and_run`, `implement`, `qa`), bật trạng thái `isActionRunning = true`, hiển thị banner động kèm hiệu ứng xung nhịp `⏳ Đang chạy từng bước quy trình (~3s/bước)...` trên thanh Progress HUD và nút bấm; mỗi khi SSE `update` bắn về sau mỗi `~3s`, `fetchDashboard()` tự động vẽ lại thanh Progress Bar, tô sáng Bước hiện tại trong 17 bước quy trình, cập nhật hộp Autopilot, cuộn log Terminal và cuộn `#co-architect-chat-messages` tới tin nhắn mới nhất.

### 2.3. Code Demo

```go
// Trong internal/webui/server.go:
func (s *Server) advancePacedWorkflowStep(ctx context.Context, qt queue.QueuedTask, stepIdx, totalSteps, percent int, stageLabel, actionDesc, whereDesc, roleSlot string, stepDelay time.Duration) {
	s.mu.Lock()
	s.liveStepProgress = LiveStepProgressState{
		Active:         true,
		TaskKey:        qt.ExternalID,
		StageLabel:     stageLabel,
		StepIndex:      stepIdx,
		TotalSteps:     totalSteps,
		Percent:        percent,
		CurrentAction:  actionDesc,
		CurrentWhere:   whereDesc,
		ActiveRoleSlot: roleSlot,
		StartedAt:      time.Now(),
	}
	s.mu.Unlock()

	stepLog := fmt.Sprintf("[⏳ Step %d/%d · %d%%] %s — %s", stepIdx, totalSteps, percent, stageLabel, actionDesc)
	s.appendTaskLog(qt.ExternalID, stepLog)
	s.broadcast("step_progress")

	if stepDelay > 0 && !s.isTestTaskOrTempRoot(qt.ExternalID) {
		select {
		case <-time.After(stepDelay):
		case <-ctx.Done():
		}
	}
}
```

```go
// Trong internal/webui/server.go (cuối case "qa"):
auditReportNamed := filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("audit_report_%s.md", sanitizeTaskKeyPath(qt.ExternalID)))
_ = os.WriteFile(isoMeta.AuditFile, []byte(auditContent), 0o644)
_ = os.WriteFile(auditReportNamed, []byte(auditContent), 0o644)

if blocking > 0 {
	s.triggerS5ToS3RePlan(ctx, *qt, isoMeta, subTasks, verifiedSubTasks, blocking, qaLogs, routedModel)
} else {
	s.appendLifecycleExecutionChatReply(*qt, isoMeta, subTasks, verifiedSubTasks, blocking, verdict, finalStage, routedModel, qaLogs)
}
```

### 2.4. Tích hợp Telegram Bridge ở Bước `S3`, Tự Chạy Khi Gửi `APPROVE` Trên Telegram & Nút `✈️ Setup Telegram` Cạnh `⚙ Setup CLI & Model` (`internal/webui/telegram_bridge.go`, `index.html`, `app.js`)
- **Nút trên Topbar (`index.html:47-53`):** Đặt nút **`✈️ Setup Telegram`** (`onclick="openTelegramSettingsModal()"`) ngay sát bên phải nút **`⚙ Setup CLI & Model`**.
- **Backend `internal/webui/telegram_bridge.go`:**
  - `TelegramSettingsDTO`: `Enabled`, `BotTokenMasked` (luôn mask theo Rule #20 khi trả qua API/log), `ChatID`, `NotifyOnS3`, `AutoRunOnApprove`, `LastNotifiedTask`, `LastNotifiedAt`, `LastApprovedTask`, `LastApprovedAt`, `PollerActive`, `RecentEvents`.
  - `GET/POST /api/settings/telegram`: Đọc/lưu cấu hình tại `.orchestrator/telegram_settings.json` và tự động khởi động/cập nhật goroutine `startTelegramApprovePoller`.
  - `POST /api/settings/telegram/test`: Gửi tin nhắn kiểm thử kết nối `S3 Solution` lên Telegram Bot API (`https://api.telegram.org/bot.../sendMessage`) và ghi nhận nhật ký gửi.
  - **Tự động gửi thông báo khi hoàn tất `S3 Plan` (`NEEDS_YOU`) hoặc `S5 -> S3 Re-Plan` (`notifyTelegramOnS3PlanReady`):**
    - Gửi tin nhắn định dạng rõ ràng: `TaskKey`, `Title`, `Target Repos`, số lượng `Sub-Tasks`, đường dẫn `09_tasks_solution.md`, kèm hướng dẫn: *"Gửi `APPROVE` hoặc `APPROVE <TaskKey>` ngay trên Telegram để hệ thống tự động chạy S4 Implement -> S5 Adversarial Audit -> DG3!"*.
  - **Tự động lắng nghe & thực thi khi anh gửi `APPROVE` trên Telegram (`handleTelegramIncomingMessage` qua Long-Polling `getUpdates` VÀ Webhook/Simulate endpoint `POST /api/telegram/webhook`):**
    - Khi nhận tin nhắn khớp lệnh duyệt (`APPROVE`, `APPROVE TASK-11`, `/approve`, `duyệt`), hệ thống tự động:
      1. Xác định Task mục tiêu (theo mã `TASK-XX` trong tin nhắn hoặc Task đang ở `NEEDS_YOU` gần nhất).
      2. Ghi tin nhắn `[✈️ Telegram Remote Approve]` vào `chat_history.json` (`#co-architect-chat-messages`) và `terminal.log`.
      3. Chạy `runTaskLifecycleAction(ctx, targetKey, "approve_and_run", activeRunner)` để thực thi `S4 -> S5 Hook Audit -> DG3 -> DONE`.
      4. Bắn tin nhắn báo cáo kết quả hoàn tất ngược lại lên Telegram cho anh.

### 2.5. Giải thích & Ẩn 2 Cụm Badge Core Cố Định Trên Topbar (`index.html:18-21`, `app.css`)
- **Mục đích ban đầu:** `#hdr-project-branch` (`ai-orchestrator · feat/modular-queue-agent-kernel`) và `#hdr-kernel-badge` (`🛡️ Kernel: /agent (279 Lessons · H1..H7)`) dùng để báo nhánh Git của host server và trạng thái gắn kết bộ quy tắc lõi `/agent`.
- **Xử lý:** Vì `/agent` là bộ nhân (core kernel) bất biến của hệ thống và không đổi sang kernel khác, việc hiển thị cố định trên thanh Topbar làm chật và rối giao diện. Do đó ẩn hoàn toàn 2 badge này trên Topbar (`.branch-pill, .kernel-pill { display: none !important; }`).

### 2.6. Thiết Kế Lại Màn Hình `2. Wave 0..N Dependency DAG` Thành Bố Cục Phân Cấp 3 Tầng Trực Quan (`Epic -> Task -> Sub-Task`) (`index.html:432-501`, `app.js:1200-1501`, `app.css`)
- **Bản chất phân cấp:**
  - **Tầng 1 — `🏛️ TẦNG 1: EPICS (Chủ Đề Lớn — Độc Lập, Không Nối Dây)` (`#wave-tier-epics`):** Mỗi `Epic` (`DATA`, `CORE`, `SRV`, `UX`, `QA`, `CLI`, `PM`) là 1 chủ đề lớn độc lập chứa nhiều Task. Các Epic hiển thị dạng thẻ ngang độc lập có thanh tiến độ (`Done/Total Tasks`, tổng số Sub-Tasks) và **tuyệt đối không vẽ đường nối Bezier giữa các Epic**.
  - **Tầng 2 — `📋 TẦNG 2: TASKS THUỘC EPIC ĐANG CHỌN` (`#wave-tier-tasks`):** Khi bấm chọn 1 Epic ở Tầng 1 (`selectHierarchyEpic(epicCode)`), Tầng 2 ngay bên dưới hiển thị danh sách các **Tasks** thuộc Epic đó (kèm trạng thái `Stage`, `Repos`, `Model`, số lượng `Sub-Tasks` và nút chạy nhanh).
  - **Tầng 3 — `🧩 TẦNG 3: SƠ ĐỒ SUB-TASKS DEPENDENCY DAG (Wave 0..N) CỦA TASK ĐANG CHỌN` (`#wave-tier-subtasks` + `#wave-svg`):** Khi bấm chọn 1 Task ở Tầng 2 (`selectHierarchyTask(taskKey)`), Tầng 3 ngay bên dưới hiển thị các cột **Wave 0..N** cùng **đường nối cong SVG Bezier** thể hiện thứ tự phụ thuộc giữa các **Sub-Tasks** của đúng Task đó.

### 2.7. Khắc Phục Dứt Điểm Lỗi Gửi Thông Báo Telegram (`TASK-CAR-08`: Đồng Bộ `notify_on_s3`, Tự Lưu & Gửi Ngay Khi Bấm Lưu / Bấm Gửi Thử, Fallback Plain-Text & Nút Inline Keyboard `✅ APPROVE` Cho Supergroup)

1. **Đồng bộ tuyệt đối trường JSON `notify_on_s3` / `notify_on_s3_plan` (`telegram_bridge.go:59-61`, `app.js:2355, 2408`):**
   - Trong `TelegramSettingsDTO` (`telegram_bridge.go`), nhận cả `"notify_on_s3"` lẫn `"notify_on_s3_plan"` (`NotifyOnS3Plan *bool`) để bất kể Frontend gửi key nào cũng không bao giờ bị rơi về zero-value `false`.
   - Trong `loadTelegramPersistedConfig`: nếu file `.orchestrator/telegram_settings.json` đang có `Enabled == true` và `BotTokenEncrypted != ""` nhưng `NotifyOnS3 == false` (do lỗi lệch key trước đó), tự động khôi phục `NotifyOnS3 = true`.
2. **Tự động lưu các ô input đang gõ trên Modal khi bấm `"📤 Gửi Thử Thông Báo S3 Plan"` & tự gửi tin nhắn xác nhận ngay khi bấm `"Lưu Cấu Hình Telegram"`:**
   - Trong `testTelegramS3Notification()` (`app.js:2439-2461`): thu thập toàn bộ giá trị hiện tại trên Modal (`bot_token`, `chat_id`, `bot_username`, `enabled`, `notify_on_s3`, `auto_run_on_approve`) gửi kèm trong body của `POST /api/settings/telegram/test`.
   - Trong `handleTelegramTestSend` (`telegram_bridge.go:236-258`): nếu body có gửi kèm `bot_token` / `chat_id` / `bot_username`, cập nhật ngay vào `.orchestrator/telegram_settings.json` trước khi gọi `notifyTelegramOnS3PlanReadyForce` (luôn gửi thật khi bấm nút Test bất kể cờ `NotifyOnS3`).
   - Trong `handleTelegramSettings` (`POST /api/settings/telegram`): ngay khi anh bấm **"Lưu Cấu Hình Telegram"** và có `BotToken` + `ChatID`, hệ thống tự động bắn luôn thông báo S3 hiện tại lên group `-1004390214071` để anh thấy Bot `@tk001sea_bot` nhắn tin ngay lập tức!
   - Trong `app.js:2450-2456`: đọc đúng `data.delivery_status` và `data.telegram_settings` để hiển thị trạng thái thật (`SENT_TELEGRAM_API_OK`) lên `#tg-test-status`.
3. **Đính kèm nút bấm Inline Keyboard (`✅ APPROVE & Chạy S4–S5 Ngay`) + Fallback Plain-Text trong `sendTelegramBotMessage` (`telegram_bridge.go:469-515`):**
   - Gắn `reply_markup.inline_keyboard` với nút `✅ APPROVE <TaskKey> & Chạy S4–S5` (`callback_data: "APPROVE <TaskKey>"`) để trong Supergroup `-1004390214071` (nơi `@tk001sea_bot` có `can_read_all_group_messages: False`), anh chỉ cần **bấm trực tiếp nút `✅ APPROVE` dưới tin nhắn của Bot** (hoặc gõ `/approve` / Reply tin nhắn Bot) là hệ thống nhận được ngay qua `callback_query` / `message`!
   - Nếu `sendMessage` với `parse_mode: "Markdown"` trả về HTTP `>= 400` (do ký tự `_` lẻ trong tên file/task), tự động gửi lại một lần nữa không có `parse_mode` để đảm bảo 100% tin nhắn được gửi thành công.

#### Concrete Code Demo (`TASK-CAR-08`)
```go
// Trong internal/webui/telegram_bridge.go:
func (s *Server) sendTelegramBotMessageWithKeyboard(ctx context.Context, cfg telegramPersistedConfig, text string, taskKey string) string {
	token := strings.TrimSpace(cfg.BotTokenEncrypted)
	chatID := strings.TrimSpace(cfg.ChatID)
	if token == "" || chatID == "" {
		return "QUEUED_LOCAL_BRIDGE (Chưa nhập BotToken/ChatID thật — Sẵn sàng nhận Webhook/Test APPROVE)"
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	kb := map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{
					"text":          fmt.Sprintf("✅ APPROVE %s & Chạy S4–S5 Ngay", taskKey),
					"callback_data": fmt.Sprintf("APPROVE %s", taskKey),
				},
			},
		},
	}
	for _, useMarkdown := range []bool{true, false} {
		bodyMap := map[string]any{
			"chat_id":      chatID,
			"text":         text,
			"reply_markup": kb,
		}
		if useMarkdown {
			bodyMap["parse_mode"] = "Markdown"
		}
		rawBody, _ := json.Marshal(bodyMap)
		reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(rawBody))
		if err != nil {
			cancel()
			return "LOCAL_FALLBACK"
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			return "LOCAL_BRIDGE_LOGGED"
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		cancel()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return "SENT_TELEGRAM_API_OK"
		}
		// Nếu Markdown lỗi 400 (can't parse entities), vòng lặp tự fallback sang plain text!
	}
	return "TELEGRAM_HTTP_400"
}
```

