# Context: Upgrade Agent Infrastructure

## Scope
- Đồng bộ hóa bộ quy trình (rules), kỹ năng (skills), và lệnh (commands/workflows) từ `everything-claude-code-main` (v1.10.0) vào `.agent`.
- Bảo vệ và duy trì `agent/` (Project Memory/Governance).

## Goals
- [ ] Cập nhật Rules (common, golang, typescript).
- [ ] Cập nhật Skills (181 skills).
- [ ] Cập nhật Workflows (Legacy commands).

## Success Criteria
- [ ] Các rules mới nhất được áp dụng.
- [ ] Không làm mất dữ liệu trong `agent/memory/`.
- [ ] Các skills mới (`brand-voice`, `social-graph-ranker`) có mặt trong `.agent/skills`.
