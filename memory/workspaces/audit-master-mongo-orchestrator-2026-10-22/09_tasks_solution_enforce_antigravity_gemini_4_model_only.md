# Giải Pháp Kỹ Thuật (09_tasks_solution_enforce_antigravity_gemini_4_model_only)

## 1. Phương Án Tối Ưu Duy Nhất
- **Nguyên tắc:** Xóa sổ 100% việc hardcode và fallback sang `gemini-2.5-flash`. Toàn bộ hệ thống AI Orchestrator chuyển sang chuẩn mô hình 4.0 nội bộ Antigravity: `gemini-4-argon (antigravity)`.
- **Chi tiết các điểm chạm:**
  1. `internal/runner/antigravity.go`:
     - Xóa bỏ `gemini-2.5-flash` khỏi danh sách `DiscoverAvailableModelRunners`.
     - Trong `buildGeminiCLIArgs`, bỏ hoàn toàn việc tự gán `chosenModel = "gemini-2.5-flash"`. Chỉ truyền cờ `-m` khi `passModel == true` và model được người dùng chỉ định rõ ràng.
  2. `internal/webui/task_llm_bridge.go`:
     - Trong `invokeArchitectLLM`, xóa bỏ `if modelName == "" || strings.Contains(..., "argon") { modelName = "gemini-2.5-flash" }`.
     - Đặt mặc định là `"gemini-4-argon (antigravity)"`.
  3. `internal/webui/task_inspector.go`:
     - Đồng bộ `CoderStdModel` mặc định thành `"gemini-4-argon (antigravity)"`.
  4. `internal/webui/static/index.html` & `internal/webui/static/app.js`:
     - Thay thế option và fallback `gemini-2.5-flash (antigravity)` thành `gemini-4-argon (antigravity)`.

## 2. Code Demo Cốt Lõi

### (a) `internal/runner/antigravity.go`
```go
func buildGeminiCLIArgs(prompt, systemPrompt, cleanModel string, passModel bool) []string {
	fullPrompt := prompt
	if strings.TrimSpace(systemPrompt) != "" {
		fullPrompt = systemPrompt + "\n\n" + prompt
	}
	args := []string{"--skip-trust", "-e", "none", "-p", fullPrompt}
	if passModel && cleanModel != "" {
		args = append(args, "-m", cleanModel)
	}
	return args
}
```

### (b) `internal/webui/task_llm_bridge.go`
```go
	modelName := strings.TrimSpace(selectedModel)
	if modelName == "" {
		modelName = "gemini-4-argon (antigravity)"
	}
	agRunner := runner.NewAntigravityRunner(modelName)
```
