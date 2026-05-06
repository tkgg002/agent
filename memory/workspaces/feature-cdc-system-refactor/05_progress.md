# Progress Log

| Timestamp | Operator | Model | Action / Status |
|-----------|----------|-------|-----------------|
| 2026-04-23 00:00 ICT | Muscle | GPT-5 | Workspace initialized for CDC system refactor. |
| 2026-04-23 00:00 ICT | Muscle | GPT-5 | Completed refactor across reconciliation, DLQ, and schema evolution files; local gofmt + targeted go test passed. |

| 2026-04-24 00:00 ICT | Muscle | GPT-5 | Cleanup masking wiring: ReconHealer now delegates raw JSON masking to shared MaskingService; worker_server DI now wires DynamicMapper, SchemaInspector, DLQHandler, and ReconHealer to one masking instance. |
| 2026-04-24 00:00 ICT | Muscle | GPT-5 | Added ReconHealer security regression tests for top-level, nested, array, heuristic masking, and OCC parity; local gofmt + go test passed. |

| 2026-04-24 00:00 ICT | Muscle | GPT-5 | Added production-grade architecture.md at repo root with overview, deployment, worker component, and deep-dive diagrams for reconciliation, DLQ, and schema evolution. |

| 2026-04-24 00:00 ICT | Muscle | GPT-5 | Added masking regression tests for DynamicMapper and DLQHandler, verified _raw_data and failed_sync_logs RawJSON stay sanitized before persistence. |

| 2026-04-24 00:00 ICT | Muscle | GPT-5 | Added SchemaInspector masking tests and completed raw-data audit for KafkaConsumer + BatchBuffer; patched both to sanitize failed_sync_logs payloads through shared MaskingService. |

| 2026-04-24 00:00 ICT | Muscle | GPT-5 | Audited recon_handler retry path; sanitized external raw_json before retry upsert and added regression tests for top-level, nested, array, and heuristic masking. |

| 2026-04-24 00:00 ICT | Muscle | GPT-5 | Audited legacy dlq_worker path; patched retry raw JSON rebuild to re-mask _raw_data before UPSERT regeneration and added regression tests. |

| 2026-04-24 09:33:09 +0700 | Muscle | [unverified] | Audited command_handler.go admin surfaces, minimized event_bridge.go payloads, added tests, and wrote security-audit-report.md |

| 2026-04-24 09:40:20 +0700 | Muscle | [unverified] | Synced CommandHandler/EventBridge API comments with sanitized-result and metadata-only contracts; added integration tests proving cdc_activity_log stores redacted admin traces. |

| 2026-04-24 09:49:51 +0700 | Muscle | [unverified] | Added DLQHandler and EventBridge integration tests; hardened DLQ free-form error sanitization via shared text_sanitizer.go and verified NATS/DB contracts end-to-end. |

| 2026-04-24 09:56:37 +0700 | Muscle | [unverified] | Added KafkaConsumer and ReconHandler integration tests; hardened KafkaConsumer.writeDLQ error-message sanitization and verified ingestion/retry DB contracts end-to-end. |

| 2026-04-24 10:06:25 +0700 | Muscle | [unverified] | Audited ancillary flows activity_logger/backfill_source_ts/transmuter; normalized free-form sanitizer usage in activity_logger and backfill_source_ts; published security-regression-matrix.md. |

| 2026-04-24 10:10:45 +0700 | Muscle | [unverified] | Extended ancillary audit to transmute_scheduler and recon_core; sanitized persisted scheduler/reconciliation error_message fields and updated security regression docs. |

| 2026-05-05 16:30 ICT | Muscle | claude-opus-4-7 | Phase 2 planning kickoff for cdc-cms-service refactor. Wrote 4 docs: 01_requirements_phase2_cms_refactor.md (10 DoD + 8 endpoint smoke + 9 lessons referenced), 02_plan_phase2_cms_refactor.md (8-pillar plan P0–P7 with risk/effort matrix + decision log), 08_tasks_phase2_cms_refactor.md (17 tasks T1–T17 dependency graph + per-pillar gate), 09_tasks_solution_phase2_cms_refactor.md (per-task code sketches + diffs + pre-commit checklist). No source code touched; service health pre-write & post-write same: cms-service :8083 /health 200, auth-service :8081 /health 200, worker :8082 alive (PID 23565). admin-api :8084 down (pre-existing, không liên quan). Awaiting user approval to start P0 (dead-code prune, 2h, lowest risk). |

| 2026-05-05 16:45 ICT | Muscle | claude-opus-4-7 | User feedback: "chưa đủ để gọi là refactor" — plan trước chỉ tách layer trong CMS (handler/service/repo), CHƯA decoupling Brain (CMS Control Plane) ↔ Muscle (Worker Data Plane) theo Separation of Powers. User mandate kiến trúc mới: CMS chỉ giữ metadata + RBAC + dispatch NATS + 202 Accepted, Worker execute mọi heavy work + write state về cdc_dw. Plan cũ (`phase2_cms_refactor`) → SUPERSEDED nhưng GIỮ NGUYÊN (CLAUDE.md §11 APPEND-only). Tạo bộ file mới `phase2_decoupling` (01_requirements / 02_plan / 08_tasks / 09_tasks_solution). Audit thực tế qua 2 Explore agent: CMS đã dispatch 12/14 trigger endpoint qua NATS đúng (✅), còn 2 INLINE (Master Swap + V2 SyncFromLegacy) cần move sang worker; Worker đã có 30+ subscriber + JobMonitor pattern (đã close-loop transmute, provisioning); 12/14 cmd subject thiếu companion `cdc.evt.X.completed` event (Lesson #1292 violation). |

| 2026-05-05 17:05 ICT | Muscle | claude-opus-4-7 | User mandate điều chỉnh 4 Pillar CQRS thay 8 pillar v1: P1 setup `internal/{domain,app/ports}` + Repository/CommandBus/QueryBus/Publisher interfaces; P2 migrate Read paths → `app/queries/` (15 endpoint); P3 migrate Action → `app/commands/` + `NATSCommandBus` + new `cdc_system.cdc_jobs` table + move 2 INLINE (Master Swap, V2 Sync) sang worker + 12 cmd thêm companion `cdc.evt.X.completed` + JobMonitor extend wildcard subscribe; P4 standardize `infra/persistence/` (raw SQL chỉ ở đây) + delete `internal/service/` cũ. Effort 13d sequential, ~16d với pre-commit gate, realistic 3 tuần với 1 engineer. Files written: 01_requirements_phase2_decoupling.md (12.7KB), 02_plan_phase2_decoupling.md (23.8KB), 08_tasks_phase2_decoupling.md (12.6KB), 09_tasks_solution_phase2_decoupling.md (20.4KB), report_phase2_cms_decoupling_2026-05-05.md. KHÔNG touch source code. Service health pre/post identical: cms 8083 /health 200, cms /ready 200, auth 8081 /health 200, worker 8082 PID 23565 alive. Awaiting user approval to start P1 (T1.1, 2d, LOW risk). |

| 2026-05-05 17:20 ICT | Muscle | claude-opus-4-7 | **P1 IMPLEMENTATION DONE (T1.1+T1.2+T1.3+T1.5)** — created 16 new Go files, ZERO modification to existing files (T1.4 wiring deferred to P2 per zero-blast-radius rule). Files: `internal/domain/job/job.go` (1.5KB Job entity + Status enum), `mapping/rule.go` (1.7KB Rule + Status + Filter), `master/binding.go` (1.3KB Binding + SchemaStatus), `reconciliation/{report.go,failed_log.go}` (1.4KB+1.0KB), `source/object.go` (1.6KB Object + Scope + ProvisioningState); `internal/app/ports/{repository.go,command_bus.go,query_bus.go,publisher.go}` (3.0KB+1.2KB+0.5KB+0.3KB) defines 6 repo interfaces + Command/CommandBus/Query/QueryBus/Publisher; placeholder `doc.go` cho 6 packages (app/queries, app/commands, infra/{persistence,messaging,http,cache}). VERIFICATION: `go build ./...` exit=0, `go vet ./...` exit=0, `go test ./internal/api/... ./internal/middleware/... ./internal/service/... -count=1` PASS (api 0.711s, middleware 1.711s, service 1.174s — no regression). LIVE SMOKE 8/8 GREEN với JWT admin@goopay.vn role=admin: /health 200, /ready 200, /api/v1/source-objects 200 (15.7KB), /api/mapping-rules 200 (18.3KB), /api/reconciliation/report 200 (874B), /api/v1/masters 200 (6.7KB), /api/v1/system/connectors 200 (5.3KB), /api/sync/health 200 (120B). Service health pre/post identical: cms 8083 /health+/ready 200, auth 8081 /health 200, worker 8082 PID 23565 alive. Lý do KHÔNG đụng `server.go` ở P1: blast radius — server boot là entry-point chia sẻ; nếu sửa boot mà repo concrete chưa có thì runtime panic do nil deps. Phase 2 (Read migration) sẽ wire ports vào server.go khi có concrete repo đầu tiên. Next: P2.T2.1 ListMappingRulesQuery demo (6h, LOW risk) — đợi user approval. Report: `report_phase2_v2_p1_2026-05-05.md`. |

| 2026-05-05 17:38 ICT | Muscle | claude-opus-4-7 | **P2.T2.1 DONE — ListMappingRulesQuery demo end-to-end** — 1st CQRS Q-side endpoint migrated. Files modified: `internal/domain/mapping/rule.go` (added MasterBindingID/SourceDatabase/SourceSchema/SourceNamespace/ShadowSchema/ShadowTable/SourcePath/SourceFormat/TransformFn to Rule + Filter expanded), `internal/app/ports/repository.go` (MappingRuleRepo gained ListPaginated method), `internal/api/mapping_rule_handler.go` (List rewritten — 100→52 lines net, raw SQL → query handler call + helper formatPgOF/ruleToRow added; constructor signature gained listQuery param), `internal/server/server.go` (wired persistence.NewMappingRuleRepo + queries.NewListMappingRulesHandler; passed to NewMappingRuleHandler — 1st DI of ports through new infra). Files NEW: `internal/infra/persistence/mapping_rule_repo_gorm.go` (concrete adapter — ListPaginated+List implemented; Save/UpdateStatus/etc stubbed errNotImplementedP4), `internal/app/queries/list_mapping_rules.go` (Query+Result+Handler). VERIFICATION (test instance bound :8093, NOT touching shared :8083): `go build ./...` PASS, `go test ./... -count=1` PASS (api 0.286s incl 16 ComputeDriftStatus subtests, middleware 0.714s, service 1.662s). LIVE smoke 8/8 200 với JWT admin (port 8093 instance): /health 35B, /ready 18B, /api/v1/source-objects 15715B, /api/mapping-rules 18289B (== P1 baseline), /api/reconciliation/report 874B, /api/v1/masters 6764B, /api/v1/system/connectors 5260B, /api/sync/health 120B. Filter probes: status=approved→count=3, source_table=legacy_orders_addtest→total=7, rule_type=mapping→32, rule_type=foobar→0 (forces 1=0), source_object_id=30→7. Contract diff `/tmp/_mapping_rules_pre.json` vs `_post2.json`: BYTE-IDENTICAL after `formatPgOF` fix forced UTC (`.UTC().Format(...) + "+00"` to mirror Postgres `TO_CHAR ... OF` UTC session output). Production instance :8083 KHÔNG restarted (sandbox denied — needs user authorization). Test instance :8093 PID 1387 left alive cho P2.T2.2 reuse. Next: P2.T2.2 ListSourceObjectsQuery + GetSourceObjectQuery. |

| 2026-05-06 08:50 ICT | Muscle | claude-opus-4-7 | **P2.T2.2 DONE — ListSourceObjectsQuery + GetSourceObjectMappingContextQuery** — 2nd & 3rd CQRS Q-side endpoints migrated (`GET /api/v1/source-objects` + `GET /api/v1/source-objects/registry/:registry_id`). Files NEW (5): `internal/app/queries/source_objects_read_models.go` (SourceObjectListItem + SourceObjectMappingContextReadModel — projection-level read models, NOT domain entities, since enrichment crosses 4 tables), `source_object_reader.go` (SourceObjectReader port colocated với consumer — single-caller pattern keeps `ports/` aggregate-focused), `list_source_objects.go` (Query+Result+Handler with page/pageSize clamp 1..500), `get_source_object_mapping_context.go` (Query+Handler returning *ReadModel; nil → 404), `internal/infra/persistence/source_object_read_repo_gorm.go` (GORM adapter — listBaseFromWhere const + buildListFilter helper; SQL lifted verbatim từ `api/source_objects_handler.go`). Files MODIFIED (2): `internal/api/source_objects_handler.go` (constructor signature +2 query handlers; SourceObjectRow/SourceObjectMappingContext converted thành type aliases để Swagger + wire-shape giữ identical; List body 140→24 lines, GetMappingContext body 87→14 lines), `internal/server/server.go` (wired `persistence.NewSourceObjectReadRepo` + 2 query handlers + passed to `NewSourceObjectsHandler`). VERIFICATION: `go build ./...` exit=0, `go vet ./...` exit=0, `go test ./internal/api/... ./internal/middleware/... ./internal/service/... -count=1` PASS (api 1.787s, middleware 0.832s, service 1.309s — no regression). LIVE smoke 8/8 200 với JWT admin trên :8093 (PID restart 1387→new PID; production :8083 KHÔNG touched per sandbox rule): /health 35B, /ready 18B, /api/v1/source-objects 15715B (== :8083 pre baseline), /api/mapping-rules 18289B, /api/reconciliation/report 874B, /api/v1/masters 6764B, /api/v1/system/connectors 5260B, /api/sync/health 120B. Contract diff (3 probes): default page=1&page_size=20 → BYTE_IDENTICAL (15715B); source_db=goopay_source → BYTE_IDENTICAL (5816B, total=7); is_active=true → BYTE_IDENTICAL (5878B, total=7). GetMappingContext id={1,5,100} → cả pre :8083 và post :8093 đều trả 500 BYTE_IDENTICAL — pre-existing SQL bug trong cdc_table_registry path, KHÔNG phải regression của T2.2 (SQL bê nguyên không sửa); ticket riêng nếu cần fix. Service health pre/post identical: cms 8083 /health+/ready 200, auth 8081 /health 200, worker :8082 /health 200 (PID 23565 still alive). Next: P2.T2.3 ListMastersQuery + GetMasterByNameQuery. |

| 2026-05-06 08:53 ICT | Muscle | claude-opus-4-7 | **P2.T2.3 DONE — ListMastersQuery** (4th CQRS Q-side endpoint, `GET /api/v1/masters`). Plan-level GetMasterByNameQuery skipped vì không có endpoint GET-by-name route hiện tại (chỉ Approve/Reject/Toggle/Swap dùng resolveMasterBindingByName helper); sẽ tạo cùng các Command tương ứng ở P3 thay vì YAGNI. Files NEW (2): `internal/app/queries/list_masters.go` (MasterListItem read model + MasterReader port + Query/Result/Handler — read DTO crosses 4 tables master_binding⨝shadow_binding⨝source_object_registry⨝connection_registry), `internal/infra/persistence/master_read_repo_gorm.go` (concrete GORM adapter; SQL lifted verbatim từ master_registry_handler.List). Files MODIFIED (2): `internal/api/master_registry_handler.go` (MasterRow → type alias `queries.MasterListItem`; constructor signature +listQ; List body 39→7 lines, raw SQL gone), `internal/server/server.go` (wired masterReader + listMastersH; passed to NewMasterRegistryHandler). VERIFICATION: `go build ./...` exit=0, `go vet ./...` exit=0, `go test ./internal/api/... ./internal/middleware/... ./internal/service/... -count=1` PASS (api 0.735s, middleware 1.251s, service 1.745s). LIVE smoke trên :8093 (PID rotated): /api/v1/masters 6764B HTTP 200, count=9, len(data)=9. Contract diff `_masters_pre.json` vs `_masters_post.json`: BYTE_IDENTICAL. 8/8 smoke 200 same sizes như T2.2. Next: P2.T2.4 GetReconReportQuery + ListFailedLogsQuery. |

| 2026-05-06 09:05 ICT | Muscle | claude-opus-4-7 | **P2.T2.4 CODE DONE — Reconciliation Q-side (3 endpoints)** — `GET /api/reconciliation/report` (LatestReport), `GET /api/reconciliation/report/:table` (TableHistory), `GET /api/failed-sync-logs` (ListFailedLogs). Files NEW (5): `internal/app/queries/recon_read_models.go` (LatestReportRow embed model.ReconciliationReport + 14 enrichment cols + 4 `gorm:"-"` post-SQL fields, FailedLogRow embed model.FailedSyncLog + 4 scope cols), `recon_reader.go` (ReconReader port — 3 methods + FailedLogFilter struct, single-caller colocated with consumer), `list_latest_reports.go` (Query/Result/Handler — empty filter, raw rows out), `get_table_history.go` (Query/Result/Handler — page>=1, size 1..100 default 20), `list_failed_logs.go` (Query/Result/Handler — TargetTable/Status/ErrorType filter, page>=1, size 1..200 default 30); `internal/infra/persistence/recon_read_repo_gorm.go` (GORM adapter — listLatestPrimary + listLatestLegacy fallback const, GetTableHistory via Model+Where+Count+Find, ListFailedLogs với failedLogsBase const + dynamic WHERE). Files MODIFIED (2): `internal/api/reconciliation_handler.go` (struct gained 3 query-handler fields + constructor signature; ReportRow/FailedLogRow → type aliases vào queries package; LatestReport body 145→32 lines giữ nguyên enrichment loop ComputeDriftStatus/ErrorMessagesVI/deriveSourceQueryMethod cùng test matrix, TableHistory 19→13 lines, ListFailedLogs 89→17 lines), `internal/server/server.go` (wired persistence.NewReconReadRepo + 3 query handlers + passed to NewReconciliationHandler). Decision: enrichment helpers ở lại api/ vì có 16 ComputeDriftStatus subtests; Q-side trả raw rows, api loop điền 4 fields `gorm:"-"`. VERIFICATION: `go build ./...` exit=0, `go vet ./...` exit=0, `go test ./internal/api/... ./internal/app/queries/... ./internal/infra/... -count=1` PASS (api 0.700s, queries no test files, infra no test files). Binary built `/tmp/cdc-cms-p2-t24` 57558194 bytes. **Restart :8093 cho live byte-identical diff: SANDBOX DENIED kill 15690** (lý do: cần user authorization theo precedent). Test instance :8093 vẫn chạy binary T2.3 (PID 15690 = `/tmp/cdc-cms-p2-t23`); production :8083 KHÔNG touched. Code-level verification: build/vet/test green, contract preserved through type aliases (queries.LatestReportRow/FailedLogRow giữ JSON tags identical). Next blocker: user OK để restart :8093 với binary mới rồi diff response 3 endpoints vs :8083 baseline trước khi đóng task. |

| 2026-05-06 09:11 ICT | Muscle | claude-opus-4-7 | **P2.T2.4 LIVE VERIFIED — BYTE_IDENTICAL** — User authorized restart :8093 (kill 15690 → start /tmp/cdc-cms-p2-t24 with same env CMS_SERVER_PORT=:8093 CMS_SERVER_NAME=cms-p2t24, PID 22077). Live diff against :8083 baseline với JWT admin@goopay.vn. **3 recon endpoints**: `/api/reconciliation/report` (default unfiltered) 874B==874B BYTE_IDENTICAL; `/api/reconciliation/report/orders` (pagination probes page=1&page_size=5/page=2/page=1&page_size=999) all BYTE_IDENTICAL — 999 clamped to 20 returning 808B identical to default; `/api/failed-sync-logs` (default 5632B + 4 filter probes target_table=orders_addtest 853B / status=failed 32B / error_type=type_mismatch 32B / combo target_table+status 32B) all BYTE_IDENTICAL. **Full 8/8 smoke vs :8083**: /health 35B, /ready 18B, /api/v1/source-objects 15715B, /api/mapping-rules 18289B, /api/reconciliation/report 874B, /api/v1/masters 6764B, /api/v1/system/connectors 5260B, /api/sync/health 120B — **all 200 + BYTE_IDENTICAL**. Service health pre/post: cms 8083 + cms 8093 + auth 8081 + worker 8082 alive. T2.4 complete. Next: P2.T2.5 GetSyncHealthQuery + GetSystemHealthSnapshotQuery. |

| 2026-05-06 09:14 ICT | Muscle | claude-opus-4-7 | **P2.T2.5 DONE — SyncHealth Q-side** (`GET /api/sync/health`). SystemHealthSnapshot endpoint (`GET /api/system/health`) SKIPPED per CLAUDE.md §6 elegance balanced — Redis-cache reader thuần, không có SQL surface để migrate; wrap Q-side chỉ thêm layer mà không tạo giá trị. Files NEW (2): `internal/app/queries/get_sync_health.go` (SyncHealthReader port + SyncHealthSnapshot struct + Query/Result/Handler — 5 aggregate counts), `internal/infra/persistence/sync_health_read_repo_gorm.go` (GORM adapter — 5 Count() calls với chaining error). Files MODIFIED (2): `internal/api/registry_handler.go` (struct +syncHealthQ, constructor signature +1 param, SyncHealth body 19→7 lines giữ wire shape), `internal/server/server.go` (wired persistence.NewSyncHealthReadRepo + queries.NewGetSyncHealthHandler vào NewRegistryHandler). VERIFICATION: `go build ./...` exit=0, `go vet ./...` exit=0, `go test ./internal/api/... ./internal/middleware/... ./internal/service/... -count=1` PASS (api 2.204s, middleware 1.684s, service 1.146s). **First-attempt failed byte-identical** — JSON key order khác (legacy `fiber.Map` → alphabetical, struct → field-declaration order). FIX: reorder struct fields alphabetically (`active_tables, approved_mapping_rules, pending_mapping_rules, tables_created, total_registered_cms`) to mirror Go map JSON serialization. **Lesson**: CQRS migrate from `fiber.Map` to typed struct cần khớp alphabetical order — Go map JSON serialize alphabetically post-1.12. LIVE 8/8 smoke trên :8093 (binary `/tmp/cdc-cms-p2-t25` PID rotated): tất cả endpoints BYTE_IDENTICAL với :8083 baseline (sync/health 120B, source-objects 15715B, mapping-rules 18289B, recon/report 874B, masters 6764B, connectors 5260B, health 35B, ready 18B). Service health pre/post identical (cms 8083+8093, auth 8081, worker 8082 alive). Next: P2.T2.6 ListConnectorsQuery (system_connectors_handler). |

---

## 2026-05-06 09:24 — P2.T2.6 ConnectorsQuery DONE (8/8 BYTE_IDENTICAL)

### Files
- NEW `internal/infra/http/kafka_connect.go` — `KafkaConnectClient` (full HTTP plumbing) + types `ConnectorState`, `ConnectorTask`, `ConnectorStatusResp`, `ConnectorView` + `FilterSafeConfig`. 8 methods: ListNames, GetStatus, GetConfig, ListPlugins, Restart, RestartTask, Create, Delete, Lifecycle.
- NEW `internal/app/queries/list_connectors.go` — `ConnectorReader` port (4 methods) + 3 handlers `ListConnectorsHandler`, `GetConnectorHandler`, `ListConnectorPluginsHandler`. List handler owns N+1 fan-out (status + config per name); per-connector failures tolerated (legacy parity).
- REWRITTEN `internal/api/system_connectors_handler.go` — handler now holds `*infrahttp.KafkaConnectClient` + 3 query handlers. Constructor takes client (NOT URL) for elegance — same pool/timeout across all 8 endpoints. Reads delegate to query handlers; writes (Restart/Pause/Resume/Create/Delete/RestartTask) call `h.client` directly. `filterSafeConfig` deleted, replaced by exported `infrahttp.FilterSafeConfig`.
- MODIFIED `internal/server/server.go` — added `infrahttp` import, built `kafkaConnectClient := infrahttp.NewKafkaConnectClient(...)` ONCE, wired 3 query handlers + handler with the same client.

### Verify
- `go build ./...` PASS
- `go vet ./...` PASS (silent)
- `go test ./internal/api/... ./internal/app/queries/... ./internal/infra/...` → ok api 0.708s, queries/cache/http/messaging/persistence no test files
- Restart :8093 with `/tmp/cdc-cms-service-t26` (env `CMS_SERVER_PORT=:8093 CMS_SERVER_NAME=cms-p2t26`); boot clean
- Live diff vs :8083 baseline (legacy `main` PID 18563):

| Endpoint | legacy | new | verdict |
|----------|--------|-----|---------|
| /api/v1/source-objects?limit=10            | 15715 | 15715 | BYTE_IDENTICAL |
| /api/mapping-rules?limit=10                | 18289 | 18289 | BYTE_IDENTICAL |
| /api/reconciliation/report?limit=5         |   874 |   874 | BYTE_IDENTICAL |
| /api/v1/masters?limit=10                   |  6764 |  6764 | BYTE_IDENTICAL |
| /api/v1/system/connectors                  |  5260 |  5260 | BYTE_IDENTICAL |
| /api/v1/system/connectors/cdc-pg-source    |  1303 |  1303 | BYTE_IDENTICAL |
| /api/v1/system/connector-plugins           |   541 |   541 | BYTE_IDENTICAL |
| /api/sync/health                           |   120 |   120 | BYTE_IDENTICAL |

### Notes
- Architectural elegance choice (CLAUDE.md §6): handler ctor took `kafkaConnectURL string` first, refactored to `*infrahttp.KafkaConnectClient` so 1 client backs all 8 endpoints (3 reads via query handlers + 5 writes via handler.client). Avoids 2 connection pools / 2 timeout configs.
- `internal/infra/http/doc.go` placeholder note ("P2 moves the Kafka Connect client here") is now satisfied.
- 5 connector writes (Restart, RestartTask, Create, Delete, Pause/Resume) intentionally LEFT in handler.go — P3 will move them to commands + worker per refactor plan.
- Task #156 (P2.T2.6) → completed.


---

## 2026-05-06 09:37 — P2.T2.7 (4 sub-tasks) DONE — 18/18 BYTE_IDENTICAL

### Scope decision (CLAUDE.md §6 elegance)

Plan called for "Wizard, Alerts, Users, AdminAudit (4 endpoints)". Actual remaining reads in code:

| Read endpoint | Decision |
|---------------|----------|
| /api/activity-log + /stats | **MIGRATED** (T2.7a) — heavy LATERAL JOIN SQL, biggest win |
| /api/v1/schedules (TransmuteSchedule) | **MIGRATED** (T2.7b) |
| /api/v1/sources List + Get | **MIGRATED** (T2.7c) |
| /api/v1/wizard/sessions/:id + /progress | **MIGRATED** (T2.7c) |
| /api/worker-schedule (ScheduleHandler) | **MIGRATED** (T2.7d) — replaces dead RegistryHandler.List |
| /api/registry (RegistryHandler.List) | SKIPPED — dead code (no route in router.go, no internal callers; comment line 75 declares it intentionally unmounted) |
| /api/alerts/{active,silenced,history} | SKIPPED — `service.AlertManager` already at the right read abstraction; wrapping adds no value. P3 will split AlertManager into commands when Ack/Silence move to commands.go. |
| /api/health (SystemHealth) | SKIPPED (already noted in T2.5) — Redis cache reader, no SQL surface |
| /api/v1/source-objects/registry/:id/dispatch-status, transform-status | RegistryHandler — write-side state, P3 |
| /api/* writes (Register, Standardize, Approve, etc.) | P3 (commands) |

### Files (T2.7a — ActivityLog: 4 NEW + 1 MOD + 1 wire)
- NEW `internal/app/queries/activity_log_read_models.go` — `ActivityLogRow` (19 fields) + `OpStat` (5 fields)
- NEW `internal/app/queries/list_activity_logs.go` — `ActivityLogReader` port + `ActivityLogFilter` + `ListActivityLogsHandler`
- NEW `internal/app/queries/get_activity_stats.go` — `GetActivityStatsHandler`
- NEW `internal/infra/persistence/activity_log_read_repo_gorm.go` — full SQL with shared `baseFromClause` + `projectionColumns`
- MOD `internal/api/activity_log_handler.go` — handler now delegates List + Stats; type aliases keep Swagger compat
- MOD `internal/server/server.go` — wire reader + 2 query handlers

### Files (T2.7b — TransmuteSchedule: 2 NEW + 1 MOD + 1 wire)
- NEW `internal/app/queries/list_transmute_schedules.go` — `TransmuteScheduleRow` + `TransmuteScheduleReader` + handler
- NEW `internal/infra/persistence/transmute_schedule_read_repo_gorm.go` — Raw SQL with master_binding LEFT JOIN
- MOD `internal/api/transmute_schedule_handler.go` — `ScheduleRow` is now `= queries.TransmuteScheduleRow` alias
- MOD `internal/server/server.go` — wire

### Files (T2.7c — Sources + Wizard: 2 NEW + 2 MOD + 1 wire)
- NEW `internal/app/queries/list_sources.go` — `SourceReader` port + List + GetSource handlers
- NEW `internal/app/queries/get_wizard_session.go` — `WizardReader` port + GetSession + GetProgress handlers (`WizardProgressView` fields alphabetized per Lesson #1294)
- MOD `internal/api/sources_handler.go` — handler no longer holds `*repository.SourceRepo`, only the 2 query handlers
- MOD `internal/api/wizard_handler.go` — Get + Progress delegate
- MOD `internal/server/server.go` — wire (existing `*SourceRepo` and `*WizardRepo` already satisfy reader interfaces; Strangler Fig — defer adapter rewrite to P4)

### Files (T2.7d — WorkerSchedule: 2 NEW + 1 MOD + 1 wire)
- NEW `internal/app/queries/list_worker_schedules.go` — `WorkerScheduleScope` + `WorkerScheduleResponse` + `WorkerScheduleReader` (port has both ListResponses + GetResponseByID since Create/Update need read-after-write projection) + ListHandler
- NEW `internal/infra/persistence/worker_schedule_read_repo_gorm.go` — moved 50-line SQL with LATERAL joins; flat `scanRow` projects to nested `WorkerScheduleResponse.Scope`
- MOD `internal/api/schedule_handler.go` — deleted `workerScheduleScanRow` + 96-line `listResponses` body; `getResponseByID` is now a 1-line shim over `h.reader.GetResponseByID`. Type aliases (`WorkerScheduleScope = queries.WorkerScheduleScope`, `WorkerScheduleResponse = queries.WorkerScheduleResponse`) preserve Swagger compat.
- MOD `internal/server/server.go` — wire reader + listQ; ScheduleHandler ctor signature +2 params

### Verify
- `go build ./...` PASS
- `go vet ./...` PASS (silent)
- `go test ./internal/api/... ./internal/app/queries/... ./internal/infra/...` → ok api 0.735s
- Restart :8093 with `/tmp/cdc-cms-service-t27` (env `CMS_SERVER_PORT=:8093 CMS_SERVER_NAME=cms-p2t27`); boot clean (PID 33841)
- Live diff vs :8083 baseline:

| Endpoint | legacy | new | verdict |
|----------|--------|-----|---------|
| /api/activity-log?page=1&page_size=20         |  6685 |  6685 | BYTE_IDENTICAL |
| /api/activity-log/stats                       |  5879 |  5879 | BYTE_IDENTICAL |
| /api/v1/schedules                             |  2797 |  2797 | BYTE_IDENTICAL |
| /api/v1/sources                               |    21 |    21 | BYTE_IDENTICAL |
| /api/worker-schedule                          |  2367 |  2367 | BYTE_IDENTICAL |
| /api/v1/sources/1 (404 path)                  |    21 |    21 | BYTE_IDENTICAL |
| /api/v1/sources/999999999 (404 path)          |    21 |    21 | BYTE_IDENTICAL |
| /api/v1/wizard/sessions/:id (404 path)        |    21 |    21 | BYTE_IDENTICAL |
| /api/v1/wizard/sessions/:id/progress (404)    |    21 |    21 | BYTE_IDENTICAL |
| 9 prior endpoints (T2.4–T2.6 regression)      | (varies) | (varies) | BYTE_IDENTICAL |

Total: 18/18 BYTE_IDENTICAL across the cumulative P2 surface.

### Notes
- ScheduleHandler.Update + Create still call `h.getResponseByID` post-write — that path is now a 1-line shim around `reader.GetResponseByID`. P3 will fold this into command handlers (ApplyScheduleUpdate emits ScheduleUpdated event with the same projection inline).
- Tasks #157, #158, #159, #160 → completed.
- Remaining P2: T2.8 (test coverage gate ≥60% for `internal/app/queries`).

---

## 2026-05-06 — P2.T2.8 done — Test coverage gate (Phase 2 / P2 closed)

### Change
- NEW `internal/app/queries/queries_test.go` — 600+ LOC, single test file, hand-rolled stubs per reader port. No mocking framework — pure interface stubs configurable via struct fields.

### Coverage
- Baseline: 0.0% (P2.T2.0 → T2.7 đã ship code mà chưa có test).
- Sau khi merge: **100.0% of statements** trong `internal/app/queries`.
- Gate yêu cầu: ≥60%. Vượt 40 điểm.

```
$ go test -cover ./internal/app/queries/...
ok  	cdc-cms-service/internal/app/queries	0.460s	coverage: 100.0% of statements
```

### Test surface (17 query handlers / 9 reader ports)
| Reader port | Handlers covered | Boundary tests |
|-------------|------------------|----------------|
| `ports.MappingRuleRepo` | ListMappingRulesHandler | page<1, size<1, size>200 |
| `SourceObjectReader` | ListSourceObjects + GetMappingContext | page<=0 → default 20; size>500 → 500 |
| `MasterReader` | ListMastersHandler | n/a (unfiltered) |
| `ReconReader` | ListLatestReports + GetTableHistory + ListFailedLogs | history: page<1,size<1,size>100; failed: page<1,size<1,size>200 |
| `SyncHealthReader` | GetSyncHealthHandler | n/a |
| `ConnectorReader` | ListConnectors + GetConnector + ListConnectorPlugins | 4-branch matrix on List (good / no-status / no-config / neither); FilterSafeConfig masking assertion |
| `ActivityLogReader` | ListActivityLogs + GetActivityStats | page<1, size<1, size>200 |
| `TransmuteScheduleReader` | ListTransmuteSchedules | n/a |
| `SourceReader` | ListSources + GetSource | n/a |
| `WizardReader` | GetWizardSession + GetWizardProgress | progress_log RawMessage round-trip; UpdatedAt equality |
| `WorkerScheduleReader` | ListWorkerSchedules | n/a |

Plus `TestQueryTypes` — covers all 19 `Query.Type()` getters in one table-driven sweep.

### Verification
```
$ go build ./... ; echo $?
0
$ go vet ./internal/app/queries/...
(no output)
$ go test -count=1 ./...
ok  cdc-cms-service/internal/api          0.951s
ok  cdc-cms-service/internal/app/queries  0.460s
ok  cdc-cms-service/internal/middleware   2.048s
ok  cdc-cms-service/internal/service      1.481s
```
Zero regression — pre-existing api/middleware/service tests vẫn xanh.

### P2 closure summary
- 8 sub-tasks (T2.0 → T2.7d) đã ship 18 byte-identical endpoints.
- T2.8 coverage gate: PASS (100% > 60%).
- Tổng: P2 (Read migration to `internal/app/queries/`) — **DONE**.
- Task #161 → completed.
- Pending: P3 (Write commands + JobMonitor), P4 (infra cleanup).

### Lesson candidate (sẽ update vào `agent/memory/global/lessons.md` khi đóng P3+P4)
- **Global Pattern [Stub-port testing for CQRS Q-handlers]**: Khi handler A phụ thuộc reader port B, viết stub C thay vì mock framework D — C là struct với public fields cấu hình return value + nhận lại side-effect args. Đúng: 1 file test/package, ≥1 happy + ≥1 error path / handler, boundary tests cho clamp logic. Lợi thế: zero deps, compile-time interface check, dễ debug khi assertion fail.

---

## 2026-05-06 — Phase 2 V2 / P3 (Write side: CommandBus + cdc_jobs + 2 canonical migrations) — DONE (canonical), PARTIAL (mechanical follow-ups)

### Tasks closed

| ID    | Status   | Detail |
|-------|----------|--------|
| T3.1  | DONE     | `internal/app/ports/command_bus.go` — CommandBus + Command port + CommandResult với `ResultBody json.RawMessage` cho sync inline response |
| T3.2  | DONE     | `internal/domain/job/job.go` (state machine), `internal/app/ports/job.go` (JobRepo), `internal/infra/persistence/job_repo.go` (GORM impl) — bind vào `cdc_system.cdc_jobs` migration 052 |
| T3.3  | DONE     | `internal/infra/messaging/nats_command_bus.go` — hybrid sync (in-process map) + async (NATS subject map) routing, idempotency rehydrate, metadata via opaque ctx keys. Test 83.1% coverage |
| T3.4a | DONE     | `internal/app/commands/ack_alert.go` — canonical sync command (Type/Validate/Handler) wrapping AlertManager. Test 85.7% |
| T3.4b | DONE     | `internal/app/commands/recon_check.go` — canonical async command (no Handler — bus subject map publishes to worker) |
| T3.5a | DONE     | `internal/api/alerts_handler.go::Ack` — refactor service-call → bus.Dispatch, return `res.ResultBody` |
| T3.5b | DONE     | `internal/api/reconciliation_handler.go::TriggerCheck` — refactor `natsClient.Publish` → `bus.Dispatch`, return 202 với `job_id` |
| T3.10 | DONE     | `internal/app/queries/get_job.go` + `internal/api/job_handler.go` + `internal/router/router.go` mount `/api/v1/jobs/:id` — close-loop UI cho async commands |
| T3.4c | PARTIAL  | 6 sync metadata commands còn lại (CreateMappingRule, UpdateMappingRule, CreateMaster, RejectMaster, CreateWizard, PatchWizard) — pattern xác lập, mechanical follow-up |
| T3.5c | PARTIAL  | 13 async API handlers còn lại (recon-heal, retry-failed, debezium-signal, etc.) — 17 subjects đã pre-register tại `server.go`, swap `natsClient.Publish` → `bus.Dispatch` mechanical |
| T3.6  | DEFERRED | Worker subscribe `cdc.cmd.master-swap` — repo `centralized-data-service`, workspace `feature-cdc-worker-jobmonitor/` đề xuất |
| T3.7  | DEFERRED | Worker subscribe `cdc.cmd.v2-sync` — same |
| T3.8  | DEFERRED | Worker emit `cdc.evt.X.completed` cho 12 existing handlers — same |
| T3.9  | DEFERRED | Worker JobMonitor wildcard `cdc.evt.*.completed` → cdc_jobs UPDATE — same |
| T3.11 | BLOCKED  | Live smoke test cần user authorize: kill PID 18563 hoặc spin port khác |

### Files changed (12 total — 7 NEW, 5 EDIT)

| Path | Action | Note |
|------|--------|------|
| `internal/app/ports/command_bus.go` | EDIT | Thêm `import "encoding/json"` + `ResultBody json.RawMessage` vào CommandResult |
| `internal/infra/messaging/nats_command_bus.go` | NEW | 8196 bytes — hybrid sync+async core |
| `internal/infra/messaging/nats_command_bus_test.go` | NEW | 8 unit tests, 83.1% coverage |
| `internal/app/queries/get_job.go` | NEW | JobReader port + Query/Handler/View |
| `internal/api/job_handler.go` | NEW | GET /api/v1/jobs/:id Swagger-annotated |
| `internal/app/commands/ack_alert.go` | NEW | Canonical sync command |
| `internal/app/commands/recon_check.go` | NEW | Canonical async command |
| `internal/app/commands/commands_test.go` | NEW | 85.7% coverage |
| `internal/app/queries/queries_test.go` | EDIT | + GetJob test cases (100% maintained) |
| `internal/server/server.go` | EDIT | jobRepo + cmdBus init, 17 async subjects pre-registered, alert.ack sync handler late-bound after alertMgr |
| `internal/router/router.go` | EDIT | + jobHandler param, mount `/api/v1/jobs/:id` |
| `internal/api/alerts_handler.go` | EDIT | bus.Dispatch refactor |
| `internal/api/reconciliation_handler.go` | EDIT | bus.Dispatch refactor |

### Build & test evidence (re-verified 2026-05-06)
```
$ go build ./...
(no output)

$ go test -count=1 -cover ./internal/app/commands/ ./internal/infra/messaging/ ./internal/app/queries/
ok  cdc-cms-service/internal/app/commands     0.623s  coverage: 85.7% of statements
ok  cdc-cms-service/internal/infra/messaging  1.036s  coverage: 83.1% of statements
ok  cdc-cms-service/internal/app/queries      1.457s  coverage: 100.0% of statements
```

Live `cdc_system.cdc_jobs` đã verify 12 cols + 4 indexes + CHECK constraint (migration 052 đã apply).

### Design decisions (snapshot — chi tiết tại `report_phase2_v2_p3_2026-05-06.md`)

1. **Hybrid bus map registry** thay vì plugin chain → giảm lookup cost cho hot path sync (alert.ack), đẩy NATS overhead chỉ vào async path.
2. **Idempotency rehydrate at JobRepo.Create** thay vì bus check → repo là source of truth cho job state, bus chỉ short-circuit khi `Status != pending`.
3. **CommandResult.ResultBody json.RawMessage** thay vì interface{} — sync handler trả wire bytes trực tiếp, FE không cần marshal lại; async path để rỗng (FE poll `/jobs/:id`).
4. **Compile-time interface assertion** `var _ ports.CommandBus = (*natsCommandBus)(nil)` ngay sau type decl — lesson từ Phase 2 V1.

### Pending (mechanical, ~3-4h work còn lại)
- T3.4c: 6 sync command files theo pattern `ack_alert.go`
- T3.5c: 13 API handler edits swap `natsClient.Publish` → `bus.Dispatch`
- T3.11: User authorize live smoke test
- Worker workspace spin-up cho T3.6-T3.9

### Lesson candidate cho global (sẽ APPEND vào `lessons.md` cùng với progress log này)
**Global Pattern [Hybrid command bus cần ResultBody trên CommandResult cho sync handlers]**: Khi bus B route command C qua 2 path (in-process sync handler X / NATS publish Y), `Result` struct phải mang optional `ResultBody json.RawMessage` để X trả wire bytes inline (FE không poll). Đúng: declare `ResultBody json.RawMessage` (nullable, async path để rỗng). Sai: ép FE luôn poll `/jobs/:id` sau Dispatch — tăng RTT 2x cho sync path không cần thiết.



---

## 2026-05-06 — Boss critique của P3 plan + gap analysis (APPEND-ONLY)

**Trigger**: Boss review plan P3, nêu 5 facts sai/off-by-line + 6 design gaps + 3 questions. Muscle đã verify từng claim với evidence trực tiếp (file LOC, line numbers, existing migrations, existing evt subjects). KẾT QUẢ:

### Facts verified (boss đúng tất cả)
- F1: master_registry_handler.go = 616 LOC, swap @ line 598 dùng `h.swap.Swap` (đã extract). KHÔNG inline.
- F2: registry_handler.go SyncFromLegacy ở 168/288/338 (plan ghi 148/268/316).
- F3: `052_create_cdc_jobs.sql` đã tồn tại upstream — T3.1 = DONE.
- F4: `cdc.evt.provisioning.step_completed` + `cdc.evt.transmute.completed` đã có. REUSE not NEW.
- F5: Worker `JobMonitor` đã subscribe → CMS-side cần wildcard `cdc.evt.>`, không list 12 subject.

### Gaps đề xuất fix
- G1 (CRITICAL): StuckJobReaper cron 30s quét pending timeout — task #170 created.
- G2: Idempotency `ON CONFLICT (idempotency_key) DO UPDATE ... RETURNING id`. Coexist với Redis middleware.
- G3 (BLOCKER cho T3.6): Worker DDL permission cho master-swap. Đề xuất đường 1 — giữ Swap ở cms-service + wrap qua cdc_jobs (KHÔNG publish NATS).
- G4: Unify `cdc.evt.provisioning.step_completed` cho cả master-create + master.bind (tránh JobMonitor double-update).
- G5: Tách SyncCommand vs AsyncCommand interface — sync 7 commands không tạo job row (tránh +1 round-trip cho action 50ms).
- G6: 12 companion evt → tách 12 sub-task canary, không deploy bulk.

### Effort revised: 5d → 7-7.5d (chấp nhận, document đầy đủ vào 02_plan)

### Trả lời 3 questions của boss (đề xuất)
- Q1 Master swap purpose: **(b) consistency** → đường 1 G3 (cms-side wrap).
- Q2 GET /jobs/:id tier: **shared** (admin + operator) — chỉ cần requireAuth().
- Q3 Idempotency-Key middleware: **coexist với cdc_jobs.idempotency_key UNIQUE** — khác layer.

### Files
- `10_gap_analysis_p3_critique_2026-05-06.md` — analysis chi tiết.
- Task #170 created — pending boss approve.

### T3.5c progress trước critique
- registry_handler.go: 6/7 sites migrated (4 trước + 2 mới: create-default-columns line 542, detect-timestamp-field line 635). 1 site batch-transform DEFER.
- mapping_rule_handler.go: 3/3 sites migrated (Backfill, BatchUpdate alter-column, BatchUpdate backfill loop). Constructor sig added bus param.
- server.go: registry+mapping handler call sites updated với cmdBus.
- system_async_test.go: NEW unit test cho RestartDebeziumCommand.
- Build: `go build ./...` PASS. Tests: full `go test -count=1 ./...` PASS.

### Status
T3.5 ~95% done (chỉ còn batch-transform DEFER). T3.6+ BLOCKED chờ boss approve Q1-Q3 + 6 gaps.

---

## 2026-05-06 — Boss critique round 2 (verdicts) → Muscle response (APPEND-ONLY)

**Trigger**: Boss review từng đề xuất trong `10_gap_analysis_p3_critique_2026-05-06.md`.

### Verdict summary
- ✅ APPROVED: G2 (idempotency với xmax check), G5 (split spec), Q2 (shared tier).
- ⚠ REVISE: G1 (per-type timeout), Q3 (Redis TTL 5-10min + reaper invalidate).
- 🔴 REPLACED: G4 (boss alt: 2 subject + wildcard), G6 (feature-flag per-subject thay 12 canary).
- 🔴 BLOCKED: G3 + Q1 master-swap async-mode (async-in-goroutine | sync-block | async-via-NATS) — Brain phải clarify.

### T3.5 verify result
- Bus interface hiện CHƯA split G5 spec → cần T3.5c refactor (~0.3d). Task #171 NEW.
- Test coverage thực tế đo: commands 90.4%, bus 86.4%. Boss "1 test/command" misread — coverage đầy đủ.
- batch-transform DEFER có lý do kỹ thuật rõ: worker đọc raw bytes, encoding/json reject MarshalJSON non-JSON output. Document đầy đủ.

### Files
- `04_decisions_p3_critique_round2_2026-05-06.md` — verdict matrix + Q1 clarification request.
- Tasks: #170 G1 reaper (REVISE per-type timeout), #171 T3.5c (NEW interface split).
- Updated: #167 T3.5 (HOLD pending T3.5c).

### Effort revised: 7-7.5d → 8-8.5d
T3.5c (NEW 0.3d) + T3.6 BLOCKED + T3.8 feature-flag (G6 REVISED 1.5d) + T3.12 per-type (G1 REVISED 0.6d).

### Status
T3.5 = HOLD (95% migration done, chờ T3.5c).
T3.6 = BLOCKED (Brain clarify Q1).
Muscle KHÔNG implement thêm cho đến khi boss approve toàn plan.

---

## 2026-05-06 — T3.5c IMPLEMENTED (Interface split SyncCommand/AsyncCommand)

**Trigger**: Boss approve toàn plan với 8 verdicts. Q1 master-swap = (a) async-in-goroutine + partial-state detector. Brain proceed T3.5c → T3.6 → T3.7-T3.12.

### Implementation
- **`internal/app/ports/command_bus.go`** REWRITTEN:
  - Split single `Command` → 3 interfaces: base `Command{Type,Validate}`, `SyncCommand` (Command + `syncCommandKind()`), `AsyncCommand` (Command + `asyncCommandKind()`).
  - Mixin embedding pattern: `SyncCommandMixin` zero-size struct + unexported method, embedders inherit kind marker for free. gRPC-Go convention.
  - Result types split: `SyncResult{JobID, ResultBody}` vs `AsyncResult{JobID, Accepted}`.
  - `CommandBus.Execute(ctx, SyncCommand) → SyncResult` + `CommandBus.Dispatch(ctx, AsyncCommand) → AsyncResult`.
- **`internal/infra/messaging/nats_command_bus.go`** REWRITTEN:
  - Common `prepare()` helper: validate → marshal → JobRepo.Create → idempotency short-circuit detect.
  - `Execute` → in-process sync map `b.sync[Type]`. Inline status update on success/fail. ResultBody bubbles to caller.
  - `Dispatch` → subject map → NATS publish. Headers `Cdc-Job-Id`, `Cdc-Correlation-Id`, `Cdc-Created-By`, `Cdc-Command-Type`. Status='pending' until JobMonitor closes loop.
  - Idempotency replay: `prepare()` returns `short=true` khi rehydrated row đã có Status terminal → caller bỏ qua handler/publish, trả ResultBody từ stored `j.Result`.
- **14 command structs migrated** (1 sync + 13 async):
  - `internal/app/commands/ack_alert.go` — embed `ports.SyncCommandMixin`.
  - `internal/app/commands/recon_check.go`, `recon_async.go`, `source_async.go`, `system_async.go` — embed `ports.AsyncCommandMixin`.
  - All add `cdc-cms-service/internal/app/ports` import.
- **`internal/api/alerts_handler.go`**: `bus.Dispatch(ctx, cmd)` → `bus.Execute(ctx, cmd)` (chỉ 1 call site sync).
- **`internal/infra/messaging/nats_command_bus_test.go`** REWRITTEN:
  - 2 fake types `fakeSyncCmd`/`fakeAsyncCmd` embed mixin tương ứng.
  - Tests split: `TestExecute_*` (Sync handler path) + `TestDispatch_*` (NATS publish path).
  - `TestExecute_IdempotentReturnsExisting` verify replay path: bus return stored `ResultBody`, không gọi handler lại.

### Verification
- `go build ./...` PASS.
- `go test -count=1 ./...` ALL PASS:
  - `internal/app/commands` — coverage 90.4%.
  - `internal/infra/messaging` — coverage 84.5% (giảm nhẹ từ 86.4% do bus code grow Execute path; vẫn trên ngưỡng 80%).
  - `internal/api`, `internal/app/queries`, `internal/middleware`, `internal/service` — đều PASS.
- Wire format unchanged: `MarshalJSON` của command vẫn output raw JSON byte-identical với pre-T3.5 publishers (mixin embed zero-size struct → không pollute JSON).

### Lesson learned (Pattern Global)
- **Pattern A**: "Type marker method ở interface X package Y → impl ở package Z không cross boundary nếu method unexported". Solution: Mixin embedding pattern — package Y export `XMixin` struct + unexported method; package Z embed `Y.XMixin`. Type-safe + zero-size.
- **Pattern B**: Khi split single interface thành nhiều variant theo dispatch mode (sync/async), tách entire chain (struct → bus method → result type) chứ không hybrid. Hybrid (1 interface + dispatch param) lose compile-time safety.
- Append vào `agent/memory/global/lessons.md` sau khi T3.6 land xong (avoid context churn mid-work).

### Tasks
- #171 T3.5c → COMPLETED.
- #167 T3.5 → COMPLETED (95% migration + interface split land).
- Next: T3.6 master-swap (boss Q1=(a) async-in-goroutine + partial-state detector).

---

## 2026-05-06 — T3.6 IMPLEMENTED (Master Swap async-in-goroutine + partial-state detector)

**Trigger**: Boss Q1=(a) approved. POST /api/v1/masters/:name/swap stop blocking on the 2-RENAME TX; instead persist a job row, kick off detached goroutine, return 202 + JobID. FE polls GET /api/jobs/:id.

### Implementation
- **`internal/service/master_swap.go`** REWRITTEN:
  - Constructor signature change: `NewMasterSwap(db, jobRepo, logger)` (was `(db, logger)`).
  - New public `SwapAsync(ctx, masterName, newTableName, reason, createdBy, correlationID) (jobID, error)`:
    1. Validate identifiers via existing `validateIdent()` (length ≤63, lowercase + digits + underscore).
    2. `detectPartialState()` probe — refuse with `master_swap_in_flight` if another `master.swap` job for the same masterName is still pending/running.
    3. Persist `cdc_jobs` row via `jobRepo.Create` (idempotency-key-less, since master swap không idempotent intrinsically — every swap creates new `_old_<ts>` table).
    4. `go runSwapGoroutine(...)` with detached `context.Background()` (HTTP ctx canceled on 202 write).
    5. Return jobID synchronously.
  - New private `detectPartialState(ctx, masterName)` — Postgres-specific raw SQL: `SELECT count(*) FROM cdc_system.cdc_jobs WHERE type='master.swap' AND status IN ('pending','running') AND payload->>'master_name' = ?`.
  - New private `runSwapGoroutine(jobID, ...)`:
    - 30s timeout context (master swap should never take >30s with `lock_timeout=3s`).
    - Defer `recover()` → mark FAILED on panic.
    - Mark RUNNING → call `runSwapTX` → mark SUCCESS/FAILED based on result.
    - Special-case lock-timeout error → prepend `lock_timeout:` to err msg for FE classification.
  - Renamed old `Swap` → private `runSwapTX` (logic unchanged: 2-RENAME TX with `SET LOCAL lock_timeout='3s'` + activity_log INSERT).
- **`internal/api/master_registry_handler.go::Swap`** UPDATED:
  - Replace blocking `h.swap.Swap(c.Context(), ...)` with `h.swap.SwapAsync(c.Context(), ...)`.
  - Pull `createdBy` via `getActor(c)`, `correlationID` via `c.Locals("correlation_id")`.
  - Map errors: `master_swap_in_flight` → 409, `invalid_*` → 400, others → 500.
  - Success path: `Status(202).JSON({status:"accepted", master_name, job_id})`.
  - Updated swagger annotations (Success 202, description mentions async + 409 conflict).
- **`internal/server/server.go:197`** UPDATED:
  - `service.NewMasterSwap(db, jobRepo, logger)` — pass jobRepo from line 153.

### New tests (`internal/service/master_swap_test.go`)
- `TestSwapAsync_RejectsInvalidMasterName` — bad chars in master name → "invalid master_name" err, no DB call.
- `TestSwapAsync_RejectsInvalidNewTableName` — space in new table name → "invalid new_table_name" err, no DB call.
- Compile-time check `var _ ports.JobRepo = (*stubJobRepoForSwap)(nil)` — guards stub against drift.
- DB-dependent paths (partial-state probe, runSwapTX RENAME chain) deferred to deploy-time E2E — `cdc_system.cdc_jobs` schema-qualified table + Postgres `->>` JSON operator unportable to sqlite, no testcontainers harness in repo today.

### Verification
- `go build ./...` PASS.
- `go test -count=1 ./...` ALL PASS (api/commands/queries/messaging/middleware/service all green).
- Wire surface change: `POST /masters/:name/swap` 200 → 202 + new JobID field. **Breaking change** for FE callers — needs FE side update.

### Architecture notes (Pattern Global candidate)
- **Pattern A**: HTTP handler kicks off in-process long-op + returns 202 → handler responsibilities = (validate, persist job row, spawn goroutine, return jobID); goroutine = detached ctx + state machine pending→running→terminal + recover panic + bounded timeout.
- **Pattern B**: Partial-state detector at-dispatch-time (NOT at-startup-time) → blocks duplicate concurrent operations on the same physical resource without needing global locks. Generic stuck-job recovery is a separate concern (T3.12 reaper).
- Append to `agent/memory/global/lessons.md` after T3.7-T3.12 land.

### Tasks
- #172 T3.6 → COMPLETED.
- Next: T3.7 source-v2-sync (Pattern A reuse), T3.8 (12 evt feature-flag), T3.9 (JobMonitor wildcard), T3.4 (7 sync commands using new interface), T3.12 (StuckJobReaper), T3.11 (final verify).

---

## 2026-05-06 — T3.12 IMPLEMENTED (StuckJobReaper per-type timeout)

**Trigger**: Boss G1 REVISE approved. Flat 30s reaper false-positives recon.check on 50GB shadow (legitimate >5min runtime). Per-type timeout map fixes the false-positive without schema change.

**Scope decision**: T3.7 (worker v2-sync), T3.8 (worker emit 12 evt), T3.9 (worker JobMonitor wildcard) are **WORKER-SIDE work** (centralized-data-service repo, NOT cdc-cms-service). Tasks #173/#174/#175 deleted from CMS workspace; tracked separately in worker workspace when that lands. Boss Q1=(a) verdict pulled master.swap back to CMS-only (no longer worker T3.6) — same logic applies to T3.7 v2-sync (CMS-owned metadata, no permission boundary issue, stays inline).

### Implementation (T3.12)
- **`internal/service/stuck_job_reaper.go`** NEW (~135 LOC):
  - `StuckJobReaper{db, logger, interval, timeouts map[string]time.Duration, defaultTO}`.
  - `DefaultJobTimeouts()` map ships per-type timeouts for all 14 NATS commands + master.swap. Tunable via constructor.
    - `master.swap`: 60s (goroutine bounded ≤30s, reaper at 60s = cushion).
    - `recon.check`: 10m (50GB shadow upper-bound).
    - `transmute`: 15m (long batches).
    - `mapping.backfill`: 30m (largest column-fill operation).
    - others 1-15m per realistic worker p99.
    - default 30s for unknown types.
  - `Run(ctx)` ticker loop; `defer t.Stop()`; ctx-cancel exits cleanly.
  - `reapOnce(ctx)` — single-roundtrip UPDATE with deterministic CASE expression:
    ```sql
    UPDATE cdc_system.cdc_jobs
       SET status='failed',
           error_message=COALESCE(NULLIF(error_message, ''), 'reaper: timeout exceeded'),
           finished_at=NOW()
     WHERE status='running'
       AND started_at IS NOT NULL
       AND started_at + (interval '1 second' * (CASE type WHEN ? THEN ? ... ELSE ? END)) < NOW()
    ```
    Sorted keys → stable plan cache. Single UPDATE per tick regardless of type-count.
- **`internal/server/server.go`** UPDATED:
  - Struct: add `stuckJobReaper *service.StuckJobReaper` + `stuckJobReaperCancel context.CancelFunc`.
  - `New()`: `service.NewStuckJobReaper(db, logger, 30*time.Second, nil)` (defaults ok).
  - `Start()`: `go s.stuckJobReaper.Run(ctx)` with cancel pair.
  - `Shutdown()`: cancel hook to stop ticker on graceful exit.
  - Add `"time"` import.
- **`internal/service/stuck_job_reaper_test.go`** NEW:
  - `TestDefaultJobTimeouts_ContainsCriticalTypes` — guards master.swap ≥30s, recon.check ≥5min, transmute ≥5min (boss G1 floor).
  - `TestNewStuckJobReaper_AppliesDefaults` — interval=0 → 30s, timeouts=nil → DefaultJobTimeouts.
  - `TestNewStuckJobReaper_AcceptsCustom` — custom map overrides; doesn't auto-merge defaults.
  - `reapOnce` SQL not unit-tested (Postgres-specific `interval` syntax; deferred E2E).

### Verification
- `go build ./...` PASS.
- `go test -count=1 ./...` ALL PASS (api/commands/queries/messaging/middleware/service all green).

### Architecture notes
- **Pattern Global candidate**: "Periodic sweep cron with per-type configurable timeout via in-process map" — avoids schema change for tunables, single-roundtrip UPDATE with CASE expression keeps the sweep O(1) regardless of type-count.

### Tasks
- #170 T3.12 → COMPLETED.
- #173 T3.7, #174 T3.8, #175 T3.9 → DELETED (worker-side, separate workspace).
- Next CMS-side: T3.4 (6 remaining sync metadata commands), T3.11 (final smoke A1-A3).

### Cumulative status this session (P3)
- DONE: T3.1, T3.2, T3.3, T3.5, T3.5c, T3.6, T3.10, T3.12, HOTFIX (9 tasks).
- IN PROGRESS: T3.4 (6/7 commands remaining — AckAlert done, 6 to go).
- PENDING: T3.11 (final verify after T3.4 lands).
- WORKER-SIDE (out of CMS scope): T3.7, T3.8, T3.9 — track in centralized-data-service workspace.


---

## 2026-05-06 — T3.4 hoàn thành (Muscle CC CLI)

### Scope
Migrate 7 sync metadata commands sang `internal/app/commands/` qua `cmdBus.RegisterSync(...)`. Sau T3.5c interface split (SyncCommandMixin), mỗi command struct embed mixin → satisfies ports.SyncCommand → routed qua `bus.Execute(ctx, cmd)`.

### Files mới (commands package)
- `internal/app/commands/update_mapping_rule.go` — UpdateMappingRuleCommand + Handler
- `internal/app/commands/create_mapping_rule.go` — CreateMappingRuleCommand + Handler (scope resolve + INSERT + re-fetch)
- `internal/app/commands/reject_master.go` — RejectMasterCommand + Handler (lookup + UPDATE)
- `internal/app/commands/create_master.go` — CreateMasterCommand + Handler (resolve shadow_binding + master_connection + INSERT)
- `internal/app/commands/create_wizard.go` — CreateWizardCommand + Handler (UUID + repo.Create)
- `internal/app/commands/patch_wizard.go` — PatchWizardCommand + Handler (allow-list updates + repo.Update + re-fetch)
- `internal/app/commands/sync_metadata_test.go` — Type/Validate guard tests cho cả 6 (+1 type-mismatch)

### Files sửa (API + server)
- `internal/api/mapping_rule_handler.go::UpdateStatus` — pre-validate (status required) → bus.Execute → map `ErrMappingRuleNotFound` → 404
- `internal/api/mapping_rule_handler.go::Create` — pre-validate (3 fields required) → bus.Execute → map `ErrMappingScopeNotFound`/`ErrMappingScopeAmbiguous`/`ErrMappingRuleAlreadyExists` → 404/409/409
- `internal/api/master_registry_handler.go::Create` — pre-validate name/schema/transform/reason → bus.Execute → map 5 sentinels (ShadowBinding x2, MasterConnection x2, AlreadyExists)
- `internal/api/master_registry_handler.go::Reject` — pre-validate name + reason → bus.Execute → map `ErrMasterNotFound` → 404, `ErrMasterNameAmbiguous` → 409
- `internal/api/master_registry_handler.go` — struct +bus field; constructor +bus param
- `internal/api/wizard_handler.go::Create` — bus.Execute (handler builds *model.WizardSession with UUID)
- `internal/api/wizard_handler.go::Patch` — bus.Execute với `ErrWizardInvalidStatus`/`ErrWizardNothingToPatch` → 400
- `internal/api/wizard_handler.go` — struct +bus field; constructor +bus param
- `internal/server/server.go` — 6 dòng `cmdBus.RegisterSync(...)` mới: `mapping.update-status`, `mapping.create`, `master.reject`, `master.create`, `wizard.create`, `wizard.patch`
- `internal/server/server.go` — call sites cho `NewMasterRegistryHandler` + `NewWizardHandler` thêm cmdBus param

### Pattern khoá (Vietnamese, để Brain reference)
**Pre-validate ở API boundary trước bus.Execute** — vì test fixture có thể tạo handler `&Foo{}` (bus=nil), validation trong Command (chạy trong bus.Execute) sẽ không reach. Pre-validate cheap input (string blank, regex, enum) ở API trước khi check bus, đẩy logic-validation (DB-dependent: scope resolve, name lookup) vào Command.Validate()/Handler.Handle().

**Sentinel error pattern** — mỗi command file export `ErrXxx = errors.New("error_code")` thay string-matching. API map qua `errors.Is(err, commands.ErrXxx)` → status code. Tránh fragile string parsing.

### Verification
1. `go build ./...` PASS.
2. `go test -count=1 ./...` PASS — toàn bộ packages: api, app/commands, app/queries, infra/messaging, middleware, service.
3. New guard tests: 7 test functions cover Type() + Validate() + 1 type-mismatch.

### Cumulative state (P3 phase 2)
- DONE: T3.1, T3.2, T3.3, T3.4, T3.5, T3.5c, T3.6, T3.10, T3.12, HOTFIX (10 tasks).
- PENDING: T3.11 (final verify + report — cleanup stale `RegisterSubject("master.swap","cdc.cmd.master-swap")` line 193, `("source.v2-sync","cdc.cmd.v2-sync")` line 194 + smoke A1-A3).
- WORKER-SIDE (out of CMS scope): T3.7, T3.8, T3.9 — tracked at centralized-data-service.

### Out of scope (Boss G1 directive)
Không lay vào worker-side T3.7/T3.8/T3.9 (centralized-data-service repo). Lines 193-194 trong server.go là dead RegisterSubject (worker chưa subscribe) — sẽ clean trong T3.11.

---

## 2026-05-06 — Cleanup server.go + phát hiện gap T3.13

### Cleanup (subtractive, low-risk)
- Xóa `internal/server/server.go:193-194`:
  - `cmdBus.RegisterSubject("master.swap", "cdc.cmd.master-swap")` — T3.6 đã chuyển sang in-process goroutine, subject không subscribe.
  - `cmdBus.RegisterSubject("source.v2-sync", "cdc.cmd.v2-sync")` — T3.7 reroute sang CMS, subject không subscribe.
- Verified: `go build ./...` PASS, `go test ./internal/infra/messaging/...` PASS.
- Còn 1 mention `master.swap` ở comment doc `nats_command_bus.go:13` — context giải thích lịch sử, giữ.

### Gap phát hiện trong verify A3 — T3.13 (Brain)
**Verify A3 idempotency wiring**: Brain accept đầy đủ 2 layer:
1. Layer 1 — `service/master_swap.go:98 SwapAsync` application-level dedup: `SELECT count(*) FROM cdc_jobs WHERE type='master.swap' AND status IN (pending,running) AND payload->>'master_name'=?` → 409 `master_swap_in_flight` cho admin click cùng master khác key.
2. Layer 2 — `infra/messaging/nats_command_bus.go:242-247 prepare() → JobRepo.Create` job-level idempotency: rehydrate same job_id cho retry cùng `Idempotency-Key`.

**Gap thật phát hiện**: `infra/persistence/job_repo_gorm.go:103-140 Create()` lệch G2 spec đã accept. Implementation hiện tại read-then-create:
```go
if j.IdempotencyKey != "" {
    Where("idempotency_key = ?", key).Take(&existing)  // SELECT
    if found { rehydrate; return }
}
db.Create(row)  // INSERT, fail UNIQUE if race
```

**Race window**: 2 concurrent retry cùng key → cả 2 SELECT miss → cả 2 INSERT → row 2 trượt UNIQUE → caller nhận generic `duplicate key` error thay vì rehydrate. Window hẹp (ms) nhưng tồn tại.

**Fix proposed (T3.13, ~0.3d)**: chuyển single round-trip atomic:
```go
db.Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "idempotency_key"}},
    DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
}).Create(row)
// re-Take để rehydrate khi conflict
```

**Decision**: Không block T3.11 smoke. Tạo task #176 P3.T3.13 schedule sau T3.11.

### A1-A3 smoke matrix (Brain proposed — Muscle ack)
Order: A2 (read-only) → A-bonus reaper (self-contained) → A3 swap (CMS-only do T3.6 in-process goroutine) → A1 recon (cần worker, defer nếu lab không có).

Pre-flight 4 câu chờ Boss:
- ✅ Port CMS = 8083 (BRD §0.1).
- ✅ JWT secret = `change-me-in-production` (BRD §10 P0 #4) trừ khi rotate.
- ⚠ Worker `centralized-data-service` up status — chưa visibility.
- ⚠ master_binding row test-safe — chưa visibility.

**Block point**: chờ Boss confirm 2 caveat rồi Muscle execute matrix. Smoke không phải low-risk action (mutate state qua HTTP) → cần user approve.

### Cumulative state (P3 phase 2)
- DONE: T3.1, T3.2, T3.3, T3.4, T3.5, T3.5c, T3.6, T3.10, T3.12, HOTFIX (10 tasks).
- DONE-cleanup: server.go dead RegisterSubject 193-194 đã xóa.
- PENDING: T3.11 (smoke A1-A3 chờ Boss approve), T3.13 (G2 race fix scheduled phase sau).

---

## 2026-05-06 15:35 ICT — T3.11 smoke matrix executed + HOTFIX-2 reaper SQL bug | Muscle | claude-opus-4-7

### Context
Boss approve "A": kill PID 18555 (parent `go run`) rồi start `/tmp/cdc-cms-host-new`. Sau Boss `ok`: kill child PID 18563 (server thực holding :8083). Restart hoàn tất, smoke matrix A2 → A-bonus reaper → A3 → A1 thực thi. Trong quá trình A-bonus phát hiện bug T3.12 (HOTFIX-2). Sửa, rebuild, smoke pass.

### HOTFIX-2 — reaper SQL bug discovered
**Symptom**: stuck job inserted, sau 35s không flip status. CMS log spam mỗi 30s:
```
ERROR: operator does not exist: interval * text (SQLSTATE 42883)
```
Sau cast outer `::int` xuất hiện error mới:
```
failed to encode args[1]: unable to encode 120 into text format for text (OID 25)
```

**Root cause**: `internal/service/stuck_job_reaper.go:124` SQL `started_at + (interval '1 second' * (CASE type WHEN ? THEN ? ... END)) < NOW()`. GORM/pgx prepared-statement type inference resolve mỗi `?` THEN-branch là TEXT (vì `WHEN ?` được compare với cột `type` text → first ? = text → infer mọi alternating value cùng nhóm text). Cast `(CASE END)::int` ngoài không sửa được vì inference param types xảy ra TRƯỚC outer cast.

**Fix**: cast TỪNG positional `?::int` ngay tại CASE THEN/ELSE (`stuck_job_reaper.go:111,114`):
```go
caseExpr.WriteString("WHEN ? THEN ?::int ")
caseExpr.WriteString("ELSE ?::int END")
```

**Verification**: rebuild → restart → 30s → log `{"msg":"reaped stuck jobs","count":1}`. DB confirm status `running → failed`, `error_message='reaper: timeout exceeded'`, `finished_at=NOW()`. Reaper tick stable qua 4+ phút, count=1 hai lần (cũng tự catch row Layer 1 manual injection sau khi master.swap 60s timeout vượt → bằng chứng phụ reaper hoạt động trên đa kịch bản).

**Task tracking**: #177 P3.HOTFIX-2 created + completed.

### Restart sequence (4 lần)
1. `kill 18555` (parent `go run`) — Boss option A
2. `kill 18563` (child holding :8083) — Boss `ok`
3. `kill 50859` (agent-spawned `/tmp/cdc-cms-host-new`) → swap sang `/tmp/cdc-cms-host-fix` (outer-cast partial fix)
4. `kill 52602` (agent-spawned fix) → swap sang `/tmp/cdc-cms-host-fix2` (per-arg cast — final fix)
- Final stable PID: **53173**, uptime 4m46s, reaper sweep nominal, 0 errors.

### Smoke matrix evidence

**A2 — GET /api/jobs/:id (read query handler)**:
| Sub | Input | Got |
|---|---|---|
| A2.1 | non-existent UUID | 404 `{"error":"job_not_found"}` ✅ |
| A2.2 | stuck job pre-reaper | 200 status=running ✅ |
| A2.3 | same job post-reaper | 200 status=failed, error_message='reaper: timeout exceeded', finished_at!=null ✅ |
| A2.4 | recon.check pending job | 200 status=pending, payload={tier:"1",table:"orders"} ✅ |

**A-bonus reaper sweep**: manual INSERT cdc_jobs(type='master.swap', status='running', started_at=NOW()-5min). Wait 30s reaper tick. DB recheck status='failed', error='reaper: timeout exceeded'. PASS.

**A3 — POST /api/v1/masters/:name/swap (T3.6 in-process goroutine)**:
| Sub | Input | Got |
|---|---|---|
| A3.1 | smoke_master_a + new_table_name + reason 50ch + Idempotency-Key | 202 `{"job_id":"113b8b04...","status":"accepted"}` ✅. DB lifecycle pending→running→failed within 22ms. error_message: `rename current: ERROR: relation "public.smoke_master_a" does not exist` (đúng — không có binding) ✅ |
| A3.2 | same body 1s sau | 202 + new jobID `e2603f0d...` (Layer 1 không block vì A3.1 đã failed — đúng spec) ✅ |
| A3.3 | manual `running` row inject for smoke_master_b → POST swap | 409 `{"error":"master_swap_in_flight","detail":"1 job(s) still pending/running for smoke_master_b"}` ✅ — Layer 1 SQL count match |

**A1 — POST /api/reconciliation/check/:table (NATS async via bus.Dispatch)**:
| Sub | Input | Got |
|---|---|---|
| A1 | table=orders, tier=1, reason 50ch + Idempotency-Key | 202 `{"job_id":"92316114...","message":"reconciliation check dispatched","table":"orders","tier":"1"}` ✅. cdc_jobs row created: type='recon.check', status='pending', payload={tier:"1",table:"orders"} ✅ |

**Notes**:
- Path đúng là `/api/reconciliation/check/:table` (`/api` group, không `/api/v1`). Master swap khác — mounted `/api/v1/masters/:name/swap`. Mixed prefix xác nhận trong router.go.
- A1 idempotency_key trong cdc_jobs row = NULL — header `Idempotency-Key` không propagate vào job row qua bus path. Probable Layer 2 wire gap (header → command → JobRepo.Create). Out-of-scope T3.11; tách task riêng nếu Boss yêu cầu.
- Worker centralized-data-service không up trong session → A1 dừng ở status='pending' (NATS message còn trong queue). CMS-side dispatch path verified end-to-end (HTTP → bus → cdc_jobs persist + nats publish). Worker pickup là scope khác.

**Cleanup**: `DELETE FROM cdc_system.cdc_jobs WHERE id IN (...)` — 5 smoke rows removed (4 master.swap failed + 1 recon.check pending). Final cdc_jobs count = 0.

### DoD T3.11
| Tiêu chí | Status |
|---|---|
| GET /api/jobs/:id (4 sub-cases: missing/running/failed/pending) | ✅ |
| Reaper sweep stale running rows (post-HOTFIX-2) | ✅ |
| Master swap async-in-goroutine end-to-end (T3.6) | ✅ |
| Master swap Layer 1 partial-state detect (409) | ✅ |
| Recon dispatch via bus → cdc_jobs persist + NATS publish | ✅ |
| Discovered + fixed reaper SQL cast bug (HOTFIX-2) | ✅ bonus |
| No leftover smoke artifacts | ✅ |

### Global Pattern lesson (write to lessons.md sau)
**Pattern [GORM/pgx prepared-statement với CASE expression: column-A WHEN value-A THEN value-B → driver infer value-B param type theo nhóm với value-A, không theo outer arithmetic context] → Result [param types lệch → operator missing hoặc encoding error]**.
- **Đúng**: cast TỪNG positional `?::int` (hoặc `::numeric`/`::bigint`) ngay trong CASE THEN/ELSE expression.
- **Sai**: outer cast `(CASE … END)::int` không sửa inference của individual params; outer cast chỉ chuyển kiểu sau khi CASE đã evaluated.
- **Test gap**: integration test với mocked GORM hoặc sqlite không phát hiện vì khác driver type-inference.
- **Triggered ở**: bất kỳ SQL build động với GORM placeholder `?` trong CASE/IIF/COALESCE mà các nhánh trả kiểu khác cột so sánh.

### Cumulative state update (P3)
- DONE: T3.1, T3.2, T3.3, T3.4, T3.5, T3.5c, T3.6, T3.10, **T3.11**, T3.12, HOTFIX-1 (recon-check envelope), **HOTFIX-2 (reaper SQL cast)** — **12 tasks**.
- PENDING: T3.13 (G2 race fix — scheduled riêng).
- **P3 phase 2 verification gate hoàn tất**. T3.11 đóng. Sẵn sàng phase tiếp.

## 2026-05-06 16:02 ICT — Đợt 1 P3 hardening: T3.13 + T3.14 (D1) bundle ✓
- **T3.13** (#176): `JobRepo.Create` Read-then-Create → atomic `INSERT … ON CONFLICT (idempotency_key) DO NOTHING` + re-Take. File: `internal/infra/persistence/job_repo_gorm.go`. G2 race window closed.
- **T3.14 D1** (#178): 27 × `messaging.WithMetadata(..., "")` → `WithMetadata(..., c.Get("Idempotency-Key"))` trong 8 handler files. `MasterSwap.SwapAsync` thêm arg `idempotencyKey`; rehydrate path skip goroutine khi `j.Status != Pending`.
- Build `go build ./...` PASS; `go test ./internal/...` 5/5 OK.
- **Smoke matrix (4 POSTs same key)**:
  - POST 1 fresh K1 → 202 J1, `cdc_jobs.idempotency_key=K1` ✓ T3.14 wire
  - POST 2/3 K1 cache hit → 200 X-Idempotent-Replay=true (middleware OK)
  - POST 4 K1 cache flushed → **202 J1 SAME id** (no replay header) → handler ran, JobRepo ON CONFLICT rehydrate ✓ T3.13
  - cdc_jobs `count=1, distinct_ids=1` → no race dupe
- Tasks #176 + #178 → completed. PENDING commit (untracked `internal/infra/`, `internal/app/`, `internal/domain/` từ trước; cần Boss quyết định scope commit atomic vs incremental).

## 2026-05-06 16:30 ICT — Đợt 2 P3 (#179): 4 handler trung → bus.Execute ✓
- **wizard.execute** — `WizardExecuteCommand` sync; replaces direct `repo.Update + AppendProgress` ở `wizard_handler.go:Execute`. Errors: `ErrWizardNotFound` (404) / `ErrWizardAlreadyRunning` (409).
- **master.approve** — `ApproveMasterCommand` sync (UPDATE + NATS `cdc.cmd.master-create` publish inline) replace direct `h.db.Exec` + `h.nats.Conn.Publish` ở `master_registry_handler.go:Approve`. 1 audit row / approve. Errors: `ErrMasterNotFound`/`Ambiguous`/`NotApprovable`.
- **source.update-v2** — `UpdateSourceObjectV2Command` sync (registry + shadow_binding 2-table write) replace inline GORM ở `source_object_actions_handler.go:UpdateV2`. Validation `validTimestampField` inlined để không kéo `api` import vào commands package.
- **alert.silence** — `SilenceAlertCommand` sync; replace direct `h.am.Silence` ở `alerts_handler.go:Silence`. Parity với existing `alert.ack` pattern.
- Server.go: 4 RegisterSync mới (alert.silence, master.approve, wizard.execute, source.update-v2). Build + `go test ./internal/...` 4/4 OK.
- Smoke 4 POST với entity không tồn tại → 4 cdc_jobs rows status=failed, idempotency_key đầy đủ. Replay master.approve cùng key → 202 same id, count=1 (Đợt 1 atomic upsert hoạt động xuyên Đợt 2 commands).
- Task #179 → completed. PENDING commit.

## 2026-05-06 16:51 ICT — Đợt 3 P3 (#180): 4 handler khó → bus ✓
- **schedule.update** — `UpdateScheduleCommand` sync (`schedule_handler.go:Update`). Replace First+Updates+ActivityLog. Errors: `ErrScheduleNotFound` (404) / `ErrScheduleNoFields` (400). API giữ getResponseByID post-bus để response shape giữ nguyên.
- **registry.update** — `UpdateRegistryCommand` sync (`registry_handler.go:Update`). Lớn nhất Đợt 3: TableRegistry Updates + cascading mapping_rule auto-approve (inactive→active) + 2× PublishReload + 2 ActivityLog rows, tất cả atomic trong handler. v2sync.SyncFromLegacy giữ post-bus ở API (read service, không destructive).
- **transmute.run** — `TransmuteRunCommand` async (`transmute_schedule_handler.go:RunNow`). Reuse subject `cdc.cmd.transmute` đã đăng ký Đợt 1; wire payload (master_table/triggered_by/correlation_id) byte-identical với legacy publish nên TransmuteHandler worker không phải sửa.
- **recon.failed-log-mark-retrying** — `MarkFailedLogRetryingCommand` sync (`reconciliation_handler.go:RetryFailedLog`). Sibling write sau RetryFailedCommand Dispatch. Idempotency-Key suffix `:mark` để tránh va UNIQUE constraint với row Dispatch chính.
- Server.go: 3 RegisterSync mới (schedule.update, recon.failed-log-mark-retrying, registry.update). Build + `go test ./internal/app/commands/...` PASS.
- Smoke 4 endpoint live (CMS d3 :8083): 200/202/202/202. cdc_jobs có 5 type: schedule.update success, registry.update success, transmute.run pending (worker async), recon.retry-failed pending, recon.failed-log-mark-retrying success. 5 idempotency_key distinct nhờ `:mark` suffix.
- Task #180 → completed. PENDING commit.

## 2026-05-06 17:11 ICT — Đợt 4 P3 (#181): 4 handler last-mile → bus ✓
- **schedule.create** — `CreateTransmuteScheduleCommand` sync (UPSERT). Validation cron-expr/mode/master_table giữ ở API; handler chạy SQL UPSERT atomic với cdc_jobs audit.
- **schedule.toggle** — `ToggleTransmuteScheduleCommand` sync. `ErrTransmuteScheduleNotFound` → 404 mapping qua errors.Is.
- **registry.register** — `RegisterRegistryCommand` sync (lớn nhất Đợt 4): atomic INSERT TableRegistry + EnsureShadowTable DDL + rollback `db.Delete` nếu DDL fail + PublishReload + ActivityLog. Dùng `ShadowTableEnsurer` interface trong commands package — commands KHÔNG import service trực tiếp, ShadowAutomator pointer match interface implicit. CreateDefaultColumnsCommand dispatch + v2sync giữ post-bus ở API (idempotent, không destructive-essential).
- **registry.bulk-register** — `BulkRegisterRegistryCommand` sync. BulkCreate + PublishReload + ActivityLog, atomic. Per-entry CreateDefaultColumns dispatch ở API loop với Idempotency-Key suffix `:cdc:<entry_id>` (mở rộng pattern `:mark` Đợt 3 thành `:<role>:<key>` cho N siblings).
- Server.go: 4 RegisterSync mới (reuse `shadowAutomator`/`db`/`natsClient` đã khởi tạo). Build + `go test ./internal/app/commands/... ./internal/api/...` PASS.
- Smoke 4 endpoint live (CMS d4 :8083): cdc_jobs 4 row distinct idempotency_key, type/status đầy đủ. Errors propagate đúng qua bus → API: schedule.toggle 404 (not_found), registry.register/bulk 500 (constraint violation pre-existing — bus + handler chạy đúng).
- Task #181 → completed. PENDING commit.

## 2026-05-06 17:30 ICT — Đợt 5 P3 (#182): 4 handler last-batch → bus ✓
- **master.toggle-active** — `ToggleMasterActiveCommand` sync (UPDATE master_binding flip is_active). Errors: `ErrMasterBindingNotFound` (404) / `ErrMasterRequiresApproved` (409 — CHECK `v2_master_active_requires_approved` trap).
- **worker-schedule.create** — `CreateWorkerScheduleCommand` sync (INSERT WorkerSchedule). Type tag `worker-schedule.*` namespaced né Đợt 4 `schedule.*` clash trên CommandBus registry; ResultBody trả `{id, created}` cho post-bus getResponseByID giữ shape.
- **schema-proposal.reject** — `RejectSchemaProposalCommand` sync với CAS guard `WHERE id=? AND status='pending'`. RowsAffected=0 → `ErrSchemaProposalNotPendingOrNotFound` → 409, replay-safe.
- **mapping.batch-update-status** — API loop dùng `UpdateMappingRuleCommand` (Đợt 1) per-rule; Idempotency-Key suffix `:rule:<id>` né UNIQUE collision khi N rule chia 1 request key (mở rộng pattern `:<role>:<key>` từ Đợt 3/4).
- Server.go: ctor `NewSchemaProposalHandler(db, bus, logger)` + 3 RegisterSync mới (master.toggle-active, worker-schedule.create, schema-proposal.reject). Build + `go test ./internal/app/commands/... ./internal/api/...` PASS.
- Smoke 4 endpoint live (CMS d5 :8083): master.toggle-active 404 (pre-bus resolveByName), worker-schedule.create 201 ID 12 (cdc_jobs success), schema-proposal.reject 409 (cdc_jobs failed, errors.Is mapping đúng), mapping batch 202 updated:0 (loop empty). cdc_jobs 2 row distinct idempotency_key.
- Task #182 → completed. cms commit `4589c55`. PENDING agent commit.

## 2026-05-06 17:50 ICT — Đợt 6 P3 (#183): schema-proposal.approve → bus ✓
- **schema-proposal.approve** — `ApproveSchemaProposalCommand` sync. To nhất P3 (tx multi-step: ALTER TABLE shadow/master + INSERT cdc_mapping_rules + UPDATE schema_proposal). API thin: parse + Reason ≥10 + bus.Execute + errors.Is mapping. Failure-mark UPDATE giữ trong handler (out-of-tx) → atomic close-loop.
- 5 sentinel: NotFound (404), NotPending (409), InvalidDataType (400), InvalidIdent (400), ApplyFailed (500 wraps tx err qua errors.Join). Regex `propTypeRe`/`propColumnRe` duplicated sang commands package (pattern Đợt 4); api package xoá 2 var dead code, giữ `propIdentRe` cho mapping_preview_handler.
- Server.go: 1 RegisterSync mới (schema-proposal.approve). Build + `go test ./internal/app/commands/... ./internal/api/...` PASS.
- Smoke 3 case (CMS d6 :8083): non-existent ID 9999999 → 404, pending shadow proposal id=1 (orders/smoke_d6_col TEXT) → 200 (ALTER `shadow_goopay_source.orders` applied, schema_proposal.status='approved', applied_at NOT NULL), replay → 409 not_pending. cdc_jobs 3 row distinct idempotency_key, error_message đúng sentinel name (1 success + 2 failed).
- Task #183 → completed. cms commit `f562b96`. PENDING agent commit.
- **P3 destructive migration coverage update**: 21 handler / 21 endpoint đã qua bus (5 đợt × 4 + Đợt 6 × 1). Còn lại system_connectors.Create/Delete (HTTP→Kafka Connect REST, không phải DB destructive) — design riêng nếu Boss cần audit.

## 2026-05-06 18:10 ICT — Đợt 7 P3 (#184): system_connectors 6 endpoint → bus ✓
- **system-connector.create** — `CreateSystemConnectorCommand` sync (HTTP Connect Create + best-effort `SourceFingerprintRepo.Upsert`; fingerprint pre-built ở API qua `parseFingerprint`).
- **system-connector.delete** — `DeleteSystemConnectorCommand` sync (HTTP Delete + best-effort `MarkDeleted`).
- **system-connector.lifecycle** — `LifecycleSystemConnectorCommand` sync (op param: `restart` | `restart-task` | `pause` | `resume`) — 1 command bao 4 endpoint, helper `dispatchLifecycle` API tránh duplicate.
- 2 narrow interface trong commands package: `KafkaConnectorWriter` (5-method) + `SourceFingerprintRepo` (2-method). `*infrahttp.KafkaConnectClient` + `*repository.SourceRepo` satisfy implicit qua structural typing — commands KHÔNG import infra/http hay repository.
- Server.go: ctor `NewSystemConnectorsHandler(client, sourceRepo, bus, logger, listQ, getQ, pluginsQ)` bổ sung bus + 3 RegisterSync mới (`system-connector.create/delete/lifecycle`). Build + `go test ./internal/app/commands/... ./internal/api/...` PASS.
- Smoke 5 negative case (CMS d7 :8083, Kafka Connect localhost:18083): restart/delete/pause/restart-task non-existent + create bad class → tất cả 502 với err detail nguyên văn từ Connect HTTP 404/500. cdc_jobs 5 row distinct idempotency_key, status=failed, `error_message` capture full diagnostic (`kafka connect HTTP 404: Unknown connector...` / `HTTP 500: Failed to find any class...`).
- Task #184 → completed. cms commit `2f5040b`. PENDING agent commit.
- **P3 destructive migration coverage final**: 27 endpoint / 24 handler đã qua bus (Đợt 1×4 + 2×4 + 3×4 + 4×4 + 5×4 + 6×1 + 7×6). System connectors là layer cuối; còn lại 4 ActivityLog write (3 reconciliation_handler + 1 registry_handler) là audit side-effect, không phải destructive — không cần bus.

## 2026-05-06 23:30 ICT — Phase 4 D2 cosmetic: /api/v1 prefix unify ✓
- Backwards-compat shim — `internal/middleware/deprecation.go` (NEW): `CanonicalAPIRoute` folds /api/v1 → /api legacy + `DeprecateLegacyAPIPath` stamps RFC 8594 Sunset (2026-12-31 GMT) + `Deprecation: true` + `Link rel="successor-version"` chỉ trên hit /api/* không phải /api/v1/*.
- Idempotency MW (`idempotency.go`): gọi `CanonicalAPIRoute` trước build Redis cache key → legacy + canonical client share cùng namespace, không fork replay cache.
- Audit MW (`audit.go::actionFor`): strip /v1 trước lookup `ActionMap` → 7 hardcoded route key giữ single-namespace, không cần duplicate.
- Router (`router.go`): deprecation MW mount app-level (cover /api/system/health ngoài apiGroup) + 26 route dual-mount (17 `dualGet` + 6 `dualPost` + 3 `dualPatch`) + `registerDestructive`/`registerDestructiveRestart` dual-mount nội bộ + `/api/v1/system/health` alias.
- Swagger: 8 `@Router` annotation `/api/X` → `/api/v1/X` ở `reconciliation_handler.go` (cosmetic doc, runtime unaffected).
- Verify: `go build ./...` PASS, `go test ./internal/middleware/...` PASS (0.78s), live smoke 3 case CMS d2 :8094 — `/api/system/health` legacy nhận `Deprecation: true` + `Sunset: Tue, 31 Dec 2026 23:59:59 GMT` + `Link: </api/v1/system/health>; rel="successor-version"`; `/api/v1/system/health` canonical sạch headers; `/health` root không bị stamp. Cùng handler trả cùng body — dual-mount đúng.
- cms commit `64af966`. PENDING agent commit. **Phase 4 D2 close**.

## 2026-05-06 23:42 ICT — T16 P6: V2 sync atomicity (db.Transaction wrap) ✓
- `SourceObjectV2SyncService` (`internal/service/source_object_v2_sync.go`) — `SyncFromLegacy` cũ chạy 2 INSERT độc lập (source_object_registry → shadow_binding) ngoài tx; failure giữa 2 bước để lại orphan source_object phá recon kỳ tới.
- Tách thân logic ra `SyncFromLegacyTx(ctx, tx, entry)`. Wrapper public `SyncFromLegacy` gọi `s.db.Transaction(...)`. Resolve helpers (`resolveSourceConnectionID`/`resolveShadowConnectionID`) nhận `*gorm.DB` param. Fail-fast `nil tx` lộ wiring bug.
- Callers (`registry_handler.go::Register`/`Update`/`BulkRegister`) giữ nguyên — atomicity nội bộ. `SyncFromLegacyTx` exposed sẵn cho future caller (RegisterRegistry handler) fold V1+V2 vào outer tx; comment `register_registry.go:24-28` document V2 sync intentionally post-bus (idempotent, no destructive op) → tx propagation qua bus không cần ngay.
- Test guards (`source_object_v2_sync_test.go`): 3 case — nil entry (cả entrypoint), nil tx fail-fast với message rõ. Rollback semantic verify ở deploy-time E2E (project convention — sqlmock không có deps).
- Verify: `go build ./...` PASS, `go test ./... -count=1` PASS (api/commands/queries/messaging/middleware/service all green, 0 regression).
- cms commit `4a2a6e7`. **T16 close** (P6 done; P5 T15 health probe split + P7 T17 test uplift còn pending).

## 2026-05-06 23:53 ICT — T15 P5: Health collector probe split ✓
- `system_health_collector.go` 781 → 271 dòng (≤300 budget). 7 probe tách thành 7 file dưới `internal/service/health/probes/` (worker, kafka_connect, debezium, kafka_lag, nats, postgres, redis) — mỗi probe plain func nhận `HTTPDeps{Client, ProbeTimeout}` + URL args, không kéo Collector struct.
- Helpers `httpGet`/`sanitizeErr`/`isSchemeByte` chuyển vào probes (`HTTPDeps.Get` + `probes.SanitizeErr`). DB queries → `system_health_queries.go` (94 dòng); FE alert/overall compute → `system_health_compute.go` (102 dòng) — cùng `service` package giữ Snapshot + Status* constants native, không duplicate.
- Probe parallel qua errgroup giữ nguyên (đã tồn tại từ trước). Cấu trúc tách cô lập domain (HTTP / DB / compute) — runtime semantic không đổi, wire format byte-identical.
- Test: `system_health_collector_test.go` cập nhật `sanitizeErr(...)` → `probes.SanitizeErr(...)` (1 import + 1 call). 5 existing test all PASS.
- Verify: `go build ./...` + `go vet ./...` + `go test ./...` PASS. Live `/api/system/health` smoke skip — refactor pure structural, JSON contract test (`TestSnapshotJSONStability`) đã cover wire format.
- cms commit `477ba19`. **T15 close**. Còn pending: P7 T17 test uplift.


