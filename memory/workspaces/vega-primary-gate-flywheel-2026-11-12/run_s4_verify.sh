#!/bin/bash
# S4/S5 verification for workspace vega-primary-gate-flywheel-2026-11-12 (4 approved tasks).
# Idempotent; every step logs to /tmp/vp_*.out. Run:  bash <this file>
set -u
A=/Users/trainguyen/Documents/work/agent
O=/Users/trainguyen/Documents/work/ai-orchestrator
V=$O/.orchestrator/vega-venv/bin/python
WS=vega-primary-gate-flywheel-2026-11-12
OUT=/tmp/vp_s4.out
: > $OUT
log(){ echo "== $*" | tee -a $OUT; }

log "1. py_compile"
python3 -m py_compile $A/tooling/laya_gate.py $A/tooling/evidence_run.py $A/tooling/hard_negative_feedback.py \
  $A/tests/test_vega_primary_flywheel.py $A/hooks/claude/claude_hook_adapter.py $A/hooks/claude/tests/test_claude_hook_adapter.py 2>&1 | tee -a $OUT
$V -m py_compile $O/sidecar/vegaml/vega_systemone_sidecar.py $O/sidecar/vegaml/vega_fit_policy.py \
  $O/sidecar/vegaml/vega_build_gold_set.py $O/sidecar/vegaml/vega_replay_ground_truth.py \
  $O/sidecar/vegaml/test_vega_fit_policy.py $O/sidecar/vegaml/test_vega_replay_ground_truth.py $O/sidecar/vegaml/test_vega_sidecar.py 2>&1 | tee -a $OUT
echo "compile_exit=$?" | tee -a $OUT

log "2. agent unit tests (new + regression)"
(cd $A && python3 -m unittest tests.test_vega_primary_flywheel tests.test_agent_core_f01_f08 hooks.claude.tests.test_claude_hook_adapter 2>&1 | tail -8) | tee -a $OUT
(cd $A && python3 -m unittest discover -s hooks/agy/tests -p 'test_*.py' 2>&1 | tail -4) | tee -a $OUT

log "3. sidecar unit tests"
(cd $O/sidecar/vegaml && $V -m unittest test_vega_fit_policy test_vega_replay_ground_truth test_vega_sidecar 2>&1 | tail -6) | tee -a $OUT

log "4. gold set (dry-run then apply, 60 rows held out)"
(cd $O/sidecar/vegaml && $V vega_build_gold_set.py --size 60 --seed 42 --apply 2>&1 | tail -8) | tee -a $OUT

log "5. REJECT examples dry-run (task 4)"
(cd $O/sidecar/vegaml && $V vega_replay_ground_truth.py --agent-ws-root $A/memory/workspaces --only rejected 2>&1 | tail -8) | tee -a $OUT

log "6. restart sidecar with new code (batch fit + gold guard)"
PID=$(cat $O/.orchestrator/vega-data/vega_sidecar.pid 2>/dev/null); [ -n "$PID" ] && kill $PID 2>/dev/null; sleep 3
(cd $O && bash sidecar/vegaml/ensure_vega_up.sh) 2>&1 | tail -2 | tee -a $OUT
echo "waiting boot refit (~7 min)..." | tee -a $OUT
for i in $(seq 1 60); do
  S=$(curl -s -m 3 http://127.0.0.1:8222/health | python3 -c 'import sys,json;print(json.load(sys.stdin)["fit"]["status"])' 2>/dev/null)
  [ "$S" = "fitted" ] && break; sleep 10
done
curl -s -m 5 http://127.0.0.1:8222/health | python3 -c '
import sys,json;d=json.load(sys.stdin);f=d["fit"]
print("fit",f["status"],"n",f.get("num_examples"),"held_out_gold",f.get("held_out_gold"),"elapsed",f.get("elapsed_sec"))
for k,v in (f.get("report") or {}).items(): print(" ",k,"fitted",v.get("fitted"),"cv",v.get("cv_accuracy"),"auto",v.get("used_by_auto"))
g=f.get("guard") or {}; print("guard gold_n",g.get("gold_n"),"verdicts",json.dumps(g.get("verdicts")),"dropped",g.get("dropped_heads"))' 2>&1 | tee -a $OUT

log "7. apply REJECT examples (skip fit; batch policy decides)"
(cd $O/sidecar/vegaml && $V vega_replay_ground_truth.py --agent-ws-root $A/memory/workspaces --only rejected --apply --skip-fit 2>&1 | tail -4) | tee -a $OUT
wc -l $O/.orchestrator/vega-data/human_gate_feedback.jsonl | tee -a $OUT

log "8. laya_gate.py real run via Vega (task 2) + DG1/DG2 for this workspace"
(cd $A && python3 tooling/laya_gate.py run --stage dg1 --workspace $WS \
  --claims memory/workspaces/$WS/06_laya_claims_dg1_vega_primary_flywheel_2026_11_12.json \
  --objective 'Vega primary gate; hard-negative flywheel' 2>&1 | tail -3) | tee -a $OUT
(cd $A && python3 tooling/laya_gate.py run --stage dg2 --workspace $WS \
  --claims memory/workspaces/$WS/06_laya_claims_dg2_vega_primary_flywheel_2026_11_12.json \
  --objective 'Code demo hard_negative_feedback + resolve_gate_url' 2>&1 | tail -3) | tee -a $OUT
S1=$(ls -t $A/memory/workspaces/$WS/06_laya_summary_dg1_*.json | head -1); S2=$(ls -t $A/memory/workspaces/$WS/06_laya_summary_dg2_*.json | head -1)
python3 -c "import json;[print(p.split('/')[-1],'gate',json.load(open(p))['gate'],'engine',json.load(open(p)).get('engine_url')) for p in ['$S1','$S2']]" | tee -a $OUT
python3 /tmp/show_summary.py $S1 $S2 2>/dev/null | tee -a $OUT

log "9. fallback kill-switch: VEGA_GATE_URL unreachable -> GATE_FALLBACK + :8200"
(cd $A && VEGA_GATE_URL=http://127.0.0.1:1/v1/predict python3 tooling/laya_gate.py run --stage dg2 --workspace $WS \
  --claims memory/workspaces/$WS/06_laya_claims_dg2_vega_primary_flywheel_2026_11_12.json --objective fallback-test 2>&1 | grep -E 'GATE_FALLBACK|"gate"' | head -3) | tee -a $OUT

log "10. evidence_run FAIL -> hard-negative into :8222 (task 3)"
BEFORE=$(wc -l < $O/.orchestrator/vega-data/human_gate_feedback.jsonl)
(cd $A && python3 tooling/evidence_run.py --workspace $WS --kind unit -- false 2>&1 | tail -2) | tee -a $OUT
AFTER=$(wc -l < $O/.orchestrator/vega-data/human_gate_feedback.jsonl)
echo "jsonl lines before=$BEFORE after=$AFTER" | tee -a $OUT
EV=$(ls -t $A/memory/workspaces/$WS/06_evidence/EV-*.json | head -1); python3 -c "import json;d=json.load(open('$EV'));print('EV',d['id'],'exit',d['exit'],'hard_negatives',d.get('hard_negatives'))" | tee -a $OUT

log "DONE -> $OUT"
