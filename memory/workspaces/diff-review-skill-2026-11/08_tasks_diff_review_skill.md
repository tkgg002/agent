# 08 — Tasks: diff_review_skill

| ID | Task | Context | Logs/Test | Definition of Done | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| T1 | Viết `skills/diff-review/SKILL.md` | Thiết kế 3 lớp luật đã APPROVE; `GEMINI.md:19,29,76,82` | — | Mô tả đủ pipeline R0→R6, nạp luật, severity, verdict, giới hạn vòng, output | ✅ |
| T2 | Viết Base Rules L1 (4 file) | `rules/review/base/` | — | Mỗi lens có checklist dạng câu hỏi có/không, mỗi mục có mã `B-SEC-xx`, `B-LOG-xx`, `B-DES-xx`, `B-CON-xx` | ✅ |
| T3 | Viết Fintech Pack L2 | `rules/review/packs/fintech.md` | — | Có 4 mục lens + `money_path_keywords` + evidence bắt buộc; mã `F-xx` | ✅ |
| T4 | Viết `tooling/diff_review/scope.py` | TC-1, TC-5 | `py_compile`; chạy trên repo temp có/không manifest | Xuất JSON đủ cờ; exit 0 | ✅ |
| T5 | Viết `report_template.md` | `tooling/diff_review/` | — | Có đủ mục Scope/Verdict/Findings/Blast-radius/Gate/Evidence | ✅ |
| T6 | Viết `review_profile.md` data-hub + `profile_template.md` | trích `system_overview_and_design_pattern.md` Mục 2, 3.1, 3.2, 3.4, 4.5 (đã đọc 2026-11-13) | `wc -c` ≤ 8KB | File tồn tại, ≤8KB | ✅ |
| T7 | Hướng dẫn đăng ký skill + mở rộng `.agent-project.json` | `09_tasks_solution` | — | Có snippet JSON & dòng OPERATOR_MAP để User dán | ✅ |
| T8 | Audit S5 + `audit_report_diff_review_skill.md` + DONE | `GEMINI.md:76-77` | output lệnh thật | File tồn tại, 05 có DONE | ✅ |
