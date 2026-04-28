# Implementation — Phase 3 Source Objects Semantics

## `src/pages/TableRegistry.tsx`

- Thêm helper:
  - `normalizeShadowSchema(sourceDB)`
  - `getShadowFqn(record)`
- Đổi success message:
  - `Table registered` -> `Source object registered`
- Đổi column:
  - `Target Table` -> `Shadow Table`
- Thêm column mới:
  - `Shadow Target`
  - hiển thị `shadow_<source_db>.<target_table>`
  - hiển thị riêng `schema=shadow_<source_db>`
- Đổi action copy:
  - `Register Table` -> `Register Source Object`
  - `Bulk Import (JSON)` -> `Bulk Import Source Objects`
  - panel count `tables` -> `objects`
- Đổi register modal:
  - title `Register New Source Object`
  - helper text giải thích registry hiện vẫn transitional
  - field `Target Table` -> `Shadow Table Name`
- Khi đi sang `MasterRegistry`, FE truyền thêm:
  - `source_shadow`
  - `source_label`
  - `source_db`
  - `source_table`

## `src/pages/MasterRegistry.tsx`

- Đọc `searchParams` để:
  - auto-fill `source_shadow`
  - auto-open create modal
- Hiển thị alert context:
  - shadow namespace thật cho operator
  - source object gốc
  - note rõ backend hiện vẫn nhận `source_shadow` theo contract legacy
- Khi đóng modal từ flow contextual, tự dọn query params.
- Sửa placeholder để không nói sai rằng backend đã nhận `shadow_<db>.<table>`.

## Kết quả

- FE đã hiển thị shadow namespace theo target architecture.
- Luồng từ source object sang master creation rõ nghĩa hơn cho operator.
- Không phá current backend `master_registry_handler`, vốn vẫn validate `source_shadow` theo regex legacy.
