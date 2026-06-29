# Requirements

## 1. Loại bỏ cascade active
- Khi toggle `is_active` của Source Object (ví dụ: qua API /api/v1/source-objects/:id/active), cờ `is_active` của `shadow_binding` tương ứng KHÔNG được tự động thay đổi theo.
- Trạng thái hoạt động của shadow_binding phải được quản lý độc lập.

## 2. Sửa cột Shadow hiển thị sai tên bảng
- Trên trang `/snapshot-monitor`, cột "Shadow" phải hiển thị chính xác tên bảng của shadow_binding được liên kết với record `snapshot_progress` đó.
- Không được lấy nhầm tên bảng shadow của binding khác thuộc cùng source object.

## 3. Sửa lỗi Sensitive Masking Strategy không chạy
- Đảm bảo khi chạy snapshot hoặc upstream, các rule masking của từng shadow binding được nạp chính xác vào memory cache.
- Rule của binding/clone A không được nạp chéo sang binding B.

## 4. Xử lý lỗi scan-fields khi shadow table trống
- Khi shadow table trống, việc scan fields không có dữ liệu để quét.
- Cần ghi nhận rõ ràng trạng thái lỗi "shadow table %s is empty" trong DB cdc_activity_log nhưng FE phải dừng polling loading và hiển thị thông báo hợp lý, không được treo/quay loading vô tận.
