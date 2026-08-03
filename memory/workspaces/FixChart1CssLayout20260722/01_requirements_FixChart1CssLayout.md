# Requirements: Fix Chart1 CSS Layout Issues

## 1. Yêu cầu chi tiết
Cập nhật file `/Users/trainguyen/Documents/work/chart1.html` để khắc phục 2 vấn đề UI/Layout:

1. **Pill badge không bị dâng lên quá cao**:
   - Chỉnh `.card-header-pill`: `top: 0; left: 0;` dính sát mép trên khung Card.
   - Giảm padding top của `.card` xuống `28px 20px 16px 20px;`.

2. **Cụm MỨC ĐỘ RỦI RO không bị sọc tràn ra ngoài**:
   - Thu gọn gauge-wrapper (`width: 175px; height: 140px;`).
   - Điều chỉnh `.risk-scale-card` padding (`10px 12px;`) và cell padding (`10px`).
   - Giúp toàn bộ bảng thang điểm nằm gọn gàng 100% bên trong Card 2 mà không bị che đè hay tràn lề.

## 2. Definition of Done
- Cập nhật đúng đoạn CSS trong `chart1.html`.
- Kiểm tra lại file `chart1.html` sau khi cập nhật đảm bảo các class CSS được thay thế chính xác.
