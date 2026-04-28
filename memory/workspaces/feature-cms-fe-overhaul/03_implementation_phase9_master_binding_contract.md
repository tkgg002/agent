# Implementation — Phase 9 Master Binding Contract

- Backend:
  - thay `master_registry_handler.go` sang V2 metadata
  - query `cdc_system.master_binding` + join `shadow_binding`, `source_object_registry`, `connection_registry`
  - create path resolve shadow binding và master connection
  - approve/reject/toggle dùng `master_binding`
  - thêm swagger/comment cho list/create/approve/reject/toggle/swap
- Frontend:
  - `TableRegistry` truyền thêm `shadow_schema`, `shadow_table`
  - `MasterRegistry` render `master_schema.master_name`
  - form create nhận `master_schema`, `shadow_schema`, `shadow_table`
  - API payload dùng V2 metadata, `source_shadow` chỉ là fallback
