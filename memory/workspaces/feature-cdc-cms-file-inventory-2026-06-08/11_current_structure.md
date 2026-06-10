# 11_current_structure — Cấu trúc & chức năng từng file (cdc-cms-service)

> Sinh 2026-06-08 từ phân tích thật (213 file non-test, go build PASS, deadcode 75 hàm).
> Cột Status: USED · ⚠DEAD? (file/symbol thừa) · ⚠PARTIAL (file dùng nhưng có hàm chết).

## Cây thư mục + LOC/file mỗi package
```
    76 LOC    1 file  cmd/server
    40 LOC    1 file  cmd/sync_v2
   218 LOC    1 file  config
  2280 LOC    1 file  docs
  6874 LOC   52 file  internal/api
   163 LOC    2 file  internal/api/dto
  3846 LOC   32 file  internal/app/commands
   275 LOC    4 file  internal/app/ports
  2128 LOC   30 file  internal/app/queries
   367 LOC    3 file  internal/bootstrap
    51 LOC    1 file  internal/domain/job
   177 LOC    3 file  internal/domain/mapping
    45 LOC    1 file  internal/domain/master
    96 LOC    2 file  internal/domain/reconciliation
    54 LOC    1 file  internal/domain/source
     6 LOC    1 file  internal/infra/cache
   618 LOC    3 file  internal/infra/http
   460 LOC    3 file  internal/infra/messaging
   794 LOC    4 file  internal/infra/observability
   525 LOC    8 file  internal/infra/observability/probes
  5117 LOC   27 file  internal/infra/persistence
   991 LOC    6 file  internal/middleware
   208 LOC    1 file  internal/migrate
   407 LOC   14 file  internal/model
    36 LOC    1 file  internal/naming
   452 LOC    1 file  internal/router
   343 LOC    1 file  internal/server
     9 LOC    1 file  migrations
    80 LOC    1 file  pkgs/database
   101 LOC    1 file  pkgs/natsconn
   121 LOC    1 file  pkgs/observability
   108 LOC    1 file  pkgs/rediscache
    91 LOC    3 file  pkgs/utils
```

## Area: api

# Inventory: AREA = api (internal/api/ + internal/api/dto/)

| File | LOC | Chức năng | Symbol chính | Status |
|---|---|---|---|---|
| internal/api/action_trace.go | 41 | Utility: chuẩn hoá X-Correlation-Id / X-CDC-Action / X-CDC-Origin từ header hoặc body vào struct `actionTrace`; dùng bởi reconciliation_handler_tools.go | `normalizeActionTrace` | USED (gọi tại reconciliation_handler_tools.go:TriggerSnapshot) |
| internal/api/activity_log_handler.go | 101 | HTTP GET /api/activity-log (list) + /api/activity-log/stats; delegate sang CQRS query handlers | `ActivityLogHandler`, `NewActivityLogHandler` | USED (wired: dualGet shared /activity-log, /activity-log/stats) |
| internal/api/alerts_handler.go | 201 | HTTP GET /alerts/active, /silenced, /history; POST /alerts/:fingerprint/ack, /silence — Alert state machine | `AlertsHandler`, `NewAlertsHandler` | USED (wired: dualGet shared /alerts/*, registerDestructive /alerts/*/ack|silence) |
| internal/api/audit_handler.go | 48 | HTTP GET /audit/qa-summary, /audit/gaps, /audit/metric-health — bảng QA dashboard | `AuditHandler`, `NewAuditHandler` | USED (wired: dualGet admin /audit/*) |
| internal/api/dto/audit_dto.go | 51 | DTO types: QASummaryResponse, GapRow, MetricHealthResponse, CriterionRow, GapCount — dùng bởi internal/app/queries | `QASummaryResponse`, `GapRow`, `MetricHealthResponse` | USED (referenced trong get_qa_summary.go, list_gaps.go, get_metric_health.go) |
| internal/api/dto/mapping_rule_dto.go | 112 | DTO types: MappingRuleRow, MappingRuleCreateRequest, MappingRuleBatchUpdateRequest, RuleToRow — wire shape cho /mapping-rules API | `MappingRuleRow`, `RuleToRow`, `MappingRuleCreateRequest` | USED (referenced trong mapping_rule_handler_list.go, _create.go, _batch.go) |
| internal/api/health_handler.go | 41 | HTTP GET /health (liveness) + GET /ready (readiness + DB ping) | `HealthHandler`, `NewHealthHandler` | USED (wired: app.Get /health, /ready) |
| internal/api/introspection_handler.go | 526 | HTTP introspection: discover Mongo databases/collections (GET + POST variants), scan shadow table, scan-raw, scan-array, shadow-columns — NATS request-reply | `IntrospectionHandler`, `NewIntrospectionHandler` | USED (wired: dualGet/dualPost shared /introspection/*) |
| internal/api/job_handler.go | 49 | HTTP GET /api/jobs/:id — đọc trạng thái async job từ cdc_system.cdc_jobs | `JobHandler`, `NewJobHandler` | USED (wired: dualGet shared /jobs/:id, gated nil-check) |
| internal/api/mapping_preview_handler.go | 114 | HTTP POST /api/v1/mapping-rules/preview — preview JsonPath rule trên live shadow rows (read-only eval) | `MappingPreviewHandler`, `NewMappingPreviewHandler` | USED (wired: destructive POST /v1/mapping-rules/preview) |
| internal/api/mapping_rule_handler.go | 19 | Struct + constructor MappingRuleHandler — root file cho handler split | `MappingRuleHandler`, `NewMappingRuleHandler` | USED (constructor gọi trong server.go) |
| internal/api/mapping_rule_handler_batch.go | 69 | Backfill (POST /mapping-rules/:id/backfill) + BatchUpdate (PATCH /mapping-rules/batch) | `MappingRuleHandler.Backfill`, `MappingRuleHandler.BatchUpdate` | USED (wired: dualPost/dualPatch admin) |
| internal/api/mapping_rule_handler_commands.go | 75 | Reload (POST /mapping-rules/reload) + UpdateStatus (PATCH /mapping-rules/:id) + helper `ptr` | `MappingRuleHandler.Reload`, `MappingRuleHandler.UpdateStatus` | USED (wired: dualPost/dualPatch admin) |
| internal/api/mapping_rule_handler_create.go | 63 | Create (POST /mapping-rules) — tạo mapping rule V2 qua CommandBus | `MappingRuleHandler.Create` | USED (wired: dualPost admin /mapping-rules) |
| internal/api/mapping_rule_handler_list.go | 63 | List (GET /mapping-rules) — phân trang + filter mapping rules V2 | `MappingRuleHandler.List` | USED (wired: dualGet shared /mapping-rules) |
| internal/api/master_mapping_rule_handler.go | 291 | HTTP CRUD cho master mapping rules: List, Save, Delete, SyncFromShadow, BatchUpdate; trigger master DDL qua NATS | `MasterMappingRuleHandler`, `NewMasterMappingRuleHandler` | USED (wired: GET /v1/master-mapping-rules shared; POST/DELETE/PUT admin) |
| internal/api/master_registry_handler.go | 55 | Struct + constructor MasterRegistryHandler + shared types CreateRequest, ApproveRequest, SwapRequest + regex masterNameRe | `MasterRegistryHandler`, `NewMasterRegistryHandler` | USED (constructor gọi trong server.go) |
| internal/api/master_registry_handler_approve.go | 97 | Approve + Reject master — gửi command qua bus, returns 202 + JobID | `MasterRegistryHandler.Approve`, `MasterRegistryHandler.Reject` | USED (wired: registerDestructive /v1/masters/:name/approve|reject) |
| internal/api/master_registry_handler_create.go | 94 | Create master — validate + dispatch CreateMasterCommand qua bus | `MasterRegistryHandler.Create` | USED (wired: registerDestructive /v1/masters) |
| internal/api/master_registry_handler_read.go | 31 | List (GET /v1/masters) — alias type MasterRow + delegate listQ handler | `MasterRegistryHandler.List`, `MasterRow` | USED (wired: shared.Get /v1/masters) |
| internal/api/master_registry_handler_resolve.go | 59 | Private helper resolveMasterBindingByName + getActor + trimCreateRequest + `trimString` (dead) | `resolveMasterBindingByName`, `getActor`, `trimCreateRequest` | ⚠PARTIAL (`trimString` unreachable — định nghĩa tại :36 nhưng 0 callers trong codebase) |
| internal/api/master_registry_handler_swap.go | 73 | Swap (POST /v1/masters/:name/swap) — atomic master table swap qua Dispatch | `MasterRegistryHandler.Swap` | USED (wired: registerDestructive /v1/masters/:name/swap) |
| internal/api/master_registry_handler_toggle.go | 52 | ToggleActive (POST /v1/masters/:name/toggle-active) — flip is_active qua bus | `MasterRegistryHandler.ToggleActive` | USED (wired: registerDestructive /v1/masters/:name/toggle-active) |
| internal/api/provisioning_handler.go | 234 | Source Provisioning Mode: GetState, Advance, Pause, Resume, Retry, Archive, SetMode; 3 hàm ForTest chỉ dùng trong test/ | `ProvisioningHandler`, `NewProvisioningHandler` | ⚠PARTIAL (`NewProvisioningHandlerForTest`, `MapErrForTest`, `ParseSourceIDForTest` — unreachable ngoài test/ dir) |
| internal/api/reconciliation_handler.go | 78 | Struct + constructor ReconciliationHandler + alias types ReportRow, FailedLogRow + helper resolveTargetTable | `ReconciliationHandler`, `NewReconciliationHandler` | USED (constructor + resolveTargetTable gọi nội bộ bởi các recon sub-handlers) |
| internal/api/reconciliation_handler_backfill.go | 98 | TriggerBackfillSourceTs (POST /recon/backfill-source-ts) + BackfillSourceTsStatus (GET /recon/backfill-source-ts/status) | `ReconciliationHandler.TriggerBackfillSourceTs`, `BackfillSourceTsStatus` | USED (wired: registerDestructive + dualGet shared) |
| internal/api/reconciliation_handler_commands.go | 82 | TriggerCheck (POST /reconciliation/check/:table) + TriggerCheckAll (POST /reconciliation/check) — dispatch NATS recon-check | `ReconciliationHandler.TriggerCheck`, `TriggerCheckAll` | USED (wired: registerDestructive /reconciliation/check) |
| internal/api/reconciliation_handler_heal.go | 47 | TriggerHeal (POST /reconciliation/heal[/:table]) — dispatch recon-heal qua NATS | `ReconciliationHandler.TriggerHeal` | USED (wired: registerDestructive /reconciliation/heal, /heal/:table) |
| internal/api/reconciliation_handler_reports.go | 71 | LatestReport (GET /reconciliation/report) + TableHistory (GET /reconciliation/report/:table) + ListFailedLogs (GET /failed-sync-logs) | `ReconciliationHandler.LatestReport`, `TableHistory`, `ListFailedLogs` | USED (wired: dualGet shared) |
| internal/api/reconciliation_handler_retry.go | 60 | RetryFailedLog (POST /failed-sync-logs/:id/retry) — re-dispatch failed sync record | `ReconciliationHandler.RetryFailedLog` | USED (wired: registerDestructive /failed-sync-logs/:id/retry) |
| internal/api/reconciliation_handler_tools.go | 77 | ResetDebeziumOffset (POST /tools/reset-debezium-offset) + TriggerSnapshot (POST /tools/trigger-snapshot/:table) | `ReconciliationHandler.ResetDebeziumOffset`, `TriggerSnapshot` | USED (wired: registerDestructive /tools/*) |
| internal/api/registry_handler.go | 50 | Struct + constructor RegistryHandler — root file cho registry handler split | `RegistryHandler`, `NewRegistryHandler` | USED (constructor gọi trong server.go) |
| internal/api/registry_handler_bulk.go | 86 | BulkRegister (POST /v1/source-objects/register-batch) — đăng ký hàng loạt + dispatch create-default-columns per entry | `RegistryHandler.BulkRegister` | USED (wired: admin.Post /v1/source-objects/register-batch) |
| internal/api/registry_handler_dispatch.go | 64 | DispatchStatus (GET /v1/source-objects/registry/:id/dispatch-status) + DetectTimestampField (POST /registry/:id/detect-timestamp-field) | `RegistryHandler.DispatchStatus`, `DetectTimestampField` | USED (wired: shared.Get + admin.Post) |
| internal/api/registry_handler_read.go | 58 | List (legacy GET registry), GetStats (legacy stats), SyncHealth (GET /sync/health) | `RegistryHandler.List`, `RegistryHandler.GetStats`, `RegistryHandler.SyncHealth` | ⚠PARTIAL (`RegistryHandler.List` và `RegistryHandler.GetStats` — 0 wire trong router.go; chỉ SyncHealth được mount tại /sync/health) |
| internal/api/registry_handler_register.go | 98 | Register (POST /v1/source-objects/register) — tạo entry + dispatch create-default-columns + optional Debezium restart | `RegistryHandler.Register` | USED (wired: admin.Post /v1/source-objects/register) |
| internal/api/registry_handler_tools_columns.go | 92 | CreateDefaultColumns (POST /registry/:id/create-default-columns) + DetectTimestampField (POST /registry/:id/detect-timestamp-field) | `RegistryHandler.CreateDefaultColumns`, `RegistryHandler.DetectTimestampField` | USED (wired: admin.Post) |
| internal/api/registry_handler_tools_scan.go | 89 | Standardize (POST /registry/:id/standardize) + ScanFields (POST /registry/:id/scan-fields) | `RegistryHandler.Standardize`, `RegistryHandler.ScanFields` | USED (wired: admin.Post) |
| internal/api/registry_handler_transform.go | 68 | Transform (POST /registry/:id/transform) + TransformStatus (GET /registry/:id/transform-status) | `RegistryHandler.Transform`, `RegistryHandler.TransformStatus` | USED (wired: admin.Post + shared.Get) |
| internal/api/registry_handler_update.go | 87 | Update (PATCH /v1/source-objects/registry/:id) — cập nhật config registry + optional Debezium restart | `RegistryHandler.Update` | USED (wired: admin.Patch /v1/source-objects/registry/:id) |
| internal/api/schedule_handler.go | 329 | Worker schedule: List (GET /worker-schedule), Create (POST /worker-schedule), Update (PATCH /worker-schedule/:id) | `ScheduleHandler`, `NewScheduleHandler` | USED (wired: dualGet shared + dualPost/dualPatch admin /worker-schedule) |
| internal/api/schema_change_handler.go | 166 | Schema change approval flow: GetPending (GET /schema-changes/pending), GetHistory, Approve, Reject | `SchemaChangeHandler`, `NewSchemaChangeHandler` | USED (wired: dualGet shared + dualPost admin /schema-changes/*) |
| internal/api/schema_proposal_handler.go | 192 | Schema proposal V2: List, Get, Approve, Reject — Sprint 5 §R9 | `SchemaProposalHandler`, `NewSchemaProposalHandler`, `propIdentRe` | USED (wired: shared.Get + registerDestructive /v1/schema-proposals/*) |
| internal/api/sensitive_fields_handler.go | 93 | Sensitive fields CRUD: List, Create, UpdateStrategy, Delete — global keyword list | `SensitiveFieldsHandler`, `NewSensitiveFieldsHandler` | USED (wired: shared.Get + admin.Post/Patch/Delete /v1/sensitive-fields) |
| internal/api/shadow_binding_actions_handler.go | 88 | PatchActive (PATCH /v1/shadow-bindings/:id) — toggle is_active trên single shadow_binding | `ShadowBindingActionsHandler`, `NewShadowBindingActionsHandler` | USED (wired: admin.Patch /v1/shadow-bindings/:id) |
| internal/api/snapshot_progress_handler.go | 77 | List (GET /snapshot-progress), Pause + Resume (POST /snapshot-progress/:id/pause|resume) — NATS control signals | `SnapshotProgressHandler`, `NewSnapshotProgressHandler` | USED (wired: dualGet shared + registerDestructive /v1/snapshot-progress/:id/pause|resume) |
| internal/api/source_object_actions_handler.go | 714 | V2 source object actions: UpdateV2, ScanFieldsV2, StandardizeV2, CreateDefaultColumnsV2, DetectTimestampFieldV2, SnapshotV2, DispatchStatusV2, TransformStatusV2; có 2 private helpers bị dead | `SourceObjectActionsHandler`, `NewSourceObjectActionsHandler` | ⚠PARTIAL (`resolveDispatchScopeBySourceObjectID` :47 và `resolveReadScopeBySourceObjectID` :80 — defined nhưng chưa gọi trực tiếp; chỉ `resolveDispatchScope`/`resolveReadScope` được dùng) |
| internal/api/source_objects_handler.go | 396 | V2 read model: List (GET /v1/source-objects), GetStats, GetMappingContext, ListShadowBindings | `SourceObjectsHandler`, `NewSourceObjectsHandler` | USED (wired: shared.Get /v1/source-objects/*, /v1/shadow-bindings) |
| internal/api/sources_handler.go | 85 | Connections/Sources CRUD: List, Create, Get — Connection Fingerprint registry | `SourcesHandler`, `NewSourcesHandler` | USED (wired: shared.Get/Post + backward-compat /v1/registry/connections) |
| internal/api/system_connectors_handler.go | 482 | Kafka Connect REST proxy: List, Get, Plugins (reads) + Create, UpdateConfig, Restart, RestartTask, Pause, Resume, Delete (writes) | `SystemConnectorsHandler`, `NewSystemConnectorsHandler` | USED (wired: shared.Get + registerDestructive /v1/system/connectors/*) |
| internal/api/system_health_handler.go | 158 | Health: Health (GET /api/v1/system/health) từ Redis cache + RestartDebezium (POST /tools/restart-debezium) | `SystemHealthHandler`, `NewSystemHealthHandler` | USED (wired: app.Get /api/system/health + registerDestructiveRestart /tools/restart-debezium) |
| internal/api/transmute_schedule_handler.go | 264 | Transmute schedules: List, Create, Toggle, Delete, RunNow + shared types PreviewRequest/PreviewResult | `TransmuteScheduleHandler`, `NewTransmuteScheduleHandler` | USED (wired: shared.Get + registerDestructive/Patch/Delete /v1/schedules/*) |
| internal/api/utils.go | 56 | Package-internal utils: intQuery, normalizeShadowIdent (dead), isValidTimestampField | `intQuery`, `isValidTimestampField` | ⚠PARTIAL (`normalizeShadowIdent` :22 unreachable — 0 callers; `intQuery` và `isValidTimestampField` USED) |
| internal/api/wizard_handler.go | 173 | Wizard state machine: Create, Patch, Execute (writes) + Get, Progress (reads) | `WizardHandler`, `NewWizardHandler` | USED (wired: admin.Post/Patch + shared.Get + registerDestructive /v1/wizard/sessions/*) |

## Area: commands

# AREA: commands — Inventory Table

Generated from grep analysis of `cdc-cms-service` repo.
- **USED** = handler constructor called in `server.go` RegisterSync/RegisterSubject, OR command struct dispatched via `bus.Dispatch` in API handlers.
- **⚠DEAD?** = 0 external references + not registered + confirmed in deadcode.txt.
- **⚠PARTIAL** = file is USED overall but contains ≥1 function listed in deadcode.txt.
- **raw gorm** = file holds `*gorm.DB` field on handler struct (direct DB access, bypassing repository abstraction).

## internal/app/commands/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/app/commands/doc.go` | 8 | Package-level docstring; mô tả pattern CQRS C-side | — | USED (package doc) |
| `internal/app/commands/ack_alert.go` | 74 | Sync handler: ACK alert qua `AlertManager` | `AckAlertHandler`, `AckAlertCommand` | **USED** – `RegisterSync("alert.ack", ...)` server.go:229 |
| `internal/app/commands/silence_alert.go` | 62 | Sync handler: Silence (mute) alert có deadline | `SilenceAlertHandler`, `SilenceAlertCommand` | **USED** – `RegisterSync("alert.silence", ...)` server.go:230 |
| `internal/app/commands/approve_master.go` | 165 | Sync handler: approve master binding, TX gorm, publish NATS reload (raw gorm) | `ApproveMasterHandler`, `ApproveMasterCommand` | **USED** – `RegisterSync("master.approve", ...)` server.go:235 |
| `internal/app/commands/reject_master.go` | 103 | Sync handler: reject master binding với note (raw gorm) | `RejectMasterHandler`, `RejectMasterCommand` | **USED** – `RegisterSync("master.reject", ...)` server.go:233 |
| `internal/app/commands/toggle_master_active.go` | 74 | Sync handler: flip `is_active` trên `master_binding` row (raw gorm) | `ToggleMasterActiveHandler`, `ToggleMasterActiveCommand` | **USED** – `RegisterSync("master.toggle-active", ...)` server.go:249 |
| `internal/app/commands/create_master.go` | 324 | Sync handler: tạo master_binding + resolve connection/shadow (raw gorm) | `CreateMasterHandler`, `CreateMasterCommand` | **USED** – `RegisterSync("master.create", ...)` server.go:234 |
| `internal/app/commands/master_swap.go` | 38 | Async Command struct: RENAME-swap master table qua NATS worker | `MasterSwapCommand` | **USED** – `RegisterSubject("master.swap", ...)` server.go:170; dispatched `master_registry_handler_swap.go:50` |
| `internal/app/commands/approve_schema_proposal.go` | 204 | Sync handler: apply schema proposal, update columns, TX gorm (raw gorm) | `ApproveSchemaProposalHandler`, `ApproveSchemaProposalCommand` | **USED** – `RegisterSync("schema-proposal.approve", ...)` server.go:252 |
| `internal/app/commands/reject_schema_proposal.go` | 75 | Sync handler: stamp schema_proposal rejected với CAS guard (raw gorm) | `RejectSchemaProposalHandler`, `RejectSchemaProposalCommand` | **USED** – `RegisterSync("schema-proposal.reject", ...)` server.go:251 |
| `internal/app/commands/create_mapping_rule.go` | 370 | Sync handler: tạo mapping rule V2 + resolve scope (raw gorm) | `CreateMappingRuleHandler`, `CreateMappingRuleCommand` | **USED** – `RegisterSync("mapping.create", ...)` server.go:232 |
| `internal/app/commands/update_mapping_rule.go` | 200 | Sync handler: cập nhật status mapping rule + publish NATS reload (raw gorm) | `UpdateMappingRuleHandler`, `UpdateMappingRuleCommand` | **USED** – `RegisterSync("mapping.update-status", ...)` server.go:231 |
| `internal/app/commands/register_registry.go` | 187 | Sync handler: register TableRegistry, TX gorm, v2 sync, publish reload (raw gorm) | `RegisterRegistryHandler`, `RegisterRegistryCommand`, `NormalizePKTypeForTest` | **USED** – `RegisterSync("registry.register", ...)` server.go:246. **⚠PARTIAL**: `NormalizePKTypeForTest` dead (deadcode.txt line 186) |
| `internal/app/commands/bulk_register_registry.go` | 105 | Sync handler: bulk register nhiều table registry entries (raw gorm) | `BulkRegisterRegistryHandler`, `BulkRegisterRegistryCommand` | **USED** – `RegisterSync("registry.bulk-register", ...)` server.go:247 |
| `internal/app/commands/update_registry.go` | 177 | Sync handler: cập nhật registry metadata + publish NATS reload (raw gorm) | `UpdateRegistryHandler`, `UpdateRegistryCommand` | **USED** – `RegisterSync("registry.update", ...)` server.go:243 |
| `internal/app/commands/update_shadow_binding.go` | 86 | Sync handler: toggle `is_active` / metadata trên shadow_binding (raw gorm) | `UpdateShadowBindingHandler`, `UpdateShadowBindingCommand` | **USED** – `RegisterSync("shadow-binding.update", ...)` server.go:240 |
| `internal/app/commands/update_source_object_v2.go` | 169 | Sync handler: write allow-listed metadata fields V2 source object (raw gorm) | `UpdateSourceObjectV2Handler`, `UpdateSourceObjectV2Command` | **USED** – `RegisterSync("source.update-v2", ...)` server.go:239 |
| `internal/app/commands/v2_sync.go` | 59 | Sync handler: mirror V1 TableRegistry → V2 source_object_registry + shadow_binding | `V2SyncHandler`, `V2SyncCommand` | **USED** – `RegisterSync("source.v2-sync", ...)` server.go:248 |
| `internal/app/commands/create_schedule.go` | 90 | Sync handler: upsert transmute_schedule row (raw gorm) | `CreateTransmuteScheduleHandler`, `CreateTransmuteScheduleCommand` | **USED** – `RegisterSync("schedule.create", ...)` server.go:244 |
| `internal/app/commands/toggle_schedule.go` | 67 | Sync handler: flip `is_enabled` trên transmute_schedule row (raw gorm) | `ToggleTransmuteScheduleHandler`, `ToggleTransmuteScheduleCommand` | **USED** – `RegisterSync("schedule.toggle", ...)` server.go:245 |
| `internal/app/commands/update_schedule.go` | 115 | Sync handler: cập nhật transmute_schedule metadata (raw gorm) | `UpdateScheduleHandler`, `UpdateScheduleCommand` | **USED** – `RegisterSync("schedule.update", ...)` server.go:241 |
| `internal/app/commands/create_worker_schedule.go` | 69 | Sync handler: insert row vào cdc_worker_schedule (raw gorm) | `CreateWorkerScheduleHandler`, `CreateWorkerScheduleCommand` | **USED** – `RegisterSync("worker-schedule.create", ...)` server.go:250 |
| `internal/app/commands/mark_failed_log_retrying.go` | 66 | Sync handler: stamp failed_sync_logs row "retrying" (raw gorm) | `MarkFailedLogRetryingHandler`, `MarkFailedLogRetryingCommand` | **USED** – `RegisterSync("recon.failed-log-mark-retrying", ...)` server.go:242 |
| `internal/app/commands/create_wizard.go` | 65 | Sync handler: tạo draft wizard session qua WizardRepo | `CreateWizardHandler`, `CreateWizardCommand` | **USED** – `RegisterSync("wizard.create", ...)` server.go:236 |
| `internal/app/commands/patch_wizard.go` | 100 | Sync handler: update allow-listed fields trên wizard session | `PatchWizardHandler`, `PatchWizardCommand` | **USED** – `RegisterSync("wizard.patch", ...)` server.go:237 |
| `internal/app/commands/wizard_execute.go` | 77 | Sync handler: flip wizard session → running + stamp progress | `WizardExecuteHandler`, `WizardExecuteCommand` | **USED** – `RegisterSync("wizard.execute", ...)` server.go:238 |
| `internal/app/commands/system_connector.go` | 298 | Sync handlers (4): Create/Delete/UpdateConfig/Lifecycle Kafka connector | `CreateSystemConnectorHandler`, `DeleteSystemConnectorHandler`, `UpdateSystemConnectorConfigHandler`, `LifecycleSystemConnectorHandler` | **USED** – `RegisterSync("system-connector.*", ...)` server.go:253-256 |
| `internal/app/commands/recon_check.go` | 43 | Async Command struct only: NATS recon-check payload | `ReconCheckCommand` | **USED** – `RegisterSubject("recon.check", ...)` server.go:155; dispatched `reconciliation_handler_commands.go` |
| `internal/app/commands/recon_async.go` | 146 | Async Command structs: ReconHeal, RetryFailed, DebeziumSignal, DebeziumSnapshot, SnapshotV2, ReconBackfillSourceTs | 6 Command structs | **USED** – tất cả 6 types đều có RegisterSubject tương ứng server.go:156-160, 171 VÀ dispatched trong API handlers |
| `internal/app/commands/source_async.go` | 171 | Async Command structs: CreateDefaultColumns, Standardize, ScanFields, DetectTimestampField, AlterColumn, Backfill | 6 Command structs | **USED** – RegisterSubject server.go:162-167; dispatched rộng rãi từ registry_handler_*, source_object_actions_handler, mapping_rule_handler_batch |
| `internal/app/commands/system_async.go` | 27 | Async Command struct: RestartDebezium | `RestartDebeziumCommand` | **USED** – `RegisterSubject("debezium.restart", ...)` server.go:161; dispatched `registry_handler_update.go:78`, `registry_handler_register.go:84`, `system_health_handler.go:144` |
| `internal/app/commands/transmute_run.go` | 32 | Async Command struct: fire immediate transmute | `TransmuteRunCommand` | **USED** – `RegisterSubject("transmute.run", ...)` server.go:168; dispatched `transmute_schedule_handler.go:218` |

## internal/app/ports/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/app/ports/command_bus.go` | 88 | Interfaces: Command, SyncCommand, AsyncCommand, CommandBus; Mixins: SyncCommandMixin, AsyncCommandMixin; SyncResult, AsyncResult | `CommandBus`, `SyncCommandMixin`, `AsyncCommandMixin` | **USED** – CommandBus implemented bởi `nats_command_bus.go`, mixins embedded bởi mọi Command struct. **⚠PARTIAL**: `SyncCommandMixin.syncCommandKind` + `AsyncCommandMixin.asyncCommandKind` dead (private marker methods, deadcode.txt) |
| `internal/app/ports/query_bus.go` | 16 | Interfaces: Query, QueryBus | `Query`, `QueryBus` | **⚠DEAD?** – Query.Type() methods in queries/ package toàn bộ bị deadcode.txt (unreachable); QueryBus interface không có implementation hoặc call site nào ngoài file định nghĩa. Queries được inject trực tiếp (handler functions), không qua QueryBus.Ask() |
| `internal/app/ports/publisher.go` | 10 | Interface: Publisher (fire-and-forget NATS publish) | `Publisher` | **⚠DEAD?** – Không có struct nào implement ports.Publisher; không có call site nào dùng `ports.Publisher`. Comment nói "legacy call-sites publish directly via this port until migrated" nhưng thực tế dùng `natsconn.NatsClient.PublishReload` trực tiếp, không qua interface này |
| `internal/app/ports/repository.go` | 161 | Port interfaces: MappingRuleRepo, SourceRepo, MasterRepo, JobRepo, ReconReportRepo, FailedSyncLogRepo, SchemaLogRepo, PendingFieldRepo, WizardRepo, SystemConnectorRepo, SensitiveFieldRepo, RegistryRepo | 12 interfaces | **USED** – 22+ references qua `ports.*Repo` across server.go, commands, infra/persistence |

---

## Summary

- **Total files analyzed**: 32 commands + 4 ports = 36 files
- **⚠DEAD?**: `internal/app/ports/publisher.go`, `internal/app/ports/query_bus.go`
- **⚠PARTIAL**: `internal/app/commands/register_registry.go` (`NormalizePKTypeForTest`), `internal/app/ports/command_bus.go` (`syncCommandKind`, `asyncCommandKind`)
- **Raw-gorm handlers** (18 files): approve_master, approve_schema_proposal, bulk_register_registry, create_mapping_rule, create_master, create_schedule, create_worker_schedule, mark_failed_log_retrying, register_registry, reject_master, reject_schema_proposal, toggle_master_active, toggle_schedule, update_mapping_rule, update_registry, update_schedule, update_shadow_binding, update_source_object_v2

## Area: queries

# Inventory — internal/app/queries/ (non-test .go files)

> Phân tích ngày 2026-06-08. Deadcode source: /tmp/deadcode.txt.
> STATUS legend:
> - **USED** = symbol được instantiate/ref từ server.go, api handler, hoặc infra adapter.
> - **⚠DEAD?** = 0 ref ngoài package + không register + xuất hiện trong deadcode.txt.
> - **⚠PARTIAL** = file USED nhưng có ≥1 hàm con bị deadcode report.

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|-------------|--------|
| `internal/app/queries/doc.go` | 6 | Package doc comment — không có symbol | — | USED (boilerplate) |
| `internal/app/queries/activity_log_read_models.go` | 46 | Read model projection: `ActivityLogRow` (per-row activity log với V2 scope) + `OpStat` (24h bucket aggregate). Dùng bởi GET /api/activity-log và /api/activity-log/stats | `ActivityLogRow`, `OpStat` | USED — ref từ `activity_log_read_repo_gorm.go`, `activity_log_handler.go` (type alias) |
| `internal/app/queries/bridge_status_reader.go` | 92 | Read port `BridgeStatusReader` cho transform-status probe + dispatch-scope resolution (source_object_actions). Định nghĩa `BridgeStatusProbe`, `DispatchScope`, sentinel errors `ErrAmbiguousDispatchScope`, `ErrSourceObjectNoActiveShadow` | `BridgeStatusReader`, `DispatchScope`, `ErrAmbiguousDispatchScope`, `ErrSourceObjectNoActiveShadow` | USED — adapter `bridge_status_repo_gorm.go` + `registry_handler.go` + `source_object_actions_handler.go` dùng rộng rãi |
| `internal/app/queries/get_activity_stats.go` | 39 | Query + Handler cho GET /api/activity-log/stats. Trả `GetActivityStatsResult{Stats24h, RecentErrors}` | `GetActivityStatsQuery`, `GetActivityStatsHandler`, `NewGetActivityStatsHandler` | **⚠PARTIAL** — Handler USED (`server.go:133`). `GetActivityStatsQuery.Type()` dead (deadcode.txt) — method không được query bus gọi, chỉ là interface marker |
| `internal/app/queries/get_job.go` | 86 | Query + Handler cho GET /api/jobs/:id. Định nghĩa `JobReader` port + `JobView` projection | `GetJobQuery`, `GetJobHandler`, `NewGetJobHandler`, `JobReader`, `JobView` | **⚠PARTIAL** — Handler USED (`server.go:151`, `job_handler.go:41`). `GetJobQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/get_metric_health.go` | 90 | Handler cho GET /api/audit/metric-health. Query Prometheus (consumer_lag, e2e_latency_p99, dlq_rate, recon_drift) → `dto.MetricHealthResponse`. Private helper `classify()` (internal) | `GetMetricHealthHandler`, `NewGetMetricHealthHandler`, `GetMetricHealthQuery` | USED — `server.go:264`, `audit_handler.go` |
| `internal/app/queries/get_qa_summary.go` | 97 | Handler cho GET /api/audit/qa. Query `QACriterionRating` + `QAGapState`, tính composite score → `dto.QASummaryResponse`. Private helper `toCriterionRows()` | `GetQASummaryHandler`, `NewGetQASummaryHandler`, `GetQASummaryQuery` | USED — `server.go:262`, `audit_handler.go` |
| `internal/app/queries/get_source_object_mapping_context.go` | 35 | Query + Handler cho GET /api/v1/source-objects/registry/{registry_id}. Delegate sang `SourceObjectReader.GetMappingContextByRegistryID` | `GetSourceObjectMappingContextQuery`, `GetSourceObjectMappingContextHandler`, `NewGetSourceObjectMappingContextHandler` | **⚠PARTIAL** — Handler USED (`server.go:110`). `GetSourceObjectMappingContextQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/get_sync_health.go` | 57 | Read port `SyncHealthReader` + query + handler cho GET /api/sync/health. Trả `SyncHealthSnapshot` (5 aggregate counts) | `SyncHealthReader`, `SyncHealthSnapshot`, `GetSyncHealthQuery`, `GetSyncHealthHandler`, `NewGetSyncHealthHandler` | **⚠PARTIAL** — Handler USED (`server.go:122`). `GetSyncHealthQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/get_table_history.go` | 53 | Query + Handler cho GET /api/reconciliation/report/:table. Paginated history từ `ReconReader.GetTableHistory`. Clamps page/size | `GetTableHistoryQuery`, `GetTableHistoryHandler`, `NewGetTableHistoryHandler` | **⚠PARTIAL** — Handler USED (`server.go:118`). `GetTableHistoryQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/get_wizard_session.go` | 97 | Read port `WizardReader` + 2 queries: `GetWizardSessionQuery` (GET /api/v1/wizard/sessions/:id) và `GetWizardProgressQuery` (GET …/progress). `WizardProgressView` projection compact | `WizardReader`, `GetWizardSessionHandler`, `NewGetWizardSessionHandler`, `GetWizardProgressHandler`, `NewGetWizardProgressHandler` | **⚠PARTIAL** — Cả 2 handler USED (`server.go:144-145`). `GetWizardSessionQuery.Type()` và `GetWizardProgressQuery.Type()` đều dead (deadcode.txt) |
| `internal/app/queries/list_activity_logs.go` | 79 | Read port `ActivityLogReader` + query + handler cho GET /api/activity-log. `ActivityLogFilter` với 8 filter fields, paginated (1..200 default 50) | `ActivityLogReader`, `ActivityLogFilter`, `ListActivityLogsQuery`, `ListActivityLogsHandler`, `NewListActivityLogsHandler` | **⚠PARTIAL** — Handler USED (`server.go:132`). `ListActivityLogsQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/list_connectors.go` | 141 | Read port `ConnectorReader` + 3 query/handler: `ListConnectors`, `GetConnector`, `ListConnectorPlugins` — proxy Kafka Connect REST | `ConnectorReader`, `ListConnectorsHandler`, `NewListConnectorsHandler`, `GetConnectorHandler`, `NewGetConnectorHandler`, `ListConnectorPluginsHandler`, `NewListConnectorPluginsHandler` | **⚠PARTIAL** — Tất cả 3 handler USED (`server.go:127-129`). `ListConnectorsQuery.Type()`, `GetConnectorQuery.Type()`, `ListConnectorPluginsQuery.Type()` đều dead (deadcode.txt) |
| `internal/app/queries/list_failed_logs.go` | 49 | Query + Handler cho GET /api/failed-sync-logs. Dùng `ReconReader.ListFailedLogs`, paginated (1..200 default 30) | `ListFailedLogsQuery`, `ListFailedLogsHandler`, `NewListFailedLogsHandler` | **⚠PARTIAL** — Handler USED (`server.go:119`). `ListFailedLogsQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/list_gaps.go` | 66 | Handler cho GET /api/audit/gaps. Query `QAGapState` với filter priority/status, trả `[]dto.GapRow` | `ListGapsQuery`, `ListGapsHandler`, `NewListGapsHandler` | USED — `server.go:263`, `audit_handler.go` |
| `internal/app/queries/list_latest_reports.go` | 42 | Query + Handler cho GET /api/reconciliation/report. Dùng `ReconReader.ListLatest`, trả `ListLatestReportsResult{Data, Count}`. Note: enrichment fields (DriftPct, ComputedStatus…) được fill bởi API layer | `ListLatestReportsQuery`, `ListLatestReportsHandler`, `NewListLatestReportsHandler` | **⚠PARTIAL** — Handler USED (`server.go:117`). `ListLatestReportsQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/list_mapping_rules.go` | 65 | Query + Handler cho GET /api/mapping-rules. Dùng `ports.MappingRuleRepo.ListPaginated`, filter + pagination | `ListMappingRulesQuery`, `ListMappingRulesHandler`, `NewListMappingRulesHandler` | **⚠PARTIAL** — Handler USED (`server.go:106`). `ListMappingRulesQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/list_masters.go` | 81 | Read port `MasterReader` + `MasterListItem` projection (cross 4 tables) + query + handler cho GET /api/v1/masters | `MasterReader`, `MasterListItem`, `ListMastersQuery`, `ListMastersHandler`, `NewListMastersHandler` | **⚠PARTIAL** — Handler USED (`server.go:114`, `master_read_repo_gorm.go`). `ListMastersQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/list_snapshot_progress.go` | 58 | Read port `SnapshotProgressReader` + query + handler cho GET /api/snapshot-progress. Paginated filter `SnapshotProgressFilter` | `SnapshotProgressReader`, `SnapshotProgressFilter`, `ListSnapshotProgressQuery`, `ListSnapshotProgressHandler`, `NewListSnapshotProgressHandler` | **⚠PARTIAL** — Handler USED (`server.go:136`, `snapshot_progress_read_repo_gorm.go`). `ListSnapshotProgressQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/list_source_objects.go` | 56 | Query + Handler cho GET /api/v1/source-objects. Dùng `SourceObjectReader.ListEnriched`, paginated (default 20, max 500) | `ListSourceObjectsQuery`, `ListSourceObjectsHandler`, `NewListSourceObjectsHandler` | **⚠PARTIAL** — Handler USED (`server.go:109`). `ListSourceObjectsQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/list_sources.go` | 77 | Read port `SourceReader` + 2 queries: `ListSources` (GET /api/v1/sources) và `GetSource` (GET /api/v1/sources/:id) | `SourceReader`, `ListSourcesHandler`, `NewListSourcesHandler`, `GetSourceHandler`, `NewGetSourceHandler` | **⚠PARTIAL** — Cả 2 handler USED (`server.go:142-143`). `ListSourcesQuery.Type()` và `GetSourceQuery.Type()` đều dead (deadcode.txt) |
| `internal/app/queries/list_transmute_schedules.go` | 64 | Read port `TransmuteScheduleReader` + `TransmuteScheduleRow` projection + query + handler cho GET /api/v1/schedules | `TransmuteScheduleReader`, `TransmuteScheduleRow`, `ListTransmuteSchedulesQuery`, `ListTransmuteSchedulesHandler`, `NewListTransmuteSchedulesHandler` | **⚠PARTIAL** — Handler USED (`server.go:140`, `transmute_schedule_read_repo_gorm.go`). `ListTransmuteSchedulesQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/list_worker_schedules.go` | 82 | Read port `WorkerScheduleReader` + `WorkerScheduleResponse` / `WorkerScheduleScope` projections + query + handler cho GET /api/worker-schedule | `WorkerScheduleReader`, `WorkerScheduleResponse`, `WorkerScheduleScope`, `ListWorkerSchedulesQuery`, `ListWorkerSchedulesHandler`, `NewListWorkerSchedulesHandler` | **⚠PARTIAL** — Handler USED (`server.go:148`, `schedule_handler.go`, `worker_schedule_read_repo_gorm.go`). `ListWorkerSchedulesQuery.Type()` dead (deadcode.txt) |
| `internal/app/queries/recon_enrichment.go` | 119 | Enrichment utilities: `ComputeDriftStatus` (drift_pct / status từ source/dest counts), `DeriveSourceQueryMethod`, `ErrorMessagesVI` map (VI messages), `TrimReconValue`, `StringOrNil` | `ComputeDriftStatus`, `DeriveSourceQueryMethod`, `ErrorMessagesVI`, `TrimReconValue`, `StringOrNil` | USED — `reconciliation_handler_reports.go`, `reconciliation_handler.go`, `reconciliation_handler_retry.go` |
| `internal/app/queries/recon_read_models.go` | 61 | Projection types: `LatestReportRow` (embed `model.ReconciliationReport` + enrichment columns), `FailedLogRow`, `FailedLogFilter`, `FailedLogRetryScope`, `BackfillRunRow`, `ReconScopeFilter`, `ErrAmbiguousScope` | `LatestReportRow`, `FailedLogRow`, `FailedLogRetryScope`, `BackfillRunRow`, `ReconScopeFilter`, `ErrAmbiguousScope` | USED — `recon_read_repo_gorm.go`, `reconciliation_handler*.go` |
| `internal/app/queries/recon_reader.go` | 116 | Read port `ReconReader` interface (7 methods): ListLatest, GetTableHistory, ListFailedLogs, ResolveTargetTableByScope, GetFailedLogByID, GetRetryScopeByLogID, ListBackfillRuns, CountTableRows | `ReconReader` | USED — `recon_read_repo_gorm.go` (adapter), `reconciliation_handler.go` (consumer) |
| `internal/app/queries/resolve_mapping_scope.go` | 129 | Handler tự chứa DB: `ResolveMappingScopeHandler.Handle` resolve source_object + shadow_binding scope để lấy `MappingRuleScope` (dùng khi create/update mapping rule). Private `ptrTrim()` helper | `ResolveMappingScopeHandler`, `NewResolveMappingScopeHandler`, `ResolveMappingScopeQuery`, `MappingRuleScope` | USED — `server.go:111`, `mapping_rule_handler.go:13,17`, `mapping_rule_handler_commands.go:21` |
| `internal/app/queries/snapshot_progress_read_models.go` | 23 | Projection type `SnapshotProgressRow` (join snapshot_progress + source_object_registry) | `SnapshotProgressRow` | USED — `snapshot_progress_read_repo_gorm.go`, `snapshot_progress_handler.go` |
| `internal/app/queries/source_object_reader.go` | 33 | Read port `SourceObjectReader` (2 methods: ListEnriched, GetMappingContextByRegistryID) + `SourceObjectListFilter` | `SourceObjectReader`, `SourceObjectListFilter` | USED — `source_object_read_repo_gorm.go` (adapter), `server.go` wires 2 handler dùng |
| `internal/app/queries/source_objects_read_models.go` | 89 | Projection types: `SourceObjectListItem` (57 fields, V2 list view) + `SourceObjectMappingContextReadModel` (mapping context view) | `SourceObjectListItem`, `SourceObjectMappingContextReadModel` | USED — `source_object_read_repo_gorm.go`, `source_objects_handler.go` (type alias) |

---

## Tóm tắt

- **Tổng files**: 29 files (không tính `doc.go` boilerplate).
- **Hoàn toàn DEAD**: **0 files** — tất cả files đều có ít nhất một symbol được sử dụng.
- **PARTIAL** (có `.Type()` method bị deadcode): **21 files** — hầu hết `*Query.Type() string` methods là marker interface nhưng không bị query bus gọi thực sự.

## Chi tiết hàm dead (từ deadcode.txt)

Tất cả hàm dead trong queries/ đều là `.Type() string` methods trên Query structs:

| File | Hàm dead |
|------|----------|
| `get_activity_stats.go` | `GetActivityStatsQuery.Type()` |
| `get_job.go` | `GetJobQuery.Type()` |
| `get_source_object_mapping_context.go` | `GetSourceObjectMappingContextQuery.Type()` |
| `get_sync_health.go` | `GetSyncHealthQuery.Type()` |
| `get_table_history.go` | `GetTableHistoryQuery.Type()` |
| `get_wizard_session.go` | `GetWizardSessionQuery.Type()` + `GetWizardProgressQuery.Type()` |
| `list_activity_logs.go` | `ListActivityLogsQuery.Type()` |
| `list_connectors.go` | `ListConnectorsQuery.Type()` + `GetConnectorQuery.Type()` + `ListConnectorPluginsQuery.Type()` |
| `list_failed_logs.go` | `ListFailedLogsQuery.Type()` |
| `list_latest_reports.go` | `ListLatestReportsQuery.Type()` |
| `list_mapping_rules.go` | `ListMappingRulesQuery.Type()` |
| `list_masters.go` | `ListMastersQuery.Type()` |
| `list_snapshot_progress.go` | `ListSnapshotProgressQuery.Type()` |
| `list_source_objects.go` | `ListSourceObjectsQuery.Type()` |
| `list_sources.go` | `ListSourcesQuery.Type()` + `GetSourceQuery.Type()` |
| `list_transmute_schedules.go` | `ListTransmuteSchedulesQuery.Type()` |
| `list_worker_schedules.go` | `ListWorkerSchedulesQuery.Type()` |

**Root cause**: Các `.Type()` methods được thiết kế cho query bus (pattern CQRS), nhưng hiện tại handlers được gọi **trực tiếp** (direct struct call), không thông qua query bus dispatch. Query bus chưa được implement cho Q-side → methods là dead code nhưng là _intentional design artifact_.

## Area: persistence

# Inventory — internal/infra/persistence/ (non-test .go files)

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/infra/persistence/activity_log_read_repo_gorm.go` | 178 | GORM read adapter cho `queries.ActivityLogReader` — query `cdc_activity_log` với LATERAL join shadow_binding + source_object_registry | `ActivityLogReadRepo`, `NewActivityLogReadRepo` | **USED** — wired ở server.go:131 |
| `internal/infra/persistence/activity_logger.go` | 151 | Single-owner write/read của `cdc_activity_log`. Hai semantic: `Log()` (sync) + `LogAsync()` (fire-and-forget goroutine) | `ActivityLogger`, `NewActivityLogger`, `Log`, `LogAsync`, `ListActivityLogs`, `BuildRowForTest` | **⚠PARTIAL** — wired server.go:176, `LogAsync` được gọi nhiều nơi; `Log` (sync) 0 call site bên ngoài; `BuildRowForTest` dead |
| `internal/infra/persistence/alert_manager.go` | 446 | AlertManager — Fire/Resolve/Silence/Ack alert vào DB + Redis. State machine cho system health alerts | `AlertManager`, `NewAlertManager` | **USED** — wired server.go:227, SetAlertManager, cmdBus sync handlers |
| `internal/infra/persistence/approval_service.go` | 150 | ApprovalService — Approve/Reject pending schema-change field | `ApprovalService`, `NewApprovalService`, `Approve`, `Reject`, `StrPtrForTest` | **⚠PARTIAL** — wired server.go:173; `Approve`/`Reject` gọi ở schema_change_handler.go; `StrPtrForTest` dead |
| `internal/infra/persistence/bridge_status_repo_gorm.go` | 185 | GORM adapter cho `queries.BridgeStatusReader` — query bridge status cho source objects | `BridgeStatusRepo`, `NewBridgeStatusRepo` | **USED** — wired server.go:124 |
| `internal/infra/persistence/doc.go` | 8 | Package doc — chú thích quy tắc persistence layer | — | **USED** (package doc) |
| `internal/infra/persistence/job_repo_gorm.go` | 223 | GORM adapter cho `ports.JobRepo` — CRUD `cdc_system.cdc_jobs` (status tracking cho async cmds) | `JobRepo`, `NewJobRepo`, `Create`, `UpdateStatus`, `Get` | **USED** — wired server.go:150, dùng bởi nats_command_bus, MasterSwap |
| `internal/infra/persistence/mapping_rule_repo_gorm.go` | 265 | GORM adapter cho `ports.MappingRuleRepo` — CRUD `cdc_system.mapping_rule_v2` | `MappingRuleRepo`, `NewMappingRuleRepo` | **USED** — wired server.go:105, handler mapping_rule |
| `internal/infra/persistence/master_mapping_rule_repo_gorm.go` | 175 | GORM adapter cho `mapping.MasterRuleRepository` — CRUD master mapping rules | `NewMasterRuleRepository`, `masterRuleRepoGorm` | **USED** — wired server.go:192, masterMappingRuleHandler |
| `internal/infra/persistence/master_read_repo_gorm.go` | 79 | GORM read adapter cho `queries.MasterReader` — query `cdc_system.master_binding` | `MasterReadRepo`, `NewMasterReadRepo` | **USED** — wired server.go:113 |
| `internal/infra/persistence/master_swap.go` | 192 | MasterSwap service — atomic RENAME swap 2 bảng master trong TX, tạo job row, chạy goroutine detached | `MasterSwap`, `NewMasterSwap`, `NewMasterSwapForTest`, `SwapAsync`, `detectPartialState`, `runSwapGoroutine`, `runSwapTX` | **⚠DEAD?** — KHÔNG wired vào server.go; `master.swap` cmd đi qua NATS subject (`cmdBus.RegisterSubject`), KHÔNG qua `persistence.MasterSwap`. 6/6 hàm trong deadcode.txt. 0 ref bên ngoài file. |
| `internal/infra/persistence/pending_field_repo_gorm.go` | 64 | GORM adapter cho `ports.PendingFieldRepo` — CRUD `cdc_system.pending_field_changes` | `PendingFieldRepo`, `NewPendingFieldRepo` | **USED** — wired server.go:100 |
| `internal/infra/persistence/provisioning_orchestrator.go` | 742 | ProvisioningOrchestrator — state-machine driver cho CDC provisioning (Advance/Pause/Resume/Retry/Archive/SetMode) qua CAS UPDATE + NATS publish | `ProvisioningOrchestrator`, `NewProvisioningOrchestrator`, `GetState`, `Advance`, `Pause`, `Resume`, `Retry`, `Archive`, `SetMode` | **⚠PARTIAL** — wired server.go:259; `readMode` dead (0 call bên trong); 3 `ForTest` funcs dead (`NewProvisioningCorrelationIDForTest`, `InjectProvisioningTraceContextForTest`, `ProvisioningEntryWithSpanForTest`) |
| `internal/infra/persistence/provisioning_state_machine.go` | 75 | Pure state machine constants + maps + helper funcs cho provisioning lifecycle — bản copy từ centralized-data-service | `ProvisioningState`, `ProvisioningTransitions`, `ProvisioningPendingToFinalize`, `ProvisioningCanAdvance`, `ProvisioningIsPending`, `ProvisioningIsTerminal` | **⚠PARTIAL** — `ProvisioningTransitions`, `ProvisioningCanAdvance`, `ProvisioningState` constants được dùng trong provisioning_orchestrator.go; `ProvisioningIsPending` + `ProvisioningIsTerminal` = 0 ref ngoài (dead). Package comment mismatch: `// Package service` nhưng `package persistence`. |
| `internal/infra/persistence/recon_read_repo_gorm.go` | 353 | GORM read adapter cho `queries.ReconReader` — LatestReport, FailedLogs, TableHistory | `ReconReadRepo`, `NewReconReadRepo` | **USED** — wired server.go:116 |
| `internal/infra/persistence/registry_repo_gorm.go` | 98 | GORM adapter cho `ports.RegistryRepo` — CRUD `cdc_table_registry` | `RegistryRepo`, `NewRegistryRepo` | **USED** — wired server.go:99 |
| `internal/infra/persistence/schema_log_repo_gorm.go` | 41 | GORM adapter cho `ports.SchemaLogRepo` — CRUD `cdc_system.schema_change_log` | `SchemaLogRepo`, `NewSchemaLogRepo` | **USED** — wired server.go:101 |
| `internal/infra/persistence/sensitive_field_repo_gorm.go` | 81 | GORM adapter cho `ports.SensitiveFieldRepo` — CRUD sensitive field config | `SensitiveFieldRepo`, `NewSensitiveFieldRepo` | **USED** — wired server.go:194 |
| `internal/infra/persistence/shadow_automator.go` | 197 | ShadowAutomator — tạo/drop shadow table trong shadow DB, quản lý DDL (CREATE TABLE LIKE) | `ShadowAutomator`, `NewShadowAutomator`, `validateIdent` | **USED** — wired server.go:174, registry_handler |
| `internal/infra/persistence/snapshot_progress_read_repo_gorm.go` | 47 | GORM read adapter cho snapshot progress queries | `SnapshotProgressReadRepoGorm`, `NewSnapshotProgressReadRepoGorm` | **USED** — wired server.go:135 |
| `internal/infra/persistence/source_object_read_repo_gorm.go` | 291 | GORM read adapter cho `queries.SourceObjectReader` — query `cdc_system.source_object_registry` | `SourceObjectReadRepo`, `NewSourceObjectReadRepo` | **USED** — wired server.go:108 |
| `internal/infra/persistence/source_object_v2_sync.go` | 501 | SourceObjectV2SyncService — sync legacy TableRegistry → source_object_registry + shadow_binding (2-INSERT pipeline) | `SourceObjectV2SyncService`, `NewSourceObjectV2SyncService`, `NewSourceObjectV2SyncServiceForTest`, `SyncFromLegacy`, `SyncFromLegacyTx`, `SyncRulesFromLegacyTx` | **⚠PARTIAL** — wired server.go:175, cmd/sync_v2/main.go; `NewSourceObjectV2SyncServiceForTest` dead |
| `internal/infra/persistence/sync_health_read_repo_gorm.go` | 49 | GORM read adapter cho `queries.SyncHealthReader` — aggregate counts cdc_table_registry + cdc_mapping_rules | `SyncHealthReadRepo`, `NewSyncHealthReadRepo` | **USED** — wired server.go:121 |
| `internal/infra/persistence/system_connector_repo_gorm.go` | 253 | GORM adapter cho `ports.SystemConnectorRepo` — CRUD `cdc_sources` (Connection-Fingerprint registry) | `SystemConnectorRepo`, `NewSystemConnectorRepo` | **USED** — wired server.go:102 |
| `internal/infra/persistence/transmute_schedule_read_repo_gorm.go` | 51 | GORM read adapter cho `queries.TransmuteScheduleReader` — query cdc_system.transmute_schedule JOIN master_binding | `TransmuteScheduleReadRepo`, `NewTransmuteScheduleReadRepo` | **USED** — wired server.go:139 |
| `internal/infra/persistence/wizard_repo_gorm.go` | 64 | GORM adapter cho `ports.WizardRepo` — CRUD wizard session + progress | `WizardRepo`, `NewWizardRepo` | **USED** — wired qua commands/wizard handlers |
| `internal/infra/persistence/worker_schedule_read_repo_gorm.go` | 158 | GORM read adapter cho `queries.WorkerScheduleReader` — query cdc_system.cdc_worker_schedule với LATERAL joins | `WorkerScheduleReadRepo`, `NewWorkerScheduleReadRepo` | **USED** — wired server.go:147 |

---

## Tổng kết

- **Tổng files**: 27 (bao gồm doc.go)
- **DEAD?**: `master_swap.go` (6/6 hàm dead, không wired vào server.go — cmd đi qua NATS RegisterSubject thay vì qua struct này)
- **PARTIAL**:
  - `activity_logger.go`: `ActivityLogger.Log` (sync, 0 call site), `BuildRowForTest` dead
  - `approval_service.go`: `StrPtrForTest` dead
  - `provisioning_orchestrator.go`: `readMode`, `NewProvisioningCorrelationIDForTest`, `InjectProvisioningTraceContextForTest`, `ProvisioningEntryWithSpanForTest` dead
  - `provisioning_state_machine.go`: `ProvisioningIsPending`, `ProvisioningIsTerminal` dead; package comment mismatch (`// Package service` vs `package persistence`)
  - `source_object_v2_sync.go`: `NewSourceObjectV2SyncServiceForTest` dead

## Area: infra

# Inventory — AREA: infra
> Codebase: `cdc-cms-service` | Base: `internal/infra/`
> Phân tích: 2026-06-08

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/infra/http/doc.go` | 6 | Package doc — outbound HTTP clients | (package doc only) | USED (package imported ở server.go, queries, commands, api) |
| `internal/infra/http/kafka_connect.go` | 222 | Typed REST client cho Kafka Connect API (list/status/config/plugins/create/update/delete/restart/lifecycle + FilterSafeConfig). Dùng bởi handler và queries | `KafkaConnectClient`, `NewKafkaConnectClient`, `FilterSafeConfig`, `ErrKafkaConnectNotFound`, `ConnectorView`, `ConnectorStatusResp`, `ConnectorTask`, `ConnectorState` | USED — `NewKafkaConnectClient` wired tại server.go:126; types dùng trong list_connectors.go, system_connectors_handler.go, commands/system_connector.go |
| `internal/infra/http/prom_client.go` | 390 | Prometheus HTTP API client (Path A: PromQL; Path B: scrape worker /metrics + tính histogram quantile in-process). Cung cấp P50/P95/P99 latency | `PromClient`, `NewPromClient`, `PromClientConfig`, `QueryPercentile`, `QueryGauge`, `QueryLatencyTriple`, `LatencyResult`, `PercentileSource` | ⚠PARTIAL — USED: `NewPromClient` wired server.go:201; `QueryLatencyTriple` gọi bởi system_health_collector.go:228; `QueryGauge` gọi bởi get_metric_health.go. DEAD: `ComputeHistogramQuantileForTest` (deadcode.txt: prom_client.go:388) — test helper unreachable từ binary |
| `internal/infra/messaging/doc.go` | 7 | Package doc — NATS adapters | (package doc only) | USED (package imported rộng rãi: server.go + 25+ api handlers) |
| `internal/infra/messaging/nats_command_bus.go` | 314 | NATS-backed CommandBus: `Execute` (sync, in-process) và `Dispatch` (async, NATS publish + job row). Registry pattern — một wiring line per command type | `natsCommandBus`, `NewNATSCommandBus`, `RegisterSync`, `RegisterSubject`, `Execute`, `Dispatch`, `WithMetadata`, `SyncHandler`, `buildCommandMsg` | ⚠PARTIAL — USED: `NewNATSCommandBus` wired server.go:153; `WithMetadata` dùng ở 25+ API handlers; `RegisterSync`/`RegisterSubject` wired server.go. DEAD: `BuildCommandMsgForTest` (deadcode.txt:308), `CtxStringForTest` (deadcode.txt:311) — test helpers unreachable |
| `internal/infra/messaging/stuck_job_reaper.go` | 139 | Periodic sweep cdc_jobs rows stuck 'running' quá per-type timeout (SQL CASE expr, 1 round-trip). Kết hợp DefaultJobTimeouts map cho 16 command types | `StuckJobReaper`, `NewStuckJobReaper`, `DefaultJobTimeouts`, `Run`, `reapOnce` | ⚠PARTIAL — USED: `NewStuckJobReaper` wired server.go:278; `Run` started goroutine server.go:317. DEAD: `IntervalForTest`, `TimeoutsForTest`, `DefaultTOForTest` (deadcode.txt:137-139) — test getters unreachable |
| `internal/infra/observability/system_health_alerts.go` | 293 | Phase 6 — tie Collector với AlertManager state machine. `evaluateAlerts` detect conditions (Debezium FAILED, HighConsumerLag, ReconDrift, InfrastructureDown) và gọi Fire()/Resolve() | `Collector.SetAlertManager`, `Collector.evaluateAlerts`, `Collector.detectConditions`, `Collector.ownsAlertName`, `toFloat64`, `detectedCondition`, `ownedAlertNames` | ⚠PARTIAL — USED: `SetAlertManager` gọi server.go:228; `evaluateAlerts` gọi nội bộ từ `collectAndCache`; `detectConditions`/`ownsAlertName` gọi từ `evaluateAlerts`. DEAD (test helpers): `detectedCondition.GetReqForTest` (deadcode.txt:287), `ToFloat64ForTest` (deadcode.txt:289), `Collector.OwnsAlertNameForTest` (deadcode.txt:290), `Collector.DetectConditionsForTest` (deadcode.txt:291) — 4 hàm unreachable |
| `internal/infra/observability/system_health_collector.go` | 267 | Orchestrator chạy health probes async (errgroup), cache snapshot JSON vào Redis (key `system_health:snapshot`, TTL 60s). Định nghĩa `Snapshot`, `CollectorConfig`, `Collector` | `Collector`, `NewCollector`, `CollectorConfig`, `Snapshot`, `Run`, `collectAndCache` | ⚠PARTIAL — USED: `NewCollector` wired server.go:208; `Run` started goroutine server.go:296; `Snapshot` used bởi system_health_handler.go:103. DEAD: `CollectOnce` (deadcode.txt: system_health_collector.go:166) — 1 hàm unreachable |
| `internal/infra/observability/system_health_compute.go` | 116 | Pure functions: `computeAlerts` (walk snapshot → flat alert list cho FE banner) và `computeOverall` (healthy/degraded/critical). Tách biệt khỏi AlertManager persistence | `computeAlerts`, `computeOverall` | ⚠PARTIAL — USED: cả hai gọi nội bộ từ `collectAndCache`. DEAD (test wrappers): `ComputeAlertsForTest` (deadcode.txt:115), `ComputeOverallForTest` (deadcode.txt:116) — 2 hàm unreachable |
| `internal/infra/observability/system_health_queries.go` | 118 | DB-derived snapshot sections: `queryReconciliation` (DISTINCT ON latest report/table), `queryFailedCount` (24h+1h failed sync, single raw SQL), `queryRecentEvents` (last 10 activity logs) | `Collector.queryReconciliation`, `Collector.queryFailedCount`, `Collector.queryRecentEvents` | USED — tất cả 3 gọi từ `collectAndCache` trong goroutines của errgroup |
| `internal/infra/observability/probes/debezium.go` | 112 | Probe tất cả Kafka Connect connectors (`DebeziumAll` enumerate all + worst-of aggregation) và probe từng connector (`Debezium` — state/task summary, FAILED trace truncate 500 chars) | `DebeziumAll`, `Debezium` | USED — `DebeziumAll` gọi từ `system_health_collector.go` trong goroutine errgroup |
| `internal/infra/observability/probes/deps.go` | 119 | Package foundation: `HTTPDeps` (shared http.Client + timeout), `Get` (context-bounded HTTP GET), `SanitizeErr` (redact URLs/credentials từ error strings trước khi cache), status constants | `HTTPDeps`, `HTTPDeps.Get`, `SanitizeErr`, `isSchemeByte`, status constants | ⚠PARTIAL — USED: `HTTPDeps` + `Get` + `SanitizeErr` dùng bởi tất cả các probe files; status constants shared. DEAD: `IsSchemeByteForTest` (deadcode.txt: probes/deps.go:119) — 1 test helper unreachable |
| `internal/infra/observability/probes/kafka_connect.go` | 36 | Probe Kafka Connect REST root (`/`) — returns cluster metadata (version, commit, latency_ms). 2xx + parsable body = up | `KafkaConnect` | USED — gọi từ `system_health_collector.go`:194 goroutine |
| `internal/infra/observability/probes/kafka_lag.go` | 125 | Scrape kafka-exporter sidecar Prometheus text endpoint, aggregate `kafka_consumergroup_lag` → `total_lag` + per-topic breakdown (với prefix filter). Graceful khi sidecar không deploy | `KafkaLag` | USED — gọi từ `system_health_collector.go`:206 goroutine |
| `internal/infra/observability/probes/nats.go` | 39 | Probe NATS JetStream monitoring `/jsz` — stream/consumer/message counts. Up = JS subsystem reachable | `NATS` | USED — gọi từ `system_health_collector.go`:195 goroutine |
| `internal/infra/observability/probes/postgres.go` | 34 | Probe CMS metadata DB — count registry rows (liveness gate) + `pg_stat_user_tables` row sum | `Postgres` | USED — gọi từ `system_health_collector.go`:196 goroutine |
| `internal/infra/observability/probes/redis.go` | 24 | Probe Redis cache via PING. nil cache → status=unknown (bootstrap safety) | `Redis` | USED — gọi từ `system_health_collector.go`:197 goroutine |
| `internal/infra/observability/probes/worker.go` | 36 | Probe centralized-data-service worker tại `/healthz` (no-auth dev endpoint, tránh JWT-gated `/health`) | `Worker` | USED — gọi từ `system_health_collector.go`:200 goroutine |
| `internal/infra/cache/doc.go` | 6 | Package doc placeholder — Redis-backed adapters (snapshot cache, idempotency locks, rate limits, alert dedup, lockout counters). Phase 2 v2 P1, chưa có implementation | (package doc only — zero symbols) | ⚠DEAD? — 0 import từ non-test code. `pkgs/rediscache` được dùng trực tiếp thay thế. Package này là DEAD PACKAGE: chỉ có doc.go, không có type/func nào, không được import ở bất kỳ file non-test nào trong repo |

---
## Tóm tắt

- **Tổng files phân tích**: 19 (bao gồm 3 doc.go)
- **DEAD?**: `internal/infra/cache/doc.go` (package chết — 0 non-test import, placeholder chưa implement)
- **PARTIAL**: 6 files có test-helper functions unreachable từ binary:
  - `http/prom_client.go`: `ComputeHistogramQuantileForTest`
  - `messaging/nats_command_bus.go`: `BuildCommandMsgForTest`, `CtxStringForTest`
  - `messaging/stuck_job_reaper.go`: `IntervalForTest`, `TimeoutsForTest`, `DefaultTOForTest`
  - `observability/system_health_alerts.go`: `GetReqForTest`, `ToFloat64ForTest`, `OwnsAlertNameForTest`, `DetectConditionsForTest` (4 hàm)
  - `observability/system_health_collector.go`: `CollectOnce`
  - `observability/system_health_compute.go`: `ComputeAlertsForTest`, `ComputeOverallForTest`
  - `observability/probes/deps.go`: `IsSchemeByteForTest`

## Area: core

# Inventory: AREA core — cdc-cms-service

Generated by: grep-based analysis (non-test .go files only)

---

## 1. internal/domain/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/domain/job/job.go` | 51 | Domain entity Job (async CDC jobs); lifecycle: pending→running→success/failed | `Job`, `Status` (4 const), `New()` | **USED** — 5 non-test files import `domain/job` |
| `internal/domain/mapping/errors.go` | 19 | Sentinel errors cho mapping repo / command handlers; map → HTTP 400/409 | `ErrInvalidScope`, `ErrDuplicate` | **USED** — 9 non-test files import `domain/mapping` |
| `internal/domain/mapping/master_rule.go` | 52 | Domain entity MasterRule (mapping_rule_master); interface repository + filter | `MasterRule`, `MasterFilter`, `MasterRuleRepository` | **USED** — import cùng package `domain/mapping` |
| `internal/domain/mapping/rule.go` | 106 | Domain entity Rule (mapping_rule_v2); enums Status/RuleType/MaskStrategy; `IsValidMaskStrategy()` | `Rule`, `Filter`, `Status`, `RuleType`, `MaskStrategy` (4 const), `IsValidMaskStrategy()` | **USED** — imported rộng rãi |
| `internal/domain/master/binding.go` | 45 | Domain entity Binding (master_binding); SchemaStatus workflow | `Binding`, `Filter`, `SchemaStatus` (4 const) | ⚠**PARTIAL/THIN** — chỉ 1 file import `domain/master` (repository.go); struct thuần anemic, 0 method |
| `internal/domain/reconciliation/failed_log.go` | 46 | Domain projection FailedLog (failed_sync_logs) + filter | `FailedLog`, `LogFilter`, `FailedLogStatus` (3 const) | ⚠**PARTIAL/THIN** — chỉ 1 file import `domain/reconciliation` (repository.go); struct anemic |
| `internal/domain/reconciliation/report.go` | 50 | Domain read-model Report (cdc_reconciliation_report) + filter | `Report`, `Filter`, `DriftStatus` (7 const) | ⚠**PARTIAL/THIN** — cùng file import với failed_log; struct anemic |
| `internal/domain/source/object.go` | 54 | Domain entity Object (source_object_registry); ProvisioningState + Scope + Filter | `Object`, `Filter`, `Scope`, `ProvisioningState` (5 const) | ⚠**PARTIAL/THIN** — chỉ 1 file import `domain/source` (repository.go); struct anemic |

---

## 2. internal/model/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/model/activity_log.go` | 23 | GORM model `cdc_activity_log`; lưu log hoạt động sync | `ActivityLog` | **USED** — 5 non-test files reference `model.ActivityLog` |
| `internal/model/alert.go` | 54 | GORM model `cdc_system.cdc_alerts`; alert status/severity consts | `Alert`, `AlertStatus*` (4), `AlertSeverity*` (3) | **USED** — 2 non-test files reference `model.Alert` |
| `internal/model/cdc_event.go` | 28 | Struct CDCEvent/CDCEventData/UpsertRecord — CloudEvent shape | `CDCEvent`, `CDCEventData`, `UpsertRecord` | ⚠**DEAD?** — 0 import ngoài chính file định nghĩa; không có file nào dùng `model.CDCEvent` hay `model.UpsertRecord` trong non-test |
| `internal/model/failed_sync_log.go` | 30 | GORM model `cdc_system.failed_sync_logs` | `FailedSyncLog` | **USED** — 3 non-test files reference `model.FailedSyncLog` |
| `internal/model/mapping_rule.go` | 27 | GORM model `cdc_mapping_rules` (legacy V1 table) | `MappingRule` | ⚠**PARTIAL** — chỉ 2 non-test files: `approval_service.go`, `update_registry.go`; bảng V1 đang được thay bởi `mapping_rule_v2` (domain/mapping) |
| `internal/model/pending_field.go` | 24 | GORM model `pending_fields`; schema discovery workflow | `PendingField` | **USED** — 3 non-test files reference `model.PendingField` |
| `internal/model/qa_audit.go` | 45 | GORM model `cdc_system.qa_gap_state` + `qa_criterion_rating`; QA audit tracking | `QAGapState`, `QACriterionRating` | **USED** — 2 non-test files reference `model.QAGapState` |
| `internal/model/reconciliation_report.go` | 29 | GORM model `cdc_reconciliation_report` | `ReconciliationReport` | **USED** — 5 non-test files reference `model.ReconciliationReport` |
| `internal/model/schema_change_log.go` | 23 | GORM model `schema_changes_log`; DDL execution audit | `SchemaChangeLog` | **USED** — 3 non-test files reference `model.SchemaChangeLog` |
| `internal/model/sensitive_field.go` | 13 | GORM model `cdc_system.sensitive_fields`; masking config | `SensitiveField` | **USED** — 2 non-test files reference `model.SensitiveField` |
| `internal/model/source.go` | 25 | GORM model `cdc_system.sources`; Kafka Connect connector metadata | `Source` | **USED** — 6 non-test files reference `model.Source` |
| `internal/model/table_registry.go` | 44 | GORM model `cdc_table_registry` (legacy V1 registry) | `TableRegistry` | **USED** — 13 non-test files reference `model.TableRegistry` |
| `internal/model/wizard_session.go` | 22 | GORM model `cdc_system.cdc_wizard_sessions`; Source→Master wizard state | `WizardSession` | **USED** — 4 non-test files reference `model.WizardSession` |
| `internal/model/worker_schedule.go` | 20 | GORM model `cdc_worker_schedule`; scheduler config | `WorkerSchedule` | **USED** — 3 non-test files reference `model.WorkerSchedule` |

---

## 3. internal/bootstrap/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/bootstrap/master_connection.go` | 46 | Seed idempotent row `default_master` vào `connection_registry` khi boot | `EnsureDefaultMasterConnection()` | **USED** — gọi trong `server.go:82` |
| `internal/bootstrap/registry_mirror.go` | 260 | Helper functions (splitHostPort, normalizeSourceEngine, nullIfEmpty, nullIfZero, slugify); body function `SyncLegacyToV2Bootstrap` đã COMMENT-OUT hoàn toàn | `splitHostPort`, `normalizeSourceEngine`, `nullIfEmpty`, `nullIfZero`, `slugify` | ⚠**DEAD** — cả 5 hàm unreachable (confirmed deadcode.txt); function chính đã bị comment; không có caller ngoài codebase |
| `internal/bootstrap/shadow_connection.go` | 61 | Seed idempotent row `default_shadow` vào `connection_registry` khi boot | `EnsureDefaultShadowConnection()` | **USED** — gọi trong `server.go:75` |

---

## 4. internal/middleware/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/middleware/audit.go` | 380 | Async audit log middleware; ghi `admin_actions` qua buffered channel | `AuditLogger`, `AuditEvent`, `NewAuditLogger()`, `Run()`, `Middleware()` | ⚠**PARTIAL** — `Stop()` và `DroppedCount()` là dead (deadcode.txt); core logic USED: `NewAuditLogger` + `Run` + `Middleware` được gọi trong server.go/router.go |
| `internal/middleware/deprecation.go` | 53 | RFC 8594 Sunset/Deprecation headers; `CanonicalAPIRoute()` fold /api/v1 → /api | `CanonicalAPIRoute()`, `DeprecateLegacyAPIPath()` | **USED** — `DeprecateLegacyAPIPath` gọi trong router.go:92; `CanonicalAPIRoute` dùng trong audit.go và idempotency.go |
| `internal/middleware/idempotency.go` | 230 | Idempotency-Key middleware (RFC draft); Redis lock + response cache | `IdempotencyConfig`, `NewIdempotency()`, `NewIdempotencyFromRedisClient()` | ⚠**PARTIAL** — `NewIdempotencyForTest()` là dead (deadcode.txt); core `NewIdempotency` + `NewIdempotencyFromRedisClient` USED qua router.go |
| `internal/middleware/jwt.go` | 86 | JWT auth; parse Bearer token; set Locals username/role | `JWTAuth()`, `GetUsername()`, `GetRole()`, `RequireRole()` | **USED** — dùng trong router.go (JWTAuth mount), api handlers dùng GetUsername/GetRole |
| `internal/middleware/ratelimit.go` | 107 | Per-user Redis rate limit (INCR+EXPIRE); destructive endpoints | `RateLimitConfig`, `NewRateLimit()` | **USED** — `NewRateLimit` gọi trong router.go:32 (`bundle.RateRestart`) |
| `internal/middleware/rbac.go` | 135 | RBAC: single-role + multi-role claim + ADMIN_USERS env fallback | `RequireOpsAdmin()`, `RequireAnyRole()`, `RoleOpsAdmin` | **USED** — `RequireOpsAdmin` gọi nhiều chỗ trong router.go (lines 129, 134, 291, etc.) |

---

## 5. internal/migrate/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/migrate/runner.go` | 208 | SQL migration runner; advisory lock + tracker table; apply ordered .sql files | `Run()` | **USED** — gọi trong `server.go:56` |

---

## 6. internal/naming/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/naming/naming.go` | 36 | Shadow schema naming convention; env-var prefix `CDC_SHADOW_SCHEMA_PREFIX` | `ShadowSchemaPrefix()`, `ShadowSchemaName()` | **USED** — `ShadowSchemaName` gọi trong `source_object_v2_sync.go:456` |

---

## 7. internal/router/ và internal/server/

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `internal/router/router.go` | 452 | Đăng ký tất cả routes Fiber; mount middleware stacks; dual-mount /api + /api/v1 | `SetupRoutes()`, `DestructiveMiddleware`, `NewDestructiveMiddleware()` | **USED** (hiển nhiên — entry point routes) |
| `internal/server/server.go` | 343 | DI root: khởi tạo DB/NATS/Redis/repos/handlers; lifecycle Start/Stop | `Server`, `New()` | **USED** (hiển nhiên — main entry) |

---

## Ghi chú: Trùng concept model/ vs domain/

| Cặp trùng | model/ file | domain/ file | Mô tả |
|-----------|-------------|--------------|-------|
| **source** | `model/source.go` → `model.Source` (Kafka connector, bảng `cdc_system.sources`) | `domain/source/object.go` → `source.Object` (source_object_registry V2) | KHÁC nhau: model.Source = connector metadata (V1), domain.Object = registered CDC source (V2). Khác bảng, khác mục đích nhưng cùng concept "nguồn dữ liệu" |
| **mapping rule** | `model/mapping_rule.go` → `model.MappingRule` (bảng `cdc_mapping_rules` V1) | `domain/mapping/rule.go` → `mapping.Rule` (bảng `mapping_rule_v2`) | TRÙNG NGHIỆP VỤ: cùng biểu diễn "mapping field"; model là V1 GORM legacy, domain là V2 clean entity. model.MappingRule đang bị deprecate dần (chỉ 2 caller còn lại) |
| **reconciliation report** | `model/reconciliation_report.go` → `model.ReconciliationReport` | `domain/reconciliation/report.go` → `reconciliation.Report` | TRÙNG: cùng bảng `cdc_reconciliation_report`; model.ReconciliationReport là GORM shape (cho persistence), domain.Report là clean domain shape (cho app layer). Dual representation cùng table |
| **failed sync log** | `model/failed_sync_log.go` → `model.FailedSyncLog` | `domain/reconciliation/failed_log.go` → `reconciliation.FailedLog` | TRÙNG: cùng bảng `cdc_system.failed_sync_logs`; tương tự ở trên — GORM vs clean domain |
| **table registry** | `model/table_registry.go` → `model.TableRegistry` (V1, bảng `cdc_table_registry`) | `domain/source/object.go` → `source.Object` (V2, `source_object_registry`) | TRÙNG CONCEPT: cùng "đối tượng CDC được đăng ký"; model là V1 có GORM tags nặng, domain là V2 abstraction. Đang trong quá trình migration V1→V2 |

---

## Nhận xét Domain Anemic

Tất cả 8 domain entity đều **anemic** (chỉ có struct + const + interface/filter, không có business method):
- `domain/job/job.go`: chỉ có `New()` — factory function, không logic
- `domain/mapping/rule.go`: chỉ có `IsValidMaskStrategy()` — validation helper đơn giản
- `domain/master/binding.go`: 0 method
- `domain/reconciliation/*.go`: 0 method  
- `domain/source/object.go`: 0 method

Business logic nằm hết ở `internal/app/commands/` và `internal/infra/persistence/`, không ở domain.

## Area: shared

# AREA: shared — Inventory Table

| File | LOC | Chức năng | Symbol chính | Status |
|------|-----|-----------|--------------|--------|
| `pkgs/database/postgres.go` | 80 | Khởi tạo GORM *gorm.DB với pool tuning, slow-query logging, warmup ping. DSN hardcode `search_path=cdc_system,public`. | `NewPostgresConnection(DBConfig)` | **USED** — imported bởi `internal/server/server.go` + nhiều `internal/api/` và `internal/app/commands/` |
| `pkgs/natsconn/nats_client.go` | 101 | Tạo NATS JetStream client, tự-reconnect, PublishReload (schema.config.reload), EnsureStreams (CDC_EVENTS, SCHEMA_DRIFT, SCHEMA_CONFIG). | `NewNatsClient`, `NatsClient.PublishReload`, `NatsClient.EnsureStreams`, `NatsClient.Close` | **USED** — `NewNatsClient` được gọi tại `internal/server/server.go` |
| `pkgs/observability/otel.go` | 121 | Khởi tạo OpenTelemetry (OTLP HTTP trace + log), set global TracerProvider, gắn OTel zap bridge. | `InitOtel`, `LogProvider`, `Tracer`, `StartSpan` | **⚠ PARTIAL** — `InitOtel` + `LogProvider` USED bởi `cmd/server/main.go`; `Tracer()` + `StartSpan()` **DEAD** (deadcode.txt xác nhận, không có caller nào trong non-test code) |
| `pkgs/rediscache/redis_client.go` | 108 | Redis client wrapper: Get/Set/Delete/DeletePattern, SetNX, Incr, Expire, TTL, Ping, Client() raw access. Dùng cho idempotency + rate limiter. | `NewRedisCache`, `RedisCache.{Get,Set,Delete,SetNX,Incr,Expire,TTL,Client,Ping,Close}` | **USED** — `NewRedisCache` gọi tại `internal/server/server.go`; methods dùng bởi `internal/middleware/ratelimit.go`, `internal/middleware/idempotency.go`, `internal/infra/persistence/alert_manager.go`, v.v. |
| `pkgs/utils/hash.go` | 13 | SHA-256 hash map[string]interface{} → hex string. | `CalculateHash(map[string]interface{})` | **⚠ DEAD** — deadcode.txt xác nhận; grep không tìm thấy caller nào trong non-test code |
| `pkgs/utils/pg_ident.go` | 28 | Quote Postgres identifier an toàn (fail-closed: trả `""` nếu ký tự lạ để chặn SQL injection). | `PgIdent(name string) string` | **USED** — imported bởi `internal/infra/persistence/recon_read_repo_gorm.go`, `bridge_status_repo_gorm.go`, `internal/app/queries/bridge_status_reader.go` |
| `pkgs/utils/type_inference.go` | 50 | Suy diễn PostgreSQL type từ Go interface{} (BOOLEAN, INTEGER, BIGINT, DECIMAL, TIMESTAMP, VARCHAR, TEXT, JSONB). | `InferDataType(interface{}) string` | **⚠ DEAD** — deadcode.txt xác nhận; grep không tìm thấy caller nào trong non-test code |
| `cmd/server/main.go` | 76 | Binary chính: load config → init OTel → wire OTel-zap bridge → `server.New()` → graceful shutdown (SIGTERM). | `main()` | **USED** — entry point production binary |
| `cmd/sync_v2/main.go` | 40 | **One-shot migration CLI**: kết nối trực tiếp hardcode DSN local (port 5433), gọi `persistence.NewSourceObjectV2SyncService` để sync legacy `TableRegistry` → V2 schema. Không dùng config/observability. | `main()` → `SourceObjectV2SyncService.SyncFromLegacy` | **⚠ DEAD (migration tool, đã hoàn thành)** — không có Makefile/Dockerfile target; DSN hardcode localhost; `NewSourceObjectV2SyncService` trong deadcode.txt (`ForTest` variant dead); README mô tả là "CLI tooling tái đồng bộ V2 (one-shot)" |
| `config/config.go` | 218 | Viper-based config loader: AppConfig (Server, DB, ShadowDB, NATS, Redis, JWT, System, Otel, Migration). Bind ~30 env vars (CMS_*). validateConfig bắt buộc port/db/jwt. | `NewConfig() (*AppConfig, error)`, `validateConfig`, tất cả `*Config` structs | **USED** — imported bởi `cmd/server/main.go`, `pkgs/database`, `pkgs/natsconn`, `pkgs/rediscache` |
| `docs/docs.go` | 2280 | **Generated (swaggo)**. Embed toàn bộ OpenAPI 3 spec của service (routes, schemas, auth). `// Code generated ... DO NOT EDIT`. | `SwaggerInfo`, `init()` đăng ký spec | **USED (generated)** — import blank `_ "cdc-cms-service/docs"` tại `cmd/server/main.go` để ginSwagger serve tự động |
| `migrations/embed.go` | 9 | Embed SQL files: `SchemaFiles` (schema/*/*.sql) và `SeedFiles` (seed/*.sql) vào binary qua `//go:embed`. | `SchemaFiles embed.FS`, `SeedFiles embed.FS` | **USED** — `internal/migrate/runner.go` đọc cả hai |

---

## Tóm tắt DEAD / PARTIAL

| Symbol | File | Kết luận |
|--------|------|----------|
| `CalculateHash` | `pkgs/utils/hash.go` | **DEAD** — không có caller trong production code |
| `InferDataType` | `pkgs/utils/type_inference.go` | **DEAD** — không có caller trong production code |
| `Tracer()` | `pkgs/observability/otel.go` | **DEAD** — không gọi trực tiếp ở đâu ngoài internal `StartSpan` |
| `StartSpan()` | `pkgs/observability/otel.go` | **DEAD** — không có caller nào trong non-test code |
| `cmd/sync_v2/main.go` | toàn bộ binary | **⚠ One-shot migration đã chạy xong** — hardcode local DSN, không production-ready |

## Ghi chú cmd/sync_v2

Binary này là **migration tool một lần** để chuyển đổi dữ liệu `TableRegistry` legacy sang schema V2. Dấu hiệu:
- DSN hardcode `localhost:5433` (không dùng AppConfig)
- Không có observability, không graceful shutdown  
- Không có Makefile/Dockerfile target
- README gọi là "one-shot"
- `NewSourceObjectV2SyncService` vẫn live ở `internal/server/server.go` cho sync runtime, nhưng binary `cmd/sync_v2` chỉ dùng cho backfill thủ công

## Ghi chú pkgs/rediscache

`pkgs/rediscache` **ĐANG DÙNG** ở nhiều điểm. `internal/infra/cache/doc.go` là placeholder trống cho phase tương lai (wrap rediscache sau ports interface) — cache layer cũ này **chưa dead**, đây chỉ là ghi chú kế hoạch refactor.

