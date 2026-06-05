# Plan — MongoDB Connection Form → URL

## Step 1 — FE refactor (`cdc-cms-web/src/pages/SourceConnectors.tsx`)
1. Thêm `connectionUrl?: string` vào `ConnectionFormValues`.
2. Refactor `buildConnectorConfig` mongo branch: `mongodb.connection.string` = `values.connectionUrl.trim()`. Xóa hàm `buildMongoConnectionString` (không còn ai gọi).
3. Refactor `parseConnectionSeed` mongo branch: trả về `{ connectionUrl: source.server_address || '', database, collectionNames, ... }`. Bỏ logic tách host/port/username/replicaSet từ URL.
4. `openCreate` mongo defaults: `connectionUrl: 'mongodb://localhost:27017/?replicaSet=rs0'` (placeholder). Giữ host/port=27017 default cho lúc user switch sang mysql/postgres.
5. `openEdit`: set `connectionUrl: seed.connectionUrl || ''`.
6. UI: bọc Host+Port row, Username+Password row, ReplicaSet+Collections row sau cùng — show có điều kiện theo `dbKind`:
   - mongodb: render **Connection URL** (full width) + **Collections** (full width). Database+TopicPrefix row dùng chung.
   - mysql/postgresql: giữ Host+Port + Username+Password như cũ.
7. Validation field `connectionUrl`: required khi mongodb, pattern `^mongodb(\+srv)?:\/\/`.

## Step 2 — Repo refactor (`cdc-cms-service/internal/infra/persistence/system_connector_repo_gorm.go`)
1. `splitHostPort`: thêm branch đầu hàm — nếu `strings.HasPrefix(addr, "mongodb://")` || `mongodb+srv://` || `postgres://` || `postgresql://` || `mysql://` → `return addr, 0`. Comment giải thích "store URI as-is; worker resolves via prefix detect".

## Step 3 — Verify
- `cd cdc-cms-web && npm run build` PASS.
- `cd cdc-cms-service && go build ./... && go vet ./... && go test ./internal/api/... ./internal/infra/persistence/...` PASS.
- Không chạm worker. Chỉ cần đảm bảo no regression.

## Step 4 — Docs
- Append `05_progress.md` (immutable).
- Update `08_tasks_mongo_url_form.md` với checkboxes.
- Tạo `09_tasks_solution_mongo_url_form.md` với diff snippet.

## Risk & Mitigation
- **R1**: Connector cũ tạo trước refactor có `server_address` rỗng → Edit form sẽ trống URL. Mitigation: giữ logic `parseConnectionSeed` fallback `cfg['mongodb.connection.string']` từ connector.config.
- **R2**: User dán URL có space hoặc thiếu `/` cuối → Debezium reject. Mitigation: chỉ trim, không tự sửa; báo lỗi từ Connect REST.
