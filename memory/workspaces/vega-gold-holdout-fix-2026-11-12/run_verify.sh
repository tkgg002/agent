#!/bin/bash
# vega-gold-holdout-fix: compile -> tests -> rebuild gold by content -> restart 0.8b sidecar (boot refit + guard)
#   -> DG1/2 via gate -> linter.  Requires: no VEGA_SIZE (0.8b).  Log: /tmp/vg_fix.out
set -u
A=/Users/trainguyen/Documents/work/agent
O=/Users/trainguyen/Documents/work/ai-orchestrator
V=$O/.orchestrator/vega-venv/bin/python
SD=$O/sidecar/vegaml
DATA=$O/.orchestrator/vega-data
WS=vega-gold-holdout-fix-2026-11-12
D=$A/memory/workspaces/$WS
OUT=/tmp/vg_fix.out
: > $OUT
log(){ echo "== $*" | tee -a $OUT; }
health(){ curl -s -m 5 http://127.0.0.1:8222/health | python3 -c '
import sys,json;d=json.load(sys.stdin);f=d["fit"]
print("model",d.get("model"),"| fit",f["status"],"n",f.get("num_examples"),"held_out_gold",f.get("held_out_gold"),"gold_rows",f.get("gold_rows"),"elapsed",f.get("elapsed_sec"))
for k,v in (f.get("report") or {}).items(): print(" ",k,"fitted",v.get("fitted"),"cv",v.get("cv_accuracy"),"auto",v.get("used_by_auto"))
g=f.get("guard") or {}; print("guard gold_n",g.get("gold_n"),"scores",json.dumps(g.get("scores")),"verdicts",json.dumps(g.get("verdicts")),"dropped",g.get("dropped_heads"))' 2>&1; }

log "0. backups (pre-fix originals are in $D/backup_pre_fix/; also snapshot current gold + fit_state)"
cp $DATA/gold_set.jsonl $DATA/gold_set.jsonl.bak_$(date +%Y%m%dT%H%M) 2>/dev/null; cp $DATA/fit_state.json $DATA/fit_state.json.bak_$(date +%Y%m%dT%H%M) 2>/dev/null
ls $DATA/*.bak_* 2>/dev/null | tail -3 | tee -a $OUT

log "1. py_compile"
$V -m py_compile $SD/vega_fit_policy.py $SD/vega_build_gold_set.py $SD/vega_replay_ground_truth.py $SD/vega_systemone_sidecar.py $SD/test_vega_fit_policy.py $SD/test_vega_replay_ground_truth.py 2>&1 | tee -a $OUT
echo "compile_exit=${PIPESTATUS[0]}" | tee -a $OUT

log "2. sidecar unit tests (expect 21 + 4 new = 25)"
(cd $SD && $V -m unittest test_vega_fit_policy test_vega_replay_ground_truth test_vega_sidecar 2>&1 | tail -6) | tee -a $OUT

log "3. rebuild gold set by content (dry-run shows dedupe stats, then --apply)"
(cd $SD && $V vega_build_gold_set.py --size 60 --seed 42 2>&1 | tail -20) | tee -a $OUT
(cd $SD && $V vega_build_gold_set.py --size 60 --seed 42 --apply 2>&1 | tail -2) | tee -a $OUT
python3 - $DATA <<'EOF' 2>&1 | tee -a $OUT
import json,sys,hashlib
D=sys.argv[1]
sk=lambda s: hashlib.sha1(json.dumps(s,sort_keys=True,ensure_ascii=False,separators=(",",":")).encode()).hexdigest()
rows=[json.loads(l) for l in open(f"{D}/human_gate_feedback.jsonl") if l.strip()]
gold=[json.loads(l) for l in open(f"{D}/gold_set.jsonl") if l.strip()]
gk={g["state_key"] for g in gold}
held=sum(1 for r in rows if sk(r["state"]) in gk)
print("rows",len(rows),"gold",len(gold),"unique gold keys",len(gk),"rows held out by content",held,"=> expected fit n",len(rows)-held)
EOF

log "4. sidecar must be 0.8b (no VEGA_SIZE). current:"
health | head -1 | tee -a $OUT
MODEL=$(curl -s -m 5 http://127.0.0.1:8222/health | python3 -c 'import sys,json;print(json.load(sys.stdin).get("model"))' 2>/dev/null)
if [ "$MODEL" != "vegaml-0.8b" ]; then
  echo "sidecar is '$MODEL' -> restarting as 0.8b" | tee -a $OUT
  PID=$(cat $DATA/vega_sidecar.pid 2>/dev/null); [ -n "$PID" ] && kill $PID 2>/dev/null; sleep 3
  (cd $O && VEGA_SIZE= bash sidecar/vegaml/ensure_vega_up.sh) 2>&1 | tail -2 | tee -a $OUT
else
  echo "restarting 0.8b to pick up new sidecar code" | tee -a $OUT
  PID=$(cat $DATA/vega_sidecar.pid 2>/dev/null); [ -n "$PID" ] && kill $PID 2>/dev/null; sleep 3
  (cd $O && bash sidecar/vegaml/ensure_vega_up.sh) 2>&1 | tail -2 | tee -a $OUT
fi
echo "waiting boot refit (~6-8 min)..." | tee -a $OUT
for i in $(seq 1 90); do
  S=$(curl -s -m 3 http://127.0.0.1:8222/health | python3 -c 'import sys,json;print(json.load(sys.stdin)["fit"]["status"])' 2>/dev/null)
  [ "$S" = "fitted" ] && break; sleep 10
done
log "5. AFTER (compare with before: n=300, ttt 0.5417 / engine 0.4583 on 48)"
health | tee -a $OUT

log "6. DG1+DG2 for this workspace via gate (Vega primary)"
(cd $A && python3 tooling/laya_gate.py run --stage dg12 --workspace $WS --claims $D/06_laya_claims_dg12_vega_gold_holdout_fix_2026_11_12.json \
  --objective 'Gold hold-out by content; unique replay ids' 2>&1 | tail -3) | tee -a $OUT
S=$(ls -t $D/06_laya_summary_dg12_*.json 2>/dev/null | head -1)
[ -n "$S" ] && python3 - "$S" <<'EOF' 2>&1 | tee -a $OUT
import sys,json
d=json.load(open(sys.argv[1])); print('== DG12 | gate',d['gate'],'| engine',d.get('engine_url'),'| trusted',d.get('trusted_rounds'))
for c in d['claims']:
    r=c['rounds']; print(f"  {c['id']:<12}{c['kind']:<14}{c['decision']:<8}", ' '.join(f"{k}={v.get('choice')}/{v.get('confidence',0):.2f}" for k,v in r.items()), (c.get('evidence_error') or '')[:60])
EOF

log "7. linter (S3 stage; audit_report comes after results)"
(cd $A && python3 tooling/verify_governance.py --workspace $WS --stage S3 2>&1 | tail -4) | tee -a $OUT
log "DONE -> $OUT"
