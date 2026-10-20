#!/usr/bin/env bash
# governance_metrics.sh — Tự đo KPI tuân thủ governance từ dữ liệu thật của repo (Antigravity Edition).
# In scorecard ra stdout + APPEND 1 snapshot có ngày vào trend-log append-only (memory/global/governance_metrics.md).

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
GLOBAL="$AGENT_DIR/memory/global"
WS="$AGENT_DIR/memory/workspaces"
CATALOG="$GLOBAL/lessons.md"
INDEX_MD="$GLOBAL/lessons/INDEX.md"
GEMINI="$AGENT_DIR/GEMINI.md"
HOOKS_JSON="$AGENT_DIR/.agents/hooks.json"
SKILLS_DIR="$AGENT_DIR/skills"
TREND="$GLOBAL/governance_metrics.md"
NOW="$(date '+%Y-%m-%d %H:%M')"
TODAY="$(date '+%Y-%m-%d')"

c() { grep -c "$@" 2>/dev/null || true; }

# ---------- A. Knowledge base (catalog = lessons.md + 2-tier INDEX.md) ----------
PATTERNS=$( [ -f "$CATALOG" ] && c '^### \[' "$CATALOG" || echo 0 )
H3_TOTAL=$( [ -f "$CATALOG" ] && c '^### ' "$CATALOG" || echo 0 )
INDEX_COUNT=$( [ -f "$INDEX_MD" ] && c '^- L[0-9]' "$INDEX_MD" || echo 0 )
INDEX_BYTES=$( [ -f "$INDEX_MD" ] && wc -c < "$INDEX_MD" | tr -d ' ' || echo 0 )
TAGS_DISTINCT=$( [ -f "$CATALOG" ] && grep -oE '#[a-z0-9][a-z0-9-]*' "$CATALOG" 2>/dev/null | sort -u | wc -l | tr -d ' ' || echo 0 )
CAT_FMT=0; [ "$H3_TOTAL" -gt 0 ] && CAT_FMT=$(( PATTERNS*100/H3_TOTAL ))

# ---------- B. Recidivism (lớp lỗi tái diễn theo tag, từ catalog) ----------
TOP_TAGS=$( [ -f "$CATALOG" ] && grep -oE '#[a-z0-9][a-z0-9-]*' "$CATALOG" 2>/dev/null | sort | uniq -c | sort -rn | head -8 | awk '{printf "%s(%s) ", $2,$1}' || echo "n/a" )

# ---------- C. Process compliance (workspaces) ----------
WS_TOTAL=0; WS_PROGRESS=0; WS_MIN_T1=0
if [ -d "$WS" ]; then
  for d in "$WS"/*/; do
    [ -d "$d" ] || continue
    WS_TOTAL=$((WS_TOTAL+1))
    ls "${d}"05_progress*.md >/dev/null 2>&1 && WS_PROGRESS=$((WS_PROGRESS+1))
    if ls "${d}"01_requirements*.md >/dev/null 2>&1 && ls "${d}"05_progress*.md >/dev/null 2>&1 && ls "${d}"08_tasks*.md >/dev/null 2>&1; then
      WS_MIN_T1=$((WS_MIN_T1+1))
    fi
  done
fi
PROGRESS_PCT=0; [ "$WS_TOTAL" -gt 0 ] && PROGRESS_PCT=$(( WS_PROGRESS*100/WS_TOTAL ))

# ---------- D. Antigravity Governance Infra ----------
RULES_G=$( [ -f "$GEMINI" ] && c -E '^### [0-9]+\.' "$GEMINI" || echo 0 )
GEMINI_BYTES=$( [ -f "$GEMINI" ] && wc -c < "$GEMINI" | tr -d ' ' || echo 0 )
SKILLS_COUNT=$( find "$SKILLS_DIR" -maxdepth 2 -name "SKILL.md" 2>/dev/null | wc -l | tr -d ' ' )
HOOK_EVENTS=$( [ -f "$HOOKS_JSON" ] && python3 -c "import json; d=json.load(open('$HOOKS_JSON')); print(len(d.get('hooks',{})))" 2>/dev/null || echo 0 )

# ---------- health flags ----------
flag() { [ "$1" = "1" ] && printf "✓" || printf "⚠"; }
F_GEMINI=$( [ "$RULES_G" -eq 23 ] && [ "$GEMINI_BYTES" -le 12000 ] && echo 1 || echo 0 )
F_HOOKS=$( [ "$HOOK_EVENTS" -ge 4 ] && echo 1 || echo 0 )
F_INDEX=$( [ "$INDEX_COUNT" -eq "$PATTERNS" ] && [ "$INDEX_BYTES" -lt 15000 ] && echo 1 || echo 0 )
F_FMT=$( [ "$CAT_FMT" -ge 95 ] && echo 1 || echo 0 )

cat <<EOF
════════════════════════════════════════════════
 GOVERNANCE METRICS (ANTIGRAVITY) — $NOW
════════════════════════════════════════════════
A. KNOWLEDGE BASE
   • Global Patterns (lessons.md) : $PATTERNS
   • 2-Tier Index (INDEX.md)      : $INDEX_COUNT entries (${INDEX_BYTES}B < 15000B)  $(flag $F_INDEX)
   • Distinct tags                : $TAGS_DISTINCT
   • Catalog format-compliance    : ${CAT_FMT}%  $(flag $F_FMT)  (### [date] / tổng ###)

B. RECIDIVISM — lớp lỗi tái diễn nhiều nhất (tag • số lần)
   $TOP_TAGS

C. PROCESS COMPLIANCE
   • Workspaces                   : $WS_TOTAL
   • Có 05_progress*.md           : $WS_PROGRESS  (${PROGRESS_PCT}%)
   • Đạt chuẩn tối thiểu T1+      : $WS_MIN_T1

D. ANTIGRAVITY INFRA
   • GEMINI.md                    : $RULES_G rules (${GEMINI_BYTES}B <= 12000B)  $(flag $F_GEMINI)
   • Core Skills (SKILL.md)       : $SKILLS_COUNT
   • Active Hook Events           : $HOOK_EVENTS (.agents/hooks.json)  $(flag $F_HOOKS)

HEALTH: gemini $(flag $F_GEMINI)  hooks $(flag $F_HOOKS)  index $(flag $F_INDEX)  catalog-fmt $(flag $F_FMT)
════════════════════════════════════════════════
EOF

# ---------- append trend snapshot (append-only) ----------
if [ -w "$GLOBAL" ] || [ -w "$TREND" ]; then
  if [ ! -f "$TREND" ]; then
    printf '# governance_metrics.md — Trend Log (APPEND-ONLY)\n\n> Sinh bởi `agent/tooling/governance_metrics.sh`. Mỗi lần chạy APPEND 1 snapshot.\n' >> "$TREND" 2>/dev/null || true
  fi
  {
    printf '\n## [%s] snapshot (Antigravity)\n' "$TODAY"
    printf -- '- patterns=%s index_entries=%s index_bytes=%s tags=%s catalog_fmt=%s%%\n' "$PATTERNS" "$INDEX_COUNT" "$INDEX_BYTES" "$TAGS_DISTINCT" "$CAT_FMT"
    printf -- '- workspaces=%s progress_compliance=%s%% t1_plus=%s\n' "$WS_TOTAL" "$PROGRESS_PCT" "$WS_MIN_T1"
    printf -- '- rules_gemini=%s gemini_bytes=%s skills=%s hook_events=%s\n' "$RULES_G" "$GEMINI_BYTES" "$SKILLS_COUNT" "$HOOK_EVENTS"
    printf -- '- recidivism_top: %s\n' "$TOP_TAGS"
  } >> "$TREND" 2>/dev/null && echo "→ Snapshot appended to $TREND" || echo "→ (Sandbox read-only mode: skipped appending to $TREND)"
fi
