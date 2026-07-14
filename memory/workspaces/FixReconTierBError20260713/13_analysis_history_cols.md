# Phân tích kỹ thuật: Bổ sung cột Lệch & Thời gian xử lý vào FE

## 1. Chi tiết thay đổi trong `ReconPipelineGrid.tsx`
- **Sửa đổi bảng Table trong component `DrillDown`:**
  - Cấu hình thuộc tính `scroll: { x: 920, y: 260 }` cho `Table` để bật thanh cuộn ngang khi giao diện Drawer (width 860) bị thu hẹp hoặc tổng chiều rộng các cột vượt quá khung nhìn, đảm bảo không bị vỡ/tràn chữ.
  - Bổ sung cột **Lệch**:
    - Sử dụng `fmtDrift(r.diff != null ? -r.diff : null)`.
    - Việc truyền `-r.diff` nhằm mục đích đảo chiều từ góc nhìn nguồn-đích (source_count - dest_count) của report gốc sang dạng đích-nguồn (dest_count - source_count) giống như `driftAB` và `driftBC` ở master grid.
    - Màu đỏ biểu thị thiếu dữ liệu ở trạm sau (`-X (thiếu)`).
    - Màu vàng biểu thị thừa dữ liệu ở trạm sau (`+X (thừa)`).
    - Màu xanh lá hiển thị `0` nếu hai trạm khớp.
  - Bổ sung cột **Thời gian xử lý**:
    - Lấy dữ liệu từ trường `duration_ms`.
    - Định dạng thông minh: dưới 1000ms hiển thị `ms` (ví dụ `320ms`), từ 1000ms trở lên hiển thị `s` (ví dụ `1.45s`).

## 2. Giải quyết lỗi biên dịch tsc tồn tại trước đó
- File `src/components/ExecuteHealModal.tsx` phát sinh lỗi do biến `EMPTY_ARRAY` được khai báo nhưng không sử dụng:
  `src/components/ExecuteHealModal.tsx:10:7 - error TS6133: 'EMPTY_ARRAY' is declared but its value is never read.`
- Chúng tôi đã loại bỏ dòng code thừa này để đưa dự án về trạng thái biên dịch thành công 100%.

## 3. Kết quả xác minh build
- Chạy lệnh `npm run build` trong `cdc-cms-web` thành công hoàn toàn.
