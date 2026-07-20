# Giải Pháp Kỹ Thuật Chi Tiết - Làm Mượt Dải Màu Gradient & Tô Màu Bong Bóng Điểm Số

## File cần chỉnh sửa
- [chart.html](file:///Users/trainguyen/Documents/work/chart.html)

## Nội dung thay đổi chi tiết

### 1. Sửa CSS trong phần `<style>` cho `.score-bubble` và `.score-bubble::after` (khoảng dòng 157-183)

**Trước khi sửa:**
```css
        /* Bong bóng điểm số 75 */
        .score-bubble {
            position: absolute;
            top: -65px;
            transform: translateX(-50%);
            background: linear-gradient(to bottom, #d4f0d4, #91d191);
            border: 2px solid #fff;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
            border-radius: 10px;
            padding: 5px 15px;
            font-size: 42px;
            font-weight: bold;
            color: #000;
            z-index: 10;
        }

        .score-bubble::after {
            content: '';
            position: absolute;
            bottom: -12px;
            left: 50%;
            transform: translateX(-50%);
            border-width: 12px 10px 0;
            border-style: solid;
            border-color: #91d191 transparent transparent transparent;
        }
```

**Sau khi sửa:**
```css
        /* Bong bóng điểm số 75 */
        .score-bubble {
            position: absolute;
            top: -65px;
            transform: translateX(-50%);
            background: linear-gradient(to bottom, var(--bubble-color-start, #d4f0d4), var(--bubble-color-end, #91d191));
            border: 2px solid #fff;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
            border-radius: 10px;
            padding: 5px 15px;
            font-size: 42px;
            font-weight: bold;
            color: #000;
            z-index: 10;
        }

        .score-bubble::after {
            content: '';
            position: absolute;
            bottom: -12px;
            left: 50%;
            transform: translateX(-50%);
            border-width: 12px 10px 0;
            border-style: solid;
            border-color: var(--bubble-color-end, #91d191) transparent transparent transparent;
        }
```

### 2. Tính toán và chèn màu động trong JavaScript (khoảng dòng 410)

**Trước khi sửa:**
```javascript
                let subBarsHtml = '';
                data.items.forEach(item => {
```

**Sau khi sửa:**
```javascript
                // Xác định màu sắc của bong bóng điểm số dựa trên mốc điểm
                let bubbleColorStart = '#a3e75e';
                let bubbleColorEnd = '#64b90c';
                if (mainScore <= 50) {
                    bubbleColorStart = '#ff837f';
                    bubbleColorEnd = '#f0231b';
                } else if (mainScore <= 65) {
                    bubbleColorStart = '#ffb875';
                    bubbleColorEnd = '#fd8101';
                } else if (mainScore <= 80) {
                    bubbleColorStart = '#ffea7c';
                    bubbleColorEnd = '#fccb00';
                }

                let subBarsHtml = '';
                data.items.forEach(item => {
```

### 3. Áp dụng biến CSS cho `.score-bubble` trong chuỗi HTML (khoảng dòng 461)

**Trước khi sửa:**
```javascript
                    <div class="main-bar-section">
                        <div class="score-bubble" style="left: ${mainScore}%;">${mainScore}</div>
                        <div class="main-marker-line" style="left: ${mainScore}%;"></div>
```

**Sau khi sửa:**
```javascript
                    <div class="main-bar-section">
                        <div class="score-bubble" style="left: ${mainScore}%; --bubble-color-start: ${bubbleColorStart}; --bubble-color-end: ${bubbleColorEnd};">${mainScore}</div>
                        <div class="main-marker-line" style="left: ${mainScore}%;"></div>
```
