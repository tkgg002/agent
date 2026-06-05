# Tasks — Snapshot Now lazy resolve

- [x] Revert `config-local.yml` — bỏ `mongodb:` block.
- [x] `recon_handler.go`: import `mongo/options`.
- [x] `recon_handler.go` `HandleDebeziumSignal`: switch 3 nhánh (signal_client / mongo_shared_client / mongo_lazy_resolve).
- [x] `recon_handler.go`: helper `insertDebeziumSignal(ctx, client, db, collection)`.
- [x] `recon_handler.go`: helper `resolveSourceMongoDSN(ctx, targetTable)` — qua `metadata.ResolveTargetRoute` → `connection_registry`.
- [x] `worker_server.go` else branch: tạo `signalOnlyHandler`, subscribe debezium-signal/snapshot. Stub fallback chỉ còn 5 subject.
- [x] Worker `go build ./...` PASS.
- [x] Worker `go vet ./...` PASS.
- [x] Worker `go test -count=1 ./internal/handler/... ./internal/server/...` PASS (handler 3.972s).
- [x] APPEND `05_progress.md`.
- [x] Tạo `09_tasks_solution_snapshot_lazy_resolve.md`.
