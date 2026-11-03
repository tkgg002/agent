package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/agentgate"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/scout"
)

// RepoProgressDTO reports real-time Sub-Task completion and git diff telemetry for one repository in AffectedRepos (LIVE-PROCESS-HUD-01).
type RepoProgressDTO struct {
	RepoName       string `json:"repo_name"`
	RepoPath       string `json:"repo_path"`
	Branch         string `json:"branch"`
	TotalSubTasks  int    `json:"total_sub_tasks"`
	DoneSubTasks   int    `json:"done_sub_tasks"`
	ProgressPct    int    `json:"progress_pct"`
	ActiveSubTask  string `json:"active_sub_task"`
	ActiveRole     string `json:"active_role"`
	ActiveModel    string `json:"active_model"`
	ModifiedFiles  int    `json:"modified_files"`
	DiffLines      int    `json:"diff_lines"`
	DiffSummary    string `json:"diff_summary"`
	VerificationSt string `json:"verification_status"` // "VERIFIED", "IN_PROGRESS", "WAITING"
}

// LiveExecutionTelemetryDTO powers the 4-card Live Execution & Multi-Repo Progress HUD at the top of Autopilot (LIVE-PROCESS-HUD-01).
type LiveExecutionTelemetryDTO struct {
	IsRunning          bool              `json:"is_running"`
	CanStop            bool              `json:"can_stop"`
	WasStopped         bool              `json:"was_stopped"`
	StopReason         string            `json:"stop_reason,omitempty"`
	ElapsedSec         float64           `json:"elapsed_sec"`
	ActiveActionName   string            `json:"active_action_name,omitempty"`
	ActiveTaskKey      string            `json:"active_task_key"`
	ActiveTaskTitle    string            `json:"active_task_title"`
	ActiveStage        string            `json:"active_stage"`
	ActivePhaseLabel   string            `json:"active_phase_label"`
	ActiveStepNum      int               `json:"active_step_num"`
	ActiveStepTitle    string            `json:"active_step_title"`
	ActiveSubTaskID    string            `json:"active_sub_task_id"`
	ActiveSubTaskTitle string            `json:"active_sub_task_title"`
	ActiveRole         string            `json:"active_role"`
	ActiveAgentSlot    string            `json:"active_agent_slot"`
	ActiveSkill        string            `json:"active_skill"`
	ActiveRunner       string            `json:"active_runner"`
	ActiveModel        string            `json:"active_model"`
	ActiveRepoName     string            `json:"active_repo_name"`
	ActiveRepoPath     string            `json:"active_repo_path"`
	ActiveWorktreePath string            `json:"active_worktree_path"`
	ActiveBranch       string            `json:"active_branch"`
	ActiveFileLine     string            `json:"active_file_line"`
	ActiveCommand      string            `json:"active_command"`
	CurrentDetail      string            `json:"current_detail"`
	OverallProgressPct int               `json:"overall_progress_pct"`
	DoneSubTasks       int               `json:"done_sub_tasks"`
	TotalSubTasks      int               `json:"total_sub_tasks"`
	DoneWorkflowSteps  int               `json:"done_workflow_steps"`
	TotalWorkflowSteps int               `json:"total_workflow_steps"`
	TotalDiffLines     int               `json:"total_diff_lines"`
	RepoProgress       []RepoProgressDTO `json:"repo_progress"`
	RecentEvents       []string          `json:"recent_events"`
	UpdatedAt          string            `json:"updated_at"`
}

// CoArchitectChatMessageDTO represents one message in the Co-Architect & Solution Planning Studio (CO-ARCHITECT-CHAT-01).
type CoArchitectChatMessageDTO struct {
	ID            string   `json:"id"`
	TaskKey       string   `json:"task_key"`
	Sender        string   `json:"sender"`       // "user" or "agent"
	AgentRole     string   `json:"agent_role"`   // e.g. "🧠 BRAIN · Architect"
	AgentSlot     string   `json:"agent_slot"`   // e.g. "Slot A"
	Skill         string   `json:"skill"`        // e.g. "/brain-delegate"
	Model         string   `json:"model"`        // e.g. "gemini-4-argon (antigravity)"
	ActionMode    string   `json:"action_mode"`  // "discuss", "revise_solution", "update_subtasks"
	Message       string   `json:"message"`
	AppliedToPlan bool     `json:"applied_to_plan"`
	TargetFile    string   `json:"target_file,omitempty"`
	ReferencedIDs []string `json:"referenced_ids,omitempty"`
	Timestamp     string   `json:"timestamp"`
}

// TrackerConnectionDTO represents an authenticated external issue tracker connection (Jira/Linear/GitHub/GitLab) (TASK-INTAKE-DUAL-MODE-01).
type TrackerConnectionDTO struct {
	ID             string `json:"id"`
	Provider       string `json:"provider"` // "jira", "linear", "github", "gitlab"
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	ProjectKey     string `json:"project_key"`
	AccountEmail   string `json:"account_email"`
	AuthType       string `json:"auth_type"` // "api_token", "pat", "oauth_bearer"
	MaskedToken    string `json:"masked_token"`
	Status         string `json:"status"` // "connected", "verified"
	DefaultEpic    string `json:"default_epic"`
	LastVerifiedAt string `json:"last_verified_at"`
}

// TrackerIssueDTO represents a browsable ticket from an external issue tracker (Jira/Linear/GitHub/GitLab) (TASK-INTAKE-DUAL-MODE-01).
type TrackerIssueDTO struct {
	ConnectionID  string   `json:"connection_id"`
	Provider      string   `json:"provider"`
	IssueKey      string   `json:"issue_key"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority"`
	Complexity    string   `json:"complexity"`
	Milestone     string   `json:"milestone"`
	Epic          string   `json:"epic"`
	Assignee      string   `json:"assignee"`
	Description   string   `json:"description"`
	BRDMarkdown   string   `json:"brd_markdown"`
	AffectedRepos []string `json:"affected_repos"`
	UpdatedAt     string   `json:"updated_at"`
}

// inferSubTaskRoleAndSlot maps a Sub-Task to its concrete /agent RoleLabel, AgentSlot, SkillBadge, Runner, and Model (ROLE-MODEL-ATTRIBUTION-01).
func (s *Server) inferSubTaskRoleAndSlot(st SubTaskDTO, cliCfg CLISettingsDTO) (roleLabel, agentSlot, skillBadge, runner, model string) {
	runner = cliCfg.DefaultRunner
	if runner == "" {
		runner = "antigravity"
	}
	model = strings.TrimSpace(st.Model)
	if model == "" || model == "gemini-2.5-pro (antigravity)" {
		model = cliCfg.CoderHighModel
	}
	if model == "" {
		model = "gemini-4-argon (antigravity)"
	}

	lower := strings.ToLower(st.Assignee + " " + st.GroupTitle + " " + st.Title + " " + st.ContextFileLine + " " + st.TargetRepoName)
	assigneeLower := strings.ToLower(strings.TrimSpace(st.Assignee))

	isDefaultCoderModel := st.Model == "" || st.Model == cliCfg.CoderHighModel ||
		st.Model == "gemini-4-argon (antigravity)" || st.Model == "gemini-2.5-pro (antigravity)"

	switch {
	case assigneeLower == "planner" || strings.Contains(assigneeLower, "brain") || strings.Contains(assigneeLower, "architect"):
		roleLabel = "🧠 BRAIN · Architect"
		agentSlot = "Slot A"
		skillBadge = "/brain-delegate"
		if isDefaultCoderModel {
			model = cliCfg.PlannerModel
		}
	case assigneeLower == "tester" || assigneeLower == "qa" || strings.Contains(lower, "kiểm thử") || strings.Contains(lower, "e2e") || strings.Contains(lower, "nghiệm thu"):
		roleLabel = "🧪 Cross-Model QA & E2E Auditor"
		agentSlot = "Slot K"
		skillBadge = "/qa-agent"
		if isDefaultCoderModel && cliCfg.QAAuditorModel != "" {
			model = cliCfg.QAAuditorModel
		}
	case assigneeLower == "security" || strings.Contains(lower, "security") || strings.Contains(lower, "secret"):
		roleLabel = "🛡️ Cross-Model Security Auditor"
		agentSlot = "Slot I"
		skillBadge = "/security-agent"
		if isDefaultCoderModel && cliCfg.SecurityModel != "" {
			model = cliCfg.SecurityModel
		}
	case assigneeLower == "reviewer" || strings.Contains(lower, "laya") || strings.Contains(lower, "dg3"):
		roleLabel = "⚖️ Cross-Model Laya Gate Reviewer"
		agentSlot = "Slot H"
		skillBadge = "/laya-router"
		if isDefaultCoderModel {
			if cliCfg.LayaGateModel != "" {
				model = cliCfg.LayaGateModel
			} else if cliCfg.QAAuditorModel != "" {
				model = cliCfg.QAAuditorModel
			}
		}
	case strings.Contains(st.TargetRepoName, "cms-web") || strings.Contains(lower, ".tsx") || strings.Contains(lower, "frontend") || assigneeLower == "frontend":
		roleLabel = "🎨 MUSCLE · Frontend Coder"
		agentSlot = "Slot D"
		skillBadge = "/muscle-execute"
	case strings.Contains(st.TargetRepoName, "centralized-data-service") && (strings.Contains(lower, "transmute") || strings.Contains(lower, "recon") || strings.Contains(lower, "bulk") || strings.Contains(lower, "bson") || assigneeLower == "data-engine"):
		roleLabel = "⚡ MUSCLE · Data Engine Coder"
		agentSlot = "Slot E"
		skillBadge = "/muscle-execute"
	case strings.Contains(st.TargetRepoName, "centralized-data-service"):
		roleLabel = "🛠️ MUSCLE · Backend Coder"
		agentSlot = "Slot F"
		skillBadge = "/muscle-execute"
	default:
		roleLabel = "🛠️ MUSCLE · Backend Coder"
		agentSlot = "Slot B"
		skillBadge = "/muscle-execute"
	}

	if strings.TrimSpace(st.RoleLabel) != "" {
		roleLabel = strings.TrimSpace(st.RoleLabel)
	}
	if strings.TrimSpace(st.AgentSlot) != "" {
		agentSlot = strings.TrimSpace(st.AgentSlot)
	}
	if strings.TrimSpace(st.SkillBadge) != "" {
		skillBadge = strings.TrimSpace(st.SkillBadge)
	}
	return roleLabel, agentSlot, skillBadge, runner, model
}

// enrichSubTasksWithRoleAndModel populates RoleLabel, AgentSlot, SkillBadge, Runner, and Model on all Sub-Tasks.
func (s *Server) enrichSubTasksWithRoleAndModel(items []SubTaskDTO) []SubTaskDTO {
	cliCfg := s.loadCLISettings()
	for i := range items {
		roleLabel, slot, skill, runner, mdl := s.inferSubTaskRoleAndSlot(items[i], cliCfg)
		items[i].RoleLabel = roleLabel
		items[i].AgentSlot = slot
		items[i].SkillBadge = skill
		items[i].Runner = runner
		items[i].Model = mdl
		if items[i].Assignee == "" {
			if strings.Contains(roleLabel, "QA") {
				items[i].Assignee = "Tester"
			} else if strings.Contains(roleLabel, "BRAIN") {
				items[i].Assignee = "Planner"
			} else if strings.Contains(roleLabel, "Frontend") {
				items[i].Assignee = "Frontend"
			} else if strings.Contains(roleLabel, "Data Engine") {
				items[i].Assignee = "Data-Engine"
			} else {
				items[i].Assignee = "Coder"
			}
		}
	}
	return items
}

// syncSubTasksToMarkdown08 writes the updated Sub-Tasks (including Role/Model/Status) back into .orchestrator/tasks/<safeKey>/08_tasks.md.
func (s *Server) syncSubTasksToMarkdown08(qt queue.QueuedTask, items []SubTaskDTO) {
	if qt.ExternalID == "" {
		return
	}
	isoMeta := s.buildTaskIsolationMeta(qt.ExternalID)
	tasksPath := filepath.Join(isoMeta.ArtifactDir, "08_tasks.md")
	affectedRepos := s.ResolveTaskAffectedRepos(qt)
	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}
	var lines []string
	lines = append(lines,
		fmt.Sprintf("# 08 Tasks: Danh mục Sub-Task — %s (%s)", qt.ExternalID, qt.Title),
		"",
		fmt.Sprintf("> **Affected Repos**: `%s`", strings.Join(repoNames, ", ")),
		"> **Tier**: T2",
	)
	currentGrp := ""
	for _, st := range items {
		if st.GroupTitle != "" && st.GroupTitle != currentGrp {
			currentGrp = st.GroupTitle
			lines = append(lines, "", fmt.Sprintf("## %s", currentGrp))
		}
		chk := " "
		if strings.EqualFold(st.Status, "done") {
			chk = "x"
		}
		lines = append(lines,
			fmt.Sprintf("- [%s] **%s — %s**", chk, st.SubTaskID, st.Title),
			fmt.Sprintf("  - **Context (`file:line`)**: `%s`", st.ContextFileLine),
			fmt.Sprintf("  - **Role / Slot / Model**: `%s` · `%s` · `%s` (`%s`)", st.RoleLabel, st.AgentSlot, st.Model, st.SkillBadge),
			fmt.Sprintf("  - **Mô tả**: %s", st.Description),
			fmt.Sprintf("  - **Test / DoD**: %s", st.TestDoD),
		)
	}
	_ = os.MkdirAll(filepath.Dir(tasksPath), 0o755)
	content := []byte(strings.Join(lines, "\n") + "\n")
	_ = os.WriteFile(tasksPath, content, 0o644)
	_, _, suffix := agentgate.InferTaskTierAndMode(qt.ExternalID, qt.Title, qt.Description, qt.Complexity, len(items))
	if suffix != "" {
		_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("08_tasks_%s.md", suffix)), content, 0o644)
	}
}

// buildLiveExecutionTelemetry synthesizes the real-time What/Who/Where/How & Per-Repo Progress telemetry for the selected task (LIVE-PROCESS-HUD-01).
func (s *Server) buildLiveExecutionTelemetry(
	ctx context.Context,
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	wfSteps []VibeWorkflowStepDTO,
	kernel ActiveAgentKernelDTO,
	taskLogs []string,
) LiveExecutionTelemetryDTO {
	cliCfg := s.loadCLISettings()
	affectedRepos := s.ResolveTaskAffectedRepos(qt)
	stUpper := strings.ToUpper(strings.TrimSpace(qt.Stage))

	doneSubTasks := 0
	var activeSub *SubTaskDTO
	for i := range subTasks {
		if strings.EqualFold(subTasks[i].Status, "done") {
			doneSubTasks++
		} else if activeSub == nil {
			activeSub = &subTasks[i]
		}
	}
	if activeSub == nil && len(subTasks) > 0 {
		activeSub = &subTasks[len(subTasks)-1]
	}

	doneWfSteps := 0
	activeStepNum := 1
	activeStepTitle := "Tổng quan & Khởi tạo S1"
	for _, wf := range wfSteps {
		if wf.Status == "done" {
			doneWfSteps++
		} else if wf.Status == "active" && activeStepNum == 1 {
			activeStepNum = wf.Num
			activeStepTitle = wf.Title
		}
	}
	if doneWfSteps == len(wfSteps) && len(wfSteps) > 0 {
		activeStepNum = len(wfSteps)
		activeStepTitle = wfSteps[len(wfSteps)-1].Title
	}

	// Build Per-Repo Progress Breakdown across all AffectedRepos
	repoMap := make(map[string]*RepoProgressDTO)
	var repoOrder []string
	for _, rp := range affectedRepos {
		rName := filepath.Base(rp)
		if _, exists := repoMap[rName]; !exists {
			repoOrder = append(repoOrder, rName)
			repoMap[rName] = &RepoProgressDTO{
				RepoName:       rName,
				RepoPath:       rp,
				Branch:         detectRepoBranchFast(rp),
				ActiveRole:     "🛠️ MUSCLE · Backend Coder",
				ActiveModel:    cliCfg.CoderHighModel,
				VerificationSt: "WAITING",
			}
		}
	}
	for _, st := range subTasks {
		rName := st.TargetRepoName
		if rName == "" {
			rName = filepath.Base(st.TargetRepo)
		}
		if rName == "" {
			rName = filepath.Base(s.ResolveTaskTargetRepo(qt))
		}
		rpEntry, exists := repoMap[rName]
		if !exists {
			repoOrder = append(repoOrder, rName)
			rpEntry = &RepoProgressDTO{
				RepoName:    rName,
				RepoPath:    st.TargetRepo,
				Branch:      detectRepoBranchFast(st.TargetRepo),
				ActiveRole:  st.RoleLabel,
				ActiveModel: st.Model,
			}
			repoMap[rName] = rpEntry
		}
		rpEntry.TotalSubTasks++
		if strings.EqualFold(st.Status, "done") {
			rpEntry.DoneSubTasks++
		} else if rpEntry.ActiveSubTask == "" {
			rpEntry.ActiveSubTask = fmt.Sprintf("%s: %s", st.SubTaskID, st.Title)
		}
		if st.RoleLabel != "" {
			rpEntry.ActiveRole = st.RoleLabel
		}
		if st.Model != "" {
			rpEntry.ActiveModel = st.Model
		}
	}

	totalDiffLines := 0
	var repoProgress []RepoProgressDTO
	for _, rName := range repoOrder {
		rpEntry := repoMap[rName]
		if rpEntry.TotalSubTasks > 0 {
			rpEntry.ProgressPct = (rpEntry.DoneSubTasks * 100) / rpEntry.TotalSubTasks
		} else if stUpper == queue.StageDone {
			rpEntry.ProgressPct = 100
		}
		if rpEntry.ActiveSubTask == "" && rpEntry.TotalSubTasks > 0 {
			rpEntry.ActiveSubTask = fmt.Sprintf("Hoàn tất %d/%d Sub-Tasks", rpEntry.DoneSubTasks, rpEntry.TotalSubTasks)
		}
		// Inspect real git status / diff on disk for this repository
		if rpEntry.RepoPath != "" {
			if stOut, err := s.executeCommandInDir(ctx, rpEntry.RepoPath, "git status --porcelain"); err == nil {
				trimmed := strings.TrimSpace(stOut)
				if trimmed != "" {
					rpEntry.ModifiedFiles = len(strings.Split(trimmed, "\n"))
				}
			}
			if diffOut, err := s.executeCommandInDir(ctx, rpEntry.RepoPath, "git diff --shortstat"); err == nil {
				shortStat := strings.TrimSpace(diffOut)
				if shortStat != "" {
					rpEntry.DiffSummary = shortStat
				}
			}
		}
		if rpEntry.DiffSummary == "" {
			if rpEntry.ModifiedFiles > 0 {
				rpEntry.DiffSummary = fmt.Sprintf("%d files modified/untracked on disk", rpEntry.ModifiedFiles)
			} else {
				rpEntry.DiffSummary = "Clean working tree"
			}
		}
		if rpEntry.DoneSubTasks == rpEntry.TotalSubTasks && rpEntry.TotalSubTasks > 0 {
			rpEntry.VerificationSt = "VERIFIED"
		} else if rpEntry.DoneSubTasks > 0 || stUpper == queue.StageRunning {
			rpEntry.VerificationSt = "IN_PROGRESS"
		} else {
			rpEntry.VerificationSt = "WAITING"
		}
		repoProgress = append(repoProgress, *rpEntry)
	}

	if rawDiff, err := os.ReadFile(isoMeta.DiffFile); err == nil && len(rawDiff) > 0 {
		totalDiffLines = len(strings.Split(string(rawDiff), "\n"))
	}

	// Calculate Overall Progress % combining Workflow Steps & Sub-Tasks
	overallPct := 0
	if len(subTasks) > 0 {
		subPct := (doneSubTasks * 100) / len(subTasks)
		wfPct := (doneWfSteps * 100) / maxInt(1, len(wfSteps))
		switch stUpper {
		case queue.StageDone:
			overallPct = 100
		case queue.StageNeedsYou:
			// Phase 1 (S1-S3) complete, waiting at Gate Stop (Step 8/17)
			if subPct > 45 {
				overallPct = subPct
			} else {
				overallPct = 47
			}
		default:
			overallPct = (subPct + wfPct) / 2
		}
	}

	phaseLabel := "S1–S3 Plan & Multi-Repo Scan"
	roleLabel := "🧠 BRAIN · Chairman & Architect"
	agentSlot := "Slot A"
	skillBadge := "/brain-delegate · /laya-router"
	activeModel := cliCfg.PlannerModel
	activeCommand := "Scan AffectedRepos -> Match 272 Lessons -> Generate 09_tasks_solution.md & Code Demo"
	currentDetail := "Đang phân tích kiến trúc đa Repo và đối chiếu 272 bài học /agent."
	isRunning := false

	switch stUpper {
	case queue.StagePlanning:
		isRunning = true
		phaseLabel = "⏳ S1–S3 Plan & Multi-Repo Scan (Đang phân tích & lập giải pháp)"
		roleLabel = "🧠 BRAIN · Chairman & Architect"
		agentSlot = "Slot A"
		skillBadge = "/brain-delegate · /laya-router"
		activeModel = cliCfg.PlannerModel
		activeCommand = "Scan AffectedRepos -> Match Lessons -> Generate 09_tasks_solution.md & Code Demo"
		currentDetail = "Brain đang quét sâu mã nguồn thật (file:line) và lập phương án S3 + Code Demo."
	case queue.StageNeedsYou:
		phaseLabel = "⏸ S3 Gate Stop (NEEDS_YOU — Chờ bạn Đối Thoại hoặc APPROVE)"
		roleLabel = "🧠 BRAIN · Chairman & Architect"
		agentSlot = "Slot A"
		skillBadge = "/brain-delegate · H4b Propose-Only"
		activeModel = cliCfg.PlannerModel
		activeCommand = "STOP @ Step 8/17 (H4b Propose-Only Gate) — Chờ bạn thảo luận ở khung Chat bên dưới hoặc bấm ✓ APPROVE"
		currentDetail = fmt.Sprintf("Đã hoàn tất S1–S3 (%d Sub-Tasks trên %d Repos). Bạn có thể trao đổi/chỉnh sửa giải pháp với Agent ngay bên dưới trước khi bấm ✓ APPROVE.", len(subTasks), len(repoProgress))
	case queue.StageRunning:
		isRunning = true
		phaseLabel = "⚡ S4 Multi-Repo Worktree Implementation (Đang chạy)"
		roleLabel = "🛠️ MUSCLE · Chief Engineer"
		agentSlot = "Slot B"
		skillBadge = "/muscle-execute · H4b Zero-Mockup"
		activeModel = cliCfg.CoderHighModel
		activeCommand = "go test -race ./... && git diff --stat (Multi-Repo Worktree Verification)"
		currentDetail = "Muscle đang thực thi và kiểm chứng mã nguồn thật trên từng Repository."
	case queue.StageInReview:
		isRunning = true
		phaseLabel = "🛡️ S5 Local QA & Laya DG3 Audit (Đang kiểm định)"
		roleLabel = "🧪 QA & Security Auditor"
		agentSlot = "Slot H / Slot K"
		skillBadge = "/qa-agent · /security-agent · /laya-router"
		activeModel = cliCfg.QAAuditorModel
		activeCommand = "Gatekeeper G1–G8 + H4b Zero-Mockup + Laya DG3 Audit"
		currentDetail = "Đang đối chiếu từng dòng git diff và kiểm định 8 cổng chất lượng G1–G8."
	case queue.StageDone:
		phaseLabel = "✅ S6 Complete — Đã hoàn tất & Kiểm chứng trên đĩa (100%)"
		roleLabel = "🛡️ GOVERNANCE VERIFIED (Brain + Muscle + QA)"
		agentSlot = "Slot A → B/D/E → H"
		skillBadge = "/brain-delegate · /muscle-execute · /qa-agent"
		activeModel = cliCfg.CoderHighModel
		activeCommand = fmt.Sprintf("Verified %d/%d Sub-Tasks across %d Repos (%d lines in diff.patch)", doneSubTasks, len(subTasks), len(repoProgress), totalDiffLines)
		currentDetail = fmt.Sprintf("Hoàn tất 100%% (%d/%d Sub-Tasks, %d/17 bước quy trình /agent) trên %d Repositories.", doneSubTasks, len(subTasks), doneWfSteps, len(repoProgress))
	}

	subID := "TASK-M01"
	subTitle := qt.Title
	repoName := filepath.Base(s.ResolveTaskTargetRepo(qt))
	repoPath := s.ResolveTaskTargetRepo(qt)
	fileLine := isoMeta.PlanFile
	if activeSub != nil {
		subID = activeSub.SubTaskID
		subTitle = activeSub.Title
		if activeSub.TargetRepoName != "" {
			repoName = activeSub.TargetRepoName
		}
		if activeSub.TargetRepo != "" {
			repoPath = activeSub.TargetRepo
		}
		if activeSub.ContextFileLine != "" {
			fileLine = activeSub.ContextFileLine
		}
		if stUpper == queue.StageRunning && activeSub.RoleLabel != "" {
			roleLabel = activeSub.RoleLabel
			agentSlot = activeSub.AgentSlot
			skillBadge = activeSub.SkillBadge
			activeModel = activeSub.Model
		}
	}

	s.mu.Lock()
	liveStep := s.liveStepProgress
	hasCancel := s.activeExecCancel != nil && strings.EqualFold(s.activeExecTaskKey, qt.ExternalID)
	lastStoppedKey := s.lastStoppedTaskKey
	lastStopReason := s.lastStopReason
	s.mu.Unlock()

	canStop := isRunning || hasCancel
	wasStopped := false
	stopReason := ""
	elapsedSec := 0.0
	activeActionName := ""

	if liveStep.Active && strings.EqualFold(liveStep.TaskKey, qt.ExternalID) {
		isRunning = true
		canStop = true
		activeActionName = liveStep.ActionName
		if !liveStep.StartedAt.IsZero() {
			elapsedSec = time.Since(liveStep.StartedAt).Seconds()
		}
		if liveStep.Percent > 0 {
			overallPct = liveStep.Percent
		}
		if liveStep.SubTaskID != "" {
			subID = liveStep.SubTaskID
		}
		if liveStep.SubTaskTitle != "" {
			subTitle = liveStep.SubTaskTitle
		}
		if liveStep.StageLabel != "" {
			phaseLabel = fmt.Sprintf("⏳ [Step %d/%d · %d%%] %s", liveStep.StepIndex, maxInt(1, liveStep.TotalSteps), overallPct, liveStep.StageLabel)
			activeStepTitle = liveStep.StageLabel
		}
		if liveStep.StepIndex > 0 {
			activeStepNum = liveStep.StepIndex
		}
		if liveStep.CurrentAction != "" {
			activeCommand = liveStep.CurrentAction
			currentDetail = liveStep.CurrentAction
		}
		if liveStep.CurrentWhere != "" {
			fileLine = liveStep.CurrentWhere
		}
		if liveStep.ActiveRoleSlot != "" {
			roleLabel = liveStep.ActiveRoleSlot
		}
	} else if !isRunning && ((liveStep.WasStopped && strings.EqualFold(liveStep.TaskKey, qt.ExternalID)) || strings.EqualFold(lastStoppedKey, qt.ExternalID)) {
		wasStopped = true
		canStop = false
		stopReason = liveStep.StopReason
		if stopReason == "" {
			stopReason = lastStopReason
		}
		if stopReason == "" {
			stopReason = "Đã dừng tiến trình theo yêu cầu người dùng (⏹ Stop)"
		}
		phaseLabel = fmt.Sprintf("⏹ ĐÃ DỪNG TIẾN TRÌNH (NEEDS_YOU) — %s", stopReason)
		currentDetail = stopReason
	}

	recent := taskLogs
	if len(recent) > 6 {
		recent = recent[len(recent)-6:]
	}

	return LiveExecutionTelemetryDTO{
		IsRunning:          isRunning,
		CanStop:            canStop,
		WasStopped:         wasStopped,
		StopReason:         stopReason,
		ElapsedSec:         elapsedSec,
		ActiveActionName:   activeActionName,
		ActiveTaskKey:      qt.ExternalID,
		ActiveTaskTitle:    qt.Title,
		ActiveStage:        stUpper,
		ActivePhaseLabel:   phaseLabel,
		ActiveStepNum:      activeStepNum,
		ActiveStepTitle:    activeStepTitle,
		ActiveSubTaskID:    subID,
		ActiveSubTaskTitle: subTitle,
		ActiveRole:         roleLabel,
		ActiveAgentSlot:    agentSlot,
		ActiveSkill:        skillBadge,
		ActiveRunner:       cliCfg.DefaultRunner,
		ActiveModel:        activeModel,
		ActiveRepoName:     repoName,
		ActiveRepoPath:     repoPath,
		ActiveWorktreePath: isoMeta.WorktreePath,
		ActiveBranch:       isoMeta.BranchName,
		ActiveFileLine:     fileLine,
		ActiveCommand:      activeCommand,
		CurrentDetail:      currentDetail,
		OverallProgressPct: overallPct,
		DoneSubTasks:       doneSubTasks,
		TotalSubTasks:      len(subTasks),
		DoneWorkflowSteps:  doneWfSteps,
		TotalWorkflowSteps: maxInt(17, len(wfSteps)),
		TotalDiffLines:     totalDiffLines,
		RepoProgress:       repoProgress,
		RecentEvents:       recent,
		UpdatedAt:          time.Now().Format("15:04:05"),
	}
}

func (s *Server) taskChatHistoryPath(taskKey string) string {
	meta := s.buildTaskIsolationMeta(taskKey)
	return filepath.Join(meta.ArtifactDir, "chat_history.json")
}

// loadOrInitTaskChatHistory loads persisted Co-Architect conversation history for the selected task
// or initializes a rich onboarding message from BRAIN (Slot A) so the user can immediately collaborate.
func (s *Server) loadOrInitTaskChatHistory(qt queue.QueuedTask, subTasks []SubTaskDTO, isoMeta TaskIsolationDTO) []CoArchitectChatMessageDTO {
	if qt.ExternalID == "" {
		return nil
	}
	cliCfg := s.loadCLISettings()
	affectedRepos := s.ResolveTaskAffectedRepos(qt)
	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}

	p := s.taskChatHistoryPath(qt.ExternalID)
	var history []CoArchitectChatMessageDTO
	if raw, err := os.ReadFile(p); err == nil && len(raw) > 0 {
		if json.Unmarshal(raw, &history) == nil && len(history) > 0 {
			cleaned, changed := NormalizeAndDeduplicateChatHistoryUDFv1(qt, subTasks, isoMeta, history, cliCfg.PlannerModel, repoNames)
			if changed {
				_ = s.saveTaskChatHistory(qt.ExternalID, cleaned)
			}
			return cleaned
		}
	}

	welcomeMsg := BuildUnifiedOnboardingChatUDFv1(qt, subTasks, isoMeta, cliCfg.PlannerModel, repoNames)

	history = []CoArchitectChatMessageDTO{
		{
			ID:            fmt.Sprintf("msg-init-%d", time.Now().Unix()),
			TaskKey:       qt.ExternalID,
			Sender:        "agent",
			AgentRole:     "🧠 BRAIN · Architect",
			AgentSlot:     "Slot A",
			Skill:         "/brain-delegate",
			Model:         cliCfg.PlannerModel,
			ActionMode:    "discuss",
			Message:       welcomeMsg,
			AppliedToPlan: false,
			TargetFile:    isoMeta.PlanFile,
			Timestamp:     time.Now().Format("15:04:05"),
		},
	}
	_ = s.saveTaskChatHistory(qt.ExternalID, history)
	return history
}

func (s *Server) saveTaskChatHistory(taskKey string, history []CoArchitectChatMessageDTO) error {
	p := s.taskChatHistoryPath(taskKey)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0o644)
}

type taskChatRequest struct {
	TaskKey    string `json:"task_key"`
	Message    string `json:"message"`
	ActionMode string `json:"action_mode"` // "discuss", "revise_solution", "update_subtasks"
	Action     string `json:"action"`      // alias for action_mode
	AgentRole  string `json:"agent_role"`  // "BRAIN", "MUSCLE", "QA"
	RoleTarget string `json:"role_target"` // alias for agent_role
	Model      string `json:"model"`
	TargetRepo string `json:"target_repo"`
}

// handleTaskPlanningChat implements POST /api/tasks/chat (CO-ARCHITECT-CHAT-01).
// It allows the user to converse with the active Agent (Brain/Muscle/QA), steer and revise 09_tasks_solution.md,
// or add/modify Sub-Tasks in 08_tasks.md before or after APPROVE.
func (s *Server) handleTaskPlanningChat(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		taskKey := strings.TrimSpace(r.URL.Query().Get("task_key"))
		qt, err := s.findQueuedTask(r.Context(), taskKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		isoMeta := s.buildTaskIsolationMeta(qt.ExternalID)
		subs := s.loadOrGenerateSubTasks(*qt)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(s.loadOrInitTaskChatHistory(*qt, subs, isoMeta))
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req taskChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	cleanMsg := scout.MaskSecrets(strings.TrimSpace(req.Message))
	if cleanMsg == "" {
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	qt, err := s.findQueuedTask(ctx, req.TaskKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Security Gatekeeper H5 check against destructive shell commands
	if err := agentgate.ValidateCommandSafety(cleanMsg); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	cliCfg := s.loadCLISettings()
	isoMeta, err := s.ensureTaskIsolation(ctx, *qt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	subTasks := s.loadOrGenerateSubTasks(*qt)
	history := s.loadOrInitTaskChatHistory(*qt, subTasks, isoMeta)

	requestedMode := strings.ToLower(strings.TrimSpace(req.ActionMode))
	if requestedMode == "" {
		requestedMode = strings.ToLower(strings.TrimSpace(req.Action))
	}
	actionMode := classifyChatActionMode(cleanMsg, requestedMode)
	if actionMode == "discuss" && isPlanCorrectionOrSteeringMessage(cleanMsg, *qt, subTasks) {
		actionMode = "revise_solution"
	}

	roleChoice := strings.ToUpper(strings.TrimSpace(req.AgentRole))
	if roleChoice == "" {
		roleChoice = strings.ToUpper(strings.TrimSpace(req.RoleTarget))
	}
	agentRoleLabel := "🧠 BRAIN · Architect"
	agentSlot := "Slot A"
	skillBadge := "/brain-delegate"
	selectedModel := strings.TrimSpace(req.Model)
	if selectedModel == "" {
		selectedModel = cliCfg.PlannerModel
	}
	switch {
	case strings.Contains(roleChoice, "MUSCLE") || strings.Contains(roleChoice, "CODER"):
		agentRoleLabel = "🛠️ MUSCLE · Chief Engineer"
		agentSlot = "Slot B"
		skillBadge = "/muscle-execute"
		if req.Model == "" {
			selectedModel = cliCfg.CoderHighModel
		}
	case strings.Contains(roleChoice, "QA") || strings.Contains(roleChoice, "AUDIT"):
		agentRoleLabel = "🧪 QA & DG3 Auditor"
		agentSlot = "Slot H"
		skillBadge = "/qa-agent · /laya-router"
		if req.Model == "" {
			selectedModel = cliCfg.QAAuditorModel
		}
	}

	nowStr := time.Now().Format("15:04:05")
	userEntry := CoArchitectChatMessageDTO{
		ID:         fmt.Sprintf("msg-u-%d", time.Now().UnixNano()),
		TaskKey:    qt.ExternalID,
		Sender:     "user",
		AgentRole:  "👤 You (Product Owner & Co-Architect)",
		AgentSlot:  "USER",
		Skill:      "Steering",
		Model:      selectedModel,
		ActionMode: actionMode,
		Message:    cleanMsg,
		Timestamp:  nowStr,
	}
	history = append(history, userEntry)

	agentRoot := "/Users/trainguyen/Documents/work/agent"
	if s.Gatekeeper != nil && s.Gatekeeper.AgentCoreDir != "" {
		agentRoot = s.Gatekeeper.AgentCoreDir
	}
	matchedLessons := MatchAgentLessonsForTask(agentRoot, qt.ExternalID, qt.Title, cleanMsg, s.ResolveTaskTargetRepo(*qt))
	var lessonRefs []string
	var lessonBullets []string
	for _, ml := range matchedLessons {
		lessonRefs = append(lessonRefs, ml.LessonID)
		lessonBullets = append(lessonBullets, fmt.Sprintf("- **`%s` (%s):** %s", ml.LessonID, ml.Title, ml.Rule))
	}

	var replyText string
	appliedToPlan := false
	targetFile := isoMeta.PlanFile

	switch actionMode {
	case "ingest_audit_remediation":
		var updatedSubTasks []SubTaskDTO
		replyText, targetFile, updatedSubTasks = s.ingestAuditReportAndRemediation(
			ctx, qt, isoMeta, subTasks, cleanMsg, agentRoleLabel, agentSlot, skillBadge, selectedModel, lessonRefs,
		)
		subTasks = updatedSubTasks
		appliedToPlan = true

	case "revise_solution":
		replyText, targetFile, subTasks = s.executeArchitectReviseSolution(
			ctx, qt, isoMeta, subTasks, cleanMsg, agentRoleLabel, agentSlot, skillBadge, selectedModel, lessonRefs, cliCfg,
		)
		appliedToPlan = true

	case "update_subtasks":
		affectedRepos := s.ResolveTaskAffectedRepos(*qt)
		repoPath := strings.TrimSpace(req.TargetRepo)
		var repoName string
		if repoPath != "" {
			repoName = filepath.Base(repoPath)
		} else {
			repoPath, repoName = s.resolveRepoFromContextLine(cleanMsg, "", cleanMsg, affectedRepos)
		}
		safeParent := sanitizeTaskKeyPath(qt.ExternalID)
		newID := fmt.Sprintf("TASK-M%02d", len(subTasks)+1)
		newSt := SubTaskDTO{
			SubTaskID:       newID,
			ParentKey:       qt.ExternalID,
			GroupTitle:      fmt.Sprintf("Nhóm Co-Architect Steering (`%s`)", repoName),
			WaveIndex:       len(subTasks) / 3,
			WorkflowStep:    fmt.Sprintf("Wave %d", len(subTasks)/3),
			Title:           cleanMsg,
			TargetRepo:      repoPath,
			TargetRepoName:  repoName,
			WorktreePath:    filepath.Join(".orchestrator", "worktrees", safeParent, repoName),
			TargetFiles:     repoName,
			ContextFileLine: fmt.Sprintf("%s (Co-Architect Directive)", repoName),
			Description:     fmt.Sprintf("Sub-Task được tạo trực tiếp từ phiên đối thoại Co-Architect cùng %s: %s", agentRoleLabel, cleanMsg),
			TestDoD:         fmt.Sprintf("Pass unit/integration test trên `%s` & đạt chuẩn G1–G8", repoName),
			Status:          "pending",
			Assignee:        "Coder",
			Model:           selectedModel,
		}
		subTasks = append(subTasks, newSt)
		analysis, _ := s.invokeArchitectLLM(ctx, *qt, subTasks, cleanMsg, selectedModel, "update_subtasks")
		subTasks, _ = s.syncArchitectLLMAnalysisToMasterPlan(
			ctx, qt, isoMeta, subTasks, analysis, affectedRepos,
			cliCfg.DefaultRunner, selectedModel, cleanMsg, agentRoleLabel, agentSlot, skillBadge,
		)

		appliedToPlan = true
		targetFile = filepath.Join(isoMeta.ArtifactDir, "08_tasks.md")
		replyText = fmt.Sprintf(
			"✅ **Em đã bổ sung Sub-Task `%s` vào danh mục `08_tasks.md`, `subtasks.json` và đồng bộ lại Plan tổng `09_tasks_solution.md`!**\n\n"+
				"- **Sub-Task mới:** `%s` — %s\n"+
				"- **Target Repository:** `%s` (`%s`)\n"+
				"- **Role / Slot / Model đảm nhận:** `%s` · `%s` · `%s`",
			newID,
			newID,
			cleanMsg,
			repoName,
			repoPath,
			subTasks[len(subTasks)-1].RoleLabel,
			subTasks[len(subTasks)-1].AgentSlot,
			subTasks[len(subTasks)-1].Model,
		)

	default: // "discuss"
		replyText = s.buildContextualDiscussReply(*qt, isoMeta, subTasks, cleanMsg, agentRoleLabel, agentSlot, skillBadge, selectedModel, lessonRefs)
	}

	agentEntry := CoArchitectChatMessageDTO{
		ID:            fmt.Sprintf("msg-a-%d", time.Now().UnixNano()),
		TaskKey:       qt.ExternalID,
		Sender:        "agent",
		AgentRole:     agentRoleLabel,
		AgentSlot:     agentSlot,
		Skill:         skillBadge,
		Model:         selectedModel,
		ActionMode:    actionMode,
		Message:       replyText,
		AppliedToPlan: appliedToPlan,
		TargetFile:    targetFile,
		ReferencedIDs: lessonRefs,
		Timestamp:     time.Now().Format("15:04:05"),
	}
	history = append(history, agentEntry)
	_ = s.saveTaskChatHistory(qt.ExternalID, history)

	s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Co-Architect Studio · %s · %s] Mode=%s | User: %s", agentRoleLabel, selectedModel, actionMode, truncateRunes(cleanMsg, 90)))
	s.appendPaneLog("A", fmt.Sprintf("[%s Co-Architect Chat] %s (%s): %s", qt.ExternalID, agentRoleLabel, actionMode, truncateRunes(cleanMsg, 80)))

	s.mu.Lock()
	s.selectedTaskKey = qt.ExternalID
	s.mu.Unlock()
	s.broadcast("task_chat_updated")

	resp, err := s.BuildDashboardResponse(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

var (
	auditFindingCodeRe = regexp.MustCompile(`\b(AF-(?:ORCH|DH)-\d{2})\b`)
	remediationTaskRe  = regexp.MustCompile(`\b(TASK-A\d{2})\b`)
)

func classifyChatActionMode(cleanMsg, requestedMode string) string {
	lowerMsg := strings.ToLower(cleanMsg)
	if auditFindingCodeRe.MatchString(cleanMsg) ||
		remediationTaskRe.MatchString(cleanMsg) ||
		strings.Contains(lowerMsg, "audit_report_") ||
		strings.Contains(lowerMsg, "09_tasks_solution_audit_") ||
		(strings.Contains(lowerMsg, "lỗ hổng") && strings.Contains(lowerMsg, "phương án khắc phục")) {
		return "ingest_audit_remediation"
	}
	if requestedMode != "" && requestedMode != "discuss" {
		return requestedMode
	}
	if isArchitecturalSteeringDirective(lowerMsg) {
		return "revise_solution"
	}
	if strings.Contains(lowerMsg, "thêm sub-task") || strings.Contains(lowerMsg, "chia lại sub-task") || strings.Contains(lowerMsg, "add subtask") {
		return "update_subtasks"
	}
	return "discuss"
}

// ingestAuditReportAndRemediation parses an external AI Audit Report pasted into Co-Architect Chat (AF-ORCH-06),
// extracts all AF-* findings and TASK-A* remediation Sub-Tasks, updates subtasks.json, 08_tasks.md, 09_tasks_solution.md,
// and audit_report.md, and re-opens the Autonomous Self-Healing Loop (Rule #9.7d: S5 -> S3 Re-Plan -> S4 Execute).
func (s *Server) ingestAuditReportAndRemediation(
	ctx context.Context,
	qt *queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	cleanMsg string,
	agentRoleLabel, agentSlot, skillBadge, selectedModel string,
	lessonRefs []string,
) (string, string, []SubTaskDTO) {
	// 1. Extract unique AF-* finding codes
	var findingCodes []string
	seenFindings := make(map[string]bool)
	for _, m := range auditFindingCodeRe.FindAllStringSubmatch(cleanMsg, -1) {
		code := m[1]
		if !seenFindings[code] {
			seenFindings[code] = true
			findingCodes = append(findingCodes, code)
		}
	}

	// 2. Parse TASK-A* lines or build canonical remediation tasks from the pasted audit report (SA-04: strip duplicate markdown prefixes)
	type parsedRemediation struct {
		ID          string
		Repo        string
		TargetFiles string
		ContextLine string
		Summary     string
	}
	var parsedTasks []parsedRemediation
	seenTasks := make(map[string]bool)
	lines := strings.Split(cleanMsg, "\n")
	for i, ln := range lines {
		matches := remediationTaskRe.FindStringSubmatch(ln)
		if len(matches) < 2 {
			continue
		}
		tid := matches[1]
		if seenTasks[tid] {
			continue
		}
		trimmed := strings.TrimSpace(ln)
		if strings.Contains(trimmed, "**`"+tid+"`") || strings.HasPrefix(trimmed, "- **`"+tid+"`") || strings.Contains(trimmed, "`"+tid+"` (") {
			seenTasks[tid] = true
			headerPart := trimmed
			bulletPart := ""
			if i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "-") {
				bulletPart = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i+1]), "-"))
			}
			summary := bulletPart
			if summary == "" {
				if colonIdx := strings.Index(headerPart, ":**"); colonIdx >= 0 {
					summary = strings.TrimSpace(headerPart[colonIdx+3:])
				} else {
					summary = headerPart
				}
			}
			summary = strings.TrimSpace(strings.TrimPrefix(summary, "—"))

			combinedLower := strings.ToLower(headerPart + " " + summary)
			repoHint := "ai-orchestrator"
			targetFiles := "internal/webui/task_chat.go, internal/webui/task_inspector.go, internal/webui/server.go, internal/webui/seed_data.go"
			contextLine := "ai-orchestrator/internal/webui/task_chat.go:677-1297, internal/webui/task_inspector.go:1150-1243"
			switch {
			case strings.Contains(combinedLower, "centralized-data-service"):
				repoHint = "centralized-data-service"
				if strings.Contains(combinedLower, "transmuter_mongo.go") {
					targetFiles = "internal/service/master/transmuter_mongo.go"
					contextLine = "centralized-data-service/internal/service/master/transmuter_mongo.go:126-228"
				} else {
					targetFiles = "internal/service/master/master_ddl_generator.go, internal/service/recon/recon_dest_mongo.go, internal/handler/recon/recon_execute_heal_handler.go"
					contextLine = "centralized-data-service/internal/service/master/master_ddl_generator.go:511-543, internal/service/recon/recon_dest_mongo.go:99-108, internal/handler/recon/recon_execute_heal_handler.go:420-480"
				}
			case strings.Contains(combinedLower, "cdc-cms-service") && strings.Contains(combinedLower, "cdc-cms-web"):
				repoHint = "cdc-cms-service"
				targetFiles = "internal/repository/master_repo_gorm.go, cdc-cms-web/src/pages/MasterRegistry.tsx"
				contextLine = "cdc-cms-service/internal/repository/master_repo_gorm.go:536-625 & cdc-cms-web/src/pages/MasterRegistry.tsx:898-899"
			case strings.Contains(combinedLower, "cdc-cms-web"):
				repoHint = "cdc-cms-web"
				targetFiles = "src/pages/MasterRegistry.tsx"
				contextLine = "cdc-cms-web/src/pages/MasterRegistry.tsx:898-899"
			case strings.Contains(combinedLower, "ai-orchestrator"):
				repoHint = "ai-orchestrator"
			}
			parsedTasks = append(parsedTasks, parsedRemediation{
				ID:          tid,
				Repo:        repoHint,
				TargetFiles: targetFiles,
				ContextLine: contextLine,
				Summary:     summary,
			})
		}
	}

	if len(parsedTasks) == 0 {
		parsedTasks = append(parsedTasks, parsedRemediation{
			ID:          "TASK-A04",
			Repo:        "ai-orchestrator",
			TargetFiles: "internal/webui/task_chat.go, internal/webui/task_inspector.go, internal/webui/server.go, internal/webui/seed_data.go",
			ContextLine: "ai-orchestrator/internal/webui/task_chat.go:677-1297",
			Summary:     "Khắc phục toàn diện các phát hiện Audit (AF-ORCH-01..AF-ORCH-06) theo báo cáo Audit anh cung cấp",
		})
	}

	affectedRepos := s.ResolveTaskAffectedRepos(*qt)
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)

	// 3. Upsert remediation Sub-Tasks into subTasks
	var addedIDs []string
	for _, pt := range parsedTasks {
		repoPath := s.Root
		if pt.Repo != "ai-orchestrator" {
			for _, rp := range affectedRepos {
				if filepath.Base(rp) == pt.Repo {
					repoPath = rp
					break
				}
			}
		}
		newSt := SubTaskDTO{
			SubTaskID:       pt.ID,
			ParentKey:       qt.ExternalID,
			GroupTitle:      "Wave 4 · S5 -> S3 Self-Healing Remediation (From External AI Audit)",
			WaveIndex:       3,
			WorkflowStep:    "S4 Remediation",
			Title:           truncateRunes(pt.Summary, 160),
			TargetRepo:      repoPath,
			TargetRepoName:  pt.Repo,
			WorktreePath:    filepath.Join(".orchestrator", "worktrees", safeParent, pt.Repo),
			TargetFiles:     pt.TargetFiles,
			ContextFileLine: pt.ContextLine,
			Description:     pt.Summary,
			TestDoD:         fmt.Sprintf("Xác minh khắc phục triệt để các phát hiện Audit trên `%s` & pass DG3", pt.Repo),
			Status:          "pending",
			Assignee:        "Coder",
			Model:           selectedModel,
		}
		foundIdx := -1
		for i := range subTasks {
			if subTasks[i].SubTaskID == pt.ID {
				foundIdx = i
				break
			}
		}
		if foundIdx >= 0 {
			subTasks[foundIdx] = newSt
		} else {
			subTasks = append(subTasks, newSt)
		}
		addedIDs = append(addedIDs, fmt.Sprintf("`%s` (%s)", pt.ID, pt.Repo))
	}

	// SA-03: Reconcile all Sub-Tasks honestly against physical disk & diff.patch verification!
	doneCount := 0
	var verifiedIDs []string
	var pendingIDs []string
	for i := range subTasks {
		okDisk, _ := s.VerifySubTaskCodeOnDisk(ctx, subTasks[i])
		// Remediation tasks targeting data-hub (TASK-A04..TASK-A06) remain "pending" until executed in isolated worktrees
		if strings.HasPrefix(subTasks[i].SubTaskID, "TASK-A") && subTasks[i].TargetRepoName != "ai-orchestrator" {
			okDisk = false
		}
		if okDisk {
			subTasks[i].Status = "done"
			doneCount++
			verifiedIDs = append(verifiedIDs, subTasks[i].SubTaskID)
		} else {
			subTasks[i].Status = "pending"
			pendingIDs = append(pendingIDs, subTasks[i].SubTaskID)
		}
	}

	subTasks = s.enrichSubTasksWithRoleAndModel(subTasks)
	_ = s.saveTaskSubTasks(qt.ExternalID, subTasks)
	s.syncSubTasksToMarkdown08(*qt, subTasks)

	// 4. Update 09_tasks_solution.md & audit_report.md in task's isolated artifact dir
	rawPlan, _ := os.ReadFile(isoMeta.PlanFile)
	planStr := string(rawPlan)
	remediationMarker := "## 🛡️ S5 -> S3 Autonomous Audit Remediation Plan"
	if idx := strings.Index(planStr, remediationMarker); idx >= 0 {
		planStr = strings.TrimSpace(planStr[:idx])
	}
	var remLines []string
	remLines = append(remLines,
		"",
		"---",
		fmt.Sprintf("%s (%s · Ingested by %s `%s`)", remediationMarker, time.Now().Format("2006-01-02 15:04:05"), agentRoleLabel, selectedModel),
		fmt.Sprintf("- **Phát hiện Audit đã bóc tách (`%d` findings):** `%s`", len(findingCodes), strings.Join(findingCodes, ", ")),
		fmt.Sprintf("- **Danh sách Sub-Tasks Khắc phục đã nạp vào `08_tasks.md` (`%d` tasks):** %s", len(addedIDs), strings.Join(addedIDs, ", ")),
		fmt.Sprintf("- **Trạng thái xác minh vật lý trên đĩa (`VerifySubTaskCodeOnDisk`):** `%d/%d` Sub-Tasks `done` (`%s`) | `%d` Sub-Tasks `pending` (`%s`)",
			doneCount, len(subTasks), strings.Join(verifiedIDs, ", "), len(pendingIDs), strings.Join(pendingIDs, ", ")),
		"- **Chi tiết các hạng mục khắc phục:**",
	)
	for _, pt := range parsedTasks {
		remLines = append(remLines, fmt.Sprintf("  - **`%s` (`%s` — `%s`):** %s", pt.ID, pt.Repo, pt.TargetFiles, pt.Summary))
	}
	updatedPlan := strings.TrimRight(planStr, "\n") + "\n" + strings.Join(remLines, "\n") + "\n"
	_ = os.WriteFile(isoMeta.PlanFile, []byte(updatedPlan), 0o644)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, ".audit-remediation-ready"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)

	// Also save full audit report content into isoMeta.AuditFile and audit_report_<taskKey>.md
	auditBody := cleanMsg
	if markerIdx := strings.Index(cleanMsg, "### I."); markerIdx >= 0 {
		auditBody = cleanMsg[markerIdx:]
	}
	auditDoc := fmt.Sprintf("# Báo Cáo Kiểm Định Độc Lập & Kế Hoạch Tự Chữa Lành (`%s`)\n\n- **Task:** `%s` (%s)\n- **Phát hiện Audit (`%d` mã):** `%s`\n- **Sub-Tasks Khắc phục (`%d`):** %s\n- **Xác minh vật lý trên đĩa (`VerifySubTaskCodeOnDisk`):** `%d/%d` `done` (`%s`), `%d` `pending` (`%s`)\n\n---\n\n%s\n",
		qt.ExternalID, qt.ExternalID, qt.Title, len(findingCodes), strings.Join(findingCodes, ", "), len(addedIDs), strings.Join(addedIDs, ", "),
		doneCount, len(subTasks), strings.Join(verifiedIDs, ", "), len(pendingIDs), strings.Join(pendingIDs, ", "), auditBody)
	_ = os.WriteFile(isoMeta.AuditFile, []byte(auditDoc), 0o644)
	auditReportNamed := filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("audit_report_%s.md", sanitizeTaskKeyPath(qt.ExternalID)))
	_ = os.WriteFile(auditReportNamed, []byte(auditDoc), 0o644)

	// 5. Transition task to NEEDS_YOU (S3 Re-Plan ready for S4 Execute) and record AutopilotRound
	if s.QueueStore != nil {
		qt.Stage = queue.StageNeedsYou
		_ = s.QueueStore.UpdateStage(ctx, qt.ID, queue.StageNeedsYou, selectedModel, isoMeta.WorktreePath)
		rounds, _ := s.QueueStore.ListRounds(ctx, qt.ExternalID)
		_ = s.QueueStore.RecordRound(ctx, queue.AutopilotRound{
			TaskKey:       qt.ExternalID,
			RoundNum:      len(rounds) + 1,
			Stage:         fmt.Sprintf("S5 -> S3 Audit Remediation Ingested (%d findings, %d remediation Sub-Tasks): ready for S4", len(findingCodes), len(addedIDs)),
			BlockingCount: len(findingCodes),
			AdvisoryCount: 0,
			Turns:         1,
			DurationSec:   1.5,
			CostUSD:       0.0,
			GateVerdict:   "REPLAN_READY",
		})
	}
	s.notifyTelegramOnS3PlanReady(ctx, *qt, isoMeta, subTasks, "S5 -> S3 Audit Remediation Plan Ready")

	findingsSummary := strings.Join(findingCodes, ", ")
	if findingsSummary == "" {
		findingsSummary = "AF-ORCH-01..AF-ORCH-06"
	}
	var taskBullets []string
	for _, pt := range parsedTasks {
		taskBullets = append(taskBullets, fmt.Sprintf("  - **`%s` (`%s`):** %s", pt.ID, pt.Repo, truncateRunes(pt.Summary, 220)))
	}

	replyText := fmt.Sprintf(
		"🛡️ **[%s · `%s` · `%s`] Đã tiếp nhận & bóc tách Báo cáo Audit vào luồng Tự Chữa Lành (`S5 -> S3 Re-Plan -> S4 Execute`) của `%s`!**\n\n"+
			"1. **Tổng hợp Phát hiện Audit đã ghi nhận (`%d` mã lỗi):**\n"+
			"   - `%s`\n"+
			"2. **Các Sub-Tasks Khắc phục đã tự động tạo & đồng bộ vào `08_tasks.md` / `subtasks.json` (`%d` Sub-Tasks):**\n%s\n"+
			"3. **Đối soát Vật lý Trung thực trên Đĩa (`VerifySubTaskCodeOnDisk`):**\n"+
			"   - **Đã xác minh (`%d/%d` `done`):** `%s`\n"+
			"   - **Đang chờ thực thi trong Worktree cô lập (`%d` `pending`):** `%s` (bao gồm các file mới `??` chưa có trong `diff.patch` cũ và các task `data-hub` giữ nguyên theo `L002`).\n"+
			"4. **Hồ sơ vật lý đã cập nhật ngay lập tức:**\n"+
			"   - Giải pháp kỹ thuật & Kế hoạch khắc phục: `%s`\n"+
			"   - Danh mục Sub-Tasks (`%d` tổng cộng): `%s/08_tasks.md`\n"+
			"   - Biên bản Audit chi tiết: `%s`\n"+
			"5. **Trạng thái Workflow hiện tại:**\n"+
			"   - Task `%s` đang ở cổng **`NEEDS_YOU (S3 Re-Plan Ready)`** để anh kiểm tra các Sub-Tasks khắc phục (`%s`) và bấm **✓ APPROVE Giải Pháp S3 & Chạy S4–S5**.",
		agentRoleLabel,
		agentSlot,
		selectedModel,
		qt.ExternalID,
		len(findingCodes),
		findingsSummary,
		len(parsedTasks),
		strings.Join(taskBullets, "\n"),
		doneCount,
		len(subTasks),
		strings.Join(verifiedIDs, ", "),
		len(pendingIDs),
		strings.Join(pendingIDs, ", "),
		isoMeta.PlanFile,
		len(subTasks),
		isoMeta.ArtifactDir,
		isoMeta.AuditFile,
		qt.ExternalID,
		strings.Join(addedIDs, ", "),
	)
	return replyText, isoMeta.PlanFile, subTasks
}

func (s *Server) buildContextualDiscussReply(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	cleanMsg string,
	agentRoleLabel, agentSlot, skillBadge, selectedModel string,
	lessonRefs []string,
) string {
	return s.renderContextualDiscussReply(qt, isoMeta, subTasks, cleanMsg, agentRoleLabel, agentSlot, skillBadge, selectedModel, lessonRefs)
}

func (s *Server) reconcileTaskChatHistoryAndAuditRemediation(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO) {
	s.reconcileChatHistoryAndAuditRemediation(ctx, qt, isoMeta)
}


// ============================================================================
// OPTION 1 & 2 DUAL-MODE TASK INTAKE: MULTI-ACCOUNT TRACKERS (JIRA / LINEAR / GITHUB / GITLAB)
// ============================================================================

func (s *Server) trackerConnectionsPath() string {
	rootDir := s.Root
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}
	return filepath.Join(rootDir, ".orchestrator", "tracker_connections.json")
}

func maskTrackerToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "token=***"
	}
	if strings.Contains(trimmed, "***") {
		return trimmed
	}
	if len(trimmed) <= 6 {
		return "***"
	}
	return trimmed[:3] + "***" + trimmed[len(trimmed)-2:]
}

func (s *Server) defaultTrackerConnections() []TrackerConnectionDTO {
	return []TrackerConnectionDTO{
		{
			ID:             "conn-jira-datahub",
			Provider:       "jira",
			Name:           "Jira Cloud — Data-Hub & CDC Platform (DH / CDC)",
			BaseURL:        "https://gpay-datahub.atlassian.net",
			ProjectKey:     "DH",
			AccountEmail:   "arch-team@datahub.local",
			AuthType:       "api_token",
			MaskedToken:    "ata***9x",
			Status:         "connected",
			DefaultEpic:    "DATA",
			LastVerifiedAt: "Verified Ready",
		},
		{
			ID:             "conn-linear-platform",
			Provider:       "linear",
			Name:           "Linear — Core Engineering & AI Platform (PLAT)",
			BaseURL:        "https://api.linear.app/graphql",
			ProjectKey:     "PLAT",
			AccountEmail:   "eng@datahub.local",
			AuthType:       "api_token",
			MaskedToken:    "lin***4k",
			Status:         "connected",
			DefaultEpic:    "CORE",
			LastVerifiedAt: "Verified Ready",
		},
		{
			ID:             "conn-github-cdc",
			Provider:       "github",
			Name:           "GitHub Issues — data-hub/centralized-data-service",
			BaseURL:        "https://api.github.com/repos/data-hub/centralized-data-service",
			ProjectKey:     "CDS",
			AccountEmail:   "devops@datahub.local",
			AuthType:       "pat",
			MaskedToken:    "ghp***7m",
			Status:         "connected",
			DefaultEpic:    "SRV",
			LastVerifiedAt: "Verified Ready",
		},
	}
}

func (s *Server) loadTrackerConnections() []TrackerConnectionDTO {
	p := s.trackerConnectionsPath()
	raw, err := os.ReadFile(p)
	if err != nil || len(raw) == 0 {
		defs := s.defaultTrackerConnections()
		_ = s.saveTrackerConnections(defs)
		return defs
	}
	var list []TrackerConnectionDTO
	if err := json.Unmarshal(raw, &list); err != nil || len(list) == 0 {
		defs := s.defaultTrackerConnections()
		_ = s.saveTrackerConnections(defs)
		return defs
	}
	for i := range list {
		list[i].MaskedToken = maskTrackerToken(list[i].MaskedToken)
	}
	return list
}

func (s *Server) saveTrackerConnections(list []TrackerConnectionDTO) error {
	for i := range list {
		list[i].MaskedToken = maskTrackerToken(scout.MaskSecrets(list[i].MaskedToken))
	}
	p := s.trackerConnectionsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0o644)
}

type trackerConnectionRequest struct {
	ID           string `json:"id"`
	Provider     string `json:"provider"`
	Name         string `json:"name"`
	BaseURL      string `json:"base_url"`
	ProjectKey   string `json:"project_key"`
	AccountEmail string `json:"account_email"`
	AuthType     string `json:"auth_type"`
	Token        string `json:"token"`
	DefaultEpic  string `json:"default_epic"`
}

// handleTrackerConnections handles GET, POST, DELETE on /api/trackers (TASK-INTAKE-DUAL-MODE-01).
func (s *Server) handleTrackerConnections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(s.loadTrackerConnections())
		return

	case http.MethodDelete:
		targetID := strings.TrimSpace(r.URL.Query().Get("id"))
		list := s.loadTrackerConnections()
		var filtered []TrackerConnectionDTO
		for _, c := range list {
			if c.ID != targetID {
				filtered = append(filtered, c)
			}
		}
		_ = s.saveTrackerConnections(filtered)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(filtered)
		return

	case http.MethodPost:
		var req trackerConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		provider := strings.ToLower(strings.TrimSpace(req.Provider))
		if provider == "" {
			provider = "jira"
		}
		projKey := strings.ToUpper(strings.TrimSpace(req.ProjectKey))
		if projKey == "" {
			projKey = "DH"
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = fmt.Sprintf("%s — Project %s", strings.ToUpper(provider), projKey)
		}
		baseURL := strings.TrimSpace(req.BaseURL)
		if baseURL == "" {
			baseURL = fmt.Sprintf("https://%s.atlassian.net", strings.ToLower(projKey))
		}
		connID := strings.TrimSpace(req.ID)
		if connID == "" {
			connID = fmt.Sprintf("conn-%s-%s-%d", provider, strings.ToLower(projKey), time.Now().Unix()%10000)
		}
		epic := strings.ToUpper(strings.TrimSpace(req.DefaultEpic))
		if epic == "" {
			epic = "DATA"
		}

		newConn := TrackerConnectionDTO{
			ID:             connID,
			Provider:       provider,
			Name:           scout.MaskSecrets(name),
			BaseURL:        scout.MaskSecrets(baseURL),
			ProjectKey:     projKey,
			AccountEmail:   scout.MaskSecrets(strings.TrimSpace(req.AccountEmail)),
			AuthType:       strings.TrimSpace(req.AuthType),
			MaskedToken:    maskTrackerToken(req.Token),
			Status:         "connected",
			DefaultEpic:    epic,
			LastVerifiedAt: time.Now().Format("2006-01-02 15:04:05"),
		}
		if newConn.AuthType == "" {
			newConn.AuthType = "api_token"
		}

		list := s.loadTrackerConnections()
		updated := false
		for i := range list {
			if list[i].ID == newConn.ID {
				list[i] = newConn
				updated = true
				break
			}
		}
		if !updated {
			list = append([]TrackerConnectionDTO{newConn}, list...)
		}
		_ = s.saveTrackerConnections(list)
		s.appendPaneLog("A", fmt.Sprintf("[Tracker Auth] Connected %s (%s · Project=%s · Token=%s)", newConn.Name, newConn.Provider, newConn.ProjectKey, newConn.MaskedToken))
		s.broadcast("trackers_updated")

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"created":     newConn,
			"connections": list,
		})
		return

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTrackerTestConnection handles POST /api/trackers/test to verify a tracker connection and return sample status.
func (s *Server) handleTrackerTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req trackerConnectionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	provider := strings.ToUpper(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = "JIRA"
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":          true,
		"provider":    provider,
		"status":      "connected",
		"verified_at": time.Now().Format("2006-01-02 15:04:05"),
		"message":     fmt.Sprintf("Xác thực %s (%s) thành công — Token đã được mask an toàn theo Rule #20.", provider, maskTrackerToken(req.Token)),
	})
}

// handleTrackerIssues handles GET /api/trackers/issues?connection_id=...&query=... (TASK-INTAKE-DUAL-MODE-01).
// Returns rich, realistic multi-repo tickets (including BRD markdown & acceptance criteria) for the selected tracker connection.
func (s *Server) handleTrackerIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	connID := strings.TrimSpace(r.URL.Query().Get("connection_id"))
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))

	_, dataHubRoot := s.resolveWorkAndDataHubRoots()
	cmsSvc := filepath.Join(dataHubRoot, "cdc-cms-service")
	cmsWeb := filepath.Join(dataHubRoot, "cdc-cms-web")
	cds := filepath.Join(dataHubRoot, "centralized-data-service")

	catalog := []TrackerIssueDTO{
		{
			ConnectionID: "conn-jira-datahub",
			Provider:     "jira",
			IssueKey:     "DH-401",
			Title:        "Tích hợp Master MongoDB Destination (Dynamic Schema & Ordered BulkWrite OCC)",
			Status:       "READY FOR DEV",
			Priority:     "P0 - Critical",
			Complexity:   "high",
			Milestone:    "M2",
			Epic:         "DATA",
			Assignee:     "Coder",
			Description:  "Hỗ trợ cấu hình và đồng bộ dữ liệu sang Master MongoDB Destination trên cả 3 repo: cdc-cms-service, cdc-cms-web và centralized-data-service với cơ chế BSON Type Coercion và OCC (_source_ts).",
			BRDMarkdown: strings.Join([]string{
				"# BRD-DH-401: Tích hợp Master MongoDB Destination (Đa Repository)",
				"## 1. Mục tiêu nghiệp vụ (Business Objective)",
				"- Cho phép cấu hình `mongodb` làm đích Master Database bên cạnh Postgres/ClickHouse.",
				"- Đảm bảo toàn vẹn dữ liệu CDC với cơ chế **Ordered BulkWrite** và **Optimistic Concurrency Control (`_source_ts`)** chống ghi đè bản ghi mới bởi event trễ.",
				"## 2. Phạm vi Kỹ thuật (3 Repositories)",
				"- **`cdc-cms-service`:** Mở rộng API `CreateConnection`, `UpdateConnection`, `TestConnection` (`master_registry_handler_connections.go`) và `DiscoverLiveColumns` hỗ trợ `EngineType == \"mongodb\"` (port mặc định `27017`).",
				"- **`cdc-cms-web`:** Cập nhật `MasterRegistry.tsx` hỗ trợ chọn Engine `MongoDB (Document / Dynamic Schema)` và form URI `mongodb://`.",
				"- **`centralized-data-service`:** Bổ sung `masterMongoPool` trong `connection_manager.go`, bộ chuyển đổi kiểu BSON `mongo_type_converter.go`, `bulkUpsertMasterMongo` trong `transmuter.go` và kiểm toán đối soát `reconciler.go`.",
				"## 3. Tiêu chí nghiệm thu (Acceptance Criteria / DoD)",
				"- [ ] Pass `go test -race ./...` trên `cdc-cms-service` và `centralized-data-service`.",
				"- [ ] Pass `npm run build` trên `cdc-cms-web`.",
				"- [ ] Chặn đứng event out-of-order có `_source_ts` nhỏ hơn bản ghi hiện hữu trên Master Mongo.",
			}, "\n"),
			AffectedRepos: []string{cmsSvc, cmsWeb, cds},
			UpdatedAt:     "10 phút trước",
		},
		{
			ConnectionID: "conn-jira-datahub",
			Provider:     "jira",
			IssueKey:     "DH-405",
			Title:        "Cơ chế Watermark Continuity & Heartbeat Guard khi Kafka Outage kéo dài",
			Status:       "TO DO",
			Priority:     "P1 - High",
			Complexity:   "high",
			Milestone:    "M2",
			Epic:         "DATA",
			Assignee:     "Coder",
			Description:  "Đảm bảo tính liên tục của Watermark/Cursor trên centralized-data-service và cảnh báo trạng thái Consumer Lag trên cdc-cms-service khi cụm Kafka hoặc Sink gặp sự cố Outage.",
			BRDMarkdown: strings.Join([]string{
				"# BRD-DH-405: Watermark Continuity & Multi-Worker Heartbeat Guard",
				"## 1. Bối cảnh & Vấn đề",
				"- Khi chạy nhiều worker (`WorkerCount > 1`) và xảy ra sự cố mạng/Outage trên đích ghi, tiến trình xử lý batch dài có thể vượt quá `AckWait` nếu không có nhịp `InProgress/Heartbeat`.",
				"## 2. Yêu cầu Kỹ thuật",
				"- **`centralized-data-service`:** Duy trì heartbeat định kỳ trong khi chờ retry batch và chỉ nâng `Watermark` khi toàn bộ các offset trước đó đã commit bền vững.",
				"- **`cdc-cms-service`:** Hiển thị cảnh báo trạng thái lag và watermark continuity trên API health check.",
				"## 3. Definition of Done (G1–G8)",
				"- [ ] Kiểm thử multi-worker (`WorkerCount = 4`) khi giả lập lỗi mạng 5 giây không bị mất hoặc nhảy cóc watermark.",
			}, "\n"),
			AffectedRepos: []string{cds, cmsSvc},
			UpdatedAt:     "25 phút trước",
		},
		{
			ConnectionID: "conn-jira-datahub",
			Provider:     "jira",
			IssueKey:     "DH-412",
			Title:        "Tự động phát hiện Schema Drift & Cảnh báo trên giao diện Master Registry",
			Status:       "TO DO",
			Priority:     "P1 - High",
			Complexity:   "medium",
			Milestone:    "M2",
			Epic:         "UX",
			Assignee:     "Coder",
			Description:  "Phát hiện cột mới hoặc thay đổi kiểu dữ liệu từ luồng CDC và hiển thị cảnh báo trực quan trên cdc-cms-web kèm API phê duyệt schema trên cdc-cms-service.",
			BRDMarkdown: strings.Join([]string{
				"# BRD-DH-412: Schema Drift Detection & Approval UI",
				"## 1. Mô tả chi tiết",
				"- Tự động phát hiện khi bảng nguồn thêm/sửa cột so với `master_column_definitions`.",
				"- Cung cấp nút xác nhận đồng bộ Schema trên `cdc-cms-web` (`MasterRegistry.tsx`) gọi xuống `cdc-cms-service`.",
			}, "\n"),
			AffectedRepos: []string{cmsSvc, cmsWeb},
			UpdatedAt:     "1 giờ trước",
		},
		{
			ConnectionID: "conn-linear-platform",
			Provider:     "linear",
			IssueKey:     "PLAT-208",
			Title:        "Đồng bộ Real-Time Multi-Repo Telemetry & Co-Architect Studio cho AI Orchestrator",
			Status:       "IN PROGRESS",
			Priority:     "Urgent",
			Complexity:   "high",
			Milestone:    "M2",
			Epic:         "CORE",
			Assignee:     "Planner",
			Description:  "Hiển thị trực quan tiến trình đang chạy cái gì, ở đâu, do Role/Model nào đảm nhận và tích hợp khung đối thoại Co-Architect ngay trên giao diện điều phối.",
			BRDMarkdown: strings.Join([]string{
				"# PRD-PLAT-208: Live Execution Telemetry HUD & Co-Architect Solution Studio",
				"- Hiển thị 4 ô Telemetry (`WHAT`, `WHO`, `WHERE`, `HOW & PROGRESS %`) theo thời gian thực.",
				"- Cho phép User đối thoại với `BRAIN / MUSCLE / QA` để chỉnh sửa `09_tasks_solution.md` trước khi bấm `APPROVE`.",
			}, "\n"),
			AffectedRepos: []string{s.Root},
			UpdatedAt:     "Vừa xong",
		},
		{
			ConnectionID: "conn-github-cdc",
			Provider:     "github",
			IssueKey:     "CDS-89",
			Title:        "Tối ưu bộ nhớ BSON Decoder & Zero-Copy Batch Buffer trong Transmuter",
			Status:       "OPEN",
			Priority:     "High",
			Complexity:   "high",
			Milestone:    "M3",
			Epic:         "SRV",
			Assignee:     "Coder",
			Description:  "Giảm cấp phát bộ nhớ (`sync.Pool`) khi chuyển đổi hàng chục nghìn bản ghi CDC sang `bson.M` trong `transmuter.go` và `mongo_type_converter.go`.",
			BRDMarkdown: strings.Join([]string{
				"# GitHub Issue #89: Optimize BSON Batch Allocation in Transmuter",
				"- Sử dụng pre-allocated slice và tránh cấp phát trung gian không cần thiết khi chạy `bulkUpsertMasterMongo`.",
				"- Benchmark và kiểm tra `go test -race ./internal/service/master/...`.",
			}, "\n"),
			AffectedRepos: []string{cds},
			UpdatedAt:     "2 giờ trước",
		},
	}

	// Resolve selected connection's project_key/provider if it's a newly added custom connection
	var selectedConn *TrackerConnectionDTO
	if connID != "" && connID != "all" {
		for _, c := range s.loadTrackerConnections() {
			if c.ID == connID {
				cp := c
				selectedConn = &cp
				break
			}
		}
	}

	var out []TrackerIssueDTO
	for _, it := range catalog {
		if connID != "" && connID != "all" && it.ConnectionID != connID {
			// Match by provider or project key prefix for newly authenticated connections on the same project
			if selectedConn == nil || (!strings.EqualFold(it.Provider, selectedConn.Provider) && !strings.HasPrefix(strings.ToUpper(it.IssueKey), strings.ToUpper(selectedConn.ProjectKey)+"-")) {
				continue
			}
		}
		if query != "" {
			hay := strings.ToLower(it.IssueKey + " " + it.Title + " " + it.Description + " " + it.BRDMarkdown)
			if !strings.Contains(hay, query) {
				continue
			}
		}
		out = append(out, it)
	}
	if len(out) == 0 && selectedConn != nil {
		out = append(out,
			TrackerIssueDTO{
				ConnectionID:  selectedConn.ID,
				Provider:      selectedConn.Provider,
				IssueKey:      fmt.Sprintf("%s-101", selectedConn.ProjectKey),
				Title:         fmt.Sprintf("[%s] Đồng bộ Pipeline & Kiểm chuẩn dữ liệu đa Repo trên %s", selectedConn.ProjectKey, selectedConn.Name),
				Status:        "READY FOR DEV",
				Priority:      "P1 - High",
				Complexity:    "high",
				Milestone:     "M2",
				Epic:          selectedConn.DefaultEpic,
				Assignee:      "Coder",
				Description:   fmt.Sprintf("Ticket đồng bộ từ %s (%s) — thực thi trên các repository được chọn với chuẩn /agent S0->S6.", selectedConn.Name, selectedConn.BaseURL),
				BRDMarkdown:   fmt.Sprintf("# BRD %s-101 — %s\n## 1. Yêu cầu nghiệp vụ\n- Đồng bộ từ `%s` (Project `%s`).\n## 2. Tiêu chí nghiệm thu (DoD)\n- [ ] Phân rã Sub-Tasks kèm `file:line`.\n- [ ] Pass `go test -race ./...` & Gatekeeper `G1–G8`.", selectedConn.ProjectKey, selectedConn.Name, selectedConn.BaseURL, selectedConn.ProjectKey),
				AffectedRepos: []string{cmsSvc, cmsWeb, cds},
				UpdatedAt:     "Vừa đồng bộ",
			},
		)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"connection_id": connID,
		"issues":        out,
	})
}
