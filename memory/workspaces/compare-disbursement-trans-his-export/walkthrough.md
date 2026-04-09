# Walkthrough: DisbursementTransHisExport Comparison

Tôi chân thành xin lỗi về sự cẩu thả khi gộp chung logic của hai feature khác nhau vào cùng một workspace. Tôi đã sửa lỗi bằng cách tạo workspace riêng biệt và cập nhật quy trình.

Dưới đây là kết quả so sánh logic cho **DisbursementTransHisExport**:

## 1. So sánh Bộ lọc (Filters)

| Tham số | Logic Gốc (Go) | Service Mới (TS) | Trạng thái |
| :--- | :--- | :--- | :--- |
| **ticketId** | Hỗ trợ exact match | **Thiếu (Missing)** | ❌ Cần bổ sung |
| **ticketName** | Exact match | Regex (case-insensitive) | ✅ Cải tiến |
| **ticketCode** | Exact match | Exact match | ✅ Khớp |
| **legacyNumber**| `receiver.phone` | `receiver.phone` | ✅ Khớp |
| **expenseType** | Exact match | Exact match | ✅ Khớp |
| **status** | Exact match | Exact match | ✅ Khớp |
| **createdAt** | Date Range | Date Range | ✅ Khớp |

## 2. So sánh Cột dữ liệu (Excel Columns)

Service tập trung (`Centralized Export Service`) đã duy trì đầy đủ các cột từ code gốc và bổ sung thêm cột **STT** để dễ theo dõi:

- **STT**: Cột mới thêm.
- **Mã khoản chi**: `item._id` (Khớp).
- **Yêu cầu chi**: `item.ticketName` (Khớp).
- **Mã yêu cầu**: `item.ticketCode` (Khớp).
- **Loại hình chi**: Mapping `Ví GooPay` / `Ngân hàng` (Khớp).
- **Mã giao dịch**: `item.transId` (Khớp).
- **Tên người nhận**: `item.receiver.fullName` (Khớp).
- **Tài khoản**: `item.receiver.phone` (Khớp).
- **Số tiền**: `item.value` (Khớp).
- **Trạng thái**: Mapping tiếng Việt (`Thành công`, `Thất bại`, `Chờ chi`) (Khớp).
- **Thời gian chi**: `DD/MM/YYYY HH:mm:ss` (Khớp).

## 3. Các thay đổi & Khuyến nghị

> [!IMPORTANT]
> - **Cần bổ sung Filter `ticketId`**: Code gốc cho phép lọc theo ID của disbursement unit. Service hiện tại đang bị thiếu field này trong hàm `buildDisbursementTransHisExportFilter`.
> - **Ngôn ngữ**: Service mới hỗ trợ đa ngôn ngữ (VI/EN) dựa trên request meta, code cũ mặc định cứng tiếng Việt.

---
Mọi tài liệu liên quan hiện đã được lưu trữ đúng tại:
`agent/memory/workspaces/compare-disbursement-trans-his-export/`
