# 08 Tasks — vega_primary_flywheel_2026_11_12

tier: T2
suffix: vega_primary_flywheel_2026_11_12
date: 2026-11-12

### [ ] TASK-VP-01: `laya_gate.py` — Vega primary, Laya fallback (R1)
- Context: `laya_gate.py:42` DEFAULT_URL; `:192-220 post_laya`; `:405/:615 args.url`; `:775/:798 --url`.
- Thay đổi: thêm `VEGA_URL`, `LAYA_FALLBACK_URL`, `resolve_gate_url(explicit)`, `post_laya(..., mode="auto")` gửi `"mode"`; `--url` default `None`; summary thêm `engine_url`.
- Test: `tests/test_agent_core_f01_f08.py` PASS + test mới `resolve_gate_url`.
- DoD: `laya_gate.py run` thật trên workspace này → raw JSON có `"mode":"auto"` trong request, summary `engine_url` = `:8222`; kill-switch test: `VEGA_URL=http://127.0.0.1:1/...` → stderr `GATE_FALLBACK`, dùng `:8200`.

### [ ] TASK-VP-02: `evidence_run.py` — hard-negative khi FAIL (R2)
- Context: `evidence_run.py:67-99 run()`.
- Thay đổi: file mới `tooling/hard_negative_feedback.py` (đơn trách nhiệm): `collect_hard_negatives(ws_dir)` + `post_hard_negatives(items, url)`; `evidence_run.run()` gọi khi `exit != 0`, ghi `hard_negatives` vào `EV-*.json`.
- Test: unit test với workspace tạm + summary giả, không HTTP.
- DoD: chạy `evidence_run.py --kind unit -- false` trên workspace này → `EV-*.json` có `hard_negatives.sent >= 0`; `:8222` nhận dòng mới `stage=HARD_NEGATIVE_RUNTIME_FAIL` (xác nhận `wc -l` jsonl).

### [ ] TASK-VP-03: DG1/DG2 (trước sửa) → S4 → DG3 → audit_report → verify_governance PASSED.
