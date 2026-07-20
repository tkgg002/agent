# Danh sách nhiệm vụ: Tích hợp Score-Bubble cho các Sub-bars và Tối ưu hóa Marker Lines

- [ ] Nghiên cứu cấu trúc code HTML hiện tại của `chart.html` để lập kế hoạch chèn phần tử
- [ ] Thiết kế và bổ sung các style CSS mới:
  - [ ] `.sub-bar-container`: Thiết lập `position: relative` và `margin-top` để tạo khoảng trống cho sub-bubble.
  - [ ] `.score-bubble.sub-bubble`: Định nghĩa kích thước bong bóng nhỏ hơn, điều chỉnh `top`, font size, padding và arrow.
  - [ ] `.sub-marker-line`: Định nghĩa vạch kẻ thẳng đứng sáng lên tương ứng với vị trí điểm số của từng sub-bar.
  - [ ] `.main-marker-line`: Sửa đổi style để tạo hiệu ứng phát sáng (glowing) với viền mờ màu sắc đồng điệu.
  - [ ] `.main-bar-wrapper::after`: Tăng chiều dài và độ dày của vạch chia 10%.
- [ ] Cập nhật hàm `renderCharts` trong mã nguồn `chart.html`:
  - [ ] Thêm logic tính toán dải màu cho từng sub-bar dựa trên tỉ lệ phần trăm `pct`.
  - [ ] Thay đổi cấu trúc HTML của từng grid item để bọc progress bar và bubble trong `.sub-bar-container`.
  - [ ] Chèn `.score-bubble.sub-bubble` và `.sub-marker-line` vào template của sub-bar.
- [ ] Xác minh kết quả giao diện bằng cách tải lại trang và kiểm tra trực quan.
