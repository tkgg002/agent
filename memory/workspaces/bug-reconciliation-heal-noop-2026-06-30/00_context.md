# Context: Lỗi Reconciliation Heal trả về noop và không sync record bị thiếu

## Hiện tượng
- MongoDB source (`payment-bill-service.payment-bills`) có 39,992 records.
- PostgreSQL shadow (`shadow_test1111.payment_bills`) có 39,991 records.
- Lệch chính xác 1 record (ID `41025`).
- Khi trigger API `POST http://localhost:8083/api/reconciliation/heal` cho bảng này, NATS activity log ghi nhận `recon-heal-a` trả về `noop` và không sync bản ghi thiếu.

## Phân tích & Phát hiện
- Trường `timestamp_field` của bảng `payment-bills` trong bảng registry (`cdc_system.cdc_table_registry`) bị cấu hình sai thành `updated_at`.
- Trong MongoDB thực tế, collection `payment-bills` không có trường `updated_at` mà chỉ có trường `lastUpdatedAt`.
- Khi chạy reconciliation quét MongoDB, nó dùng filter theo `updated_at` nên trả về 0 records.
- Ở phía shadow DB, query sử dụng `_source_ts` để quét và tìm thấy 39,991 records.
- Vì thế, engine tính toán: `missingFromDest` (số record có ở source nhưng thiếu ở dest) = 0.
- Do đó, engine báo cáo không có record nào cần heal ở shadow, dẫn đến kết quả `noop`.
