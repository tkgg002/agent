# 02_plan_full.md — Master Execution Plan (Toàn bộ quá trình)

> **ADR-002**: Strangler Fig — từng bước nhỏ, compile+test+commit sau mỗi nhóm  
> **ADR-001**: Package name theo sub-folder (package master, package shadow...)  
> **ADR-003**: Không có `internal/domain/` prefix

---

## PHASE 1: model/ → 4 sub-folders

**Mục tiêu**: Tổ chức 18 model files vào sub-folder theo plane

### Bước 1.1 — model/system/ (ít rủi ro nhất, không có downstream dep)

```bash
mkdir -p internal/model/system
```

| Move | File cũ | File mới | Package mới |
|---|---|---|---|
| [ ] | model/activity_log.go | model/system/activity_log.go | package system |
| [ ] | model/snapshot_dlq.go | model/system/snapshot_dlq.go | package system |
| [ ] | model/reconciliation_report.go | model/system/reconciliation_report.go | package system |

```bash
go build ./internal/model/system/...
git commit -m "refactor: move model/system structs to sub-folder"
```

### Bước 1.2 — model/source/

```bash
mkdir -p internal/model/source
```

| Move | File cũ | File mới | Package mới |
|---|---|---|---|
| [ ] | model/connection_registry.go | model/source/connection_registry.go | package source |
| [ ] | model/source_object_registry.go | model/source/source_object_registry.go | package source |
| [ ] | model/table_registry.go | model/source/table_registry.go | package source ⚠️V1 |
| [ ] | model/schema_change_log.go | model/source/schema_change_log.go | package source |

```bash
go build ./internal/model/source/...
git commit -m "refactor: move model/source structs to sub-folder"
```

### Bước 1.3 — model/shadow/

```bash
mkdir -p internal/model/shadow
```

| Move | File cũ | File mới | Package mới |
|---|---|---|---|
| [ ] | model/shadow_binding.go | model/shadow/shadow_binding.go | package shadow |
| [ ] | model/cdc_event.go | model/shadow/cdc_event.go | package shadow |
| [ ] | model/failed_sync_log.go | model/shadow/failed_sync_log.go | package shadow |
| [ ] | model/pending_field.go | model/shadow/pending_field.go | package shadow |
| [ ] | model/sensitive_field.go | model/shadow/sensitive_field.go | package shadow |

```bash
go build ./internal/model/shadow/...
git commit -m "refactor: move model/shadow structs to sub-folder"
```

### Bước 1.4 — model/master/

```bash
mkdir -p internal/model/master
```

| Move | File cũ | File mới | Package mới |
|---|---|---|---|
| [ ] | model/master_binding.go | model/master/master_binding.go | package master |
| [ ] | model/mapping_rule_v2.go | model/master/mapping_rule_v2.go | package master |
| [ ] | model/mapping_rule.go | model/master/mapping_rule.go | package master ⚠️V1 |
| [ ] | model/sync_runtime_state.go | model/master/sync_runtime_state.go | package master |
| [ ] | model/worker_schedule.go | model/master/worker_schedule.go | package master |
| [ ] | model/transmute_schedule.go | model/master/transmute_schedule.go | package master |

```bash
go build ./internal/model/...
go build ./...         # Kiểm tra toàn bộ callers
go test ./...
git commit -m "refactor: move model/master structs to sub-folder"
```

---

## PHASE 2: repository/ → 4 sub-folders + 3 repos mới

### Bước 2.1 — repository/source/

```bash
mkdir -p internal/repository/source
```

| Move | File cũ | File mới | Package mới |
|---|---|---|---|
| [ ] | repository/connection_registry_repo.go | repository/source/connection_registry_repo.go | package source |
| [ ] | repository/source_object_registry_repo.go | repository/source/source_object_registry_repo.go | package source |
| [ ] | repository/registry_repo.go | repository/source/registry_repo.go | package source |
| [ ] | repository/schema_log_repo.go | repository/source/schema_log_repo.go | package source |

```bash
go build ./internal/repository/source/...
git commit -m "refactor: move repository/source to sub-folder"
```

### Bước 2.2 — repository/shadow/ + TẠO failed_sync_log_repo.go

```bash
mkdir -p internal/repository/shadow
```

| Move | File cũ | File mới | Package mới |
|---|---|---|---|
| [ ] | repository/shadow_binding_repo.go | repository/shadow/shadow_binding_repo.go | package shadow |
| [ ] | repository/pending_field_repo.go | repository/shadow/pending_field_repo.go | package shadow |

**TẠO MỚI** `repository/shadow/failed_sync_log_repo.go` (extract từ inline GORM trong batch_buffer.go + dlq_state_machine.go):

```go
package shadow

import (
    "context"
    "github.com/.../internal/model/shadow"
    "gorm.io/gorm"
)

type FailedSyncLogRepo struct { db *gorm.DB }

func NewFailedSyncLogRepo(db *gorm.DB) *FailedSyncLogRepo
func (r *FailedSyncLogRepo) Create(ctx context.Context, log *shadow.FailedSyncLog) error
func (r *FailedSyncLogRepo) GetByID(ctx context.Context, id uint64) (*shadow.FailedSyncLog, error)
func (r *FailedSyncLogRepo) Update(ctx context.Context, log *shadow.FailedSyncLog) error
func (r *FailedSyncLogRepo) GetPendingByTable(ctx context.Context, tableName string, limit int) ([]shadow.FailedSyncLog, error)
func (r *FailedSyncLogRepo) CountPending(ctx context.Context, tableName string) (int64, error)
```

```bash
go build ./internal/repository/shadow/...
git commit -m "refactor: move repository/shadow + create FailedSyncLogRepo"
```

### Bước 2.3 — repository/master/

```bash
mkdir -p internal/repository/master
```

| Move | File cũ | File mới | Package mới |
|---|---|---|---|
| [ ] | repository/master_binding_repo.go | repository/master/master_binding_repo.go | package master |
| [ ] | repository/mapping_rule_v2_repo.go | repository/master/mapping_rule_v2_repo.go | package master |
| [ ] | repository/mapping_rule_repo.go | repository/master/mapping_rule_repo.go | package master ⚠️V1 |
| [ ] | repository/sync_runtime_state_repo.go | repository/master/sync_runtime_state_repo.go | package master |
| [ ] | repository/transmute_schedule_repo.go | repository/master/transmute_schedule_repo.go | package master |

```bash
go build ./internal/repository/master/...
git commit -m "refactor: move repository/master to sub-folder"
```

### Bước 2.4 — repository/recon/ + TẠO 2 repos mới

```bash
mkdir -p internal/repository/recon
```

**TẠO MỚI** `repository/recon/snapshot_dlq_repo.go` (extract từ snapshot_runner_handler.go):

```go
package recon

type SnapshotDLQRepo struct { db *gorm.DB }

func NewSnapshotDLQRepo(db *gorm.DB) *SnapshotDLQRepo
func (r *SnapshotDLQRepo) Create(ctx, item *system.SnapshotDLQ) error
func (r *SnapshotDLQRepo) GetPending(ctx) ([]system.SnapshotDLQ, error)
func (r *SnapshotDLQRepo) MarkDone(ctx, id int64) error
func (r *SnapshotDLQRepo) MarkError(ctx, id int64, errMsg string) error
```

**TẠO MỚI** `repository/recon/reconciliation_report_repo.go` (extract từ recon_core.go):

```go
package recon

type ReconciliationReportRepo struct { db *gorm.DB }

func NewReconciliationReportRepo(db *gorm.DB) *ReconciliationReportRepo
func (r *ReconciliationReportRepo) Create(ctx, report *system.ReconciliationReport) error
func (r *ReconciliationReportRepo) GetByTable(ctx, targetTable string, limit int) ([]system.ReconciliationReport, error)
func (r *ReconciliationReportRepo) GetLatest(ctx, targetTable string) (*system.ReconciliationReport, error)
```

```bash
go build ./internal/repository/...
go build ./...
go test ./...
git commit -m "refactor: move repository/recon + create SnapshotDLQRepo + ReconciliationReportRepo"
```

---

## PHASE 3: service/ → 5 sub-folders

### Bước 3.1 — service/governance/ (ít coupling nhất, làm trước)

```bash
mkdir -p internal/service/governance
# Move 10 files:
mv service/masking_service.go service/governance/
mv service/schema_inspector.go service/governance/
mv service/schema_validator.go service/governance/
mv service/activity_logger.go service/governance/
mv service/partition_dropper.go service/governance/
mv service/wal_monitor.go service/governance/
mv service/full_count_aggregator.go service/governance/
mv service/debezium_signal.go service/governance/
mv service/timestamp_detector.go service/governance/
mv service/backfill_source_ts.go service/governance/
# Đổi package declaration → package governance
go build ./internal/service/governance/... && go build ./...
git commit -m "refactor: move service/governance to sub-folder"
```

### Bước 3.2 — service/source/

```bash
mkdir -p internal/service/source
# Move 8 files:
mv service/metadata_registry_service.go service/source/
mv service/registry_service.go service/source/
mv service/connection_manager.go service/source/
mv service/connection_overrides.go service/source/
mv service/connector_resolver.go service/source/
mv service/source_router.go service/source/
mv service/mongo_introspection.go service/source/
mv service/scan_service.go service/source/
# package source
go build ./internal/service/source/... && go build ./...
git commit -m "refactor: move service/source to sub-folder"
```

### Bước 3.3 — service/shadow/

```bash
mkdir -p internal/service/shadow
# Move 7 files:
mv service/schema_adapter.go service/shadow/
mv service/dynamic_mapper.go service/shadow/
mv service/child_explode.go service/shadow/
mv service/enrichment_service.go service/shadow/
mv service/bridge_service.go service/shadow/
mv service/type_resolver.go service/shadow/
mv service/text_sanitizer.go service/shadow/
# package shadow
go build ./internal/service/shadow/... && go build ./...
git commit -m "refactor: move service/shadow to sub-folder"
```

### Bước 3.4 — service/master/

```bash
mkdir -p internal/service/master
# Move 7 files + transmute/ folder:
mv service/master_ddl_generator.go service/master/
mv service/transmuter.go service/master/
mv service/transmute_scheduler.go service/master/
mv service/child_explode_master.go service/master/
mv service/job_monitor.go service/master/
mv service/transform_registry.go service/master/
mv -r service/transmute service/master/
# package master
go build ./internal/service/master/... && go build ./...
git commit -m "refactor: move service/master to sub-folder"
```

### Bước 3.5 — service/recon/ + Tách recon_core.go

```bash
mkdir -p internal/service/recon
# Move 7 files:
mv service/recon_source_agent.go service/recon/
mv service/recon_dest_agent.go service/recon/
mv service/recon_heal.go service/recon/
mv service/recon_alert.go service/recon/
mv service/dlq_worker.go service/recon/
mv service/provisioning_orchestrator.go service/recon/
mv service/provisioning_state_machine.go service/recon/
```

**Tách recon_core.go (1900L) → 3 files** (đây là bước phức tạp nhất trong Phase 3):

| File mới | Funcs | Từ dòng |
|---|---|---|
| `service/recon/recon_engine.go` | NewReconCore, CheckAll, PruneAllOrphans, helpers | Base |
| `service/recon/recon_tier_a.go` | RunTier1/2/3, RunOrphanPrune, lag/scan | Source↔Shadow |
| `service/recon/recon_tier_b.go` | RunSegmentB, RunSegmentBFor, RunRowDiffB, diffIDTs | Shadow↔Master |

```bash
# package recon
go build ./internal/service/recon/... && go build ./... && go test ./...
git commit -m "refactor: move service/recon + split recon_core.go into 3 files"
```

---

## PHASE 4: handler/ → 5 sub-folders + Tách command_handler.go

> ⚠️ command_handler.go (3437 dòng, 77 funcs) phải tách trước khi xóa.

### Bước 4.1 — handler/shadow/ (không tách, move nguyên)

```bash
mkdir -p internal/handler/shadow
mv handler/kafka_consumer.go handler/shadow/       # 41 funcs
mv handler/event_handler.go handler/shadow/        # 15 funcs
mv handler/event_bridge.go handler/shadow/         # 16 funcs
mv handler/batch_buffer.go handler/shadow/         # 26 funcs
mv handler/consumer_pool.go handler/shadow/        # 7 funcs
# package shadow
go build ./internal/handler/shadow/... && go build ./...
git commit -m "refactor: move handler/shadow to sub-folder"
```

### Bước 4.2 — handler/recon/ (move nguyên)

```bash
mkdir -p internal/handler/recon
mv handler/recon_handler.go handler/recon/         # 20 funcs
mv handler/recon_heal_v4.go handler/recon/         # 9 funcs
# package recon
go build ./internal/handler/recon/... && go build ./...
git commit -m "refactor: move handler/recon to sub-folder"
```

### Bước 4.3 — Tách command_handler.go → 6 handler files

**4.3a: handler/source/sync_handler.go** — Funcs từ L.2818-L.3221:
```
HandleSyncRegister / HandleSyncState / HandleRestartDebezium
verifyDebeziumConnector / detectConnectorName
connectGET / connectPOST / connectPUT / connectCall
```
Struct mới: `SyncHandler{db, natsConn, kafkaConnectURL, metadataReg, logger}`

**4.3b: handler/master/schema_ddl_handler.go** — Funcs từ L.136-L.1237, L.2568-L.2985-L.3154:
```
HandleStandardize / HandleCreateDefaultColumns / HandleAlterColumn / HandleDropGINIndex
ensureCDCColumns* / hasColumn* / tableExists* / listShadowColumns*
normalizePGType / isSafeIdent / isSafeType / systemFieldSet
normalizeMappingRuleDataType / processDiscoveryRows / bridgeMappingRulesToV2
```

**4.3c: handler/master/batch_transform_handler.go** — L.1301-L.1816:
```
HandleMasterSwap / HandleBatchTransform / detectPrimaryKey
```

**4.3d: handler/orchestration/discover_handler.go** — L.407-L.687, L.962-L.1237, L.2693-L.2817:
```
HandleDiscover / HandleScanFields
scanFieldsMongoSource / scanFieldsDebezium / findSimilarCollections
inferSQLTypeFromLegacyCatalogProp
```

**4.3e: handler/orchestration/scan_handler.go** — L.1238, L.1817-L.2540:
```
HandleBackfill / HandleScanRawData / HandleScanArrayFields / HandlePeriodicScan
validScanIdent / explodePathToPGPath / flattenJSONWithTypes / buildCastExpr
```

**4.3f: handler/orchestration/mongo_discover_handler.go** — L.1378-L.1571:
```
HandleDiscoverMongoDatabases / HandleDiscoverMongoCollections / replyMongoDiscovery
```

**Shared helpers** (còn lại sau khi tách hết Handle*):
```
publishResult / publishResultWithSubject     → mỗi handler tự implement
writeActivity                               → inject *service.ActivityLogger
resolveTargetTableConfig                    → inject MetadataRegistry
sanitizeAdminError / sanitizeAdminResultMap → internal/admin/sanitize.go
quoteCommandIdent / quoteCommandQualifiedTable → pkgs/sqlutil/quote.go
```

```bash
# package master, package source, package orchestration
go build ./internal/handler/master/... && go build ./internal/handler/source/...
go build ./internal/handler/orchestration/... && go build ./... && go test ./...
git commit -m "refactor: split command_handler.go into domain-specific handlers"
```

### Bước 4.4 — handler/orchestration/ (move phần còn lại)

```bash
mv handler/provisioning_handler.go handler/orchestration/
mv handler/provisioning_step_handlers.go handler/orchestration/
mv handler/provisioning_emit.go handler/orchestration/
mv handler/snapshot_runner_handler.go handler/orchestration/
mv handler/dlq_handler.go handler/orchestration/
mv handler/dlq_state_machine.go handler/orchestration/
mv handler/dlq_circuit_breaker.go handler/orchestration/
# package orchestration
go build ./internal/handler/orchestration/... && go build ./...
git commit -m "refactor: move remaining orchestration handlers to sub-folder"
```

---

## PHASE 5: server/ — Update DI Wiring + Cleanup

### Bước 5.1 — Update imports trong worker_server.go

Dùng gopls rename hoặc IDE để auto-update references. Nếu làm thủ công, dùng named imports:

```go
import (
    reposource "github.com/.../internal/repository/source"
    reposhadow "github.com/.../internal/repository/shadow"
    repomaster "github.com/.../internal/repository/master"
    reporecon  "github.com/.../internal/repository/recon"

    svcsource "github.com/.../internal/service/source"
    svcshadow  "github.com/.../internal/service/shadow"
    svcmaster  "github.com/.../internal/service/master"
    svcgov     "github.com/.../internal/service/governance"
    svcrecon   "github.com/.../internal/service/recon"

    hdrshadow "github.com/.../internal/handler/shadow"
    hdrrecon  "github.com/.../internal/handler/recon"
    hdrsource "github.com/.../internal/handler/source"
    hdrmaster "github.com/.../internal/handler/master"
    hdrorch   "github.com/.../internal/handler/orchestration"
)
```

```bash
go build ./... && go test ./...
git commit -m "refactor: update DI wiring imports in worker_server.go"
```

### Bước 5.2 — Cleanup empty root folders

```bash
# Verify không còn file nào ở root layer folders
ls internal/model/*.go     # phải trống
ls internal/repository/*.go
ls internal/service/*.go
ls internal/handler/*.go

# Merge internal/activity/ → internal/model/system/
# (chỉ chứa taxonomy enums, copy constants vào activity_log.go)

# Xóa sau khi verify
rmdir internal/model internal/repository internal/service internal/handler

go build ./... && go test ./...
git commit -m "refactor: cleanup empty root layer folders after restructure"
```

---

## Tổng kết

| Phase | Bước | Files | Commit message pattern |
|---|---|---|---|
| 1 — model | 4 bước | 18 move | `refactor: move model/<sub> to sub-folder` |
| 2 — repository | 4 bước | 11 move + 3 new | `refactor: move repository/<sub> + create <Repo>` |
| 3 — service | 5 bước | 40 move, recon_core tách 3 | `refactor: move service/<sub> to sub-folder` |
| 4 — handler | 4 bước | 14 move, command_handler tách 6 | `refactor: split/move handler/<sub>` |
| 5 — server | 2 bước | 1 update + cleanup | `refactor: update DI wiring + cleanup` |
| **Tổng** | **19 bước** | **~86 files** | **~19 commits** |

## Compile gate tổng (sau cùng)

```bash
go build ./...
go test ./...
make run   # kiểm tra service start OK
```

