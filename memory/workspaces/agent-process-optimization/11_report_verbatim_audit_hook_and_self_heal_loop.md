---
tier: T2
suffix: verbatim_audit_hook_and_self_heal_loop
workspace: agent-process-optimization
created_at: 2026-10-20T17:03:20+07:00
---

# BÁO CÁO TỔNG KẾT (`11_report_verbatim_audit_hook_and_self_heal_loop.md`)

## 1. Tóm tắt Thay đổi

1. **Nhúng nguyên văn 100% Prompt Audit của User vào `hooks/agy/agy_guard.py` (`VERBATIM_USER_AUDIT_PROMPT`, dòng `78-109`):**
   - Khai báo hằng số `VERBATIM_USER_AUDIT_PROMPT` chứa nguyên văn 6 câu mệnh lệnh Audit của User và nhúng trực tiếp vào `FINAL_AUDIT_MSG` (`H6`) cũng như `handle_stop` (`H7`).
2. **Bắt buộc lượt Phản biện độc lập ở `H7` (`handle_stop`, dòng `705-814`):**
   - Tăng ngưỡng dừng từ `n >= 2` lên `n >= 3`.
   - Khi `run_linter and src and n == 0`, `handle_stop` luôn dội ngược (`decision: "continue"`, `cnt_file = "1"`) kèm `VERBATIM_USER_AUDIT_PROMPT` để buộc Agent phải thực hiện 1 lượt rà soát phản biện thực sự trước khi được phép báo OK.
3. **Vòng lặp Tự Khắc Phục Khép Kín `S5 → S3 → S4 → S5` không bắt User `APPROVE` lần 2 (`H4b`, dòng `595-623`):**
   - Khi phát hiện bug ở `S5` (`last_aud_idx > last_sol_idx`), `H4b` vẫn chặn sửa mò (`decision: "deny"`) và yêu cầu cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md`.
   - Tuy nhiên, `H4b` kiểm tra `APPROVE` dựa trên `first_sol_idx = sol[0] if sol else -1`, nên ngay sau khi Agent cập nhật `09_tasks_solution` trong vòng tự khắc phục, quyền sửa source code (`S4`) tự động mở khoá ngay trong cùng lượt mà không dừng lại bắt User `APPROVE` lần 2.
4. **Đồng bộ Tài liệu & Kiểm thử:**
   - Cập nhật `GEMINI.md:74-80`, `skills/qa-agent/SKILL.md:71`, `skills/muscle-execute/SKILL.md:73`, và `hooks/agy/tests/test_agy_guard.py` (`test_29`, `test_31`).
   - Toàn bộ 31/31 tests (`hooks/agy/tests`), 32/32 tests (`tooling/tests`), 3/3 mutants (`mutation_gate.py` `KILLED`), và `DG3` (`PASS`) đều hoàn tất.
