# Yêu cầu chi tiết: Đối soát và Heal bảng export-jobs bị lệch và noop

## 1. Yêu cầu chi tiết
- **Mục tiêu**: Tìm hiểu tại sao luồng đối soát (Reconciliation) của bảng `export-jobs` (source: `centrallized-export-service.export-jobs`, shadow: `shadow_testexp.export_jobs`) đang bị lệch 1 bản ghi nhưng chạy `recon-heal` luôn trả về `noop`.
- **Hành vi mong muốn**:
  - Khi trigger `recon-heal`, hệ thống phải phát hiện ra bản ghi bị lệch giữa MongoDB (`export-jobs`) và shadow PostgreSQL (`export_jobs`).
  - Thực hiện đồng bộ (heal) bản ghi bị lệch đó sang shadow PostgreSQL thành công.
  - Chạy lại đối soát báo trạng thái khớp (`ok`).

## 2. Tiêu chí hoàn thành (Definition of Done)
- Tìm ra nguyên nhân gốc rễ (Root Cause) tại sao đối soát báo `noop`.
- Thiết kế giải pháp sửa đổi trong code hoặc registry cấu hình.
- Viết tài liệu thiết kế giải pháp vào `09_tasks_solution_recon_export_jobs.md` và trình User phê duyệt trước khi sửa source code.
- Phối hợp với Muscle (hoặc tự thực thi các lệnh phụ trợ) để sửa và verify.
- Chứng minh bằng log hoặc test case rằng bản ghi đã được heal thành công và đối soát không còn báo lệch/noop sai.
- Không phá vỡ các luồng đối soát khác.
