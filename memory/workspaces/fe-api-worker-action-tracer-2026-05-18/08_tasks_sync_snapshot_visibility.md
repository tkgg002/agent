# Tasks — Sync Fields + Snapshot Now Visibility

- [x] `HandleCreateDefaultColumns`: log err của `GetActiveRulesBySourceTable` + Info count + Warn empty.
- [x] `HandleCreateDefaultColumns`: ALTER loop kèm trace_id + columns_skipped + summary.
- [x] `processDiscoveryRows`: log Warn cho mỗi Create fail + summary (discovered/already_mapped/inserted/insert_errors).
- [x] `HandleDebeziumSignal`: dispatch_path var; log Info nhánh signal_client / mongo_direct_insert; Warn kèm hint `worker_server gating`.
- [x] Worker `go build ./...` PASS.
- [x] Worker `go vet ./...` PASS.
- [x] Worker `go test ./internal/handler/... ./internal/service/...` PASS (handler 3.860s).
- [x] APPEND `05_progress.md`.
- [x] Tạo `09_tasks_solution_sync_snapshot_visibility.md`.
