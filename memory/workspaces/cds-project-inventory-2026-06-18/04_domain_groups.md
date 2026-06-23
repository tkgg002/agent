# Domain Groups Analysis — `centralized-data-service/internal/`

> **Nguồn**: Code review thực tế — 141 production Go files  
> **Phương pháp**: Phân loại theo vòng đời dữ liệu trong CDC pipeline  
> **Bao gồm**: handler/ + service/ + model/ + repository/

---

## Pipeline tổng quan (căn cứ phân nhóm)

```
Source DB ──[Debezium]──► Kafka ──[sinkworker]──► Shadow DB ──[transmuter]──► Master DB
    │                                                  │                          │
  [scan/discover]                               [reconcile]                  [reconcile]
    │                                                  │
  [provisioning wizard]                         [heal/backfill]
```

Domain groups được đặt tên theo **vai trò trong pipeline**, không theo layer kỹ thuật.

---

## 7 Domains × 4 Layers

---

### Domain 1: `source` — Quản lý kết nối & đăng ký nguồn

> **Trả lời "what"**: Nguồn dữ liệu là gì? Kết nối vào bằng cách nào? Có bảng nào đang active?

```
Model           │ Repository                    │ Service                        │ Handler
────────────────┼───────────────────────────────┼────────────────────────────────┼────────────────────────────
ConnectionRegistry    connection_registry_repo   metadata_registry_service.go    HandleSyncRegister
SourceObjectRegistry  source_object_registry_repo registry_service.go            HandleSyncState
TableRegistry(legacy) registry_repo              connection_manager.go           HandleRestartDebezium
SchemaChangeLog       schema_log_repo            connection_overrides.go
                                                 connector_resolver.go
                                                 source_router.go
```

**Models & tables:**
- `connection_registry` → thông tin kết nối DB nguồn (host, port, secret_ref, engine_type)
- `source_object_registry` → từng bảng/collection nguồn đang được track
- `table_registry` ⚠️ (legacy) → god model tổng hợp, sẽ bỏ khi V2 hoàn thiện
- `schema_change_log` → lịch sử schema drift của nguồn

**Chức năng đặc trưng**: Resolve DSN, cache in-memory (MetadataRegistry), NATS reload trigger.

---

### Domain 2: `schema` — Quản lý mapping & DDL

> **Trả lời "how"**: Trường nguồn `user_id` sẽ ánh xạ thành cột `user_id BIGINT` ở shadow như thế nào?

```
Model           │ Repository                    │ Service                        │ Handler
────────────────┼───────────────────────────────┼────────────────────────────────┼────────────────────────────
MappingRuleV2         mapping_rule_v2_repo       master_ddl_generator.go         HandleStandardize
MappingRule(V1)⚠️     mapping_rule_repo ⚠️       schema_inspector.go             HandleCreateDefaultColumns
PendingField          pending_field_repo         schema_validator.go             HandleAlterColumn
SensitiveField        (inline GORM)              type_resolver.go                HandleDropGINIndex
                                                 text_sanitizer.go               HandleScanFields (schema part)
                                                 transform_registry.go           MasterDDLHandler.*
                                                 transmute/strategy.go
```

**Models & tables:**
- `mapping_rule_v2` → field mapping đã được duyệt: `source_field → target_column, data_type, status`
- `pending_fields` → fields vừa discover, chờ review/approve trước khi ALTER TABLE
- `sensitive_fields` → cấu hình masking strategy per-field (global)
- `mapping_rule` ⚠️ deprecated → V1, đang được migrate sang V2

**Chức năng đặc trưng**: Approve/reject pending fields, generate DDL, type coercion rules.

---

### Domain 3: `ingestion` — Thu thập dữ liệu vào Shadow

> **Trả lời "flow"**: CDC event đến từ Kafka → xử lý ra sao → ghi vào shadow table nào?

```
Model           │ Repository                    │ Service                        │ Handler
────────────────┼───────────────────────────────┼────────────────────────────────┼────────────────────────────
CDCEvent              shadow_binding_repo        schema_adapter.go               kafka_consumer.go
ShadowBinding         (inline GORM sinkworker)   dynamic_mapper.go               event_handler.go
UpsertRecord                                     enrichment_service.go           event_bridge.go
                                                 schema_validator.go             batch_buffer.go
                                                 child_explode.go                consumer_pool.go
                                                 source_router.go
                                                 ─── sinkworker/ ───
                                                 sinkworker.go
                                                 envelope.go
                                                 upsert.go
                                                 schema_manager.go
```

**Models & tables:**
- `CDCEvent` → Debezium envelope shape (in-memory, không persist)
- `ShadowBinding` → bind source_object → shadow_table (schema + connection)
- `UpsertRecord` → internal batch write record

**Chức năng đặc trưng**: Adaptive batching, OCC via `_source_ts`, sinkworker fencing token.

---

### Domain 4: `discovery` — Khám phá & quét dữ liệu nguồn

> **Trả lời "what fields"**: Collection MongoDB `payments` có những trường gì? Kiểu dữ liệu nào?

```
Model           │ Repository                    │ Service                        │ Handler
────────────────┼───────────────────────────────┼────────────────────────────────┼────────────────────────────
SnapshotDLQ           (inline GORM)              mongo_introspection.go          snapshot_runner_handler.go
SourceObjectRegistry  source_object_registry_repo scan_service.go               HandleDiscover
                                                 timestamp_detector.go           HandleScanRawData
                                                 backfill_source_ts.go           HandleScanArrayFields
                                                 child_explode.go (array)        HandlePeriodicScan
                                                                                 HandleDiscoverMongoDatabases
                                                                                 HandleDiscoverMongoCollections
                                                                                 HandleBackfill
```

**Models & tables:**
- `SnapshotDLQ` → track snapshot jobs bị fail, retry state
- Dùng chung `SourceObjectRegistry` với domain `source`

**Chức năng đặc trưng**: MongoDB `Find` sampling, infer field types từ docs, snapshot bypass Debezium.

---

### Domain 5: `transmute` — Vật chất hóa Shadow → Master

> **Trả lời "where"**: Dữ liệu đã ở shadow, bao giờ và bằng cách nào nó xuất hiện trong Master DB?

```
Model           │ Repository                    │ Service                        │ Handler
────────────────┼───────────────────────────────┼────────────────────────────────┼────────────────────────────
MasterBinding         master_binding_repo        transmuter.go                   transmute_handler.go
TransmuteSchedule     transmute_schedule_repo    transmute_scheduler.go          HandleBatchTransform
SyncRuntimeState      sync_runtime_state_repo    job_monitor.go                  HandleMasterSwap
WorkerSchedule        (inline GORM)              child_explode_master.go
                                                 transform_registry.go
                                                 type_resolver.go
```

**Models & tables:**
- `master_binding` → bind shadow_table → master_table với mapping V2
- `transmute_schedule` → lịch chạy: cron + fencing token + FOR UPDATE SKIP LOCKED
- `sync_runtime_state` → track DDL apply state, tránh chạy lại DDL thừa
- `worker_schedule` → lịch tổng quát

**Chức năng đặc trưng**: Chunked UPDATE tránh lock, DDL apply với cache invalidation, cron + fencing.

---

### Domain 6: `recon` — Đối soát & sửa lỗi dữ liệu

> **Trả lời "integrity"**: Source có 1M docs, shadow có 999K rows → record nào bị thiếu? Sai ở đâu?

```
Model           │ Repository                    │ Service                        │ Handler
────────────────┼───────────────────────────────┼────────────────────────────────┼────────────────────────────
ReconciliationReport  (inline GORM) ⚠️           recon_core.go                   recon_handler.go
FailedSyncLog         (inline GORM) ⚠️           recon_source_agent.go           recon_heal_v4.go
                                                 recon_dest_agent.go
                                                 recon_heal.go
                                                 recon_alert.go
                                                 full_count_aggregator.go
                                                 wal_monitor.go
                                                 debezium_signal.go
```

**Models & tables:**
- `cdc_reconciliation_report` → kết quả mỗi lần chạy tier1/2/3 + segment A/B
- `failed_sync_logs` → DLQ: events lỗi chờ retry với exponential backoff

**⚠️ Thiếu repo**: `ReconciliationReport` và `FailedSyncLog` write thẳng qua GORM inline → cần tạo repo riêng khi refactor.

---

### Domain 7: `platform` — Hạ tầng & Vận hành

> **Trả lời "how it runs"**: DLQ retry, masking PII, provisioning wizard, admin API, monitoring.

```
Model           │ Repository                    │ Service                        │ Handler
────────────────┼───────────────────────────────┼────────────────────────────────┼────────────────────────────
ActivityLog           (inline GORM) ⚠️           masking_service.go              dlq_handler.go
                                                 activity_logger.go              dlq_state_machine.go
                                                 dlq_worker.go                   dlq_circuit_breaker.go
                                                 partition_dropper.go            provisioning_handler.go
                                                 bridge_service.go               provisioning_step_handlers.go
                                                 provisioning_orchestrator.go    provisioning_emit.go
                                                 provisioning_state_machine.go
                                                 ─── admin/ ───
                                                 server.go / helpers.go
                                                 ─── server/ ───
                                                 worker_server.go (DI root)
```

**Models & tables:**
- `cdc_activity_log` → audit log mọi action operator

**⚠️ Thiếu repo**: `ActivityLog` write inline → cần repo riêng.

---

## Bảng tổng hợp: 18 Models + 11 Repos → Domain

### Models (18 files)

| Model | DB Table | Domain |
|---|---|---|
| `connection_registry` | `cdc_system.connection_registry` | **source** |
| `source_object_registry` | `cdc_system.source_object_registry` | **source** |
| `table_registry` ⚠️ | `cdc_system.cdc_table_registry` | **source** (legacy, sẽ bỏ) |
| `schema_change_log` | `cdc_system.schema_changes_log` | **source** |
| `shadow_binding` | `cdc_system.shadow_binding` | **ingestion** |
| `cdc_event` | (in-memory) | **ingestion** |
| `mapping_rule_v2` | `cdc_system.mapping_rule_v2` | **schema** |
| `mapping_rule` ⚠️ | `cdc_system.cdc_mapping_rules` | **schema** (V1 deprecated) |
| `pending_field` | `cdc_system.pending_fields` | **schema** |
| `sensitive_field` | `cdc_system.sensitive_fields` | **schema** |
| `snapshot_dlq` | `cdc_system.snapshot_dlq` | **discovery** |
| `master_binding` | `cdc_system.master_binding` | **transmute** |
| `transmute_schedule` | `cdc_system.transmute_schedule` | **transmute** |
| `sync_runtime_state` | `cdc_system.sync_runtime_state` | **transmute** |
| `worker_schedule` | `cdc_system.cdc_worker_schedule` | **transmute** |
| `reconciliation_report` | `cdc_system.cdc_reconciliation_report` | **recon** |
| `failed_sync_log` | `cdc_system.failed_sync_logs` | **recon** |
| `activity_log` | `cdc_system.cdc_activity_log` | **platform** |

### Repositories (11 files hiện có + 3 cần tạo)

| Repository | Domain | Ghi chú |
|---|---|---|
| `connection_registry_repo` | **source** | ✅ Có |
| `source_object_registry_repo` | **source** | ✅ Có |
| `registry_repo` | **source** | ✅ Có (legacy) |
| `schema_log_repo` | **source** | ✅ Có |
| `shadow_binding_repo` | **ingestion** | ✅ Có |
| `mapping_rule_v2_repo` | **schema** | ✅ Có |
| `mapping_rule_repo` ⚠️ | **schema** | ✅ Có (V1 deprecated) |
| `pending_field_repo` | **schema** | ✅ Có |
| `master_binding_repo` | **transmute** | ✅ Có |
| `transmute_schedule_repo` | **transmute** | ✅ Có |
| `sync_runtime_state_repo` | **transmute** | ✅ Có |
| `reconciliation_report_repo` | **recon** | ❌ **Chưa có — cần tạo** |
| `failed_sync_log_repo` | **recon** | ❌ **Chưa có — cần tạo** |
| `activity_log_repo` | **platform** | ❌ **Chưa có — cần tạo** |

---

## Kiến trúc thư mục đề xuất

```
internal/
├── source/
│   ├── model.go                    ← ConnectionRegistry, SourceObjectRegistry, SchemaChangeLog
│   ├── repository.go               ← Port interface
│   ├── service/
│   │   ├── metadata_registry.go
│   │   ├── connection_manager.go
│   │   └── connection_overrides.go
│   └── handler/
│       └── sync_handler.go         ← HandleSyncRegister, HandleSyncState, HandleRestartDebezium
│
├── schema/
│   ├── model.go                    ← MappingRuleV2, PendingField, SensitiveField
│   ├── repository.go
│   ├── service/
│   │   ├── master_ddl_generator.go
│   │   ├── schema_inspector.go
│   │   └── type_resolver.go
│   └── handler/
│       ├── ddl_handler.go          ← HandleStandardize, HandleCreateDefaultColumns, HandleAlterColumn
│       └── master_ddl_handler.go
│
├── ingestion/
│   ├── model.go                    ← ShadowBinding, CDCEvent
│   ├── repository.go
│   ├── service/
│   │   ├── schema_adapter.go
│   │   ├── dynamic_mapper.go
│   │   └── child_explode.go
│   ├── handler/
│   │   ├── kafka_consumer.go
│   │   ├── event_handler.go
│   │   └── batch_buffer.go
│   └── sinkworker/
│
├── discovery/
│   ├── model.go                    ← SnapshotDLQ
│   ├── repository.go
│   ├── service/
│   │   ├── mongo_introspection.go
│   │   ├── timestamp_detector.go
│   │   └── backfill_source_ts.go
│   └── handler/
│       ├── discover_handler.go     ← HandleDiscover, HandleScanFields
│       ├── scan_handler.go         ← HandleScanRawData, HandleScanArrayFields
│       └── snapshot_runner.go
│
├── transmute/
│   ├── model.go                    ← MasterBinding, TransmuteSchedule, SyncRuntimeState
│   ├── repository.go
│   ├── service/
│   │   ├── transmuter.go
│   │   ├── transmute_scheduler.go
│   │   └── job_monitor.go
│   └── handler/
│       ├── transmute_handler.go
│       └── batch_transform_handler.go
│
├── recon/
│   ├── model.go                    ← ReconciliationReport, FailedSyncLog
│   ├── repository.go               ← (cần tạo mới)
│   ├── service/
│   │   ├── recon_core.go           → tách: tier1.go, tier2.go, segment_b.go
│   │   ├── recon_source_agent.go
│   │   ├── recon_dest_agent.go
│   │   ├── recon_heal.go
│   │   └── debezium_signal.go
│   └── handler/
│       ├── recon_handler.go
│       └── recon_heal_v4.go
│
└── platform/
    ├── model.go                    ← ActivityLog
    ├── repository.go               ← (cần tạo mới)
    ├── masking/
    │   └── masking_service.go
    ├── dlq/
    │   ├── dlq_handler.go
    │   └── dlq_worker.go
    ├── provisioning/
    │   └── step_handlers.go
    └── server/
        └── worker_server.go        ← DI root
```
