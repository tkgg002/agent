# Phân tích kỹ thuật: Bổ sung cột Lệch & Thời gian xử lý vào FE

## 1. Chi tiết thay đổi trong `ReconPipelineGrid.tsx`
- **Tối ưu hóa hiển thị trong `DrillDown`:**
  - Thay vì sử dụng các cột `Lệch` và `Thời gian xử lý` riêng lẻ gây tốn diện tích và phải bật scroll ngang, chúng tôi đã hợp nhất các thông tin này vào cột `Chi tiết`.
  - Định dạng thống nhất của cột `Chi tiết` mới là: `"[Thời gian xử lý] : [Nguồn] → [Đích] ([Lệch])"` kèm các thông tin bổ sung nếu có (thiếu, stale, đã heal).
  - Ví dụ thực tế: `85ms : 2,713,267 → 2,713,279 (-12)`.
  - Định dạng hiển thị chênh lệch (drift) trong ngoặc đơn:
    - Sử dụng hướng tính drift: `v = -r.diff` để đồng bộ với Master Grid.
    - Màu đỏ biểu thị thiếu dữ liệu ở trạm sau (`(v)` âm).
    - Màu vàng biểu thị thừa dữ liệu ở trạm sau (`(+v)` dương).
    - Màu xanh hiển thị khớp (`(0)`).
  - Thời gian xử lý (`duration_ms`) được định dạng thông minh (sử dụng đơn vị `ms` hoặc `s` tùy giá trị).

## 2. Giải quyết lỗi biên dịch tsc tồn tại trước đó
- File `src/components/ExecuteHealModal.tsx` phát sinh lỗi do biến `EMPTY_ARRAY` được khai báo nhưng không sử dụng:
  `src/components/ExecuteHealModal.tsx:10:7 - error TS6133: 'EMPTY_ARRAY' is declared but its value is never read.`
- Chúng tôi đã loại bỏ dòng code thừa này để đưa dự án về trạng thái biên dịch thành công 100%.

## 3. Kết quả xác minh build
- Chạy lệnh `npm run build` trong `cdc-cms-web` thành công hoàn toàn.
