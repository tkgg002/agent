# 09 Tasks Solution — vega_gold_holdout_fix_2026_11_12

tier: T2

## Root cause (VERIFIED /tmp/vp_s7.out:6-7)
`vega_replay_ground_truth.py` đặt `task_id = replay-claim-<ws>-<claim id>`; claim id `K1..K4`, `S-RC-01`… lặp qua dg1/dg2/dg3 cùng workspace → 1 id trùng tới 55 dòng. `exclude_gold` loại theo `task_id` → 209/527 dòng bị loại oan, gold **không** held-out thật.

## Phương án duy nhất: hold-out theo nội dung
| # | File | Diff |
|---|---|---|
| T1 | `vega_fit_policy.py` | `+import hashlib`; `state_key(state)` = sha1(canonical JSON); `gold_state_keys()`; `exclude_gold(..., gold_keys)` loại khi `state_key(row.state) in keys` **hoặc** `task_id in keys` (tương thích test cũ) |
| T2 | `vega_build_gold_set.py` | `dedupe_by_content()` — 1 dòng/state_key, **bỏ state có nhãn mâu thuẫn**; gold row ghi `state_key`; summary thêm `dedupe`, `rows_held_out_by_content` |
| T3 | `vega_replay_ground_truth.py:49-53` | `task_id = replay-claim-<ws>-<stem file claims>-<id>` |
| T4 | `vega_systemone_sidecar.py:132-136,156-157` | `gold_keys = gold_state_keys(...)`; `held_out_gold` = số dòng thật bị loại; `+gold_rows` |
| T5 | tests | fit_policy +3 (`state_key` bất biến thứ tự key; loại theo nội dung không theo id — regression cho vp_s7:6-7; dedupe/conflict); replay +1 (unique qua dg files) |

## Revert
Sidecar: đổi lại `gold_ids = fit_policy.gold_task_ids(...)`, `held_out_gold: len(gold_ids)`, bỏ `gold_rows`. 3 file còn lại: backup nguyên văn trong `backup_pre_fix/` của workspace này.

## Code Demo (trích thật)
```python
# vega_fit_policy.py
def state_key(state: Any) -> str:
    canon = json.dumps(state, sort_keys=True, ensure_ascii=False, separators=(",", ":"))
    return hashlib.sha1(canon.encode("utf-8")).hexdigest()

def exclude_gold(examples, feedback_rows, gold_keys):
    out = []
    for ex, row in zip(examples, feedback_rows):
        if state_key(row.get("state")) in gold_keys or str(row.get("task_id")) in gold_keys:
            continue
        out.append(ex)
    return out
```

## Kỳ vọng runtime (chưa đo — ghi để so)
- `build_gold_set` summary: `conflicting_states_dropped` > 0 có thể (claims trùng nội dung khác nhãn giữa workspace là nhiễu thật); `rows_held_out_by_content` ≈ gold + vài dòng trùng nội dung.
- `/health` sau fit: `num_examples` ≈ 527 − held_out (≫ 318); guard `verdict.ttt/engine` mới — **không đoán trước**; nếu ttt < 0.75 head vẫn bị drop, đó là kết quả thật.
