#!/bin/bash
# wrapup.sh — Trigger learning loop at the end of each session

set -e

echo "=== Starting Hermes Learning Loop Session Wrap-up ==="

# Define directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Verify Python scripts exist
SOP_PATCHER="$WORK_DIR/agent/tooling/learning_loop/sop_patcher.py"
SKILL_MINER="$WORK_DIR/agent/tooling/learning_loop/skill_miner.py"
RULE_PROMOTER="$WORK_DIR/agent/tooling/learning_loop/rule_promoter.py"

if [ ! -f "$SOP_PATCHER" ] || [ ! -f "$SKILL_MINER" ] || [ ! -f "$RULE_PROMOTER" ]; then
    echo "Error: Learning loop tooling scripts not found!"
    exit 1
fi

echo "1. Running SOP Patcher to check if any workflow patches are requested..."
# Currently, sop_patcher requires instruction from logs, which is processed on demand or mock tested.
# We run it with a dry run check.
python3 "$SOP_PATCHER" --check || echo "SOP Patcher skipped (no instruction found)."

echo ""
echo "2. Running Skill Miner to extract skills from active workspaces..."
# Skill miner parses progress log to generate SKILL.md under agent/skills/
python3 "$SKILL_MINER" --force || echo "Skill Miner warning: failed to mine skills."

echo ""
echo "3. Running Rule Promoter to check repeating tags in lessons.md..."
# Rule promoter automatically promotes tags repeating >= 3 times to GEMINI.md/CLAUDE.md rules
python3 "$RULE_PROMOTER" || echo "Rule Promoter warning: failed to promote rules."

echo ""
echo "=== Hermes Learning Loop Session Wrap-up Completed Successfully ==="
