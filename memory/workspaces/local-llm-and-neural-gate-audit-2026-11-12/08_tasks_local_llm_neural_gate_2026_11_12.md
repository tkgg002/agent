# 08 Tasks — local_llm_neural_gate_2026_11_12

tier: T2
suffix: local_llm_neural_gate_2026_11_12
date: 2026-11-12

## Nhánh (a) — Local LLM (được phép thực thi ngay: hạ tầng máy, không phải source code)

### [x] TASK-LLM-01: Cài Ollama + pull `qwen3.5:9b`
- **Mô tả + Context:** `brew install ollama` (→ `/opt/homebrew/bin/ollama` 0.40.16), `brew services start ollama`, `ollama pull qwen3.5:9b` (tag từ ollama.com/library/qwen3.5: 6.6–7.6 GB, 256K ctx, vision+tools+thinking).
- **Logs/Test:** `/tmp/ollama_install.log`, `/tmp/ollama_pull_9b.log`; `curl :11434/api/tags`.
- **DoD:** `ollama list` hiện `qwen3.5:9b`; `/api/tags` trả model.

### [x] TASK-LLM-02: Benchmark thật trên M4/16GB
- **Mô tả:** `ollama run qwen3.5:9b --verbose` với prompt coding cố định; đo `prompt eval rate`, `eval rate`, `total duration`; đo RSS `ollama` khi model loaded (`ps -o rss`); test `enable_thinking=false` qua `/api/chat` với `"think": false`; test tool-calling JSON.
- **DoD:** Bảng số đo thật ghi vào `06_validation_*`; kết luận 9B có vừa RAM không (swap/pageouts trước–sau).

### [x] TASK-LLM-03: Khuyến nghị cấu hình & đấu nối
- **Mô tả:** Đề xuất `models.env` slot `MUSCLE_POOL` thêm `ollama:qwen3.5:9b` (Propose-Only — không sửa file `models.env` chưa APPROVE); hướng dẫn Cline/Continue trỏ `http://127.0.0.1:11434/v1`.
- **DoD:** Nằm trong `09_tasks_solution_*` mục A.

## Nhánh (b) — Audit Dual Neural Gate + Pipeline Feedback (Propose-Only tới khi APPROVE)

### [x] TASK-GATE-01: Audit hiện trạng `:8200` / `:8222` (S2, read-only)
- **Context file:line:** `laya_gate.py:42`; `vega_systemone_sidecar.py:41,86-110,362-394`; `vega_human_gate.go:100-126,187-210`; `kernel_bridge.go:145,694`; `human_gate_feedback.jsonl`; `canonical_questions.json`.
- **DoD:** Bảng finding `F1..Fn` kèm `REF:` trong `09_tasks_solution_*`.

### [x] TASK-GATE-02: Thiết kế pipeline để VegaML học được (S3)
- **Mô tả:** Đúng 1 phương án: dùng `qwen3.5:9b` local sinh labelled examples (có kiểm chứng đối chiếu code thật) cho 2 head `c_req_check` / `dg0_dg1_routing` + bổ sung mẫu `verdict` có `flow`/`evidence` đủ; cơ chế gửi `POST :8222/v1/systemone/feedback` với `auto_fit=false` rồi `POST /v1/systemone/fit` 1 lần; ghi vào file **tách riêng** (không chạm `human_gate_feedback.jsonl` thật khi chưa APPROVE).
- **DoD:** Code Demo (Python script đơn trách nhiệm trong `ai-orchestrator/sidecar/vegaml/`), DG1 + DG2 `PASS`.

### [x] TASK-GATE-03: DG1/DG2 qua `laya_gate.py` rồi DỪNG CHỜ `APPROVE`
- **DoD:** `06_laya_summary_dg1_*.json`, `06_laya_summary_dg2_*.json` có `PASS`; append `05_progress`.

## Sau `APPROVE` (S4–S6, chưa làm)
- [x] TASK-GATE-04: Thực thi script sinh mẫu + fit; verify `/health` `used_by_auto=true` hoặc `fitted=true` cho 2 head.
- [x] TASK-GATE-05: DG3 + `audit_report_*` + `verify_governance.py --workspace local-llm-and-neural-gate-audit-2026-11-12`.
