# So sánh logic DisbursementTicketExport

Báo cáo chi tiết về sự khác biệt giữa implementation tại `centralized-export-service` và code gốc tại `work-feat/export/stepplaning/ExportDisbursementTicket`.

## 1. Thành phần tham chiếu
- **Centralized Service**: `GetAllDisbursementTicketExportHandler.ts` và `disbursement-ticket.model.ts`
- **Code gốc**: `info.md` (NodeJS implementation gốc), `module_ticket.go` và `entity_ticket.go` (Go implementation).

## 2. So sánh Mapping Dữ liệu (Fields)

Cả hai phiên bản đều sử dụng MongoDB làm nguồn dữ liệu chính cho Ticket.

| Field Name | Centralized Service | Original (info.md/excel) | Ghi chú |
| :--- | :--- | :--- | :--- |
| **Mã yêu cầu** | `ticketCode` | `ticketCode` | Giống nhau |
| **Yêu cầu chi** | `ticketName` | `ticketName` | Giống nhau |
| **Trạng thái** | `status` | `status` | Giống nhau |
| **Tên tổ chức/cá nhân** | `disbursementUnitName` | `disbursementUnitName` | Giống nhau |
| **Loại hình chi** | `expenseType` | `expenseType` | Giống nhau |
| **Tổng khoản chi** | `totalExpense` | `totalExpense` | Giống nhau |
| **Tổng số tiền** | `totalAmount` | `totalAmount` | Giống nhau |
| **Thời gian tạo** | `createdAt` | `createdAt` | Giống nhau |
| **Thời gian cập nhật** | `updatedAt` | `updatedAt` | Giống nhau |

> [!NOTE]
> Centralized service lấy trực tiếp từ `disbursement-ticket.model.ts`. Code gốc sử dụng NATS request sang `disbursement.export-ticket` để lấy dữ liệu.

## 3. So sánh Logic Filter

| Feature | Centralized Service | Original (info.md) | Nhận xét |
| :--- | :--- | :--- | :--- |
| **Cơ chế lọc** | Nhận `filter` object từ query | Gửi explicit các trường filter | Tương đồng |
| **Pagination** | Cursor-based (`cursor`, `limit`) | Không thấy explicit skip/limit trong `info.md`, thường là fetch all | Centralized service tối ưu hơn cho tập dữ liệu lớn |
| **Sắp xếp** | Mặc định `{ createdAt: -1 }` | `sortBy: "createdAt"`, `sortType: -1` | Giống nhau |

## 4. Cấu trúc File Excel
Trong `info.md`, cấu trúc cột được định nghĩa rõ ràng:
1. Mã yêu cầu
2. Yêu cầu chi
3. Trạng thái
4. Tên tổ chức/cá nhân
5. Loại hình chi
6. Tổng khoản chi
7. Tổng số tiền
8. Thời gian tạo
9. Thời gian cập nhật

Centralized service hiện tại mới chỉ xử lý ở tầng dữ liệu (Handler), việc format Excel sẽ được xử lý ở tầng `ExportDataTransfer`.

## 5. Kết luận
Logic tại `centralized-export-service` đã **bao phủ đầy đủ** các trường dữ liệu và logic lọc cơ bản của code gốc.
- **Ưu điểm**: Có cơ chế cursor-based pagination giúp tránh overload khi dữ liệu Ticket lớn.
- **Rủi ro**: Cần đảm bảo `disbursementUnitName` luôn được cập nhật chính xác trong Ticket Model, vì bản gốc có thể lấy từ Unit Service nếu không có sẵn trong Ticket.
