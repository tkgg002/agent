#!/usr/bin/env bash
# wrapup.sh — Antigravity Session Wrap-up (S6)
# Usage: bash scripts/wrapup.sh --workspace <workspace_name>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Starting Antigravity Session Wrap-up (S6) ==="

echo "1. Rebuilding 2-tier Lessons Index (INDEX.md + topics/*.md)..."
python3 "$AGENT_DIR/tooling/build_lessons_index.py"

if [ $# -ge 1 ]; then
    echo ""
    echo "2. Running Process Governance Linter..."
    python3 "$AGENT_DIR/tooling/verify_governance.py" "$@"
else
    echo ""
    echo "2. (Skipped verify_governance.py — pass --workspace <name> or --tier T0 to run)"
fi

echo ""
echo "3. Collecting Governance Metrics..."
bash "$AGENT_DIR/tooling/governance_metrics.sh"

echo ""
echo "=== Session Wrap-up Completed Successfully ==="
