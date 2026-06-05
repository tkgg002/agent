# Plan — Fix MongoDB Direct Connection Bug

## Step 1 — `internal/service/mongo_introspection.go`
1. `DiscoverDatabases` (line 23): xóa `.SetDirect(true)` → `mongo.Connect(ctx, options.Client().ApplyURI(uri))`.
2. `DiscoverCollections` (line 41): xóa `.SetDirect(true)`.
3. `IntrospectCollection` (line 60): xóa `.SetDirect(true)`.
4. Không cần thay đổi import, không cần helper mới.

## Step 2 — Verify
- `cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go build ./...` → EXIT=0.
- `go vet ./...` → EXIT=0.
- `go test ./internal/service/... ./internal/handler/...` → EXIT=0.

## Step 3 — Docs
- APPEND `05_progress.md` với 1-2 dòng diff summary + verify result.
- Tạo `08_tasks_mongo_direct_fix.md` (checkbox).
- Tạo `09_tasks_solution_mongo_direct_fix.md` (diff snippet + grep cheatsheet).

## Risk

- **R1**: SetDirect(true) có thể từng có lý do (e.g. single-host dev mongo không lên replica set). Mitigation: driver default behavior với URI `mongodb://localhost:27017/` (không có replicaSet) là direct connection. Vẫn work.
- **R2**: Production prod-mongo có replicaSet → trước đây failed; sau khi fix sẽ work. Không có regression.
- **R3**: Test ngừng pass? — `internal/service/mongo_introspection*_test.go` không tồn tại (đã grep), nên không có unit test trực tiếp; chỉ rely vào build/vet.

## Out of scope (đề xuất next task)

- Refactor `worker_server.go:164` để reconCore lazy-init qua `connection_registry`. Lý do: hiện tại nếu user không config `mongodb.url` trong worker config-local.yml → toàn bộ recon path (Snapshot Now, recon-check, recon-heal, retry-failed, backfill-source-ts, detect-timestamp-field) sẽ chỉ log error stub. Lazy-init giúp user manage Mongo qua CMS connection_registry mà không cần restart worker.
