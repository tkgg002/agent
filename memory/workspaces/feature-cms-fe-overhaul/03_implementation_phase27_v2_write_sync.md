# Phase 27 Implementation - V2 Write Sync

## Quyết định kỹ thuật

- Chưa bỏ ngay `cdc_table_registry`.
- Chọn hướng an toàn: `legacy write gatekeeper`, `V2 sync follower`.

## Thay đổi

- Thêm `SourceObjectV2SyncService`.
- Service này:
  - resolve source connection
  - resolve shadow connection
  - upsert `cdc_system.source_object_registry`
  - upsert `cdc_system.shadow_binding`
- `RegistryHandler.Register`
- `RegistryHandler.Update`
- `RegistryHandler.BulkRegister`
  - đều gọi sync service sau write thành công
