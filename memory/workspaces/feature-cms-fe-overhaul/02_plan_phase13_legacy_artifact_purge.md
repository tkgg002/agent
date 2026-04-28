# Plan — Phase 13 Legacy Artifact Purge

## Kế hoạch thực hiện

1. Audit usage của `QueueMonitoring`, `CDCInternalRegistry`, `CDCInternalRegistryHandler`.
2. Xác nhận:
   - route đã gỡ
   - menu đã gỡ
   - server/router không còn wire handler
3. Xóa vật lý các file legacy.
4. Verify:
   - grep không còn reference sống
   - `go test ./...` ở `cdc-cms-service`
   - `npm run build` ở `cdc-cms-web`
5. Ghi đầy đủ workspace docs và append immutable progress log.
