# 04 Decisions (Append-ONLY) — vega_primary_flywheel_2026_11_12

- [2026-11-12T18:05:00+07:00] [User] APPROVE 4 task: (1) batch-fit + gold guard trong sidecar, (2) laya_gate.py Vega :8222 primary / Laya :8200 fallback, (3) evidence_run FAIL → hard-negative, (4) REJECT examples từ lịch sử DG1.
- [2026-11-12T22:55:00+07:00] [Agent:Claude] DECISION: giữ Vega làm primary gate dù gold guard REJECT head `verdict` (ttt 0.54 < 0.75). Lý do: (a) kiến trúc fallback + kill-switch đã verify; (b) khi head bị drop, mode=auto tự về engine → không tệ hơn trước; (c) chỉ dữ liệu Human Gate thật mới nâng được gold acc — replay thêm chỉ tăng cv ảo. Guard là feature, không phải bug.
- [2026-11-12T22:56:00+07:00] [Agent:Claude] DECISION: KHÔNG hạ VEGA_GOLD_MIN_ACC để "cho qua". Ngưỡng 0.75 giữ nguyên; cho phép override bằng env khi có lý do ghi trong 04_decisions.
- [2026-11-12T22:57:00+07:00] [Agent:Claude] DECISION: `vega_human_gate.go` giữ `AutoFit:true` (semantic = eligible), sidecar quyết định batch — tránh sửa Go + rebuild binary trong task này.

## Resolve FLAG (DG1 20261112T225322 / DG2 20261112T225325 — engine Vega, confidence engine 0.41–0.49 dưới DECISIVE_CONF)
- S-RC-01 — REF: /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:43-43 `DEFAULT_URL = "http://127.0.0.1:8200/v1/predict"` — literal "8200" có trong dòng.
- S-FIX-02 — REF: /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:213-213 `print(f"GATE_FALLBACK: ...", file=sys.stderr)` — runtime: /tmp/vp_s4.out:70 in ra `GATE_FALLBACK: http://127.0.0.1:1/v1/predict unhealthy -> using http://127.0.0.1:8200/v1/predict`.
- S-IMPACT-01 — REF: /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:230-230 `payload = {"state": state, "questions": questions, "mode": mode}`; runtime: summary `engine_mode: "auto"` (/tmp/vp_s4b.out:13).
- S-PATTERN-01 — REF: /Users/trainguyen/Documents/work/agent/tooling/hard_negative_feedback.py:57-57 `"labels": {"verdict": "INCORRECT"},`; runtime: sink :18222 nhận 2 item labels INCORRECT (/tmp/vp_s4b.out:28 sent=2).
- D-BEHAV-01 — REF: /Users/trainguyen/Documents/work/agent/tooling/hard_negative_feedback.py:50-51 `if c.get("decision") != "ACCEPT" or str(c.get("kind","")).startswith("control_"): continue`; test `tests/test_vega_primary_flywheel.py` PASS (62/62, /tmp/vp_s4.out:13-15).
- [2026-11-12T22:59:38+07:00] [Agent:laya_gate] DG1 S-RC-01 FLAG resolved: REF: tooling/laya_gate.py:43-43 DEFAULT_URL = "http://127.0.0.1:8200/v1/predict"
- [2026-11-12T22:59:38+07:00] [Agent:laya_gate] DG1 S-FIX-02 FLAG resolved: REF: tooling/laya_gate.py:213-213 print(... file=sys.stderr); runtime /tmp/vp_s4.out:70 GATE_FALLBACK

## Resolve FLAG DG3 (20261112T230011 — engine conf 0.43–0.50)
- A-GATE-02 — REF: /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:230-230 `payload = {"state": state, "questions": questions, "mode": mode}`; runtime summary `engine_mode: "auto"`.
- A-FIT-01 — REF: /Users/trainguyen/Documents/work/ai-orchestrator/sidecar/vegaml/vega_systemone_sidecar.py:445-446 `fit_result = {"status": "deferred", ...}`; runtime: 18 feedback với auto_fit → không refit (/tmp/vp_s4b.out:3).
- A-FIT-03 — REF: /Users/trainguyen/Documents/work/ai-orchestrator/sidecar/vegaml/vega_fit_policy.py:128-128 `if _pred_str(...) == _label_str(labels[q]):`; test_vega_fit_policy 8/8 PASS.
- A-HN-01 — REF: /Users/trainguyen/Documents/work/agent/tooling/hard_negative_feedback.py:67-67 `method="POST"`; runtime sink :18222 `do_POST` nhận 2 item (/tmp/vp_s4b.out:28).
- A-HN-02 — REF: /Users/trainguyen/Documents/work/agent/tooling/evidence_run.py:107-107 `return proc.returncode`; runtime `EVIDENCE EV-... exit=1` với lệnh `false` (/tmp/vp_s4b.out:26).

- [2026-11-12T22:59:38+07:00] [Agent:laya_gate] DG1 S-IMPACT-01 FLAG resolved: REF: tooling/laya_gate.py:230-230 payload has "mode"; summary engine_mode=auto
- [2026-11-12T22:59:38+07:00] [Agent:laya_gate] DG1 S-PATTERN-01 FLAG resolved: REF: tooling/hard_negative_feedback.py:57-57 "labels": {"verdict": "INCORRECT"}; sink :18222 sent=2
- [2026-11-12T22:59:38+07:00] [Agent:laya_gate] DG2 D-BEHAV-01 FLAG resolved: REF: tooling/hard_negative_feedback.py:50-51 skip when decision!=ACCEPT or kind startswith control_; test PASS
- [2026-11-12T23:08:21+07:00] [Agent:laya_gate] DG3 A-GATE-02 FLAG resolved: REF: tooling/laya_gate.py:230-230 payload has "mode"; summary engine_mode=auto
- [2026-11-12T23:08:21+07:00] [Agent:laya_gate] DG3 A-HN-01 FLAG resolved: REF: tooling/hard_negative_feedback.py:67-67 method="POST"; sink :18222 do_POST sent=2
- [2026-11-12T23:08:21+07:00] [Agent:laya_gate] DG3 A-HN-02 FLAG resolved: REF: tooling/evidence_run.py:107-107 return proc.returncode; EVIDENCE exit=1 with false
- [2026-11-12T23:25:27+07:00] [Agent:laya_gate] DG3 A-FIT-01 FLAG resolved: REF: /Users/trainguyen/Documents/work/ai-orchestrator/sidecar/vegaml/vega_systemone_sidecar.py:445-446 status deferred; runtime new_since_fit=18 no refit
- [2026-11-12T23:25:27+07:00] [Agent:laya_gate] DG3 A-FIT-03 FLAG resolved: REF: /Users/trainguyen/Documents/work/ai-orchestrator/sidecar/vegaml/vega_fit_policy.py:128-128 exact ==; test_vega_fit_policy 8/8
