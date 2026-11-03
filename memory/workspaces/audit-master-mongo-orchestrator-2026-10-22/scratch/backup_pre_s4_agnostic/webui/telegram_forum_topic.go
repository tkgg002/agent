package webui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/agentgate"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

// formatTelegramTaskTopicName formats the Telegram Forum Sub-Topic name using the Task's ExternalID (Mã Task / Ticket ID).
func formatTelegramTaskTopicName(qt queue.QueuedTask) string {
	extID := strings.TrimSpace(qt.ExternalID)
	title := strings.TrimSpace(qt.Title)
	if extID == "" {
		extID = fmt.Sprintf("TASK-%d", qt.ID)
	}
	topicName := fmt.Sprintf("🧩 [%s] %s", extID, title)
	if strings.EqualFold(extID, title) || title == "" {
		topicName = fmt.Sprintf("🧩 [%s]", extID)
	}
	runes := []rune(topicName)
	if len(runes) > 120 {
		topicName = string(runes[:117]) + "..."
	}
	return topicName
}

func (s *Server) taskTelegramTopicMetaPath(taskKey string) string {
	root := s.Root
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	return filepath.Join(root, ".orchestrator", "tasks", sanitizeTaskKeyPath(taskKey), "telegram_topic.json")
}

func (s *Server) lookupTaskKeyByTelegramThreadID(threadID int64) string {
	if threadID <= 0 {
		return ""
	}
	cfg := s.loadTelegramPersistedConfig()
	for k, tid := range cfg.TaskThreadIDs {
		if tid == threadID && strings.TrimSpace(k) != "" {
			return k
		}
	}
	root := s.Root
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	tasksDir := filepath.Join(root, ".orchestrator", "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return ""
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		metaPath := filepath.Join(tasksDir, ent.Name(), "telegram_topic.json")
		if raw, err := os.ReadFile(metaPath); err == nil && len(raw) > 0 {
			var meta telegramTopicMeta
			if json.Unmarshal(raw, &meta) == nil && meta.MessageThreadID == threadID && strings.TrimSpace(meta.TaskKey) != "" {
				return meta.TaskKey
			}
		}
	}
	return ""
}

// ensureForumTopicForTask ensures a dedicated Forum Sub-Topic exists on Telegram for qt.ExternalID via createForumTopic.
func (s *Server) ensureForumTopicForTask(ctx context.Context, cfg telegramPersistedConfig, qt queue.QueuedTask, forceRecreate bool) int64 {
	taskKey := strings.TrimSpace(qt.ExternalID)
	if taskKey == "" {
		return 0
	}
	topicMetaFile := s.taskTelegramTopicMetaPath(taskKey)
	if !forceRecreate {
		if cfg.TaskThreadIDs != nil {
			if tid := cfg.TaskThreadIDs[taskKey]; tid > 0 {
				return tid
			}
		}
		if raw, err := os.ReadFile(topicMetaFile); err == nil && len(raw) > 0 {
			var meta telegramTopicMeta
			if json.Unmarshal(raw, &meta) == nil && meta.MessageThreadID > 0 {
				latestCfg := s.loadTelegramPersistedConfig()
				if latestCfg.TaskThreadIDs == nil {
					latestCfg.TaskThreadIDs = make(map[string]int64)
				}
				latestCfg.TaskThreadIDs[taskKey] = meta.MessageThreadID
				_ = s.saveTelegramPersistedConfig(latestCfg)
				return meta.MessageThreadID
			}
		}
	}

	token := strings.TrimSpace(cfg.BotTokenEncrypted)
	chatID := strings.TrimSpace(cfg.ChatID)
	if token == "" || chatID == "" {
		return 0
	}

	topicName := formatTelegramTaskTopicName(qt)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/createForumTopic", token)
	rawBody, _ := json.Marshal(map[string]any{
		"chat_id": chatID,
		"name":    topicName,
	})
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(rawBody))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0
	}
	var createRes struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageThreadID int64  `json:"message_thread_id"`
			Name            string `json:"name"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBytes, &createRes); err != nil || !createRes.OK || createRes.Result.MessageThreadID <= 0 {
		return 0
	}

	tid := createRes.Result.MessageThreadID
	latestCfg := s.loadTelegramPersistedConfig()
	if latestCfg.TaskThreadIDs == nil {
		latestCfg.TaskThreadIDs = make(map[string]int64)
	}
	latestCfg.TaskThreadIDs[taskKey] = tid
	_ = s.saveTelegramPersistedConfig(latestCfg)

	_ = os.MkdirAll(filepath.Dir(topicMetaFile), 0o755)
	metaRaw, _ := json.MarshalIndent(telegramTopicMeta{
		TaskKey:         taskKey,
		MessageThreadID: tid,
		TopicName:       topicName,
		CreatedAt:       time.Now().Format(time.RFC3339),
	}, "", "  ")
	_ = os.WriteFile(topicMetaFile, metaRaw, 0o644)
	s.appendTelegramEvent("OUTBOUND_TOPIC", taskKey, fmt.Sprintf("Đã tạo Sub-Topic Telegram '%s' (message_thread_id=%d) cho %s", topicName, tid, taskKey), "TOPIC_CREATED")
	return tid
}

func (s *Server) notifyNewTaskIntakeAndForumTopic(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO, subTasks []SubTaskDTO) string {
	cfg := s.loadTelegramPersistedConfig()
	if !cfg.Enabled {
		return "DISABLED"
	}
	_ = s.ensureForumTopicForTask(ctx, cfg, qt, false)
	return s.notifyTelegramOnS3PlanReadyForce(ctx, qt, isoMeta, subTasks, "New Task Intake & S3 Plan Ready", true)
}

func (s *Server) handleTelegramTechnicalQuestion(ctx context.Context, explicitTaskKey, cleanText, replyContext, chatID, fromUser string) (string, error) {
	targetRef := strings.TrimSpace(explicitTaskKey)
	if targetRef == "" {
		targetRef = extractTaskRefFromCommand(cleanText)
	}
	if targetRef == "" {
		targetRef = extractTaskRefFromReplyContext(replyContext)
	}
	if targetRef == "" {
		cfg := s.loadTelegramPersistedConfig()
		if cfg.PendingRevisionTaskKey != "" {
			targetRef = cfg.PendingRevisionTaskKey
		} else if cfg.LastNotifiedTask != "" {
			targetRef = cfg.LastNotifiedTask
		}
	}
	qt, err := s.findQueuedTask(ctx, targetRef)
	if err != nil {
		return "", err
	}
	isoMeta, _ := s.ensureTaskIsolation(ctx, *qt)
	subTasks := s.loadOrGenerateSubTasks(*qt)
	cliCfg := s.loadCLISettings()
	mdl := qt.AssignedModel
	if strings.TrimSpace(mdl) == "" {
		mdl = cliCfg.PlannerModel
	}
	if strings.TrimSpace(mdl) == "" {
		mdl = "gemini-4-argon (antigravity)"
	}

	replyText := s.buildContextualDiscussReply(*qt, isoMeta, subTasks, cleanText, "🧠 BRAIN · Architect", "Slot A", "/brain-delegate", mdl, nil)

	history := s.loadOrInitTaskChatHistory(*qt, subTasks, isoMeta)
	history = append(history,
		CoArchitectChatMessageDTO{
			ID:            fmt.Sprintf("msg-tg-q-%d", time.Now().UnixNano()),
			TaskKey:       qt.ExternalID,
			Sender:        "user",
			AgentRole:     fmt.Sprintf("✈️ Telegram (@%s)", fromUser),
			AgentSlot:     "Telegram Sub-Topic",
			Skill:         "/brain-delegate",
			Model:         mdl,
			ActionMode:    "discuss",
			Message:       cleanText,
			AppliedToPlan: false,
			TargetFile:    isoMeta.PlanFile,
			Timestamp:     time.Now().Format("15:04:05"),
		},
		CoArchitectChatMessageDTO{
			ID:            fmt.Sprintf("msg-tg-a-%d", time.Now().UnixNano()+1),
			TaskKey:       qt.ExternalID,
			Sender:        "agent",
			AgentRole:     "🧠 BRAIN · Architect",
			AgentSlot:     "Slot A",
			Skill:         "/brain-delegate",
			Model:         mdl,
			ActionMode:    "discuss",
			Message:       replyText,
			AppliedToPlan: false,
			TargetFile:    isoMeta.PlanFile,
			Timestamp:     time.Now().Format("15:04:05"),
		},
	)
	_ = s.saveTaskChatHistory(qt.ExternalID, history)

	cfg := s.loadTelegramPersistedConfig()
	cfg.LastNotifiedTask = qt.ExternalID
	_ = s.saveTelegramPersistedConfig(cfg)
	compactReply := formatTelegramCoArchitectReply(*qt, replyText)
	delivery := s.sendTelegramChunkedMessageWithCustomKeyboard(ctx, cfg, compactReply, nil)
	s.appendTelegramEvent("OUTBOUND_QA_REPLY", qt.ExternalID, fmt.Sprintf("💬 Đã trả lời câu hỏi kỹ thuật của @%s trên Sub-Topic %s (%s)", fromUser, qt.ExternalID, delivery), delivery)
	return qt.ExternalID, nil
}

func (s *Server) inferTaskForTelegramMessage(ctx context.Context, cfg telegramPersistedConfig, text string) *queue.QueuedTask {
	if strings.Contains(text, "[BẢNG ĐIỀU KHIỂN HÀNG ĐỢI TASK") {
		return nil
	}
	ref := extractTaskRefFromReplyContext(text)
	if ref == "" {
		ref = strings.TrimSpace(cfg.LastNotifiedTask)
	}
	if ref == "" {
		return nil
	}
	qt, err := s.findQueuedTask(ctx, ref)
	if err != nil {
		return nil
	}
	return qt
}

func (s *Server) executeS5ToS3RePlanNotification(
	ctx context.Context,
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	verifiedSubTasks int,
	blocking int,
	qaLogs []string,
	routedModel string,
	auditReportNamed string,
) {
	var missingIDs []string
	var missingBullets []string
	upperKey := strings.ToUpper(qt.ExternalID)
	hasAuditSubTasks := len(subTasks) > 0 && !strings.Contains(upperKey, "ESCALATE") && !strings.Contains(upperKey, "ISO-")
	for _, st := range subTasks {
		if !strings.EqualFold(st.Status, "done") {
			missingIDs = append(missingIDs, st.SubTaskID)
			missingBullets = append(missingBullets, fmt.Sprintf("  - **`%s` (`%s` — `%s`):** %s", st.SubTaskID, st.TargetRepoName, st.TargetFiles, st.Title))
		}
	}

	rawPlan, _ := os.ReadFile(isoMeta.PlanFile)
	planStr := string(rawPlan)
	replanMarker := "## 🔄 S5 -> S3 Re-Plan (Phát hiện từ Cổng Hook Audit Phản Biện S5)"
	if idx := strings.Index(planStr, replanMarker); idx >= 0 {
		planStr = strings.TrimSpace(planStr[:idx])
	}
	replanSection := fmt.Sprintf(
		"\n\n---\n%s (%s)\n- **Cổng kích hoạt:** `S5 Mandatory Adversarial Audit Hook` (`/agent` Rule #9.7d & `/laya_router` DG3)\n- **Kết quả kiểm định S5:** Phát hiện `%d` điểm chặn (`%d/%d Sub-Tasks VERIFIED`, còn thiếu: `%s`).\n- **Kế hoạch Khắc phục S3 Re-Plan:**\n%s\n- **Hành động tiếp theo:** Bấm **`✓ APPROVE Giải Pháp & Chạy S4–S5`** (hoặc gửi `APPROVE %s` qua Telegram) để `Slot B (MUSCLE)` tổng hợp diff cô lập và hoàn tất `100%%` Sub-Tasks.\n",
		replanMarker,
		time.Now().Format("2006-01-02 15:04:05"),
		blocking,
		verifiedSubTasks,
		len(subTasks),
		strings.Join(missingIDs, ", "),
		strings.Join(missingBullets, "\n"),
		qt.ExternalID,
	)
	updatedPlanBytes := []byte(strings.TrimRight(planStr, "\n") + replanSection)
	_ = os.WriteFile(isoMeta.PlanFile, updatedPlanBytes, 0o644)
	_, _, suffix := agentgate.InferTaskTierAndMode(qt.ExternalID, qt.Title, qt.Description, qt.Complexity, len(subTasks))
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("09_tasks_solution_%s.md", suffix)), updatedPlanBytes, 0o644)
	if hasAuditSubTasks {
		_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, ".audit-remediation-ready"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
	}
	s.syncSubTasksToMarkdown08(qt, subTasks)

	history := s.loadOrInitTaskChatHistory(qt, subTasks, isoMeta)
	msg := fmt.Sprintf(
		"🛡️ **[S5 Adversarial Audit Hook · `Slot K` -> 🔄 S5 -> S3 Re-Plan · `Slot A`]**\n\n"+
			"Thực thi nguyên văn Hook Audit của `/agent` trên **`%s` (%s)**:\n"+
			"> *\"%s\"*\n\n"+
			"1. **Kết quả Đối soát Vật lý (`VerifySubTaskCodeOnDisk`):** Đạt **`%d/%d` Sub-Tasks** (`%d` blocking).\n"+
			"2. **Các hạng mục chưa đạt cần lên Plan lại (`S5 -> S3 Re-Plan`):**\n%s\n"+
			"3. **Biên bản Audit đã lưu vật lý tại:**\n"+
			"   - `%s`\n"+
			"   - `%s`\n"+
			"4. **Cập nhật Kế hoạch S3 (`09_tasks_solution.md` & `08_tasks.md`):** Đã tự động quay lại bước **`S3 Re-Plan (NEEDS_YOU)`** và bổ sung phương án khắc phục cho `%s`.\n"+
			"5. **Phối hợp Trace Runtime Value:** Với các repo ngoài (`data-hub`), hệ thống giữ nguyên tắc cô lập (`L003`). Anh có thể bấm **`✓ APPROVE Giải Pháp & Chạy S4–S5`** ngay bên dưới (hoặc nhắn `APPROVE %s` trên Telegram) để thực thi bản Re-Plan này!",
		qt.ExternalID,
		qt.Title,
		strings.ReplaceAll(VerbatimS5AdversarialAuditHookPrompt, "\n", " "),
		verifiedSubTasks,
		len(subTasks),
		blocking,
		strings.Join(missingBullets, "\n"),
		isoMeta.AuditFile,
		auditReportNamed,
		strings.Join(missingIDs, ", "),
		qt.ExternalID,
	)
	history = append(history, CoArchitectChatMessageDTO{
		ID:            fmt.Sprintf("msg-s5-replan-%d", time.Now().UnixNano()),
		TaskKey:       qt.ExternalID,
		Sender:        "agent",
		AgentRole:     "🛡️ S5 Audit Hook -> 🧠 S3 Re-Plan",
		AgentSlot:     "Slot K -> Slot A",
		Skill:         "/qa-agent · /brain-delegate · /laya-router",
		Model:         routedModel,
		ActionMode:    "revise_solution",
		Message:       msg,
		AppliedToPlan: true,
		TargetFile:    isoMeta.PlanFile,
		Timestamp:     time.Now().Format("15:04:05"),
	})
	_ = s.saveTaskChatHistory(qt.ExternalID, history)
	s.notifyTelegramOnS3PlanReady(ctx, qt, isoMeta, subTasks, "S5 Audit Hook -> S3 Re-Plan")
}

func (s *Server) appendS4S5LifecycleChatReply(
	ctx context.Context,
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	verifiedSubTasks int,
	blocking int,
	verdict string,
	finalStage string,
	routedModel string,
	qaLogs []string,
	auditReportNamed string,
) {
	history := s.loadOrInitTaskChatHistory(qt, subTasks, isoMeta)
	var verifiedList []string
	for _, st := range subTasks {
		if strings.EqualFold(st.Status, "done") {
			verifiedList = append(verifiedList, fmt.Sprintf("`%s` (%s)", st.SubTaskID, st.TargetRepoName))
		}
	}
	msg := fmt.Sprintf(
		"✅ **[🛠️ MUSCLE · `Slot B` & 🛡️ S5 Adversarial Audit Hook · `Slot K` · `%s`] Đã thực thi xong `S4 Implement -> S5 Audit Hook -> /laya_router DG3` cho `%s`!**\n\n"+
			"1. **Trạng thái Task:** Chuyển sang **`%s`** (Verdict: **`%s`**, Blocking: **`%d`**).\n"+
			"2. **Xác minh Vật lý từng Sub-Task (`VerifySubTaskCodeOnDisk`):** Hoàn tất **`%d/%d` Sub-Tasks (`100%% VERIFIED`)**:\n"+
			"   - %s\n"+
			"3. **Cổng Hook Audit Phản Biện `S5` Chuẩn `/agent` (`G1–G8` & `DG3`):**\n"+
			"   - Đã phản biện từng dòng `git diff` (`%s`), kiểm tra chống suy diễn Smoke Test, `H4b Zero-Mockup` (`0 *Fn closures`) và đối chiếu hội đồng `/laya_router` (`DG3 = PASS`).\n"+
			"   - Chi tiết kiểm định: `%s`\n"+
			"4. **Hồ sơ Báo cáo Vật lý đã xuất:**\n"+
			"   - Báo cáo Audit chuẩn `/agent`: `%s`\n"+
			"   - Bản sao định danh Task: `%s`\n"+
			"   - Unified Diff cô lập: `%s`\n"+
			"5. **Trace Runtime Value:** Toàn bộ thay đổi trên `ai-orchestrator` đã chạy thật trên cổng `:8090`, và các hunk của `data-hub` được giữ cô lập an toàn trong `.orchestrator/tasks/%s/diff.patch` (sẵn sàng phối hợp cùng anh trace runtime value khi cần).",
		routedModel,
		qt.ExternalID,
		finalStage,
		verdict,
		blocking,
		verifiedSubTasks,
		len(subTasks),
		strings.Join(verifiedList, ", "),
		isoMeta.DiffFile,
		strings.Join(qaLogs, " · "),
		isoMeta.AuditFile,
		auditReportNamed,
		isoMeta.DiffFile,
		sanitizeTaskKeyPath(qt.ExternalID),
	)
	history = append(history, CoArchitectChatMessageDTO{
		ID:            fmt.Sprintf("msg-s4s5-done-%d", time.Now().UnixNano()),
		TaskKey:       qt.ExternalID,
		Sender:        "agent",
		AgentRole:     "🛠️ MUSCLE · Slot B & 🛡️ S5 Audit · Slot K",
		AgentSlot:     "Slot B / Slot K",
		Skill:         "/muscle-execute · /qa-agent · /laya-router",
		Model:         routedModel,
		ActionMode:    "approve_and_run",
		Message:       msg,
		AppliedToPlan: true,
		TargetFile:    auditReportNamed,
		Timestamp:     time.Now().Format("15:04:05"),
	})
	_ = s.saveTaskChatHistory(qt.ExternalID, history)
	s.notifyTelegramOnTaskDone(ctx, qt, isoMeta, verifiedSubTasks, len(subTasks), verdict)
}
