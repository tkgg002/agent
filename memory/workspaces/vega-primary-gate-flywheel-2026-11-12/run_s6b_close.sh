#!/bin/bash
# S6b: resolve DG3 FLAGs, fit-cap probe (deeper), linter with stderr captured.
set -u
A=/Users/trainguyen/Documents/work/agent
O=/Users/trainguyen/Documents/work/ai-orchestrator
V=$O/.orchestrator/vega-venv/bin/python
WS=vega-primary-gate-flywheel-2026-11-12
D=$A/memory/workspaces/$WS
OUT=/tmp/vp_s6b.out
: > $OUT
log(){ echo "== $*" | tee -a $OUT; }
cd $A

log "1. resolve DG3 FLAGs"
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg3 --id A-GATE-02 --reason 'REF: tooling/laya_gate.py:230-230 payload has "mode"; summary engine_mode=auto' 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg3 --id A-FIT-01  --reason 'REF: sidecar/vegaml/vega_systemone_sidecar.py:445-446 status deferred; runtime new_since_fit=18 no refit' 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg3 --id A-FIT-03  --reason 'REF: sidecar/vegaml/vega_fit_policy.py:128-128 exact ==; test_vega_fit_policy 8/8' 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg3 --id A-HN-01   --reason 'REF: tooling/hard_negative_feedback.py:67-67 method="POST"; sink :18222 do_POST sent=2' 2>&1 | tail -1 | tee -a $OUT
python3 tooling/laya_gate.py resolve --workspace $WS --stage dg3 --id A-HN-02   --reason 'REF: tooling/evidence_run.py:107-107 return proc.returncode; EVIDENCE exit=1 with false' 2>&1 | tail -1 | tee -a $OUT
S3=$(ls -t $D/06_laya_summary_dg3_*.json | head -1); python3 -c "import json;print('dg3 after resolve: gate', json.load(open('$S3'))['gate'])" | tee -a $OUT

log "2. fit n=300: where is the cap? (search whole vegaml package for literal 300 / subsample)"
$V - <<'EOF' 2>&1 | tee -a $OUT
import vegaml, pathlib, re
root = pathlib.Path(vegaml.__file__).parent
for p in sorted(root.rglob("*.py")):
    for i, l in enumerate(p.read_text(encoding="utf-8", errors="ignore").splitlines(), 1):
        if re.search(r'\b300\b|max_examples|max_train|subsample|n_train|\[:\s*n\b|random\.sample|rng\.choice', l):
            print(f"{p.name}:{i}: {l.strip()[:110]}")
EOF
echo "--- sidecar env ---" | tee -a $OUT
grep -n 'MAX_FIT\|FIT_MAX\|300\|MIN_FIT_EXAMPLES =' $O/sidecar/vegaml/vega_systemone_sidecar.py | tee -a $OUT

log "3. governance linter (stdout+stderr)"
python3 tooling/verify_governance.py --workspace $WS 2>&1 | tee -a $OUT
log "DONE -> $OUT"
