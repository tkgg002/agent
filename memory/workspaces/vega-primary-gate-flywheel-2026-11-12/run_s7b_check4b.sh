#!/bin/bash
# S7b: is the 4b sidecar alive / downloaded / fitted?  No restart here. Decide after reading /tmp/vp_s7b.out
set -u
O=/Users/trainguyen/Documents/work/ai-orchestrator
DATA=$O/.orchestrator/vega-data
HF=$O/.orchestrator/vega-hf
OUT=/tmp/vp_s7b.out
: > $OUT
log(){ echo "== $*" | tee -a $OUT; }

log "1. process"
PID=$(cat $DATA/vega_sidecar.pid 2>/dev/null); echo "pid=$PID" | tee -a $OUT
ps -o pid,rss,%cpu,etime,command -p "$PID" 2>/dev/null | tail -1 | awk '{printf "pid %s RSS %.2f GB cpu %s%% etime %s\n",$1,$2/1048576,$3,$4}' | tee -a $OUT
lsof -nP -iTCP:8222 -sTCP:LISTEN 2>/dev/null | tail -1 | tee -a $OUT

log "2. health (5s timeout)"
curl -s -m 5 http://127.0.0.1:8222/health | python3 -c '
import sys,json;d=json.load(sys.stdin);f=d["fit"]
print("model",d.get("model"),"device",d.get("device"),"ready",d.get("model_ready", d.get("ready")),"| fit",f["status"],"n",f.get("num_examples"),"elapsed",f.get("elapsed_sec"))
g=f.get("guard") or {}; print("guard",json.dumps(g.get("scores")),json.dumps(g.get("verdicts")),"dropped",g.get("dropped_heads"))' 2>&1 | tee -a $OUT

log "3. 4B weights on disk?"
ls -d $HF/hub/models--Qwen--Qwen3.5-4B* 2>/dev/null | tee -a $OUT
du -sh $HF/hub/models--Qwen--Qwen3.5-4B* 2>/dev/null | tee -a $OUT
find $HF/hub -name '*.incomplete' 2>/dev/null | head -5 | tee -a $OUT
ls -d $HF/hub/models--nandakishorm--vega-U* 2>/dev/null | tee -a $OUT; du -sh $HF/hub/models--nandakishorm--vega-U* 2>/dev/null | tee -a $OUT

log "4. sidecar log tail (errors / progress)"
grep -v 'GET /health' $DATA/vega_sidecar.log | tail -25 | tee -a $OUT

log "5. memory"
vm_stat | grep -E 'Pages free|Pages inactive|Pageouts|Swapins|Swapouts' | tee -a $OUT
sysctl -n vm.swapusage | tee -a $OUT

log "6. predict probe (30s) — is /v1/predict answering at all?"
START=$(date +%s)
curl -s -m 30 -X POST http://127.0.0.1:8222/v1/predict -H 'Content-Type: application/json' \
 -d '{"state":{"claim":"DEFAULT_URL points to port 8200.","evidence":"DEFAULT_URL = \"http://127.0.0.1:8200/v1/predict\""},"questions":{"verdict":{"type":"choice","options":["CORRECT","INCORRECT","UNSUPPORTED"],"instructions":"Does the evidence support the claim?"}},"mode":"auto"}' \
 | python3 -c 'import sys,json;d=json.load(sys.stdin);a=d.get("result",d).get("answers",{}).get("verdict",{});print("verdict",a.get("choice"),a.get("confidence"),"readout",a.get("readout"),"abstain",a.get("abstain"))' 2>&1 | tee -a $OUT
echo "probe took $(( $(date +%s) - START ))s" | tee -a $OUT
log "DONE -> $OUT"
