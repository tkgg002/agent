#!/usr/bin/env bash
# Rule 14 (Governance Pre-flight) — Stop
# Nhắc checklist quản trị cuối turn (non-blocking, ngắn gọn). Tắt qua /hooks nếu nhiễu.
set -euo pipefail
echo '{"systemMessage":"⛳ Pre-flight (Rule 14/16): (1) file đã tạo VẬT LÝ trong workspace? (2) 05_progress.md đã APPEND? (3) DoD G1–G8 pass? (4) đã liệt kê Skills đã dùng?"}'
exit 0
