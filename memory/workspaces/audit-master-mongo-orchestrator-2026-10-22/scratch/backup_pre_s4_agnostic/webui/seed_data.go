package webui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

var (
	markdownTaskHeadingRe = regexp.MustCompile(`^##\s*\[([xX ])\]\s*Task\s+(\d+)\s*(.+)$`)
	markdownTaskBulletRe  = regexp.MustCompile(`^\s*-\s*\[([xX ])\]\s*\*\*\[(TASK-\d+)\]\s*([^*]+)\*\*`)
)

type parsedWorkspaceTask struct {
	Key        string
	Title      string
	Done       bool
	Milestone  string
	Epic       string
	Assignee   string
	Complexity string
	DependsOn  []string
	Critical   bool
	Score      float64
}

func (s *Server) workspaceDir() string {
	wsName := strings.TrimSpace(s.WorkspaceName)
	if wsName == "" {
		wsName = "feature-modular-agent-orchestrator-2026-10-20"
	}
	if s.Gatekeeper != nil && s.Gatekeeper.AgentCoreDir != "" {
		cand := filepath.Join(s.Gatekeeper.AgentCoreDir, "memory", "workspaces", wsName)
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
	}
	cand := filepath.Join(filepath.Dir(s.Root), "agent", "memory", "workspaces", wsName)
	if st, err := os.Stat(cand); err == nil && st.IsDir() {
		return cand
	}
	return ""
}

func findFirstMatchingFile(dir, prefix, suffix string) string {
	if dir == "" {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			matches = append(matches, filepath.Join(dir, name))
		}
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

func parseTasksFromWorkspaceMarkdown(mdPath string) []parsedWorkspaceTask {
	if mdPath == "" {
		return nil
	}
	f, err := os.Open(mdPath)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var out []parsedWorkspaceTask
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		var done bool
		var key, rawTitle string

		if mh := markdownTaskHeadingRe.FindStringSubmatch(line); len(mh) == 4 {
			done = strings.EqualFold(mh[1], "x")
			key = "TASK-" + strings.TrimSpace(mh[2])
			rawTitle = strings.TrimSpace(mh[3])
			if idx := strings.Index(rawTitle, ":"); idx >= 0 && idx+1 < len(rawTitle) {
				rawTitle = strings.TrimSpace(rawTitle[idx+1:])
			}
			rawTitle = strings.TrimPrefix(rawTitle, "(")
			rawTitle = strings.TrimSuffix(rawTitle, ")")
			rawTitle = strings.ReplaceAll(rawTitle, "`", "")
		} else if mb := markdownTaskBulletRe.FindStringSubmatch(line); len(mb) == 4 {
			done = strings.EqualFold(mb[1], "x")
			key = strings.TrimSpace(mb[2])
			rawTitle = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(mb[3]), ":"))
		} else {
			continue
		}

		epic, assignee, complexity, milestone := classifyWorkspaceTaskMeta(key, rawTitle)

		var deps []string
		if len(out) > 0 {
			deps = []string{out[len(out)-1].Key}
		}
		score := 190.0 - float64(len(out))*6.0
		out = append(out, parsedWorkspaceTask{
			Key:        key,
			Title:      rawTitle,
			Done:       done,
			Milestone:  milestone,
			Epic:       epic,
			Assignee:   assignee,
			Complexity: complexity,
			DependsOn:  deps,
			Critical:   true,
			Score:      score,
		})
	}
	return out
}

func classifyWorkspaceTaskMeta(key, title string) (epic, assignee, complexity, milestone string) {
	lower := strings.ToLower(title)
	milestone = "M1"
	complexity = "high"
	epic = "CORE"
	assignee = "Coder"

	switch {
	case strings.Contains(lower, "queue") || strings.Contains(lower, "fts5") || strings.Contains(lower, "memory"):
		epic = "DATA"
		assignee = "Planner"
		milestone = "M1"
	case strings.Contains(lower, "scout") || strings.Contains(lower, "sorting"):
		epic = "CLI"
		assignee = "Coder"
		complexity = "medium"
		milestone = "M1"
	case strings.Contains(lower, "wave") || strings.Contains(lower, "worktree") || strings.Contains(lower, "routing"):
		epic = "SRV"
		assignee = "Coder"
		milestone = "M1"
	case strings.Contains(lower, "gate") || strings.Contains(lower, "laya") || strings.Contains(lower, "security"):
		epic = "QA"
		assignee = "Security"
		milestone = "M1"
	case strings.Contains(lower, "af-1") || strings.Contains(lower, "re-audit") || strings.Contains(lower, "khắc phục"):
		epic = "QA"
		assignee = "Tester"
		milestone = "M2"
	case strings.Contains(lower, "ui") || strings.Contains(lower, "webui") || strings.Contains(lower, "mockup") || strings.Contains(lower, "shell"):
		epic = "UX"
		assignee = "Coder"
		milestone = "M2"
	}
	if key == "TASK-1" {
		assignee = "Planner"
	}
	return epic, assignee, complexity, milestone
}

// mapTaskCrossScreenLinks maps a task's Assignee and Stage to its canonical Vibe Studio AgentSlot (A..K)
// and WorkflowStepNum (1..17) so Screen 1 (Wave), Screen 2 (Autopilot), and Screen 3 (Vibe Studio) share 1 entity graph.
func mapTaskCrossScreenLinks(assignee, stage string) (slot string, stepNum int) {
	switch strings.TrimSpace(assignee) {
	case "Planner":
		slot = "A"
	case "Coder":
		slot = "B"
	case "Tester":
		slot = "K"
	case "Reviewer":
		slot = "H"
	case "Security":
		slot = "I"
	default:
		slot = "E"
	}
	switch strings.ToUpper(strings.TrimSpace(stage)) {
	case queue.StageRaw, queue.StagePrioritized:
		stepNum = 1
	case queue.StagePlanning, queue.StageNeedsYou:
		stepNum = 9
	case queue.StageReady, queue.StageRunning, queue.StageBlocked:
		stepNum = 11
	case queue.StageInReview:
		stepNum = 12
	case queue.StageDone:
		stepNum = 16
	default:
		stepNum = 11
	}
	return slot, stepNum
}

// SyncRealWorkspaceState purges any legacy screenshot seed data and synchronizes real tasks and Laya gate
// rounds from the active /agent workspace (08_tasks_*.md and 06_laya_summary_*.json) into SQLite queue.db.
func (s *Server) SyncRealWorkspaceState(ctx context.Context) error {
	if s.QueueStore == nil {
		return nil
	}
	if err := s.QueueStore.PurgeMockSeedData(ctx); err != nil {
		return err
	}

	wsDir := s.workspaceDir()
	tasksFile := findFirstMatchingFile(wsDir, "08_tasks_", ".md")
	parsedTasks := parseTasksFromWorkspaceMarkdown(tasksFile)

	existing, err := s.QueueStore.ListAll(ctx)
	if err != nil {
		return err
	}
	existingByKey := make(map[string]queue.QueuedTask, len(existing))
	for _, item := range existing {
		existingByKey[item.ExternalID] = item
	}

	coderCfg := s.Cfg.Agents["coder"]
	deletedSet := s.loadDeletedTasksSet()
	doneByKey := make(map[string]bool, len(parsedTasks))
	for _, pt := range parsedTasks {
		doneByKey[pt.Key] = pt.Done
	}

	for idx, pt := range parsedTasks {
		if deletedSet[strings.ToUpper(strings.TrimSpace(pt.Key))] {
			continue
		}
		waveIdx := idx / 2
		labels := []string{
			"milestone:" + pt.Milestone,
			"epic:" + pt.Epic,
			"assignee:" + pt.Assignee,
			"active_chain:true",
		}
		for _, dep := range pt.DependsOn {
			labels = append(labels, "depends:"+dep)
		}
		if pt.Critical {
			labels = append(labels, "critical:true")
		}

		stage := queue.StageReady
		if pt.Done {
			stage = queue.StageDone
		} else if len(pt.DependsOn) > 0 {
			allParentDone := true
			for _, dep := range pt.DependsOn {
				if !doneByKey[dep] {
					allParentDone = false
					break
				}
			}
			if !allParentDone {
				stage = queue.StageBlocked
			}
		}

		mdl := coderCfg.ResolveModelForComplexity(pt.Complexity)
		if pt.Assignee == "Planner" {
			mdl = s.Cfg.Agents["pm"].Model
			if mdl == "" {
				mdl = "opus"
			}
		}

		if cur, ok := existingByKey[pt.Key]; ok {
			if cur.Source != "" && cur.Source != "agent-workspace" {
				continue
			}
			targetStage := cur.Stage
			if pt.Done && cur.Stage != queue.StageDone {
				targetStage = queue.StageDone
			}
			_ = s.QueueStore.UpdateWaveAndStage(ctx, cur.ID, waveIdx, targetStage, mdl, pt.Score)
			continue
		}

		id, inserted, enqErr := s.QueueStore.EnqueueRaw(ctx, queue.RawTask{
			ExternalID:  pt.Key,
			Source:      "agent-workspace",
			Title:       pt.Title,
			Description: pt.Title,
			Severity:    "high",
			Labels:      labels,
			SLAHours:    12,
			Complexity:  pt.Complexity,
		})
		if enqErr == nil && (inserted || id > 0) {
			_ = s.QueueStore.UpdateWaveAndStage(ctx, id, waveIdx, stage, mdl, pt.Score)
		}
	}

	// Sync real Laya Decision Gate summary files (06_laya_summary_*.json) into autopilot_rounds idempotently.
	if wsDir != "" {
		entries, readErr := os.ReadDir(wsDir)
		if readErr == nil {
			var summaryFiles []string
			for _, e := range entries {
				if !e.IsDir() && strings.HasPrefix(e.Name(), "06_laya_summary_") && strings.HasSuffix(e.Name(), ".json") {
					summaryFiles = append(summaryFiles, filepath.Join(wsDir, e.Name()))
				}
			}
			sort.Strings(summaryFiles)
			for idx, sf := range summaryFiles {
				already, chkErr := s.QueueStore.HasRoundWithSummaryPath(ctx, sf)
				if chkErr == nil && already {
					continue
				}
				rawBytes, rErr := os.ReadFile(sf)
				if rErr != nil {
					continue
				}
				var layaDoc struct {
					Stage  string `json:"stage"`
					Gate   string `json:"gate"`
					Claims []struct {
						ID       string `json:"id"`
						Decision string `json:"decision"`
					} `json:"claims"`
				}
				if json.Unmarshal(rawBytes, &layaDoc) != nil || layaDoc.Stage == "" {
					continue
				}
				blocking := 0
				advisory := 0
				for _, c := range layaDoc.Claims {
					switch strings.ToUpper(c.Decision) {
					case "BLOCK":
						blocking++
					case "FLAG":
						advisory++
					}
				}
				taskKey := "TASK-1"
				if len(parsedTasks) > 0 {
					mappedIdx := idx % len(parsedTasks)
					taskKey = parsedTasks[mappedIdx].Key
				}
				verdictLower := strings.ToLower(layaDoc.Gate)
				var stageLabel string
				switch strings.ToLower(strings.TrimSpace(layaDoc.Stage)) {
				case "dg0":
					stageLabel = fmt.Sprintf("Round %d · pre-plan critique (DG0 :8200): %s", (idx/3)+1, verdictLower)
				case "dg1":
					stageLabel = fmt.Sprintf("Round %d · plan review: %s", (idx/3)+1, verdictLower)
				case "dg2":
					stageLabel = fmt.Sprintf("Round %d · implement review: %s", (idx/3)+1, verdictLower)
				default:
					stageLabel = fmt.Sprintf("QA round %d: %s", (idx/3)+1, verdictLower)
				}
				_ = s.QueueStore.RecordRound(ctx, queue.AutopilotRound{
					TaskKey:       taskKey,
					RoundNum:      idx + 1,
					Stage:         stageLabel,
					BlockingCount: blocking,
					AdvisoryCount: advisory,
					Turns:         len(layaDoc.Claims),
					DurationSec:   1,
					CostUSD:       0.0,
					GateVerdict:   strings.ToUpper(layaDoc.Gate),
					SummaryPath:   sf,
				})
			}
		}
	}

	return nil
}

func (s *Server) buildRealVibeProjects(activeProject string, allTasks []queue.QueuedTask) []VibeProjectDTO {
	parentDir := filepath.Dir(s.Root)
	entries, err := os.ReadDir(parentDir)
	var projects []VibeProjectDTO

	// Group tasks by AgentSlot so each sidebar agent reflects its real assigned tasks.
	activeTaskBySlot := make(map[string]string)
	taskCountBySlot := make(map[string]int)
	stageBySlot := make(map[string]string)
	for _, qt := range allTasks {
		assignee := extractSingleLabel(qt.Labels, "assignee:", "Coder")
		slot, _ := mapTaskCrossScreenLinks(assignee, qt.Stage)
		taskCountBySlot[slot]++
		if activeTaskBySlot[slot] == "" || qt.Stage != queue.StageDone {
			activeTaskBySlot[slot] = qt.ExternalID
			stageBySlot[slot] = qt.Stage
		}
	}

	agentDefs := []struct {
		Slot string
		Name string
		Role string
	}{
		{Slot: "A", Name: "PM / Planner", Role: "Planner"},
		{Slot: "B", Name: "Coder (Opus/Sonnet)", Role: "Coder"},
		{Slot: "C", Name: "Terminal (Shell)", Role: "Terminal"},
		{Slot: "D", Name: "Priority Sorter", Role: "Planner"},
		{Slot: "E", Name: "Wave Executor", Role: "Coder"},
		{Slot: "F", Name: "Agent Gatekeeper", Role: "Reviewer"},
		{Slot: "G", Name: "Memory FTS5 BM25", Role: "Planner"},
		{Slot: "H", Name: "Laya Router DG0-3 (:8200)", Role: "Reviewer"},
		{Slot: "I", Name: "Security Agent", Role: "Security"},
		{Slot: "J", Name: "CLI & SSE Daemon", Role: "Coder"},
		{Slot: "K", Name: "QA Adversarial", Role: "Tester"},
	}

	var sidebarAgents []VibeSidebarAgentDTO
	for _, ad := range agentDefs {
		tKey := activeTaskBySlot[ad.Slot]
		cnt := taskCountBySlot[ad.Slot]
		stg := stageBySlot[ad.Slot]
		statusText := "Sẵn sàng"
		tone := "idle"
		if ad.Slot == "C" {
			statusText = "Shell thật (" + filepath.Base(s.Root) + ")"
			tone = "running"
		} else if tKey != "" {
			switch stg {
			case queue.StageDone:
				statusText = fmt.Sprintf("Hoàn tất · %s (%d task)", tKey, cnt)
				tone = "idle"
			case queue.StageRunning:
				statusText = fmt.Sprintf("Đang chạy · %s", tKey)
				tone = "running"
			case queue.StageBlocked:
				statusText = fmt.Sprintf("Chờ mở khóa · %s", tKey)
				tone = "waiting"
			default:
				statusText = fmt.Sprintf("Phụ trách · %s (%d task)", tKey, cnt)
				tone = "sent"
			}
		}
		sidebarAgents = append(sidebarAgents, VibeSidebarAgentDTO{
			Slot:          ad.Slot,
			Name:          ad.Name,
			Role:          ad.Role,
			Host:          "local",
			Status:        statusText,
			Tone:          tone,
			ActiveTaskKey: tKey,
		})
	}

	if err == nil {
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			candPath := filepath.Join(parentDir, e.Name())
			_, hasGit := os.Stat(filepath.Join(candPath, ".git"))
			_, hasManifest := os.Stat(filepath.Join(candPath, ".agent-project.json"))
			_, hasGemini := os.Stat(filepath.Join(candPath, "GEMINI.md"))
			if hasGit != nil && hasManifest != nil && hasGemini != nil {
				continue
			}
			code := strings.ToUpper(e.Name())
			if len(code) > 2 {
				code = code[:2]
			}
			isCurrent := e.Name() == activeProject
			proj := VibeProjectDTO{
				Code:       code,
				Name:       e.Name(),
				Path:       candPath,
				Active:     isCurrent,
				BadgeCount: 0,
				IdleNote:   "Workspace trên đĩa",
			}
			if isCurrent {
				proj.BadgeCount = len(sidebarAgents)
				proj.IdleNote = ""
				proj.Agents = sidebarAgents
			}
			projects = append(projects, proj)
		}
	}

	if len(projects) == 0 {
		projects = append(projects, VibeProjectDTO{
			Code:       "AO",
			Name:       activeProject,
			Path:       s.Root,
			Active:     true,
			BadgeCount: len(sidebarAgents),
			Agents:     sidebarAgents,
		})
	}
	return projects
}

func (s *Server) buildRealWorkflowSteps(allTasks []queue.QueuedTask) []VibeWorkflowStepDTO {
	wsDir := s.workspaceDir()
	reqFile := findFirstMatchingFile(wsDir, "01_requirements_", ".md")
	analysisFile := findFirstMatchingFile(wsDir, "13_analysis_", ".md")
	solFile := findFirstMatchingFile(wsDir, "09_tasks_solution_", ".md")
	dg1File := findFirstMatchingFile(wsDir, "06_laya_summary_dg1_", ".json")
	dg2File := findFirstMatchingFile(wsDir, "06_laya_summary_dg2_", ".json")
	dg3File := findFirstMatchingFile(wsDir, "06_laya_summary_dg3_", ".json")
	tasksFile := findFirstMatchingFile(wsDir, "08_tasks_", ".md")
	valFile := findFirstMatchingFile(wsDir, "06_validation_", ".md")
	auditFile := findFirstMatchingFile(wsDir, "audit_report_", ".md")
	repFile := findFirstMatchingFile(wsDir, "11_report_", ".md")
	progFile := findFirstMatchingFile(wsDir, "05_progress_", ".md")
	manifestFile := filepath.Join(s.Root, ".agent-project.json")
	binFile := filepath.Join(s.Root, "bin", "orchestrator")

	doneTasks := 0
	linkedByStep := make(map[int][]string)
	for _, qt := range allTasks {
		if qt.Stage == queue.StageDone {
			doneTasks++
		}
		assignee := extractSingleLabel(qt.Labels, "assignee:", "Coder")
		_, stepNum := mapTaskCrossScreenLinks(assignee, qt.Stage)
		linkedByStep[stepNum] = append(linkedByStep[stepNum], qt.ExternalID)
	}
	allTaskKeys := make([]string, 0, len(allTasks))
	for _, qt := range allTasks {
		allTaskKeys = append(allTaskKeys, qt.ExternalID)
	}

	stepStatusFromPath := func(p string) (string, string) {
		if p == "" {
			return "pending", "chưa tạo"
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return "done", filepath.Base(p)
		}
		return "pending", "chưa có"
	}

	s1St, s1Note := stepStatusFromPath(reqFile)
	s3St, s3Note := stepStatusFromPath(analysisFile)
	s4St, s4Note := stepStatusFromPath(solFile)
	s6St, s6Note := stepStatusFromPath(dg1File)
	s7St, s7Note := stepStatusFromPath(dg2File)
	s8St, s8Note := stepStatusFromPath(manifestFile)
	s9St, s9Note := stepStatusFromPath(progFile)
	s10St, s10Note := stepStatusFromPath(tasksFile)
	s12St, s12Note := stepStatusFromPath(valFile)
	s13St, s13Note := stepStatusFromPath(auditFile)
	s14St, s14Note := stepStatusFromPath(repFile)
	s15St, s15Note := stepStatusFromPath(dg3File)
	s16St, s16Note := stepStatusFromPath(binFile)

	buildStatus := "active"
	if len(allTasks) > 0 && doneTasks == len(allTasks) {
		buildStatus = "done"
	}
	buildNote := fmt.Sprintf("%d/%d tasks DONE", doneTasks, len(allTasks))

	steps := []VibeWorkflowStepDTO{
		{Num: 1, Title: "Tổng quan", PhaseCode: "S1", Note: s1Note, Status: s1St, ArtifactPath: reqFile, LinkedTaskKeys: allTaskKeys},
		{Num: 2, Title: "Quy tắc", PhaseCode: "S0", Note: fmt.Sprintf("GEMINI.md (%d lessons)", s.IndexedLessons), Status: "done", ArtifactPath: "GEMINI.md", LinkedTaskKeys: allTaskKeys},
		{Num: 3, Title: "Bối cảnh", PhaseCode: "S2", Note: s3Note, Status: s3St, ArtifactPath: analysisFile, LinkedTaskKeys: linkedByStep[1]},
		{Num: 4, Title: "Kiến trúc", PhaseCode: "S3", Note: s4Note, Status: s4St, ArtifactPath: solFile, LinkedTaskKeys: allTaskKeys},
		{Num: 5, Title: "Dữ liệu & luồng", PhaseCode: "S3", Note: fmt.Sprintf("queue.db (%d tasks)", len(allTasks)), Status: "done", ArtifactPath: solFile, LinkedTaskKeys: allTaskKeys},
		{Num: 6, Title: "An ninh", PhaseCode: "DG1", Note: s6Note, Status: s6St, ArtifactPath: dg1File, LinkedTaskKeys: allTaskKeys},
		{Num: 7, Title: "Giao diện", PhaseCode: "DG2", Note: s7Note, Status: s7St, ArtifactPath: dg2File, LinkedTaskKeys: allTaskKeys},
		{Num: 8, Title: "Kế hoạch lớn", PhaseCode: "S4", Note: s8Note, Status: s8St, ArtifactPath: manifestFile, LinkedTaskKeys: allTaskKeys},
		{Num: 9, Title: "Kế hoạch việc", PhaseCode: "S1-S6", Note: s9Note, Status: s9St, ArtifactPath: progFile, LinkedTaskKeys: allTaskKeys},
		{Num: 10, Title: "Phân rã việc", PhaseCode: "S1", Note: s10Note, Status: s10St, ArtifactPath: tasksFile, LinkedTaskKeys: allTaskKeys},
		{Num: 11, Title: "Xây dựng", PhaseCode: "S4", Note: buildNote, Status: buildStatus, ArtifactPath: tasksFile, LinkedTaskKeys: linkedByStep[11]},
		{Num: 12, Title: "Xác minh", PhaseCode: "S4-G3", Note: s12Note, Status: s12St, ArtifactPath: valFile, LinkedTaskKeys: linkedByStep[12]},
		{Num: 13, Title: "Tài liệu", PhaseCode: "S6", Note: s14Note, Status: s14St, ArtifactPath: repFile, LinkedTaskKeys: allTaskKeys},
		{Num: 14, Title: "Đánh giá", PhaseCode: "S5", Note: s13Note, Status: s13St, ArtifactPath: auditFile, LinkedTaskKeys: allTaskKeys},
		{Num: 15, Title: "Chuẩn bị phát hành", PhaseCode: "DG3", Note: s15Note, Status: s15St, ArtifactPath: dg3File, LinkedTaskKeys: allTaskKeys},
		{Num: 16, Title: "Triển khai", PhaseCode: "S6", Note: s16Note, Status: s16St, ArtifactPath: binFile, LinkedTaskKeys: linkedByStep[16]},
		{Num: 17, Title: "Vận hành & cải tiến", PhaseCode: "S6", Note: fmt.Sprintf("INDEX.md (%d lessons)", s.IndexedLessons), Status: "done", ArtifactPath: "memory/global/lessons/INDEX.md", LinkedTaskKeys: allTaskKeys},
	}
	return steps
}

func (s *Server) buildInitialRealPanes(ctx context.Context) []VibePaneDTO {
	var allTasks []queue.QueuedTask
	if s.QueueStore != nil {
		allTasks, _ = s.QueueStore.ListAll(ctx)
	}

	linkedBySlot := make(map[string][]string)
	activeTaskBySlot := make(map[string]string)
	for _, qt := range allTasks {
		assignee := extractSingleLabel(qt.Labels, "assignee:", "Coder")
		slot, _ := mapTaskCrossScreenLinks(assignee, qt.Stage)
		linkedBySlot[slot] = append(linkedBySlot[slot], qt.ExternalID)
		if activeTaskBySlot[slot] == "" || qt.Stage != queue.StageDone {
			activeTaskBySlot[slot] = qt.ExternalID
		}
	}

	// Run real git status & git log for Slot C (Terminal)
	var termLines []string
	if out, err := s.executeTerminalCommand(ctx, "git status -sb && git log -n 3 --oneline && go version"); err == nil {
		for _, ln := range strings.Split(out, "\n") {
			if strings.TrimSpace(ln) != "" {
				termLines = append(termLines, ln)
			}
		}
	}
	if len(termLines) == 0 {
		termLines = []string{" Working directory: " + s.Root}
	}

	var gates []string
	if s.Gatekeeper != nil {
		for _, g := range s.Gatekeeper.EvaluateLiveGateBadges(s.Root, s.WorkspaceName, s.IndexedLessons) {
			gates = append(gates, fmt.Sprintf("• %s: %s", g.Name, g.Status))
		}
	}
	if len(gates) == 0 {
		gates = []string{"• Gatekeeper initialized."}
	}

	// Read real tail lines from 05_progress_*.md
	var progressTail []string
	if progPath := findFirstMatchingFile(s.workspaceDir(), "05_progress_", ".md"); progPath != "" {
		if raw, err := os.ReadFile(progPath); err == nil {
			lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
			for i := len(lines) - 1; i >= 0 && len(progressTail) < 3; i-- {
				ln := strings.TrimSpace(lines[i])
				if strings.HasPrefix(ln, "- [") {
					progressTail = append([]string{ln}, progressTail...)
				}
			}
		}
	}
	if len(progressTail) == 0 {
		progressTail = []string{"• Chưa có dòng log tiến độ trong workspace."}
	}

	pmModel := s.Cfg.Agents["pm"].Model
	if pmModel == "" {
		pmModel = "opus"
	}
	coderHigh := s.Cfg.Agents["coder"].ResolveModelForComplexity("high")
	coderStd := s.Cfg.Agents["coder"].ResolveModelForComplexity("low")

	slotTaskOrDefault := func(slot string, fallbackIdx int) string {
		if k := activeTaskBySlot[slot]; k != "" {
			return k
		}
		if len(allTasks) > 0 {
			return allTasks[fallbackIdx%len(allTasks)].ExternalID
		}
		return "TASK-1"
	}

	return []VibePaneDTO{
		{
			Slot:           "C",
			Runner:         "terminal",
			RunnerBadge:    ">_",
			Title:          "Terminal (Shell thật)",
			AssigneeRole:   "Terminal",
			ActiveTaskKey:  slotTaskOrDefault("E", 0),
			LinkedTaskKeys: linkedBySlot["B"],
			StatusText:     "Shell · " + filepath.Base(s.Root),
			StatusTone:     "running",
			Model:          "sh",
			Compact:        false,
			Worktree:       s.Root,
			Lines:          termLines,
		},
		{
			Slot:           "A",
			Runner:         "claude",
			RunnerBadge:    "✦",
			Title:          "PM / Planner (" + slotTaskOrDefault("A", 0) + ")",
			AssigneeRole:   "Planner",
			ActiveTaskKey:  slotTaskOrDefault("A", 0),
			LinkedTaskKeys: linkedBySlot["A"],
			StatusText:     "Phụ trách · " + slotTaskOrDefault("A", 0),
			StatusTone:     "idle",
			Model:          pmModel,
			Compact:        false,
			Worktree:       s.workspaceDir(),
			Lines:          progressTail,
		},
		{
			Slot:           "I",
			Runner:         "claude",
			RunnerBadge:    "✦",
			Title:          "Security Gate (" + slotTaskOrDefault("I", 3) + ")",
			AssigneeRole:   "Security",
			ActiveTaskKey:  slotTaskOrDefault("I", 3),
			LinkedTaskKeys: linkedBySlot["I"],
			StatusText:     "Giám sát H3/H5 · " + slotTaskOrDefault("I", 3),
			StatusTone:     "idle",
			Model:          pmModel,
			Compact:        false,
			Worktree:       s.Root,
			Lines:          gates,
		},
		{
			Slot:           "G",
			Runner:         "antigravity",
			RunnerBadge:    "◎",
			Title:          "Memory FTS5 (" + slotTaskOrDefault("A", 0) + ")",
			AssigneeRole:   "Planner",
			ActiveTaskKey:  slotTaskOrDefault("A", 0),
			LinkedTaskKeys: linkedBySlot["A"],
			StatusText:     fmt.Sprintf("%d lessons indexed", s.IndexedLessons),
			StatusTone:     "idle",
			Model:          coderStd,
			Compact:        true,
			Worktree:       filepath.Join(s.Root, ".orchestrator", "memory.db"),
			Lines: []string{
				fmt.Sprintf("SQLite FTS5 BM25 corpus: %d bài học từ /agent/memory/global/lessons/topics.", s.IndexedLessons),
				"Gõ từ khóa bất kỳ vào ô dưới để truy vấn trực tiếp FTS5 BM25.",
			},
		},
		{
			Slot:           "E",
			Runner:         "antigravity",
			RunnerBadge:    "◎",
			Title:          "Wave Executor (" + slotTaskOrDefault("B", 2) + ")",
			AssigneeRole:   "Coder",
			ActiveTaskKey:  slotTaskOrDefault("B", 2),
			LinkedTaskKeys: linkedBySlot["B"],
			StatusText:     "Điều phối · " + slotTaskOrDefault("B", 2),
			StatusTone:     "sent",
			Model:          coderHigh,
			Compact:        true,
			Worktree:       filepath.Join(s.Root, ".orchestrator", "queue.db"),
			Lines: []string{
				fmt.Sprintf("Đang quản lý %d task thật trong SQLite queue.db.", len(allTasks)),
				"Bấm '▶ Chạy Wave Kế Tiếp' để thực thi kiểm định và mở khóa Wave phụ thuộc.",
			},
		},
		{
			Slot:           "K",
			Runner:         "claude",
			RunnerBadge:    "✦",
			Title:          "QA Gatekeeper (" + slotTaskOrDefault("K", 4) + ")",
			AssigneeRole:   "Tester",
			ActiveTaskKey:  slotTaskOrDefault("K", 4),
			LinkedTaskKeys: linkedBySlot["K"],
			StatusText:     "Kiểm định · " + slotTaskOrDefault("K", 4),
			StatusTone:     "idle",
			Model:          coderStd,
			Compact:        false,
			Worktree:       s.Root,
			Lines: []string{
				"Thực thi kiểm định thực tế: ScanProductionFiles (Zero-Mockup) + go vet ./...",
				fmt.Sprintf("Workspace kiểm định: %s", s.WorkspaceName),
			},
		},
		{
			Slot:           "B",
			Runner:         "claude",
			RunnerBadge:    "✦",
			Title:          "Coder Worker (" + slotTaskOrDefault("B", 1) + ")",
			AssigneeRole:   "Coder",
			ActiveTaskKey:  slotTaskOrDefault("B", 1),
			LinkedTaskKeys: linkedBySlot["B"],
			StatusText:     "Phụ trách · " + slotTaskOrDefault("B", 1),
			StatusTone:     "idle",
			Model:          coderHigh,
			Compact:        true,
			Worktree:       s.Root,
			Lines: []string{
				fmt.Sprintf("Dynamic Model Routing: high -> %s | medium/low -> %s", coderHigh, coderStd),
			},
		},
		{
			Slot:           "H",
			Runner:         "laya-local",
			RunnerBadge:    "❖",
			Title:          "Laya Router (" + slotTaskOrDefault("H", 3) + ")",
			AssigneeRole:   "Reviewer",
			ActiveTaskKey:  slotTaskOrDefault("H", 3),
			LinkedTaskKeys: linkedBySlot["H"],
			StatusText:     "Decision Gate DG0-DG3 (:8200 · 0 Token)",
			StatusTone:     "idle",
			Model:          "convaiinnovations/laya (local :8200)",
			Compact:        true,
			Worktree:       s.workspaceDir(),
			Lines: []string{
				"Đồng bộ kết quả thật từ 06_laya_summary_dg0/dg1/dg2/dg3_*.json trên local :8200 (0 LLM token).",
			},
		},
		{
			Slot:           "D",
			Runner:         "claude",
			RunnerBadge:    "✦",
			Title:          "Priority Sorter (" + slotTaskOrDefault("B", 1) + ")",
			AssigneeRole:   "Planner",
			ActiveTaskKey:  slotTaskOrDefault("B", 1),
			LinkedTaskKeys: linkedBySlot["A"],
			StatusText:     "Queue Triage",
			StatusTone:     "idle",
			Model:          coderStd,
			Compact:        true,
			Worktree:       filepath.Join(s.Root, ".orchestrator", "queue.db"),
			Lines: []string{
				"Sorter chấm điểm ưu tiên từ Severity, SLA, Blast Radius (FTS5) và đẩy vào Wave DAG.",
			},
		},
		{
			Slot:           "F",
			Runner:         "claude",
			RunnerBadge:    "✦",
			Title:          "Governance Linter (" + slotTaskOrDefault("I", 3) + ")",
			AssigneeRole:   "Reviewer",
			ActiveTaskKey:  slotTaskOrDefault("I", 3),
			LinkedTaskKeys: linkedBySlot["I"],
			StatusText:     "Rule #0-#22",
			StatusTone:     "idle",
			Model:          pmModel,
			Compact:        true,
			Worktree:       s.workspaceDir(),
			Lines: []string{
				"Kiểm tra verify_governance.py & bất biến 05_progress_*.md.",
			},
		},
		{
			Slot:           "J",
			Runner:         "antigravity",
			RunnerBadge:    "▣",
			Title:          "SSE Event Bus (" + slotTaskOrDefault("B", 5) + ")",
			AssigneeRole:   "Coder",
			ActiveTaskKey:  slotTaskOrDefault("B", 5),
			LinkedTaskKeys: linkedBySlot["B"],
			StatusText:     "Streaming /api/stream",
			StatusTone:     "running",
			Model:          coderStd,
			Compact:        true,
			Worktree:       s.Root,
			Lines: []string{
				"Phát sự kiện trực tiếp từ bus.Bus và SQLite queue.db tới cả 3 màn hình.",
			},
		},
	}
}

