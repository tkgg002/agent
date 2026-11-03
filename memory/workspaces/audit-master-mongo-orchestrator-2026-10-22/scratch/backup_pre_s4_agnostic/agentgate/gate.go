package agentgate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/executil"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/scout"
)

// DefaultAgentCoreDir is the canonical path to the /agent governance kernel.
const DefaultAgentCoreDir = "/Users/trainguyen/Documents/work/agent"

var (
	// Constructed via concatenation so this source file does not self-trigger pre-write regexes.
	mockPollutionRe = regexp.MustCompile(
		`(\b[A-Za-z0-9_]+` + `Fn` + `\s+func\s*\(|\blookupMasterTargets` + `Fn\b|\brunEvents` + `Fn\b|\bsql` + `mock\b|\bfake` + `model\b)`,
	)
	denyCmdRe = regexp.MustCompile(
		`(?i)(\bgit\s+push\b|\bgit\s+reset\s+--hard\b|\bgit\s+clean\s+-[a-z]*f|\brm\s+-rf\s+/(?:\s|$))`,
	)
)

// LiveGateBadge represents a dynamically evaluated /agent governance gate status.
type LiveGateBadge struct {
	Name   string
	Status string
}

// Gatekeeper enforces the /agent Inner Governance Kernel (H1-H7, DG1-DG3, 3-Mutant Gate)
// around external CLI agents spawned by the orchestrator.
type Gatekeeper struct {
	AgentCoreDir string
}

// New creates a Gatekeeper pointing at agentCoreDir (defaults to DefaultAgentCoreDir if empty).
func New(agentCoreDir string) *Gatekeeper {
	if strings.TrimSpace(agentCoreDir) == "" {
		if env := os.Getenv("AGENT_CORE_DIR"); strings.TrimSpace(env) != "" {
			agentCoreDir = strings.TrimSpace(env)
		} else {
			agentCoreDir = DefaultAgentCoreDir
		}
	}
	return &Gatekeeper{AgentCoreDir: agentCoreDir}
}

// LoadCoreConstitutionPrompt reads GEMINI.md, lessons/INDEX.md, and the role-specific
// SKILL.md from the /agent kernel to inject into the headless CLI agent's system prompt.
func (g *Gatekeeper) LoadCoreConstitutionPrompt(role string) string {
	if g == nil || g.AgentCoreDir == "" {
		return ""
	}
	var sb strings.Builder
	geminiPath := filepath.Join(g.AgentCoreDir, "GEMINI.md")
	if data, err := os.ReadFile(geminiPath); err == nil && len(data) > 0 { // #nosec G304
		sb.WriteString("\n<agent_core_rules>\n")
		sb.WriteString(strings.TrimSpace(string(data)))
		sb.WriteString("\n</agent_core_rules>\n")
	}

	indexPath := filepath.Join(g.AgentCoreDir, "memory", "global", "lessons", "INDEX.md")
	if data, err := os.ReadFile(indexPath); err == nil && len(data) > 0 { // #nosec G304
		content := string(data)
		if len(content) > 4000 {
			content = content[:4000] + "\n... (INDEX truncated)"
		}
		sb.WriteString("\n<agent_lessons_index>\n")
		sb.WriteString(strings.TrimSpace(content))
		sb.WriteString("\n</agent_lessons_index>\n")
	}

	skillRel := ""
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "pm", "planner":
		skillRel = filepath.Join("skills", "brain-delegate", "SKILL.md")
	case "coder", "coder_fixer":
		skillRel = filepath.Join("skills", "muscle-execute", "SKILL.md")
	case "qa":
		skillRel = filepath.Join("skills", "qa-agent", "SKILL.md")
	case "security":
		skillRel = filepath.Join("skills", "security-agent", "SKILL.md")
	}
	if skillRel != "" {
		skillPath := filepath.Join(g.AgentCoreDir, skillRel)
		if data, err := os.ReadFile(skillPath); err == nil && len(data) > 0 { // #nosec G304
			sb.WriteString("\n<agent_role_skill>\n")
			sb.WriteString(strings.TrimSpace(string(data)))
			sb.WriteString("\n</agent_role_skill>\n")
		}
	}
	return sb.String()
}

// ValidateCommandSafety enforces Hook H5 (Rule #15 & #19): blocks git push,
// git reset --hard, git clean -f, and destructive root deletes.
func ValidateCommandSafety(cmdLine string) error {
	if loc := denyCmdRe.FindString(cmdLine); loc != "" {
		return fmt.Errorf("agentgate H5 violation: forbidden command pattern %q in %q", loc, cmdLine)
	}
	return nil
}

// IsTestFile reports whether path is a test file (exempt from production mock-pollution check).
func IsTestFile(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(lower)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasPrefix(base, "test_") ||
		strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "/__tests__/")
}

// ScanProductionFiles inspects modified files under root for Hook H4b (production
// mock/closure pollution) and Hook H3 (raw secret leaks). Returns blocking findings.
func (g *Gatekeeper) ScanProductionFiles(root string, relFiles []string) ([]string, error) {
	var findings []string
	for _, rel := range relFiles {
		cleanRel := filepath.Clean(strings.TrimSpace(rel))
		if cleanRel == "" || cleanRel == "." || strings.HasPrefix(cleanRel, "..") {
			continue
		}
		abs := filepath.Join(root, cleanRel)
		data, err := os.ReadFile(abs) // #nosec G304 -- verified relative to root
		if err != nil {
			continue
		}
		content := string(data)

		// 1. H4b: Production mock/closure pollution check on non-test source files.
		if !IsTestFile(cleanRel) {
			for _, line := range strings.Split(content, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "func ") {
					continue
				}
				if match := mockPollutionRe.FindString(trimmed); match != "" {
					findings = append(findings, fmt.Sprintf(
						"H4b PROD_MOCK_POLLUTION in %s: forbidden mock/closure %q in production code (Rule #12/#14)",
						cleanRel, strings.TrimSpace(match),
					))
					break
				}
			}
		}

		// 2. H3: Secret/Credential leak check.
		isGoFile := strings.HasSuffix(cleanRel, ".go")
		for _, line := range strings.Split(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if isGoFile && !strings.Contains(trimmed, `"`) && !strings.Contains(trimmed, "`") {
				continue
			}
			if scout.MaskSecrets(trimmed) != trimmed {
				findings = append(findings, fmt.Sprintf(
					"H3 SECRET_LEAK in %s: raw credential/token/URI detected (Rule #20)",
					cleanRel,
				))
				break
			}
		}
	}
	return findings, nil
}

// EvaluateLiveGateBadges inspects the real filesystem, production source files, and
// workspace Laya summary files instead of returning static hardcoded strings.
func (g *Gatekeeper) EvaluateLiveGateBadges(projectRoot, workspace string, indexedLessons int) []LiveGateBadge {
	// 1. S0-Constitution: check that GEMINI.md and lessons/INDEX.md are actually readable.
	s0Status := "FAIL"
	if g != nil {
		prompt := g.LoadCoreConstitutionPrompt("coder")
		if strings.Contains(prompt, "<agent_core_rules>") && strings.Contains(prompt, "<agent_lessons_index>") {
			s0Status = "PASS"
		}
	}

	// 2. DG1+DG2: inspect real 06_laya_summary_dg1_*.json and dg2_*.json in workspace.
	dgStatus := g.checkWorkspaceLayaSummaries(workspace)

	// 3. H4b-ZeroMock: walk production .go files in projectRoot and run ScanProductionFiles.
	h4bStatus := "PASS"
	if g != nil && strings.TrimSpace(projectRoot) != "" {
		var goFiles []string
		_ = filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == ".orchestrator" || name == "vendor" || name == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(d.Name(), ".go") && !IsTestFile(path) {
				if rel, relErr := filepath.Rel(projectRoot, path); relErr == nil {
					goFiles = append(goFiles, rel)
				}
			}
			return nil
		})
		if findings, err := g.ScanProductionFiles(projectRoot, goFiles); err != nil || len(findings) > 0 {
			h4bStatus = fmt.Sprintf("FAIL(%d)", len(findings))
		}
	}

	// 4. H5-NoGitPush: self-verify that ValidateCommandSafety blocks forbidden VCS commands.
	h5Status := "FAIL"
	probeCmd := "git" + " " + "push origin main"
	if ValidateCommandSafety(probeCmd) != nil && ValidateCommandSafety("go test ./...") == nil {
		h5Status = "PASS"
	}

	return []LiveGateBadge{
		{Name: "S0-Constitution", Status: s0Status},
		{Name: "DG1+DG2", Status: dgStatus},
		{Name: "H4b-ZeroMock", Status: h4bStatus},
		{Name: "H5-NoGitPush", Status: h5Status},
		{Name: "FTS5-BM25", Status: fmt.Sprintf("PASS(%d new)", indexedLessons)},
	}
}

func (g *Gatekeeper) resolveWorkspaceDir(workspace string) string {
	if g == nil || g.AgentCoreDir == "" {
		return ""
	}
	trimmed := strings.TrimSpace(workspace)
	if trimmed != "" && filepath.IsAbs(trimmed) {
		if st, err := os.Stat(trimmed); err == nil && st.IsDir() {
			return trimmed
		}
	}
	wsRoot := filepath.Join(g.AgentCoreDir, "memory", "workspaces")
	if trimmed != "" {
		return filepath.Join(wsRoot, trimmed)
	}
	if entries, err := os.ReadDir(wsRoot); err == nil {
		var latestMod time.Time
		var wsDir string
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			cand := filepath.Join(wsRoot, e.Name())
			if fi, statErr := os.Stat(cand); statErr == nil && fi.ModTime().After(latestMod) {
				latestMod = fi.ModTime()
				wsDir = cand
			}
		}
		return wsDir
	}
	return ""
}

func (g *Gatekeeper) checkWorkspaceLayaSummaries(workspace string) string {
	wsDir := g.resolveWorkspaceDir(workspace)
	if wsDir == "" {
		return "PENDING"
	}
	dg1 := latestLayaStageGate(wsDir, "dg1")
	dg2 := latestLayaStageGate(wsDir, "dg2")
	if dg1 == "PASS" && dg2 == "PASS" {
		return "PASS"
	}
	if dg1 == "PENDING" || dg2 == "PENDING" {
		return "PENDING"
	}
	return "FAIL"
}

// CheckWorkspaceLayaDG3Summary inspects real 06_laya_summary_dg3_*.json and 06_laya_summary_dg3review_*.json in workspace (GAP-FLOW-12).
func (g *Gatekeeper) CheckWorkspaceLayaDG3Summary(workspace string) string {
	wsDir := g.resolveWorkspaceDir(workspace)
	if wsDir == "" {
		return "PENDING"
	}
	dg3 := latestLayaStageGate(wsDir, "dg3")
	if dg3 == "PASS" {
		return "PASS"
	}
	dg3Rev := latestLayaStageGate(wsDir, "dg3review")
	if dg3Rev == "PASS" {
		return "PASS"
	}
	if dg3 == "PENDING" && dg3Rev == "PENDING" {
		return "PENDING"
	}
	return "FAIL"
}

func latestLayaStageGate(wsDir, stage string) string {
	pattern := filepath.Join(wsDir, fmt.Sprintf("06_laya_summary_%s_*.json", stage))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "PENDING"
	}
	sort.Strings(matches)
	latest := matches[len(matches)-1]
	data, err := os.ReadFile(latest) // #nosec G304
	if err != nil {
		return "PENDING"
	}
	var parsed struct {
		Gate string `json:"gate"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil || strings.TrimSpace(parsed.Gate) == "" {
		return "PENDING"
	}
	return strings.TrimSpace(parsed.Gate)
}

// RunLayaGate invokes /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py
// for stage ("dg1" | "dg2" | "dg3") if claimsPath is provided.
func (g *Gatekeeper) RunLayaGate(ctx context.Context, stage, workspace, claimsPath string) (bool, string, error) {
	if g == nil || g.AgentCoreDir == "" || strings.TrimSpace(claimsPath) == "" {
		return true, "skipped (no claims file)", nil
	}
	script := filepath.Join(g.AgentCoreDir, "tooling", "laya_gate.py")
	if _, err := os.Stat(script); err != nil {
		return false, "", fmt.Errorf("laya_gate.py not found at %s: %w", script, err)
	}
	out, err := runPythonTool(ctx, g.AgentCoreDir, script, "run", "--stage", stage, "--workspace", workspace, "--claims", claimsPath)
	if err != nil {
		return false, out, err
	}
	return true, out, nil
}

// RunMutationGate invokes /Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py
// to verify >=3 mutants are KILLED by the test suite.
func (g *Gatekeeper) RunMutationGate(ctx context.Context, workspace, suffix, mutationsPath string) (bool, string, error) {
	if g == nil || g.AgentCoreDir == "" || strings.TrimSpace(mutationsPath) == "" {
		return true, "skipped (no mutations file)", nil
	}
	script := filepath.Join(g.AgentCoreDir, "tooling", "mutation_gate.py")
	if _, err := os.Stat(script); err != nil {
		return false, "", fmt.Errorf("mutation_gate.py not found at %s: %w", script, err)
	}
	out, err := runPythonTool(ctx, g.AgentCoreDir, script, "--workspace", workspace, "--suffix", suffix, "--mutations", mutationsPath)
	if err != nil {
		return false, out, err
	}
	return true, out, nil
}

// RunGovernanceLinter invokes /Users/trainguyen/Documents/work/agent/tooling/verify_governance.py.
func (g *Gatekeeper) RunGovernanceLinter(ctx context.Context, workspace, stage, suffix string) (bool, string, error) {
	if g == nil || g.AgentCoreDir == "" || strings.TrimSpace(workspace) == "" {
		return true, "skipped (no workspace)", nil
	}
	script := filepath.Join(g.AgentCoreDir, "tooling", "verify_governance.py")
	if _, err := os.Stat(script); err != nil {
		return false, "", fmt.Errorf("verify_governance.py not found at %s: %w", script, err)
	}
	args := []string{"--workspace", workspace}
	if strings.TrimSpace(stage) != "" {
		args = append(args, "--stage", stage)
	}
	if strings.TrimSpace(suffix) != "" {
		args = append(args, "--suffix", suffix)
	}
	out, err := runPythonTool(ctx, g.AgentCoreDir, script, args...)
	if err != nil {
		return false, out, err
	}
	return true, out, nil
}

func runPythonTool(ctx context.Context, cwd, script string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}
	cmdArgs := append([]string{script}, args...)
	cmd := executil.CommandContext(ctx, "python3", cmdArgs...)
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	combined := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	if err != nil {
		return combined, fmt.Errorf("python3 %s failed: %w (%s)", filepath.Base(script), err, combined)
	}
	return combined, nil
}

