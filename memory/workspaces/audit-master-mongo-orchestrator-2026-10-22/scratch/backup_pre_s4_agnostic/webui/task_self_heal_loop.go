package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/agentgate"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

// shouldContinuePersistentBugFixLoop implements persistent bug-hunting retry logic:
// Even after baseMaxLoops (3 retries), if S5 discovers a new actionable TASK-BUG-* or the failure signature
// progresses (curFailureSig != prevFailureSig), keep retrying up to hardCapLoops until the task is 100% DONE.
func shouldContinuePersistentBugFixLoop(loop, baseMaxLoops, hardCapLoops int, newBugSubTaskCreated bool, prevFailureSig, curFailureSig string) bool {
	if hardCapLoops <= 0 {
		hardCapLoops = 8
	}
	if baseMaxLoops <= 0 {
		baseMaxLoops = 3
	}
	if loop >= hardCapLoops {
		return false
	}
	if loop < baseMaxLoops {
		return true
	}
	if newBugSubTaskCreated {
		return true
	}
	if curFailureSig != "" && prevFailureSig != "" && curFailureSig != prevFailureSig {
		return true
	}
	return false
}

// hasUnapplyableSubTaskPatch checks if any Sub-Task directory holds an explicitly corrupt diff.patch
// that lacks valid unified diff headers (`--- a/` and `+++ b/`), indicating an unremediable patch input.
func (s *Server) hasUnapplyableSubTaskPatch(taskKey string, subTasks []SubTaskDTO) bool {
	safeParent := sanitizeTaskKeyPath(taskKey)
	for _, st := range subTasks {
		safeSub := sanitizeTaskKeyPath(st.SubTaskID)
		p := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSub, "diff.patch")
		raw, err := os.ReadFile(p)
		if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
			continue
		}
		txt := string(raw)
		if !strings.Contains(txt, "--- ") || !strings.Contains(txt, "+++ ") {
			return true
		}
	}
	return false
}

// diagnoseAndCreateS5BugSubTask inspects S5 QA results when a task is not yet DONE, creates an explicit TASK-BUG-XX
// Sub-Task if a new defect is identified, and prepares S4 auto-remediation without requiring a 2nd Human APPROVE (Rule #9.8d).
func (s *Server) diagnoseAndCreateS5BugSubTask(ctx context.Context, qt queue.QueuedTask, isoMeta TaskIsolationDTO) (bool, string, string) {
	subTasks := s.loadOrGenerateSubTasks(qt)
	var missingIDs []string
	var firstMissing *SubTaskDTO
	bugCount := 0
	for i := range subTasks {
		if strings.HasPrefix(strings.ToUpper(subTasks[i].SubTaskID), "TASK-BUG-") {
			bugCount++
		}
		if okDisk, _ := s.VerifySubTaskCodeOnDisk(ctx, subTasks[i]); !okDisk {
			missingIDs = append(missingIDs, subTasks[i].SubTaskID)
			if firstMissing == nil {
				firstMissing = &subTasks[i]
			}
		}
	}
	rawAudit, _ := os.ReadFile(isoMeta.AuditFile)
	auditStr := string(rawAudit)
	rawLogs, _ := os.ReadFile(isoMeta.LogFile)
	taskLogs := string(rawLogs)
	hasCorruptPatch := s.hasUnapplyableSubTaskPatch(qt.ExternalID, subTasks)
	hasHardGateFailureInLogs := strings.Contains(auditStr, "FAILED") || strings.Contains(taskLogs, "[5A Hard Gate FAIL]") || strings.Contains(taskLogs, "FAILED (EV-")
	failureSig := fmt.Sprintf("missing=%s|corrupt=%t|hardfail=%t|bugs=%d", strings.Join(missingIDs, ","), hasCorruptPatch, hasHardGateFailureInLogs, bugCount)

	if hasCorruptPatch {
		return false, "", failureSig
	}

	remMarkerPath := filepath.Join(isoMeta.ArtifactDir, ".audit-remediation-ready")
	_, remMarkerErr := os.Stat(remMarkerPath)

	// Ensure .user-approved and .audit-remediation-ready are set so the next S4 pass synthesizes & applies the fix
	nowStamp := []byte(time.Now().Format(time.RFC3339) + "\n")
	_ = os.MkdirAll(isoMeta.ArtifactDir, 0o755)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, ".user-approved"), nowStamp, 0o644)
	_ = os.WriteFile(remMarkerPath, nowStamp, 0o644)

	if remMarkerErr != nil && !hasHardGateFailureInLogs {
		// Enabling .audit-remediation-ready unlocks existing remediation Sub-Tasks on the next loop
		return true, "S5-REMEDIATE-EXISTING", failureSig
	}

	if firstMissing == nil && !hasHardGateFailureInLogs {
		return false, "", failureSig
	}

	bugID := fmt.Sprintf("TASK-BUG-%02d", bugCount+1)
	for _, st := range subTasks {
		if strings.EqualFold(st.SubTaskID, bugID) {
			return false, bugID, failureSig
		}
	}
	if bugCount >= 4 {
		return false, "", failureSig
	}

	targetRepo := s.ResolveTaskTargetRepo(qt)
	targetRepoName := filepath.Base(targetRepo)
	ctxLine := targetRepoName + "/internal/"
	title := fmt.Sprintf("Khắc phục tự động lỗi phát hiện tại S5 Audit cho %s", qt.ExternalID)
	if firstMissing != nil {
		targetRepo = firstMissing.TargetRepo
		targetRepoName = firstMissing.TargetRepoName
		ctxLine = firstMissing.ContextFileLine
		title = fmt.Sprintf("Auto-Fix S5 Defect cho %s (%s trên `%s`)", firstMissing.SubTaskID, firstMissing.TargetFiles, targetRepoName)
	}
	cliCfg := s.loadCLISettings()
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	bugSub := SubTaskDTO{
		SubTaskID:       bugID,
		ParentKey:       qt.ExternalID,
		GroupTitle:      fmt.Sprintf("Nhóm Auto-Heal S5: Khắc phục lỗi phát sinh (`%s`)", targetRepoName),
		WaveIndex:       0,
		WorkflowStep:    "Wave 0 (S5 Auto-Heal)",
		Title:           title,
		TargetRepo:      targetRepo,
		TargetRepoName:  targetRepoName,
		WorktreePath:    filepath.Join(".orchestrator", "worktrees", safeParent, targetRepoName),
		TargetFiles:     ctxLine,
		ContextFileLine: ctxLine,
		Description:     fmt.Sprintf("Tự động tạo từ S5 Adversarial Audit (Rule #9.8d) để sửa dứt điểm lỗi %s mà không dừng chờ APPROVE lần 2.", failureSig),
		TestDoD:         fmt.Sprintf("Pass VerifySubTaskCodeOnDisk & targeted build trên `%s`.", targetRepoName),
		Stage:           "APPROVED",
		Status:          "running",
		Assignee:        "Coder",
		Model:           cliCfg.CoderHighModel,
	}
	subTasks = append(subTasks, bugSub)
	safeBug := sanitizeTaskKeyPath(bugID)
	bugDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeBug)
	_ = os.MkdirAll(bugDir, 0o755)
	_ = os.WriteFile(filepath.Join(bugDir, ".user-approved"), nowStamp, 0o644)
	_ = os.WriteFile(filepath.Join(bugDir, ".subtask-stage"), []byte("APPROVED\n"), 0o644)

	subTasks = s.enrichSubTasksWithRoleAndModel(subTasks)
	subTasks = s.populateSubTaskFractalContractsAndArtifacts(qt, subTasks)
	_ = s.saveTaskSubTasks(qt.ExternalID, subTasks)
	s.syncSubTasksToMarkdown08(qt, subTasks)
	s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[S5 -> S4 Auto-Bug Sub-Task] Phát hiện lỗi tại S5 -> Đã tự động tạo %s (%s) và tiếp tục vòng lặp sửa lỗi tự chủ cho tới khi DONE!", bugID, title))
	return true, bugID, failureSig
}

// runAutonomousS4S5WithSelfHealing implements Persistent Autonomous Self-Healing (Rule #9.8d & TASK-CHAT-06):
// - Executes S4 -> S5 -> DG3.
// - If S5 finds a defect, diagnoses & creates an explicit TASK-BUG-XX Sub-Task and continues retrying (even past 3 loops
//   via shouldContinuePersistentBugFixLoop as long as new TASK-BUG-XX is discovered or error signature progresses) until DONE.
func (s *Server) runAutonomousS4S5WithSelfHealing(ctx context.Context, taskKey, activeRunner string, stepDelay time.Duration) error {
	cliCfg := s.loadCLISettings()
	maxSelfHealLoops := cliCfg.MaxSelfHealRetries
	if maxSelfHealLoops <= 0 {
		maxSelfHealLoops = 3
	}
	hardCapLoops := maxSelfHealLoops + 5
	if s.QueueStore != nil {
		_, _ = s.QueueStore.DeduplicateTaskRounds(ctx, taskKey)
	}

	prevFailureSig := ""
	for loop := 1; loop <= hardCapLoops; loop++ {
		if ctx.Err() != nil {
			return nil
		}
		if err := s.runSingleStageWithDelay(ctx, taskKey, "implement", activeRunner, stepDelay); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
		if err := s.runSingleStageWithDelay(ctx, taskKey, "qa", activeRunner, stepDelay); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
		qt, err := s.findQueuedTask(ctx, taskKey)
		if err != nil || qt.Stage == queue.StageDone {
			if s.QueueStore != nil {
				_, _ = s.QueueStore.DeduplicateTaskRounds(ctx, taskKey)
			}
			return err
		}
		isoMeta := s.buildTaskIsolationMeta(qt.ExternalID)
		_, _, suffix := agentgate.InferTaskTierAndMode(qt.ExternalID, qt.Title, qt.Description, qt.Complexity, 0)
		mdl := qt.AssignedModel
		if mdl == "" {
			mdl = cliCfg.CoderHighModel
		}

		newBugCreated, bugID, curFailureSig := s.diagnoseAndCreateS5BugSubTask(ctx, *qt, isoMeta)
		if shouldContinuePersistentBugFixLoop(loop, maxSelfHealLoops, hardCapLoops, newBugCreated, prevFailureSig, curFailureSig) {
			prevFailureSig = curFailureSig
			if s.Gatekeeper != nil {
				_, _ = s.Gatekeeper.AppendImmutableProgressLog(isoMeta.ArtifactDir, suffix, mdl,
					fmt.Sprintf("SELF_HEAL_LOOP_%d Persistent S5->S4->S5 (%s · Loop %d · Bug=%s): Tự động sửa lỗi phát sinh & chạy lại 5A/5B cho tới khi DONE", loop, qt.ExternalID, loop, bugID))
			}
			continue
		}

		// When retries are exhausted and no further progress can be made, trigger H6 Circuit Breaker & Wave Rollback
		_ = s.AtomicRollbackWaveSubTasks(qt.ExternalID, s.loadOrGenerateSubTasks(*qt), 0, fmt.Sprintf("H6 Circuit Breaker Open after %d failed Self-Heal loops (%s)", loop, curFailureSig))
		if s.Gatekeeper != nil {
			_, _, _ = s.Gatekeeper.RunHarness2RecursionH6(ctx, qt.ExternalID, isoMeta.ArtifactDir)
			_, _ = s.Gatekeeper.AppendImmutableProgressLog(isoMeta.ArtifactDir, suffix, mdl,
				fmt.Sprintf("ESCALATE_H6 (%s · Loop %d): Tạm dừng tại NEEDS_YOU sau khi không còn phát hiện tiến triển sửa lỗi mới", qt.ExternalID, loop))
		}
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Hook H6 · Loop %d] Tạm dừng tại NEEDS_YOU sau khi lặp cùng chữ ký lỗi (%s).", loop, curFailureSig))
		break
	}
	return nil
}

// hasDirectGoFiles returns true if absDir contains at least one non-test .go source file directly inside it.
func hasDirectGoFiles(absDir string) bool {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			return true
		}
	}
	return false
}

// buildTargetedRepoCheckCommand constructs a clean, deterministic Go check command for external repos
// by targeting the exact packages touched by Sub-Tasks (excluding scratch/, docs/, test/, or dirs without direct .go files per L002).
func (s *Server) buildTargetedRepoCheckCommand(repoPath string, subTasks []SubTaskDTO) string {
	checkCmd := "go vet ./..."
	manifestRaw, mErr := os.ReadFile(filepath.Join(repoPath, ".agent-project.json"))
	if mErr == nil {
		var manifest struct {
			Checks []string `json:"checks"`
		}
		if json.Unmarshal(manifestRaw, &manifest) == nil && len(manifest.Checks) > 0 && strings.TrimSpace(manifest.Checks[0]) != "" {
			checkCmd = strings.TrimSpace(manifest.Checks[0])
		}
	}
	if filepath.Clean(repoPath) == filepath.Clean(s.Root) {
		return checkCmd
	}
	if !strings.Contains(checkCmd, "./...") {
		return checkCmd
	}

	repoName := filepath.Base(repoPath)
	pkgSet := make(map[string]bool)
	var pkgs []string
	for _, st := range subTasks {
		if !strings.EqualFold(strings.TrimSpace(st.TargetRepoName), repoName) &&
			filepath.Clean(strings.TrimSpace(st.TargetRepo)) != filepath.Clean(repoPath) {
			continue
		}
		rawRefs := st.ContextFileLine + " " + st.TargetFiles
		for _, rel := range extractCandidateRelPaths(rawRefs, repoName) {
			cleanRel := strings.TrimPrefix(strings.TrimSuffix(rel, "/"), "./")
			if cleanRel == "" || strings.HasPrefix(cleanRel, "scratch/") || strings.HasPrefix(cleanRel, "docs/") || cleanRel == "test" || strings.HasPrefix(cleanRel, "test/") || cleanRel == "tests" || strings.HasPrefix(cleanRel, "tests/") {
				continue
			}
			absCandidate := filepath.Join(repoPath, cleanRel)
			pkgDir := cleanRel
			if fi, err := os.Stat(absCandidate); err == nil && !fi.IsDir() {
				if !strings.HasSuffix(cleanRel, ".go") {
					continue
				}
				pkgDir = filepath.Dir(cleanRel)
			} else if strings.Contains(filepath.Base(cleanRel), ".") {
				if !strings.HasSuffix(cleanRel, ".go") {
					continue
				}
				pkgDir = filepath.Dir(cleanRel)
			}
			if pkgDir == "." || pkgDir == "" || strings.HasPrefix(pkgDir, "scratch") || strings.HasPrefix(pkgDir, "docs") || pkgDir == "test" || strings.HasPrefix(pkgDir, "test/") || pkgDir == "tests" || strings.HasPrefix(pkgDir, "tests/") {
				continue
			}
			absPkgDir := filepath.Join(repoPath, pkgDir)
			if fi, err := os.Stat(absPkgDir); err == nil && fi.IsDir() && hasDirectGoFiles(absPkgDir) {
				targetPkg := "./" + strings.TrimPrefix(pkgDir, "./")
				if !pkgSet[targetPkg] {
					pkgSet[targetPkg] = true
					pkgs = append(pkgs, targetPkg)
				}
			}
		}
	}
	if len(pkgs) > 0 {
		sort.Strings(pkgs)
		return "go build " + strings.Join(pkgs, " ")
	}
	return checkCmd
}

// finishLiveStepProgress marks liveStepProgress as completed (Step 17/17 · 100%, Active=false, CanStop=false)
// so the WebUI progress bar reaches 100% DONE cleanly without staying stuck at 15/17.
func (s *Server) finishLiveStepProgress(taskKey string) {
	s.mu.Lock()
	if s.liveStepProgress.TaskKey == "" || strings.EqualFold(s.liveStepProgress.TaskKey, taskKey) {
		s.liveStepProgress.Active = false
		s.liveStepProgress.CanStop = false
		s.liveStepProgress.WasStopped = false
		if s.liveStepProgress.StepIndex < 17 {
			s.liveStepProgress.StepIndex = 17
			s.liveStepProgress.TotalSteps = 17
			s.liveStepProgress.Percent = 100
		}
	}
	s.mu.Unlock()
	s.broadcast("step_progress")
}
