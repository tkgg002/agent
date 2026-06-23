# 03_implementation_handler.md — internal/handler/

## Tổng quan
Layer handler nhận NATS messages và Kafka events, điều phối sang services.

---

## action_trace.go (996 bytes)
- Theo dõi trace action cho CDC pipeline

## batch_buffer.go (17,984 bytes)
- `BatchBuffer` — buffer trung gian gom events trước khi flush vào DB shadow

## command_handler.go (124,742 bytes — 3437 dòng) ⚠️ FILE LỚN NHẤT
**Struct chính**: `CommandHandler`

| Function | Mô tả |
|---|---|
| `NewCommandHandler(...)` | Khởi tạo handler với db, nats, services |
| `SetKafkaConnectURL(url)` | Inject Kafka Connect URL |
| `SetNATSConn(conn)` | Inject NATS connection |
| `SetMetadataRegistry(metadata)` | Inject metadata registry |
| `SetMongoService(svc)` | Inject MongoDB introspection service |
| `SetMongoURL(url)` | Set MongoDB URL |
| `SetTransformChunkSize(n)` | Set chunk size cho transform |
| `SetConnectionOverrides(overrides)` | Set connection override map |
| `HandleStandardize(msg)` | NATS: chuẩn hóa schema sau discovery |
| `HandleCreateDefaultColumns(msg)` | NATS: tạo default columns cho shadow table |
| `HandleDiscover(msg)` | NATS: scan fields từ source |
| `HandleBackfill(msg)` | NATS: backfill data từ source sang shadow |
| `HandleMasterSwap(msg)` | NATS: swap shadow → master |
| `HandleDiscoverMongoDatabases(msg)` | NATS: liệt kê databases MongoDB |
| `HandleDiscoverMongoCollections(msg)` | NATS: liệt kê collections MongoDB |
| `HandleBatchTransform(msg)` | NATS: transform batch records |
| `HandleScanRawData(msg)` | NATS: scan raw JSON data từ source |
| `HandleScanArrayFields(msg)` | NATS: scan array/nested fields |
| `HandlePeriodicScan(msg)` | NATS: periodic schema scan |
| `HandleDropGINIndex(msg)` | NATS: drop GIN index |
| `HandleScanFields(msg)` | NATS: scan fields từ Debezium schema |
| `HandleSyncRegister(msg)` | NATS: đồng bộ registry state |
| `HandleSyncState(msg)` | NATS: sync connector state |
| `HandleRestartDebezium(msg)` | NATS: restart Debezium connector |
| `HandleAlterColumn(msg)` | NATS: ALTER COLUMN trên shadow table |
| `ensureCDCColumns(tableName)` | Đảm bảo có columns CDC (_id, _gpay_id, ...) |
| `ensureCDCColumnsInSchema(schema, table)` | Như trên nhưng trong schema cụ thể |
| `scanFieldsMongoSource(...)` | Scan fields từ MongoDB source |
| `scanFieldsDebezium(...)` | Scan fields từ Debezium schema registry |
| `processDiscoveryRows(...)` | Xử lý rows sau discovery |
| `bridgeMappingRulesToV2(...)` | Migrate mapping rules sang V2 |
| `detectPrimaryKey(...)` | Detect PK của table trong DB |
| `publishResult(msg, result)` | Publish kết quả về NATS |
| `publishResultWithSubject(...)` | Publish về subject cụ thể |
| `writeActivity(...)` | Ghi activity log |
| `isSafeIdent(s)` | Validate SQL identifier |
| `isSafeType(t)` | Validate SQL type whitelist |

## consumer_pool.go (3,990 bytes)
- `ConsumerPool` — pool của Kafka consumers, manage goroutines

## dlq_circuit_breaker.go (1,902 bytes)
- Circuit breaker cho DLQ processing

## dlq_handler.go (13,047 bytes)
- `DLQHandler` — xử lý Dead Letter Queue messages

## dlq_state_machine.go (14,297 bytes)
- State machine cho DLQ: `pending` → `processing` → `success`/`failed`

## event_bridge.go (8,392 bytes)
- Bridge events giữa Kafka và NATS

## event_handler.go (14,547 bytes)
- `EventHandler` — xử lý CDC events từ Kafka, routing theo op (c/u/d)

## kafka_consumer.go (53,547 bytes)
- `KafkaConsumer` — core Kafka consumer
- `FilterMatchingTopicsForTest(...)` — filter topics theo prefix
- `ExtractDLQMetadataForTest(...)` — extract DLQ metadata
- Adaptive batching: tự điều chỉnh batch size theo lag

## master_ddl_handler.go (6,808 bytes — 183 dòng)
- `MasterDDLHandler` — xử lý DDL cho master table
- `HandleMasterAlterColumn(msg)` — ALTER COLUMN trên master
- `HandleMasterCreate(msg)` — CREATE TABLE master

## provisioning_emit.go (3,298 bytes)
- `emitStepCompleted(...)` — emit event khi provisioning step hoàn thành

## provisioning_handler.go (2,057 bytes)
- `ProvisioningHandler` — nhận events từ provisioning orchestrator
- `HandleStepCompleted(msg)` — xử lý step completion event

## provisioning_step_handlers.go (28,472 bytes — 837 dòng)
- `ProvisioningStepHandler` — xử lý từng step của provisioning wizard
- `HandleShadowBind(msg)` — bind source → shadow DB
- `HandleScheduleEnable(msg)` — enable sync schedule
- `inferSourceColumns(...)` — tự động infer columns từ source
- `inferPGCols(...)` — infer columns từ PostgreSQL
- `inferMySQLCols(...)` — infer columns từ MySQL
- `inferMongoCols(...)` — infer columns từ MongoDB BSON
- `pgSafeType(dataType)` — convert type sang PG-safe
- `mysqlToPGType(dataType)` — convert MySQL → PG type
- `bsonToPGType(v)` — convert BSON → PG type

## recon_handler.go (29,242 bytes — 836 dòng)
- `ReconHandler` — xử lý reconciliation
- `HandleReconCheck(msg)` — check data integrity source vs shadow
- `HandleReconHeal(msg)` — heal mismatches
- `HandleRetryFailed(msg)` — retry failed sync records
- `HandleDebeziumSignal(msg)` — xử lý Debezium signals (snapshot, resume)
- `HandleBackfillSourceTs(msg)` — backfill timestamp fields
- `HandleDetectTimestampField(msg)` — detect timestamp column

## recon_heal_v4.go (12,405 bytes — 328 dòng)
- Extension của ReconHandler cho Segment B (V4 healing)
- `healSegmentA(...)` — heal via snapshot re-trigger
- `healSegmentB(...)` — heal via targeted re-insert
- `healThresholdBlocked(...)` — block heal nếu vượt threshold

## snapshot_runner_handler.go (36,246 bytes — 978 dòng)
- `SnapshotRunner` — chạy snapshot MongoDB → shadow table
- `Handle(msg)` — entry point xử lý snapshot request
- `runSnapshot(ctx, payload, jobID)` — core snapshot loop
- `claimProgress(...)` — claim/resume snapshot progress
- `checkpoint(...)` — checkpoint cursor position
- `markProgressDone(...)` — mark snapshot complete (có guard 99% threshold)
- `markProgressError(...)` — mark snapshot failed
- `buildResumeFilter(lastSeen)` — build MongoDB filter để resume

## transmute_handler.go (10,389 bytes — 283 dòng)
- `TransmuteHandler` — xử lý transmute (shadow → master transform)
- `HandleTransmuteShadow(msg)` — trigger transmute cho shadow binding
- `HandleTransmute(msg)` — trigger transmute theo schedule
- `publishCompleted(...)` — notify completion
