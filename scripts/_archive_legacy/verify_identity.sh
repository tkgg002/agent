#!/bin/bash
# Description: Identity Verification Script for Antigravity Brain/Muscle
# Usage: ./verify_identity.sh

SOURCE_FILE="./agent/models.env"

echo "--- IDENTITY VERIFICATION REPORT ---"
echo "Timestamp: $(date)"

# Check Environment Variables
echo "[System] Active Environment Models:"
env | grep -E "BRAIN_MODEL|MUSCLE_MODEL|ANTHROPIC_DEFAULT" | sed 's/^/  /'

# Check Model Pool Configuration
if [ -f "$SOURCE_FILE" ]; then
    echo "[Config] File: $SOURCE_FILE found."
    echo "[Config] Active Pools:"
    grep "_POOL=" "$SOURCE_FILE" | sed 's/^/  /'
else
    echo "[Config] models.env NOT FOUND."
fi

# Verification Logic
if [ ! -z "$BRAIN_MODEL" ]; then
    echo "[Result] BRAIN is confirmed as $BRAIN_MODEL"
else
    echo "[Result] BRAIN identity UNKNOWN (Defaults to system provider)"
fi

echo "------------------------------------"
