# Active Plans Registry

> **Maintained by**: Brain (Antigravity)
> **Last Updated**: 2026-06-02
> **Purpose**: Registry để Brain biết workspace nào đang active → load đúng context khi bắt đầu phiên mới. KHÔNG phải cơ chế agent communication.

| Workspace | Project | Status | Last Active |
|-----------|---------|--------|-------------|
| upgrade-core-system | Upgrade Core Brain/Muscle System | ✅ Done | 2026-02-25 |
| feature-refactor-2026 | GooPay Core Refactor 2026 | ✅ Done — sẵn sàng tiếp tục | 2026-02-25 |
| optimize-brain-muscle-models | Tối ưu hóa model cho Brain/Muscle | ✅ Done (V2 Quota & Multi-Muscle) | 2026-02-25 |
| compare-disbursement-export | So sánh logic DisbursementTicketExport | ⏸ Paused | 2026-02-25 |
| compare-disbursement-trans-his-export | So sánh logic DisbursementTransHisExport | ✅ Done | 2026-02-27 |
| feature-merchant-export-activation-info | Bổ sung thông tin kích hoạt Merchant Export | ✅ Done | 2026-03-12 |
| feature-id-expired-notification-log-export | Tạo IDExpiredNotificationLogExport type | ✅ Done | 2026-02-27 |
| feature-fee-configuration | Cấu hình phí dịch vụ (Fee Configuration) | ⏸ Paused | 2026-03-03 |
| feature-cdc-integration | CDC Integration (Debezium-only sau commit 8ef7d71 remove airbyte) — Phase F (F1+F3) Done 2026-05-04 | ✅ Done | 2026-05-04 |
| feature-system-refactor-2026-05 | System Refactor 2026-05 — bucket B1+B2 (hygiene + tooling), 4 service local smoke | 🟡 Active | 2026-05-04 |
| feature-export-driver-search | Driver Info & Approximate Search in Exports | ✅ Done | 2026-03-24 |
| upgrade-agent-infrastructure | Nâng cấp hạ tầng Agent v1.10.0 (Brain/Muscle) | ✅ Done | 2026-04-06 |
| feature-trans-his-collection-export | Export TransHis Collection | 🟡 Active | 2026-04-09 |
| feature-multi-pg-isolation-e2e | Tách 4 PG containers (auth/cdc/dest/source) + E2E auto-pipeline | 🟡 Active | 2026-04-28 |
| feature-cdc-system-refactor | Task #19 service-tier drainage — Đợt G/H/I closed by max → Đợt J handed off to x2 (cms-lane locked) | 🟡 Active | 2026-05-07 |
| centralized-data-service-config-audit | Audit DB Connections, GORM configs & Fix Tests | ✅ Done | 2026-05-18 |
| feature-cdc-activity-log-metrics | Sửa đổi RowsAffected & DB Materialization Metrics | 🟡 Active | 2026-05-21 |
| bug-schema-drift-loop-2026-05-25 | Schema drift loop + SLOW SQL in event handler | ✅ Done | 2026-05-26 |
| bug-partition-dropper-relation-missing-2026-05-26 | Postgres relation missing in partition dropper | ✅ Done | 2026-05-26 |
| bug-snapshot-v2-host-uri-2026-05-21 | LWW Guard cho Dual-Stream Consistency | ✅ Done | 2026-05-25 |
| bug-kafka-consumer-empty-topics-2026-05-26 | Kafka Consumer empty topics & CMS schema qualifier | ✅ Done | 2026-05-26 |
| bug-cms-slow-sql-probes-2026-05-26 | Slow SQL in probes & system health checks | ✅ Done | 2026-05-26 |
| feature-sensitive-masking-opt-out-2026-06-02 | Sensitive Masking Opt-Out & UI Controls | ✅ Done | 2026-06-02 |
| bug-snapshot-cache-latency-binding-66 | Giải quyết stale cache masking khi snapshot v2 | ✅ Done | 2026-06-02 |
| feature-masters-page-audit-2026-06-02 | Masters Page Audit & Enhancements | ✅ Done | 2026-06-03 |
| feature-sync-shadow-master-bindings-2026-06-04 | Syncing Shadow And Master Bindings | ✅ Done | 2026-06-04 |
| feature-table-registry-ui-enhancement-2026-06-09 | Table Registry UI Enhancement | ✅ Done | 2026-06-09 |




---

## Notes
- **Active** 🟡: Đang làm trong phiên hiện tại hoặc phiên gần nhất
- **Paused** ⏸: Tạm dừng, sẽ tiếp tục sau
- **Done** ✅: Hoàn thành, archived
- Khi bắt đầu phiên mới: Brain đọc bảng này → load workspace có status Active đầu tiên
|-----------|---------|--------|-------------|

## 2026-04-27 Updates

- `feature-cms-fe-overhaul`: đã hoàn thành Phase 2 FE Navigation Refactor trong `cdc-cms-web`; bước kế tiếp là refactor data model/API usage của các page sang semantics V2.
- `feature-cms-fe-overhaul`: đã hoàn thành Phase 8 `worker-schedule` contract refactor; `cms-fe` hiện được chốt theo mô hình 2 luồng, trong đó operator-flow (monitoring / backup / retry / reconcile) phải được giữ lại nhưng làm gọn API surface.

## 2026-05-07 Updates

- `feature-cdc-system-refactor`: Task #19 service-tier drainage — Đợt G+H+I closed (cms `b4a3461`, agent `c141012`). Cluster A (alert_manager + approval_service + provisioning_orchestrator + state_machine + master_swap + shadow_automator + source_object_v2_sync + registry_repo + mapping_rule_repo) → `infra/persistence/`. Đợt J (cluster A* system_health_* + cluster C probes/) → `infra/observability/{,probes/}` đã plan + tasks (agent `dd21443`), x2 thi công.
- Lane lock effective from `b4a3461`: max owns worker (`centralized-data-service/`) + workspace docs; x2 owns cms (`cdc-cms-service/`).
- Pending Task #19 closure: x2 lands Đợt J commit + APPEND `05_progress.md` → Task #19 marked CLOSED.
- Worker-lane sub-issues queued for max post-Đợt-J: P3 prune residue 1 row legacy_1, duplicate close-loop log dedup, Track E (Mongo CDC) plan kickoff.

## 2026-05-07 09:48 ICT Updates (post-Đợt-J)

- `feature-cdc-system-refactor`: **Task #19 CLOSED** ✅ — Đợt J landed by x2 (cms `b453d36`, agent `57d1b2a`). 10 đợt A→J done; `internal/service/` removed entirely; cms hexagonal-aligned (`app/{commands,queries,ports}` + `domain/` + `infra/{cache,http,messaging,persistence,observability}` + `api/` + `server/` + `middleware/` + `router/` + `model/`). Q3 (rebuild + restart cms-server) ALSO done by x2: PID 52079 `/tmp/cdc-cms-service-postJ` post-`b453d36` binary, smoke test 3 endpoint PASS (200 trên `/health`, `/api/system/health`, `/api/v1/source-objects/registry/1/dispatch-status`).
- Workspace `feature-cdc-system-refactor`: status row trong table có thể đọc là 🟡 Active (audit trail) — actual phase: ⏸ Paused (Task #19 đóng, chờ next directive).
- Next priorities cho max-Brain (post-Task-#19): plan Track E (Mongo CDC, blocked on Boss brief — lesson L-1436 không bịa scope), P3 prune residue 1 row legacy_1 investigation (worker-lane), duplicate close-loop log dedup investigation (worker-lane).
- max output 2026-05-07 ICT: `report_flow1_connect_source_2026-05-07.md` (overview Wizard step 1–5 manual operator flow per Boss directive).

## 2026-05-07 16:30+ ICT Updates (Flow 1 push iter#46–#72)

- **Workspace re-active 🟡**: `feature-cdc-system-refactor` từ Paused → Active vì Boss directive "**bằng mọi giá phải lên đc flow1 này**" (multi-iter /loop session).
- **Path A → Path B migration LIVE**: cms binary `/tmp/cdc-cms-service-flow1` PID 43919 (mtime 11:21, swap 13:54) chạy commit `0eddad0 feat(cms): support hybrid shadow db configuration (A3)`. Log evidence: `PostgreSQL (shadow data plane) connected port=5436 cdc_shadow`. Path B (5436 cdc_shadow) có 7 schemas + 11 shadow tables LIVE.
- **G-11 master/shadow hyphen**: ✅ CLOSED (iter#68 evidence). master_binding id=37 + shadow_binding id=62 cho src 44 (`src_mongodb_payment_bill_service_refund_requests`) đều có underscore. SQL backfill skip.
- **G-12 (NEW) worker A3 hybrid**: ⏳ OPEN — `/tmp/cdc-worker-host` PID 90006 vẫn chạy binary May 5 09:39 (pre-A3-hybrid). Working tree đã có shadowDB *gorm.DB* + adapter swap nhưng UNCOMMITTED + UNBUILT. Worker query Path A `cdc_dw` thay vì Path B → ERROR `column "_id" does not exist` + `relation does not exist`. Cần verb `commit a3-worker` → `ship g11`.
- **G-13 (NEW) Mongo PK cast bigint**: ⏳ OPEN (defer post-Flow-1). Transmuter hardcoded `"_id"::bigint` không cast được Mongo ObjectId. Recommend: dùng PG source cho Flow 1 P1 happy-path để né.
- **Scope divergence x2**: x2 (Antigravity:gemini-1.5-pro) đã pivot Phase 2 P3 (CQRS refactor + FE polling) — `08_tasks_phase2_p3.md` + `09_tasks_solution_phase2_p3.md` + master_registry_handler thin-adapter. Phase 2 P3 KHÔNG unblock Flow 1; recommend Boss verb `defer phase2, focus flow1`.
- **Brain output post-iter#46**: `report_flow1_loop_iter46_2026-05-07.md` + `report_flow1_loop_iter68_2026-05-07.md` + (đang) `report_flow1_loop_iter72_decision_pack.md`. 05_progress.md đã APPEND iter#46 + iter#47 + iter#68 (215487 bytes / 2700+ lines).
- **Halt status**: Brain đã hoàn thành scope plan + document + coordinate. Block còn lại đều cần Boss explicit verb (CLAUDE.md §1, §11, §12 + Auto Mode rule #5). Verb dictionary: `commit a3-worker` | `ship g11` | `smoke flow1 pg` | `defer phase2, focus flow1` | (or) `max switch muscle` để Brain cross-line execute.

## 2026-05-08 Audit Update

- `audit-flow1-3repos-2026-05-08`: workspace audit mới để re-check `cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`, và `flow1` trên CMS.
- Kết quả chính:
  - `cdc-cms-service` test PASS.
  - `cdc-cms-web` build FAIL ở `src/pages/flow1/*`.
  - `centralized-data-service` binary build PASS nhưng test FAIL (`scratch/`, `internal/handler`, `internal/service`).
  - `flow1` NOT READY do FE build lỗi + contract drift FE↔CMS + runtime drift dấu hiệu stale CMS binary.

## 2026-05-18 Updates

- `bug-mongo-url-dynamic-source-2026-05-18`: ✅ Done. Fix `MetadataRegistryService.GetSourceDSN` multi-scheme resolver (literal DSN / env-pointer / build-from-fields / AES last resort). Worker không còn yêu cầu env `MONGODB_URL` cho source mà user add động qua UI cdc-cms. Files: `internal/service/metadata_registry_service.go` + `internal/service/metadata_registry_dsn_test.go`. Build PASS, vet PASS, 6/6 helper tests PASS, full service suite no regression. Report: `report_dynamic_source_dsn_fix_2026-05-18.md`. Lesson global appended.

## 2026-05-21 Updates

- `FixSourceObjectListingDedupe`: ✅ Done. Listing API `GET /api/v1/source-objects` từng trả 6 row sai do JOIN `cdc_table_registry` không scope `source_connection_id` (cross-connection bleed sau migration 054/055/056). Fix: LATERAL + connection-aware predicate trong `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go`. SQL verify 4 rows trên gpay-postgres-cdc/cdc_dw. Build/vet/test PASS. Binary `/tmp/cdc-cms-service-fixdedupe`. Lesson appended `L-listing-join-misses-identity-tier-column`.

## 2026-05-25 Updates

- `feature-connector-default-collections-2026-05-25`: 🟡 **Plan-only Active** (chờ user verb để Muscle thực thi). Phase `default_collections`. User yêu cầu: khi tạo Mongo connector qua UI mà để trống Collections → CDC toàn bộ collections của database. Audit Brain xác nhận runtime đã đúng (FE `compactConfig` drop empty → BE pass-through → Debezium default CDC all). Gap duy nhất = UX hint + list view display. Strategy = Phương án A FE-only (xem ADR-001). Doc set đầy đủ 00..10 + report template tạo theo §7. Brain KHÔNG sửa code (tuân thủ §12 + user directive "đảm bảo ko sửa code rồi hẵng chạy tiếp néh"). Files đề xuất edit (chờ approve): `cdc-cms-web/src/pages/SourceConnectors.tsx` (Form.Item Collections thêm `extra` text) + list view component (render fallback `(All collections)`). Verb chờ: `execute` / `revise` / `defer`.

## 2026-05-26 Updates

- `feature-connector-default-collections-2026-05-25`: ⏭️ **SUPERSEDED** by `feature-connector-check-connection-2026-05-26` (xem ADR-009 workspace mới). Lý do: User propose UX explicit hơn (Check Connection → scan → multi-select default all) thay vì FE-only hint. Doc cũ giữ nguyên làm lịch sử quyết định.
- `feature-connector-check-connection-2026-05-26`: 🟡 **Plan-only Active** (chờ user verb để Muscle thực thi). Phase `check_connection`. User propose flow: nhập MongoDB Connection URL + Database → click "Check Connection" → progress bar → BE scan collections → render multi-select default all → uncheck nếu không muốn CDC → nút Create chỉ hiện khi check PASS. Pattern extensible cho MySQL/PG sau (defer ADR-001). Audit Brain xác nhận ~70% infra đã có (worker `DiscoverDatabases/Collections` accept full URI, BE endpoint `/api/introspection/mongo/...` exists, NATS subjects registered). Gap: (1) worker handler đang build `mongodb://host:port` inline → drop auth (Edit #1, #2); (2) FE chưa có hook + Collections là `<Input>` text (Edit #5-14). Strategy = EXTEND không refactor: ADD POST body endpoint `{uri,database}` (ADR-002, tránh leak vào GET query), giữ GET legacy (R12). Effort ~6h30m. Doc set 11 file đầy đủ (`00_context` → `report`). ADR-001..009 (Mongo P0, POST body, 5-case VN map, default all selected, Spin không Progress fake, gate Create, defer cross-DB interface, no wizard session, supersede workspace cũ). Brain KHÔNG sửa code (§12). Verb chờ: `execute` / `revise` / `defer` / `model: opus|sonnet`.

## 2026-05-26 — bug-first-snapshot-no-write
- **Status**: Done
- **Workspace**: `agent/memory/workspaces/bug-first-snapshot-no-write-2026-05-26`
- **Issue**: snapshot.v2 lần đầu báo success với rows>0 nhưng shadow 0 row.
- **Root cause**: chuỗi 4 lớp — cache stale → silent-skip → caller discard written → metric đếm len(batch) thay vì written.
- **Fix**: pre-flight ReloadAll + hard-assert route + Warn log + dùng written return + rowsTotal += batchWritten.
- **Files**: centralized-data-service/internal/handler/{event_handler.go, snapshot_runner_handler.go}.
- **Verify**: go build + go vet + go test handler all PASS.
- **Lesson**: L-CDC-route-empty-silent-skip-2026-05-26 (lessons.md).

## 2026-05-26 — bug-reconcile-mongodb-not-configured
- **Status**: Done
- **Workspace**: `agent/memory/workspaces/bug-reconcile-mongodb-not-configured-2026-05-26`
- **Issue**: Scheduler ghi `reconcile ALL skipped — scheduler reconCore not initialized (MongoDB not configured)` mỗi 60s. Feature reconcile dead trên V2 deployment.
- **Root cause**: 2 lớp chồng — (L1) `worker_server.go` gate cả init ReconCore bằng legacy `cfg.MongoDB.URL`; (L2) `MetadataRegistryService.ReloadAll` synthesize V2 `TableRegistry` mà bỏ trống `SourceURL` field dù connection_registry đã đủ thông tin.
- **Fix (2 layers + defense in depth)**:
  - L2: ReloadAll build `connectionURIByCode` qua helper `resolveSourceURIFromConn` → pass `sourceURI` vào `synthesizeLegacyTableRegistry` → `cfg.SourceURL` populate per-source.
  - L1: Bỏ guard `cfg.MongoDB.URL` quanh ReconCore init; `mongoClientShared` vẫn gate (legacy default); reconHandler luôn register 5 NATS subjects với nil-check ở handler-side; runReconcileCycle log Warn → Error "wiring regression".
  - Defense: `ReconSourceAgent.getClient` hard-assert khi `sourceURL=="" && defaultClient==nil` → error operator-facing rõ.
- **Files**: 
  - `centralized-data-service/internal/service/metadata_registry_service.go`
  - `centralized-data-service/internal/service/metadata_registry_service_test.go`
  - `centralized-data-service/internal/server/worker_server.go`
  - `centralized-data-service/internal/service/recon_source_agent.go`
- **Verify**: `go build ./...` EXIT=0, `go vet ./internal/...` EXIT=0, `go test ./internal/service/ -count=1` PASS 0.525s, `go test ./internal/handler/ -count=1` PASS 3.771s. Limitation: chưa có Mongo/NATS/Worker live để runtime smoke — manual smoke checklist trong report.
- **Lesson**: `L-2026-05-26-legacy-config-gate-kills-feature` (lessons.md) — Global pattern: legacy single-config gate phủ lên feature-construction → V2 deployment chết âm thầm. Đúng: tách C1-default-client init khỏi F-init, F luôn init, populate per-source identity từ registry, hard-assert defense.
- **Open follow-up**: refactor ReconHealer/BackfillSourceTsService/TimestampDetector/FullCountAggregator sang per-source URIs (defer scope). Cân nhắc deprecate `cfg.MongoDB` block. Prometheus metric `recon_default_client_missing_total` cảnh báo legacy fallback path.
- **Report**: `report_reconcile_mongodb_not_configured_2026-05-26.md`

## 2026-05-26 — audit-cdc-qa-process
- **Status**: Done (audit-only, không sửa code per user directive + §12 GEMINI)
- **Workspace**: `agent/memory/workspaces/audit-cdc-qa-process-2026-05-26`
- **Trigger**: User audit 5 nhóm × 16 tiêu chí QA cho CDC pipeline (Functional/Stability/Performance/Resource/Metric).
- **Method**: 2 Explore subagent thorough quét codebase parallel → matrix rating L0..L4 với file:line evidence → gap analysis P0/P1/P2 với code demo.
- **Result**: Composite score **35/64 ≈ 54.7%** (L4=1, L3=6, L2=4, L1=5, L0=0). 16 gap → 4 P0 blocker + 5 P1 release + 7 P2 backlog. Effort ước tính 24h Muscle work để đóng P0+P1 (→ ~80%).
- **Điểm sáng**: Data Reconciliation L4 (3-tier hash + golden test cross-store + 5 metric), DLQ L3 (write-before-publish + state machine + PII mask + semantic contract test), OTel L3 (3-signal + severity sampling), Concurrency L3 (circuit breaker per source/dest + fencing).
- **Điểm yếu P0**:
  1. `cdc_kafka_consumer_lag` metric DEAD (định nghĩa không có `.Set()` call → alert dead).
  2. OTel Collector exporter chỉ `debug` stdout (traces production không persist).
  3. Prometheus production thiếu scrape `cdc-worker:9090` + `kafka-exporter:9308`.
  4. Pipeline-level DLQ circuit breaker thiếu (vi phạm L-CDC-circuit-breaker-2026-05-22).
- **Files audit**: 8 file workspace (00_context, 01_requirements, 02_plan, 05_progress, 06_validation matrix, 07_status_report, 10_gap_analysis, report).
- **Lesson new**: `L-2026-05-26-metric-defined-but-never-set` — Global pattern "metric/alert dead vì define-only, không có call-site → silent monitor failure + false confidence". Đúng flow: definition+call-site coupling, smoke test metric value ≠ 0, synthetic alert test, definition near use, cross-ref grep CI block.
- **Verb chờ**: User quyết định prioritize fix gap P0 (4 item ~6.5h) trước go-live hay defer.
- **Report**: `report_cdc_qa_process_audit_2026-05-26.md`

## 2026-05-27 — plan-cdc-qa-gap-fix
- **Status**: PLAN READY (Brain plan-only, chưa thực thi)
- **Workspace**: `agent/memory/workspaces/plan-cdc-qa-gap-fix-2026-05-27`
- **Trigger**: User yêu cầu "lên plan để vá các gap trong audit này. đảm bảo hệ thống đáp ứng đc & trên cms-fe có nơi để admin audit." dựa trên audit `audit-cdc-qa-process-2026-05-26` (35/64 ≈ 54.7%).
- **Method**: Brain plan-only theo §1 + §12. Spawn 2 Explore subagent parallel (cdc-cms-web + cdc-cms-service backend) để khảo sát struct cho UI. 4-phase sequencing với dependency diagram + per-gap code demo (Go/TS/SQL/YAML/Bash).
- **Result**: Plan toàn diện 4 phase tổng **58.5h**:
  - **P0 (6.5h)**: G-1 ConsumerLag Set, G-2 OTel exporter, G-3 Prom scrape+alert, G-4 DLQ Circuit Breaker → score +9 → 44/64 (68.75%).
  - **P1 (20h)**: G-5 failover smoke, G-6 WAL alert, G-7 pprof+goleak, G-8 ordering test, G-9 drift E2E testcontainers → +7 → 51/64 (79.7%).
  - **P2 (16h)**: G-10..G-16 (tier3 config, batches counter, adaptive batch, per-source pool, 4 runbook, chaos network, k6 load) → +5 → 56/64 (87.5%).
  - **UI (16h)**: BE 3 endpoint admin-only (`/audit/qa-summary`, `/gaps`, `/metric-health`) + migration 0060 seed 16 gap + FE AuditPage 4 section (Composite/Metric Cards/Rating Matrix/Gap List) trên cdc-cms-web với React Query polling.
- **ADR**: 7 ADR (scrape interval adaptive, DLQ CB configurable, gap state DB vs YAML, polling vs SSE, admin-only, semaphore vs separate pool, goleak lib).
- **Files**: 21 file workspace theo Full Doc Set §7 (00_context → 10_gap_analysis + 4 file 03_implementation_phase_* + 4 file 08_tasks_phase_* + 4 file 09_tasks_solution_phase_* + report_*).
- **Governance**: ✓ §11 APPEND-only, ✓ §12 Brain Code Prohibition (code demo trong MD), ✓ §14 Pre-flight 21 file vật lý.
- **Verb chờ User**: `execute p0` | `execute p1` | `execute p2` | `execute ui` | `revise` | `defer`.
- **Report**: `report_plan_cdc_qa_gap_fix_2026-05-27.md`

## 2026-05-27 — feature-dashboard-v2-monitoring
- **Status**: 🟡 Plan-only Active (Brain plan-only, chưa thực thi)
- **Workspace**: `agent/memory/workspaces/feature-dashboard-v2-monitoring-2026-05-27`
- **Trigger**: User cung cấp Technical Spec Dashboard V2 (4 critical diagnostic block: Time-to-Live Countdown / Snapshot Mode Module / Unified Timeline Crosshair / In-memory Queue Health + 3-tab dashboard: Snapshot Commander / Streaming Real-time / DLQ & Schema Drift). Yêu cầu lập 2 plan: (1) bổ sung monitoring info backend CDS, (2) hiển thị trên cms-fe (task riêng).
- **Method**: Brain plan-only theo §1 + §12. Cross-ref audit `audit-cdc-qa-process-2026-05-26` → close 6/7 monitoring gap. Doc set 13 file (00..10 + report) + code demo Go ~600 dòng + TSX ~400 dòng. 9 ADR khoá decision (worker in-place, aggregator vị trí cms-service, polling vs WS, trace col, Recharts syncId, smoke gate CI).
- **Result**: Plan tách 2 track parallel:
  - **Backend (16.5h)**: 6 metric mới (ingest_rate, consume_rate, transient_errors, snapshot_active_slots, snapshot_progress, snapshot_throughput, snapshot_eta); wire ConsumerLag.Set (G1); classifier extract (G3); probe `debezium_queue.go` mới (B4); 5 endpoint mới `/api/v1/dashboard/{timeline,snapshot/active,snapshot/:id/prioritize,dlq/recent,drift/recent}` ở cms-service; migration `_otel_trace_id` cho `failed_sync_logs`; smoke gate `make smoke-metrics` block CI.
  - **Frontend (16h)**: `DashboardV2.tsx` 3 tab; `TtcWidget.tsx` Vạch Nguy Hiểm 4 trạng thái + blink; `UnifiedCrosshairChart.tsx` 3 chart synced cursor; `PayloadViewerModal.tsx` JSON viewer + SigNoz link; 5 hook React Query; i18n VN-first; a11y `prefers-reduced-motion`.
- **Files**: 14 file workspace (00_context, 01_requirements_{backend,frontend}, 02_plan_{backend,frontend}, 03_implementation_{backend,frontend}, 04_decisions, 05_progress, 08_tasks_{backend,frontend}, 09_tasks_solution_{backend,frontend}, report).
- **Gap mapping audit 2026-05-26**: G1 (lag dead) ✓ T-BE-03; G2 (transient err counter) ✓ T-BE-04..05; G3 (sinkworker classify) ✓ T-BE-06; G4 (queue probe) ✓ T-BE-09; G5 (snapshot signal) ✓ T-BE-07..08; G7 (DLQ payload UI) ✓ T-BE-18..20 + T-FE-17; G6 (drift trace_id) defer.
- **Governance**: ✓ §11 APPEND-only, ✓ §12 Brain Code Prohibition, ✓ §14 Pre-flight 14 file vật lý ở workspace.
- **Verb chờ User**: `execute backend` | `execute frontend` | `execute both` | `revise <section>` | `defer` | `model: opus|sonnet|haiku`.
- **Report**: `report_dashboard_v2_plan_2026-05-27.md`

## 2026-05-27 — plan-sensitive-masking-fix
- **Status**: PLAN READY (Brain plan-only)
- **Workspace**: `agent/memory/workspaces/plan-sensitive-masking-fix-2026-05-27`
- **Trigger**: User report hệ thống CDC đang hardcode `"***"` vào DB shadow → vi phạm Luật 91/2025/QH15 Điều 13 + NĐ 356/2025/NĐ-CP + VBHN 25/VBHN-NHNN. Chế tài: phạt 3 tỷ VND hoặc 5% doanh thu năm liền kề.
- **Method**: Brain plan-only theo §1+§12. Spawn 1 Explore subagent thorough quét 3 service (centralized-data-service + cdc-cms-service + cdc-cms-web) → evidence file:line cụ thể (M1-M10).
- **Evidence vấn đề**: `masking_service.go:91,133,152-153` hardcode `"***"`; `cdc_mapping_rules` không có `mask_strategy`; masking apply ở write-path phá hủy gốc; không API CRUD config; không audit log; test rời rạc assert `"***"` lock anti-pattern.
- **Result**: Plan 3-phase tổng **40h**:
  - **P0 — Engine (16h)**: M-1 migration ENUM 015_mask_strategy + 2 audit table + seed; M-2 Strategy interface + 4 impl (NONE/DROP/HASH_HMAC/PARTIAL); M-3 HMAC KeyProvider env-based; M-4 MaskingService refactor (bỏ `"***"` khỏi DB path, giữ text_sanitizer); M-5 unit test ≥ 90% với assert `NotEqual("***",...)`.
  - **P1 — Worker Integration (10h)**: M-6 DynamicMapper; M-7 BatchBuffer/Recon/Consumer align; M-8 AuditWriter sample 1% (batch 500 / 5s); M-9 E2E testcontainers postgres.
  - **P2 — Admin + UI + Backfill (14h)**: M-10 cdc-cms-service API GET/PUT `/admin/mapping-rules/:id/mask-config` + audit endpoint; M-11 cdc-cms-web tab Sensitive Masking + StrategySelector + MaskPreview client-side; M-12 backfill script Cobra với dry-run default; M-13 `docs/compliance/sensitive-masking-vn-law.md` law mapping.
- **Strategy → field mapping** (ADR-003): NONE cho non-sensitive; DROP cho password/OTP/PIN/CVV; HASH_HMAC cho CCCD/card_number/account_number; PARTIAL cho phone/email.
- **ADR**: 8 ADR (HMAC vs SHA256 thuần, K8s Secret vs Vault, default seed classification, DDM defer, backfill NULL vs HMAC, audit sample 1%, ENUM TYPE, giữ text_sanitizer).
- **Files**: 18 file workspace Full Doc Set §7 (00..10 + 3 implementation_phase_p0/p1/p2 + 3 tasks_phase + 3 tasks_solution_phase + report_*).
- **Governance**: ✓ §11 APPEND-only, ✓ §12 Brain Code Prohibition (code demo trong MD), ✓ §14 Pre-flight 18 file vật lý.
- **Verb chờ User**: `execute p0` | `execute p1` | `execute p2` | `revise` | `defer`.
- **Report**: `report_sensitive_masking_fix_2026-05-27.md`

## 2026-05-28 — bug-snapshot-progress-mismatch
- **Status**: DONE — Muscle apply complete + verify PASS (2026-05-28). LOC delta thực tế: **+211 NET trên 3 file** (snapshot_runner_handler.go +45, prometheus.go +12, snapshot_runner_handler_test.go +154 NEW). Tests: 4 sub-test guard + 2 static guard = **6/6 PASS**. Runtime smoke defer ops phase.
- **Workspace**: `agent/memory/workspaces/bug-snapshot-progress-mismatch-2026-05-28`
- **Trigger**: User report `SELECT COUNT(*) FROM events = 177,980` nhưng UI Snapshot Monitor `status=done, rows_processed=41,342 (23.23%)`. Regression sau fix `snapshot-zero-records-2026-05-27` (đã fix Flush chain + counter PG RowsAffected, nhưng còn 3 root cause khác).
- **3 Root cause**:
  - **A**: `snapshot_runner_handler.go:553-555` — `if len(batch) < p.BatchSize { break }` break sớm khi Mongo `SecondaryPreferred` trả partial do replication lag.
  - **B**: `snapshot_runner_handler.go:352-357` + `:569` — pause `break` fall-through xuống `markProgressDone` ghi đè status paused→done.
  - **C**: `snapshot_runner_handler.go:712-721` — `markProgressDone` không guard `rowsTotal vs total_rows`.
- **Math chứng minh**: 177,980/5000 = 35 batches + 1 tail; actual 41,342 ≈ 8.27 batches → cursor partial ở batch 8-9; throughput 460 rows/s không phải timeout (threshold 41/s).
- **Fix HOLISTIC (1 PR)**: 5 patch production + 1 metric + 3 test, enforce invariant `status=done IFF rows_processed >= total_rows * 0.99` ở **`markProgressDone` (terminal edge)** — defense-in-depth chống whack-a-mole.
- **Patch list**: S1 DELETE early-exit (-7); S2 pause `break`→`return nil` (+4); S3 markProgressDone guard threshold 0.99 (+20); S4 call site update (+2); S5 capture totalRows local (+2); O1 Prometheus `SnapshotPartialDoneTotal{reason}` (+8); T1 TestMarkDoneGuardsCompleteness (+50); T2 TestPauseDoesNotFallThrough (+30); T3 TestCursorPartialMidStream (+50).
- **LOC delta ước tính**: +26 production / +130 test = **~+156 NET trên 3 file** (`snapshot_runner_handler.go`, `metrics.go`, `snapshot_runner_handler_test.go`).
- **ADR**: 7 ADR (cursor exhaustion via `len==0`, threshold 0.99, pause=terminal, metric path, test strategy mock, invariant edge enforcement, KHÔNG đổi ReadPreference).
- **Files**: 12 file workspace Full Doc Set §7 (00..10 + report).
- **Governance**: ✓ §1 plan-only ✓ §11 APPEND ✓ §12 prohibition ✓ §14 pre-flight 12 file.
- **Verb chờ User**: `execute` | `revise <patch_id>` | `defer`.
- **Report**: `report_bug_snapshot_progress_mismatch_2026-05-28.md`

## 2026-06-01 — audit-cdc-testing-rerun
- **Status**: DONE (re-audit verification only, KHÔNG sửa code per §12)
- **Workspace**: `agent/memory/workspaces/audit-cdc-testing-rerun-2026-06-01`
- **Trigger**: User yêu cầu audit lại sau 2-3 vòng fix theo 5 nhóm × 16 tiêu chí testing CDC. Note user: "không cheat config/db để đạt kết quả".
- **Method**: 3 Explore subagent parallel (P0/P1/P2+NEW) + cross-verify `go build/vet/test -short` thực tế. Mọi rating đi kèm file:line evidence; mọi claim PASS chứng minh bằng exit code thực.
- **Result**: Composite score **50/64 = 78.1%** (vs audit gốc 35/64 = 54.7%, **Δ +15 điểm / +23.4 pp**). Chưa đạt target plan 56/64 (87.5%) — còn cách 6 điểm.
- **Verdict per cluster**: 4/4 P0 FIXED ✅, 2 P1 FIXED + 3 P1 PARTIAL, 4 P2 FIXED + 3 P2 PARTIAL, 2 G-NEW FIXED + 1 G-NEW FAKE-PARTIAL.
- **Critical findings**:
  - ✅ Pre-existing failures resolved: `TestSanitizeMongoDSN` (no test exists) + `internal/handler` kafka-go goleak (PASS).
  - ✅ 3 service Go: build + vet EXIT 0.
  - ❌ **FAKE detected**: `centralized-data-service/pkgs/database/metrics_callback_test.go` KHÔNG TỒN TẠI dù `report_execute_remaining_gaps_2026-05-27.md` claim "2 test PASS".
  - ⚠ 2 FAIL mới ở `cdc-cms-service` (mapping_rule validation message format drift) — KHÔNG thuộc 16 gap nhưng block CI.
  - ⚠ 5 gap PARTIAL: G-5 missing `cdc_kafka_consumer_offset`, G-6 missing WAL auto-resume, G-9 path doc drift, G-15 chaos iptables không portable, G-16 k6 sai target.
- **Gap residual**: 8 item, effort ~15.5h để đạt 87.5% target (xem `10_gap_analysis.md`).
- **LOC tracked** (cumulative 3 phase fix, re-verified hôm nay): production ~+250, test ~+400 (-50 missing FAKE), config ~+150, scripts/docs ~+400. Total ~+1200 LOC.
- **Files**: 8 file workspace Full Doc Set §7 (00_context, 01_requirements, 02_plan, 05_progress, 06_validation, 07_status_report, 10_gap_analysis, report_audit_testing_rerun_2026-06-01.md) = 803 dòng.
- **Governance**: ✓ §3 plan-verify ✓ §7 Full Doc Set ✓ §11 APPEND-only ✓ §12 no source edit ✓ §14 pre-flight 8 file vật lý.
- **Verb chờ User**: `execute residual` (fix 8 gap, 15.5h) | `prioritize fake first` (chỉ G3-RES + G5-RES, 2.25h) | `accept 78%` | `re-audit after fix`.
- **Report**: `report_audit_testing_rerun_2026-06-01.md`

## 2026-06-01 — feature-cdc-cms-hexagonal-refactor (Iteration v2)
- **Status**: PENDING_USER_REVIEW (Brain plan complete, awaiting approve)
- **Workspace**: `agent/memory/workspaces/feature-cdc-cms-hexagonal-refactor-2026-06-01`
- **Iteration**: v2 — supersede v1 `feature-cdc-cms-service-restructure-2026-05-19` (PENDING REVIEW 13 ngày, không tiến) nếu User approve.
- **Trigger**: User yêu cầu refactor cấu trúc `cdc-cms-service` theo hướng hexagonal + clean BC, đầy đủ workspace doc + code demo. Rule áp dụng: đọc lessons trước, core systems, không cheat DB/config, plan rõ ràng + code demo, report files thay đổi + LOC.
- **Method (Brain only)**: 4-phase roadmap thay vì 11 phase của v1. P0 (coverage gate ≥60%) → P1 (port split God Interface 151 LOC → 9 file port-per-aggregate) → P2 (server.go 333 LOC → ≤80 LOC + 5 pure-func file) → P3 (18 commands raw `*gorm.DB` → port hẹp) → P4 OPTIONAL (vertical slice 8 BC + `internal/platform` + `internal/shared` technical-only).
- **Result Brain phase**: 12 file vật lý (00→10 + report) = **3,710 LOC markdown / ~144 KB**. 8 ADR đã chốt, 8 risk có mitigation, 45 task mandatory + 10 optional với DoD per task.
- **Decisions key (link 04_decisions.md)**:
  - ADR-01 Vertical Slice (Phase 4 OPTIONAL)
  - ADR-02 REJECT Shared Kernel cho domain (False Cognate)
  - ADR-03 Pure Function Composition Root (KHÔNG receiver-state, KHÔNG DI framework)
  - ADR-04 Coverage Gate ≥60% MANDATORY trước P1
  - ADR-05 1 commit / 1 command Phase 3 (stop rule: 3 fail liên tiếp escalate)
  - ADR-06 REJECT DI framework (Wire/Fx/Dig)
  - ADR-07 `internal/shared/` CHỈ technical primitive (naming/pg/id), KHÔNG domain enum
  - ADR-08 Wizard BC được phép cross-BC import (Saga exception, linter document)
- **8 Bounded Context** (link 00_context.md §3): source, mapping, master, transform, reconciliation, wizard (saga), system_control, observability.
- **Critical evidence (Gap)**:
  - Gap A: `internal/app/ports/repository.go` 151 LOC / 10 interface (God Interface) 🔴 HIGH
  - Gap B: 18/32 commands import `gorm.io/gorm` (56% bypass hexagonal port) 🔴 HIGH
  - Gap C: `internal/server/server.go` 333 LOC composition root phình 🟡 MED
  - Gap D: Flat layer (API 52 / Cmd 32 / Query 30) 🟡 MED
  - Gap E: `internal/bootstrap/` 0% test coverage 🟠 RISK
  - Gap F: Naming overlap `model/` vs `domain/` 🟡 MED
  - Gap G: Audit logic scatter middleware + persistence + queries 🟢 LOW
- **LOC tracked**: workspace doc 3,710 LOC markdown. **0 LOC source code changed** (Brain §12 compliance). 0 memory global overwrite (chỉ APPEND file này).
- **Files workspace** (12): 00_context, 01_requirements, 02_plan, 03_implementation, 04_decisions, 05_progress, 06_validation, 07_status, 08_tasks, 09_tasks_solution, 10_gap_analysis, report_2026-06-01.
- **Governance**: ✓ §1 Brain không nhúng code ✓ §3 plan-verify (4 gate G0-G3 + G4 optional) ✓ §7 workspace-first, append-only progress, Full Doc Set ✓ §11 zero memory overwrite ✓ §12 zero source edit ✓ §13 ADR cho 8 quyết định khó ✓ §14 pre-flight 12 file vật lý verified.
- **Pending Questions** (6 — link 07_status.md §5): P0 estimate 3-5d OK?, P4 opt-in?, `go-arch-lint` vs `depguard`?, Wizard cross-BC OK?, 1 PR/batch vs 1 PR all?, `internal/platform/observability/` tách?
- **Verb chờ User**: `approve workspace` (start P0) | `request changes <file>` | `prefer v1` (archive v2) | `answer questions` (trả lời 6 Q rồi approve).
- **Report**: `report_2026-06-01.md`

## 2026-06-02 Updates

- `feature-sensitive-masking-opt-out-2026-06-02`: Đã hoàn thành triển khai việc chuyển đổi cấu hình dữ liệu Masking và Mapping từ string-based sang `shadow_binding_id` (`int64`) để cô lập cấu hình theo từng shadow table binding (tránh xung đột trùng tên). Đã refactor legacy RegistryService, DynamicMapper, MaskingService (hỗ trợ dynamic resolution cho int64/string fallback) và các handlers/consumers. Toàn bộ test suite đã pass 100% (exit code 0).

### 2026-06-02 — Workspace `bug-gpay-id-trigger-contract-2026-06-02` (BRAIN PLAN COMPLETE)
- **Path**: `agent/memory/workspaces/bug-gpay-id-trigger-contract-2026-06-02/`
- **Severity**: 🔴 HIGH — prod CDC pipeline block, error `null value in column "_gpay_id"` SQLSTATE 23502 trên `tokens` (và mọi V2 shadow table).
- **Trigger**: User report `flush after batch (enqueued=5000, persisted=784): batch upsert chunk failed`. Bug bóc lộ sau khi fix vòng trước (event_handler PK lookup + batch_buffer remove `effectivePK == "id"` condition) đẩy flow vào đến INSERT layer — chính fix trước ĐÚNG, chỉ bóc lộ bug ẩn từ V2 shadow design v1.25.
- **Root cause (1 dòng)**: Contract Drift 3 lớp — comment Go (`batch_buffer.go:246-250` claim "sonyflake trigger fills"), Go DDL (`schema_manager.go:226` không có DEFAULT), migration SQL (`018_sonyflake_v125_foundation.sql:130` trigger chỉ fencing, KHÔNG fill `_gpay_id`). 0 nơi implement thực sự.
- **Local OK, prod fail**: schema drift — local có thể dùng V1 sequence DEFAULT từ migration 003 hoặc dev patch tay; prod sink lazy `createShadowTable()` lần đầu encounter table → DDL không DEFAULT → INSERT 23502.
- **Solution chốt (ADR-01 + ADR-06)**: Single source = DB-side DEFAULT `cdc_internal.sf_nextval()`. Reject Go-side fill (dư thừa). Migration `019_sonyflake_default_fill.sql` tạo function + idempotent ALTER heal mọi V2 shadow table existing.
- **Patch tối thiểu**: 1 migration (~90 LOC SQL) + 1 dòng Go DDL `schema_manager.go:226` + 1 comment rewrite `batch_buffer.go:246-250` + 1 integration test (~100 LOC testcontainers).
- **8 ADR**: 01 DB DEFAULT single source, 02 session var `app.fencing_machine_id`, 03 ALTER metadata-only heal, 04 migration 019 naming, 05 comment rewrite chính xác, 06 reject Go-side fill, 07 testcontainers Postgres, 08 không đụng V1 path.
- **6-phase plan (~4h effort)**: P0 reproduce → P1 migration → P2 Go DDL → P3 comment → P4 integration test → P5 deploy prod → P6 verify 24h + lesson.
- **22 task** (T0.1 → T6.4) với DoD per task + 6 gate verification (G0-G5 + G6 24h post-deploy).
- **Brain files**: 12 vật lý (00→10 + report_2026-06-02) = **1,992 LOC markdown / ~75 KB**. 0 source code change, 0 memory overwrite.
- **5 Pending Questions cho User**: (1) Random 8-bit seq OK hay monotonic per-machine? (2) Whitelist schema áp dụng migration? (3) Live deploy vs maintenance window? (4) Service nào khác cần backport? (5) Confirm format lesson global?
- **Lesson candidate (chờ User confirm)**: Pattern [Layer A claim contract X, Layer B claim Y, Layer C claim Z — but no layer implements X∩Y∩Z] → bug ẩn cho đến khi upstream refactor bóc lộ. Đúng: CI gate grep comment claim vs symbol implementation; canonical layer + link reference.
- **Verb chờ User**: `approve workspace` (start P0 reproduce) | `answer pending questions` | `request changes` | `reject solution` (Brain re-plan với approach khác).
- **Report**: `report_2026-06-02.md`

### 2026-06-02 — Workspace `bug-snapshot-v2-mapping-cache-stale-2026-06-02` (BRAIN PLAN COMPLETE, MINIMAL PATCH)
- **Path**: `agent/memory/workspaces/bug-snapshot-v2-mapping-cache-stale-2026-06-02/`
- **Severity**: 🔴 HIGH — snapshot.v2 source 66 (và mọi source có rule chưa bind shadow) ghi data nhưng 0 mapping field apply → master layer trống/sai → operator re-snapshot manual.
- **Trigger**: User report "snapshot.v2 source 66, dù đã có field approve nhưng khi snapshot vẫn ko mapping qua. có vẻ dính cache cũ. cache này rất vớ vẩn".
- **Root cause (1 dòng)**: `cdc-cms-service/internal/app/commands/update_mapping_rule.go:177-179` — NATS publish reload bị gate bởi `rule.ShadowTable != nil && *rule.ShadowTable != ""`. Khi `mapping_rule_v2.shadow_binding_id IS NULL` và JOIN `shadow_binding` trả NULL → publish skip silent → worker `centralized-data-service` không nhận `schema.config.reload` → `mappingCache` stale → snapshot.v2 dùng cache cũ → 0 mapping field mới.
- **Pattern match**: L-3110 (2026-05-18) **đảo ngược** — producer-conditional, consumer-unconditional = silent drop. Lesson hiện tại chỉ cover mặt subscriber-conditional → cần append entry inverse.
- **Patch tối thiểu (CHỐT)**: **1 file, ~5 LOC** — bỏ gate `ShadowTable != nil` trong approve handler. Truyền `shadowTable=""` khi nil. Worker `ReloadAll()` là full reload, hoàn toàn an toàn với empty string. 0 schema change, 0 NATS contract change, backwards-compat 100%.
- **Reject over-engineering**: Plan ban đầu Brain viết 8 ADR + 8 phase (worker post-reload count check, dispatch publish trước, Prometheus metrics, CI grep gate, integration test testcontainers) — User feedback "quá lâu, chớt quớt, bug tí teo" → cắt xuống 1 file patch. Defense-in-depth metrics + CI gate = future workspace (G6 systemic).
- **Brain files** (10 vật lý): 00_context, 01_requirements, 02_plan, 03_implementation, 04_decisions, 05_progress, 06_validation, 07_status, 08_tasks, 09_tasks_solution, 10_gap_analysis, report_2026-06-02. **Lưu ý**: 02_plan + 04_decisions viết theo phong cách multi-phase ban đầu → giữ làm forensics evidence, KHÔNG dùng cho execution. Execution chỉ `03_implementation.md §1`.
- **3-step muscle workflow**: (1) Apply patch `update_mapping_rule.go:177-179` (~5 LOC), (2) restart `cdc-cms-service`, (3) smoke test approve→snapshot→assert column rule mới có ở shadow.
- **Lesson candidate (chờ User confirm fix work prod)**: Global Pattern [Producer P publish signal S gated bởi derived attribute D, consumer C subscribe unconditional]. D nil/empty → P skip → C never trigger → cache/state stale silent. Đúng: P publish unconditional khi mutation success; payload mang identity primary key, không phải derived attribute. Inverse của L-3110.
- **Verb chờ User**: `apply patch` (muscle execute) | `request changes` | `reject solution`.
- **Report**: `report_2026-06-02.md`

### 2026-06-03 — Workspace `release-prod-roadmap-2026-06-03` (BRAIN ROADMAP, PLAN-ONLY)
- **Path**: `agent/memory/workspaces/release-prod-roadmap-2026-06-03/`
- **Mục tiêu**: Lộ trình audit nốt 4 luồng còn lại + E2E + hardening → release prod. Phase 1 (Source→Shadow) đã testing xong.
- **Bản đồ 5 luồng**: F1 Source→Shadow ✅ done · F2 Master/Transmute 🟡 audited (chờ execute, workspace `feature-masters-page-audit-2026-06-02`) · F3 SystemHealth 🔴 · F4 Reconcile 🔴 · F5 Monitor 🔴.
- **Timeline đề xuất (T0=06-03, 1 Muscle tuần tự)**: Phase A Master finalize (06-03→05) → B SystemHealth (→06-09) → C Reconcile (→06-13) → D Monitor (→06-16) → E Auth-debt+E2E+Hardening+Staging (→06-22) → F Prod readiness+Release (→06-25). **Go-live target 2026-06-25 ±2d**. Bật 2-Muscle fan-out B/C/D → rút ~06-19.
- **Prod blocker đã flag**: cdc-auth-service 0 test (E1, song song sớm); Staging mới Local smoke (E3 risk HIGH).
- **Doc set**: 00_context, 01_requirements_release, 02_plan_release, 08_tasks_release, 05_progress.
- **Chờ User**: trả lời Q1–Q4 (deadline / staging / SLA / incremental vs big-bang) + verb `chốt timeline` | `execute phase A` | `parallel` | `revise`.


### 2026-06-04 — Workspace `normalize-global-lessons-2026-06-03` (MEMORY MAINTENANCE, ✅ DONE)
- **Path**: `agent/memory/workspaces/normalize-global-lessons-2026-06-03/`
- **Mục tiêu**: Thống kê + tổng hợp + sắp xếp + chuẩn hoá toàn bộ `lessons.md` (530KB/5.1k dòng) → Global Patterns theo Rule 13.
- **Kết quả**: `agent/memory/global/lessons_global_normalized.md` — 229 Global Pattern, 8 nhóm taxonomy, 100% Rule 13, tag 750→496. Index APPEND vào lessons.md. Lesson gốc nguyên vẹn (md5 verified — Rule 11 OK).
- **Tooling**: `tooling_assemble_lessons.py` (tái lập: 9 sub-agent chunk → part-files → Python assembly group+sort).
- **Duy trì**: lesson mới vẫn APPEND vào lessons.md; định kỳ re-generate file normalized; LUÔN check wc -l trước/sau (đã gặp mid-session append +3 lesson).
- **Status**: ✅ DONE.

### 2026-06-04 — Workspace `feature-sync-shadow-master-bindings-2026-06-04` (✅ DONE)
- **Path**: `agent/memory/workspaces/feature-sync-shadow-master-bindings-2026-06-04/`
- **Mục tiêu**: Đồng bộ tự động các mapping rules từ shadow sang master và tự động duyệt (approve) rules khi Approve Master Binding để giải quyết triệt để lỗi sinh ra empty table (chỉ có default columns) ở destination.
- **Kết quả**: Implement transactional logic trong cdc-cms-service (`approve_master.go`) để sync/approve 100% các rules tự động. Đã verify bảng vật lý `b3` được tạo ra đầy đủ tất cả các business columns và database rules đồng bộ thành công.

### 2026-06-08 — Workspace `feature-cdc-cms-file-inventory-2026-06-08` (ANALYSIS, ✅ DONE)
- **Mục tiêu**: inventory cấu trúc + chức năng 213 file `cdc-cms-service` + đề xuất reorg + dò file chưa dùng.
- **Phương pháp**: go reachability + deadcode tool (75 hàm) + 7 sub-agent grep-analysis.
- **Kết quả**: `11_current_structure.md` (toàn bộ file + status), `12_unused_files.md` (8 file dead/575 LOC + 2 nghi + categories), `13_proposed_structure.md` (vertical slice 8 BC + DEAD bucket).
- **Nối tiếp**: v1 restructure + v2 hexagonal-refactor (đều pending review). 0 code change.
- **Next**: user duyệt danh sách dead → 1 PR remove (~575 LOC) verify go build+deadcode.
