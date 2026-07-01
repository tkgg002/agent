# Kế Hoạch Triển Khai Chi Tiết của AI (Sửa đổi Frontend)

## 1. Mục tiêu phiên làm việc
- Thực thi sửa đổi code Frontend cho 3 file:
  - `src/hooks/useReconStatus.ts`
  - `src/components/ConfirmDestructiveModal.tsx`
  - `src/pages/DataIntegrity.tsx`
  trong dự án `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web` để hỗ trợ trigger heal theo tham số từ UI (chế độ Window hoặc Full-diff có bộ lọc thời gian và giới hạn tối đa 30 ngày).
- Đảm bảo dự án compile/build thành công không có lỗi TypeScript hay linter.

## 2. Kế hoạch chi tiết thực thi của Muscle
- **Bước 1**: Đọc và phân tích kỹ mã nguồn hiện tại của 3 file Frontend cần sửa đổi.
- **Bước 2**: Thực hiện chỉnh sửa từng file bằng `replace_file_content`:
  1. `src/hooks/useReconStatus.ts`: Cập nhật GraphQL/API mutation parameters, thêm `mode`, `startTime`, `endTime` vào payload của `/api/reconciliation/heal`.
  2. `src/components/ConfirmDestructiveModal.tsx`:
     - Thêm prop `isHeal`.
     - Cập nhật hàm `onConfirm` để nhận thêm `mode`, `startTime`, `endTime`.
     - Thêm UI Controls (radio button chọn Window/Full-diff, các trường datetime-local cho startTime và endTime).
     - Thiết lập validation khoảng thời gian chọn (Window mặc định 7 ngày, Full-diff tối đa 30 ngày, validation lỗi nếu startTime > endTime).
  3. `src/pages/DataIntegrity.tsx`:
     - Cập nhật action type và hàm trigger heal modal để truyền thêm prop `isHeal`.
     - Cập nhật `handleConfirm` nhận thêm các tham số `mode`, `startTime`, `endTime` để truyền cho mutation.
- **Bước 3**: Chạy build dự án Frontend bằng `npm run build` hoặc script tương ứng để kiểm tra lỗi compile.
- **Bước 4**: Lưu kết quả phân tích và báo cáo vào `11_report_fe_heal.md` và cập nhật `05_progress_tier2_check.md`.
