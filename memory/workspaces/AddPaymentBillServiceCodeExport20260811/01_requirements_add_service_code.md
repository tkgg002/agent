# Requirement Spec: Add Service Code to PaymentBillExport

## 1. Context & Objective
Yêu cầu từ User: Bổ sung trường `serviceCode` (Mã dịch vụ) vào logic xuất file Excel đơn hàng `PaymentBillExport` trong file `/Users/trainguyen/Documents/work/centralized-export-service/logics/export/payment-bill/payment-bill-export.pure.ts` và các thành phần liên quan (DTO params, test unit).

## 2. Detailed Requirements
1. **Filter Query**: 
   - Thêm `serviceCode` vào danh sách `directFields` trong `buildPaymentBillFilter` của `payment-bill-export.pure.ts` để filter theo `serviceCode` nếu client truyền vào.
2. **DTO Params**:
   - Khai báo thêm trường `serviceCode?: string` (kèm decorator `@IsOptional()`, `@IsString()`) trong class `PaymentBillExportParams` (`data-transfers/params/payment-bill/payment-bill-export.params.ts`).
3. **Export Columns Config (`getConfig`)**:
   - Thêm cột `{ vi: "Mã dịch vụ", en: "Service code" }` vào danh sách `columns` trong `getConfig` của `payment-bill-export.pure.ts` (vị trí ngay sau `Mã đơn hàng`).
4. **Transform Excel Row (`transformRow`)**:
   - Trích xuất `item.serviceCode || ""` để gán vào cột `Mã dịch vụ` tương ứng trong mảng trả về của `transformRow`.
5. **Unit Tests**:
   - Cập nhật test case trong `test/unit/pure/payment-bill.pure.test.ts` để test suite pass 100% với cấu trúc 19 cột mới và giá trị `serviceCode`.
