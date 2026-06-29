# Requirements: Core Fix for Batch Transform Schema Drift

## Goal
Khắc phục lỗi batch-transform bị lỗi `column does not exist` do sự lệch pha (drift) giữa danh sách mapping rules được duyệt trên CMS và cấu trúc bảng thực tế trong Database. Giải pháp phải mang tính chất core system, tự phục hồi và bền bỉ (resilient), không sử dụng các biện pháp cheat DB hoặc sửa đổi thủ công cấu hình registry.

## Detailed Requirements
1. **Kiểm tra sự tồn tại của Column trước khi Transform**:
   - Khi xử lý lệnh `batch-transform` cho một target table, trước khi đưa một `TargetColumn` từ Mapping Rule vào câu lệnh `UPDATE` (cụ thể là `setClauses` và `whereClauses`), worker bắt buộc phải kiểm tra xem cột đó có tồn tại trong schema của bảng thực tế trên Shadow DB hay không.
   - Sử dụng hàm `HasColumnInSchema` có sẵn của `BaseHandler` để kiểm tra.
   
2. **Xử lý Graceful khi cột không tồn tại**:
   - Nếu cột không tồn tại, bỏ qua mapping rule đó (skip), không đưa vào câu lệnh UPDATE.
   - Ghi log cảnh báo (`Warn`) rõ ràng về việc cột thiếu trong DB thực tế.
   - Vẫn cho phép cập nhật các cột khác tồn tại bình thường.
   - Nếu toàn bộ các cột trong rules đều không tồn tại (sau khi lọc), trả về kết quả thành công với ghi chú "no columns to transform" thay vì thực hiện câu lệnh UPDATE rỗng.

3. **Chống Cheat DB / Khôi phục Cấu hình Gốc**:
   - Khôi phục cấu hình database registry của User về trạng thái gốc (V1 active, V2 inactive).
   - Chạy lệnh batch-transform trên V1. Hệ thống phải bỏ qua cột `__v` một cách an toàn (ghi log warn) và hoàn thành transform thành công cho các cột hiện có thay vì crash lỗi `column "__v" does not exist`.

4. **Đồng bộ hóa Whitelist Kiểu Dữ Liệu An Toàn cho Primary Key**:
   - Sửa lỗi lệnh `create-default-columns` bị crash do nhận kiểu dữ liệu Primary Key có tham số độ dài như `VARCHAR(24)`.
   - Hàm validate `IsSafeType` trong `base_handler.go` cần sử dụng biểu thức chính quy (Regex) đồng bộ với whitelist của `TypeResolver` thay vì switch case tĩnh, cho phép chấp nhận các kiểu dữ liệu hợp lệ như `VARCHAR(N)`, `CHAR(N)`, `NUMERIC(P, S)`.

