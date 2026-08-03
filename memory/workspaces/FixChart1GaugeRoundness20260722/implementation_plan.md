# Implementation Plan - Fix Gauge Chart Roundness (240°)

## Proposed Changes
### [chart1.html](file:///Users/trainguyen/Documents/work/chart1.html)
- Thay đổi cấu trúc SVG path của Gauge Chart từ bán nguyệt 180° (`M 21 98 A 69 69 0 0 1 159 98`) sang vòng cung 240° (`M 41.11 124 A 68 68 0 1 1 158.89 124`).
- Cập nhật chiều dài vòng cung `totalArcLength = 284.84` và tính toán `strokeDashoffset` tương ứng.
- Điều chỉnh kích thước CSS `.gauge-wrapper` và `.gauge-svg` sang `width: 200px; height: 160px;`, căn vị trí điểm số `top: 70px;`.

## Verification Plan
- Mở file HTML và đối chiếu với thiết kế gauge tròn.
