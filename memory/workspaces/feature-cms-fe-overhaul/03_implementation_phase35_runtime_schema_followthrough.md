# Implementation — Phase 35 Runtime Schema Follow-through

## Audit kết quả

- `HandleDiscover` vẫn introspect `information_schema.columns` với `table_schema='public'`.
- `HandleBackfill` vẫn `UPDATE "target_table"` không qualify schema.
- `SchemaInspector` vẫn cache và đọc schema dựa trên `PendingFieldRepo.GetTableColumns()` mặc định `public`.

## Thay đổi đã áp dụng

### 1. PendingFieldRepo

- [pending_field_repo.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/repository/pending_field_repo.go)
- Thêm `GetTableColumnsInSchema(ctx, schemaName, tableName)`.
- `GetTableColumns()` cũ giữ lại như wrapper fallback về `public`.

### 2. SchemaInspector

- [schema_inspector.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/schema_inspector.go)
- Thêm `metadata MetadataRegistry`.
- Thêm `SetMetadataRegistry()`.
- `getTableSchema()` giờ:
  - resolve `shadow_schema` từ `ResolveTargetRoute(target_table)`
  - cache theo `schema.table`
  - gọi `GetTableColumnsInSchema()`

### 3. Worker wiring

- [worker_server.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/server/worker_server.go)
- Đã inject `registrySvc` vào `SchemaInspector`.

### 4. CommandHandler

- [command_handler.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/handler/command_handler.go)
- `HandleDiscover()` giờ introspect column theo schema đã resolve.
- `HandleBackfill()` giờ update bằng `schema.table` và quote `target_column` đúng cách.
