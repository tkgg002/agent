---
name: session-wrapup
description: >-
  Session Close & Continuity workflow (S6 Close, Save/Resume Session, Rule #18, Rule #22) —
  completes audit_report, 06_validation, 11_report, 12_implementation_plan, appends DONE to
  05_progress, audits lessons, and runs verify_governance.py --workspace <ws>.
---

# Session Wrapup & Continuity Skill (S6 Close)

> **Quy tắc cốt lõi:** Rule #0 (Lesson & Skill summary), Rule #4 (Pre/Post-flight Sync), Rule #6 (Lesson Pre-flight Check), Rule #9 (S6 Close), Rule #18 (Governance & Final Adversarial Audit), Rule #22 (Process Linter).

## Khi nào sử dụng
- Ở chặng **S6 CLOSE** trước khi kết thúc một task `T1–T3` hoặc trước khi tạm dừng/chuyển giao phiên làm việc.
- Khi bắt đầu lại một phiên làm việc dang dở (`Resume Session`).

## A. Quy trình Tiếp nối Đầu Phiên (Resume Session)
1. Đọc `/Users/trainguyen/Documents/work/agent/GEMINI.md` và `/Users/trainguyen/Documents/work/agent/memory/global/lessons/INDEX.md`.
2. Đọc `05_progress_<suffix>.md` (phần cuối), `08_tasks_<suffix>.md`, và `09_tasks_solution_<suffix>.md` trong workspace đang mở.
3. Kiểm tra `git status -s` tại đúng thư mục repo của component đang làm để nắm trạng thái code thực tế.

## B. Quy trình Đóng Phiên Chuẩn (S6 Close — 6 Bước Bắt Buộc)

### Bước 1 — Hoàn thiện Hồ sơ Nghiệm thu, Phản biện & Báo cáo Vật lý
Đảm bảo các file sau đã được ghi đầy đủ trong `agent/memory/workspaces/<ws>/` (tùy theo Tier `T1/T2/T3`):
- `audit_report_<suffix>.md` (**Bắt buộc cho T2/T3 và mọi phiên có sửa code**): Báo cáo Audit phản biện cuối phiên đủ 6 phần (Phản biện từng dòng diff & từng task, Chống suy diễn Smoke Test & Trace Runtime thật, Kiểm tra Concurrency/Timeout/Data Continuity, Kiểm tra Suy diễn/Báo cáo láo, Self-Improvement Loop, và kết quả `/laya_router` `DG3`).
- `06_validation_<suffix>.md`: Lệnh kiểm thử/runtime đã chạy + output thực tế + kết quả Decision Gate (`DG1/DG2/DG3`).
- `11_report_<suffix>.md`: Danh sách các file đã thay đổi + số lượng dòng code (`git diff --stat`) + tóm tắt nội dung thay đổi.
- `12_implementation_plan_<suffix>.md` (và `14_walkthrough_<suffix>.md` nếu có): Kế hoạch triển khai chi tiết đã thực hiện.
- `08_tasks_<suffix>.md`: Đánh dấu `[x]` vào các task đã hoàn tất.

### Bước 2 — Append `DONE` vào `05_progress_<suffix>.md`
`view_file` phần cuối `05_progress_<suffix>.md` và dùng `replace_file_content` nối thêm dòng `DONE`:
```markdown
- [YYYY-MM-DDTHH:MM:SS+07:00] [Agent:<Model>] DONE <tóm tắt kết quả & bằng chứng verify>
```

### Bước 3 — Tạo Local Git Restore-Point (CẤM `git push`)
- Kiểm tra `git status -s` và tạo local commit restore-point (hoặc lưu bản `.bak`) theo Rule #19.
- **TUYỆT ĐỐI KHÔNG chạy `git push` (Rule #15).**

### Bước 4 — Tự Kiểm tra 4 Câu hỏi Tri thức (Rule #6.5)
1. Phiên này có phát sinh bài học mới cần ghi vào `lessons.md` không?
2. Nếu có ghi lesson, đã đúng Canonical Format 5 phần `### [YYYY-MM-DD]` và chạy `build_lessons_index.py` chưa?
3. Có vi phạm lesson nào trong suốt quá trình thực hiện phiên này không? (Nếu có, liệt kê trung thực cho User).
4. Đã cập nhật trạng thái vào `active_plans.md` (nếu thay đổi lộ trình dự án) chưa?

### Bước 5 — Chạy Linter Quy trình (`verify_governance.py` — Rule #22)
```bash
python3 /Users/trainguyen/Documents/work/agent/tooling/verify_governance.py --workspace <workspace_name>
```
- Bắt buộc nhận kết quả `PASSED 🟢` trước khi kết thúc câu trả lời. Nếu `FAILED 🔴`, phải bổ sung ngay phần thiếu.

### Bước 6 — Cấu trúc Phản hồi Cuối cùng cho User (Rule #0)
Cuối câu trả lời luôn có 2 mục bắt buộc:
- **Kiểm tra Lesson:** Liệt kê các lesson đã tuân thủ / đã mắc phải và khắc phục trong phiên.
- **Skills & Công cụ đã sử dụng:** Liệt kê danh sách các skill và công cụ đã dùng.
