# Kế hoạch xử lý lỗi trùng lặp dòng pipeline trong ReconPipelineGrid

## Các bước thực hiện
1. **Nghiên cứu nguyên nhân**:
   - Kiểm tra logic của hàm `buildPipelines` để xem liệu có trường hợp một record A (segment `source_shadow`) được map với nhiều record B (segment `shadow_master`), hoặc ngược lại, dẫn đến trùng lặp dữ liệu không.
   - Kiểm tra xem API có trả về các record trùng lặp (ví dụ: các record của cùng một table nhưng khác tier hoặc checked_at) khiến `rows` có các phần tử trùng lặp mà không được lọc ra mới nhất.
   - Phân tích logic `flatData` và các `key` của bảng.
2. **Đề xuất giải pháp**:
   - Nếu lỗi do logic map của `buildPipelines`: Cập nhật logic tìm kiếm để loại trừ các `a` đã được map (claimed), đảm bảo mỗi `a` chỉ map với tối đa một `b` và không bị trùng lặp.
   - Nếu lỗi do API trả về nhiều record cho cùng một segment/table: Thực hiện deduplicate (chỉ lấy run mới nhất dựa trên `checked_at` hoặc `id` lớn nhất) trước khi đưa vào `buildPipelines`.
3. **Thực thi sửa mã nguồn**:
   - Cập nhật `ReconPipelineGrid.tsx` với logic đúng.
4. **Xác thực**:
   - Biên dịch frontend bằng `npx tsc -b`.
   - Xác nhận giao diện hiển thị chính xác, không còn trùng lặp.
