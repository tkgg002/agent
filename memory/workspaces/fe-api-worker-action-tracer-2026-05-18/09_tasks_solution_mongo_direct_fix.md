# Solution — Fix MongoDB Direct Connection Bug

## Bản chất bug

`MongoIntrospectionService` ép `SetDirect(true)` cho mọi URI. MongoDB Go driver reject URI có `?replicaSet=` hoặc nhiều host khi SetDirect=true:
```
a direct connection cannot be made if multiple hosts are specified
```

Pattern đúng (đã có trong cùng worker): `pkgs/mongodb/client.go:20` chỉ `options.Client().ApplyURI(cfg.URL)` — driver tự auto-detect:
- URI `mongodb://host:27017/` (single host, no replicaSet) → direct mode.
- URI `mongodb://h1,h2/?replicaSet=rs0` hoặc `mongodb+srv://...` → replica-set mode.

## Diff

### `internal/service/mongo_introspection.go`

```diff
 func (s *MongoIntrospectionService) DiscoverDatabases(uri string) ([]string, error) {
     ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
     defer cancel()
-    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetDirect(true))
+    // Driver auto-detects topology from URI: single host → direct mode;
+    // multi-host / ?replicaSet= → replica-set mode. SetDirect(true) forces
+    // direct and conflicts with replicaSet URIs ("direct connection cannot
+    // be made if multiple hosts are specified").
+    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
     ...
 }

 func (s *MongoIntrospectionService) DiscoverCollections(uri, dbName string) ([]string, error) {
     ...
-    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetDirect(true))
+    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
     ...
 }

 func (s *MongoIntrospectionService) IntrospectCollection(uri string, dbName, collectionName string, sampleSize int) (map[string]interface{}, error) {
     ...
-    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetDirect(true))
+    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
     ...
 }
```

## Verify

- `go build ./...` EXIT=0.
- `go vet ./...` EXIT=0.
- `go test -count=1 ./internal/service/... ./internal/handler/...` PASS (service 0.893s, handler 4.378s).

## Tác động lên 3 lỗi user báo

1. **Quét field "direct connection cannot be made"** → khỏi. URI `mongodb://...?replicaSet=rs0` giờ chạy thẳng.
2. **create-default-columns rows_affected=19 ko có field mới** → khỏi. Auto-discovery (`scanFieldsMongoSource → IntrospectCollection`) giờ trả về fieldMap, `processDiscoveryRows` insert rule mới → ALTER block lần kế tiếp sẽ có columns_added > 0.
3. **Snapshot Now `reconCore not initialized`** → CHƯA fix code. Đây là config: user phải thêm vào `centralized-data-service/config-local.yml`:
   ```yaml
   mongodb:
     url: "mongodb://host:27017/?replicaSet=rs0"
   ```
   Khi worker reboot với `cfg.MongoDB.URL != ""` → `reconCore` init thành công → 7 NATS subject (debezium-snapshot, debezium-signal, recon-check, recon-heal, retry-failed, recon-backfill-source-ts, detect-timestamp-field) sẽ bound vào handler thật thay vì stub. Refactor reconCore lazy-init từ `connection_registry` là future task — surface lớn (ReconCore + 5 dependent services).

## Grep cheatsheet

```bash
# Verify fix sau khi user click Quét field
grep "failed to introspect mongo source" worker.log     # phải KHÔNG còn xuất hiện
grep "processDiscoveryRows summary" worker.log          # phải thấy discovered_total > 0
grep "ALTER TABLE summary" worker.log                   # columns_added > 0 ở lần Sync tiếp theo

# Verify Snapshot Now (sau khi user add mongodb.url config)
grep "Reconciliation Core initialized" worker.log       # boot OK
grep "reconciliation handlers registered" worker.log    # 7 subject registered
grep "debezium signal: using" worker.log                # dispatch path khi click Snapshot Now
```

## Bước tiếp theo

- User restart worker (`Ctrl-C` tty003 → `go run cmd/worker/main.go`) để pickup binary mới.
- User click thử "Quét field" → expected log `processDiscoveryRows summary inserted=N`.
- (Optional cho Snapshot Now) User add `mongodb.url` vào `config-local.yml`, restart worker.

## Note kiến trúc (cho future)

`worker_server.go:164` còn 1 architectural smell: reconCore + 5 dependent services (ReconHealer, FullCountAggregator, BackfillSourceTsService, TimestampDetector, ReconHandler) cùng share `mongoClientShared` được lazy-init từ config TĨNH. Không tận dụng được `cdc_system.connection_registry` — nơi CMS đã store URI per-source. Nếu mỗi source dùng Mongo cluster khác nhau, hiện tại reconCore chỉ kết nối được vào 1 cluster duy nhất từ config-local.yml.

Refactor đề xuất (out of scope task này):
- Move reconCore từ "init once tại boot" thành "init per-source khi cần", resolve URI từ `connection_registry` table.
- Hoặc giữ shared client nhưng cho phép multi-cluster pool (map[clusterID]*mongo.Client).
