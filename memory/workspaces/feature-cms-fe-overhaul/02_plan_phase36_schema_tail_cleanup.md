# Plan — Phase 36 Schema Tail Cleanup

1. Audit caller thật của `event_bridge`, `transform_service`, `registry_repo`.
2. Hardening `event_bridge` theo `shadow_schema`.
3. Hardening `transform_service` theo cùng pattern, không đổi wiring nếu chưa có caller.
4. Thêm schema-aware wrappers vào `cms registry_repo`.
5. Verify:
   - `gofmt`
   - `go test` ở worker
   - `go test` ở CMS service
6. Ghi docs phase và append progress/status/gap.
