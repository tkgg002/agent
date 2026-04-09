# Kế hoạch thực thi (Implementation Plan)

## Mục tiêu
1. **Thêm thông tin Driver**: Bổ sung `Driver ID`, `Driver Name`, `Driver Phone` vào file Excel export của `PaymentBill` và `PaymentHistory`.
2. **Tìm kiếm gần đúng (Approximate Search)**: Áp dụng `$regex` (từ 3 ký tự trở lên) cho các field:
   - Mã đơn hàng (`orderId`)
   - Mã merchant (`merchant.id` hoặc `merchantInfo.id`)
   - Tài khoản Merchant (`merchant.email` hoặc `merchantInfo.email`)
   - Mã Payer (`transactionParties.payer.merchantCode`)
   - Mã Payee (`transactionParties.payee.merchantCode`)
   - Mã GD Merchant (`merchantTransId`)
   - Mã GD Đối tác (`partnerCode` / `trackingId`)

## Chi tiết thay đổi

### 1. `centralized-export-service/data-transfers/params/payment-bill/payment-bill-export.params.ts`
- Thêm `@IsOptional()` và `@IsString()` cho các tham số search còn thiếu: `merchantEmail`, `orderId`, `merchantTransId`, `partnerCode` v.v. để params validator không bỏ qua.

### 2. `centralized-export-service/data-transfers/params/payment/payment-history-export.params.ts`
- Kiểm tra lại các trường param. Các trường search đều đã được định nghĩa (`orderId`, `merchantId`, `accountMerchant`, `partnerCode`, `transId`, `payerMerchantCode`, `payeeMerchantCode`, `trackingId`).

### 3. Logic build filter (trong `.pure.ts`)
- **Payment Bill** (`logics/export/payment-bill/payment-bill-export.pure.ts`):
  - Khởi tạo hàm `buildRegexSearch` để trả về `{ $regex: value, $options: "i" }` khi `value.length >= 3`, ngược lại trả về `value`.
  - Cập nhật hàm `buildPaymentBillFilter` với logic search mới.
- **Payment History** (`logics/export/payment/payment-history-export.pure.ts`):
  - Cập nhật hàm `buildPaymentHistoryFilter` tương tự với các trường search được yêu cầu.

### 4. Bổ sung thông tin Driver
- **Handler**: 
  - Tại `GetAllPaymentBillExportHandler.ts`, bổ sung select field cho object chứa thông tin driver.
  - Tại `GetAllPaymentHistoryExportHandler.ts`, update mapper để lấy thông tin `driver.id`, `driver.name`, `driver.phone`.
- **Export Columns (`getConfig`)**:
  - Bổ sung 3 column mới vào cấu hình headers column (`columns`).
- **Transform Row (`transformRow`)**:
  - Map data từ kết quả query sang array tương ứng với các cột mới thêm.

## User Review Required
> [!IMPORTANT]
> Em không tìm thấy cấu trúc lưu thông tin tài xế trong source code hiện tại.
> **Anh/chị vui lòng xác nhận giúp em thông tin driver đang được lưu trữ ở field nào trong DB (ví dụ: `extraInfo.driverInfo.id` hay `driver.id`, v.v.) đối với collections `Payment` và `PaymentBill`?**
> Và với điều kiện tìm kiếm ">= 3 ký tự", nếu user nhập dưới 3 ký tự thì fallback về match chính xác (exact match) hay ném lỗi validate ạ?
