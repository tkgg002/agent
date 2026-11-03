package webui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/gitclient"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

// restoreFalsePositiveMaskedCodePlaceholders repairs Go/TS format placeholders and variable references in unified diffs.
func restoreFalsePositiveMaskedCodePlaceholders(patch string) string {
	if patch == "" {
		return ""
	}
	repaired := patch
	replacements := [][2]string{
		{"postgres://***:***@%s", "postgres://%s:%s@%s"},
		{"postgresql://***:***@%s", "postgresql://%s:%s@%s"},
		{"mongodb://***:***@%s", "mongodb://%s:%s@%s"},
		{"mongodb+srv://***:***@%s", "mongodb+srv://%s:%s@%s"},
		{"mysql://***:***@%s", "mysql://%s:%s@%s"},
		{"redis://***:***@%s", "redis://%s:%s@%s"},
		{"amqp://***:***@%s", "amqp://%s:%s@%s"},
		{"password:***\n", "password: connForm.password,\n"},
		{"password:***\r\n", "password: connForm.password,\r\n"},
		{"COALESCE(cr.connection_code, 'default') AS master_connection_key,", "COALESCE(cr.connection_code, '') AS master_connection_key,"},
	}
	for _, pair := range replacements {
		repaired = strings.ReplaceAll(repaired, pair[0], pair[1])
	}
	return repaired
}

// isSubTaskApprovedForWorkingTreeApply returns true when a Sub-Task has been explicitly approved
// (via per-subtask `.user-approved` marker, approved/in-review/done lifecycle state, or parent `APPROVE ALL` when not in step-by-step mode).
func (s *Server) isSubTaskApprovedForWorkingTreeApply(qt queue.QueuedTask, st SubTaskDTO) bool {
	safeParent := sanitizeTaskKeyPath(qt.ExternalID)
	safeSub := sanitizeTaskKeyPath(st.SubTaskID)
	parentTaskDir := filepath.Join(s.Root, ".orchestrator", "tasks", safeParent)
	subApprovedMarker := filepath.Join(parentTaskDir, "subtasks", safeSub, ".user-approved")
	if _, err := os.Stat(subApprovedMarker); err == nil {
		return true
	}
	stageUpper := strings.ToUpper(strings.TrimSpace(st.Stage))
	if stageUpper == "APPROVED" || stageUpper == "IN_REVIEW" || stageUpper == "DONE" {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(st.Status), "done") {
		return true
	}
	_, parentApprovedErr := os.Stat(filepath.Join(parentTaskDir, ".user-approved"))
	_, stepByStepErr := os.Stat(filepath.Join(parentTaskDir, ".step-by-step-subtasks"))
	if parentApprovedErr == nil && stepByStepErr != nil {
		return true
	}
	return false
}

// extractDiffSectionRelPath extracts the relative target path `b/<relPath>` from a `diff --git a/... b/...` section.
func extractDiffSectionRelPath(diffSection string) string {
	firstLine := strings.SplitN(strings.TrimSpace(diffSection), "\n", 2)[0]
	if idx := strings.Index(firstLine, " b/"); idx >= 0 {
		return strings.TrimSpace(firstLine[idx+3:])
	}
	return ""
}

// stripTrailingMultiRepoSeparators removes multi-repo summary headers (`# ===...`, `## git status`, `## git diff`)
// that may follow the last hunk of a `diff --git` section in combined patches.
func stripTrailingMultiRepoSeparators(diffSection string) string {
	lines := strings.Split(strings.ReplaceAll(diffSection, "\r\n", "\n"), "\n")
	var out []string
	for _, ln := range lines {
		if strings.HasPrefix(ln, "# ====================================================================") ||
			strings.HasPrefix(ln, "# REPO: ") ||
			strings.HasPrefix(ln, "## git status") ||
			strings.HasPrefix(ln, "## git diff") {
			break
		}
		out = append(out, ln)
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// splitSingleFileDiffIntoHunks splits a `diff --git` section with multiple `@@ ... @@` blocks
// into individual valid unified diff patches (each retaining the `diff --git ... --- a/... +++ b/...` header).
func splitSingleFileDiffIntoHunks(cleanSection string) []string {
	lines := strings.Split(cleanSection, "\n")
	var headerLines []string
	var hunks [][]string
	var curHunk []string
	inHunk := false
	for _, ln := range lines {
		if strings.HasPrefix(ln, "@@ ") {
			if len(curHunk) > 0 {
				hunks = append(hunks, curHunk)
			}
			curHunk = []string{ln}
			inHunk = true
			continue
		}
		if !inHunk {
			headerLines = append(headerLines, ln)
		} else {
			curHunk = append(curHunk, ln)
		}
	}
	if len(curHunk) > 0 {
		hunks = append(hunks, curHunk)
	}
	if len(headerLines) == 0 || len(hunks) <= 1 {
		return nil
	}
	header := strings.Join(headerLines, "\n")
	var res []string
	for _, h := range hunks {
		res = append(res, header+"\n"+strings.Join(h, "\n"))
	}
	return res
}

func applySingleUnifiedPatchPayload(ctx context.Context, repoDir, payload string) bool {
	tmpFile, err := os.CreateTemp("", "orch-subtask-patch-*.patch")
	if err != nil {
		return false
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if !strings.HasSuffix(payload, "\n") {
		payload += "\n"
	}
	if _, wErr := tmpFile.WriteString(payload); wErr != nil {
		_ = tmpFile.Close()
		return false
	}
	_ = tmpFile.Close()

	runGit := func(args ...string) error {
		runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(runCtx, "git", append([]string{"-C", repoDir}, args...)...)
		return cmd.Run()
	}

	// 1. Check if already applied in the working tree (idempotent reverse check)
	if runGit("apply", "--reverse", "--check", tmpPath) == nil ||
		runGit("apply", "-C1", "--reverse", "--check", tmpPath) == nil {
		return true
	}

	// 2. Standard exact apply
	if runGit("apply", "--check", tmpPath) == nil {
		if runGit("apply", tmpPath) == nil {
			return true
		}
	}

	// 3. Context-1 apply (safe without leaving conflict markers)
	if runGit("apply", "-C1", "--check", tmpPath) == nil {
		if runGit("apply", "-C1", tmpPath) == nil {
			return true
		}
	}

	return false
}

// tryGitApplySectionToRepo idempotently applies a single `diff --git` section onto `repoDir`'s working tree
// without ever leaving `UU` conflict markers (`<<<<<<<` / `>>>>>>>`).
func tryGitApplySectionToRepo(ctx context.Context, repoDir, diffSection string) bool {
	trimmed := stripTrailingMultiRepoSeparators(strings.TrimSpace(diffSection))
	if !strings.HasPrefix(trimmed, "diff --git ") || strings.Contains(trimmed, "Isolated Remediation Hunk for ") {
		return false
	}
	relPath := extractDiffSectionRelPath(trimmed)
	if relPath != "" {
		absTarget := filepath.Join(repoDir, relPath)
		if _, statErr := os.Stat(absTarget); statErr != nil {
			isNewFile := strings.Contains(trimmed, "new file mode") || strings.Contains(trimmed, "--- /dev/null")
			parentDir := filepath.Dir(relPath)
			if !isNewFile || parentDir == "." || parentDir == "" {
				return false
			}
			if pStat, pErr := os.Stat(filepath.Join(repoDir, parentDir)); pErr != nil || !pStat.IsDir() {
				return false
			}
		}
	}
	if applySingleUnifiedPatchPayload(ctx, repoDir, trimmed) {
		return gitclient.CheckGoPackageCompilesOrRevertRelFile(ctx, repoDir, relPath)
	}
	// Fallback: split multi-hunk file diff into individual `@@ ... @@` hunks and apply all clean hunks
	anyHunkApplied := false
	for _, singleHunk := range splitSingleFileDiffIntoHunks(trimmed) {
		if applySingleUnifiedPatchPayload(ctx, repoDir, singleHunk) {
			anyHunkApplied = true
		}
	}
	if anyHunkApplied {
		return gitclient.CheckGoPackageCompilesOrRevertRelFile(ctx, repoDir, relPath)
	}
	return false
}

// applyApprovedSubTaskPatchToTargetRepo applies the real unified diff hunks belonging to an approved Sub-Task
// directly onto `st.TargetRepo`'s working tree so `git status` immediately reflects the code changes of each approved Sub-Task.
func (s *Server) applyApprovedSubTaskPatchToTargetRepo(ctx context.Context, qt queue.QueuedTask, st SubTaskDTO, subHunks []string) int {
	if s.isTestTaskOrTempRoot(qt.ExternalID) || strings.HasPrefix(strings.ToUpper(qt.ExternalID), "TASK-ISO-") {
		return 0
	}
	if !s.isSubTaskApprovedForWorkingTreeApply(qt, st) {
		return 0
	}
	repoDir := strings.TrimSpace(st.TargetRepo)
	if repoDir == "" {
		return 0
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		return 0
	}

	applied := 0
	seenRelPaths := make(map[string]bool)
	for _, rawHunk := range subHunks {
		cleanHunk := restoreFalsePositiveMaskedCodePlaceholders(rawHunk)
		if !strings.HasPrefix(strings.TrimSpace(cleanHunk), "diff --git ") ||
			strings.Contains(cleanHunk, "Isolated Remediation Hunk for ") {
			continue
		}
		if rel := extractDiffSectionRelPath(cleanHunk); rel != "" {
			seenRelPaths[rel] = true
		}
		if tryGitApplySectionToRepo(ctx, repoDir, cleanHunk) {
			applied++
		}
	}

	// Also apply any companion diff sections from the parent diff.patch that belong to `repoDir`
	// and are not claimed by an unapproved Sub-Task (by exact file or package directory).
	parentPatch := restoreFalsePositiveMaskedCodePlaceholders(string(s.readTaskOrSiblingDiffPatch(qt.ExternalID)))
	if strings.Contains(parentPatch, "diff --git ") {
		ownCandidateDirs := make(map[string]bool)
		for _, cand := range extractCandidateRelPaths(st.ContextFileLine+" "+st.TargetFiles, filepath.Base(repoDir)) {
			cleanCand := strings.TrimPrefix(strings.TrimSuffix(cand, "/"), "./")
			if cleanCand != "" {
				ownCandidateDirs[filepath.Dir(cleanCand)] = true
				if strings.HasSuffix(cand, "/") {
					ownCandidateDirs[cleanCand] = true
				}
			}
		}

		allSubTasks := s.loadOrGenerateSubTasks(qt)
		unapprovedCandidates := make(map[string]bool)
		unapprovedCandidateDirs := make(map[string]bool)
		for _, other := range allSubTasks {
			if strings.EqualFold(other.SubTaskID, st.SubTaskID) || s.isSubTaskApprovedForWorkingTreeApply(qt, other) {
				continue
			}
			otherRefs := other.ContextFileLine + " " + other.TargetFiles
			for _, cand := range extractCandidateRelPaths(otherRefs, filepath.Base(other.TargetRepo)) {
				cleanCand := strings.TrimPrefix(strings.TrimSuffix(cand, "/"), "./")
				if cleanCand == "" {
					continue
				}
				if !strings.HasSuffix(cand, "/") {
					unapprovedCandidates[cleanCand] = true
					dir := filepath.Dir(cleanCand)
					if dir != "." && dir != "" && !ownCandidateDirs[dir] {
						unapprovedCandidateDirs[dir] = true
					}
				} else if !ownCandidateDirs[cleanCand] {
					unapprovedCandidateDirs[cleanCand] = true
				}
			}
		}
		for _, sec := range strings.Split(parentPatch, "diff --git ") {
			trimmed := strings.TrimSpace(sec)
			if trimmed == "" {
				continue
			}
			fullSec := "diff --git " + trimmed
			if strings.Contains(fullSec, "Isolated Remediation Hunk for ") {
				continue
			}
			rel := extractDiffSectionRelPath(fullSec)
			if rel == "" || seenRelPaths[rel] || unapprovedCandidates[rel] || unapprovedCandidateDirs[filepath.Dir(rel)] {
				continue
			}
			if tryGitApplySectionToRepo(ctx, repoDir, fullSec) {
				seenRelPaths[rel] = true
				applied++
			}
		}
	}

	if applied > 0 {
		s.appendTaskLog(qt.ExternalID, fmt.Sprintf("[Sub-Task %s · Working-Tree Git Apply] Đã apply %d hunk code thật xuống %s (git status đã cập nhật).", st.SubTaskID, applied, filepath.Base(repoDir)))
	}
	return applied
}

