# Yêu Cầu Chi Tiết - Làm Mượt Dải Màu Gradient & Tô Màu Bong Bóng Điểm Số

Nhiệm vụ: Cập nhật dải màu nền của thanh tiến trình (progress bar) trong file `chart.html` để các dải màu chuyển đổi mềm mại (gradient smooth/nối tiếp nhau) thay vì có các vạch màu phân tách cứng nhắc như hiện tại. Đồng thời, cập nhật màu sắc của bong bóng điểm số (`.score-bubble`) tương ứng với màu của phân khúc điểm hiện tại và dọn dẹp các lỗi cú pháp nhỏ.

## Chi tiết yêu cầu:
1. **Làm mượt gradient của `.progress-wrapper`:**
   - Chuyển đổi từ linear-gradient có các stops màu trùng nhau (hard-stops) sang linear-gradient có stops màu nối tiếp nhau mềm mại.
   - Các điểm mốc phân chia màu:
     - 0%: Đỏ (`#f0231b`)
     - 50%: Cam (`#fd8101`)
     - 65%: Vàng (`#fccb00`)
     - 80%: Xanh lá (`#64b90c`)
     - 100%: Xanh lá (`#64b90c`)
2. **Cập nhật màu sắc của `.score-bubble` theo phân khúc:**
   - Bong bóng điểm số sẽ tự động đổi màu theo phân khúc điểm hiện tại (sử dụng CSS Variables và JS động):
     - Dưới hoặc bằng 50: Đỏ (từ `#ff837f` đến `#f0231b`)
     - Từ 51 đến 65: Cam (từ `#ffb875` đến `#fd8101`)
     - Từ 66 đến 80: Vàng (từ `#ffea7c` đến `#fccb00`)
     - Trên 80: Xanh lá (từ `#a3e75e` đến `#64b90c`)
   - Mũi tên của bong bóng (`.score-bubble::after`) cũng phải thay đổi màu sắc tương ứng để tạo cảm giác liền mạch.
3. **Sửa lỗi cú pháp trong `chart.html`:**
   - Dọn dẹp ký tự thừa `=` tại dòng 440 (`</div>=`).
4. **Kiểm thử trực quan:**
   - Sử dụng `browser_subagent` để mở và kiểm tra giao diện của `chart.html`.
