# Requirements - Update Recommendations for chart1.html

## Scope
Cập nhật file `/Users/trainguyen/Documents/work/chart1.html`:

1. Thêm thuộc tính `recommendations` trực tiếp vào từng đối tượng khách hàng (id: 1, id: 2, id: 3) trong mảng `chartsData`.
2. Đảm bảo hàm `renderCharts` parse linh hoạt cả khi item trong `recommendations` là string thuần túy lẫn object `{ text, icon }`:
   `const recText = typeof rec === 'string' ? rec : rec.text;`
3. Tự động gán icon tương ứng cho 4 câu nếu `rec` là string:
   - Mục 1 ("Duy trì hạn mức...") -> shield/check
   - Mục 2 ("Đánh giá CIC...") -> clock
   - Mục 3 ("Theo dõi tình hình...") -> eye
   - Mục 4 ("Đề xuất cung cấp...") -> file
