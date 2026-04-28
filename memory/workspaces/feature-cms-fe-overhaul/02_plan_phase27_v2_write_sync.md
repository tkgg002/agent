# Phase 27 Plan - V2 Write Sync

1. Audit dữ liệu đầu vào của `Register/Update/BulkRegister`.
2. Audit schema V2 cần sync.
3. Tạo service sync từ legacy row sang `source_object_registry` + `shadow_binding`.
4. Cắm service vào `RegistryHandler`.
5. Verify compile/test.
