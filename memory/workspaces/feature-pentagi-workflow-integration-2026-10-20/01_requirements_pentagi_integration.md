# 01 — Requirements: Phân tích Workflow `/agent` Hiện tại & Thiết kế Tích hợp PentAGI (`pentagi_integration`)

tier: T2
workspace: feature-pentagi-workflow-integration-2026-10-20
suffix: pentagi_integration
environment: local

## 1. Bối cảnh & Yêu cầu của User
User cung cấp thông tin về hệ thống pentest tự động đa tác tử **[vxcontrol/pentagi](https://github.com/vxcontrol/pentagi)** và yêu cầu:
> `/agent em xem workflow của mình hiện tại là gì, tích hợp thêm cái này thì ntn`

## 2. Danh sách Yêu cầu Chi tiết (`R1` – `R5`)

| Mã Req | Mô tả Yêu cầu | Tiêu chí Nghiệm thu (Acceptance Criteria) |
| :--- | :--- | :--- |
| **R1** | **Hệ thống hóa Workflow hiện tại của `/agent` (`S0 → S6`)** | Trình bày đầy đủ, chính xác kèm dẫn chứng `file:line` về luồng `S0 → S6` (`GEMINI.md:66-81`), 10 Core Skills (`OPERATOR_MAP.md:5-18`), 7 Lifecycle Hooks `H1–H7` (`hooks/agy/agy_guard.py:7-22`), 3 cổng Decision Gate `DG1–DG3` (`skills/laya-router/SKILL.md:19-26`) và 8 Quality Gates `G1–G8` (`skills/qa-agent/SKILL.md:18-103`). |
| **R2** | **Chỉ ra khoảng trống (Gap Analysis) giữa `/agent` và PentAGI** | Xác định rõ 3 khoảng trống thực tế: (1) `/security-agent` (`skills/security-agent/SKILL.md:17-38`) hiện chỉ quét tĩnh SAST/Regex, thiếu kiểm thử xâm nhập động (DAST/Black-box runtime); (2) Chưa có cầu nối tự động chuyển báo cáo PentAGI chạy đêm thành hồ sơ `S1 Intake` có lọc False Positive & che chắn Secret (Rule #20); (3) Rule #15 Escalation (`>3 lần lặp` — `GEMINI.md:107`) chưa được cưỡng chế tự động bằng bộ đếm trong Hook `H6` như cơ chế `Execution Monitor` của PentAGI. |
| **R3** | **Thiết kế Tích hợp PentAGI vào `/security-agent` & `.agent-project.json` (2 Chế độ: Inline S5 & Overnight Batch S1)** | Thiết kế **1 phương án tối ưu duy nhất** (`09_tasks_solution_pentagi_integration.md`): (a) Mở rộng `.agent-project.json` (`skills/workspace-docs/SKILL.md:52-69`) hỗ trợ cấu hình `pentest_check` (giữ `/agent` 100% project-agnostic theo Rule #3 / ADR-07); (b) Xây dựng `tooling/pentagi_bridge.py` kết nối REST/GraphQL API của PentAGI cho cả **Chế độ Inline S5** (quét nhanh các route sửa trong `git diff`) và **Chế độ Overnight Batch → S1 Intake** (kéo báo cáo đêm, mask secret theo Rule #20, sinh `01_requirements` + `08_tasks`, giữ nguyên kỷ luật `S2/S3/DG1/DG2` chờ User `APPROVE` mới sửa code). |
| **R4** | **Tích hợp cơ chế "Execution Monitor / Loop Prevention" của PentAGI vào Hook `H6` (`agy_guard.py`)** | Thiết kế nâng cấp `handle_post_invocation` (`hooks/agy/agy_guard.py:596-636`) để theo dõi số lần lặp check/test lỗi liên tiếp (`>= 3 lần` theo Rule #15) và tự động tiêm thông điệp `<escalation_mentor>` yêu cầu dừng sửa mò, báo cáo User để re-plan. |
| **R5** | **Khép kín Vòng lặp Ngược `S5 Audit → S3 Re-Plan → DG1/DG2 → Wait APPROVE` (`H4b` & `GEMINI.md`)** | Giải quyết triệt để điểm hở hiện tại ở bước `S5 Audit` (vốn đang cho phép tự sửa lỗi trong phạm vi tại `GEMINI.md:78`, `skills/qa-agent/SKILL.md:71` và `H4b` tại `agy_guard.py:570-579` không khóa lại quyền sửa code sau khi đã có `APPROVE` ở S3 đầu tiên): thiết kế chốt chặn `S5 → S3 Re-plan Loop` bắt buộc khi `S5 Audit` phát hiện lỗi phải quay ngược về cập nhật Plan (`08_tasks` + `09_tasks_solution`), chạy lại `DG1/DG2` và chờ User `APPROVE` mới được sửa code. |
| **R6** | **Tuân thủ Kỷ luật Governance (`S1 → S3 → DG1/DG2 → Wait APPROVE`)** | Hoàn thành hồ sơ S1 (`01`, `05`, `08`), S3 (`09_tasks_solution_pentagi_integration.md`), chạy `laya_gate.py` đạt `PASS` cho cả `DG1`, `DG2`, `DG3`, chạy `verify_governance.py --stage S3` đạt `PASSED 🟢`, và giữ trạng thái **Propose-Only** chờ User `APPROVE`. |
