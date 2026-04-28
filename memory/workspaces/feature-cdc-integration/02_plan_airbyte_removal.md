# Plan — Bỏ luồng Airbyte, tận dụng code cho Debezium

> **Date**: 2026-04-21
> **Author**: Muscle (claude-opus-4-7[1m]) — SOP Stage 2 PLAN
> **Scope**: 3 repos — cdc-worker (Go), cms-api (Go), cms-fe (React/TS)
> **Trigger**: v7.2 Parallel System đã prove — Debezium → cdc_internal path production-ready. Airbyte legacy "để tự bơi" policy được approve; giờ tiến thêm 1 bước = **CẮT BỎ luồng Airbyte khỏi admin plane** + refit CMS/FE cho Debezium.

---

## 0. Architecture target sau migration

```
┌─────────────────┐           ┌──────────────────┐
│  Mongo (source) │──Debezium─►  Kafka topics    │
└─────────────────┘           │  cdc.goopay.*    │
                              └────────┬─────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │  SinkWorker      │
                              │  (cmd/sinkworker)│
                              └────────┬─────────┘
                                       │ GORM UPSERT
                                       ▼
                              ┌──────────────────┐
                              │  cdc_internal.*  │  (shadow tables — new home)
                              │  + table_registry│
                              └──────────────────┘
                                       ▲
                                       │ Admin plane
                              ┌──────────────────┐
                              │  CMS API (Fiber) │
                              │  /api/v1/tables  │
                              │  /api/recon/*    │
                              │  /api/system/*   │
                              └────────┬─────────┘
                                       │
                              ┌──────────────────┐
                              │  CMS FE (React)  │
                              │  /cdc-internal   │
                              │  /data-integrity │
                              │  /system-health  │
                              └──────────────────┘
```

**Đi khỏi picture**: `pkgs/airbyte/*`, `/api/airbyte/*`, `/sources` FE page, Airbyte client in worker, sync_engine branching logic ở FE.

**Legacy Airbyte public schema (40 tables)**: KHÔNG xóa — còn docs/analysts query trực tiếp. Chỉ dừng admin-control qua CMS.

---

## 1. Scan summary — 3 repos

### 1.1 cdc-worker (`centralized-data-service`)

| Category | Count | Examples |
|---|---|---|
| AIRBYTE-ONLY (delete) | 1 file | `pkgs/airbyte/client.go` |
| SHARED (refactor) | 6 files | `internal/handler/command_handler.go` (2066 LOC: `HandleAirbyteBridge` L495, `HandleIntrospect` L394, `ensureCDCColumns` hard-coded `_source='airbyte'`), `internal/service/source_router.go` (96 LOC: `ShouldUseAirbyte`), `internal/service/bridge_service.go`, `internal/handler/kafka_consumer.go` (715 LOC — references in comments), `config/config.go` (AirbyteConfig struct), `internal/server/worker_server.go` (import) |
| NAMING-ARTIFACT (rename) | 5 files | `internal/model/cdc_event.go` comment, `internal/model/table_registry.go` AirbyteSourceID/ConnectionID fields, `internal/service/dynamic_mapper.go`, `internal/service/scan_service.go`, `model/schema_change_log.go` constants |
| DEBEZIUM-NATIVE (keep) | 6 files | `cmd/sinkworker/main.go`, `internal/sinkworker/*` (envelope, upsert, sinkworker, schema_manager) — **NO cross-imports với `handler/` hay `pkgs/airbyte/`** ✅ Parallel Independence verified |

### 1.2 cms-api (`cdc-cms-service`)

| Category | Count | Examples |
|---|---|---|
| AIRBYTE-ONLY (delete) | 3 files | `pkgs/airbyte/client.go`, `pkgs/airbyte/reconciliation.go`, `internal/api/airbyte_handler.go` (8 routes) |
| SHARED (refactor) | 6 files | `internal/server/server.go` (5 lines inject airbyteClient), `internal/api/registry_handler.go` (`SyncFromAirbyte` L869, `RefreshCatalog` L549, `GetStatus` L401 airbyte lookup), `internal/router/router.go` (Airbyte route group L79-88), `internal/service/reconciliation_service.go` (airbyteClient field, interval polling), `internal/service/approval_service.go` (airbyteClient in constructor), `internal/service/system_health_collector.go` (Airbyte probe L32) |
| NAMING-ARTIFACT (rename) | 2 files | `internal/model/table_registry.go` legacy Airbyte* fields (keep columns — FK + migration safety; deprecate in code), `config/config.go` AirbyteConfig struct |
| DEBEZIUM-NATIVE (keep) | 4 files | `internal/api/cdc_internal_registry_handler.go` (NEW — từ Phase 2 S4), `internal/api/reconciliation_handler.go` (TriggerSnapshot, ResetDebeziumOffset qua NATS), `internal/api/system_health_handler.go`, middleware + repo layer |

### 1.3 cms-fe (`cdc-cms-web`)

| Category | Count | Examples |
|---|---|---|
| AIRBYTE-ONLY (delete) | 1 page | `src/pages/SourceConnectors.tsx` + menu item `key="sources"` + route `/sources` |
| MIXED (refactor) | 4 files | `src/pages/TableRegistry.tsx` (sync_engine filter L19/191/395-398, airbyteSources fetch L274), `src/pages/ActivityManager.tsx` (airbyte-sync trong ALL_OPERATIONS L27), `src/pages/ActivityLog.tsx` (filters cmd-bridge-airbyte + scan-airbyte-streams L149/151), `src/pages/QueueMonitoring.tsx` (GET /api/airbyte/jobs L64) |
| CDC-NATIVE (keep) | 6 pages | `Dashboard.tsx`, `SchemaChanges.tsx`, `MappingFieldsPage.tsx`, `DataIntegrity.tsx`, `SystemHealth.tsx`, `CDCInternalRegistry.tsx` (NEW from Phase 2 S4) |
| HOOK cleanup | 1 hook | `src/hooks/useRegistry.ts` remove `useSyncAirbyte` L25-31 |
| TYPE cleanup | 1 file | `src/types/index.ts` TableRegistry interface — `airbyte_connection_id/source_id/destination_id` đã nullable → comment deprecation; không remove (back-compat) |

---

## 2. Patterns — cái gì giữ nguyên, cái gì đổi

### 2.1 Patterns giữ nguyên (reuse across Debezium path)

| Pattern | Location | Why keep |
|---|---|---|
| Fiber + JWT + RBAC middleware chain | `cms-api/internal/middleware/*` | Độc lập Airbyte — cấp quyền chung |
| Destructive chain (JWT → RequireOpsAdmin → Idempotency Redis → Audit INSERT) | `cms-api/internal/router/router.go` | Đã dùng cho `/api/v1/tables/:name` PATCH |
| GORM repository pattern | `cms-api/internal/repository/*` | Không đặc trưng Airbyte — `RegistryRepo` + `MappingRuleRepo` dùng chung |
| NATS subject naming `cdc.cmd.*` / `cdc.result.*` / `cdc.event.*` | `worker/internal/handler/*` | Transport pattern — chỉ cần prune các subject airbyte-* |
| Prometheus metrics prefix `cdc_*` | `worker/pkgs/metrics/*` | Continue |
| Viper YAML + env override | `both services/config/config.go` | Continue |
| React Query keys convention `['<resource>']` | `cms-fe/src/hooks/*` | Continue |
| `cmsApi` axios + Idempotency-Key + X-Action-Reason | `cms-fe/src/services/api.ts` + CDCInternalRegistryHandler mutation | Continue — đã dùng cho `/v1/tables` PATCH |
| `useAsyncDispatch` (202 + poll) | `cms-fe/src/hooks/useAsyncDispatch.ts` | Continue cho các destructive admin ops qua NATS |
| AntD Table + Modal + Switch + Form | shared UI kit | Continue |

### 2.2 Patterns cần đổi

| Pattern | Current | Target |
|---|---|---|
| Source routing | `service/source_router.go` với 2 branch | 1 branch Debezium; delete `ShouldUseAirbyte` |
| Health probe | Airbyte + Debezium + Kafka | Debezium + Kafka only |
| Activity log operations | 7 ops (airbyte-sync, cmd-bridge-airbyte, scan-airbyte-streams, ...) | 5 ops (Debezium snapshot, Debezium signal, recon check, heal, backfill) |
| Registry model `sync_engine` enum | `{airbyte, debezium, both}` | `{debezium}` (keep enum for migration, single value) |
| Bootstrapping | Airbyte client inject into 5 constructors | Remove — `airbyteClient` biến mất khỏi DI graph |

---

## 3. Migration execution plan — 5 phases

### Phase R1 — CMS FE prune (low-risk, UI-only, isolated)
**Scope**: Xóa panel Airbyte khỏi sidebar, prune mixed pages. User action visible immediately.

| Task | File | Change |
|---|---|---|
| R1.1 | `src/App.tsx` | DELETE menu item `key="sources"` + route `/sources`; DELETE `lazy import SourceConnectors` |
| R1.2 | `src/pages/SourceConnectors.tsx` | DELETE file |
| R1.3 | `src/pages/TableRegistry.tsx` | DELETE sync_engine filter + airbyteSources fetch + "Refresh Catalog" button; table chỉ list rows với sync_engine='debezium' |
| R1.4 | `src/pages/ActivityManager.tsx` | REMOVE 'airbyte-sync' khỏi ALL_OPERATIONS |
| R1.5 | `src/pages/ActivityLog.tsx` | REMOVE 'cmd-bridge-airbyte' + 'scan-airbyte-streams' khỏi filter options |
| R1.6 | `src/pages/QueueMonitoring.tsx` | DELETE Airbyte jobs widget + useQuery `/api/airbyte/jobs` |
| R1.7 | `src/hooks/useRegistry.ts` | DELETE `useSyncAirbyte` hook |
| R1.8 | `src/types/index.ts` | COMMENT `@deprecated` trên airbyte_* fields của TableRegistry interface |

**Verify**: `tsc --noEmit` 0 error; `npm run build` bundle size giảm; menu sidebar hết 1 item.

**Rollback**: `git revert` — no data change.

**Effort**: 2-3h.

### Phase R2 — CMS API prune (medium-risk, drop Airbyte routes + DI)
**Scope**: Xóa handler Airbyte, refactor registry_handler, cập nhật router + server bootstrap.

| Task | File | Change |
|---|---|---|
| R2.1 | `internal/api/airbyte_handler.go` | DELETE file |
| R2.2 | `internal/router/router.go` | DELETE lines 79-88 (Airbyte route group) + remove `airbyteHandler` param từ SetupRoutes |
| R2.3 | `internal/api/registry_handler.go` | DELETE `SyncFromAirbyte` L869, `RefreshCatalog` L549; refactor `GetStatus` L401 skip airbyte lookup |
| R2.4 | `internal/service/reconciliation_service.go` | Remove `airbyteClient` field; replace Airbyte polling với Debezium connector status check (`GET /connectors/goopay-mongodb-cdc/status`) |
| R2.5 | `internal/service/approval_service.go` | Remove `airbyteClient` từ constructor + callers |
| R2.6 | `internal/service/system_health_collector.go` | Delete Airbyte probe block (L32); keep Debezium + Kafka + kafka-exporter |
| R2.7 | `internal/server/server.go` | Delete 5 lines airbyteClient setup + handler wiring |
| R2.8 | `pkgs/airbyte/` | DELETE directory (client.go + reconciliation.go) |
| R2.9 | `config/config.go` | DELETE `AirbyteConfig` struct + env parse; keep YAML key if any downstream tool peeks (harmless) |
| R2.10 | `internal/model/table_registry.go` | COMMENT `// Deprecated: Airbyte legacy field` on AirbyteSourceID/ConnectionID — giữ cột DB cho migration safety, không dùng trong code mới |

**Verify**: `go build ./... && go vet ./... && go test ./...` PASS. Startup log: "OpenTelemetry initialized" + "server listening :8083" no "airbyte client" log line.

**Rollback**: `git revert` — DB schema untouched. If downstream tool broke, hotfix.

**Effort**: 4-6h.

### Phase R3 — Worker prune (high-risk — touches command dispatch)
**Scope**: Remove Airbyte Bridge + Introspect handlers; refactor source_router.

| Task | File | Change |
|---|---|---|
| R3.1 | `internal/handler/command_handler.go` | DELETE `HandleAirbyteBridge` (L495+) + `HandleIntrospect` (L394+); remove `airbyteClient` field + constructor param; refactor `ensureCDCColumns` dùng config-driven `_source` thay hardcode `'airbyte'` |
| R3.2 | `internal/service/source_router.go` | DELETE `ShouldUseAirbyte`; `ShouldUseDebezium` thành default, có thể đổi tên → `IsDebeziumManaged` |
| R3.3 | `internal/service/bridge_service.go` | DELETE file (chỉ dùng cho Airbyte → CDC bridge) |
| R3.4 | `internal/handler/kafka_consumer.go` (715 LOC) | UPDATE comments — drop references "Airbyte"; giữ consume loop logic vì không dính Debezium (pipeline đó đã có sinkworker riêng) |
| R3.5 | `internal/server/worker_server.go` | DELETE `import pkgs/airbyte`; drop airbyte params từ handler construction |
| R3.6 | `pkgs/airbyte/client.go` | DELETE file (worker copy của SDK) |
| R3.7 | `config/config.go` | Remove `AirbyteConfig` struct |
| R3.8 | `internal/model/cdc_event.go` | UPDATE comment "CloudEvents CDC message from Debezium" (bỏ "Airbyte") |
| R3.9 | `internal/model/table_registry.go` | Mark Airbyte* fields `@deprecated` same as CMS R2.10 |
| R3.10 | NATS subjects | Remove subscriptions for `cdc.cmd.airbyte-*` patterns |

**Verify**: `go build ./... && go vet ./... && go test ./...` PASS; worker startup log "subscribed N NATS subjects" — N giảm tương ứng; `cmd/sinkworker` vẫn up độc lập.

**Rollback**: `git revert` + restart worker. Sinkworker untouched → Debezium path không bị gián đoạn trong rollback.

**Effort**: 6-8h.

### Phase R4 — DB schema deprecation (không destructive — soft deprecation)
**Scope**: Mark Airbyte columns `@deprecated` ở tầng DB, không drop (tránh break analyst queries).

| Task | File | Change |
|---|---|---|
| R4.1 | NEW migration `migrations/XYZ_airbyte_deprecation_markers.sql` | `COMMENT ON COLUMN cdc_table_registry.airbyte_source_id IS 'DEPRECATED 2026-04-21: Airbyte pipeline removed'`; same for airbyte_connection_id, airbyte_destination_id, airbyte_raw_table, airbyte_destination_name |
| R4.2 | OPTIONAL `migrations/XYZ_airbyte_logs_partitioning_retention.sql` | Drop partitions `cdc_activity_log_*` chứa operations `airbyte-*` cũ (giữ 30 ngày cuối để audit); KHÔNG drop các cdc_activity_log records, chỉ clean housekeeping |

**Verify**: `\d cdc_table_registry` show comments; downstream analyst queries vẫn hoạt động.

**Rollback**: `COMMENT IS NULL`.

**Effort**: 1h.

### Phase R5 — Docs + tests cleanup
**Scope**: Update workspace docs, remove Airbyte-specific integration tests.

| Task | Files | Change |
|---|---|---|
| R5.1 | `test/integration/*airbyte*.go` | DELETE integration test files |
| R5.2 | `test/integration/*bridge*.go` | DELETE if Airbyte-specific |
| R5.3 | Workspace `00_context.md` + `tech_stack.md` | APPEND "Airbyte legacy deprecated 2026-04-21; Debezium is sole CDC engine for cdc_internal" |
| R5.4 | Workspace NEW `03_implementation_airbyte_removal.md` | Evidence bundle per-phase (files deleted, lines refactored, test runs) |
| R5.5 | `05_progress.md` | APPEND per-phase entries |

**Effort**: 2h.

---

## 4. Execution ordering & dependencies

```
R1 (FE) ──────────────┐
                      ├──► R4 (DB markers)
R2 (CMS API) ─────────┤
                      │
R3 (Worker) ──────────┘
                              │
                              ▼
                          R5 (Docs + tests)
```

- **R1 can start first** (FE isolated, drops UI dep on `/api/airbyte/*` — backend continues to serve for moment).
- **R2 + R3 parallel after R1 merged** — both can touch their services independently. Dependency: R1 must be deployed first so FE stops calling /api/airbyte/* before CMS removes them.
- **R4 after R2 + R3** — DB marker once code stops reading.
- **R5 continuous**.

---

## 5. Risk + mitigation

| # | Risk | Mitigation |
|---|---|---|
| 1 | Airbyte còn đang sync vào legacy public.* schema → phá vỡ "để tự bơi" contract | KHÔNG đụng Airbyte instance; chỉ xóa CMS admin của nó. User vẫn manage Airbyte qua Airbyte UI port 18000. |
| 2 | Analyst query legacy public.* fail sau khi CMS prune | Không liên quan — CMS không proxy query; analyst dùng psql hoặc BI tool trực tiếp |
| 3 | Migration 017 hoặc earlier có FK constraint tới cdc_table_registry.airbyte_* | Check trước — hiện không có FK, cột nullable → an toàn |
| 4 | Downstream reporting tool đọc `activity_log` có opcode `airbyte-*` | Giữ opcode trong DB, chỉ remove khỏi FE filter |
| 5 | `sync_engine` enum có `both` value → data loss nếu registry row sync_engine='both' | 2 tables active (export_jobs + refund_requests) đều sync_engine='debezium' → an toàn |
| 6 | Worker/CMS restart sau refactor có thể leak stale goroutine Airbyte polling | `reconciliation_service` refactor phải cancel old ticker before remove field |
| 7 | Phase R3 lớn (2066-line command_handler.go) — risk regression của non-Airbyte code | Chia nhỏ commits theo handler; test suite CI pass mỗi commit |

---

## 6. DoD (per phase)

### R1 DoD
- `npm run build` 0 error, `tsc --noEmit` 0 error
- Sidebar không còn item "Source Connectors"
- Vào `/registry` không thấy sync_engine filter, chỉ list Debezium rows
- Vào `/activity-manager` + `/activity-log` không còn filter airbyte-*
- `/queue` không còn Airbyte jobs widget
- Manual smoke: CDCInternalRegistry page vẫn hoạt động

### R2 DoD
- `go build ./... && go vet ./...` 0 error cho cms-api
- `curl localhost:8083/api/airbyte/sources` → **404** (was 200)
- `curl localhost:8083/api/v1/tables` → 200 với `export_jobs + refund_requests`
- Startup log không còn "Airbyte client initialized"
- `/api/system/health` response không còn `airbyte` section (chỉ debezium + kafka)

### R3 DoD
- `go build ./... && go vet ./... && go test ./...` 0 error cho worker
- Worker startup log: subscribe N subjects (N giảm 2-4 so với trước)
- SinkWorker vẫn drain sạch stream events (test: mongosh insert doc → cdc_internal row xuất hiện trong ≤2s)
- NATS subjects `cdc.cmd.airbyte-*` không có subscriber (test `nats pub cdc.cmd.airbyte-bridge` không có consumer)

### R4 DoD
- `SELECT column_name, col_description(...) FROM information_schema.columns WHERE table_name='cdc_table_registry' AND column_name LIKE 'airbyte_%'` — comments hiển thị "DEPRECATED"
- Analyst query mẫu trên public.* không bị affected

### R5 DoD
- Workspace doc NEW + progress log APPENDED
- Integration test airbyte-* removed khỏi `go test ./...` output

---

## 7. Effort + timeline

| Phase | Effort | Engineer |
|---|---|---|
| R1 FE | 2-3h | 1 FE |
| R2 CMS API | 4-6h | 1 BE |
| R3 Worker | 6-8h | 1 BE (ưu tiên senior vì command_handler 2066 LOC) |
| R4 DB | 1h | BE |
| R5 Docs | 2h | BE |
| **Total sequential** | **15-20h** | 2 engineers song song: R1 (FE) + R2/R3 (BE parallel) → 10-12h wall |

---

## 8. Out-of-scope (explicitly deferred)

1. **Airbyte instance shutdown** — vẫn chạy on port 18000 để user tự sync legacy public.*. Sẽ decommission khi consumers migrate xong.
2. **Drop Airbyte-only tables/columns trong DB** — soft deprecation trước (R4); hard drop sau 1 quarter quan sát.
3. **Rename legacy `cdc_table_registry` thành `cdc_legacy_registry`** — bóng lẫn với `cdc_internal.table_registry`, nhưng rename = destructive breaking change; defer.
4. **Auth overhaul** — JWT + RBAC giữ nguyên, không touch.
5. **OTel fiber middleware tracing** — separate initiative, không blocker.

---

## 9. Approval gate

**Chờ user duyệt**:
- Option (A) **Go ahead R1→R5 tuần tự** — Muscle execute full.
- Option (B) **R1 only, pause for review** — FE prune xong, review trước khi R2/R3.
- Option (C) **Mở scope thêm** — R1→R5 + decommission Airbyte instance + drop DB columns (destructive, yêu cầu thêm planning R6/R7).
- Option (D) **Chỉnh plan** — thêm/bớt phase; call out trước.

**Recommend (B)** — R1 FE visible nhất, user verify UX trước khi BE prune sâu. Low-risk incremental.

---

## 10. SOP Stage coverage

| Stage | Status |
|---|---|
| 1 INTAKE | ✅ User order + scan requirements clear |
| 2 PLAN | ✅ Doc này |
| 3-7 | ⏳ Gated on user A/B/C/D |
