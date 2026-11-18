# L3 Review Profile — `data-hub` (trích `system_overview_and_design_pattern.md` §0, §2, §3.1, §3.4, §4.5 — 2026-11-17)

> Nguồn gốc chi tiết: `memory/global/project/data-hub/system_overview_and_design_pattern.md` (76.9KB). File này là bản rút ≤8KB cho lens review; khi nghi ngờ, `grep` file gốc theo `CP-x` / tên bảng.

## 1. Repos & kiến trúc
| Repo | Kiến trúc | Layer order (ngoài → trong) | Prefix path từng layer |
| :--- | :--- | :--- | :--- |
| `centralized-data-service` | Clean Layered, 7 nhóm chức năng (§2.1) | `server > handler > service > repository > model` | `server: internal/server/`, `handler: internal/handler/<group>/`, `service: internal/service/<group>/`, `repository: internal/repository/`, `model: internal/model/` |
| `cdc-cms-service` | Hexagonal + CQRS + Screaming (§2.2) | `api > app/commands,app/queries > domain,app/ports > infra` | `api: internal/api/<group>/`, `commands: internal/app/commands/<group>/`, `queries: internal/app/queries/<group>/`, `domain: internal/domain/<group>/`, `ports: internal/app/ports/` |
| `cdc-cms-web` | Feature/Hook-driven React (§2.3) | `pages > hooks > services,types` | `pages: src/pages/`, `components: src/components/<domain>/`, `hooks: src/hooks/`, `services: src/services/`, `types: src/types/` |

## 2. Invariants (§0 — nâng severity khi vi phạm)
- `INV-1 (S0)` — Live baseline: CDC, Shadow Ingest, Transmute (PG/Mongo/MySQL/ClickHouse), Snapshot, Recon, Execute-Heal, Governance, CMS đang chạy thật. Cấm sửa chữ ký hàm, đổi tên/kiểu trường struct JSON/GORM, đổi hành vi mặc định của hàm đang phục vụ luồng cũ.
- `INV-2 (S0)` — Open-Closed: mở rộng qua hàm/method mới, option/builder `With*`, hoặc phân nhánh theo `engine_type`; giữ nguyên 100% đường chạy nguồn/đích cũ.
- `INV-3 (S1)` — Cấm đổi tên/đập cây thư mục; code mới đặt đúng layer & group ở mục 1.
- `INV-4 (S1)` — Web: trường mới trong `src/types/index.ts` phải optional `?`; UI/Modal phức tạp tách ra `src/components/<domain>/`.
- `INV-5 (S1)` — Test local thiếu DB/NATS, thư mục `docs/*_bk*`, `scratch/` **không** tính là lỗi hệ thống.

## 3. Shared chokepoints (§3.1) & callers phải bảo toàn — repo `centralized-data-service`
| CP | Symbol / path | Callers (`file:line`) | Bất biến |
| :--- | :--- | :--- | :--- |
| `CP-1` | `MetadataRegistryService.ResolveSourceRoutes` (`internal/service/source/metadata_registry_service.go:469`) | `handler/shadow/event_handler.go:251,255`; `handler/orchestration/snapshot_runner_handler.go:528`; `handler/source/bridge_handler.go:279`, `bridge_upsert_fanout.go:82`; `handler/recon/recon_heal_route.go:80-132` | Hỗ trợ topic 5-part & 4-part; bảng có/không `schema.`; 1→N shadow phải chọn route theo `targetTable`/`bindingScope`, cấm `routes[0]` |
| `CP-2` | `EventHandler.HandleRaw` / `processEvent` (`internal/handler/shadow/event_handler.go:95,248`) | `event_handler.go:244`; `kafka_consumer.go:748`; `snapshot_runner_handler.go:822`; `recon_heal_fetch.go:102`; `recon_heal_postgres.go:81` | Envelope đủ `source_conn` + `WithBindingScope`; tách `subject` an toàn với `schema.table`; `_raw_data` là ground truth |
| `CP-3` | `BatchBuffer` + `shadow.DedupByConflictKey` (`internal/handler/shadow/batch_buffer.go:181,454`, `internal/model/shadow/upsert_dedup.go:14`, `batch_buffer_fanout.go:23`) | `server_setup.go:152,158,159`; `bridge_upsert_fanout.go:199` | `cdc`/`snapshot`/`heal` dùng BatchBuffer riêng; dedup theo PK giữ `SourceTsMs` mới nhất; chỉ publish transmute cho bản ghi `persisted` |
| `CP-4` | `TransmuteHandler` + `TableDebouncer` + `TransmuterModule` (`internal/handler/master/transmute_handler.go:132,293`, `transmute_event_date_split.go:121`, `internal/service/master/transmuter.go:167,653`) | BatchBuffer fanout; `bridge_upsert_fanout.go:342`; `transmute_scheduler.go:171`; `recon_execute_heal_segb.go:774,784`; CMS `cdc.cmd.transmute` | `switch req.TriggeredBy` giữ đủ 6 nhãn; có `events`/`_source_ids` → `execRunEvents`; hỗ trợ `copy_1_to_1`/`flatten`/`multi_shadow` × 4 đích |
| `CP-5` | `ReconCore` + `ChunkStreamBucketEngine` + `ExecuteHealHandler` (`internal/service/recon/recon_smoke.go:681`, `recon_stream_bucket_engine.go:233`, `internal/handler/recon/recon_execute_heal_handler.go:79`, `recon_execute_heal_segb.go:26`) | `server_jobs.go:182`; `cdc.cmd.recon-check/recon-heal/execute-heal` | Trục thời gian SegA/SegB theo `timestamp_field`, `WindowFreezeMargin`, chế độ `_id_only`; SegA phân nhánh theo DB nguồn (`recon_heal_fetch.go` vs `recon_heal_postgres.go`); SegB hỗ trợ SQL & Mongo |

## 4. Contracts liên service (§3.4)
- **JetStream streams:** `CDC_EVENTS` (`cdc.goopay.>`), `CDC_TRANSMUTE` (`cdc.cmd.transmute-shadow`), `CDC_CONTROL` (`cdc.control.commands.>`).
- **NATS cmd/RPC `cdc.cmd.*`:** producer `cdc-cms-service` → consumer `centralized-data-service`; phản hồi `cdc.job.result` (→ `async_jobs`), `cdc.activity.log` (→ `activity_logs`). Đổi tên subject / bỏ trường payload = `B-CON` S0.
- **Write-owner bảng `cdc_system.*`:**
  - `cdc-cms-service` sở hữu: `sources`, `connection_registry`, `source_object_registry`, `shadow_binding`, `mapping_rule_v2`, `child_explode_configs`, `master_connections`, `master_binding`, `mapping_rule_master`, `transmute_schedule`.
  - `centralized-data-service` sở hữu: `cdc_recon_smoke_result`, `cdc_reconciliation_results`, `cdc_recon_jobs`, `failed_sync_logs`, `daily_source_counts`, `transmute_history`; chỉ cập nhật runtime cols `last_gpay_id`, `last_run_at`, `ddl_status`, `timestamp_field`.
  - Cả 2: `pending_fields`, `schema_change_logs`, `schema_proposals`, `sensitive_fields`, `cdc_worker_schedule`, `async_jobs`, `activity_logs`.
  - Repo ghi vào bảng không phải owner = `B-CON` S1.

## 5. God-files & nơi CẤM nhồi `if/else` (§4.5)
| Khi thêm | Interface | Đặt file mới tại | CẤM sửa `if/else` trong |
| :--- | :--- | :--- | :--- |
| `TransformType` | `transmute.Strategy` (+`DeletePolicyProvider`, `BatchPreEnricher`, `PostIncrementalCatchupHook`) | `internal/service/master/transmute/<type>.go` (`init(){Register}`) | `transmuter.go`, `transmuter_events_batch.go`, `recon_tier_b.go` |
| `transform_fn` / `@directive` | `TransformFunc` / `FieldDirectiveFunc` | `internal/service/master/transform_registry.go` hoặc `directive_<name>.go` | `transmuter.go` (`extractColumns`) |
| Master engine | `MasterSinkDriver`, `MasterDDLDriver`, `ReconDestDriver` | `service/master/transmuter_<engine>.go`, `master_ddl_generator_<engine>.go`, `service/recon/recon_dest_<engine>.go`, `handler/recon/recon_execute_heal_segb_<engine>.go` | `transmuter.go`, `master_ddl_generator.go`, `recon_tier_b.go`, `recon_execute_heal_segb.go` |
| Source engine | `SourceIntrospectorDriver`, `ReconSourceDriver`, `SourceHealFetcherDriver` | `service/source/introspect_<engine>.go`, `service/recon/recon_source_<engine>.go`, `handler/recon/recon_heal_<engine>.go` | `recon_source_agent.go`, `recon_smoke.go`, `recon_tier_a.go`, `recon_heal_fetch.go`, `scan_service.go` |

`god_file_loc` khuyến nghị: 600 (manifest ghi đè). Thêm `case` vào cột cuối = `B-DES` S1.

## 6. Money-path / dữ liệu nhạy cảm đặc thù
- Keywords bổ sung: `trans_his`, `_gpay_id`, `_raw_data`, `sensitive_fields`, `pending_fields`.
- `_raw_data` chứa payload gốc có thể có PII → mọi log/debug in `_raw_data` = `B-SEC-02` S0 trừ khi mask qua `sensitive_fields`.

## 7. Lệnh kiểm nhanh đặc thù (0 token)
```bash
grep -rn "ResolveSourceRoutes(" internal/ --include=*.go          # callers CP-1
grep -rn "TriggeredBy" internal/handler/master/ internal/service/master/ --include=*.go   # CP-4 nhãn
grep -rnE '"cdc\.cmd\.[a-z.-]+"' internal/ --include=*.go | sort -u  # subject inventory
```
