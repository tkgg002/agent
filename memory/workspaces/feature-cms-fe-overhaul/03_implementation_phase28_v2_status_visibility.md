# Phase 28 Implementation - V2 Status Visibility

- `SourceObjectsHandler.List`
  - thêm:
    - `shadow_binding_id`
    - `bridge_status`
    - `metadata_status`
- `GetMappingContext`
  - thêm các field status tương ứng
- `TableRegistry.tsx`
  - thêm cột `Metadata`
  - đổi warning copy theo semantics mới
  - modal register nói rõ row mới sẽ sync V2 ngay
