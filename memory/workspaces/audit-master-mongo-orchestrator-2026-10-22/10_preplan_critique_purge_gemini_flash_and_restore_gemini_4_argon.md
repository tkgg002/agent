---
task_id: FIX-GEMINI-FLASH-DOWNGRADE
stage: S2.5 Pre-Plan Critique (DG0)
target_repo: /Users/trainguyen/Documents/work/ai-orchestrator
timestamp: 2026-10-27T17:03:30+07:00
---

# 10_preplan_critique: Phản Biện Trước Khi Lên Giải Pháp (DG0)

## 1. Reality Check đối chiếu với mã nguồn thật trên đĩa
- **C-REQ-01:** User yêu cầu gài `gemini 4.0` (`gemini-4-argon`) và xóa bỏ `gemini-2.5-flash` vì bị lỗi `TerminalQuotaError` hết hạn mức ngày.
  - *Evidence:* Lỗi `TerminalQuotaError: You have exhausted your daily quota on this model` xuất hiện trong `tasks/TASK-13/chat_history.json:490` do CLI chạy `gemini -m gemini-2.5-flash`.
- **C-REALITY-02:** `internal/webui/task_llm_bridge.go:235-238` đang hạ cấp model xuống `gemini-2.5-flash`.
  - *Evidence:* `modelName := strings.TrimSpace(selectedModel); if modelName == "" || strings.Contains(strings.ToLower(modelName), "argon") { modelName = "gemini-2.5-flash" }`. Khớp 100% với dòng 235-238 trên đĩa.
- **C-REALITY-03:** `internal/runner/antigravity.go:193-195` đang ép `chosenModel = "gemini-2.5-flash"`.
  - *Evidence:* Khớp 100% dòng 193-195 trên đĩa.
- **C-CONTRA-04:** Liệu xóa `gemini-2.5-flash` có làm hỏng các test case hiện tại không?
  - *Phản biện:* Kiểm tra các test trong `internal/runner` và `internal/webui`. `server_test.go:589` kỳ vọng `PlannerModel` và `CoderHighModel` là `"gemini-4-argon (antigravity)"`. Cập nhật `CoderStdModel` sang `"gemini-4-argon (antigravity)"` làm tăng tính đồng bộ và sạch sẽ.
