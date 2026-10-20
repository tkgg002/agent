# 08 — Tasks & Execution Checklist: Tích hợp PentAGI vào Hệ thống `/agent` (`pentagi_integration`)

> **Trạng thái hiện tại:** `PROPOSED (S3 — Chờ User APPROVE)`
> **Workspace:** `feature-pentagi-workflow-integration-2026-10-20` | **Suffix:** `pentagi_integration` | **Tier:** `T2`

---

## Pha 1 — Điều tra & Thiết kế Kiến trúc (`S1 → S3` — Brain Role)

- [x] **T1.1 — S0/S1 Intake:** Khởi tạo workspace `feature-pentagi-workflow-integration-2026-10-20`, lập `01_requirements_pentagi_integration.md` (`tier: T2`), append `START` vào `05_progress_pentagi_integration.md`, lập `08_tasks_pentagi_integration.md`.
- [x] **T1.2 — S2 Read-Only Investigation:** Đối chiếu toàn bộ luồng `S0 → S6` của `/agent` (`GEMINI.md:66-81`, `OPERATOR_MAP.md:1-32`, `skills/*/SKILL.md`, `hooks/agy/agy_guard.py:1-636`, `tooling/verify_governance.py:1-293`) với kiến trúc của `vxcontrol/pentagi` (`README.md:66-1505`).
- [x] **T1.3 — S3 Single-Best-Solution Design:** Soạn `09_tasks_solution_pentagi_integration.md` gồm đủ 5 phần chuẩn `/brain-delegate` (Root Cause/Gap Analysis, Giải pháp Kỹ thuật 3 Trụ cột, Blast Radius & Rollback Plan, Code Demo, Bảng kết quả `DG1/DG2`).
- [x] **T1.4 — Decision Gate `DG1` & `DG2` + Linter S3:** Chạy `laya_gate.py run --stage dg1` và `--stage dg2` đạt `PASS`, chạy `verify_governance.py --workspace feature-pentagi-workflow-integration-2026-10-20 --suffix pentagi_integration --stage S3` đạt `PASSED 🟢`, trình User và **DỪNG CHỜ `APPROVE`**.

---

## Pha 2 — Triển khai Thực thi sau khi có `APPROVE` (`S4 → S6` — Muscle Role)

### Task T2.1 — Xây dựng `tooling/pentagi_bridge.py` & Unit Tests `tooling/tests/test_pentagi_bridge.py`
- **[Mô tả]:**
  - Tạo CLI cầu nối `agent/tooling/pentagi_bridge.py` hỗ trợ 2 lệnh:
    1. `trigger`: Gọi GraphQL API `/api/v1/graphql` (`createFlow`) và REST API `/api/v1/flows/<id>` của PentAGI để chạy kiểm thử xâm nhập có giới hạn phạm vi (scoped DAST) cho các endpoint vừa thay đổi ở chặng `S5`.
    2. `intake`: Đọc báo cáo Markdown xuất từ PentAGI (Nightly Flow Report hoặc qua API), tự động mask mọi chuỗi nhạy cảm theo regex `SECRET` (Rule #20), bóc tách danh sách lỗ hổng (`Critical/High/Medium`) kèm PoC và sinh bộ hồ sơ `S1 Intake` (`01_requirements_<suffix>.md`, `05_progress_<suffix>.md`, `08_tasks_<suffix>.md`) để đưa vào luồng `S2 Reproduce → S3 Solution (DG1/DG2) → Wait APPROVE`.
- **[Context & Logs]:**
  - `skills/security-agent/SKILL.md:17-49` (4 tầng kiểm tra tĩnh hiện tại).
  - `hooks/agy/agy_guard.py:41-46` (biểu thức chính quy `SECRET` bảo vệ `agent/memory/`).
  - `PentAGI README.md:1175-1185, 1264-1348` (đặc tả REST `/api/v1/flows` & GraphQL `createFlow` với Bearer token).
- **[Definition of Done]:**
  - Unit tests trong `tooling/tests/test_pentagi_bridge.py` kiểm tra đầy đủ: (a) parse báo cáo Markdown & tạo hồ sơ `01/05/08` chuẩn format; (b) mask 100% secret/token/password trong báo cáo trước khi ghi vào workspace; (c) kiểm tra chặn URL ngoài danh sách `allowed_targets` (chống bắn nhầm vào production khi chưa uỷ quyền); (d) chạy `python3 -m unittest discover -s tooling/tests -p 'test_*.py' -v` đạt `PASS`.

### Task T2.2 — Nâng cấp Hook `H4b` (Khép vòng `S5 Audit → S3 Re-Plan`) & Hook `H6` (`Execution Monitor / Loop Prevention`) trong `hooks/agy/agy_guard.py`
- **[Mô tả]:**
  - **(a) Khép kín vòng lặp ngược `S5 Audit → S3 Re-Plan` trong `H4b` (`hooks/agy/agy_guard.py:530-580`):**
    - Hiện tại `H4b` chỉ kiểm tra xem sau `09_tasks_solution` (`sol[-1]`) đã có `APPROVE` hay chưa, dẫn tới khi sang pha `S5 Audit` (đã tạo/cập nhật `audit_report_<suffix>.md`), quyền sửa source code vẫn mở từ lệnh `APPROVE` cũ ở S3.
    - Nâng cấp `H4b`: nếu phát hiện đã bước vào pha `S5 Audit` (`last_audit_idx > last_sol_idx`, tức là đã chạm `audit_report_` sau lần lập `09_tasks_solution` gần nhất), **chặn cứng (`decision: "deny"`)** mọi lệnh sửa source code! Bắt buộc Agent phải **quay ngược về bước S3** cập nhật lại `09_tasks_solution_<suffix>.md` (`last_sol_idx > last_audit_idx`) + `08_tasks_<suffix>.md`, chạy lại `DG1/DG2` (`PASS` mới hơn `last_sol_idx`) và chờ **lệnh `APPROVE` mới của User** thì mới mở khóa cho sửa source code ở S4.
  - **(b) Học hỏi cơ chế `Execution Monitoring (Adviser/Mentor)` của PentAGI vào `H6` (`hooks/agy/agy_guard.py:596-636`):**
    - Lưu trạng thái số lần `checks` thất bại liên tiếp trong `<state_dir>/<conv_id>.<u_idx>.loop`.
    - Khi đạt ngưỡng `>= 3` lần lặp thất bại trong cùng một lượt (khớp `GEMINI.md:108` — Rule #15 Escalation), tự động tiêm cảnh báo `🚨 ESCALATION MENTOR (Rule #15 - Loop Prevention)` buộc Agent dừng sửa mò và quay về Re-plan cùng User.
- **[Context & Logs]:**
  - `GEMINI.md:78`, `skills/qa-agent/SKILL.md:71`, `skills/muscle-execute/SKILL.md:66` (quy định hiện tại đang cho phép tự sửa lỗi tại S5 nếu trong phạm vi mà không bắt quay lại S3 làm Plan).
  - `hooks/agy/agy_guard.py:570-579` (`H4b` check `sol[-1]` và `APPROVE`).
  - `hooks/agy/agy_guard.py:596-636` (`handle_post_invocation`).
- **[Definition of Done]:**
  - Bổ sung unit tests vào `hooks/agy/tests/test_agy_guard.py` xác minh: (1) sau khi ghi `audit_report_*.md` (S5), mọi lệnh sửa source code bị `H4b` `deny` cho tới khi cập nhật lại `09_tasks_solution_*.md` và có `APPROVE` mới của User; (2) khi `checks` fail 3 lần liên tiếp thì `H6` trả về thông điệp Escalation; toàn bộ test suite `hooks/agy` đạt `PASS`.

### Task T2.3 — Cập nhật `GEMINI.md`, `skills/qa-agent/SKILL.md`, `skills/muscle-execute/SKILL.md`, `skills/security-agent/SKILL.md` & `skills/workspace-docs/SKILL.md`
- **[Mô tả]:**
  - Cập nhật `GEMINI.md:78`, `skills/qa-agent/SKILL.md:71`, `skills/muscle-execute/SKILL.md:66`: Thay thế quy tắc "tự sửa lỗi trong phạm vi ở S5" bằng quy tắc **Closed-Loop S5 → S3 Re-Plan**: khi `S5 Audit` (hoặc PentAGI DAST) phát hiện bất kỳ lỗi/lỗ hổng nào, **CẤM tự ý sửa source code ngay tại S5**; bắt buộc ghi vào `audit_report_<suffix>.md`, quay ngược về **S3** cập nhật Plan (`09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md`), chạy lại `DG1/DG2` và chờ User `APPROVE` mới được quay lại `S4` thực thi.
  - Cập nhật `skills/workspace-docs/SKILL.md:52-69` & `skills/security-agent/SKILL.md:17-49`: Bổ sung `pentest_check`, `allowed_pentest_targets` và Tầng 5 Dynamic Pentest Gate (`pentagi_bridge.py`).
- **[Context & Logs]:**
  - `GEMINI.md:74-80`, `skills/qa-agent/SKILL.md:57-72`, `skills/muscle-execute/SKILL.md:63-68`, `skills/security-agent/SKILL.md:1-50`, `skills/workspace-docs/SKILL.md:52-70`.
- **[Definition of Done]:**
  - Tài liệu rule & skill đồng nhất 100%, không chứa thông tin hardcode của bất kỳ dự án cụ thể nào (Rule #3 Agnostic Core), vượt qua `DG3` (`laya_gate.py run --stage dg3`) và `audit_report_pentagi_integration.md` + `verify_governance.py` đạt `PASSED 🟢`.
