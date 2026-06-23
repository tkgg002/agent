# Phase 5: Domain `transmute`

## Mục tiêu
`internal/transmute/` — Shadow → Master materialisation.

---

## Bước 5.1: `internal/transmute/model.go`

| Struct | Table |
|---|---|
| `MasterBinding` | `cdc_system.master_binding` |
| `TransmuteSchedule` | `cdc_system.transmute_schedule` |
| `SyncRuntimeState` | `cdc_system.sync_runtime_state` |
| `WorkerSchedule` | `cdc_system.cdc_worker_schedule` |

---

## Bước 5.2: `internal/transmute/repository.go` (Port — MỚI)

```go
package transmute

type MasterBindingRepository interface {
    GetByID(ctx, id int64) (*MasterBinding, error)
    GetByCode(ctx, code string) (*MasterBinding, error)
    GetByMasterTable(ctx, masterTable string) (*MasterBinding, error)
    ListBySourceObject(ctx, sourceObjectID int64) ([]MasterBinding, error)
    ListActiveBySourceObject(ctx, sourceObjectID int64) ([]MasterBinding, error)
    ListActiveByShadowBinding(ctx, shadowBindingID int64) ([]MasterBinding, error)
    Create(ctx, item *MasterBinding) error
    Update(ctx, item *MasterBinding) error
}

type TransmuteScheduleRepository interface {
    GetPending(ctx) ([]TransmuteSchedule, error)
    MarkDone(ctx, id int64) error
}

type SyncRuntimeStateRepository interface {
    Get(ctx, masterBindingID int64) (*SyncRuntimeState, error)
    Upsert(ctx, state *SyncRuntimeState) error
    MarkDone(ctx, masterBindingID int64) error
}
```

---

## Bước 5.3: Move GORM Repos → `internal/transmute/repository/`

| File cũ | File mới |
|---|---|
| `internal/repository/master_binding_repo.go` | `internal/transmute/repository/gorm_master_binding_repo.go` |
| `internal/repository/transmute_schedule_repo.go` | `internal/transmute/repository/gorm_transmute_schedule_repo.go` |
| `internal/repository/sync_runtime_state_repo.go` | `internal/transmute/repository/gorm_sync_runtime_state_repo.go` |

**Functions `master_binding_repo.go`** (9 funcs) — giữ nguyên logic, update import:
- `NewMasterBindingRepo(db)`
- `GetByID(ctx, id)`
- `GetByCode(ctx, code)`
- `GetByMasterTable(ctx, masterTable)`
- `ListBySourceObject(ctx, sourceObjectID)`
- `ListActiveBySourceObject(ctx, sourceObjectID)`
- `ListActiveByShadowBinding(ctx, shadowBindingID)`
- `Create(ctx, item)`
- `Update(ctx, item)`

---

## Bước 5.4: Move Services → `internal/transmute/service/`

| File cũ | File mới |
|---|---|
| `internal/service/transmuter.go` | `internal/transmute/service/transmuter.go` |
| `internal/service/transmute_scheduler.go` | `internal/transmute/service/transmute_scheduler.go` |
| `internal/service/job_monitor.go` | `internal/transmute/service/job_monitor.go` |
| `internal/service/child_explode_master.go` | `internal/transmute/service/child_explode_master.go` |

**Key functions trong `transmuter.go`** (28 funcs — file 899L):

| Func | Hành động |
|---|---|
| `NewTransmuterModule(...)` | Move |
| `Run(ctx, masterName, onlySourceIDs)` | Move |
| `loadMaster(ctx, name)` | Move (private) |
| `shadowActive(row)` | Move (private) |
| `loadRules(ctx, row)` | Move (private) |
| `InvalidateRuleCache(bindingID, masterTable)` | Move |
| `fetchShadowBatch(ctx, row, cursor, onlyIDs)` | Move (private) |
| `processBatch(ctx, binding, rules, rows)` | Move (private) |
| `toTransmuteRules(rules)` | Move (private) |
| `extractColumnsFn(ctx)` | Move (private) |
| `extractColumns(ctx, raw, rules)` | Move (private) |
| `upsertMaster(ctx, binding, record)` | Move (private) |
| `gjsonValueToGo(r)` | Move (private) |
| `unwrapMongoExtJSON(v)` | Move (private) |
| `mongoNumberToGo(n)` | Move (private) |
| `isJSONColumnType(dataType)` | Move (private) |
| `coerceForColumn(v, dataType)` | Move (private) |
| `isTimestampColumnType(dataType)` | Move (private) |
| `epochToTime(n)` | Move (private) |
| `deterministicGpayID(shadowGpayID, keySuffix)` | Move (private) |
| `sqlBindValueTransmute(v)` | Move (private) |
| `quoteTransmuteIdent/Qualified(...)` | Move (private) |
| `markRuntimeSuccess/Failure/Skipped(...)` | Move (private) |
| `persistRuntimeState(ctx, masterBindingID, mutate)` | Move (private) |

---

## Bước 5.5: Move Handlers → `internal/transmute/handler/`

**Toàn bộ** `internal/handler/transmute_handler.go` → `internal/transmute/handler/transmute_handler.go`:

| Func | Hành động |
|---|---|
| `NewTransmuteHandler(svc, db, conn, logger, activity)` | Move |
| `HandleTransmuteShadow(msg)` | Move |
| `HandleTransmute(msg)` | Move |
| `publishCompleted(req, res, runErr)` | Move (private) |
| `reply(msg, resp)` | Move (private) |
| `replyErr(msg, correlationID, errMsg)` | Move (private) |

**Tách từ `command_handler.go`** → `internal/transmute/handler/batch_transform_handler.go`:

| Func | Từ dòng |
|---|---|
| `HandleBatchTransform(msg)` | L.1572 |
| `HandleMasterSwap(msg)` | L.1301 |

---

## Bước 5.6: Compile Check

```bash
go build ./internal/transmute/...
go test ./internal/transmute/...
```


---

# Phase 6: Domain `recon`

## Mục tiêu
`internal/recon/` — Drift detection + healing engine.

---

## Bước 6.1: `internal/recon/model.go`

| Struct | Table |
|---|---|
| `ReconciliationReport` | `cdc_system.cdc_reconciliation_report` |
| `FailedSyncLog` | `cdc_system.failed_sync_logs` |

---

## Bước 6.2: `internal/recon/repository.go` (Port — MỚI)

```go
package recon

// ⚠️ Cả 2 repository này CẦN TẠO MỚI — hiện tại write inline qua GORM

type ReconciliationReportRepository interface {
    Create(ctx, report *ReconciliationReport) error
    GetByTable(ctx, targetTable string, limit int) ([]ReconciliationReport, error)
    GetLatest(ctx, targetTable string) (*ReconciliationReport, error)
}

type FailedSyncLogRepository interface {
    GetPendingByTable(ctx, tableName string, limit int) ([]FailedSyncLog, error)
    GetByID(ctx, id uint64) (*FailedSyncLog, error)
    Update(ctx, log *FailedSyncLog) error
    Create(ctx, log *FailedSyncLog) error
    CountPending(ctx, tableName string) (int64, error)
}
```

---

## Bước 6.3: Tạo mới GORM Repos

```
internal/recon/repository/
├── gorm_reconciliation_report_repo.go  ← TẠO MỚI
└── gorm_failed_sync_log_repo.go        ← TẠO MỚI
```

Extract inline GORM queries từ `recon_core.go` và `batch_buffer.go` vào repos này.

---

## Bước 6.4: Move Services → `internal/recon/service/`

| File cũ | File mới |
|---|---|
| `internal/service/recon_core.go` | `internal/recon/service/recon_core.go` → tách thêm |
| `internal/service/recon_source_agent.go` | `internal/recon/service/recon_source_agent.go` |
| `internal/service/recon_dest_agent.go` | `internal/recon/service/recon_dest_agent.go` |
| `internal/service/recon_heal.go` | `internal/recon/service/recon_heal.go` |
| `internal/service/recon_alert.go` | `internal/recon/service/recon_alert.go` |
| `internal/service/full_count_aggregator.go` | `internal/recon/service/full_count_aggregator.go` |
| `internal/service/wal_monitor.go` | `internal/recon/service/wal_monitor.go` |
| `internal/service/debezium_signal.go` | `internal/recon/service/debezium_signal.go` |

**Tách `recon_core.go` (1900L, 50 funcs) thành 3 files**:

### `internal/recon/service/recon_tier_a.go` — Source ↔ Shadow

| Func | Từ dòng |
|---|---|
| `RunTier1(ctx, entry)` | L.587 |
| `RunTier2(ctx, entry)` | L.757 |
| `RunOrphanPrune(ctx, entry)` | L.873 |
| `PruneAllOrphans(ctx)` | L.1023 |
| `RunTier3(ctx, entry)` | L.1042 |
| `CheckAll(ctx)` | L.1171 |
| `pickScanRange(ctx, entry)` | L.542 |
| `pickScanRangeWithLag(ctx, entry)` | L.548 |
| `adaptiveFreeze(lagMs)` | L.501 |
| `lagBetween(upstream, downstream)` | L.515 |
| `upsertReconLag(ctx, table, col, lagMs)` | L.528 |
| `listActiveTableConfigs(ctx)` | L.1281 |
| `stampA(report, entry)` | L.1342 |
| `isOffPeak(now)`, `IsOffPeakForTest(start, end, now)` | L.1154 |

### `internal/recon/service/recon_tier_b.go` — Shadow ↔ Master

| Func | Từ dòng |
|---|---|
| `RunSegmentB(ctx, ref, deep)` | L.1381 |
| `RunSegmentBFor(ctx, masterTable, deep)` | L.1722 |
| `CheckAllSegmentB(ctx)` | L.1735 |
| `RunRowDiffB(ctx, ref, ids)` | L.1617 |
| `stampB(report, ref)` | L.1348 |
| `listActiveMasterBindings(ctx)` | L.1361 |
| `diffIDTs(shadow, master)` | L.1575 |
| `normalizeDiffVal(v)` | L.1709 |
| `SetMasterAgent(agent)` | L.1323 |
| `SetPlaneDBs(shadowDB, masterDB)` | L.1598 |

### `internal/recon/service/recon_core.go` — Base + shared

| Func | Hành động |
|---|---|
| `NewReconCore(...)` | Giữ |
| `errorReport(entry, checkType, tier, err)` | Giữ (private) |
| `diffIDs(a, b)` | Giữ (private) |
| `abs(x)`, `md5Hex(data)` | Giữ (private) |
| `unwrapBSONValue(v)` | Giữ (private) |
| `fnvHash32(s)` | Giữ (private) |
| `advisoryLockKey(name)` | Giữ (private) |

---

## Bước 6.5: Move Handlers → `internal/recon/handler/`

**Toàn bộ** `internal/handler/recon_handler.go` + `recon_heal_v4.go` → `internal/recon/handler/`:

**`recon_handler.go`** (20 funcs):

| Func | Hành động |
|---|---|
| `NewReconHandler(reconCore, db, schema, logger)` | Move |
| `WithBackfill(b, pub)` | Move |
| `WithHealer(healer)` | Move |
| `WithMaskingService(masking)` | Move |
| `WithTimestampDetector(td)` | Move |
| `WithMetadataRegistry(metadata)` | Move |
| `WithSignalClient(s)` | Move |
| `HandleReconCheck(msg)` | Move |
| `handleReconCheckSegmentB(ctx, msg, table, deep)` | Move (private) |
| `HandleReconHeal(msg)` | Move |
| `HandleRetryFailed(msg)` | Move |
| `HandleDebeziumSignal(msg)` | Move |
| `HandleBackfillSourceTs(msg)` | Move |
| `HandleDetectTimestampField(msg)` | Move |
| `resolveTargetTableConfig(targetTable)` | Move (private) |
| `resolveTableConfigByID(id)` | Move (private) |
| `updateFailedLog(id, status, errMsg)` | Move (private) |
| `logActivity(operation, table, status, rows, err)` | Move (private) |
| `sanitizeRetryRawJSON(table, raw)` | Move (private) |

**`recon_heal_v4.go`** (9 funcs) — Move toàn bộ:
- `WithShadowDB`, `WithNatsPublisher`, `healThresholdBlocked`
- `healSegmentB(ctx, msg, table)`
- `mapGpayToSourceIDs(ctx, shadowRel, gpayIDs)`
- `healSegmentA(ctx, msg, table)`
- `buildSnapshotIDFilter(engine, pkField, ids)`
- `respondErr(msg, err)`, `abs64(v)`

---

## Bước 6.6: Compile Check

```bash
go build ./internal/recon/...
go test ./internal/recon/...
```


---

# Phase 7: Domain `platform`

## Mục tiêu
`internal/platform/` — DLQ, masking, provisioning, admin, monitoring.

---

## Bước 7.1: `internal/platform/model.go`

| Struct | Table |
|---|---|
| `ActivityLog` | `cdc_system.cdc_activity_log` |

---

## Bước 7.2: Tạo `internal/platform/repository/gorm_activity_log_repo.go` (MỚI)

```go
type ActivityLogRepository interface {
    Insert(ctx, log *ActivityLog) error
    ListByTable(ctx, tableName string, limit int) ([]ActivityLog, error)
}
```

---

## Bước 7.3: Move theo sub-domain

### `internal/platform/masking/`

| File cũ | File mới |
|---|---|
| `internal/service/masking_service.go` | `internal/platform/masking/masking_service.go` |

**Key functions** (27 funcs — giữ nguyên toàn bộ):
- `NewMaskingService(db, logger, defaults...)`
- `SetMetadataRegistry(registrySvc)`, `SetHMACKey(key)`, `SetAESKey(key)`
- `MaskTableData(bindingKey, data)`
- `MaskJSONPayload(bindingKey, data)`
- `MaskFieldSample(bindingKey, field, value)`
- `MaskByStrategy(value, strategy)`
- `HashValue(v)`, `EncryptValue(v)`, `DecryptValue(s)`
- Private: `resolveBindingID`, `resolveMaskMap`, `maskMapRecursive`, `maskAnyRecursive`, `lookupMask`
- Crypto: `deriveAES256Key`, `encryptAESGCM`, `decryptAESGCM`

### `internal/platform/dlq/`

| File cũ | File mới |
|---|---|
| `internal/handler/dlq_handler.go` | `internal/platform/dlq/dlq_handler.go` |
| `internal/handler/dlq_state_machine.go` | `internal/platform/dlq/dlq_state_machine.go` |
| `internal/handler/dlq_circuit_breaker.go` | `internal/platform/dlq/dlq_circuit_breaker.go` |
| `internal/service/dlq_worker.go` | `internal/platform/dlq/dlq_worker.go` |

**Key functions `dlq_handler.go`** (21 funcs):
- `NewDLQHandler(nats, logger, dbs...)`
- `SetMaskingService(masking)`
- `HandleWithRetry/HandleWithRetryContext(...)`
- `sendToDLQ(ctx, subject, data, sourceTable, err)` (private)
- `buildFailedSyncLog(subject, data, sourceTable, err)` (private)
- `markPublishFailure(ctx, failedLogID, publishErr)` (private)
- `ReplayDLQ(dlqData)`, `extractDLQRecordID(data)` (private)
- Log helpers: `logInfo`, `logWarn`, `logError`
- `sanitizeDLQError(errMsg)`, `truncateDLQError(s, max)` (private)

**Key functions `dlq_state_machine.go`** (11 funcs):
- `NewDLQStateMachine(...)`, `Start(ctx)`, `RunOnce(ctx)`
- `retryOne(ctx, row)` (private)
- `nextReplayDelay(retryCount)` (private)
- `ReplayFailedLog(ctx, id)`, `classifyDLQErr(err)` (private)
- Log helpers

### `internal/platform/provisioning/`

| File cũ | File mới |
|---|---|
| `internal/handler/provisioning_handler.go` | `internal/platform/provisioning/provisioning_handler.go` |
| `internal/handler/provisioning_step_handlers.go` | `internal/platform/provisioning/step_handlers.go` |
| `internal/handler/provisioning_emit.go` | `internal/platform/provisioning/emit.go` |
| `internal/service/provisioning_orchestrator.go` | `internal/platform/provisioning/orchestrator.go` |
| `internal/service/provisioning_state_machine.go` | `internal/platform/provisioning/state_machine.go` |

**Key functions `orchestrator.go`** (22 funcs):
- `NewProvisioningOrchestrator(db, conn, logger)`
- `readState(ctx, sourceID)`, `readMode(ctx, sourceID)` (private)
- `nextLogSeq(ctx, sourceID)` (private)
- `casUpdateState(...)` (private)
- `publishCmd(...)` (private)
- `newCorrelationID(sourceID, step)` (private)
- `Advance(ctx, sourceID, actor)`
- `seedMasterBindingForAdvance(...)` (private)
- `lookupSourceTableForSource(...)`, `lookupMasterTableForSource(...)` (private)
- `HandleStepCompleted(ctx, ev)`
- `SetMode(ctx, sourceID, target, actor)`
- `Pause/Resume/Retry/Archive(ctx, sourceID, actor)`
- `RecoveryLoop(ctx)`, `recoveryTick(ctx)` (private)

**Key functions `step_handlers.go`** (18 funcs):
- `NewProvisioningStepHandler(...)`
- `HandleShadowBind(msg)`, `HandleScheduleEnable(msg)`
- `isMongoEngine(eng)` (private)
- `fetchSourceEngine(...)` (private)
- `preflightMongoSource(...)` (private)
- `resolveShadowTarget(...)` (private)
- `upsertShadowBinding(...)` (private)
- `inferSourceColumns(...)` (private)
- `pickSourceDSN(...)` (private)
- `inferPGCols(...)`, `inferMySQLCols(...)`, `inferMongoCols(...)` (private)
- `resolveMongoSampledType(types)` (private)
- `pgSafeType(dataType)`, `mysqlToPGType(dataType)`, `bsonToPGType(v)` (private)
- `firstNonEmpty(v, fallback)` (private)

### `internal/platform/activity/`

| File cũ | File mới |
|---|---|
| `internal/service/activity_logger.go` | `internal/platform/activity/activity_logger.go` |
| `internal/activity/` (taxonomy enums) | `internal/platform/activity/taxonomy.go` |

### `internal/platform/admin/` — giữ nguyên path

```
internal/admin/ → internal/platform/admin/
```

### `internal/platform/server/`

`internal/server/worker_server.go` → `internal/platform/server/worker_server.go`

Đây là DI root — update toàn bộ imports để point đến các domain mới. **Không thay đổi logic wiring.**

---

## Bước 7.4: Compile Check Toàn bộ

```bash
go build ./...
go test ./...
```

---

## Checklist Shared Utilities (sau P7)

Sau khi tất cả domains xong, review các private functions xuất hiện ở nhiều domain:

| Function | Xuất hiện ở | Giải pháp |
|---|---|---|
| `quoteIdent(v)` | Nhiều handlers | `internal/platform/sqlutil/quote.go` |
| `publishResult(msg, result)` | command_handler | `internal/platform/natsutil/reply.go` |
| `writeActivity(op, table, ...)` | Nhiều handlers | Dùng `ActivityLogger` từ platform |
| `resolveTargetTableConfig(...)` | Nhiều handlers | Dùng `MetadataRegistry` inject |
| `sanitizeAdminError(errMsg)` | command_handler | `internal/platform/admin/sanitize.go` |
