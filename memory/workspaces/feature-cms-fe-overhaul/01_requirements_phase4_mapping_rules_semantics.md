# Requirements — Phase 4 Mapping Rules Semantics

## Mục tiêu

- Đổi `MappingFieldsPage` sang ngôn ngữ `Mapping Rules`.
- Hiển thị rõ context:
  - source object
  - shadow target
- Giảm bớt các assumption operator-facing dựa hoàn toàn vào `target_table`.

## Phạm vi

- `src/pages/MappingFieldsPage.tsx`
- `src/components/AddMappingModal.tsx`

## Điều kiện hoàn thành

1. Page title/copy phản ánh `Mapping Rules`.
2. Operator nhìn thấy source object và shadow target hiện hành.
3. Các note compatibility nói rõ backend vẫn đang dùng contract legacy.
4. Build FE pass.
