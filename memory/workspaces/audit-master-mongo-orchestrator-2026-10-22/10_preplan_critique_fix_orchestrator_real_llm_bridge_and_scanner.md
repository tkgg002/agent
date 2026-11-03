# 10. Pre-Plan Adversarial Critique (DG0) — `fix_orchestrator_real_llm_bridge_and_scanner`

## 1. Phản biện Thực tế Vật lý trên Máy (`Reality Check`)
- **Kiểm tra CLI `gemini` vs `-m gemini-4-argon`:**
  - Trên máy hiện có `/Users/trainguyen/.nvm/versions/node/v22.18.0/bin/gemini` (`v0.38.2`).
  - Khi gọi `gemini -m gemini-4-argon -p ...`, Google API trả về `429 Quota exceeded ... limit: 0, model: gemini-4-argon` vì `gemini-4-argon` là tên model nội bộ trên Antigravity IDE (`gemini-4-argon (antigravity)`).
  - Ngược lại, khi gọi `/Users/trainguyen/.nvm/versions/node/v22.18.0/bin/gemini --skip-trust -p <prompt>` với `GEMINI_CLI_TRUST_WORKSPACE=true` (không truyền `-m gemini-4-argon`), CLI chạy thành công (`exit=0`) và sinh câu trả lời tiếng Việt chuẩn xác.
  - Ngoài ra, nếu thiếu `--skip-trust` / `GEMINI_CLI_TRUST_WORKSPACE=true` ở thư mục chưa trust, `gemini -p` thoát với `exit code 55`. Do đó `internal/runner/antigravity.go` bắt buộc phải truyền `--skip-trust` và `GEMINI_CLI_TRUST_WORKSPACE=true`.

## 2. Claims Kiểm định Cổng `DG0` (`/laya_router`)

### Claims Chính (`C-REQ`, `C-REALITY`, `C-CONTRA`)
- [C-REQ] Trong `internal/webui/task_chat.go`, nhánh `"discuss"` gọi `s.buildContextualDiscussReply(qt, msg, roleBadge, slotLabel, selectedModel)` và `internal/webui/task_chat_knowledge.go` chỉ dùng `fmt.Sprintf` ghép các mảng chuỗi tĩnh `rootCauses`, `solutions`, `codeBlock`, `subTaskLines` mà chưa gọi `AntigravityRunner.Complete`.
  REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat_knowledge.go:16-118
- [C-REALITY] Trong `internal/runner/antigravity.go`, hàm `resolveCommand` xử lý `strings.Contains(baseName, "gemini")` hiện chỉ trả về `binPath, []string{"-p", prompt}` mà chưa truyền cờ `--skip-trust`.
  REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity.go:151-155
- [C-CONTRA] Trong `internal/webui/task_inspector.go`, hàm `inspectTargetRepoSymbols` mới chỉ skip `.git`, `.orchestrator`, `vendor`, `node_modules`, `scratch`, `backups_*` mà chưa skip thư mục build `dist` hay `build`, dẫn tới quét lọt file `dist/models/trans-his/trans-his.model.js`.
  REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:2955-2965

### Calibration Controls (`control_true` & `control_false`)
- [CTRL-TRUE-1] control_true: Hàm `classifyChatActionMode` trong `internal/webui/task_chat.go` trả về `"discuss"` ở cuối hàm khi không khớp các nhánh từ khóa cứng phía trên.
  REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go:1019-1041
- [CTRL-FALSE-1] control_false: Hàm `renderContextualDiscussReply` trong `internal/webui/task_chat_knowledge.go` hiện đang khởi tạo `runner.NewAntigravityRunner` và gọi `.Complete(ctx, req)` trực tiếp ở dòng 50.
  REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat_knowledge.go:26-118
- [CTRL-FALSE-2] control_false: Hàm `inspectTargetRepoSymbols` trong `internal/webui/task_inspector.go` hiện đã có điều kiện `name == "dist" || name == "build"` để chặn toàn bộ thư mục build.
  REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:2955-2965
