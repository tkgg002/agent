package webui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/notify"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

func buildCleanKeyDiffPreview(qt queue.QueuedTask, primaryFile, codeFence, rawSnippet string) string {
	_ = qt
	cleaned := strings.TrimSpace(rawSnippet)
	if cleaned != "" && !strings.Contains(cleaned, "private dbConnection: Connection;") && !strings.Contains(cleaned, "// TargetRepo: /") {
		lines := strings.Split(cleaned, "\n")
		var out []string
		for _, ln := range lines {
			t := strings.TrimSpace(ln)
			if strings.HasPrefix(t, "// Trích xuất trực tiếp từ TargetRepo:") {
				continue
			}
			out = append(out, ln)
			if len(out) >= 14 {
				break
			}
		}
		if len(out) > 0 {
			return strings.TrimSpace(strings.Join(out, "\n"))
		}
	}
	if codeFence == "typescript" {
		return fmt.Sprintf("// Điểm chạm chính: %s\nexport async function executeTaskStep(ctx: Record<string, unknown>): Promise<void> {\n  // Giữ nguyên signature interface cho tầng Service\n}", filepath.Base(primaryFile))
	}
	return fmt.Sprintf("// Điểm chạm chính: %s\nfunc ExecuteIsolatedTaskStep(ctx context.Context) error {\n\treturn nil\n}", filepath.Base(primaryFile))
}

// formatTelegramCompactApprovalCard formats the H4b S3 Solution Approval Dossier into the canonical 4-Section Engineering Manager Template (1 single Telegram message).
func (s *Server) formatTelegramCompactApprovalCard(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO, subTasks []SubTaskDTO, reasonLabel string) (string, [][]map[string]string) {
	_ = reasonLabel
	d := s.extractTaskApprovalDossier(ctx, qt, isoMeta, subTasks)
	llmAnalysis := s.loadTaskLLMAnalysis(qt.ExternalID)
	affectedRepos := s.ResolveTaskAffectedRepos(qt)
	primaryRepoPath := ""
	if len(affectedRepos) > 0 {
		primaryRepoPath = affectedRepos[0]
	}
	dedupFiles := normalizeAndDeduplicateRepoRelativePaths(d.ChangedFiles, d.AffectedRepos...)
	if len(dedupFiles) == 0 {
		var fallbackRaw []string
		for _, st := range subTasks {
			fallbackRaw = append(fallbackRaw, st.TargetFiles, st.ContextFileLine)
		}
		dedupFiles = normalizeAndDeduplicateRepoRelativePaths(fallbackRaw, d.AffectedRepos...)
	}
	stackProfile := detectRepoTechStackProfile(primaryRepoPath, dedupFiles)

	primaryFile := "module.go"
	if len(dedupFiles) > 0 {
		primaryFile = dedupFiles[0]
		for _, f := range dedupFiles {
			if strings.Contains(strings.ToLower(f), "model") || strings.Contains(strings.ToLower(f), "service") || strings.Contains(strings.ToLower(f), "handler") {
				primaryFile = f
				break
			}
		}
	} else if len(subTasks) > 0 {
		primaryFile = normalizeOrchestratorDisplayPath(subTasks[0].TargetFiles, d.AffectedRepos...)
	}
	primaryBase := filepath.Base(primaryFile)
	if primaryBase == "" || primaryBase == "." {
		primaryBase = "module.go"
	}

	safeTaskKey := sanitizeTaskKeyPath(d.TaskKey)
	worktreeRel := fmt.Sprintf(".orchestrator/worktrees/%s", safeTaskKey)
	branchName := strings.TrimSpace(isoMeta.BranchName)
	if branchName == "" {
		branchName = fmt.Sprintf("task/%s", safeTaskKey)
	}

	reqSummary := strings.TrimSpace(d.Title)
	if llmAnalysis != nil && strings.TrimSpace(llmAnalysis.DirectAnswer) != "" {
		reqSummary = strings.TrimSpace(llmAnalysis.DirectAnswer)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⏸ **[H4b GATE · DUYỆT GIẢI PHÁP S3]** — `%s` (#%d)\n", d.TaskKey, d.TaskID))
	sb.WriteString(fmt.Sprintf("🎯 **Yêu cầu & Tóm tắt:** %s\n", reqSummary))
	sb.WriteString(fmt.Sprintf("📦 **Target Repo:** `%s` *(%s)*\n", strings.Join(d.AffectedRepos, ", "), stackProfile.Label))
	sb.WriteString(fmt.Sprintf("🌿 **Worktree / Branch:** `%s` (`%s`)\n", worktreeRel, branchName))

	if len(d.UserRevisions) > 0 {
		lastRev := cleanTelegramMarkdownLine(d.UserRevisions[len(d.UserRevisions)-1])
		sb.WriteString(fmt.Sprintf("✏️ **Chỉ đạo vừa cập nhật:** %s\n", lastRev))
	}

	sb.WriteString("\n---\n")
	sb.WriteString("🏗 **1. KIẾN TRÚC & CHIẾN LƯỢC CHUYỂN ĐỔI (ARCHITECTURE PLAN)**\n")
	if llmAnalysis != nil && (len(llmAnalysis.Architecture) > 0 || len(llmAnalysis.Solutions) > 0) {
		for _, a := range llmAnalysis.Architecture {
			sb.WriteString("• " + a + "\n")
		}
		for _, sol := range llmAnalysis.Solutions {
			sb.WriteString("• " + sol + "\n")
		}
	} else {
		sb.WriteString(fmt.Sprintf("• **Pattern áp dụng:** Single-Responsibility Modular Strategy trên `%s`, giữ nguyên signature hàm ở tầng interface hiện hữu.\n", strings.Join(d.AffectedRepos, ", ")))
		sb.WriteString("• **Điểm chạm chính (Core Touchpoints):**\n")
		touchLimit := 3
		if len(subTasks) < touchLimit {
			touchLimit = len(subTasks)
		}
		for i := 0; i < touchLimit; i++ {
			st := subTasks[i]
			relTouch := normalizeOrchestratorDisplayPath(st.ContextFileLine, d.AffectedRepos...)
			sb.WriteString(fmt.Sprintf("  - `%s`: %s\n", relTouch, st.Title))
		}
		sb.WriteString("• **Cơ chế Rollback / Toggle:** Thực thi cô lập trên nhánh Worktree riêng và bảo toàn tương thích ngược 100% với luồng hiện hành.\n")
	}

	sb.WriteString("\n⚠️ **2. KIỂM SOÁT RỦI RO (RISK & GUARDRAILS)**\n")
	if llmAnalysis != nil && len(llmAnalysis.Risks) > 0 {
		for _, r := range llmAnalysis.Risks {
			sb.WriteString("• " + r + "\n")
		}
	} else {
		sb.WriteString("• **Memory & Concurrency:** Kiểm soát tài nguyên, context timeout và tránh phát sinh race condition/goroutine leak.\n")
		sb.WriteString("• **Tương thích Hợp đồng Dữ liệu:** Đảm bảo DTO/Schema đầu ra khớp 100% với caller hiện tại và đạt chuẩn G1–G8.\n")
	}

	sb.WriteString("\n---\n")
	sb.WriteString(fmt.Sprintf("💻 **3. KEY DIFF / CODE PREVIEW (`%s`)**\n", primaryBase))
	rawCodeSnippet := d.CodeDemoSnippet
	if llmAnalysis != nil && strings.TrimSpace(llmAnalysis.CodeDemo) != "" {
		rawCodeSnippet = llmAnalysis.CodeDemo
	}
	codePreview := buildCleanKeyDiffPreview(qt, primaryFile, stackProfile.CodeFence, rawCodeSnippet)
	sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n", stackProfile.CodeFence, codePreview))

	totalSubs := len(subTasks)
	sb.WriteString("\n---\n")
	sb.WriteString(fmt.Sprintf("📋 **4. LỘ TRÌNH THỰC THI (%d SUB-TASKS · %d FILES)**\n", totalSubs, len(dedupFiles)))
	if totalSubs <= 4 {
		for idx, st := range subTasks {
			waveTitle := fmt.Sprintf("Wave %d (Foundation & Model)", st.WaveIndex)
			if st.WaveIndex > 0 || idx > 0 {
				waveIndexDisplay := st.WaveIndex
				if waveIndexDisplay == 0 && idx > 0 {
					waveIndexDisplay = idx
				}
				waveTitle = fmt.Sprintf("Wave %d (Routing & Integration)", waveIndexDisplay)
			}
			icon := "⏳"
			if strings.EqualFold(st.Status, "done") || strings.EqualFold(st.Stage, "DONE") {
				icon = "✅"
			}
			subSummary := st.Title
			dodText := fmt.Sprintf("`%s`", strings.Trim(st.TestDoD, "`"))
			sb.WriteString(fmt.Sprintf("• **%s:**\n", waveTitle))
			sb.WriteString(fmt.Sprintf("  %d. %s `%s`: %s\n", idx+1, icon, st.SubTaskID, subSummary))
			sb.WriteString(fmt.Sprintf("  *(DoD: %s)*\n", dodText))
		}
	} else {
		for _, wLine := range compactSubTasksByWaveLines(subTasks) {
			sb.WriteString(wLine + "\n")
		}
		sb.WriteString(fmt.Sprintf("  *(DoD: %s)*\n", stackProfile.BuildAndDoD))
	}

	sb.WriteString(fmt.Sprintf("\n📁 **Files thay đổi (%d):**\n", len(dedupFiles)))
	if len(dedupFiles) > 0 {
		var quotedFiles []string
		for _, f := range dedupFiles {
			quotedFiles = append(quotedFiles, fmt.Sprintf("`%s`", f))
		}
		sb.WriteString(strings.Join(quotedFiles, ", ") + "\n")
	} else {
		sb.WriteString("`09_tasks_solution.md`, `08_tasks.md`\n")
	}
	sb.WriteString(fmt.Sprintf("📄 **Artifacts:** `09_tasks_solution.md` | `08_tasks.md` | `diff.patch` *(tại `.orchestrator/tasks/%s/`)*\n", safeTaskKey))

	firstSubID := "M01"
	secondSubID := "M02"
	if len(subTasks) >= 1 {
		parts := strings.Split(subTasks[0].SubTaskID, "-")
		firstSubID = parts[len(parts)-1]
	}
	if len(subTasks) >= 2 {
		parts := strings.Split(subTasks[1].SubTaskID, "-")
		secondSubID = parts[len(parts)-1]
	}

	sb.WriteString("\n---\n")
	sb.WriteString("⚡ **HÀNH ĐỘNG DUYỆT (CHỌN 1 TRONG 3):**\n")
	sb.WriteString(fmt.Sprintf("• **[⚡ Approve All]**: Chạy tự động `%s (Wave 0) ➔ %s (Wave 1) ➔ S5 Audit`.\n", firstSubID, secondSubID))
	sb.WriteString(fmt.Sprintf("• **[🧩 Step-by-Step]**: Duyệt và kiểm tra kết quả từng Sub-Task (`%s` xong mới mở khóa `%s`).\n", firstSubID, secondSubID))
	sb.WriteString(fmt.Sprintf("• **[✏️ Revise (#%d)]**: Bấm nút sửa hoặc reply `/revise #%d <nội dung>` (VD: *\"Tách riêng PgTransHisRepository thay vì nhét chung vào TransHisModel\"*).", d.TaskID, d.TaskID))

	refToken := fmt.Sprintf("#%d", d.TaskID)
	if d.TaskID <= 0 {
		refToken = d.TaskKey
	}
	rows := [][]map[string]string{
		{
			{
				"text":          fmt.Sprintf("⚡ Approve All (%s Wave 0 ➔ %s Wave 1 ➔ S5) (%s)", firstSubID, secondSubID, refToken),
				"callback_data": fmt.Sprintf("APPROVE_ALL %s", refToken),
			},
		},
		{
			{
				"text":          fmt.Sprintf("🧩 Step-by-Step (Duyệt Từng Sub-Task 1/%d ➔ %d/%d) (%s)", totalSubs, totalSubs, totalSubs, refToken),
				"callback_data": fmt.Sprintf("STEP_SUBTASKS %s", refToken),
			},
		},
		{
			{
				"text":          fmt.Sprintf("✏️ Revise (%s)", refToken),
				"callback_data": fmt.Sprintf("REVISE_PROMPT %s", refToken),
			},
			{
				"text":          "📋 Hàng Đợi (/task)",
				"callback_data": "/task",
			},
		},
	}
	return sb.String(), rows
}

func compactSubTasksByWaveLines(subTasks []SubTaskDTO) []string {
	if len(subTasks) == 0 {
		return []string{"• Chưa có Sub-Task nào được khởi tạo."}
	}
	type waveGroup struct {
		waveIndex int
		items     []string
	}
	var waves []waveGroup
	waveMap := make(map[int]int)
	for _, st := range subTasks {
		icon := "⏳"
		if strings.EqualFold(st.Status, "done") || strings.EqualFold(st.Stage, "DONE") {
			icon = "✅"
		}
		shortFile := filepath.Base(strings.TrimSpace(st.TargetFiles))
		if shortFile == "" || shortFile == "." {
			shortFile = filepath.Base(strings.Split(strings.TrimSpace(st.ContextFileLine), ":")[0])
		}
		item := fmt.Sprintf("%s `%s`", icon, st.SubTaskID)
		if shortFile != "" && shortFile != "." {
			item = fmt.Sprintf("%s `%s` (`%s`)", icon, st.SubTaskID, shortFile)
		}
		idx, ok := waveMap[st.WaveIndex]
		if !ok {
			idx = len(waves)
			waveMap[st.WaveIndex] = idx
			waves = append(waves, waveGroup{waveIndex: st.WaveIndex})
		}
		waves[idx].items = append(waves[idx].items, item)
	}
	var lines []string
	for _, wg := range waves {
		lines = append(lines, fmt.Sprintf("• **Wave %d:** %s", wg.waveIndex, strings.Join(wg.items, ", ")))
	}
	return lines
}

func escapeTelegramHTML(s string) string {
	return notify.EscapeTelegramHTML(s)
}

func convertTelegramMarkdownToHTML(md string) string {
	return notify.ConvertTelegramMarkdownToHTML(md)
}

func extractExistingTelegramRevisionSection(planStr string) string {
	revHeader := "## ✏️ Phản hồi & Điều chỉnh Giải pháp từ Telegram (User Feedback Revisions)"
	idx := strings.Index(planStr, revHeader)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(planStr[idx:])
}
