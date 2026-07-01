# Plan: Hide Disabled Master Tables in Data Integrity

## Active Plan & Checklist
- [ ] 1. Khởi tạo workspace và phân tích yêu cầu (DONE)
- [ ] 2. Định vị file và đoạn mã cần sửa đổi:
  - Frontend: `DataIntegrity.tsx` và `ReconPipelineGrid.tsx`
- [ ] 3. Thực hiện sửa đổi code frontend:
  - Fetch `masters` và `schedules` tại `DataIntegrity.tsx` (hoặc tái sử dụng cache).
  - Lọc `reportList` ngay tại `DataIntegrity.tsx` để ẩn các bảng có master sync bị tắt.
  - Cập nhật số liệu thống kê ở các card.
- [ ] 4. Kiểm tra cục bộ:
  - Compile build frontend và verify logic.
- [ ] 5. Chạy Security agent review.
- [ ] 6. Nghi nghiệm thu và hoàn tất workspace docs.
