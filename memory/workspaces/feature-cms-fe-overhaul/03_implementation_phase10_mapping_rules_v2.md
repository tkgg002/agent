# Implementation — Phase 10 Mapping Rules V2

- Backend:
  - thay `mapping_rule_handler.go` sang query/insert/update `cdc_system.mapping_rule_v2`
  - list query join `source_object_registry` + `shadow_binding`
  - create path resolve scope theo `source_object_id` hoặc source/shadow metadata
  - reload/backfill/batch update dispatch theo `shadow_table`
  - update swagger/comment cho list/create/reload/update/backfill
- Frontend:
  - update type `MappingRule`
  - `MappingFieldsPage` gọi list/reload với source/shadow params V2
  - `AddMappingModal` gửi hidden source/shadow metadata để create rule V2
