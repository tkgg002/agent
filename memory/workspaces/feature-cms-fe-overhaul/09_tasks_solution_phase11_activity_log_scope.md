# Solution — Phase 11 Activity Log Scope

## Hướng xử lý

- Không thêm endpoint mới.
- Enrich trực tiếp `GET /api/activity-log` và `GET /api/activity-log/stats`.
- `useAsyncDispatch` chấp nhận `statusParams` để caller dần bỏ assumption `target_table`.
- Giữ `target_table` như compatibility fallback.

## Definition of Done

1. `activity-log` trả source/shadow context V2.
2. `activity-log` filter được theo source/shadow metadata.
3. `useAsyncDispatch` hỗ trợ params giàu hơn.
4. Swagger/comment đồng bộ.
5. `go test ./...` và `npm run build` pass.
