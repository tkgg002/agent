# Solution Phase 2 — Metadata Registry

## Outcome

Hệ thống đã bước qua mốc quan trọng:

- không còn phụ thuộc hoàn toàn vào `cdc_table_registry` cho ingest lookup nữa
- routing source -> shadow đã bắt đầu đọc từ `cdc_system.source_object_registry` + `cdc_system.shadow_binding`

Nhưng mình chủ động giữ compatibility layer cho phần mapping và write path để không làm gãy worker đang chạy.

## Why this sequencing is correct

Nếu ép runtime đi full V2 ngay ở phase này thì sẽ gãy ở 2 điểm:

1. `BatchBuffer` chưa biết `schema`/`connection`
2. `DynamicMapper` hiện vẫn phục vụ shadow write path chứ chưa chỉ làm master projection

Vì vậy sequence đúng là:

1. V2 metadata registry
2. V2-aware event routing
3. write path carry `schema + connection key`
4. connection manager in write path
5. transmuter/master full V2

## Recommended next slice

1. Mở rộng `model.UpsertRecord`
   - thêm `SchemaName`
   - thêm `ConnectionKey`
   - có thể thêm `PhysicalTableFQN`
2. Refactor `BatchBuffer`
   - group theo `connection key + schema + table`
3. Refactor `SchemaAdapter/PrepareForCDCInsert`
   - nhận schema/table rõ ràng
4. Cắm `ConnectionManager` vào `EventHandler`/`BatchBuffer`
