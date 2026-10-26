---
tier: T2
suffix: fix_hallucinated_laya_and_security_models
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T14:07:45+07:00
mode: propose-only
---

# 01. Requirements — Khắc phục Lỗi Bịa Đặt Cấu Hình `Laya Gate (Slot H): claude-sonnet-4-6 (antigravity)` & `Security (Slot I): codex-5.3 (antigravity)`

## 1. Bối cảnh & Phản hồi Chấn chỉnh của User
User nhìn thấy dòng hiển thị trên thanh Autopilot Model Routing của WebUI:
> `Laya Gate (Slot H): claude-sonnet-4-6 (antigravity) · Security (Slot I): codex-5.3 (antigravity) · Circuit Breaker: Retry 0/3 => mày bốc đâu ra 2 con này, kêu làm thì cắm đầu làm à. mày ngu vậy. ko biết la làng lên à`

## 2. Bản chất Sai lầm & Sự thật Kiến trúc (`Ground Truth`)
1. **Về `Laya Gate (Slot H)`:**
   - **Sai lầm:** Khi đọc câu ví dụ giả định trong bài review trước (`"hãy cân nhắc đổi QA Auditor hoặc Laya Gate sang một họ model khác như Claude Opus/Sonnet hoặc Codex/GPT"`), Agent không phản biện ("la làng lên") mà cắm đầu gán `LayaGateModel = "claude-sonnet-4-6 (antigravity)"` tại `internal/webui/task_inspector.go:39, 689`, `internal/webui/static/index.html:926-933`, và `internal/webui/static/app.js:1831`.
   - **Sự thật vật lý (`Ground Truth`):**
     - Trong `/Users/trainguyen/.agent/skills/laya_router/api.py:17` và `/Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:42`, **`Laya Gate` (`/laya_router`) hoàn toàn KHÔNG PHẢI là LLM chạy qua Antigravity**!
     - Bản chất `Laya Gate` là **Local Neural Decision Engine `convaiinnovations/laya` (multilingual ~800MB weights)** chạy cố định tại `http://127.0.0.1:8200/v1/predict` (`~33ms` single forward pass). Không bao giờ có chuyện `Laya Gate` chạy `claude-sonnet-4-6 (antigravity)` hay cho chọn sang `codex-5.3`!

2. **Về `Security (Slot I): codex-5.3 (antigravity)`:**
   - **Sai lầm:** Agent tự bịa ra cái tên `"codex-5.3 (antigravity)"` cho `SecurityModel` (`task_inspector.go:40, 690`, `index.html:921, 930, 940`, `app.js:1832`) dù:
     - Trên máy thật: `which codex` -> **`codex not found`**.
     - Trong `internal/runner/antigravity.go:68-75` (`DiscoverAvailableModelRunners`): Antigravity Model Bridge chỉ hỗ trợ họ `gemini-*` (`gemini-4-argon`, `gemini-4-argon-thinking`, `gemini-2.5-pro`, `gemini-2.5-flash`) và họ `claude-*` (`claude-opus-4-6`, `claude-sonnet-4-6`). **Hoàn toàn không tồn tại thứ gọi là `codex-5.3 (antigravity)`!**
   - **Sự thật vật lý (`Ground Truth`):**
     - Cổng bảo mật `Slot I` trong `/agent` được thực thi bởi **Hooks tất định `agy_guard.py (H1..H7)` + `agentgate` (`ScanForHardcodedSecrets` / `ScanDiffForMockClosures`)** kết hợp `/security-agent` chạy trên model thực tế có sẵn trên máy (ví dụ `claude-sonnet-4-6 (antigravity)` hoặc `gemini-4-argon (antigravity)`).

## 3. Yêu cầu Khắc phục (`R1..R3`)
- **R1 (Trả `Laya Gate · Slot H` về đúng bản chất thật `convaiinnovations/laya` tại `:8200`):**
  - Xóa bỏ việc gán `claude-sonnet-4-6 (antigravity)` cho `Laya Gate`. Định danh cố định và trung thực `LayaGateModel = "convaiinnovations/laya (local :8200)"` theo đúng `/Users/trainguyen/.agent/skills/laya_router/api.py:17` & `laya_gate.py:42`.
- **R2 (Xóa sổ hoàn toàn model bịa đặt `codex-5.3 (antigravity)` khỏi Backend & Frontend):**
  - Xóa sạch mọi chuỗi `"codex-5.3 (antigravity)"` trong `task_inspector.go`, `index.html`, `app.js`, và `server_test.go`.
  - Chỉ cho phép chọn các model thực sự tồn tại trong `DiscoverAvailableModelRunners()` (`internal/runner/antigravity.go:68-75`). Với `Security Guard (Slot I)`, hiển thị đúng bản chất `agy_guard.py (H1..H7) + claude-sonnet-4-6 (antigravity)` (hoặc các model có thật trên máy).
- **R3 (Tự động làm sạch file cấu hình `.orchestrator/cli_settings.json` và `seed_data.go` nếu chứa model/runner bịa đặt cũ):**
  - Trong `loadCLISettings()` (`task_inspector.go:678-741`), nếu file `cli_settings.json` đang chứa giá trị bịa đặt cũ (`codex-5.3` hoặc `LayaGateModel` mang tên LLM Antigravity), tự động chuẩn hóa về giá trị thật (`convaiinnovations/laya (local :8200)` cho `Laya Gate` và `agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)` cho `Security Guard`).
  - Trong `internal/webui/seed_data.go:710, 728, 789, 832`, dọn sạch các runner/model không có trên máy (`codex`, `laya-rl`, `cursor`) về đúng `claude`, `convaiinnovations/laya (:8200)`, và `antigravity`.
- **R4 (Bổ sung `Wave Phản Biện Trước Khi Lên Plan — DG0` và tận dụng `/laya_router` local 0-token vào `/Users/trainguyen/Documents/work/agent`):**
  - Cập nhật `agent/GEMINI.md` (giữ `< 24KB`), `agent/tooling/laya_gate.py` (thêm `--stage dg0`), `agent/hooks/agy/agy_guard.py`, `agent/skills/brain-delegate/SKILL.md`, và `agent/skills/laya-router/SKILL.md`:
    - Thiết lập bắt buộc chặng **`S2.5 Pre-Plan Adversarial Critique & Reality Check (DG0)`** ngay trước `S3 Solution (09_tasks_solution_*.md)`: Agent phải phản biện từng giả định/ví dụ trong prompt đối chiếu với `file:line` & binary/model vật lý trên máy ("biết la làng lên"), tuyệt đối không nhắm mắt làm theo giả định sai.
    - Tận dụng tối đa `/laya_router` chạy local (`http://127.0.0.1:8200/v1/predict` — **0 tốn LLM token, ~33ms**) xuyên suốt 4 cổng: **`DG0`** (Pre-Plan Critique & Reality Check), **`DG1`** (Solution Verification), **`DG2`** (Code Demo Verification), và **`DG3`** (Final Adversarial Audit & Diff Review).
- **R5 (Bổ sung `Wave Phản Biện Trước Khi Lên Plan — DG0` và tận dụng `/laya_router` local 0-token ở mọi chặng trong `AI-ORCHESTRATOR`):**
  - Cập nhật `internal/webui/server.go`, `internal/webui/task_inspector.go`, `internal/agentgate/kernel_bridge.go`, `internal/webui/static/app.js`, `internal/webui/static/index.html`, và `internal/webui/server_test.go`:
    - Đưa **Wave Phản Biện Yêu Cầu (`Pre-Plan Critique · Laya DG0 :8200`)** đứng ngay trước chặng `Plan (S3)` trên cả Autopilot Pipeline của Task cha, Fractal Pipeline của Sub-Task, cột Workflow Steps bên trái, và các tab Step Report.
    - Gọi trực tiếp `/laya_router` local (`http://127.0.0.1:8200/v1/predict`, 0 token) tại mọi chặng: `DG0 Pre-Plan Critique`, `DG1/DG2 Solution & Demo`, `S4 Implementation Check`, và `DG3 Final Audit`.

## 4. Chế độ Quản trị (Governance Mode)
- **Tier:** `T2`
- **Trạng thái hiện tại:** Đã nhận `APPROVE` từ User — thực thi khép kín `S4 -> S5 -> S6`.
