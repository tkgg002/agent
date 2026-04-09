# Implementation Plan (Refined Pattern)

## Dual-Language Strategy

### [EN] Implementation
1. **Auxiliary Query**: Create `GetMerchantExportAuxiliaryQuery` to aggregate BusinessLine and History data.
2. **Auxiliary Handler**: Implement the handler that joins data in memory efficiently.
3. **Pure Logic**: Update `merchant-export.pure.ts` to use this composite auxiliary query instead of simple business line query.

### [VI] Triển khai (Mẫu thiết kế tinh gọn)
1. **Auxiliary Query**: Tạo `GetMerchantExportAuxiliaryQuery` để tổng hợp dữ liệu BusinessLine và Lịch sử.
2. **Auxiliary Handler**: Triển khai handler để join dữ liệu trong bộ nhớ một cách hiệu quả.
3. **Pure Logic**: Cập nhật `merchant-export.pure.ts` sử dụng composite auxiliary query thay vì business line query đơn thuần.

## Verification
- Unit test for Handler: `test/unit/domain/handlers/GetAllMerchantExportHandler.test.ts`
- Unit test for Pure Logic: `test/unit/pure/merchant-export.pure.test.ts`
