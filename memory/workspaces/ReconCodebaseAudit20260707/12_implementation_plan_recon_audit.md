# Kế hoạch Triển khai: Recon Codebase Audit

## 1. Phương tiếp cận
1. **Research & Verify**: Đọc rules hệ thống tại `GEMINI.md` và `lessons.md`. Xác minh lỗi vi phạm quy trình Governance ở phiên trước (thiếu workspace files và không làm theo quy trình agent).
2. **Workspace Creation**: Tạo thư mục workspace `ReconCodebaseAudit20260707` và các file tài liệu theo chuẩn Governance: `01_requirements`, `05_progress`, `08_tasks`.
3. **Audit Execution**: Sử dụng kết quả chi tiết từ 2 subagent chạy trước đó để cấu trúc lại báo cáo hoàn chỉnh.
4. **Documentation**:
   - `13_analysis_recon_audit.md`: Ghi chi tiết kết quả phân tích logic, dead code, code rác, các lỗi critical (SQL injection, context key).
   - `11_report_recon_audit.md`: Tổng quan các file đã kiểm tra, số dòng phân tích.
5. **Post-mortem & Lessons**:
   - Tìm hiểu vì sao model không nạp hoặc bỏ qua rules từ `GEMINI.md`.
   - Cập nhật bài học rút ra vào `lessons.md` toàn cục.

## 2. Kế hoạch cụ thể
- **Phase 1: Setup Workspace & Governance Documents**
  - Tạo `01_requirements_recon_audit.md`, `05_progress_recon_audit.md`, `08_tasks_recon_audit.md` (Đang thực hiện).
- **Phase 2: Technical Analysis & Logging**
  - Ghi nhận chi tiết kết quả phân tích vào `13_analysis_recon_audit.md` và `11_report_recon_audit.md`.
- **Phase 3: Update Global Lessons & Project Context**
  - Đọc và append bài học kinh nghiệm về việc tuân thủ quy trình Governance và quản trị workspace vào `agent/memory/global/lessons.md`.
- **Phase 4: Setup Fix Solutions & Design**
  - Tạo `09_tasks_solution_recon_audit.md` mô tả thiết kế kỹ thuật của các bug fix (SQL Injection, Context Keys, ShadowPrefix).
- **Phase 5: Execute Fixes (Muscle Delegation)**
  - Delegate một Muscle subagent để áp dụng các thay đổi code Go một cách an toàn.
  - Chạy `go test ./internal/handler/recon/...` và `go test ./internal/service/recon/...` để xác minh tính ổn định.
- **Phase 6: Final Verification & Process Linter**
  - Chạy `python3 agent/tooling/verify_governance.py --workspace ReconCodebaseAudit20260707` để thông qua Quality Gate.
