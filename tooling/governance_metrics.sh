#!/usr/bin/env bash
# governance_metrics.sh — Tự đo KPI tuân thủ governance từ dữ liệu thật của repo.
# LAYOUT (cập nhật 2026-06-05): lessons.md = CATALOG chuẩn hoá (### [date], Rule 13);
#   ls_old.md = raw audit-log cũ archived. (Trước đây catalog nằm ở lessons_global_normalized.md.)
# In scorecard ra stdout + APPEND 1 snapshot có ngày vào trend-log append-only.
ROOT="${CLAUDE_PROJECT_DIR:-/Users/trainguyen/Documents/work}"
GLOBAL="$ROOT/agent/memory/global"
WS="$ROOT/agent/memory/workspaces"
CATALOG="$GLOBAL/lessons.md"          # catalog chuẩn hoá hiện hành
RAW="$GLOBAL/ls_old.md"               # raw audit-log archived (fallback nếu không có)
GEMINI="$ROOT/agent/GEMINI.md"
CLAUDEMD="$ROOT/CLAUDE.md"
SETTINGS="$ROOT/.claude/settings.json"
TREND="$GLOBAL/governance_metrics.md"
NOW="$(date '+%Y-%m-%d %H:%M')"; TODAY="$(date '+%Y-%m-%d')"

c() { grep -c "$@" 2>/dev/null || true; }

# ---------- A. Knowledge base (catalog = lessons.md) ----------
PATTERNS=$( [ -f "$CATALOG" ] && c '^### \[' "$CATALOG" || echo 0 )
H3_TOTAL=$( [ -f "$CATALOG" ] && c '^### ' "$CATALOG" || echo 0 )
TAGS_DISTINCT=$( [ -f "$CATALOG" ] && grep -oE '#[a-z0-9][a-z0-9-]*' "$CATALOG" 2>/dev/null | sort -u | wc -l | tr -d ' ' || echo 0 )
# format-compliance của catalog: % header ### là dạng ### [date]
CAT_FMT=0; [ "$H3_TOTAL" -gt 0 ] && CAT_FMT=$(( PATTERNS*100/H3_TOTAL ))

# ---------- B. Recidivism (lớp lỗi tái diễn theo tag, từ catalog) ----------
TOP_TAGS=$( [ -f "$CATALOG" ] && grep -oE '#[a-z0-9][a-z0-9-]*' "$CATALOG" 2>/dev/null | sort | uniq -c | sort -rn | head -8 | awk '{printf "%s(%s) ", $2,$1}' || echo "n/a" )

# ---------- C. Process compliance (workspaces) ----------
WS_TOTAL=0; WS_PROGRESS=0; WS_FULLDOC=0
if [ -d "$WS" ]; then
  for d in "$WS"/*/; do
    [ -d "$d" ] || continue
    WS_TOTAL=$((WS_TOTAL+1))
    [ -f "${d}05_progress.md" ] && WS_PROGRESS=$((WS_PROGRESS+1))
    if ls "${d}"01_*.md >/dev/null 2>&1 && ls "${d}"02_*.md >/dev/null 2>&1 && ls "${d}"08_*.md >/dev/null 2>&1; then
      WS_FULLDOC=$((WS_FULLDOC+1)); fi
  done
fi
PROGRESS_PCT=0; [ "$WS_TOTAL" -gt 0 ] && PROGRESS_PCT=$(( WS_PROGRESS*100/WS_TOTAL ))

# ---------- D. Governance infra ----------
RULES_G=$( [ -f "$GEMINI" ] && c -E '^[0-9]+\. ' "$GEMINI" || echo 0 )
RULES_C=$( [ -f "$CLAUDEMD" ] && c -E '^## [0-9]+\.' "$CLAUDEMD" || echo 0 )
SYNC="DRIFT"; [ "$RULES_G" = "$RULES_C" ] && SYNC="OK"
HOOKS=0; [ -f "$SETTINGS" ] && HOOKS=$(jq '[.hooks // {} | to_entries[] | .value[].hooks[]] | length' "$SETTINGS" 2>/dev/null || echo 0)
RAW_PRESENT="no"; RAW_HEADERS=0
if [ -f "$RAW" ]; then RAW_PRESENT="yes"; RAW_HEADERS=$(c '^## \[' "$RAW"); fi

# ---------- health flags ----------
flag() { [ "$1" = "1" ] && printf "✓" || printf "⚠"; }
F_SYNC=$( [ "$SYNC" = "OK" ] && echo 1 || echo 0 )
F_HOOKS=$( [ "$HOOKS" -ge 5 ] && echo 1 || echo 0 )
F_PROGRESS=$( [ "$PROGRESS_PCT" -ge 80 ] && echo 1 || echo 0 )
F_FMT=$( [ "$CAT_FMT" -ge 95 ] && echo 1 || echo 0 )

cat <<EOF
════════════════════════════════════════════════
 GOVERNANCE METRICS — $NOW
 (catalog=lessons.md · raw archive=ls_old.md)
════════════════════════════════════════════════
A. KNOWLEDGE BASE
   • Global Patterns (catalog)    : $PATTERNS
   • Distinct tags                : $TAGS_DISTINCT
   • Catalog format-compliance    : ${CAT_FMT}%  $(flag $F_FMT)  (### [date] / tổng ###)
   • Raw audit-log archived       : $RAW_PRESENT (ls_old.md, $RAW_HEADERS raw headers)

B. RECIDIVISM — lớp lỗi tái diễn nhiều nhất (tag • số lần)
   $TOP_TAGS

C. PROCESS COMPLIANCE
   • Workspaces                   : $WS_TOTAL
   • Có 05_progress.md            : $WS_PROGRESS  (${PROGRESS_PCT}%)  $(flag $F_PROGRESS)
   • Có full doc-set (01/02/08)   : $WS_FULLDOC

D. GOVERNANCE INFRA
   • Rules GEMINI=$RULES_G / CLAUDE=$RULES_C → sync $SYNC  $(flag $F_SYNC)
   • Active hooks (enforcement)   : $HOOKS  $(flag $F_HOOKS)

HEALTH: sync $(flag $F_SYNC)  hooks $(flag $F_HOOKS)  progress $(flag $F_PROGRESS)  catalog-fmt $(flag $F_FMT)
════════════════════════════════════════════════
EOF

# ---------- append trend snapshot (append-only) ----------
if [ ! -f "$TREND" ]; then
  printf '# governance_metrics.md — Trend Log (APPEND-ONLY)\n\n> Sinh bởi `agent/tooling/governance_metrics.sh`. Mỗi lần chạy APPEND 1 snapshot. KHÔNG sửa snapshot cũ (Rule 11).\n' >> "$TREND"
fi
{
  printf '\n## [%s] snapshot\n' "$TODAY"
  printf -- '- patterns=%s tags=%s catalog_fmt=%s%% raw_archive=%s\n' "$PATTERNS" "$TAGS_DISTINCT" "$CAT_FMT" "$RAW_PRESENT"
  printf -- '- workspaces=%s progress_compliance=%s%% fulldoc=%s\n' "$WS_TOTAL" "$PROGRESS_PCT" "$WS_FULLDOC"
  printf -- '- rules_gemini=%s rules_claude=%s sync=%s hooks=%s\n' "$RULES_G" "$RULES_C" "$SYNC" "$HOOKS"
  printf -- '- recidivism_top: %s\n' "$TOP_TAGS"
} >> "$TREND"
echo "→ Snapshot appended to $TREND"
