---
task_id: FIX-GEMINI-FLASH-DOWNGRADE
title: Danh sách Task xử lý gỡ bỏ gemini-2.5-flash và cài đặt gemini-4-argon / gemini 4.0
tier: T2
workspace: audit-master-mongo-orchestrator-2026-10-22
target_repo: /Users/trainguyen/Documents/work/ai-orchestrator
model: gemini-4-argon (antigravity)
timestamp: 2026-10-27T17:03:15+07:00
---

# 08_tasks: Xóa Sạch `gemini-2.5-flash` và Cài Đặt `gemini-4-argon` / `gemini 4.0`

- [ ] **TASK-MDL-PURGE-01 — Gỡ bỏ ép model `gemini-2.5-flash` trong `task_llm_bridge.go`**
  - **Mô tả + Context file:line:** Tại `internal/webui/task_llm_bridge.go:235-238`, xóa bỏ đoạn ép `strings.Contains(strings.ToLower(modelName), "argon")` thành `"gemini-2.5-flash"`. Nếu `modelName == ""` thì gán mặc định `"gemini-4-argon (antigravity)"`.
  - **Definition of Done:** `modelName` giữ nguyên mô hình mà User chọn, không bao giờ tự ý chuyển về flash.

- [ ] **TASK-MDL-PURGE-02 — Gỡ bỏ ép `chosenModel = "gemini-2.5-flash"` và xóa flash khỏi Discovery trong `antigravity.go`**
  - **Mô tả + Context file:line:** Tại `internal/runner/antigravity.go:193-195`, xóa bỏ đoạn gán `chosenModel = "gemini-2.5-flash"`. Tại `antigravity.go:72, 87`, xóa `"antigravity/gemini-2.5-flash"` và `"gemini-2.5-flash"` khỏi danh sách mô hình trả về. Trong `buildGeminiCLIArgs`, chỉ truyền `-m` khi `passModel == true` và `chosenModel != ""`.
  - **Definition of Done:** Khi gọi AntigravityRunner với `gemini-4-argon (antigravity)`, không append `-m gemini-2.5-flash`.

- [ ] **TASK-MDL-PURGE-03 — Cập nhật `CoderStdModel` mặc định và WebUI Form Options**
  - **Mô tả + Context file:line:**
    - Tại `internal/webui/task_inspector.go:37, 655`: đổi `CoderStdModel` mặc định thành `"gemini-4-argon (antigravity)"`.
    - Tại `internal/webui/static/index.html:909`: xóa option `gemini-2.5-flash (antigravity)`.
    - Tại `internal/webui/static/app.js:3173, 3210`: đổi fallback từ `"gemini-2.5-flash (antigravity)"` thành `"gemini-4-argon (antigravity)"`.
  - **Definition of Done:** WebUI và backend không còn bất kỳ chỗ nào trỏ về `gemini-2.5-flash`.

- [ ] **TASK-MDL-PURGE-04 — Test & Verify Runtime**
  - **Mô tả + Context:** Chạy `go test -v ./internal/runner/...` và `go test -v ./internal/webui/...`, build lại binary `./bin/orchestrator` và restart Server `:8090`.
  - **Definition of Done:** Unit test PASS, Server `:8090` hoạt động bình thường với model `gemini-4-argon (antigravity)`.
