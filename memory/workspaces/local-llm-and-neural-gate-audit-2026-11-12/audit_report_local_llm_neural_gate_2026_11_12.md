# Audit Report — local_llm_neural_gate_2026_11_12

tier: T2
suffix: local_llm_neural_gate_2026_11_12
date: 2026-11-12
status: DONE (S6)

## Direct Summary
(A) VegaML `:8222` đã học được từ ground-truth thật: `verdict` cv **0.592 → 0.893**, `used_by_auto=true`; hotfix nhãn bool giúp `c_req_check`/`dg0_dg1_routing` `fitted=true`. (B) Ollama + `qwen35-muscle` (Qwen3.5-9B) chạy local, đo thật 16 tok/s, đã đấu vào `models.env`. Tổng 5 file source/config thay đổi, 11/11 unit test PASS, DG1/DG2/DG3 PASS, 0 `git push`.

## 1. Phản biện từng dòng diff & từng task
| File | Thay đổi | Phản biện |
|---|---|---|
| `ai-orchestrator/sidecar/vegaml/vega_replay_ground_truth.py` (mới, 156 dòng) | Replay 437 control + 10 approved → `/feedback` (`auto_fit=false`) → `/fit` ×1; dry-run mặc định; backup `.bak_<ts>` | ✅ Không chạm sidecar; không ghi jsonl trực tiếp (sidecar sở hữu file). ⚠️ `task_id` dùng tên thư mục workspace — chỉ để truy vết, không ảnh hưởng fit |
| `test_vega_replay_ground_truth.py` (mới, 4 test) | Không HTTP, không chạm `vega-data` thật | ✅ `DATA_DIR` được override trong subprocess → 100% isolated |
| `vega_systemone_sidecar.py:65-74, :93` (+12 dòng, 406→418) | `to_fit_label()` bool → `"true"/"false"` khớp `vegaml.label_ids` | ✅ Root cause thật: `vegaml/__init__.py:447-451` `label_ids(boolean)=["false","true"]`; `ttt.py:48-57` raise khi label ngoài tập. `normalize_label_for_vega` giữ nguyên (còn dùng ở `/feedback` để ghi jsonl) |
| `test_vega_sidecar.py` (+2 test, sửa 1 assert `True`→`"true"`) | Assert cũ kiểm đúng kiểu bool — chính là bug | ✅ Test mới assert `to_fit_label(...) in vegaml.label_ids(spec)` |
| `agent/models.env:12,16-22` | Slot `qwen:qwen-2.5-coder:sk-muscle2` (chưa từng dùng) → `ollama:qwen35-muscle:local` + 4 env | ✅ `bash -n` OK; backup `.bak_20261112T1710`. Brain pool không đổi |
| `agent/tooling/local_llm/Modelfile.qwen35-9b-muscle`, `local_llm_smoke.py` (mới) | Modelfile think-off ctx16k; smoke qua `/v1` OpenAI-compatible | ⚠️ Smoke lần 1–3 timeout do tranh MPS với VegaML fit — đã tăng timeout 900s; lần 4 xem §2 |

## 2. Chống suy diễn Smoke Test & Trace Runtime thật
- **Fit lần 1** (`/tmp/vega_replay_apply.log`, 395.4s): `verdict` 497 mẫu cv 0.893 / engine 0.276 / `auto=true`; `c_req_check` **`fitted=false`** → phát hiện bug nhãn bool (không suy diễn từ test).
- **Sidecar restart** (PID 10181 → 11668 qua `ensure_vega_up.sh`), boot refit 361s: `c_req_check fitted=true (12)`, `dg0_dg1_routing fitted=true (12)`, `verdict` giữ 0.893.
- **Probe `/v1/predict` thật** (`/tmp/vega_probe.out`): claim `AutoFit true` → `mode=engine` UNSUPPORTED 0.45 abstain; `mode=ttt/auto` **CORRECT 0.97**; claim đảo `AutoFit false` → **INCORRECT 0.92**, `readout=ttt`, ~130ms. Đây là chính câu bẫy Laya `:8200` trượt 2 lần.
- **Ollama bench** (`/tmp/bench_ollama.out`): 16.3 tok/s gen, 22.3 tok/s tool-call, tool_calls JSON đúng; `/api/ps` runner mlx 8.17GB.
- **Chưa đo:** smoke `qwen35-muscle` end-to-end qua `/v1` — kết quả ghi ở §6 khi có.

## 3. Concurrency / Timeout / Data Continuity
- `model_lock` trong sidecar khóa `/v1/predict` suốt fit (361–395s). Trong phiên: Ollama warm-up + fit cùng lúc → smoke timeout. **Khuyến nghị (task riêng):** `AutoFit:true` (`vega_human_gate.go:127`) → batch; không chạy LLM local song song lúc fit.
- `human_gate_feedback.jsonl`: 51 → 509 dòng, append-only, backup `.bak_20261112T1704` trước apply; không có dòng nào bị mất (sidecar là writer duy nhất).
- Laya `:8200` (PID 17309) **không bị chạm**, `laya_gate.py` vẫn trỏ `:8200`.

## 4. Kiểm tra suy diễn / báo cáo láo
- Mọi số (cv, tok/s, dòng jsonl, PID) đều có file log hoặc lệnh tái hiện. Số **không có** em ghi rõ "chưa đo".
- Text User dán 4 lần: em fact-check và bác 7 điểm sai (Kev, deepseek-v4.1, q_name trùng, cv 0.53, `open("w")`, `:8200/fit` 404, so khớp `in`).
- Sai sót của em trong phiên: (1) REF `:176` thay vì `:175` → DG2 BLOCK, đã sửa; (2) smoke test so tên model thiếu `:latest`; (3) timeout smoke quá ngắn.

## 5. Self-Improvement Loop
- Lesson đề xuất (Global, chưa ghi vào `lessons.md` — chờ anh duyệt vì cần `build_lessons_index.py`): *"Nhãn huấn luyện phải là thành viên của tập option do thư viện khai báo (`label_ids`), không phải kiểu dữ liệu Python tương đương; assert trong test phải kiểm tra tư cách thành viên thay vì kiểu."*
- Tripwire: trước khi gọi `fit()` của bất kỳ thư viện ML nào, in `label_ids(question)` và `set(labels)` rồi so sánh.

## 6. Kết quả `/laya_router` DG3
- `06_laya_summary_dg3_20261112T172646.json`: trusted `[r3]`; ACCEPT A-R-01, A-SCOPE-01; FLAG A-F-01, A-F-02, A-R-02 (r2 CORRECT 0.50–0.71 nhưng r3 không quyết định) → `resolve` ×3 kèm `sed` + unit test + runtime `/health` (ghi `04_decisions`) → **PASS**.
- Ghi chú: Laya `:8200` yếu với claim "trả về chuỗi X" (3/5 FLAG); cùng claim đưa VegaML `:8222` mode=auto cho CORRECT 0.97. Đề xuất tách task: `laya_gate.py --url :8222` cho `verdict`.
- Smoke `qwen35-muscle` (lần 4): xem dòng cuối `05_progress`.
