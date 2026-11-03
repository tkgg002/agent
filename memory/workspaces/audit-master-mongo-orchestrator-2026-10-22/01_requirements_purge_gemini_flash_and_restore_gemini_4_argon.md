---
task_id: FIX-GEMINI-FLASH-DOWNGRADE
title: Gỡ bỏ triệt để gemini-2.5-flash và khôi phục gemini-4-argon / gemini 4.0 trên toàn bộ ai-orchestrator
tier: T2
workspace: audit-master-mongo-orchestrator-2026-10-22
target_repo: /Users/trainguyen/Documents/work/ai-orchestrator
model: gemini-4-argon (antigravity)
timestamp: 2026-10-27T17:03:00+07:00
---

# 01_requirements: Gỡ Bỏ Triệt Để `gemini-2.5-flash` và Cài Đặt `gemini-4-argon` / `gemini 4.0`

## 1. Bối cảnh & Vấn đề (Trigger)
User chat trên Co-Architect Studio với cấu hình `gemini-4-argon (antigravity)` nhưng bị hệ thống trả về lỗi cạn kiệt quota:
```
Lỗi Model Runner (gemini-2.5-flash): TerminalQuotaError: You have exhausted your daily quota on this model.
at classifyGoogleError (file:///Users/trainguyen/.nvm/versions/node/v22.18.0/lib/node_modules/@google/gemini-cli/bundle/chunk-SBG6CUNK.js:304428:16)
at retryWithBackoff (file:///Use...
(Model AI CLI bị lỗi hoặc đã chạm ngưỡng hạn mức API. Vui lòng kiểm tra API Key hoặc runner trên máy.)
```
User chỉ đạo trực tiếp: *"sao mày gắn con gemini-2.5-flash vào, má nó chứ. đã nói gài gemini 4.0"*

## 2. Nguyên nhân gốc rễ (Root Cause)
1. Tại `internal/webui/task_llm_bridge.go:235-238`, code tự ý gán đè:
   ```go
   modelName := strings.TrimSpace(selectedModel)
   if modelName == "" || strings.Contains(strings.ToLower(modelName), "argon") {
       modelName = "gemini-2.5-flash"
   }
   ```
   Điều này khiến bất kỳ lựa chọn `argon` nào cũng bị ép thành `gemini-2.5-flash`.
2. Tại `internal/runner/antigravity.go:193-195`, hàm `buildGeminiCLIArgs` gán đè:
   ```go
   chosenModel := cleanModel
   if chosenModel == "" || (!passModel && (strings.Contains(strings.ToLower(chosenModel), "argon") || strings.Contains(strings.ToLower(chosenModel), "antigravity"))) {
       chosenModel = "gemini-2.5-flash"
   }
   ```
   Và append `-m gemini-2.5-flash` vào lệnh `gemini` CLI.
3. Tại `internal/webui/task_inspector.go:37, 655`, cấu hình `CoderStdModel` mặc định bị đặt thành `"gemini-2.5-flash (antigravity)"`.
4. Tại `internal/webui/static/index.html:909` và `internal/webui/static/app.js:3173, 3210`, option và fallback bị gán `"gemini-2.5-flash (antigravity)"`.
5. Tại `internal/runner/antigravity.go:72, 87`, danh sách model runner có chứa `"antigravity/gemini-2.5-flash"` và `"gemini-2.5-flash"`.

## 3. Yêu cầu & Mục tiêu (DoD)
- [ ] Xóa bỏ hoàn toàn chuỗi `"gemini-2.5-flash"` khỏi `task_llm_bridge.go`. `modelName` nếu rỗng thì mặc định là `"gemini-4-argon (antigravity)"`, tuyệt đối không được ép về flash.
- [ ] Xóa bỏ đoạn gán `chosenModel = "gemini-2.5-flash"` trong `antigravity.go`. Xóa bỏ `"gemini-2.5-flash"` khỏi `DiscoverAvailableModelRunners`.
- [ ] Trong `task_inspector.go`, đổi `CoderStdModel` mặc định thành `"gemini-4-argon (antigravity)"`.
- [ ] Trong `index.html` và `app.js`, đổi toàn bộ option và fallback của `coder_std_model` thành `"gemini-4-argon (antigravity)"`.
- [ ] Toàn bộ unit tests (`go test ./internal/runner/...` và `go test ./internal/webui/...`) đều PASS xanh 100%.
- [ ] Tuân thủ Rule #13: Trình bày giải pháp duy nhất, chạy DG1/DG2 và DỪNG CHỜ APPROVE trước khi sửa code.
