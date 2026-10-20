#!/usr/bin/env bash
# Rule 7 (No-Shadow) — PostToolUse / Write|Edit
# Sau khi sửa SOURCE CODE → nhắc model APPEND 05_progress.md trong cùng turn (non-blocking).
set -euo pipefail
input=$(cat)
fp=$(printf '%s' "$input" | jq -r '.tool_input.file_path // .tool_response.filePath // empty')
[ -z "$fp" ] && exit 0

case "$fp" in
  *.go|*.ts|*.tsx|*.js|*.jsx|*.py|*.sql|*.java|*.rb|*.rs)
    jq -n --arg f "$(basename "$fp")" '{
      hookSpecificOutput: {
        hookEventName: "PostToolUse",
        additionalContext: ("Rule 7 (No-Shadow): vừa sửa source `" + $f + "`. BẮT BUỘC APPEND 1 dòng vào 05_progress.md của workspace TRONG CÙNG turn ([Timestamp][Agent:Model] mục đích thay đổi). Khi báo Done nhớ Rule 16-G8 (bằng chứng verify).")
      }
    }'
    ;;
esac
exit 0
