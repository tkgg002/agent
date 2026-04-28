# Plan Phase 3 — Write Path V2 Foundation

## English

1. Extend ingest records so runtime can carry schema and connection identity, not just table name.
2. Upgrade the metadata route to expose the shadow connection key.
3. Make the batch buffer group and write by `connection key + schema + table`.
4. Upgrade schema preparation/upsert SQL generation to support schema-qualified target tables.
5. Keep compatibility for legacy callers that still assume `public.<table>`.
6. Validate service/handler/server packages after the refactor.

## Tiếng Việt

1. Mở rộng ingest record để runtime mang được `schema + connection`, không chỉ còn `tableName`.
2. Mở rộng metadata route để trả ra `shadow connection key`.
3. Nâng `BatchBuffer` để group và ghi theo `connection key + schema + table`.
4. Nâng `SchemaAdapter` để prepare/upsert được với bảng có schema-qualified name.
5. Giữ compatibility cho các caller cũ vẫn assume `public.<table>`.
6. Verify lại `service`, `handler`, `server`.
