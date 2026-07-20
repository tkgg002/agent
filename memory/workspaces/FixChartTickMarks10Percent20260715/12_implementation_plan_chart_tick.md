# Kế Hoạch Triển Khai - Chia vạch 10% bằng kẻ xám trong chart.html

## User Review Required

> [!IMPORTANT]
> Giải pháp sử dụng CSS pseudo-element `::after` kết hợp với `linear-gradient` và `background-size` để tự động vẽ 9 vạch kẻ dọc tương ứng với các mốc 10%, 20%, ..., 90% trên tất cả các thanh tiến trình `.progress-wrapper`. Phương án này không làm thay đổi cấu trúc HTML hiện tại của file và đảm bảo tính tùy biến linh hoạt khi co giãn màn hình.

## Proposed Changes

### [Component UI]

#### [MODIFY] [chart.html](file:///Users/trainguyen/Documents/work/chart.html)

Thêm quy tắc CSS sau vào phần `<style>` trong file `chart.html`:

```css
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
```

**Giải thích kỹ thuật:**
1. `width: 90%` giới hạn vùng vẽ vạch của pseudo-element.
2. `background-size: 11.1111111111% 100%` chia pseudo-element thành 9 phần bằng nhau (100% / 9 = 11.1111111111%).
3. Ở mỗi phần 11.1111111111% (tương đương 10% chiều rộng tổng thể), ta vẽ 1 kẻ dọc 1px màu xám đậm mờ (`rgba(0, 0, 0, 0.15)`) nằm ở rìa phải của phần đó.
4. Nhờ đó, 9 vạch kẻ sẽ nằm chính xác tại các mốc `10%`, `20%`, `30%`, `40%`, `50%`, `60%`, `70%`, `80%`, `90%`. Mốc `100%` không có vạch kẻ để tránh đè lên góc bo cong bên phải của thanh tiến trình.
5. Do sử dụng `rgba(0, 0, 0, 0.15)` nên vạch sẽ tự động làm tối màu nền phía sau (hiện thị tốt trên cả nền xanh lá `.progress-fill`, nền xám `.progress-empty`, và các dải màu nền đỏ/cam/vàng/xanh khác).

## Verification Plan

### Manual Verification
- Sử dụng công cụ `browser_subagent` để mở file `chart.html` cục bộ và chụp hình hoặc kiểm tra giao diện để xem các vạch chia 10% có được căn đều và hiển thị chính xác, đẹp mắt hay không.
