# 12 — Implementation Plan: Tối ưu hoá quy trình `/agent`

## 1. Kiến trúc Tổng thể Đã Triển khai
1. **Master Workflow `S0 → S6`:**
   - `S0 (Boot)`: Đọc `GEMINI.md` + `memory/global/lessons/INDEX.md` + mở duy nhất `topics/0X_*.md` thuộc domain.
   - `S1 (Classify & Workspace)`: Phân loại `T0` / `Audit` / `T1` / `T2` / `T3`; tạo workspace + ghi `05_progress` `START`.
   - `S2 (Investigate & DG1)`: Phân tích thực chứng, chạy `laya_gate.py run --stage dg1`.
   - `S3 (Propose & DG2 -> User APPROVE)`: Viết `09_tasks_solution_*.md`, chạy `laya_gate.py run --stage dg2`, chờ lệnh `APPROVE` của User.
   - `S4 (Execute)`: Muscle sửa code; `H6` tự động chạy `checks` từ `.agent-project.json`.
   - `S5 (Verify DoD G1–G8 & DG3)`: Chạy test + `runtime_check`, chạy `laya_gate.py` (`DG3`), ghi `06_validation_*.md`.
   - `S6 (Wrap-up)`: Ghi `11_report_*.md`, `05_progress_*.md` (`DONE`), chạy `verify_governance.py --workspace <ws>`.

## 2. Chi tiết Thực thi từng Phase
- **Phase 0:** Sao lưu `GEMINI.md` gốc sang `rules/GEMINI_FULL_ARCHIVE_2026-10.md`; thiết lập ADRs `ADR-01`..`ADR-08`.
- **Phase 1:** Rút gọn `GEMINI.md` còn `11,928B`; tạo 10 skill chuẩn tại `agent/skills/*/SKILL.md`; dọn 96 workflow phẳng vào `workflows/_archive/`; đăng ký `.agents/skills.json`.
- **Phase 2:** Viết `tooling/laya_gate.py` và `tooling/tests/test_laya_gate.py` (`6/6` tests PASS).
- **Phase 3:** Dọn hook `.sh` cũ vào `hooks/_archive_claude/`; viết `hooks/agy/agy_guard.py`, `.agent-project.json`, `.agents/hooks.json`, và `hooks/agy/tests/test_agy_guard.py` (`16/16` tests PASS).
- **Phase 4:** Viết `tooling/build_lessons_index.py` + `tooling/tests/test_build_lessons_index.py`; sinh `memory/global/lessons/INDEX.md` (`14,576B`) và 5 file `topics/01..05_*.md` (`241/241` bài học).
- **Phase 5:** Nâng cấp `tooling/verify_governance.py` + `tooling/tests/test_verify_governance.py`, cập nhật `OPERATOR_MAP.md` và `tooling/governance_metrics.sh`.
