# Giải Pháp Kỹ Thuật Chi Tiết - Fix Chart Tick Marks 10%

## File cần chỉnh sửa
- [chart.html](file:///Users/trainguyen/Documents/work/chart.html)

## Nội dung thay đổi chi tiết

### Sửa CSS trong phần `<style>` (tại khoảng dòng 94)

**Trước khi sửa:**
```css
        .progress-wrapper {
            position: relative;
            width: 100%;
            /* 4 dải màu theo chuẩn Đỏ 50% - Cam 15% - Vàng 15% - Xanh 20% */
            background: linear-gradient(to right,
                    #f0231b 0%, #f0231b 50%,
                    #fd8101 50%, #fd8101 65%,
                    #fccb00 65%, #fccb00 80%,
                    #64b90c 80%, #64b90c 100%);
            border: 0px solid #b3b3b3;
            box-sizing: border-box;
            display: flex;
        }

        .progress-fill {
```

**Sau khi sửa:**
```css
        .progress-wrapper {
            position: relative;
            width: 100%;
            /* 4 dải màu theo chuẩn Đỏ 50% - Cam 15% - Vàng 15% - Xanh 20% */
            background: linear-gradient(to right,
                    #f0231b 0%, #f0231b 50%,
                    #fd8101 50%, #fd8101 65%,
                    #fccb00 65%, #fccb00 80%,
                    #64b90c 80%, #64b90c 100%);
            border: 0px solid #b3b3b3;
            box-sizing: border-box;
            display: flex;
        }

        /* Vẽ vạch chia 10% bằng kẻ xám */
        .progress-wrapper::after {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 90%;
            bottom: 0;
            pointer-events: none;
            z-index: 10;
            background-image: linear-gradient(to right, transparent calc(100% - 1px), rgba(0, 0, 0, 0.15) calc(100% - 1px));
            background-size: 11.1111111111% 100%;
        }

        .progress-fill {
```
