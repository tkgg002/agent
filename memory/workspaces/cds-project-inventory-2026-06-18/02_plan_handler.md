# 02_plan_handler.md — Phase 4: Tổ chức lại `internal/handler/`

## Hiện trạng: 14 production files + command_handler.go (77 funcs, 3437 dòng)

---

## `handler/source/` — Debezium connector, sync state

**Files move:**

| File mới | Funcs tách từ |
|---|---|
| `handler/source/sync_handler.go` | `command_handler.go` L.2818-2984 |

**Funcs đưa vào `sync_handler.go`**:
```go
// Struct mới thay thế CommandHandler cho domain này:
type SyncHandler struct {
    db               *gorm.DB
    natsConn         *nats.Conn
    kafkaConnectURL  string
    metadataRegistry service.MetadataRegistry
    logger           *zap.Logger
}

func NewSyncHandler(...) *SyncHandler
func (h *SyncHandler) HandleSyncRegister(msg *nats.Msg)    // L.2818
func (h *SyncHandler) HandleSyncState(msg *nats.Msg)       // L.2877
func (h *SyncHandler) HandleRestartDebezium(msg *nats.Msg) // L.2945
func (h *SyncHandler) verifyDebeziumConnector(ctx)         // L.2864 [private]
func (h *SyncHandler) detectConnectorName(entry)           // L.3169 [private]
func (h *SyncHandler) connectGET/POST/PUT/Call(...)        // L.3177-3221 [private]
```

---

## `handler/shadow/` — NATS/Kafka consumer pipeline

**Files move (giữ nguyên toàn bộ):**

| File cũ | File mới |
|---|---|
| `kafka_consumer.go` (41 funcs) | `handler/shadow/kafka_consumer.go` |
| `event_handler.go` (15 funcs) | `handler/shadow/event_handler.go` |
| `event_bridge.go` (16 funcs) | `handler/shadow/event_bridge.go` |
| `batch_buffer.go` (26 funcs) | `handler/shadow/batch_buffer.go` |
| `consumer_pool.go` (7 funcs) | `handler/shadow/consumer_pool.go` |

---

## `handler/master/` — DDL commands, transform, swap

**Files move + tách từ command_handler.go:**

| File mới | Nguồn |
|---|---|
| `handler/master/schema_ddl_handler.go` | Tách `command_handler.go` L.360-962 |
| `handler/master/batch_transform_handler.go` | Tách `command_handler.go` L.1572-1800 |
| `handler/master/master_ddl_handler.go` | Move `master_ddl_handler.go` (5 funcs) |

**Funcs `schema_ddl_handler.go`**:
```go
type SchemaDDLHandler struct {
    shadowDB         *gorm.DB
    natsConn         *nats.Conn
    metadataRegistry service.MetadataRegistry
    mappingRuleRepo  *repository.MappingRuleV2Repo
    pendingFieldRepo *repository.PendingFieldRepo
    logger           *zap.Logger
}

func NewSchemaDDLHandler(...) *SchemaDDLHandler
func (h *SchemaDDLHandler) HandleStandardize(msg)             // L.360
func (h *SchemaDDLHandler) HandleCreateDefaultColumns(msg)    // L.688
func (h *SchemaDDLHandler) HandleAlterColumn(msg)             // L.2985
func (h *SchemaDDLHandler) HandleDropGINIndex(msg)            // L.2568
// Privates:
func ensureCDCColumns / ensureCDCColumnsInSchema / hasColumn / hasColumnInSchema
func tableExists / tableExistsInSchema
func listShadowColumns / listShadowColumnsWithType
func normalizePGType / isSafeIdent / isSafeType / systemFieldSet
func normalizeMappingRuleDataType / processDiscoveryRows / bridgeMappingRulesToV2
```

**Funcs `batch_transform_handler.go`**:
```go
type BatchTransformHandler struct { ... }
func NewBatchTransformHandler(...) *BatchTransformHandler
func (h *BatchTransformHandler) HandleBatchTransform(msg)  // L.1572
func (h *BatchTransformHandler) HandleMasterSwap(msg)      // L.1301
func (h *BatchTransformHandler) detectPrimaryKey(...)      // L.1794 [private]
```

---

## `handler/recon/` — Reconciliation, backfill, DLQ

**Files move (giữ nguyên):**

| File cũ | File mới |
|---|---|
| `recon_handler.go` (20 funcs) | `handler/recon/recon_handler.go` |
| `recon_heal_v4.go` (9 funcs) | `handler/recon/recon_heal_v4.go` |

---

## `handler/orchestration/` — Provisioning, snapshot, scan, discovery

**Files move + tách:**

| File mới | Nguồn |
|---|---|
| `handler/orchestration/provisioning_handler.go` | Move `provisioning_handler.go` (1 func) |
| `handler/orchestration/provisioning_step_handlers.go` | Move `provisioning_step_handlers.go` (18 funcs) |
| `handler/orchestration/provisioning_emit.go` | Move `provisioning_emit.go` (1 func) |
| `handler/orchestration/snapshot_runner.go` | Move `snapshot_runner_handler.go` (14 funcs) |
| `handler/orchestration/discover_handler.go` | Tách `command_handler.go` L.407-687, L.962-1237 |
| `handler/orchestration/scan_handler.go` | Tách `command_handler.go` L.1817-2540 |
| `handler/orchestration/mongo_discover_handler.go` | Tách `command_handler.go` L.1378-1571 |
| `handler/orchestration/dlq_handler.go` | Move `dlq_handler.go` (21 funcs) |
| `handler/orchestration/dlq_state_machine.go` | Move `dlq_state_machine.go` (11 funcs) |

**Funcs `discover_handler.go`**:
```go
type DiscoverHandler struct { ... }
func NewDiscoverHandler(...) *DiscoverHandler
func (h *DiscoverHandler) HandleDiscover(msg)           // L.962
func (h *DiscoverHandler) HandleScanFields(msg)         // L.2745
func scanFieldsMongoSource(ctx, ...)                    // L.407 [private]
func scanFieldsDebezium(ctx, ...)                       // L.2693 [private]
func findSimilarCollections(target, all)                // L.551 [private]
func inferSQLTypeFromLegacyCatalogProp(prop)            // L.2659 [private]
```

**Funcs `scan_handler.go`**:
```go
type ScanHandler struct { ... }
func NewScanHandler(...) *ScanHandler
func (h *ScanHandler) HandleScanRawData(msg)            // L.1817
func (h *ScanHandler) HandleScanArrayFields(msg)        // L.2018
func (h *ScanHandler) HandlePeriodicScan(msg)           // L.2375
func (h *ScanHandler) HandleBackfill(msg)               // L.1238
func validScanIdent / explodePathToPGPath               // [private]
func flattenJSONWithTypes / buildCastExpr               // [private]
func detectPrimaryKey(execDB, schemaName, tableName)    // [private]
```

**Funcs `mongo_discover_handler.go`**:
```go
type MongoDiscoverHandler struct { ... }
func NewMongoDiscoverHandler(...) *MongoDiscoverHandler
func (h *MongoDiscoverHandler) HandleDiscoverMongoDatabases(msg)    // L.1378
func (h *MongoDiscoverHandler) HandleDiscoverMongoCollections(msg)  // L.1440
func replyMongoDiscovery(msg, replyTo, payload)                     // [private]
```

---

## Shared utilities từ command_handler.go (sau khi tách)

Sau khi tách hết các Handle* và helper chuyên biệt, các funcs sau **chuyển thành package-level utilities**:

| Func | Move sang |
|---|---|
| `quoteCommandIdent(v)` / `quoteCommandQualifiedTable(...)` | `pkgs/sqlutil/` |
| `publishResult(msg, result)` / `publishResultWithSubject(...)` | Mỗi handler tự implement |
| `writeActivity(op, table, ...)` | Inject `*service.ActivityLogger` |
| `resolveTargetTableConfig(ctx, targetTable)` | Inject `service.MetadataRegistry` |
| `sanitizeAdminError(errMsg)` / `sanitizeAdminResultMap(...)` | `internal/admin/sanitize.go` |
| `logCommandResult(result, fields...)` | Tích hợp vào từng handler |

---

## Tổng kết tách command_handler.go (3437 dòng → 0 dòng)

| Handler mới | Funcs nhận | Từ dòng |
|---|---|---|
| `handler/source/sync_handler.go` | 7 funcs | L.2818-3221 |
| `handler/master/schema_ddl_handler.go` | 18 funcs | L.136-1137 |
| `handler/master/batch_transform_handler.go` | 3 funcs | L.1301-1816 |
| `handler/orchestration/discover_handler.go` | 6 funcs | L.407-1237 |
| `handler/orchestration/scan_handler.go` | 9 funcs | L.1817-2540 |
| `handler/orchestration/mongo_discover_handler.go` | 3 funcs | L.1378-1571 |
| Shared utilities → pkgs/ | 8 funcs | L.3070-3435 |
