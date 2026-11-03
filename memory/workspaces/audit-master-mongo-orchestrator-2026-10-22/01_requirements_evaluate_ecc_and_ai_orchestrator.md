---
tier: T2
suffix: evaluate_ecc_and_ai_orchestrator
---

# 01 — Requirements: Đánh giá chuyên sâu `affaan-m/ecc` (Everything Claude Code + ECC 2.0) và Đối sánh Kiến trúc với `ai-orchestrator` + `/agent` Core

## 1. Yêu cầu của User
- Đọc và mổ xẻ trực tiếp mã nguồn thực tế của repository `https://github.com/affaan-m/ecc` (`everything-claude-code`).
- Đánh giá khách quan, toàn diện kiến trúc, điểm mạnh, điểm yếu/giới hạn thực tế của `affaan-m/ecc` (bao gồm cả bộ Agent Harness/Plugin `v1.10.0` lẫn `ecc2` Rust TUI + `ecc_dashboard.py` + `scripts/` + `hooks/`) và đối sánh trực diện với hệ thống **`ai-orchestrator`** (`/Users/trainguyen/Documents/work/ai-orchestrator`) cùng **`/agent` Core** (`/Users/trainguyen/Documents/work/agent`).

## 2. Ràng buộc Thực thi Bắt buộc (Lesson-as-Enforced-Constraint — `L011`, `L026`, `LS-20261021-prior-workspace-lookup`, `LS-20261020-readme-only`)
- **`TRIPWIRE-OK: LS-20261021-prior-workspace-lookup`**: Đã tra cứu `memory/workspaces/upgrade-agent-infrastructure/00_context.md` và `02_plan.md` để nắm lịch sử đồng bộ `everything-claude-code` (`v1.10.0`) trước đây.
- **`TRIPWIRE-OK: LS-20261020-readme-only`**: Không chỉ đọc `README.md` — đã clone toàn bộ mã nguồn thật của `https://github.com/affaan-m/ecc` về `/Users/trainguyen/.gemini/antigravity/brain/029e438b-df78-4af7-bdf5-559bbd0375ab/scratch/ecc` để đọc trực tiếp `ecc2/Cargo.toml`, `ecc2/src/*.rs`, `ecc_dashboard.py`, `src/ecc- orchestrator`, `hooks/hooks.json`, `scripts/`, `agents/`, `skills/` và đối chiếu `file:line` với `/Users/trainguyen/Documents/work/ai-orchestrator/internal/**`.
- **`L026` (Direct Summary First)**: Mở đầu phản hồi bằng khối Summary trả lời trực tiếp đánh giá về `ecc` và `ai-orchestrator` trước khi đi vào phân tích chi tiết.
- **`Rule #9` & `Rule #13` (Propose-Only)**: Giữ nguyên kỷ luật Propose-Only (không tự ý sửa source code của `ai-orchestrator` khi chưa có `APPROVE`), kiểm định các nhận định và đề xuất nâng cấp qua hội đồng `/laya_router` (`DG0`, `DG1/DG2`).
