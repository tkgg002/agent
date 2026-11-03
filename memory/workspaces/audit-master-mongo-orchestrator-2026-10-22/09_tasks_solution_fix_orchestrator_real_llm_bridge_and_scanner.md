# 09. Technical Solution & Code Demo — Khắc phục Toàn diện Hệ thống `ai-orchestrator` (`fix_orchestrator_real_llm_bridge_and_scanner`)

## 1. Phương án Kiến trúc Duy nhất & Tối ưu Nhất (Single Optimal Architecture — Rule #9.5)
Để giải quyết triệt để 5 lỗi chí mạng của `ai-orchestrator` mà hoàn toàn tuân thủ kỷ luật **Single-Responsibility (`NOW <= HEAD` trên các file lớn)**:
1. **Tầng Runner (`internal/runner/antigravity.go`):**
   - Loại bỏ nhánh gọi nhầm `agy -p` (vì `agy` là binary Electron IDE launcher không hỗ trợ `-p`).
   - Chuẩn hóa tên model (`normalizeAntigravityModel`): strip `" (antigravity)"` và `"antigravity/"`. Khi gọi `gemini` CLI, nếu model là nhãn nội bộ Antigravity (`gemini-4-argon`, `gemini-3.1-pro-high`) thì không truyền `-m` (tránh lỗi `429 limit: 0 for model: gemini-4-argon`), đồng thời bắt buộc truyền `--skip-trust` và biến môi trường `GEMINI_CLI_TRUST_WORKSPACE=true`, `COLORTERM=truecolor`, `TERM=xterm-256color` (tránh lỗi `exit code 55`).
   - Lọc bỏ các dòng log stderr/tool warning (`Error executing tool read_file...`, `Warning: Basic terminal...`) khỏi luồng token trả về.
2. **Tầng WebUI LLM Bridge (`internal/webui/task_llm_bridge.go` — File mới đơn trách nhiệm):**
   - Đấu nối trực tiếp `handleTaskPlanningChat` (`task_chat.go`), `renderContextualDiscussReply` (`task_chat_knowledge.go`) và `handleTelegramTechnicalQuestion` (`telegram_forum_topic.go`) vào `AntigravityRunner.Complete(ctx, req)` thông qua `s.invokeArchitectLLM(...)`.
   - Nạp đúng Ground-Truth Context của Task hiện tại: câu hỏi/chỉ đạo mới của User + trích đoạn code thật từ các file nguồn sạch của `qt.TargetRepo` + `09_tasks_solution.md` + `subtasks.json`.
3. **Tầng Scanner & Language Fence (`internal/webui/task_inspector.go`, `internal/webui/unified_doc_format.go`):**
   - Chặn tuyệt đối thư mục build (`dist`, `build`, `out`, `.next`, `coverage`, `target`, `.cache`) và file biên dịch (`*.min.js`, `*.d.ts`, `*.map`) trong `inspectTargetRepoSymbols`.
   - Chặn hàm boilerplate TypeScript (`function adopt(value)`, `__awaiter`, `__generator`, `__importDefault`).
   - Ưu tiên file nguồn (`.ts`, `.tsx`, `.go`, `.py`, `.sql`) đứng trước `.js`.
   - Thay thế toàn bộ chỗ hardcode ````go` bằng `detectCodeFenceLang(relPath, codeSnippet)`.
   - Khóa `FindMatchingAgentWorkspace` theo `TargetRepo` để chấm dứt lỗi lấy nhầm hồ sơ của workspace khác (`NATS/Kafka DLQ`), và sửa `extractTaskDeepKnowledge` không bốc mục `Gate G2: Red -> Green` làm Root Cause.
4. **Tầng Nhận diện Chỉ đạo Kiến trúc & Re-Open/Re-Plan (`internal/webui/task_chat.go`):**
   - Nâng cấp `classifyChatActionMode` nhận diện các câu chỉ đạo đổi hướng giải pháp/kiến trúc tự nhiên của User (*"giải pháp của anh là"*, *"hiện tại mày đang làm"*, *"tạo 1 cái type v2"*, *"v1 vẫn là cái cũ"*, *"không đè lên"*, *"tách v1/v2"*).
   - Khi nhận chỉ đạo đổi hướng kiến trúc (`revise_solution`), tự động **re-open Task từ `DONE` về `NEEDS_YOU`**, xóa `.user-approved`, reset Sub-Tasks khỏi trạng thái `DONE` ảo, tái tạo `09_tasks_solution.md` & `subtasks.json` bằng kết quả suy luận của `gemini-4-argon` (trên file nguồn sạch `.ts`/`.go`), và báo cáo trung thực trạng thái chờ duyệt (`NEEDS_YOU · 0/N DONE`).

---

## 2. Code Demo Cụ thể cho Từng Hạng mục (`DG2`)

### 2.1. Khắc phục `internal/runner/antigravity.go` (Truyền `--skip-trust`, `GEMINI_CLI_TRUST_WORKSPACE=true`, chuẩn hóa `gemini-4-argon`)
```go
// File: internal/runner/antigravity.go
func normalizeAntigravityModel(raw string) (cleanModel string, passModelFlagToGemini bool) {
	m := strings.TrimSpace(raw)
	if idx := strings.Index(strings.ToLower(m), "(antigravity)"); idx >= 0 {
		m = strings.TrimSpace(m[:idx])
	}
	m = strings.TrimPrefix(m, "antigravity/")
	lower := strings.ToLower(m)
	// Internal Antigravity IDE aliases like "gemini-4-argon" have limit: 0 on public Gemini CLI -m flag;
	// let gemini CLI use its authenticated default model when an internal alias is selected.
	if strings.Contains(lower, "argon") || strings.Contains(lower, "antigravity") {
		return m, false
	}
	return m, strings.HasPrefix(lower, "gemini-2.") || strings.HasPrefix(lower, "gemini-1.")
}

func (r AntigravityRunner) resolveCommand(prompt, systemPrompt, model string) (string, []string, string, error) {
	cleanModel, passGeminiModel := normalizeAntigravityModel(model)
	geminiBin := findGeminiBinary()
	claudeBin := findClaudeBinary()

	lowerModel := strings.ToLower(cleanModel)
	if strings.Contains(lowerModel, "claude") || strings.Contains(lowerModel, "opus") || strings.Contains(lowerModel, "sonnet") {
		if claudeBin != "" {
			args := []string{"-p", prompt, "--output-format", "stream-json", "--verbose", "--no-session-persistence"}
			if cleanModel != "" {
				args = append(args, "--model", cleanModel)
			}
			if systemPrompt != "" {
				args = append(args, "--system-prompt", systemPrompt)
			}
			return claudeBin, args, "claude", nil
		}
	}

	if geminiBin != "" {
		fullPrompt := prompt
		if strings.TrimSpace(systemPrompt) != "" {
			fullPrompt = systemPrompt + "\n\n" + prompt
		}
		args := []string{"--skip-trust", "-p", fullPrompt}
		if passGeminiModel && cleanModel != "" {
			args = append(args, "-m", cleanModel)
		}
		return geminiBin, args, "gemini", nil
	}
	return "", nil, "", fmt.Errorf("antigravity runner: no working gemini or claude CLI found on host")
}
```

### 2.2. Đấu nối thật `AntigravityRunner` (`gemini-4-argon`) vào `internal/webui/task_llm_bridge.go`
```go
// File: internal/webui/task_llm_bridge.go
package webui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/runner"
)

type ArchitectLLMAnalysis struct {
	RootCauses   []string `json:"root_causes"`
	Solutions    []string `json:"solutions"`
	CodeDemo     string   `json:"code_demo"`
	CodeLang     string   `json:"code_lang"`
	DirectAnswer string   `json:"direct_answer"`
}

// invokeArchitectLLM connects Co-Architect Studio (/brain-delegate) directly to AntigravityRunner (gemini-4-argon)
// with real source snippets from qt.TargetRepo (excluding dist/ and build/).
func (s *Server) invokeArchitectLLM(
	ctx context.Context,
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	userPrompt, selectedModel, actionMode string,
) (*ArchitectLLMAnalysis, error) {
	llmCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	symbolHits, _ := inspectTargetRepoSymbols(qt.TargetRepo, qt.Title+" "+userPrompt)
	var repoContext strings.Builder
	for i, h := range symbolHits {
		if i >= 4 {
			break
		}
		repoContext.WriteString(fmt.Sprintf("// File: %s:%d (%s)\n%s\n\n", h.RelPath, h.LineNum, h.Signature, h.Snippet))
	}

	sysPrompt := "[System Note: Đã đọc GEMINI.md và lessons/INDEX.md — trả lời trực tiếp ngay bằng tiếng Việt, KHÔNG gọi tool read_file.]\n" +
		"Bạn là BRAIN · Architect (Slot A · /brain-delegate) trong AI-Orchestrator. " +
		"Phân tích trực tiếp yêu cầu/chỉ đạo mới nhất của User dựa trên mã nguồn thật của repository đích. " +
		"Tuyệt đối không nhai lại tiêu đề cũ, không chạm vào thư mục build (dist/), và tuân thủ đúng mẫu kiến trúc User yêu cầu (ví dụ: Strategy/Factory Pattern giữ nguyên V1 và tách V2 chạy song song)."

	userMsg := fmt.Sprintf(
		"Task ID: %s\nTask Title: %s\nTarget Repo: %s\nChế độ: %s\nChỉ đạo/Câu hỏi mới của User: %q\n\nMã nguồn thực tế trên repo:\n%s\n\n"+
			"Hãy trả về JSON hợp lệ với cấu trúc: {\"root_causes\": [\"...\"], \"solutions\": [\"...\"], \"code_lang\": \"ts|go|js|sql\", \"code_demo\": \"...\"}",
		qt.ExternalID, qt.Title, qt.TargetRepo, actionMode, userPrompt, repoContext.String(),
	)

	agRunner := runner.NewAntigravityRunner("", qt.TargetRepo)
	stream, err := agRunner.Complete(llmCtx, runner.CompletionRequest{
		Model:        selectedModel,
		SystemPrompt: sysPrompt,
		Messages:     []runner.ConvMessage{{Role: "user", Content: userMsg}},
	})
	if err != nil {
		return nil, err
	}
	var raw strings.Builder
	for tok := range stream {
		if tok.Error != nil {
			return nil, tok.Error
		}
		raw.WriteString(tok.Text)
	}
	return parseArchitectLLMAnalysis(raw.String(), symbolHits), nil
}
```

### 2.3. Chặn `dist/`, `build/`, Boilerplate `function adopt(value)` và Tự động chọn Language Fence (`internal/webui/task_inspector.go` & `unified_doc_format.go`)
```go
// File: internal/webui/task_inspector.go
func isSkippedBuildOrVendorDir(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ".git", ".orchestrator", "vendor", "node_modules", "scratch",
		"dist", "build", "out", ".next", "coverage", "target", ".cache", "tmp", ".turbo":
		return true
	}
	return strings.HasPrefix(name, "backups_")
}

func isCompilerBoilerplateSymbol(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "function adopt(") ||
		strings.Contains(trimmed, "__awaiter") ||
		strings.Contains(trimmed, "__generator") ||
		strings.Contains(trimmed, "__importDefault") ||
		strings.Contains(trimmed, "Object.defineProperty(exports")
}

func detectCodeFenceLang(relPath, snippet string) string {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(relPath))) {
	case ".ts", ".tsx":
		return "ts"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "js"
	case ".py":
		return "python"
	case ".sql":
		return "sql"
	case ".sh":
		return "bash"
	case ".go":
		return "go"
	}
	if strings.Contains(snippet, "interface ") || strings.Contains(snippet, "export class ") || strings.Contains(snippet, "mongoose") {
		return "ts"
	}
	return "go"
}
```

### 2.4. Nhận diện Chỉ đạo Đổi hướng Kiến trúc (`classifyChatActionMode`) & Re-Open Task từ `DONE` về `NEEDS_YOU` (`internal/webui/task_chat.go`)
```go
// File: internal/webui/task_chat.go
func classifyChatActionMode(message string) string {
	lower := strings.ToLower(strings.TrimSpace(message))
	// ...
	for _, kw := range []string{
		"sửa lại giải pháp", "cập nhật giải pháp", "đổi giải pháp",
		"sửa phương án", "thay đổi thiết kế", "bổ sung vào giải pháp",
		"sửa 09_tasks_solution", "giải pháp của anh là", "ý của anh là",
		"hiện tại mày đang làm", "tạo 1 cái type v2", "v1 vẫn là cái cũ",
		"không đè lên", "bỏ cái export type cũ đi, đè lên nó",
		"tách v1", "tách v2", "sửa lại hướng", "re-plan", "làm lại giải pháp",
	} {
		if strings.Contains(lower, kw) {
			return "revise_solution"
		}
	}
	return "discuss"
}
```
Khi `actionMode == "revise_solution"`:
1. Xóa `.user-approved` trong `taskArtifactDir`, gọi `s.queue.UpdateTaskStatus(qt.ID, queue.StatusNeedsYou, "")` để chuyển Task từ `DONE` về `NEEDS_YOU`.
2. Gọi `s.replanTaskWithArchitectDirective(ctx, qt, msg, selectedModel)` để cập nhật `09_tasks_solution.md`, `08_tasks.md` và `subtasks.json` (loại bỏ mọi điểm chạm `dist/`, cập nhật các Sub-Task sang kiến trúc mới với trạng thái `pending` / `NEEDS_YOU`, `0/N DONE`).
3. Trả lời trực tiếp bằng kết quả từ `gemini-4-argon` kèm trạng thái thật `Stage S3_SOLUTION (NEEDS_YOU · 0/2 DONE — Đã Re-Open & Cập nhật Plan theo chỉ đạo mới, chờ anh APPROVE)`.

### 2.5. Cách ly tuyệt đối `.orchestrator/tasks/<task_id>` khỏi `memory/workspaces/`, xóa bỏ Regex `"Phương án khắc phục:"`, khóa `selectedChainKey` & Chẩn đoán 5 Điểm nghẽn Transmute cho `cdc-01` (`TASK-ORCH-05` & `TASK-ORCH-06`)

#### (A) Cách ly `.orchestrator/tasks/<task_id>` khỏi `agent/memory/workspaces/` (`internal/webui/task_inspector.go:568-575, 868-886, 1292-1299, 3223-3256`)
- **Nguyên nhân nhiễm chéo (`VERIFIED`):** Trong `cdc-01` (*"kiểm tra cho tao vì sao transmute bị thiếu"*), phần mô tả có dòng `[mongodb] traitestpbs / payment-bill-service ... master_payment_bill_service`. Do `ResolveTaskAffectedRepos` (`task_inspector.go:569`) và `FindMatchingAgentWorkspace` (`task_inspector.go:877`) kiểm tra `strings.Contains(lower, "mongo") && strings.Contains(lower, "master")`, hệ thống khớp nhầm `cdc-01` vào `feat-master-mongo-destination-2026-10-20` rồi copy nguyên văn `08_tasks_master_mongo.md` (`TASK-M01..M10`) và `09_tasks_solution_master_mongo.md` sang `.orchestrator/tasks/cdc-01/`.
- **Giải pháp khắc phục:**
  1. Trong `ResolveTaskAffectedRepos` (`task_inspector.go:568-575`) và `FindMatchingAgentWorkspace` (`task_inspector.go:868-939`): Chỉ khớp `feat-master-mongo-destination-2026-10-20` khi `isMasterMongoDestinationTask(qt)` == `true` (tức chỉ dành riêng cho đúng task `"tích hợp Master MongoDB Destination"` / `TASK-11`), và **tắt hoàn toàn nhánh quét tự do `memory/workspaces/`** đối với mọi Task khác (`cdc-01`, `TASK-13`, `Core-01`, v.v.).
  2. Trong `loadOrGenerateSubTasks` (`task_inspector.go:1292-1299`) và `generateDeepTaskArtifacts` (`task_inspector.go:3223-3256`): **Cấm tuyệt đối** copy `ws08Path` / `ws09Path` từ `memory/workspaces/` đè vào `.orchestrator/tasks/<task_id>/08_tasks.md` và `09_tasks_solution.md` của task mới.

#### (B) Xóa bỏ Regex String-Prefixing `"- **Phương án khắc phục:** Workspace**: ..."` và luôn gọi `invokeArchitectLLM` (`internal/webui/task_chat_knowledge.go:70-205` & `internal/webui/unified_doc_format.go:298-311`)
```go
// File: internal/webui/task_chat_knowledge.go
func (s *Server) renderContextualDiscussReply(...) string {
	// ...
	// LUÔN gọi invokeArchitectLLM trước cho mọi câu hỏi/thảo luận trong /brain-delegate!
	var rootCauses, solutions []string
	var codeSnippet string
	if llmRes, err := s.invokeArchitectLLM(context.Background(), qt, subTasks, cleanMsg, selectedModel, "discuss"); err == nil && llmRes != nil {
		rootCauses = llmRes.RootCauses
		solutions = llmRes.Solutions
		codeSnippet = llmRes.CodeDemo
	}
	if len(rootCauses) == 0 && len(solutions) == 0 {
		rootCauses, solutions, codeSnippet, _ = s.extractTaskDeepKnowledge(qt, isoMeta, subTasks)
	}
	return BuildUnifiedCoArchitectReply(
		qt, isoMeta, subTasks, cleanMsg,
		agentRoleLabel, agentSlot, selectedModel,
		rootCauses, solutions, codeSnippet, repoNames, lessonRefs,
	)
}
```
Và trong `BuildUnifiedCoArchitectReply` (`internal/webui/unified_doc_format.go:303-310`), bỏ hoàn toàn việc ghép tiền tố máy móc `"- **Phương án khắc phục:** "` trước từng dòng `sol`!

#### (C) Khóa chặt `selectedChainKey` trong `static/app.js:52-66` để đang làm `cdc-01` không nhảy sang Plan của `TASK-13`
```javascript
// File: internal/webui/static/app.js:52-66
async function fetchDashboard(optionalTaskKey) {
  try {
    let url = "/api/dashboard";
    const effectiveKey = optionalTaskKey || selectedChainKey;
    if (effectiveKey) {
      url += `?task_key=${encodeURIComponent(effectiveKey)}`;
    }
    const res = await fetch(url);
    if (!res.ok) return;
    dashboardState = await res.json();
    if (dashboardState.active_task_key && !selectedChainKey) {
      selectedChainKey = dashboardState.active_task_key;
    }
    syncDefaultReportTabForStage();
    renderAll();
  } catch (err) {
    console.error("fetchDashboard error:", err);
  }
}
```
Và trong `server.go:670-674` (`handleDashboard`), chỉ cập nhật `s.selectedTaskKey` khi request từ UI chọn tường minh (`runTaskAction(..., "select")`), hoặc trả về đúng `task_key` được truyền trên query parameter mà không ghi đè biến toàn cục `s.selectedTaskKey` của các tab khác.

---

## 5. Thiết kế kỹ thuật & Code Demo cho `TASK-ORCH-07` (`REQ-ORCH-08` — Luôn mở đầu bằng đoạn Summary ngắn trả lời trực tiếp câu hỏi của User trước khi vào nội dung chính)

### 5.1. Nguyên nhân gốc rễ
- Trong [`internal/webui/task_llm_bridge.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_bridge.go#L18-L24), struct `ArchitectLLMAnalysis` đã có trường `DirectAnswer string` (`json:"direct_answer"`), nhưng tại `invokeArchitectLLM` (`dòng 228-230`) prompt JSON schema chỉ yêu cầu `{"root_causes": [...], "solutions": [...], "code_lang": "...", "code_demo": "..."}` mà quên mất `"direct_answer": "..."`.
- Tại [`internal/webui/unified_doc_format.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/unified_doc_format.go#L258-L304) (`BuildUnifiedCoArchitectReply`), hàm nhảy thẳng vào `## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ` mà không đặt 1 đoạn **Summary ngắn trả lời trực tiếp câu hỏi của User** ở ngay đầu phản hồi.

### 5.2. Giải pháp & Code Demo (`internal/webui/task_llm_bridge.go` & `internal/webui/unified_doc_format.go`)
```go
// 1. File: internal/webui/task_llm_bridge.go:226-231
userMsg := fmt.Sprintf(
	"Task ID: %s\nTask Title: %s\nTarget Repo: %s\nAction Mode: %s\nChỉ đạo/Câu hỏi mới của User: %q\n\nMã nguồn thực tế trên repo:\n%s\n\n"+
		"Trả về duy nhất 1 JSON object hợp lệ (không bọc markdown ngoài) theo mẫu: "+
		"{\"direct_answer\": \"Đoạn summary ngắn (2-3 câu) trả lời trực tiếp câu hỏi/yêu cầu của User\", \"root_causes\": [\"...\"], \"solutions\": [\"...\"], \"code_lang\": \"ts|go|js|sql\", \"code_demo\": \"...\"}",
	qt.ExternalID, qt.Title, filepath.Base(targetRepo), actionMode, userPrompt, repoSnippetBuilder.String(),
)

// 2. File: internal/webui/unified_doc_format.go:292-304
var sb strings.Builder
sb.WriteString(fmt.Sprintf("> 📌 **Task:** `%s` · **Stage:** `%s` · **Role:** `%s (%s)` · **Model:** `%s` · **Sub-Tasks:** `%d/%d DONE`\n\n",
	qt.ExternalID, qt.Stage, agentRoleLabel, agentSlot, selectedModel, doneCnt, len(subTasks)))

sb.WriteString("### 💬 Tóm Tắt & Trả Lời Trực Tiếp (Direct Summary)\n")
sb.WriteString(fmt.Sprintf("- **Câu hỏi của anh:** *\"%s\"*\n", truncateRunes(cleanMsg, 180)))
sb.WriteString(fmt.Sprintf("- **Trả lời trực tiếp:** %s\n\n", directSummary))

sb.WriteString("## 1. 🔍 Nguyên Nhân Gốc Rễ & Phân Tích Chuyên Sâu (`Root Cause & Analysis`)\n")
```

---

## 6. Thiết kế kỹ thuật & Code Demo cho `TASK-ORCH-08` (`REQ-ORCH-09` — 4 Chuẩn Formatter & Prompt Builder của `ai-orchestrator`)

### 6.1. Phản biện Thực tế (`S2.5 DG0 Reality Check`)
- Codebase vật lý của `ai-orchestrator` (`/Users/trainguyen/Documents/work/ai-orchestrator`) được viết bằng **Go** tại `internal/webui/telegram_message_format.go`, `internal/webui/unified_doc_format.go`, `internal/webui/task_inspector.go`, và `internal/webui/task_llm_bridge.go`.
- Dưới đây là thiết kế đầy đủ cả bản **TypeScript / Python** (theo yêu cầu mẫu của User) lẫn bản **Go** khớp 100% vào các file thực tế của `ai-orchestrator`:
  1. **Deduplicate & Relative Path (`normalizeRelativeFiles`):** Cắt bỏ prefix `<repoName>/` và line suffix `:line`, đưa vào `Set` khử trùng lặp.
  2. **Dynamic Language & DoD (`detectRepoProfile`):** Kiểm tra `package.json` (`typescript` + `npm run build && npm test`) vs `go.mod` (`go` + `go test -race ./...`).
  3. **Loại bỏ Meta-System Noise:** Xóa bỏ hoàn toàn các câu *"Đã quét trực tiếp ... file mã nguồn..."*, *"Mọi thay đổi ở bước S4 được cô lập trong Git Worktree theo Rule #12, #13, #19..."* và hướng dẫn bấm nút dài dòng để 100% tin nhắn nằm gọn trong 1 message (`< 4096` ký tự).
  4. **Không cắt `...` (`truncate`) trên trường định danh:** Giữ nguyên vẹn tên nhánh `branch`, relative file path, lệnh `DoD` và nội dung tóm tắt Sub-Task ở mục 3; đồng thời làm sạch Code Demo của `TASK-13` (tách riêng `PostgresTransHisRepository` dùng `PostgresDB` tại `dbHandler/postgres.ts` khỏi Mongoose `TransHisModel`, không trộn lẫn `pgPool`, `knexPg` và `dbConnection` Mongo).

### 6.2. Code Demo (Go — tích hợp trực tiếp vào `internal/webui/telegram_message_format.go` & `task_llm_bridge.go`)
```go
// DetectRepoStackAndDoD checks package.json vs go.mod in targetRepo
func DetectRepoStackAndDoD(repoPath string) (codeFence string, dodCmd string) {
	if repoPath != "" {
		if _, err := os.Stat(filepath.Join(repoPath, "package.json")); err == nil {
			return "typescript", "npm run build && npm test"
		}
		if _, err := os.Stat(filepath.Join(repoPath, "go.mod")); err == nil {
			return "go", "go test -race ./..."
		}
	}
	return "go", "go test -race ./..."
}

// NormalizeAndDeduplicateRelativeFiles strips "<repoName>/" prefix and ":line" suffix, returning unique relative paths.
func NormalizeAndDeduplicateRelativeFiles(rawFiles []string, repoNames []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, raw := range rawFiles {
		clean := strings.TrimSpace(strings.Split(raw, " ")[0])
		clean = strings.TrimSpace(strings.Split(clean, ":")[0])
		clean = strings.Trim(clean, "`'\"")
		for _, rn := range repoNames {
			if rn != "" && strings.HasPrefix(clean, rn+"/") {
				clean = strings.TrimPrefix(clean, rn+"/")
			}
		}
		if clean == "" || clean == "." || clean == "/" {
			continue
		}
		key := strings.ToLower(clean)
		if !seen[key] {
			seen[key] = true
			out = append(out, fmt.Sprintf("`%s`", clean))
		}
	}
	return out
}
```

---

## 7. Thiết kế tự chữa lành 5 phát hiện Audit S5 (`TASK-ORCH-09` — `S5 -> S3 -> S4 -> S5` Self-Healing Loop)

### 7.1. Phân tách Module Đơn Trách Nhiệm (`internal/webui/repo_tech_profile.go` & `internal/webui/task_revise_workflow.go`) và đưa `subtask_git_apply.go <= HEAD`
1. **`internal/webui/repo_tech_profile.go` (Module mới đơn trách nhiệm `< 180` dòng):**
   - Sở hữu duy nhất toàn bộ hàm chuẩn hóa/khử trùng lặp đường dẫn và nhận diện Tech Stack:
     - `normalizeOrchestratorDisplayPath(raw string) string`
     - `normalizeRelativeRepoFiles(rawFiles []string, repoName string) []string`
     - `normalizeAndDeduplicateRepoRelativePaths(rawPaths []string, repoNames []string) []string`
     - `detectRepoTechStackProfile(repoPath string, filePaths []string) repoTechStackProfile`
     - `detectTargetRepoLanguageAndDoD(repoPath string) (string, string)`
     - `detectCodeFenceLang(filePath string) string`
   - Xóa bỏ hoàn toàn các bản copy trùng lặp trong `task_llm_bridge.go` và `telegram_message_format.go`.
2. **`internal/webui/task_revise_workflow.go` (Module mới đơn trách nhiệm `< 170` dòng):**
   - Tách `executeArchitectReviseSolution` và `buildCleanRoomSpecSectionFromSubTasks` ra khỏi `task_llm_bridge.go`, giữ `task_llm_bridge.go` gọn nhẹ (`< 310` dòng) chỉ phụ trách xây dựng prompt & gọi `AntigravityRunner.Complete`.
3. **Đưa `internal/webui/subtask_git_apply.go` từ `344` dòng về `<= 312` dòng (`HEAD`):**
   - Chuyển `isIsolatedWorktreeRepoPath` sang `internal/webui/subtask_wave_engine.go` và tinh gọn `subtask_git_apply.go` xuống `<= 305` dòng (`< HEAD 312` dòng).

### 7.2. Khắc phục văn bản mâu thuẫn `Knex/PG Pool` trong `BuildUnifiedS3SolutionDoc`, normalize cột `Điểm chạm` và chuẩn hóa `TestDoD`
- Trong [`internal/webui/unified_doc_format.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/unified_doc_format.go#L228-L266):
  - Tại bảng Sub-Tasks (`dòng 232`), đổi `touchpoint := strings.TrimSpace(st.ContextFileLine)` thành `touchpoint := normalizeOrchestratorDisplayPath(st.ContextFileLine)` để cắt sạch prefix `centralized-export-service/`.
  - Tại mục `3.1. Kiến Trúc & Chiến Lược Chuyển Đổi` (`dòng 258-260`), thay thế dòng mẫu cũ nhắc tới `Knex/PG Pool` & chuyển đổi trực tiếp `trans-his.model.ts` bằng mô tả chính xác:
    `"- **Điểm chạm chính (Core Touchpoints):** dbHandler/postgres.ts (Quản lý kết nối PostgreSQL qua pg.Pool thuần cho luồng V2), models/trans-his/trans-his.model.ts (Giữ nguyên 100% truy vấn Mongoose V1 và định tuyến sang PostgresDB khi chạy V2)."`
- Trong [`internal/webui/task_inspector.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go#L1133-L1182) (`DecomposeMultiRepoCodebaseSubTasks`):
  - Gán `TestDoD: fmt.Sprintf("`%s`", defaultDoDCmd)` (`npm run build && npm test` cho `package.json`, `go test -race ./...` cho `go.mod`) đồng nhất trên tất cả Sub-Tasks, và khôi phục assertion nghiêm ngặt trong `task_llm_bridge_test.go`.

---

## 8. Thiết kế kỹ thuật & Code Demo cho `TASK-ORCH-10` (`REQ-ORCH-10` — Xóa sổ 100% Hardcode `Postgres/V1-V2` & Đấu nối Model `invokeArchitectLLM` Toàn trình vào Lập Plan Tổng, Khung Chat và Phản hồi Sửa Plan)

### 8.1. Chẩn đoán Nguyên nhân Gốc rễ (`VERIFIED` trên `.orchestrator/tasks/EX-002/` và `internal/webui/`)
Khi User tạo task `EX-002: centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn làm thêm qua clickhouse` (trong mô tả có nhắc giữ nguyên V1 Mongo, V2 Postgres và làm thêm V3 ClickHouse), hệ thống xảy ra 4 lỗi nghiêm trọng:
1. **Hardcode `Postgres/V1-V2` trong Engine (`task_llm_bridge.go:189-239, 358-404`, `telegram_message_format.go:135-161, 228-315`, `unified_doc_format.go:256-265`, `task_inspector.go:3292-3294`):**
   - Hàm `isDualVersionV1V2Directive` (`task_llm_bridge.go:189`) và `isPostgresOrDualSourceMigrationTask` (`telegram_message_format.go:135`) thấy trong mô tả của `EX-002` có chữ `"v1"` và `"v2"` liền trả về `true`, ép `buildSubTaskModuleTitleAndDesc`, `synthesizeDirectAnswerSummary`, `BuildUnifiedS3SolutionDoc`, `buildConcreteCodeDemoForTask`, `buildCleanKeyDiffPreview` xuất ra văn mẫu cứng `PostgresDB` (`dbHandler/postgres.ts`) của `TASK-13`.
2. **Đánh rơi `llmRes.DirectAnswer` và nhiễm `sysPrompt` + thiếu `chat_history` trong Khung Chat (`task_chat_knowledge.go:72-93`, `unified_doc_format.go:344`, `task_llm_bridge.go:279-286`):**
   - Trong `renderContextualDiscussReply` (`task_chat_knowledge.go:72-93`), dù đã gọi `s.invokeArchitectLLM(...)`, code chỉ lấy `RootCauses`, `Solutions`, `CodeDemo` mà **quên truyền `llmRes.DirectAnswer` vào `BuildUnifiedCoArchitectReply`** (`unified_doc_format.go:344`). `BuildUnifiedCoArchitectReply` tự gọi lại hàm tĩnh `synthesizeDirectAnswerSummary` -> ghi đè câu mở đầu bằng văn mẫu `PostgresDB`!
   - Tại `task_llm_bridge.go:279`, `sysPrompt` có câu nhắc cứng `TASK-13 ... PostgresDB (dbHandler/postgres.ts)`, và `userMsg` (`dòng 281-285`) không truyền `qt.Description` nguyên bản cũng không truyền lịch sử chat (`chat_history.json`). Khi User hỏi câu thứ 2 (*"tao hỏi là sao mà bị nhầm lẫn á."*), model không thấy lịch sử câu 1 và thấy `dbHandler/postgres.ts` trong prompt nên trả ra lại `PostgresDb`.
3. **Bộ sinh Plan tổng (`generateDeepTaskArtifacts` tại `task_inspector.go:3200-3249`) chưa gắn Model (`invokeArchitectLLM`):**
   - Khi tạo Task mới (`EX-002`), `generateDeepTaskArtifacts` không gọi `invokeArchitectLLM` để Model phân tích yêu cầu và sinh `Sub-Tasks`, `Architecture Plan`, `Risks`, `Code Demo`, mà chỉ chạy `inspectTargetRepoSymbols` bốc 2 file có sẵn trên đĩa (`trans-his.model.ts`, `dbHandler/postgres.ts`) rồi ghép vào template tĩnh.
4. **Khung Chat (`handleTaskPlanningChat`) và Phản hồi Telegram (`handleTelegramReviseSolutionCommand` / `applyTelegramSolutionFeedbackToPlan`) không tự động cập nhật lại Plan tổng bằng Model:**
   - Khi User nhắn *"sao mày lại làm tiếp postgress, cái task này là làm clickhouse mà"*, `classifyChatActionMode` (`task_chat.go:932`) xếp vào `"discuss"` (`AppliedToPlan = false`), chỉ trả lời chat mà không cập nhật `09_tasks_solution.md`, `08_tasks.md`, `subtasks.json`, `01_requirements.md`.
   - Trên Telegram, `applyTelegramSolutionFeedbackToPlan` (`telegram_bridge.go:2088`) chỉ nối 1 dòng bullet xuống cuối file `09_tasks_solution.md` mà không gọi `invokeArchitectLLM` để viết lại Plan tổng.

### 8.2. Giải pháp & Code Demo (`internal/webui/task_llm_bridge.go`, `task_chat_knowledge.go`, `unified_doc_format.go`, `task_inspector.go`, `telegram_message_format.go`, `telegram_bridge.go`)

```go
// 1. File: internal/webui/task_llm_bridge.go
// XÓA SỔ 100% isDualVersionV1V2Directive và mọi đoạn hardcode PostgresDB.
// Mở rộng ArchitectLLMAnalysis để Model sinh trọn bộ Plan tổng + Sub-Tasks + DirectAnswer + CodeDemo:
type ArchitectLLMSubTaskSpec struct {
	SubTaskID       string `json:"sub_task_id"`
	WaveIndex       int    `json:"wave_index"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	TargetFile      string `json:"target_file"`
	ContextFileLine string `json:"context_file_line"`
	TestDoD         string `json:"test_dod"`
}

type ArchitectLLMAnalysis struct {
	DirectAnswer     string                    `json:"direct_answer"`
	ShouldUpdatePlan bool                      `json:"should_update_plan"`
	Architecture     []string                  `json:"architecture"`
	Risks            []string                  `json:"risks"`
	RootCauses       []string                  `json:"root_causes"`
	Solutions        []string                  `json:"solutions"`
	SubTasks         []ArchitectLLMSubTaskSpec `json:"sub_tasks"`
	ChangedFiles     []string                  `json:"changed_files"`
	CodeDemo         string                    `json:"code_demo"`
	CodeLang         string                    `json:"code_lang"`
}

// Trong invokeArchitectLLM: truyền đủ qt.Description + lịch sử chat gần nhất + SubTasks hiện tại:
func (s *Server) invokeArchitectLLM(
	ctx context.Context,
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	userPrompt, selectedModel, actionMode string,
) (*ArchitectLLMAnalysis, error) {
	// ...
	recentChat := s.buildRecentChatTranscriptForPrompt(qt.ExternalID, 6)
	sysPrompt := "[System Note: Đã đọc GEMINI.md và lessons/INDEX.md — trả lời trực tiếp ngay bằng tiếng Việt, KHÔNG gọi tool read_file.]\n" +
		"Bạn là BRAIN · Architect (Slot A · /brain-delegate) trong hệ thống AI-Orchestrator.\n" +
		"QUY TẮC BẮT BUỘC:\n" +
		"1. `direct_answer`: Đoạn Summary ngắn (2-4 câu) trả lời TRỰC TIẾP và CHÍNH XÁC câu hỏi/chỉ đạo vừa nhắn của User.\n" +
		"2. `should_update_plan`: Đặt `true` nếu đang ở chế độ `plan`/`revise_solution`, HOẶC nếu User đang góp ý/đính chính hướng giải pháp, chỉ ra Plan hiện tại bị nhầm (ví dụ: yêu cầu làm ClickHouse nhưng Plan đang ghi Postgres), hoặc yêu cầu thêm/sửa kiến trúc.\n" +
		"3. Phân tích đúng 100% yêu cầu trong Task Title + Task Description + Chỉ đạo mới của User (tuyệt đối không râu ông nọ cắm cằm bà kia sang bài toán cũ). Nếu Task yêu cầu tích hợp DB/module mới chưa có file trên đĩa (như `dbHandler/clickhouse.ts`), phải liệt kê đúng file mới đó vào `changed_files`, `sub_tasks` và viết `code_demo` thực chất cho đúng yêu cầu đó.\n" +
		fmt.Sprintf("4. Ngôn ngữ Repo đích là `%s`, lệnh DoD chuẩn là `%s`.", fenceLang, dodCmd)

	userMsg := fmt.Sprintf(
		"Task ID: %s\nTask Title: %s\nTask Description: %s\nTarget Repo: %s\nAction Mode: %s\n"+
			"Lịch sử trao đổi gần nhất:\n%s\n"+
			"Sub-Tasks hiện tại:\n%s\n"+
			"Tin nhắn/Chỉ đạo mới nhất của User: %q\n\n"+
			"Mã nguồn thực tế tham khảo trên repo:\n%s\n\n"+
			"Trả về duy nhất 1 JSON object hợp lệ theo mẫu:\n"+
			"{\"direct_answer\": \"...\", \"should_update_plan\": true, \"architecture\": [\"...\"], \"risks\": [\"...\"], \"root_causes\": [\"...\"], \"solutions\": [\"...\"], \"changed_files\": [\"...\"], \"sub_tasks\": [{\"sub_task_id\": \"...\", \"wave_index\": 0, \"title\": \"...\", \"description\": \"...\", \"target_file\": \"...\", \"context_file_line\": \"...\", \"test_dod\": %q}], \"code_lang\": %q, \"code_demo\": \"...\"}",
		qt.ExternalID, qt.Title, qt.Description, filepath.Base(targetRepo), actionMode,
		recentChat, currentSubTasksSummary, userPrompt, repoSnippetBuilder.String(), dodCmd, fenceLang,
	)
	// Gọi AntigravityRunner.Complete(llmCtx, ...) và parse JSON trả về!
}
```

```go
// 2. File: internal/webui/task_chat_knowledge.go & internal/webui/task_chat.go
// Giữ nguyên llmRes.DirectAnswer của Model và tự động cập nhật Plan tổng (09_tasks_solution.md, 08_tasks.md, subtasks.json, 01_requirements.md)
// ngay khi Model hoặc bộ nhận diện phát hiện User đang sửa hướng / đính chính nhầm lẫn!
func (s *Server) renderContextualDiscussReply(...) string {
	// ...
	llmRes, _ := s.invokeArchitectLLM(context.Background(), qt, subTasks, cleanMsg, selectedModel, "discuss")
	if llmRes != nil && (llmRes.ShouldUpdatePlan || isPlanCorrectionOrSteeringMessage(cleanMsg, qt, subTasks)) {
		subTasks = s.syncArchitectLLMAnalysisToMasterPlan(context.Background(), &qt, isoMeta, subTasks, llmRes, cleanMsg, agentRoleLabel, selectedModel)
	}
	return BuildUnifiedCoArchitectReplyWithDirectAnswer(
		qt, isoMeta, subTasks, cleanMsg,
		agentRoleLabel, agentSlot, selectedModel,
		llmRes.DirectAnswer, llmRes.RootCauses, llmRes.Solutions, llmRes.CodeDemo, repoNames, lessonRefs,
	)
}
```

```go
// 3. File: internal/webui/task_inspector.go & internal/webui/telegram_bridge.go
// Trong generateDeepTaskArtifacts (khi tạo Task / lên Plan tổng) và handleTelegramReviseSolutionCommand (khi phản hồi từ Telegram):
// Luôn gọi s.invokeArchitectLLM(...) và s.syncArchitectLLMAnalysisToMasterPlan(...) để Plan tổng (09_tasks_solution.md, 08_tasks.md, subtasks.json, 01_requirements.md)
// và Thẻ Telegram H4b được sinh 100% từ kết quả suy luận của Model (không còn bất kỳ hàm hardcode Postgres nào).
```
