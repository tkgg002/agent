# 02 — High-Level Plan: Tối ưu hoá hệ thống `/agent`

## 1. Khai báo Skills sử dụng (Rule #7 — Pre-flight Declaration)
1. `brain-delegate`: Thiết kế Master Workflow `S0–S6`, phân tầng Tier `T0–T3`, lập hồ sơ giải pháp `09_tasks_solution_master_workflow.md`.
2. `laya-router`: Thiết kế và kiểm định cổng quyết định `tooling/laya_gate.py` (`DG1`, `DG2`, `DG3`).
3. `muscle-execute`: Triển khai code cho `laya_gate.py`, `agy_guard.py`, `build_lessons_index.py`, `verify_governance.py` và chạy bộ `unittest`.
4. `workspace-docs`: Chuẩn hóa tài liệu workspace và chỉ mục 2 tầng `lessons/INDEX.md`.
5. `session-wrapup`: Kiểm định DoD `G1–G8`, chạy `verify_governance.py`, xuất báo cáo `11_report_*.md`.

## 2. Lộ trình 5 Phase
- **Phase 0 (Governance & Backup):** Tạo restore-point `rules/GEMINI_FULL_ARCHIVE_2026-10.md`, ghi nhận ADRs (`04_decisions_process_optimization.md`) và nhật ký `05_progress_process_optimization.md`.
- **Phase 1 (Core Rules & 10 Skills):** Rút gọn `GEMINI.md` xuống `< 12 KB` (giữ đủ Rule `0–22`), xây dựng 10 skill chuẩn tại `agent/skills/*/SKILL.md`, lưu trữ 96 workflow cũ vào `agent/workflows/_archive/`, cấu hình `.agents/skills.json`.
- **Phase 2 (Decision Gate `/laya_router`):** Xây dựng `tooling/laya_gate.py` (Mode A Calibrated Claim Gate + Mode B Agentic Debate Review) và bộ test `tooling/tests/test_laya_gate.py`.
- **Phase 3 (Antigravity Hooks `H1–H7`):** Lưu trữ hook Claude cũ vào `hooks/_archive_claude/`, xây dựng `hooks/agy/agy_guard.py`, `.agent-project.json`, `.agents/hooks.json`, và 16 kịch bản test trong `hooks/agy/tests/test_agy_guard.py`.
- **Phase 4 (2-Tier Lessons Index):** Xây dựng `tooling/build_lessons_index.py`, sinh `memory/global/lessons/INDEX.md` (`< 15 KB`) + 5 file `topics/*.md` bảo toàn 100% bài học (`241/241`).
- **Phase 5 (Process Linter & Wrap-up):** Nâng cấp `tooling/verify_governance.py` hỗ trợ Tier `T0–T3`, bắt buộc `--workspace`, viết unit test `test_verify_governance.py`, cập nhật `OPERATOR_MAP.md` & `governance_metrics.sh`, đóng phiên S6.
