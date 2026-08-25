# Implementation Plan: Add Service Code to PaymentBillExport

## Summary
Bổ sung thuộc tính `serviceCode` (Mã dịch vụ) vào luồng xuất file Excel `PaymentBillExport` theo chuẩn kiến trúc của `centralized-export-service`.

## Target Files
1. `data-transfers/params/payment-bill/payment-bill-export.params.ts`
2. `logics/export/payment-bill/payment-bill-export.pure.ts`
3. `test/unit/pure/payment-bill.pure.test.ts`

## Verification Steps
- Chạy unit tests: `npx vitest run test/unit/pure/payment-bill.pure.test.ts`
- Báo cáo kết quả verifyPASS.
