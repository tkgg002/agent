# Context: Lỗi Đối Soát Các Bảng Thuộc Connector Đã Bị Xóa Vẫn Tiếp Tục Quét

## Hiện tượng
- Khi thực hiện xóa một Connector (ví dụ `payment-bills`), shadow binding và master binding tương ứng không được dọn dẹp sạch sẽ trong DB.
- Hàm chạy đối soát `recon smoke` (CheckAllUnified) vẫn tiếp tục quét qua các bảng thuộc connector đã bị xóa, dẫn đến lệch dữ liệu ảo (như báo cáo `transmute: +40,054 (thừa)`).
- Tại UI `/data-integrity`, khi master tắt, nó vẫn hiện thông tin đối soát hoặc không được dọn dẹp đúng cách.

## Vấn đề cần giải quyết
1. Khi xóa một connector, hàm `FullCleanup` trong backend chỉ set `master_connection_id = NULL` ở `master_binding`. Điều này giữ lại bản ghi `master_binding` và khiến nó trở thành mồ côi (orphan) thay vì được xóa hoàn toàn khỏi DB.
2. Cần cập nhật `FullCleanup` để xóa sạch `master_binding` liên quan.
3. Cần kiểm tra xem có chỗ nào trong logic đối soát `recon smoke` vẫn lấy các bindings mồ côi này để chạy không.
