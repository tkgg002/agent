# Implementation — Phase 34 Shadow Schema Runtime Hardening

## Audit kết quả

- `SchemaValidator.introspectColumns()` vẫn query `information_schema.columns` với `table_schema='public'`.
- `CommandHandler` còn nhiều query/update trực tiếp trên `"target_table"` không qualify schema.
- `drop-gin-index` cũng drop index không qualify schema.

## Thay đổi đã áp dụng

### 1. Metadata registry

- Thêm method `ResolveTargetRoute(targetTable string)` vào `MetadataRegistry`.
- `MetadataRegistryService` giữ thêm `targetRouteMap` để lookup O(1) theo `target_table`.
- `RegistryService` legacy cũng implement method tương thích.
- Test `metadata_registry_service_test.go` được mở rộng để cover lookup theo `target_table`.

### 2. Schema validator

- `introspectColumns()` giờ resolve `shadow_schema` từ `MetadataRegistry.ResolveTargetRoute()`.
- Fallback vẫn là `public` nếu route chưa có.

### 3. Command handler

- Thêm helper:
  - `quoteCommandQualifiedTable(schema, table)`
  - `hasColumnInSchema()`
  - `resolveTargetRoute()`
  - `resolveTargetSchema()`
- Các path sau đã schema-aware:
  - `HandleBatchTransform`
  - `HandleScanRawData`
  - `HandlePeriodicScan`
  - `HandleDropGINIndex`
  - `scanFieldsDebezium`
- `tableExists()` và `hasColumn()` giờ infer schema theo target route thay vì mặc định `public`.
- `DROP INDEX` ở path cleanup giờ qualify theo schema của shadow table.
