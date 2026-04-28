# Plan — Phase 6 Debezium-only Pruning

1. Audit route/page/API surface để xác nhận phần nào đã lỗi thời với Debezium-only.
2. Loại bỏ `QueueMonitoring` khỏi navigation chính và redirect route cũ.
3. Loại `bridge` / `airbyte-sync` khỏi operation pickers và activity filters.
4. Dọn Airbyte copy/component khỏi `SystemHealth`, `DataIntegrity`, `MappingFieldsPage`.
5. Build thật và ghi residual backend debt.
