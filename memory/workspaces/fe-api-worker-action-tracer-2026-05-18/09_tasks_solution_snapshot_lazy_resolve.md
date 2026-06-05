# Solution — Snapshot Now lazy resolve từ connection_registry

## Bản chất bug

Worker hard-code Mongo URL từ `config-local.yml` → ép user phải config tĩnh trong khi source mongo là dynamic per-connection (CMS UI khai báo nhiều cluster qua `connection_registry`). User: "ko thể bỏ vào env".

Bệnh: `worker_server.go:164` gate `reconCore` trên `cfg.MongoDB.URL != ""`. Khi rỗng → 7 stub subject log error.

## Fix

3 nhánh dispatch trong `HandleDebeziumSignal`, ưu tiên signal_client → shared mongoClient → **lazy resolve từ connection_registry**:

### `internal/handler/recon_handler.go` — HandleDebeziumSignal

```diff
-	signalConfigured := h.signal != nil && h.signal.IsConfigured()
-	if signalConfigured {
-		dispatchPath = "signal_client"
-		signalID, err = h.signal.TriggerIncrementalSnapshot(...)
-	} else {
-		dispatchPath = "mongo_direct_insert"
-		if h.mongoClient == nil {
-			h.logger.Warn("action trace failed", ..., "mongodb client not configured")
-			return
-		}
-		signalID, err = ...InsertOne(signalDoc)
-	}
+	signalConfigured := h.signal != nil && h.signal.IsConfigured()
+	switch {
+	case signalConfigured:
+		dispatchPath = "signal_client"
+		signalID, err = h.signal.TriggerIncrementalSnapshot(...)
+
+	case h.mongoClient != nil:
+		dispatchPath = "mongo_shared_client"
+		signalID, err = h.insertDebeziumSignal(ctx, h.mongoClient, db, collection)
+
+	default:
+		dispatchPath = "mongo_lazy_resolve"
+		dsn, resolveErr := h.resolveSourceMongoDSN(ctx, payload.Table)
+		if resolveErr != nil {
+			h.logger.Warn("action trace failed", ..., zap.Error(resolveErr))
+			return
+		}
+		client, _ := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
+		defer client.Disconnect(...)
+		signalID, err = h.insertDebeziumSignal(ctx, client, db, collection)
+	}
```

### `internal/handler/recon_handler.go` — Helpers mới

```go
func (h *ReconHandler) insertDebeziumSignal(ctx, client, db, collection) (string, error) {
    doc := bson.M{
        "type": "execute-snapshot",
        "data": bson.M{"data-collections": []string{db+"."+collection}, "type": "incremental"},
    }
    res, err := client.Database(db).Collection("debezium_signal").InsertOne(ctx, doc)
    if err != nil { return "", err }
    return fmt.Sprintf("%v", res.InsertedID), nil
}

func (h *ReconHandler) resolveSourceMongoDSN(ctx, targetTable) (string, error) {
    route := h.metadata.ResolveTargetRoute(targetTable)
    if route == nil || route.SourceObject == nil { return "", ... }
    var conn model.ConnectionRegistry
    h.db.First(&conn, route.SourceObject.SourceConnectionID)
    if strings.HasPrefix(conn.Host, "mongodb://") { return *conn.Host, nil }
    return fmt.Sprintf("mongodb://%s:%d/", *conn.Host, *conn.Port), nil
}
```

### `internal/server/worker_server.go` — Decouple Signal/Snapshot khỏi reconCore

```diff
 } else {
-    logger.Warn("reconciliation handlers NOT registered (MongoDB not configured)")
-    // ... stub fallback 7 subject
+    logger.Warn("reconciliation handlers NOT registered ...; Snapshot Now still served via lazy-resolve from connection_registry")
+    signalOnlyHandler := handler.NewReconHandler(nil, db, nil, schemaAdapter, logger).
+        WithMetadataRegistry(registrySvc).
+        WithMaskingService(maskingSvc)
+    natsClient.Conn.Subscribe("cdc.cmd.debezium-signal", signalOnlyHandler.HandleDebeziumSignal)
+    natsClient.Conn.Subscribe("cdc.cmd.debezium-snapshot", signalOnlyHandler.HandleDebeziumSignal)
+    // stub fallback cho 5 subject còn lại (recon-check/heal/retry/backfill/detect)
 }
```

### `config/config-local.yml`

```diff
-mongodb:
-  url: "mongodb://localhost:17017/?directConnection=true"
```

## Verify

- `go build ./...` EXIT=0.
- `go vet ./...` EXIT=0.
- `go test -count=1 ./internal/handler/... ./internal/server/...` PASS (handler 3.972s).

## Bước user cần làm

1. Ctrl-C worker tty003 (PID cũ 13779) → `go run cmd/worker/main.go`.
2. Boot log expected: `debezium signal/snapshot handlers registered (lazy resolve mode)`.
3. Click "Snapshot Now" cho `export-jobs` (connection_registry id=2 đã update DB ở turn trước thành `mongodb://localhost:17017/?directConnection=true`).
4. Worker stdout expected:
   - `debezium signal received trace_id=fe-snapshot_now-<uuid> table=sd_export_jobs ...`
   - `debezium signal: lazy resolve from connection_registry table=sd_export_jobs database=centralized-export-service collection=export-jobs`
   - `debezium signal dispatched dispatch_path=mongo_lazy_resolve signal_id=<ObjectID>`

## Grep cheatsheet

```bash
grep "lazy resolve mode" worker.log              # boot OK
grep "debezium signal: lazy resolve" worker.log  # mỗi click Snapshot Now
grep "dispatch_path=mongo_lazy_resolve" worker.log  # success log
grep "cannot resolve source mongo DSN" worker.log   # URI không có trong connection_registry
```

## Lý do thiết kế

- KHÔNG cache lazy client: source URI có thể thay đổi qua CMS giữa 2 click. Mỗi click reload URI từ DB.
- Overhead `mongo.Connect` mỗi click chấp nhận được — Snapshot Now là user-triggered, frequency thấp.
- Vẫn giữ branch `mongo_shared_client` nếu có ngày cần ép config (e.g., production Debezium signal-only mongo cluster riêng).

## Note kiến trúc (cho future)

5 subject còn lại (recon-check/heal/retry/backfill/detect) vẫn cần `reconCore` lazy init từ `connection_registry` để Reconciliation hoạt động khi user không config `cfg.MongoDB.URL`. Surface bao gồm ReconCore + ReconSourceAgent + ReconDestAgent + ReconHealer + FullCountAggregator + BackfillSourceTsService + TimestampDetector — đều bind 1 mongoClient ở boot. Refactor đề xuất: introduce `MongoClientProvider` interface có method `ClientForTarget(targetTable) (*mongo.Client, error)` resolve động qua `connection_registry`, các service nhận provider thay vì client. Out of scope cho task này.
