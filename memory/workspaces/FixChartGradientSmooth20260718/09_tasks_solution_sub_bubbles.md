# Giải pháp Kỹ thuật: Tích hợp Score-Bubble cho Sub-bars và Tối ưu hóa Marker

Tệp tin cần sửa đổi: `chart.html` (đường dẫn: `/Users/trainguyen/Documents/work/chart.html`)

## 1. Các thay đổi về CSS (Styles)

### Thay đổi 1: Sửa màu nền `.progress-fill` để hiển thị gradient
```diff
         .progress-fill {
-            background: #00a651;
-            /* Thanh màu xanh lá chạy bên trong */
+            background: transparent;
+            /* Thanh chạy bên trong trong suốt để lộ gradient nền */
             height: 100%;
         }
```

### Thay đổi 2: Tối ưu hóa `.main-bar-wrapper::after` (Tick marks)
```diff
         .main-bar-wrapper::after {
-            background-size: 11.1111111111% 10px, 11.1111111111% 10px;
+            background-image:
+                linear-gradient(to right, transparent calc(100% - 2px), rgba(0, 0, 0, 0.4) calc(100% - 2px)),
+                linear-gradient(to right, transparent calc(100% - 2px), rgba(0, 0, 0, 0.4) calc(100% - 2px));
+            background-size: 11.1111111111% 14px, 11.1111111111% 14px;
         }
```

### Thay đổi 3: Cải tiến `.main-marker-line` (Glowing marker line)
```diff
         .main-marker-line {
             position: absolute;
             top: -15px;
             bottom: 35px;
-            width: 1px;
-            background-color: #333;
+            width: 2px;
+            background-color: #fff;
+            box-shadow: 0 0 8px var(--bubble-color-end, #64b90c);
             z-index: 5;
         }
```

### Thay đổi 4: Bổ sung CSS cho các thành phần Sub-bar (sau `.sub-bar-wrapper .progress-empty`)
```css
        .sub-bar-container {
            position: relative;
            margin-top: 40px;
        }

        .score-bubble.sub-bubble {
            top: -45px;
            font-size: 20px;
            padding: 3px 10px;
            border-radius: 6px;
            border-width: 1.5px;
            box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
        }

        .score-bubble.sub-bubble::after {
            bottom: -7px;
            border-width: 7px 6px 0;
        }

        .sub-marker-line {
            position: absolute;
            top: -10px;
            bottom: 0px;
            width: 1.5px;
            background-color: #fff;
            box-shadow: 0 0 6px var(--bubble-color-end, #64b90c);
            z-index: 5;
        }
```

## 2. Các thay đổi về JS trong hàm `renderCharts`

### Thay đổi 5: Cập nhật vòng lặp kết xuất `data.items.forEach`
```diff
                 let subBarsHtml = '';
                 data.items.forEach(item => {
                     const pct = item.max > 0 ? (item.point / item.max) * 100 : 0;
                     let progressHtml = '';
                     if (pct === 100) {
                         progressHtml = `
                             <div class="progress-fill" style="width: 100%; border-radius: 8px;"></div>
                             <div class="progress-empty" style="width: 0%; display: none;"></div>
                         `;
                     } else if (pct === 0) {
                         progressHtml = `
                             <div class="progress-fill" style="width: 0%; display: none;"></div>
                             <div class="progress-empty" style="border-radius: 8px; border-left: none;"></div>
                         `;
                     } else {
                         progressHtml = `
                             <div class="progress-fill" style="width: ${pct}%;"></div>
                             <div class="progress-empty"></div>
                         `;
                     }
 
-                    subBarsHtml += `
-                        <div>
-                            <div class="item-header">
-                                <h3 class="item-title">${item.title} <span>(max ${item.max})</span></h3>
-                                <span class="item-score">${item.point}/${item.max}</span>
-                            </div>
-                            <div class="progress-wrapper sub-bar-wrapper">
-                                <div class="sub-marker" style="left: ${pct}%;"></div>
-                                ${progressHtml}
-                            </div>
-                        </div>
-                    `;
+                    // Xác định màu sắc của bong bóng điểm số dựa trên phần trăm của sub-bar
+                    let subBubbleColorStart = '#a3e75e';
+                    let subBubbleColorEnd = '#64b90c';
+                    if (pct <= 50) {
+                        subBubbleColorStart = '#ff837f';
+                        subBubbleColorEnd = '#f0231b';
+                    } else if (pct <= 65) {
+                        subBubbleColorStart = '#ffb875';
+                        subBubbleColorEnd = '#fd8101';
+                    } else if (pct <= 80) {
+                        subBubbleColorStart = '#ffea7c';
+                        subBubbleColorEnd = '#fccb00';
+                    }
+
+                    subBarsHtml += `
+                        <div class="sub-bar-section">
+                            <div class="item-header">
+                                <h3 class="item-title">${item.title} <span>(max ${item.max})</span></h3>
+                                <span class="item-score">${item.point}/${item.max}</span>
+                            </div>
+                            <div class="sub-bar-container" style="--bubble-color-start: ${subBubbleColorStart}; --bubble-color-end: ${subBubbleColorEnd};">
+                                <div class="score-bubble sub-bubble" style="left: ${pct}%;">${item.point}</div>
+                                <div class="sub-marker-line" style="left: ${pct}%;"></div>
+                                <div class="progress-wrapper sub-bar-wrapper">
+                                    ${progressHtml}
+                                </div>
+                            </div>
+                        </div>
+                    `;
                 });
```
