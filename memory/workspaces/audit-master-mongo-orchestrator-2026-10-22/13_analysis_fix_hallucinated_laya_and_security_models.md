---
tier: T2
suffix: fix_hallucinated_laya_and_security_models
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T14:08:45+07:00
mode: propose-only
---

# 13. Analysis — Kiểm Chứng Sự Thật Vật Lý (`Ground Truth`) Về `Laya Gate (Slot H)` & `Security (Slot I)`

## 1. Bằng chứng `VERIFIED` trên hệ thống thật
1. **`Laya Gate (Slot H)` là Local Model `convaiinnovations/laya` tại `:8200`, KHÔNG PHẢI `claude-sonnet-4-6 (antigravity)` (`VERIFIED`):**
   - Tại `/Users/trainguyen/.agent/skills/laya_router/api.py:15-18`:
     ```python
     print("⏳ [STARTUP] Đang dựng ma trận và nạp Laya Weights (800MB)... Xin chờ ~30s.")
     # Nạp model 1 lần duy nhất khi khởi động API
     router_agent = laya.load("convaiinnovations/laya", subfolder="multilingual")
     ```
   - Tại `/Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:42`:
     ```python
     DEFAULT_URL = "http://127.0.0.1:8200/v1/predict"
     ```
   - Như vậy, `Laya Gate` không hề gọi qua Antigravity hay Claude Sonnet, mà luôn gọi trực tiếp HTTP `127.0.0.1:8200/v1/predict` chạy model local `convaiinnovations/laya`.

2. **`codex-5.3 (antigravity)` hoàn toàn KHÔNG TỒN TẠI trên Antigravity lẫn trên máy của anh (`VERIFIED`):**
   - Kết quả probe thực tế trên máy:
     - `which gemini` -> `/Users/trainguyen/.nvm/versions/node/v22.18.0/bin/gemini`
     - `which claude` -> `/Users/trainguyen/.local/bin/claude`
     - `which codex` -> `codex not found`
     - `which opencode` -> `opencode not found`
   - Tại `/Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity.go:68-75` (`DiscoverAvailableModelRunners`):
     ```go
     Models: []string{
         "antigravity/gemini-4-argon",
         "antigravity/gemini-4-argon-thinking",
         "antigravity/gemini-2.5-pro",
         "antigravity/gemini-2.5-flash",
         "antigravity/claude-opus-4-6",
         "antigravity/claude-sonnet-4-6",
     },
     ```
   - Không hề có `codex-5.3 (antigravity)`. Việc hiển thị `Security (Slot I): codex-5.3 (antigravity)` là lỗi bịa đặt (hallucination) do nhắm mắt chép ví dụ từ prompt mà không kiểm chứng môi trường thật và không phản biện lại.

3. **Vị trí các dòng code bịa đặt cần dọn sạch (`VERIFIED`):**
   - `internal/webui/task_inspector.go:39-40, 689-690, 726-732`
   - `internal/webui/static/index.html:921, 926-944`
   - `internal/webui/static/app.js:1831-1833, 3040-3041, 3077-3078`
   - `internal/webui/server_test.go:1667-1672`
