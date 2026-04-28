# Implementation — Phase 4 Mapping Rules Semantics

## `src/pages/MappingFieldsPage.tsx`

- Thêm helper:
  - `normalizeShadowSchema(sourceDB)`
  - `getShadowFqn(registry)`
- Đổi title:
  - `Mapping Fields` -> `Mapping Rules`
- Thêm `Alert` context đầu trang:
  - source object
  - shadow target
  - note rằng API hiện vẫn dựa vào `source_table` + `target_table`
- Mở rộng `Descriptions`:
  - thêm `Shadow Schema`
  - thêm `Shadow Target`
- Đổi copy:
  - `Back to Registry` -> `Back to Source Objects`
  - `System Default Fields (auto-created)` -> `System Default Fields (auto-created on shadow table)`
  - `Custom Mapping Rules` -> `Mapping Rules`
  - `unmapped fields found in _raw_data` -> `unmapped fields found in shadow raw payload`
  - `Cập nhật Field vào Table` -> `Sync Fields to Shadow`
  - preview modal `Shadow table` -> `Shadow target`
- Điều chỉnh một số message để không gợi ý nhầm sang Airbyte path chính.

## `src/components/AddMappingModal.tsx`

- Đổi label:
  - `Source Table` -> `Source Object Table`
  - `Source Field (in _raw_data)` -> `Source Field (from shadow raw payload)`
  - `Target Column (PostgreSQL)` -> `Target Column`
- Thêm tooltip nói rõ backend legacy vẫn nhận `source_table` làm identity chính.

## Kết quả

- Operator nhìn `MappingFieldsPage` sẽ hiểu page này là lớp rule của source object/shadow target.
- FE không giả vờ backend đã V2-native; thay vào đó nói rõ phần transitional.
