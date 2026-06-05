# Plan — Fix Snapshot source-key + Sync Fields rows_affected miscount

## Step 1 — `internal/handler/recon_handler.go::resolveSourceMongoDSN`
1. Đổi param name `targetTable` → `table` (semantically can be source OR target).
2. Thay `route := h.metadata.ResolveTargetRoute(targetTable)` bằng:
   - `entry := h.resolveTargetTableConfig(table)` (cùng fallback chain target → sd_+target → source → DB).
   - `route := h.metadata.ResolveTargetRoute(entry.TargetTable)`.
3. Error message bao gồm cả input ban đầu lẫn target đã resolve để debug.

## Step 2 — `internal/handler/command_handler.go::HandleCreateDefaultColumns`
1. Thêm helper `listShadowColumns(schema, table) map[string]struct{}` truy vấn `information_schema.columns`.
2. Pre-fetch `existingCols` trước khi loop rules.
3. Trong loop: nếu column đã trong `existingCols` → `columnsAlreadyExist++` + `continue`. Không gọi ALTER.
4. Sau ALTER thành công: insert column vào set, `columnsAdded++`.
5. Log summary thêm field `columns_already_exist`.

## Step 3 — Verify
- `go build ./...` PASS.
- `go vet ./...` PASS.
- `go test -count=1 ./internal/handler/... ./internal/server/...` PASS.

## Step 4 — Docs
- APPEND `05_progress.md`.
- Tạo `08_tasks_*.md` + `09_tasks_solution_*.md`.

## Risk
- **R1** (snapshot): nếu `cdc_table_registry` có hai row trùng source_table → `GetTableConfigBySource` không deterministic. ACCEPT — existing behavior, log warn upstream đã có.
- **R2** (sync): `information_schema.columns` query mỗi click → cost không đáng kể. ACCEPT.
- **R3** (sync): race condition giữa pre-fetch và ALTER (concurrent admin) → IF NOT EXISTS vẫn safe. ACCEPT.
