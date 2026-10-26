---
tier: T2
suffix: fix_hallucinated_laya_and_security_models
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T14:09:30+07:00
mode: propose-only
---

# 09. Tasks Solution & Code Demo — Khắc Phục Cấu Hình Model Bịa Đặt ở `Laya Gate (Slot H)` & `Security (Slot I)`

> **Phương án Tối Ưu Duy Nhất (`Rule #9.4` & `Rule #13` — Propose-Only):** Dọn sạch 100% cấu hình bịa đặt `Laya Gate (Slot H): claude-sonnet-4-6 (antigravity)` và `Security (Slot I): codex-5.3 (antigravity)` do lỗi nhắm mắt làm theo ví dụ trong prompt trước đây. Trả `Laya Gate` và `Security Guard` về đúng 100% engine/model thực tế đang tồn tại trên máy của anh.

---

## 1. Kiến Trúc & Định Tuyến Engine/Model Thực Tế Trên Máy (`Ground Truth`)

| Slot / Vai trò | Cấu hình sai (bịa đặt trước đây) | Cấu hình đúng (vật lý có thật trên máy) | Bằng chứng `file:line` |
| :--- | :--- | :--- | :--- |
| **`Planner / Architect` (`Slot A / G`)** | `gemini-4-argon (antigravity)` | `gemini-4-argon (antigravity)` | `internal/runner/antigravity.go:68-75` |
| **`MUSCLE Coder` (`Slot B / D / E`)** | `gemini-4-argon (antigravity)` | `gemini-4-argon (antigravity)` | `internal/runner/antigravity.go:68-75` |
| **`Cross-Model QA Auditor` (`Slot K`)** | `claude-opus-4-6 (antigravity)` | `claude-opus-4-6 (antigravity)` *(có thật trong Antigravity & `/Users/trainguyen/.local/bin/claude`)* | `internal/runner/antigravity.go:73, 99` |
| **`Laya Gate` (`Slot H` — `DG1/DG2/DG3`)** | ❌ `claude-sonnet-4-6 (antigravity)` *(bịa đặt — Laya không phải Claude!)* | ✅ **`convaiinnovations/laya (local :8200)`** *(Local Neural Decision Engine 800MB chạy trên `http://127.0.0.1:8200/v1/predict`)* | `/Users/trainguyen/.agent/skills/laya_router/api.py:17` & `laya_gate.py:42` |
| **`Security & Kernel Guard` (`Slot I` — `H1..H7`)** | ❌ `codex-5.3 (antigravity)` *(bịa đặt — `codex not found` & không có trong Antigravity!)* | ✅ **`agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)`** *(Hooks tất định `agy_guard.py` + model Claude Sonnet 4.6 có thật trên máy)* | `/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:1` & `antigravity.go:74, 100` |

---

## 2. Concrete Code Demo (`Go`, `HTML`, `JS`)

### 2.1. Backend (`internal/webui/task_inspector.go:39-45, 637-741`)
```go
func extractModelFamilyPrefix(model string) string {
	lower := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(lower, "convaiinnovations/laya") || strings.HasPrefix(lower, "laya"):
		return "laya-local-8200"
	case strings.Contains(lower, "agy_guard"):
		if strings.Contains(lower, "claude") {
			return "agy-guard+claude"
		}
		return "agy-guard"
	case strings.HasPrefix(lower, "gemini"):
		return "gemini"
	case strings.HasPrefix(lower, "claude"):
		return "claude"
	default:
		if idx := strings.IndexAny(lower, "- ("); idx > 0 {
			return lower[:idx]
		}
		return lower
	}
}

func (s *Server) loadCLISettings() CLISettingsDTO {
	agentRoot := "/Users/trainguyen/Documents/work/agent"
	if s.Gatekeeper != nil && s.Gatekeeper.AgentCoreDir != "" {
		agentRoot = s.Gatekeeper.AgentCoreDir
	}
	def := CLISettingsDTO{
		DefaultRunner:      s.activeRunner,
		PlannerModel:       "gemini-4-argon (antigravity)",
		CoderHighModel:     "gemini-4-argon (antigravity)",
		CoderStdModel:      "gemini-2.5-flash (antigravity)",
		QAAuditorModel:     "claude-opus-4-6 (antigravity)",
		LayaGateModel:      "convaiinnovations/laya (local :8200)",
		SecurityModel:      "agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)",
		MaxSelfHealRetries: 3,
		AgentKernelRoot:    agentRoot,
		EnforceApproveGate: true,
	}
	// ...
	// Sanitize any previously persisted hallucinated model names
	if loaded.LayaGateModel == "" ||
		strings.Contains(loaded.LayaGateModel, "claude-sonnet-4-6") ||
		strings.Contains(loaded.LayaGateModel, "gemini-") ||
		strings.Contains(loaded.LayaGateModel, "codex") {
		loaded.LayaGateModel = def.LayaGateModel
	}
	if loaded.SecurityModel == "" ||
		strings.Contains(loaded.SecurityModel, "codex") ||
		loaded.SecurityModel == "gemini-2.5-pro (antigravity)" {
		loaded.SecurityModel = def.SecurityModel
	}
	if strings.Contains(loaded.QAAuditorModel, "codex") {
		loaded.QAAuditorModel = def.QAAuditorModel
	}
```

### 2.2. Frontend (`internal/webui/static/app.js:1830-1834` & `internal/webui/static/index.html:916-945`)
```javascript
    const qaMdl = dashboardState.qa_auditor_model || dashboardState.cli_settings?.qa_auditor_model || "claude-opus-4-6 (antigravity)";
    const layaMdl = dashboardState.laya_gate_model || dashboardState.cli_settings?.laya_gate_model || "convaiinnovations/laya (local :8200)";
    const secMdl = dashboardState.security_model || dashboardState.cli_settings?.security_model || "agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)";
    routingEl.innerHTML = `Planner: <b>${escapeHtml(dashboardState.planner_model || "gemini-4-argon")}</b> · Coder: <b>${escapeHtml(dashboardState.coder_high_model || "gemini-4-argon")}</b> · Cross-Model QA (Slot K): <b>${escapeHtml(qaMdl)}</b> · Laya Gate (Slot H): <b>${escapeHtml(layaMdl)}</b> · Security (Slot I): <b>${escapeHtml(secMdl)}</b> · Circuit Breaker: <b>Retry ${retryCnt}/${maxRetries}</b>`;
```

### 2.3. `/agent` Core Upgrade (`tooling/laya_gate.py`, `GEMINI.md`, `hooks/agy/agy_guard.py`) — `TASK-MDL-03`
- **`tooling/laya_gate.py` (`--stage dg0`):**
```python
    p_run = sub.add_parser("run", help="Run Mode A Calibrated Claim Gate (dg0, dg1, dg2, dg12, dg3)")
    p_run.add_argument("--stage", required=True, choices=["dg0", "dg1", "dg2", "dg12", "dg3"])
```
- **`GEMINI.md` (Rule #9.3 & Rule #10):**
  - Thêm bước **`S2.5 PRE-PLAN ADVERSARIAL CRITIQUE (DG0 — Phản biện yêu cầu trước khi lên Plan)`** ngay trước `S3 SOLUTION`: Đối chiếu từng giả định/ví dụ trong prompt với `file:line` & binary/model thật trên máy ("biết la làng lên" bác bỏ giả định sai), chạy `/laya_router` local (`laya_gate.py run --stage dg0`, `:8200`, 0 LLM token) kiểm định các claim `C-REQ`, `C-REALITY`, `C-CONTRA`.
  - Nâng Rule #10 thành **4 cổng bắt buộc chạy trên `/laya_router` local (`:8200`, 0 token)**: `DG0` (Pre-Plan Critique), `DG1` (Solution), `DG2` (Code Demo), `DG3` (Final Adversarial Audit & Diff Review).

### 2.4. `AI-ORCHESTRATOR` Upgrade (`internal/webui/server.go`, `internal/webui/task_inspector.go`, `internal/agentgate/kernel_bridge.go`) — `TASK-MDL-04`
- **Bổ sung Wave Phản Biện Trước Khi Lên Plan (`critique` / `DG0 Pre-Plan Critique · Laya :8200`)** đứng ngay trước `plan` trên:
  1. `buildAutopilotView` (`server.go:540-625`): thêm stage `ID: "critique"`, `Title: "Phản Biện Yêu Cầu (Laya DG0 :8200)"` trước `ID: "plan"`.
  2. `buildSubTaskPipelineStages` & `buildWorkflowSteps` (`task_inspector.go`): thêm chặng `S2.5 Pre-Plan Critique (DG0 · Laya :8200 · 0 Token)` trước `S3 Plan & Solution (DG1/DG2)`.
  3. `BuildAndRunCalibratedLayaGate` (`internal/agentgate/kernel_bridge.go`) & `runSingleStageWithDelay` (`server.go`): gọi trực tiếp `/laya_router` local (`http://127.0.0.1:8200/v1/predict`, 0 token) ở **mọi chặng**: `dg0` (`critique`), `dg1` (`plan`), `dg2` (`implement`), và `dg3` (`qa` / `finalize`), sinh `10_preplan_critique_<suffix>.md` + `06_laya_summary_dg0_*.json`.

### 2.5. Vòng Lặp Tự Chữa Lành `S5 -> S3 -> S4 -> S5` (`TASK-MDL-05`): Đồng Bộ Hàm Ánh Xạ Click `app.js` & Luồng CLI `internal/orchestrator/task_runner.go`
- **Phát hiện tại `S5 Adversarial Stop-Audit` (`H7 Pass #1`):**
  1. `internal/webui/static/app.js:473-479` (`mapStepNumToReportTab`) và `app.js:1808-1819` (`mapAutopilotBoxIndexToReportTab`) vẫn dùng bảng chỉ số cũ của 6 stages / 16 steps nên khi click vào stage `critique` (`idx=1`) hoặc Step 4 (`stepNum=4`) chưa mở đúng tab `DG0`.
  2. `internal/orchestrator/task_runner.go:428-455, 838-845` (bộ chạy CLI `orchestrator run` / `daemon`) nhảy từ `Phase 0: Research` sang `Phase 2: Decompose` mà chưa chạy `Phase 1.5 (S2.5): Pre-Plan Adversarial Critique Wave (DG0 · Laya :8200 · 0 Token)` và chưa gọi `BuildAndRunCalibratedLayaGate` cho `dg0`, `dg1`, `dg2`, `dg3`.
- **Code Demo khắc phục dứt điểm (`app.js` & `task_runner.go`):**
```javascript
function mapStepNumToReportTab(stepNum) {
  if (stepNum <= 2) return "S1";
  if (stepNum === 3) return "S2";
  if (stepNum === 4) return "DG0";
  if (stepNum <= 10) return "S3";
  if (stepNum <= 12) return "S4";
  return "S5";
}

function mapAutopilotBoxIndexToReportTab(idx) {
  const grp = (dashboardState?.autopilot_stages || [])[idx];
  const gid = String(grp?.id || "").toLowerCase();
  if (gid === "start") return "S1";
  if (gid === "critique") return "DG0";
  if (gid === "plan" || gid === "you") return "S3";
  if (gid === "implement") return "S4";
  if (gid === "qa" || gid === "pr") return "S5";
  if (idx === 0) return "S1";
  if (idx === 1) return "DG0";
  if (idx === 2 || idx === 3) return "S3";
  if (idx === 4) return "S4";
  return "S5";
}
```
- Trong `internal/orchestrator/task_runner.go`: bổ sung hàm `runPrePlanCritiqueDG0(ctx, spec, ctxFragment)` chạy ngay trước `runDecomposeGate`, sinh `10_preplan_critique.md` và gọi `tr.gatekeeper.BuildAndRunCalibratedLayaGate(ctx, "dg0", ...)` trên local `:8200` (0 LLM token), đồng thời gọi `BuildAndRunCalibratedLayaGate` cho `dg1`, `dg2`, `dg3`.
