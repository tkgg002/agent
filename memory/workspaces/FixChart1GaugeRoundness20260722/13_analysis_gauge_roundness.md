# Phân Tích Kỹ Thuật Làm Tròn Gauge Chart (240°)

## 1. Hiện trạng trong `chart1.html`
Cấu trúc SVG Gauge hiện tại:
```html
<svg class="gauge-svg" viewBox="0 0 180 110">
    <path d="M 21 98 A 69 69 0 0 1 159 98" fill="none" stroke="#e2e8f0" stroke-width="16" stroke-linecap="round"/>
</svg>
```
- Đường kính R=69, góc quét 180° (bán nguyệt phẳng đáy).
- Tổng chiều dài cung `strokeDasharray = 216.7`.

## 2. Giải pháp chuyển sang Gauge 240° (Tròn như ảnh mẫu)
- **Hình học SVG:**
  - `viewBox = "0 0 200 160"`
  - Tâm vòng tròn: `(cx, cy) = (100, 90)`
  - Bán kính: `R = 68`
  - Góc bắt đầu (Bottom-Left): `150°` -> Tọa độ: `x1 = 100 + 68 * cos(150°) ≈ 41.11`, `y1 = 90 + 68 * sin(150°) = 124.00`
  - Góc kết thúc (Bottom-Right): `30°` -> Tọa độ: `x2 = 100 + 68 * cos(30°) ≈ 158.89`, `y2 = 90 + 68 * sin(30°) = 124.00`
  - Đường cong SVG Arc: `M 41.11 124 A 68 68 0 1 1 158.89 124`
  - Góc quét = `240°` (`large-arc-flag = 1`, `sweep-flag = 1`).
  - Chiều dài cung tròn đầy đủ: `L = (240 / 360) * 2 * π * 68 = 284.84`
- **Công thức Dashoffset:**
  - `strokeDasharray = 284.84`
  - `strokeDashoffset = 284.84 - (284.84 * (mainScore / 100))`
- **Căn chỉnh CSS:**
  - Cập nhật `.gauge-wrapper`: `width: 200px; height: 160px;`
  - Cập nhật `.gauge-score-container`: `top: 75px; left: 50%; transform: translateX(-50%);`
