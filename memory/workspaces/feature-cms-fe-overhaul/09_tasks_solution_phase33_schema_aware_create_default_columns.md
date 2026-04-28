# Phase 33 Solution — Schema-aware Create Default Columns

## Giải pháp chốt
- Không chỉ direct-V2 hóa `create-default-columns`, mà direct-V2 hóa theo cách schema-aware.
- Worker path giờ có thể phản ánh trạng thái create thành công về `cdc_system.shadow_binding`, nên V2-only row không còn mù trạng thái sau action.

## Ý nghĩa thực chiến
- Đây là bước đầu tiên operator-flow không còn chỉ “đi vòng qua bridge cũ” cho action DDL quan trọng nhất.
- CMS FE bớt phụ thuộc `registry_id` ở cả:
  - `TableRegistry`
  - `MappingFieldsPage`

## Còn lại
- Cần audit sâu hơn các action mapping/sync còn dùng bridge.
- Cần cân nhắc dọn dần legacy `is_table_created` khi V2 metadata đủ mạnh.
