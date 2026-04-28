# Implementation — Phase 20 Mapping Context Read Model

## Backend

- Mở rộng `source_objects_handler.go`
- Thêm:
  - `SourceObjectMappingContext`
  - `GetMappingContext()`
- Route mới:
  - `GET /api/v1/source-objects/registry/:registry_id`
- Query hiện tại:
  - base từ `cdc_table_registry` vì route nhận `registry_id`
  - enrich bằng `cdc_system.source_object_registry`
  - enrich bằng `cdc_system.shadow_binding`
  - enrich bằng latest `cdc_reconciliation_report`
- Swagger:
  - thêm annotation cho endpoint mới

## Frontend

- Thêm type `SourceObjectMappingContext`
- Refactor `MappingFieldsPage.tsx`
  - load context từ `/api/v1/source-objects/registry/:id`
  - không còn fetch `/api/registry` toàn cục
  - dùng `shadow_schema`/`physical_table_fqn` từ read-model mới
  - giữ `create-default-columns` qua `registry_id`

## Kết quả mong muốn

- page mappings bớt phụ thuộc compatibility shell
- operator không mất capability hiện tại
- read path đúng kiến trúc hơn
