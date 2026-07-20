# Phân tích Kỹ thuật: Tích hợp Score-Bubble cho Sub-bars và Tối ưu hóa Marker

Nhiệm vụ: Cung cấp tính năng bong bóng điểm số động và các vạch định vị phát sáng cho tất cả các thanh tiến trình con (sub-bars) trong grid con.

## 1. Thiết kế Giao diện & Trải nghiệm Người dùng (UI/UX)
- **Gradient của Progress Fill:**
  Thanh `.progress-fill` trước đây có màu nền xanh lá cứng (#00a651) che lấp gradient đẹp mắt bên dưới của `.progress-wrapper`. Bằng cách thay đổi màu nền của `.progress-fill` thành `transparent`, dải màu nền gradient được hiển thị liền mạch trên toàn bộ chiều dài phần tiến độ của thanh, mang lại hiệu ứng mượt mà và trực quan hơn.
- **Glowing Marker Lines:**
  Để đạt được hiệu ứng phát sáng (glowing), đường kẻ vạch của cả main bar và các sub-bar được điều chỉnh:
  - Chiều rộng tăng lên để tăng sự hiện diện thị giác.
  - Sử dụng màu trắng để tạo độ tương phản cao với nền màu.
  - Áp dụng `box-shadow` mờ nhạt sử dụng CSS variable `--bubble-color-end`, đồng điệu tuyệt đối với màu sắc bong bóng điểm số của thanh đó.
- **Tick Marks (Vạch chia 10%):**
  Chiều rộng vạch tăng lên `2px` (đậm hơn) và chiều dài tăng lên `14px` (dài hơn), tạo nên sự phân tách mốc rõ ràng trên main bar.

## 2. Giải pháp Định vị Bong bóng trên Sub-bars
- Do mỗi sub-bar nằm trong một phần tử con của layout grid, việc tạo bong bóng tuyệt đối yêu cầu một phần tử cha bọc ngoài có thuộc tính `position: relative`.
- Ta đã thêm lớp `.sub-bar-container` làm thẻ bọc cho `.score-bubble.sub-bubble`, `.sub-marker-line` và `.progress-wrapper.sub-bar-wrapper`.
- Lớp `.sub-bar-container` có `margin-top: 40px` để tạo khoảng trống phía trên thanh sub-bar, tránh bong bóng bị đè hoặc ghi đè lên tiêu đề nhóm của sub-bar đó.
- Bong bóng `.sub-bubble` được scale nhỏ lại (`top: -45px`, `font-size: 20px`, `padding: 3px 10px`) để vừa vặn hoàn hảo với tỉ lệ giao diện của sub-bars.
