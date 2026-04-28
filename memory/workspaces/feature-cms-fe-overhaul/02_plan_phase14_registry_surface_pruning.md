# Plan — Phase 14 Registry Surface Pruning

## Kế hoạch thực hiện

1. Audit `registry_handler.go`, router và FE call sites.
2. Chia endpoint thành:
   - operator-flow còn sống
   - transitional nhưng còn caller
   - dead/retired không còn caller
3. Gỡ route dead khỏi router.
4. Xóa dead handler tương ứng.
5. Cập nhật swagger/comment của nhóm endpoint còn sống sang semantics `Source Objects`.
6. Verify bằng `go test ./...`, `npm run build`, và grep route dead.
