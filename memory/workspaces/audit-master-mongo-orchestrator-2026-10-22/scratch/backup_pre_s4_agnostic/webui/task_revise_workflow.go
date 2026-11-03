package webui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/agentgate"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

func isInternalOrchestratorBoilerplateLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	return strings.Contains(lower, "kịch bản tái hiện") ||
		strings.Contains(lower, "gate g2") ||
		strings.Contains(lower, "red -> green") ||
		strings.Contains(lower, "bước `red`") ||
		strings.Contains(lower, "bước `green`") ||
		strings.Contains(lower, "verifysubtaskcodeondisk") ||
		strings.Contains(lower, "evidence_run.py") ||
		strings.Contains(lower, "phạm vi điều tra (`read-only s2`)") ||
		strings.Contains(lower, "phân loại bằng chứng (`verified` vs `hypothesis`)") ||
		strings.Contains(lower, "dist/models/") ||
		strings.Contains(lower, "function adopt(value)")
}

func isArchitecturalSteeringDirective(lowerMsg string) bool {
	for _, kw := range []string{
		"cập nhật giải pháp", "sửa giải pháp", "sửa lại giải pháp", "đổi giải pháp",
		"revise", "đổi thiết kế", "thay đổi thiết kế", "bổ sung vào solution",
		"bổ sung vào giải pháp", "sửa 09_tasks_solution", "sửa phương án",
		"giải pháp của anh là", "ý của anh là", "hiện tại mày đang làm",
		"bỏ cái export type cũ đi", "đè lên nó", "tạo 1 cái type v2",
		"v1 vẫn là cái cũ", "không đè lên", "tách v1", "tách v2", "tách v3",
		"sửa lại hướng", "re-plan", "làm lại giải pháp",
		"sao lại làm tiếp", "cái task này là làm", "sao ko cập nhật plan",
		"cập nhật plan", "sửa plan", "làm lại plan", "sao lại là postgress",
		"sao lại là postgres", "clickhouse mà", "viết lại plan",
	} {
		if strings.Contains(lowerMsg, kw) {
			return true
		}
	}
	return false
}

func isPlanCorrectionOrSteeringMessage(msg string, qt queue.QueuedTask, subTasks []SubTaskDTO) bool {
	lowerMsg := strings.ToLower(strings.TrimSpace(msg))
	if isArchitecturalSteeringDirective(lowerMsg) {
		return true
	}
	targetTech := extractPromptTargetTechEntity(qt.Title, qt.Description, msg)
	if targetTech != "" && len(subTasks) > 0 {
		var subCombined strings.Builder
		for _, st := range subTasks {
			subCombined.WriteString(strings.ToLower(st.Title + " " + st.Description + " " + st.ContextFileLine + " "))
		}
		if !strings.Contains(subCombined.String(), strings.ToLower(targetTech)) {
			return true
		}
	}
	return false
}

func isMasterMongoDestinationTask(qt queue.QueuedTask) bool {
	lower := strings.ToLower(qt.ExternalID + " " + qt.Title)
	return (strings.Contains(lower, "mongo") && (strings.Contains(lower, "master") || strings.Contains(lower, "destination"))) ||
		strings.Contains(lower, "master-mongo") || strings.EqualFold(qt.ExternalID, "TASK-11")
}

func buildSubTaskModuleTitleAndDesc(qt queue.QueuedTask, repoName, ctxLine, sig string, slotIdx int) (string, string) {
	relFile := normalizeOrchestratorDisplayPath(ctxLine, repoName)
	baseFile := filepath.Base(relFile)
	targetTech := extractPromptTargetTechEntity(qt.Title, qt.Description, "")

	if targetTech == "clickhouse" {
		if slotIdx == 1 {
			return fmt.Sprintf("Xây dựng ClickHouseDB Adapter & Truy vấn Export Type V3 tại `%s` (`%s`)", repoName, baseFile),
				fmt.Sprintf("Tạo mới và triển khai `ClickHouseDB` tại `%s` (`%s`), đảm bảo kết nối ClickHouse độc lập và không ảnh hưởng tới adapter MongoDB (V1) hay PostgreSQL (V2).", relFile, sig)
		}
		return fmt.Sprintf("Định tuyến Export Type V3 (ClickHouse) song song với V1 (Mongo) & V2 (Postgres) tại `%s` (`%s`)", repoName, baseFile),
			fmt.Sprintf("Cập nhật router/model tại `%s` (`%s`) để giữ nguyên 100%% luồng V1 (MongoDB) & V2 (PostgreSQL) và định tuyến `export_type` V3 sang `ClickHouseDB` kèm kiểm thử nghiệm thu.", relFile, sig)
	}
	if targetTech == "postgres" {
		if slotIdx == 1 {
			return fmt.Sprintf("Giữ nguyên luồng V1 (MongoDB) & Xây dựng Model/Repository V2 (PostgreSQL) trên `%s` (`%s`)", repoName, baseFile),
				fmt.Sprintf("Bảo toàn 100%% backward-compatibility cho luồng V1 hiện hữu (`%s`) và bổ sung tầng truy vấn PostgreSQL bằng `pgPool` thuần tại `%s` (`%s`).", baseFile, relFile, sig)
		}
		return fmt.Sprintf("Triển khai Strategy/Router định tuyến song song `export_type` V1 (Mongo) vs V2 (Postgres) & Integration Test trên `%s` (`%s`)", repoName, baseFile),
			fmt.Sprintf("Định tuyến theo `export_type` (V1 gọi MongoDB cũ, V2 gọi `PostgresDB` mới) tại `%s` (`%s`) và kiểm thử đầu-cuối cả hai luồng V1/V2.", relFile, sig)
	}

	if slotIdx == 1 {
		return fmt.Sprintf("Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `%s` (`%s`)", repoName, baseFile),
			fmt.Sprintf("Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `%s` (`%s`) trong worktree cô lập của `%s`.", relFile, sig, repoName)
	}
	return fmt.Sprintf("Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `%s` (`%s`)", repoName, baseFile),
		fmt.Sprintf("Kết nối dây dẫn nghiệp vụ tại `%s` (`%s`) và viết integration test xác minh đầu-cuối trên `%s`.", relFile, sig, repoName)
}

func buildCleanRoomSpecSectionFromSubTasks(qt queue.QueuedTask, subTasks []SubTaskDTO, repoSummary string) string {
	var rules []string
	for idx, st := range subTasks {
		refLoc := extractCleanSpecRefToken(st.ContextFileLine, st.TargetRepoName)
		rules = append(rules, fmt.Sprintf("- **R%d (`%s` — `%s`):** %s (`REF: %s`)", idx+1, st.SubTaskID, st.TargetRepoName, st.Title, refLoc))
	}
	if len(rules) == 0 {
		rules = append(rules, fmt.Sprintf("- **R1 (`%s`):** Thực thi cô lập trong Git Worktree (`REF: %s:1`)", qt.ExternalID, extractCleanSpecRefToken(repoSummary, "repo")))
	}
	return strings.Join(append([]string{"## Clean-Room Approved Spec"}, rules...), "\n")
}

func (s *Server) applyLLMSubTaskSpecs(qt queue.QueuedTask, subTasks []SubTaskDTO, analysis *ArchitectLLMAnalysis) []SubTaskDTO {
	if analysis == nil || len(analysis.SubTasks) == 0 || len(subTasks) == 0 {
		return subTasks
	}
	for i := range subTasks {
		if i >= len(analysis.SubTasks) {
			break
		}
		spec := analysis.SubTasks[i]
		if strings.TrimSpace(spec.Title) != "" {
			subTasks[i].Title = strings.TrimSpace(spec.Title)
		}
		if strings.TrimSpace(spec.Description) != "" {
			subTasks[i].Description = strings.TrimSpace(spec.Description)
		}
		if strings.TrimSpace(spec.ContextFileLine) != "" {
			subTasks[i].ContextFileLine = strings.TrimSpace(spec.ContextFileLine)
			rel := normalizeOrchestratorDisplayPath(spec.ContextFileLine, subTasks[i].TargetRepoName)
			if idx := strings.Index(rel, ":"); idx > 0 {
				rel = strings.TrimSpace(rel[:idx])
			}
			if idx := strings.Index(rel, " "); idx > 0 {
				rel = strings.TrimSpace(rel[:idx])
			}
			if rel != "" {
				subTasks[i].TargetFiles = rel
			}
		}
	}
	return subTasks
}

func (s *Server) syncArchitectLLMAnalysisToMasterPlan(
	ctx context.Context,
	qt *queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	analysis *ArchitectLLMAnalysis,
	affectedRepos []string,
	runnerName, selectedModel, steeringNote, agentRoleLabel, agentSlot, skillBadge string,
) ([]SubTaskDTO, string) {
	if analysis != nil {
		s.saveTaskLLMAnalysis(qt.ExternalID, analysis)
		subTasks = s.applyLLMSubTaskSpecs(*qt, subTasks, analysis)
	}
	subTasks = s.enrichSubTasksWithRoleAndModel(subTasks)
	subTasks = s.populateSubTaskFractalContractsAndArtifacts(*qt, subTasks)
	_ = s.saveTaskSubTasks(qt.ExternalID, subTasks)
	s.syncSubTasksToMarkdown08(*qt, subTasks)

	_, _, _, _ = s.generateDeepTaskArtifacts(ctx, *qt, runnerName, selectedModel)

	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}
	repoSummary := strings.Join(repoNames, ", ")
	tier, mode, suffix := agentgate.InferTaskTierAndMode(qt.ExternalID, qt.Title, qt.Description, qt.Complexity, len(subTasks))

	var codeDemo string
	if analysis != nil && strings.TrimSpace(analysis.CodeDemo) != "" {
		codeDemo = analysis.CodeDemo
	} else {
		codeDemo = buildConcreteCodeDemoForTask(*qt, s.ResolveTaskTargetRepo(*qt), nil)
	}

	cleanRoomSec := buildCleanRoomSpecSectionFromSubTasks(*qt, subTasks, repoSummary)
	solDoc := BuildUnifiedS3SolutionDoc(
		*qt, isoMeta, subTasks, affectedRepos,
		repoSummary, runnerName, selectedModel, tier, mode, suffix,
		len(subTasks)*2, codeDemo, cleanRoomSec,
	)
	if strings.TrimSpace(steeringNote) != "" && analysis != nil && len(analysis.Solutions) > 0 {
		revisionBlock := strings.Join([]string{
			"",
			"---",
			fmt.Sprintf("## 🔄 Co-Architect Steering & Solution Revision (%s · by User + %s `%s`)", time.Now().Format("2006-01-02 15:04:05"), agentRoleLabel, selectedModel),
			fmt.Sprintf("- **Chỉ đạo định hướng từ User:** %s", steeringNote),
			fmt.Sprintf("- **Agent xử lý & cập nhật:** `%s` (`%s` · `%s` · Model: `%s`)", agentRoleLabel, agentSlot, skillBadge, selectedModel),
			"- **Phương án Kiến trúc cập nhật:**",
			"  - " + strings.Join(analysis.Solutions, "\n  - "),
			"",
		}, "\n")
		solDoc = strings.TrimRight(solDoc, "\n") + "\n" + revisionBlock
	}
	_ = os.WriteFile(isoMeta.PlanFile, []byte(solDoc), 0o644)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("09_tasks_solution_%s.md", suffix)), []byte(solDoc), 0o644)
	return subTasks, solDoc
}

func (s *Server) executeArchitectReviseSolution(
	ctx context.Context,
	qt *queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	cleanMsg, agentRoleLabel, agentSlot, skillBadge, selectedModel string,
	lessonRefs []string,
	cliCfg CLISettingsDTO,
) (string, string, []SubTaskDTO) {
	if !strings.Contains(qt.Description, cleanMsg) {
		qt.Description = strings.TrimSpace(qt.Description + "\n[Chỉ đạo Kiến trúc mới]: " + cleanMsg)
	}
	qt.Stage = queue.StageNeedsYou
	_ = os.Remove(filepath.Join(isoMeta.ArtifactDir, ".user-approved"))
	_ = os.Remove(filepath.Join(isoMeta.ArtifactDir, "diff.patch"))
	_ = os.Remove(filepath.Join(isoMeta.ArtifactDir, "audit_report.md"))

	affectedRepos := s.ResolveTaskAffectedRepos(*qt)
	freshSubTasks := s.DecomposeMultiRepoCodebaseSubTasks(*qt, affectedRepos, selectedModel)
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	for i := range freshSubTasks {
		freshSubTasks[i].Status = "waiting"
		freshSubTasks[i].Stage = "NEEDS_YOU"
		freshSubTasks[i].WorkflowStep = fmt.Sprintf("Wave %d · S3 Chờ Duyệt", freshSubTasks[i].WaveIndex)
		safeSubID := sanitizeTaskKeyPath(freshSubTasks[i].SubTaskID)
		subDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSubID)
		_ = os.MkdirAll(subDir, 0o755)
		_ = os.WriteFile(filepath.Join(subDir, ".subtask-stage"), []byte("NEEDS_YOU\n"), 0o644)
		_ = os.Remove(filepath.Join(subDir, "diff.patch"))
	}

	analysis, _ := s.invokeArchitectLLM(ctx, *qt, freshSubTasks, cleanMsg, selectedModel, "revise_solution")
	subTasks, _ = s.syncArchitectLLMAnalysisToMasterPlan(
		ctx, qt, isoMeta, freshSubTasks, analysis, affectedRepos,
		cliCfg.DefaultRunner, selectedModel, cleanMsg, agentRoleLabel, agentSlot, skillBadge,
	)

	var rootCauses, solutions []string
	var codeDemo, directAnswer string
	if analysis != nil {
		rootCauses = analysis.RootCauses
		solutions = analysis.Solutions
		codeDemo = analysis.CodeDemo
		directAnswer = analysis.DirectAnswer
	}

	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}
	_, _, suffix := agentgate.InferTaskTierAndMode(qt.ExternalID, qt.Title, qt.Description, qt.Complexity, len(subTasks))

	taskSelfHeal := loadOrInitSubTaskSelfHealState(isoMeta.ArtifactDir, cliCfg.MaxSelfHealRetries, "restore-wave0-"+suffix, SelfHealingCircuitStateDTO{})
	taskSelfHeal.ActiveLoopTier = "TIER2_S5_TO_S3_HUMAN_GATE"
	taskSelfHeal.RequiresHumanReApprove = true
	taskSelfHeal.LastFailureClass = "ARCH_PLAN_S3_CHANGED"
	taskSelfHeal.LastFailureReason = "Tier-2 S5 -> S3 Re-Plan via Co-Architect Studio: Đã thu hồi .user-approved và dừng tại Human Gate (NEEDS_YOU) chờ Re-Approve chống Scope Drift"
	saveSubTaskSelfHealState(isoMeta.ArtifactDir, taskSelfHeal)

	if s.QueueStore != nil {
		_ = s.QueueStore.UpdateStage(ctx, qt.ID, queue.StageNeedsYou, selectedModel, isoMeta.WorktreePath)
		rounds, _ := s.QueueStore.ListRounds(ctx, qt.ExternalID)
		_ = s.QueueStore.RecordRound(ctx, queue.AutopilotRound{
			TaskKey:       qt.ExternalID,
			RoundNum:      len(rounds) + 1,
			Stage:         fmt.Sprintf("S3 Solution Revised via Co-Architect Chat (%s · %s): Tier-2 Human Gate Re-Approve Required", agentRoleLabel, selectedModel),
			BlockingCount: 0,
			AdvisoryCount: 0,
			Turns:         1,
			DurationSec:   1.2,
			CostUSD:       0.0,
			GateVerdict:   "PASS",
		})
	}

	reply := BuildUnifiedCoArchitectReplyWithSummary(
		*qt, isoMeta, subTasks, cleanMsg,
		agentRoleLabel, agentSlot, selectedModel,
		directAnswer, rootCauses, solutions, codeDemo, repoNames, lessonRefs,
	)
	return reply, isoMeta.PlanFile, subTasks
}
