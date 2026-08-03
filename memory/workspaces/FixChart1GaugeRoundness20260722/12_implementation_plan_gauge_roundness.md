# Kế Hoạch Triển Khai - Làm Tròn Gauge Chart Trong chart1.html

## 1. Các File Thay Đổi
- `[MODIFY] /Users/trainguyen/Documents/work/chart1.html`

## 2. Chi Tiết Thay Đổi
1. Thay đổi tính toán strokeDashoffset trong JavaScript:
   ```javascript
   const totalArcLength = 284.84;
   const strokeDashoffset = totalArcLength - (totalArcLength * (Math.min(mainScore, 100) / 100));
   ```
2. Thay đổi thẻ SVG và Path trong template render `chart1.html`:
   ```html
   <svg class="gauge-svg" viewBox="0 0 200 160">
       <defs>
           <linearGradient id="gaugeGrad_${index}" x1="0%" y1="0%" x2="100%" y2="0%">
               <stop offset="0%" stop-color="#ef4444" />
               <stop offset="35%" stop-color="#ea580c" />
               <stop offset="65%" stop-color="#eab308" />
               <stop offset="100%" stop-color="#22c55e" />
           </linearGradient>
       </defs>
       <path d="M 41.11 124 A 68 68 0 1 1 158.89 124" fill="none" stroke="#e2e8f0" stroke-width="16" stroke-linecap="round"/>
       <path d="M 41.11 124 A 68 68 0 1 1 158.89 124" fill="none" stroke="url(#gaugeGrad_${index})" stroke-width="16" stroke-linecap="round" stroke-dasharray="284.84" stroke-dashoffset="${strokeDashoffset}"/>
   </svg>
   ```
3. Cập nhật CSS:
   ```css
   .gauge-wrapper {
       position: relative;
       width: 200px;
       height: 160px;
       display: flex;
       flex-direction: column;
       align-items: center;
       justify-content: center;
       flex-shrink: 0;
   }
   .gauge-svg {
       width: 200px;
       height: 160px;
   }
   .gauge-score-container {
       position: absolute;
       top: 72px;
       left: 50%;
       transform: translateX(-50%);
       text-align: center;
   }
   ```

## 3. Verification Plan
- Chạy mở file HTML trên trình duyệt hoặc kiểm tra cấu trúc mã nguồn HTML/SVG để đảm bảo không bị lệch layout.
