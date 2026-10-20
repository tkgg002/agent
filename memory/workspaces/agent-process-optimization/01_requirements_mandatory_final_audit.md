# 01 — Requirements: Tích hợp Cổng Mandatory Final Adversarial Audit (`audit_report_*.md`) & Cưỡng chế bằng Hook (`H6/H7`) ở Cuối Workflow

tier: T2

## 1. Yêu cầu từ User (`2026-10-19T11:33:51+07:00`)
> *"sau khi tao làm xong, tao chạy 'Audit lại quá trình vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update. Kiểm tra các xem có suy diễn, báo cáo láo không. Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop). Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng Lưu toàn bộ vào file audit_report_*.md trong workspace của task Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value.' => và vẫn ra rất nhiều bug. mày thêm cái audit vào vào cuối workflow cho tao. và luôn chạy hook script để nó luôn đc kích hoạt, chứ tao rất phiền vì phải đi nhắc"*

## 2. Phạm vi & Ràng buộc
- **Phạm vi:**
  1. `GEMINI.md` (Rule #4, Rule #9 S5/S6, Rule #14 Gates G3/G7/G8, Rule #18).
  2. Các Core Skills liên quan: `skills/qa-agent/SKILL.md`, `skills/session-wrapup/SKILL.md`, `skills/workspace-docs/SKILL.md`, `skills/muscle-execute/SKILL.md`.
  3. Lifecycle Hooks Dispatcher `hooks/agy/agy_guard.py` (`H6 PostInvocation` và `H7 Stop`) + unit tests `hooks/agy/tests/test_agy_guard.py`.
  4. Process Linter `tooling/verify_governance.py` (`TIER_RULES` và kiểm tra cấu trúc nội dung `audit_report_*.md`) + unit tests `tooling/tests/test_verify_governance.py`.
- **Ràng buộc:** Propose-Only ở bước S1–S3 (lập `09_tasks_solution_mandatory_final_audit.md`, chạy `laya_gate.py` cho `DG1/DG2`, trình User `APPROVE` trước khi sửa code).

## 3. Tiêu chí Chấp nhận (Acceptance Criteria — DoD)
- **AC1 (Workflow & Skills):** Quy trình **Mandatory Final Adversarial Audit** (6 yêu cầu: phản biện quá trình & từng task/từng file/từng dòng update `git diff`; kiểm tra suy diễn/báo cáo láo; áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục; chạy QC gắt gao + soạn đầy đủ context gọi `/laya_router` `DG3` nhận xét lại quyết định; lưu toàn bộ vào `audit_report_<suffix>.md`; tuyệt đối cấm test bằng mockup/hallucinate kết quả runtime — nếu thiếu môi trường phải chủ động đề nghị phối hợp với User trace runtime value thật) được quy định rõ trong `GEMINI.md` và các Core Skills.
- **AC2 (Auto-Trigger qua Hook `H6 PostInvocation`):** Ngay khi Agent sửa bất kỳ file source code nào (`src_edits`), `H6` tự động tiêm thông điệp bắt buộc thực thi **Mandatory Final Adversarial Audit Protocol** kèm danh sách 6 mục kiểm tra để Agent tự chạy mà User không bao giờ phải gõ lệnh nhắc.
- **AC3 (Hard-Block qua Hook `H7 Stop` & `verify_governance.py`):** Khi có thay đổi source code (`src_edits`) hoặc đóng phiên `S6` cho `T2/T3`, `H7` và `verify_governance.py` **CHẶN CỨNG (`decision: "continue"`)** nếu:
  1. Chưa tạo/cập nhật `audit_report_<suffix>.md` sau lần sửa code cuối.
  2. File `audit_report_<suffix>.md` thiếu 1 trong 5 phần bắt buộc (`Line-by-Line Diff Critique`, `Anti-Speculation & Anti-Mockup Runtime Trace`, `Self-Improvement Loop`, `Laya Router DG3 Verdict`, `Final QC Conclusion`).
  3. Chưa chạy `/laya_router` (`DG3 = PASS`) sau lần sửa code cuối.
