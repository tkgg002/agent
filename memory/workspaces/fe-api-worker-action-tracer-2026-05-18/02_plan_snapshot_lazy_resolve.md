# Plan — Snapshot Now lazy resolve

## Step 1 — `internal/handler/recon_handler.go`
1. Import `go.mongodb.org/mongo-driver/mongo/options`.
2. `HandleDebeziumSignal` dispatch nhánh đổi `if/else` thành `switch`:
   - Case A `signalConfigured` → `signal_client`.
   - Case B `h.mongoClient != nil` → `mongo_shared_client` (giữ behavior cũ).
   - Default → `mongo_lazy_resolve`: gọi `resolveSourceMongoDSN(table)` → `mongo.Connect(dsn)` → `insertDebeziumSignal` → `Disconnect`.
3. Tạo helper `insertDebeziumSignal(ctx, client, db, collection) (signalID, err)` — shared giữa shared client path và lazy connect path.
4. Tạo helper `resolveSourceMongoDSN(ctx, targetTable) (dsn, err)`:
   - `h.metadata.ResolveTargetRoute(targetTable) → *ResolvedSourceRoute`.
   - `route.SourceObject.SourceConnectionID` → `connection_registry` row.
   - URI prefix (`mongodb://`/`mongodb+srv://`) → dùng thẳng. Bare host → `mongodb://host:port/`.

## Step 2 — `internal/server/worker_server.go`
1. Else branch của `if reconCore != nil`:
   - Tạo minimal `signalOnlyHandler := handler.NewReconHandler(nil, db, nil, schemaAdapter, logger).WithMetadataRegistry(registrySvc).WithMaskingService(maskingSvc)`.
   - Subscribe `cdc.cmd.debezium-signal` + `cdc.cmd.debezium-snapshot` vào signalOnlyHandler.
2. Stub fallback (reconNotConfigured) chỉ còn 5 subject: recon-check, recon-heal, retry-failed, recon-backfill-source-ts, detect-timestamp-field.

## Step 3 — Config
- Revert `config-local.yml`: bỏ `mongodb:` block (đã thêm tạm thời).

## Step 4 — Verify
- `go build ./...` PASS.
- `go vet ./...` PASS.
- `go test -count=1 ./internal/handler/... ./internal/server/...` PASS.

## Step 5 — Docs
- APPEND `05_progress.md`.
- Tạo `08_tasks_snapshot_lazy_resolve.md` + `09_tasks_solution_snapshot_lazy_resolve.md`.

## Risk

- **R1**: Mỗi click Snapshot Now mở 1 mongo connection mới → overhead. Mitigation: lazy resolve chỉ chạy khi shared client không có. Production có thể set `cfg.MongoDB.URL` để dùng shared client lại (path B). Đây chỉ là fallback cho local dev / dynamic source.
- **R2**: `h.db.WithContext(ctx).First(&conn, connID).Error` mỗi click query DB → minor. ACCEPT vì click frequency thấp.
- **R3**: `mongo.Connect` không retry/backoff. Nếu source down → click fail. ACCEPT (user fix URI rồi click lại).
- **R4**: `Disconnect` defer trong context với timeout 15s — nếu InsertOne treo lâu hơn 15s, context cancel kills the request nhưng client cleanup background. ACCEPT.
