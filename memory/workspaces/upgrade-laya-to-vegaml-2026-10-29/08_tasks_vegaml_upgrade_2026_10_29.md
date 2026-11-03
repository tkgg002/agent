# 08 Tasks: Nâng cấp thay thế Laya bằng VegaML (`NandhaKishorM/vegaml`) trong `ai-orchestrator`

tier: T2
suffix: vegaml_upgrade_2026_10_29
date: 2026-10-29

## Danh sách Sub-Tasks Thực Thi (Sau khi có `APPROVE`)

### [x] TASK-VEGA-01: Đóng gói `sidecar/vegaml/` tự thân trong `ai-orchestrator` (Server `:8222` đa giao thức + Auto-Bootstrap, giữ nguyên Laya `:8200`)
- **Mô tả & Context `file:line`:**
  - Tạo mới `ai-orchestrator/sidecar/vegaml/vega_systemone_sidecar.py` (chạy mặc định tại `127.0.0.1:8222` qua `VEGA_PORT=8222`, giữ nguyên `Laya 421M` đang chạy độc lập tại `127.0.0.1:8200` cho `/agent`):
    - Nạp `vegaml` (`vega-U` 0.8B mặc định hoặc `4b` qua `VEGA_SIZE`, hỗ trợ `VEGA_DEVICE=mps|cpu|cuda` và fallback an toàn nếu môi trường chưa tải xong model).
    - Cung cấp đầy đủ các endpoints trên cổng `:8222`:
      1. `GET /health`: Trả `{"status": "ok", "engine": "NandhaKishorM/vegaml", "model": ..., "port": 8222, "device": ..., "ttt_status": ..., "num_examples": ...}`.
      2. `POST /v1/systemone` & `POST /v1/predict` & `POST /v1/decide`: Dịch hai chiều `"noul" <-> "boolean"`, chuẩn hóa trường `probs`/`boolean`/`expected_score`/`conformal_set` của `vegaml 0.14.0`, trả đồng thời `"answers"` ở root (chuẩn `/v1/systemone`) và `"result": {"answers": ...}` (chuẩn `laya_gate.py:199,480`).
      3. `POST /v1/route`: Tương thích với `/v1/route` của Laya.
      4. `POST /v1/systemone/feedback` & `POST /v1/systemone/fit`: Lưu mẫu duyệt từ `Human Gate (● You)` vào `.orchestrator/vega-data/human_gate_feedback.jsonl` + `canonical_questions.json` (bảo toàn kiểu `bool` cho `boolean`, `int` cho `score`, `str` cho `choice`) và chạy `v.fit(examples, questions)` nạp Task Adapter (`0.84M` params) với `asyncio.Lock()`.
  - Tạo mới `ai-orchestrator/sidecar/vegaml/requirements.txt` và `ai-orchestrator/sidecar/vegaml/ensure_vega_up.sh` (tự tạo venv tại `.orchestrator/vega-venv`, cài đặt `vegaml>=0.14.0` + `fastapi` + `uvicorn`, khởi chạy background `:8222` và kiểm tra `:8222/health`).
- **Logs / Test:**
  - Chạy unit test & integration test cho bộ dịch schema 2 chiều (`/v1/systemone`, `/v1/predict`, `/v1/systemone/feedback`, `/v1/systemone/fit`, `/health`).
- **Definition of Done (DoD):**
  - `sidecar/vegaml/` nằm hoàn toàn trong repo `ai-orchestrator` tại `:8222`, không chạm vào tiến trình `Laya :8200` của `/agent`.

### [x] TASK-VEGA-02: Xây dựng `internal/agentgate/vega_sidecar.go` & Đấu nối Dual-Gate (`VegaML :8222` + `Laya :8200`) trong `kernel_bridge.go`
- **Mô tả & Context `file:line`:**
  - Tạo mới `ai-orchestrator/internal/agentgate/vega_sidecar.go` (module đơn trách nhiệm):
    - `EnsureVegaSidecarUp(ctx)`: Tự tìm và chạy `sidecar/vegaml/ensure_vega_up.sh` trong `ai-orchestrator` khi `http://127.0.0.1:8222/health` chưa lên.
    - `DecideSystemOneHTTP(ctx, state, questions, mode)`: Gọi trực tiếp `POST http://127.0.0.1:8222/v1/systemone`, bóc tách đầy đủ `readout` (`"engine"` | `"ttt"`), `abstain`, `unbound`, `prediction_set`.
    - `SubmitHumanGateFeedbackHTTP(ctx, req)`: Gọi `POST http://127.0.0.1:8222/v1/systemone/feedback` để dạy VegaML từ quyết định `Human Gate`.
    - Hỗ trợ cơ chế **Dual Local Neural Gate (`Laya :8200` + `VegaML :8222`)**:
      - Với micro-claims ngắn (`< 2K tokens`) và `/agent` kernel (`laya_gate.py`): tận dụng tốc độ `~33ms` của `Laya :8200` (nếu `:8200` đang chạy, hoặc tự động trỏ `laya_gate.py --url http://127.0.0.1:8222/v1/predict` nếu chạy độc lập trên máy chỉ có `ai-orchestrator`!).
      - Với hồ sơ `DG0/DG1/DG2/DG3` toàn cục của `ai-orchestrator` (ghép `10_preplan_critique.md` + `09_tasks_solution.md` + `diff.patch` dài `4K–73K tokens`, chấm `score`, kiểm tra `abstain`/`unbound` và chạy `TTT`): gọi trực tiếp `VegaML :8222` (`/v1/systemone`) và hợp nhất kết quả vào `06_laya_summary_<stage>_<ts>.json`.
  - Cập nhật gọn `ai-orchestrator/internal/agentgate/kernel_bridge.go:138-157` và `kernel_bridge.go:595-741` (đảm bảo `wc -l <= 822`).
- **Logs / Test:**
  - `go test -v ./internal/agentgate/...`
- **Definition of Done (DoD):**
  - `kernel_bridge.go` giữ `NOW <= 822` dòng (`L006`); toàn bộ test trong `internal/agentgate` PASS.

### [x] TASK-VEGA-03: Đấu nối `Human Gate (● You)` Auto-Feedback (`:8222/v1/systemone/feedback` -> `v.fit()`) & Hỗ trợ Song Song `VegaML :8222` + `Laya :8200` trên WebUI
- **Mô tả & Context `file:line`:**
  - Tạo mới `ai-orchestrator/internal/webui/vega_human_gate.go` (module đơn trách nhiệm):
    - Hàm `emitVegaHumanGateFeedbackAsync(taskKey, stage, approved, reason, isoMeta)`: Đọc `10_preplan_critique.md`, `09_tasks_solution.md`, `diff.patch` / snapshot, đóng gói bộ câu hỏi chuẩn (`c_req_check` kiểu `noul`/`boolean`, `dg0_dg1_routing` kiểu `choice`) và nhãn thực tế do User duyệt (`PASS_TO_HUMAN_GATE` khi Approve, `REJECT_REWRITE_PLAN` khi Revise/Reject) rồi bắn bất đồng bộ sang `POST http://127.0.0.1:8222/v1/systemone/feedback`.
    - Cung cấp HTTP handlers `POST /api/vega/feedback`, `POST /api/vega/fit`, `GET /api/vega/status` trên WebUI.
  - Đấu nối 1 dòng gọi tại các caller thực tế (`L002`, giữ `NOW <= HEAD` trên mọi file lớn `L006`):
    - `internal/webui/server.go:1388` (`case "approve":` khi User bấm Approve Task).
    - `internal/webui/subtask_wave_engine.go:240` (`case "subtask_approve":` khi User bấm Approve từng Sub-Task).
    - `internal/webui/telegram_bridge.go:1960` (`handleTelegramReviseSolutionCommand` khi User gửi lệnh Revise/Sửa Plan S3).
  - Cập nhật dropdown và `extractModelFamilyPrefix` tại `internal/webui/task_inspector.go`, `internal/webui/static/index.html` để hỗ trợ cả `NandhaKishorM/vegaml (vega-U 0.8B · local :8222)` và `convaiinnovations/laya (local :8200)`.
- **Logs / Test:**
  - `go test -v ./internal/webui/...`
- **Definition of Done (DoD):**
  - Toàn bộ file lớn (`server.go <= 3231`, `task_inspector.go <= 4024`, `telegram_bridge.go <= 2790`) tuân thủ nghiêm ngặt `NOW <= HEAD` (`L006`); test suite `go test ./...` xanh 100%; có biên lai `EV-...` thật từ `evidence_run.py`.


---

## Re-Audit Fix Tasks (sau `APPROVE` 2026-10-29T16:17) — ánh xạ 1-1 `F1..F10`

- [x] **TASK-VEGA-FIX-01 (F1):** `kernel_bridge.go` — laya_gate.py chỉ chạy trên Laya `:8200`, bỏ `--url` định tuyến sang `:8222`. DoD: `grep VegaPredictURL` = 0 trong production; DG3 claim A-FIX-01 ACCEPT.
- [x] **TASK-VEGA-FIX-02 (F2):** File mới `internal/agentgate/neural_gate_boot.go` (`ensureLayaUp` khôi phục logic HEAD + `EnsureDualNeuralGates`). DoD: `TestEnsureDualNeuralGatesReportsStatesSeparately` PASS.
- [x] **TASK-VEGA-FIX-03 (F3):** `vega_human_gate.go` — chỉ gửi 2 nhãn có nguồn từ người duyệt. DoD: `TestVegaHumanGateFeedbackOnlyHumanLabels` PASS.
- [x] **TASK-VEGA-FIX-04 (F4):** Mirror chỉ ghi khi sidecar offline. DoD: `TestVegaFeedbackRouteBuffersLocallyOnlyWhenSidecarOffline` (đúng 1 dòng) PASS.
- [x] **TASK-VEGA-FIX-05 (F5):** Test không mock, không ghi kho học thật; dọn 3 dòng `VEGA-LIVE-01` (backup `.bak_20261029T1621`). DoD: kho học còn 0 dòng sau toàn bộ test.
- [x] **TASK-VEGA-FIX-06 (F6):** Timeout decide 60s + `VEGA_DECIDE_TIMEOUT_SEC`. DoD: `TestVegaDecideTimeoutOverride` PASS.
- [x] **TASK-VEGA-FIX-07 (F7):** `ensure_vega_up.sh` `exit 1` khi health không 2xx. DoD: chạy thật trên cổng xung đột trả exit=1.
- [x] **TASK-VEGA-FIX-08 (F8):** Revert `index.html` + `task_inspector.go` về HEAD. DoD: `git status` không còn 2 file này.
- [x] **TASK-VEGA-FIX-09 (F9):** `WriteVegaPhysicsSummary` ghi file riêng `06_vega_physics_*`; nhãn fallback về như HEAD. DoD: live test sinh file với `advisory_only=true`.
- [x] **TASK-VEGA-FIX-10 (F10):** `HF_HOME` trong `.orchestrator/vega-hf`. DoD: sidecar boot với `HF_HUB_OFFLINE=1` → `/health` `ok`.

## Round 4 Tasks (sau `APPROVE` 2026-10-29T16:38) — xử lý dứt điểm `P2/R1`, `R2`, `P3`, `R4`

- [x] **TASK-VEGA-R4-01 (`P2 + R1`):** `internal/agentgate/kernel_bridge.go` (`checkLayaHealthHTTP`, `BuildAndRunCalibratedLayaGate`) — hỗ trợ fallback `GET :8200/openapi.json` khi `:8200/health` trả `404` trên Laya server hiện hành để `LayaUp=true` và nhánh Laya-live + `WriteVegaPhysicsSummary` (`06_vega_physics_*`) thực thi thật khi `liveAllowed=true`. DoD: `EnsureDualNeuralGates` trả `layaUp=true, vegaUp=true` khi cả 2 daemon đang chạy; `BuildAndRunCalibratedLayaGate` sinh cả `06_laya_summary_*` và `06_vega_physics_*` khi live.
- [x] **TASK-VEGA-R4-02 (`R2`):** `internal/webui/vega_human_gate.go` (`persistLocalVegaFeedbackMirror`) — đọc và merge `canonical_questions.json` hiện có trên đĩa trước khi ghi thay vì ghi đè bằng tập câu hỏi con. DoD: unit test kiểm chứng khóa câu hỏi có sẵn (`architecture_safety_score`) được giữ nguyên sau khi mirror offline ghi thêm feedback.
- [x] **TASK-VEGA-R4-03 (`P3`):** Khắc phục 2 test FAIL tồn đọng tại `internal/agentgate/live_schema_parser.go:13`, `internal/webui/static/task_edit_modal.js:200`, và `internal/webui/task_llm_bridge_test.go:223-237` (dùng `t.TempDir()` + nạp `TASK-13` nếu queue trống để cô lập 100% kho `queue.db` thật). DoD: `go test ./... -count=1` toàn repo `ai-orchestrator` PASS 100% (`exit=0`) kèm biên lai `EV-...`.
- [x] **TASK-VEGA-R4-04 (`R4`):** Dọn file tạm `.incomplete` (`67 MB`) trong `~/.cache/huggingface/hub/models--Qwen--Qwen3.5-0.8B/blobs/*.incomplete` do tiến trình probe bị hủy ở lượt trước để lại. DoD: `ls ~/.cache/huggingface/hub/models--Qwen--Qwen3.5-0.8B/blobs/*.incomplete` trả về 0 file.

