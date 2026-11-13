#!/bin/bash
# S6 close-out: resolve FLAGs via CLI (writes 04_decisions), DG3 via Vega, fit-cap probe, governance linter.
set -u
A=/Users/trainguyen/Documents/work/agent
O=/Users/trainguyen/Documents/work/ai-orchestrator
V=$O/.orchestrator/vega-venv/bin/python
WS=vega-primary-gate-flywheel-2026-11-12
D=$A/memory/workspaces/$WS
OUT=/tmp/vp_s6.out
: > $OUT
log(){ echo "== $*" | tee -a $OUT; }

log "1. resolve FLAG claims (REF thật, xem 04_decisions)"
cd $A
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg1 --id S-RC-01      --reason 'REF: tooling/laya_gate.py:43-43 DEFAULT_URL = "http://127.0.0.1:8200/v1/predict"' 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg1 --id S-FIX-02     --reason 'REF: tooling/laya_gate.py:213-213 print(... file=sys.stderr); runtime /tmp/vp_s4.out:70 GATE_FALLBACK' 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg1 --id S-IMPACT-01  --reason 'REF: tooling/laya_gate.py:230-230 payload has "mode"; summary engine_mode=auto' 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg1 --id S-PATTERN-01 --reason 'REF: tooling/hard_negative_feedback.py:57-57 "labels": {"verdict": "INCORRECT"}; sink :18222 sent=2' 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg2 --id D-BEHAV-01   --reason 'REF: tooling/hard_negative_feedback.py:50-51 skip when decision!=ACCEPT or kind startswith control_; test PASS' 2>&1 | tail -1 | tee -a $OUT
for st in dg1 dg2; do S=$(ls -t $D/06_laya_summary_${st}_*.json | head -1); python3 -c "import json;print('$st after resolve: gate', json.load(open('$S'))['gate'])" | tee -a $OUT; done

log "2. DG3 via Vega"
python3 tooling/laya_gate.py run --stage dg3 --workspace $WS --claims $D/06_laya_claims_dg3_vega_primary_flywheel_2026_11_12.json \
  --objective 'Audit: Vega primary + batch fit + gold guard + hard-negative' 2>&1 | tail -3 | tee -a $OUT
S3=$(ls -t $D/06_laya_summary_dg3_*.json | head -1)
python3 - "$S3" <<'EOF' 2>&1 | tee -a $OUT
import sys,json
d=json.load(open(sys.argv[1])); print('== DG3 | gate',d['gate'],'| engine',d.get('engine_url'),'| trusted',d.get('trusted_rounds'))
for c in d['claims']:
    r=c['rounds']; print(f"  {c['id']:<12}{c['kind']:<14}{c['decision']:<8}", ' '.join(f"{k}={v.get('choice')}/{v.get('confidence',0):.2f}" for k,v in r.items()), (c.get('evidence_error') or '')[:60])
EOF

log "3. why fit n=300 (grep vegaml source for caps)"
$V - <<'EOF' 2>&1 | tee -a $OUT
import inspect, re, vegaml
src = inspect.getsource(vegaml)
for i,l in enumerate(src.splitlines(),1):
    if re.search(r'300|max_examples|max_n|\[:\s*\w+\]|n_max|per_label', l): print(i, l.rstrip()[:120])
try:
    import vegaml.ttt as t
    for i,l in enumerate(inspect.getsource(t).splitlines(),1):
        if re.search(r'300|max_|cap|\[:', l): print('ttt', i, l.rstrip()[:120])
except Exception as e: print('ttt src', e)
EOF

log "4. governance linter"
python3 tooling/verify_governance.py --workspace $WS 2>&1 | tail -6 | tee -a $OUT
log "DONE -> $OUT"
