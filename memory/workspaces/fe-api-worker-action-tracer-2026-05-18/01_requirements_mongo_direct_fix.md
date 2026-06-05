# Requirements — Fix MongoDB Direct Connection Bug

## Origin
User log (3 lỗi runtime liền nhau sau khi đã chuyển sang URL form):

1. **Quét field**: `failed to introspect mongo source: a direct connection cannot be made if multiple hosts are specified`
2. **create-default-columns**: success, `rows_affected=19`, "nhung ko có field" (không có field mới được thêm vào shadow).
3. **Snapshot Now / scheduler**: `reconcile ALL skipped - reconCore not initialized (MongoDB not configured)`.

## Phân tích Root Cause

### Bug #1 — Mongo driver SetDirect(true) vs replicaSet URI
- File: `centralized-data-service/internal/service/mongo_introspection.go` — 3 call sites (line 23, 41, 60).
- Code: `mongo.Connect(ctx, options.Client().ApplyURI(uri).SetDirect(true))`.
- URI người dùng (đã đúng): `mongodb://host1:27017,host2:27017/?replicaSet=rs0` (hoặc `mongodb+srv://`).
- MongoDB Go driver: khi `SetDirect(true)` + URI có nhiều host hoặc có `?replicaSet=` → reject ngay tại Connect/Validate với error trên.
- Đối chiếu pattern đúng (worker prod): `pkgs/mongodb/client.go:20` chỉ dùng `options.Client().ApplyURI(cfg.URL)` — driver tự auto-detect từ URI.

### Bug #2 — Hệ quả của Bug #1
- `HandleCreateDefaultColumns` → `scanFieldsDebezium` → `scanFieldsMongoSource` → `IntrospectCollection` ERR (do Bug #1).
- Err được swallow tại `command_handler.go:501` (`Warn + continue with existing rules`) — đúng pattern, không phải bug.
- Hậu quả: ALTER block chạy 19 rule cũ thành công (cdc_mapping_rule_v2 đã có sẵn từ session trước), nhưng KHÔNG có field mới được scan từ source → user thấy "0 new field".

### Bug #3 — reconCore=nil khi cfg.MongoDB.URL rỗng
- File: `centralized-data-service/internal/server/worker_server.go:164`.
- Code: `if cfg.MongoDB.URL != "" { ... init reconCore ... }`.
- Nếu `config-local.yml` không có `mongodb:` block → `reconCore=nil` → 7 stub subscriber log error khi user click Snapshot Now, scheduler log "reconcile ALL skipped".
- Đây không phải code bug — là config requirement chưa được docs hóa.

## Functional Requirements

- FR-1: Fix `mongo_introspection.go` — remove `.SetDirect(true)` từ cả 3 method (`DiscoverDatabases`, `DiscoverCollections`, `IntrospectCollection`). Driver tự auto-detect: single host → direct mode; multi-host hoặc replicaSet → replica set mode.
- FR-2: Build/vet/test worker PASS không break test hiện có.
- FR-3: Docs hóa requirement `cfg.MongoDB.URL` cho Snapshot Now (cập nhật `05_progress.md` + workspace docs).

## Out of Scope

- Refactor `worker_server.go` reconCore init thành lazy-resolve qua `connection_registry` (như `scanFieldsMongoSource` đã làm). Đây là surface lớn (ReconCore, ReconSourceAgent, ReconDestAgent, ReconHealer, FullCountAggregator, BackfillSourceTsService, TimestampDetector đều tham chiếu mongoClient ngay tại boot). Để future task.
- Sửa FE — URI người dùng nhập đã đúng.
- Migration DB — `cdc_system.connection_registry.host` vẫn là VARCHAR chứa full URI.

## Definition of Done

- [ ] `internal/service/mongo_introspection.go` không còn `.SetDirect(true)`.
- [ ] `go build ./...` PASS.
- [ ] `go vet ./...` PASS.
- [ ] `go test ./internal/service/... ./internal/handler/...` PASS.
- [ ] APPEND `05_progress.md`.
- [ ] Tạo `02_plan`, `08_tasks`, `09_tasks_solution` đầy đủ.
- [ ] User click thử "Quét field" → log expected: `processDiscoveryRows summary discovered_total=N inserted=M`.
- [ ] User click thử "Snapshot Now" với `mongodb.url` config-local.yml đã set → log expected: `debezium signal: using SignalClient path` (hoặc `mongo_direct_insert`).
