# Báo Cáo Thay Đổi - Fix Chart Tick Marks 10%

## Tóm tắt thay đổi
Chúng ta đã thêm vạch chia 10% bằng kẻ xám vào tất cả các thanh tiến trình (`.progress-wrapper`) trong `chart.html` bằng cách sử dụng CSS pseudo-element. Thay đổi này không tác động đến mã HTML hiện tại và tự động co giãn theo chiều rộng của màn hình.

## Các file đã thay đổi
- **File:** [chart.html](file:///Users/trainguyen/Documents/work/chart.html)
- **Số lượng dòng code thay đổi:** ~14 dòng CSS thêm mới.
- **Nội dung thay đổi:**
  Thêm quy tắc CSS `.progress-wrapper::after` để vẽ 9 đường kẻ dọc 1px màu xám mờ (`rgba(0, 0, 0, 0.15)`) chia đều chiều rộng thanh tiến trình thành các phần 10%.

## Kết quả kiểm thử
Đã kiểm tra bằng browser và xác minh trực quan qua screenshot:
- Các vạch chia hiển thị đúng tại các vị trí 10%, 20%, 30%, 40%, 50%, 60%, 70%, 80%, 90%.
- Không bị lệch khi hiển thị trên các kích thước thanh tiến trình khác nhau.
- Không gây ảnh hưởng đến phần fill xanh lá (`.progress-fill`) hay màu nền cảnh báo bên dưới.
