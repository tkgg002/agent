# Plan — Phase 19 Shadow Bindings Dual View

1. Audit API và call-site hiện có liên quan `shadow_binding`.
2. Thêm read API tối thiểu `GET /api/v1/shadow-bindings`.
3. Refactor `TableRegistry` thành dual-view:
   - Source Objects
   - Shadow Bindings
4. Verify:
   - `go test ./...`
   - `npm run build`
   - `make swagger` nếu tool có sẵn
5. Ghi gap còn lại cho phase sau.
