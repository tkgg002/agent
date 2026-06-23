# 03_implementation_model_repo.md — internal/model + internal/repository

---

## internal/model/ — GORM DB Entities (18 files)

| File | Struct | Mô tả |
|---|---|---|
| `activity_log.go` | `ActivityLog` | Log các hoạt động CDC (op, table, status, rows) |
| `cdc_event.go` | `CDCEvent`, `CDCEventData`, `UpsertRecord` | Event từ Kafka (before/after/op) |
| `connection_registry.go` | `ConnectionRegistry` | Thông tin kết nối nguồn (host, port, user, pass, type) |
| `failed_sync_log.go` | `FailedSyncLog` | Log các records sync thất bại (để retry) |
| `mapping_rule.go` | `MappingRule` | V1: ánh xạ source_field → target_field |
| `mapping_rule_v2.go` | `MappingRuleV2` | V2: ánh xạ với status approval workflow |
| `master_binding.go` | `MasterBinding` | Binding shadow table → master table |
| `pending_field.go` | `PendingField` | Fields chưa được approve để thêm vào schema |
| `reconciliation_report.go` | `ReconciliationReport` | Kết quả reconciliation (source count, dest count, mismatch) |
| `schema_change_log.go` | `SchemaChangeLog` | Log schema changes (ALTER, field additions) |
| `sensitive_field.go` | `SensitiveField` | Config masking (field name, strategy) |
| `shadow_binding.go` | `ShadowBinding` | Binding source object → shadow table |
| `snapshot_dlq.go` | `SnapshotDLQ` | DLQ cho snapshot failures |
| `source_object_registry.go` | `SourceObjectRegistry` | Registry source objects (table/collection) |
| `sync_runtime_state.go` | `SyncRuntimeState` | Runtime state của sync job (cursor, last_success) |
| `table_registry.go` | `TableRegistry` | Registry chính: source → target mapping (engine, schema, connector) |
| `transmute_schedule.go` | `TransmuteSchedule` | Lịch transmute cho mỗi master binding |
| `worker_schedule.go` | `WorkerSchedule` | Worker job schedule config |

---

## internal/repository/ — Data Access Layer (11 files)

| File | Struct | CRUD Methods |
|---|---|---|
| `connection_registry_repo.go` | `ConnectionRegistryRepo` | `Create`, `Update` |
| `mapping_rule_repo.go` | `MappingRuleRepo` | `GetAllActive`, `GetByTable`, `Create`, `CreateIfNotExists`, `GetAll` |
| `mapping_rule_v2_repo.go` | `MappingRuleV2Repo` | `ListBySourceObject`, `ListActiveByMasterBinding`, `ListActiveBySourceObject`, `ListActiveBySourceObjectAndBinding`, `Create`, `Update`, `GetActiveRulesBySourceTable`, `ListGlobalSensitiveFields` |
| `master_binding_repo.go` | `MasterBindingRepo` | `GetByID`, `GetByCode`, `GetByMasterTable`, `ListBySourceObject`, `ListActiveBySourceObject`, `ListActiveByShadowBinding`, `Create`, `Update` |
| `pending_field_repo.go` | `PendingFieldRepo` | `GetByID`, `GetByStatus`, `Update`, `UpsertPendingField`, `GetTableColumns`, `GetTableColumnsInSchema` |
| `registry_repo.go` | `RegistryRepo` | `GetAllActive`, `GetByID`, `GetByTargetTable`, `GetAll(filter)`, `Create`, `Update`, `BulkCreate`, `GetStats` |
| `schema_log_repo.go` | `SchemaLogRepo` | `Create`, `GetByTable` |
| `shadow_binding_repo.go` | `ShadowBindingRepo` | `GetByID`, `GetByCode`, `GetActiveBySourceObject`, `ListBySourceObject`, `Create`, `Update` |
| `source_object_registry_repo.go` | `SourceObjectRegistryRepo` | `GetAll`, `GetActive`, `GetByID`, `GetByNormalizedKey`, `Create`, `Update` |
| `sync_runtime_state_repo.go` | `SyncRuntimeStateRepo` | `ListBySourceObject`, `GetByShadowBinding`, `GetByMasterBinding`, `Create`, `Update` |
| `transmute_schedule_repo.go` | `TransmuteScheduleRepo` | `GetByMasterBinding` |
