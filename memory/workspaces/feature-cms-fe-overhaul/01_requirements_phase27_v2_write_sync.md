# Phase 27 Requirements - V2 Write Sync

## Mục tiêu

- Bắt đầu kéo write-path của source objects về `cdc_system`.
- Không làm gãy operator-flow hiện tại vốn vẫn còn dùng `cdc_table_registry` bridge.

## Yêu cầu

1. Sau mỗi `register/update/bulk register` thành công, metadata V2 phải được sync sang:
   - `cdc_system.source_object_registry`
   - `cdc_system.shadow_binding`
2. Logic hiện tại của CMS operator-flow không được gãy.
3. Không đổi FE contract ở phase này nếu chưa cần.

## Definition of Done

- Có service sync V2 từ legacy row.
- `RegistryHandler` gọi sync service sau write thành công.
- `go test ./...` pass.
