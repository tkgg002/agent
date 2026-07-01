# Requirements: Hide Disabled Master Tables in Data Integrity

## User Requirement
- Ẩn các bảng đối soát khỏi giao diện `/data-integrity` khi master sync của bảng đó bị tắt.
- Ví dụ cụ thể: Bảng `payment_bills` thuộc `payment-bill-service` có master là `master_payment_bill_service` hiển thị `Sync: Tắt` thì không hiện ra trên trang `/data-integrity` nữa.

## Technical Requirements
1. Xác định chính xác trạng thái "master sync tắt" dựa trên metadata:
   - Không tìm thấy cấu hình master tương ứng trong `/api/v1/masters` (tức là đã bị xóa).
   - Hoặc master config tồn tại nhưng `is_active === false` (chưa duyệt hoặc bị tắt).
2. Lọc danh sách report (`reportList`) trước khi hiển thị trên:
   - Component `ReconPipelineGrid` (tab Pipelines).
   - Component `Table` (tab Tổng quan).
   - Các card thống kê ở đầu trang (`Tổng bảng`, `Khớp`, `Lệch`) để đảm bảo số liệu thống kê nhất quán với danh sách hiển thị.
3. Không làm ảnh hưởng đến các bảng chưa từng cấu hình master (chỉ chạy source -> shadow, segment: `source_shadow` và không có segment `shadow_master`).
