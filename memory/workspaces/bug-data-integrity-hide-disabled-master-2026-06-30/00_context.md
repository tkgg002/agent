# Context: Hide Disabled Master Tables in Data Integrity

## Scope & Objective
- **Objective**: Khi master sync của một bảng bị tắt (trạng thái hiển thị `Sync: Tắt` hoặc `Sync: Tắt (Chưa duyệt)`), trang `/data-integrity` (địa chỉ `http://localhost:5173/data-integrity`) không hiển thị bảng đó trong danh sách đối soát.
- **Context**: 
  - User báo lỗi đối soát bảng `payment_bills` thuộc `payment-bill-service` có master là `master_payment_bill_service` hiển thị lệch (`ingest: 0`, `transmute: +40,054 (thừa)`) dù master sync đang tắt.
  - Việc hiển thị chênh lệch khi sync đã tắt gây nhiễu cho vận hành.
- **Target**: Lọc (filter) và ẩn các bảng này trên giao diện Data Integrity (cả danh sách và thống kê).
