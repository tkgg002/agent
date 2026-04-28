# Implementation — Phase 36 Schema Tail Cleanup

## Audit kết quả

- `event_bridge` có test nhưng không thấy wiring runtime sống; tuy nhiên poll path vẫn nên harden để tránh đụng nhầm `public` nếu bật lại.
- `transform_service` hiện không có caller runtime, nhưng vẫn là helper sống trong repo nên nên được làm schema-aware tối thiểu.
- `cms registry_repo` có các helper raw SQL cũ (`ScanRawKeys`, `PerformBackfill`, `GetDBColumns`) dù chưa có call-site mới, nhưng đây là compatibility shell hợp lý để chuẩn bị schema-aware path.

## Thay đổi đã áp dụng

### 1. Event bridge

- [event_bridge.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/handler/event_bridge.go)
- Thêm:
  - `quoteEventBridgeIdent()`
  - `quoteEventBridgeQualifiedTable()`
  - `resolveTargetSchema()`
- `pollChanges()` giờ query theo `schema.table` thay vì `"target_table"` mặc định.

### 2. Transform service

- [transform_service.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/transform_service.go)
- Thêm:
  - `metadata MetadataRegistry`
  - `SetMetadataRegistry()`
  - helper resolve schema/qualified table
- `BatchTransform()` giờ update theo `schema.table`.

### 3. CMS registry repository

- [registry_repo.go](/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/repository/registry_repo.go)
- Thêm schema-aware wrappers:
  - `ScanRawKeysInSchema()`
  - `PerformBackfillInSchema()`
  - `GetDBColumnsInSchema()`
- Method cũ vẫn giữ làm wrapper fallback về `public`.
