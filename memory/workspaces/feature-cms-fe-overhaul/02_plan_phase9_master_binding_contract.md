# Plan — Phase 9 Master Binding Contract

1. Audit `master_registry_handler.go`, `MasterRegistry.tsx`, `TableRegistry.tsx`, và metadata V2 (`master_binding`, `shadow_binding`, `source_object_registry`, `connection_registry`).
2. Refactor backend `v1/masters`:
   - list từ `cdc_system.master_binding`
   - create resolve qua V2 shadow/master metadata
   - approve/reject/toggle dùng bảng V2
   - update swagger/comment
3. Refactor FE `MasterRegistry`:
   - render master namespace thật
   - submit `master_schema + shadow_schema + shadow_table`
   - chỉ giữ `source_shadow` như compatibility fallback
4. Refactor `TableRegistry` truyền đủ context khi deep-link sang `MasterRegistry`.
5. Verify bằng `go test ./...` và `npm run build`.
