# Plan — Phase 10 Mapping Rules V2

1. Audit `mapping_rule_handler.go`, `MappingFieldsPage.tsx`, `AddMappingModal.tsx`, type definitions, và schema `cdc_system.mapping_rule_v2`.
2. Refactor backend `mapping-rules`:
   - list từ `mapping_rule_v2`
   - create theo V2 scope
   - reload/backfill/batch update resolve qua shadow target
   - update swagger/comment
3. Refactor FE `MappingFieldsPage` + `AddMappingModal`:
   - gửi source/shadow context thực
   - dùng response V2 nhưng giữ compatibility với page hiện tại
4. Verify bằng `go test ./...` và `npm run build`.
