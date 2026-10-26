---
tier: T2
suffix: telegram_approval_detail_and_feedback_revision
workspace: audit-master-mongo-orchestrator-2026-10-22
status: PROPOSE_ONLY_WAITING_APPROVE
created_at: 2026-10-26T13:30:30+07:00
---

# 09. Giải Pháp Kỹ Thuật & Code Demo (`S3 Solution`): Hiển Thị Đầy Đủ Hồ Sơ Duyệt Giải Pháp S3 (`Nguyên Nhân` + `Giải Pháp` + `Những Thay Đổi`) & Luồng Phản Hồi Chỉnh Sửa Giải Pháp (`Revise Solution`) Trực Tiếp Trên Telegram

> [!IMPORTANT]
> Tuân thủ nghiêm ngặt quy tắc **Single-Best Solution (Rule #9 S3)** và **Propose-Only (Rule #1A, #13)**. Không có bất kỳ file source code nào bị chỉnh sửa cho đến khi nhận được lệnh `APPROVE` từ User.

---

## 1. Phương Án Kỹ Thuật Tối Ưu Duy Nhất (Single Optimal Solution)

Giải pháp giải quyết triệt để cả **2 bài toán** mà User chỉ ra trên Telegram:
1. **Hiển thị đầy đủ nội dung chờ Approve (`Nguyên nhân` + `Giải pháp & Code Demo` + `Những thay đổi: Sub-Tasks & Danh sách Files bị ảnh hưởng + Đường dẫn file vật lý`)** trong cả bảng `/task` lẫn Bản tin Chi tiết `⏸ [APPROVAL REQUIRED — HỒ SƠ DUYỆT GIẢI PHÁP S3]`.
2. **Cung cấp luồng Phản hồi / Chỉnh sửa Giải pháp (`Revise Solution`) trực tiếp trên Telegram** khi User không đồng ý Approve và muốn góp ý sửa lại thiết kế S3 hoặc Sub-Tasks trước khi chạy `S4–S5`.

### 1.1. Bộ Trích Xuất Hồ Sơ Duyệt Giải Pháp S3 (`extractTaskApprovalDossier` trong `internal/webui/telegram_bridge.go`)
- Đọc trực tiếp từ các file vật lý của Task trong `.orchestrator/tasks/<safeKey>/` (`09_tasks_solution.md`, `13_analysis.md`, `08_tasks.md`, `diff.patch` và `git status --short` / `CandidateFiles` của `subTasks`) để xây dựng struct `TelegramApprovalDossier`:
  - **`RootCauseBullets` (`[]string`) & `RootCauseSection` (`string`):** Trích xuất các mục trong `## 1. Root Cause...` / `## 1. Tổng quan...` từ `09_tasks_solution.md` và bảng `VERIFIED file:line` từ `13_analysis.md`.
  - **`SolutionBullets` (`[]string`) & `SolutionSection` (`string`):** Trích xuất các mục trong `## 2. Giải pháp Kỹ thuật...` từ `09_tasks_solution.md` cùng các khối `## 🔄 Co-Architect Steering & Solution Revision` (nếu User vừa phản hồi sửa giải pháp).
  - **`CodeDemoPreview` (`string`):** Trích xuất khối code demo tiêu biểu từ `09_tasks_solution.md` để User đọc ngay trên điện thoại.
  - **`SubTaskLines` (`[]string`):** Liệt kê đầy đủ `1..N` Sub-Tasks theo định dạng:
    `1. [TASK-M01] (cdc-cms-service) Mở rộng API Create/Update Master Connection hỗ trợ MongoDB — Context: master_registry_handler_connections.go:172-249`
  - **`AffectedFiles` (`[]string`):** Thu thập danh sách tên file cụ thể (không trùng lặp) từ `CandidateFiles` của tất cả `subTasks`, `diff.patch` và `git status --short` (thông qua hàm mới `inspectTaskRepoDiffFilesAndSummary` trong `internal/webui/server.go:2684`).
  - **`PhysicalPaths`:** Đường dẫn đầy đủ tới `09_tasks_solution.md`, `08_tasks.md`, `13_analysis.md`, `diff.patch`.

### 1.2. Nâng Cấp Bảng `/task` (`handleTelegramTaskListCommand`) & Bản Tin `⏸ [APPROVAL REQUIRED — HỒ SƠ DUYỆT GIẢI PHÁP S3]`
1. **Trong bảng `/task` (`handleTelegramTaskListCommand`):**
   - Với các Task ở trạng thái `✅ ĐÃ XONG` hoặc `⏳ HÀNG ĐỢI`: Giữ hiển thị gọn 1 dòng.
   - Với các Task ở trạng thái **`⏸ CHỜ APPROVE (#N)` (`NEEDS_YOU`)**: Bung ngay khối tóm tắt hồ sơ chờ duyệt bên dưới dòng tên Task:
     - `📄 File Plan:` `.orchestrator/tasks/<safeKey>/09_tasks_solution.md`
     - `🔍 Nguyên nhân:` (Tóm tắt các nguyên nhân cốt lõi kèm `file:line`)
     - `💡 Giải pháp:` (Tóm tắt giải pháp kỹ thuật S3)
     - `📋 N Sub-Tasks dự kiến chạy:` (Liệt kê các Sub-Tasks kèm `[SubTaskID] (Repo) Title — file:line`)
     - `📝 M files bị ảnh hưởng:` (Liệt kê tên các files cụ thể: `internal/api/master/..., internal/service/master/...`)
   - **Nút bấm Inline Keyboard dưới bảng `/task` cho Task `⏸ CHỜ APPROVE`:**
     - Thay vì nút mù *"Chạy ngay"*, hiển thị cặp nút:
       - **`[ 👁 Xem Đủ Plan & Duyệt (#ID) ]`** (`VIEW_PLAN #<ID>`)
       - **`[ ✏️ Phản hồi / Sửa Giải Pháp (#ID) ]`** (`REVISE_PROMPT #<ID>`)
2. **Bản tin Chi tiết `⏸ [APPROVAL REQUIRED — HỒ SƠ DUYỆT GIẢI PHÁP S3]` (`notifyTelegramOnS3PlanReadyForce` & `handleTelegramViewPlanCommand` khi bấm `👁 Xem Đủ Plan & Duyệt` hoặc gõ `/plan #<ID>`):**
   - Hiển thị đầy đủ 100% **Nguyên nhân + Giải pháp + Code Demo + Đủ `1..N` Sub-Tasks + Đủ danh sách `M` Files bị ảnh hưởng + Đường dẫn file vật lý**.
   - Nếu tổng độ dài vượt `3,800` ký tự của 1 tin nhắn Telegram, hàm `sendTelegramChunkedMessageWithCustomKeyboard` tự động chia thành các phần liên tiếp (`[Phần 1/2]`, `[Phần 2/2]`) và gắn **3 nút bấm Inline Keyboard** ở phần cuối:
     - **`[ ✅ Approve & Chạy S4–S5 Ngay ]`** (`APPROVE #<ID>`)
     - **`[ ✏️ Phản hồi / Sửa Giải Pháp ]`** (`REVISE_PROMPT #<ID>`)
     - **`[ 📋 Mở Hàng Đợi Task (/task) ]`** (`/task`)

### 1.3. Luồng Phản Hồi / Chỉnh Sửa Giải Pháp Trực Tiếp Trên Telegram (`Interactive Solution Feedback & Revision Flow`)
- **Bổ sung trạng thái `PendingRevisionTaskKey` vào `telegramPersistedConfig` & `TelegramSettingsDTO`:**
  - Khi User bấm nút **`[ ✏️ Phản hồi / Sửa Giải Pháp ]`** (`REVISE_PROMPT #<ID>`), Bot lưu `cfg.PendingRevisionTaskKey = qt.ExternalID` vào `.orchestrator/telegram_settings.json` và gửi tin nhắn hướng dẫn (kèm `force_reply: true` của Telegram API để tự bật bàn phím nhập liệu trên điện thoại):
    > *"✍️ **[PHẢN HỒI / CHỈNH SỬA GIẢI PHÁP S3]***
    > *• **Task đang chọn sửa:** `<TaskKey>` (`#<ID>`)*
    > *• **File cập nhật:** `.orchestrator/tasks/<safeKey>/09_tasks_solution.md` & `08_tasks.md`*
    > *👉 Hãy **nhắn trực tiếp nội dung bạn muốn sửa hoặc bổ sung** ở tin nhắn tiếp theo (VD: `Đổi batch size mặc định thành 500 và thêm retry backoff cho BulkWrite`, hoặc `Thêm sub-task kiểm tra index MongoDB`). Brain (`Slot A`) sẽ cập nhật ngay Giải pháp S3 và gửi lại bản Plan mới để bạn duyệt! (Gõ `/cancel` để hủy)."*
- **Hỗ trợ 3 cách gửi phản hồi sửa giải pháp tự nhiên nhất:**
  1. **Cách 1 (Sau khi bấm nút `[ ✏️ Phản hồi / Sửa Giải Pháp ]`):** Bất kỳ tin nhắn văn bản nào tiếp theo của User (không phải `/task`, `RUN_TASK`, `APPROVE`, `/cancel`) sẽ tự động được áp dụng làm phản hồi sửa giải pháp cho `cfg.PendingRevisionTaskKey`!
  2. **Cách 2 (Reply trực tiếp vào tin nhắn Plan của Bot trên Telegram):** Khi User vuốt Reply vào tin nhắn của Bot có chứa tên Task / `#<ID>` và gõ nội dung góp ý (không phải lệnh `APPROVE`), Bot tự động nhận diện Task từ `replyContext` và áp dụng phản hồi sửa giải pháp!
  3. **Cách 3 (Gõ lệnh trực tiếp `/revise <Task> <nội dung>`, `/feedback <Task> <nội dung>`, hoặc `SỬA <Task>: <nội dung>`):** Áp dụng ngay lập tức trong 1 câu lệnh!
- **Xử lý cập nhật Giải pháp (`handleTelegramReviseSolutionCommand`):**
  1. Đối chiếu bài học `/agent` (`MatchAgentLessonsForTask`) và cập nhật trực tiếp khối `## 🔄 Co-Architect Steering & Solution Revision (via Telegram @<user>)` vào `.orchestrator/tasks/<safeKey>/09_tasks_solution.md` (đồng bộ cả file `09_tasks_solution_<suffix>.md`).
  2. Nếu nội dung phản hồi có yêu cầu thêm/chia Sub-Task (`thêm sub-task`, `add subtask`, `chia lại sub-task`), tự động tạo thêm Sub-Task mới vào `subtasks.json` và `08_tasks.md`.
  3. Ghi lịch sử đối thoại vào `chat_history.json` của Co-Architect Studio trên WebUI `:8090`, thu hồi `.user-approved` cũ, giữ Task ở trạng thái `NEEDS_YOU` (`⏸ CHỜ APPROVE`), xóa `PendingRevisionTaskKey`, và **bắn ngay bản tin `⏸ [APPROVAL REQUIRED — ĐÃ CẬP NHẬT THEO PHẢN HỒI CỦA BẠN]`** mới nhất lên Telegram để User kiểm tra và bấm **`[ ✅ Approve & Chạy S4–S5 Ngay ]`**!

### 1.4. Hỗ Trợ Định Danh Mọi Loại `TaskKey` (`#<ID>`, `CDC-RECON-01`, `tích hợp Master MongoDB Destination`)
- Nâng cấp `findQueuedTask(ctx, taskKey)` (`internal/webui/server.go:1051-1087`) và `extractTaskRefFromTelegramText`:
  - Hỗ trợ tìm theo `#<ID>` (VD `#1`, `#2` ánh xạ tới `qt.ID` hoặc vị trí hàng đợi), khớp chính xác `qt.ExternalID` (kể cả có khoảng trắng tiếng Việt như `tích hợp Master MongoDB Destination` hoặc nhiều dấu gạch ngang như `CDC-RECON-01`), hoặc khớp `sanitizeTaskKeyPath(qt.ExternalID)`.
  - Trong `callback_data` của các nút Inline Keyboard (`VIEW_PLAN #<ID>`, `REVISE_PROMPT #<ID>`, `APPROVE #<ID>`, `RUN_TASK #<ID>`), sử dụng mã định danh ngắn `#<ID>` khi `qt.ExternalID` dài hoặc chứa khoảng trắng để luôn ngắn gọn (`< 24 bytes`), tuyệt đối không bao giờ chạm giới hạn 64 bytes của Telegram API!

---

## 2. Code Demo Chi Tiết (Đối Chiếu `file:line` Thực Tế)

### 2.1. Mở rộng `inspectTaskRepoDiffFilesAndSummary` & `findQueuedTask` trong [`internal/webui/server.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go#L1051-L1087)

```go
// Trong internal/webui/server.go:1051-1087 — Hỗ trợ tìm Task theo #<ID>, ExternalID có khoảng trắng/nhiều dấu gạch ngang, hoặc sanitizeTaskKeyPath:
func (s *Server) findQueuedTask(ctx context.Context, taskKey string) (*queue.QueuedTask, error) {
	if s.QueueStore == nil {
		return nil, fmt.Errorf("queue store is not initialized")
	}
	all, err := s.QueueStore.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("no tasks in queue")
	}
	taskKey = strings.TrimSpace(taskKey)
	if taskKey != "" {
		cleanRef := strings.TrimPrefix(strings.ToUpper(taskKey), "#")
		if idNum, nErr := strconv.ParseInt(cleanRef, 10, 64); nErr == nil && idNum > 0 {
			for i := range all {
				if all[i].ID == idNum {
					return &all[i], nil
				}
			}
			if int(idNum) <= len(all) {
				return &all[idNum-1], nil
			}
		}
		safeTarget := sanitizeTaskKeyPath(taskKey)
		for i := range all {
			if strings.EqualFold(all[i].ExternalID, taskKey) ||
				strings.EqualFold(sanitizeTaskKeyPath(all[i].ExternalID), safeTarget) {
				return &all[i], nil
			}
		}
		for i := range all {
			if strings.Contains(strings.ToLower(all[i].ExternalID), strings.ToLower(taskKey)) ||
				strings.Contains(strings.ToLower(all[i].Title), strings.ToLower(taskKey)) {
				return &all[i], nil
			}
		}
	}
	// Fallback về selectedTaskKey hoặc task chưa DONE đầu tiên...
```

Và bổ sung `inspectTaskRepoDiffFilesAndSummary` tại [`internal/webui/server.go:2684-2725`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go#L2684-L2725) trả về cả danh sách tên file `[]string` (kết hợp `diff.patch`, `git status --short`, và `CandidateFiles` từ `subTasks`):

```go
func (s *Server) inspectTaskRepoDiffFilesAndSummary(ctx context.Context, qt queue.QueuedTask, iso TaskIsolationDTO, affectedRepos []string, subTasks []SubTaskDTO) ([]string, string) {
	seenFiles := make(map[string]bool)
	var orderedFiles []string
	addFile := func(f string) {
		f = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(f, "b/"), "a/"))
		f = strings.TrimPrefix(f, "./")
		if f == "" || seenFiles[f] {
			return
		}
		seenFiles[f] = true
		orderedFiles = append(orderedFiles, f)
	}
	if rawDiff, err := os.ReadFile(iso.DiffFile); err == nil && len(rawDiff) > 0 {
		for _, ln := range strings.Split(string(rawDiff), "\n") {
			ln = strings.TrimSpace(ln)
			if strings.HasPrefix(ln, "diff --git ") {
				parts := strings.Fields(ln)
				if len(parts) >= 4 {
					addFile(parts[3])
				}
			}
		}
	}
	for _, st := range subTasks {
		rawRefs := st.ContextFileLine + " " + st.TargetFiles
		for _, rel := range extractCandidateRelPaths(rawRefs, st.TargetRepoName) {
			cleanRel := strings.TrimSuffix(strings.TrimPrefix(rel, "./"), "/")
			if cleanRel != "" && strings.Contains(filepath.Base(cleanRel), ".") {
				addFile(cleanRel)
			}
		}
	}
	// ... thu thập thêm từ git status --short trên affectedRepos ...
	return orderedFiles, summary
}
```

### 2.2. Bộ Trích Xuất Hồ Sơ Duyệt Giải Pháp S3 & Luồng Phản Hồi Sửa Giải Pháp trong [`internal/webui/telegram_bridge.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L60-L1074)

```go
// TelegramApprovalDossier chứa đầy đủ Nguyên nhân, Giải pháp, Code Demo và Những thay đổi (Sub-Tasks + Files + Paths)
// để hiển thị trên Telegram khi Task ở trạng thái ⏸ CHỜ APPROVE (NEEDS_YOU).
type TelegramApprovalDossier struct {
	TaskRefToken      string
	RootCauseBullets  []string
	RootCauseSection  string
	SolutionBullets   []string
	SolutionSection   string
	CodeDemoPreview   string
	LatestRevision    string
	SubTaskLines      []string
	AffectedFiles     []string
	PlanFilePath      string
	TasksFilePath     string
	AnalysisFilePath  string
	DiffFilePath      string
}

func (s *Server) extractTaskApprovalDossier(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO, subTasks []SubTaskDTO) TelegramApprovalDossier {
	refToken := fmt.Sprintf("#%d", qt.ID)
	if qt.ID <= 0 && !strings.ContainsAny(qt.ExternalID, " \t") && len(qt.ExternalID) <= 24 {
		refToken = qt.ExternalID
	}
	tasksFile := filepath.Join(isoMeta.ArtifactDir, "08_tasks.md")
	analysisFile := filepath.Join(isoMeta.ArtifactDir, "13_analysis.md")

	rawPlanBytes, _ := os.ReadFile(isoMeta.PlanFile)
	rawAnalysisBytes, _ := os.ReadFile(analysisFile)
	planText := strings.TrimSpace(string(rawPlanBytes))
	analysisText := strings.TrimSpace(string(rawAnalysisBytes))

	rcBullets, rcSection := extractMarkdownRootCauseSection(planText, analysisText, qt)
	solBullets, solSection, codeDemo, latestRev := extractMarkdownSolutionAndDemoSection(planText, qt)

	var subLines []string
	for idx, st := range subTasks {
		repoLbl := st.TargetRepoName
		if repoLbl == "" {
			repoLbl = filepath.Base(st.TargetRepo)
		}
		ctxLine := strings.TrimSpace(st.ContextFileLine)
		if ctxLine == "" {
			ctxLine = strings.TrimSpace(st.TargetFiles)
		}
		subLines = append(subLines, fmt.Sprintf("%d. `[%s]` (%s) %s — `%s`",
			idx+1, st.SubTaskID, repoLbl, truncateRunes(st.Title, 90), truncateRunes(ctxLine, 75)))
	}

	affectedFiles, _ := s.inspectTaskRepoDiffFilesAndSummary(ctx, qt, isoMeta, s.ResolveTaskAffectedRepos(qt), subTasks)

	return TelegramApprovalDossier{
		TaskRefToken:     refToken,
		RootCauseBullets: rcBullets,
		RootCauseSection: rcSection,
		SolutionBullets:  solBullets,
		SolutionSection:  solSection,
		CodeDemoPreview:  codeDemo,
		LatestRevision:   latestRev,
		SubTaskLines:     subLines,
		AffectedFiles:    affectedFiles,
		PlanFilePath:     isoMeta.PlanFile,
		TasksFilePath:    tasksFile,
		AnalysisFilePath: analysisFile,
		DiffFilePath:     isoMeta.DiffFile,
	}
}
```

Và luồng xử lý phản hồi chỉnh sửa giải pháp trên Telegram (`REVISE_PROMPT`, Reply tin nhắn, hoặc `/revise`) trong [`internal/webui/telegram_bridge.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L589-L678):

```go
// Khi User bấm nút [ ✏️ Phản hồi / Sửa Giải Pháp ] (REVISE_PROMPT #<ID>):
func (s *Server) handleTelegramRevisePromptCommand(ctx context.Context, explicitTaskKey, cleanText, chatID, fromUser string) (string, error) {
	qt, err := s.resolveTelegramTargetTask(ctx, explicitTaskKey, cleanText)
	if err != nil {
		return "", err
	}
	isoMeta, _ := s.ensureTaskIsolation(ctx, *qt)
	cfg := s.loadTelegramPersistedConfig()
	cfg.PendingRevisionTaskKey = qt.ExternalID
	cfg.LastNotifiedTask = qt.ExternalID
	if strings.TrimSpace(chatID) != "" && strings.TrimSpace(cfg.ChatID) == "" {
		cfg.ChatID = strings.TrimSpace(chatID)
	}
	_ = s.saveTelegramPersistedConfig(cfg)

	promptMsg := fmt.Sprintf(
		"✍️ *[CHẾ ĐỘ PHẢN HỒI / SỬA GIẢI PHÁP S3]*\n"+
			"• *Task đang chọn sửa:* `%s` (`#%d` — %s)\n"+
			"• *File Giải pháp:* `%s`\n"+
			"• *File Sub-Tasks:* `%s`\n\n"+
			"👉 *Hãy nhắn trực tiếp nội dung bạn muốn chỉnh sửa ở tin nhắn tiếp theo* (hoặc gõ `/revise #%d <nội dung>`):\n"+
			"• VD1: `Đổi batch size mặc định thành 500 và bổ sung retry backoff + DLQ cho BulkWrite MongoDB`\n"+
			"• VD2: `Thêm sub-task kiểm tra index _gpay_id và _source_ts trên MongoDB đích`\n\n"+
			"🧠 *BRAIN (`Slot A`)* sẽ lập tức cập nhật `09_tasks_solution.md` & `08_tasks.md` và gửi lại Hồ sơ Duyệt Giải pháp mới ngay trên Telegram để bạn bấm *✅ Approve & Chạy*! (Gõ `/cancel` nếu muốn hủy).",
		qt.ExternalID, qt.ID, qt.Title,
		isoMeta.PlanFile, filepath.Join(isoMeta.ArtifactDir, "08_tasks.md"),
		qt.ID,
	)
	_ = s.sendTelegramBotMessageForceReply(ctx, cfg, promptMsg)
	s.appendTelegramEvent("OUTBOUND_REVISE_PROMPT", qt.ExternalID, fmt.Sprintf("✍️ @%s bật chế độ Phản hồi / Sửa Giải pháp S3 cho %s — Đang chờ tin nhắn góp ý tiếp theo", fromUser, qt.ExternalID), "WAITING_FEEDBACK")
	return qt.ExternalID, nil
}

// Khi User nhắn nội dung phản hồi sửa giải pháp (tin nhắn tiếp theo sau REVISE_PROMPT, Reply vào bản tin Plan, hoặc gõ /revise <Task> <nội dung>):
func (s *Server) handleTelegramReviseSolutionCommand(ctx context.Context, qt *queue.QueuedTask, feedbackText, chatID, fromUser string) (string, error) {
	isoMeta, _ := s.ensureTaskIsolation(ctx, *qt)
	subTasks := s.loadOrGenerateSubTasks(*qt)
	cliCfg := s.loadCLISettings()
	selectedModel := cliCfg.PlannerModel
	if selectedModel == "" {
		selectedModel = "gemini-4-argon (antigravity)"
	}

	// 1. Cập nhật trực tiếp vào 09_tasks_solution.md (và 08_tasks.md nếu thêm Sub-Task) + chat_history.json
	updatedSubTasks, targetFile := s.applyTelegramSolutionFeedbackToPlan(ctx, qt, isoMeta, subTasks, feedbackText, fromUser, selectedModel)

	// 2. Xóa PendingRevisionTaskKey sau khi đã áp dụng phản hồi
	cfg := s.loadTelegramPersistedConfig()
	cfg.PendingRevisionTaskKey = ""
	cfg.LastNotifiedTask = qt.ExternalID
	_ = s.saveTelegramPersistedConfig(cfg)

	// 3. Bắn ngay bản tin ⏸ [APPROVAL REQUIRED — ĐÃ CẬP NHẬT THEO PHẢN HỒI CỦA BẠN] đầy đủ kèm nút [ ✅ Approve & Chạy S4–S5 Ngay ]!
	reason := fmt.Sprintf("Đã cập nhật Giải pháp S3 theo phản hồi của @%s: %s", fromUser, truncateRunes(feedbackText, 70))
	deliveryStatus := s.notifyTelegramOnS3PlanReadyForce(ctx, *qt, isoMeta, updatedSubTasks, reason, true)
	s.appendTelegramEvent("INBOUND_REVISE_S3", qt.ExternalID, fmt.Sprintf("✍️ @%s phản hồi sửa giải pháp %s (%s): %q -> Đã cập nhật %s & gửi lại Plan mới (%s)", fromUser, qt.ExternalID, targetFile, truncateRunes(feedbackText, 80), filepath.Base(targetFile), deliveryStatus), "REVISED_NEEDS_YOU")
	return qt.ExternalID, nil
}
```

---

## 3. Phân Tích Rủi Ro & Blast Radius
- **Không ảnh hưởng luồng `APPROVE` và `/task` hiện hữu:** Mọi lệnh `APPROVE`, `/task`, `RUN_TASK` cũ vẫn hoạt động 100% tương thích ngược; đồng thời các Task có tên tiếng Việt dài (`tích hợp Master MongoDB Destination`) hoặc nhiều dấu gạch ngang (`CDC-RECON-01`) nay được định danh chính xác qua cả tên lẫn mã `#<ID>`.
- **Không bị lỗi HTTP 400 của Telegram khi nội dung dài:** Hàm `sendTelegramChunkedMessageWithCustomKeyboard` tự động tách văn bản theo dòng ở ngưỡng an toàn `3,800` ký tự và có cơ chế fallback sang plain-text nếu Markdown vướng ký tự đặc biệt.
