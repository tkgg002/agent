# Yêu Cầu Chi Tiết - Cập Nhật Phần Kiến Nghị & Editable UX trên chart1.html

## 1. Mục tiêu
Cập nhật file `/Users/trainguyen/Documents/work/chart1.html` để cho phép chỉnh sửa phần "Kiến Nghị" và "Nhận Xét" linh hoạt cả trong code JavaScript lẫn trực tiếp trên UI (giao diện trình duyệt).

## 2. Chi tiết Yêu cầu
### Yêu cầu 1: Cấu hình linh hoạt trong JavaScript
- Định nghĩa mảng danh sách kiến nghị mặc định `DEFAULT_RECOMMENDATIONS` chứa 4 mục kiến nghị cùng icon SVG tương ứng:
  1. `{ text: "Duy trì hạn mức hiện tại", icon: "shield" }` (hoặc icon SVG tương ứng: shield-check)
  2. `{ text: "Đánh giá CIC 6 tháng/lần", icon: "clock" }` (hoặc icon clock)
  3. `{ text: "Theo dõi tình hình sử dụng hạn mức phù hợp với nhu cầu sản xuất của khách hàng", icon: "eye" }` (hoặc icon eye)
  4. `{ text: "Đề xuất cung cấp nâng thêm giá trị thư bảo lãnh nhằm giảm rủi ro của tín chấp", icon: "file" }` (hoặc icon file-text)
- Thêm thuộc tính `recommendations` vào mảng `chartsData` cho từng khách hàng (nếu không khai báo thì mặc định lấy `DEFAULT_RECOMMENDATIONS`).
- Cập nhật hàm `renderCharts` để render phần Kiến Nghị (Card 4) động theo `(data.recommendations || DEFAULT_RECOMMENDATIONS)`.

### Yêu cầu 2: Chỉnh sửa trực tiếp trên UI (Editable UI)
- Thêm thuộc tính `contenteditable="true"` cho thẻ `<div class="rec-text" contenteditable="true">` trong Card Kiến Nghị.
- Thêm thuộc tính `contenteditable="true"` cho các mục Nhận Xét `<li class="feedback-item">` (ví dụ trên phần text nhận xét).

### Yêu cầu 3: Styling Editable UX chuyên nghiệp
- Thêm CSS hỗ trợ cho các thẻ có `contenteditable="true"`:
  - Khi hover: con trỏ chuột đổi thành text (`cursor: text`), đổi màu nền nhẹ hoặc viền nhạt để báo hiệu có thể click chỉnh sửa.
  - Khi `:focus`: đường viền outline/border viền xanh mượt (`#0b4fba`), padding/border-radius nhẹ nhàng, hiệu ứng trực quan chuyên nghiệp.
