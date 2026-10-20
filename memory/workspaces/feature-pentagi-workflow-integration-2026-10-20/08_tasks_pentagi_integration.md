# 08 — Tasks & Execution Checklist: Tích hợp PentAGI vào Hệ thống `/agent` (`pentagi_integration`)

> **Trạng thái hiện tại:** `COMPLETED (S4–S6 — Đã thực thi & vượt qua S5 Audit / DG3)`
> **Workspace:** `feature-pentagi-workflow-integration-2026-10-20` | **Suffix:** `pentagi_integration` | **Tier:** `T2`

---

## Pha 1 — Điều tra & Thiết kế Kiến trúc (`S1 → S3` — Brain Role)

- [x] **T1.1 — S0/S1 Intake:** Khởi tạo workspace `feature-pentagi-workflow-integration-2026-10-20`, lập `01_requirements_pentagi_integration.md` (`tier: T2`), append `START` vào `05_progress_pentagi_integration.md`, lập `08_tasks_pentagi_integration.md`.
- [x] **T1.2 — S2 Read-Only Investigation:** Đối chiếu toàn bộ luồng `S0 → S6` của `/agent` (`GEMINI.md:66-81`, `OPERATOR_MAP.md:1-32`, `skills/*/SKILL.md`, `hooks/agy/agy_guard.py:1-636`, `tooling/verify_governance.py:1-293`) với kiến trúc của `vxcontrol/pentagi` (`README.md:66-1505`).
- [x] **T1.3 — S3 Single-Best-Solution Design:** Soạn `09_tasks_solution_pentagi_integration.md` gồm đủ 5 phần chuẩn `/brain-delegate` (Root Cause/Gap Analysis, Giải pháp Kỹ thuật 4 Trụ cột, Blast Radius & Rollback Plan, Code Demo, Bảng kết quả `DG1/DG2`).
- [x] **T1.4 — Decision Gate `DG1` & `DG2` + Linter S3:** Chạy `laya_gate.py run --stage dg1` và `--stage dg2` đạt `PASS`, chạy `verify_governance.py --workspace feature-pentagi-workflow-integration-2026-10-20 --suffix pentagi_integration --stage S3` đạt `PASSED 🟢`, trình User và **DỪNG CHỜ `APPROVE`**.

---

## Pha 2 — Triển khai Thực thi sau khi có `APPROVE` (`S4 → S6` — Muscle Role)

- [x] **Task T2.1 — Xây dựng `tooling/pentagi_bridge.py` & Unit Tests `tooling/tests/test_pentagi_bridge.py`:**
  - Đã tạo `tooling/pentagi_bridge.py` hỗ trợ `intake` (parse báo cáo Markdown của PentAGI, mask 100% secret/DB URI theo Rule #20, sinh bộ hồ sơ `01_requirements`, `05_progress`, `08_tasks` chống ghi đè theo Rule #17) và `trigger` (kiểm tra ranh giới prefix `_matches_prefix_boundary` với `allowed_pentest_targets` trước khi gọi PentAGI API).
  - Đã viết và chạy 5 unit tests trong `tooling/tests/test_pentagi_bridge.py` (tổng cộng 25/25 tests trong `tooling/tests` đạt `OK`).
- [x] **Task T2.2 — Nâng cấp Hook `H4b` (Khép vòng `S5 Audit → S3 Re-Plan`) & Hook `H6` (`Execution Monitor / Loop Prevention`) trong `hooks/agy/agy_guard.py`:**
  - Đã bổ sung kiểm tra `last_aud_idx > last_sol_idx` trong `handle_pre_write` (`hooks/agy/agy_guard.py:577-598`) chặn cứng sửa source code sau khi đã bước vào `S5 Audit`, bắt buộc quay về `S3` cập nhật `09_tasks_solution_<suffix>.md` và chờ `APPROVE` mới.
  - Đã bổ sung bộ đếm `<conv_id>.<u_idx>.loop` và `ESCALATION_MENTOR_MSG` trong `handle_post_invocation` (`hooks/agy/agy_guard.py:649-668`) khi `checks` thất bại liên tiếp `>= 3` lần (Rule #15).
  - Đã bổ sung `test_29` và `test_30` vào `hooks/agy/tests/test_agy_guard.py` (tổng cộng 30/30 tests đạt `OK`).
- [x] **Task T2.3 — Cập nhật `GEMINI.md`, `skills/qa-agent/SKILL.md`, `skills/muscle-execute/SKILL.md`, `skills/security-agent/SKILL.md` & `skills/workspace-docs/SKILL.md`:**
  - Đã cập nhật đồng bộ quy tắc `Closed-Loop S5 → S3 Re-Plan`, mở rộng `.agent-project.json` với `allowed_pentest_targets` + `pentest_check`, và nâng cấp `/security-agent` lên 5 tầng.
