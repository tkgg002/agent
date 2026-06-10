#!/usr/bin/env bash
# Rule 19 (No-Secret-in-Memory) — PreToolUse / Write|Edit
# BLOCK writing RAW secrets/credentials vào memory files. High-precision patterns để
# tránh false-positive trên prose ("password") và giá trị đã mask (***).
set -euo pipefail
input=$(cat)
fp=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')
[ -z "$fp" ] && exit 0

# Chỉ quét memory files
scan=0
case "$fp" in
  */agent/memory/*) scan=1 ;;
esac
case "$(basename "$fp")" in
  lessons.md|lessons_global_normalized.md|05_progress.md|04_decisions.md|active_plans.md|project_context.md) scan=1 ;;
esac
[ "$scan" -eq 0 ] && exit 0

# Nội dung sắp ghi: Write(.content) hoặc Edit(.new_string)
content=$(printf '%s' "$input" | jq -r '.tool_input.content // .tool_input.new_string // empty')
[ -z "$content" ] && exit 0

hit=$(printf '%s' "$content" | grep -nEi \
  -e 'AKIA[0-9A-Z]{16}' \
  -e 'sk-[A-Za-z0-9]{16,}' \
  -e 'gh[opsu]_[A-Za-z0-9]{20,}' \
  -e 'Bearer[[:space:]]+[A-Za-z0-9._-]{20,}' \
  -e '-----BEGIN[[:space:]].*PRIVATE KEY' \
  -e '://[A-Za-z0-9_.-]+:[^@/[:space:]*]{3,}@' \
  2>/dev/null | head -3 || true)

if [ -n "$hit" ]; then
  jq -n --arg h "$hit" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: ("Rule 19 (No-Secret-in-Memory): phát hiện secret/credential thô sắp ghi vào memory file. PHẢI mask trước (vd `***`, `token=***`, `mongodb://***:***@host`). Dòng khớp:\n" + $h)
    }
  }'
  exit 0
fi
exit 0
