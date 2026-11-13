# Recon kiến trúc `cdc-cms-service` (branch recon-heal, đọc working tree hiện tại)

**Direct Summary:** Đây là Fiber monolith, gom wiring thủ công trong 1 God-constructor `internal/server/server.go:82-397`. Pattern dự định là hexagonal + CQRS (api → app/commands|queries → domain/ports ← infra), nhưng hiện chỉ áp dụng **một phần**: chỉ có CommandBus (NATS + cdc_jobs) chạy thật; `QueryBus` khai báo mà **không có implementation nào**. Có 33/~90 file api import `internal/infra/*` (59 lần gọi `messaging.WithMetadata`); có 4 file api chạy SQL trực tiếp; nhiều logic nghiệp vụ nằm trong `infra/persistence` (ApprovalService, ProvisioningOrchestrator, ShadowAutomator, AlertManager). Auth chỉ là verify JWT HMAC bằng shared secret, không gọi sang cdc-auth-service lúc runtime, và có **bypass `dev-token`**.

> Lưu ý: trong lúc chạy em lỡ ghi 1 file nháp vào `.../brain/df7b6768.../scratch/handlers.txt` (đây là output awk map handler→command; file này không nằm trong codebase). Ngoài file đó em không ghi/sửa gì khác.

---
## 1. Entrypoints & wiring
| Binary | Vai trò | Ref |
|---|---|---|
| `cmd/server` | Service chính: config → OTel → `server.New` → `Start` → shutdown khi nhận SIGTERM | `cmd/server/main.go:28-92` |
| `cmd/sync_v2` | Tool 1 lần: đọc `model.TableRegistry` rồi gọi `SourceObjectV2SyncService.SyncFromLegacy` | `cmd/sync_v2/main.go:23-35` |
| `cmd/dump` | Tool debug: dump `ReconReadRepo.ListLatest` ra JSON | `cmd/dump/dump.go:25-34` |

⚠️ `cmd/dump/dump.go:17` và `cmd/sync_v2/main.go:16` **hardcode DSN có user/password** (`postgres://***:***@localhost:5433/cdc_dw`).

**Trình tự `server.New`** (`internal/server/server.go`):
1. Infra: Postgres control-plane `:87`; `migrate.Run` `:93`; shadowDB riêng (tuỳ chọn) `:97-110`; seed qua `bootstrap.EnsureDefaultShadowConnection` `:112` và `EnsureDefaultMasterConnection` `:116`; NATS + `EnsureStreams` `:120-126`; Redis `:128`; `KafkaConnectClient` `:133`; `PromClient` `:134`.
2. ~35 repos (đều là `infra/persistence/*`) `:145-180`.
3. Core: `cmdBus := messaging.NewNATSCommandBus(conn, jobRepo)` `:185`; `natsPublisher` `:186`; `reloadPublisher` `:187`; `rpcClient` `:188`; `ddlPubNew` (`infra/messaging/nats`) `:189`; ShadowAutomator, ActivityLogger, ApprovalService, ProvisioningOrchestrator (đều trong infra/persistence) `:191-194`; AlertManager `:196`; StuckJobReaper `:197`; health Collector `:198-207`.
4. Query handlers được inject thẳng, không qua bus `:212-246`.
5. Command handlers: 21 async subject `RegisterSubject` `:268-287`, 32 sync handler `RegisterSync` `:290-321`, và 10 master-rule handler inject trực tiếp vào API, không qua bus `:251-265`.
6. API handlers `:326-364`. Một số handler nhận cả `db`/infra concrete, ví dụ `NewSnapshotProgressHandler(..., db)` `:362`, `NewReconciliationHandler(..., db, ...)` `:364`, `persistencePkg.NewTransformJobRepo(db)` `:343`.
7. Fiber: CORS `*` `:373-378`; `OtelPropagator` `:379`; `HttpTracer` `:380`; swagger `:381`; `router.SetupRoutes` `:383`. Goroutine nền: healthCollector, auditLogger, alert resolver, stuckJobReaper `:406-447`.

**Buses (`internal/app/ports`):**
- `CommandBus{Execute(SyncCommand), Dispatch(AsyncCommand)}` `ports/command_bus.go:85-88`. Sealed bằng mixin `:51-59`.
- Impl `natsCommandBus`: `Execute` ghi job vào `cdc_jobs` rồi chạy handler in-process `infra/messaging/nats_command_bus.go:157-196`. `Dispatch` ghi `cdc_jobs` rồi `PublishMsg` lên subject đã map, header có job_id/correlation/OTel `:206-255`.
- `QueryBus{Ask}` `ports/query_bus.go:14-16`: **không có impl và không có caller** (grep `QueryBus|.Ask(` chỉ ra đúng file này).
- `ports.MasterDDLPublisher` `ports/master_ddl_publisher.go:8-12`: **dead**, không ai dùng. Cái thật đang dùng là `domain/master/publisher.go:6-14`.
- `bootstrap/registry_mirror.go:8-190` bị comment-out toàn bộ (dead), chỉ còn helper.

**Middleware:** JWTAuth `middleware/jwt.go:12`; RequireRole `jwt.go:72`; RequireOpsAdmin/RequireAnyRole `rbac.go:107,133`; Idempotency (Redis) `idempotency.go`; RateLimit `ratelimit.go`; Audit (ghi `cdc_system.admin_actions`) `audit.go:1-5`; DeprecateLegacyAPIPath `router.go:119`; OtelPropagator; HttpTracer.

---
## 2. Kiến trúc layer thực tế
**Pattern dự định:** Hexagonal + CQRS. `ports.Command` có Type/Validate (`command_bus.go:12-21`). Async command được publish nguyên `json.Marshal(cmd)`. Domain chỉ import domain (`internal/domain/*` chỉ import `domain/governance`). Repo interface nằm ở `app/ports/repository.go` (19 interface, `:32-295`) và `domain/*/repository.go`.

**Trace A — sync, inject trực tiếp (master drop-column):**
1. Route `POST /api/v1/master-mapping-rules/:id/drop-column`: `router.go:416` (group admin `:373`, JWT `:130`).
2. API: `MasterMappingRuleHandler.DropColumn` `api/master/master_mapping_rule_handler.go:262-273` → `h.dropColCmdHandler.Handle(ctx, masterCmd.DropColumnCommand{ID})` `:268`.
3. Command: `app/commands/master/drop_column.go:33-87`. Invariant status=rejected và không phải cột `_` `:39-44`. Check conflict `:47`. Saga 2 bước `:65-81`.
4. Domain: entity `governance.MasterRule` `domain/governance/master_rule.go:12`. Port `master.MasterRuleRepository` `domain/master/repository.go:9-33`. Port `master.MasterDDLPublisher` `domain/master/publisher.go:6`.
5. Infra persistence: `masterRuleRepoGorm` `infra/persistence/master/master_mapping_rule_repo_gorm.go:21`; `GetByID` `:112`; `CheckColumnConflict` `:291`; `UpdateInMasterStatus` (UPDATE `cdc_system.mapping_rule_master`) `:373-375`.
6. Infra messaging: `NatsMasterDDLPublisher.PublishMasterDropColumn` `infra/messaging/nats/master_ddl_publisher.go:47-71`. Publish `cdc.cmd.master-alter-column` `:66`, rồi chờ reply qua `SubscribeSync(cdc.evt.master-drop.<ns>)` `:52-53`.
7. Wiring: `server.go:262` và `:355`.

**Trace B — async qua bus (recon check):**
1. `POST /api/reconciliation/check/:table` `router.go:189` (destructive chain `:132`).
2. `ReconciliationHandler.TriggerCheck` `api/recon/reconciliation_handler_commands.go:19`. Dựng `reconCmd.ReconCheckCommand` `:55`, gọi `messaging.WithMetadata` (**vi phạm: import infra**) rồi `h.bus.Dispatch` `:72`.
3. `ReconCheckCommand` `app/commands/recon/recon_check.go:27-55`, Type `"recon.check"`.
4. Bus map sang `cdc.cmd.recon-check` `server.go:268`. Insert `cdc_jobs` rồi publish `nats_command_bus.go:221-246`.

**Vi phạm (đếm trên file non-test):**
| Loại | Số lượng | Ví dụ path:line |
|---|---|---|
| api import `infra/messaging` | 29 file (59 lần, tất cả là `messaging.WithMetadata`) | `api/recon/reconciliation_handler_commands.go:11`, `api/master/master_registry_handler_create.go:7`, `api/source/registry_handler_register.go:7` |
| api import `infra/persistence*` | 6 file | `api/source/registry_handler.go:5-6`, `api/source/source_object_actions_handler.go:17`, `api/master/master_transmute_job_handler.go:9`, `api/system/alerts_handler.go:9`, `api/governance/schema_change_handler.go:7`, `api/governance/provisioning_handler.go:7` |
| api import `infra/http`/`observability` | 2 | `api/source/system_connectors_handler.go:11` (Stop/ResetOffsets gọi thẳng KafkaConnect `:278,295-299`), `api/system/system_health_handler.go:11` |
| api import `pkgs/natsconn`, nats.go, rediscache | 3 file | `api/system/nats_monitor_handler.go:15,18`, `api/system/nats_monitor_governance.go:9`, `api/system/system_health_handler.go:13` |
| **api chạy SQL thô / gorm** | 6 file import gorm, 13 call | `api/recon/reconciliation_handler_delete_report.go:23` (DELETE `cdc_reconciliation_report`), `api/scheduler/snapshot_progress_handler.go:65,93` (UPDATE `snapshot_progress`), `api/system/introspection_handler_postgres.go:134-206`, `api/source/system_connectors_credentials.go:68,82`, `api/master/master_registry_handler_connections_probe.go:289`, `..._probe_sql.go:120,164` |
| api import `internal/model` | 5 file | `api/source/registry_handler_bulk.go:9`, `api/source/sources_handler.go:5`, `api/scheduler/schedule_handler.go:12` |
| api publish NATS trực tiếp, bypass bus | 16 call | `api/source/source_object_actions_handler.go:784`, `api/source/registry_handler_transform.go:31`, `api/master/master_registry_handler_connections.go:307`, `api/scheduler/snapshot_progress_handler.go:77,115`, RPC `api/system/introspection_handler.go:49-419`, `introspection_index_handler.go:40,102,143` |
| app import infra | 3 file | `app/commands/source/register_registry.go:14` (persistenceShadow), `app/commands/scheduler/run_now.go:13` (infra/persistence), `app/commands/source/debezium_connector.go:13` và `app/queries/system/get_metric_health.go:8` (infra/http) |
| app dùng gorm/SQL thô | 2 file | `app/commands/shadow/delete_shadow_binding.go:12,98-207`, `app/commands/master/delete_master_binding.go:11,70-74` (`*gorm.DB` được inject `server.go:302,313`) |
| `internal/model` dùng ngoài infra/persistence | 23 file (app/commands 8, app/queries 8, app/ports 1, api 4, infra/observability 2, cmd 1) | `app/ports/repository.go:23-26` (port lộ GORM struct), `app/queries/recon/recon_reader.go:8`, `infra/observability/system_health_queries.go:15-16` |
| logic nghiệp vụ nằm trong infra | — | `persistenceGov.NewApprovalService` `server.go:193`, `NewProvisioningOrchestrator` (giữ `*nats.Conn`, publish trực tiếp `provisioning_orchestrator.go:250`), state machine `provisioning_state_machine.go:44-49`, `persistenceShadow.NewShadowAutomator` `server.go:191` |
| Handler bỏ qua hẳn app layer | ≈57/170 method không gọi cmd/query/bus | ví dụ connections CRUD `master_registry_handler_connections.go:28,60,169,332`, provisioning `provisioning_handler.go:77-167`, sensitive fields `sensitive_fields_handler.go:24-80`, `sources_handler.go:44` (`h.repo.Upsert`) |

Phân bố 170 method handler: 52 dùng `h.bus.*`, 14 gọi command handler trực tiếp, 44 gọi query handler, 16 publish/RPC trực tiếp, ≈57 không qua app layer.

---
## 3. API surface (prefix `/api`, đa số mirror thêm `/api/v1/...` qua `dual`/`registerDestructive` `router.go:136-181`)
Public, không JWT: `/health`, `/ready`, `/api/system/health`, `/api/system/nats/overview` `router.go:122-127`.

**Recon** (destructive = OpsAdmin + Idempotency + Audit):
- POST `/reconciliation/check`, `/check/:table` `router.go:188-189` → `TriggerCheckAll` `reconciliation_handler_commands.go:101` / `TriggerCheck` `:19` → `ReconCheckCommand` (async `recon.check`).
- POST `/reconciliation/heal`, `/heal/:table` `router.go:190-191` → `TriggerHeal` `reconciliation_handler_heal.go:18` → `ReconHealCommand` (async `recon.heal`).
- POST `/reconciliation/execute-heal` `router.go:192` → `TriggerExecuteHeal` `reconciliation_handler_execute_heal.go:31` → `ExecuteHealCommand` (async `execute-heal`).
- POST `/reconciliation/prune[/:table]` `router.go:193-194` → `TriggerPrune` `reconciliation_handler_commands.go:183` → ReconCheckCommand.
- POST `/failed-sync-logs/:id/retry` `router.go:195` → `RetryFailedLog` `reconciliation_handler_retry.go:12` → `RetryFailedCommand` (async) + `MarkFailedLogRetryingCommand` (sync).
- POST `/tools/reset-debezium-offset` `router.go:196` → `reconciliation_handler_tools.go:13` → `DebeziumSignalCommand`. POST `/tools/trigger-snapshot/:table` `router.go:197` → `:37` → `DebeziumSnapshotCommand`.
- POST `/recon/backfill-source-ts` `router.go:198` → `reconciliation_handler_backfill.go:15` → `ReconBackfillSourceTsCommand`.
- DELETE `/reconciliation/report/:id` `router.go:199` → `reconciliation_handler_delete_report.go:11` → **raw SQL**.
- GET `/reconciliation/report` `router.go:335` → `LatestReport` `reconciliation_handler_reports.go:13` (ListLatestReports Q); `/report/:table` `:336` → `TableHistory` `:80`; `/failed-sync-logs` `:337` → `:98`; `/report/:table/unhealed` `:339` → `reconciliation_handler_execute_heal.go:17`; `/reconciliation/jobs/active` `:340` → `reconciliation_handler_reports.go:130`; `/recon/backfill-source-ts/status` `:338` → `reconciliation_handler_backfill.go:57`.

**Source / Registry / Connectors:**
- Destructive `/v1/system/connectors` (create) `router.go:206` → `system_connectors_handler.go:133` → `CreateSystemConnectorCommand` (sync). restart/tasks restart/pause/resume `:207-210` → `:101,114,309,314` → `LifecycleSystemConnectorCommand`. stop/offsets `:211-212` → `:273,286` → **gọi thẳng KafkaConnect client**. recover `:213` → `:444` → `RecoverSystemConnectorCommand` (async). bridge-oplog `:214` → `:515` → `BridgeOplogCommand` (async). PATCH config `:216` → `:198` → UpdateSystemConnectorConfig. DELETE `:218` → `:254` → DeleteSystemConnector.
- GET `/v1/system/connectors[/:name]`, `/connector-plugins` `router.go:299-301` → `:66,75,92` → query qua KafkaConnect.
- GET `/sync/health` `:285` → `registry_handler_read.go:52`. GET `/v1/source-objects`, `/stats`, `/registry/:registry_id`, `/v1/shadow-bindings` `:286-289` → `source_objects_handler.go:95,73,176,133` (query).
- dispatch-status/transform-status `:290-293` → `source_object_actions_handler.go:424,651`, `registry_handler_dispatch.go:13`, `registry_handler_transform.go:54`.
- `/v1/sources` GET/POST/GET :id (và alias `/v1/registry/connections`) `:294-298` → `sources_handler.go:34,44,93`. POST đi thẳng `repo.Upsert`.
- Admin: POST `/v1/source-objects/register` `:385` → `registry_handler_register.go:19` → `RegisterRegistryCommand` + `CreateDefaultColumnsCommand` + `V2SyncCommand` + `RestartDebeziumCommand`. register-batch `:386` → `registry_handler_bulk.go:19`. PATCH `/v1/source-objects/:id` `:387` → `source_object_actions_handler.go:117` → `UpdateSourceObjectV2Command`. create-default-columns / scan-fields / standardize / detect-timestamp-field / snapshot-v2 `:389-393` → `:197,352,290,482,548` (async). transform `:394` → `:712` → **publish thẳng `cdc.cmd.batch-transform`**. transform-cancel `:395` → `:863`. transform-job-status `:396` → `:817`.
- V1 bridge `/v1/source-objects/registry/:id*` `:399-404` → `registry_handler_update.go:18` (UpdateRegistryCommand), `registry_handler_tools_scan.go:16,52`, `registry_handler_transform.go:14` (publish thẳng), `registry_handler_tools_columns.go:14,57`.

**Shadow / Mapping v2:**
- DELETE `/v1/shadow-bindings/:id` `router.go:241` → `shadow_binding_actions_handler.go:88` → `DeleteShadowBindingCommand` (sync). PATCH `:388` → `:45` → `UpdateShadowBindingCommand`.
- POST `/v1/mapping-rules/preview` `:251` → `mapping_preview_handler.go:35` → PreviewMapping Q.
- GET `/mapping-rules` `:319` → `mapping_rule_handler_list.go:29`. POST `/mapping-rules` `:378` → `mapping_rule_handler_create.go:25` → `CreateMappingRuleCommand`. PATCH `/mapping-rules/batch` `:379` → `mapping_rule_handler_batch.go:34` → UpdateMappingRule + AlterColumn + Backfill. PATCH `/mapping-rules/:id` `:380` → `mapping_rule_handler_commands.go:53` → `UpdateMappingRuleCommand`. POST `/mapping-rules/reload` `:381` → `:20` → `reloadPub.PublishReload` trực tiếp. POST `/mapping-rules/:id/backfill` `:382` → `mapping_rule_handler_batch.go:19` → `BackfillCommand`.

**Master:**
- Destructive: POST `/v1/masters` `router.go:234` → `master_registry_handler_create.go:16` → `CreateMasterCommand` (sync `master.create`). approve/reject `:235-236` → `master_registry_handler_approve.go:17,73` → Approve/RejectMasterCommand. toggle-active `:237` → `master_registry_handler_toggle.go:14`. swap `:238` → `master_registry_handler_swap.go:27` → `MasterSwapCommand` (async). PATCH spec `:239` → `master_registry_handler_update_spec.go:18`. DELETE `/v1/masters/:id` `:240` → `master_registry_handler_delete.go:12` → DeleteMasterBindingCommand.
- GET `/v1/masters` `:305` → `master_registry_handler_read.go:21` (ListMasters Q). master-connections list/test/create/update/delete `:304-315` → `master_registry_handler_connections.go:28,60,169,332`, `..._probe.go:159` → **gọi repo/DB trực tiếp**, update có publish `connection.config.changed` `:307`.
- transmute-job-status / transmute-cancel `:316-317,418-419` → `master_transmute_job_handler.go:36,98` (persistence trực tiếp).
- `/v1/master-mapping-rules*` `:318,407-416` → `master_mapping_rule_handler.go:47-314` → 10 command handler gọi trực tiếp (Save/Delete/UpdateColumnMetadata/ApproveDDL/ApproveDDLAll/DropColumn/DropRejected/SyncFromShadow/BatchUpdate).

**Governance:**
- `/schema-changes/pending|history` `router.go:278-279` → `schema_change_handler.go:42,148`. approve/reject `:376-377` → `:78,114` → `persistenceGov.ApprovalService` (infra).
- `/v1/schema-proposals` GET `:280-281` → `schema_proposal_handler.go:34,48`. approve/reject `:245-246` → `:74,137` → bus Approve/RejectSchemaProposalCommand.
- `/v1/sensitive-fields` CRUD `:282,422-424` → `sensitive_fields_handler.go:24-80` (repo trực tiếp).
- `/v1/cms/sources/:id/provisioning*` `:263-269` → `provisioning_handler.go:77-167` → ProvisioningOrchestrator (infra).
- `/audit/qa-summary|gaps|metric-health` `:432-434` → `audit_handler.go:43,54,81` (query).

**Scheduler:**
- Wizard: POST `/v1/wizard/sessions` `router.go:437` → `wizard_handler.go:49`. PATCH `:438` → `:92`. execute `:244` → `:132` (sync). GET `:322-323` → `:74,158`.
- Transmute `/v1/schedules` POST/run-now/PATCH/DELETE/GET `:247-250,324` → `transmute_schedule_handler.go:78,204,158,184,59` (command handler trực tiếp).
- `/worker-schedule` GET/PATCH/POST/DELETE `:325,427-429` → `schedule_handler.go:167,192,295,252` (bus).
- `/snapshot-progress` GET `:326`, pause/resume `:252-253` → `snapshot_progress_handler.go:23,56,82` → **raw gorm + publish thẳng**.
- `/jobs/:id` `:367` → `job_handler.go:33`.

**System:**
- restart-debezium `router.go:205` → `system_health_handler.go:92` → `RestartDebeziumCommand`.
- NATS stream purge/config/delete msg/stream/consumer `:222-230` → `nats_monitor_handler.go:423-608` (JetStream API trực tiếp). GET overview/messages `:329-330` → `:60,222`.
- activity-log list/stats/daily-flow/clear `:331-334,201` → `activity_log_handler.go:56,94,165,120`.
- alerts ack/silence `:257-258` → `alerts_handler.go:96,141` (bus). active/silenced/history `:360-362`.
- introspection `:343-356` → NATS RPC sang worker, xem mục 4. `DiscoverPostgresTablesPost` → `introspection_handler_postgres.go:70` (mở DB và query thẳng).

---
## 4. Outbound contracts
**NATS async qua CommandBus** (payload = `json.Marshal(<XCommand>)`, header job_id/correlation; `server.go:268-287`):

`cdc.cmd.recon-check` (ReconCheckCommand `recon_check.go:27`), `cdc.cmd.recon-heal`, `cdc.cmd.execute-heal`, `cdc.cmd.retry-failed`, `cdc.cmd.recon-backfill-source-ts`, `cdc.cmd.debezium-signal`, `cdc.cmd.debezium-snapshot`, `cdc.cmd.restart-debezium`, `cdc.cmd.create-default-columns`, `cdc.cmd.standardize`, `cdc.cmd.scan-fields`, `cdc.cmd.detect-timestamp-field`, `cdc.cmd.backfill`, `cdc.cmd.alter-column`, `cdc.cmd.transmute`, `cdc.cmd.master-create`, `cdc.cmd.master-swap`, `cdc.cmd.snapshot.v2`, `cdc.cmd.system-connector-recover`, `cdc.cmd.bridge-oplog`. Struct cụ thể nằm trong `app/commands/{recon,source,master}/*` (ví dụ `recon_async.go:29-174`).

**NATS publish trực tiếp, bypass bus:**
- `schema.config.reload` {table,user_id,action,field,timestamp} `infra/messaging/nats_publisher.go:72-89`. Bản trùng ở `pkgs/natsconn/nats_client.go:141-151`.
- `cdc.cmd.master-alter-column` (action alter/drop/drop-table, kèm reply_to `cdc.evt.master-{alter,drop,drop-table}.<ns>`) `master_ddl_publisher.go:25-91`; `cdc.cmd.alter-column` (shadow drop-table) `:103-117`; `cdc.cmd.master-create` `:135` và `app/commands/governance/approve_master.go:129`.
- `cdc.cmd.batch-transform` `source_object_actions_handler.go:784`, `registry_handler_transform.go:31` (payload thô = tên bảng).
- `cdc.cmd.snapshot.v2` `snapshot_progress_handler.go:115`; `cdc.control.commands.<sourceObjectID>` "pause" `:76-77`.
- `connection.config.changed` (connection_code) `master_registry_handler_connections.go:307`.
- Provisioning: `cdc.cmd.shadow.bind`, `cdc.cmd.master.bind`, `cdc.cmd.discover`, `cdc.cmd.schedule.enable` `provisioning_state_machine.go:45-48` → `provisioning_orchestrator.go:250`.

**NATS request-reply (RPC) sang worker** (`natsPublisher.Request`, SubscribeSync trên inbox `cdc.evt.*.<ns>`, `nats_publisher.go:39-66`): `cdc.cmd.introspect.mongo.databases|collections`, `cdc.cmd.scan-raw-data`, `cdc.cmd.introspect`, `cdc.cmd.scan-array` (`introspection_handler.go:49-419`), `cdc.cmd.introspect-indexes|create-index|drop-index` (`introspection_index_handler.go:40,102,143`).

**Subscribe:** không có subscriber lâu dài. Chỉ có `SubscribeSync` tạm cho reply. CMS **không nghe** `cdc.evt.*.completed`.

**JetStream streams do CMS tạo:** `CDC_EVENTS` (`cdc.goopay.>`), `SCHEMA_DRIFT`, `SCHEMA_CONFIG`, `CDC_TRANSMUTE` (`cdc.cmd.transmute-shadow`) `pkgs/natsconn/nats_client.go:25-73`.

⚠️ Comment `ports/command_bus.go:37-39` nói "JobMonitor (worker side) closes the row", nhưng `centralized-data-service` **không có file .go nào tham chiếu `cdc_jobs`**. JobMonitor thật chỉ update `transmute_schedule` (`centralized-data-service/internal/service/master/job_monitor.go:89`). Vì vậy row async trong `cdc_jobs` có lẽ chỉ được đóng bởi `StuckJobReaper` → `failed` (`stuck_job_reaper.go:118`). Đây là **HYPOTHESIS**, cần trace runtime.

**HTTP outbound:**
- Kafka Connect REST: `infra/http/kafka_connect.go:88-223` (list/status/config/plugins/restart/create/update/delete/stop/offsets/lifecycle).
- Prometheus và worker `/metrics`: `prom_client.go:229`.
- Health probes (Worker, KafkaConnect, NATS monitor, Kafka exporter): `observability/probes/deps.go:43`, `kafka_lag.go:51`.
- NATS monitor HTTP: `api/system/nats_monitor_handler.go:651`.
- **Không có** HTTP call nào sang centralized-data-service admin-api hay cdc-auth-service (chỉ có 1 comment ở `infra/http/doc.go:2`).

**Kafka:** không có Kafka client (go.mod không có sarama/franz/confluent). Kafka chỉ xuất hiện dưới dạng config Debezium signal (`signal.kafka.topic=cdc.signal.commands`, bootstrap) đẩy vào connector config `system_connectors_credentials.go:104-106`, `config/config.go:55-57`.

---
## 5. Data stores
- DB control-plane (`cdc`) + shadowDB tuỳ chọn (`server.go:87,97-110`). Redis dùng cho idempotency, ratelimit, health cache, alerts.
- Migrations: runner `internal/migrate/runner.go:33` (tracker `cdc_system.schema_migrations` `:21`), SQL embed `migrations/embed.go`. Thư mục: `schema/{core,cdc_system_model(029-082),recon_dlq,registry,worker,ops,ids,partitioning,audit_security}`, `seed/*`. `cluster/*` **không** apply, chỉ để DBA chạy tay (`runner.go:65-73`).

**Bảng theo domain** (đều trong schema `cdc_system`, rút từ grep trong `infra/persistence/<d>`):
| Domain | Bảng |
|---|---|
| source | source_object_registry, connection_registry, sources, cdc_table_registry, mapping_rule_v2, cdc_mapping_rules, shadow_binding, master_binding, transform_jobs |
| shadow | shadow_binding, source_object_registry, master_binding, cdc_table_registry, cdc_activity_log, fn ensure_shadow_sonyflake_trigger, cùng DDL trên shadowDB (ShadowAutomator) |
| master | master_binding, mapping_rule_master, mapping_rule_v2, connection_registry, sync_runtime_state, transmute_jobs, sensitive_fields, cdc_jobs |
| recon | cdc_reconciliation_report, failed_sync_logs, recon_jobs, recon_lag, cdc_recon_smoke_result, cdc_table_registry |
| governance | schema_proposal, pending_fields, schema_changes_log, sensitive_fields, qa_gap_state, qa_criterion_rating, cdc_mapping_rules, fn append_step_log_capped (provisioning) |
| scheduler | cdc_wizard_sessions, cdc_worker_schedule, transmute_schedule, snapshot_progress, cdc_jobs |
| system | cdc_activity_log, cdc_alerts, admin_actions (middleware) |

**Bảng có thể dùng chung với centralized-data-service** (số file .go bên CDS có tham chiếu): source_object_registry=31, connection_registry=29, shadow_binding=27, master_binding=22, sources=18, mapping_rule_v2=13, cdc_activity_log=10, mapping_rule_master=9, transmute_schedule=8, cdc_table_registry=8, failed_sync_logs=7, recon_jobs=4, sensitive_fields=4, cdc_reconciliation_report=3; transform_jobs, transmute_jobs, snapshot_progress, sync_runtime_state, cdc_mapping_rules, pending_fields mỗi bảng 1–2.

Chỉ CMS dùng (CDS = 0 file): `cdc_jobs`, `admin_actions`, `cdc_wizard_sessions`, `qa_gap_state`.

---
## 6. Domain vs `internal/model`
| Domain (`internal/domain`) | GORM model (`internal/model`) | Overlap |
|---|---|---|
| `governance.Rule` (mapping_rule_v2) `rule.go:60` | `governance.MappingRule` → `cdc_mapping_rules` (V1) `model/governance/mapping_rule.go:5,25` | 2 thế hệ mapping V1/V2 |
| `governance.MasterRule` `master_rule.go:12` | (không có) | Có **2 interface MasterRuleRepository** trùng: `domain/governance/master_rule.go:52` và `domain/master/repository.go:9` (CMS dùng cái sau) |
| `source.Object` (source_object_registry) `object.go:30` | `source.TableRegistry` → `cdc_table_registry` `table_registry.go:5,44`; `source.Source` → `sources` `source.go:9,29` | V1 registry vs V2 object; sync qua `SourceObjectV2SyncService` (`server.go:147`) |
| `recon.Report` `report.go:24`, `recon.FailedLog` `failed_log.go:18` | `recon.ReconciliationReport` `reconciliation_report.go:8,60`, `recon.FailedSyncLog` `failed_sync_log.go:8,30` | Trùng 1-1. Domain recon chỉ được dùng ở `ports/repository.go` và `failed_sync_log_repo_gorm.go`; queries recon dùng thẳng reconmodel (`app/queries/recon/recon_reader.go:8`) |
| `master.Binding` `binding.go:23` | (không có, repo dùng SQL) | — |
| `system.Job` `job.go:26` | `jobRow` trong persistence `job_repo_gorm.go:49` | Đúng chuẩn adapter |
| `scheduler.TransmuteScheduleHeader`, `shadow.ShadowSampleRow` | `scheduler.WizardSession`, `WorkerSchedule`; `system.ActivityLog`, `Alert`; `governance.PendingField/SchemaChangeLog/SchemaProposal/SensitiveField/QA*` | Phần lớn **không có domain tương ứng** → GORM struct bị dùng làm entity luôn, lộ ra tới tận `app/ports/repository.go:23-26` |

---
## 7. Auth
- `api.Use(middleware.JWTAuth(cfg))` `router.go:130`. Verify HMAC bằng `cfg.JWT.Secret` `jwt.go:28-33` và set Locals `username/role/roles` `:44-51`. Không gọi cdc-auth-service lúc runtime. Token do cdc-auth-service phát (HS256, cùng secret) `cdc-auth-service/internal/service/auth_service.go:153-163` (**suy luận**: shared secret qua config, cần xác nhận ở deploy).
- ⚠️ **Bypass:** `Bearer dev-token` → admin, không có guard theo môi trường `jwt.go:21-26`.
- RBAC: group `shared` admin|operator `router.go:275`; `admin` `:373`; destructive = `RequireOpsAdmin` (ops-admin hoặc admin) `rbac.go:133-134` + fallback env `ADMIN_USERS` `rbac.go:38-51`. Idempotency-Key (Redis) + Audit vào `admin_actions` + ratelimit restart 3/h `router.go:78-95`.
- Config bắt buộc có jwt.secret; ở production chặn placeholder `config/config.go:215-218`.
- Endpoint public không auth: `/api/system/health`, `/api/system/nats/overview` `router.go:123-127`. CORS `*` `server.go:374`.

---
```mermaid
flowchart TD
  R["router + middleware (JWT/RBAC/Idem/Audit)"] --> API["internal/api/*"]
  API --> BUS["app/ports CommandBus"]
  API --> CMDH["app/commands (direct Handle)"]
  API --> QRY["app/queries"]
  BUS --> CMDH
  CMDH --> DOM["internal/domain/*"]
  CMDH --> PORTS["app/ports + domain ports"]
  QRY --> PORTS
  PERS["infra/persistence"] -.implements.-> PORTS
  MSG["infra/messaging (natsCommandBus, publishers)"] -.implements.-> BUS
  HTTPX["infra/http (KafkaConnect, Prom)"] -.implements.-> PORTS
  PERS --> PG[("Postgres cdc_system + shadowDB")]
  MSG --> NATS[("NATS → centralized-data-service worker")]
  HTTPX --> KC[("Kafka Connect / Prometheus")]
  API -.->|"VIOLATION WithMetadata x59"| MSG
  API -.->|"VIOLATION raw SQL/gorm"| PG
  API -.->|"VIOLATION direct publish/RPC"| NATS
  API -.->|"VIOLATION"| PERS
  API -.->|"VIOLATION Stop/ResetOffsets"| HTTPX
  API -.->|"VIOLATION"| MODEL["internal/model (GORM)"]
  CMDH -.->|"VIOLATION gorm tx"| PG
  CMDH -.->|"VIOLATION"| PERS
  CMDH -.->|"VIOLATION"| HTTPX
  PORTS -.->|"VIOLATION ports expose GORM"| MODEL
  QRY -.->|"VIOLATION"| MODEL
  PERS --> MODEL
  PERS -.->|"business logic in infra: Approval/Provisioning/ShadowAutomator"| NATS
```

**UNKNOWN / HYPOTHESIS:**
- Ai đóng row `cdc_jobs` khi job async chạy thành công: chưa tìm thấy trong code (HYPOTHESIS: chỉ reaper đánh failed).
- Secret JWT có thật sự giống nhau giữa auth và cms ở deploy không: UNKNOWN.
- Payload cụ thể của các RPC introspection: chưa đọc chi tiết.

Lesson áp dụng: no-guess (mọi claim đều gắn path:line), tách VERIFIED/HYPOTHESIS. Công cụ đã dùng: run_command (grep/sed/awk read-only), view_file.