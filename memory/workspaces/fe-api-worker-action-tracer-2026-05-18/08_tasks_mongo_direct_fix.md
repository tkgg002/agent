# Tasks — Fix MongoDB Direct Connection Bug

- [x] `internal/service/mongo_introspection.go` `DiscoverDatabases`: xóa `.SetDirect(true)`.
- [x] `internal/service/mongo_introspection.go` `DiscoverCollections`: xóa `.SetDirect(true)`.
- [x] `internal/service/mongo_introspection.go` `IntrospectCollection`: xóa `.SetDirect(true)`.
- [x] Comment giải thích vì sao bỏ — driver auto-detect topology.
- [x] Worker `go build ./...` PASS.
- [x] Worker `go vet ./...` PASS.
- [x] Worker `go test -count=1 ./internal/service/... ./internal/handler/...` PASS (service 0.893s, handler 4.378s).
- [x] APPEND `05_progress.md`.
- [x] Tạo `09_tasks_solution_mongo_direct_fix.md`.
