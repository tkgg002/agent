# Plan — Phase 7 Backend API Pruning

1. Audit route usage giữa FE và CMS service.
2. Chốt các route legacy không còn được FE dùng và không còn đúng Debezium-only.
3. Prune router + server wiring.
4. Cập nhật swagger/comment để tránh docs nói sai.
5. Verify bằng `go test ./...` và `npm run build`.
