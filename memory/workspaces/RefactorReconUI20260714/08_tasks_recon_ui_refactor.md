# Danh sách Task Chi tiết

## Phase 1: Chuẩn bị & Thiết kế Kế hoạch
- [x] Đọc GEMINI.md và lessons.md để nắm bắt quy trình vận hành và tránh lỗi lặp lại.
- [x] Khởi tạo workspace RefactorReconUI20260714 và các tài liệu bắt buộc.
- [x] Soạn thảo kế hoạch thực hiện chi tiết 12_implementation_plan_recon_ui_refactor.md.

## Phase 2: Thực thi
- [x] Cập nhật ConfirmDestructiveModal.tsx:
  - [x] Mặc định checkMode là '2h'.
  - [x] Ẩn lựa chọn Deep Check bằng style display: none.
  - [x] Ẩn UI chọn chặng đối soát (segment selector).
- [x] Cập nhật ExecuteHealModal.tsx:
  - [x] Thêm logic lọc `reports` theo `segment` prop.
  - [x] Thêm logic lọc `healedReports` theo `segment` prop.
- [x] Rà soát an toàn thông qua /security-agent hoặc review kiểm tra cục bộ.

## Phase 3: Kiểm tra & Nghiệm thu
- [x] Chạy frontend để kiểm tra modal (hoặc verify build FE thành công).
- [x] Chạy linter quy trình verify_governance.py.
- [x] Hoàn thành walkthrough.md.
