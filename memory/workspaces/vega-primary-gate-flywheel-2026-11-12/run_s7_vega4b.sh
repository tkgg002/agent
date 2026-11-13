#!/bin/bash
# S7: (0) fix 2 DG3 resolves (absolute REF)  (1) verify fit-cap hypothesis (task_id collisions)
#     (2) switch sidecar to VEGA_SIZE=4b, boot-refit + gold guard, compare   (3) DG3 re-run via 4b
# Rollback: bash <this> --rollback   (restarts sidecar without VEGA_SIZE)
set -u
A=/Users/trainguyen/Documents/work/agent
O=/Users/trainguyen/Documents/work/ai-orchestrator
V=$O/.orchestrator/vega-venv/bin/python
WS=vega-primary-gate-flywheel-2026-11-12
D=$A/memory/workspaces/$WS
DATA=$O/.orchestrator/vega-data
OUT=/tmp/vp_s7.out
log(){ echo "== $*" | tee -a $OUT; }
health(){ curl -s -m 5 http://127.0.0.1:8222/health | python3 -c '
import sys,json;d=json.load(sys.stdin);f=d["fit"]
print("model",d.get("model") or d.get("vega_model") or d.get("size"),"device",d.get("device"),"| fit",f["status"],"n",f.get("num_examples"),"held_out_gold",f.get("held_out_gold"),"elapsed",f.get("elapsed_sec"))
for k,v in (f.get("report") or {}).items(): print(" ",k,"fitted",v.get("fitted"),"cv",v.get("cv_accuracy"),"auto",v.get("used_by_auto"))
g=f.get("guard") or {}; print("guard gold_n",g.get("gold_n"),"scores",json.dumps(g.get("scores")),"verdicts",json.dumps(g.get("verdicts")),"dropped",g.get("dropped_heads"))' 2>&1; }
restart_sidecar(){  # $1 = VEGA_SIZE or ""
  PID=$(cat $DATA/vega_sidecar.pid 2>/dev/null); [ -n "$PID" ] && kill $PID 2>/dev/null; sleep 3
  (cd $O && VEGA_SIZE="$1" bash sidecar/vegaml/ensure_vega_up.sh) 2>&1 | tail -2 | tee -a $OUT
  echo "waiting load+refit (4b: download ~8GB first time, then fit; up to 40 min)..." | tee -a $OUT
  for i in $(seq 1 240); do
    S=$(curl -s -m 3 http://127.0.0.1:8222/health | python3 -c 'import sys,json;print(json.load(sys.stdin)["fit"]["status"])' 2>/dev/null)
    [ "$S" = "fitted" ] && break; sleep 10
  done
  health | tee -a $OUT
}

if [ "${1:-}" = "--rollback" ]; then
  : > $OUT; log "ROLLBACK -> 0.8b"; restart_sidecar ""; log "DONE"; exit 0
fi
: > $OUT

log "0. fix 2 DG3 resolves (absolute REF)"
cd $A
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg3 --id A-FIT-01 --reason "REF: $O/sidecar/vegaml/vega_systemone_sidecar.py:445-446 status deferred; runtime new_since_fit=18 no refit" 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg3 --id A-FIT-03 --reason "REF: $O/sidecar/vegaml/vega_fit_policy.py:128-128 exact ==; test_vega_fit_policy 8/8" 2>&1 | tail -1 | tee -a $OUT
S3=$(ls -t $D/06_laya_summary_dg3_*.json | head -1); python3 -c "import json;print('dg3 gate', json.load(open('$S3'))['gate'])" | tee -a $OUT

log "1. fit-cap hypothesis: gold task_ids collide (K1..K4 reused across dg1/dg2/dg3 of same ws) -> exclude_gold drops siblings"
python3 - $DATA <<'EOF' 2>&1 | tee -a $OUT
import json,sys,collections
D=sys.argv[1]
rows=[json.loads(l) for l in open(f"{D}/human_gate_feedback.jsonl") if l.strip()]
gold=[json.loads(l) for l in open(f"{D}/gold_set.jsonl") if l.strip()]
gid={str(g["task_id"]) for g in gold}
cnt=collections.Counter(str(r.get("task_id")) for r in rows)
dropped=sum(cnt[t] for t in gid)
print("rows",len(rows),"gold_ids",len(gid),"rows_dropped_by_exclude_gold",dropped,"=> fit n =",len(rows)-dropped)
dups={t:cnt[t] for t in gid if cnt[t]>1}
print("gold ids with duplicates:",len(dups),"examples:",list(dups.items())[:5])
EOF

log "2a. free GPU memory: unload Ollama model (4B reader on MPS needs ~8GB)"
curl -s -m 10 http://127.0.0.1:11434/api/generate -d '{"model":"qwen35-muscle","keep_alive":0}' >/dev/null 2>&1; sleep 2
curl -s -m 5 http://127.0.0.1:11434/api/ps | tee -a $OUT; echo | tee -a $OUT
vm_stat | grep -E 'Pages free|Pageouts' | tee -a $OUT

log "2b. BEFORE snapshot (0.8b)"
health | tee -a $OUT

log "2c. restart sidecar with VEGA_SIZE=4b"
restart_sidecar 4b
tail -5 $DATA/vega_sidecar.log | tee -a $OUT
ps -o rss,etime,command -p $(cat $DATA/vega_sidecar.pid) | tail -1 | awk '{printf "sidecar RSS %.1f GB\n",$1/1048576}' | tee -a $OUT
vm_stat | grep -E 'Pages free|Pageouts' | tee -a $OUT

log "3. DG3 via Vega 4b (same claims) — compare confidence vs 0.8b run 230011"
python3 tooling/laya_gate.py run --stage dg3 --workspace $WS --claims $D/06_laya_claims_dg3_vega_primary_flywheel_2026_11_12.json \
  --objective 'Audit (4b reader): Vega primary + batch fit + gold guard + hard-negative' 2>&1 | tail -3 | tee -a $OUT
S3=$(ls -t $D/06_laya_summary_dg3_*.json | head -1)
python3 - "$S3" <<'EOF' 2>&1 | tee -a $OUT
import sys,json
d=json.load(open(sys.argv[1])); print('== DG3(4b) | gate',d['gate'],'| trusted',d.get('trusted_rounds'))
for c in d['claims']:
    r=c['rounds']; print(f"  {c['id']:<12}{c['kind']:<14}{c['decision']:<8}", ' '.join(f"{k}={v.get('choice')}/{v.get('confidence',0):.2f}" for k,v in r.items()))
EOF
log "DONE -> $OUT   (rollback: bash $0 --rollback)"
