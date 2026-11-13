# backup_pre_fix — bản gốc trước task vega_gold_holdout_fix_2026_11_12

Terminal Cline hỏng nên không `cp` được trước khi sửa. Đoạn code gốc bị thay (nguyên văn, đủ để revert tay):

## vega_fit_policy.py (gốc, dòng 80-87 cũ)
```python
def gold_task_ids(gold_rows: List[Dict[str, Any]]) -> set:
    return {str(r.get("task_id")) for r in gold_rows if r.get("task_id")}


def exclude_gold(examples: List[Tuple[Any, Dict[str, Any]]], feedback_rows: List[Dict[str, Any]],
                 gold_ids: set) -> List[Tuple[Any, Dict[str, Any]]]:
    """examples[i] corresponds to feedback_rows[i]; drop those whose task_id is in the gold set."""
    return [ex for ex, row in zip(examples, feedback_rows) if str(row.get("task_id")) not in gold_ids]
```
(+ không có `import hashlib`; docstring module dòng 8-9 cũ: "They are EXCLUDED from fit (matched by task_id) and used to score the fitted heads.")

## vega_build_gold_set.py (gốc)
- Không có `from vega_fit_policy import state_key`, `Tuple`, hàm `dedupe_by_content`.
- `select_gold` không có dòng `cands, _ = dedupe_by_content(cands)`.
- `summary` cũ: `{"total_rows": len(rows), "gold": len(gold), "by_label": {...}, "workspaces": ...}`.

## vega_replay_ground_truth.py (gốc, dòng 49-50 cũ)
```python
            yield {
                "task_id": f"replay-claim-{Path(f).parent.name}-{c.get('id', '?')}",
```

## vega_systemone_sidecar.py (gốc, dòng 132-134 & 156 cũ)
```python
    gold_ids = fit_policy.gold_task_ids(fit_policy.load_gold(GOLD_FILE))
    if gold_ids:
        examples = fit_policy.exclude_gold(examples, rows_snapshot, gold_ids)
...
        "held_out_gold": len(gold_ids),
```

## test_vega_replay_ground_truth.py (gốc, dòng 42 cũ)
```python
        self.assertEqual(out[0]["task_id"], "replay-claim-ws-a-K1")
```
