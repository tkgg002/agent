# Plan — Phase 11 Activity Log Scope

1. Audit `activity_log_handler.go`, `ActivityLog.tsx`, `useAsyncDispatch.ts`, và metadata V2.
2. Refactor backend `activity-log`:
   - enrich list response bằng source/shadow scope
   - add scope filters
   - enrich `recent_errors`
   - update swagger/comment
3. Refactor `useAsyncDispatch`:
   - add `statusParams`
   - keep `targetTable` as compatibility fallback
4. Refactor `ActivityLog.tsx` để render scope V2.
5. Verify bằng `go test ./...` và `npm run build`.
