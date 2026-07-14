# Specs/Requirements: Recon Codebase Audit

## 1. Yêu cầu Nhiệm vụ
- Phân tích mã nguồn hai thư mục của module Recon:
  - Thư mục handler: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon`
  - Thư mục service: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon`
- Xác định và liệt kê chi tiết:
  - Dead code (hàm, struct, biến, hằng không sử dụng)
  - Code rác, dư thừa, trùng lặp (duplicate logic)
  - Điểm bất hợp lý trong logic, code smell, rủi ro bảo mật (như SQL Injection)
  - Sự không nhất quán (inconsistencies) trong coding style, error handling, status strings, naming conventions.
- Tạo báo cáo audit chi tiết và lưu trữ vật lý trong workspace.

## 2. Quy trình Tuân thủ Agent
- Khởi tạo thư mục workspace: `/Users/trainguyen/Documents/work/agent/memory/workspaces/ReconCodebaseAudit20260707`
- Tạo đầy đủ tài liệu workspace:
  - `01_requirements_*.md`: Yêu cầu chi tiết
  - `05_progress_*.md`: Nhật ký tiến độ
  - `08_tasks_*.md`: Danh sách task
  - `12_implementation_plan_*.md`: Kế hoạch triển khai của AI
  - `13_analysis_*.md`: Báo cáo phân tích và kết quả audit
  - `11_report_*.md`: Báo cáo overview các file đã thay đổi/phân tích
- Lưu trữ tri thức (lessons) nếu có bài học mới rút ra.
