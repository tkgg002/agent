# 06 Validation — local_llm_neural_gate_2026_11_12

tier: T2
suffix: local_llm_neural_gate_2026_11_12
date: 2026-11-12

## A. Nhánh (a) — Benchmark thật `qwen3.5:9b` trên Ollama 0.40.16 / Apple M4 16GB

**Môi trường:** `sysctl` → `Apple M4`, `16 GB`; Ollama `/opt/homebrew/Cellar/ollama/0.40.16`; `ollama list` → `qwen3.5:9b d63f830f28d5 7.6 GB`; `/api/ps` → `runner=mlx`, `size_vram=8.17 GB`, `context_length=8192`.
**Script:** `/tmp/bench_ollama.py` (lưu bản sao bên dưới), output `/tmp/bench_ollama.out`. Trong lúc đo, `:8200` + `:8222` + 6 Postgres + Mongo + Redis + ClickHouse vẫn đang chạy (không tắt gì).

| Run | Cấu hình | prompt_tok | prompt tok/s | gen_tok | **gen tok/s** | total_s | Ghi chú |
|---|---|---|---|---|---|---|---|
| code-run1 | think=false, num_ctx=8192, max 512 | 58 | n/a (cache) | 512 | **16.1** | 236.7 | Lần đầu sau load: ~200s là MLX compile/warm-up, không phải gen |
| code-run2 | như trên, warm | 58 | 397.9 | 512 | **16.3** | 31.5 | Số thật ổn định |
| tool-call | tools=[classify_claim], max 200 | 361 | 134.1 | 64 | **22.3** | 6.6 | `tool_calls` hợp lệ: `{"verdict":"INCORRECT","reason":"...port 8200, not 8222"}` — **đúng** |
| think-on | think=true, max 400 | 23 | 100.8 | 400 | **28.3** | 14.4 | `thinking_len=850`, **answer rỗng** → cạn budget trong `<think>` (khớp cảnh báo HF model card về thinking loop) |

**RAM/swap:** `vm_stat Pageouts` trước 9,965,551 → sau 9,970,474 (+4,923 trang ≈ 19 MB — không đáng kể). `memory_pressure` free 31% → **20%** khi model loaded. Kết luận: 9B **vừa** 16GB khi đang chạy toàn bộ stack DB của anh, nhưng không còn dư; không thể lên 27B.

**Chất lượng code (quan sát, không phải benchmark):** prompt yêu cầu "in place, không allocate" → model vẫn `make([]int, 0, len(xs))` rồi tự chú thích. 9B làm Muscle phụ được, **không** làm Brain.

**Kết luận (a):**
- Dùng `think=false` mặc định; chỉ bật think khi có `num_predict ≥ 2000`.
- Tool-calling đủ tốt để làm pre-filter / sinh draft, nhưng nhãn cho Vega vẫn **không** lấy từ LLM (xem 09 §3.A).
- Cấu hình khuyến nghị: `OLLAMA_KV_CACHE_TYPE=q8_0`, `OLLAMA_FLASH_ATTENTION=1` (brew caveat gợi ý), `num_ctx=8192–16384`.

## B. Nhánh (b) — Dữ liệu replay có sẵn (read-only, VERIFIED)
- `glob agent/memory/workspaces/*/06_laya_claims_*.json` → 148 file, 1047 claim; **437** claim `control_true|control_false` có đủ `claim`+`evidence` (top: `audit-master-mongo-orchestrator-2026-10-22` 186).
- `find ai-orchestrator/.orchestrator/tasks -name .user-approved` → **10** (tất cả dưới `t-ch-h-p-Master-MongoDB-Destination/subtasks/TASK-M*/`), mỗi thư mục có `09_tasks_solution.md`, `10_preplan_critique.md`, `diff.patch`.
- `human_gate_feedback.jsonl` → 51 dòng: 49 `verdict` seed (17 CORRECT / 16 INCORRECT / 16 UNSUPPORTED), 2 `c_req_check=false`. 0 dòng APPROVE.
- `:8222/health` → `fit.status=fitted, num_examples=50, verdict.cv_accuracy=0.592, engine_accuracy=0.612, used_by_auto=false`; `c_req_check` & `dg0_dg1_routing` `fitted=false, examples=1`.
- `vegaml 0.14.0`: `Vega.fit(examples, questions, folds=5, min_examples=6)`; `Vega.decide(state, questions, mode=None, alpha='0.1', views=1, review_below=None)`.

## C. Decision Gate (`laya_gate.py` trên Laya `:8200` thật)
- DG1 v1 `20261112T165242` → `NEEDS_RESOLUTION` (trusted `[r2]`; 4 FLAG do claim ghép ý).
- DG1 v2 `20261112T165553` → ACCEPT: S-RC-02, S-RC-03, S-FIX-01, S-PATTERN-01; FLAG: S-RC-01, S-IMPACT-01 → `resolve` với `REF:` (ghi `04_decisions_local-llm-and-neural-gate-audit-2026-11-12.md`) → **PASS**.
- DG2 v1 `20261112T165330` → `BLOCK` D-API-01 (REF `:176` ≠ đĩa `:175`) — sửa REF.
- DG2 v2 `20261112T165540` → 3/3 ACCEPT → **PASS** (trusted `[r3]`).
- Control calibration: K4 (`control_true`) đúng ở mọi round; K2 (`AutoFit:false`) → `UNSUPPORTED` thay vì `INCORRECT` ở r1/r2 cả 2 lần (điểm yếu engine với boolean đảo).

## E. Kết quả S4–S5 (runtime thật sau APPROVE)
- **Replay** (`/tmp/vega_replay_apply.log`): backup `.bak_20261112T1704`; 458 POST 200 OK (jsonl 51→509); fit 395.4s → `verdict` 497 mẫu **cv 0.893 / engine 0.276 / auto=true**; `c_req_check` `fitted=false` (bug nhãn bool).
- **Hotfix** `to_fit_label()` → unittest `test_vega_sidecar` + `test_vega_replay_ground_truth`: **11/11 OK** (53s, có 1 live read-only test). Sidecar restart PID 11668, boot refit 361s → `c_req_check fitted=true (12)`, `dg0_dg1_routing fitted=true (12)`, `verdict` giữ 0.893.
- **Probe `/v1/predict`** (`/tmp/vega_probe.out`): `AutoFit true` → engine UNSUPPORTED 0.45 (abstain) / **ttt CORRECT 0.97**; `AutoFit false` → **ttt INCORRECT 0.92**; `readout=ttt`, ~130ms.
- **Smoke `qwen35-muscle`** (`/tmp/smoke6.out`, exit 0): `/api/tags` có model; `/api/chat think=false` 7.0s → `'Hà Nội'`; tool-call 10.9s → `verdict=INCORRECT` **MATCH**. Phát hiện: qua `/v1/chat/completions` Qwen3.5 vẫn sinh `reasoning`, `content` rỗng khi `max_tokens` nhỏ → smoke/Muscle dùng `/api/chat` native.
- **DG3** `06_laya_summary_dg3_20261112T172646.json`: 2 ACCEPT + 3 FLAG → resolve (REF + test + `/health`) → **PASS**.

## D. Script benchmark (bản sao `/tmp/bench_ollama.py`)
`POST /api/chat` với `{"model":"qwen3.5:9b","stream":false,"think":<bool>,"options":{"temperature":0.2,"num_ctx":8192,"num_predict":N}}`; tính `tok/s = eval_count / (eval_duration/1e9)`. Output gốc `/tmp/bench_ollama.out`.
