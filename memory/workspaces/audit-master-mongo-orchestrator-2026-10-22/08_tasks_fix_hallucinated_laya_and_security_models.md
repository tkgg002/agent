---
tier: T2
suffix: fix_hallucinated_laya_and_security_models
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T14:08:30+07:00
mode: propose-only
---

# 08. Tasks — Dọn Sạch Cấu Hình Model Bịa Đặt ở `Laya Gate (Slot H)` & `Security (Slot I)`

## Danh sách Task Thực thi (Sau khi có `APPROVE`)

- [x] **TASK-MDL-01: Trả `Laya Gate (Slot H)` về đúng bản chất Local Neural Decision Engine `convaiinnovations/laya (local :8200)` và xóa sổ `codex-5.3 (antigravity)` ở Backend**
  - **Context (`file:line`):**
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:39-45` (`CLISettingsDTO`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:637-741` (`extractModelFamilyPrefix`, `EvaluateCrossModelDiversity`, `loadCLISettings`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:2989` (comment S5B)
  - **Mô tả:**
    - Đổi mặc định `LayaGateModel` từ `"claude-sonnet-4-6 (antigravity)"` sang đúng bản chất vật lý **`"convaiinnovations/laya (local :8200)"`** (theo `/Users/trainguyen/.agent/skills/laya_router/api.py:17` & `laya_gate.py:42`).
    - Đổi mặc định `SecurityModel` từ `"codex-5.3 (antigravity)"` (model không tồn tại trên Antigravity lẫn trên máy) sang **`"agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)"`** (kết hợp Hook tất định `agy_guard.py` + model có thật trên Antigravity/Claude CLI).
    - Trong `loadCLISettings()`, nếu file `.orchestrator/cli_settings.json` cũ đang lưu `"claude-sonnet-4-6 (antigravity)"` cho `LayaGateModel` hoặc `"codex-5.3 (antigravity)"` ở bất kỳ slot nào, tự động migrate về giá trị thật.
    - Cập nhật `extractModelFamilyPrefix` nhận diện `"convaiinnovations/laya"` / `"laya"` -> `"laya-8200"` và `"agy_guard.py"` -> `"agy-guard+claude"`.
  - **Logs/Test:** `go test -v -race -run TestTaskPipe_MultiAgentResilienceAndCrossModelGates ./internal/webui/...`
  - **Definition of Done (DoD):** Không còn bất kỳ dòng code/mặc định nào gán `claude-sonnet-4-6 (antigravity)` cho `Laya Gate` hay `codex-5.3 (antigravity)` cho `Security Guard`.

- [x] **TASK-MDL-02: Cập nhật UI Modal Setup CLI (`index.html`) và Thanh Autopilot Model Routing (`app.js`) chỉ hiển thị đúng Engine/Model có thật trên máy**
  - **Context (`file:line`):**
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:916-945`
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:1830-1834, 3040-3041, 3077-3078`
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server_test.go:1665-1675`
  - **Mô tả:**
    - Xóa sạch toàn bộ thẻ `<option value="codex-5.3 (antigravity)">` bịa đặt trong `index.html:921, 930, 940`.
    - Tại ô `cfg-laya-gate-model` (`index.html:926-933`), khóa/hiển thị rõ engine thật: `convaiinnovations/laya (local :8200)`.
    - Tại ô `cfg-security-model` (`index.html:938-944`), chỉ liệt kê đúng cơ chế thật có trên máy: `agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)`, `agy_guard.py (H1..H7) · claude-opus-4-6 (antigravity)`, `agy_guard.py (H1..H7) · gemini-4-argon (antigravity)`.
    - Đồng bộ fallback trong `app.js:1831-1833, 3040-3041, 3077-3078` và cập nhật assertion trong `server_test.go:1665-1675`.
  - **Logs/Test:** `go test -v -race ./internal/webui/...` & `grep -rn "codex-5.3" internal/` trả về 0 kết quả.
  - **Definition of Done (DoD):** Thanh Autopilot Model Routing, Modal Setup CLI và `seed_data.go` hiển thị trung thực 100% engine/model có thật trên máy.

- [x] **TASK-MDL-03: Nâng cấp `/Users/trainguyen/Documents/work/agent` — Bổ sung Wave Phản Biện Trước Khi Lên Plan (`S2.5 Pre-Plan Critique — DG0`) & Tận dụng `/laya_router` Local (`:8200`, 0 Token)**
  - **Context (`file:line`):**
    - `/Users/trainguyen/Documents/work/agent/GEMINI.md:66-87` (Rule #9 Master Workflow & Rule #10 Decision Gate)
    - `/Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:327, 634-650` (`--stage dg0|dg1|dg2|dg12|dg3`)
    - `/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:824-891` (`H4a` S1 Intake & Pre-Plan Critique Gate)
    - `/Users/trainguyen/Documents/work/agent/skills/brain-delegate/SKILL.md:1-60`
    - `/Users/trainguyen/Documents/work/agent/skills/laya-router/SKILL.md:1-80`
  - **Mô tả:**
    - Mở rộng `tooling/laya_gate.py` hỗ trợ `--stage dg0` (`Pre-Plan Adversarial Critique & Reality Check`: kiểm định các claim `C-REQ`, `C-REALITY`, `C-CONTRA` đối chiếu `REF: path:start-end` với thực tế codebase/môi trường trước khi lên Plan `S3`).
    - Cập nhật `GEMINI.md` (giữ dung lượng `< 24KB`): bổ sung bước **`S2.5 PRE-PLAN ADVERSARIAL CRITIQUE (DG0 — Phản biện yêu cầu trước khi lên Plan)`** vào Rule #9 và nâng Rule #10 lên **4 cổng `/laya_router` local 0-token (`DG0`, `DG1`, `DG2`, `DG3`)**. Bắt buộc Agent "la làng lên" (cảnh báo & bác bỏ bằng `file:line` + runtime) nếu prompt chứa giả định/model/tool không có thật.
    - Đồng bộ `skills/brain-delegate/SKILL.md` và `skills/laya-router/SKILL.md`, bổ sung unit test kiểm định `--stage dg0` trong `tests/test_laya_gate.py`.
  - **Logs/Test:** `pytest -q tests/test_laya_gate.py tests/test_agy_guard.py`
  - **Definition of Done (DoD):** `laya_gate.py run --stage dg0` hoạt động trơn tru, `GEMINI.md` `< 24KB`, toàn bộ test suite `/agent` PASS.

- [x] **TASK-MDL-04: Nâng cấp `AI-ORCHESTRATOR` — Bổ sung Wave Phản Biện Yêu Cầu (`Pre-Plan Critique · Laya DG0 :8200`) Trước `Plan (S3)` & Tận Dụng `/laya_router` Local Ở Mọi Chặng**
  - **Context (`file:line`):**
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/agentgate/kernel_bridge.go:220-340` (`BuildAndRunCalibratedLayaGate`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:540-625, 1650-1910` (`buildAutopilotView`, `runSingleStageWithDelay`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:755-830, 1460-1560, 1850-2100, 2600-2995` (`buildWorkflowSteps`, `buildSubTaskPipelineStages`, `runSubTaskLifecycleAction`, `generateDynamicStageArtifacts`)
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:270-305` & `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:815-920, 2650-2730`
  - **Mô tả:**
    - Bổ sung chặng **Wave Phản Biện Trước Khi Lên Plan (`DG0 Pre-Plan Critique · Laya :8200`)** ngay trước `Plan (S3)` trên:
      1. Thanh Autopilot Pipeline (`buildAutopilotView` trong `server.go`): thêm stage `critique` (`Phản Biện Yêu Cầu · Laya DG0 :8200`) trước `plan`, hiển thị rõ `0 LLM Token · Local :8200`.
      2. Thanh Sub-Task Fractal Pipeline (`buildSubTaskPipelineStages` trong `task_inspector.go`) & cột Workflow Steps (`buildWorkflowSteps`): thêm bước `S2.5 Pre-Plan Critique (DG0 · Laya :8200)` ngay trước `S3 Plan & Solution (DG1/DG2)`.
      3. Bộ máy thực thi (`runSingleStageWithDelay` & `runSubTaskLifecycleAction`) và `BuildAndRunCalibratedLayaGate`: gọi `/laya_router` local (`http://127.0.0.1:8200/v1/predict`, 0 token) ở **mọi chặng**: `DG0` (Pre-Plan Critique), `DG1/DG2` (Solution & Demo), `S4` (Implementation Verification), và `DG3` (Final Adversarial Audit), xuất biên bản `10_preplan_critique_*.md` và `06_laya_summary_dg0_*.json`.
  - **Logs/Test:** `go test -v -race ./internal/webui/... ./internal/agentgate/...`
  - **Definition of Done (DoD):** Wave Phản Biện `DG0` chạy trước `Plan` ở cả Task cha và Sub-Task, gọi `/laya_router` local `:8200` (0 token) xuyên suốt mọi chặng, và pass toàn bộ test + live HTTP `:8090`.

- [x] **TASK-MDL-05 (S5 -> S3 -> S4 -> S5 Self-Healing Loop): Đồng bộ hàm ánh xạ chỉ số click trên Frontend (`mapAutopilotBoxIndexToReportTab`, `mapStepNumToReportTab` trong `app.js`) và tích hợp Wave Phản Biện `DG0` vào CLI `TaskRunner` (`internal/orchestrator/task_runner.go`)**
  - **Context (`file:line`):**
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:473-479, 1808-1819`
    - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/orchestrator/task_runner.go:428-455, 535-550, 838-845`
  - **Mô tả:**
    - Cập nhật `mapStepNumToReportTab(stepNum)` (`stepNum === 4 -> "DG0"`) và `mapAutopilotBoxIndexToReportTab(idx)` (ánh xạ theo `grp.id === "critique" -> "DG0"` và fallback chỉ số 7 stage) trong `app.js`.
    - Bổ sung bước `runPrePlanCritiqueDG0` đứng trước `runDecomposeGate` trong `internal/orchestrator/task_runner.go`, sinh `10_preplan_critique.md` và gọi `tr.gatekeeper.BuildAndRunCalibratedLayaGate` cho `dg0`, `dg1`, `dg2`, `dg3` trên local `:8200` (0 LLM token).
  - **Logs/Test:** `go test -v -race ./internal/webui/... ./internal/orchestrator/...`
  - **Definition of Done (DoD):** Click vào thẻ `critique` hoặc Step 4 mở đúng tab `DG0`; cả WebUI lẫn CLI `TaskRunner` đều chạy `DG0` trước `Plan` và gọi `/laya_router` local `:8200` ở mọi chặng.
