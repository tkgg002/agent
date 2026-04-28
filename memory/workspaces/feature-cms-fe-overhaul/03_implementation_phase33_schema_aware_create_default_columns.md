# Phase 33 Implementation — Schema-aware Create Default Columns

## CMS backend
- `source_object_actions_handler.go`
  - mở rộng `sourceObjectDispatchScope` với:
    - `shadow_schema`
    - `primary_key_field`
    - `primary_key_type`
  - thêm direct route:
    - `POST /api/v1/source-objects/{id}/create-default-columns`
- `router.go`
  - mount route direct mới

## Worker backend
- `command_handler.go`
  - thêm helper schema-aware:
    - `ensureCDCColumnsInSchema`
    - `tableExistsInSchema`
  - `HandleStandardize` giờ hiểu `shadow_schema`
  - `HandleCreateDefaultColumns` giờ hiểu:
    - `source_object_id`
    - `shadow_schema`
  - worker update thêm:
    - `cdc_system.shadow_binding.ddl_status = 'created'`
  - vẫn giữ update legacy bridge `is_table_created` theo `target_table` để compatibility không gãy

## FE
- `TableRegistry.tsx`
  - `handleCreateTable()` ưu tiên direct V2 route cho row V2-only
- `MappingFieldsPage.tsx`
  - `handleSyncFields()` ưu tiên direct V2 route khi có `source_object_id`

## Root-cause fix
- Sửa thẳng debt “public schema assumption” cho operator-flow path thay vì chỉ thêm facade.
