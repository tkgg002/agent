# Kế Hoạch Triển Khai - linearGradient Fix

## Nội dung thay đổi
1. File target: `/Users/trainguyen/Documents/work/chart1.html`
2. Đưa khối `<linearGradient>` vào trong thẻ `<defs>` chuẩn SVG1.1 / SVG2.
3. Thiết lập các stop color:
   - `offset="0%"` -> `#dc2626`
   - `offset="49%"` -> `#dc2626`
   - `offset="50%"` -> `#f97316`
   - `offset="64%"` -> `#f97316`
   - `offset="65%"` -> `#eab308`
   - `offset="84%"` -> `#eab308`
   - `offset="85%"` -> `#16a34a`
   - `offset="100%"` -> `#16a34a`
4. Căn chỉnh `x1="6.7%"`, `x2="93.3%"` khớp với độ dài thực tế của hai chân gauge arc.
