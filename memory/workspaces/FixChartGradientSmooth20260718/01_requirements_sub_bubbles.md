# Yêu cầu: Hiển thị Score-Bubble ở tất cả các Grid con và tối ưu hóa Marker

Yêu cầu chi tiết từ người dùng:
1. **Hiển thị Score-Bubble cho các Sub-bar**:
   - Thêm bong bóng điểm số (`.score-bubble.sub-bubble`) cho từng mục sub-bar trong grid con.
   - Bong bóng điểm số phải hiển thị giá trị điểm thực tế (`item.point`) của sub-bar đó.
   - Bong bóng phải nằm ở vị trí tương ứng với phần trăm tiến trình (`left: pct%`).
   - Màu sắc của bong bóng (gradient nền và mũi tên) phải tương ứng với dải màu của tiến trình được chọn (Red, Orange, Yellow, Green) dựa trên tỉ lệ phần trăm `pct = (item.point / item.max) * 100`.

2. **Tối ưu hóa Marker Line**:
   - `.main-marker-line` (đường kẻ vạch của main bar) phải là một đường line sáng lên (glowing line) bằng cách sử dụng màu nền trắng và hiệu ứng `box-shadow` đồng màu với dải màu hiện tại của bubble.
   - Thêm đường kẻ vạch tương tự `.sub-marker-line` sáng lên đồng màu cho các sub-bar.

3. **Cải tiến Tick Marks**:
   - `.main-bar-wrapper::after` (vạch chia 10%) phải dài hơn và đậm màu hơn để tăng độ tương phản trực quan.

4. **Đảm bảo không phá vỡ Layout**:
   - Điều chỉnh khoảng cách giữa các phần tử trong grid để bong bóng điểm số hiển thị rõ ràng, không bị đè nấp hay làm lệch bố cục grid.
