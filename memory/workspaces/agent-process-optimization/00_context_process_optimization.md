# 00 — Context: Tối ưu hoá toàn diện hệ thống `/agent` trên Google Antigravity

## 1. Bối cảnh & Vấn đề gốc rễ
- Hệ thống `/Users/trainguyen/Documents/work/agent` là hạt nhân điều phối (Agentic Core) dùng chung cho mọi dự án.
- Trước khi tối ưu:
  1. `GEMINI.md` dài `30,949 bytes` vượt ngưỡng cắt `24,000 bytes` của Antigravity khiến các quy tắc từ `Rule #15` đến `Rule #22` bị cắt mất trong System Prompt.
  2. `agent/workflows/` gồm 96 file `.md` phẳng theo cơ chế cũ, không được Antigravity tự động nhận diện như `skills/<name>/SKILL.md`.
  3. `agent/hooks/` chứa các shell script cũ của Claude Code (`CLAUDE_TOOL_NAME`, `.claude/settings.json`), không tương thích với contract `hooks.json` của Antigravity (`PreInvocation`, `PreToolUse`, `PostInvocation`, `Stop`).
  4. `agent/memory/global/lessons.md` dài `487 KB` (`241` bài học) nếu đọc nguyên khối sẽ gây tràn Context Window.
  5. Chưa tích hợp cổng kiểm định `/laya_router` (`127.0.0.1:8200`) thành công cụ CLI tự động có câu đối chứng (`control_true` / `control_false` / `corrupted-diff`).
  6. Core cần tách bạch hoàn toàn khỏi thông tin đặc thù dự án (ports, services, câu lệnh build/test riêng) thông qua manifest `<project-root>/.agent-project.json`.

## 2. Phạm vi & Ranh giới Kiến trúc (Strict Agnostic Core)
- **Trong `/Users/trainguyen/Documents/work/agent`:**
  - `GEMINI.md` (`11,928 bytes <= 12,000 bytes`, bảo toàn đủ 23 rule headers `0`–`22`).
  - `skills/<name>/SKILL.md` (10 Core Skills chuẩn Antigravity) + `.agents/skills.json`.
  - `hooks/agy/agy_guard.py` (7 chốt chặn `H1–H7`) + `.agents/hooks.json`.
  - `tooling/laya_gate.py`, `tooling/build_lessons_index.py`, `tooling/verify_governance.py`, `tooling/governance_metrics.sh` + bộ unit test `unittest`.
  - `memory/global/lessons/INDEX.md` (`< 15 KB`) + `memory/global/lessons/topics/01..05_*.md`.
- **Tại từng repo dự án:**
  - `.agent-project.json` khai báo `components` (`name`, `path`, `checks`, `test`, `runtime_check`) và `.agents/{hooks,skills}.json`.
