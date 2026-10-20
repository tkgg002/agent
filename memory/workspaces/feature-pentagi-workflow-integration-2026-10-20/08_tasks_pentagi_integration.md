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

### Task T2.2 — Nâng cấp Hook `H6` (`hooks/agy/agy_guard.py`) với cơ chế "Execution Monitor / Loop Prevention" (Rule #15)
- **[Mô tả]:**
  - Học hỏi cơ chế `Execution Monitoring (Adviser/Mentor)` của PentAGI (`PentAGI README.md:485-496`), nâng cấp `handle_post_invocation` trong `hooks/agy/agy_guard.py:471-511`:
    - Lưu trạng thái số lần `checks` thất bại liên tiếp hoặc sửa lặp lại cùng một file mà vẫn lỗi trong `<state_dir>/<conv_id>.<u_idx>.loop`.
    - Khi đạt ngưỡng `>= 3` lần lặp thất bại trong cùng một lượt (khớp `GEMINI.md:107` — Rule #15 Escalation), tự động tiêm cảnh báo `🚨 ESCALATION MENTOR (Rule #15 - Loop Prevention)` buộc Agent dừng sửa mò và báo cáo User để re-plan.
- **[Context & Logs]:**
  - `GEMINI.md:107`: *"Escalation (>3 lần lặp): Thử sửa cùng 1 lỗi >3 lần thất bại → DỪNG NGAY, báo cáo để re-plan."*
  - `hooks/agy/agy_guard.py:471-511` (`handle_post_invocation`).
- **[Definition of Done]:**
  - Bổ sung test case vào `hooks/agy/tests/test_agy_guard.py` xác minh khi `checks` fail 3 lần liên tiếp thì `handle_post_invocation` trả về thông điệp Escalation; toàn bộ test suite `hooks/agy` đạt `PASS`.

### Task T2.3 — Cập nhật `skills/security-agent/SKILL.md` & `skills/workspace-docs/SKILL.md`
- **[Mô tả]:**
  - Cập nhật `skills/workspace-docs/SKILL.md:52-69`: Bổ sung trường tùy chọn `pentest_check` và `allowed_pentest_targets` vào chuẩn `.agent-project.json`.
  - Cập nhật `skills/security-agent/SKILL.md:17-49`: Bổ sung **Tầng 5: Dynamic Pentest Gate (DAST via `pentagi_bridge.py`)** ở chặng S5 và quy trình **Nightly Pentest Intake (`S1`)**.
- **[Context & Logs]:**
  - `skills/security-agent/SKILL.md:1-50`, `skills/workspace-docs/SKILL.md:52-70`.
- **[Definition of Done]:**
  - Tài liệu skill rõ ràng, không chứa thông tin hardcode của bất kỳ dự án cụ thể nào (tuân thủ Rule #3 Agnostic Core), vượt qua `DG3` (`laya_gate.py run --stage dg3`) và `audit_report_pentagi_integration.md` + `verify_governance.py` đạt `PASSED 🟢`.
