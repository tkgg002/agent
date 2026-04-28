# Phase 31 Plan — Direct Scan Fields & Transform Status

1. Audit worker payload contract cho `scan-fields` và `transform-status`.
2. Mở CMS direct routes theo `source_object_id` cho 2 action/read này.
3. Refactor FE `useRegistry` và `TableRegistry` sang ưu tiên direct V2.
4. Giữ bridge routes làm fallback.
5. Verify `go test ./...` và `npm run build`.
6. Ghi docs + append progress/status/gap.
