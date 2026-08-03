# Yêu cầu: Cập nhật Gauge Chart trong chart1.html cho tròn hơn (240° Gauge Arc)

## 1. Mục tiêu
- Thay đổi biểu đồ Điểm Tín Dụng Tổng Hợp (Gauge Chart) từ bán nguyệt 180° (phẳng đáy) thành biểu đồ vòng cung tròn 240° (dạng gauge/speedometer chuẩn như ảnh mẫu của người dùng).
- Đảm bảo điểm số và chữ `/100` hiển thị căn giữa chuẩn xác trong lòng gauge.
- Cập nhật gradient màu rủi ro mềm mại từ Đỏ -> Cam -> Vàng -> Xanh lá.
- Điều chỉnh CSS container `.gauge-wrapper` và `.gauge-svg` để phù hợp với chiều cao mới của gauge tròn 240°.

## 2. Tiêu chí nghiệm thu (DoD)
- [ ] Gauge Arc hiển thị dạng 240° tròn, mềm mại ở cả 2 đầu (stroke-linecap round).
- [ ] Điểm số mainScore và /100 căn giữa chuẩn xác.
- [ ] Hiển thị mượt mà trên tất cả 3 card khách hàng trong `chart1.html`.
- [ ] Không làm phá vỡ layout hay các card khác trong dashboard.
