# Tasks — MongoDB Connection Form → URL

- [x] FE: thêm `connectionUrl` vào `ConnectionFormValues`.
- [x] FE: refactor `buildConnectorConfig` mongo nhánh dùng `values.connectionUrl`.
- [x] FE: xóa `buildMongoConnectionString` (dead code).
- [x] FE: refactor `parseConnectionSeed` mongo nhánh → seed `connectionUrl` từ `source.server_address` + fallback `connector.config['mongodb.connection.string']`.
- [x] FE: `openCreate` mongo default URL.
- [x] FE: `openEdit` mongo set URL.
- [x] FE: UI render conditional — mongodb hiện URL + Collections; ẩn Host/Port/Username/Password/ReplicaSet.
- [x] FE: validation `connectionUrl` regex `^mongodb(\+srv)?://`.
- [x] Repo: `splitHostPort` URI-prefix branch.
- [x] FE build PASS (1s).
- [x] API build + vet + test PASS (persistence 0.849s, api cached).
- [x] Append `05_progress.md`.
- [x] Tạo `09_tasks_solution_mongo_url_form.md`.
