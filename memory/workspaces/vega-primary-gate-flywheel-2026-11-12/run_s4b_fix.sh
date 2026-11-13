#!/bin/bash
# S4b: fix-up after run_s4_verify.sh (log /tmp/vp_s4.out).
#  (a) remove the 2 noise rows that step 10 injected (test cmd `false` on the REAL workspace -> correct claims labelled INCORRECT)
#  (b) explain fit n=300 (vegaml cap?)   (c) re-run DG1/DG2 with corrected REFs   (d) re-test step 10 against a throwaway port (no real data touched)
set -u
A=/Users/trainguyen/Documents/work/agent
O=/Users/trainguyen/Documents/work/ai-orchestrator
V=$O/.orchestrator/vega-venv/bin/python
WS=vega-primary-gate-flywheel-2026-11-12
J=$O/.orchestrator/vega-data/human_gate_feedback.jsonl
OUT=/tmp/vp_s4b.out
: > $OUT
log(){ echo "== $*" | tee -a $OUT; }

log "a. remove noise rows (stage HARD_NEGATIVE_RUNTIME_FAIL, task_id EV-20261112T220501-8b8d:*) — backup first"
cp $J $J.bak_$(date +%Y%m%dT%H%M)
python3 - "$J" <<'EOF' 2>&1 | tee -a $OUT
import sys,json
p=sys.argv[1]; rows=[l for l in open(p,encoding='utf-8') if l.strip()]
keep=[l for l in rows if not ('"HARD_NEGATIVE_RUNTIME_FAIL"' in l and '"EV-20261112T220501-8b8d:' in l)]
open(p,'w',encoding='utf-8').write(''.join(keep))
print('rows',len(rows),'->',len(keep),'removed',len(rows)-len(keep))
EOF
# keep fit_state consistent (last_fit_lines must not exceed real line count)
python3 - "$O/.orchestrator/vega-data/fit_state.json" "$J" <<'EOF' 2>&1 | tee -a $OUT
import sys,json
st=json.load(open(sys.argv[1])); n=sum(1 for l in open(sys.argv[2]) if l.strip())
print('fit_state last_fit_lines',st['last_fit_lines'],'jsonl now',n,'new_since_fit',n-st['last_fit_lines'])
EOF

log "b. why fit n=300? inspect vegaml.fit signature / caps"
$V - <<'EOF' 2>&1 | tee -a $OUT
import inspect,vegaml,re
src=inspect.getsource(vegaml.Vega.fit) if hasattr(vegaml,'Vega') else inspect.getsource(vegaml.load().fit)
for i,l in enumerate(src.splitlines(),1):
    if re.search(r'max_|cap|\[:|300|limit|sample', l): print(i, l.rstrip())
EOF

log "c. DG1/DG2 re-run via Vega with corrected REFs"
(cd $A && python3 tooling/laya_gate.py run --stage dg1 --workspace $WS \
  --claims memory/workspaces/$WS/06_laya_claims_dg1_vega_primary_flywheel_2026_11_12.json \
  --objective 'Vega primary gate; hard-negative flywheel' 2>&1 | tail -3) | tee -a $OUT
(cd $A && python3 tooling/laya_gate.py run --stage dg2 --workspace $WS \
  --claims memory/workspaces/$WS/06_laya_claims_dg2_vega_primary_flywheel_2026_11_12.json \
  --objective 'Code demo hard_negative_feedback + resolve_gate_url' 2>&1 | tail -3) | tee -a $OUT
S1=$(ls -t $A/memory/workspaces/$WS/06_laya_summary_dg1_*.json | head -1); S2=$(ls -t $A/memory/workspaces/$WS/06_laya_summary_dg2_*.json | head -1)
python3 - "$S1" "$S2" <<'EOF' 2>&1 | tee -a $OUT
import sys,json
for p in sys.argv[1:]:
    d=json.load(open(p)); print('==',p.split('/')[-1],'| gate',d['gate'],'| engine',d.get('engine_url'),'| trusted',d.get('trusted_rounds'))
    for c in d['claims']:
        r=c['rounds']; print(f"  {c['id']:<14}{c['kind']:<14}{c['decision']:<8}", ' '.join(f"{k}={v.get('choice')}/{v.get('confidence',0):.2f}" for k,v in r.items()), (c.get('evidence_error') or '')[:60])
EOF

log "d. evidence_run FAIL -> hard-negative, THROWAWAY sink (VEGA_FEEDBACK_URL -> local dummy :18222), real jsonl untouched"
BEFORE=$(wc -l < $J)
python3 - <<'EOF' &
from http.server import BaseHTTPRequestHandler,HTTPServer
import json,sys
N=[0]
class H(BaseHTTPRequestHandler):
    def do_POST(s):
        n=int(s.headers.get('Content-Length',0)); b=json.loads(s.rfile.read(n)); N[0]+=1
        print('SINK got',b['task_id'],b['labels'],b['stage'],flush=True)
        s.send_response(200); s.end_headers(); s.wfile.write(b'{"status":"ok"}')
    def log_message(s,*a): pass
srv=HTTPServer(('127.0.0.1',18222),H); srv.timeout=1
import time; t=time.time()
while time.time()-t<40: srv.handle_request()
print('SINK total',N[0],flush=True)
EOF
SINK=$!
sleep 1
(cd $A && VEGA_FEEDBACK_URL=http://127.0.0.1:18222/v1/systemone/feedback python3 tooling/evidence_run.py --workspace $WS --kind unit -- false 2>&1 | tail -2) | tee -a $OUT
sleep 2
AFTER=$(wc -l < $J)
echo "real jsonl before=$BEFORE after=$AFTER (must be equal)" | tee -a $OUT
EV=$(ls -t $A/memory/workspaces/$WS/06_evidence/EV-*.json | head -1); python3 -c "import json;d=json.load(open('$EV'));print('EV',d['id'],'exit',d['exit'],'hard_negatives',d.get('hard_negatives'))" | tee -a $OUT
wait $SINK 2>/dev/null

log "e. health snapshot"
curl -s -m 5 http://127.0.0.1:8222/health | python3 -c '
import sys,json;d=json.load(sys.stdin);f=d["fit"]
print("fit",f["status"],"n",f.get("num_examples"),"held_out_gold",f.get("held_out_gold"))
for k,v in (f.get("report") or {}).items(): print(" ",k,"fitted",v.get("fitted"),"cv",v.get("cv_accuracy"),"auto",v.get("used_by_auto"))
g=f.get("guard") or {}; print("guard",json.dumps(g.get("verdicts")),"dropped",g.get("dropped_heads"))' 2>&1 | tee -a $OUT
log "DONE -> $OUT"
