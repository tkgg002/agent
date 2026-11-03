---
task_id: FIX-GEMINI-FLASH-DOWNGRADE
title: Phương án kỹ thuật duy nhất — Gỡ bỏ sạch sẽ gemini-2.5-flash và cài đặt gemini-4-argon / gemini 4.0
tier: T2
workspace: audit-master-mongo-orchestrator-2026-10-22
target_repo: /Users/trainguyen/Documents/work/ai-orchestrator
model: gemini-4-argon (antigravity)
timestamp: 2026-10-27T17:03:45+07:00
---

# 09_tasks_solution: Giải Pháp Kỹ Thuật Duy Nhất & Code Demo

## 1. Tóm tắt giải pháp (Executive Summary)
Gỡ bỏ hoàn toàn việc hạ cấp ngầm mô hình sang `gemini-2.5-flash` tại 5 vị trí trong `ai-orchestrator`:
1. `internal/webui/task_llm_bridge.go`: Giữ nguyên `selectedModel`, nếu rỗng thì dùng mặc định `gemini-4-argon (antigravity)`.
2. `internal/runner/antigravity.go`: Xóa bỏ gán `chosenModel = "gemini-2.5-flash"`, xóa `gemini-2.5-flash` khỏi danh sách Discover.
3. `internal/webui/task_inspector.go`: Đổi `CoderStdModel` mặc định sang `gemini-4-argon (antigravity)`.
4. `internal/webui/static/index.html`: Xóa option `gemini-2.5-flash (antigravity)`.
5. `internal/webui/static/app.js`: Đổi fallback sang `gemini-4-argon (antigravity)`.

## 2. Chi tiết thay đổi mã nguồn (Code Diff Demo)

### 2.1. `internal/webui/task_llm_bridge.go`
```go
// TRƯỚC (Lỗi hạ cấp model):
modelName := strings.TrimSpace(selectedModel)
if modelName == "" || strings.Contains(strings.ToLower(modelName), "argon") {
    modelName = "gemini-2.5-flash"
}

// SAU (Giữ đúng chỉ đạo của User):
modelName := strings.TrimSpace(selectedModel)
if modelName == "" {
    modelName = "gemini-4-argon (antigravity)"
}
```

### 2.2. `internal/runner/antigravity.go`
```go
// TRƯỚC:
func buildGeminiCLIArgs(prompt, systemPrompt, cleanModel string, passModel bool) []string {
    fullPrompt := prompt
    if strings.TrimSpace(systemPrompt) != "" {
        fullPrompt = systemPrompt + "\n\n" + prompt
    }
    args := []string{"--skip-trust", "-e", "none", "-p", fullPrompt}
    chosenModel := cleanModel
    if chosenModel == "" || (!passModel && (strings.Contains(strings.ToLower(chosenModel), "argon") || strings.Contains(strings.ToLower(chosenModel), "antigravity"))) {
        chosenModel = "gemini-2.5-flash"
    }
    if chosenModel != "" {
        args = append(args, "-m", chosenModel)
    }
    return args
}

// SAU:
func buildGeminiCLIArgs(prompt, systemPrompt, cleanModel string, passModel bool) []string {
    fullPrompt := prompt
    if strings.TrimSpace(systemPrompt) != "" {
        fullPrompt = systemPrompt + "\n\n" + prompt
    }
    args := []string{"--skip-trust", "-e", "none", "-p", fullPrompt}
    chosenModel := cleanModel
    if chosenModel == "" {
        chosenModel = "gemini-4-argon"
    }
    if passModel && chosenModel != "" {
        args = append(args, "-m", chosenModel)
    }
    return args
}
```
Và trong `DiscoverAvailableModelRunners()`:
Xóa `"antigravity/gemini-2.5-flash"` và `"gemini-2.5-flash"`.

### 2.3. `internal/webui/task_inspector.go`
```go
// Dòng 37:
CoderStdModel string `json:"coder_std_model"` // Slot D/F: gemini-4-argon (antigravity)

// Dòng 655:
CoderStdModel: "gemini-4-argon (antigravity)",
```

### 2.4. `internal/webui/static/index.html` & `app.js`
- Xóa option `gemini-2.5-flash (antigravity)`.
- Fallback `coder_std_model` chuyển thành `"gemini-4-argon (antigravity)"`.

## 3. Danh sách Claims cho Decision Gate (DG1)
- `S-RC-01`: Lỗi `TerminalQuotaError` phát sinh do `task_llm_bridge.go:237` và `antigravity.go:194` lén lút hạ cấp `modelName` xuống `gemini-2.5-flash`. (REF: `internal/webui/task_llm_bridge.go:235-238`)
- `S-FIX-02`: Giải pháp xóa sạch toàn bộ chuỗi `gemini-2.5-flash` và khôi phục `gemini-4-argon` đảm bảo hệ thống tuân thủ 100% chỉ đạo của User. (REF: `internal/webui/task_llm_bridge.go:235-238`, `internal/runner/antigravity.go:192-200`)
- `S-IMPACT-03`: Thay đổi này giúp tránh hoàn toàn lỗi 429 quota hạn mức của flash model trên Free Tier và kích hoạt đúng model Gemini 4. (REF: `internal/runner/antigravity.go:68-88`)
- `S-PATTERN-04`: Tuân thủ triệt để Rule #3 (Agnostic Core) và Rule #12 (Tư duy Core Systems), không tự ý bypass model mà User yêu cầu. (REF: `internal/webui/task_inspector.go:651-662`)
