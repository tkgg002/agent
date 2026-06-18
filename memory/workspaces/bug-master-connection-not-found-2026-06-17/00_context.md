# Context: Lỗi master_connection_not_found

## 1. Mô tả lỗi
User báo cáo lỗi: `{"error":"master_connection_not_found"}`.
Lỗi này thường xảy ra khi hệ thống cố gắng tìm kết nối đến database Master (hoặc Master connection trong registry) nhưng không tìm thấy connection hợp lệ hoặc có cấu hình nào đó bị thiếu.

## 2. Các workspace liên quan trước đây
- `bug-ambiguous-master-connection-2026-06-10`
- `bug-shadow-conn-cdc-cms-2026-05-18`

## 3. Mục tiêu
- Xác định nguyên nhân gốc rễ (Root Cause) gây ra lỗi `master_connection_not_found`.
- Lập kế hoạch khắc phục và thực thi sửa lỗi.
- Đảm bảo lỗi không tái phát.
