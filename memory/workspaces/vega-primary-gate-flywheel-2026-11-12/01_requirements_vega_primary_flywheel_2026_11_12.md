# 01 Requirements: Vega `:8222` làm Decision Engine chính cho `laya_gate.py` + khép kín Flywheel (hard-negative từ `evidence_run.py`)

tier: T2
suffix: vega_primary_flywheel_2026_11_12
date: 2026-11-12

## 1. Yêu cầu của User (nguyên văn)
> "mục tiêu tao làm vega là để thay laya... cho `laya_gate.py` dùng `:8222` mode=auto cho `verdict`, giữ `:8200` làm fallback, rồi nối `evidence_run.py` FAIL → hard-negative. Khi đó flywheel khép kín thật."

## 2. Yêu cầu kỹ thuật
- **R1** `laya_gate.py` mặc định gọi `http://127.0.0.1:8222/v1/predict` với `"mode": "auto"` cho mọi round `verdict`; nếu `:8222` không trả `/health` 200 → tự dùng `http://127.0.0.1:8200/v1/predict` (Laya) và in cảnh báo `GATE_FALLBACK` ra stderr; `--url` thủ công vẫn ưu tiên cao nhất. Summary JSON ghi `engine_url` thật đã dùng.
- **R2** `evidence_run.py` khi lệnh FAIL (`exit != 0`): tìm `06_laya_summary_dg{1,2,3}_*.json` mới nhất trong workspace, lấy claim `decision == ACCEPT` (không phải control) → POST `verdict=INCORRECT` tới `:8222/v1/systemone/feedback` (`auto_fit=false`, `stage=HARD_NEGATIVE_RUNTIME_FAIL`, `task_id=<EV-id>:<claim-id>`); ghi số mẫu đã gửi vào record `EV-*.json`. Fail-soft: sidecar down → ghi `hard_negatives: {"sent":0,"error":...}`, không đổi exit code.
- **R3** Không hardcode tên dự án/service (ADR-07); port cấu hình qua env `VEGA_GATE_URL`, `LAYA_FALLBACK_URL`, `VEGA_FEEDBACK_URL`.
- **R4** `tests/test_agent_core_f01_f08.py` hiện có vẫn PASS; thêm test cho `resolve_gate_url` (fallback) và `collect_hard_negatives` (không HTTP thật).

## 3. Hiện trạng VERIFIED
- `laya_gate.py:42` `DEFAULT_URL=:8200`; `:192-220 post_laya` payload chỉ `{state, questions}`; `:233` gọi trong `ask_round_mode_a`; `:405,615` dùng `args.url`.
- Laya `api.py:35-37` `PredictRequest(state: dict, questions: dict)` — pydantic mặc định bỏ qua field thừa → gửi `mode` không gây 422. `:8200/health` → **404** (không có), `/openapi.json` → 200.
- Vega `:8222/health` → 200; `/v1/predict` nhận `mode` (`sidecar.py:163`); `verdict` head cv 0.893 `auto=true`.
- `evidence_run.py:67-99 run()` ghi `EV-*.json`, return `proc.returncode`; không có hook nào về Vega.
- Hook `agy_guard.py:906` chỉ grep chuỗi `evidence_run.py --kind` → thêm flag không phá hook.

## 4. Ràng buộc từ bài học
- **L045** caller thật: `ask_round_mode_a` (`:233`), `review` path (`:615`), `cmd_run` (`:405`).
- **L046/G2**: User đã ra lệnh rõ → coi là APPROVE; vẫn chạy DG1/DG2 trước khi sửa.
- **L064**: không sửa sidecar, không sửa Laya, không đụng `ai-orchestrator` Go.
- **G5**: `tests/test_agent_core_f01_f08.py` phải PASS sau sửa.
