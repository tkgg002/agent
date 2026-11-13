# Tổng quan hệ thống `centralized-data-service` (READ-ONLY, working tree nhánh recon-heal)
Phạm vi: 271 file Go không phải test, ngoài docs/scratch/vendor. Mọi tham chiếu đều ở dạng `path:line`, tương đối với repo root. `UNKNOWN` = chưa xác minh được trong code.

## 1. Entrypoints & wiring

**Config chung**: `config.NewConfig()` `config/config.go:164`. File config lấy từ env `cfgPath`/`CFG_PATH`, mặc định `./config/config-local.yml` (`config/config.go:167-172`), có `AutomaticEnv`. `AppConfig` gồm SystemDB/ShadowDB/MasterDB/ReadReplica/Nats/Redis/Kafka/MongoDB/Debezium/ConnectionOverrides (`config/config.go:18-36`). DB role: `cdc`(control-plane), `shadow`, `dest` (`pkgs/database/multi.go:19-22`).

### cmd/admin-api (`cmd/admin-api/main.go`)
- Mở GORM tới control-plane (`ADMIN_DB_URL` hoặc SystemDB) (`:32-33`), kết nối NATS thô (`:47`), lấy Debezium URL và Schema Registry URL (`:56-64`), listen mặc định `127.0.0.1:8090` (`:69`). Bắt buộc có `ADMIN_API_TOKEN`, trừ khi `ADMIN_API_DEV=true` (`:70-79`).
- `admin.NewServer` (`internal/admin/server.go:97`) chạy gin, middleware gồm otel, body-limit, rate-limit, auth. Chỉ có 2 route: `GET /healthz` (`internal/admin/server.go:121`) và `POST /v2/sources/register` → `RegisterHandler.HandleRegister` (`internal/handler/source/source_register.go:53`) → `SourceRegistrationService` (`internal/service/source/registration.go`):
  - UPSERT `cdc_system.source_object_registry` (`registration.go:190`) và `cdc_system.shadow_binding` (`registration.go:221`);
  - PUT config connector Debezium (`registration.go:273`);
  - publish `cdc.cmd.kafka.refresh-topics` (`registration.go:129`).
- Không có goroutine/ticker nền nào ngoài HTTP server.

### cmd/worker (`cmd/worker/main.go`): binary chính
- Khởi tạo idgen sonyflake (`:52`) và OTel + log bridge (`:56-91`), rồi gọi `server.NewWorkerServer` (`:93`). Metrics Prometheus riêng ở cổng 9090 (`:103`). `srv.Start()` (`:108`) làm 3 việc: chạy mọi hook `onStart` trong goroutine riêng, `startScheduler()`, rồi chạy Fiber listen (`internal/server/server.go:61-75`).
- **Wiring trong `NewWorkerServer`** (`internal/server/server_setup.go:51-574`):
  - Hạ tầng:
    - `database.Registry` gồm control-plane `db` và `shadowDB` (nếu lỗi thì fallback về `db`) (`:52-65`), cộng read replica (`:70`).
    - `NatsClient` (`:79`) gọi `EnsureStreams(prefix+".>")` (`:99`).
    - Redis (`:103`); Mongo client dùng chung (`:118-122`).
  - Repos đều nhận `db` (control-plane) (`:124-136`): TableRegistry, ReconciliationReport, ActivityLog, MappingRule(v1), PendingField, ConnectionRegistry, SourceObjectRegistry, ShadowBinding, MappingRuleV2, SyncRuntimeState, FailedSyncLog, MasterBinding, TransmuteSchedule.
  - `MetadataRegistryService`, inject connectionRepo, sourceObjectRepo, shadowBindingRepo, mappingRuleV2Repo (`:139`).
  - `MaskingService` → được inject vào: SchemaInspector, BatchBuffer×3, DLQHandler, DynamicMapper, SysOps, Healer (`:141-148`, cùng các dòng sau).
  - `ConnectionManager` (`:146`); `SchemaInspector` (pendingRepo, redis, nats) (`:147`).
  - `SchemaAdapter(shadowDB)` (`:151`).
  - 3 `BatchBuffer`: cdc / snapshot / heal (`:152-159`, factory ở `server_setup_buffers.go:19-45`), mỗi cái nhận db, schemaAdapter, connMgr, masking, NATS+JS, failedSyncLogRepo.
  - Recon:
    - `ReconSourceAgent(mongo)` và `ReconDestAgent(shadowDB)` (`:161-162`);
    - `ReconCore` (`:165`), inject thêm registrySvc, connMgr, NATS (`:166-170`), masterAgent và plane DBs (`:172-175`);
    - hook startup `ReapOrphanRunsFromDeadInstances` (`:177-187`).
  - DLQ: `DLQHandler` (`:195`) **được tạo nhưng không subscribe/không dùng ở đâu**. `DLQStateMachine` là hook onStart (`:198-199`).
  - Shadow ingest:
    - `DynamicMapper` (`:201`);
    - 3 `EventHandler` (cdc/snapshot/heal), mỗi cái gắn 1 BatchBuffer + `ChildExplodeService` (`:204-206`, `server_setup_buffers.go:47-59`);
    - `ConsumerPool` JetStream cho mỗi prefix, durable `<groupId>-<prefix>` (`:209-218`).
  - Core subscriptions: `schema.config.reload` (`:220`), `connection.config.changed` (`:226`).
  - Command handlers, queue group `cdc-worker-group` (`:275-300`): Discover, MongoDiscover, Scan, Sync, BatchTransform, SchemaDDL, Bridge.
  - Master:
    - `TypeResolver`, `MasterDDLGenerator`, `TransmuterModule`, `TransmuteHandler` (`:302-308`);
    - `TransmuteConsumerPool` (stream `CDC_TRANSMUTE`, consumer `transmute-shadow-consumer-v2`, AckWait 30s, MaxDeliver 5) (`:311-335`);
    - `TransmuteScheduler` onStart (`:343-344`); `JobMonitor` (`:346-347`); `MasterDDLHandler` (`:349-352`).
  - Provisioning: chỉ bật khi `PROVISIONING_ORCHESTRATOR_ENABLED=1` (`:354-370`), gồm ProvisioningOrchestrator.RecoveryLoop, ShadowBindHandler, ScheduleEnableHandler.
  - Debezium signal: `DebeziumSignalClient` (Kafka) (`:377-386`) và SysOps "signal-only" (`:392-404`).
  - `SnapshotRunner` (`:406-419`).
  - `ReconHealer` (`:421`). Nếu có mongo: `BackfillSourceTsService`, `TimestampDetector`, `FullCountAggregator` onStart (`:423-432`).
  - Recon handlers: ReconBase → CheckHandler, ExecuteHealHandler (dùng healEventHandler), CheckHealHandler (dùng healer), SysOpsHandler (`:440-488`).
  - `ChunkStreamBucketEngine` + `ReconJobWorker` onStart (`:495-509`).
  - Index governance (`:512-516`); `PartitionDropper` onStart (`:519-520`).
  - `KafkaConsumer`, chỉ khi `kafka.enabled` (`:522-540`): inject cdcEventHandler, registrySvc, PerSourcePool, SchemaValidator, masking. Được set vào `snapshotRunner.SetTopicController` (`:534`).
  - Fiber routes: `/health`, `/ready`, `/metrics`, `/api/reconciliation/jobs/active|:job_id`, `/api/v1/internal/stats` (`:549-567`).
  - `SetupSchedulerJobs` (`:571`).
- **Scheduler** (`internal/server/server_scheduler.go:17-174`):
  - Ngủ 5s, rồi seed `cdc_worker_schedule` (transform 5m, field-scan 60m, partition-check 1440m, reconcile 30m) (`:24-29`).
  - Ticker 2s (`:60`), có lock Redis `cdc:lock:scheduler_poll` (`:67`), atomic claim bằng UPDATE `last_run_at` (`:96-107`).
  - Ưu tiên `scheduleJobs` V2 (`:133`), fallback legacy `jobs` (`:152`).
- **Jobs** (`internal/server/server_jobs.go`):
  - `bridge`: no-op "legacy bridge retired" (`:62-70`).
  - `transform`: publish `cdc.cmd.batch-transform` cho mỗi entry active (`:73-118`).
  - `field-scan`: publish `cdc.cmd.scan-raw-data` hoặc `cdc.cmd.periodic-scan` (`:121-151`).
  - `partition-check`: gọi `ensure_cdc_partition` (`:154-177`).
  - `reconcile`: `ReapStaleRuns` + `CheckAllUnified` (smoke) (`:182-238`).
  - `source_count` (`:241-248`); `confluence` (`:251-258`).

### cmd/sinkworker (`cmd/sinkworker/main.go`): ingest Debezium→shadow độc lập, không qua NATS
- Hằng `consumerGroup="cdc-v125-sink-worker"`, `topicPattern=^cdc\.goopay\..*` (hardcode) (`:30-31`).
- Gọi `cdc_system.claim_machine_id` (`:94`). Goroutine heartbeat 30s qua `heartbeat_machine_id`; mất fencing token thì tự cancel (`:113`, `:260-293`).
- NATS chỉ dùng để publish trigger transmute (`:115`).
- `SchemaManager` + `MaskingService` + `sinkworker.New` (`:122-144`).
- Discover topic một lần lúc start (`:150`); `kafka.NewReader`, commit thủ công (`:160-170`); vòng lặp Fetch→`HandleMessage`→Commit (`:184-218`).

## 2. Pipeline end-to-end

```mermaid
flowchart LR
  SRC[(Source Mongo/PG/MySQL/SFTP)] --> DBZ[Debezium / Kafka Connect]
  DBZ --> K[(Kafka topics cdc.goopay.*)]
  DBZ -. "Debezium NATS sink? UNKNOWN" .-> NJS[(JetStream CDC_EVENTS prefix.>)]
  K --> KC[KafkaConsumer worker]
  K --> SW[sinkworker]
  NJS --> CP[ConsumerPool]
  KC --> EH[EventHandler.HandleRaw]
  CP --> EH
  SNAP[SnapshotRunner Mongo cursor] --> EH
  HEALA[ExecuteHeal SegA fetch by IDs] --> EH
  EH --> BB[BatchBuffer cdc/snapshot/heal]
  BB --> SH[(Shadow tables shadowDB)]
  SW --> SH
  BRG[BridgeHandler oplog] --> SH
  BB -- fail --> FSL[(cdc_system.failed_sync_logs)]
  FSL --> DLQSM[DLQStateMachine] --> NJS
  BB --> TS{{cdc.cmd.transmute-shadow JS CDC_TRANSMUTE}}
  SW --> TS
  BRG --> TS
  TS --> TH[TransmuteHandler] --> TM[TransmuterModule] --> MS[(Master tables dest DB/Mongo/MySQL/CH)]
  TSCH[TransmuteScheduler] --> TC{{cdc.cmd.transmute}} --> TH
  TH --> EVT{{cdc.evt.transmute.completed}} --> JM[JobMonitor]
  RC[recon-check] --> RJ{{cdc.event.recon.job_created}} --> RJW[ReconJobWorker+ChunkStreamBucketEngine]
  RJW --> RPT[(recon_jobs / cdc_reconciliation_report)]
  SMK[reconcile job CheckAllUnified] --> SMR[(cdc_recon_smoke_result)]
  RPT --> EXH[execute-heal] --> HEALA
  EXH -- SegB --> TC
```

| Stage | Entry (path:line) | Input | Output | Package |
|---|---|---|---|---|
| CDC qua NATS | `ConsumerPool.worker` `internal/handler/shadow/consumer_pool.go:77`; PullSubscribe tại `:45`; Fetch 100/5s `:87`, Nak/Ack `:111-114` | JetStream `CDC_EVENTS`, subject `<prefix>.>` | `EventHandler.Handle` `event_handler.go:243` | handler/shadow |
| CDC qua Kafka | `KafkaConsumer.Start` `kafka_consumer.go:183`; `processMessage` `:531` (decode Avro inline `:560-575`) → `HandleRaw` `:748` | Topic khớp `kafka.topicPrefix`, group `kafka.groupId` (`kafka_consumer.go:168-172`) | EventHandler; commit offset an toàn qua `StatefulWatermarkTracker` (`watermark_tracker.go:66-70`) | handler/shadow |
| Route + map | `EventHandler.HandleRaw` `event_handler.go:95` → `processEvent` `:248`; adapter SFTP `:135` | JSON CDCEvent (`internal/model/shadow/cdc_event.go:5`) | `UpsertRecord` (`cdc_event.go:36`) vào BatchBuffer; route lấy từ `MetadataRegistryService.ResolveSourceRoutes` (`metadata_registry_service.go:469`) | handler/shadow, service/shadow (DynamicMapper, ChildExplode) |
| Ghi shadow | `BatchBuffer.Add` `batch_buffer.go:181` → `flushWithContextMode` `:348` → `batchUpsert` `:454`; soft delete `:786` | UpsertRecord | Bảng shadow (SQL do `SchemaAdapter.BuildBatchUpsertSQLsInSchema` sinh, `service/shadow/schema_adapter.go:535`); tự ALTER/CREATE qua `PrepareForCDCInsert*` (`schema_adapter.go:134-263`) | handler/shadow, service/shadow |
| DLQ ghi | `batchUpsertWithRetry` `batch_buffer_dlq.go:23` → `evacuateFailedChunkToDLQ` `:60` (CreateInBatches `:93`); Kafka DLQ `dlq_helper.go:54` | Lỗi DB | `cdc_system.failed_sync_logs` | handler/shadow |
| DLQ replay | `DLQStateMachine.Start` `handler/dlq/dlq_state_machine.go:94`; `RunOnce` `:131` (SELECT status pending/failed/retrying, `:156-157`); `retryOne` `:211` | failed_sync_logs | Publish NATS với subject = `row.KafkaTopic`, fallback `cdc.dlq` (`:234-244`), đi lại vào CDC_EVENTS | handler/dlq |
| Trigger transmute | `BatchBuffer.publishTransmuteTrigger` `batch_buffer_fanout.go:23` (JS `:104-125`, fallback core `:143-147`) | Batch đã flush xong | `cdc.cmd.transmute-shadow` (Nats-Msg-Id dedup) | handler/shadow |
| Transmute | `TransmuteConsumerPool` → `HandleTransmuteShadowTask` `transmute_handler.go:724`; `HandleTransmuteShadow` `:132` (payload inline struct `:154-163`); `HandleTransmute` `:293` (`TransmuteRequest` `:259`) | `cdc.cmd.transmute-shadow`, `.core-fallback`, `cdc.cmd.transmute` | `TransmuterModule.Run` `service/master/transmuter.go:167` / `RunEvents` `:653` → `bulkUpsertMaster` `:1613`; rule lấy từ `mapping_rule_master ⋈ mapping_rule_v2` (`transmuter.go:926-927`); kết quả `cdc.result.transmute` (`transmute_handler.go:487`), `cdc.evt.transmute.completed` (`:472`) | handler/master, service/master (+ transmute/, transmuter_mongo/mysql/clickhouse) |
| Lịch transmute | `TransmuteScheduler.tick` `service/master/transmute_scheduler.go:103` (SELECT `transmute_schedule` … FOR UPDATE SKIP LOCKED `:118-130`) | `cdc_system.transmute_schedule` | `cdc.cmd.transmute` (`:171`) | service/master |
| Đóng vòng | `JobMonitor.HandleCompleted` `service/master/job_monitor.go:72` | `cdc.evt.transmute.completed` | Cập nhật transmute_schedule; publish `cdc.evt.provisioning.step_completed` (`:164`) | service/master |
| Master DDL | `MasterDDLHandler.HandleMasterCreate` `master_ddl_handler.go:137`, `AlterColumn` `:43`, `Swap` `:232` | `cdc.cmd.master-create` / `master.bind` / `master-alter-column` / `master-swap` | DDL trên master DB; `cdc.result.master-create` (`:219`), `cdc.evt.master-swap.completed` (`:267`, `:280`) | service/master (master_ddl_generator*) |
| Snapshot | `SnapshotRunner.Handle` `handler/orchestration/snapshot_runner_handler.go:145` → `runSnapshot` `:196`. SFTP: tạo topic + Kafka Connect (`:225-300`). Mongo: claim `snapshot_progress` `:460`, cursor `:672` → `snapshotEventHandler.HandleRaw` `:822`, subject tổng hợp `cdc.snapshot.<db>.<coll>` `:588`; pause/resume qua `cdc.control.commands.<id>` `:631` | `cdc.cmd.snapshot.v2` (`snapshotV2Payload` `snapshot_runner_utils.go:34`) | Shadow (qua snapshotBatchBuffer); `cdc_system.snapshot_progress` | handler/orchestration |
| Debezium signal | `SysOpsHandler.HandleDebeziumSignal` `recon_sysops_handler.go:151` → `DebeziumSignalClient` (`service/governance/debezium_signal.go`, topic mặc định `cdc.signal.commands` `:33`) | `cdc.cmd.debezium-signal`, `cdc.cmd.debezium-snapshot` | Kafka signal topic, gây incremental snapshot | handler/recon, service/governance |
| Bridge | `BridgeHandler.HandleBridgeOplog` `handler/source/bridge_handler.go:130` → `processBridge` `:157`; Mongo change stream `bridge_mongo.go:26`; `batchUpsert` riêng `bridge_upsert_fanout.go:199`; `triggerTransmute` `:340` | `cdc.cmd.bridge-oplog` | Shadow, `cdc.cmd.transmute-shadow` (`:378`), `cdc.result.bridge-oplog` | handler/source |
| Sinkworker | `SinkWorker.HandleMessage` `internal/sinkworker/worker.go:176`; resolve topic→shadow qua shadow_binding `:109-173`; `EnsureShadowTableInSchema` `:318`; `upsertWithFencing` `:466` | Kafka `^cdc\.goopay\..*` | Shadow; `cdc.cmd.transmute-shadow` (core NATS, không JS) `:397` | sinkworker |
| Recon smoke | Job `reconcile` → `ReconCore.CheckAllUnified` `service/recon/recon_smoke.go:681` → `RunTotalOnlyA` `:154` / `RunTotalOnlyB` `:427` | Source count, shadow, master | `cdc_recon_smoke_result`, `cdc_recon_cycle_summary` (`model/recon/recon_smoke_model.go:45,64`) | service/recon |
| Recon job (async) | `CheckHandler.HandleReconCheck` `handler/recon/recon_check_handler.go:109`: INSERT `recon_jobs` `:212`, publish `:235-241` → `ReconJobWorker` (queue `recon-job-workers`, `recon_job_worker.go:166`) → `HandleJobEvent` `:199` → `engine.ExecuteSegment` `:278` → `ChunkStreamBucketEngine.Execute` `recon_stream_bucket_engine.go:233` / `ExecuteSegment` `:569` | `cdc.cmd.recon-check` → `cdc.event.recon.job_created` (`ReconJobCreatedEvent` `recon_job_worker.go:44`) | `recon_jobs`, `cdc_reconciliation_report` | handler/recon, service/recon |
| Tier A (Source↔Shadow) | `RunOrphanPrune` `recon_tier_a.go:445`, `RunHashWindowCheck` `:636`, `RunDeepCheck` `:1032`; Tier A publish `cdc.cmd.transmute-shadow` `:575-580` | | report | service/recon |
| Tier B / Segment B (Shadow↔Master) | `RunHashWindowCheckB` `recon_tier_b.go:27`, `RunDeepCheckB` `:335`, `RunRowDiffB` `:604`, `RunSegmentB(For)` `:743/751`, `CheckAllSegmentB` `:763`; `ResolveSegmentBPKPair` `recon_engine_segment_b.go:46` | | report | service/recon |
| Heal propose | `CheckHealHandler.HandleReconHeal` `recon_check_heal_handler.go:56`: SegA `:119` (gọi `RunHashWindowCheck` `:142`), SegB `:194` (`RunSegmentBFor` `:211`), legacy `HealWindow` `:302` (`service/recon/recon_heal.go:630` → `schemaAdapter.UpsertRecord`, khoảng `:725`) | `cdc.cmd.recon-heal` | | handler/recon |
| Heal execute | `ExecuteHealHandler.HandleExecuteHeal` `recon_execute_heal_handler.go:79` → `processSingleReport` `:168`. SegA: `fetchAndWriteChunked` `:377` → `FetchAndWriteByIDsWithAccepted` `recon_heal_fetch.go:26` → healEventHandler.HandleRaw (subject `cdc.snapshot.*` `:85`), sau đó `cdc.cmd.transmute-shadow` `:360`. SegB: `executeHealSegB` `recon_execute_heal_segb.go:26` → `cdc.cmd.transmute` `:774/784`; prune master Mongo `:397` | `cdc.cmd.execute-heal` + `cdc_reconciliation_report` | Shadow/Master, report | handler/recon |
| Schema drift/DDL | `SchemaInspector.InspectEvent` `service/governance/schema_inspector.go:90` → `pending_fields`, `schema.drift.detected` `:289`. `SchemaDDLHandler.HandleStandardize` `:72`, `HandleAlterColumn` `:616`, `HandleCreateDefaultColumns` `:127`, `HandleDropGinIndex` `:543` | `cdc.cmd.standardize` / `alter-column` / … | Shadow DDL, `cdc.result.alter-column` | service/governance, handler/shadow |
| Source count | `SourceCountRunner.Run` `service/sourcecount/source_count_runner.go:75`: count SQL `:281`, Mongo `:317`, file `:403` | Schedule `source_count` | UPSERT `cdc_system.daily_source_counts` `:220` | service/sourcecount |
| Confluence | `ConfluenceReportRunner.Run` `service/confluence/confluence_runner.go:604` (đọc `source_object_registry` `:230`, `cdc_activity_log` `:375`) | Schedule `confluence` | Trang Confluence qua `pkgs/confluence/client.go:210` | service/confluence |

**Ghi chú**: ai publish vào subject `cdc.goopay.>` của CDC_EVENTS thì không thấy trong repo. Nguồn hợp lệ duy nhất thấy được là DLQ replay. Nguồn production (Debezium Server NATS sink?) là **UNKNOWN**.

## 3. Messaging contracts

**JetStream streams** (`pkgs/natsconn/nats_client.go`, ensure tại `nats_stream_provision.go:16`):

| Stream | Subjects | Cấu hình | Ref |
|---|---|---|---|
| CDC_EVENTS | `<topicPrefix>.>` | Limits, 24h, 512MiB | `:24-36` |
| SCHEMA_DRIFT | `schema.drift.detected` | 7 ngày | `:39-50` |
| SCHEMA_CONFIG | `schema.config.reload` | | `:53-64` |
| CDC_TRANSMUTE | `cdc.cmd.transmute-shadow` | WorkQueue, 1h, dedup 2m | `:68-80` |

**JS consumers**:
- `<groupId>-<prefix>`: pull, CDC_EVENTS (`server_setup.go:211-212`).
- `transmute-shadow-consumer-v2`: CDC_TRANSMUTE (`server_setup.go:319-325`, AddStream/AddConsumer tại `transmute_consumer_pool.go:138/182`).

**NATS Subscribe** (tất cả trong `internal/server/server_setup.go`; Q = queue `cdc-worker-group` trừ khi ghi khác):
- `schema.config.reload` `:220` và `connection.config.changed` `:226` (plain).
- Q: `cdc.cmd.standardize` `:277`, `cdc.cmd.discover` `:278`, `cdc.cmd.introspect` `:279`, `cdc.cmd.scan-raw-data` `:283`, `cdc.cmd.scan-array` `:284`, `cdc.cmd.periodic-scan` `:285`, `cdc.cmd.batch-transform` `:287`, `cdc.cmd.drop-gin-index` `:288`, `cdc.cmd.create-default-columns` `:289`, `cdc.cmd.introspect.mongo.databases|collections` `:290-291`, `cdc.cmd.scan-fields` `:292`, `cdc.cmd.sync-register` `:293`, `cdc.cmd.sync-state` `:294`, `cdc.cmd.restart-debezium` `:295`, `cdc.cmd.system-connector-recover` và `cdc.cmd.system-connector.recover` `:296-297`, `cdc.cmd.bridge-oplog` `:299`, `cdc.cmd.alter-column` `:300`.
- Queue `transmute-workers`: `cdc.cmd.transmute` `:337`, `cdc.cmd.transmute-shadow.core-fallback` `:338`, `cdc.cmd.transmute-shadow` (chỉ khi không có JS) `:340`.
- `cdc.evt.transmute.completed` (JobMonitor, plain) `:347`.
- Q: `cdc.cmd.master-create` `:350`, `cdc.cmd.master-alter-column` `:351`, `cdc.cmd.master-swap` `:352`.
- Chỉ khi bật provisioning: `cdc.evt.provisioning.step_completed` `:357`, Q `cdc.cmd.shadow.bind` `:367`, `cdc.cmd.schedule.enable` `:368`, `cdc.cmd.master.bind` `:369`.
- Q: `cdc.cmd.debezium-signal` / `cdc.cmd.debezium-snapshot` `:403-404`.
- Queue `cdc-snapshot-runner`: `cdc.cmd.snapshot.v2` `:419`.
- Q: `cdc.cmd.recon-check` `:482`, `cdc.cmd.recon-job-status` `:483`, `cdc.cmd.recon-heal` `:484`, `cdc.cmd.execute-heal` `:485`, `cdc.cmd.retry-failed` `:486`, `cdc.cmd.recon-backfill-source-ts` `:487`, `cdc.cmd.detect-timestamp-field` `:488`, `cdc.cmd.introspect-indexes` / `create-index` / `drop-index` `:514-516`.
- `cdc.cmd.kafka.refresh-topics` `:537`.
- Ngoài server_setup: `cdc.event.recon.job_created` (queue `recon-job-workers`, `service/recon/recon_job_worker.go:166`); `cdc.control.commands.<id>` (động, `snapshot_runner_handler.go:631`).

**NATS Publish**:
- Command:
  - `cdc.cmd.batch-transform`, `cdc.cmd.scan-raw-data`, `cdc.cmd.periodic-scan` (`server_jobs.go:97/131/141`).
  - `cdc.cmd.transmute-shadow`: `batch_buffer_fanout.go:104`, `batch_transform_handler.go:543`, `bridge_upsert_fanout.go:378`, `recon_execute_heal_handler.go:360`, `recon_tier_a.go:575`, `sinkworker/worker.go:397`. Payload: struct inline `transmute_handler.go:154-163` với `TransmuteEventItem` (`model/shadow/cdc_event.go:67`).
  - `cdc.cmd.transmute` (`TransmuteRequest` `transmute_handler.go:259`): `transmute_scheduler.go:171`, `transmute_handler.go:247`, `recon_execute_heal_segb.go:774/784`.
  - `cdc.cmd.master-create` (`service/metadata/mapping_utils.go:263`).
  - `cdc.cmd.shadow.bind` / `master.bind` / `discover` / `schedule.enable` (`service/orchestration/provisioning_orchestrator.go:64-67`).
  - `cdc.cmd.kafka.refresh-topics` (`registration.go:129`).
- Event:
  - `cdc.evt.transmute.completed` (`transmute_handler.go:472`).
  - `cdc.evt.provisioning.step_completed` (`handler/base/provisioning_emit.go:79`, `job_monitor.go:164`).
  - `cdc.evt.master-swap.completed` (`master_ddl_handler.go:267`).
  - `cdc.event.recon.job_created` (`recon_check_handler.go:235`).
  - `schema.drift.detected` (`schema_inspector.go:289`).
- Result: tiền tố `cdc.result.*` (`handler/base/base_handler.go:99`). Các subject: alter-column, bridge-oplog, introspect-indexes, master-create, recon-backfill-source-ts, restart-debezium, scan-array, scan-fields, scan-raw-data, sync-register, sync-state, system-connector-recover, transmute (xem các dòng trong `sync_handler.go`, `scan_handler.go`, …). Thêm `cdc.reply.create-default-columns` (`schema_ddl_handler.go:498`) và reply-to động.
- DLQ: `cdc.dlq` (`dlq_handler.go:24,180`); replay về subject gốc (`dlq_handler.go:251`, `dlq_state_machine.go:234-244`).
- **Code chết (không được wire)**:
  - `moleculer.event.<table>` (`event_bridge.go:203`; `NewEventBridge` không có caller).
  - `cdc.pipeline.paused` / `cdc.pipeline.resume` (`dlq_circuit_breaker.go:31-32`; không có caller `SetDLQCircuitBreaker`).
  - `cdc.snapshot.resume` (`governance/wal_monitor.go:23`; `NewWALMonitor` không có caller).

**Kafka**:
- Worker group `kafka.groupId`, topic theo `kafka.topicPrefix` (`kafka_consumer.go:168-172`).
- Sinkworker group `cdc-v125-sink-worker`, topic `^cdc\.goopay\..*` (`cmd/sinkworker/main.go:30-31`).
- Signal topic `cdc.signal.commands` (`debezium_signal.go:33`).
- SnapshotRunner tự tạo topic SFTP (`snapshot_runner_handler.go:952`).
- ⚠️ Nếu worker (kafka.enabled) và sinkworker cùng đọc `cdc.goopay.*` bằng 2 group khác nhau thì mỗi event được ghi shadow 2 lần. Cần xác nhận cấu hình deploy (**UNKNOWN**).

## 4. Data stores
- **Postgres control-plane `cdc_system.*`** (model TableName):
  - Registry & binding: `connection_registry` (`model/source/connection_registry.go:27`), `source_object_registry` (`model/source/source_object_registry.go:51`), `cdc_table_registry` (legacy V1, `model/source/table_registry.go:106`), `shadow_binding` (`model/shadow/shadow_binding.go:24`), `master_binding` (`model/master/master_binding.go:30`).
  - Mapping: `cdc_mapping_rules` (v1, `model/master/mapping_rule.go:27`), `mapping_rule_v2` (`model/master/mapping_rule_v2.go:30`), `mapping_rule_master` (không có model, raw SQL ở `transmuter.go:926`, `master_ddl_generator.go:117,448`, `scan_service.go:358,445`, `recon_tier_b.go:637`).
  - Runtime & lịch: `sync_runtime_state`, `transmute_schedule`, `cdc_worker_schedule`.
  - Recon: `cdc_reconciliation_report`, `cdc_recon_smoke_result`, `cdc_recon_cycle_summary`, `recon_jobs` (`repository/recon_job_repo.go:33`), `recon_runs`, `recon_lag`.
  - Job: `transmute_jobs`, `transform_jobs`.
  - DLQ & lỗi: `failed_sync_logs`, `snapshot_dlq`.
  - Governance & log: `pending_fields`, `sensitive_fields`, `schema_changes_log`, `cdc_activity_log`.
  - Khác: `daily_source_counts`, `snapshot_progress`, `sources` (raw_config_sanitized, `snapshot_runner_handler.go:245`), `cdc_alerts`, `worker_registry`.
  - Function: `claim_machine_id`, `heartbeat_machine_id`, `ensure_cdc_partition`, `append_step_log_capped`.
- **Shadow DB (role `shadow`)**: các bảng shadow động, schema/table lấy từ shadow_binding. Ghi bởi BatchBuffer, BridgeHandler, ReconHealer, SinkWorker (sinkworker dùng **control-plane `db`** chứ không phải shadowDB, `cmd/sinkworker/main.go:84,122`). Các cột hệ thống: `model/shadow/system_columns.go`, `service/shadow/system_columns.go`.
- **Master/dest**: Postgres role `dest` + Mongo/MySQL/ClickHouse qua ConnectionManager (`service/master/transmuter_{mongo,mysql,clickhouse}.go`, `master_ddl_generator_*`).
- **MongoDB**: client dùng chung `cfg.MongoDB.URL` (`server_setup.go:118`) cho recon source agent, healer, backfill, timestamp detector, full count. Snapshot/bridge mở client theo connection (`snapshot_runner_handler.go:498`).
- **Redis**:
  - lock scheduler (`server_scheduler.go:67`);
  - cache `schema:*` (`server_setup.go:222`);
  - truyền vào SchemaInspector và ReconCore (`:147,165`).
- **Bảng dùng chung với cdc-cms-service**: đã grep `../cdc-cms-service`, chỉ xác minh ở mức file. Có tham chiếu tới connection_registry, source_object_registry, shadow_binding, master_binding, mapping_rule_v2, mapping_rule_master, cdc_table_registry, cdc_worker_schedule, transmute_schedule, recon_jobs, cdc_reconciliation_report, failed_sync_logs, cdc_activity_log, snapshot_progress, sources, sensitive_fields, pending_fields, cdc_recon_smoke_result, cdc_mapping_rules, transmute_jobs, transform_jobs, schema_changes_log. Ví dụ `cdc-cms-service/internal/infra/persistence/master/mapping_rule_repo_gorm.go`, `.../scheduler/transmute_schedule_repository_gorm.go`. `daily_source_counts`: CMS không có tham chiếu Go nào.

## 5. Package responsibility map
- **server**: DI container `WorkerServer` (`server.go:18`), job registry, scheduler poller.
- **admin**: gin admin API, chỉ có endpoint register source (`admin/server.go`).
- **activity**: taxonomy TriggeredBy (`activity/taxonomy.go`).
- **naming**: chuẩn hoá tên (`naming/naming.go`).
- **sinkworker**: consumer Kafka→shadow độc lập (SinkWorker, SchemaManager, avro_decode, sql_builder).
- **handler/base**: BaseHandler publish result/reply, emit provisioning.
- **handler/shadow**: EventHandler, BatchBuffer (flush/DLQ/fanout), ConsumerPool, KafkaConsumer, WatermarkTracker, SchemaDDLHandler, BatchTransformHandler, ShadowBindHandler. Chết: EventBridge, DLQCircuitBreaker.
- **handler/source**: Discover/MongoDiscover/ScanFields, SyncHandler (Debezium connector ops), BridgeHandler, RegisterHandler (gin).
- **handler/scan**: ScanHandler (scan-raw-data / array / periodic).
- **handler/master**: TransmuteHandler (+ debounce, poison-pill split, event-date split), TransmuteConsumerPool, MasterDDLHandler.
- **handler/orchestration**: SnapshotRunner, ProvisioningHandler, ScheduleEnableHandler.
- **handler/recon**: ReconBase, CheckHandler, CheckHealHandler, ExecuteHealHandler (SegA/SegB, dialect SQL), SysOpsHandler, JobHandler.
- **handler/dlq**: DLQHandler (không dùng), DLQStateMachine.
- **handler/governance**: IndexHandler.
- **service/source**: MetadataRegistryService (cache registry trong RAM, dựng từ connection/source_object/shadow_binding/mapping_v2, `metadata_registry_service.go:82`), ConnectionManager, ScanService, DiscoverService, SourceRegistrationService, ConnectorResolver, MongoIntrospection. Thêm RegistryService legacy (`registry_service.go:62`, không có caller production), BridgeService (không có caller), `test_routing2.go` (hàm `TestRoutingLogic` trong file non-test, **có DSN Postgres kèm user/pass hardcode**, `test_routing2.go:18`; mask: `postgres://***:***@localhost:5433/cdc_dw`).
- **service/shadow**: SchemaAdapter (DDL + SQL upsert), DynamicMapper, ChildExplode, TypeResolver. EnrichmentService không có caller.
- **service/master**:
  - TransmuterModule và các strategy (`transmute/` copy_1_to_1, flatten, multi_shadow);
  - MasterDDLGenerator (pg/mysql/mongo/ch), TransmuteScheduler, JobMonitor;
  - Mongo converters, `transform_registry` (whitelist `transform_fn`).
- **service/recon**: ReconCore (`recon_engine.go:237`), smoke, tier A/B, segment B, stream, ChunkStreamBucketEngine, ReconJobWorker, ReconHealer, Source/Dest agents (mongo/sql dialects/hash). Thêm DLQWorker (`dlq_worker.go:106`, **không có caller**), BinaryDrillDownEngine (`recon_bisection_engine.go:83`, không có caller), recon_legacy/recon_dest_legacy.
- **service/governance**: Masking, SchemaInspector/Validator, IndexManager + drivers, PartitionDropper, FullCountAggregator, BackfillSourceTs, TimestampDetector, DebeziumSignalClient, ActivityLogger. WALMonitor không có caller.
- **service/orchestration**: ProvisioningOrchestrator (feature flag).
- **service/metadata**: interface `MetadataRegistry`, `ResolvedSourceRoute` (`metadata/metadata_registry.go:10,36`), mapping utils.
- **service/sourcecount**, **service/confluence**: job báo cáo.
- **repository/***: repo GORM cho các model trên. ReconJob/TransformJob/TransmuteJob nằm ở root `repository/`, struct định nghĩa trong file repo, không thuộc `model/`.
- **model/***: struct bảng (source/shadow/master/recon/system).
- **pkgs**:
  - database: multi-role registry, pgx, per-source pool;
  - natsconn: client + stream provision;
  - kafka/avro, rediscache, mongodb, crypto/aes, idgen/sonyflake, metrics, observability (otel, log bridge);
  - confluence client, sqlutil, utils.

**Chồng chéo / trùng lặp** (VERIFIED theo file:line):
1. **Registry kép**: `cdc_table_registry` (V1, `TableRegistryRepo`) song song với `source_object_registry` + `shadow_binding`. Repo phải JOIN qua `so.legacy_registry_id` (`repository/source/table_registry_repo.go:91-94`). Recon/scheduler vẫn dùng `TableRegistry` (`server_jobs.go:83`).
2. **Ba bảng mapping**: `cdc_mapping_rules` (v1: discover/schemaDDL/RegistryService), `mapping_rule_v2`, `mapping_rule_master` (raw SQL, không model/repo).
3. **Resolver PK/timestamp phân tán**:
   - `extractPrimaryKey` (`event_handler.go:499`), `detectPrimaryKey` (`batch_transform_handler.go:477`), `resolvePGPrimaryKeyCol` (`recon_heal_postgres.go:131`), `resolveMasterPKFromSpec` (`recon_execute_heal_segb.go:715`);
   - `parsePKFromSpec` (`master_ddl_generator_utils.go:10`), `resolvePKFields` (`recon_stream_bucket_engine.go:368`), `getPrimaryKeyColumn` (`recon_stream.go:439`), `resolveShadowPKField` (`recon_smoke_guard.go:15`), `ResolveSegmentBPKPair` (`recon_engine_segment_b.go:46`);
   - Timestamp: `resolveTimestampField` (`recon_query.go:21`), `ParseTransformSpecTimestampField` (`recon_engine_segment_b.go:70`), `ParseTimestampField` (`transmute/multi_shadow.go:27`), `resolveMaster/SourceTimestampField` (`master_ddl_generator_mongo.go:26,83`).
4. **Ba đường ghi shadow có upsert riêng**: `BatchBuffer.batchUpsert` (`batch_buffer.go:454`), `BridgeHandler.batchUpsert` (`bridge_upsert_fanout.go:199`), SinkWorker (`worker.go:466` + `SchemaManager` trùng vai `SchemaAdapter`). Riêng `ReconHealer.applyOne` gọi `SchemaAdapter.UpsertRecord`.
5. **Trigger transmute nhân bản**: `publishTransmuteTrigger` + `resolvePostIngestSchedule` có ở cả `batch_buffer_fanout.go:23,160` và `sinkworker/worker.go:372,407`; thêm bản thứ 3 ở `bridge_upsert_fanout.go:340`.
6. **Avro decode 3–4 bản**: `kafka_consumer.go:560`, `pkgs/kafka/avro.go`, `sinkworker/avro_decode.go`, `handler/shadow/avro_helper.go`.
7. **DLQ 3 bản**: DLQHandler (không wire), DLQStateMachine (đang chạy), `service/recon/dlq_worker.go` (không có caller).
8. **Recon legacy và mới**:
   - Đường sync tier A/B trong CheckHandler bị comment (`recon_check_handler.go:257-310`). Vì vậy `executeCheckAll*`/`executeCheckSegment*` (`:390-458`) thành code chết.
   - Chỉ còn đường async job (ChunkStreamBucketEngine). `reconCoreChunkAdapter` (`recon_job_worker.go:127`) còn tồn tại cho constructor cũ.
   - Tier A/B vẫn được CheckHealHandler dùng (`recon_check_heal_handler.go:142,211`).
9. **SQL ngoài repository**: số lượng `.Raw/.Exec` gồm service/master 122, service/recon 95, service/source 36, service/shadow 36, service/governance 35, handler/recon 23, handler/orchestration 17, sinkworker 15, handler/shadow 12, **so với repository 12**. Ví dụ handler chạy SQL trực tiếp: `snapshot_runner_handler.go:230,245,598`; `recon_check_handler.go:192` (chuỗi INSERT dựng bằng Sprintf chỉ để làm span attr, ghi DB thật đi qua `jobRepo.Create` `:212`).
10. **Trùng hằng/handler**:
    - Hằng `SubjectProvisioningStepCompleted` khai báo 3 lần (`handler/base/provisioning_emit.go:13`, `handler/orchestration/provisioning_handler.go:26`, `service/master/job_monitor.go:34`).
    - Hai subject recover connector (`server_setup.go:296-297`).
    - `HandleProvisionShadowTable` (`schema_ddl_handler.go:452`) không được subscribe.

## 6. Domain glossary
- **Source connection**: `ConnectionRegistry` (`model/source/connection_registry.go:8`), gồm ConnectionCode, RoleType, EngineType, SecretRef, OptionsJSON. Prefix topic được map về connection (`metadata_registry_service.go:571`).
- **Source object registry**: `SourceObjectRegistry` (`model/source/source_object_registry.go:8`), một bảng/collection nguồn: ObjectCode, SourceConnectionID, PK/Timestamp field, CDCMode, SyncEngine.
- **Table registry (legacy)**: `TableRegistry` (`model/source/table_registry.go:9`). SourceDB/Table/URL, TargetTable, PK, TimestampField, ExpectedFields, SensitiveFields, FullSource/DestCount. Vẫn là đơn vị làm việc của recon và scheduler.
- **Shadow binding**: `ShadowBinding` (`model/shadow/shadow_binding.go:5`). Ánh xạ source object → bảng shadow (Schema/Table/FQN, WriteMode, DDLStatus, ParentBindingID/ExplodePath cho child explode).
- **Master binding**: `MasterBinding` (`model/master/master_binding.go:8`). Ánh xạ shadow → bảng master (MasterConnectionID, TransformType/Spec, SchemaStatus approved/rejected).
- **Mapping rule**:
  - v1 `MappingRule` (`model/master/mapping_rule.go:5`), theo SourceTable;
  - v2 `MappingRuleV2` (`model/master/mapping_rule_v2.go:5`), theo SourceObject/Shadow/MasterBinding, có SourcePath, TransformFn;
  - `mapping_rule_master` là bảng nối master_binding ↔ mapping_v2 kèm `in_master` (raw SQL).
- **Transmute**: chuyển dữ liệu shadow → master. `TransmuterModule` (`service/master/transmuter.go:33`), `Run` `:167` (incremental/full theo cursor), `RunEvents` `:653` (event 1:1), `TransmuteResult` `:115`. Lịch ở `TransmuteSchedule` (`model/master/transmute_schedule.go:8`), job ở `TransmuteJob` (`repository/transmute_job_repo.go:29`).
- **Batch transform**: `cdc.cmd.batch-transform` → `BatchTransformHandler.HandleBatchTransform` (`batch_transform_handler.go:89`), job ở `transform_jobs`. Ngữ nghĩa chính xác ("transform `_raw_data` thành cột shadow") là **HYPOTHESIS**, chưa đọc thân hàm.
- **Recon job/report**: `ReconJob` (`repository/recon_job_repo.go:11`, PENDING→…, progress, diff counts), `ReconciliationReport` (`model/recon/reconciliation_report.go:8`, Segment, Missing/Stale IDs, các trường Healed*/Pruned*, MasterSchema/Table, window).
- **Segment**: A = Source↔Shadow; B = Shadow↔Master (`SegmentSourceShadow`/`SegmentShadowMaster` dùng ở `recon_check_heal_handler.go:184,258`).
- **Smoke**: kiểm tra tổng số lượng định kỳ. `SmokeResult` / `CycleSummary` (`model/recon/recon_smoke_model.go:9,45-64`), do `CheckAllUnified` (`recon_smoke.go:681`) sinh ra.
- **Heal**: sửa drift. Có 3 dạng:
  - SegA: lấy lại bản ghi nguồn theo ID rồi đi lại đường shadow ingest;
  - SegB: phát `cdc.cmd.transmute` với `_source_ids`;
  - legacy: `ReconHealer.HealWindow` (`recon_heal.go:630`) upsert trực tiếp.
  Report được claim/release qua `reportRepo.ReleaseHealClaim` (`recon_execute_heal_handler.go:179`).
- **DLQ**: `FailedSyncLog` (`model/shadow/failed_sync_log.go:8`, Status, RetryCount/MaxRetries/NextRetryAt, KafkaTopic/Offset); `SnapshotDLQ` (`model/recon/snapshot_dlq.go:8`).
- **Watermark/cursor**:
  - Kafka offset an toàn: `StatefulWatermarkTracker.CalculateSafeCommitOffsetsWithPending` (`handler/shadow/watermark_tracker.go:70`); marker DLQ/NoOp (`batch_buffer.go:213,231`).
  - Cursor transmute: `fetchShadowBatch(cursor)` (`transmuter.go:987`), lưu trạng thái trong `SyncRuntimeState.LastCursorJSON` / `LastSourceTs` (`model/master/sync_runtime_state.go:17-18`).
  - Snapshot: `snapshot_progress` + cluster time (`snapshot_runner_handler.go:583`).
  - Recon: `ReconJob.CheckpointTS` (`recon_job_repo.go:23`).
- **Fencing**: sinkworker claim machine_id/token (`cmd/sinkworker/main.go:94`, `worker.go:466`). TransmuteScheduler set `app.fencing_*` (`transmute_scheduler.go:110-111`).

**UNKNOWN cần xác minh**:
- Nguồn publish vào CDC_EVENTS `cdc.goopay.>`.
- Worker và sinkworker có chạy đồng thời trên cùng topic không.
- Ngữ nghĩa chi tiết của batch-transform.
- `CheckAllSegmentA` (`recon_engine_run.go:151`) chỉ thấy tham chiếu trong chính file, chưa thấy caller ngoài.

Skill/công cụ đã dùng: run_command (grep/sed/cat, chỉ đọc), view_file. Không sửa file nào.