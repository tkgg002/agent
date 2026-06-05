# 03_implementation — Debezium Signal Kafka Migration

## Files changed (Pha B + C)

### Backend (Go)

| File | Loại thay đổi |
|---|---|
| `internal/handler/recon_handler.go` | Bỏ field `mongoClient`, `connectionOverrides`; bỏ method `WithConnectionOverrides`; đổi chữ ký `NewReconHandler` (5→4 args); rewrite `HandleDebeziumSignal` (chỉ Kafka path + reject khi unconfigured); xoá helper `insertDebeziumSignal` + `resolveSourceMongoDSN`; xoá imports `bson`, `mongo`, `options`. |
| `internal/handler/recon_handler_integration_test.go` | `NewReconHandler(nil, db, nil, schema, logger)` → `NewReconHandler(nil, db, schema, logger)`. |
| `internal/server/worker_server.go` | Bỏ block tạo `mongoClientForRecon`; bỏ `.WithConnectionOverrides(connectionOverrides)`; cập nhật call `NewReconHandler`. |

### Connector configs

| File | Loại thay đổi |
|---|---|
| `deployments/debezium/mongodb-connector.json` | Bỏ `centralized-export-service.debezium_signal` khỏi `collection.include.list`; bỏ `signal.data.collection`; thêm `signal.enabled.channels=kafka`, `signal.kafka.topic=cdc.signal.commands`, `signal.kafka.bootstrap.servers=gpay-kafka:9092`. |
| `cdc-cms-web/src/pages/SourceConnectors.tsx` | (Đã có sẵn 3 key Kafka signal ở cả 3 nhánh mongo/mysql/pg — chỉ verify; không edit thêm). |

## Diff summary (`HandleDebeziumSignal`)

Trước:
```go
switch {
case signalConfigured:
    signalID, err = h.signal.TriggerIncrementalSnapshot(ctx, db, collection, payload.Filter)
case h.mongoClient != nil:
    signalID, err = h.insertDebeziumSignal(ctx, h.mongoClient, db, collection)  // writes source
default:
    // lazy connect → insertDebeziumSignal  → writes source
}
```

Sau:
```go
if h.signal == nil || !h.signal.IsConfigured() {
    err := fmt.Errorf("debezium signal client not configured ...; refusing to write to source DB")
    h.logActivity(..., "error", 0, err)
    return
}
signalID, err := h.signal.TriggerIncrementalSnapshot(ctx, db, collection, payload.Filter)
```

## Verify (Pha D)

```
$ go build ./...            # clean
$ go vet ./...              # clean
$ npx tsc --noEmit ...      # SourceConnectors.tsx clean (TableRegistry.tsx errors pre-existing)
```

## Files **không** đụng (ngoài scope)

- `internal/service/debezium_signal.go` — đã rewrite ở phase trước, giữ nguyên.
- `internal/service/recon_heal.go::HealWindow` — caller dùng đúng chữ ký 4-arg sẵn.
- `cdc-cms-web/src/pages/SourceConnectors.tsx::buildConnectorConfig` — đã được wire sẵn 3 key signal.
- `cdc-cms-web/src/pages/TableRegistry.tsx` — TS lint errors pre-existing, không phải scope migration này.
