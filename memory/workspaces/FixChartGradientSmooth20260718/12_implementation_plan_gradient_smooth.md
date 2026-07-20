# Kế Hoạch Triển Khai - Làm Mượt Dải Màu Gradient & Đổi Màu Bong Bóng Điểm Số trong chart.html

## User Review Required

> [!IMPORTANT]
> 1. Giải pháp điều chỉnh thuộc tính `background` của `.progress-wrapper` từ dạng dải màu phân tách cứng (hard stops) sang dạng chuyển màu mềm mại liên tục (smooth linear-gradient).
> 2. Cập nhật màu nền và mũi tên của `.score-bubble` tự động đồng bộ theo màu của phân khúc điểm hiện tại (Đỏ, Cam, Vàng, Xanh lá) thông qua các biến CSS (`--bubble-color-start` và `--bubble-color-end`) được tiêm động bằng JavaScript.
> 3. Sửa lỗi cú pháp dư thừa ký tự `=` ở dòng 440 của `chart.html`.

## Proposed Changes

### [Component UI]

#### [MODIFY] [chart.html](file:///Users/trainguyen/Documents/work/chart.html)

**1. Thay đổi CSS của `.score-bubble` và `.score-bubble::after` trong `<style>`:**
Sử dụng các biến CSS để hỗ trợ nhận giá trị màu động từ JavaScript.

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

**2. Tính toán dải màu bong bóng và chèn các biến CSS trong hàm `renderCharts`:**

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
```

Chèn các biến này vào style của `.score-bubble` trong HTML template:
```html
<div class="score-bubble" style="left: ${mainScore}%; --bubble-color-start: ${bubbleColorStart}; --bubble-color-end: ${bubbleColorEnd};">${mainScore}</div>
```

## Verification Plan

### Manual Verification
- Sử dụng công cụ `browser_subagent` để mở file `chart.html` cục bộ, chụp ảnh màn hình và kiểm tra:
  - Gradient của thanh tiến trình đã chuyển màu mượt mà.
  - Bong bóng điểm số của 3 biểu đồ (70 điểm -> màu Vàng, 60 điểm -> màu Cam, 85 điểm -> màu Xanh lá) đổi màu chính xác và đồng bộ với mũi tên chỉ điểm bên dưới.
  - Không còn ký tự `=` thừa ở các thanh phụ.
