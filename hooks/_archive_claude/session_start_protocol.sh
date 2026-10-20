#!/usr/bin/env bash
# Rule 7/15 (Startup Protocol) — SessionStart
# Tiêm nhắc nhở đọc memory đầu phiên vào context của model.
set -euo pipefail
jq -n '{
  hookSpecificOutput: {
    hookEventName: "SessionStart",
    additionalContext: "STARTUP PROTOCOL (GEMINI/CLAUDE Rule 7/15): TRƯỚC khi làm task, ĐỌC — agent/memory/global/lessons.md (CATALOG chuẩn hoá 8 nhóm Rule 13, tra cứu nhanh), active_plans.md, project_context.md, tech_stack.md; và context workspace đang chạy. ls_old.md = raw audit-log cũ đã archive (chỉ tra khi cần lịch sử). Khi làm: Rule 16 (DoD G1–G8 trước khi báo Done), Rule 18 (restore-point/backup trước khi sửa file quan trọng), Rule 19 (không ghi secret thô vào memory)."
  }
}'
exit 0
