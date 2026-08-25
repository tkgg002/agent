# Tasks Checklist: Add Service Code to PaymentBillExport

- [x] Task 1: Update DTO Params `PaymentBillExportParams` in `data-transfers/params/payment-bill/payment-bill-export.params.ts` (Add `serviceCode`).
- [x] Task 2: Update `payment-bill-export.pure.ts` (Add `serviceCode` filter, columns header, and row transformation).
- [x] Task 3: Update `test/unit/pure/payment-bill.pure.test.ts` for unit test alignment.
- [x] Task 4: Run unit tests (`npx vitest run test/unit/pure/payment-bill.pure.test.ts`) & integration tests to verify 100% PASS.
