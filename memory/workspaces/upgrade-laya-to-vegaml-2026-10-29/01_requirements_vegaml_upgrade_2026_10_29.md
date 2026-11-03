# 01 Requirements: Nâng cấp thay thế Laya bằng VegaML (`NandhaKishorM/vegaml`) tự thân trong `ai-orchestrator`

tier: T2
suffix: vegaml_upgrade_2026_10_29
date: 2026-10-29

## 1. Yêu cầu của User
- Triển khai kiến trúc **Dual Local Neural Gate (`0 LLM Token`)**:
  - **Neo giữ `Laya` (`convaiinnovations/laya` 421M) tại cổng `:8200`:** Tiếp tục phục vụ repo `/agent` độc lập (`tooling/laya_gate.py`, `hooks/agy/agy_guard.py`) và làm tầng **Fast Micro-Claim Verifier (`~33ms`, ngữ cảnh ngắn `< 2K tokens`)**.
  - **Triển khai `VegaML` (`NandhaKishorM/vegaml` — `vega-U` 0.8B / 4B) tại cổng `:8222` tự thân bên trong `ai-orchestrator`:** Đảm nhận vai trò **Deep-Context Physics Gate (`73K tokens` + Prefix Cache `>= 4,096 tokens`)**, chấm thang điểm thứ tự `score`, phát hiện bất định (`abstain`, `unbound`, `prediction_set`) và **Test-Time Training (`v.fit()`) trực tiếp từ `Human Gate (● You)`**. Khi mang `ai-orchestrator` đi nơi khác không có `/agent` hay Laya `:8200`, `VegaML :8222` tự gánh 100% mọi cổng `DG0–DG3`.
- **Ràng buộc vị trí lưu trữ & kiến trúc đóng gói (Portability):**
  - Toàn bộ Sidecar VegaML (`:8222`), script khởi tạo môi trường ảo (`sidecar/vegaml/` & `.orchestrator/vega-venv`), client kết nối và cơ chế học Test-Time Training (`v.fit()` từ `Human Gate`) **bắt buộc phải nằm gọn bên trong repository `/Users/trainguyen/Documents/work/ai-orchestrator`** để có thể mang `ai-orchestrator` đi nơi khác độc lập.
  - Repository `/Users/trainguyen/Documents/work/agent` hoạt động độc lập ở bên ngoài giữ nguyên kết nối tới `Laya :8200` (và có thể trỏ `--url http://127.0.0.1:8222/v1/predict` sang VegaML bất cứ lúc nào vì Sidecar `:8222` hỗ trợ đầy đủ cả `/v1/systemone`, `/v1/decide`, `/v1/predict`, `/v1/route`, `/v1/systemone/feedback`, `/v1/systemone/fit`, `/health`).
- **Yêu cầu kỹ thuật cốt lõi của VegaML Sidecar (`:8222`) & Orchestrator Integration:**
  1. **Two-Way Schema Translation (`/v1/systemone` & `/v1/predict` <-> `vegaml.decide()`):**
     - Tự động dịch kiểu câu hỏi `"noul"` sang `"boolean"` cho `vegaml >= 0.5.0` (`0.14.0` mới nhất trên PyPI) và dịch ngược kết quả về cấu trúc chuẩn `/v1/systemone` + `/v1/predict` (`result.answers`).
     - Bảo toàn và khai thác các tín hiệu vật lý 64-D & Conformal Prediction của VegaML: `readout` (`"engine"` | `"ttt"`), `abstain` (`bool`), `unbound` (`bool`), `prediction_set` (`conformal_set`). Khi `abstain == true` hoặc `unbound == true`, đánh dấu cần chuyển tiếp (`FLAG` / `ESCALATE_TO_SLOT_K`).
  2. **Test-Time Training (`v.fit()`) trực tiếp từ `Human Gate (● You)` tại `:8222`:**
     - Lưu trữ bền vững (`human_gate_feedback.jsonl` & `canonical_questions.json`) trong `.orchestrator/vega-data/`.
     - Khi User bấm **`APPROVE`** (`approve`, `approve_all`, `subtask_approve`) hoặc **`REVISE / REJECT`** (`revise_solution`) trên WebUI hoặc Telegram ở cổng `Human Gate (NEEDS_YOU)`, `ai-orchestrator` tự động bắn bất đồng bộ (`fire-and-forget`) trạng thái thực tế (`10_preplan_critique.md`, `09_tasks_solution.md`, code snapshot/diff) kèm nhãn duyệt (`labels`) vào `POST http://127.0.0.1:8222/v1/systemone/feedback`.
     - Khi đủ mẫu (`>= MIN_FIT_EXAMPLES = 6`), tự động chạy `v.fit(examples, questions)` nạp Task Adapter (`0.84M` params) để chế độ `mode="auto"` tự động dùng `"ttt"` cho các câu hỏi vượt ngưỡng Cross-Validation và fallback về `"engine"` (Physics Engine 12 bước) cho câu hỏi còn lại.

## 2. Ràng buộc Thực thi từ Bài học (Technical Constraints — S0 Boot)
- **`L002` (`Explicit Runtime Call-Site & Lifecycle Trigger Wiring`):** Không chỉ đặt file Python sidecar rời rạc; bắt buộc đấu nối rõ ràng vào từng `caller file:line` trong luồng vận hành của `ai-orchestrator` (`S0 Boot` tại `kernel_bridge.go`, `DG0/DG1/DG2/DG3` tại `kernel_bridge.go`, `Human Gate Approve/Revise` tại `server.go`, `subtask_wave_engine.go`, `telegram_bridge.go`).
- **`L003` (`Propose-Only Discipline & Explicit Approval Gate`):** Giữ chế độ **100% Read-Only (Propose-Only)** đối với source code tại các bước `S1 -> S2.5 (DG0) -> S3 -> DG1/DG2`. Hiển thị trực tiếp **Code Demo** trên khung chat và dừng chờ lệnh `APPROVE` tường minh của anh trước khi thực thi `S4`.
- **`L006` (`Single-Responsibility Module Split & Line-Budget Guard`):** Các file lớn trong `ai-orchestrator` (`server.go` 3231 dòng, `task_inspector.go` 4024 dòng, `kernel_bridge.go` 822 dòng, `telegram_bridge.go` 2790 dòng) bắt buộc giữ `NOW <= HEAD` sau khi sửa; tách toàn bộ logic VegaML Sidecar Manager, Native Client và Human Gate Feedback Emitter ra các file đơn trách nhiệm riêng (`sidecar/vegaml/*`, `internal/agentgate/vega_sidecar.go`, `internal/webui/vega_human_gate.go`).
- **`L009` (`Pre-Plan Adversarial Critique Wave DG0`):** Phản biện ngược mọi điểm lệch giữa code mẫu trong tài liệu với hệ thống thực tế (`GET /health`, `POST /v1/predict` bọc `result.answers`, phiên bản `vegaml 0.14.0` trên PyPI, kiểu dữ liệu boolean/score trong `v.fit()`) và kiểm định qua `laya_gate.py run --stage dg0` trước khi lập `09_tasks_solution`.
- **`L016` & `L019` (`Zero-Mockup Evidence & Honest Process State`):** Không dùng mock closure (`*Fn`) trong code production; khi thực thi ở `S4/S5`, kiểm tra thật tiến trình đang giữ cổng `:8200` (`lsof -iTCP:8200`), chạy test thật qua `evidence_run.py` lấy biên lai `EV-...` và đo lại `wc -l` thực tế.
- **`L020` (`Direct-Answer Summary First, Macro Topology & 1-to-1 Finding Mapping`):** Mở đầu phản hồi bằng khối **Direct Summary** 2–4 câu, trình bày **Macro Runtime Topology** và ánh xạ giải pháp 1-1 theo từng phát hiện `F1..F4`.
