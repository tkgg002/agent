# Plan — Sync Fields + Snapshot Now Visibility

## Step 1 — `internal/handler/command_handler.go` `HandleCreateDefaultColumns`
1. Block load rules → đổi `if err == nil` thành `if err != nil { Error log }` + else `Info log` count + warn nếu len(rules)==0.
2. Loop ALTER TABLE → mỗi rule: log column+type trước, log warn nếu fail (đã có) + tăng `columnsSkipped`; cuối loop log summary `rules_total / columns_added / columns_skipped`.
3. Tất cả log kèm `trace_id` để grep theo correlation id của FE click.

## Step 2 — `internal/handler/command_handler.go` `processDiscoveryRows`
1. Đổi `if err == nil { added++ }` thành `if err != nil { Warn log; continue } added++`.
2. Cuối hàm log summary `discovered_total / already_mapped / inserted / insert_errors`.

## Step 3 — `internal/handler/recon_handler.go` `HandleDebeziumSignal`
1. Thêm var `dispatchPath` string ("signal_client" vs "mongo_direct_insert").
2. Trước khi chạy `TriggerIncrementalSnapshot`, log Info "using SignalClient path" + db+collection.
3. Trước khi chạy mongo direct insert, log Info "using MongoClient direct-insert fallback" + signal_collection.
4. Nếu mongoClient nil → Warn kèm hint `worker_server.go gating reconCore=nil`.
5. Err log kèm `dispatch_path`, `database`, `collection`.
6. Success log kèm `dispatch_path`.

## Step 4 — Verify
- `go build ./...` PASS.
- `go vet ./...` PASS.
- `go test ./internal/handler/... ./internal/service/...` PASS (không break test hiện có).

## Step 5 — Docs
- APPEND `05_progress.md`.
- Tạo `08_tasks_sync_snapshot_visibility.md` checkbox + `09_tasks_solution_sync_snapshot_visibility.md` diff.

## Risk
- **R1**: Log volume tăng — mỗi rule là 1 dòng. Mitigation: rule count thường <50 cho 1 table.
- **R2**: Nếu rule có data_type rỗng (legacy seed), ALTER sẽ fail. Log Warn đã capture; không cần block.
