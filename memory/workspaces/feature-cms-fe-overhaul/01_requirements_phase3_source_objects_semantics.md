# Requirements — Phase 3 Source Objects Semantics

## Mục tiêu

- Đẩy `TableRegistry` tiến gần hơn tới semantics `Source Objects`.
- Làm rõ cho operator rằng mỗi row là:
  - source object
  - shadow target
  - context để tạo master
- Giữ compatibility với backend transitional hiện tại, không ép đổi contract API trong cùng phase.

## Phạm vi

- `src/pages/TableRegistry.tsx`
- `src/pages/MasterRegistry.tsx`

## Điều kiện hoàn thành

1. `TableRegistry` hiển thị được shadow namespace theo convention `shadow_<source_db>.<table>`.
2. Hành động đi sang `MasterRegistry` mang đủ context shadow cho operator.
3. `MasterRegistry` nói thật về contract legacy hiện tại của `source_shadow`.
4. FE build pass.
