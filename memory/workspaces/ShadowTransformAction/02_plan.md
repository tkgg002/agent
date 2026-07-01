# Plan: ShadowTransformAction

Kế hoạch thực hiện thêm nút Transform vào Shadow Actions:

## Giai đoạn 1: Khảo sát & Research
1. Tìm kiếm đường dẫn hoặc component tương ứng với trang `/shadow` hoặc "Shadow Actions" trên frontend `cdc-cms-web`.
2. Tìm kiếm API thực hiện transform dữ liệu shadow/master trong backend `cdc-cms-service` hoặc `centralized-data-service`.

## Giai đoạn 2: Thiết kế API và Giao diện
1. (Nếu cần) Bổ sung API endpoint trong backend để hỗ trợ gọi transform thủ công.
2. Thêm nút "Transform" vào giao diện frontend tại phần "Shadow Actions".
3. Kết nối nút bấm với API endpoint tương ứng, xử lý trạng thái loading và thông báo kết quả.

## Giai đoạn 3: Xác minh
1. Kiểm tra tính khả dụng của nút bấm và API bằng cách chạy kiểm thử hoặc tương tác giao diện.
