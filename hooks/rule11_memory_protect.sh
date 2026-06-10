#!/usr/bin/env bash
# Rule 11 (Memory File Protection) — PreToolUse / Write
# BLOCK overwriting an EXISTING memory file via the Write tool (Data Destruction guard).
# Allow: new files, non-memory files, và Edit/Bash-append (không match hook này).
set -euo pipefail
input=$(cat)
fp=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')
[ -z "$fp" ] && exit 0

is_mem=0
case "$(basename "$fp")" in
  lessons.md|lessons_global_normalized.md|05_progress.md|04_decisions.md|active_plans.md|project_context.md|tech_stack.md) is_mem=1 ;;
esac
case "$fp" in
  */agent/memory/*) is_mem=1 ;;
esac

if [ "$is_mem" -eq 1 ] && [ -f "$fp" ]; then
  jq -n --arg f "$fp" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: ("Rule 11 (Memory File Protection): CẤM dùng Write đè memory file đang tồn tại: " + $f + ". Memory file chỉ được APPEND. Dùng Edit (sửa có chủ đích) hoặc Bash `>>` (append cuối file). Đây là chống Data Destruction — lỗi nghiêm trọng nhất.")
    }
  }'
  exit 0
fi
exit 0
