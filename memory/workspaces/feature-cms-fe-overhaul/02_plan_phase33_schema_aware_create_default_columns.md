# Phase 33 Plan — Schema-aware Create Default Columns

1. Audit worker payload + DDL assumptions cho `create-default-columns`.
2. Mở direct CMS route theo `source_object_id`.
3. Mở rộng worker payload:
   - `source_object_id`
   - `shadow_schema`
4. Sửa worker `create-default-columns` và `standardize` thành schema-aware.
5. Refactor FE `TableRegistry` và `MappingFieldsPage`.
6. Verify:
   - `go test ./...` ở `cdc-cms-service`
   - `go test ./internal/service ./internal/handler ./internal/server` ở `centralized-data-service`
   - `npm run build` ở `cdc-cms-web`
