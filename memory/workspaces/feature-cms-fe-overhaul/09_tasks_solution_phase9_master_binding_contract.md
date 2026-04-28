# Solution — Phase 9 Master Binding Contract

## Hướng xử lý

- Không tạo route mới.
- Chuyển toàn bộ `v1/masters` sang dùng `cdc_system.master_binding`.
- Dùng `shadow_schema + shadow_table` làm identity operator-facing chính.
- Chỉ giữ `source_shadow` ở mức fallback/compatibility.

## Definition of Done

1. Không còn query/insert/update `cdc_internal.master_table_registry` trong master API.
2. FE không còn buộc operator nhập “legacy shadow identifier”.
3. `TableRegistry -> MasterRegistry` truyền đủ context V2.
4. Swagger/comment đồng bộ với contract mới.
5. `go test ./...` và `npm run build` pass.
