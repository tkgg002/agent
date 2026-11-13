# 01 Requirements — vega_gold_holdout_fix_2026_11_12

tier: T2
suffix: vega_gold_holdout_fix_2026_11_12
domain: ai-orchestrator/sidecar/vegaml (decision engine), agent/tooling (none)

## Vấn đề (REF thật, /tmp/vp_s7.out:6-7)
- Gold set 48 `task_id`, nhưng 28 id **trùng** trong feedback jsonl (1 id trùng 55 lần). `exclude_gold` loại theo `task_id` → loại 209/527 dòng → fit n=318 thay vì 479.
- Root cause: `vega_replay_ground_truth.py:50` `task_id = replay-claim-<ws>-<claim id>`; claim id `K1..K4` tái dùng qua dg1/dg2/dg3 cùng workspace → không unique.
- Hệ quả: gold **không thật sự held-out** (nội dung gold có thể còn trong fit dưới id khác; dòng vô tội bị loại). Số guard `ttt 0.54 / engine 0.46` chưa đáng tin.

## Yêu cầu
- R1: hold-out theo **hash nội dung `state`** (`state_key`), không theo `task_id`. REF: vega_fit_policy.py (mới `state_key`, `gold_state_keys`, `exclude_gold`).
- R2: gold builder khử trùng lặp nội dung và **loại mẫu có nhãn mâu thuẫn** (cùng state, khác verdict); ghi `state_key` vào gold. REF: vega_build_gold_set.py.
- R3: replay `task_id` unique theo file nguồn: `replay-claim-<ws>-<claims-file-stem>-<id>`. REF: vega_replay_ground_truth.py:50. Dòng cũ trong jsonl không sửa (append-only) — R1 làm id cũ vô hại.
- R4: sidecar dùng R1; `/health` báo `held_out_gold` = số dòng bị loại thật. REF: vega_systemone_sidecar.py run_ttt_fit_sync.
- R5: test cho R1–R3; không chạm jsonl thật trong test.
- R6: sau rollback 0.8b: rebuild gold (`--apply`) → `/fit` → so guard mới với 0.54/0.46; không hạ `VEGA_GOLD_MIN_ACC`.

## Ngoài phạm vi
- Không sửa `vega_human_gate.go`; không đổi schema jsonl; không thử 4b (đã đóng: swap 36GB, /tmp/vp_s7b.out:56).
