# Tasks — Snapshot source-key + Sync Fields field count

- [x] `recon_handler.go::resolveSourceMongoDSN`: param `table`, dùng `resolveTargetTableConfig` chain, sau đó `ResolveTargetRoute(entry.TargetTable)`.
- [x] `command_handler.go::HandleCreateDefaultColumns`: pre-fetch `existingCols`, count `columnsAlreadyExist`.
- [x] `command_handler.go`: helper `listShadowColumns(schema, table) map[string]struct{}`.
- [x] `go build ./...` PASS.
- [x] `go vet ./...` PASS.
- [x] `go test -count=1 ./internal/handler/... ./internal/server/...` PASS (handler 3.928s).
- [x] APPEND `05_progress.md`.
- [x] Tạo `09_tasks_solution_*.md`.
