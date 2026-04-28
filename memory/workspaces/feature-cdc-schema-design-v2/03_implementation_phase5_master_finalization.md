# Implementation — Phase 5 Master Finalization

## Code Changes

- `internal/service/master_ddl_generator.go`
  - thêm `SyncRuntimeStateRepo`
  - ghi `ddl_status` + `last_success_at/last_error_at`
  - expose `EnsureMaster(ctx, masterName)` để transmuter gọi trước khi upsert

- `internal/service/transmuter.go`
  - nhận thêm `SyncRuntimeStateRepo` và `MasterDestinationEnsurer`
  - auto-ensure master destination trước khi xử lý batch
  - ghi runtime state `success/failure/skipped` cho `master_binding`

- `internal/handler/transmute_handler.go`
  - payload `transmute-shadow` hỗ trợ:
    - `shadow_schema`
    - `shadow_connection_key`
    - `shadow_binding_code`
  - query master bindings ưu tiên identity-aware lookup

- `internal/sinkworker/sinkworker.go`
  - publish `transmute-shadow` với:
    - `shadow_schema=cdc_internal`
    - `shadow_connection_key=default`

- `internal/server/worker_server.go`
  - wiring thêm `SyncRuntimeStateRepo`
  - dùng chung `MasterDDLGenerator` cho cả command path và transmuter ensure path

## Why This Matters

- Tránh ambiguity khi nhiều shadow bindings có cùng `shadow_table` nhưng khác schema/connection.
- Giảm phụ thuộc vận hành thủ công: master path có thể tự chuẩn bị namespace/table trước lúc transmute.
- Tạo dấu vết runtime state rõ hơn để sau khi wipe dữ liệu có thể quan sát bootstrap mới.
