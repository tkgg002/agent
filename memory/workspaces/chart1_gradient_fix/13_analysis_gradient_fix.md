# Phân Tích Gốc Rễ Lỗi Lệch Màu Gauge Chart (Root Cause Analysis)

## 1. Vấn đề thực tế
- Khách hàng có Điểm Tín Dụng (VD: 80 điểm -> Mức độ rủi ro: **RỦI RO THẤP**, màu đại diện: **VÀNG** `#eab308`).
- Tuy nhiên, đường vẽ gauge arc tiến trình lại bị tô bằng `<linearGradient>` chạy từ Đỏ sang Cam sang Vàng sang Xanh trên toàn bộ chiều ngang X.
- Dẫn đến khi stroke phủ 80% độ dài vòng cung, phần chân bên trái của vòng cung lại bị hiển thị màu ĐỎ (Rủi ro rất cao) và CAM, chỉ có phần đuôi mới tới màu VÀNG!
- Khiến kết quả thị giác bị **SAI LỆCH NGHIỆM TRỌNG**: Số điểm hiển thị màu Vàng, nhãn Rủi ro hiển thị màu Vàng, nhưng thanh arc trực quan lại bị Đỏ/Cam ở đoạn đầu!

## 2. Giải pháp triệt để
- Chuẩn hóa hàm `getRiskDetails(score)` để đồng bộ 100% nhãn rủi ro với bảng Thang điểm.
- Thanh arc tiến trình (`progress path`) tô bằng màu chính xác đại diện cho Mức Độ Rủi Ro hiện tại của khách hàng: `stroke="${riskInfo.color}"` (hoặc dải gradient tone-sur-tone thuộc dải màu đó).
- Giữ thanh arc nền (`#e2e8f0`) đóng vai trò là khung đỡ 0 - 100 điểm.
- Nhờ đó:
  - Khách hàng 80 điểm $\rightarrow$ Thanh Gauge Arc màu **VÀNG** `#eab308` đồng bộ 100% với số điểm & nhãn Rủi ro Thấp.
  - Khách hàng 35 điểm $\rightarrow$ Thanh Gauge Arc màu **ĐỎ** `#dc2626` đồng bộ 100% với số điểm & nhãn Rủi ro Cao.
  - Khách hàng 90 điểm $\rightarrow$ Thanh Gauge Arc màu **XANH LÁ** `#16a34a` đồng bộ 100% với số điểm & nhãn Rủi ro Rất Thấp.
