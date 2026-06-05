# 02_plan — Debezium Signal Kafka Migration

## Step-by-step

### Pha A — Service layer (đã hoàn thành phase trước)
1. Rewrite `internal/service/debezium_signal.go`: thay `mongo.Client.InsertOne` bằng `kafka.Writer.WriteMessages`.
2. Cập nhật `config/config.go::DebeziumConfig`: thay `SignalDatabase/SignalCollection` bằng `SignalKafkaTopic`. Thêm env `CDS_DEBEZIUM_SIGNAL_KAFKA_TOPIC`.
3. Cập nhật `config-local.yml` + `config-production.yml`: thay `signalCollection: debezium_signal` bằng `signalKafkaTopic: cdc.signal.commands`.

### Pha B — Handler (PR này)
4. `internal/handler/recon_handler.go`:
   - Bỏ field `mongoClient` khỏi struct `ReconHandler`.
   - Bỏ field `connectionOverrides` + method `WithConnectionOverrides`.
   - Đổi chữ ký `NewReconHandler` từ 5 → 4 tham số (bỏ `mongoClient *mongo.Client`).
   - Trong `HandleDebeziumSignal`: bỏ switch 3 nhánh; chỉ giữ Kafka signal path. Nếu `signal` nil hoặc chưa configured → trả lỗi và log, **không** fallback ghi source.
   - Xoá helper `insertDebeziumSignal`.
   - Xoá `resolveSourceMongoDSN` (dead sau khi xoá fallback).
   - Xoá imports `bson`, `mongo`, `options`.

5. `internal/handler/recon_handler_integration_test.go`:
   - Cập nhật call `NewReconHandler(nil, db, nil, schema, logger)` → `NewReconHandler(nil, db, schema, logger)`.

6. `internal/server/worker_server.go`:
   - Bỏ block tạo `mongoClientForRecon` (chỉ phục vụ ReconHandler cũ).
   - Bỏ `.WithConnectionOverrides(connectionOverrides)` trên reconHandler chain.
   - Cập nhật call `handler.NewReconHandler(reconCore, db, mongoClientForRecon, schemaAdapter, logger)` → `handler.NewReconHandler(reconCore, db, schemaAdapter, logger)`.

### Pha C — Connector configs (PR này)
7. `deployments/debezium/mongodb-connector.json`:
   - Bỏ `centralized-export-service.debezium_signal` khỏi `collection.include.list`.
   - Bỏ key `signal.data.collection`.
   - Thêm `signal.enabled.channels`, `signal.kafka.topic`, `signal.kafka.bootstrap.servers`.

8. `cdc-cms-web/src/pages/SourceConnectors.tsx::buildConnectorConfig`:
   - 3 nhánh mongo/mysql/pg đã có sẵn 3 key Kafka signal — verify.

### Pha D — Verify
9. `go build ./...`
10. `go vet ./...`
11. `npx tsc -b` (CMS web) — chỉ check SourceConnectors.tsx, lỗi TableRegistry.tsx out of scope.

## Rationale
- **Tại sao chỉ `kafka` mà không `source,kafka`?** `source` channel khiến Debezium scan signal collection trong source DB — đúng cái ta muốn loại bỏ. `kafka` only đảm bảo source DB không bị chạm.
- **Tại sao reject thay vì fallback?** Fallback nghĩa là ghi source. Nguyên tắc đã rõ: source là read-only. Reject + log → operator phải fix config Kafka trước khi snapshot.
- **Tại sao bỏ `connectionOverrides` ở ReconHandler nhưng giữ ở `CommandHandler` + `MetadataRegistryService`?** Override URI cho connection_registry vẫn cần ở các nơi khác (dynamic source). Chỉ ReconHandler không còn cần vì đã xoá `resolveSourceMongoDSN`.

## Risk
- **Breaking change**: thay đổi chữ ký `NewReconHandler` public. Đã grep toàn repo — chỉ 2 call sites (`worker_server.go` + integration test). Đã cập nhật cả 2.
- **Operator burden**: production cluster phải có Kafka brokers wired vào worker config. Nếu thiếu, "Snapshot Now" sẽ báo lỗi rõ ràng ngay.
