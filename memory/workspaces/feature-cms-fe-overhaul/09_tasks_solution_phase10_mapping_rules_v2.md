# Solution — Phase 10 Mapping Rules V2

## Hướng xử lý

- Không tạo route mới.
- Chuyển logic API `mapping-rules` sang `cdc_system.mapping_rule_v2`.
- Dùng source/shadow metadata V2 để resolve scope.
- Giữ `source_table` / `table` như fallback tạm thời để không gãy UI cũ ngay lập tức.

## Definition of Done

1. API `mapping-rules` không còn đọc/ghi `cdc_mapping_rules`.
2. FE gửi được source/shadow context thực khi list/create/reload.
3. Backfill/reload/batch update dispatch đúng shadow target.
4. Swagger/comment đồng bộ với contract mới.
5. `go test ./...` và `npm run build` pass.
