# Phase 27 Solution - V2 Write Sync

## Quyết định chính

- Chưa cố viết write-model V2 hoàn chỉnh ngay.
- Mỗi write legacy thành công sẽ sync sang V2 để `cdc_system` dần trở thành source-of-truth chính.

## Kết quả thực tế

- `Register/Update/BulkRegister` giờ sync thêm sang:
  - `cdc_system.source_object_registry`
  - `cdc_system.shadow_binding`
- compile/test backend pass
- chưa thay FE contract ở phase này
