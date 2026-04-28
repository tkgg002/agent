# Plan — Phase 21 Registry Bridge Action Facade

1. Audit action nào còn thật sự cần `registry_id`.
2. Thêm facade handler dưới namespace `source-objects`.
3. Chuyển FE call-sites sang facade mới.
4. Verify:
   - `go test ./...`
   - `npm run build`
   - `make swagger` nếu tool có sẵn
5. Ghi rõ gap còn lại: action nào vẫn thực sự legacy ở semantics backend.
