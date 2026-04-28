# Phase 33 Requirements — Schema-aware Create Default Columns

## Mục tiêu
- Xử lý debt lớn còn lại của operator-flow: `create-default-columns`.
- Không chỉ bọc API, mà phải sửa worker path để không còn ngầm assume `public.<target_table>`.

## Audit kết luận
- `create-default-columns` legacy worker path đang hardcode `public` và update `is_table_created` qua `cdc_table_registry`.
- `standardize` direct route mới mở ở phase trước cũng còn tiềm ẩn schema drift nếu worker chỉ nhìn `public`.
- `source_objects` read-model hiện đã support `sb.ddl_status = 'created'` như nguồn sự thật V2 cho trạng thái table-created.

## Điều kiện Done
- CMS có direct route `create-default-columns` theo `source_object_id`
- Worker nhận được `shadow_schema`
- Worker update được `shadow_binding.ddl_status='created'`
- FE `TableRegistry` và `MappingFieldsPage` dùng được direct path mới
