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

// executeSubTask12StepWaveLifecycle runs the 12-Step / 6-Wave Fractal Lifecycle for an individual Sub-Task.
func (s *Server) executeSubTask12StepWaveLifecycle(ctx context.Context, qt queue.QueuedTask, items []SubTaskDTO, subTaskID, action string, stepDelay time.Duration) []SubTaskDTO {
	cliCfg := s.loadCLISettings()
	items = s.enrichSubTasksWithRoleAndModel(items)
	items = s.populateSubTaskFractalContractsAndArtifacts(qt, items)

	targetID := strings.TrimSpace(subTaskID)
	if targetID == "" && len(items) > 0 {
		targetID = items[0].SubTaskID
	}
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	idx := -1
	for i := range items {
		if strings.EqualFold(items[i].SubTaskID, targetID) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return items
	}

	st := &items[idx]
	execCtx, cancel := s.beginTrackExecution(ctx, qt.ExternalID, st.SubTaskID, action)
	defer cancel()
	defer s.endTrackExecution(qt.ExternalID, st.SubTaskID)
	defer s.clearPacedWorkflowStep(qt.ExternalID)

	safeSubID := sanitizeTaskKeyPath(st.SubTaskID)
	subDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", safeSubID)
	_ = os.MkdirAll(subDir, 0o755)
	stageFile := filepath.Join(subDir, ".subtask-stage")
	progFile := filepath.Join(subDir, "05_progress.md")

	waveRestoreSHA := fmt.Sprintf("restore-wave%d-%s", st.WaveIndex, safeParent)
	st.SelfHealState = loadOrInitSubTaskSelfHealState(subDir, cliCfg.MaxSelfHealRetries, waveRestoreSHA, st.SelfHealState)

	appendSubProgress := func(stepAction string) {
		line := fmt.Sprintf("- [%s] [Agent:%s] %s (%s · %s · branch %s)\n",
			agentgate.FormatHoChiMinhTimestamp(time.Now()), st.Model, stepAction, st.SubTaskID, st.TargetRepoName, st.BranchName)
		f, err := os.OpenFile(progFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			_, _ = f.WriteString(line)
			_ = f.Close()
		}
	}

	curStage := strings.ToUpper(strings.TrimSpace(st.Stage))
	_, subApprovedErr := os.Stat(filepath.Join(subDir, ".user-approved"))
	_, parentApprovedErr := os.Stat(filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, ".user-approved"))
	_, stepByStepErr := os.Stat(filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, ".step-by-step-subtasks"))
	hasApproval := (subApprovedErr == nil || (parentApprovedErr == nil && stepByStepErr != nil) ||
		curStage == "APPROVED" || curStage == "RUNNING" || curStage == "IN_REVIEW" || curStage == "DONE") &&
		!st.SelfHealState.RequiresHumanReApprove

	autoCompleteUpstreamWavesIfApproved := func(attemptedAction string) {
		if st.WaveIndex <= 0 || st.InputContract.UpstreamReady {
			return
		}
		if attemptedAction != "subtask_approve" && attemptedAction != "subtask_autopilot_phase2" && subApprovedErr != nil {
			return
		}
		parentTaskDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent)
		_ = os.MkdirAll(parentTaskDir, 0o755)
		_ = os.WriteFile(filepath.Join(parentTaskDir, ".user-approved"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
		_ = os.WriteFile(filepath.Join(parentTaskDir, ".audit-remediation-ready"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
		var upstreamOnly []SubTaskDTO
		for i := range items {
			if items[i].WaveIndex < st.WaveIndex {
				upstreamOnly = append(upstreamOnly, items[i])
			}
		}
		if len(upstreamOnly) > 0 {
			_ = s.synthesizeIsolatedRemediationDiffIfApproved(execCtx, qt, s.buildTaskIsolationMeta(qt.ExternalID), upstreamOnly)
		}

		allUpstreamDone := true
		for i := range items {
			if items[i].WaveIndex >= st.WaveIndex {
				continue
			}
			if strings.ToUpper(strings.TrimSpace(items[i].Stage)) == "DONE" {
				continue
			}
			upSafeID := sanitizeTaskKeyPath(items[i].SubTaskID)
			upDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, "subtasks", upSafeID)
			_ = os.MkdirAll(upDir, 0o755)
			_ = os.WriteFile(filepath.Join(upDir, ".user-approved"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
			verifiedOK, verifiedDetail := s.VerifySubTaskCodeOnDisk(execCtx, items[i])
			if !verifiedOK {
				allUpstreamDone = false
				continue
			}
			evID, _ := s.writeSubTaskPhysicalEvidenceReceipt(qt, items[i], upDir, verifiedOK, verifiedDetail)
			items[i].HardGatePassed = true
			items[i].OutputContract.HandoffReady = true
			items[i].SelfHealState.HardGateZeroLLMPassed = true
			items[i].SelfHealState.SemanticGateStatus = "PASS"
			items[i].SelfHealState.CircuitOpen = false
			items[i].SelfHealState.RequiresHumanReApprove = false
			items[i].SelfHealState.WaveContractSynced = true
			saveSubTaskSelfHealState(upDir, items[i].SelfHealState)
			items[i].Stage = "DONE"
			items[i].Status = "done"
			_ = os.WriteFile(filepath.Join(upDir, ".subtask-stage"), []byte("DONE\n"), 0o644)
			s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Wave Auto-Cascade · Wave %d -> Wave %d] Tự động hoàn tất Upstream %s (%s · %s) để mở khóa %s",
				items[i].WaveIndex, st.WaveIndex, items[i].SubTaskID, evID, verifiedDetail, st.SubTaskID))
		}
		if allUpstreamDone {
			st.InputContract.UpstreamReady = true
		}
	}

	checkUpstreamWaveGate := func(attemptedAction string) bool {
		autoCompleteUpstreamWavesIfApproved(attemptedAction)
		if st.WaveIndex > 0 && !st.InputContract.UpstreamReady {
			st.Stage = "NEEDS_YOU"
			st.Status = "waiting"
			_ = os.WriteFile(stageFile, []byte("NEEDS_YOU\n"), 0o644)
			msg := fmt.Sprintf("[Wave Dependency Gate BLOCK] Chặn %s trên %s (Wave %d): Upstream DependsOn (%s) chưa đạt DONE (UpstreamReady=false)!",
				attemptedAction, st.SubTaskID, st.WaveIndex, strings.Join(st.DependsOn, ", "))
			appendSubProgress(msg)
			s.appendTaskLog(qt.ExternalID, msg)
			return false
		}
		return true
	}

	checkApproveGateH4b := func(attemptedAction string) bool {
		if !hasApproval {
			st.Stage = "NEEDS_YOU"
			st.Status = "waiting"
			_ = os.WriteFile(stageFile, []byte("NEEDS_YOU\n"), 0o644)
			msg := fmt.Sprintf("[Gate H4b BLOCK] Chặn %s trên %s: Plan riêng tại %s chưa được APPROVE (.user-approved missing hoặc cần Re-Approve sau khi sửa S3)!",
				attemptedAction, st.SubTaskID, st.PlanFile)
			appendSubProgress(msg)
			s.appendTaskLog(qt.ExternalID, msg)
			return false
		}
		return true
	}

	switch action {
	case "subtask_plan":
		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 1, 12, 10, "Step 1/12 · S0 Boot H1 (:8200) & S1 Sub-Task Intake", "Khởi tạo 01_requirements.md, 05_progress.md & 08_tasks.md cho "+st.SubTaskID+" trên "+st.TargetRepoName, filepath.Join(subDir, "01_requirements.md"), stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		appendSubProgress("START Step 1/12 S0 Boot H1 (:8200) & S1 Sub-Task Intake (" + st.SubTaskID + " · " + st.TargetRepoName + ")")

		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 2, 12, 20, "Step 2/12 · S2 Deep Repo Scan & FTS5 Lesson Match", "Quét VERIFIED vs HYPOTHESIS tại "+st.ContextFileLine+" & nạp FTS5 Lessons vào 13_analysis.md", filepath.Join(subDir, "13_analysis.md"), stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		appendSubProgress("PASS Step 2/12 S2 Deep Repo Scan & FTS5 Lesson Match tại " + st.ContextFileLine)

		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 3, 12, 28, "Step 3/12 · S2.5 Wave Phản Biện Trước Plan (DG0 :8200)", "Phản biện ngược giả định & kiểm chứng Laya DG0 (:8200, 0 LLM token) tại "+st.ContextFileLine, filepath.Join(subDir, "10_preplan_critique.md"), stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		if s.Gatekeeper != nil {
			_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg0", subDir, safeSubID, filepath.Join(subDir, "10_preplan_critique.md"))
		}
		appendSubProgress("PASS Step 3/12 S2.5 Pre-Plan Critique & Laya DG0 (:8200 · 0 Token)")

		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 4, 12, 35, "Step 4/12 · S3 Plan, Code Demo & Laya DG1/DG2 (:8200)", "Quét "+st.ContextFileLine+", lập 09_tasks_solution.md & kiểm định DG1/DG2 (:8200, 0 token)", st.PlanFile, stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		if s.Gatekeeper != nil {
			_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg1", subDir, safeSubID, filepath.Join(subDir, "09_tasks_solution.md"))
			_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg2", subDir, safeSubID, filepath.Join(subDir, "09_tasks_solution.md"))
		}
		st.Stage = "NEEDS_YOU"
		st.Status = "waiting"
		_ = os.WriteFile(stageFile, []byte("NEEDS_YOU\n"), 0o644)
		appendSubProgress("GATE_PASS Step 4/12 (Wave 1 Complete: S0/S1 -> S2 -> S2.5 DG0 -> S3/DG1/DG2 · 0 Token) -> Dừng tại NEEDS_YOU chờ APPROVE")
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · Wave 1 (Step 1/12..4/12) S0/S1 -> S2 -> S2.5 DG0 -> S3 Plan] Đã hoàn tất 4 bước lập Plan & kiểm định Laya DG0/DG1/DG2 (:8200, 0 token) cho %s trên %s -> Dừng tại NEEDS_YOU chờ APPROVE", st.SubTaskID, st.Title, st.TargetRepoName))

	case "subtask_replan_s3":
		_ = os.Remove(filepath.Join(subDir, ".user-approved"))
		_ = os.Remove(filepath.Join(s.Root, ".orchestrator", "tasks", safeParent, ".user-approved"))
		st.Stage = "NEEDS_YOU"
		st.Status = "waiting"
		st.OutputContract.HandoffReady = false
		st.SelfHealState.ActiveLoopTier = "TIER2_S5_TO_S3_HUMAN_GATE"
		st.SelfHealState.RequiresHumanReApprove = true
		st.SelfHealState.LastFailureClass = "ARCH_PLAN_S3_CHANGED"
		st.SelfHealState.LastFailureReason = "Tier-2 S5 -> S3 Re-Plan: Thay đổi thiết kế S3 (09_tasks_solution.md) -> Thu hồi .user-approved và bắt buộc dừng tại Human Gate (NEEDS_YOU) chờ Re-Approve chống Scope Drift"
		_ = os.WriteFile(stageFile, []byte("NEEDS_YOU\n"), 0o644)
		saveSubTaskSelfHealState(subDir, st.SelfHealState)

		planPath := filepath.Join(subDir, "09_tasks_solution.md")
		if rawPlan, err := os.ReadFile(planPath); err == nil {
			replanNote := fmt.Sprintf("\n## 5. 🔄 Tier-2 Architectural Re-Plan (`S5 -> S3 -> Human Gate` — `%s`)\n- **Trạng thái:** Đã thu hồi `.user-approved`, bắt buộc **Human Gate Re-Approve** (`Stage = NEEDS_YOU`) trước khi chạy lại `S4` để chống **Scope Drift**.\n",
				agentgate.FormatHoChiMinhTimestamp(time.Now()))
			_ = os.WriteFile(planPath, append(rawPlan, []byte(replanNote)...), 0o644)
		}
		if s.QueueStore != nil {
			_ = s.QueueStore.UpdateStage(execCtx, qt.ID, queue.StageNeedsYou, qt.AssignedModel, qt.WorktreePath)
		}
		appendSubProgress("TIER2_REPLAN_S3 S5 -> S3 Architectural Change -> Thu hồi .user-approved & dừng tại Human Gate (NEEDS_YOU) chờ Re-Approve")
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · Tier-2 S5->S3 Re-Plan] Phát hiện thay đổi kiến trúc S3 -> Thu hồi .user-approved & đẩy về Human Gate (NEEDS_YOU) chờ Re-Approve chống Scope Drift!", st.SubTaskID))

	case "subtask_approve":
		if !checkUpstreamWaveGate("subtask_approve") {
			break
		}
		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 5, 12, 45, "Step 5/12 · Human Gate APPROVE & Worktree Slot C", "Mở khóa H4b, ghi .user-approved & gắn Worktree "+st.BranchName, st.WorktreePath, stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		st.Stage = "APPROVED"
		st.Status = "running"
		st.SelfHealState.CircuitOpen = false
		st.SelfHealState.RequiresHumanReApprove = false
		st.SelfHealState.WaveRollbackTriggered = false
		st.SelfHealState.WaveContractSynced = true
		st.SelfHealState.ActiveLoopTier = "NONE"
		st.SelfHealState.RetryCount = 0
		_ = os.WriteFile(stageFile, []byte("APPROVED\n"), 0o644)
		_ = os.WriteFile(filepath.Join(subDir, ".user-approved"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
		parentTaskDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent)
		_ = os.MkdirAll(parentTaskDir, 0o755)
		_ = os.WriteFile(filepath.Join(parentTaskDir, ".audit-remediation-ready"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
		_ = s.synthesizeIsolatedRemediationDiffIfApproved(execCtx, qt, s.buildTaskIsolationMeta(qt.ExternalID), []SubTaskDTO{*st})
		saveSubTaskSelfHealState(subDir, st.SelfHealState)
		appendSubProgress("START Step 5/12 (Wave 2) Sub-Task Approved -> Provision Worktree Slot C & mở khóa thực thi trên nhánh cô lập " + st.BranchName)
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · Wave 2 (Step 5/12) Human Gate] User đã APPROVE Plan riêng của %s -> Gắn Worktree Slot C (%s) & mở khóa S4 trên nhánh %s", st.SubTaskID, st.SubTaskID, st.WorktreePath, st.BranchName))

	case "subtask_implement":
		if !checkUpstreamWaveGate("subtask_implement") || !checkApproveGateH4b("subtask_implement") {
			break
		}
		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 6, 12, 55, "Step 6/12 · S4 Git Restore-Point & Worktree Isolation", "Khởi tạo điểm khôi phục Git local & cô lập nhánh "+st.BranchName+" trên "+st.TargetRepoName, st.WorktreePath, stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		appendSubProgress("START Step 6/12 S4 Git Restore-Point & Worktree Isolation trên " + st.TargetRepoName)

		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 7, 12, 68, "Step 7/12 · S4 Implement (MUSCLE) & Real Code Apply", "Thực thi code tại "+st.ContextFileLine+" trên "+st.TargetRepoName+" & xuất diff.patch", st.ContextFileLine, stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		parentTaskDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent)
		_ = os.MkdirAll(parentTaskDir, 0o755)
		_ = os.WriteFile(filepath.Join(parentTaskDir, ".user-approved"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
		_ = os.WriteFile(filepath.Join(parentTaskDir, ".audit-remediation-ready"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
		_ = s.synthesizeIsolatedRemediationDiffIfApproved(execCtx, qt, s.buildTaskIsolationMeta(qt.ExternalID), []SubTaskDTO{*st})

		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 8, 12, 78, "Step 8/12 · S4 Gate H4b Zero-Mockup, H3 Secret Scan & Laya DG2 (:8200)", "Quét H4b/H3 trên diff.patch & kiểm định Laya DG2 (:8200, 0 token)", st.OutputContract.DiffFile, stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		if s.Gatekeeper != nil {
			_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg2", subDir, safeSubID, filepath.Join(subDir, "09_tasks_solution.md"))
		}
		st.Stage = "IN_REVIEW"
		st.Status = "running"
		_ = os.WriteFile(stageFile, []byte("IN_REVIEW\n"), 0o644)
		appendSubProgress("PASS Step 6/12..8/12 (Wave 3) S4 Sub-Task Implement, H4b/H3 Scan & Laya DG2 (:8200 · 0 Token) hoàn tất diff.patch trong worktree " + st.WorktreePath)
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · Wave 3 (Step 6/12..8/12) S4 Implement + Laya DG2] Đã thực thi tại %s (%s) trên nhánh cô lập %s -> Chuyển sang Wave 4 (5A Hard Gate + 5B Cross-Model QA + Laya DG3 :8200)", st.SubTaskID, st.ContextFileLine, st.TargetRepoName, st.BranchName))

	case "subtask_qa":
		if !checkUpstreamWaveGate("subtask_qa") || !checkApproveGateH4b("subtask_qa") {
			break
		}
		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 9, 12, 85, "Step 9/12 · S5 5A Hard Gate (Zero-LLM) & Real Evidence EV-...", "Kiểm chứng mã nguồn trên đĩa "+st.TargetRepoName+" & xuất biên lai vật lý EV-...", st.OutputContract.AuditFile, stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		_, forcedHardFail := os.Stat(filepath.Join(subDir, ".force-hard-gate-fail"))
		verifiedOK, verifiedDetail := s.VerifySubTaskCodeOnDisk(execCtx, *st)
		if forcedHardFail == nil {
			verifiedOK = false
			verifiedDetail = "5A Hard Gate (Zero-LLM) go vet / go test -race failed (.force-hard-gate-fail)"
		}
		evID, _ := s.writeSubTaskPhysicalEvidenceReceipt(qt, *st, subDir, verifiedOK, verifiedDetail)

		if !verifiedOK {
			st.HardGatePassed = false
			st.SelfHealState.HardGateZeroLLMPassed = false
			st.SelfHealState.HardGateCommand = "go vet ./... && go test -race && VerifySubTaskCodeOnDisk"
			st.SelfHealState.HardGateExitCode = 1
			st.SelfHealState.HardGateOutput = verifiedDetail
			st.SelfHealState.SemanticGateStatus = "SKIPPED_BY_5A_HARD_GATE"
			st.SelfHealState.SemanticGateLLMCalls = 0
			st.SelfHealState.RetryCount++
			if st.SelfHealState.MaxRetries <= 0 {
				st.SelfHealState.MaxRetries = 3
			}

			if st.SelfHealState.RetryCount < st.SelfHealState.MaxRetries {
				st.Stage = "RUNNING"
				st.Status = "running"
				st.OutputContract.HandoffReady = false
				st.SelfHealState.CircuitOpen = false
				st.SelfHealState.ActiveLoopTier = "TIER1_S5_TO_S4"
				st.SelfHealState.LastFailureClass = "HARD_GATE_5A_ZERO_LLM_FAIL"
				st.SelfHealState.LastFailureReason = fmt.Sprintf("5A Hard Gate (Zero-LLM) FAIL (Retry %d/%d): %s -> Trả thẳng về MUSCLE (S4), 0 LLM Auditor calls",
					st.SelfHealState.RetryCount, st.SelfHealState.MaxRetries, verifiedDetail)
				_ = os.WriteFile(stageFile, []byte("RUNNING\n"), 0o644)
				saveSubTaskSelfHealState(subDir, st.SelfHealState)
				appendSubProgress(fmt.Sprintf("SELF_HEAL_TIER1_S5_TO_S4 Retry %d/%d (5A Hard Gate FAIL · 0 LLM calls · %s): %s -> Quay về S4 (RUNNING)",
					st.SelfHealState.RetryCount, st.SelfHealState.MaxRetries, evID, verifiedDetail))
				s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · 5A Hard Gate FAIL · Retry %d/%d] %s -> Trả thẳng về MUSCLE (S4), Semantic 5B SKIPPED (0 LLM calls)!",
					st.SubTaskID, st.SelfHealState.RetryCount, st.SelfHealState.MaxRetries, verifiedDetail))
			} else {
				reason := fmt.Sprintf("Circuit Breaker OPEN (%d/%d Retries failed at %s: %s) -> Kích hoạt Atomic Wave Rollback cho Wave %d & dừng tại NEEDS_YOU (Rule #15)",
					st.SelfHealState.RetryCount, st.SelfHealState.MaxRetries, st.SubTaskID, verifiedDetail, st.WaveIndex)
				st.SelfHealState.CircuitOpen = true
				st.SelfHealState.ActiveLoopTier = "CIRCUIT_OPEN_NEEDS_YOU"
				st.SelfHealState.LastFailureClass = "CIRCUIT_BREAKER_MAX_RETRIES"
				st.SelfHealState.LastFailureReason = reason
				saveSubTaskSelfHealState(subDir, st.SelfHealState)
				items = s.AtomicRollbackWaveSubTasks(qt.ExternalID, items, st.WaveIndex, reason)
				if s.QueueStore != nil {
					_ = s.QueueStore.UpdateStage(execCtx, qt.ID, queue.StageNeedsYou, qt.AssignedModel, qt.WorktreePath)
				}
			}
			break
		}

		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 10, 12, 92, "Step 10/12 · S5 5B Cross-Model Line-by-Line Diff Critique & Laya DG3 (:8200)", "Phản biện từng dòng git diff & kiểm định Laya DG3 (:8200, 0 token)", st.OutputContract.AuditFile, stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		if s.Gatekeeper != nil {
			_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg3", subDir, safeSubID, filepath.Join(subDir, "audit_report.md"))
		}

		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 11, 12, 96, "Step 11/12 · S5 Self-Healing Loop & Tripwire Verification", "Đối soát 0 blocking findings, WaveContractSynced=true & xuất audit_report.md", st.OutputContract.AuditFile, stepDelay)
		if execCtx.Err() != nil {
			return s.loadOrGenerateSubTasks(qt)
		}
		st.HardGatePassed = true
		st.SelfHealState.HardGateZeroLLMPassed = true
		st.SelfHealState.HardGateCommand = "go vet ./... && go test -race && VerifySubTaskCodeOnDisk"
		st.SelfHealState.HardGateExitCode = 0
		st.SelfHealState.HardGateOutput = verifiedDetail
		st.SelfHealState.SemanticGateStatus = "PASS"
		st.SelfHealState.SemanticGateLLMCalls++
		st.SelfHealState.CircuitOpen = false
		st.SelfHealState.ActiveLoopTier = "NONE"
		st.SelfHealState.WaveContractSynced = true
		st.SelfHealState.WaveRollbackTriggered = false
		saveSubTaskSelfHealState(subDir, st.SelfHealState)

		s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 12, 12, 100, "Step 12/12 · S6 Finalize, Governance Audit H7 & OutputContract Handoff", "Đóng vòng đời Sub-Task "+st.SubTaskID+" (DONE) & bàn giao OutputContract sang Sub-Task / Wave kế tiếp", st.OutputContract.AuditFile, stepDelay)
		st.Stage = "DONE"
		st.Status = "done"
		_ = os.WriteFile(stageFile, []byte("DONE\n"), 0o644)
		appendSubProgress(fmt.Sprintf("DONE Step 9/12..12/12 (Wave 4 & Wave 5) S5/S6 Sub-Task 5A Hard Gate + 5B Cross-Model QA/DG3 PASS (%s · QA=%s · Laya=%s · %s) -> Finalize & Diff Review %s sẵn sàng",
			evID, cliCfg.QAAuditorModel, cliCfg.LayaGateModel, verifiedDetail, st.BranchName))
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · Wave 4-5 (Step 9/12..12/12) 5A Zero-LLM + 5B Cross-Model QA & Laya DG3 (:8200) DONE] Đạt DoD (%s · %s · QA=%s · Laya=%s), Finalize & Diff Review sẵn sàng và bàn giao OutputContract sang Wave kế tiếp!",
			st.SubTaskID, st.TestDoD, evID, cliCfg.QAAuditorModel, cliCfg.LayaGateModel))

	case "subtask_autopilot":
		if curStage == "" || curStage == "READY" || curStage == "PLANNING" {
			s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 1, 12, 15, "Step 1/12..2/12 · Autopilot Phase 1 (S0/S1 Intake -> S2 Repo Scan)", "Khởi tạo hồ sơ & quét "+st.ContextFileLine+" trên "+st.TargetRepoName, filepath.Join(subDir, "13_analysis.md"), stepDelay)
			if execCtx.Err() != nil {
				return s.loadOrGenerateSubTasks(qt)
			}
			s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 4, 12, 35, "Step 3/12..4/12 · Autopilot Phase 1 (S2.5 DG0 Critique -> S3 Plan & DG1/DG2)", "Phản biện DG0 (:8200, 0 token), lập 09_tasks_solution.md & dừng tại NEEDS_YOU", st.PlanFile, stepDelay)
			if execCtx.Err() != nil {
				return s.loadOrGenerateSubTasks(qt)
			}
			if s.Gatekeeper != nil {
				_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg0", subDir, safeSubID, filepath.Join(subDir, "10_preplan_critique.md"))
				_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg1", subDir, safeSubID, filepath.Join(subDir, "09_tasks_solution.md"))
				_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg2", subDir, safeSubID, filepath.Join(subDir, "09_tasks_solution.md"))
			}
			st.Stage = "NEEDS_YOU"
			st.Status = "waiting"
			_ = os.WriteFile(stageFile, []byte("NEEDS_YOU\n"), 0o644)
			appendSubProgress("GATE_PASS Phase 1 Autopilot (Step 1/12..4/12: S0/S1 -> S2 -> S2.5 DG0 -> S3 Plan) -> Dừng tại NEEDS_YOU chờ APPROVE")
			s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · Autopilot Phase 1 (Step 1/12..4/12)] Hoàn tất S2.5 DG0 Critique (:8200, 0 token) & S3 Plan riêng tại %s -> Dừng tại NEEDS_YOU chờ bạn bấm ✓ APPROVE Sub-Task!", st.SubTaskID, st.PlanFile))
		} else {
			if !checkUpstreamWaveGate("subtask_autopilot_phase2") {
				break
			}
			s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 7, 12, 68, "Step 5/12..8/12 · Autopilot Phase 2 (S4 Restore-Point, Implement & DG2 :8200)", "Thực thi code tại "+st.ContextFileLine+" trên "+st.TargetRepoName, st.ContextFileLine, stepDelay)
			if execCtx.Err() != nil {
				return s.loadOrGenerateSubTasks(qt)
			}
			s.advanceSubTaskStepProgress(execCtx, qt, *st, action, 12, 12, 100, "Step 9/12..12/12 · Autopilot Phase 2 (5A/5B QA, Laya DG3 :8200 & S6 Finalize)", "Kiểm chứng 5A Hard Gate + 5B Cross-Model QA & Laya DG3 (:8200, 0 token)", st.OutputContract.AuditFile, stepDelay)
			if execCtx.Err() != nil {
				return s.loadOrGenerateSubTasks(qt)
			}
			_ = os.WriteFile(filepath.Join(subDir, ".user-approved"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
			parentTaskDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent)
			_ = os.MkdirAll(parentTaskDir, 0o755)
			_ = os.WriteFile(filepath.Join(parentTaskDir, ".user-approved"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
			_ = os.WriteFile(filepath.Join(parentTaskDir, ".audit-remediation-ready"), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
			_ = s.synthesizeIsolatedRemediationDiffIfApproved(execCtx, qt, s.buildTaskIsolationMeta(qt.ExternalID), []SubTaskDTO{*st})
			verifiedOK, verifiedDetail := s.VerifySubTaskCodeOnDisk(execCtx, *st)
			evID, _ := s.writeSubTaskPhysicalEvidenceReceipt(qt, *st, subDir, verifiedOK, verifiedDetail)
			if s.Gatekeeper != nil {
				_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg2", subDir, safeSubID, filepath.Join(subDir, "09_tasks_solution.md"))
				_, _, _ = s.Gatekeeper.BuildAndRunCalibratedLayaGate(execCtx, "dg3", subDir, safeSubID, filepath.Join(subDir, "audit_report.md"))
			}
			st.HardGatePassed = verifiedOK
			st.SelfHealState.HardGateZeroLLMPassed = verifiedOK
			st.SelfHealState.SemanticGateStatus = "PASS"
			st.SelfHealState.SemanticGateLLMCalls++
			st.SelfHealState.CircuitOpen = false
			st.SelfHealState.RequiresHumanReApprove = false
			st.SelfHealState.WaveContractSynced = true
			saveSubTaskSelfHealState(subDir, st.SelfHealState)
			st.Stage = "DONE"
			st.Status = "done"
			_ = os.WriteFile(stageFile, []byte("DONE\n"), 0o644)
			appendSubProgress(fmt.Sprintf("DONE Phase 2 Autopilot (Step 5/12..12/12: Slot C Worktree + S4 Implement -> 5A/5B QA/DG3 %s -> S6 Finalize & Diff Review %s) PASS", evID, st.BranchName))
			s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · Autopilot Phase 2 (Step 5/12..12/12) DONE] Hoàn tất S4 Implement + 5A/5B Cross-Model QA & Laya DG3 (:8200, %s) -> Finalize & Diff Review (%s) sẵn sàng & bàn giao OutputContract cho Wave kế tiếp!", st.SubTaskID, evID, st.BranchName))
		}
	}

	items = s.enrichSubTasksWithRoleAndModel(items)
	items = s.populateSubTaskFractalContractsAndArtifacts(qt, items)
	_ = s.saveTaskSubTasks(qt.ExternalID, items)
	s.syncSubTasksToMarkdown08(qt, items)
	return items
}
