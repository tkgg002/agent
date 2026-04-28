# Plan — Phase 18 Source Objects Read Path V2

1. Audit contract thật của `TableRegistry` và action dependencies.
2. Thiết kế read-model mới cho `GET /api/v1/source-objects`.
3. Implement backend handler + route + wiring + swagger annotations.
4. Refactor FE `TableRegistry` sang read path mới.
5. Giữ action legacy trên `/api/registry`, nhưng disable minh bạch khi thiếu `registry_id`.
6. Verify:
   - `go test ./...`
   - `npm run build`
   - thử regenerate swagger nếu tool có sẵn
7. Ghi gap còn lại cho phase sau.
