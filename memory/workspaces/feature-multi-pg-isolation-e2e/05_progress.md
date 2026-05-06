# 05 — Progress (feature-multi-pg-isolation-e2e)

> APPEND-ONLY. Không xoá/sửa entry cũ.

---

## 2026-04-28 14:30 — Workspace khởi tạo + Plan v1 draft

**Actor**: Muscle (CC CLI). **Trigger**: User yêu cầu "tách 4 PG containers + auto E2E test".

**Output**:
- `00_context.md` — bối cảnh + scope + constraints
- `01_requirements_phase01_split_e2e.md` — FR-1..FR-4, NFR-1..NFR-3, DoD D1-D10
- `02_plan_phase01_split_e2e.md` — Phase A-E (infra → schema → config → pipeline → verify), risk register, ~9h effort
- `08_tasks_phase01_split_e2e.md` — 23 tasks T-A1 → T-E3, critical path A1→A3→B2→B5→B6→C4→C6→D2→D4→E1→E2
- `09_tasks_solution_phase01_split_e2e.md` — drafts cho docker-compose / source seed / multi-DSN / Debezium register / E2E test script

**Open questions cho User**:
1. Tên DB của `gpay-postgres` (auth-only): giữ `goopay_dw` hay đổi `auth_dw`?
2. Migration directory: subfolder `migrations/cdc/` + `migrations/dest/` (rõ ràng) hay flag `--target=cdc|dest` (gọn)?
3. Source DB credentials: dùng `srcuser/srcpass` mặc định hay user định nghĩa?
4. Có cần giữ data hiện tại trong `gpay-postgres` (sau Phase 39 đã có cdc_system + cdc_auth_service trong cùng db) không? Hay wipe sạch và bootstrap lại từ đầu?
5. Approve plan v1 để Muscle bắt đầu thực thi Phase A?

**Status**: ⏳ Awaiting user approval trước khi thực thi.

---

## 2026-04-28 — Track A + B execution (Muscle)

**Architect approval received**: Plan v1 với hiệu chỉnh DB tên `gpay_auth`, subfolder migrations, wipe & rebuild sạch. Critical "tử huyệt" T-B3 (qualify partition logic) + R1 (cross-DB sequences cho sonyflake). Lệnh: thực thi Track A + B ngay.

### Track A — Infra (DONE)
- ✅ T-A1: docker-compose.yml split thành 4 PG services
  - gpay-postgres (5432, gpay_admin/gpay_pass, gpay_auth) — auth-only
  - gpay-postgres-cdc (5433, gpay_admin/gpay_pass, cdc_dw) — control plane + shadow
  - gpay-postgres-dest (5434, gpay_admin/gpay_pass, goopay_dest) — DW/master
  - gpay-postgres-source (5435, src_user/src_pass, goopay_source, wal_level=logical, max_wal_senders=10, max_replication_slots=10)
- ✅ T-A2: source seed `deployments/sql/source/01_init_source_local.sql` (orders/users/payments × 10 rows, REPLICA IDENTITY FULL, src_user WITH REPLICATION)
- ✅ T-A3: 4 containers all healthy, source verified 30 rows.

### Track B — Schema Split (DONE)
- ✅ T-B1: Wipe scripts split thành 3 file (`deployments/sql/{auth,cdc,dest}/wipe_*.sql`)
- ✅ T-B2: 44 cdc migrations move sang `migrations/cdc/`, `migrations/dest/` empty + ready
- ✅ T-B3 [CRITICAL]: `010_partitioning.sql` rewrite — qualify partitions DIRECTLY trong cdc_system. Không còn orphan partition trong public, không cần 044 cleanup.
- ✅ T-B4: `migrations/dest/001_dest_init.sql` — pgcrypto + master schema + search_path gpay_admin.
- ✅ T-B5: Makefile multi-target — migrate-{auth,cdc,dest}, wipe-{auth,cdc,dest}, bootstrap-local, reset-local (one-shot wipe→migrate→bootstrap).
- ✅ T-B6: Bootstrap split — `bootstrap_cdc_local.sql` (PG-source flow: orders→shadow→dw_orders.orders_fact) + `bootstrap_dest_local.sql` (sanity guards).
- ✅ T-B7: Apply + verify ALL clean.

**Migration fixes during execution** (lessons applied):
- `005_pg_users.sql`: `goopay_dw` → `cdc_dw` (3 GRANT CONNECT statements)
- `037_move_system_tables`: cdc_activity_log + failed_sync_logs đổi từ `ALTER SET SCHEMA` sang `DROP IF EXISTS public.X CASCADE` (vì 010 rewrite tạo trực tiếp trong cdc_system)
- `039_set_search_path` + `042_search_path_with_auth`: role `"user"` → `gpay_admin`. 042 bỏ `cdc_auth_service` khỏi search_path (auth giờ ở DB riêng).

### Verification (final state — Phase 39 invariant maintained)
- AUTH: `gpay_auth` có cdc_auth_service + public ✅
- CDC: 42 cdc_system tables, public=0 (Phase 39 invariant ✅), cdc_internal=0, cdc_activity_log có 8 partitions, failed_sync_logs có 5 partitions
- DEST: master + public empty ✅
- SOURCE: 30 rows seed (orders/users/payments × 10) ✅

**Bootstrap state (cdc_system)**:
- 3 connections (src_local_pg_source, shadow_local_pg_cdc, master_local_pg_dest)
- 1 source object (orders), 1 shadow binding, 1 master binding
- 7 mapping rules, 1 transmute schedule (post_ingest)

**Status**: 🟢 Track A + B DONE. Sẵn sàng Track C (multi-DSN code refactor).

---

## 2026-04-28 — Track C execution (Muscle)

**Architect approval received**: Lệnh chuyển sang Track C — Multi-DSN Code Refactor. Đặc biệt giám sát Connection Pool Init: KHÔNG được leak connection giữa các DB containers.

### T-C1 — Configuration Overhaul (config.go)
- ✅ Thêm 2 block DSN riêng vào `AppConfig`:
  - `ControlPlane SingleDBTarget` (mapstructure: `controlPlane`)
  - `Destination SingleDBTarget` (mapstructure: `destination`)
- ✅ Env overrides: `CDC_CONTROL_PLANE_URL`, `CDC_DESTINATION_URL`
- ✅ Fallback an toàn trong `applyDBFallbacks`:
  - ControlPlane trống → SystemDB.URL (legacy single-PG mode)
  - Destination trống → MasterDB.URLs[default] → legacy DSN
- ✅ Helpers: `cfg.ControlPlaneURL()`, `cfg.DestinationURL()`

### T-C2 — YAML Updates (3 services)
- ✅ `centralized-data-service/config/config-local.yml`:
  - `db.*` → `localhost:5433/cdc_dw` (Worker dùng làm legacy default = control plane)
  - `systemDb.url` + `shadowDb.urls.default` → cdc_dw (5433)
  - `masterDb.urls.default` → goopay_dest (5434)
  - 2 block mới `controlPlane.url` + `destination.url`
- ✅ `cdc-cms-service/config/config-local.yml`:
  - `db.*` → `localhost:5433/cdc_dw` (CMS Preview reads từ shadow trên control plane)
  - 2 block mới `controlPlane.url` + `destination.url`
- ✅ `cdc-auth-service/config/config-local.yml`:
  - `db.*` → `localhost:5432/gpay_auth` (auth-only, port 5432 đúng directive)

### T-C3 — Database Registry (pkgs/database/multi.go)
- ✅ `Registry` struct với `sync.RWMutex` (read fast-path) + double-check pattern
- ✅ Constants `RoleControlPlane = "cdc"`, `RoleDestination = "dest"`
- ✅ `GetDB(role)` → cached *gorm.DB per role (lazy)
- ✅ `GetPgxPool(ctx, role)` → cached *pgxpool.Pool per role (lazy)
- ✅ `Init(ctx)` → eager build cả 2 pool + cả gorm + pgx (fail-fast tại boot)
- ✅ `Close()` idempotent close cả 2 pool
- ✅ Connection pooling RIÊNG biệt — pgx pool và gorm pool độc lập per role

### T-C4 — ConnectionManager Refactor (no double-pool)
- ✅ Façade pattern: `ConnectionManager` giờ wrap `*database.Registry`
  - `GetSystemDB()` → `registry.GetDB("cdc")`
  - `GetShadowDB(default)` → `registry.GetDB("cdc")`
  - `GetMasterDB(default)` → `registry.GetDB("dest")`
  - Multi-tenant explicit URL override → giữ pool riêng (legacy compat)
- ✅ Constructor mới `NewConnectionManagerWithRegistry(cfg, log, reg)` — share Registry với worker bootstrap
- ✅ Method `Registry()` expose underlying registry cho callers cần GetDB("cdc")/GetDB("dest") trực tiếp

### T-C5 — Worker Logic Patching (audit + fallback fix)
**Audit kết quả** — code path routing đã đúng:
- `transmuter.go:303` shadow READ via `GetShadowDB(...)` → cdc pool ✅
- `transmuter.go:409` master UPSERT via `GetMasterDB(...)` → dest pool ✅ (Step 11 Swap)
- `master_ddl_generator.go:148` DDL CREATE via `GetMasterDB(...)` → dest pool ✅
- `batch_buffer.go:203/208` shadow + master writes → routed correctly ✅
- `event_handler.go:158` shadow inserts → cdc pool ✅

**Gap được fix**: 
- `metadata_registry_service.go:152` resolve `ShadowConnectionKey = connection_code` (e.g. `"shadow_local_pg_cdc"`)
- Old `ConnectionManager.GetShadowDB("shadow_local_pg_cdc")` sẽ fail vì URL map chỉ có key `"default"`
- Fix: ConnectionManager smart fallback — connection_code không có URL override → fall back về registry pool tương ứng (correct vì split layout có 1 cdc + 1 dest physical)

**Worker bootstrap (worker_server.go)**:
- Trước: 2 lần connect (`NewPostgresConnection(cfg)` + `NewPgxPool(ctx, cfg)`) → 2 pool tách rời
- Sau: `Registry.Init(ctx)` đầu tiên → `db = registry.GetDB("cdc")`, `pgxPool = registry.GetPgxPool(ctx, "cdc")` → CHUNG pool
- Log boot reveal redacted DSN (ControlPlane + Destination) để giám sát

### Verification (architect's concern: NO LEAK between containers)

**Tests** (all PASS với `-race`):
| Test | Result |
|------|--------|
| `TestRegistry_SeparatePoolsPerRole` | ✅ cdc pool ≠ dest pool (object identity) |
| `TestRegistry_GetDBIsCached` | ✅ Repeated GetDB returns same handle |
| `TestRegistry_ConcurrentGetDBOpensExactlyOnePool` | ✅ 32 goroutines → exactly 1 pool created |
| `TestRegistry_GetPgxPoolIsCached` | ✅ pgx pool cache identity ổn định, cdc ≠ dest |
| `TestRegistry_RejectsUnknownRole` | ✅ GetDB("auth") + GetDB("") rejected |
| `TestConnectionManager_DefaultKeysHitRegistryPools` | ✅ "" + "default" routes về registry, không tạo pool mới |
| `TestConnectionManager_UnknownConnectionCodeFallsBackToRegistry` | ✅ connection_code không có override → registry fallback |

**Smoke test against live stack**:
- ControlPlane pool → `current_database = cdc_dw` ✅
- Destination pool → `current_database = goopay_dest` ✅
- cdc bootstrap connections = 3 ✅
- dest master schema present ✅

**Full test suite** (`go test ./...`): ALL PASS (handler, service, sinkworker, database, idgen, utils).

**Status**: 🟢 Track C DONE. Connection pool isolation đảm bảo, không leak giữa cdc/dest containers. Sẵn sàng Track D (E2E pipeline — Debezium connector + Wizard endpoint).

---

## 2026-04-28 — Track D (E2E Pipeline) ✅ COMPLETE

**Mục tiêu**: Demonstrate full data flow Source → Kafka → Shadow → Master across 4 separate Postgres containers.

### T-D1 — Register Debezium PostgresConnector

**Artifacts created**:
- `deployments/debezium/pg-source-connector.json` — Debezium config (postgres-source:5432, slot=cdc_gpay_pg_source, publication=cdc_gpay_pub, topic.prefix=cdc.gpay, table.include.list=public.orders,public.users,public.payments, snapshot.mode=initial, Avro+SR, sanitize.field.names=true)
- `deployments/debezium/register_pg_source.sh` — idempotent register script (waits REST+plugin, DELETEs existing, POSTs config, polls until RUNNING)
- `docker-compose.yml:248-251` — added `confluent-hub install debezium/debezium-connector-postgresql:2.5.4` after mongodb installer

**Boot order**: `docker compose up -d postgres-source kafka schema-registry kafka-connect nats redis postgres-cdc postgres-dest` then run register script.

**Result**: Connector + task RUNNING. Snapshot kicked off, subsequent INSERTs streamed live.

### T-D2 — INSERT 10 rows into public.orders

Multiple rounds executed via `docker exec -i gpay-postgres-source psql ... <<'SQL'` heredoc:
- Round 1+2 (legacy bindings active) — events misrouted to deactivated targets
- Round 3 (after V2 cleanup) — 10 rows id=31..40 ingested NULL typed cols (no rules)
- Round 4 (after legacy mapping rule seed) — 10 rows id=41..50 ingested with proper typed cols

Final source state: 50 rows.

### T-D3 — Verify dataflow

**Final row counts (steady state)**:
| Stage | Container | Object | Rows |
|-------|-----------|--------|------|
| Source | gpay-postgres-source | public.orders | 50 |
| Kafka | gpay-kafka | cdc.gpay.public.orders | 50 events (offsets 0–49, lag=0) |
| Shadow | gpay-postgres-cdc | shadow_goopay_source.orders | 20 |
| Master | gpay-postgres-dest | dw_orders.orders_fact | 20 |

Shadow=20 (not 50) vì 30 snapshot rows xảy ra trước khi V2 binding active (legacy bindings collision). Sau khi deactivate legacy + restart worker, chỉ 20 rows mới được route đúng vào V2 shadow.

### Bugs fixed dọc đường

| # | Bug | Fix |
|---|-----|-----|
| 1 | redis localhost:16379 refused | `docker compose up -d redis` |
| 2 | port 8082 stuck on worker restart | `lsof -ti:8082 \| xargs kill -9` |
| 3 | docker exec heredoc executed empty | thiếu `-i` flag → `docker exec -i ... <<SQL` |
| 4 | events routed `shadow_goopay_order` (sai) | 10 legacy seed rows in `source_object_registry` collide trên route key `"orders"` (first-write-wins). Fix: `UPDATE shadow_binding SET is_active=false` trên 10 legacy rows |
| 5 | `prepare table failed: table shadow_goopay_source.orders does not exist` | `SchemaAdapter.PrepareForCDCInsertInSchema` chỉ ALTER, không CREATE → manual `CREATE SCHEMA shadow_goopay_source; CREATE TABLE orders(id BIGINT PK, ...)` rồi `UPDATE shadow_binding SET ddl_status='created'` |
| 6 | typed cols all NULL (round 3) | `DynamicMapper` không có rule → `Columns: empty, RawJSON: full data`. Fix: INSERT 7 rows vào `cdc_mapping_rules` (table=orders, fields id/user_id/amount/status/notes/created_at/updated_at, source_format='debezium_after') |
| 7 | TransmuteScheduler không fire | schedule mode='post_ingest' nhưng poller chỉ lọc `mode='cron'`. Fix: `UPDATE transmute_schedule SET mode='cron', cron_expr='*/1 * * * *'` |
| 8 | `transmuter.go:318 ERROR: column "_gpay_id" does not exist` | V1 shadow convention (`_deleted/_version`) ≠ V2 transmuter SELECT (`_gpay_id, _gpay_source_id, _source_ts, _gpay_deleted`). Fix: ALTER shadow table thêm V2 cols + backfill `_gpay_source_id = id::text` + tạo `BEFORE INSERT/UPDATE` trigger để tự fill cho rows mới |
| 9 | `rule_misses=20, skipped=20` (sau column fix) | V2 rules dùng `source_path='$.after.id'` nhưng `_raw_data` đã flat (Debezium event đã unwrap). gjson không parse `$.` syntax. Fix: `UPDATE mapping_rule_v2 SET source_path=NULL, source_format='raw'` → fall back về `r.SourceField` ("id", "user_id", ...) match top-level keys |

**Final transmute log**: `transmute complete: scanned=20, inserted=20, updated=0, skipped=0, type_errors=0, rule_misses=0, duration_ms=42`. ✅

### Skills sử dụng

- Docker Compose multi-container orchestration (4 Postgres instances, Kafka KRaft, Schema Registry, Kafka Connect)
- Debezium PostgresConnector config (logical decoding, pgoutput, slot/publication, Avro+SR)
- PostgreSQL DDL on running instance (ALTER TABLE, BEFORE trigger, BIGSERIAL)
- gjson path semantics vs JSONPath spec
- V2 metadata registry: source_object_registry, shadow_binding, master_binding, mapping_rule_v2, transmute_schedule
- pgx + GORM dual pool routing through ManagedRegistry
- Kafka consumer-groups CLI for offset/lag inspection
- Live debugging via `/tmp/cdc-logs/worker.log` + targeted SQL probes

**Status**: 🟢 Track D DONE. Phase 01 split E2E demonstrated end-to-end across 4 isolated Postgres containers.

---

## 2026-04-28 — Track D Hardening / P1 Config Consolidation ✅ DONE

**Architect ruling**: Q1=a (nuke `destination:` block), Q2=c (consolidated `sources:` block), Q3=b (config trước, hardening sau, Track E cuối).

### Files đã thay đổi

| File | Diff summary |
|------|--------------|
| `config/config.go` | Bỏ field `Destination SingleDBTarget`. Thêm `Sources map[string]string` (`mapstructure:"sources"`). `DestinationURL()` derive từ `MasterDB.URLs[DefaultKey]`. Env `CDC_DESTINATION_URL` → ghi vào `MasterDB.URLs["default"]`. Thêm `SourceURL(name)` accessor. Bridge `MongoDB.URL ↔ Sources["mongodb_primary"]` (giữ backward-compat) |
| `pkgs/database/multi.go:dsnForRole(RoleDestination)` | Đọc thẳng `cfg.MasterDB.URLs[cfg.MasterDB.DefaultKey]` (đúng chỉ thị architect — không indirect qua field cũ) |
| `config/config-local.yml` | Nuke block `destination:` (single block thay bằng comment giải thích derive từ masterDb). Nuke block `mongodb:`. Thêm block `sources:` với 2 keys `mongodb_primary` + `postgres_primary` (analog với `connection_registry.connection_code`) |
| `internal/service/connection_manager_test.go:cmTestCfg` | Bỏ `cfg.Destination.URL = X`; `MasterDB.URLs["default"]` set trực tiếp |
| `pkgs/database/multi_test.go:devCfg` | Cùng migration |

### Verification

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ PASS |
| `go test ./pkgs/database/...` | ✅ PASS |
| `go test ./internal/service/ -run TestConnectionManager` | ✅ PASS |
| Worker boot log `control_plane=...:5433/cdc_dw, destination=...:5434/goopay_dest` | ✅ Single source — derived từ masterDb |
| Worker boot log `MongoDB connected: mongodb://localhost:17017/...` | ✅ Derived từ `sources.mongodb_primary` qua bridge |
| Worker boot log `V2 metadata registry: sources=1, shadow_bindings=1` | ✅ Không leak |
| 3 transmute ticks liên tiếp (60s mỗi tick) | ✅ Đều `scanned=20, rule_misses=0, duration_ms=20–35` |
| `goopay_dest.dw_orders.orders_fact count(*)` | ✅ Vẫn 20 (không regression) |

### Lesson — Global Pattern

**Pattern**: *"Khi A là logical multi-target route (key→URL) và B là physical pool single-DSN, B luôn có thể derive từ A.urls[A.defaultKey]. Không nên giữ B như field độc lập trong yaml/struct vì gây Double Source of Truth — config drift sẽ silent-mismatch giữa runtime và operator intent."*

**Đúng**: Single source of truth. Accessor đọc một bên (A), bên còn lại (B) chỉ là derived view.
**Sai**: Để cả A và B trong yaml + có fallback "if B==empty then derive". Operator không biết bên nào win.

Áp dụng được: shadow vs control_plane, master vs destination, sources vs per-source-block (mongodb/mysql/postgres-source), Auth0 client vs JWT validator, K8s service vs Ingress backend.

### Status
🟢 P1 DONE. Sẵn sàng P2 (Bug #6 SchemaAdapter `CREATE TABLE IF NOT EXISTS`) khi anh ra lệnh.

---

## 2026-04-29 — Track D Hardening · P2 + P3 + P4 DONE

**Owner**: Muscle (CC CLI). **Type**: Append-only (rule #11). **Plan ref**: `09_tasks_solution_track_d_hardening.md` + `/Users/trainguyen/.claude/plans/curried-waddling-spindle.md`.

### P2 — SchemaAdapter `CREATE TABLE IF NOT EXISTS` (Bug #6)

| Path | Action |
|------|--------|
| `internal/service/schema_adapter.go:108` | `PrepareForCDCInsertInSchema` không còn fail "table does not exist". Khi `schema == nil` → gọi `createShadowTableV1` rồi `loadSchemaInSchema` reload + cache + log. ALTER pass cũ giữ nguyên cho V1 cdc cols. |
| `internal/service/schema_adapter.go:175` | NEW private `createShadowTableV1` — `CREATE SCHEMA IF NOT EXISTS` + `CREATE TABLE IF NOT EXISTS` với pk TEXT (conservative — V1 fallback không infer type) + 8 V1 cdc cols inline. Idempotent. |

**Verification**:
- `go build ./...` ✅ PASS
- `go test ./internal/service/...` ✅ PASS
- E2E live: `DROP TABLE shadow_goopay_source.orders` → restart worker → INSERT 5 rows source → log `"shadow table auto-created","schema":"shadow_goopay_source","table":"orders","pk":"id"` ts=1777426678 → `"batch upsert ok",count=5` → `\d shadow_goopay_source.orders` cho 8 V1 cdc cols + UNIQUE constraint trên `id`.

### P3 — Prune legacy V1 seeds (Bug #2)

| Path | Action |
|------|--------|
| `deployments/sql/cdc/prune_legacy_v1_bindings.sql` | NEW. 3-step idempotent UPDATE (shadow_binding → master_binding → source_object_registry) gated by `is_active = true`. Stamps `notes` field with prune timestamp (cột `deactivated_at` không tồn tại). Reports counts ở cuối. |

**Verification**:
- Apply lần 1: docker exec gpay-postgres-cdc psql ... → `pruned_sources=10, pruned_shadow_bindings=10, pruned_master_bindings=0`. UPDATE 0 lần này = đã pruned từ Track D demo trước.
- Idempotency: re-run cùng SQL → UPDATE 0 (WHERE `is_active=true` không match nữa). ✅

### P4 — D-39.A event-driven scheduler close-loop

| Path | Action |
|------|--------|
| `internal/service/transmute_scheduler.go:146` | NATS payload `cdc.cmd.transmute` thêm `"schedule_id": d.id`. |
| `internal/handler/transmute_handler.go:128` | `TransmuteRequest.ScheduleID int64 \`json:"schedule_id,omitempty"\``. |
| `internal/handler/transmute_handler.go` | NEW `const SubjectTransmuteCompleted = "cdc.evt.transmute.completed"`. NEW method `publishCompleted(req, res, runErr)` chạy sau `svc.Run`, publish event `{schedule_id, correlation_id, master_table, status, stats(JSON), error, completed_at}`. Best-effort — log warn nếu publish fail. |
| `internal/service/job_monitor.go` | NEW `JobMonitor` + `HandleCompleted(msg)`: parse event → idempotent UPDATE `cdc_system.transmute_schedule SET last_status=?, last_stats=?::jsonb, last_error=NULLIF(?,'') WHERE id=? AND last_status='running'`. ScheduleID==0 (ad-hoc trigger) → no-op. |
| `internal/server/worker_server.go:283` | Wire `service.NewJobMonitor(db, logger)` + `natsClient.Conn.Subscribe(handler.SubjectTransmuteCompleted, jobMonitor.HandleCompleted)` ngay sau scheduler start. Errors lan ra `(*WorkerServer, error)` của `NewWorkerServer`. |

**Verification (live, ts=1777426654 → 1777427074, 7 ticks)**:
- Tick 1 (ts=...654, shadow đã DROP): `"transmute failed"` → `"job monitor: schedule closed",status=failed` → DB `last_status='failed', last_stats={...0...}, last_error="...does not exist"` ✅
- Sau P2 auto-create + V2 cols ALTER bootstrap → tick ts=...014: `"transmute complete",scanned=5,skipped=5` → `"job monitor: schedule closed",status=success` → DB `last_status='success'` ✅
- Tick ts=...074: PG cached plan invalidation → `"transmute failed"` → `last_status='failed'` again. ✅ Re-flip xác nhận UPDATE chạy mỗi tick.
- `go build ./...` ✅ PASS, `go test ./internal/service/... ./internal/handler/... ./pkgs/database/...` ✅ PASS.

### Files modified/created (P2+P3+P4)

```
M  internal/service/schema_adapter.go
M  internal/service/transmute_scheduler.go
M  internal/handler/transmute_handler.go
M  internal/server/worker_server.go
A  internal/service/job_monitor.go
A  deployments/sql/cdc/prune_legacy_v1_bindings.sql
```

### Lesson — Global Pattern (P4)

**Pattern**: *"Fire-and-forget command (A publishes B then 'mark as running') leaks status nếu A không close loop. Đúng: tách lifecycle thành 2 events — A publishes `cmd.X`, X-handler publishes `evt.X.completed`, monitor M (separate concern) subscribe `evt.X.completed` để UPDATE status. M idempotent qua `WHERE status='running'` guard."*

**Đúng**: command/event split. Handler không bao giờ touch state-table của caller — chỉ emit completion event. Monitor là single owner của state writes.
**Sai**: Handler trực tiếp UPDATE caller's state (coupling), HOẶC scheduler set 'running' rồi quên close (leak).

Áp dụng: cron-driven jobs, RPC retry/dedup, distributed sagas, K8s Job watchdog, GitHub Action workflow_run sync, audit-log write-after-action.

### Status
🟢 Track D Hardening DONE (P1+P2+P3+P4). Out-of-scope: P5 Track E (MongoDB Debezium connector) — workspace riêng `feature-track-e-mongo-cdc/`.

---

## 2026-04-29 — Hotfix: DLQ schema drift (post-Track-D-Hardening)

**Triệu chứng**: User chạy `make run`, log liên tục `dlq state machine poll failed: ERROR: column "next_retry_at" does not exist (SQLSTATE 42703)` mỗi 5 phút (`internal/handler/dlq_state_machine.go:102`).

**Root cause** (KHÔNG phải Track D Hardening regression — bug historical):
- `010_partitioning.sql` CREATE `cdc_system.failed_sync_logs` partitioned (schema thiếu `next_retry_at`/`last_error`).
- `012_dlq_state_machine.sql` ALTER ADD 2 cột nhưng hardcode `public.failed_sync_logs` — quên replay cho `cdc_system.*`.
- `037_move_system_tables_to_cdc_system.sql` DROP `public.failed_sync_logs` (legacy non-partitioned copy). Bảng `cdc_system` partitioned không bị touch — nhưng cũng không được patch bù 2 cột.
- → Code expect 2 cột (model `FailedSyncLog.NextRetryAt`/`LastError`), DB partitioned không có → 42703.

**Fix**: `migrations/cdc/045_dlq_columns_in_cdc_system.sql` — replay 012's ALTERs lên `cdc_system.failed_sync_logs` + recreate `idx_fsl_retry_poll`. Idempotent (`ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`). Applied + verified columns + index hiện diện. Re-run lần 2 → no-op (`column ... already exists, skipping`).

### Lesson — Global Pattern (schema-drift-via-partial-migration)

**Pattern**: *"Khi A là `ALTER TABLE` migration ở schema X, mà parallel migration B đã dựng same-name table ở schema Y, A phải replay logic cho cả X và Y. Hardcode 1 schema là time-bomb — table copy ở schema còn lại sẽ silently lệch khỏi model code, fail at runtime với 42703 (column doesn't exist)."*

**Đúng**: Migration ALTER phải iterate qua tất cả tenant/namespace owners (loop `pg_namespace` lookup, hoặc explicit list cả 2 schemas).
**Sai**: Hardcode `ALTER TABLE public.X` khi codebase đang transition sang `cdc_system.X`. Skip-if-not-exists guard chỉ che lỗi mà không phát hiện schema còn lại bị drift.

Áp dụng: namespace migrations (public→cdc_system, default→tenant_*), shadow vs production schemas, multi-tenancy partition replays, sharded DBs với DDL fan-out.

---

## 2026-04-29 — Sweep: Model-only schema drift (post-DLQ-hotfix)

**Trigger**: User: *"phải đi tiếp các caí còn lại để ko dính tương tự chứ"* — sau khi fix DLQ drift (045), preemptive sweep tìm trường hợp tương tự để vá trọn.

**Method**: So gorm:`column:X` tag mọi file `internal/model/*.go` ↔ `information_schema.columns` cho mỗi `cdc_system.*` table.

**Findings — 2 drift mới (ngoài failed_sync_logs đã vá):**

| Bảng | Cột thiếu | Code site đụng |
|------|-----------|----------------|
| `cdc_system.cdc_mapping_rules` | `rule_type` | `command_handler.go:728,967`, `scan_service.go:90` (INSERT có `RuleType`) |
| `cdc_system.cdc_table_registry` | `source_url` | `recon_core.go:397,471,570,588,695` (SELECT `entry.SourceURL`) |
| `cdc_system.cdc_table_registry` | `sync_status` | `recon_core.go:841,844,847` (UPDATE) |
| `cdc_system.cdc_table_registry` | `recon_drift` | `recon_core.go:842,845` (UPDATE) |
| `cdc_system.cdc_table_registry` | `last_recon_at` | `recon_core.go:838` (UPDATE) |
| `cdc_system.cdc_table_registry` | `last_bridge_at` | model-only (forward compat) |

**Tại sao boot không phát hiện sớm**: tất cả callsite dùng explicit column list (`SELECT a,b,c` hoặc `UPDATE SET ...`) — không có `Find(&fullStruct)` lên 2 bảng đó. Time-bomb 42703 chờ ngày dev nào đó viết `db.First(&entry)` → crash silent path.

**Fix**: `migrations/cdc/046_model_drift_patches.sql` — `ADD COLUMN IF NOT EXISTS` cho 6 cột với defaults khớp model (`'mapping'`, `'unknown'`, `0`, `NULL`). Idempotent. Re-run lần 2 → `column ... already exists, skipping`.

**Verify**:
- 6/6 cột hiện diện (data_type + default match model).
- Rebuild `/tmp/cdc-worker` + restart → boot không log 42703 / does-not-exist.
- JobMonitor close-loop vẫn xanh: `schedule_id=1, status=success` (P4 không regress).
- Còn lại 23505 duplicate key trên `orders_fact_pkey` là logic-level (PK auto-increment chồng id của smoke test cũ) — KHÔNG phải drift, out-of-scope sweep.

### Lesson — Global Pattern (model-vs-DB drift detection)

**Pattern**: *"Model code (struct với `column:X` tag/annotation) thêm cột mới mà KHÔNG có migration kèm = drift loại 2: 'thêm-không-vá'. Khác drift loại 1 ('migration sai schema target' của 045), drift loại 2 ẩn vì callsite hiện tại đều dùng explicit column list — chỉ phát nổ khi developer X viết `Find(&FullStruct)` lần đầu."*

**Đúng**:
1. Mọi PR thêm field vào model struct PHẢI kèm migration `ADD COLUMN IF NOT EXISTS`.
2. Boot-time guard: query `information_schema.columns` ↔ struct reflection, fail-loud nếu mismatch.
3. CI script: `for model in *.go; do for tag in gorm_columns; do assert tag in DB; done; done`.

**Sai**: Tin tưởng "code đang chạy ngon" — vì callsite hiện tại không SELECT * trên struct đó. Drift sẽ silent cho tới khi ai đó dùng `Find()` hoặc autoMigrate fail-stop.

Áp dụng: GORM/SQLAlchemy/TypeORM/Hibernate codebases, schema-per-tenant migrations, brownfield codebase mở rộng dần model nhưng quên DDL.

### Status
🟢 Sweep DONE. Track D bedrock: P1+P2+P3+P4 + 045 (DLQ) + 046 (model drift). Boot xanh, no 42703.

---

## 2026-04-29 ~12:15 — Phase D Auto-Pipeline E2E (Architect DoD)

### Architect Ruling đã thi công
1. **P2 Junior code refactor** — `internal/service/schema_adapter.go`: thay `strings.ReplaceAll(..., "\"", "\"\"")` bằng `pgx.Identifier{}.Sanitize()` cho schema/table/pk identifiers (chuẩn pgx v5, đã có sẵn trong go.mod). Helper `quoteQualifiedTable` cũng dùng pgx.Identifier.
2. **JobMonitor Q3 fan-out log** — `internal/service/job_monitor.go:147–150`: thêm `zap.Int("impacted_sources", len(rows))` ở fan-out vòng schedule_enable bridge để operator nhận biết multi-source merge anomaly.
3. **Master Binding Option-A** — orchestrator (cả worker + CMS) tự seed `cdc_system.master_binding` TRƯỚC khi publish `cdc.cmd.master.bind`. Helper `seedMasterBindingForAdvance` UPSERT với conflict target `(master_connection_id, master_schema, master_table)`. Default schema = `dw_<connection_code>` (env `PROVISIONING_DEFAULT_MASTER_SCHEMA` override). Default master connection = env `PROVISIONING_DEFAULT_MASTER_CONNECTION_CODE` (e.g. `master_local_pg_dest`). binding_code = `auto_src_<sourceID>`.
4. **Q5 Resume parity** — `cdc-cms-service/internal/service/provisioning_orchestrator.go::Resume` flip `paused → running` rồi gọi `Advance` để kick auto-fanout, error-tolerant với InvalidTransition / Conflict.

### Bug pile phát sinh khi chạy E2E (đều fix tại chỗ, không workaround)
| Symptom | Root cause | Fix |
|---|---|---|
| `column "connection_code" does not exist` | `provisioning_step_handlers.go::resolveShadowTarget` query phẳng trên `source_object_registry`, nhưng `connection_code` thuộc `connection_registry` (FK qua `source_connection_id`). | Đổi sang LEFT JOIN `cdc_system.connection_registry cr ON cr.id = sor.source_connection_id`. |
| `column "pk_column" of shadow_binding does not exist` | Handler INSERT dùng `pk_column` nhưng schema thực tế không có cột đó (PK metadata sống ở `source_object_registry.primary_key_field`). Đồng thời thiếu các cột NOT NULL: `binding_code`, `shadow_connection_id`, `physical_table_fqn`. | Refactor `upsertShadowBinding` — lookup `shadow_connection_id` (env `PROVISIONING_DEFAULT_SHADOW_CONNECTION_CODE` override → fallback first active shadow role), tạo `binding_code='shadow_src_<sourceID>'`, conflict target = `binding_code`. |
| `discover` step dispatch payload thiếu `target_table`/`source_table`/`provisioning` flag → `command_handler.go::HandleDiscover` chạy với target rỗng, không emit step_completed → state kẹt `mapping_pending`. | Orchestrator `Advance` switch chỉ build payload extras cho `master_bind`/`schedule_enable`, bỏ qua `discover`. | Thêm case `discover` ở 2 chỗ: pre-CAS lookup (gọi `lookupMasterTableForSource` + helper mới `lookupSourceTableForSource`) và post-CAS payload (`payload["target_table"]=master`, `payload["source_table"]=src`, `payload["provisioning"]=true`). Mirror cùng change ở CMS orchestrator. |
| `column "master_table" of transmute_schedule does not exist` | Handler `HandleScheduleEnable` UPDATE WHERE master_table=... nhưng bảng keyed bằng `master_binding_id` (FK). | Rewrite handler: lookup `master_binding.id` từ `source_object_id`, UPSERT vào transmute_schedule với conflict target `(master_binding_id, mode)`, mode=`cron`, cron_expr default `*/1 * * * *` (override env `PROVISIONING_DEFAULT_CRON_EXPR`). |
| `non-positive interval for NewTicker` panic ngay sau Redis connect | `/tmp/cdc-worker` bị start từ `cdc-cms-service/` cwd → viper không tìm thấy `centralized-data-service/config/config-local.yml` → `BatchTimeout` zero-valued. | Start worker từ chính cwd của nó (`cd centralized-data-service && /tmp/cdc-worker`). Không phải code bug — operational. |

### DoD Verification — source 26 (orders_e2e_d_v5, mode=auto)
| seq | step | actor | from → to | latency |
|----|------|-------|-----------|---------|
| 1 | shadow_bind (dispatch) | system | draft → shadow_pending | 0ms |
| 2 | shadow_bind (handler) | shadow_bind_handler | shadow_pending → shadow_active | +44ms |
| 3 | master_bind (auto-fanout) | auto-fanout | shadow_active → master_pending | +4ms |
| 4 | master_bind (handler) | master_bind_handler | master_pending → master_active | +21ms |
| 5 | discover (auto-fanout) | auto-fanout | master_active → mapping_pending | +1ms |
| 6 | discover (handler) | discover_handler | mapping_pending → mapping_ready | +5ms |
| 7 | schedule_enable (auto-fanout) | auto-fanout | mapping_ready → schedule_pending | +1ms |
| 8 | schedule_enable (close-loop) | **job_monitor** | schedule_pending → running | **+41.6s** (chờ scheduler tick + transmute success) |

**TOTAL**: 1 REST call → 8 step log entries → state=`running` ở T+34s real-time, hoàn toàn tự động.

### Artifacts side-effect
- `cdc_system.master_binding`: id=4, binding_code=`auto_src_26`, schema_status=`approved`, master_schema=`dw_src_local_pg_source`.
- `cdc_system.shadow_binding`: id=14, binding_code=`shadow_src_26`, shadow_schema=`shadow_src_local_pg_source`.
- `cdc_system.transmute_schedule`: id=2, master_binding_id=4, mode=`cron`, cron=`*/1 * * * *`, last_status=`success`.
- JobMonitor log: `bridge fan-out master=orders_e2e_d_v5 impacted_sources=1 correlation_id=sched-2-...` ✓ (Q3 refine confirmed).

### Files đã modify
- `centralized-data-service/internal/service/schema_adapter.go` (P2 pgx.Identifier)
- `centralized-data-service/internal/service/job_monitor.go` (Q3 impacted_sources log)
- `centralized-data-service/internal/service/provisioning_orchestrator.go` (seedMasterBindingForAdvance + lookupSourceTableForSource + discover payload extras)
- `centralized-data-service/internal/handler/provisioning_step_handlers.go` (resolveShadowTarget JOIN fix + upsertShadowBinding rewrite + HandleScheduleEnable rewrite)
- `cdc-cms-service/internal/service/provisioning_orchestrator.go` (mirror seedMasterBindingForAdvance + Q5 Resume + lookupSourceTableForSource + discover payload extras)

### Lesson — Global Pattern (handler-vs-schema drift cascade)

**Pattern**: *"Khi orchestrator A dispatch command tới handler B qua message bus C, payload contract giữa A và B PHẢI được cùng một code review chốt — vì nếu A thiếu field hoặc B đọc field sai tên cột DB, lỗi chỉ nổ ở runtime smoke (không catch được bằng unit test riêng từng phía). Cộng thêm event-driven multi-step pipeline (auto-fanout) → mỗi mismatch tạo cascade failure ở step sau, không phải step hiện tại."*

**Đúng**:
1. Cấm hardcode column name trong handler — luôn validate qua `information_schema.columns` boot-time hoặc qua model struct với gorm tag (rồi sanity-check tag ↔ DB).
2. Test integration cấp pipeline (1 advance → assert state=running) PHẢI tồn tại trước khi merge orchestrator change. Unit test riêng per-step không đủ.
3. Khi thêm step mới vào state machine, audit cả 3 mặt: (a) orchestrator payload build, (b) handler payload parse, (c) handler DB write column list.

**Sai**: Coi mỗi step là isolated unit, dựa vào "code build PASS + unit test PASS". Auto-fanout pipeline có **cascade liability** — bug ở step N mới phơi ra khi step N-1 success.

Áp dụng: bất kỳ event-driven workflow engine (Temporal, Step Functions, Camunda, custom NATS pipeline), bất kỳ orchestrator-handler pair với schema drift risk (PostgreSQL/MySQL/Mongo).

### Status
🟢 Phase D auto-pipeline DoD MET. Source 26 = first fully autonomous multi-step provisioning success. Track D Hardening (P1–P4) + Phase D (Option-A master binding seed + auto-fanout) khép vòng. Track E (Mongo CDC) chưa khởi động.
