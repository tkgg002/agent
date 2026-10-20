---
tier: T2
workspace: agent-process-optimization
suffix: verbatim_audit_hook_and_self_heal_loop
---

# 01 — Requirements: Tích hợp Nguyên văn Prompt Audit của User vào Hook & Tự động Khép kín Vòng lặp Tự khắc phục (`S5 -> S3 -> S4 -> S5`) không bắt `APPROVE` lại (`verbatim_audit_hook_and_self_heal_loop`)

## 1. Bối cảnh & Phản ánh trực tiếp từ User
User phản ánh sau khi chạy thử thực tế trên các task (`shadow_master_update_drop`, `harness2_recursive_integration`, `modular_agent_orchestrator`):
> *"tao đã chạy thử, đã báo ok, nhưng tao cho audit = "Audit lại "quá trình" vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update. 
> Kiểm tra các xem có suy diễn, báo cáo láo không. 
> Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop).
> Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng 
> Lưu toàn bộ vào file audit_report_*.md trong workspace của task
> Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value." => kết quả vẫn còn bug, lỗ hổng. và lại phải tìm cách giải quyết, xong bắt tao approve. tao đã nói là lấy nguyên cái promt của tao vào hook audit mà. sao mày ko làm"*

## 2. Danh sách Yêu cầu Bắt buộc (`R1 → R4`)

| Req ID | Mô tả Yêu cầu | Tiêu chí Nghiệm thu (Acceptance Criteria) |
| :--- | :--- | :--- |
| **`R1`** | **Nhúng nguyên văn 100% từng chữ Prompt Audit của User vào Hook (`hooks/agy/agy_guard.py`)** | Khai báo hằng số `VERBATIM_USER_AUDIT_PROMPT` chứa nguyên văn từng chữ Prompt Audit của User, sử dụng trực tiếp trong cả `FINAL_AUDIT_MSG` (`H6`) và thông điệp chặn của `handle_stop` (`H7`). Không cắt gọt hay biên tập lại câu chữ gốc của User. |
| **`R2`** | **Chặn đứng "Audit hình thức 1 lượt / Báo OK khi còn bug" tại Hook `H7` (`handle_stop`)** | Khi lượt hiện tại có sửa source code (`if src:`):<br>1. Bắt buộc phải có lệnh chạy `git diff` hoặc `git show` thật **sau lần sửa code cuối cùng (`last = src[-1][0]`)** để kiểm tra từng file & từng dòng update thực tế (chống viết `audit_report_*.md` từ trí nhớ/tóm tắt nén ngữ cảnh).<br>2. Ở lần dừng đầu tiên sau khi sửa code (`n == 0` và chưa qua lượt Phản Biện Độc Lập sau khi đóng code), hoặc khi thiếu bất kỳ điều kiện S5/S6 nào, `H7` trả về `decision: "continue"` kèm **nguyên văn `VERBATIM_USER_AUDIT_PROMPT`** để ép Agent chuyển từ tâm lý Builder sang tâm lý Adversarial Auditor, tìm và diệt sạch bug trước khi được báo "OK" cho User. |
| **`R3`** | **Tự động khép kín vòng lặp `S5 → S3 → S4 → S5` (Self-Healing Loop) mà KHÔNG bắt User `APPROVE` lần 2** | Sửa lỗi logic tại `hooks/agy/agy_guard.py:578-602` và `GEMINI.md:78` + `skills/qa-agent/SKILL.md:71`:<br>1. Khi `S5 Audit` (`audit_report_*.md`) phát hiện bug/lỗ hổng, Agent vẫn bắt buộc quay về `S3` cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` và chạy `DG1`/`DG2` (`last_sol_idx > last_aud_idx` — chống sửa mò không theo plan).<br>2. **Tuy nhiên**, một khi trong phiên đã có lệnh `APPROVE` ban đầu của User (`sol[0]`) hoặc User yêu cầu chạy Audit kèm *"Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop)"*, sau khi cập nhật `09_tasks_solution_<suffix>.md` và đạt `DG1 PASS`, Hook `H4b` **CHO PHÉP Agent sửa luôn source code (`S4`), chạy lại test và Audit lại (`S5`) khép kín cho tới khi hết sạch bug rồi mới báo OK**, tuyệt đối không dừng lại bắt User phải `APPROVE` lần 2! |
| **`R4`** | **Kiểm thử Unit Tests & 3-Mutant Verification Gate** | Toàn bộ unit tests trong `hooks/agy/tests/test_agy_guard.py` và `tooling/tests/` đạt `PASS` 100%, đồng thời vượt qua `tooling/mutation_gate.py` (`3/3 KILLED`) và `/laya_router` (`DG1`, `DG2`, `DG3`). |
