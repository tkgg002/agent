# Plan — Phase 22 Transform Status Facade

1. Audit các call-site còn trỏ trực tiếp `/api/registry`.
2. Thêm facade cho `transform-status`.
3. Refactor FE `TableRegistry`.
4. Verify:
   - `go test ./...`
   - `npm run build`
   - `make swagger` nếu tool có sẵn
5. Ghi rõ phần nào của `/api/registry` sẽ còn giữ lại như compatibility shell thật.
