# 08 — Tasks: Tích hợp Nguyên văn Prompt Audit vào Hook & Tự động Khép kín Vòng lặp Tự khắc phục (`verbatim_audit_hook_and_self_heal_loop`)

- **Workspace:** `agent-process-optimization`
- **Suffix:** `verbatim_audit_hook_and_self_heal_loop`
- **Tier:** `T2`
- **Trạng thái:** `S6 DONE`

---

## Danh sách Task Thực thi (Sau khi có `APPROVE`)

### [x] Task `T9.1`: Cập nhật `GEMINI.md` (`L78`) và `skills/qa-agent/SKILL.md` (`L71`) cho Vòng lặp Tự khắc phục Khép kín (`S5 → S3 → S4 → S5` Autonomous Self-Healing Loop)
- **Mô tả & Phạm vi (`[Mô tả]`):**
  - Cập nhật `GEMINI.md:78` và `skills/qa-agent/SKILL.md:71`:
    - Nhúng nguyên văn Prompt Audit của User vào quy định bước `S5`.
    - Thay thế quy định *"chờ User `APPROVE` mới khi phát hiện bug ở `S5`"* bằng quy trình **Autonomous Closed-Loop Self-Healing (`S5 → S3 → S4 → S5`)**: Khi đã có `APPROVE` thực thi task (hoặc khi User kích hoạt `Self-Improvement Loop`), nếu `S5 Audit` phát hiện bug/lỗ hổng/thiếu sót so với Plan ban đầu, Agent bắt buộc: (1) ghi `lessons.md` + chạy `build_lessons_index.py`, (2) quay ngược về `S3` cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` và chạy lại `DG1`/`DG2` (`PASS`), (3) **tự động chuyển sang `S4` sửa dứt điểm source code + chạy test/runtime**, và (4) **quay lại `S5` chạy `git diff` + `DG3` + cập nhật `audit_report_<suffix>.md` cho tới khi sạch bóng bug rồi mới báo OK cho User** (không dừng lại bắt User `APPROVE` lần 2; chỉ dừng phối hợp User khi cần trace runtime value trên môi trường thật hoặc chạm ngưỡng Escalation >3 lần thất bại).
- **Dữ liệu & Bằng chứng (`[Context & Logs]`):**
  - `GEMINI.md:74-80`
  - `skills/qa-agent/SKILL.md:57-72`
- **Tiêu chí Hoàn thành (`[Definition of Done]`):**
  - Dung lượng `GEMINI.md` giữ dưới ngưỡng `< 18 KB`.

### [x] Task `T9.2`: Nâng cấp `hooks/agy/agy_guard.py` (`VERBATIM_USER_AUDIT_PROMPT`, `H4b` Self-Heal Unlock, `H7` Mandatory `git diff` & Verbatim Audit Stop-Bounce)
- **Mô tả & Phạm vi (`[Mô tả]`):**
  - Trong `hooks/agy/agy_guard.py`:
    1. **Nhúng nguyên văn 100% Prompt Audit của User (`L78-98`):**
       - Định nghĩa `VERBATIM_USER_AUDIT_PROMPT` giữ nguyên từng chữ prompt của User và đưa trực tiếp vào đầu `FINAL_AUDIT_MSG` (`H6`) cũng như thông điệp chặn của `handle_stop` (`H7`).
    2. **Mở khóa Vòng lặp Tự khắc phục Khép kín `S5 → S3 → S4 → S5` tại `H4b` (`L577-603`):**
       - Giữ nguyên chốt chặn `if last_aud_idx > last_sol_idx: return {"decision": "deny", ...}` để bắt buộc Agent phải quay về `S3` cập nhật `09_tasks_solution_<suffix>.md` trước khi sửa code (chống sửa mò không cập nhật plan).
       - Sửa điều kiện kiểm tra `ok` (`APPROVE`) tại `L594-598`: Kiểm tra xem trong phiên đã có `USER_INPUT` khớp `APPROVE` (sau `sol[0]`) hoặc `SELF_IMPROVEMENT_PROMPT` (`r"Self-Improvement Loop|Vòng Lặp Phản Tỉnh"`). Nhờ đó, sau khi Agent cập nhật `09_tasks_solution_<suffix>.md` (`last_sol_idx > last_aud_idx`) và chạy `DG1 PASS` trong vòng lặp `S5 → S3 → S4`, Agent được phép sửa ngay source code để diệt tận gốc bug mà **không bị `H4b` chặn bắt xin User `APPROVE` lần 2**!
    3. **Bắt buộc chạy `git diff`/`git show` sau lần sửa code cuối & Bơm nguyên văn `VERBATIM_USER_AUDIT_PROMPT` tại `H7` (`L718-772`):**
       - Trong `handle_stop`, khi `src` khác rỗng (`if src:`), kiểm tra `touched(st, r"\bgit\b.*\b(diff|show)\b", last)`: bắt buộc Agent phải chạy `git diff` hoặc `git show` thật sau lần sửa code cuối (`last = src[-1][0]`) trước khi viết `audit_report_<suffix>.md`.
       - Khi `miss` khác rỗng (đặc biệt khi `src` khác rỗng), đính kèm nguyên văn `VERBATIM_USER_AUDIT_PROMPT` vào `reason` trả về của `handle_stop` để cưỡng chế Agent thực thi đúng 100% prompt phản biện của User trước khi được dừng.
- **Dữ liệu & Bằng chứng (`[Context & Logs]`):**
  - `hooks/agy/agy_guard.py:78-98, 577-603, 718-772`
- **Tiêu chí Hoàn thành (`[Definition of Done]`):**
  - Cập nhật & bổ sung unit tests trong `hooks/agy/tests/test_agy_guard.py` đạt `PASS` 100%.

### [x] Task `T9.3`: Chạy Kiểm thử Đột biến (`mutation_gate.py`), Chạy `DG3`, Xuất `audit_report_verbatim_audit_hook_and_self_heal_loop.md` & Đóng phiên `S6`
- **Mô tả & Phạm vi (`[Mô tả]`):**
  - Chạy `mutation_gate.py` với $\ge 3$ bản cố tình làm hỏng logic mới trong `agy_guard.py` (`3/3 KILLED`).
  - Thực thi đầy đủ `S5` (chạy `git diff` thật sau lần sửa code cuối, gọi `/laya_router` `DG3`, xuất `audit_report_verbatim_audit_hook_and_self_heal_loop.md`, `06_validation_verbatim_audit_hook_and_self_heal_loop.md`, `11_report_verbatim_audit_hook_and_self_heal_loop.md`) và chạy `verify_governance.py`.
- **Dữ liệu & Bằng chứng (`[Context & Logs]`):**
  - `hooks/agy/tests/test_agy_guard.py`
  - `tooling/mutation_gate.py`
- **Tiêu chí Hoàn thành (`[Definition of Done]`):**
  - `verify_governance.py --workspace agent-process-optimization --suffix verbatim_audit_hook_and_self_heal_loop` đạt `PASSED 🟢`.
