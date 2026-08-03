# Kế Hoạch Động Hóa & Cho Phép Chỉnh Sửa Phần Kiến Nghị Trong chart1.html

## 1. Yêu cầu mới
Cho phép nội dung Kiến Nghị có thể chỉnh sửa được:
- Đưa mảng `recommendations` vào đối tượng từng khách hàng trong `chartsData` (nếu không khai báo thì dùng danh sách mặc định 4 mục vừa cập nhật).
- Cho phép chỉnh sửa trực tiếp trên giao diện trình duyệt bằng thuộc tính `contenteditable="true"` cho thẻ nội dung kiến nghị `.rec-text`, giúp người dùng có thể nhấp chuột vào sửa nội dung trực tiếp dễ dàng.

## 2. Các thay đổi kỹ thuật
1. Khai báo danh sách icon chuẩn map với từng mục kiến nghị (hoặc bổ sung helper `getRecommendationIcon`).
2. Thêm mảng `recommendations` vào `chartsData` của từng khách hàng.
3. Thêm `contenteditable="true"` vào thẻ `.rec-text` (và `.feedback-item` nếu cần).
4. Thêm style focus/hover tinh tế cho `.rec-text[contenteditable="true"]` để trải nghiệm người dùng mượt mà và trực quan.
