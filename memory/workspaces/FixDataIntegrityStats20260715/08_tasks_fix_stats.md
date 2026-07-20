# Danh sách Task chi tiết: Sửa lỗi thống kê UI

## Phase 1: Phân tích & Lập phương án
- [x] Phát hiện nguyên nhân gốc rễ và vị trí lỗi trong code frontend (`DataIntegrity.tsx` và `ReconPipelineGrid.tsx`)
- [/] Lập kế hoạch chi tiết (Implementation Plan) để sửa đổi code
- [ ] Xin phê duyệt từ User

## Phase 2: Thực thi sửa đổi
- [ ] Export hàm `buildPipelines` và `overallStatus` từ `ReconPipelineGrid.tsx`
- [ ] Import và sử dụng các hàm trên trong `DataIntegrity.tsx` để tính toán chính xác các chỉ số thống kê
- [ ] Khởi chạy frontend ở local để kiểm tra hiển thị

## Phase 3: Kiểm thử & Xác thực
- [ ] Xác nhận các chỉ số Tổng bảng, Khớp, Lệch hiển thị đúng số lượng bảng thực tế
- [ ] Chạy linter quy trình và báo cáo kết quả
