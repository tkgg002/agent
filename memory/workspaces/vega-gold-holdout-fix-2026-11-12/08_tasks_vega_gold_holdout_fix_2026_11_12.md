# 08 Tasks — vega_gold_holdout_fix_2026_11_12

- [ ] T1 `vega_fit_policy.py`: `state_key()`, `gold_state_keys()`, `exclude_gold()` theo state_key (giữ chữ ký cũ tương thích test).
- [ ] T2 `vega_build_gold_set.py`: dedupe theo state_key, loại nhãn mâu thuẫn, ghi `state_key`.
- [ ] T3 `vega_replay_ground_truth.py:50`: task_id thêm stem file claims.
- [ ] T4 sidecar `run_ttt_fit_sync`: dùng state_key; `held_out_gold` = số dòng thật bị loại.
- [ ] T5 tests (fit_policy +3, replay +1); py_compile; regression 21 test sidecar.
- [ ] T6 runtime (USER chạy, sau rollback 0.8b): rebuild gold → fit → guard mới; DG1/DG2/DG3 workspace này qua Vega; linter.
