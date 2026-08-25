# Kế hoạch triển khai: Thêm Mã Dịch Vụ (serviceCode) vào PaymentBillExport

Bổ sung trường `serviceCode` (Mã dịch vụ) vào bộ logic xuất dữ liệu Excel `PaymentBillExport` trong service `centralized-export-service`.

## User Review Required

> [!NOTE]
> Giải pháp duy nhất và tối ưu được chọn:
> 1. **Filter Params**: Bổ sung `serviceCode` làm filter tuỳ chọn trong DTO validation params (`PaymentBillExportParams`).
> 2. **Filter Query**: Thêm `serviceCode` vào danh sách `directFields` trong bộ lọc MongoDB (`buildPaymentBillFilter`).
> 3. **Excel Columns**: Thêm cột `{ vi: "Mã dịch vụ", en: "Service code" }` ở vị trí ngay sau cột "Mã đơn hàng" (Order ID).
> 4. **Row Data Transform**: Trích xuất `item.serviceCode || ""` đưa vào mảng dữ liệu xuất Excel (`transformRow`).
> 5. **Unit Test**: Cập nhật lại test suite `payment-bill.pure.test.ts` khớp với 19 cột xuất ra và kiểm tra tính chính xác của dữ liệu.

## Open Questions

Không có. Yêu cầu rõ ràng và tuân thủ đúng kiến trúc chuẩn của service.

---

## Proposed Changes

### Centralized Export Service

#### [MODIFY] [payment-bill-export.params.ts](file:///Users/trainguyen/Documents/work/centralized-export-service/data-transfers/params/payment-bill/payment-bill-export.params.ts)
- Bổ sung trường `serviceCode?: string` với `@IsOptional()` và `@IsString()`.

#### [MODIFY] [payment-bill-export.pure.ts](file:///Users/trainguyen/Documents/work/centralized-export-service/logics/export/payment-bill/payment-bill-export.pure.ts)
- Thêm `"serviceCode"` vào mảng `directFields` trong `buildPaymentBillFilter`.
- Bổ sung cột `{ vi: "Mã dịch vụ", en: "Service code" }` vào danh sách `columns` trong `getConfig`.
- Bổ sung `item.serviceCode || ""` vào mảng trả về trong `transformRow`.

#### [MODIFY] [payment-bill.pure.test.ts](file:///Users/trainguyen/Documents/work/centralized-export-service/test/unit/pure/payment-bill.pure.test.ts)
- Cập nhật số lượng cột kỳ vọng là `19` và các index truy cập trường tương ứng (`serviceCode` ở index 8, `channelID` ở index 13, `state` ở index 16, `createdAt` ở index 17).

---

## Verification Plan

### Automated Tests
- Chạy unit test riêng cho logic xuất PaymentBill:
  ```bash
  npx vitest run test/unit/pure/payment-bill.pure.test.ts
  ```
