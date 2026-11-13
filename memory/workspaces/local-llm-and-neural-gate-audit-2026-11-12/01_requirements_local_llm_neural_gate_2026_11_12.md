# 01 Requirements: Local LLM (Qwen3.5-9B/Ollama) + Audit Dual Neural Gate (`:8200` Laya / `:8222` VegaML) & Pipeline Feedback

tier: T2
suffix: local_llm_neural_gate_2026_11_12
date: 2026-11-12

## 1. Yêu cầu của User (nguyên văn, 2 nhánh "cả a và b")
- **(a) Local LLM phụ:** Máy MacBook Pro Apple M4 (chip M4 thường, 16 GB unified, 512 GB SSD — `sysctl` xác nhận `Apple M4`, `16 GB`, free `81Gi`). Chọn, cài đặt và **benchmark thật** một LLM sinh text chạy local làm "con phụ" cho Agent (Muscle nhẹ / tiền xử lý / sinh dữ liệu), không thay vai Brain.
- **(b) Audit Dual Neural Gate + Pipeline Feedback:** Điều tra thực trạng 2 decision engine đang chạy song song (`Laya :8200` và `VegaML :8222`), xác định vì sao VegaML chưa học được từ feedback (`used_by_auto: false`, 2 head `c_req_check` / `dg0_dg1_routing` chỉ có 2/6 mẫu), và **đề xuất (Propose-Only)** pipeline để VegaML học được — dùng LLM local ở (a) sinh labelled examples nếu phù hợp.

## 2. Hiện trạng đã VERIFIED tại S0/S2 (runtime thật, `file:line`)
- `:8200` — PID 17309, `/Users/trainguyen/.agent/skills/laya_router/api.py:96` (`uvicorn.run(... port=8200)`), engine `convaiinnovations/laya`, paths `/v1/route`, `/v1/predict` (openapi.json). Uptime 23 ngày.
- `:8222` — PID 10181, `/Users/trainguyen/Documents/work/ai-orchestrator/sidecar/vegaml/vega_systemone_sidecar.py:1-15`, engine `NandhaKishorM/vegaml` model `vegaml-0.8b` (reader = `Qwen/Qwen3.5-0.8B`, `config.json: model_type=qwen3_5`), device `mps`. `/health` → `fit.status=fitted`, `num_examples=50`, `verdict.cv_accuracy=0.592 < engine_accuracy=0.612` → `used_by_auto=false`; `c_req_check` & `dg0_dg1_routing` → `fitted=false, examples=1` (cần ≥ `MIN_FIT_EXAMPLES=6`, `vega_systemone_sidecar.py:41`).
- `agent/tooling/laya_gate.py:42` — `DEFAULT_URL = "http://127.0.0.1:8200/v1/predict"`; `agent/` core **0 tham chiếu** tới `8222` (grep).
- `human_gate_feedback.jsonl` (51 dòng): 49 dòng `stage=CALIBRATION_GROUND_TRUTH` (seed `vegaml-seed-*`, label `verdict`), chỉ **2 dòng** từ Human Gate thật (`stage=HUMAN_GATE_S3_REVISE`, label `c_req_check=false`, `dg0_dg1_routing=REJECT_REWRITE_PLAN`). **0 dòng APPROVE** → tập học 1 chiều.
- Ollama `0.40.16` đã cài (`/opt/homebrew/bin/ollama`), service `sh.brew.ollama` đã start; `qwen3.5:9b` (6.6 GB, Q4) đang pull. Ollama library có tag `qwen3.5:{0.8b,2b,4b,9b,27b,35b}` + `*-mlx`.

## 3. Ràng buộc Thực thi từ Bài học (Technical Constraints — S0 Boot, topic T1)
- **L007 / L023 (Strict Physical Reference):** Mọi khẳng định về model/port/engine phải trích `file:line` hoặc output `curl`/`lsof` thật; không dựa vào text "fact-check" do User dán (đã phát hiện text đó sai về Qwen3.5).
- **L045 (Explicit Runtime Call-Site):** Pipeline feedback phải chỉ rõ caller thật: `internal/webui/vega_human_gate.go:100-126` (`buildVegaHumanGateFeedbackPayload`), `:187` (`emitVegaHumanGateFeedbackAsync`), sidecar `:362-394` (`record_human_gate_feedback`), `:86-110` (`run_ttt_fit_sync`).
- **L046 / G2 (Propose-Only):** Em ở vai Brain → **cấm sửa source code** (`.go`, `.py`) trong `ai-orchestrator/` và `agent/` cho tới khi User gõ `APPROVE`. Chỉ được: cài Ollama (hạ tầng máy, không phải source), tạo file workspace, chạy lệnh read-only/benchmark.
- **L059 / G3-G4 (Zero-Mockup, Runtime Exercise):** Benchmark tok/s phải chạy thật qua `ollama run --verbose` / `/api/generate`; không ước tính.
- **L062 (Honest Process State):** Báo rõ cái chưa đo / chưa làm được (ví dụ hook H1–H7 không chạy trên Claude Code).
- **L064 (Strict Scope):** Không đụng `:8200`/`:8222` đang chạy (uptime 13–23 ngày), không restart, không ghi vào `human_gate_feedback.jsonl` thật.
- **G6 (Global Lesson Hygiene):** Lesson mới (nếu có) phải trừu tượng hóa, không chứa `/Users/...`.

## 4. Definition of Done
- **(a):** `qwen3.5:9b` chạy được trên Ollama `:11434`; có số đo thật `eval rate` (tok/s), `prompt eval rate`, RAM RSS của `ollama` khi load, kiểm tra tool-calling + `enable_thinking=false`; ghi vào `06_validation_*`.
- **(b):** `09_tasks_solution_*` với đúng 1 phương án, Code Demo, DG1 + DG2 `PASS` qua `laya_gate.py`; dừng chờ `APPROVE`.
