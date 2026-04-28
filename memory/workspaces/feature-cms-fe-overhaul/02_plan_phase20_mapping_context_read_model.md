# Plan — Phase 20 Mapping Context Read Model

1. Audit `MappingFieldsPage` và các endpoint nó đang gọi.
2. Thêm detail/read-model V2 theo `registry_id`.
3. Refactor page mappings dùng API mới thay vì tải cả `/api/registry`.
4. Giữ `create-default-columns` trên legacy path để không gãy operator-flow.
5. Verify:
   - `go test ./...`
   - `npm run build`
   - `make swagger` nếu tool có sẵn
