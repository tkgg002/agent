# Kế Hoạch Triển Khai Chi Tiết (Implementation Plan)

## 1. Định nghĩa Mặc định & Dữ liệu (`DEFAULT_RECOMMENDATIONS`)
Tạo mảng `DEFAULT_RECOMMENDATIONS` trong `<script>`:
```javascript
const DEFAULT_RECOMMENDATIONS = [
    {
        text: "Duy trì hạn mức hiện tại",
        icon: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><polyline points="9 12 11 14 15 10"/></svg>`
    },
    {
        text: "Đánh giá CIC 6 tháng/lần",
        icon: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`
    },
    {
        text: "Theo dõi tình hình sử dụng hạn mức phù hợp với nhu cầu sản xuất của khách hàng",
        icon: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>`
    },
    {
        text: "Đề xuất cung cấp nâng thêm giá trị thư bảo lãnh nhằm giảm rủi ro của tín chấp",
        icon: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>`
    }
];
```

Cập nhật `chartsData`:
Cho phép truyền thuộc tính `recommendations` tùy chỉnh cho từng khách hàng nếu muốn, hoặc nếu không truyền thì mặc định fallback về `DEFAULT_RECOMMENDATIONS`.

## 2. Dynamic Rendering & Content Editable
Trong hàm `renderCharts`:
1. Render phần Nhận xét chung:
   Bọc nội dung chữ nhận xét trong một thẻ `<span class="feedback-text" contenteditable="true">` hoặc trực tiếp trên `<li class="feedback-item">` (tốt nhất là phần văn bản `<span class="feedback-text" contenteditable="true">${comment}</span>`) để dấu tick icon không bị ảnh hưởng khi gõ text.

2. Render phần Kiến nghị:
   Lấy danh sách `const recommendations = data.recommendations || DEFAULT_RECOMMENDATIONS;`
   Duyệt từng mục `rec` để sinh HTML:
   ```html
   <div class="rec-item">
       <div class="rec-icon-circle">
           ${rec.icon}
       </div>
       <div class="rec-text" contenteditable="true">${rec.text}</div>
   </div>
   ```

## 3. CSS cho Editable UX
Thêm quy tắc CSS vào phần `<style>`:
```css
/* Editable UX */
[contenteditable="true"] {
    outline: none;
    transition: all 0.2s ease;
    border-radius: 4px;
    padding: 2px 4px;
    margin: -2px -4px;
}

[contenteditable="true"]:hover {
    cursor: text;
    background-color: rgba(11, 79, 186, 0.04);
    box-shadow: 0 0 0 1px rgba(11, 79, 186, 0.2);
}

[contenteditable="true"]:focus {
    background-color: #ffffff;
    box-shadow: 0 0 0 2px #0b4fba;
    color: #0b4fba;
}
```
