# Danh Sách Công Việc (08_tasks_enforce_antigravity_gemini_4_model_only)

## TASK-AG4-01: Gỡ bỏ toàn bộ fallback gemini-2.5-flash trong internal/runner/antigravity.go
- **Mô tả:** Trong `buildGeminiCLIArgs`, xóa bỏ logic gán `chosenModel = "gemini-2.5-flash"` khi model là argon hoặc antigravity. Giữ nguyên mô hình được truyền vào (`gemini-4-argon`).
- **File:line:** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity.go:186-200`
- **DoD:** `go test -v ./internal/runner/...` PASS.

## TASK-AG4-02: Đồng bộ Model gemini-4-argon trong task_llm_bridge.go và task_inspector.go
- **Mô tả:** Trong `invokeArchitectLLM` (`task_llm_bridge.go:210`), xóa bỏ logic `modelName = "gemini-2.5-flash"`; giữ nguyên `gemini-4-argon (antigravity)`. Đồng bộ `CoderStdModel` trong `task_inspector.go` sang `gemini-4-argon (antigravity)`.
- **File:line:** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_bridge.go:210-220`
- **DoD:** `go test -v ./internal/webui/...` PASS.
