# 02_plan_repository.md — Phase 2: Tổ chức lại `internal/repository/`

## Hiện trạng: 11 files, flat trong `repository/`

```
connection_registry_repo.go     schema_log_repo.go
mapping_rule_repo.go            shadow_binding_repo.go
mapping_rule_v2_repo.go         source_object_registry_repo.go
master_binding_repo.go          sync_runtime_state_repo.go
pending_field_repo.go           transmute_schedule_repo.go
registry_repo.go
```

---

## Mục tiêu: 4 sub-folder + tạo 3 repo mới còn thiếu

### `internal/repository/source/`

| File cũ | File mới | Struct/Table |
|---|---|---|
| `connection_registry_repo.go` | `repository/source/connection_registry_repo.go` | `ConnectionRegistry` |
| `source_object_registry_repo.go` | `repository/source/source_object_registry_repo.go` | `SourceObjectRegistry` |
| `registry_repo.go` | `repository/source/registry_repo.go` | `TableRegistry` (V1 legacy) |
| `schema_log_repo.go` | `repository/source/schema_log_repo.go` | `SchemaChangeLog` |

**Functions `connection_registry_repo.go`** (4 funcs):
```go
func NewConnectionRegistryRepo(db *gorm.DB) *ConnectionRegistryRepo
func (r *ConnectionRegistryRepo) GetAll(ctx) ([]model.ConnectionRegistry, error)
func (r *ConnectionRegistryRepo) GetByID(ctx, id uint) (*model.ConnectionRegistry, error)
func (r *ConnectionRegistryRepo) GetByCode(ctx, code string) (*model.ConnectionRegistry, error)
// Cần thêm: ListActive(ctx) — hiện đang query inline tại service
```

**Functions `registry_repo.go`** (9 funcs):
```go
func NewRegistryRepo(db *gorm.DB) *RegistryRepo
func (r *RegistryRepo) GetAllActive(ctx) ([]model.TableRegistry, error)
func (r *RegistryRepo) GetByID(ctx, id uint) (*model.TableRegistry, error)
func (r *RegistryRepo) GetByTargetTable(ctx, targetTable string) (*model.TableRegistry, error)
func (r *RegistryRepo) GetAll(ctx, filter RegistryFilter) ([]model.TableRegistry, int64, error)
func (r *RegistryRepo) Create(ctx, entry *model.TableRegistry) error
func (r *RegistryRepo) Update(ctx, entry *model.TableRegistry) error
func (r *RegistryRepo) BulkCreate(ctx, entries []model.TableRegistry) (int, error)
func (r *RegistryRepo) GetStats(ctx) (*RegistryStats, error)
```

---

### `internal/repository/shadow/`

| File cũ | File mới | Struct/Table |
|---|---|---|
| `shadow_binding_repo.go` | `repository/shadow/shadow_binding_repo.go` | `ShadowBinding` |
| `pending_field_repo.go` | `repository/shadow/pending_field_repo.go` | `PendingField` |
| *(TẠO MỚI)* | `repository/shadow/failed_sync_log_repo.go` | `FailedSyncLog` |

**Functions `shadow_binding_repo.go`** (7 funcs):
```go
func NewShadowBindingRepo(db *gorm.DB) *ShadowBindingRepo
func (r *ShadowBindingRepo) GetByID(ctx, id int64) (*model.ShadowBinding, error)
func (r *ShadowBindingRepo) GetByCode(ctx, code string) (*model.ShadowBinding, error)
func (r *ShadowBindingRepo) GetActiveBySourceObject(ctx, sourceObjectID int64) (*model.ShadowBinding, error)
func (r *ShadowBindingRepo) ListBySourceObject(ctx, sourceObjectID int64) ([]model.ShadowBinding, error)
func (r *ShadowBindingRepo) Create(ctx, item *model.ShadowBinding) error
func (r *ShadowBindingRepo) Update(ctx, item *model.ShadowBinding) error
```

**TẠO MỚI `failed_sync_log_repo.go`** — extract từ inline GORM trong `batch_buffer.go`, `dlq_state_machine.go`:
```go
func NewFailedSyncLogRepo(db *gorm.DB) *FailedSyncLogRepo
func (r *FailedSyncLogRepo) Create(ctx, log *model.FailedSyncLog) error
func (r *FailedSyncLogRepo) GetByID(ctx, id uint64) (*model.FailedSyncLog, error)
func (r *FailedSyncLogRepo) Update(ctx, log *model.FailedSyncLog) error
func (r *FailedSyncLogRepo) GetPendingByTable(ctx, tableName string, limit int) ([]model.FailedSyncLog, error)
func (r *FailedSyncLogRepo) CountPending(ctx, tableName string) (int64, error)
```

---

### `internal/repository/master/`

| File cũ | File mới | Struct/Table |
|---|---|---|
| `master_binding_repo.go` | `repository/master/master_binding_repo.go` | `MasterBinding` |
| `mapping_rule_v2_repo.go` | `repository/master/mapping_rule_v2_repo.go` | `MappingRuleV2` |
| `mapping_rule_repo.go` | `repository/master/mapping_rule_repo.go` ⚠️ V1 | `MappingRule` |
| `sync_runtime_state_repo.go` | `repository/master/sync_runtime_state_repo.go` | `SyncRuntimeState` |
| `transmute_schedule_repo.go` | `repository/master/transmute_schedule_repo.go` | `TransmuteSchedule` |

**Functions `master_binding_repo.go`** (9 funcs):
```go
func NewMasterBindingRepo(db *gorm.DB) *MasterBindingRepo
func (r *MasterBindingRepo) GetByID(ctx, id int64) (*model.MasterBinding, error)
func (r *MasterBindingRepo) GetByCode(ctx, code string) (*model.MasterBinding, error)
func (r *MasterBindingRepo) GetByMasterTable(ctx, masterTable string) (*model.MasterBinding, error)
func (r *MasterBindingRepo) ListBySourceObject(ctx, sourceObjectID int64) ([]model.MasterBinding, error)
func (r *MasterBindingRepo) ListActiveBySourceObject(ctx, sourceObjectID int64) ([]model.MasterBinding, error)
func (r *MasterBindingRepo) ListActiveByShadowBinding(ctx, shadowBindingID int64) ([]model.MasterBinding, error)
func (r *MasterBindingRepo) Create(ctx, item *model.MasterBinding) error
func (r *MasterBindingRepo) Update(ctx, item *model.MasterBinding) error
```

**Functions `mapping_rule_v2_repo.go`** (8 funcs):
```go
func NewMappingRuleV2Repo(db *gorm.DB) *MappingRuleV2Repo
func (r *MappingRuleV2Repo) ListBySourceObject(ctx, sourceObjectID int64) ([]model.MappingRuleV2, error)
func (r *MappingRuleV2Repo) ListActiveByMasterBinding(ctx, masterBindingID int64) ([]model.MappingRuleV2, error)
func (r *MappingRuleV2Repo) ListActiveBySourceObject(ctx, sourceObjectID int64) ([]model.MappingRuleV2, error)
func (r *MappingRuleV2Repo) ListActiveBySourceObjectAndBinding(ctx, sourceObjectID, shadowBindingID int64) ([]model.MappingRuleV2, error)
func (r *MappingRuleV2Repo) Create(ctx, item *model.MappingRuleV2) error
func (r *MappingRuleV2Repo) Update(ctx, item *model.MappingRuleV2) error
func (r *MappingRuleV2Repo) GetActiveRulesBySourceTable(ctx, sourceTable string) ([]model.MappingRuleV2, error)
func (r *MappingRuleV2Repo) ListGlobalSensitiveFields(ctx) ([]model.SensitiveField, error)
```

---

### `internal/repository/recon/`

| File cũ | File mới | Struct/Table |
|---|---|---|
| *(TẠO MỚI)* | `repository/recon/snapshot_dlq_repo.go` | `SnapshotDLQ` |
| *(TẠO MỚI)* | `repository/recon/reconciliation_report_repo.go` | `ReconciliationReport` |

**TẠO MỚI `snapshot_dlq_repo.go`** — extract từ inline GORM trong `snapshot_runner_handler.go`:
```go
func NewSnapshotDLQRepo(db *gorm.DB) *SnapshotDLQRepo
func (r *SnapshotDLQRepo) Create(ctx, item *model.SnapshotDLQ) error
func (r *SnapshotDLQRepo) GetPending(ctx) ([]model.SnapshotDLQ, error)
func (r *SnapshotDLQRepo) MarkDone(ctx, id int64) error
func (r *SnapshotDLQRepo) MarkError(ctx, id int64, errMsg string) error
```

**TẠO MỚI `reconciliation_report_repo.go`** — extract từ inline GORM trong `recon_core.go`:
```go
func NewReconciliationReportRepo(db *gorm.DB) *ReconciliationReportRepo
func (r *ReconciliationReportRepo) Create(ctx, report *model.ReconciliationReport) error
func (r *ReconciliationReportRepo) GetByTable(ctx, targetTable string, limit int) ([]model.ReconciliationReport, error)
func (r *ReconciliationReportRepo) GetLatest(ctx, targetTable string) (*model.ReconciliationReport, error)
```

---

## Tóm tắt thay đổi

| Sub-folder | Files move | Files mới tạo |
|---|---|---|
| `repository/source/` | 4 files | 0 |
| `repository/shadow/` | 2 files | 1 (`failed_sync_log_repo.go`) |
| `repository/master/` | 5 files | 0 |
| `repository/recon/` | 0 files | 2 (`snapshot_dlq_repo.go`, `reconciliation_report_repo.go`) |
| **Tổng** | **11 files move** | **3 files mới** |
