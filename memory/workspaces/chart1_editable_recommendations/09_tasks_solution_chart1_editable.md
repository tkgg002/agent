# Hồ Sơ Giải Pháp Kỹ Thuật (Technical Solution)

## Mục tiêu
Cập nhật file `/Users/trainguyen/Documents/work/chart1.html` để:
1. Thêm mảng `DEFAULT_RECOMMENDATIONS` mặc định trong JS.
2. Cho phép từng customer trong `chartsData` có thuộc tính `recommendations` tùy chọn (fallback về `DEFAULT_RECOMMENDATIONS`).
3. Render Card 4 (Kiến Nghị) động từ mảng kiến nghị.
4. Gán `contenteditable="true"` cho thẻ `.rec-text` và `.feedback-text` (hoặc `.feedback-item`).
5. Bổ sung hiệu ứng CSS chuyên nghiệp cho `[contenteditable="true"]` (hover, focus với màu viền `#0b4fba`).

## Thay đổi chi tiết trong `chart1.html`

### 1. Thêm `DEFAULT_RECOMMENDATIONS`
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

### 2. Cập nhật `comments` và `recommendations` rendering trong `renderCharts`
```javascript
// Render danh sách nhận xét
let commentsHtml = '';
(data.comments || []).forEach(comment => {
    commentsHtml += `
    <li class="feedback-item">
        <div class="check-icon">
            <svg viewBox="0 0 24 24" fill="none"><path d="M20 6L9 17l-5-5"/></svg>
        </div>
        <span class="feedback-text" contenteditable="true">${comment}</span>
    </li>
`;
});

// Render danh sách kiến nghị
const recommendations = data.recommendations || DEFAULT_RECOMMENDATIONS;
let recommendationsHtml = '';
recommendations.forEach(rec => {
    const iconSvg = rec.icon || SVG_ICONS.folder;
    recommendationsHtml += `
    <div class="rec-item">
        <div class="rec-icon-circle">
            ${iconSvg}
        </div>
        <div class="rec-text" contenteditable="true">${rec.text}</div>
    </div>
`;
});
```

### 3. Cập nhật HTML Card 4 (Kiến Nghị)
```html
<div class="card">
    <div class="card-header-pill">Kiến Nghị</div>
    <div class="recommendation-list-grid">
        ${recommendationsHtml}
    </div>
</div>
```

### 4. CSS cho Editable elements
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
    background-color: rgba(11, 79, 186, 0.05);
    box-shadow: 0 0 0 1px rgba(11, 79, 186, 0.2);
}

[contenteditable="true"]:focus {
    background-color: #ffffff;
    box-shadow: 0 0 0 2px #0b4fba;
    color: #0b4fba;
}
```
