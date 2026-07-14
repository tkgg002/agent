# Yêu cầu chi tiết - Đồng bộ Mapping Rules khi Approve Master

## Bối cảnh
Khi một Master Table được phê duyệt, hệ thống sẽ tự động nhân bản (clone) các mapping rules từ Shadow DB sang Master DB. Hiện tại, các mapping rules này bị clone với trạng thái mặc định là `'pending'`, dẫn đến việc transmuter chạy ở runtime không thể tìm thấy rules đã được duyệt (yêu cầu `'approved'`) và ném lỗi `no approved mapping rules found`.

## Yêu cầu
1. Thay đổi logic nhân bản mapping rules để trạng thái của rules được clone kế thừa trực tiếp từ trạng thái của rule nguồn (`v2.status`) thay vì gán cứng `'pending'`.
2. Khắc phục toàn bộ lỗi biên dịch (interface mismatch và import sai package) trong integration tests của `cdc-cms-service`.
3. Chạy thành công toàn bộ test suite tích hợp của `cdc-cms-service` để xác thực logic.
