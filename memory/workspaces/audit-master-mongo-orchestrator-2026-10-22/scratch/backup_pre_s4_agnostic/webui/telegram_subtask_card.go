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
	"regexp"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/notify"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

var (
	telegramAbsWorkPathRe     = regexp.MustCompile(`/Users/[^\s` + "`" + `"'|)]+/([^/\s` + "`" + `"'|)]+/[^/\s` + "`" + `"'|)]+)`)
	telegramSubTaskReplyRefRe = regexp.MustCompile("(?i)(?:SUB-TASK\\s+\\d+/\\d+\\s*·\\s*`([^`]+)`|Sub-Task\\s*\\[\\d+/\\d+\\]:\\*\\s*`([^`]+)`|REVISE_SUB_PROMPT\\s+#\\d+\\s+([A-Za-z0-9_-]+))")
)

// stripBotMentionAnywhere removes @<bot_username> mentions from slash commands and plain-text feedback messages.
func stripBotMentionAnywhere(text string) string {
	return notify.StripBotMentionAnywhere(text)
}

// extractSubTaskRefFromReplyContext extracts a Sub-Task ID (e.g. CORE-01-S01 or TASK-M01) from a replied-to Telegram message.
func extractSubTaskRefFromReplyContext(replyContext string) string {
	rc := strings.TrimSpace(replyContext)
	if rc == "" {
		return ""
	}
	if m := telegramSubTaskReplyRefRe.FindStringSubmatch(rc); len(m) > 0 {
		for i := 1; i < len(m); i++ {
			if strings.TrimSpace(m[i]) != "" {
				return strings.TrimSpace(m[i])
			}
		}
	}
	return ""
}

// isPureTelegramQuestion returns true only when a message is genuinely asking a question (and not giving a revision directive).
func isPureTelegramQuestion(text string) bool {
	cleaned := stripBotMentionAnywhere(text)
	lower := strings.ToLower(cleaned)
	if lower == "" || strings.HasPrefix(lower, "/") {
		return false
	}
	// If the user gives an explicit revision/change instruction, treat it as feedback revision unless it ends with '?'
	if !strings.HasSuffix(lower, "?") {
		for _, directive := range []string{
			"sửa ", "sua ", "đổi ", "doi ", "thêm ", "them ", "bổ sung", "bo sung",
			"cập nhật", "cap nhat", "bỏ ", "xóa ", "xoa ", "chuyển ", "chuyen ",
			"tăng ", "tang ", "giảm ", "giam ", "dùng ", "dung ", "áp dụng", "ap dung",
			"không dùng", "ko dùng", "phải ", "phai ",
		} {
			if strings.Contains(lower, directive) {
				return false
			}
		}
	}
	if strings.HasSuffix(lower, "?") {
		return true
	}
	for _, qTok := range []string{"vì sao", "tại sao", "lỗi gì", "sao ko", "sao không", "là gì", "như nào", "thế nào"} {
		if strings.Contains(lower, qTok) {
			return true
		}
	}
	return false
}

// resolveInboundTelegramFeedbackTarget resolves the target Task reference and optional Sub-Task ID for any non-slash Telegram message.
func (s *Server) resolveInboundTelegramFeedbackTarget(ctx context.Context, explicitTaskKey, cleanText, replyContext string, cfg telegramPersistedConfig) (taskRef string, subTaskID string) {
	if subFromReply := extractSubTaskRefFromReplyContext(replyContext); subFromReply != "" {
		subTaskID = subFromReply
	}
	// 1. If the user is replying to a specific Bot card/prompt (`reply_to_message`), honor that message's task reference first!
	if ref := extractTaskRefFromReplyContext(replyContext); ref != "" {
		return ref, subTaskID
	}
	// 2. If the user explicitly wrote `#43` or `Core-01` inside their message text
	if ref := extractTaskRefFromCommand(cleanText); ref != "" {
		return ref, subTaskID
	}
	// 3. If the message arrived inside a dedicated Forum Sub-Topic thread
	if strings.TrimSpace(explicitTaskKey) != "" {
		return strings.TrimSpace(explicitTaskKey), subTaskID
	}
	// 4. If the user previously clicked `✏️ Phản hồi / Sửa Giải Pháp` (`PendingRevisionTaskKey`)
	if strings.TrimSpace(cfg.PendingRevisionTaskKey) != "" {
		if subTaskID == "" {
			subTaskID = strings.TrimSpace(cfg.PendingRevisionSubTaskID)
		}
		return strings.TrimSpace(cfg.PendingRevisionTaskKey), subTaskID
	}
	// 5. Fallback to the most recently viewed/notified task on Telegram
	if strings.TrimSpace(cfg.LastNotifiedTask) != "" {
		return strings.TrimSpace(cfg.LastNotifiedTask), subTaskID
	}
	// 6. Fallback to the currently selected task on the Web UI
	s.mu.Lock()
	sel := strings.TrimSpace(s.selectedTaskKey)
	s.mu.Unlock()
	if sel != "" {
		return sel, subTaskID
	}
	if s.QueueStore != nil {
		if all, err := s.QueueStore.ListAll(ctx); err == nil && len(all) > 0 {
			for _, qt := range all {
				if strings.EqualFold(qt.Stage, queue.StageNeedsYou) {
					return qt.ExternalID, subTaskID
				}
			}
			return all[0].ExternalID, subTaskID
		}
	}
	return "", subTaskID
}

// sendTelegramPromptWithForceReply sends a prompt message with Telegram ForceReply markup so the user's client automatically opens the Reply box to @tk001sea_bot (bypassing Group Privacy Mode).
func (s *Server) sendTelegramPromptWithForceReply(ctx context.Context, cfg telegramPersistedConfig, text string, placeholder string) string {
	token := strings.TrimSpace(cfg.BotTokenEncrypted)
	chatID := strings.TrimSpace(cfg.ChatID)
	if token == "" || chatID == "" {
		return "QUEUED_LOCAL_BRIDGE (Chưa nhập BotToken/ChatID thật — Sẵn sàng nhận Webhook/Test APPROVE)"
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	if len([]rune(placeholder)) > 60 {
		placeholder = truncateRunes(placeholder, 60)
	}
	if placeholder == "" {
		placeholder = "Nhập nội dung phản hồi / chỉnh sửa giải pháp..."
	}
	forceReplyMarkup := map[string]any{
		"force_reply":             true,
		"selective":               true,
		"input_field_placeholder": placeholder,
	}

	var threadID int64
	if qt := s.inferTaskForOutboundTelegramMessage(ctx, cfg, text); qt != nil {
		threadID = s.ensureTelegramTaskTopic(ctx, cfg, *qt, false)
	}

	lastStatus := "LOCAL_FALLBACK"
	for _, useMarkdown := range []bool{true, false} {
		bodyMap := map[string]any{
			"chat_id":      chatID,
			"text":         text,
			"reply_markup": forceReplyMarkup,
		}
		if threadID > 0 {
			bodyMap["message_thread_id"] = threadID
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
		lastStatus = fmt.Sprintf("TELEGRAM_HTTP_%d", resp.StatusCode)
	}
	return lastStatus
}

// cleanCompactTelegramBullet strips UDF-v1 metadata noise, generic executive summary boilerplate, long filesystem paths, and Markdown table syntax.
func cleanCompactTelegramBullet(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	for _, skipSub := range []string{
		"*task:*", "**task:**",
		"*target repo:*", "**target repo:**",
		"*hồ sơ kiến trúc:*", "**hồ sơ kiến trúc:**",
		"*trạng thái:*", "**trạng thái:**",
		"chẩn đoán trọng tâm", "nguyên tắc thực thi & cô lập", "trạng thái cổng",
		"đã quét trực tiếp", "rule #12, #13, #19",
		"| field", "| :---", "|---", "| bước", "| sub-task", "| hạng mục",
	} {
		if strings.Contains(lower, skipSub) {
			return ""
		}
	}
	if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
		cells := strings.Split(strings.Trim(trimmed, "|"), "|")
		var nonEmpty []string
		for _, c := range cells {
			cTrim := strings.TrimSpace(c)
			if cTrim != "" && !strings.HasPrefix(cTrim, ":---") && !strings.HasPrefix(cTrim, "---") {
				nonEmpty = append(nonEmpty, cTrim)
			}
		}
		if len(nonEmpty) == 0 {
			return ""
		}
		trimmed = "• " + strings.Join(nonEmpty, " — ")
	}
	trimmed = cleanTelegramMarkdownLine(trimmed)
	trimmed = telegramAbsWorkPathRe.ReplaceAllString(trimmed, "$1")
	trimmed = strings.TrimLeft(trimmed, "-•* \t")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}
	return "• " + trimmed
}

func filterCompactTelegramBullets(lines []string, maxItems int) []string {
	if maxItems <= 0 {
		maxItems = 3
	}
	var out []string
	seen := make(map[string]bool)
	for _, ln := range lines {
		cleaned := cleanCompactTelegramBullet(ln)
		if cleaned == "" {
			continue
		}
		key := strings.ToLower(cleaned)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, cleaned)
		if len(out) >= maxItems {
			break
		}
	}
	return out
}

// formatTelegramCompactSubTaskCard renders a concise 1-card review summary for a single Sub-Task [k/N] in Wave W.
func (s *Server) formatTelegramCompactSubTaskCard(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO, subTasks []SubTaskDTO, targetIdx int, headerNote string) (string, [][]map[string]string) {
	_ = ctx
	_ = isoMeta
	total := len(subTasks)
	if targetIdx < 0 || targetIdx >= total {
		targetIdx = 0
	}
	st := subTasks[targetIdx]
	doneCnt := 0
	for _, item := range subTasks {
		if strings.EqualFold(item.Status, "done") || strings.EqualFold(item.Stage, "DONE") {
			doneCnt++
		}
	}

	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	safeSub := sanitizeTaskKeyPath(st.SubTaskID)
	subPlanFile := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSub, "09_tasks_solution.md")
	rawSubPlan, _ := os.ReadFile(subPlanFile)
	rcLines, solLines, revLines, _ := extractMarkdownRootCauseAndSolution(string(rawSubPlan), "")

	var sb strings.Builder
	if strings.TrimSpace(headerNote) != "" {
		sb.WriteString(headerNote + "\n\n")
	}
	relTouch := normalizeOrchestratorDisplayPath(st.ContextFileLine, st.TargetRepoName)
	sb.WriteString(fmt.Sprintf("🧩 **[SUB-TASK REVIEW · BƯỚC %d/%d · WAVE %d (#%d)]**\n", targetIdx+1, total, st.WaveIndex, qt.ID))
	sb.WriteString(fmt.Sprintf("• **Task cha:** `%s` (#%d) · **Tiến độ:** `%s`\n", qt.ExternalID, qt.ID, renderSubTaskProgressBar(doneCnt, total)))
	sb.WriteString(fmt.Sprintf("• **Sub-Task [%d/%d]:** `%s` — **%s**\n", targetIdx+1, total, st.SubTaskID, st.Title))
	sb.WriteString(fmt.Sprintf("• **Điểm chạm:** `📦 %s` · 📄 `%s`\n", st.TargetRepoName, relTouch))

	if len(revLines) > 0 {
		sb.WriteString("\n✏️ **Phản hồi Sub-Task vừa cập nhật:**\n")
		sb.WriteString("  " + cleanTelegramMarkdownLine(revLines[len(revLines)-1]) + "\n")
	}

	sb.WriteString("\n🔍 **1. Mục tiêu & Giải pháp:**\n")
	for _, b := range filterCompactTelegramBullets(append(rcLines, solLines...), 3) {
		sb.WriteString("  " + b + "\n")
	}
	sb.WriteString(fmt.Sprintf("\n🎯 **2. Nghiệm thu (DoD):** %s\n", st.TestDoD))

	refToken := fmt.Sprintf("#%d", qt.ID)
	if qt.ID <= 0 {
		refToken = qt.ExternalID
	}
	rows := [][]map[string]string{
		{
			{
				"text":          fmt.Sprintf("✅ Approve Sub-Task %d/%d (%s) & Chạy", targetIdx+1, total, st.SubTaskID),
				"callback_data": fmt.Sprintf("APPROVE_SUB %s %s", refToken, st.SubTaskID),
			},
		},
		{
			{
				"text":          fmt.Sprintf("✏️ Sửa Plan Sub-Task %d/%d (%s)", targetIdx+1, total, st.SubTaskID),
				"callback_data": fmt.Sprintf("REVISE_SUB_PROMPT %s %s", refToken, st.SubTaskID),
			},
			{
				"text":          fmt.Sprintf("📋 Bảng %d Sub-Tasks", total),
				"callback_data": fmt.Sprintf("LIST_SUBTASKS %s", refToken),
			},
		},
		{
			{
				"text":          fmt.Sprintf("⚡ Approve All Còn Lại (Chạy hết %d/%d -> S5)", total-doneCnt, total),
				"callback_data": fmt.Sprintf("APPROVE_ALL %s", refToken),
			},
		},
	}
	return sb.String(), rows
}

// formatTelegramCoArchitectReply converts UDF-v1 Markdown into a clean, single-card Telegram message without raw pipe tables.
func formatTelegramCoArchitectReply(qt queue.QueuedTask, rawReply string) string {
	lines := strings.Split(strings.ReplaceAll(rawReply, "\r\n", "\n"), "\n")
	var out []string
	out = append(out, fmt.Sprintf("🧠 **[CO-ARCHITECT · #%d `%s`]**", qt.ID, qt.ExternalID))
	inCode := false
	codeLines := 0
	fenceLang := detectCodeFenceLang("", rawReply)
	if fenceLang == "" {
		fenceLang = "go"
	}
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") {
			if !inCode {
				inCode = true
				codeLines = 0
				tag := strings.TrimSpace(strings.TrimPrefix(t, "```"))
				if tag == "ts" || tag == "typescript" {
					tag = "typescript"
				} else if tag == "" {
					tag = fenceLang
				}
				out = append(out, "```"+tag)
			} else {
				inCode = false
				out = append(out, "```")
			}
			continue
		}
		if inCode {
			if codeLines < 10 {
				out = append(out, ln)
				codeLines++
			}
			continue
		}
		if t == "" || strings.HasPrefix(t, "---") {
			continue
		}
		if strings.HasPrefix(t, "#") {
			h := cleanTelegramMarkdownLine(strings.TrimLeft(t, "# "))
			out = append(out, "\n**"+h+"**")
			continue
		}
		if b := cleanCompactTelegramBullet(t); b != "" {
			out = append(out, b)
		}
	}
	joined := strings.Join(out, "\n")
	return truncateRunes(joined, 2800)
}
