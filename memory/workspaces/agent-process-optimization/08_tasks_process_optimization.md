# 08 — Tasks: Tối ưu hoá quy trình Agent

Chi tiết giải pháp: `13_analysis_process_optimization.md`. Mỗi phase cần User APPROVE riêng.

## Phase 0 — Chuẩn bị
- [x] T0.1 Backup `GEMINI.md` đầy đủ sang `rules/GEMINI_FULL_ARCHIVE_2026-10.md` (restore-point) — KHÔNG push
- [x] T0.2 Xác nhận contract hooks Antigravity (`PreInvocation`, `PreToolUse`, `PostInvocation`, `Stop`)

## Phase 1 — Rules + Skills (U1, U2)
- [x] T1.1 Tách `GEMINI.md` → Core ≤ 12KB (`11,928B`, đủ 23 rules `0`–`22`); chuyển chi tiết sang skills
- [x] T1.2 Giữ Rule #8 là 1 dòng Deprecated (ADR-01) để bảo toàn số thứ tự Rule #9–#22
- [x] T1.3 Tạo `.agents/skills.json` trỏ `agent/skills` và `~/.agent/skills`
- [x] T1.4 Xây dựng 10 skill lõi chuẩn `agent/skills/<name>/SKILL.md` (bỏ `git push`, bỏ hardcode dự án)
- [x] T1.5 Archive 96 workflows phẳng cũ → `workflows/_archive/` + `workflows/README.md`

## Phase 2 — Hooks (U3)
- [x] T2.1 `.agents/hooks.json` + `hooks/agy/agy_guard.py` (`H1–H7`) + `.agent-project.json`
- [x] T2.2 Test 16 kịch bản fixture trong `hooks/agy/tests/test_agy_guard.py` (`16/16 PASS`)

## Phase 3 — Lessons (U4)
- [x] T3.1 Script `tooling/build_lessons_index.py` sinh `lessons/INDEX.md` (`14,576B < 15KB`) + `lessons/topics/*.md` (Read-Only trên `lessons.md`)
- [x] T3.2 Đối soát `241 = 241` bài học; unit test `tooling/tests/test_build_lessons_index.py` (`PASS`)

## Phase 4 — Linter, Tier, Vệ sinh (U5–U7)
- [x] T4.1 Nâng cấp `verify_governance.py` (bắt buộc `--workspace`, hỗ trợ Tier `T0–T3`, đường dẫn động) + unit test `test_verify_governance.py` (`5/5 PASS`)
- [x] T4.2 Archive `organize_skills.sh` và 6 hook `.sh` Claude cũ vào `hooks/_archive_claude/` (`chmod -x`)
- [x] T4.3 Cập nhật `OPERATOR_MAP.md` và `tooling/governance_metrics.sh` theo kiến trúc Antigravity

---

## Cập nhật 2026-10-16 (thay thế lộ trình trên — chi tiết: `09_tasks_solution_master_workflow.md` mục 6)
- [x] Q1 bỏ Claude Code → huỷ phần sync CLAUDE.md (ADR-01)
- [x] Q2 không có key thật → huỷ T4.3 phần `models.env` (ADR-02)
- [x] Q3 tier T0–T3 (ADR-03)
- [x] Phase 0: restore-point (`rules/GEMINI_FULL_ARCHIVE_2026-10.md`) + ADRs (`04_decisions_process_optimization.md`)
- [x] Phase 1: GEMINI Core (`11,928B`, mục 9 Master Workflow, mục 10 Decision Gate, Rule 8 deprecated) + `.agents/skills.json` + 10 skills lõi (`laya-router`, `audit-propose-only`, v.v.)
- [x] Phase 2: `agent/tooling/laya_gate.py` + `tooling/tests/test_laya_gate.py` (`5/5 PASS`, bao gồm Mode A, Mode B Debate, Backtest)
- [x] Phase 3: `agent/hooks/agy/agy_guard.py` + `.agents/hooks.json` + `.agent-project.json` + 16 test fixture (`16/16 PASS`)
- [x] Phase 4: `tooling/build_lessons_index.py` + `lessons/INDEX.md` (`14,576B`) + `lessons/topics/01..05_*.md` (`241/241` entry)
- [x] Phase 5: `tooling/verify_governance.py` theo tier `T0–T3` + `test_verify_governance.py` (`5/5 PASS`) + cập nhật `OPERATOR_MAP.md`, `governance_metrics.sh`, và đóng gói hồ sơ S6 (`00, 02, 06, 11, 12, 14`)

---

## Phase 6 — Khắc phục 5 Lỗ hổng Bỏ qua Workflow & Hook từ Thực chiến (`09_tasks_solution_workflow_enforcement_fix.md`)
- [x] T6.1 **Giải phóng `<skills>` Context Budget & Discovery (`~/.gemini/config/skills/`:58-75):** Lưu trữ 438 skill ngoại lai sang `~/.gemini/config/skills_archive/`, tạo symlink 10 Core Skills vào `~/.gemini/config/skills/` (Priority 3) và đồng bộ `/Users/trainguyen/Documents/work/data-hub/.agents/{skills,hooks}.json` (Priority 2). Inline checklist bắt buộc của `S1 → S6` vào `GEMINI.md:66-75`.
- [x] T6.2 **Sửa lỗi `tgt()` bọc dấu `"` trên `transcript.jsonl` thật (`hooks/agy/agy_guard.py:72-121`):** Thêm `_unquote()` cho toàn bộ `args` đọc từ `transcript.jsonl` để `is_src()`, `src_edits()`, `H6`, `H7` hoạt động chính xác 100%.
- [x] T6.3 **Nâng cấp `H1`, `H4`, `H7` trong `hooks/agy/agy_guard.py:214-495`:**
  - `H1`: Chạy theo từng lượt `USER_INPUT` mới (`last_user_idx`), nhắc tạo `01/05/08` trước `09`.
  - `H4`: Khi ghi `09_tasks_solution_<suffix>.md`, chặn (`deny`) nếu chưa tạo `01_requirements_<suffix>.md`, `08_tasks_<suffix>.md`, hoặc chưa append `START` vào `05_progress_*.md` trong lượt hiện tại.
  - `H7`: Reset bộ đếm `.stop` theo từng lượt `USER_INPUT`; chặn Shadow Analysis (đọc/tra cứu sâu `>= 4` file dự án hoặc sửa `workspaces/` mà không cập nhật `05_progress` / hồ sơ workspace).
- [x] T6.4 **Nâng cấp `tooling/verify_governance.py:54-230`:** Hỗ trợ `--suffix <suffix>` và `--stage S3|S6` để chặn việc PASS "ăn theo" file cũ khi tái sử dụng workspace. Cập nhật unit tests cho cả `test_agy_guard.py` (`20/20 PASS`) và `test_verify_governance.py` (`7/7 PASS`).
