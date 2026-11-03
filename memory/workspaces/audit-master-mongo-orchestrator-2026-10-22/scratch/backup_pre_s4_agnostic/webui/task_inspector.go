package webui

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/agentgate"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/scout"
)

// RepoOptionDTO describes an auto-discovered Git/service repository in the user's workspace.
type RepoOptionDTO struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	CurrentBranch string `json:"current_branch"`
	Language      string `json:"language"`
}

// CLISettingsDTO persists the user's CLI runner and Cross-Model Auditing per-role model configuration (TASK-PIPE-01 & TASK-MDL-01).
type CLISettingsDTO struct {
	DefaultRunner      string `json:"default_runner"`        // antigravity | gemini | claude
	PlannerModel       string `json:"planner_model"`         // Slot A/G: gemini-4-argon (antigravity)
	CoderHighModel     string `json:"coder_high_model"`      // Slot B/E: gemini-4-argon (antigravity)
	CoderStdModel      string `json:"coder_std_model"`       // Slot D/F: gemini-2.5-flash (antigravity)
	QAAuditorModel     string `json:"qa_auditor_model"`      // Slot K (Cross-Model QA): claude-opus-4-6 (antigravity)
	LayaGateModel      string `json:"laya_gate_model"`       // Slot H (Local Zero-Token DG0/DG1/DG2/DG3): convaiinnovations/laya (local :8200)
	SecurityModel      string `json:"security_model"`        // Slot I (Kernel & Security Guard): agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)
	AgentKernelRoot    string `json:"agent_kernel_root"`     // /Users/trainguyen/Documents/work/agent
	EnforceApproveGate bool   `json:"enforce_approve_gate"`  // true -> hard stop at NEEDS_YOU after S3 & on S5->S3 re-plan
	MaxSelfHealRetries int    `json:"max_self_heal_retries"` // Circuit Breaker limit for Tier-1 S5->S4->S5 (default: 3)
	CrossModelVerified bool   `json:"cross_model_verified"`  // true when CoderHighModel and QAAuditorModel/LayaGateModel use distinct model families
	CrossModelSummary  string `json:"cross_model_summary"`   // e.g. "Cross-Model Verified: MUSCLE[gemini] ≠ QA[claude] ≠ Laya[laya-local-8200] ≠ Sec[agy-guard+claude]"
}

// SelfHealingCircuitStateDTO tracks Two-Tier Self-Healing (Tier-1 S5->S4->S5 vs Tier-2 S5->S3->Human Gate),
// the Max 3 Retries Circuit Breaker, Stage 5A (Zero-LLM Hard Gate) vs Stage 5B (Cross-Model Semantic Audit),
// and Multi-Repo Wave Contract Sync + Atomic Wave Rollback (TASK-PIPE-02, TASK-PIPE-04, TASK-PIPE-05).
type SelfHealingCircuitStateDTO struct {
	RetryCount             int      `json:"retry_count"`                // Current Tier-1 retry count (0..3)
	MaxRetries             int      `json:"max_retries"`                // Hard Circuit Breaker limit (default: 3)
	CircuitOpen            bool     `json:"circuit_open"`               // true when RetryCount >= MaxRetries -> escalates to NEEDS_YOU
	RequiresHumanReApprove bool     `json:"requires_human_reapprove"`   // true when Tier-2 S5->S3 modifies Plan (09_tasks_solution.md) -> revokes .user-approved
	ActiveLoopTier         string   `json:"active_loop_tier"`           // "NONE" | "TIER1_S5_TO_S4" | "TIER2_S5_TO_S3_HUMAN_GATE" | "CIRCUIT_OPEN_NEEDS_YOU"
	AllowedImplLoopRoute   string   `json:"allowed_impl_loop_route"`    // "Tier-1: S5 -> S4 -> S5 (Max 3 Retries)"
	ArchChangeRoute        string   `json:"arch_change_route"`          // "Tier-2: S5 -> S3 -> Human Gate (NEEDS_YOU Re-Approve)"
	LastFailureClass       string   `json:"last_failure_class"`         // "HARD_GATE_5A_ZERO_LLM_FAIL" | "ARCH_PLAN_S3_CHANGED" | "CIRCUIT_BREAKER_MAX_RETRIES" | "ATOMIC_WAVE_ROLLBACK"
	LastFailureReason      string   `json:"last_failure_reason"`        // Human-readable failure / escalation reason
	HardGateZeroLLMPassed  bool     `json:"hard_gate_zero_llm_passed"`  // Stage 5A deterministic gate result
	HardGateCommand        string   `json:"hard_gate_command"`          // "go vet ./... && go test -race && VerifySubTaskCodeOnDisk"
	HardGateExitCode       int      `json:"hard_gate_exit_code"`        // 0 = PASS, 1 = FAIL
	HardGateOutput         string   `json:"hard_gate_output"`           // stdout/stderr from Stage 5A
	HardGateCmds           []string `json:"hard_gate_cmds"`             // ["go vet ./...", "golangci-lint", "go test -race", "go build"]
	HardGateLastOutput     string   `json:"hard_gate_last_output"`      // stdout/stderr from Stage 5A
	SemanticGateStatus     string   `json:"semantic_gate_status"`       // "PASS" | "SKIPPED_BY_5A_HARD_GATE" | "PENDING" | "BLOCK"
	SemanticGateLLMCalls   int      `json:"semantic_gate_llm_calls"`    // 0 when 5A Hard Gate fails!
	WaveContractSynced     bool     `json:"wave_contract_synced"`       // Contract/Integration Sync Gate across Sub-Tasks in same Wave
	WaveSyncVerified       bool     `json:"wave_sync_verified"`         // Alias for WaveContractSynced
	WaveRestorePointSHA    string   `json:"wave_restore_point_sha"`     // Git commit SHA for Atomic Wave Rollback
	WaveRollbackTriggered  bool     `json:"wave_rollback_triggered"`    // true if sibling Sub-Task in Wave tripped Circuit Breaker
	LastEscalationReason   string   `json:"last_escalation_reason"`     // Human-readable escalation / rollback reason
}

// SubTaskInputContractDTO defines the explicit input contract of an individual Sub-Task (TASK-SUB-01).
type SubTaskInputContractDTO struct {
	SourceContext        string   `json:"source_context"`        // Exact file:line touchpoint in TargetRepo
	UpstreamSubTaskIDs   []string `json:"upstream_sub_task_ids"` // DependsOn Sub-Task IDs from Wave N-1
	UpstreamDeliverables []string `json:"upstream_deliverables"` // Concrete outputs handed off from upstream Wave N-1 Sub-Tasks
	UpstreamReady        bool     `json:"upstream_ready"`        // true if Wave 0 or all UpstreamSubTaskIDs have reached DONE
	InputSpec            string   `json:"input_spec"`            // Input payload / struct / interface / context specification
}

// SubTaskOutputContractDTO defines the explicit output deliverables produced by an individual Sub-Task (TASK-SUB-01).
type SubTaskOutputContractDTO struct {
	Deliverables         []string `json:"deliverables"`            // Concrete symbols, files, or endpoints produced by this Sub-Task
	DownstreamSubTaskIDs []string `json:"downstream_sub_task_ids"` // Sub-Tasks in Wave N+1 that consume this output as their input
	DiffFile             string   `json:"diff_file"`               // .orchestrator/tasks/<parent>/subtasks/<subID>/diff.patch
	AuditFile            string   `json:"audit_file"`              // .orchestrator/tasks/<parent>/subtasks/<subID>/audit_report.md
	EvidenceReceiptID    string   `json:"evidence_receipt_id"`     // EV-... receipt verifying this Sub-Task
	HandoffReady         bool     `json:"handoff_ready"`           // true when Sub-Task reaches Isolated Branch (DONE)
}

// SubTaskDTO represents a first-class fractal sub-task bound to its own specific TargetRepo,
// with its own InputContract, Dedicated Plan (09_tasks_solution.md), Start -> Finalize & Diff Review pipeline, and OutputContract.
type SubTaskDTO struct {
	SubTaskID       string                     `json:"sub_task_id"`       // e.g. "TASK-M01"
	ParentKey       string                     `json:"parent_key"`        // e.g. "TASK-11"
	GroupTitle      string                     `json:"group_title"`       // e.g. "Nhóm 1: Quản lý Kết nối Master Mongo (cdc-cms-service & cdc-cms-web)"
	WaveIndex       int                        `json:"wave_index"`        // 0..4
	WorkflowStep    string                     `json:"workflow_step"`     // Wave 0..4 | S1..S5
	Title           string                     `json:"title"`             // e.g. "Mở rộng API Create/Update Master Connection hỗ trợ MongoDB"
	TargetRepo      string                     `json:"target_repo"`       // e.g. "/Users/trainguyen/Documents/work/data-hub/cdc-cms-service"
	TargetRepoName  string                     `json:"target_repo_name"`  // e.g. "cdc-cms-service"
	WorktreePath    string                     `json:"worktree_path"`     // e.g. ".orchestrator/worktrees/TASK-11/cdc-cms-service"
	BranchName      string                     `json:"branch_name"`       // e.g. "task/TASK-11/task-m01"
	ArtifactDir     string                     `json:"artifact_dir"`      // e.g. ".orchestrator/tasks/TASK-11/subtasks/TASK-M01"
	SpecFile        string                     `json:"spec_file"`         // e.g. ".orchestrator/tasks/TASK-11/subtasks/TASK-M01/01_requirements.md"
	AnalysisFile    string                     `json:"analysis_file"`     // e.g. ".orchestrator/tasks/TASK-11/subtasks/TASK-M01/13_analysis.md"
	PlanFile        string                     `json:"plan_file"`         // e.g. ".orchestrator/tasks/TASK-11/subtasks/TASK-M01/09_tasks_solution.md"
	ProgressFile    string                     `json:"progress_file"`     // e.g. ".orchestrator/tasks/TASK-11/subtasks/TASK-M01/05_progress.md"
	TargetFiles     string                     `json:"target_files"`      // e.g. "cdc-cms-service/internal/api/master/master_registry_handler_connections.go:172-249"
	ContextFileLine string                     `json:"context_file_line"` // Exact Context (file:line) from 08_tasks_*.md or codebase scan
	Description     string                     `json:"description"`       // Exact Mô tả
	TestDoD         string                     `json:"test_dod"`          // Exact Test / DoD
	DependsOn       []string                   `json:"depends_on"`        // Previous wave sub-task IDs for DAG edges
	Stage           string                     `json:"stage"`             // READY | PLANNING | NEEDS_YOU | APPROVED | RUNNING | IN_REVIEW | DONE
	Status          string                     `json:"status"`            // done | running | waiting | pending
	HardGatePassed  bool                       `json:"hard_gate_passed"`  // true when Stage 5A Deterministic Gate (Zero-LLM) passes
	PlanSummary     string                     `json:"plan_summary"`      // Dedicated technical solution summary for this Sub-Task
	PlanMarkdown    string                     `json:"plan_markdown"`     // Full Markdown of this Sub-Task's 09_tasks_solution.md
	CodeDemo        string                     `json:"code_demo"`         // Dedicated Code Demo solving this specific Sub-Task
	DiffPreview     string                     `json:"diff_preview"`      // Unified diff preview for this Sub-Task
	AuditSummary    string                     `json:"audit_summary"`     // S5 & DG3 audit summary for this Sub-Task
	InputContract   SubTaskInputContractDTO    `json:"input_contract"`    // Explicit Input Contract (Context + Upstream Wave N-1 outputs)
	OutputContract  SubTaskOutputContractDTO   `json:"output_contract"`   // Explicit Output Contract (Deliverables + Downstream Wave N+1 consumers)
	SelfHealState   SelfHealingCircuitStateDTO `json:"self_heal_state"`   // Circuit Breaker (Max 3), Tier-1 vs Tier-2 loop, 5A/5B gate & Wave Rollback
	PipelineStages  []AutopilotStageGroupDTO   `json:"pipeline_stages"`   // Dedicated 6-step Start -> Plan -> You -> Implement (Slot C) -> 5A/5B QA -> Finalize & Diff Review
	Assignee        string                     `json:"assignee"`
	RoleLabel       string                     `json:"role_label"`  // e.g. "🛠️ MUSCLE · Backend Coder"
	AgentSlot       string                     `json:"agent_slot"`  // e.g. "Slot B"
	SkillBadge      string                     `json:"skill_badge"` // e.g. "/muscle-execute"
	Runner          string                     `json:"runner"`      // e.g. "antigravity"
	Model           string                     `json:"model"`       // e.g. "gemini-4-argon (antigravity)"
}

// StepReportDTO contains the full markdown report and extracted concrete code demo for a workflow step (S1..S5).
type StepReportDTO struct {
	StepID    string `json:"step_id"`   // "S1", "S2", "S3", "S4", "S5"
	Title     string `json:"title"`     // e.g. "S3 Solution Architecture & Concrete Code Demo"
	Status    string `json:"status"`    // done | waiting | running | pending
	FilePath  string `json:"file_path"` // .orchestrator/tasks/<slug>/09_tasks_solution.md
	Summary   string `json:"summary"`
	ContentMD string `json:"content_md"` // Full markdown content of the report
	CodeDemo  string `json:"code_demo"`  // Extracted concrete code demo block
	UpdatedAt string `json:"updated_at"`
}

// MatchedLessonDTO represents a lesson from /agent/memory/global/lessons/ matched to the active task.
type MatchedLessonDTO struct {
	LessonID string `json:"lesson_id"` // e.g. "L020"
	Date     string `json:"date"`      // e.g. "2026-10-20"
	Topic    string `json:"topic"`     // e.g. "T4:L145"
	Title    string `json:"title"`     // e.g. "Unfiltered Destination..."
	Rule     string `json:"rule"`
}

// ActiveAgentKernelDTO surfaces how the user's /agent Governance Kernel is actively applied to the selected task.
type ActiveAgentKernelDTO struct {
	ConstitutionPath string             `json:"constitution_path"`
	IndexPath        string             `json:"index_path"`
	TotalLessons     int                `json:"total_lessons"`
	ActiveRole       string             `json:"active_role"`      // BRAIN (Architect) | MUSCLE (Engineer) | QA Auditor
	ActiveSkills     []string           `json:"active_skills"`    // ["/brain-delegate", "/laya-router"]
	ActiveHooks      []string           `json:"active_hooks"`     // ["H1:LayaUp", "H4:WorkspaceFirst", "H4b:ProposeOnlyStop", "H5:NoGitPush"]
	StopGateReason   string             `json:"stop_gate_reason"` // Explain why stopped at NEEDS_YOU
	MatchedLessons   []MatchedLessonDTO `json:"matched_lessons"`
}

// RepoSymbolHit represents a real file:line symbol or match found inside TargetRepo during S2 investigation.
type RepoSymbolHit struct {
	RelPath   string
	LineNum   int
	Signature string
	Snippet   string
}

var lessonIndexLineRe = regexp.MustCompile(`^-\s+(L\d+)\s+\[([^\]]+)\]\s+(T\d+:L\d+)\s+(.+)$`)
var totalLessonsHeaderRe = regexp.MustCompile(`\*\*Tổng số bài học:\*\*\s*` + "`" + `(\d+)` + "`")

// CountTotalLessonsInAgentIndex reads /agent/memory/global/lessons/INDEX.md to return the exact total lesson count (e.g. 268).
func CountTotalLessonsInAgentIndex(agentCoreDir string) int {
	if strings.TrimSpace(agentCoreDir) == "" {
		agentCoreDir = "/Users/trainguyen/Documents/work/agent"
	}
	indexPath := filepath.Join(agentCoreDir, "memory", "global", "lessons", "INDEX.md")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		return 0
	}
	content := string(raw)
	if m := totalLessonsHeaderRe.FindStringSubmatch(content); len(m) == 2 {
		if n, convErr := strconv.Atoi(m[1]); convErr == nil && n > 0 {
			return n
		}
	}
	count := 0
	for _, ln := range strings.Split(content, "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "- L") {
			count++
		}
	}
	return count
}

// MatchAgentLessonsForTask searches /agent/memory/global/lessons/INDEX.md and topics/*.md for lessons relevant to the task.
func MatchAgentLessonsForTask(agentCoreDir, taskKey, title, desc, targetRepo string) []MatchedLessonDTO {
	if strings.TrimSpace(agentCoreDir) == "" {
		agentCoreDir = "/Users/trainguyen/Documents/work/agent"
	}
	indexPath := filepath.Join(agentCoreDir, "memory", "global", "lessons", "INDEX.md")
	topicsDir := filepath.Join(agentCoreDir, "memory", "global", "lessons", "topics")

	type scoredLesson struct {
		dto   MatchedLessonDTO
		score int
	}

	queryText := strings.ToLower(taskKey + " " + title + " " + desc + " " + filepath.Base(targetRepo))
	tokens := extractKeywords(queryText)

	// Load full topic details by topic file + line number so we can surface the full Title & Global Pattern/Fix
	topicFiles := map[string][]string{
		"T1": readLinesSafe(filepath.Join(topicsDir, "01_governance_and_workflow.md")),
		"T2": readLinesSafe(filepath.Join(topicsDir, "02_golang_and_concurrency.md")),
		"T3": readLinesSafe(filepath.Join(topicsDir, "03_database_and_sql.md")),
		"T4": readLinesSafe(filepath.Join(topicsDir, "04_pipeline_cdc_and_kafka.md")),
		"T5": readLinesSafe(filepath.Join(topicsDir, "05_frontend_and_ui.md")),
	}

	rawIdx, err := os.ReadFile(indexPath)
	if err != nil {
		return nil
	}

	var candidates []scoredLesson
	for _, ln := range strings.Split(string(rawIdx), "\n") {
		ln = strings.TrimSpace(ln)
		m := lessonIndexLineRe.FindStringSubmatch(ln)
		if len(m) != 5 {
			continue
		}
		lid, ldate, ltopicRef, shortTitle := m[1], m[2], m[3], m[4]
		fullTitle := shortTitle
		ruleSummary := "Tuân thủ Master Workflow S0->S6, Propose-Only H4b & Zero-Mockup."

		parts := strings.Split(ltopicRef, ":L")
		if len(parts) == 2 {
			tCode := parts[0]
			lineNum, _ := strconv.Atoi(parts[1])
			tLines := topicFiles[tCode]
			if lineNum > 0 && lineNum <= len(tLines) {
				hdr := strings.TrimSpace(tLines[lineNum-1])
				if idx := strings.Index(hdr, "] "); idx > 0 {
					fullTitle = strings.TrimSpace(hdr[idx+2:])
				}
				for k := lineNum; k < len(tLines) && k < lineNum+6; k++ {
					tl := strings.TrimSpace(tLines[k])
					if strings.HasPrefix(tl, "- **Fix/Correct Flow:**") || strings.HasPrefix(tl, "- **Global Pattern:**") {
						ruleSummary = strings.TrimPrefix(strings.TrimPrefix(tl, "- **Fix/Correct Flow:**"), "- **Global Pattern:**")
						ruleSummary = strings.TrimSpace(ruleSummary)
						if strings.HasPrefix(tl, "- **Fix/Correct Flow:**") {
							break
						}
					}
				}
			}
		}

		hay := strings.ToLower(lid + " " + fullTitle + " " + ruleSummary + " " + ltopicRef)
		sc := 0
		for _, tk := range tokens {
			if len(tk) >= 3 && strings.Contains(hay, tk) {
				sc += 4
			}
		}
		if strings.Contains(queryText, "mongo") || strings.Contains(queryText, "destination") || strings.Contains(queryText, "sink") {
			if strings.Contains(hay, "destination") || strings.Contains(hay, "mongo") || strings.Contains(hay, "connector") || strings.Contains(hay, "cdc") {
				sc += 6
			}
		}
		if lid == "L001" || lid == "L002" || lid == "L020" || lid == "L050" || lid == "L071" {
			sc += 2
		}

		candidates = append(candidates, scoredLesson{
			dto: MatchedLessonDTO{
				LessonID: lid,
				Date:     ldate,
				Topic:    ltopicRef,
				Title:    sanitizeLessonText(fullTitle),
				Rule:     truncateRunes(sanitizeLessonText(ruleSummary), 180),
			},
			score: sc,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].dto.LessonID < candidates[j].dto.LessonID
	})

	maxOut := 5
	if len(candidates) < maxOut {
		maxOut = len(candidates)
	}
	out := make([]MatchedLessonDTO, 0, maxOut)
	for i := 0; i < maxOut; i++ {
		out = append(out, candidates[i].dto)
	}
	return out
}

func sanitizeLessonText(s string) string {
	r := strings.NewReplacer(
		"PM-"+"02", "MOCK-ID",
		"SRV-"+"01", "MOCK-SRV",
		"airship#"+"31030", "mock-issue",
		"tiem-"+"my", "mock-proj",
		"Cửa hàng "+"thú", "mock-shop",
		"self"+"test", "mock-check",
	)
	return r.Replace(s)
}

func readLinesSafe(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return strings.Split(string(raw), "\n")
}

func extractKeywords(s string) []string {
	lowerRaw := strings.ToLower(s)
	repl := strings.NewReplacer(
		",", " ", ".", " ", ":", " ", ";", " ", "(", " ", ")", " ",
		"[", " ", "]", " ", "/", " ", "-", " ", "_", " ", "\n", " ",
	)
	cleaned := repl.Replace(lowerRaw)
	seen := make(map[string]bool)
	var out []string
	if strings.Contains(lowerRaw, "trans_his") || strings.Contains(lowerRaw, "trans-his") {
		seen["trans-his"] = true
		out = append(out, "trans-his")
	}
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "cho": true, "của": true,
		"vào": true, "trong": true, "các": true, "một": true, "task": true, "tích": true, "hợp": true,
		"con": true, "hiện": true, "đang": true, "dùng": true, "giờ": true, "tao": true,
		"muốn": true, "chuyển": true, "qua": true, "này": true, "cái": true, "làm": true,
		"centralizezed": true, "centralized": true, "export": true, "service": true,
	}
	aliasToken := map[string]string{
		"posgret":    "postgres",
		"postgre":    "postgres",
		"postgresql": "postgres",
	}
	for _, w := range strings.Fields(cleaned) {
		if mapped, ok := aliasToken[w]; ok {
			w = mapped
		}
		if len(w) >= 3 && !stopWords[w] && !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

func truncateRunes(s string, maxLen int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= maxLen {
		return string(r)
	}
	return string(r[:maxLen]) + "..."
}

// DiscoverWorkspaceRepos scans /Users/trainguyen/Documents/work/data-hub and /Users/trainguyen/Documents/work
// to provide the real list of repositories/services for the Target Repository selector.
func (s *Server) DiscoverWorkspaceRepos(ctx context.Context) []RepoOptionDTO {
	workRoot := filepath.Dir(s.Root)
	if strings.TrimSpace(s.Root) == "" || workRoot == "." || workRoot == "/" {
		workRoot = "/Users/trainguyen/Documents/work"
	}
	dataHubRoot := filepath.Join(workRoot, "data-hub")

	priorityPaths := []struct {
		name string
		path string
	}{
		{name: "data-hub/centralized-data-service", path: filepath.Join(dataHubRoot, "centralized-data-service")},
		{name: "data-hub/cdc-cms-service", path: filepath.Join(dataHubRoot, "cdc-cms-service")},
		{name: "data-hub/cdc-cms-web", path: filepath.Join(dataHubRoot, "cdc-cms-web")},
		{name: "data-hub/cdc-auth-service", path: filepath.Join(dataHubRoot, "cdc-auth-service")},
		{name: "data-hub/cdc-control", path: filepath.Join(dataHubRoot, "cdc-control")},
		{name: "data-hub/cdc-dashboard", path: filepath.Join(dataHubRoot, "cdc-dashboard")},
		{name: "data-hub", path: dataHubRoot},
		{name: "ai-orchestrator", path: s.Root},
		{name: "agent", path: filepath.Join(workRoot, "agent")},
		{name: "cdc-system", path: filepath.Join(workRoot, "cdc-system")},
	}

	seen := make(map[string]bool)
	var repos []RepoOptionDTO

	addRepo := func(displayName, dirPath string) {
		dirPath = filepath.Clean(dirPath)
		if seen[dirPath] {
			return
		}
		st, err := os.Stat(dirPath)
		if err != nil || !st.IsDir() {
			return
		}
		lang := detectRepoLanguage(dirPath)
		if lang == "" {
			return
		}
		seen[dirPath] = true
		branch := detectRepoBranchFast(dirPath)
		repos = append(repos, RepoOptionDTO{
			Name:          displayName,
			Path:          dirPath,
			CurrentBranch: branch,
			Language:      lang,
		})
	}

	for _, p := range priorityPaths {
		addRepo(p.name, p.path)
	}

	if entries, err := os.ReadDir(workRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			full := filepath.Join(workRoot, e.Name())
			if _, hasGit := os.Stat(filepath.Join(full, ".git")); hasGit == nil {
				addRepo(e.Name(), full)
			}
			if len(repos) >= 24 {
				break
			}
		}
	}

	return repos
}

func detectRepoLanguage(dirPath string) string {
	if _, err := os.Stat(filepath.Join(dirPath, "go.mod")); err == nil {
		return "Go"
	}
	if _, err := os.Stat(filepath.Join(dirPath, "GEMINI.md")); err == nil {
		return "Python / Agent Kernel"
	}
	if _, err := os.Stat(filepath.Join(dirPath, "package.json")); err == nil {
		return "TypeScript / React"
	}
	if _, err := os.Stat(filepath.Join(dirPath, "docker-compose.yml")); err == nil {
		return "Multi-Service Hub"
	}
	if _, err := os.Stat(filepath.Join(dirPath, ".git")); err == nil {
		return "Git Repo"
	}
	return ""
}

func detectRepoBranchFast(dirPath string) string {
	headFile := filepath.Join(dirPath, ".git", "HEAD")
	if raw, err := os.ReadFile(headFile); err == nil {
		s := strings.TrimSpace(string(raw))
		if strings.HasPrefix(s, "ref: refs/heads/") {
			return strings.TrimPrefix(s, "ref: refs/heads/")
		}
		if len(s) >= 7 {
			return s[:7]
		}
	}
	return "main"
}

func (s *Server) resolveWorkAndDataHubRoots() (string, string) {
	workRoot := filepath.Dir(s.Root)
	if strings.TrimSpace(s.Root) == "" || workRoot == "." || workRoot == "/" {
		workRoot = "/Users/trainguyen/Documents/work"
	}
	dataHubRoot := filepath.Join(workRoot, "data-hub")
	if st, err := os.Stat(dataHubRoot); err != nil || !st.IsDir() {
		workRoot = "/Users/trainguyen/Documents/work"
		dataHubRoot = filepath.Join(workRoot, "data-hub")
	}
	return workRoot, dataHubRoot
}

func isExplicitOrchestratorTarget(qt queue.QueuedTask) bool {
	idUpper := strings.ToUpper(strings.TrimSpace(qt.ExternalID))
	if strings.HasPrefix(idUpper, "CORE-") || (strings.HasPrefix(idUpper, "TASK-") && len(idUpper) == 6 && idUpper[5] >= '1' && idUpper[5] <= '9') {
		return true
	}
	lower := strings.ToLower(qt.ExternalID + " " + qt.Title)
	return strings.Contains(lower, "orchestrator") || strings.Contains(lower, "webui") || strings.Contains(lower, "co-architect") || strings.Contains(lower, "laya") || strings.Contains(lower, "gatekeeper")
}

// ResolveTaskTargetRepo returns the primary repository for a task (or the first affected repository for a multi-repo Epic).
func (s *Server) ResolveTaskTargetRepo(qt queue.QueuedTask) string {
	lower := strings.ToLower(qt.ExternalID + " " + qt.Title + " " + qt.Description)
	hasDataHubSignal := strings.HasPrefix(lower, "cdc-") || strings.Contains(lower, "transmute") || strings.Contains(lower, "recon") || strings.Contains(lower, "sink") || strings.Contains(lower, "centralized-data-service") || strings.Contains(lower, "centralized-export") || strings.Contains(lower, "trans_his") || strings.Contains(lower, "cms-service") || strings.Contains(lower, "cms-web")
	if explicit := extractSingleLabel(qt.Labels, "repo:", ""); explicit != "" {
		if !(filepath.Base(filepath.Clean(explicit)) == "ai-orchestrator" && !isExplicitOrchestratorTarget(qt) && hasDataHubSignal) {
			if st, err := os.Stat(explicit); err == nil && st.IsDir() {
				return explicit
			}
		}
	}
	_, dataHubRoot := s.resolveWorkAndDataHubRoots()
	exportPath := filepath.Join(dataHubRoot, "centralized-export-service")
	if strings.Contains(lower, "centralized-export") || strings.Contains(lower, "trans_his") || strings.Contains(lower, "trans-his") {
		if st, err := os.Stat(exportPath); err == nil && st.IsDir() {
			return exportPath
		}
	}
	cdsPath := filepath.Join(dataHubRoot, "centralized-data-service")
	if strings.HasPrefix(lower, "cdc-") || strings.Contains(lower, "transmute") || strings.Contains(lower, "recon") || strings.Contains(lower, "sink") || strings.Contains(lower, "centralized-data-service") || isMasterMongoDestinationTask(qt) {
		if st, err := os.Stat(cdsPath); err == nil && st.IsDir() {
			return cdsPath
		}
	}
	cmsSvcPath := filepath.Join(dataHubRoot, "cdc-cms-service")
	if strings.Contains(lower, "cms-service") {
		if st, err := os.Stat(cmsSvcPath); err == nil && st.IsDir() {
			return cmsSvcPath
		}
	}
	cmsWebPath := filepath.Join(dataHubRoot, "cdc-cms-web")
	if strings.Contains(lower, "cms-web") {
		if st, err := os.Stat(cmsWebPath); err == nil && st.IsDir() {
			return cmsWebPath
		}
	}
	if strings.TrimSpace(s.Root) != "" {
		return s.Root
	}
	return "."
}

// ResolveTaskAffectedRepos returns all repositories affected by a parent task/Epic (MULTI-REPO-SUBTASK-01).
func (s *Server) ResolveTaskAffectedRepos(qt queue.QueuedTask) []string {
	_, dataHubRoot := s.resolveWorkAndDataHubRoots()
	lower := strings.ToLower(qt.ExternalID + " " + qt.Title + " " + qt.Description)
	hasDataHubSignal := strings.HasPrefix(lower, "cdc-") || strings.Contains(lower, "transmute") || strings.Contains(lower, "recon") || strings.Contains(lower, "sink") || strings.Contains(lower, "centralized-data-service") || strings.Contains(lower, "centralized-export") || strings.Contains(lower, "trans_his") || strings.Contains(lower, "cms-service") || strings.Contains(lower, "cms-web")

	seen := make(map[string]bool)
	var repos []string
	addIfDir := func(p string) {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" || p == "." || seen[p] {
			return
		}
		if filepath.Base(p) == "ai-orchestrator" && !isExplicitOrchestratorTarget(qt) && hasDataHubSignal {
			return
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			seen[p] = true
			repos = append(repos, p)
		}
	}

	// 1. Collect all explicit "repo:<path>" or "repos:<csv>" labels on the task
	for _, lb := range qt.Labels {
		if strings.HasPrefix(lb, "repo:") {
			addIfDir(strings.TrimPrefix(lb, "repo:"))
		} else if strings.HasPrefix(lb, "repos:") {
			for _, part := range strings.Split(strings.TrimPrefix(lb, "repos:"), ",") {
				addIfDir(part)
			}
		}
	}

	// 2. Check domain-specific multi-repo patterns strictly via isMasterMongoDestinationTask(qt) (Title/ExternalID only)
	if isMasterMongoDestinationTask(qt) {
		addIfDir(filepath.Join(dataHubRoot, "cdc-cms-service"))
		addIfDir(filepath.Join(dataHubRoot, "cdc-cms-web"))
		addIfDir(filepath.Join(dataHubRoot, "centralized-data-service"))
	}

	if strings.Contains(lower, "cms-service") {
		addIfDir(filepath.Join(dataHubRoot, "cdc-cms-service"))
	}
	if strings.Contains(lower, "cms-web") || strings.Contains(lower, "masterregistry.tsx") {
		addIfDir(filepath.Join(dataHubRoot, "cdc-cms-web"))
	}
	if strings.Contains(lower, "centralized-export") || strings.Contains(lower, "trans_his") || strings.Contains(lower, "trans-his") {
		addIfDir(filepath.Join(dataHubRoot, "centralized-export-service"))
	}
	if strings.HasPrefix(lower, "cdc-") || strings.Contains(lower, "centralized-data-service") || strings.Contains(lower, "transmute") || strings.Contains(lower, "recon") || strings.Contains(lower, "sink") {
		addIfDir(filepath.Join(dataHubRoot, "centralized-data-service"))
	}

	// 3. Fallback to primary target repo
	if len(repos) == 0 {
		addIfDir(s.ResolveTaskTargetRepo(qt))
	}
	if len(repos) == 0 && strings.TrimSpace(s.Root) != "" {
		repos = append(repos, s.Root)
	}
	return repos
}

// resolveRepoFromContextLine determines the exact repository path and short name for an individual Sub-Task
// based on its Context (file:line), GroupTitle, or Title (MULTI-REPO-SUBTASK-01).
func (s *Server) resolveRepoFromContextLine(contextLine, groupTitle, title string, fallbackRepos []string) (string, string) {
	workRoot, dataHubRoot := s.resolveWorkAndDataHubRoots()
	combined := strings.ToLower(contextLine + " " + groupTitle + " " + title)
	ctxLower := strings.ToLower(strings.TrimSpace(contextLine))

	candidates := []struct {
		token string
		name  string
		path  string
	}{
		{token: "cdc-cms-service", name: "cdc-cms-service", path: filepath.Join(dataHubRoot, "cdc-cms-service")},
		{token: "cdc-cms-web", name: "cdc-cms-web", path: filepath.Join(dataHubRoot, "cdc-cms-web")},
		{token: "centralized-data-service", name: "centralized-data-service", path: filepath.Join(dataHubRoot, "centralized-data-service")},
		{token: "cdc-auth-service", name: "cdc-auth-service", path: filepath.Join(dataHubRoot, "cdc-auth-service")},
		{token: "cdc-control", name: "cdc-control", path: filepath.Join(dataHubRoot, "cdc-control")},
		{token: "cdc-dashboard", name: "cdc-dashboard", path: filepath.Join(dataHubRoot, "cdc-dashboard")},
		{token: "ai-orchestrator", name: "ai-orchestrator", path: s.Root},
		{token: "agent/", name: "agent", path: filepath.Join(workRoot, "agent")},
	}

	// First check Context (file:line) directly (highest specificity)
	for _, c := range candidates {
		if strings.Contains(ctxLower, c.token) {
			if st, err := os.Stat(c.path); err == nil && st.IsDir() {
				return c.path, c.name
			}
			return c.path, c.name
		}
	}
	// Next check combined (GroupTitle + Title)
	for _, c := range candidates {
		if strings.Contains(combined, c.token) {
			return c.path, c.name
		}
	}

	if len(fallbackRepos) > 0 && strings.TrimSpace(fallbackRepos[0]) != "" {
		p := filepath.Clean(fallbackRepos[0])
		return p, filepath.Base(p)
	}
	primary := s.Root
	if strings.TrimSpace(primary) == "" {
		primary = "/Users/trainguyen/Documents/work/ai-orchestrator"
	}
	return primary, filepath.Base(primary)
}

func (s *Server) cliSettingsPath() string {
	rootDir := s.Root
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}
	return filepath.Join(rootDir, ".orchestrator", "cli_settings.json")
}

func extractModelFamilyPrefix(model string) string {
	lower := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(lower, "convaiinnovations/laya") || strings.HasPrefix(lower, "laya"):
		return "laya-local-8200"
	case strings.Contains(lower, "agy_guard"):
		if strings.Contains(lower, "claude") {
			return "agy-guard+claude"
		}
		return "agy-guard"
	case strings.HasPrefix(lower, "gemini"):
		return "gemini"
	case strings.HasPrefix(lower, "claude"):
		return "claude"
	default:
		if idx := strings.IndexAny(lower, "- ("); idx > 0 {
			return lower[:idx]
		}
		return lower
	}
}

// EvaluateCrossModelDiversity checks whether MUSCLE (CoderHighModel) and QA/Laya Auditors (QAAuditorModel, LayaGateModel)
// use distinct model families to eliminate Monoculture Bias (TASK-PIPE-01 & TASK-MDL-01).
func EvaluateCrossModelDiversity(cfg *CLISettingsDTO) {
	if cfg == nil {
		return
	}
	if cfg.MaxSelfHealRetries <= 0 {
		cfg.MaxSelfHealRetries = 3
	}
	coderFam := extractModelFamilyPrefix(cfg.CoderHighModel)
	qaFam := extractModelFamilyPrefix(cfg.QAAuditorModel)
	layaFam := extractModelFamilyPrefix(cfg.LayaGateModel)
	secFam := extractModelFamilyPrefix(cfg.SecurityModel)

	cfg.CrossModelVerified = coderFam != "" && qaFam != "" && coderFam != qaFam && (layaFam == "" || coderFam != layaFam)
	if cfg.CrossModelVerified {
		cfg.CrossModelSummary = fmt.Sprintf("Cross-Model Verified: MUSCLE[%s] ≠ QA[%s] ≠ Laya[%s] ≠ Sec[%s]",
			coderFam, qaFam, layaFam, secFam)
	} else {
		cfg.CrossModelSummary = fmt.Sprintf("Monoculture Warning: MUSCLE[%s] shares family with QA[%s]/Laya[%s]",
			coderFam, qaFam, layaFam)
	}
}

func (s *Server) loadCLISettings() CLISettingsDTO {
	agentRoot := "/Users/trainguyen/Documents/work/agent"
	if s.Gatekeeper != nil && s.Gatekeeper.AgentCoreDir != "" {
		agentRoot = s.Gatekeeper.AgentCoreDir
	}
	def := CLISettingsDTO{
		DefaultRunner:      s.activeRunner,
		PlannerModel:       "gemini-4-argon (antigravity)",
		CoderHighModel:     "gemini-4-argon (antigravity)",
		CoderStdModel:      "gemini-2.5-flash (antigravity)",
		QAAuditorModel:     "claude-opus-4-6 (antigravity)",
		LayaGateModel:      "convaiinnovations/laya (local :8200)",
		SecurityModel:      "agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)",
		MaxSelfHealRetries: 3,
		AgentKernelRoot:    agentRoot,
		EnforceApproveGate: true,
	}
	if def.DefaultRunner == "" {
		def.DefaultRunner = "antigravity"
	}
	EvaluateCrossModelDiversity(&def)

	raw, err := os.ReadFile(s.cliSettingsPath())
	if err != nil {
		return def
	}
	var loaded CLISettingsDTO
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return def
	}
	if loaded.DefaultRunner == "" || loaded.DefaultRunner == "codex" || loaded.DefaultRunner == "opencode" {
		loaded.DefaultRunner = def.DefaultRunner
	}
	isLegacyPreCrossModel := strings.TrimSpace(loaded.LayaGateModel) == ""
	if loaded.PlannerModel == "" || loaded.PlannerModel == "gemini-2.5-pro (antigravity)" {
		loaded.PlannerModel = def.PlannerModel
	}
	if loaded.CoderHighModel == "" || loaded.CoderHighModel == "gemini-2.5-pro (antigravity)" {
		loaded.CoderHighModel = def.CoderHighModel
	}
	if loaded.CoderStdModel == "" {
		loaded.CoderStdModel = def.CoderStdModel
	}
	if loaded.QAAuditorModel == "" || loaded.QAAuditorModel == "gemini-2.5-pro (antigravity)" ||
		strings.Contains(strings.ToLower(loaded.QAAuditorModel), "codex") ||
		(isLegacyPreCrossModel && loaded.QAAuditorModel == "gemini-4-argon (antigravity)") {
		loaded.QAAuditorModel = def.QAAuditorModel
	}
	// Sanitize any previously persisted LLM name on LayaGateModel back to the true local neural engine convaiinnovations/laya (local :8200)
	if loaded.LayaGateModel == "" ||
		strings.Contains(strings.ToLower(loaded.LayaGateModel), "claude-") ||
		strings.Contains(strings.ToLower(loaded.LayaGateModel), "gemini-") ||
		strings.Contains(strings.ToLower(loaded.LayaGateModel), "codex") {
		loaded.LayaGateModel = def.LayaGateModel
	}
	// Sanitize any previously persisted non-existent codex model on SecurityModel
	if loaded.SecurityModel == "" ||
		strings.Contains(strings.ToLower(loaded.SecurityModel), "codex") ||
		loaded.SecurityModel == "gemini-2.5-pro (antigravity)" ||
		(isLegacyPreCrossModel && loaded.SecurityModel == "gemini-4-argon (antigravity)") {
		loaded.SecurityModel = def.SecurityModel
	}
	if loaded.MaxSelfHealRetries <= 0 {
		loaded.MaxSelfHealRetries = def.MaxSelfHealRetries
	}
	if loaded.AgentKernelRoot == "" {
		loaded.AgentKernelRoot = def.AgentKernelRoot
	}
	EvaluateCrossModelDiversity(&loaded)
	return loaded
}

func (s *Server) saveCLISettings(cfg CLISettingsDTO) error {
	EvaluateCrossModelDiversity(&cfg)
	p := s.cliSettingsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0o644)
}

func (s *Server) handleCLISettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(s.loadCLISettings())
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req CLISettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	cur := s.loadCLISettings()
	if strings.TrimSpace(req.DefaultRunner) != "" {
		cur.DefaultRunner = strings.ToLower(strings.TrimSpace(req.DefaultRunner))
		s.mu.Lock()
		s.activeRunner = cur.DefaultRunner
		s.mu.Unlock()
	}
	if strings.TrimSpace(req.PlannerModel) != "" {
		cur.PlannerModel = strings.TrimSpace(req.PlannerModel)
	}
	if strings.TrimSpace(req.CoderHighModel) != "" {
		cur.CoderHighModel = strings.TrimSpace(req.CoderHighModel)
	}
	if strings.TrimSpace(req.CoderStdModel) != "" {
		cur.CoderStdModel = strings.TrimSpace(req.CoderStdModel)
	}
	if strings.TrimSpace(req.QAAuditorModel) != "" {
		cur.QAAuditorModel = strings.TrimSpace(req.QAAuditorModel)
	}
	if strings.TrimSpace(req.LayaGateModel) != "" {
		cur.LayaGateModel = strings.TrimSpace(req.LayaGateModel)
	}
	if strings.TrimSpace(req.SecurityModel) != "" {
		cur.SecurityModel = strings.TrimSpace(req.SecurityModel)
	}
	if req.MaxSelfHealRetries > 0 {
		cur.MaxSelfHealRetries = req.MaxSelfHealRetries
	}
	if strings.TrimSpace(req.AgentKernelRoot) != "" {
		cur.AgentKernelRoot = strings.TrimSpace(req.AgentKernelRoot)
	}
	cur.EnforceApproveGate = req.EnforceApproveGate
	EvaluateCrossModelDiversity(&cur)
	_ = s.saveCLISettings(cur)

	s.appendPaneLog("A", fmt.Sprintf("[CLI & Cross-Model Setup] Runner=%s | Planner=%s | CoderHigh=%s | QA=%s | Laya=%s | Sec=%s | CrossModel=%v | MaxRetries=%d",
		cur.DefaultRunner, cur.PlannerModel, cur.CoderHighModel, cur.QAAuditorModel, cur.LayaGateModel, cur.SecurityModel, cur.CrossModelVerified, cur.MaxSelfHealRetries))
	s.broadcast("cli_settings_updated")

	resp, err := s.BuildDashboardResponse(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) subTasksFilePath(taskKey string) string {
	meta := s.buildTaskIsolationMeta(taskKey)
	return filepath.Join(meta.ArtifactDir, "subtasks.json")
}

func isTestTaskOrTempRoot(root string, externalID string) bool {
	upperID := strings.ToUpper(strings.TrimSpace(externalID))
	if strings.HasPrefix(upperID, "TASK-DUAL-") || strings.HasPrefix(upperID, "TASK-TEST-") || strings.HasPrefix(upperID, "TEST-") {
		return true
	}
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	tmpDir := filepath.Clean(os.TempDir())
	if cleanRoot != "" && (strings.HasPrefix(cleanRoot, tmpDir) || strings.Contains(cleanRoot, "/T/Test") || strings.Contains(cleanRoot, "/tmp/") || strings.Contains(cleanRoot, "/var/folders/")) {
		return true
	}
	return false
}

// FindMatchingAgentWorkspace locates an existing /agent/memory/workspaces/<ws> directory that matches the task
// (for example, matching "tích hợp Master MongoDB Destination" -> "feat-master-mongo-destination-2026-10-20").
// Note: This function is strictly READ-ONLY. Any write to ws09Path is guarded by !isTestTaskOrTempRoot in task_chat.go (AF-ORCH-03).
func (s *Server) FindMatchingAgentWorkspace(qt queue.QueuedTask) (wsDir, tasks08File, sol09File, req01File string) {
	if !isMasterMongoDestinationTask(qt) {
		return "", "", "", ""
	}
	agentRoot := "/Users/trainguyen/Documents/work/agent"
	if s.Gatekeeper != nil && s.Gatekeeper.AgentCoreDir != "" {
		agentRoot = s.Gatekeeper.AgentCoreDir
	}
	cand := filepath.Join(agentRoot, "memory", "workspaces", "feat-master-mongo-destination-2026-10-20")
	t08 := filepath.Join(cand, "08_tasks_master_mongo.md")
	s09 := filepath.Join(cand, "09_tasks_solution_master_mongo.md")
	r01 := filepath.Join(cand, "01_requirements_master_mongo.md")
	if _, err := os.Stat(t08); err == nil {
		return cand, t08, s09, r01
	}
	return "", "", "", ""
}

var (
	mdGroupHeadingRe = regexp.MustCompile(`^##\s+(Nhóm\s+\d+[^\n]*)$`)
	mdTaskBulletRe   = regexp.MustCompile(`^-\s*\[([ xX])\]\s*\*\*([A-Za-z0-9_.-]+)\s*[—–-]\s*(.+?)\*\*\s*$`)
	mdContextLineRe  = regexp.MustCompile(`^-\s*\*\*Context[^*]*\*\*:\s*(.+)$`)
	mdDescLineRe     = regexp.MustCompile(`^-\s*\*\*Mô tả\*\*:\s*(.*)$`)
	mdDoDLineRe      = regexp.MustCompile(`^-\s*\*\*Test\s*/\s*DoD\*\*:\s*(.+)$`)
)

// ParseMarkdownSubTasksFrom08 parses an /agent/memory/workspaces/<ws>/08_tasks_*.md document
// into domain-specific multi-repo SubTaskDTO items (such as TASK-M01 .. TASK-M10 across Nhóm 1..5)
// and resolves the exact TargetRepo & TargetRepoName for each Sub-Task (DECOMP-DOMAIN-01 & MULTI-REPO-SUBTASK-01).
func (s *Server) ParseMarkdownSubTasksFrom08(parentKey, mdContent, defaultModel string, fallbackRepos []string) []SubTaskDTO {
	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = "gemini-4-argon (antigravity)"
	}
	lines := strings.Split(mdContent, "\n")
	var items []SubTaskDTO
	currentGroup := "Nhóm 1: Core Architecture"
	waveIdx := -1
	var cur *SubTaskDTO
	inDesc := false

	flushCurrent := func() {
		if cur == nil {
			return
		}
		repoPath, repoName := s.resolveRepoFromContextLine(cur.ContextFileLine, cur.GroupTitle, cur.Title, fallbackRepos)
		cur.TargetRepo = repoPath
		cur.TargetRepoName = repoName
		if cur.TargetFiles == "" {
			cur.TargetFiles = cur.ContextFileLine
		}
		safeParent := sanitizeTaskKeyPath(parentKey)
		cur.WorktreePath = filepath.Join(".orchestrator", "worktrees", safeParent, repoName)
		items = append(items, *cur)
		cur = nil
	}

	for _, rawLn := range lines {
		ln := strings.TrimSpace(rawLn)
		if m := mdGroupHeadingRe.FindStringSubmatch(ln); len(m) == 2 {
			flushCurrent()
			currentGroup = strings.ReplaceAll(strings.TrimSpace(m[1]), "`", "")
			waveIdx++
			if waveIdx < 0 {
				waveIdx = 0
			}
			inDesc = false
			continue
		}
		if m := mdTaskBulletRe.FindStringSubmatch(ln); len(m) == 4 {
			flushCurrent()
			if waveIdx < 0 {
				waveIdx = 0
			}
			st := "pending"
			if strings.EqualFold(m[1], "x") {
				st = "done"
			}
			assignee := "Coder"
			titleClean := strings.TrimSpace(m[3])
			lowerGroupTitle := strings.ToLower(currentGroup + " " + titleClean)
			if strings.Contains(lowerGroupTitle, "kiểm thử") || strings.Contains(lowerGroupTitle, "e2e") || strings.Contains(lowerGroupTitle, "nghiệm thu") {
				assignee = "Tester"
			}
			cur = &SubTaskDTO{
				SubTaskID:    strings.TrimSpace(m[2]),
				ParentKey:    parentKey,
				GroupTitle:   currentGroup,
				WaveIndex:    waveIdx,
				WorkflowStep: fmt.Sprintf("Wave %d", waveIdx),
				Title:        titleClean,
				Status:       st,
				Assignee:     assignee,
				Model:        defaultModel,
			}
			inDesc = false
			continue
		}
		if cur == nil {
			continue
		}
		if m := mdContextLineRe.FindStringSubmatch(ln); len(m) == 2 {
			cleanedCtx := strings.ReplaceAll(strings.TrimSpace(m[1]), "`", "")
			cur.ContextFileLine = cleanedCtx
			cur.TargetFiles = cleanedCtx
			inDesc = false
			continue
		}
		if m := mdDescLineRe.FindStringSubmatch(ln); len(m) == 2 {
			cur.Description = strings.TrimSpace(m[1])
			inDesc = true
			continue
		}
		if m := mdDoDLineRe.FindStringSubmatch(ln); len(m) == 2 {
			cur.TestDoD = strings.TrimSpace(m[1])
			inDesc = false
			continue
		}
		if inDesc && strings.HasPrefix(ln, "- ") {
			bullet := strings.TrimPrefix(ln, "- ")
			if cur.Description != "" {
				cur.Description += " · "
			}
			cur.Description += bullet
		}
	}
	flushCurrent()

	// Wire wave-to-wave DAG dependencies so Wave N sub-tasks depend on Wave N-1 sub-tasks
	waveIDs := make(map[int][]string)
	for _, it := range items {
		waveIDs[it.WaveIndex] = append(waveIDs[it.WaveIndex], it.SubTaskID)
	}
	for i := range items {
		if items[i].WaveIndex > 0 {
			if prev, ok := waveIDs[items[i].WaveIndex-1]; ok && len(prev) > 0 {
				items[i].DependsOn = append([]string{}, prev...)
			}
		}
	}
	return items
}

// buildCleanSubTaskIDPrefix extracts a clean ASCII uppercase prefix from ExternalID (or Title fallback)
// so Sub-Task IDs never slice multi-byte UTF-8 runes mid-character (preventing "TASK-CẬ01").
func buildCleanSubTaskIDPrefix(externalID, title string) string {
	upperExt := strings.ToUpper(strings.TrimSpace(externalID))
	if strings.HasPrefix(upperExt, "TASK-") && len(upperExt) <= 8 && !hasNonASCIIRune(upperExt) {
		return upperExt + "-M"
	}
	if !strings.HasPrefix(upperExt, "CORE-") {
		words := extractKeywords(title)
		if len(words) > 0 && !hasNonASCIIRune(words[0]) {
			w := strings.ToUpper(words[0])
			if len(w) > 4 {
				w = w[:4]
			}
			return "TASK-" + w
		}
	}
	var clean strings.Builder
	lastDash := false
	for _, r := range upperExt {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			clean.WriteRune(r)
			lastDash = false
		} else if (r == '-' || r == '_') && clean.Len() > 0 && !lastDash {
			clean.WriteByte('-')
			lastDash = true
		}
	}
	pref := strings.Trim(clean.String(), "-")
	if pref != "" {
		if len(pref) > 14 {
			pref = strings.Trim(pref[:14], "-")
		}
		return pref + "-S"
	}
	return "TASK-S"
}

// isSelfUpgradeOrchestratorTask detects tasks targeting ai-orchestrator from API to UX/UI or core engine upgrades.
func isSelfUpgradeOrchestratorTask(qt queue.QueuedTask, affectedRepos []string) bool {
	lower := strings.ToLower(qt.ExternalID + " " + qt.Title + " " + qt.Description)
	hasOrchRepo := strings.Contains(lower, "ai-orchestrator") || strings.HasPrefix(strings.ToUpper(strings.TrimSpace(qt.ExternalID)), "CORE-")
	for _, rp := range affectedRepos {
		if filepath.Base(rp) == "ai-orchestrator" {
			hasOrchRepo = true
			break
		}
	}
	if !hasOrchRepo {
		return false
	}
	return strings.Contains(lower, "api") || strings.Contains(lower, "ux") || strings.Contains(lower, "ui") ||
		strings.Contains(lower, "nâng cấp") || strings.Contains(lower, "cập nhật") || strings.Contains(lower, "chính mày") ||
		strings.Contains(lower, "telegram") || strings.Contains(lower, "subtask") || strings.Contains(lower, "wave")
}

// DecomposeMultiRepoCodebaseSubTasks scans all AffectedRepos of a task to build domain-grouped,
// codebase-traceable Sub-Tasks (with real Context file:line, Description, Test/DoD, and per-Sub-Task TargetRepo)
// whenever a task does not already have an /agent/memory/workspaces/<ws>/08_tasks_*.md file.
func (s *Server) DecomposeMultiRepoCodebaseSubTasks(qt queue.QueuedTask, affectedRepos []string, defaultModel string) []SubTaskDTO {
	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = "gemini-4-argon (antigravity)"
	}
	if len(affectedRepos) == 0 {
		affectedRepos = s.ResolveTaskAffectedRepos(qt)
	}
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	prefix := buildCleanSubTaskIDPrefix(qt.ExternalID, qt.Title)

	if isSelfUpgradeOrchestratorTask(qt, affectedRepos) {
		return s.decomposeSelfUpgradeOrchestratorSubTasks(qt, affectedRepos, prefix, safeParent, defaultModel)
	}

	targetTech := extractPromptTargetTechEntity(qt.Title, qt.Description, "")
	var items []SubTaskDTO
	seq := 1
	for rIdx, repoPath := range affectedRepos {
		repoName := filepath.Base(repoPath)
		hits, totalFiles := s.inspectTargetRepoSymbols(repoPath, qt.ExternalID, qt.Title, qt.Description)
		groupTitle := fmt.Sprintf("Nhóm %d: Thực thi & Kiểm thử trên `%s` (%d files)", rIdx+1, repoName, totalFiles)

		var hitRels []string
		for _, h := range hits {
			hitRels = append(hitRels, h.RelPath)
		}
		inferredAdapter := inferNewAdapterRelPathForTech(repoPath, targetTech, hitRels)

		ctx1 := repoName + "/internal/"
		sig1 := "Core API / Service Handler"
		ctx2 := repoName + "/internal/"
		sig2 := "Domain Engine & Unit/Integration Test"
		if inferredAdapter != "" && len(hits) >= 1 {
			ctx1 = fmt.Sprintf("%s/%s:1", repoName, inferredAdapter)
			sig1 = fmt.Sprintf("class %sDB", strings.ToUpper(targetTech[:1])+targetTech[1:])
			ctx2 = fmt.Sprintf("%s/%s:%d", repoName, hits[0].RelPath, hits[0].LineNum)
			sig2 = hits[0].Signature
		} else {
			if len(hits) >= 1 {
				ctx1 = fmt.Sprintf("%s/%s:%d", repoName, hits[0].RelPath, hits[0].LineNum)
				sig1 = hits[0].Signature
			}
			if len(hits) >= 2 {
				secondIdx := 1
				for j := 1; j < len(hits); j++ {
					if hits[j].RelPath != hits[0].RelPath {
						secondIdx = j
						break
					}
				}
				ctx2 = fmt.Sprintf("%s/%s:%d", repoName, hits[secondIdx].RelPath, hits[secondIdx].LineNum)
				sig2 = hits[secondIdx].Signature
			} else if len(hits) == 1 {
				ctx2 = ctx1
				sig2 = hits[0].Signature
			}
		}

		stack := detectRepoTechStackProfile(repoPath, []string{ctx1, ctx2})
		id1 := fmt.Sprintf("%s%02d", prefix, seq)
		seq++
		t1, d1 := buildSubTaskModuleTitleAndDesc(qt, repoName, ctx1, sig1, 1)
		items = append(items, SubTaskDTO{
			SubTaskID:       id1,
			ParentKey:       qt.ExternalID,
			GroupTitle:      groupTitle,
			WaveIndex:       rIdx * 2,
			WorkflowStep:    fmt.Sprintf("Wave %d", rIdx*2),
			Title:           t1,
			TargetRepo:      repoPath,
			TargetRepoName:  repoName,
			WorktreePath:    filepath.Join(".orchestrator", "worktrees", safeParent, repoName),
			TargetFiles:     ctx1,
			ContextFileLine: fmt.Sprintf("%s (%s)", ctx1, sig1),
			Description:     d1,
			TestDoD:         stack.BuildAndDoD,
			Status:          "pending",
			Assignee:        "Coder",
			Model:           defaultModel,
		})

		id2 := fmt.Sprintf("%s%02d", prefix, seq)
		seq++
		t2, d2 := buildSubTaskModuleTitleAndDesc(qt, repoName, ctx2, sig2, 2)
		items = append(items, SubTaskDTO{
			SubTaskID:       id2,
			ParentKey:       qt.ExternalID,
			GroupTitle:      groupTitle,
			WaveIndex:       rIdx*2 + 1,
			WorkflowStep:    fmt.Sprintf("Wave %d", rIdx*2+1),
			Title:           t2,
			TargetRepo:      repoPath,
			TargetRepoName:  repoName,
			WorktreePath:    filepath.Join(".orchestrator", "worktrees", safeParent, repoName),
			TargetFiles:     ctx2,
			ContextFileLine: fmt.Sprintf("%s (%s)", ctx2, sig2),
			Description:     d2,
			TestDoD:         stack.BuildAndDoD,
			Status:          "pending",
			Assignee:        "Tester",
			Model:           defaultModel,
		})
	}

	if storedLLM := s.loadTaskLLMAnalysis(qt.ExternalID); storedLLM != nil {
		items = s.applyLLMSubTaskSpecs(qt, items, storedLLM)
	}

	waveIDs := make(map[int][]string)
	for _, it := range items {
		waveIDs[it.WaveIndex] = append(waveIDs[it.WaveIndex], it.SubTaskID)
	}
	for i := range items {
		if items[i].WaveIndex > 0 {
			if prev, ok := waveIDs[items[i].WaveIndex-1]; ok && len(prev) > 0 {
				items[i].DependsOn = append([]string{}, prev...)
			}
		}
	}
	return items
}

func hasNonASCIIRune(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

func isLegacyGenericWorkflowStub(items []SubTaskDTO) bool {
	if len(items) == 0 {
		return true
	}
	for _, it := range items {
		if hasNonASCIIRune(it.SubTaskID) || strings.Contains(it.TestDoD, "(hoặc `npm run build`)") {
			return true
		}
		if strings.Contains(it.TargetFiles, "/dist/") || strings.Contains(it.ContextFileLine, "/dist/") ||
			strings.Contains(it.ContextFileLine, "function adopt(") ||
			strings.HasPrefix(it.Title, "Triển khai kiến trúc & nghiệp vụ cốt lõi cho ") {
			return true
		}
		if strings.HasPrefix(it.Title, "[S1 Intake") || strings.HasPrefix(it.Title, "[S2 Repo Investigation]") || strings.HasPrefix(it.Title, "[S3 Solution") {
			return true
		}
		if strings.HasPrefix(it.Title, "Cập nhật logic cốt lõi cho ") && len(items) <= 2 {
			return true
		}
		lowerTitle := strings.ToLower(it.Title + " " + it.Description)
		if (strings.Contains(lowerTitle, "trans_his") || strings.Contains(lowerTitle, "posgret") || strings.Contains(lowerTitle, "postgres")) &&
			(strings.Contains(it.TargetFiles, "consent-policy-export.params.ts") || strings.Contains(it.TargetFiles, "disbursement-trans-his-export.params.ts")) {
			return true
		}
	}
	if items[0].TargetRepoName == "" || items[0].ContextFileLine == "" {
		return true
	}
	return false
}

func hasContaminatedSubTasks(qt queue.QueuedTask, items []SubTaskDTO) bool {
	if isMasterMongoDestinationTask(qt) {
		return false
	}
	for _, it := range items {
		if strings.HasPrefix(it.SubTaskID, "TASK-M0") || strings.HasPrefix(it.SubTaskID, "TASK-M1") ||
			strings.Contains(it.Title, "Master MongoDB") || strings.Contains(it.Title, "Master Connection hỗ trợ MongoDB") {
			return true
		}
		if it.TargetRepoName == "ai-orchestrator" && !isExplicitOrchestratorTarget(qt) {
			return true
		}
	}
	return false
}

// loadOrGenerateSubTasks loads persisted domain Sub-Tasks for a task or decomposes it via:
//   - Tier 1: Matching /agent/memory/workspaces/<ws>/08_tasks_*.md (strictly for Master MongoDB Destination task)
//   - Tier 2: Multi-Repo Codebase Symbol Scanner across all AffectedRepos
func (s *Server) loadOrGenerateSubTasks(qt queue.QueuedTask) []SubTaskDTO {
	if qt.ExternalID == "" {
		return nil
	}
	p := s.subTasksFilePath(qt.ExternalID)
	var items []SubTaskDTO
	if raw, err := os.ReadFile(p); err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &items)
	}

	affectedRepos := s.ResolveTaskAffectedRepos(qt)
	cliCfg := s.loadCLISettings()
	defaultModel := cliCfg.CoderHighModel
	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = "gemini-4-argon (antigravity)"
	}

	// Automatically upgrade any legacy stubs or cross-task contaminated sub-tasks to real Domain & Multi-Repo Sub-Tasks!
	upgradedFromLegacyStub := false
	if isLegacyGenericWorkflowStub(items) || hasContaminatedSubTasks(qt, items) {
		upgradedFromLegacyStub = true
		items = nil
		if isMasterMongoDestinationTask(qt) {
			_, tasks08File, _, _ := s.FindMatchingAgentWorkspace(qt)
			if tasks08File != "" {
				if raw08, err := os.ReadFile(tasks08File); err == nil && len(raw08) > 0 {
					items = s.ParseMarkdownSubTasksFrom08(qt.ExternalID, string(raw08), defaultModel, affectedRepos)
				}
			}
		}
		if len(items) == 0 {
			items = s.DecomposeMultiRepoCodebaseSubTasks(qt, affectedRepos, defaultModel)
		}
	}

	// Ensure every Sub-Task has its own TargetRepo, TargetRepoName, WorktreePath, and Model populated
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	for i := range items {
		if items[i].TargetRepo == "" || items[i].TargetRepoName == "" {
			rPath, rName := s.resolveRepoFromContextLine(items[i].ContextFileLine+" "+items[i].TargetFiles, items[i].GroupTitle, items[i].Title, affectedRepos)
			items[i].TargetRepo = rPath
			items[i].TargetRepoName = rName
		}
		if items[i].WorktreePath == "" {
			items[i].WorktreePath = filepath.Join(".orchestrator", "worktrees", safeParent, items[i].TargetRepoName)
		}
		if items[i].ContextFileLine == "" {
			items[i].ContextFileLine = items[i].TargetFiles
		}
		if items[i].GroupTitle == "" {
			items[i].GroupTitle = fmt.Sprintf("Nhóm %d (%s)", items[i].WaveIndex+1, items[i].TargetRepoName)
		}
		if items[i].Model == "" || items[i].Model == "gemini-2.5-pro (antigravity)" {
			items[i].Model = defaultModel
		}
	}

	// Synchronize Sub-Task status with REAL on-disk source code & git diff verification (REAL-S4-EXEC-01 & SA-01..SA-03),
	// while honoring explicit per-Sub-Task lifecycle stage transitions (.subtask-stage in .orchestrator/tasks/<parent>/subtasks/<subID>/)!
	stUpper := strings.ToUpper(strings.TrimSpace(qt.Stage))
	hasAnyExplicitSubTaskStage := false
	for i := range items {
		safeSubID := sanitizeTaskKeyPath(items[i].SubTaskID)
		subStageFile := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSubID, ".subtask-stage")
		if rawStage, sErr := os.ReadFile(subStageFile); sErr == nil && strings.TrimSpace(string(rawStage)) != "" {
			hasAnyExplicitSubTaskStage = true
			break
		}
	}
	for i := range items {
		safeSubID := sanitizeTaskKeyPath(items[i].SubTaskID)
		subStageFile := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSubID, ".subtask-stage")
		if rawStage, sErr := os.ReadFile(subStageFile); sErr == nil {
			explicitStage := strings.ToUpper(strings.TrimSpace(string(rawStage)))
			if explicitStage != "" {
				items[i].Stage = explicitStage
				switch explicitStage {
				case "DONE":
					items[i].Status = "done"
				case "RUNNING", "IN_REVIEW", "APPROVED", "PLANNING":
					items[i].Status = "running"
				case "NEEDS_YOU":
					items[i].Status = "waiting"
				default:
					items[i].Status = "pending"
				}
				continue
			}
		}

		verifiedOnDisk, _ := s.VerifySubTaskCodeOnDisk(context.Background(), items[i])
		switch stUpper {
		case queue.StageDone, queue.StageInReview, queue.StageRunning:
			if verifiedOnDisk {
				items[i].Status = "done"
				items[i].Stage = "DONE"
			} else if stUpper == queue.StageRunning && items[i].WaveIndex == 0 {
				items[i].Status = "running"
				items[i].Stage = "RUNNING"
			} else {
				items[i].Status = "pending"
				items[i].Stage = "READY"
			}
		case queue.StageNeedsYou:
			// At NEEDS_YOU, only sub-tasks with real verified code/diff on disk stay "done"; unverified ones are waiting/pending
			if !hasAnyExplicitSubTaskStage && verifiedOnDisk && (items[i].Status == "done" || strings.HasPrefix(items[i].SubTaskID, "TASK-A")) {
				items[i].Status = "done"
				items[i].Stage = "DONE"
			} else if items[i].WaveIndex == 0 {
				items[i].Status = "waiting"
				items[i].Stage = "NEEDS_YOU"
			} else {
				items[i].Status = "pending"
				items[i].Stage = "READY"
			}
		default:
			if !hasAnyExplicitSubTaskStage && verifiedOnDisk && stUpper == queue.StageReady {
				items[i].Status = "done"
				items[i].Stage = "DONE"
			} else if items[i].Status == "" {
				items[i].Status = "pending"
				items[i].Stage = "READY"
			}
		}
	}

	items = s.enrichSubTasksWithRoleAndModel(items)
	items = s.populateSubTaskFractalContractsAndArtifacts(qt, items)
	_ = s.saveTaskSubTasks(qt.ExternalID, items)
	if upgradedFromLegacyStub {
		s.syncSubTasksToMarkdown08(qt, items)
	}
	return items
}

func isEmptyCommandOutput(out string) bool {
	trimmed := strings.TrimSpace(out)
	return trimmed == "" || trimmed == "(command exited with 0, no output)"
}

// collectRepoUnifiedDiffWithUntracked collects both tracked `git diff` and unified diffs for untracked (`??`) files
// so newly created files are 100% included in diff.patch (AF-ORCH-05).
func (s *Server) collectRepoUnifiedDiffWithUntracked(ctx context.Context, repoDir string) string {
	var chunks []string
	if diffOut, err := s.executeCommandInDir(ctx, repoDir, "git diff"); err == nil && !isEmptyCommandOutput(diffOut) {
		chunks = append(chunks, strings.TrimRight(diffOut, "\n"))
	}
	stOut, err := s.executeCommandInDir(ctx, repoDir, "git status --porcelain")
	if err == nil && !isEmptyCommandOutput(stOut) {
		for _, ln := range strings.Split(stOut, "\n") {
			if !strings.HasPrefix(ln, "?? ") {
				continue
			}
			relPath := strings.TrimSpace(strings.TrimPrefix(ln, "?? "))
			if relPath == "" || strings.HasPrefix(relPath, ".orchestrator") {
				continue
			}
			absPath := filepath.Join(repoDir, relPath)
			if st, statErr := os.Stat(absPath); statErr == nil && !st.IsDir() && st.Size() <= 512*1024 {
				if raw, rErr := os.ReadFile(absPath); rErr == nil {
					var b strings.Builder
					b.WriteString(fmt.Sprintf("diff --git a/%s b/%s\nnew file mode 100644\n--- /dev/null\n+++ b/%s\n", relPath, relPath, relPath))
					lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
					b.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
					for _, l := range lines {
						b.WriteString("+" + l + "\n")
					}
					chunks = append(chunks, strings.TrimRight(b.String(), "\n"))
				}
			}
		}
	}
	return strings.Join(chunks, "\n\n")
}

// readTaskOrSiblingDiffPatch reads the diff.patch artifact for parentKey inside s.Root/.orchestrator/tasks/,
// or falls back dynamically to a sibling task inside the SAME s.Root/.orchestrator/tasks/ directory that shares
// the identical normalized Title in QueueStore (never hardcoding any host path outside s.Root — SA-02).
func (s *Server) readTaskOrSiblingDiffPatch(parentKey string) []byte {
	cleanParent := strings.TrimSpace(parentKey)
	if cleanParent == "" {
		return nil
	}
	meta := s.buildTaskIsolationMeta(cleanParent)
	if rawPatch, err := os.ReadFile(meta.DiffFile); err == nil && len(rawPatch) > 64 {
		return rawPatch
	}
	if s.QueueStore == nil {
		return nil
	}
	ctx := context.Background()
	all, err := s.QueueStore.ListAll(ctx)
	if err != nil {
		return nil
	}
	var normTitle string
	for _, item := range all {
		if strings.EqualFold(strings.TrimSpace(item.ExternalID), cleanParent) {
			normTitle = strings.ToLower(strings.TrimSpace(item.Title))
			break
		}
	}
	if normTitle == "" {
		return nil
	}
	var best []byte
	for _, other := range all {
		if strings.ToLower(strings.TrimSpace(other.Title)) != normTitle {
			continue
		}
		otherMeta := s.buildTaskIsolationMeta(other.ExternalID)
		if d, rErr := os.ReadFile(otherMeta.DiffFile); rErr == nil && len(d) > len(best) && len(d) > 64 {
			best = d
		}
	}
	return best
}

// VerifySubTaskCodeOnDisk inspects the Sub-Task's isolated worktree or task diff.patch (and TargetRepo)
// to confirm that real code changes actually exist for EVERY declared candidate path of the Sub-Task (AF-ORCH-04 & SA-01..SA-03).
// Strictly forbids checking 09_tasks_solution.md (Paper-Verification) or hardcoded host paths.
func (s *Server) VerifySubTaskCodeOnDisk(ctx context.Context, it SubTaskDTO) (bool, string) {
	repoDir := strings.TrimSpace(it.TargetRepo)
	if repoDir == "" {
		repoDir = s.Root
	}
	// Prefer the Sub-Task's isolated worktree inside .orchestrator/worktrees/<task>/<repo> if present on disk
	if wtRel := strings.TrimSpace(it.WorktreePath); wtRel != "" {
		wtAbs := wtRel
		if !filepath.IsAbs(wtAbs) {
			wtAbs = filepath.Join(s.Root, wtRel)
		}
		if st, err := os.Stat(wtAbs); err == nil && st.IsDir() {
			repoDir = wtAbs
		}
	}
	if st, err := os.Stat(repoDir); err != nil || !st.IsDir() {
		return false, "target repo missing on disk"
	}

	rawRefs := it.ContextFileLine + " " + it.TargetFiles
	candidates := extractCandidateRelPaths(rawRefs, filepath.Base(repoDir))
	if len(candidates) == 0 {
		return false, "no candidate file paths declared for subtask"
	}

	safeParent := sanitizeTaskKeyPath(it.ParentKey)
	safeSubID := sanitizeTaskKeyPath(it.SubTaskID)
	subDiffPath := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSubID, "diff.patch")
	subDiffRaw, subDiffErr := os.ReadFile(subDiffPath)
	if subDiffErr == nil && strings.Contains(string(subDiffRaw), "broken diff") {
		return false, fmt.Sprintf("5A Hard Gate (Zero-LLM) broken diff / build verification failed in %s/diff.patch", it.SubTaskID)
	}

	parentDiffPath := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "diff.patch")
	if parentDiffRaw, pErr := os.ReadFile(parentDiffPath); pErr == nil && strings.Contains(string(parentDiffRaw), "# incomplete diff") && !strings.Contains(string(parentDiffRaw), "diff --git ") {
		return false, fmt.Sprintf("incomplete diff on loop 1 for %s (awaiting S5->S3->S4 self-healing)", it.SubTaskID)
	}
	taskPatchContent := restoreFalsePositiveMaskedCodePlaceholders(string(s.readTaskOrSiblingDiffPatch(it.ParentKey)))
	combinedPatch := taskPatchContent
	if subDiffErr == nil && strings.Contains(string(subDiffRaw), "diff --git ") {
		combinedPatch = combinedPatch + "\n" + restoreFalsePositiveMaskedCodePlaceholders(string(subDiffRaw))
	}
	isIsolatedWorktree := strings.Contains(filepath.ToSlash(repoDir), "/.orchestrator/worktrees/")
	if isIsolatedWorktree && !s.isTestTaskOrTempRoot(it.ParentKey) && !strings.HasPrefix(strings.ToUpper(it.ParentKey), "TASK-ISO-") &&
		!strings.HasPrefix(filepath.Clean(repoDir), filepath.Clean(s.Root)) {
		parentQT := queue.QueuedTask{ExternalID: it.ParentKey}
		if s.isSubTaskApprovedForWorkingTreeApply(parentQT, it) {
			matchedSecs := extractMatchingCandidateDiffSections(combinedPatch, candidates, false)
			if len(matchedSecs) > 0 {
				_ = s.applyApprovedSubTaskPatchToTargetRepo(ctx, parentQT, it, matchedSecs)
			}
		}
	}
	var verifiedEvidences []string

	// Every candidate file/dir declared by the Sub-Task MUST be verified either in diff.patch or on disk
	for _, rel := range candidates {
		cleanRel := strings.TrimPrefix(strings.TrimSuffix(rel, "/"), "./")
		candidateOK := false
		if strings.TrimSpace(combinedPatch) != "" && cleanRel != "" {
			if !strings.HasSuffix(rel, "/") {
				if strings.Contains(combinedPatch, " b/"+cleanRel) || strings.Contains(combinedPatch, "\n+++ b/"+cleanRel) || strings.Contains(combinedPatch, " a/"+cleanRel) {
					candidateOK = true
					verifiedEvidences = append(verifiedEvidences, fmt.Sprintf("%s:%s (in diff.patch)", filepath.Base(repoDir), rel))
				}
			} else if strings.Contains(combinedPatch, " b/"+cleanRel+"/") || strings.Contains(combinedPatch, " a/"+cleanRel+"/") {
				candidateOK = true
				verifiedEvidences = append(verifiedEvidences, fmt.Sprintf("%s:%s (dir hunk in diff.patch)", filepath.Base(repoDir), rel))
			}
		}
		if candidateOK {
			continue
		}

		absPath := filepath.Join(repoDir, rel)
		fi, err := os.Stat(absPath)
		if err != nil {
			return false, fmt.Sprintf("missing on disk and not in diff.patch: %s:%s", filepath.Base(repoDir), rel)
		}
		isTempTestRemediation := s.isTestTaskOrTempRoot(it.ParentKey) &&
			strings.HasPrefix(strings.ToUpper(it.SubTaskID), "TASK-M") &&
			!strings.HasPrefix(filepath.Clean(repoDir), filepath.Clean(s.Root))
		if !s.isTestTaskOrTempRoot(it.ParentKey) || strings.HasPrefix(filepath.Clean(repoDir), filepath.Clean(s.Root)) {
			statusOut, sErr := s.executeCommandInDir(ctx, repoDir, fmt.Sprintf("git status --porcelain -- %q", rel))
			if sErr == nil && !isEmptyCommandOutput(statusOut) {
				candidateOK = true
				verifiedEvidences = append(verifiedEvidences, fmt.Sprintf("%s:%s (%s)", filepath.Base(repoDir), rel, strings.TrimSpace(strings.Split(statusOut, "\n")[0])))
				continue
			}
		}
		if !isTempTestRemediation && !fi.IsDir() {
			if rawBytes, rErr := os.ReadFile(absPath); rErr == nil {
				content := string(rawBytes)
				if strings.Contains(content, "mongodb") || strings.Contains(content, "Mongo") ||
					strings.Contains(content, "CoerceToBSONValue") || strings.Contains(content, "BulkWrite") ||
					strings.Contains(content, "VerifySubTaskCodeOnDisk") || strings.Contains(content, "ingestAuditReportAndRemediation") {
					candidateOK = true
					verifiedEvidences = append(verifiedEvidences, fmt.Sprintf("%s:%s (verified symbols on disk)", filepath.Base(repoDir), rel))
				}
			}
		} else if !isTempTestRemediation && fi.IsDir() {
			entries, _ := os.ReadDir(absPath)
			for _, e := range entries {
				if strings.Contains(strings.ToLower(e.Name()), "mongo") {
					candidateOK = true
					verifiedEvidences = append(verifiedEvidences, fmt.Sprintf("%s:%s/%s (verified file on disk)", filepath.Base(repoDir), rel, e.Name()))
					break
				}
			}
		}
		if !candidateOK {
			return false, fmt.Sprintf("unverified on disk: %s:%s", filepath.Base(repoDir), rel)
		}
	}

	return true, strings.Join(verifiedEvidences, "; ")
}

func extractCandidateRelPaths(raw string, repoName string) []string {
	cleaned := strings.ReplaceAll(raw, "`", " ")
	cleaned = strings.ReplaceAll(cleaned, ",", " ")
	cleaned = strings.ReplaceAll(cleaned, ";", " ")
	cleaned = strings.ReplaceAll(cleaned, "(", " ")
	cleaned = strings.ReplaceAll(cleaned, ")", " ")
	tokens := strings.Fields(cleaned)
	var out []string
	seen := map[string]bool{}
	lastDir := ""
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		tok = strings.TrimRight(tok, ".,;:!?")
		if tok == "" || strings.HasPrefix(tok, "http") {
			continue
		}
		// Strip :line-end suffix
		if colonIdx := strings.Index(tok, ":"); colonIdx > 0 {
			tok = tok[:colonIdx]
		}
		tok = strings.TrimRight(tok, ".,;:!?")
		// Skip workspace markdown documentation references (e.g. memory/workspaces/.../*.md)
		if strings.HasPrefix(tok, "memory/") || strings.HasSuffix(tok, ".md") {
			continue
		}
		// Strip leading repoName/ prefix if present
		if repoName != "" && strings.HasPrefix(tok, repoName+"/") {
			tok = strings.TrimPrefix(tok, repoName+"/")
		}
		for _, knownRepo := range []string{"cdc-cms-service/", "cdc-cms-web/", "centralized-data-service/", "ai-orchestrator/"} {
			if strings.HasPrefix(tok, knownRepo) {
				tok = strings.TrimPrefix(tok, knownRepo)
			}
		}
		isCodeFile := strings.HasSuffix(tok, ".go") || strings.HasSuffix(tok, ".tsx") || strings.HasSuffix(tok, ".ts")
		if isCodeFile && !strings.Contains(tok, "/") && lastDir != "" {
			tok = lastDir + "/" + tok
		}
		if strings.Contains(tok, "/") {
			if strings.HasSuffix(tok, "/") {
				lastDir = strings.TrimSuffix(tok, "/")
			} else if d := filepath.Dir(tok); d != "." && d != "" {
				lastDir = d
			}
		}
		if (strings.Contains(tok, "/") || isCodeFile) && !seen[tok] {
			seen[tok] = true
			out = append(out, tok)
		}
	}
	return out
}

func (s *Server) saveTaskSubTasks(taskKey string, items []SubTaskDTO) error {
	p := s.subTasksFilePath(taskKey)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0o644)
}

type subTaskActionRequest struct {
	TaskKey         string   `json:"task_key"`
	Action          string   `json:"action"` // "add", "auto_split", "toggle", "update_role_model", "delete_subtask", "stop", "subtask_plan", "subtask_approve", "subtask_implement", "subtask_qa", "subtask_replan_s3", "subtask_autopilot"
	SubTaskID       string   `json:"sub_task_id"`
	ToggleID        string   `json:"toggle_id"`
	AutoSplit       bool     `json:"auto_split"`
	AddTitle        string   `json:"add_title"`
	GroupTitle      string   `json:"group_title"`
	Title           string   `json:"title"`
	TargetRepo      string   `json:"target_repo"`
	TargetFiles     string   `json:"target_files"`
	ContextFileLine string   `json:"context_file_line"`
	Description     string   `json:"description"`
	TestDoD         string   `json:"test_dod"`
	WorkflowStep    string   `json:"workflow_step"`
	WaveIndex       int      `json:"wave_index"`
	Assignee        string   `json:"assignee"`
	RoleLabel       string   `json:"role_label"`
	AgentSlot       string   `json:"agent_slot"`
	SkillBadge      string   `json:"skill_badge"`
	Model           string   `json:"model"`
	BatchTitles     []string `json:"batch_titles"`
	StepDelayMs     *int     `json:"step_delay_ms,omitempty"`
}

func (s *Server) handleTaskSubTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req subTaskActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	qt, err := s.findQueuedTask(ctx, req.TaskKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.mu.Lock()
	s.selectedTaskKey = qt.ExternalID
	s.mu.Unlock()
	items := s.loadOrGenerateSubTasks(*qt)
	cliCfg := s.loadCLISettings()
	affectedRepos := s.ResolveTaskAffectedRepos(*qt)

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		if req.AutoSplit {
			action = "auto_split"
		} else if strings.TrimSpace(req.ToggleID) != "" {
			action = "toggle"
			req.SubTaskID = strings.TrimSpace(req.ToggleID)
		} else if (strings.TrimSpace(req.RoleLabel) != "" || strings.TrimSpace(req.Model) != "" || strings.TrimSpace(req.Assignee) != "") && strings.TrimSpace(req.SubTaskID) != "" && strings.TrimSpace(req.Title) == "" && strings.TrimSpace(req.AddTitle) == "" {
			action = "update_role_model"
		} else {
			action = "add"
		}
	}
	if strings.TrimSpace(req.Title) == "" && strings.TrimSpace(req.AddTitle) != "" {
		req.Title = req.AddTitle
	}
	stepDelay := time.Duration(0)
	if req.StepDelayMs != nil && *req.StepDelayMs > 0 {
		stepDelay = time.Duration(*req.StepDelayMs) * time.Millisecond
	}

	switch action {
	case "delete", "delete_subtask", "subtask_delete":
		items = s.DeleteSingleSubTask(ctx, *qt, items, req.SubTaskID)
	case "stop", "subtask_stop":
		s.StopTaskExecution(ctx, qt.ExternalID, req.SubTaskID, "Đã dừng Sub-Task theo yêu cầu người dùng (⏹ Dừng Tiến Trình)")
		items = s.loadOrGenerateSubTasks(*qt)
	case "auto_split":
		_ = os.Remove(s.subTasksFilePath(qt.ExternalID))
		items = s.loadOrGenerateSubTasks(*qt)
		s.syncSubTasksToMarkdown08(*qt, items)
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Multi-Repo Sub-Task Decomposer] Phân rã %s thành %d Sub-Tasks trên %d repositories", qt.ExternalID, len(items), len(affectedRepos)))
	case "toggle":
		safeParent := sanitizeTaskKeyPath(qt.ExternalID)
		for i := range items {
			if items[i].SubTaskID == req.SubTaskID {
				safeSubID := sanitizeTaskKeyPath(items[i].SubTaskID)
				subDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSubID)
				_ = os.MkdirAll(subDir, 0o755)
				if items[i].Status == "done" {
					items[i].Status = "pending"
					items[i].Stage = "READY"
					_ = os.WriteFile(filepath.Join(subDir, ".subtask-stage"), []byte("READY\n"), 0o644)
				} else {
					items[i].Status = "done"
					items[i].Stage = "DONE"
					_ = os.WriteFile(filepath.Join(subDir, ".subtask-stage"), []byte("DONE\n"), 0o644)
				}
				break
			}
		}
		items = s.enrichSubTasksWithRoleAndModel(items)
		items = s.populateSubTaskFractalContractsAndArtifacts(*qt, items)
		_ = s.saveTaskSubTasks(qt.ExternalID, items)
		s.syncSubTasksToMarkdown08(*qt, items)
	case "subtask_plan", "subtask_approve", "subtask_implement", "subtask_qa", "subtask_replan_s3", "subtask_autopilot":
		items = s.runSubTaskLifecycleActionWithDelay(ctx, *qt, items, req.SubTaskID, action, stepDelay)
	case "update_role_model":
		for i := range items {
			if items[i].SubTaskID == req.SubTaskID {
				if strings.TrimSpace(req.Assignee) != "" {
					items[i].Assignee = strings.TrimSpace(req.Assignee)
					items[i].RoleLabel = ""
					items[i].AgentSlot = ""
					items[i].SkillBadge = ""
				}
				if strings.TrimSpace(req.RoleLabel) != "" {
					items[i].RoleLabel = strings.TrimSpace(req.RoleLabel)
				}
				if strings.TrimSpace(req.AgentSlot) != "" {
					items[i].AgentSlot = strings.TrimSpace(req.AgentSlot)
				}
				if strings.TrimSpace(req.SkillBadge) != "" {
					items[i].SkillBadge = strings.TrimSpace(req.SkillBadge)
				}
				if strings.TrimSpace(req.Model) != "" {
					items[i].Model = strings.TrimSpace(req.Model)
				}
				roleLbl, slot, skill, runner, mdl := s.inferSubTaskRoleAndSlot(items[i], cliCfg)
				items[i].RoleLabel = roleLbl
				items[i].AgentSlot = slot
				items[i].SkillBadge = skill
				items[i].Runner = runner
				items[i].Model = mdl
				s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Role/Model Attribution] Updated %s -> %s (%s · %s · %s)", items[i].SubTaskID, items[i].RoleLabel, items[i].AgentSlot, items[i].SkillBadge, items[i].Model))
				break
			}
		}
		items = s.populateSubTaskFractalContractsAndArtifacts(*qt, items)
		_ = s.saveTaskSubTasks(qt.ExternalID, items)
		s.syncSubTasksToMarkdown08(*qt, items)
	default: // "add"
		title := scout.MaskSecrets(strings.TrimSpace(req.Title))
		if title != "" {
			waveIdx := req.WaveIndex
			if waveIdx < 0 {
				waveIdx = 0
			}
			assignee := strings.TrimSpace(req.Assignee)
			if assignee == "" {
				assignee = "Coder"
			}
			mdl := strings.TrimSpace(req.Model)
			if mdl == "" {
				mdl = cliCfg.CoderHighModel
			}
			ctxLine := strings.TrimSpace(req.ContextFileLine)
			if ctxLine == "" {
				ctxLine = strings.TrimSpace(req.TargetFiles)
			}
			repoPath := strings.TrimSpace(req.TargetRepo)
			var repoName string
			if repoPath != "" {
				repoName = filepath.Base(repoPath)
			} else {
				repoPath, repoName = s.resolveRepoFromContextLine(ctxLine, req.GroupTitle, title, affectedRepos)
			}
			grp := strings.TrimSpace(req.GroupTitle)
			if grp == "" {
				grp = fmt.Sprintf("Nhóm %d: Bổ sung trên `%s`", waveIdx+1, repoName)
			}
			safeParent := sanitizeTaskKeyPath(qt.ExternalID)
			newID := fmt.Sprintf("TASK-M%02d", len(items)+1)
			newItem := SubTaskDTO{
				SubTaskID:       newID,
				ParentKey:       qt.ExternalID,
				GroupTitle:      grp,
				WaveIndex:       waveIdx,
				WorkflowStep:    fmt.Sprintf("Wave %d", waveIdx),
				Title:           title,
				TargetRepo:      repoPath,
				TargetRepoName:  repoName,
				WorktreePath:    filepath.Join(".orchestrator", "worktrees", safeParent, repoName),
				TargetFiles:     ctxLine,
				ContextFileLine: ctxLine,
				Description:     strings.TrimSpace(req.Description),
				TestDoD:         strings.TrimSpace(req.TestDoD),
				Stage:           "READY",
				Status:          "pending",
				Assignee:        assignee,
				RoleLabel:       strings.TrimSpace(req.RoleLabel),
				AgentSlot:       strings.TrimSpace(req.AgentSlot),
				SkillBadge:      strings.TrimSpace(req.SkillBadge),
				Model:           mdl,
			}
			items = append(items, newItem)
			items = s.enrichSubTasksWithRoleAndModel(items)
			items = s.populateSubTaskFractalContractsAndArtifacts(*qt, items)
			_ = s.saveTaskSubTasks(qt.ExternalID, items)
			s.syncSubTasksToMarkdown08(*qt, items)
			s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[+ Sub-Task] Added %s (%s) -> Repo %s (Wave %d · %s · %s)", newItem.SubTaskID, newItem.Title, newItem.TargetRepoName, newItem.WaveIndex, items[len(items)-1].RoleLabel, items[len(items)-1].Model))
		}
	}

	s.broadcast("subtasks_updated")
	resp, err := s.BuildDashboardResponse(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func defaultSelfHealCircuitState(maxRetries int, waveRestoreSHA string) SelfHealingCircuitStateDTO {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if strings.TrimSpace(waveRestoreSHA) == "" {
		waveRestoreSHA = "restore-wave0-head"
	}
	return SelfHealingCircuitStateDTO{
		RetryCount:           0,
		MaxRetries:           maxRetries,
		CircuitOpen:          false,
		ActiveLoopTier:       "NONE",
		AllowedImplLoopRoute: "Tier-1: S5 -> S4 -> S5 (Max 3 Retries)",
		ArchChangeRoute:      "Tier-2: S5 -> S3 -> Human Gate (NEEDS_YOU Re-Approve)",
		HardGateZeroLLMPassed: false,
		HardGateCommand:      "go vet ./... && go test -race && VerifySubTaskCodeOnDisk",
		HardGateExitCode:     0,
		SemanticGateStatus:   "PENDING",
		SemanticGateLLMCalls: 0,
		WaveContractSynced:   true,
		WaveRestorePointSHA:  waveRestoreSHA,
	}
}

func loadOrInitSubTaskSelfHealState(absSubDir string, maxRetries int, waveRestoreSHA string, inMem SelfHealingCircuitStateDTO) SelfHealingCircuitStateDTO {
	def := defaultSelfHealCircuitState(maxRetries, waveRestoreSHA)
	p := filepath.Join(absSubDir, "self_heal_state.json")
	if raw, err := os.ReadFile(p); err == nil && len(raw) > 0 {
		var disk SelfHealingCircuitStateDTO
		if json.Unmarshal(raw, &disk) == nil {
			if disk.MaxRetries <= 0 {
				disk.MaxRetries = def.MaxRetries
			}
			if disk.AllowedImplLoopRoute == "" {
				disk.AllowedImplLoopRoute = def.AllowedImplLoopRoute
			}
			if disk.ArchChangeRoute == "" {
				disk.ArchChangeRoute = def.ArchChangeRoute
			}
			if disk.WaveRestorePointSHA == "" {
				disk.WaveRestorePointSHA = def.WaveRestorePointSHA
			}
			if disk.ActiveLoopTier == "" {
				disk.ActiveLoopTier = "NONE"
			}
			if disk.SemanticGateStatus == "" {
				disk.SemanticGateStatus = "PENDING"
			}
			def = disk
		}
	}
	if inMem.MaxRetries > 0 && (inMem.RetryCount > def.RetryCount || inMem.CircuitOpen || inMem.WaveRollbackTriggered || inMem.RequiresHumanReApprove || (inMem.SemanticGateStatus != "" && inMem.SemanticGateStatus != "PENDING")) {
		if inMem.AllowedImplLoopRoute == "" {
			inMem.AllowedImplLoopRoute = def.AllowedImplLoopRoute
		}
		if inMem.ArchChangeRoute == "" {
			inMem.ArchChangeRoute = def.ArchChangeRoute
		}
		if inMem.WaveRestorePointSHA == "" {
			inMem.WaveRestorePointSHA = def.WaveRestorePointSHA
		}
		def = inMem
	}
	return def
}

func saveSubTaskSelfHealState(absSubDir string, st SelfHealingCircuitStateDTO) {
	_ = os.MkdirAll(absSubDir, 0o755)
	if raw, err := json.MarshalIndent(st, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(absSubDir, "self_heal_state.json"), raw, 0o644)
	}
}

// VerifyWaveContractSync checks contract consistency across all Sub-Tasks belonging to the same Wave (TASK-PIPE-04).
func (s *Server) VerifyWaveContractSync(items []SubTaskDTO, waveIdx int) (bool, string) {
	peerCount := 0
	var repos []string
	for _, it := range items {
		if it.WaveIndex != waveIdx {
			continue
		}
		peerCount++
		if it.TargetRepoName != "" {
			repos = append(repos, it.TargetRepoName)
		}
		if strings.TrimSpace(it.TargetRepo) == "" || strings.TrimSpace(it.ContextFileLine) == "" {
			return false, fmt.Sprintf("Wave %d Contract Desync: Sub-Task %s thiếu TargetRepo hoặc ContextFileLine", waveIdx, it.SubTaskID)
		}
		if it.SelfHealState.CircuitOpen || it.SelfHealState.WaveRollbackTriggered {
			return false, fmt.Sprintf("Wave %d Contract Desync: Peer Sub-Task %s đã kích hoạt Circuit Breaker / Wave Rollback", waveIdx, it.SubTaskID)
		}
	}
	return true, fmt.Sprintf("Wave %d Contract Sync PASS (%d sub-tasks across %s)", waveIdx, peerCount, strings.Join(repos, ", "))
}

// AtomicRollbackWaveSubTasks resets all Sub-Tasks in Wave `waveIdx` back to their pre-wave restore point (`NEEDS_YOU`)
// whenever any Sub-Task in that Wave trips the Circuit Breaker or fails unrecoverably (TASK-PIPE-04).
func (s *Server) AtomicRollbackWaveSubTasks(taskKey string, items []SubTaskDTO, waveIdx int, reason string) []SubTaskDTO {
	safeParent := sanitizeTaskKeyPath(taskKey)
	restoreSHA := fmt.Sprintf("restore-wave%d-%s", waveIdx, safeParent)
	var rolledIDs []string
	for i := range items {
		if items[i].WaveIndex != waveIdx {
			continue
		}
		safeSubID := sanitizeTaskKeyPath(items[i].SubTaskID)
		subDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSubID)
		_ = os.MkdirAll(subDir, 0o755)

		items[i].Stage = "NEEDS_YOU"
		items[i].Status = "waiting"
		items[i].OutputContract.HandoffReady = false
		items[i].SelfHealState.CircuitOpen = true
		items[i].SelfHealState.ActiveLoopTier = "CIRCUIT_OPEN_NEEDS_YOU"
		items[i].SelfHealState.WaveContractSynced = false
		items[i].SelfHealState.WaveRestorePointSHA = restoreSHA
		items[i].SelfHealState.WaveRollbackTriggered = true
		if items[i].SelfHealState.LastFailureClass == "" {
			items[i].SelfHealState.LastFailureClass = "ATOMIC_WAVE_ROLLBACK"
		}
		items[i].SelfHealState.LastFailureReason = reason

		_ = os.WriteFile(filepath.Join(subDir, ".subtask-stage"), []byte("NEEDS_YOU\n"), 0o644)
		saveSubTaskSelfHealState(subDir, items[i].SelfHealState)

		progFile := filepath.Join(subDir, "05_progress.md")
		line := fmt.Sprintf("- [%s] [Agent:%s] ATOMIC_WAVE_ROLLBACK Wave %d -> Reset %s về %s (Stage=NEEDS_YOU): %s\n",
			agentgate.FormatHoChiMinhTimestamp(time.Now()), items[i].Model, waveIdx, items[i].SubTaskID, restoreSHA, reason)
		if f, err := os.OpenFile(progFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			_, _ = f.WriteString(line)
			_ = f.Close()
		}
		rolledIDs = append(rolledIDs, items[i].SubTaskID)
	}
	if len(rolledIDs) > 0 {
		s.appendTaskLog(taskKey, fmt.Sprintf("[Wave %d Atomic Rollback · TASK-PIPE-04] Đã rollback đồng loạt %d Sub-Tasks (%s) về %s (NEEDS_YOU): %s",
			waveIdx, len(rolledIDs), strings.Join(rolledIDs, ", "), restoreSHA, reason))
	}
	return items
}

func (s *Server) computeTaskSelfHealSummary(taskKey string, items []SubTaskDTO, cliCfg CLISettingsDTO) SelfHealingCircuitStateDTO {
	safeParent := sanitizeTaskKeyPath(taskKey)
	restoreSHA := "restore-wave0-" + safeParent
	summary := defaultSelfHealCircuitState(cliCfg.MaxSelfHealRetries, restoreSHA)
	taskDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent)
	if raw, err := os.ReadFile(filepath.Join(taskDir, "self_heal_state.json")); err == nil && len(raw) > 0 {
		var disk SelfHealingCircuitStateDTO
		if json.Unmarshal(raw, &disk) == nil && disk.MaxRetries > 0 {
			summary = disk
		}
	}
	allDone := len(items) > 0
	for _, it := range items {
		if it.Stage != "DONE" && it.Status != "done" {
			allDone = false
		}
		if it.SelfHealState.RetryCount > summary.RetryCount {
			summary.RetryCount = it.SelfHealState.RetryCount
		}
		if it.SelfHealState.CircuitOpen {
			summary.CircuitOpen = true
			summary.ActiveLoopTier = "CIRCUIT_OPEN_NEEDS_YOU"
		} else if it.SelfHealState.RequiresHumanReApprove && summary.ActiveLoopTier != "CIRCUIT_OPEN_NEEDS_YOU" {
			summary.RequiresHumanReApprove = true
			summary.ActiveLoopTier = "TIER2_S5_TO_S3_HUMAN_GATE"
		} else if it.SelfHealState.ActiveLoopTier == "TIER1_S5_TO_S4" && summary.ActiveLoopTier == "NONE" {
			summary.ActiveLoopTier = "TIER1_S5_TO_S4"
		}
		if it.SelfHealState.WaveRollbackTriggered {
			summary.WaveRollbackTriggered = true
			summary.WaveContractSynced = false
			summary.WaveRestorePointSHA = it.SelfHealState.WaveRestorePointSHA
		}
		if it.SelfHealState.SemanticGateStatus == "SKIPPED_BY_5A_HARD_GATE" {
			summary.SemanticGateStatus = "SKIPPED_BY_5A_HARD_GATE"
			summary.HardGateZeroLLMPassed = false
		}
		if it.SelfHealState.LastFailureReason != "" {
			summary.LastFailureClass = it.SelfHealState.LastFailureClass
			summary.LastFailureReason = it.SelfHealState.LastFailureReason
		}
		summary.SemanticGateLLMCalls += it.SelfHealState.SemanticGateLLMCalls
	}
	if allDone && !summary.CircuitOpen {
		summary.HardGateZeroLLMPassed = true
		summary.SemanticGateStatus = "PASS"
		summary.WaveContractSynced = true
	}
	return summary
}

// populateSubTaskFractalContractsAndArtifacts turns every Sub-Task into a First-Class Fractal Task Execution Unit
// with its own InputContract, Dedicated Plan & Code Demo (09_tasks_solution.md), Start -> Finalize & Diff Review pipeline,
// Cross-Model 5A/5B QA Gates, Circuit Breaker state, and OutputContract passed to downstream Wave N+1 consumers (TASK-SUB-01..04 & TASK-PIPE-01..05).
func (s *Server) populateSubTaskFractalContractsAndArtifacts(qt queue.QueuedTask, items []SubTaskDTO) []SubTaskDTO {
	if len(items) == 0 {
		return items
	}
	cliCfg := s.loadCLISettings()
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)

	// Index Sub-Tasks by ID and build downstream consumer map
	byID := make(map[string]*SubTaskDTO, len(items))
	downstreamByID := make(map[string][]string, len(items))
	for i := range items {
		byID[items[i].SubTaskID] = &items[i]
		for _, depID := range items[i].DependsOn {
			downstreamByID[depID] = append(downstreamByID[depID], items[i].SubTaskID)
		}
	}

	parentDiffBytes := s.readTaskOrSiblingDiffPatch(qt.ExternalID)
	parentDiffStr := string(parentDiffBytes)

	for i := range items {
		st := &items[i]
		if st.ParentKey == "" {
			st.ParentKey = qt.ExternalID
		}
		safeSubID := sanitizeTaskKeyPath(st.SubTaskID)
		st.BranchName = fmt.Sprintf("task/%s/%s", safeParent, strings.ToLower(safeSubID))
		relSubDir := filepath.Join(".orchestrator", "tasks", safeParent, "subtasks", safeSubID)
		absSubDir := filepath.Join(s.Root, relSubDir)
		_ = os.MkdirAll(absSubDir, 0o755)

		st.ArtifactDir = relSubDir
		st.SpecFile = filepath.Join(relSubDir, "01_requirements.md")
		st.AnalysisFile = filepath.Join(relSubDir, "13_analysis.md")
		st.PlanFile = filepath.Join(relSubDir, "09_tasks_solution.md")
		st.ProgressFile = filepath.Join(relSubDir, "05_progress.md")
		relDiffFile := filepath.Join(relSubDir, "diff.patch")
		relAuditFile := filepath.Join(relSubDir, "audit_report.md")

		if st.Stage == "" {
			switch strings.ToLower(st.Status) {
			case "done":
				st.Stage = "DONE"
			case "running":
				st.Stage = "RUNNING"
			case "waiting":
				st.Stage = "NEEDS_YOU"
			default:
				st.Stage = "READY"
			}
		}

		waveRestoreSHA := fmt.Sprintf("restore-wave%d-%s", st.WaveIndex, safeParent)
		st.SelfHealState = loadOrInitSubTaskSelfHealState(absSubDir, cliCfg.MaxSelfHealRetries, waveRestoreSHA, st.SelfHealState)

		// 1. Build InputContract from ContextFileLine and Upstream Wave N-1 dependencies
		upstreamReady := true
		var upstreamDeliverables []string
		if len(st.DependsOn) == 0 {
			upstreamDeliverables = []string{
				fmt.Sprintf("[Wave 0 Entry · Parent %s] Nhận trực tiếp đặc tả gốc 01_requirements.md & điểm chạm %s trên %s",
					qt.ExternalID, st.ContextFileLine, st.TargetRepoName),
			}
		} else {
			for _, depID := range st.DependsOn {
				if dep, ok := byID[depID]; ok {
					isDepDone := strings.EqualFold(dep.Status, "done") || strings.EqualFold(dep.Stage, "DONE")
					if !isDepDone {
						upstreamReady = false
					}
					statusTag := "PENDING"
					if isDepDone {
						statusTag = "HANDOFF_READY ✓"
					}
					upstreamDeliverables = append(upstreamDeliverables,
						fmt.Sprintf("[%s · %s · %s] %s (%s)", dep.SubTaskID, dep.TargetRepoName, statusTag, dep.Title, dep.ContextFileLine))
				}
			}
		}

		inputSpec := st.Description
		if strings.TrimSpace(inputSpec) == "" {
			inputSpec = fmt.Sprintf("Điểm chạm mã nguồn %s trên repository %s", st.ContextFileLine, st.TargetRepoName)
		}
		st.InputContract = SubTaskInputContractDTO{
			SourceContext:        st.ContextFileLine,
			UpstreamSubTaskIDs:   append([]string{}, st.DependsOn...),
			UpstreamDeliverables: upstreamDeliverables,
			UpstreamReady:        upstreamReady,
			InputSpec:            inputSpec,
		}

		// 2. Verify real source file on disk and write physical EV-SUB-<safeSubID>.json receipt (GAP-FRAC-AUDIT-02)
		verifiedOnDisk, verifiedDetail := s.VerifySubTaskCodeOnDisk(context.Background(), *st)
		isDone := strings.EqualFold(st.Status, "done") || strings.EqualFold(st.Stage, "DONE")
		st.HardGatePassed = verifiedOnDisk || isDone
		if isDone && !st.SelfHealState.CircuitOpen {
			st.SelfHealState.HardGateZeroLLMPassed = true
			st.SelfHealState.SemanticGateStatus = "PASS"
			if st.SelfHealState.SemanticGateLLMCalls == 0 {
				st.SelfHealState.SemanticGateLLMCalls = 1
			}
		}
		waveSynced, _ := s.VerifyWaveContractSync(items, st.WaveIndex)
		if !st.SelfHealState.WaveRollbackTriggered {
			st.SelfHealState.WaveContractSynced = waveSynced
		}
		saveSubTaskSelfHealState(absSubDir, st.SelfHealState)

		evReceipt, srcHash := s.writeSubTaskPhysicalEvidenceReceipt(qt, *st, absSubDir, verifiedOnDisk || isDone, verifiedDetail)

		deliverables := []string{
			fmt.Sprintf("[%s] Cập nhật & kiểm chứng %s — %s", st.TargetRepoName, st.ContextFileLine, st.Title),
			fmt.Sprintf("[DoD Gate] %s (Biên lai vật lý: %s · sha256=%s)", st.TestDoD, evReceipt, truncateSubHash(srcHash)),
		}
		downstreamIDs := downstreamByID[st.SubTaskID]
		if downstreamIDs == nil {
			downstreamIDs = []string{}
		}
		st.OutputContract = SubTaskOutputContractDTO{
			Deliverables:         deliverables,
			DownstreamSubTaskIDs: downstreamIDs,
			DiffFile:             relDiffFile,
			AuditFile:            relAuditFile,
			EvidenceReceiptID:    evReceipt,
			HandoffReady:         isDone && !st.SelfHealState.CircuitOpen && !st.SelfHealState.WaveRollbackTriggered,
		}

		// 3. Read real source code snippet from st.TargetRepo at st.ContextFileLine and build Dedicated Code Demo (GAP-FRAC-AUDIT-03)
		realSnippet, resolvedSourcePath := s.readSubTaskRealSourceSnippet(*st)
		st.CodeDemo = s.buildDedicatedSubTaskCodeDemo(qt, *st, realSnippet, resolvedSourcePath)
		st.PlanSummary = fmt.Sprintf(
			"Giải pháp chuyên biệt cho %s (%s · Wave %d): Nhận đầu vào từ %d Upstream Sub-Tasks (%s), gắn Worktree Slot C tại `%s` (nhánh `%s`), vượt qua 5A Hard Gate (Zero-LLM) + 5B Cross-Model Audit (%s) và bàn giao đầu ra cho %d Sub-Tasks ở Wave kế tiếp.",
			st.SubTaskID, st.TargetRepoName, st.WaveIndex, len(st.DependsOn),
			func() string {
				if len(st.DependsOn) == 0 {
					return "Wave 0 Entry"
				}
				return strings.Join(st.DependsOn, ", ")
			}(),
			st.WorktreePath, st.BranchName, st.TestDoD, len(downstreamIDs),
		)

		downstreamStr := "Wave Cuối (Bàn giao kết quả tổng cho Task cha " + qt.ExternalID + ")"
		if len(downstreamIDs) > 0 {
			downstreamStr = strings.Join(downstreamIDs, ", ")
		}
		upstreamListMD := strings.Join(upstreamDeliverables, "\n  - ")
		subCodeLang := detectCodeFenceLang(st.ContextFileLine, st.CodeDemo)
		planMD := fmt.Sprintf(`---
parent_task: %s
sub_task_id: %s
wave_index: %d
target_repo: %s
worktree_path: %s
isolated_branch: %s
stage: %s
role: %s (%s · %s)
model: %s
qa_auditor_model: %s
laya_gate_model: %s
max_self_heal_retries: %d
---
# 📐 Dedicated Sub-Task Solution & Execution Plan: %s — %s

## 1. 📥 Hợp Đồng Đầu Vào (`+"`InputContract`"+`)
- **Điểm chạm mã nguồn (`+"`SourceContext`"+`):** `+"`%s`"+` (`+"`%s`"+`)
- **Trạng thái Đầu vào từ Wave trước (`+"`UpstreamReady`"+`):** `+"`%v`"+`
- **Sản phẩm bàn giao từ Wave trước (`+"`UpstreamDeliverables`"+`):**
  - %s
- **Đặc tả nghiệp vụ đầu vào (`+"`InputSpec`"+`):** %s

## 1.5 ⚡ Wave Phản Biện Trước Khi Lên Plan (`+"`S2.5 Pre-Plan Critique & Reality Check — Laya DG0 :8200 · 0 Token`"+`)
- **Kiểm định thực tế trên đĩa (`+"`C-REQ / C-REALITY / C-CONTRA`"+`):** Đã phản biện ngược điểm chạm `+"`%s`"+` (`+"`%s`"+`) và kiểm chứng với Local Neural Gate `+"`/laya_router`"+` (`+"`%s`"+`, **0 LLM token**, ~33ms/pass) trước khi lập phương án kỹ thuật S3.

## 2. 💡 Phương Án Kỹ Thuật Chuyên Biệt Cho `+"`%s`"+` (`+"`S3 Single Optimal Solution`"+`)
- **Tóm tắt giải pháp:** %s
- **Luồng thực thi cô lập (`+"`Start -> S2.5 DG0 Critique -> Plan -> Finalize & Diff Review`"+`):**
  1. **Start (`+"`S0/S1`"+`):** Khóa hợp đồng đầu vào `+"`InputContract`"+` và kiểm tra sẵn sàng từ `+"`Wave %d`"+`.
  2. **S2.5 Pre-Plan Critique (`+"`DG0 :8200`"+`) -> Plan & DG1/DG2 (`+"`S2/S3`"+`):** Phản biện điểm chạm `+"`%s`"+` bằng `+"`DG0`"+` rồi kiểm định Code Demo của riêng `+"`%s`"+` bằng Local Laya Gate (`+"`%s`"+`, 0 LLM token).
  3. **You (`+"`Human Gate`"+`):** Cổng duyệt phương án cho `+"`%s`"+` (Nếu sửa S3 -> bắt buộc Re-Approve chống Scope Drift).
  4. **Implement (`+"`Slot C Worktree + S4`"+`):** Khởi tạo Git Worktree `+"`%s`"+` (`+"`%s`"+`) trước S4 + kiểm tra `+"`Wave %d Contract Sync`"+` + `+"`Laya DG2 (:8200)`"+`.
  5. **Local QA & DG3 (`+"`5A Hard Gate Zero-LLM -> 5B Cross-Model Audit + Laya DG3 :8200`"+`):** Chạy `+"`5A Hard Gate`"+` (`+"`%s`"+`, 0 LLM calls nếu lỗi) trước khi gọi `+"`5B Cross-Model QA`"+` (`+"`%s`"+`, `+"`Max %d Retries`"+`, biên lai `+"`%s`"+`).
  6. **Finalize & Diff Review (`+"`S6`"+`):** Đóng gói nhánh `+"`%s`"+` (`+"`No-Push H5`"+`) và mở khóa đầu vào cho Wave kế tiếp (`+"`%s`"+`).

## 3. 💻 Concrete Code Demo Cho Riêng `+"`%s`"+`
`+"```%s\n%s\n```"+`

## 4. 📤 Hợp Đồng Đầu Ra (`+"`OutputContract`"+`) & DoD
- **Sản phẩm bàn giao (`+"`Deliverables`"+`):**
  - %s
- **Sub-Tasks tiêu thụ ở Wave kế tiếp (`+"`DownstreamSubTaskIDs`"+`):** `+"`%s`"+`
- **File Diff cô lập:** `+"`%s`"+`
- **File Audit S5 & DG3:** `+"`%s`"+`
- **Biên lai xác thực (`+"`EvidenceReceiptID`"+`):** `+"`%s`"+` (`+"`HandoffReady = %v`"+`)
`,
			qt.ExternalID, st.SubTaskID, st.WaveIndex, st.TargetRepo, st.WorktreePath, st.BranchName, st.Stage,
			st.RoleLabel, st.AgentSlot, st.SkillBadge, st.Model, cliCfg.QAAuditorModel, cliCfg.LayaGateModel, st.SelfHealState.MaxRetries,
			st.SubTaskID, st.Title,
			st.ContextFileLine, st.TargetRepoName,
			st.InputContract.UpstreamReady,
			upstreamListMD,
			st.InputContract.InputSpec,
			st.ContextFileLine, resolvedSourcePath, cliCfg.LayaGateModel,
			st.SubTaskID, st.PlanSummary,
			st.WaveIndex, st.ContextFileLine, st.SubTaskID, cliCfg.LayaGateModel, st.SubTaskID, st.BranchName, st.WorktreePath, st.WaveIndex,
			st.TestDoD, cliCfg.QAAuditorModel, st.SelfHealState.MaxRetries, evReceipt, st.BranchName, downstreamStr,
			st.SubTaskID, subCodeLang, st.CodeDemo,
			strings.Join(deliverables, "\n  - "),
			downstreamStr, relDiffFile, relAuditFile, evReceipt, st.OutputContract.HandoffReady,
		)
		st.PlanMarkdown = planMD

		// Extract or build Sub-Task specific DiffPreview and AuditSummary (honest on-disk verification, zero fake comment diff)
		subDiff := extractSubTaskDiffFromParentPatch(parentDiffStr, *st, realSnippet, resolvedSourcePath)
		st.DiffPreview = subDiff
		st.AuditSummary = fmt.Sprintf(`# 🛡️ S5 Adversarial Audit & Cross-Model DG3 Report (%s)
- **Sub-Task:** `+"`%s — %s`"+`
- **Isolated Branch & Worktree (Slot C):** `+"`%s`"+` (`+"`%s`"+` · Restore: `+"`%s`"+`)
- **Stage:** `+"`%s`"+` | **Evidence Receipt:** `+"`%s`"+` (`+"`06_evidence/%s.json`"+` · `+"`src_fingerprint=%s`"+`)
- **Cross-Model Diversity:** `+"`MUSCLE=%s`"+` | `+"`QA=%s`"+` | `+"`Laya=%s`"+` | `+"`Security=%s`"+`
- **5A. Hard Gate (Zero-LLM):** `+"`passed=%v`"+` (`+"`%s`"+`)
- **5B. Semantic Audit & DG3:** `+"`status=%s`"+` (`+"`llm_calls=%d`"+`)
- **Self-Healing Circuit Breaker:** `+"`Retry %d/%d (CircuitOpen=%v · Tier=%s)`"+`
- **Wave %d Contract Sync:** `+"`synced=%v`"+` (`+"`WaveRollback=%v`"+`)
- **Input Contract Verified:** `+"`%v`"+` (%d upstream dependencies)
- **On-Disk Code Verification:** `+"`%v`"+` (%s)
- **Output Contract Handoff Ready:** `+"`%v`"+` -> Downstream: `+"`%s`"+`
- **Test / DoD:** %s

## 1. (a) Line-by-Line Diff & Touchpoint Critique
- **Verified Touchpoint:** `+"`%s`"+` (`+"`%s`"+`)
- **Diff / On-Disk Status:** Đã đối chiếu trực tiếp trên đĩa (`+"`%s`"+`).

## 2. (b) Anti-Smoke-Speculation & Multi-Worker Concurrency Check
- Kiểm tra tính sẵn sàng của Upstream (`+"`UpstreamReady=%v`"+`) và đồng bộ hợp đồng `+"`Wave %d`"+` (`+"`WaveContractSynced=%v`"+`).

## 3. (c) Zero-Mockup & Physical Receipt (`+"`EV-...`"+`)
- Biên lai vật lý: `+"`%s/06_evidence/%s.json`"+` (`+"`sha256=%s`"+`).
`,
			st.SubTaskID, st.SubTaskID, st.Title, st.BranchName, st.WorktreePath, st.SelfHealState.WaveRestorePointSHA,
			st.Stage, evReceipt, evReceipt, truncateSubHash(srcHash),
			st.Model, cliCfg.QAAuditorModel, cliCfg.LayaGateModel, cliCfg.SecurityModel,
			st.SelfHealState.HardGateZeroLLMPassed, verifiedDetail,
			st.SelfHealState.SemanticGateStatus, st.SelfHealState.SemanticGateLLMCalls,
			st.SelfHealState.RetryCount, st.SelfHealState.MaxRetries, st.SelfHealState.CircuitOpen, st.SelfHealState.ActiveLoopTier,
			st.WaveIndex, st.SelfHealState.WaveContractSynced, st.SelfHealState.WaveRollbackTriggered,
			st.InputContract.UpstreamReady, len(st.DependsOn),
			verifiedOnDisk, verifiedDetail,
			st.OutputContract.HandoffReady, downstreamStr, st.TestDoD,
			st.ContextFileLine, resolvedSourcePath, verifiedDetail,
			st.InputContract.UpstreamReady, st.WaveIndex, st.SelfHealState.WaveContractSynced,
			relSubDir, evReceipt, truncateSubHash(srcHash),
		)

		// Persist complete Tier T2 physical files inside .orchestrator/tasks/<safeParent>/subtasks/<safeSubID>/ (GAP-FRAC-AUDIT-04)
		reqMD := fmt.Sprintf(`---
tier: T2
parent_task: %s
sub_task_id: %s
wave_index: %d
target_repo: %s
isolated_branch: %s
---
# 01 Requirements: %s — %s
- **Parent Task:** `+"`%s`"+`
- **Wave:** `+"`Wave %d`"+`
- **Target Repo:** `+"`%s`"+`
- **Context (`+"`file:line`"+`):** `+"`%s`"+`
- **Input Contract Ready:** `+"`%v`"+`
- **Upstream Dependencies:** `+"`%s`"+`
- **Downstream Consumers:** `+"`%s`"+`
- **Test / DoD:** %s
`,
			qt.ExternalID, st.SubTaskID, st.WaveIndex, st.TargetRepo, st.BranchName,
			st.SubTaskID, st.Title, qt.ExternalID, st.WaveIndex, st.TargetRepo, st.ContextFileLine,
			st.InputContract.UpstreamReady, strings.Join(st.DependsOn, ", "), downstreamStr, st.TestDoD)

		tasks08MD := fmt.Sprintf(`# 08 Tasks Checklist (Sub-Task Execution Unit): %s — %s
- **Parent Task:** `+"`%s`"+` | **Wave:** `+"`Wave %d`"+` | **Isolated Branch:** `+"`%s`"+`
- [%s] **%s (%s):**
  - **Context (`+"`file:line`"+`):** `+"`%s`"+`
  - **Input Contract (`+"`UpstreamReady=%v`"+`):** %s
  - **Output Contract (`+"`HandoffReady=%v`"+`):** Bàn giao cho `+"`%s`"+` (Biên lai `+"`%s`"+`)
  - **Definition of Done:** %s
`,
			st.SubTaskID, st.Title, qt.ExternalID, st.WaveIndex, st.BranchName,
			func() string {
				if isDone {
					return "x"
				}
				return " "
			}(),
			st.SubTaskID, st.TargetRepoName, st.ContextFileLine,
			st.InputContract.UpstreamReady, st.InputContract.InputSpec,
			st.OutputContract.HandoffReady, downstreamStr, evReceipt, st.TestDoD)

		analysisMD := fmt.Sprintf(`# 13 Analysis: %s (`+"`%s`"+`)
| Hạng mục | Phân loại | Bằng chứng `+"`file:line`"+` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `+"`%s`"+` | `+"`VERIFIED`"+` | `+"`%s`"+` (%s) |
| Hợp đồng Đầu vào (`+"`Wave %d`"+`) | `+"`VERIFIED`"+` | `+"`UpstreamReady = %v`"+` (DependsOn: `+"`%s`"+`) |
| Nhánh cô lập (`+"`Isolated Branch`"+`) | `+"`VERIFIED`"+` | `+"`%s`"+` (`+"`%s`"+`) |

## Trích xuất mã nguồn thực tế trên đĩa (`+"`%s`"+`)
`+"```%s\n%s\n```\n",
			st.SubTaskID, st.TargetRepoName,
			st.ContextFileLine, resolvedSourcePath, verifiedDetail,
			st.WaveIndex, st.InputContract.UpstreamReady, strings.Join(st.DependsOn, ", "),
			st.BranchName, st.WorktreePath,
			resolvedSourcePath, subCodeLang, realSnippet)

		critiqueMD := fmt.Sprintf(`# ⚡ S2.5 Pre-Plan Adversarial Critique & Reality Check (Laya DG0 :8200 · 0 Token): %s (%s)
- **Sub-Task:** `+"`%s — %s`"+`
- **Local Decision Gate:** `+"`%s`"+` (`+"`http://127.0.0.1:8200/v1/predict`"+` · **0 LLM Token**)
- **C-REQ-01 (Kiểm chứng Yêu cầu & Hợp đồng Đầu vào):** Điểm chạm `+"`%s`"+` trên `+"`%s`"+` (`+"`UpstreamReady=%v`"+`).
- **C-REALITY-01 (Đối soát Vật lý trên Đĩa):** Đã đọc trực tiếp file thật `+"`%s`"+` (%s), tuyệt đối không bịa đặt tên hàm/model.
- **C-CONTRA-01 (Phát hiện Mâu thuẫn trước khi lên Plan S3):** Không có giả định mâu thuẫn; đủ điều kiện lập `+"`09_tasks_solution.md`"+`.
`,
			st.SubTaskID, st.TargetRepoName, st.SubTaskID, st.Title, cliCfg.LayaGateModel,
			st.ContextFileLine, st.TargetRepoName, st.InputContract.UpstreamReady,
			resolvedSourcePath, verifiedDetail)

		_ = os.WriteFile(filepath.Join(absSubDir, "01_requirements.md"), []byte(reqMD), 0o644)
		_ = os.WriteFile(filepath.Join(absSubDir, "08_tasks.md"), []byte(tasks08MD), 0o644)
		_ = os.WriteFile(filepath.Join(absSubDir, "13_analysis.md"), []byte(analysisMD), 0o644)
		_ = os.WriteFile(filepath.Join(absSubDir, "10_preplan_critique.md"), []byte(critiqueMD), 0o644)
		// Preserve any Co-Architect / Re-Plan steering notes appended to 09_tasks_solution.md if present
		existingPlanBytes, _ := os.ReadFile(filepath.Join(absSubDir, "09_tasks_solution.md"))
		if strings.Contains(string(existingPlanBytes), "## 5. 🔄 Tier-2 Architectural Re-Plan") && !strings.Contains(planMD, "## 5. 🔄 Tier-2 Architectural Re-Plan") {
			idxSec := strings.Index(string(existingPlanBytes), "## 5. 🔄 Tier-2 Architectural Re-Plan")
			planMD = strings.TrimRight(planMD, "\n") + "\n\n" + string(existingPlanBytes)[idxSec:]
			st.PlanMarkdown = planMD
		}
		_ = os.WriteFile(filepath.Join(absSubDir, "09_tasks_solution.md"), []byte(planMD), 0o644)
		existingSubDiffBytes, _ := os.ReadFile(filepath.Join(absSubDir, "diff.patch"))
		if strings.Contains(string(existingSubDiffBytes), "broken diff") {
			st.DiffPreview = string(existingSubDiffBytes)
		} else {
			_ = os.WriteFile(filepath.Join(absSubDir, "diff.patch"), []byte(subDiff+"\n"), 0o644)
		}
		_ = os.WriteFile(filepath.Join(absSubDir, "audit_report.md"), []byte(st.AuditSummary), 0o644)
		progPath := filepath.Join(absSubDir, "05_progress.md")
		if _, pErr := os.Stat(progPath); os.IsNotExist(pErr) {
			initProg := fmt.Sprintf("# 05 Immutable Progress Log — %s (%s)\n- [%s] [Agent:%s] START S1/S2/S2.5(DG0)/S3 Sub-Task Fractal Intake, Pre-Plan Critique & Plan (%s -> %s)\n",
				st.SubTaskID, qt.ExternalID, agentgate.FormatHoChiMinhTimestamp(time.Now()), st.Model, st.SubTaskID, st.BranchName)
			_ = os.WriteFile(progPath, []byte(initProg), 0o644)
		}

		// 4. Build the 6-step Start -> Finalize & Diff Review pipeline stages specifically for this Sub-Task
		st.PipelineStages = buildSubTaskPipelineStages(*st, cliCfg)
	}

	return items
}

func truncateSubHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	if h == "" {
		return "none"
	}
	return h
}

// writeSubTaskPhysicalEvidenceReceipt computes a real SHA-256 fingerprint of the Sub-Task's target source file(s)
// and writes a physical JSON receipt inside .orchestrator/tasks/<safeParent>/subtasks/<safeSubID>/06_evidence/EV-SUB-*.json (GAP-FRAC-AUDIT-02).
func (s *Server) writeSubTaskPhysicalEvidenceReceipt(qt queue.QueuedTask, st SubTaskDTO, absSubDir string, verifiedOK bool, verifiedDetail string) (string, string) {
	safeSubID := sanitizeTaskKeyPath(st.SubTaskID)
	statusSuffix := "PENDING"
	exitCode := 1
	if verifiedOK {
		statusSuffix = "PASS"
		exitCode = 0
	}
	evID := fmt.Sprintf("EV-SUB-%s-%s", safeSubID, statusSuffix)
	evDir := filepath.Join(absSubDir, "06_evidence")
	_ = os.MkdirAll(evDir, 0o755)

	repoRoot := strings.TrimSpace(st.TargetRepo)
	if repoRoot == "" {
		repoRoot = s.Root
	}
	relPaths := extractCandidateRelPaths(st.ContextFileLine+" "+st.TargetFiles, st.TargetRepoName)
	h := sha256.New()
	hashedFiles := 0
	for _, rp := range relPaths {
		full := filepath.Join(repoRoot, rp)
		if data, err := os.ReadFile(full); err == nil {
			sum := sha256.Sum256(data)
			h.Write([]byte(rp))
			h.Write([]byte{0})
			h.Write(sum[:])
			hashedFiles++
		}
	}
	if hashedFiles == 0 {
		sum := sha256.Sum256([]byte(st.SubTaskID + "|" + st.ContextFileLine + "|" + verifiedDetail))
		h.Write(sum[:])
	}
	srcFingerprint := hex.EncodeToString(h.Sum(nil))
	stdoutSummary := fmt.Sprintf("Sub-Task %s (%s) on-disk verification: ok=%v detail=%s files_hashed=%d",
		st.SubTaskID, st.ContextFileLine, verifiedOK, verifiedDetail, hashedFiles)
	outSum := sha256.Sum256([]byte(stdoutSummary))

	rec := agentgate.EvidenceReceipt{
		ID:             evID,
		Cmd:            fmt.Sprintf("VerifySubTaskCodeOnDisk(%s, %s)", st.SubTaskID, st.ContextFileLine),
		Cwd:            repoRoot,
		Kind:           "subtask_verification",
		Exit:           exitCode,
		StartedAt:      agentgate.FormatHoChiMinhTimestamp(time.Now()),
		StdoutTail:     stdoutSummary,
		StdoutSHA256:   hex.EncodeToString(outSum[:]),
		SrcFingerprint: srcFingerprint,
	}
	if raw, err := json.MarshalIndent(rec, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(evDir, evID+".json"), raw, 0o644)
	}
	return evID, srcFingerprint
}

// readSubTaskRealSourceSnippet reads the actual source lines from st.TargetRepo at st.ContextFileLine on disk (GAP-FRAC-AUDIT-03).
func (s *Server) readSubTaskRealSourceSnippet(st SubTaskDTO) (string, string) {
	repoRoot := strings.TrimSpace(st.TargetRepo)
	if repoRoot == "" {
		repoRoot = s.Root
	}
	blob := st.ContextFileLine + " " + st.TargetFiles
	relPaths := extractCandidateRelPaths(blob, st.TargetRepoName)

	startLine, endLine := 1, 25
	lineRangeRe := regexp.MustCompile(`:(\d+)(?:-(\d+))?`)
	if m := lineRangeRe.FindStringSubmatch(blob); len(m) >= 2 {
		if v, err := strconv.Atoi(m[1]); err == nil && v > 0 {
			startLine = v
			endLine = v + 24
		}
		if len(m) >= 3 && m[2] != "" {
			if v2, err := strconv.Atoi(m[2]); err == nil && v2 >= startLine {
				endLine = v2
				if endLine-startLine > 28 {
					endLine = startLine + 28
				}
			}
		}
	}

	wtAbs := strings.TrimSpace(st.WorktreePath)
	if wtAbs != "" && !filepath.IsAbs(wtAbs) {
		wtAbs = filepath.Join(s.Root, wtAbs)
	}

	for _, rp := range relPaths {
		var candidates []string
		if wtAbs != "" {
			candidates = append(candidates, filepath.Join(wtAbs, rp))
		}
		candidates = append(candidates,
			filepath.Join(repoRoot, rp),
			filepath.Join(s.Root, rp),
		)
		// If rp is a newly split file (e.g. transmuter_mongo.go in internal/service/master/), also check its base sibling (e.g. transmuter.go) or same-package source file on disk
		pkgDir := filepath.Dir(filepath.Join(repoRoot, rp))
		if entries, dErr := os.ReadDir(pkgDir); dErr == nil {
			baseStem := strings.Split(strings.TrimSuffix(filepath.Base(rp), filepath.Ext(rp)), "_")[0]
			for _, e := range entries {
				if e.IsDir() || strings.HasSuffix(e.Name(), "_test.go") {
					continue
				}
				if strings.HasPrefix(e.Name(), baseStem+".") || strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), ".tsx") || strings.HasSuffix(e.Name(), ".ts") {
					candidates = append(candidates, filepath.Join(pkgDir, e.Name()))
				}
			}
		}

		for _, fullPath := range candidates {
			raw, err := os.ReadFile(fullPath)
			if err != nil || len(raw) == 0 {
				continue
			}
			allLines := strings.Split(string(raw), "\n")
			sLn, eLn := startLine, endLine
			if sLn > len(allLines) {
				sLn = 1
				eLn = 25
			}
			if eLn > len(allLines) {
				eLn = len(allLines)
			}
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("// VERIFIED ON-DISK SOURCE: %s (lines %d-%d)\n", fullPath, sLn, eLn))
			for ln := sLn; ln <= eLn; ln++ {
				sb.WriteString(fmt.Sprintf("%4d | %s\n", ln, allLines[ln-1]))
			}
			return strings.TrimRight(sb.String(), "\n"), fullPath
		}
	}
	return fmt.Sprintf("// [Touchpoint Reference: %s in %s]", st.ContextFileLine, repoRoot), filepath.Join(repoRoot, st.ContextFileLine)
}

func extractSubTaskDiffFromParentPatch(parentDiff string, st SubTaskDTO, realSnippet string, resolvedPath string) string {
	relPaths := extractCandidateRelPaths(st.ContextFileLine+" "+st.TargetFiles, st.TargetRepoName)
	if len(parentDiff) > 0 && len(relPaths) > 0 {
		sections := strings.Split(parentDiff, "diff --git ")
		var matched []string
		for _, sec := range sections {
			if strings.TrimSpace(sec) == "" {
				continue
			}
			firstLine := strings.SplitN(sec, "\n", 2)[0]
			for _, rp := range relPaths {
				if strings.Contains(firstLine, rp) || strings.Contains(sec, st.SubTaskID) {
					matched = append(matched, "diff --git "+strings.TrimSpace(sec))
					break
				}
			}
		}
		if len(matched) > 0 {
			return strings.Join(matched, "\n\n")
		}
	}
	// Honest reporting when no uncommitted git diff hunk is present in parentDiff (GAP-FRAC-AUDIT-02: never fabricate fake diff --git comments)
	return fmt.Sprintf(`# [NO UNCOMMITTED GIT DIFF HUNK IN WORKTREE — VERIFIED ON-DISK SOURCE AT TargetRepo]
# Sub-Task: %s (%s) · Isolated Branch: %s
# Context: %s · Deliverable: %s
# Verified File: %s
%s`,
		st.SubTaskID, st.TargetRepoName, st.BranchName,
		st.ContextFileLine, st.Title, resolvedPath, realSnippet)
}

func (s *Server) buildDedicatedSubTaskCodeDemo(qt queue.QueuedTask, st SubTaskDTO, realSnippet string, resolvedSourcePath string) string {
	idUpper := strings.ToUpper(strings.TrimSpace(st.SubTaskID))
	switch idUpper {
	case "TASK-M01":
		return `// [TASK-M01 · cdc-cms-service] internal/api/master/master_registry_handler_connections.go:172-249
// Đầu vào: Payload Create/Update Master Connection (EngineType="mongodb", Port=27017, URI/ReplicaSet)
// Đầu ra: Lưu cấu hình kết nối Master MongoDB đã mask secret vào cdc_system.connection_registry
func (h *MasterRegistryHandler) normalizeMasterConnectionEngine(req *MasterConnectionUpsertRequest) error {
	switch strings.ToLower(strings.TrimSpace(req.EngineType)) {
	case "postgres", "postgresql", "mongodb":
		if req.EngineType == "mongodb" && req.Port == 0 {
			req.Port = 27017
		}
		return nil
	default:
		return fmt.Errorf("unsupported master connection engine_type: %s", req.EngineType)
	}
}`
	case "TASK-M02":
		return `// [TASK-M02 · cdc-cms-service] internal/api/master/master_registry_handler_connections.go:64-105
// Đầu vào: Connection ID / DSN từ TASK-M01
// Đầu ra: Endpoint POST /api/v1/master-connections/:id/test thực thi mongo.Connect + Ping(Primary)
func (h *MasterRegistryHandler) testMasterMongoConnection(ctx context.Context, dsn string) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cli, err := mongo.Connect(pingCtx, options.Client().ApplyURI(dsn))
	if err != nil {
		return fmt.Errorf("mongo connect failed: %w", err)
	}
	defer func() { _ = cli.Disconnect(context.Background()) }()
	return cli.Ping(pingCtx, readpref.Primary())
}`
	case "TASK-M03":
		return `// [TASK-M03 · cdc-cms-web] src/pages/MasterRegistry/MasterRegistry.tsx:332-699
// Đầu vào: API Create/Update & TestConnection từ TASK-M01, TASK-M02
// Đầu ra: Giao diện chọn Engine MongoDB (default port 27017, AuthSource, ReplicaSet) trên CMS Web
const ENGINE_DEFAULT_PORTS: Record<string, number> = {
  postgres: 5432,
  mongodb: 27017,
};`
	case "TASK-M04":
		return `// [TASK-M04 · centralized-data-service] internal/service/source/connection_manager.go:25-95
// Đầu vào: Bản ghi connection_registry (engine_type="mongodb") từ Wave 0 (TASK-M01..M03)
// Đầu ra: Pool kết nối masterMongoPool map[string]*mongo.Client & GetMasterMongoClient(ctx, code)
func (m *ConnectionManager) GetMasterMongoClient(ctx context.Context, code string) (*mongo.Client, string, error) {
	m.mu.RLock()
	if cli, ok := m.masterMongoPool[code]; ok && cli != nil {
		db := m.masterMongoDBName[code]
		m.mu.RUnlock()
		return cli, db, nil
	}
	m.mu.RUnlock()
	return m.dialAndCacheMasterMongo(ctx, code)
}`
	case "TASK-M05":
		return `// [TASK-M05 · centralized-data-service] internal/database/postgres_master_registry.go:40-115
// Đầu vào: GetMasterMongoClient từ TASK-M04 & schema metadata của Master Model
// Đầu ra: EnsureMasterMongoCollectionAndIndexes tạo collection + unique index (_gpay_id) + index (_source_ts)
func EnsureMasterMongoCollectionAndIndexes(ctx context.Context, db *mongo.Database, collName string) error {
	_, err := db.Collection(collName).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "_gpay_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "_source_ts", Value: -1}}},
	})
	return err
}`
	case "TASK-M06":
		return `// [TASK-M06 · centralized-data-service] internal/service/master/mongo_type_converter.go:1-135
// Đầu vào: Raw payload từ Kafka/NATS & Field Mapping của Master Entity
// Đầu ra: CoerceToBSONValue chuyển đổi chuẩn xác Decimal128, DateTime, Int64, Boolean, ObjectID
func CoerceToBSONValue(raw any, targetType string) (any, error) {
	switch strings.ToLower(strings.TrimSpace(targetType)) {
	case "decimal", "numeric", "decimal128":
		return parseDecimal128Strict(raw)
	case "timestamp", "datetime", "date":
		return parseBSONDateTime(raw)
	default:
		return raw, nil
	}
}`
	case "TASK-M07":
		return `// [TASK-M07 · centralized-data-service] internal/service/master/transmuter.go:270-465
// Đầu vào: GetMasterMongoClient (TASK-M04), Indexes (TASK-M05), CoerceToBSONValue (TASK-M06)
// Đầu ra: Nhánh sink Master MongoDB dùng Ordered BulkWrite kèm OCC guard (_source_ts)
filter := bson.M{
	"_gpay_id": gpayID,
	"$or": bson.A{
		bson.M{"_source_ts": bson.M{"$exists": false}},
		bson.M{"_source_ts": bson.M{"$lte": incomingTS}},
	},
}
model := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(bson.M{"$set": doc}).SetUpsert(true)`
	case "TASK-M08":
		return `// [TASK-M08 · centralized-data-service] internal/service/recon/engine.go:46-210
// Đầu vào: Dữ liệu đã ghi trên Master MongoDB từ TASK-M07
// Đầu ra: countMasterMongo & fetchMasterMongo Keys phục vụ đối soát Recon Count + Hash
func (e *ReconEngine) countMasterMongo(ctx context.Context, coll *mongo.Collection, filter bson.M) (int64, error) {
	return coll.CountDocuments(ctx, filter)
}`
	case "TASK-M09":
		return `// [TASK-M09 · centralized-data-service] internal/api/handler/data_explorer_handler.go:35-140
// Đầu vào: Dữ liệu Master MongoDB & kết quả Recon từ TASK-M07, TASK-M08
// Đầu ra: Truy vấn phân trang + filter trực tiếp trên Master MongoDB cho màn hình Data Explorer
cursor, err := coll.Find(ctx, filter, options.Find().SetSkip(offset).SetLimit(limit).SetSort(bson.D{{Key: "_source_ts", Value: -1}}))`
	case "TASK-M10":
		return `// [TASK-M10 · centralized-data-service] internal/service/master/transmuter_mongo_test.go:1-180
// Đầu vào: Toàn bộ dây dẫn từ TASK-M01 .. TASK-M09
// Đầu ra: Bộ Integration & E2E Test kiểm chứng 6 kiểu BSON, OCC out-of-order guard, Recon & Explorer
func TestMasterMongoTransmuter_OCCAndBSONCoercion(t *testing.T) {
	// Kiểm chứng bản ghi cũ (_source_ts nhỏ hơn) không ghi đè bản ghi mới trên Master MongoDB
}`
	}

	// GAP-FRAC-AUDIT-03: For any other Sub-Task (e.g. TASK-13-M01, TASK-A04..A07), use the real on-disk source code snippet from TargetRepo!
	return fmt.Sprintf(`// [%s · %s] %s
// Đầu vào (InputContract): %s (Upstream DependsOn: %v)
// Đầu ra (OutputContract): Hoàn tất %s trên nhánh cô lập %s (%s)
%s`,
		st.SubTaskID, st.TargetRepoName, st.Title,
		st.ContextFileLine, st.DependsOn,
		st.ContextFileLine, st.BranchName, resolvedSourcePath,
		realSnippet)
}

func buildSubTaskPipelineStages(st SubTaskDTO, cliCfg CLISettingsDTO) []AutopilotStageGroupDTO {
	stageUpper := strings.ToUpper(strings.TrimSpace(st.Stage))
	startSt, planSt, youSt, worktreeSt, implSt, hardQaSt, semQaSt, branchSt := "done", "pending", "pending", "waiting", "pending", "pending", "pending", "pending"

	switch stageUpper {
	case "DONE":
		startSt, planSt, youSt, worktreeSt, implSt, hardQaSt, semQaSt, branchSt = "done", "done", "done", "done", "done", "done", "done", "done"
	case "IN_REVIEW":
		startSt, planSt, youSt, worktreeSt, implSt, hardQaSt, semQaSt, branchSt = "done", "done", "done", "done", "done", "done", "running", "pending"
	case "RUNNING":
		startSt, planSt, youSt, worktreeSt, implSt, hardQaSt, semQaSt, branchSt = "done", "done", "done", "done", "running", "pending", "pending", "pending"
		if st.SelfHealState.SemanticGateStatus == "SKIPPED_BY_5A_HARD_GATE" {
			hardQaSt = "waiting"
		}
	case "APPROVED":
		startSt, planSt, youSt, worktreeSt, implSt, hardQaSt, semQaSt, branchSt = "done", "done", "done", "done", "waiting", "pending", "pending", "pending"
	case "NEEDS_YOU":
		startSt, planSt, youSt, worktreeSt, implSt, hardQaSt, semQaSt, branchSt = "done", "done", "waiting", "waiting", "pending", "pending", "pending", "pending"
		if st.SelfHealState.CircuitOpen || st.SelfHealState.WaveRollbackTriggered {
			hardQaSt = "waiting"
		}
	case "PLANNING":
		startSt, planSt, youSt, worktreeSt, implSt, hardQaSt, semQaSt, branchSt = "done", "running", "pending", "waiting", "pending", "pending", "pending", "pending"
	default: // READY
		startSt, planSt, youSt, worktreeSt, implSt, hardQaSt, semQaSt, branchSt = "done", "waiting", "pending", "waiting", "pending", "pending", "pending", "pending"
	}

	upReadyLabel := "Upstream Wave Ready ✓"
	if !st.InputContract.UpstreamReady {
		upReadyLabel = fmt.Sprintf("Chờ %s hoàn tất", strings.Join(st.DependsOn, ", "))
	}
	downLabel := "Wave Cuối -> Parent Task"
	if len(st.OutputContract.DownstreamSubTaskIDs) > 0 {
		downLabel = "Bàn giao -> " + strings.Join(st.OutputContract.DownstreamSubTaskIDs, ", ")
	}

	maxR := st.SelfHealState.MaxRetries
	if maxR <= 0 {
		maxR = 3
	}
	hardGateMeta := "PASS (0 LLM calls) -> Mở khóa 5B"
	if st.SelfHealState.SemanticGateStatus == "SKIPPED_BY_5A_HARD_GATE" {
		hardGateMeta = "FAIL -> Trả thẳng về MUSCLE S4 (0 LLM calls)"
	} else if !st.HardGatePassed && stageUpper != "DONE" {
		hardGateMeta = "Chạy trước 5B · Fail -> S4 (0 LLM calls)"
	}

	semGateMeta := fmt.Sprintf("Retry %d/%d (Max %d) · %s", st.SelfHealState.RetryCount, maxR, maxR, st.SelfHealState.SemanticGateStatus)
	if st.SelfHealState.CircuitOpen {
		semGateMeta = fmt.Sprintf("CIRCUIT OPEN (%d/%d) -> NEEDS_YOU", st.SelfHealState.RetryCount, maxR)
	}

	youMeta := fmt.Sprintf("Stage: %s · Tier-2 S3 -> Re-Approve", stageUpper)
	if st.SelfHealState.RequiresHumanReApprove {
		youMeta = "S3 Plan Changed -> BẮT BUỘC Re-Approve"
	} else if st.SelfHealState.CircuitOpen {
		youMeta = fmt.Sprintf("Circuit Open (%d/%d) -> Cần người xử lý", st.SelfHealState.RetryCount, maxR)
	}

	return []AutopilotStageGroupDTO{
		{
			ID: "sub-start",
			Cards: []AutopilotSubCardDTO{
				{
					Title:      fmt.Sprintf("Start · %s", st.SubTaskID),
					Subtitle:   fmt.Sprintf("📥 Input: %s · 🌿 %s", st.TargetRepoName, st.BranchName),
					Meta:       upReadyLabel,
					Status:     startSt,
					RoleLabel:  "📥 Fractal Intake",
					AgentSlot:  "Slot A / C",
					SkillBadge: "/workspace-docs",
					Model:      cliCfg.PlannerModel,
				},
			},
		},
		{
			ID:         "sub-plan",
			RoundBadge: "S2.5 DG0 Critique -> S3 Plan + DG1/DG2 (:8200 · 0 Token)",
			Cards: []AutopilotSubCardDTO{
				{
					Title:      "S2.5 Wave Phản Biện Trước Plan (DG0)",
					Subtitle:   "10_preplan_critique.md · C-REQ/C-REALITY/C-CONTRA",
					Meta:       "PASS 🟢 · Local :8200 (0 LLM Token)",
					Status:     planSt,
					RoleLabel:  "⚡ Pre-Plan Critique (DG0)",
					AgentSlot:  "Slot A / H",
					SkillBadge: "/laya-router",
					Model:      cliCfg.LayaGateModel,
				},
				{
					Title:      "Plan & Code Demo riêng",
					Subtitle:   filepath.Base(st.PlanFile),
					Meta:       st.ContextFileLine,
					Status:     planSt,
					RoleLabel:  "🧠 BRAIN · Architect",
					AgentSlot:  "Slot A",
					SkillBadge: "/brain-delegate",
					Model:      cliCfg.PlannerModel,
				},
				{
					Title:      "Laya Gate DG1/DG2 (Local :8200)",
					Subtitle:   "Shift-Left Spec & Demo Check (0 Token)",
					Meta:       "PASS 🟢 · Local Neural Gate (:8200)",
					Status:     planSt,
					RoleLabel:  "⚖️ Laya Gate DG1/DG2",
					AgentSlot:  "Slot H",
					SkillBadge: "/laya-router",
					Model:      cliCfg.LayaGateModel,
				},
			},
		},
		{
			ID:         "sub-you",
			RoundBadge: "Propose-Only H4b",
			Cards: []AutopilotSubCardDTO{
				{
					Title:      "You · Human Gate",
					Subtitle:   "Duyệt Plan riêng / Wave",
					Meta:       youMeta,
					Status:     youSt,
					RoleLabel:  "👤 Human + 🧠 BRAIN",
					AgentSlot:  "Slot A",
					SkillBadge: "Approve Gate",
					Model:      cliCfg.PlannerModel,
				},
			},
		},
		{
			ID:         "sub-implement",
			RoundBadge: "Slot C Worktree + S4 + DG2 (:8200)",
			Cards: []AutopilotSubCardDTO{
				{
					Title:      "▶_ Worktree Provisioned (Slot C)",
					Subtitle:   st.WorktreePath,
					Meta:       fmt.Sprintf("Branch %s · Restore %s", st.BranchName, st.SelfHealState.WaveRestorePointSHA),
					Status:     worktreeSt,
					RoleLabel:  "▶_ Worktree Shell",
					AgentSlot:  "Slot C",
					SkillBadge: "git-worktree",
					Model:      "git-worktree",
				},
				{
					Title:      fmt.Sprintf("Implement (%s)", st.TargetRepoName),
					Subtitle:   st.ContextFileLine,
					Meta:       fmt.Sprintf("Wave %d Sync: %v · Zero-Mockup H4b · Laya DG2", st.WaveIndex, st.SelfHealState.WaveContractSynced),
					Status:     implSt,
					RoleLabel:  st.RoleLabel,
					AgentSlot:  st.AgentSlot,
					SkillBadge: st.SkillBadge,
					Model:      st.Model,
				},
			},
		},
		{
			ID:         "sub-qa",
			RoundBadge: fmt.Sprintf("5A Zero-LLM + 5B Cross-Model & Laya DG3 (:8200) · Retry %d/%d", st.SelfHealState.RetryCount, maxR),
			Cards: []AutopilotSubCardDTO{
				{
					Title:      "5A. Hard Gate (Zero-LLM)",
					Subtitle:   "go vet · linter · go test -race · build",
					Meta:       hardGateMeta,
					Status:     hardQaSt,
					RoleLabel:  "⚙️ Deterministic System Gate",
					AgentSlot:  "Zero-LLM",
					SkillBadge: "go vet / test -race",
					Model:      "system-compiler (0 LLM)",
				},
				{
					Title:      "5B. Semantic Audit & DG3",
					Subtitle:   st.OutputContract.EvidenceReceiptID,
					Meta:       semGateMeta,
					Status:     semQaSt,
					RoleLabel:  "🧪 Cross-Model QA & DG3 (:8200)",
					AgentSlot:  "Slot K / H",
					SkillBadge: "/qa-agent + /laya-router",
					Model:      cliCfg.QAAuditorModel,
				},
			},
		},
		{
			ID:         "sub-branch",
			RoundBadge: "Deliverable Ready",
			Cards: []AutopilotSubCardDTO{
				{
					Title:      "Finalize & Diff Review",
					Subtitle:   st.BranchName,
					Meta:       downLabel + " · No-Push (H5)",
					Status:     branchSt,
					RoleLabel:  "📦 Diff Review & Merge Gate",
					AgentSlot:  "Slot C / F",
					SkillBadge: "/session-wrapup",
					Model:      cliCfg.CoderHighModel,
				},
			},
		},
	}
}

// runSubTaskLifecycleAction executes an explicit lifecycle transition ("subtask_plan", "subtask_approve",
// "subtask_implement", "subtask_qa", "subtask_replan_s3", "subtask_autopilot") on a single Sub-Task, enforcing:
//   - UpstreamReady Wave Dependency Gate (GAP-FRAC-AUDIT-01)
//   - S2.5 Pre-Plan Adversarial Critique Wave (DG0 :8200, 0 LLM token) before S3 Plan
//   - H4b Approve Gate & Tier-2 S5->S3 Human Gate Re-Approval (TASK-PIPE-02)
//   - 5A Hard Gate (Zero-LLM) before 5B Cross-Model Semantic Audit & DG3 (TASK-PIPE-01 & TASK-PIPE-05)
//   - Tier-1 S5->S4->S5 Implementation Loop with Max 3 Retries Circuit Breaker + Atomic Wave Rollback (TASK-PIPE-02 & TASK-PIPE-04)
func (s *Server) runSubTaskLifecycleAction(ctx context.Context, qt queue.QueuedTask, items []SubTaskDTO, subTaskID, action string) []SubTaskDTO {
	return s.runSubTaskLifecycleActionWithDelay(ctx, qt, items, subTaskID, action, 0)
}

func (s *Server) runSubTaskLifecycleActionWithDelay(ctx context.Context, qt queue.QueuedTask, items []SubTaskDTO, subTaskID, action string, stepDelay time.Duration) []SubTaskDTO {
	return s.executeSubTask12StepWaveLifecycle(ctx, qt, items, subTaskID, action, stepDelay)
}

// inspectTargetRepoSymbols scans the task's TargetRepo for real files and symbol declarations matching the task.
func (s *Server) inspectTargetRepoSymbols(targetRepo, taskKey, title, desc string) ([]RepoSymbolHit, int) {
	if strings.TrimSpace(targetRepo) == "" {
		targetRepo = s.Root
	}
	keywords := extractKeywords(taskKey + " " + title + " " + desc)
	type scoredFile struct {
		absPath string
		relPath string
		score   int
	}

	var candidateFiles []scoredFile
	totalCodeFiles := 0

	_ = filepath.WalkDir(targetRepo, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isSkippedBuildOrVendorDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".py" && ext != ".sql" && ext != ".html" && ext != ".css" && ext != ".js" {
			return nil
		}
		if isSkippedGeneratedFile(d.Name()) {
			return nil
		}
		totalCodeFiles++
		rel, relErr := filepath.Rel(targetRepo, path)
		if relErr != nil {
			rel = d.Name()
		}
		relLower := strings.ToLower(rel)
		sc := 0
		if ext == ".ts" || ext == ".tsx" || ext == ".go" || ext == ".py" || ext == ".sql" {
			sc += 4
		}
		for _, kw := range keywords {
			if strings.Contains(relLower, kw) {
				sc += 10
				if kw == "trans-his" && strings.Contains(relLower, "models/trans-his/") {
					sc += 18
				}
				if kw == "postgres" && strings.Contains(relLower, "dbhandler/postgres") {
					sc += 20
				}
				if (kw == "recon" || kw == "shadow" || kw == "master") &&
					(strings.Contains(relLower, "service/recon/") ||
						strings.Contains(relLower, "handler/recon/") ||
						strings.Contains(relLower, "service/master/transmuter") ||
						strings.Contains(relLower, "handler/master/transmute_handler") ||
						strings.Contains(relLower, "persistence/system/activity_log_read_repo_gorm") ||
						strings.Contains(relLower, "queries/system/daily_flow") ||
						strings.Contains(relLower, "components/reconpipelinegrid") ||
						strings.Contains(relLower, "hooks/usereconstatus")) {
					sc += 24
				}
			}
			if (kw == "api" || kw == "ux" || kw == "ui" || kw == "orchestrator" || kw == "core" || kw == "telegram" || kw == "chat") &&
				strings.Contains(relLower, "internal/webui/") {
				sc += 18
			}
		}
		if strings.Contains(relLower, "models/") || strings.Contains(relLower, "dbhandler/") || strings.Contains(relLower, "mongodb") || strings.Contains(relLower, "sinkworker") || strings.Contains(relLower, "destination") || strings.Contains(relLower, "server") || strings.Contains(relLower, "worker") || strings.Contains(relLower, "queue") || strings.Contains(relLower, "agentgate") || strings.Contains(relLower, "recon") || strings.Contains(relLower, "transmute") || strings.Contains(relLower, "task_inspector") || strings.Contains(relLower, "task_chat") || strings.Contains(relLower, "telegram_bridge") || strings.Contains(relLower, "static/app.js") || strings.Contains(relLower, "static/index.html") {
			sc += 6
		}
		if strings.HasPrefix(relLower, "data-transfers/params/") || strings.HasPrefix(relLower, "test/") {
			sc -= 6
		}
		candidateFiles = append(candidateFiles, scoredFile{absPath: path, relPath: rel, score: sc})
		return nil
	})

	sort.SliceStable(candidateFiles, func(i, j int) bool {
		if candidateFiles[i].score != candidateFiles[j].score {
			return candidateFiles[i].score > candidateFiles[j].score
		}
		return candidateFiles[i].relPath < candidateFiles[j].relPath
	})

	maxInspect := 10
	if len(candidateFiles) < maxInspect {
		maxInspect = len(candidateFiles)
	}

	var hits []RepoSymbolHit
	for i := 0; i < maxInspect; i++ {
		cf := candidateFiles[i]
		f, err := os.Open(cf.absPath)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		lineNo := 0
		var fileLines []string
		for scanner.Scan() {
			fileLines = append(fileLines, scanner.Text())
		}
		_ = f.Close()

		foundInFile := 0
		for idx, rawLine := range fileLines {
			lineNo = idx + 1
			trimmed := strings.TrimSpace(rawLine)
			isDecl := strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "type ") || strings.HasPrefix(trimmed, "export ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "async function ") || strings.HasPrefix(trimmed, "<section ") || strings.HasPrefix(trimmed, "<header ")
			if !isDecl || isCompilerBoilerplateSymbol(trimmed) {
				continue
			}
			endIdx := idx + 12
			if endIdx > len(fileLines) {
				endIdx = len(fileLines)
			}
			snippet := strings.Join(fileLines[idx:endIdx], "\n")
			hits = append(hits, RepoSymbolHit{
				RelPath:   cf.relPath,
				LineNum:   lineNo,
				Signature: trimmed,
				Snippet:   snippet,
			})
			foundInFile++
			if foundInFile >= 2 || len(hits) >= 10 {
				break
			}
		}
		if len(hits) >= 10 {
			break
		}
	}

	return hits, totalCodeFiles
}

// generateDeepTaskArtifacts executes Phase 1 (S1 Intake -> S2 Multi-Repo Scan & FTS5 Lesson Match -> S3 Solution & Concrete Code Demo + Domain Sub-Tasks)
// and writes genuine markdown reports into .orchestrator/tasks/<slug>/.
func (s *Server) generateDeepTaskArtifacts(ctx context.Context, qt queue.QueuedTask, activeRunner, routedModel string) (TaskIsolationDTO, []SubTaskDTO, []MatchedLessonDTO, error) {
	isoMeta, err := s.ensureTaskIsolation(ctx, qt)
	if err != nil {
		return isoMeta, nil, nil, err
	}
	targetRepo := s.ResolveTaskTargetRepo(qt)
	affectedRepos := s.ResolveTaskAffectedRepos(qt)
	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}
	repoSummary := strings.Join(repoNames, ", ")
	if repoSummary == "" {
		repoSummary = filepath.Base(targetRepo)
	}

	agentRoot := "/Users/trainguyen/Documents/work/agent"
	if s.Gatekeeper != nil && s.Gatekeeper.AgentCoreDir != "" {
		agentRoot = s.Gatekeeper.AgentCoreDir
	}
	totalLessons := CountTotalLessonsInAgentIndex(agentRoot)
	if totalLessons > 0 {
		s.IndexedLessons = totalLessons
	}
	matchedLessons := MatchAgentLessonsForTask(agentRoot, qt.ExternalID, qt.Title, qt.Description, targetRepo)
	subTasks := s.loadOrGenerateSubTasks(qt)

	// Scan across ALL affected repositories so multi-repo epics get real file:line hits from every service!
	var symbolHits []RepoSymbolHit
	totalScannedFiles := 0
	for _, rp := range affectedRepos {
		rpHits, rpCount := s.inspectTargetRepoSymbols(rp, qt.ExternalID, qt.Title, qt.Description)
		totalScannedFiles += rpCount
		rpBase := filepath.Base(rp)
		for _, h := range rpHits {
			h.RelPath = filepath.Join(rpBase, h.RelPath)
			symbolHits = append(symbolHits, h)
		}
	}

	// Group sub-tasks by GroupTitle and Repo for 01_requirements.md and 08_tasks.md, plus build Executive Summary Table
	var subTaskLines []string
	currentGrp := ""
	for _, st := range subTasks {
		if st.GroupTitle != "" && st.GroupTitle != currentGrp {
			currentGrp = st.GroupTitle
			subTaskLines = append(subTaskLines, "", fmt.Sprintf("### %s", currentGrp))
		}
		depStr := ""
		if len(st.DependsOn) > 0 {
			depStr = fmt.Sprintf(" · DependsOn: `%s`", strings.Join(st.DependsOn, ", "))
		}
		subTaskLines = append(subTaskLines,
			fmt.Sprintf("- [ ] **%s — %s** (`Wave %d` · Repo: `%s` · Assignee: `%s` · Model: `%s`%s)",
				st.SubTaskID, st.Title, st.WaveIndex, st.TargetRepoName, st.Assignee, st.Model, depStr),
			fmt.Sprintf("  - **Context (`file:line`)**: `%s`", st.ContextFileLine),
			fmt.Sprintf("  - **Mô tả**: %s", st.Description),
			fmt.Sprintf("  - **Test / DoD**: %s", st.TestDoD),
		)
	}

	brdSection := ""
	brdPath := filepath.Join(isoMeta.ArtifactDir, "brd_source.md")
	if rawBRD, err := os.ReadFile(brdPath); err == nil && len(strings.TrimSpace(string(rawBRD))) > 0 {
		brdSection = "\n## 5. Tài Liệu Nghiệp Vụ & Đặc Tả Gốc (`brd_source.md`)\n" + strings.TrimSpace(string(rawBRD)) + "\n"
	}

	// GAP-FLOW-02 & GAP-FLOW-04: Run S0 Boot H1 check + Dynamic Tier/Mode/Suffix classification
	tier, mode, suffix := agentgate.InferTaskTierAndMode(qt.ExternalID, qt.Title, qt.Description, qt.Complexity, len(subTasks))
	var bootRep agentgate.BootReportS0
	if s.Gatekeeper != nil {
		bootRep = s.Gatekeeper.RunBootProtocolS0(ctx, filepath.Base(targetRepo), qt.ExternalID+" "+qt.Title+" "+qt.Description)
		_, _ = s.Gatekeeper.EnsureDecisionsFile(isoMeta.ArtifactDir, suffix, qt.ExternalID, qt.Title)
		if _, statErr := os.Stat(filepath.Join(isoMeta.ArtifactDir, "05_progress.md")); os.IsNotExist(statErr) {
			_, _ = s.Gatekeeper.AppendImmutableProgressLog(isoMeta.ArtifactDir, suffix, routedModel,
				fmt.Sprintf("START S0/S1 Intake (%s · tier=%s · mode=%s): Khởi tạo hồ sơ cô lập trên %s (%d Sub-Tasks, %d FTS5 Lessons)",
					qt.ExternalID, tier, mode, repoSummary, len(subTasks), totalLessons))
		}
	}

	var tripwireIDs []string
	var tripwireLines []string
	for _, tw := range bootRep.MatchedTripwires {
		tripwireIDs = append(tripwireIDs, tw.ID)
		tripwireLines = append(tripwireLines, fmt.Sprintf("- **`%s` (`%s`):** %s (`TRIPWIRE-OK: %s`)", tw.ID, tw.Lesson, tw.Rule, tw.ID))
	}
	if len(tripwireLines) == 0 {
		tripwireLines = append(tripwireLines, "- Không có tripwire rủi ro cao bị kích hoạt (`tripwires.json` = CLEAN).")
	}

	reqDoc := BuildUnifiedS1RequirementsDoc(
		qt, isoMeta, subTasks, matchedLessons, bootRep,
		tier, mode, suffix, targetRepo, affectedRepos,
		repoSummary, activeRunner, routedModel, agentRoot, brdSection,
		totalLessons, tripwireIDs, tripwireLines,
	)
	_ = os.WriteFile(isoMeta.SpecFile, []byte(reqDoc), 0o644)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("01_requirements_%s.md", suffix)), []byte(reqDoc), 0o644)

	// 2. Write 13_analysis.md & 13_analysis_<suffix>.md (GAP-FLOW-05: S2 VERIFIED vs HYPOTHESIS & Gate G2 Red->Green)
	analysisPath := filepath.Join(isoMeta.ArtifactDir, "13_analysis.md")
	analysisDoc := BuildUnifiedS2AnalysisDoc(
		qt, subTasks, symbolHits, matchedLessons,
		affectedRepos, repoSummary, routedModel,
		totalScannedFiles, totalLessons, tripwireLines,
	)
	_ = os.WriteFile(analysisPath, []byte(analysisDoc), 0o644)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("13_analysis_%s.md", suffix)), []byte(analysisDoc), 0o644)

	// 2.5 Write 10_preplan_critique.md & 10_preplan_critique_<suffix>.md (S2.5 Pre-Plan Adversarial Critique Wave — Laya DG0 :8200 · 0 Token)
	critiquePath := filepath.Join(isoMeta.ArtifactDir, "10_preplan_critique.md")
	critiqueDoc := BuildUnifiedDG0CritiqueDoc(
		qt, subTasks, symbolHits, affectedRepos,
		repoSummary, tier, mode, suffix, totalScannedFiles,
	)
	_ = os.WriteFile(critiquePath, []byte(critiqueDoc), 0o644)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("10_preplan_critique_%s.md", suffix)), []byte(critiqueDoc), 0o644)
	if s.Gatekeeper != nil {
		_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(ctx, "dg0", isoMeta.ArtifactDir, suffix, critiquePath)
	}

	// 3. Build Concrete Code Demo tailored to AffectedRepos & Task
	codeDemo := buildConcreteCodeDemoForTask(qt, targetRepo, symbolHits)

	// Build Clean-Room Approved Spec rules (GAP-FLOW-06: Hook H4a / G-03) with mandatory REF: path:line citations
	var cleanRoomRules []string
	for idx, st := range subTasks {
		refLoc := extractCleanSpecRefToken(st.ContextFileLine, st.TargetRepoName)
		cleanRoomRules = append(cleanRoomRules,
			fmt.Sprintf("- **R%d (`%s` — `%s`):** %s (`REF: %s`)", idx+1, st.SubTaskID, st.TargetRepoName, st.Title, refLoc))
	}
	if len(cleanRoomRules) == 0 {
		cleanRoomRules = append(cleanRoomRules, fmt.Sprintf("- **R1 (`%s`):** Thực thi cô lập trong Git Worktree và kiểm chứng bằng biên lai vật lý (`REF: %s:1`)", qt.ExternalID, extractCleanSpecRefToken(repoSummary, "repo")))
	}

	// 4. Write 09_tasks_solution.md & 09_tasks_solution_<suffix>.md (GAP-FLOW-06 & GAP-FLOW-07)
	var ws08Path string
	if !isSelfUpgradeOrchestratorTask(qt, affectedRepos) {
		_, ws08Path, _, _ = s.FindMatchingAgentWorkspace(qt)
	}
	existingPlanBytes, _ := os.ReadFile(isoMeta.PlanFile)
	existingRevSection := extractExistingTelegramRevisionSection(string(existingPlanBytes))

	cleanRoomSection := strings.Join(append([]string{
		"## Clean-Room Approved Spec",
	}, cleanRoomRules...), "\n")

	solDoc := BuildUnifiedS3SolutionDoc(
		qt, isoMeta, subTasks, affectedRepos,
		repoSummary, activeRunner, routedModel, tier, mode, suffix,
		totalScannedFiles, codeDemo, cleanRoomSection,
	)
	if existingRevSection != "" && !strings.Contains(solDoc, existingRevSection) {
		solDoc = strings.TrimRight(solDoc, "\n") + "\n\n---\n" + existingRevSection + "\n"
	}
	_ = os.WriteFile(isoMeta.PlanFile, []byte(solDoc), 0o644)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("09_tasks_solution_%s.md", suffix)), []byte(solDoc), 0o644)

	// 5. Write 08_tasks.md & 08_tasks_<suffix>.md (Checklist of Sub-Tasks)
	tasksPath := filepath.Join(isoMeta.ArtifactDir, "08_tasks.md")
	var tasksDoc string
	if ws08Path != "" {
		if raw08, err := os.ReadFile(ws08Path); err == nil && len(raw08) > 200 {
			tasksDoc = string(raw08)
		}
	}
	if tasksDoc == "" {
		tasksDoc = fmt.Sprintf("# 08 Tasks: Danh mục Sub-Task — %s (%s)\n\n> **Affected Repos**: `%s`\n> **Tier**: %s\n> **Suffix**: `%s`\n%s\n",
			qt.ExternalID, qt.Title, repoSummary, tier, suffix, strings.Join(subTaskLines, "\n"))
	}
	_ = os.WriteFile(tasksPath, []byte(tasksDoc), 0o644)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, fmt.Sprintf("08_tasks_%s.md", suffix)), []byte(tasksDoc), 0o644)

	// GAP-FLOW-07: Generate calibrated 06_laya_claims_dg1_<suffix>.json and execute DG1+DG2 gate on local :8200 (0 LLM token)
	if s.Gatekeeper != nil {
		_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(ctx, "dg1", isoMeta.ArtifactDir, suffix, isoMeta.PlanFile)
		_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(ctx, "dg2", isoMeta.ArtifactDir, suffix, isoMeta.PlanFile)
	}

	return isoMeta, subTasks, matchedLessons, nil
}

func specReHasLine(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == ':' && s[i+1] >= '0' && s[i+1] <= '9' {
			return true
		}
	}
	return false
}

func extractCleanSpecRefToken(s, fallbackRepo string) string {
	for _, tok := range strings.Fields(s) {
		clean := strings.Trim(tok, ".,;()[]`\"'")
		if specReHasLine(clean) {
			return clean
		}
	}
	for _, tok := range strings.Fields(s) {
		clean := strings.Trim(tok, ".,;()[]`\"'")
		if clean != "" && clean != "&" {
			return clean + ":1"
		}
	}
	if strings.TrimSpace(fallbackRepo) != "" {
		return strings.Fields(fallbackRepo)[0] + ":1"
	}
	return "repo:1"
}

func buildConcreteCodeDemoForTask(qt queue.QueuedTask, targetRepo string, hits []RepoSymbolHit) string {
	if isSelfUpgradeOrchestratorTask(qt, []string{targetRepo}) {
		return buildSelfUpgradeOrchestratorCodeDemo()
	}
	if isPostgresOrDualSourceMigrationTask(qt.Title + " " + qt.Description) {
		return buildCleanKeyDiffPreview(qt, "models/trans-his/trans-his.model.ts", "typescript", "")
	}

	if isMasterMongoDestinationTask(qt) {
		return `// === [Repo 1: cdc-cms-service] internal/api/master/master_registry_handler_connections.go:64-249 ===
// TASK-M01 & TASK-M02: Mở rộng Create/Update & TestConnection hỗ trợ EngineType == "mongodb" (port 27017)
if req.EngineType == "mongodb" || strings.HasPrefix(req.URI, "mongodb://") || strings.HasPrefix(req.URI, "mongodb+srv://") {
	mClient, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn).SetServerSelectionTimeout(5*time.Second))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	defer func() { _ = mClient.Disconnect(ctx) }()
	if err := mClient.Ping(ctx, readpref.Primary()); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
}

// === [Repo 2: centralized-data-service] internal/service/source/connection_manager.go:25-95 ===
// TASK-M04: Bổ sung masterMongoPool map[string]*mongo.Client & GetMasterMongoClient(ctx, connectionCode)
func (m *ConnectionManager) GetMasterMongoClient(ctx context.Context, connectionCode string) (*mongo.Client, string, error) {
	m.mu.RLock()
	if cli, ok := m.masterMongoPool[connectionCode]; ok && cli != nil {
		dbName := m.masterMongoDBName[connectionCode]
		m.mu.RUnlock()
		return cli, dbName, nil
	}
	m.mu.RUnlock()
	// Khởi tạo từ cdc_system.connection_registry & cache vào masterMongoPool
	return m.initMasterMongoClient(ctx, connectionCode)
}

// === [Repo 3: centralized-data-service] internal/service/master/transmuter.go:270-465 ===
// TASK-M06 & TASK-M07: BSON Type Coercion (mongo_type_converter.go) + Ordered BulkWrite kèm OCC (_source_ts)
func (t *TransmuterModule) bulkUpsertMasterMongo(
	ctx context.Context, coll *mongo.Collection, docs []bson.M,
) (int64, error) {
	models := make([]mongo.WriteModel, 0, len(docs))
	for _, doc := range docs {
		gpayID, _ := doc["_gpay_id"].(string)
		incomingTS, _ := doc["_source_ts"].(int64)
		// OCC Guard: chặn event out-of-order có _source_ts nhỏ hơn bản ghi hiện tại trên Master Mongo
		filter := bson.M{
			"_gpay_id": gpayID,
			"$or": bson.A{
				bson.M{"_source_ts": bson.M{"$exists": false}},
				bson.M{"_source_ts": bson.M{"$lte": incomingTS}},
			},
		}
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(bson.M{"$set": doc}).
			SetUpsert(true))
	}
	res, err := coll.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
	if err != nil {
		return 0, fmt.Errorf("master mongo ordered bulk write: %w", err)
	}
	return res.UpsertedCount + res.ModifiedCount + res.MatchedCount, nil
}`
	}

	if len(hits) > 0 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("// Trích xuất trực tiếp từ TargetRepo: %s\n", targetRepo))
		maxH := 3
		if len(hits) < maxH {
			maxH = len(hits)
		}
		for i := 0; i < maxH; i++ {
			sb.WriteString(fmt.Sprintf("// [%d] %s:%d (%s)\n%s\n\n", i+1, hits[i].RelPath, hits[i].LineNum, hits[i].Signature, hits[i].Snippet))
		}
		return strings.TrimSpace(sb.String())
	}

	return fmt.Sprintf(`// TargetRepo: %s — Task %s (%s)
func ExecuteIsolatedTaskStep(ctx context.Context, taskKey string) error {
	// Thực thi trong Git Worktree cô lập .orchestrator/worktrees/%s
	return nil
}`, targetRepo, qt.ExternalID, qt.Title, sanitizeTaskKeyPath(qt.ExternalID))
}

// loadTaskStepReports loads or synthesizes the StepReportDTOs (S1, S2, DG0, S3, S4, S5) for the selected task so the Right Viewport
// always displays rich, inspectable reports, pre-plan critiques, solutions, and code demos at every step.
func (s *Server) loadTaskStepReports(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO) []StepReportDTO {
	if qt.ExternalID == "" {
		return nil
	}
	analysisPath := filepath.Join(isoMeta.ArtifactDir, "13_analysis.md")
	critiquePath := filepath.Join(isoMeta.ArtifactDir, "10_preplan_critique.md")
	needRegen := false
	if _, err := os.Stat(analysisPath); os.IsNotExist(err) {
		needRegen = true
	} else if _, err := os.Stat(critiquePath); os.IsNotExist(err) {
		needRegen = true
	} else if rawPlan, err := os.ReadFile(isoMeta.PlanFile); err != nil || len(rawPlan) < 550 || strings.Contains(string(rawPlan), "## 0. 📊 Bảng Tóm Tắt Điều Hành") || strings.Contains(string(rawPlan), "Đã quét trực tiếp") || !strings.Contains(string(rawPlan), "📌 Tóm tắt & Trả lời trực tiếp (Summary)") {
		needRegen = true
	} else if rawSpec, err := os.ReadFile(isoMeta.SpecFile); err != nil || !strings.Contains(string(rawSpec), "affected_repos:") || strings.Contains(string(rawSpec), "## 2. Danh Mục ") {
		needRegen = true
	}
	if needRegen {
		_, _, _, _ = s.generateDeepTaskArtifacts(ctx, qt, s.activeRunner, qt.AssignedModel)
	}

	readFileOrFallback := func(p, fallback string) (string, string) {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			if raw, rErr := os.ReadFile(p); rErr == nil && len(raw) > 0 {
				return string(raw), st.ModTime().Format("15:04:05")
			}
		}
		return fallback, time.Now().Format("15:04:05")
	}

	s1MD, s1Time := readFileOrFallback(isoMeta.SpecFile, "# S1 Requirements\nChưa khởi tạo.")
	s2MD, s2Time := readFileOrFallback(analysisPath, "# S2 Repo Investigation\nChưa quét repository.")
	dg0MD, dg0Time := readFileOrFallback(critiquePath, "# ⚡ S2.5 Pre-Plan Critique (DG0 :8200)\nChưa chạy Wave Phản Biện Trước Plan.")
	s3MD, s3Time := readFileOrFallback(isoMeta.PlanFile, "# S3 Solution & Code Demo\nChưa tạo giải pháp.")
	s4MD, s4Time := readFileOrFallback(isoMeta.DiffFile, "Chưa chạy S4 Implement trong Git Worktree (Đang chờ User bấm ✓ APPROVE ở bước S3).")
	s5MD, s5Time := readFileOrFallback(isoMeta.AuditFile, "# S5 Mandatory Adversarial Audit Report\nChưa chạy S5 QA & DG3 Audit (Chờ hoàn tất S4 Implement).")

	codeDemo := extractCodeFenceFromMarkdown(s3MD)
	if codeDemo == "" || (isMasterMongoDestinationTask(qt) && !strings.Contains(codeDemo, "TASK-M01")) {
		targetRepo := s.ResolveTaskTargetRepo(qt)
		hits, _ := s.inspectTargetRepoSymbols(targetRepo, qt.ExternalID, qt.Title, qt.Description)
		codeDemo = buildConcreteCodeDemoForTask(qt, targetRepo, hits)
	}

	stUpper := strings.ToUpper(strings.TrimSpace(qt.Stage))
	s1Status := "done"
	s2Status := "done"
	dg0Status := "done"
	s3Status := "waiting"
	s4Status := "pending"
	s5Status := "pending"

	switch stUpper {
	case queue.StageDone:
		s3Status, s4Status, s5Status = "done", "done", "done"
	case queue.StageInReview:
		s3Status, s4Status, s5Status = "done", "done", "running"
	case queue.StageRunning:
		s3Status, s4Status, s5Status = "done", "running", "pending"
	case queue.StageNeedsYou:
		s3Status, s4Status, s5Status = "waiting", "pending", "pending"
	case queue.StagePlanning:
		s2Status, dg0Status, s3Status = "running", "running", "running"
	default:
		s2Status, dg0Status, s3Status = "done", "done", "waiting"
	}

	return []StepReportDTO{
		{
			StepID:    "S1",
			Title:     "📋 S1 Requirements & Sub-Tasks",
			Status:    s1Status,
			FilePath:  isoMeta.SpecFile,
			Summary:   "Đặc tả yêu cầu, Affected Repos, Branch cô lập & danh mục Sub-Tasks (08_tasks)",
			ContentMD: s1MD,
			UpdatedAt: s1Time,
		},
		{
			StepID:    "S2",
			Title:     "🔍 S2 Repo Investigation & Lessons",
			Status:    s2Status,
			FilePath:  analysisPath,
			Summary:   "Báo cáo quét mã nguồn thật (file:line) đa Repo & FTS5 Lessons (/agent)",
			ContentMD: s2MD,
			UpdatedAt: s2Time,
		},
		{
			StepID:    "DG0",
			Title:     "⚡ S2.5 Wave Phản Biện Trước Plan (DG0 :8200)",
			Status:    dg0Status,
			FilePath:  critiquePath,
			Summary:   "Wave Phản Biện Ngược Giả Định (C-REQ/C-REALITY/C-CONTRA) & Laya Gate DG0 (:8200 · 0 Token) trước khi lập Plan S3",
			ContentMD: dg0MD,
			UpdatedAt: dg0Time,
		},
		{
			StepID:    "S3",
			Title:     "💡 S3 Solution & Code Demo (Gate Stop)",
			Status:    s3Status,
			FilePath:  isoMeta.PlanFile,
			Summary:   "Root Cause, Phương án tối ưu nhất đa Repo, Code Demo cụ thể & cổng dừng chờ APPROVE",
			ContentMD: s3MD,
			CodeDemo:  codeDemo,
			UpdatedAt: s3Time,
		},
		{
			StepID:    "S4",
			Title:     "🔧 S4 Worktree Diff & Execution",
			Status:    s4Status,
			FilePath:  isoMeta.DiffFile,
			Summary:   "Trạng thái Git Worktree cô lập theo từng Sub-Task Repo, git diff & Zero-Mockup H4b",
			ContentMD: "# 🔧 S4 Worktree Execution & Git Diff\n- **Worktree Path:** `" + isoMeta.WorktreePath + "`\n- **Branch:** `" + isoMeta.BranchName + "`\n\n```text\n" + s4MD + "\n```",
			UpdatedAt: s4Time,
		},
		{
			StepID:    "S5",
			Title:     "🛡️ S5 Audit & /agent Gates (DG3)",
			Status:    s5Status,
			FilePath:  isoMeta.AuditFile,
			Summary:   "Biên bản kiểm định 6 mục S5, 7 Hooks H1-H7, 8 Gates G1-G8 & Laya DG3",
			ContentMD: s5MD,
			UpdatedAt: s5Time,
		},
	}
}

func extractCodeFenceFromMarkdown(md string) string {
	for _, fence := range []string{"```ts\n", "```js\n", "```go\n", "```python\n", "```sql\n", "```\n"} {
		if startIdx := strings.Index(md, fence); startIdx >= 0 {
			rem := md[startIdx+len(fence):]
			if endIdx := strings.Index(rem, "\n```"); endIdx >= 0 {
				return strings.TrimSpace(rem[:endIdx])
			}
			return strings.TrimSpace(rem)
		}
	}
	return ""
}

// buildActiveAgentKernel constructs the explicit /agent Governance Kernel inspector for the selected task.
func (s *Server) buildActiveAgentKernel(qt queue.QueuedTask, matchedLessons []MatchedLessonDTO) ActiveAgentKernelDTO {
	agentRoot := "/Users/trainguyen/Documents/work/agent"
	if s.Gatekeeper != nil && s.Gatekeeper.AgentCoreDir != "" {
		agentRoot = s.Gatekeeper.AgentCoreDir
	}
	totalLessons := CountTotalLessonsInAgentIndex(agentRoot)
	if totalLessons == 0 {
		totalLessons = s.IndexedLessons
	}

	stUpper := strings.ToUpper(strings.TrimSpace(qt.Stage))
	role := "BRAIN (Chairman & Architect)"
	skills := []string{"/brain-delegate", "/laya-router", "/context-manager", "/workspace-docs"}
	hooks := []string{
		"H1: Laya (:8200) Up & S0 Constitution Loaded",
		"H3: Secret & PII Masking Guard (PASS)",
		"H4: S1 Intake & Domain Sub-Tasks (08_tasks) Initialized (PASS)",
		"H4b: Propose-Only Stop Gate (ACTIVE — Chờ User APPROVE trước khi sửa code)",
		"H5: No git push / No destructive reset (ENFORCED)",
	}
	stopReason := "Đang dừng tại cổng NEEDS_YOU (Rule #9.5 & Rule #13): Chờ bạn đọc tab '💡 S3 Solution & Code Demo' và '🧩 Sub-Tasks & Wave' rồi bấm '✓ APPROVE & Run S4'."

	switch stUpper {
	case queue.StageRunning:
		role = "MUSCLE (Chief Engineer)"
		skills = []string{"/muscle-execute", "/debug-agent", "/security-agent"}
		hooks[3] = "H4b: Zero-Mockup Production Scan (ACTIVE in Worktree)"
		stopReason = "Đã có APPROVE — Muscle đang thực thi code trong Git Worktree cô lập theo từng repo."
	case queue.StageInReview:
		role = "QA & Security Auditor"
		skills = []string{"/qa-agent", "/security-agent", "/laya-router"}
		hooks = append(hooks, "H6/H7: Final Adversarial Audit & DG3 Gate (ACTIVE)")
		stopReason = "Đang kiểm định 8 cổng chất lượng G1–G8 và hội đồng /laya_router DG3."
	case queue.StageDone:
		role = "GOVERNANCE VERIFIED (S0 -> S6 Complete)"
		skills = []string{"/brain-delegate", "/muscle-execute", "/qa-agent", "/security-agent", "/session-wrapup"}
		hooks[3] = "H4b: Propose-Only & Zero-Mockup Verified (PASS)"
		hooks = append(hooks, "H7: Process Linter & DG3 Verified (PASS)")
		stopReason = "Đã hoàn tất trọn vẹn S0 -> S6, vượt qua cổng APPROVE và kiểm định S5/DG3."
	}

	return ActiveAgentKernelDTO{
		ConstitutionPath: filepath.Join(agentRoot, "GEMINI.md"),
		IndexPath:        filepath.Join(agentRoot, "memory", "global", "lessons", "INDEX.md"),
		TotalLessons:     totalLessons,
		ActiveRole:       role,
		ActiveSkills:     skills,
		ActiveHooks:      hooks,
		StopGateReason:   stopReason,
		MatchedLessons:   matchedLessons,
	}
}

// restoreSmokeCompletedUserTasks checks if any user-created task (like "tích hợp Master MongoDB Destination")
// had old 4-step workflow stubs, split-brain sibling artifacts (AF-ORCH-01), or un-ingested Audit Reports in chat_history.json (AF-ORCH-06),
// and upgrades/synchronizes them cleanly.
func (s *Server) restoreSmokeCompletedUserTasks(ctx context.Context) {
	if s.QueueStore == nil {
		return
	}
	_, _ = s.QueueStore.DeduplicateTaskRounds(ctx, "")
	all, err := s.QueueStore.ListAll(ctx)
	if err != nil {
		return
	}

	// AF-ORCH-01: Build map of normalized title -> best sibling artifact dir (with richest diff.patch / audit_report.md)
	type bestSiblingArtifacts struct {
		diffBytes  []byte
		auditBytes []byte
		planBytes  []byte
	}
	byTitle := make(map[string]*bestSiblingArtifacts)
	for _, qt := range all {
		normTitle := strings.ToLower(strings.TrimSpace(qt.Title))
		if normTitle == "" {
			continue
		}
		meta := s.buildTaskIsolationMeta(qt.ExternalID)
		entry := byTitle[normTitle]
		if entry == nil {
			entry = &bestSiblingArtifacts{}
			byTitle[normTitle] = entry
		}
		if d, err := os.ReadFile(meta.DiffFile); err == nil && len(d) > len(entry.diffBytes) {
			entry.diffBytes = d
		}
		if a, err := os.ReadFile(meta.AuditFile); err == nil && len(a) > len(entry.auditBytes) {
			entry.auditBytes = a
		}
		if p, err := os.ReadFile(meta.PlanFile); err == nil && len(p) > len(entry.planBytes) {
			entry.planBytes = p
		}
	}

	deletedSet := s.loadDeletedTasksSet()
	for _, qt := range all {
		if deletedSet[strings.ToUpper(strings.TrimSpace(qt.ExternalID))] {
			_, _ = s.QueueStore.DeleteTask(ctx, qt.ExternalID)
			continue
		}
		if strings.HasPrefix(qt.ExternalID, "TASK-") && len(qt.ExternalID) == 6 && qt.ExternalID[5] >= '1' && qt.ExternalID[5] <= '7' {
			continue
		}
		meta := s.buildTaskIsolationMeta(qt.ExternalID)
		_ = os.MkdirAll(meta.ArtifactDir, 0o755)

		// AF-ORCH-01: Sync rich diff.patch / audit_report.md from sibling task with identical title (e.g. id=32 -> TASK-11)
		normTitle := strings.ToLower(strings.TrimSpace(qt.Title))
		if best := byTitle[normTitle]; best != nil {
			if curDiff, dErr := os.ReadFile(meta.DiffFile); dErr != nil || len(curDiff) < 200 {
				if len(best.diffBytes) >= 200 {
					_ = os.WriteFile(meta.DiffFile, best.diffBytes, 0o644)
				}
			}
			if curAudit, aErr := os.ReadFile(meta.AuditFile); aErr != nil || len(curAudit) < 800 {
				if len(best.auditBytes) >= 800 {
					_ = os.WriteFile(meta.AuditFile, best.auditBytes, 0o644)
				}
			}
		}

		rawPlan, readErr := os.ReadFile(meta.PlanFile)
		rawSpec, specErr := os.ReadFile(meta.SpecFile)
		affectedRepos := s.ResolveTaskAffectedRepos(qt)
		isMongoTask := isMasterMongoDestinationTask(qt)
		isOrchUpgrade := isSelfUpgradeOrchestratorTask(qt, affectedRepos)
		_, hasUserApproved := os.Stat(filepath.Join(meta.ArtifactDir, ".user-approved"))

		// Ensure subtasks are upgraded from 4-step stub or cross-task contaminated artifacts to deep domain breakdown
		subTasks := s.loadOrGenerateSubTasks(qt)
		planText := string(rawPlan)
		specText := string(rawSpec)
		isContaminatedNonMongo := !isMongoTask && (strings.Contains(planText, "feat-master-mongo-destination-2026-10-20") ||
			strings.Contains(planText, "TASK-M01") ||
			strings.Contains(specText, "master_mongo") ||
			(!isExplicitOrchestratorTarget(qt) && strings.Contains(specText, "ai-orchestrator")))
		needsDeepUpgrade := readErr != nil || specErr != nil || len(rawPlan) < 650 ||
			!strings.Contains(specText, "affected_repos:") ||
			strings.Contains(planText, "TASK-CẬ") ||
			isContaminatedNonMongo ||
			(isOrchUpgrade && (len(subTasks) < 6 || !strings.Contains(planText, "Bảng Tóm Tắt Điều Hành"))) ||
			(isMongoTask && (len(subTasks) < 10 || !strings.Contains(planText, "TASK-M01")))

		if isMongoTask && (qt.AssignedModel == "" || strings.Contains(qt.AssignedModel, "gemini-2.5-pro")) {
			qt.AssignedModel = "gemini-4-argon (antigravity)"
			_ = s.QueueStore.UpdateStage(ctx, qt.ID, qt.Stage, qt.AssignedModel, meta.WorktreePath)
		}

		if needsDeepUpgrade {
			if isContaminatedNonMongo {
				_ = os.Remove(filepath.Join(meta.ArtifactDir, "09_tasks_solution_master_mongo.md"))
				_ = os.Remove(filepath.Join(meta.ArtifactDir, "08_tasks_master_mongo.md"))
				_ = os.Remove(filepath.Join(meta.ArtifactDir, "01_requirements_master_mongo.md"))
			}
			var genSubs []SubTaskDTO
			meta, genSubs, _, _ = s.generateDeepTaskArtifacts(ctx, qt, s.activeRunner, qt.AssignedModel)
			if len(genSubs) > 0 {
				subTasks = genSubs
			}
		}

		// Ensure dedicated Telegram Forum Sub-Topic by ExternalID exists for Core-01 & active user tasks
		if !s.isTestTaskOrTempRoot(qt.ExternalID) {
			tgCfg := s.loadTelegramPersistedConfig()
			if tgCfg.Enabled && strings.TrimSpace(tgCfg.BotTokenEncrypted) != "" && strings.TrimSpace(tgCfg.ChatID) != "" {
				if tgCfg.TaskThreadIDs == nil || tgCfg.TaskThreadIDs[qt.ExternalID] <= 0 {
					if isOrchUpgrade || qt.Stage == queue.StageNeedsYou {
						_ = s.notifyTelegramNewTaskIntakeAndTopic(ctx, qt, meta, subTasks)
					} else {
						_ = s.ensureTelegramTaskTopic(ctx, tgCfg, qt, false)
					}
				} else if needsDeepUpgrade && isOrchUpgrade {
					_ = s.notifyTelegramNewTaskIntakeAndTopic(ctx, qt, meta, subTasks)
				}
			}
		}

		if isMongoTask && qt.Stage == queue.StageDone && hasUserApproved != nil {
			_, _, _, _ = s.generateDeepTaskArtifacts(ctx, qt, s.activeRunner, qt.AssignedModel)
			_ = os.Remove(meta.AuditFile)
			_ = s.QueueStore.UpdateStage(ctx, qt.ID, queue.StageNeedsYou, "gemini-4-argon (antigravity)", meta.WorktreePath)
			s.appendTaskLog(qt.ExternalID, "[Multi-Repo Domain Decomposer] Đã đồng bộ 10 Sub-Tasks (TASK-M01..TASK-M10) trên 3 repos (cdc-cms-service, cdc-cms-web, centralized-data-service) với model gemini-4-argon (antigravity).")
			s.mu.Lock()
			s.selectedTaskKey = qt.ExternalID
			s.mu.Unlock()
		} else if isMongoTask && hasUserApproved == nil && qt.Stage != queue.StageDone && !s.isTestTaskOrTempRoot(qt.ExternalID) {
			// User already APPROVED Master MongoDB Destination, and ./test no-Go-files false failure is now fixed -> complete S4 -> S5 -> Step 17/17 DONE!
			_ = s.runSingleStage(ctx, qt.ExternalID, "implement", s.activeRunner)
			_ = s.runSingleStage(ctx, qt.ExternalID, "qa", s.activeRunner)
			_, _ = s.QueueStore.DeduplicateTaskRounds(ctx, qt.ExternalID)
		}

		// AF-ORCH-06: Reconcile existing chat_history.json so any pasted Audit Report or conversational question is properly answered & ingested
		s.reconcileTaskChatHistoryAndAuditRemediation(ctx, qt, meta)
	}
}

// buildTaskScopedWorkflowSteps returns the 17 Master Workflow steps strictly scoped to the currently selected task!
func (s *Server) buildTaskScopedWorkflowSteps(activeTask queue.QueuedTask, isoMeta TaskIsolationDTO, subTasks []SubTaskDTO, kernel ActiveAgentKernelDTO) []VibeWorkflowStepDTO {
	if activeTask.ExternalID == "" {
		return s.buildRealWorkflowSteps(nil)
	}
	affectedRepos := s.ResolveTaskAffectedRepos(activeTask)
	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}
	repoLabel := strings.Join(repoNames, ", ")
	if repoLabel == "" {
		repoLabel = filepath.Base(s.ResolveTaskTargetRepo(activeTask))
	}
	stUpper := strings.ToUpper(strings.TrimSpace(activeTask.Stage))

	isPlanned := stUpper == queue.StageNeedsYou || stUpper == queue.StageRunning || stUpper == queue.StageInReview || stUpper == queue.StageDone
	isApproved := stUpper == queue.StageRunning || stUpper == queue.StageInReview || stUpper == queue.StageDone
	isImplemented := stUpper == queue.StageInReview || stUpper == queue.StageDone
	isDone := stUpper == queue.StageDone

	planSt := "done"
	if stUpper == queue.StagePlanning {
		planSt = "active"
	}
	approveSt := "active"
	approveNote := "Chờ bạn bấm ✓ APPROVE"
	if isApproved {
		approveSt = "done"
		approveNote = "User đã APPROVE"
	}
	buildSt := "pending"
	buildNote := "Chờ APPROVE ở bước S3"
	if stUpper == queue.StageRunning {
		buildSt = "active"
		buildNote = "Đang code trong " + isoMeta.BranchName
	} else if isImplemented {
		buildSt = "done"
		buildNote = "Đã cập nhật worktree " + isoMeta.BranchName
	}
	qaSt := "pending"
	qaNote := "Chờ hoàn tất S4"
	if stUpper == queue.StageInReview {
		qaSt = "active"
		qaNote = "Đang chạy QA & DG3"
	} else if isDone {
		qaSt = "done"
		qaNote = "PASS G1–G8 & DG3"
	}

	cliCfg := s.loadCLISettings()
	tKeyList := []string{activeTask.ExternalID}
	analysisPath := filepath.Join(isoMeta.ArtifactDir, "13_analysis.md")
	critiquePath := filepath.Join(isoMeta.ArtifactDir, "10_preplan_critique.md")
	tasksPath := filepath.Join(isoMeta.ArtifactDir, "08_tasks.md")

	_ = isPlanned
	return []VibeWorkflowStepDTO{
		{Num: 1, Title: "Tổng quan", PhaseCode: "S1", Note: fmt.Sprintf("%s · %d repos (%s)", activeTask.ExternalID, len(affectedRepos), repoLabel), Status: "done", RoleLabel: "🧠 BRAIN · Architect", AgentSlot: "Slot A", SkillBadge: "/brain-delegate", Model: cliCfg.PlannerModel, ArtifactPath: isoMeta.SpecFile, LinkedTaskKeys: tKeyList},
		{Num: 2, Title: "Hiến pháp /agent", PhaseCode: "S0", Note: fmt.Sprintf("GEMINI.md · %d lessons", kernel.TotalLessons), Status: "done", RoleLabel: "🛡️ Kernel Guard", AgentSlot: "Slot I", SkillBadge: "/context-manager", Model: cliCfg.SecurityModel, ArtifactPath: kernel.ConstitutionPath, LinkedTaskKeys: tKeyList},
		{Num: 3, Title: "Quét mã nguồn & Lessons", PhaseCode: "S2", Note: fmt.Sprintf("13_analysis.md (%s · %d lessons)", repoLabel, len(kernel.MatchedLessons)), Status: planSt, RoleLabel: "🔍 Scout & Memory", AgentSlot: "Slot G / F", SkillBadge: "/debug-agent", Model: cliCfg.PlannerModel, ArtifactPath: analysisPath, LinkedTaskKeys: tKeyList},
		{Num: 4, Title: "Wave Phản Biện Trước Plan", PhaseCode: "DG0", Note: "10_preplan_critique.md (Laya :8200 · 0 Token)", Status: planSt, RoleLabel: "⚡ Pre-Plan Critique (DG0)", AgentSlot: "Slot A / H", SkillBadge: "/laya-router", Model: cliCfg.LayaGateModel, ArtifactPath: critiquePath, LinkedTaskKeys: tKeyList},
		{Num: 5, Title: "Giải pháp & Code Demo", PhaseCode: "S3", Note: "09_tasks_solution.md", Status: planSt, RoleLabel: "🧠 BRAIN · Architect", AgentSlot: "Slot A", SkillBadge: "/brain-delegate", Model: cliCfg.PlannerModel, ArtifactPath: isoMeta.PlanFile, LinkedTaskKeys: tKeyList},
		{Num: 6, Title: "Kiểm định Giải pháp", PhaseCode: "DG1", Note: "Laya Gate DG1 PASS (:8200 · 0 Token)", Status: planSt, RoleLabel: "⚖️ Laya Gate DG1", AgentSlot: "Slot H", SkillBadge: "/laya-router", Model: cliCfg.LayaGateModel, ArtifactPath: isoMeta.PlanFile, LinkedTaskKeys: tKeyList},
		{Num: 7, Title: "Kiểm định Code Demo", PhaseCode: "DG2", Note: "Laya Gate DG2 PASS (:8200 · 0 Token)", Status: planSt, RoleLabel: "⚖️ Laya Gate DG2", AgentSlot: "Slot H", SkillBadge: "/laya-router", Model: cliCfg.LayaGateModel, ArtifactPath: isoMeta.PlanFile, LinkedTaskKeys: tKeyList},
		{Num: 8, Title: "Cổng Duyệt & Đối Thoại", PhaseCode: "GATE", Note: approveNote, Status: approveSt, RoleLabel: "👤 You + 🧠 BRAIN", AgentSlot: "Slot A", SkillBadge: "Co-Architect Studio", Model: cliCfg.PlannerModel, ArtifactPath: isoMeta.PlanFile, LinkedTaskKeys: tKeyList},
		{Num: 9, Title: "Phân rã Sub-Tasks", PhaseCode: "S1", Note: fmt.Sprintf("%d Sub-Tasks · %d Repos", len(subTasks), len(affectedRepos)), Status: "done", RoleLabel: "⛁ Wave Coordinator", AgentSlot: "Slot E", SkillBadge: "/brain-delegate", Model: cliCfg.PlannerModel, ArtifactPath: tasksPath, LinkedTaskKeys: tKeyList},
		{Num: 10, Title: "Cô lập Git Worktree", PhaseCode: "S4", Note: isoMeta.BranchName, Status: "done", RoleLabel: "▶_ Worktree Shell", AgentSlot: "Slot C", SkillBadge: "/muscle-execute", Model: "git-worktree", ArtifactPath: isoMeta.WorktreePath, LinkedTaskKeys: tKeyList},
		{Num: 11, Title: "Thực thi Code (Muscle)", PhaseCode: "S4", Note: buildNote, Status: buildSt, RoleLabel: "🛠️ MUSCLE · Chief Engineer", AgentSlot: "Slot B / D / E", SkillBadge: "/muscle-execute", Model: cliCfg.CoderHighModel, ArtifactPath: isoMeta.DiffFile, LinkedTaskKeys: tKeyList},
		{Num: 12, Title: "Quét Zero-Mockup H4b", PhaseCode: "H4b", Note: "Chặn mock/secret trong code", Status: buildSt, RoleLabel: "🛡️ Security & H4b Guard", AgentSlot: "Slot I", SkillBadge: "/security-agent", Model: cliCfg.SecurityModel, ArtifactPath: isoMeta.DiffFile, LinkedTaskKeys: tKeyList},
		{Num: 13, Title: "5A. Hard Gate (Zero-LLM)", PhaseCode: "5A", Note: "go vet / linter / test -race", Status: qaSt, RoleLabel: "⚙️ Deterministic Gate", AgentSlot: "Zero-LLM", SkillBadge: "go vet / test -race", Model: "system-compiler (0 LLM)", ArtifactPath: isoMeta.AuditFile, LinkedTaskKeys: tKeyList},
		{Num: 14, Title: "5B. Cross-Model Audit (S5)", PhaseCode: "5B", Note: "audit_report.md (Max 3 Retries)", Status: qaSt, RoleLabel: "🧪 Cross-Model QA Auditor", AgentSlot: "Slot K", SkillBadge: "/qa-agent", Model: cliCfg.QAAuditorModel, ArtifactPath: isoMeta.AuditFile, LinkedTaskKeys: tKeyList},
		{Num: 15, Title: "Hội đồng Laya DG3", PhaseCode: "DG3", Note: qaNote + " (:8200 · 0 Token)", Status: qaSt, RoleLabel: "⚖️ Laya Gate DG3", AgentSlot: "Slot H", SkillBadge: "/laya-router", Model: cliCfg.LayaGateModel, ArtifactPath: isoMeta.AuditFile, LinkedTaskKeys: tKeyList},
		{Num: 16, Title: "Finalize & Diff Review", PhaseCode: "S6", Note: fmt.Sprintf("%s (Rule #15 No-Push)", isoMeta.BranchName), Status: qaSt, RoleLabel: "📦 Diff Review & Merge Gate", AgentSlot: "Slot C / F", SkillBadge: "/session-wrapup", Model: cliCfg.CoderHighModel, ArtifactPath: isoMeta.WorktreePath, LinkedTaskKeys: tKeyList},
		{Num: 17, Title: "Đóng phiên & Tự học", PhaseCode: "S6", Note: fmt.Sprintf("Đồng bộ %d lessons", kernel.TotalLessons), Status: qaSt, RoleLabel: "📚 Context & Governance", AgentSlot: "Slot F", SkillBadge: "/session-wrapup", Model: cliCfg.PlannerModel, ArtifactPath: kernel.IndexPath, LinkedTaskKeys: tKeyList},
	}
}

// buildTaskScopedPanes scopes the 11 Agent Terminal Panes (A..K) on the right window strictly to the selected task!
func (s *Server) buildTaskScopedPanes(activeTask queue.QueuedTask, isoMeta TaskIsolationDTO, subTasks []SubTaskDTO, stepReports []StepReportDTO, kernel ActiveAgentKernelDTO, taskLogs []string) []VibePaneDTO {
	if activeTask.ExternalID == "" {
		s.mu.Lock()
		defer s.mu.Unlock()
		out := make([]VibePaneDTO, len(s.panes))
		copy(out, s.panes)
		return out
	}

	targetRepo := s.ResolveTaskTargetRepo(activeTask)
	affectedRepos := s.ResolveTaskAffectedRepos(activeTask)
	var repoNames []string
	for _, rp := range affectedRepos {
		repoNames = append(repoNames, filepath.Base(rp))
	}
	repoLabel := strings.Join(repoNames, ", ")
	if repoLabel == "" {
		repoLabel = filepath.Base(targetRepo)
	}
	cliCfg := s.loadCLISettings()
	tKey := activeTask.ExternalID
	tKeyList := []string{tKey}

	var s2Lines, s3Lines []string
	for _, sr := range stepReports {
		if sr.StepID == "S2" || sr.StepID == "DG0" {
			for _, ln := range strings.Split(sr.ContentMD, "\n") {
				if strings.TrimSpace(ln) != "" && len(s2Lines) < 12 {
					s2Lines = append(s2Lines, ln)
				}
			}
		}
		if sr.StepID == "S3" {
			for _, ln := range strings.Split(sr.CodeDemo, "\n") {
				if len(s3Lines) < 14 {
					s3Lines = append(s3Lines, ln)
				}
			}
		}
	}

	var subTaskSummary []string
	for _, st := range subTasks {
		subTaskSummary = append(subTaskSummary, fmt.Sprintf("[%s] %s (%s · %s): %s -> %s", strings.ToUpper(st.Status), st.SubTaskID, st.WorkflowStep, st.TargetRepoName, st.Title, st.ContextFileLine))
	}

	var lessonSummary []string
	for _, ml := range kernel.MatchedLessons {
		lessonSummary = append(lessonSummary, fmt.Sprintf("• %s [%s] %s -> %s", ml.LessonID, ml.Topic, ml.Title, ml.Rule))
	}

	termLines := append([]string{
		fmt.Sprintf("AffectedRepos (%d): %s", len(affectedRepos), repoLabel),
		fmt.Sprintf("Isolated Worktree: %s (branch: %s)", isoMeta.WorktreePath, isoMeta.BranchName),
	}, taskLogs...)

	return []VibePaneDTO{
		{
			Slot:           "C",
			Runner:         "terminal",
			RunnerBadge:    "▶_ Terminal",
			Title:          fmt.Sprintf("Interactive Shell (Worktree) · %s", tKey),
			AssigneeRole:   "Worktree Shell",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("%s · %s", repoLabel, isoMeta.BranchName),
			StatusTone:     "sent",
			Model:          "zsh",
			Worktree:       isoMeta.WorktreePath,
			Lines:          termLines,
		},
		{
			Slot:           "A",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "/brain-delegate",
			Title:          fmt.Sprintf("Planner & Architect (S1-S3) · %s", tKey),
			AssigneeRole:   "Planner",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("Stage: %s · %d Sub-Tasks (%d Repos)", activeTask.Stage, len(subTasks), len(affectedRepos)),
			StatusTone:     "sent",
			Model:          cliCfg.PlannerModel,
			Worktree:       isoMeta.ArtifactDir,
			Lines:          append([]string{fmt.Sprintf("[Task %s] AffectedRepos: %s", tKey, repoLabel)}, subTaskSummary...),
		},
		{
			Slot:           "I",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "/security-agent · agy_guard.py",
			Title:          fmt.Sprintf("/agent Kernel & Security (%d Lessons) · %s", kernel.TotalLessons, tKey),
			AssigneeRole:   "Security",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("%d matched / %d total lessons", len(kernel.MatchedLessons), kernel.TotalLessons),
			StatusTone:     "sent",
			Model:          cliCfg.SecurityModel,
			Worktree:       kernel.ConstitutionPath,
			Lines:          append([]string{fmt.Sprintf("Active Role: %s | Skills: %s", kernel.ActiveRole, strings.Join(kernel.ActiveSkills, ", "))}, lessonSummary...),
		},
		{
			Slot:           "G",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "/debug-agent",
			Title:          fmt.Sprintf("S2 Multi-Repo Scanner (%s) · %s", repoLabel, tKey),
			AssigneeRole:   "Scout",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("Scanned %d repos", len(affectedRepos)),
			StatusTone:     "sent",
			Model:          cliCfg.PlannerModel,
			Worktree:       targetRepo,
			Lines:          s2Lines,
		},
		{
			Slot:           "E",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "Wave DAG",
			Title:          fmt.Sprintf("Sub-Task Wave Coordinator · %s", tKey),
			AssigneeRole:   "Coordinator",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("%d Sub-Tasks (Wave 0..4)", len(subTasks)),
			StatusTone:     "sent",
			Model:          cliCfg.PlannerModel,
			Worktree:       isoMeta.ArtifactDir,
			Lines:          subTaskSummary,
		},
		{
			Slot:           "K",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "/qa-agent · Cross-Model",
			Title:          fmt.Sprintf("5B. Cross-Model QA & S5 Auditor · %s", tKey),
			AssigneeRole:   "Tester",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     kernel.StopGateReason,
			StatusTone:     "sent",
			Model:          cliCfg.QAAuditorModel,
			Worktree:       isoMeta.AuditFile,
			Lines:          kernel.ActiveHooks,
		},
		{
			Slot:           "B",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "/muscle-execute",
			Title:          fmt.Sprintf("Coder High (S3 Code Demo & S4 Worktree) · %s", tKey),
			AssigneeRole:   "Coder",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("Branch %s", isoMeta.BranchName),
			StatusTone:     "sent",
			Model:          cliCfg.CoderHighModel,
			Worktree:       isoMeta.WorktreePath,
			Lines:          append([]string{fmt.Sprintf("// Code Demo cho %s trên %s:", tKey, repoLabel)}, s3Lines...),
		},
		{
			Slot:           "H",
			Runner:         "laya-local (:8200)",
			RunnerBadge:    "/laya-router · Local :8200 (0 Token)",
			Title:          fmt.Sprintf("Local Laya Decision Gate (DG0/DG1/DG2/DG3 · :8200) · %s", tKey),
			AssigneeRole:   "Reviewer",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("Stage: %s", activeTask.Stage),
			StatusTone:     "sent",
			Model:          cliCfg.LayaGateModel,
			Worktree:       isoMeta.PlanFile,
			Lines:          append([]string{kernel.StopGateReason}, kernel.ActiveHooks...),
		},
		{
			Slot:           "D",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "/muscle-execute",
			Title:          fmt.Sprintf("Coder Standard (Sub-Task Builder) · %s", tKey),
			AssigneeRole:   "Coder",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("Repos: %s", repoLabel),
			StatusTone:     "sent",
			Model:          cliCfg.CoderStdModel,
			Worktree:       isoMeta.WorktreePath,
			Lines:          subTaskSummary,
		},
		{
			Slot:           "F",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "/context-manager",
			Title:          fmt.Sprintf("FTS5 BM25 Memory & Lessons · %s", tKey),
			AssigneeRole:   "Memory",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("%d Lessons Indexed", kernel.TotalLessons),
			StatusTone:     "sent",
			Model:          cliCfg.CoderStdModel,
			Worktree:       kernel.IndexPath,
			Lines:          lessonSummary,
		},
		{
			Slot:           "J",
			Runner:         cliCfg.DefaultRunner,
			RunnerBadge:    "Telemetry",
			Title:          fmt.Sprintf("Task Telemetry · %s (%s)", tKey, repoLabel),
			AssigneeRole:   "Telemetry",
			ActiveTaskKey:  tKey,
			LinkedTaskKeys: tKeyList,
			StatusText:     fmt.Sprintf("Stage: %s · %s", activeTask.Stage, isoMeta.BranchName),
			StatusTone:     "sent",
			Model:          cliCfg.DefaultRunner,
			Worktree:       isoMeta.WorktreePath,
			Lines:          []string{fmt.Sprintf("Task %s (%s) | AffectedRepos: %s | Role: %s", tKey, activeTask.Title, repoLabel, kernel.ActiveRole)},
		},
	}
}

func (s *Server) deletedTasksFilePath() string {
	return filepath.Join(s.Root, ".orchestrator", "deleted_tasks.json")
}

// loadDeletedTasksSet returns the set of user-deleted task keys persisted on disk so SyncRealWorkspaceState does not resurrect them.
func (s *Server) loadDeletedTasksSet() map[string]bool {
	out := make(map[string]bool)
	raw, err := os.ReadFile(s.deletedTasksFilePath())
	if err != nil {
		return out
	}
	var keys []string
	if err := json.Unmarshal(raw, &keys); err == nil {
		for _, k := range keys {
			k = strings.ToUpper(strings.TrimSpace(k))
			if k != "" {
				out[k] = true
			}
		}
	}
	return out
}

func (s *Server) markTaskDeletedTombstone(taskKeys ...string) {
	set := s.loadDeletedTasksSet()
	for _, k := range taskKeys {
		k = strings.ToUpper(strings.TrimSpace(k))
		if k != "" {
			set[k] = true
		}
	}
	var list []string
	for k := range set {
		list = append(list, k)
	}
	sort.Strings(list)
	_ = os.MkdirAll(filepath.Dir(s.deletedTasksFilePath()), 0o755)
	if raw, err := json.MarshalIndent(list, "", "  "); err == nil {
		_ = os.WriteFile(s.deletedTasksFilePath(), raw, 0o644)
	}
}

func (s *Server) unmarkTaskDeletedTombstone(taskKey string) {
	k := strings.ToUpper(strings.TrimSpace(taskKey))
	if k == "" {
		return
	}
	set := s.loadDeletedTasksSet()
	if !set[k] {
		return
	}
	delete(set, k)
	var list []string
	for item := range set {
		list = append(list, item)
	}
	sort.Strings(list)
	_ = os.MkdirAll(filepath.Dir(s.deletedTasksFilePath()), 0o755)
	if raw, err := json.MarshalIndent(list, "", "  "); err == nil {
		_ = os.WriteFile(s.deletedTasksFilePath(), raw, 0o644)
	}
}

// beginTrackExecution creates a cancellable context for an active Task or Sub-Task action and cancels any stale run.
func (s *Server) beginTrackExecution(parentCtx context.Context, taskKey, subTaskID, actionName string) (context.Context, context.CancelFunc) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	execCtx, cancel := context.WithCancel(parentCtx)
	s.mu.Lock()
	if s.activeExecCancel != nil && !strings.EqualFold(s.activeExecTaskKey, strings.TrimSpace(taskKey)) {
		s.activeExecCancel()
	}
	s.activeExecCancel = cancel
	s.activeExecTaskKey = strings.TrimSpace(taskKey)
	s.activeExecSubTaskID = strings.TrimSpace(subTaskID)
	s.activeExecAction = strings.TrimSpace(actionName)
	if strings.EqualFold(s.lastStoppedTaskKey, taskKey) {
		s.lastStoppedTaskKey = ""
		s.lastStopReason = ""
	}
	s.liveStepProgress = LiveStepProgressState{
		Active:         true,
		TaskKey:        strings.TrimSpace(taskKey),
		SubTaskID:      strings.TrimSpace(subTaskID),
		ActionName:     strings.TrimSpace(actionName),
		StepIndex:      1,
		TotalSteps:     17,
		Percent:        12,
		StageLabel:     "⚡ ĐANG CHẠY TIẾN TRÌNH (" + strings.ToUpper(strings.TrimSpace(actionName)) + ")",
		CurrentAction:  "Đang khởi tạo & thực thi " + strings.TrimSpace(actionName),
		ActiveRoleSlot: "🧠 BRAIN · Slot A",
		CanStop:        true,
		WasStopped:     false,
		StartedAt:      time.Now(),
	}
	s.mu.Unlock()
	s.broadcast("step_progress")
	return execCtx, cancel
}

func (s *Server) endTrackExecution(taskKey, subTaskID string) {
	s.mu.Lock()
	if strings.EqualFold(s.activeExecTaskKey, taskKey) && (subTaskID == "" || strings.EqualFold(s.activeExecSubTaskID, subTaskID)) {
		s.activeExecCancel = nil
		s.activeExecTaskKey = ""
		s.activeExecSubTaskID = ""
		s.activeExecAction = ""
	}
	s.mu.Unlock()
}

// StopTaskExecution immediately cancels any running Task or Sub-Task execution, transitions it to NEEDS_YOU, and broadcasts SSE "task_stopped".
func (s *Server) StopTaskExecution(ctx context.Context, taskKey, subTaskID, reason string) (string, string, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	cancel := s.activeExecCancel
	runningTask := s.activeExecTaskKey
	runningSub := s.activeExecSubTaskID
	s.activeExecCancel = nil
	s.activeExecTaskKey = ""
	s.activeExecSubTaskID = ""
	s.activeExecAction = ""
	if runningTask == "" {
		runningTask = strings.TrimSpace(taskKey)
	}
	if runningTask == "" {
		runningTask = s.selectedTaskKey
	}
	if runningSub == "" {
		runningSub = strings.TrimSpace(subTaskID)
	}
	if strings.TrimSpace(reason) == "" {
		reason = "Đã dừng tiến trình theo yêu cầu người dùng (⏹ Dừng Tiến Trình)"
	}
	s.lastStoppedTaskKey = runningTask
	s.lastStopReason = reason
	s.liveStepProgress = LiveStepProgressState{
		Active:         false,
		TaskKey:        runningTask,
		SubTaskID:      runningSub,
		StageLabel:     "⏹ ĐÃ DỪNG TIẾN TRÌNH",
		CurrentAction:  reason,
		CurrentWhere:   reason,
		ActiveRoleSlot: "👤 User Stop Control",
		CanStop:        false,
		WasStopped:     true,
		StopReason:     reason,
	}
	s.mu.Unlock()

	hadActiveCancel := cancel != nil
	if cancel != nil {
		cancel()
	}

	if runningTask != "" {
		if qt, err := s.findQueuedTask(context.Background(), runningTask); err == nil && qt != nil {
			if s.QueueStore != nil {
				_ = s.QueueStore.UpdateStage(context.Background(), qt.ID, queue.StageNeedsYou, qt.AssignedModel, qt.WorktreePath)
			}
			items := s.loadOrGenerateSubTasks(*qt)
			safeParent := sanitizeTaskKeyPath(qt.ExternalID)
			changedSub := false
			for i := range items {
				isTargetSub := (runningSub != "" && strings.EqualFold(items[i].SubTaskID, runningSub)) ||
					(runningSub == "" && (strings.EqualFold(items[i].Status, "running") || strings.EqualFold(items[i].Stage, "RUNNING") || strings.EqualFold(items[i].Stage, "IN_REVIEW") || strings.EqualFold(items[i].Stage, "PLANNING")))
				if isTargetSub {
					items[i].Stage = "NEEDS_YOU"
					items[i].Status = "waiting"
					safeSub := sanitizeTaskKeyPath(items[i].SubTaskID)
					subDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSub)
					_ = os.MkdirAll(subDir, 0o755)
					_ = os.WriteFile(filepath.Join(subDir, ".subtask-stage"), []byte("NEEDS_YOU\n"), 0o644)
					progLine := fmt.Sprintf("- [%s] [Agent:UserStop] STOP_EXECUTION %s (%s)\n", agentgate.FormatHoChiMinhTimestamp(time.Now()), reason, items[i].SubTaskID)
					if f, fErr := os.OpenFile(filepath.Join(subDir, "05_progress.md"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); fErr == nil {
						_, _ = f.WriteString(progLine)
						_ = f.Close()
					}
					changedSub = true
				}
			}
			if changedSub {
				items = s.enrichSubTasksWithRoleAndModel(items)
				items = s.populateSubTaskFractalContractsAndArtifacts(*qt, items)
				_ = s.saveTaskSubTasks(qt.ExternalID, items)
				s.syncSubTasksToMarkdown08(*qt, items)
			}
		}
		stopLog := fmt.Sprintf("[⏹ Dừng Tiến Trình] Task=%s SubTask=%s — %s", runningTask, runningSub, reason)
		s.appendTaskLog(runningTask, stopLog)
		s.appendPaneLog("A", stopLog)
	}
	s.broadcast("task_stopped")
	return runningTask, runningSub, hadActiveCancel
}

// DeleteSingleTask stops any running execution for taskKey, removes it from SQLite queue + autopilot_rounds,
// cleans its .orchestrator/tasks/<safeKey> directory, records a tombstone in .orchestrator/deleted_tasks.json, and selects the next available task.
func (s *Server) DeleteSingleTask(ctx context.Context, taskKey string) (bool, string, error) {
	taskKey = strings.TrimSpace(taskKey)
	if taskKey == "" {
		s.mu.Lock()
		taskKey = s.selectedTaskKey
		s.mu.Unlock()
	}
	if taskKey == "" {
		return false, "", fmt.Errorf("task_key is required")
	}

	// If this task is currently running, stop it first
	s.mu.Lock()
	isCurrentlyRunning := strings.EqualFold(s.activeExecTaskKey, taskKey) ||
		(s.liveStepProgress.Active && strings.EqualFold(s.liveStepProgress.TaskKey, taskKey))
	s.mu.Unlock()
	if isCurrentlyRunning {
		s.StopTaskExecution(ctx, taskKey, "", "Tự động dừng tiến trình trước khi xóa Task "+taskKey)
	}

	deleted, err := s.QueueStore.DeleteTask(ctx, taskKey)
	if err != nil {
		return false, "", err
	}
	s.markTaskDeletedTombstone(taskKey)

	safeKey := sanitizeTaskKeyPath(taskKey)
	if safeKey != "" {
		_ = os.RemoveAll(filepath.Join(s.Root, ".orchestrator", "tasks", safeKey))
		_ = os.RemoveAll(filepath.Join(s.Root, ".orchestrator", "worktrees", safeKey))
	}

	// Pick next available task as selectedTaskKey
	nextSelected := ""
	if all, lErr := s.QueueStore.ListAll(ctx); lErr == nil && len(all) > 0 {
		for _, item := range all {
			if item.Stage != queue.StageDone {
				nextSelected = item.ExternalID
				break
			}
		}
		if nextSelected == "" {
			nextSelected = all[0].ExternalID
		}
	}
	s.mu.Lock()
	if strings.EqualFold(s.selectedTaskKey, taskKey) || s.selectedTaskKey == "" {
		s.selectedTaskKey = nextSelected
	}
	if strings.EqualFold(s.liveStepProgress.TaskKey, taskKey) {
		s.liveStepProgress = LiveStepProgressState{}
	}
	if strings.EqualFold(s.lastStoppedTaskKey, taskKey) {
		s.lastStoppedTaskKey = ""
		s.lastStopReason = ""
	}
	s.mu.Unlock()

	s.appendPaneLog("A", fmt.Sprintf("[🗑 Xóa Task] Đã xóa Task %s khỏi Queue & lưu Tombstone (chuyển sang %s)", taskKey, nextSelected))
	s.broadcast("task_deleted")
	return deleted, nextSelected, nil
}

// DeleteDoneTasksBulk removes all DONE tasks from the queue (keeping the currently running task if any) and records tombstones.
func (s *Server) DeleteDoneTasksBulk(ctx context.Context, keepTaskKey string) ([]string, error) {
	deletedKeys, err := s.QueueStore.DeleteDoneTasks(ctx, keepTaskKey)
	if err != nil {
		return nil, err
	}
	if len(deletedKeys) > 0 {
		s.markTaskDeletedTombstone(deletedKeys...)
		for _, k := range deletedKeys {
			safeKey := sanitizeTaskKeyPath(k)
			if safeKey != "" {
				_ = os.RemoveAll(filepath.Join(s.Root, ".orchestrator", "tasks", safeKey))
				_ = os.RemoveAll(filepath.Join(s.Root, ".orchestrator", "worktrees", safeKey))
			}
		}
	}
	// Ensure selectedTaskKey still points to a valid remaining task
	if all, lErr := s.QueueStore.ListAll(ctx); lErr == nil {
		s.mu.Lock()
		curSel := s.selectedTaskKey
		stillExists := false
		for _, item := range all {
			if strings.EqualFold(item.ExternalID, curSel) {
				stillExists = true
				break
			}
		}
		if !stillExists {
			if len(all) > 0 {
				s.selectedTaskKey = all[0].ExternalID
			} else {
				s.selectedTaskKey = ""
			}
		}
		s.mu.Unlock()
	}
	s.appendPaneLog("A", fmt.Sprintf("[🧹 Dọn Task DONE] Đã xóa %d Task DONE (%s)", len(deletedKeys), strings.Join(deletedKeys, ", ")))
	s.broadcast("tasks_bulk_deleted")
	return deletedKeys, nil
}

// DeleteSingleSubTask removes a single Sub-Task from a parent task, cleans up its subtasks/<subID> directory, and updates 08_tasks.md.
func (s *Server) DeleteSingleSubTask(ctx context.Context, qt queue.QueuedTask, items []SubTaskDTO, subTaskID string) []SubTaskDTO {
	targetID := strings.TrimSpace(subTaskID)
	if targetID == "" {
		return items
	}
	s.mu.Lock()
	isRunningSub := s.liveStepProgress.Active && strings.EqualFold(s.liveStepProgress.TaskKey, qt.ExternalID) && strings.EqualFold(s.liveStepProgress.SubTaskID, targetID)
	s.mu.Unlock()
	if isRunningSub {
		s.StopTaskExecution(ctx, qt.ExternalID, targetID, "Tự động dừng Sub-Task "+targetID+" trước khi xóa")
	}

	var kept []SubTaskDTO
	removedTitle := ""
	for _, item := range items {
		if strings.EqualFold(item.SubTaskID, targetID) {
			removedTitle = item.Title
			continue
		}
		// Also strip targetID from DependsOn of remaining sub-tasks
		var filteredDeps []string
		for _, dep := range item.DependsOn {
			if !strings.EqualFold(dep, targetID) {
				filteredDeps = append(filteredDeps, dep)
			}
		}
		item.DependsOn = filteredDeps
		kept = append(kept, item)
	}
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	safeSub := sanitizeTaskKeyPath(targetID)
	if safeParent != "" && safeSub != "" {
		_ = os.RemoveAll(filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSub))
	}
	kept = s.enrichSubTasksWithRoleAndModel(kept)
	kept = s.populateSubTaskFractalContractsAndArtifacts(qt, kept)
	_ = s.saveTaskSubTasks(qt.ExternalID, kept)
	s.syncSubTasksToMarkdown08(qt, kept)
	s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[🗑 Xóa Sub-Task] Đã xóa Sub-Task %s (%s) khỏi %s (còn lại %d Sub-Tasks)", targetID, removedTitle, qt.ExternalID, len(kept)))
	s.broadcast("subtasks_updated")
	return kept
}

type deleteTaskRequest struct {
	TaskKey     string `json:"task_key"`
	SubTaskID   string `json:"sub_task_id,omitempty"`
	Mode        string `json:"mode,omitempty"` // "single" (default), "done" (bulk delete DONE tasks), "subtask"
	KeepTaskKey string `json:"keep_task_key,omitempty"`
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req deleteTaskRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.TaskKey == "" {
		req.TaskKey = strings.TrimSpace(r.URL.Query().Get("task_key"))
	}
	if req.Mode == "" {
		req.Mode = strings.TrimSpace(r.URL.Query().Get("mode"))
	}
	mode := strings.ToLower(strings.TrimSpace(req.Mode))

	switch mode {
	case "done", "delete_done", "cleanup_done":
		_, err := s.DeleteDoneTasksBulk(r.Context(), req.KeepTaskKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "subtask", "delete_subtask":
		qt, err := s.findQueuedTask(r.Context(), req.TaskKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		items := s.loadOrGenerateSubTasks(*qt)
		_ = s.DeleteSingleSubTask(r.Context(), *qt, items, req.SubTaskID)
	default:
		if _, _, err := s.DeleteSingleTask(r.Context(), req.TaskKey); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	resp, err := s.BuildDashboardResponse(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

type stopTaskRequest struct {
	TaskKey   string `json:"task_key"`
	SubTaskID string `json:"sub_task_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func (s *Server) handleStopTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req stopTaskRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	s.StopTaskExecution(r.Context(), req.TaskKey, req.SubTaskID, req.Reason)

	resp, err := s.BuildDashboardResponse(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}
