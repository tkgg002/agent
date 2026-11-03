package webui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

// RepoTechStackProfile describes the detected language, code fence, and Definition of Done commands for a repository.
type RepoTechStackProfile struct {
	Label       string // "Node.js / TypeScript" | "Go"
	CodeFence   string // "typescript" | "go"
	BuildAndDoD string // "npm run build && npm test" | "go test -race ./..."
	ModelDoD    string // "npm run build && npm test" | "go test -race ./..."
}

func isSkippedBuildOrVendorDir(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch lower {
	case ".git", ".orchestrator", "vendor", "node_modules", "scratch",
		"dist", "build", "out", ".next", "coverage", "target", ".cache", "tmp", ".turbo", "backups_20260929_bson":
		return true
	}
	return strings.HasPrefix(lower, "backups_")
}

func isSkippedGeneratedFile(fileName string) bool {
	lower := strings.ToLower(strings.TrimSpace(fileName))
	return strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".min.js") ||
		strings.HasSuffix(lower, ".bundle.js") ||
		strings.HasSuffix(lower, ".d.ts") ||
		strings.HasSuffix(lower, ".js.map") ||
		strings.HasSuffix(lower, ".d.ts.map")
}

func isCompilerBoilerplateSymbol(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "function adopt(") ||
		strings.Contains(trimmed, "__awaiter") ||
		strings.Contains(trimmed, "__generator") ||
		strings.Contains(trimmed, "__importDefault") ||
		strings.Contains(trimmed, "__exportStar") ||
		strings.Contains(trimmed, "Object.defineProperty(exports")
}

// normalizeOrchestratorDisplayPath strips /Users/.../ prefixes, repo prefixes, a/, b/, ./, and :line suffixes.
func normalizeOrchestratorDisplayPath(rawPath string, repoNames ...string) string {
	p := strings.TrimSpace(rawPath)
	p = strings.Trim(p, "`*\"'")
	if idx := strings.Index(p, " ("); idx > 0 {
		p = strings.TrimSpace(p[:idx])
	}
	p = strings.TrimPrefix(strings.TrimPrefix(p, "b/"), "a/")
	p = strings.TrimPrefix(p, "./")
	if p == "" {
		return ""
	}
	if idx := strings.Index(p, ".orchestrator/"); idx >= 0 {
		return p[idx:]
	}
	for _, rName := range repoNames {
		rTrim := filepath.Base(strings.TrimSpace(rName))
		if rTrim == "" || rTrim == "." || rTrim == "/" {
			continue
		}
		lowerP := strings.ToLower(p)
		lowerRepo := strings.ToLower(rTrim)
		if idx := strings.Index(lowerP, "/"+lowerRepo+"/"); idx >= 0 {
			p = p[idx+len(lowerRepo)+2:]
			break
		}
		if strings.HasPrefix(lowerP, lowerRepo+"/") {
			p = p[len(lowerRepo)+1:]
			break
		}
	}
	if strings.HasPrefix(p, "/Users/") || strings.HasPrefix(p, "/var/folders/") {
		parts := strings.Split(filepath.ToSlash(p), "/")
		if len(parts) > 2 {
			p = strings.Join(parts[len(parts)-2:], "/")
		}
	}
	if colonIdx := strings.Index(p, ":"); colonIdx > 0 {
		p = p[:colonIdx]
	}
	return strings.TrimPrefix(strings.TrimSuffix(p, "/"), "./")
}

// normalizeAndDeduplicateRepoRelativePaths deduplicates relative repo paths case-insensitively while preserving camelCase.
func normalizeAndDeduplicateRepoRelativePaths(rawPaths []string, repoNames ...string) []string {
	seenIdx := make(map[string]int, len(rawPaths))
	out := make([]string, 0, len(rawPaths))
	hasUpper := func(s string) bool {
		for _, r := range s {
			if r >= 'A' && r <= 'Z' {
				return true
			}
		}
		return false
	}
	for _, raw := range rawPaths {
		clean := normalizeOrchestratorDisplayPath(raw, repoNames...)
		if clean == "" || clean == "internal" || strings.HasPrefix(clean, ".orchestrator/") || strings.HasPrefix(strings.ToLower(clean), "dist/") {
			continue
		}
		key := strings.ToLower(clean)
		if existingIdx, exists := seenIdx[key]; exists {
			if !hasUpper(out[existingIdx]) && hasUpper(clean) {
				out[existingIdx] = clean
			}
			continue
		}
		seenIdx[key] = len(out)
		out = append(out, clean)
	}
	return out
}

func normalizeRelativeRepoFiles(rawFiles []string, repoNames ...string) []string {
	return normalizeAndDeduplicateRepoRelativePaths(rawFiles, repoNames...)
}

// detectTargetRepoLanguageAndDoD checks package.json vs go.mod in repoPath to return the canonical code fence and DoD command.
func detectTargetRepoLanguageAndDoD(repoPath string) (codeFenceLang string, dodCmd string) {
	trimmed := strings.TrimSpace(repoPath)
	if trimmed != "" {
		pkgJSON := filepath.Join(trimmed, "package.json")
		goMod := filepath.Join(trimmed, "go.mod")
		if _, errPkg := os.Stat(pkgJSON); errPkg == nil {
			if _, errGo := os.Stat(goMod); os.IsNotExist(errGo) {
				return "typescript", "npm run build && npm test"
			}
		}
		if _, errGo := os.Stat(goMod); errGo == nil {
			return "go", "go test -race ./..."
		}
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "export-service") || strings.Contains(lower, "-web") || strings.Contains(lower, "frontend") {
		return "typescript", "npm run build && npm test"
	}
	return "go", "go test -race ./..."
}

// detectRepoTechStackProfile inspects package.json / go.mod and changed file extensions to return an accurate tech-stack profile.
func detectRepoTechStackProfile(repoPath string, changedFiles []string) RepoTechStackProfile {
	fence, dod := detectTargetRepoLanguageAndDoD(repoPath)
	if fence == "typescript" {
		return RepoTechStackProfile{
			Label:       "Node.js / TypeScript",
			CodeFence:   "typescript",
			BuildAndDoD: dod,
			ModelDoD:    dod,
		}
	}
	for _, f := range changedFiles {
		lower := strings.ToLower(strings.TrimSpace(f))
		if strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx") || strings.HasSuffix(lower, ".js") {
			return RepoTechStackProfile{
				Label:       "Node.js / TypeScript",
				CodeFence:   "typescript",
				BuildAndDoD: "npm run build && npm test",
				ModelDoD:    "npm run build && npm test",
			}
		}
	}
	return RepoTechStackProfile{
		Label:       "Go",
		CodeFence:   "go",
		BuildAndDoD: "go test -race ./...",
		ModelDoD:    "go test -race ./...",
	}
}

func detectCodeFenceLang(relPath, snippet string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(relPath)))
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		return "typescript"
	case ".py":
		return "python"
	case ".sql":
		return "sql"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".go":
		return "go"
	}
	if strings.Contains(snippet, ".ts") ||
		strings.Contains(snippet, "export class ") ||
		strings.Contains(snippet, "export async function ") ||
		strings.Contains(snippet, "interface ") ||
		strings.Contains(snippet, "pgPool") ||
		strings.Contains(snippet, "mongoose") ||
		strings.Contains(snippet, "require(") ||
		strings.Contains(snippet, "async ") {
		return "typescript"
	}
	return "go"
}

func (s *Server) workspaceMatchesTargetRepo(wsDir string, qt queue.QueuedTask) bool {
	if strings.TrimSpace(wsDir) == "" {
		return false
	}
	wsNameLower := strings.ToLower(filepath.Base(wsDir))
	taskIDLower := strings.ToLower(strings.TrimSpace(qt.ExternalID))
	if taskIDLower != "" && strings.Contains(wsNameLower, taskIDLower) {
		return true
	}
	targetRepo := strings.TrimSpace(s.ResolveTaskTargetRepo(qt))
	if targetRepo == "" {
		return true
	}
	repoBase := strings.ToLower(filepath.Base(targetRepo))
	if repoBase == "" || repoBase == "." || repoBase == "/" {
		return true
	}
	if strings.Contains(wsNameLower, repoBase) {
		return true
	}
	entries, err := os.ReadDir(wsDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fn := e.Name()
		if !strings.HasPrefix(fn, "01_requirements") && !strings.HasPrefix(fn, "09_tasks_solution") {
			continue
		}
		raw, rErr := os.ReadFile(filepath.Join(wsDir, fn))
		if rErr == nil && strings.Contains(strings.ToLower(string(raw)), repoBase) {
			return true
		}
	}
	return false
}

// extractPromptTargetTechEntity identifies the primary target technology being added/implemented in the task title or description.
func extractPromptTargetTechEntity(title, description, userPrompt string) string {
	titleLower := strings.ToLower(title + " " + userPrompt)
	descLower := strings.ToLower(description)
	candidates := []string{"clickhouse", "elasticsearch", "opensearch", "redis", "kafka", "nats", "rabbitmq", "postgres", "postgresql", "mongodb"}
	// 1. Check explicit title / userPrompt first (e.g. "làm thêm qua clickhouse")
	for _, tech := range candidates {
		if strings.Contains(titleLower, "thêm "+tech) ||
			strings.Contains(titleLower, "qua "+tech) ||
			strings.Contains(titleLower, "sang "+tech) ||
			strings.Contains(titleLower, "làm "+tech) ||
			strings.Contains(titleLower, "triển khai "+tech) {
			return tech
		}
	}
	if strings.Contains(titleLower, "clickhouse") || strings.Contains(descLower, "clickhouse") {
		return "clickhouse"
	}
	for _, tech := range candidates {
		if strings.Contains(titleLower, tech) {
			return tech
		}
	}
	for _, tech := range candidates {
		if strings.Contains(descLower, "triển khai") && strings.Contains(descLower, tech) {
			return tech
		}
	}
	return ""
}

// inferNewAdapterRelPathForTech checks if the repository has an existing dbHandler/adapter directory and proposes the new adapter relative path if not yet on disk.
func inferNewAdapterRelPathForTech(repoPath, targetTech string, existingRelPaths []string) string {
	tech := strings.ToLower(strings.TrimSpace(targetTech))
	if tech == "" || tech == "mongodb" || tech == "mongo" {
		return ""
	}
	if tech == "postgresql" {
		tech = "postgres"
	}
	ext := ".go"
	fence, _ := detectTargetRepoLanguageAndDoD(repoPath)
	if fence == "typescript" {
		ext = ".ts"
	}
	for _, rel := range existingRelPaths {
		relSlash := filepath.ToSlash(rel)
		lower := strings.ToLower(relSlash)
		if strings.Contains(lower, "dbhandler/") || strings.Contains(lower, "adapters/database/") {
			dir := filepath.ToSlash(filepath.Dir(relSlash))
			return dir + "/" + tech + ext
		}
	}
	if fence == "typescript" {
		if _, err := os.Stat(filepath.Join(repoPath, "src/common/adapters/database/dbHandler")); err == nil {
			return "src/common/adapters/database/dbHandler/" + tech + ".ts"
		}
	}
	return ""
}

