package agentgate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/agentgate"
)

func TestGatekeeperConstitutionAndProductionScan(t *testing.T) {
	tmp := t.TempDir()

	// 1. Setup a mini /agent core directory and verify LoadCoreConstitutionPrompt.
	coreDir := filepath.Join(tmp, "agent")
	_ = os.MkdirAll(filepath.Join(coreDir, "memory", "global", "lessons"), 0o750)
	_ = os.MkdirAll(filepath.Join(coreDir, "skills", "brain-delegate"), 0o750)
	_ = os.WriteFile(filepath.Join(coreDir, "GEMINI.md"), []byte("# GEMINI CORE RULES"), 0o600)
	_ = os.WriteFile(filepath.Join(coreDir, "memory", "global", "lessons", "INDEX.md"), []byte("# LESSONS INDEX"), 0o600)
	_ = os.WriteFile(filepath.Join(coreDir, "skills", "brain-delegate", "SKILL.md"), []byte("# BRAIN SKILL"), 0o600)

	gk := agentgate.New(coreDir)
	prompt := gk.LoadCoreConstitutionPrompt("pm")
	if !strings.Contains(prompt, "GEMINI CORE RULES") || !strings.Contains(prompt, "LESSONS INDEX") || !strings.Contains(prompt, "BRAIN SKILL") {
		t.Fatalf("expected full constitution prompt for pm, got: %s", prompt)
	}

	// 2. Verify ValidateCommandSafety (Hook H5).
	if err := agentgate.ValidateCommandSafety("git push origin main"); err == nil {
		t.Fatalf("expected git push to be blocked by H5")
	}
	if err := agentgate.ValidateCommandSafety("git reset --hard HEAD~1"); err == nil {
		t.Fatalf("expected git reset --hard to be blocked by H5")
	}
	if err := agentgate.ValidateCommandSafety("go test ./..."); err != nil {
		t.Fatalf("expected go test to be allowed, got: %v", err)
	}

	// 3. Verify ScanProductionFiles (Hook H4b mock pollution & Hook H3 secret leak).
	projDir := filepath.Join(tmp, "proj")
	_ = os.MkdirAll(projDir, 0o750)

	// Construct polluted code dynamically so test file remains clean.
	pollutedCode := "package p\ntype S struct {\n\trun" + "Events" + "Fn func() error\n}\n"
	_ = os.WriteFile(filepath.Join(projDir, "service.go"), []byte(pollutedCode), 0o600)
	_ = os.WriteFile(filepath.Join(projDir, "service_test.go"), []byte(pollutedCode), 0o600)

	cleanCode := "package p\nfunc Add(a, b int) int { return a + b }\n"
	_ = os.WriteFile(filepath.Join(projDir, "clean.go"), []byte(cleanCode), 0o600)

	findings, err := gk.ScanProductionFiles(projDir, []string{"service.go", "service_test.go", "clean.go"})
	if err != nil {
		t.Fatalf("ScanProductionFiles err: %v", err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0], "service.go") || !strings.Contains(findings[0], "H4b PROD_MOCK_POLLUTION") {
		t.Fatalf("expected exactly 1 H4b finding on service.go (and 0 on service_test.go/clean.go), got: %+v", findings)
	}

	// 4. Verify EvaluateLiveGateBadges reports BLOCK on polluted projDir and PENDING when DG1/DG2 summaries are absent.
	badges := gk.EvaluateLiveGateBadges(projDir, "missing-ws", 5)
	badgeMap := make(map[string]string)
	for _, b := range badges {
		badgeMap[b.Name] = b.Status
	}
	if badgeMap["S0-Constitution"] != "PASS" {
		t.Errorf("expected S0-Constitution PASS, got %q", badgeMap["S0-Constitution"])
	}
	if badgeMap["DG1+DG2"] != "PENDING" {
		t.Errorf("expected DG1+DG2 PENDING when workspace summaries do not exist, got %q", badgeMap["DG1+DG2"])
	}
	if !strings.HasPrefix(badgeMap["H4b-ZeroMock"], "FAIL") {
		t.Errorf("expected H4b-ZeroMock FAIL when service.go has mock field, got %q", badgeMap["H4b-ZeroMock"])
	}
}

