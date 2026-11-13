# Audit Report — vega_primary_flywheel_2026_11_12

tier: T2
suffix: vega_primary_flywheel_2026_11_12
date: 2026-11-12
status: S5 DONE (code + runtime verified) — S6 chờ run_s6_close.sh (resolve ×5, DG3, linter)

## Direct Summary
4 task APPROVE đã code + verify runtime thật (`/tmp/vp_s4.out`, `/tmp/vp_s4b.out`): **62/62 + 21/21 test PASS**; `laya_gate.py` chạy qua **Vega `:8222` mode=auto**, kill-switch về Laya `:8200` hoạt động; batch-fit + gold guard hoạt động; `evidence_run` FAIL → hard-negative POST OK, exit code giữ nguyên. **Phát hiện quan trọng:** gold guard **REJECT** head `verdict` (held-out 0.54 < 0.75) → cv 0.893 hôm qua là ảo do leak giữa workspace. Vega đã thay Laya về **kiến trúc**, chưa thay về **chất lượng** — cần dữ liệu Human Gate thật.

## 1. Phản biện từng dòng update (git diff) & từng task
| File | Thay đổi | Phản biện |
|---|---|---|
| `agent/tooling/laya_gate.py` (+~30, backup `.bak_20261112T1750`) | `VEGA_URL/LAYA_FALLBACK_URL/GATE_MODE` env; `_health_ok` (:197); `resolve_gate_url` (:207); `post_laya` gửi `mode` (:230); summary +`engine_url/engine_mode`; `--url` default None | ✅ `--url` explicit vẫn thắng → backward-compat. ✅ Laya `PredictRequest` bỏ qua `mode` (pydantic default). ⚠️ `_health_ok` probe mỗi lần `cmd_run` (+3s timeout max) — chấp nhận |
| `agent/tooling/hard_negative_feedback.py` (mới) | ACCEPT non-control của DG gần nhất → `verdict=INCORRECT`, `auto_fit=false` | ✅ fail-soft, không ghi jsonl trực tiếp. ⚠️ **Giả định**: FAIL runtime ⇒ claims ACCEPT sai. Không luôn đúng (fail do env). Đã giới hạn: chỉ `kind` không phải control, `auto_fit=false` → không tự refit |
| `agent/tooling/evidence_run.py:94-101` | gọi `feed_runtime_fail` khi exit≠0 | ✅ exit code không đổi (:107); `agy_guard.py` grep chuỗi `evidence_run.py` vẫn khớp |
| `sidecar/vegaml/vega_fit_policy.py` (mới) | `should_fit` batch 30; `exclude_gold`; `score_gold` exact `==`; `guard_verdicts` | ✅ pure functions, 8 test. ✅ Không rewrite jsonl (state riêng `fit_state.json`) |
| `vega_systemone_sidecar.py` | `run_ttt_fit_sync` hold-out gold + guard + `heads.pop` (:121-122); `/feedback` → `deferred` (:445-446); race `_last_loaded_rows` fix (:86,:131) | ✅ guard chạy **trong** `model_lock` ngay sau fit → không có cửa sổ head xấu phục vụ request |
| `vega_build_gold_set.py` (mới) | stratified, seed 42, round-robin workspace | ⚠️ Ra **48** không phải 60 (UNSUPPORTED chỉ 8 mẫu đủ evidence) |
| `vega_replay_ground_truth.py` | `iter_rejected_plan_examples`, `--only`, `--skip-fit` | ✅ 18 mẫu từ DG1 lần-đầu BLOCK thật, 0 bịa |
| `vega_human_gate.go:127-129` | giữ `AutoFit:true`, comment semantic mới | ✅ không rebuild Go |
| `hooks/claude/*` + `.claude/settings.json` | adapter Claude Code ↔ `agy_guard.dispatch()` | ✅ 7 test PASS; `agy_guard.py`, `.agents/hooks.json` không đổi |

## 2. Runtime thật (không suy diễn)
- DG1/DG2 summary: `engine_url=http://127.0.0.1:8222/v1/predict`, `engine_mode=auto` (`/tmp/vp_s4b.out:13,20`).
- Kill-switch: `GATE_FALLBACK: http://127.0.0.1:1/v1/predict unhealthy -> using http://127.0.0.1:8200/v1/predict` (`/tmp/vp_s4.out:70`).
- Boot refit: `n=300 held_out_gold=48`; `verdict cv 0.875 auto=True` → guard `ttt 0.5417 vs engine 0.4583 → REJECT (<0.75)` → `dropped ['verdict']` (`/tmp/vp_s4.out:39-43`).
- Batch: 18 REJECT rows → `new_since_fit=18 < 30` → không refit (`/tmp/vp_s4b.out:3`).
- Hard-negative: sink `:18222` nhận 2 item, jsonl thật 527→527 (`/tmp/vp_s4b.out:27-28`).
- Test: 62/62 agent, 21/21 sidecar (`/tmp/vp_s4.out:13-21`).

## 3. Findings
- **F-A (quan trọng):** cv 0.893 (fit) vs gold 0.54 (held-out) → leak: replay claims trùng/na ná giữa workspace nên cross-val lạc quan. Guard làm đúng. Engine 0.46 trên gold cũng thấp → bài "claim + REF code" khó với reader 0.8B.
- **F-B:** `fit n=300` ≠ 509−48=461 — **chưa giải thích**, script S6 bước 3 soi `vegaml` cap.
- **F-C:** Sau drop head, DG chấm bằng engine conf 0.41–0.49 < `DECISIVE_CONF 0.5` → nhiều FLAG → tốn resolve thủ công (mục tiêu "bỏ 15 phút resolve" **chưa đạt**).
- **F-D (lỗi em):** claims viết trước khi sửa code → REF lệch 3 chỗ; bước 10 verify chạy `false` trên workspace thật → 2 dòng nhiễu (đã xoá, backup `.bak_<ts>`).

## 4. Kiểm tra suy diễn & báo cáo láo (trung thực)
- Số em từng báo sai: cv **0.893** (phiên trước) → gold held-out thật **0.54**. Nguyên nhân: leak giữa workspace trong cross-val; em đã không giữ held-out trước khi báo. Ghi nhận là lỗi báo cáo lạc quan, không phải lỗi dữ liệu.
- Mọi số trong report này có REF về `/tmp/vp_s4.out`, `/tmp/vp_s4b.out`, `/tmp/vp_s6.out` (do USER chạy, em đọc file — terminal Cline hỏng suốt S4–S6).
- Chưa đo / chưa biết: nguyên nhân `fit n=300`; gold acc khi `VEGA_SIZE=4b`; latency DG khi head được KEEP.
- Mục tiêu "bớt 15 phút resolve mỗi DG": **CHƯA ĐẠT** — DG3 vẫn cần resolve 5/7 claim thủ công.

## 4b. Concurrency / Data Continuity
- Guard + fit đều trong `model_lock`; `/health` gọi `load_stored_training_data` → đã `clear()` + snapshot để không lệch index gold.
- `human_gate_feedback.jsonl` append-only; 2 lần backup trong task; xoá 2 dòng nhiễu bằng rewrite **có backup** (ngoại lệ ghi rõ ở 05_progress).

## 4c. Kết quả QC /laya_router (DG3 qua Vega `:8222`, `/tmp/vp_s6.out:13-20`)
- DG1 → **PASS**, DG2 → **PASS** (sau 5 resolve có REF). DG3 lần 1: `NEEDS_RESOLUTION` — ACCEPT A-GATE-01, A-FIT-02; FLAG 5 (A-GATE-02, A-FIT-01, A-FIT-03, A-HN-01, A-HN-02) do engine conf 0.43–0.50 < 0.5, không có BLOCK. Resolve trong `04_decisions` + `run_s6b_close.sh`.
- Engine đang là `engine` readout (head `verdict` bị guard drop) → chất lượng tương đương Laya; đây là **bằng chứng runtime** cho F-C.

## 5. Next (task riêng, theo thứ tự)
1. Soi F-B (cap 300) — nếu `vegaml.fit` cap, cần stratified sample thay vì cắt đầu.
2. Nâng chất lượng gold acc: (a) dữ liệu Human Gate thật qua `ai-orchestrator`; (b) thử `VEGA_SIZE=4b` (0 code change) rồi so gold acc engine/ttt.
3. Hạ `DECISIVE_CONF` **không** làm — thay vào đó cho `laya_gate` coi `abstain=false & readout=ttt` là decisive khi head được KEEP.

## 6. Lessons đề xuất (Global, chờ anh duyệt)
- "cv trên tập fit không phải accuracy: luôn giữ gold held-out theo *nguồn* (workspace), không theo dòng."
- "Viết claims DG SAU khi code đã ổn định; chạy `grep -n` lấy REF ngay trước DG."
- "Script verify không được tạo side-effect trên dữ liệu production (dùng sink/port tạm)."
