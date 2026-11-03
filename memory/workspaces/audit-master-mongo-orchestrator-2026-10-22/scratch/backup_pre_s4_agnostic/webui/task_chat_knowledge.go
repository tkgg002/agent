package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

// renderContextualDiscussReply answers the user's question directly and naturally in Vietnamese ("anh - em")
// grounded in physical Root Causes (13_analysis.md), Solutions (09_tasks_solution.md), Audit Reports, and Sub-Tasks.
func (s *Server) renderContextualDiscussReply(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	cleanMsg string,
	agentRoleLabel, agentSlot, skillBadge, selectedModel string,
	lessonRefs []string,
) string {
	lowerMsg := strings.ToLower(cleanMsg)
	doneCnt := 0
	var pendingList []string
	for _, st := range subTasks {
		if strings.EqualFold(st.Status, "done") {
			doneCnt++
		} else if len(pendingList) < 6 {
			pendingList = append(pendingList, fmt.Sprintf("`%s` (%s)", st.SubTaskID, st.TargetRepoName))
		}
	}
	affectedRepos := s.ResolveTaskAffectedRepos(qt)
	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}

	// Case 1: User asks who the agent is / which model & role is running ("mày là ai, model nào")
	if strings.Contains(lowerMsg, "mày là ai") || strings.Contains(lowerMsg, "model nào") || strings.Contains(lowerMsg, "role nào") {
		return fmt.Sprintf(
			"Chào anh! Em đang trực ở khung **Co-Architect Studio** của task **`%s` (%s)** với định danh:\n\n"+
				"- **Vai trò (Role):** `%s` (`%s` · Skill: `%s`)\n"+
				"- **Model đang xử lý:** `%s`\n"+
				"- **Bộ não quản trị (Governance Kernel):** Gắn trực tiếp với `/Users/trainguyen/Documents/work/agent/GEMINI.md` (Master Workflow `S0 -> S6`, 7 Lifecycle Hooks `H1–H7`, và kho bài học trong `lessons/INDEX.md`).\n"+
				"- **Phân công đa Agent trên `%s`:**\n"+
				"  - **Slot A (`BRAIN · Architect`):** Thiết kế `09_tasks_solution.md`, bóc tách `08_tasks.md` và đối thoại trực tiếp với anh tại đây.\n"+
				"  - **Slot B (`MUSCLE · Chief Engineer`):** Thực thi code cho từng Sub-Task sau khi có lệnh `APPROVE`.\n"+
				"  - **Slot H (`QA & DG3 Auditor`):** Kiểm định 8 cổng chất lượng `G1–G8`, quét `H4b Zero-Mockup` và xuất `audit_report.md`.",
			qt.ExternalID, qt.Title, agentRoleLabel, agentSlot, skillBadge, selectedModel, qt.ExternalID,
		)
	}

	// Case 2: User asks how working in ai-orchestrator Web Studio compares to native Antigravity IDE
	if strings.Contains(lowerMsg, "cảm nhận") || strings.Contains(lowerMsg, "antigravity") || strings.Contains(lowerMsg, "so với") {
		return fmt.Sprintf(
			"Dạ anh, dưới góc nhìn của **`%s` (`%s`)** khi vận hành song song cả hai môi trường, điểm khác biệt cốt lõi giữa **AI-Orchestrator Web Studio (`:8090`)** và **App Antigravity IDE** nằm ở **tầm nhìn điều phối đa luồng vs. độ sâu công cụ trực tiếp**:\n\n"+
				"1. **Điểm mạnh của AI-Orchestrator Web Studio (`:8090`) nơi anh em mình đang đứng:**\n"+
				"   - **Bản đồ Trực quan Đa Repo & Đa Sub-Task:** Anh nhìn thấy ngay tiến độ từng Sub-Task (`%d/%d` Sub-Tasks của `%s`), phân bổ rõ Role/Slot/Model nào chịu trách nhiệm cho repo nào (`%s`), kèm hồ sơ cô lập riêng tại `%s`.\n"+
				"   - **Quản trị Hàng đợi & Tự động Bóc tách Audit:** Khi anh dán một báo cáo Audit độc lập (`AF-ORCH-*`, `AF-DH-*`, `TASK-A*`) vào khung chat này, hệ thống tự động nhận diện, trích xuất các Sub-Tasks khắc phục vào `08_tasks.md` và mở lại vòng lặp `S5 -> S3 Re-Plan -> S4 Execute`.\n"+
				"2. **Điểm mạnh của App Antigravity IDE (với `/agent` Core & Hooks `H1–H7`):**\n"+
				"   - **Cưỡng chế Hook Vật lý Tức thời (`agy_guard.py`):** Mọi lệnh `view_file`, `replace_file_content`, `run_command` đều đi qua 7 cổng `H1–H7` và bắt buộc có biên lai runtime thật `EV-...` từ `evidence_run.py` + kiểm định hội đồng `/laya_router` (`DG1/DG2/DG3`).\n"+
				"3. **Sự kết hợp tốt nhất:**\n"+
				"   - Dùng **Web Studio `:8090`** làm bảng chỉ huy trung tâm (Command Center) để anh theo dõi tiến độ, review `09_tasks_solution.md` / `diff.patch` / `audit_report.md` và định hướng kiến trúc với em; đồng thời giữ nguyên kỷ luật `/agent` (`GEMINI.md`, `H1–H7`, Strict Repo Boundary `L002`) ở tầng thực thi!",
			agentRoleLabel, selectedModel, doneCnt, len(subTasks), qt.ExternalID, strings.Join(repoNames, ", "), isoMeta.ArtifactDir,
		)
	}

	var rootCauses, solutions []string
	var codeSnippet, directAnswer string
	if llmRes, err := s.invokeArchitectLLM(context.Background(), qt, subTasks, cleanMsg, selectedModel, "discuss"); err == nil && llmRes != nil {
		rootCauses = llmRes.RootCauses
		solutions = llmRes.Solutions
		codeSnippet = strings.TrimSpace(llmRes.CodeDemo)
		directAnswer = strings.TrimSpace(llmRes.DirectAnswer)
		if llmRes.ShouldUpdatePlan || isPlanCorrectionOrSteeringMessage(cleanMsg, qt, subTasks) {
			cliCfg := s.loadCLISettings()
			qtCopy := qt
			subTasks, _ = s.syncArchitectLLMAnalysisToMasterPlan(
				context.Background(), &qtCopy, isoMeta, subTasks, llmRes, affectedRepos,
				cliCfg.DefaultRunner, selectedModel, cleanMsg, agentRoleLabel, agentSlot, skillBadge,
			)
		}
	}
	if len(rootCauses) == 0 || len(solutions) == 0 || codeSnippet == "" {
		fbRC, fbSol, fbCode, _ := s.extractTaskDeepKnowledge(qt, isoMeta, subTasks)
		if len(rootCauses) == 0 {
			rootCauses = fbRC
		}
		if len(solutions) == 0 {
			solutions = fbSol
		}
		if codeSnippet == "" {
			codeSnippet = fbCode
		}
	}
	return BuildUnifiedCoArchitectReplyWithSummary(
		qt, isoMeta, subTasks, cleanMsg,
		agentRoleLabel, agentSlot, selectedModel,
		directAnswer, rootCauses, solutions, codeSnippet, repoNames, lessonRefs,
	)
}

func isMarkdownMetadataOrLessonNoiseLine(clean string) bool {
	lower := strings.ToLower(strings.TrimSpace(clean))
	for _, prefix := range []string{
		"task key:", "**task key", "affected repositories", "**affected repositories",
		"isolated worktree", "**isolated worktree", "cli runner", "**cli runner",
		"active `/agent`", "decision gate", "trạng thái cổng", "workspace**:",
		"**workspace**:", "suffix**:", "**suffix**:", "tier**:", "**tier**:",
		"trạng thái**:", "**trạng thái**:", "giai đoạn**:", "**giai đoạn**:",
		"ngày tạo**:", "**ngày tạo**:", "mục tiêu**:", "**mục tiêu**:",
		"cms connection api", "l010 [", "l079 [", "**l010", "**l079",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return strings.Contains(lower, "feat-master-mongo-destination-2026-10-20") ||
		strings.Contains(lower, "suffix**: master_mongo") ||
		strings.Contains(lower, "đúng:**")
}

func (s *Server) extractTaskDeepKnowledge(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
) (rootCauses []string, solutions []string, codeSnippet string, fileTouches []string) {
	seenRC := make(map[string]bool)
	seenSol := make(map[string]bool)
	addRC := func(line string) {
		clean := strings.TrimSpace(strings.TrimLeft(line, "-*•0123456789. "))
		if len(clean) < 18 || seenRC[clean] || len(rootCauses) >= 8 || isInternalOrchestratorBoilerplateLine(clean) || isMarkdownMetadataOrLessonNoiseLine(clean) {
			return
		}
		seenRC[clean] = true
		rootCauses = append(rootCauses, clean)
	}
	addSol := func(line string) {
		clean := strings.TrimSpace(strings.TrimLeft(line, "-*•0123456789. "))
		if len(clean) < 18 || seenSol[clean] || len(solutions) >= 8 || isInternalOrchestratorBoilerplateLine(clean) || isMarkdownMetadataOrLessonNoiseLine(clean) {
			return
		}
		seenSol[clean] = true
		solutions = append(solutions, clean)
	}

	var candidateFiles []string
	if isoMeta.ArtifactDir != "" {
		candidateFiles = append(candidateFiles,
			filepath.Join(isoMeta.ArtifactDir, "13_analysis.md"),
			isoMeta.PlanFile,
			isoMeta.AuditFile,
		)
	}

	for _, fPath := range candidateFiles {
		if strings.TrimSpace(fPath) == "" {
			continue
		}
		raw, err := os.ReadFile(fPath)
		if err != nil || len(raw) == 0 {
			continue
		}
		lines := strings.Split(string(raw), "\n")
		section := ""
		inCodeBlock := false
		var curCode strings.Builder
		for _, ln := range lines {
			trim := strings.TrimSpace(ln)
			if strings.HasPrefix(trim, "```") {
				if !inCodeBlock {
					inCodeBlock = true
					curCode.Reset()
				} else {
					inCodeBlock = false
					candCode := curCode.String()
					if codeSnippet == "" && len(candCode) > 40 && !isCompilerBoilerplateSymbol(candCode) && !strings.Contains(candCode, "dist/models/") {
						codeSnippet = candCode
					}
				}
				continue
			}
			if inCodeBlock {
				curCode.WriteString(ln + "\n")
				continue
			}
			lowerLn := strings.ToLower(trim)
			if strings.HasPrefix(trim, "#") {
				if isInternalOrchestratorBoilerplateLine(trim) {
					section = ""
					continue
				}
				if strings.Contains(lowerLn, "sơ hở") || strings.Contains(lowerLn, "lỗi") || strings.Contains(lowerLn, "root cause") || strings.Contains(lowerLn, "nguyên nhân") || strings.Contains(lowerLn, "vì sao") || strings.Contains(lowerLn, "chẩn đoán") || strings.Contains(lowerLn, "điểm nghẽn") {
					section = "rc"
					if strings.Contains(lowerLn, "sơ hở f") || strings.Contains(lowerLn, "lỗi ") {
						addRC(strings.TrimLeft(trim, "# "))
					}
				} else if strings.Contains(lowerLn, "giải pháp") || strings.Contains(lowerLn, "phương án") || strings.Contains(lowerLn, "solution") || strings.Contains(lowerLn, "khắc phục") || strings.Contains(lowerLn, "clean-room") {
					section = "sol"
					if strings.Contains(lowerLn, "giải pháp cho") {
						addSol(strings.TrimLeft(trim, "# "))
					}
				} else {
					section = ""
				}
				continue
			}
			if strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* ") || (len(trim) > 3 && trim[0] >= '1' && trim[0] <= '9' && trim[1] == '.') {
				if strings.Contains(lowerLn, "nguyên nhân") || strings.Contains(lowerLn, "root cause") || strings.Contains(lowerLn, "sơ hở") || strings.Contains(lowerLn, "vấn đề hiện tại") || strings.Contains(lowerLn, "bỏ sót") || strings.Contains(lowerLn, "mất update") || strings.Contains(lowerLn, "mismatch") {
					addRC(trim)
				} else if strings.Contains(lowerLn, "giải pháp") || strings.Contains(lowerLn, "khắc phục") || strings.Contains(lowerLn, "bổ sung") || strings.Contains(lowerLn, "nâng cấp") || strings.Contains(lowerLn, "cập nhật") {
					addSol(trim)
				} else if section == "rc" {
					addRC(trim)
				} else if section == "sol" {
					addSol(trim)
				}
			}
		}
	}

	for _, st := range subTasks {
		statusBadge := "⏳ PENDING"
		if strings.EqualFold(st.Status, "done") || strings.EqualFold(st.Stage, "DONE") {
			statusBadge = "✅ DONE"
		}
		fileTouches = append(fileTouches, fmt.Sprintf("**`%s`** (`%s` · `%s` · `%s`): %s — `%s`",
			st.SubTaskID, st.WorkflowStep, st.TargetRepoName, statusBadge, st.Title, st.ContextFileLine))
		if len(solutions) < 6 && strings.TrimSpace(st.Description) != "" {
			addSol(fmt.Sprintf("**`%s` (`%s`):** %s (`%s`)", st.SubTaskID, st.TargetRepoName, st.Description, st.ContextFileLine))
		}
	}
	if len(rootCauses) == 0 {
		for _, st := range subTasks {
			addRC(fmt.Sprintf("Điểm chạm cần xử lý tại **`%s`** (`%s`): %s", st.ContextFileLine, st.TargetRepoName, st.Title))
		}
	}
	return rootCauses, solutions, codeSnippet, fileTouches
}

func (s *Server) reconcileChatHistoryAndAuditRemediation(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO) {
	chatFile := s.taskChatHistoryPath(qt.ExternalID)
	raw, err := os.ReadFile(chatFile)
	if err != nil || len(raw) == 0 {
		return
	}
	var history []CoArchitectChatMessageDTO
	if err := json.Unmarshal(raw, &history); err != nil || len(history) == 0 {
		return
	}

	subTasks := s.loadOrGenerateSubTasks(qt)
	affectedRepos := s.ResolveTaskAffectedRepos(qt)
	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}
	isMongoTask := isMasterMongoDestinationTask(qt)
	modified := false

	if len(history) > 0 && history[0].Sender == "agent" && !isMongoTask &&
		(strings.Contains(history[0].Message, "TASK-M01") || strings.Contains(history[0].Message, "feat-master-mongo-destination-2026-10-20") || (!isExplicitOrchestratorTarget(qt) && strings.Contains(history[0].Message, "ai-orchestrator"))) {
		history[0].Message = BuildUnifiedOnboardingChatUDFv1(qt, subTasks, isoMeta, history[0].Model, repoNames)
		modified = true
	}

	for i := 0; i < len(history); i++ {
		msg := history[i]
		if msg.Sender != "user" {
			continue
		}
		trueMode := classifyChatActionMode(msg.Message, msg.ActionMode)
		agentIdx := -1
		if i+1 < len(history) && history[i+1].Sender == "agent" {
			agentIdx = i + 1
		}
		if agentIdx < 0 {
			continue
		}
		agentReply := history[agentIdx]
		isContaminatedNonMongo := !isMongoTask && (strings.Contains(agentReply.Message, "feat-master-mongo-destination-2026-10-20") ||
			strings.Contains(agentReply.Message, "TASK-M01") ||
			strings.Contains(agentReply.Message, "Master Connection hỗ trợ MongoDB"))
		isOldStaticTemplate := isContaminatedNonMongo ||
			strings.Contains(agentReply.Message, "- **Phương án khắc phục:** ") ||
			strings.Contains(agentReply.Message, "Phương án khắc phục: Workspace**:") ||
			(strings.Contains(agentReply.Message, "Phản hồi cho yêu cầu của bạn trên") &&
				strings.Contains(agentReply.Message, "Các bài học `/agent` khớp trực tiếp với nội dung bạn vừa nêu:")) ||
			strings.Contains(agentReply.Message, "Nếu anh muốn áp dụng ngay chỉ đạo này vào `09_tasks_solution.md`") ||
			strings.Contains(agentReply.Message, "Ý kiến / Câu hỏi của anh:") ||
			strings.Contains(agentReply.Message, "Kịch Bản Tái Hiện Lỗi") ||
			strings.Contains(agentReply.Message, "Triển khai kiến trúc & nghiệp vụ cốt lõi cho") ||
			strings.Contains(agentReply.Message, "dist/models/") ||
			strings.Contains(agentReply.Message, "function adopt(value)")
		hasDuplicatePrefix := strings.Contains(agentReply.Message, "):** 1. **`TASK-A") ||
			!strings.Contains(agentReply.Message, "Đối soát Vật lý Trung thực trên Đĩa")

		hasStaleWorkerPath := false
		for _, st := range subTasks {
			if strings.Contains(st.TargetFiles, "internal/worker/") || strings.Contains(st.ContextFileLine, "internal/worker/") ||
				strings.Contains(st.TargetFiles, "/dist/") || strings.Contains(st.ContextFileLine, "/dist/") {
				hasStaleWorkerPath = true
				break
			}
		}

		if trueMode == "ingest_audit_remediation" && (!agentReply.AppliedToPlan || isContaminatedNonMongo || hasDuplicatePrefix || hasStaleWorkerPath) {
			history[i].ActionMode = "ingest_audit_remediation"
			qtCopy := qt
			newReply, targetFile, updatedSubTasks := s.ingestAuditReportAndRemediation(
				ctx, &qtCopy, isoMeta, subTasks, msg.Message,
				agentReply.AgentRole, agentReply.AgentSlot, agentReply.Skill, agentReply.Model, agentReply.ReferencedIDs,
			)
			subTasks = updatedSubTasks
			history[agentIdx].ActionMode = "ingest_audit_remediation"
			history[agentIdx].Message = newReply
			history[agentIdx].AppliedToPlan = true
			history[agentIdx].TargetFile = targetFile
			modified = true
		} else if trueMode == "revise_solution" && (!agentReply.AppliedToPlan || isContaminatedNonMongo || hasStaleWorkerPath) {
			history[i].ActionMode = "revise_solution"
			qtCopy := qt
			cliCfg := s.loadCLISettings()
			newReply, targetFile, updatedSubTasks := s.executeArchitectReviseSolution(
				ctx, &qtCopy, isoMeta, subTasks, msg.Message,
				agentReply.AgentRole, agentReply.AgentSlot, agentReply.Skill, agentReply.Model, agentReply.ReferencedIDs, cliCfg,
			)
			subTasks = updatedSubTasks
			history[agentIdx].ActionMode = "revise_solution"
			history[agentIdx].Message = newReply
			history[agentIdx].AppliedToPlan = true
			history[agentIdx].TargetFile = targetFile
			modified = true
		} else if trueMode == "revise_solution" && strings.Contains(agentReply.Message, "- **Phương án khắc phục:** ") {
			history[agentIdx].Message = strings.ReplaceAll(agentReply.Message, "- **Phương án khắc phục:** ", "- ")
			modified = true
		} else if trueMode == "discuss" && isOldStaticTemplate {
			targetRepo := s.ResolveTaskTargetRepo(qt)
			hits, _ := s.inspectTargetRepoSymbols(targetRepo, qt.ExternalID, qt.Title, msg.Message+" "+qt.Description)
			targetTech := extractPromptTargetTechEntity(qt.Title, qt.Description, msg.Message)
			synth := synthesizeSymbolGroundedArchitectAnalysis(qt, subTasks, hits, msg.Message, targetRepo, targetTech)
			newReply := BuildUnifiedCoArchitectReplyWithSummary(
				qt, isoMeta, subTasks, msg.Message,
				agentReply.AgentRole, agentReply.AgentSlot, agentReply.Model,
				synth.DirectAnswer, synth.RootCauses, synth.Solutions, synth.CodeDemo, repoNames, agentReply.ReferencedIDs,
			)
			history[agentIdx].Message = newReply
			modified = true
		}
	}

	if modified {
		_ = s.saveTaskChatHistory(qt.ExternalID, history)
	}
}
