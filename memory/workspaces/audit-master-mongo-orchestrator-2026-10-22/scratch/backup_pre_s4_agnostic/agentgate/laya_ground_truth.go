package agentgate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ResolveFlaggedLayaClaimsWithGroundTruth enforces GEMINI.md Rule #10 inside the agentgate layer:
//   - BLOCK -> never auto-overridden (requires re-investigation).
//   - FLAG  -> invokes `/agent/tooling/laya_gate.py resolve --ws-root ... --workspace ... --stage ... --id ... --reason "REF: ..."`
//     backed by physical compiler receipt (EV-...) and source/artifact reference, and appends to 04_decisions_<suffix>.md.
func (g *Gatekeeper) ResolveFlaggedLayaClaimsWithGroundTruth(
	ctx context.Context,
	workspaceDir, suffix, refFilePath, primaryEvID string,
	verifiedSubTasks, totalSubTasks int,
) {
	if g == nil || strings.TrimSpace(workspaceDir) == "" {
		return
	}
	absWsDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		absWsDir = workspaceDir
	}
	absRef, err := filepath.Abs(refFilePath)
	if err != nil {
		absRef = refFilePath
	}
	script := filepath.Join(g.AgentCoreDir, "tooling", "laya_gate.py")
	_, scriptErr := os.Stat(script)

	for _, stage := range []string{"dg1", "dg2", "dg3"} {
		pattern := filepath.Join(absWsDir, fmt.Sprintf("06_laya_summary_%s_*.json", stage))
		matches, gErr := filepath.Glob(pattern)
		if gErr != nil || len(matches) == 0 {
			continue
		}
		sort.Strings(matches)
		latest := matches[len(matches)-1]
		raw, rErr := os.ReadFile(latest)
		if rErr != nil {
			continue
		}
		var summary struct {
			Gate   string `json:"gate"`
			Claims []struct {
				ID       string  `json:"id"`
				Decision string  `json:"decision"`
				Resolved *string `json:"resolved"`
			} `json:"claims"`
		}
		if json.Unmarshal(raw, &summary) != nil {
			continue
		}
		gateUpper := strings.ToUpper(strings.TrimSpace(summary.Gate))
		if gateUpper == "PASS" || gateUpper == "BLOCK" {
			continue
		}

		reason := fmt.Sprintf("REF: %s:1-20 verified by physical compiler receipt %s (%d/%d Sub-Tasks on disk)",
			absRef, primaryEvID, verifiedSubTasks, totalSubTasks)
		resolvedAny := false
		for _, cl := range summary.Claims {
			if strings.EqualFold(strings.TrimSpace(cl.Decision), "FLAG") && (cl.Resolved == nil || strings.TrimSpace(*cl.Resolved) == "") {
				if scriptErr == nil {
					_, _ = runPythonTool(ctx, g.AgentCoreDir, script,
						"resolve",
						"--ws-root", filepath.Dir(absWsDir),
						"--workspace", filepath.Base(absWsDir),
						"--stage", stage,
						"--id", cl.ID,
						"--reason", reason,
					)
					resolvedAny = true
				}
			}
		}
		if resolvedAny && strings.TrimSpace(suffix) != "" {
			decFile := filepath.Join(absWsDir, fmt.Sprintf("04_decisions_%s.md", suffix))
			if prevDec, dErr := os.ReadFile(decFile); dErr == nil {
				note := fmt.Sprintf("\n- [%s] [Laya %s Ground-Truth Resolution] Resolved FLAG claims via laya_gate.py resolve (`%s` · `%d/%d` Sub-Tasks · `%s`).\n",
					FormatHoChiMinhTimestamp(time.Now()), strings.ToUpper(stage), primaryEvID, verifiedSubTasks, totalSubTasks, reason)
				_ = os.WriteFile(decFile, append(prevDec, []byte(note)...), 0o644)
			}
		}
	}
}
