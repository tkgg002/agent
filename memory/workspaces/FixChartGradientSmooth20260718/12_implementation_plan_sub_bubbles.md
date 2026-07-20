# Kế hoạch triển khai: Tích hợp Score-Bubble cho các Sub-bars và Tối ưu hóa Marker Lines

Mục tiêu: Đảm bảo giao diện hiển thị bong bóng điểm số động và các vạch kẻ sáng lên (glowing marker lines) đồng bộ trên cả main bar và tất cả các sub-bar trong grid con.

## Đề xuất Thay đổi

### 1. Style CSS trong [chart.html](file:///Users/trainguyen/Documents/work/chart.html)
- **Làm trong suốt progress-fill**: Thay thế màu nền xanh lá cứng (`#00a651`) của `.progress-fill` thành `transparent` để lộ dải màu gradient từ `.progress-wrapper`.
- **Cải tiến `.main-marker-line`**:
  - Đổi màu nền sang trắng (`#fff`).
  - Thêm `box-shadow: 0 0 8px var(--bubble-color-end, #64b90c)` để tạo hiệu ứng phát sáng đồng điệu với màu của bong bóng điểm số.
  - Tăng độ rộng lên `1.5px` hoặc `2px`.
- **Tối ưu `.main-bar-wrapper::after` (Tick marks)**:
  - Thay đổi `background-image` thành vạch chia đậm màu hơn (`rgba(0, 0, 0, 0.4)`) và rộng `2px` thay vì `1px`.
  - Tăng kích thước vạch chia lên `14px` để kéo dài hơn.
- **Thêm Style mới cho các Sub-bars**:
  - `.sub-bar-container`: Thiết lập `position: relative` và `margin-top: 40px` để chừa khoảng trống cho các bong bóng điểm số phía trên mỗi thanh sub-bar.
  - `.score-bubble.sub-bubble`: Bong bóng thu nhỏ, `top: -45px`, `font-size: 20px`, `padding: 3px 10px` và arrow nhỏ hơn.
  - `.sub-marker-line`: Kẻ dọc chỉ điểm sáng mờ, `position: absolute`, `top: -10px`, `bottom: 0`, `width: 1.5px`, nền trắng và `box-shadow` phát sáng theo màu bubble của sub-bar đó.

### 2. Logic JS trong `renderCharts` của [chart.html](file:///Users/trainguyen/Documents/work/chart.html)
- Tính toán màu sắc bong bóng động cho từng sub-bar giống như main-bar dựa trên tỉ lệ phần trăm `pct` của sub-bar đó:
  - `pct <= 50`: Đỏ (`#ff837f` đến `#f0231b`)
  - `pct <= 65`: Cam (`#ffb875` đến `#fd8101`)
  - `pct <= 80`: Vàng (`#ffea7c` đến `#fccb00`)
  - `pct > 80`: Xanh lá (`#a3e75e` đến `#64b90c`)
- Sửa đổi template chuỗi của `subBarsHtml`:
  - Loại bỏ phần tử `.sub-marker` cũ nằm bên trong `.sub-bar-wrapper` để tránh trùng lặp.
  - Tạo cấu trúc `.sub-bar-container` chứa `.score-bubble.sub-bubble`, `.sub-marker-line` và `.progress-wrapper.sub-bar-wrapper`.
  - Truyền các biến CSS `--bubble-color-start` và `--bubble-color-end` động vào inline style của `.sub-bar-container` để áp dụng riêng biệt cho từng sub-bar.

## Kế hoạch Xác minh
- Mở file HTML trực tiếp trên trình duyệt để kiểm tra trực quan giao diện.
- Kiểm tra xem:
  - Các bong bóng điểm số có hiển thị ở tất cả các sub-bar con không.
  - Màu sắc bong bóng có khớp với tỉ lệ phần trăm của từng mục không (ví dụ: 15/20 là 75% -> màu Vàng).
  - Vạch kẻ dọc có phát sáng đúng màu và kéo dài xuống dưới không.
  - Vạch chia 10% trên thanh chính có dài và đậm màu hơn không.
