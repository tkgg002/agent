# 08. Tasks — `vega-failfast-fallback-2026-11-18` (suffix `vega_failfast`)

| # | Task | Context | DoD | Trạng thái |
| :--- | :--- | :--- | :--- | :---: |
| 1 | Restart Vega độc lập qua `ensure_vega_up.sh`, kiểm `/v1/predict` < 3s | `ensure_vega_up.sh:20-72` | health 200 + predict < 3s | ☐ |
| 2 | Backup `laya_gate.py` rồi thêm fail-fast 3s + fallback Laya không retry Vega | `laya_gate.py:217-249` | code + test | ☐ |
| 3 | Chạy `tests/test_vega_primary_flywheel.py` + test mới qua `evidence_run.py` | tests/ | EV- exit 0 | ☐ |
| 4 | Chạy lại R5 `review-export-GP2-4633` r2 | skill diff-review | `06_laya_summary` PASS | ☐ |
