# Báo cáo thay đổi: Tích hợp Score-Bubble cho các Sub-bars

Danh sách các file và số dòng thay đổi để triển khai bong bóng điểm số và tối ưu các đường vạch kẻ.

## Danh sách tệp tin thay đổi

### 1. [chart.html](file:///Users/trainguyen/Documents/work/chart.html)
- **Tổng số dòng thay đổi:** ~80 dòng.
- **Chi tiết thay đổi:**
  - **CSS Styles (dòng 114 - 126):**
    - Sửa `.main-bar-wrapper::after` để tăng kích thước vạch chia lên `14px` và rộng `2px` với màu đậm hơn (`rgba(0, 0, 0, 0.4)`).
    - Thay thế thuộc tính nền xanh cứng `#00a651` của `.progress-fill` thành `transparent` để lộ dải gradient.
  - **CSS Styles (dòng 185 - 192):**
    - Cải tiến `.main-marker-line` có nền màu trắng, tăng độ rộng lên `2px` và thêm bóng phát sáng `box-shadow` đồng màu bong bóng điểm số.
  - **CSS Styles (dòng 252 - 273):**
    - Loại bỏ `.sub-marker` cũ.
    - Thêm `.sub-bar-container` để định vị relative cho sub-bar.
    - Định nghĩa `.score-bubble.sub-bubble` với kích cỡ thu nhỏ phù hợp sub-bar.
    - Định nghĩa `.sub-marker-line` là vạch kẻ trắng dọc phát sáng theo màu bong bóng của sub-bar đó.
  - **Javascript (dòng 425 - 458):**
    - Cập nhật vòng lặp `data.items.forEach` để tính toán màu sắc động (`subBubbleColorStart`, `subBubbleColorEnd`) cho từng sub-bar dựa trên tỷ lệ điểm số cụ thể.
    - Bọc progress wrapper của từng sub-bar trong `.sub-bar-container`.
    - Sinh mã HTML chèn `.score-bubble.sub-bubble` và `.sub-marker-line` tương ứng.
