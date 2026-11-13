# Bản đồ contract liên service DS ↔ CMS ↔ WEB (chỉ đọc, chưa sửa file nào)

**Tóm tắt:** Giữa các service không có package contract dùng chung. Mọi payload NATS đều là struct, `map` hoặc chuỗi JSON khai báo riêng ở từng repo. Hiện đa số json tag còn khớp, nhưng có 4 lỗi contract thật:
- `cdc.cmd.backfill` được publish nhưng không có subscriber.
- Hai bên kiểm tra data_type theo whitelist khác nhau.
- Job async trong `cdc_jobs` không có đường nào đóng lại.
- Model của bảng dùng chung bị lệch kiểu nullable.

Ký hiệu: DS = centralized-data-service, CMS = cdc-cms-service, WEB = cdc-cms-web.

## 1. NATS

**Header đi kèm mọi lệnh async của CMS** (`nats_command_bus.go:141-144`, `:326-341`): `Cdc-Job-Id`, `Cdc-Correlation-Id`, `Cdc-Created-By`, `Cdc-Command-Type`. Payload là `json.Marshal(cmd)` (`nats_command_bus.go:269`).
- DS đọc các header này tại `master_ddl_handler.go:254`, `snapshot_runner_handler.go:166,175`, `recon_check_handler.go:133` và `pkgs/natsconn/action_trace.go:25,28`.
- Hai bên chỉ thống nhất bằng chuỗi literal, không có hằng dùng chung.

**Cách DS trả lời RPC** (`base_handler.go:49-76`), theo thứ tự ưu tiên: `payload.reply_to` → `msg.Reply` → publish `cdc.result.<cmd>`.
- RPC của CMS (`nats_publisher.go:39-66`) dùng `SubscribeSync(replySubject)` và set `msg.Reply`, nên DS trả đúng chỗ.
- **CMS không có `Subscribe` nào khác** (grep chỉ ra `SubscribeSync` ở `nats_publisher.go:43` và `master_ddl_publisher.go:26,53,79,104`). Vì vậy mọi `cdc.result.*` và `cdc.evt.*` DS phát ra không ai trong CMS nghe.

| Subject | Publisher (path:line) | Subscriber DS | Payload CMS ↔ DS | Kiểu / Stream |
|---|---|---|---|---|
| cdc.cmd.batch-transform | CMS `source_object_actions_handler.go:784` (struct cục bộ, :762-770); CMS `registry_handler_transform.go:31` (**chuỗi tên bảng thô**); DS `server_jobs.go:97` (thô) | `server_setup.go:287` → `BatchTransformPayload` `batch_transform_handler.go:26-37` | Tag khớp 7/7. Payload thô chạy được nhờ fallback ở `batch_transform_handler.go:91-95` | pub/sub, core |
| cdc.cmd.master-create | CMS `approve_master.go:124-129` (map), `master_ddl_publisher.go:116-135` (map); DS `mapping_utils.go:263` | `server_setup.go:350` → `masterCreateRequest` `master_ddl_handler.go:111-125` | Khớp: master_binding_id, master_table, triggered_by, correlation_id | pub/sub |
| cdc.cmd.master-alter-column | CMS `master_ddl_publisher.go:40,66,91` (map + reply_to) | `:351` → `masterAlterColumnRequest` `master_ddl_handler.go:30-37` | Khớp. Comment field Action (:35) chưa ghi `drop_table`, dù code xử lý ở :71 | RPC qua reply_to |
| cdc.cmd.alter-column | CMS `master_ddl_publisher.go:117`; bus `server.go:281` (`AlterColumnCommand` `alter_column.go:11-17`) | `:300` → struct ẩn danh `schema_ddl_handler.go:617-623` | Tag khớp. **Whitelist data_type lệch (xem §4)** | pub/sub + reply_to |
| cdc.cmd.master-swap | bus `server.go:284` (`MasterSwapCommand` `master_swap.go:14-19`) | `:352` → struct ẩn danh `master_ddl_handler.go:233-237` | Khớp | async |
| cdc.cmd.transmute | bus `server.go:282` (`TransmuteRunCommand` `transmute_run.go:15-26`); DS `transmute_scheduler.go:171`, `recon_execute_heal_segb.go:784` | `:337` → `TransmuteRequest` `transmute_handler.go:259-269` | CMS là tập con, khớp | async, queue `transmute-workers` |
| cdc.cmd.transmute-shadow | DS `sinkworker/worker.go:397` | JetStream `server_setup.go:321` + fallback `:340` | Chỉ nội bộ DS | **JS CDC_TRANSMUTE** |
| cdc.cmd.snapshot.v2 | bus `server.go:285` (`SnapshotV2Command` `recon_async.go:145-153`); raw JSON `snapshot_progress_handler.go:112-115` | `:419` → `snapshotV2Payload` `snapshot_runner_utils.go:34-42` | Struct CMS **thiếu `progress_id`**; chỉ đường resume raw có field này | async |
| cdc.control.commands.{id} | CMS `snapshot_progress_handler.go:76-77` (chuỗi `"pause"`) | DS `snapshot_runner_handler.go:631-632` | Payload thô | pub/sub |
| cdc.cmd.recon-check / recon-heal / execute-heal / retry-failed / recon-backfill-source-ts | bus `server.go:268-272` (`recon_check.go:27-40`, `recon_async.go:18-82,167-171`) | `:482-487` (`reconCheckPayload` `recon_check_handler.go:22-34`, `reconHealPayload` `recon_check_heal_handler.go:20-33`) | recon-check khớp 12/12. recon-heal: DS có thêm legacy, mode, start/end_time **dạng string** (CMS không gửi) | async |
| cdc.cmd.debezium-signal / debezium-snapshot / restart-debezium / create-default-columns / standardize / scan-fields / detect-timestamp-field / system-connector-recover / bridge-oplog | bus `server.go:273-287` | `:403,404,295,289,277,292,488,296,299` | Chưa so từng field (UNKNOWN) | async |
| **cdc.cmd.backfill** | bus `server.go:280` (`BackfillCommand` `backfill.go:11-17`), gọi từ `mapping_rule_handler_batch.go:27,70` (route `router.go:382`, WEB `MappingRuleList.tsx:59`) | **Subscriber đã bị comment** `server_setup.go:282` | **Không có consumer** | ORPHAN |
| cdc.cmd.introspect*, scan-raw-data, scan-array, introspect-indexes, create-index, drop-index | CMS `introspection_handler.go:49-419`, `introspection_index_handler.go:40,102,143` | `:279-291,514-516` | RPC dùng `msg.Reply` | request/reply |
| cdc.cmd.shadow.bind / master.bind / discover / schedule.enable | CMS `provisioning_state_machine.go:45-48` → `provisioning_orchestrator.go:250`; DS cũng có orchestrator `provisioning_orchestrator_helpers.go:206` | `:367-369`, `:278` | **Hai orchestrator chạy song song ở hai repo** | pub/sub |
| schema.config.reload | CMS `nats_client.go:151`, `nats_publisher.go:84` | `server_setup.go:220` (bỏ qua payload) | ok | JS SCHEMA_CONFIG |
| connection.config.changed | CMS `master_registry_handler_connections.go:307` (connection_code thô) | `server_setup.go:226` | ok | core |
| cdc.result.*, cdc.evt.master-swap.completed (`master_ddl_handler.go:267,280`, kèm header `Cdc-Job-Status`), cdc.evt.transmute.completed | DS | Chỉ DS nghe nội bộ (`:347`); **CMS không nghe** | ORPHAN về phía CMS | — |
| schema.drift.detected | DS `schema_inspector.go:289` | Không ai subscribe | Chỉ nằm trong JS SCHEMA_DRIFT | ORPHAN |
| cdc.snapshot.resume | DS `wal_monitor.go:23,209` | Không ai subscribe | — | ORPHAN |
| cdc.pipeline.resume | Không ai publish (chỉ do operator gửi tay) | DS `dlq_circuit_breaker.go:32,36` | — | ORPHAN |
| sync-register, sync-state, drop-gin-index, recon-job-status, system-connector.recover (dạng dấu chấm) | Không thấy publisher trong 2 repo | `:293,294,288,483,297` | — | Có subscriber, không có publisher |

**JetStream streams:** cả hai repo đều tự `EnsureStreams` 4 stream CDC_EVENTS, SCHEMA_DRIFT, SCHEMA_CONFIG, CDC_TRANSMUTE.
- DS: `server_setup.go:99`, cấu hình ở `pkgs/natsconn/nats_client.go:25-78`.
- CMS: `server.go:124`, `nats_monitor_handler.go:212`, cấu hình ở `nats_client.go:25-80`.
- Giá trị config hiện giống nhau. Khác biệt: subject của CDC_EVENTS bên DS lấy từ `kafka.topicPrefix`, bên CMS hardcode `cdc.goopay.>` (`nats_client.go:27`, `nats_stream_provision.go:20`).

## 2. Bảng DB dùng chung (schema cdc_system)

**Migration:** chỉ CMS chạy migration (`internal/migrate/runner.go:141,153` đọc `migrations/schema/*/*.sql` qua `embed.go`). DS chỉ có `migrations/dest/001_dest_init.sql` và không gọi AutoMigrate.
- CMS dùng TableName không schema và dựa vào `search_path` (`pkgs/database/postgres.go:18`). DS luôn ghi đủ `cdc_system.`.

| Bảng | Model DS | Model CMS | Khác biệt |
|---|---|---|---|
| cdc_reconciliation_report | `model/recon/reconciliation_report.go:19,50,10` | `model/recon/reconciliation_report.go:15,10` | **source_count: DS `*int64` (NULL = chưa biết) ↔ CMS `int64 default:0`**. DS có error_code, CMS không. target_table: DS `gorm:"-"`, CMS map vào cột thật. CMS thêm omitempty cho source_host/table/type |
| master_binding | `model/master/master_binding.go:8-28` | `master_repo_gorm.go:36-50` | CMS thiếu binding_code, master_connection_id, physical_table_fqn, transform_type, transform_spec, created_by. Cả 4 cột đầu là NOT NULL (`032_v2_master_binding.sql:7,10,14,15`). Nhánh Create `master_repo_gorm.go:163-179` vì vậy sẽ vi phạm NOT NULL; nhánh này còn gán `MasterSchema: b.ShadowSchema` (:167). Có thật được gọi không: UNKNOWN. shadow_binding_id: DS `*int64`, CMS `int64`. master_database: DS `*string`, CMS `string` |
| failed_sync_logs | `model/shadow/failed_sync_log.go:29-30` | `model/recon/failed_sync_log.go` | CMS thiếu next_retry_at, last_error |
| cdc_mapping_rules | `model/master/mapping_rule.go:12-13` | `model/governance/mapping_rule.go` | CMS thiếu is_sensitive_field, mask_strategy |
| cdc_table_registry | `model/source/table_registry.go:20,53,57,64` | `model/source/table_registry.go:7,11` | DS có thêm timestamp_field_*, full_*_count, expected_fields, sensitive_fields, batch_flush_interval_seconds. CMS có source_connection_id, DS không có |
| transform_jobs | `repository/transform_job_repo.go:28` | `infra/persistence/transform_job_repo.go:15` | CMS có source_object_id, DS không. DS không có json tag |
| transmute_jobs | `repository/transmute_job_repo.go:29` | `infra/persistence/transmute_job_repo.go:31` | Giống nhau |
| sensitive_fields | `model/shadow/sensitive_field.go:3-6` (chỉ 2 field, không có PK) | `model/governance/sensitive_field.go:5` | CMS có thêm id, created_at, updated_at |
| cdc_recon_smoke_result | `recon_smoke_model.go:12` | `model/recon_smoke.go` | CMS thiếu trace_id |
| cdc_activity_log, cdc_worker_schedule, pending_fields, schema_changes_log | — | — | Khớp (đã diff) |
| connection_registry | `discover_handler_sftp.go:49` (OptionsJSON) | `domain/master/binding.go:67` (thêm ở commit 9b30e89) | Cột options_json do `029_v2_connection_registry.sql:41` tạo |
| source_object_registry, shadow_binding, mapping_rule_v2, transmute_schedule, sync_runtime_state, recon_jobs | Có model GORM | CMS truy cập qua `tx.Table(...)` hoặc struct cục bộ (ví dụ `source_repo_gorm.go:332`, `recon_read_repo_gorm.go:581`) | Chưa so từng cột (UNKNOWN) |
| cdc_jobs | DS không đụng (grep 0 kết quả) | `job_repo_gorm.go:49` | Xem rủi ro #3 |

## 3. HTTP

**WEB → CMS:**
- Gọi qua `cmsApi` (`services/api.ts:11`). `workerApi` (`api.ts:14`) được khai báo nhưng không dùng ở đâu.
- Route CMS định nghĩa trong `router.go:129-438`. Hàm `dual` (:136-164) đăng ký mỗi route ở cả `/x` và `/v1/x`.
- Đã đối chiếu khoảng 110 path WEB gọi với route CMS: đều có route tương ứng, không thấy 404 nào.

**DTO lệch:**
- `JobResponse` (`hooks/useAsyncJob.ts:22-31`) vs `JobView` (`get_job.go:35-48`): TS khai báo `payload`/`result` là `string`, nhưng Go trả `json.RawMessage` (object). TS bắt buộc `updated_at`, Go không có field này. Go có thêm started_at, finished_at, created_by, correlation_id.
- `SourceObjectRow` (`types/index.ts:92`, `notes` bắt buộc) vs `SourceObjectListItem` (`source_objects_read_models.go:18`): Go không có `notes`, nhưng `TableRegistry.tsx:971` vẫn đọc `record.notes`.
- Khớp: ShadowBindingRow, MappingRule (TS là tập con của `dto/mapping_rule_dto.go:10`), TableRegistry, PendingField, SchemaChangeLog.

**CMS → DS:**
- Chỉ có health probe GET `<workerUrl>/healthz` (`probes/worker.go:14`) và `/metrics` (`prom_client.go:229`).
- `/healthz` chỉ có trên admin server gin (`internal/admin/server.go:121`). Server fiber chỉ có `/health` (`server_setup.go:549`). Nếu `CMS_SYSTEM_WORKER_URL` (`config.go:160`) trỏ vào cổng fiber thì probe sẽ báo down. Giá trị deploy thực tế: UNKNOWN.

## 4. Logic domain bị nhân đôi giữa hai repo

- **`internal/naming`:** `ShadowSchemaPrefix`/`ShadowSchemaName` giống nhau (DS :23,35 / CMS :22,34). `NormalizeIdentifier` chỉ có ở DS (:41, có cắt 63 byte). `IsSystemColumn` chỉ có ở CMS (:39), và trùng thêm với `pkgs/utils/dbutil/validate.go:55`.
- **Tên shadow schema được suy ra theo 4 cách khác nhau:**
  - CMS: `shadow_`+`slugifyIdentifier(connectionCode)` (`source_object_v2_sync_helpers.go:148,159-180`, không cắt 63 byte).
  - DS bind: `shadow_`+`connCode` thô, không slug (`provisioning_shadow_bind.go:203-208`).
  - DS sinkworker: `shadow_`+`sourceDB`, chỉ thay `-` và `.` (`sinkworker/utils.go:321-328`).
  - DS registration: `db` hoặc `db+"_mongo"` (`registration.go:390-396`).
  - Tên bảng: DS dùng `NormalizeIdentifier` (`registration.go:216`) ở một chỗ và `lower + "-"→"_"` (`utils.go:317-318`) ở chỗ khác.
- **Whitelist data_type:**
  - DS `IsSafeType` (`base_handler.go:194-199`): bắt buộc `VARCHAR(n)` và `NUMERIC(p,s)`; không chấp nhận INT, BOOL, CHAR hay VARCHAR trần; có TIME, INTERVAL, BYTEA, INET, kiểu mảng, ENUM.
  - CMS `pkgs/utils/pg_validator.go:9` và `pkgs/utils/dbutil/validate.go:7` (2 bản, bản sau có thêm OBJECTID/OID): chấp nhận INT, BOOL, VARCHAR trần, NUMERIC(10).
  - Hệ quả: CMS duyệt xong thì DS trả `invalid data_type` (`schema_ddl_handler.go:678-679`).
- **transform_fn:** DS `transform_registry.go:31-43` và CMS `master_mapping_rule_handler.go:174-187` hiện có đúng cùng 11 tên, nhưng được chép tay, không có nguồn chung. Thêm `isJSONBTransformFn` (`update_column_metadata.go:99`) đoán theo prefix.
- **CanonicalMongoExtJSON:** chỉ có ở DS, nhưng tự nhân đôi trong nội bộ: `kafka_consumer.go:926` và `sinkworker/utils.go:77`. `camelToSnake` có 3 bản (`schema_ddl_handler.go:709`, `index_manager.go:404`, `recon_tier_a.go:268`); `quoteIdent` có 5 bản.
- **Timestamp resolution:** DS `recon_query.go:21` vs CMS `multi_shadow_helper.go:75,95` — logic khác nhau, chưa so chi tiết.
- **`pkgs/natsconn`:** DS có thêm `action_trace.go`. `EnsureStreams` khác chữ ký (DS nhận `cdcSubjects`, CMS không có tham số). Commit CMS 7707aef vừa bỏ cơ chế `preserveExistingSubjects`, nay cả hai bên đều có thể ghi đè subjects của stream.
- **Provisioning orchestrator:** tồn tại ở cả DS (`service/orchestration/provisioning_orchestrator*.go`) và CMS (`infra/persistence/governance/provisioning_orchestrator.go`), cùng publish các subject bind.

## 5. Git log -15: các commit chạm contract

- **CMS 9b30e89 / DS b8326bf (2026-10-29, "update master mongo"):** CMS sửa `create_master.go`, `binding.go` (thêm options_json, sslmode…), `master_repo_gorm.go` (-271 dòng), `router.go`. DS sửa transmuter/ddl generator nhưng không sửa `model/master/master_binding.go` hay model connection. Bảng chạm từ một phía.
- **CMS 7707aef + 381695d + 80bf337 / DS 786db91 + 34374a9 + 3369c31 (2026-10-26):** cả hai repo cùng sửa `pkgs/natsconn/nats_client.go` và `nats_stream_provision.go`, nhưng kết quả vẫn khác nhau (xem §4). Đây là chỗ hai bên thay đổi song song mà không đồng bộ.
- **CMS 818064d / DS 778dbad (2026-10-22, transform_fn jsonb):** sửa đồng bộ hai bên (allowlist + registry) và thêm migration 081. Hiện khớp, nhưng mỗi lần thêm hàm đều phải sửa tay ở 2 chỗ.
- **CMS 80bf337:** thêm 3 dòng vào `source_objects_read_models.go`. WEB 7a1a376 thêm 3 dòng vào `types/index.ts` cùng ngày (có vẻ đồng bộ).
- **DS fa20799 / 3369c31 (transmute consumer pool, batch_transform_handler):** chỉ phía DS. CMS publisher không đổi, payload vẫn tương thích.

## Top 10 rủi ro contract (xếp theo mức độ)

1. **`cdc.cmd.backfill` không có consumer.** CMS dispatch (`server.go:280`, `mapping_rule_handler_batch.go:27,70`, WEB `MappingRuleList.tsx:59`) nhưng subscriber DS đã bị comment (`server_setup.go:282`). Backfill hiện không hề chạy.
2. **Whitelist data_type lệch** giữa CMS (`pg_validator.go:9`, `dbutil/validate.go:7`) và DS (`base_handler.go:194`). Ví dụ INT, BOOL, VARCHAR trần qua được CMS nhưng bị DS từ chối.
3. **Job async trong `cdc_jobs` không có đường đóng:** DS không ghi `cdc_jobs`, CMS không subscribe `cdc.result.*` hay `cdc.evt.*`. Reaper chỉ quét `status='running'` (`stuck_job_reaper.go:118-123`), trong khi Dispatch tạo job ở trạng thái pending. Thêm nữa key `"transmute"` (:41) không khớp type `"transmute.run"` (`transmute_run.go:29`). WEB poll `/jobs/:id` có khả năng thấy pending mãi mãi (HYPOTHESIS, cần trace runtime).
4. **Model `master_binding` lệch:** nhánh Create của CMS thiếu 4 cột NOT NULL và gán nhầm MasterSchema = ShadowSchema (`master_repo_gorm.go:163-179` vs `032_v2_master_binding.sql:7-15`).
5. **`source_count`:** DS dùng NULL nghĩa là "chưa biết", CMS dùng `int64 default:0`. Ngữ nghĩa NULL bị mất khi CMS đọc hoặc ghi.
6. **Tên shadow schema/table có 4 thuật toán khác nhau** (xem §4). Đổi một chỗ là CMS và DS trỏ sang 2 schema khác nhau.
7. **Hai bên cùng provision 4 stream JetStream** với logic subjects khác nhau (CMS hardcode `cdc.goopay.>`, vừa bỏ preserve). Nếu `topicPrefix` khác `cdc.goopay` thì hai bên ghi đè lẫn nhau.
8. **Payload toàn bộ khai báo riêng ở từng repo,** không có schema dùng chung. Nhiều chỗ dùng `map[string]any`, chuỗi JSON `fmt.Sprintf` hoặc chuỗi thô, ví dụ `master_ddl_publisher.go:19-119`, `snapshot_progress_handler.go:112`, `registry_handler_transform.go:31`. Header `Cdc-*` cũng chỉ là literal. Đổi tên một tag là vỡ âm thầm, không có lỗi biên dịch.
9. **DTO HTTP lệch:** `JobResponse` (payload/result kiểu string, `updated_at` không tồn tại) và `SourceObjectRow.notes`.
10. **Probe `/healthz` của CMS phụ thuộc** vào việc workerUrl trỏ đúng admin server gin chứ không phải fiber. Kèm theo là các subject chỉ có một đầu: `schema.drift.detected`, `cdc.snapshot.resume`, `cdc.pipeline.*`, `cdc.evt.master-swap.completed`.

**Chưa kiểm (UNKNOWN):** so từng cột của source_object_registry, shadow_binding, mapping_rule_v2; payload của debezium-*, standardize, scan-fields, create-default-columns; giá trị cấu hình workerUrl và topicPrefix khi deploy.

Lesson áp dụng: không suy diễn, mọi khẳng định gắn `path:line`; tách VERIFIED và HYPOTHESIS. Công cụ đã dùng: grep, awk, diff, git log/show, view_file (đều chỉ đọc).