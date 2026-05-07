# Report — Flow 1: Connect Source (manual operator path)

> **Author**: max (Brain, Opus 4.7) | **Date**: 2026-05-07 ICT
> **Scope**: Overview các bước manual operator phải làm để đưa 1 source mới vào pipeline CDC, từ "click connect" trên UI → output là **shadow schema + shadow table tự động đầy data từ source**.
> **Method**: Read code thực tế qua Explore agent + Read file:line. KHÔNG guess. Mọi claim có evidence file:line ở dưới.
> **Boundary**: Flow 1 = **Wizard step 1–5** (Connect-Source half). Step 6–11 (Master Layer) là Flow 2 — out of scope.

---

## 0. Bối cảnh

- Operator đăng nhập qua `cdc-cms-web` (auth qua `cdc-auth-service`).
- Wizard FE: `cdc-cms-web/src/pages/SourceToMasterWizard.tsx` — 11 step, Flow 1 = step 1–5 (`SourceToMasterWizard.tsx:51–56`).
- 2 backend chạm:
  - `cdc-cms-service` (Fiber, port 8083) — control-plane proxy + UI gateway.
  - `centralized-data-service/admin-api` — provisioning orchestrator (DB write + Debezium PATCH + NATS publish).
- Source DB hỗ trợ: PostgreSQL (5435), MongoDB (17017), MariaDB (13307). Shadow target: PostgreSQL `cdc-metadata` (5433).
- Connector hardcoded: `goopay-mongodb-cdc` / `cdc-pg-source` / `cdc-mariadb-source` (`centralized-data-service/internal/admin/helpers.go:110–119`).

---

## 1. Step 1 — Tạo Debezium Connector (Wizard step 1)

| Field | Evidence |
|---|---|
| **UI step** | `SourceToMasterWizard.tsx:52` — `'1. Debezium Connector'`, `goto: '/sources'`, `verify: 'Connector state=RUNNING'` |
| **Page** | `cdc-cms-web/src/pages/SourceConnectors.tsx:171` — `cmsApi.get('/api/v1/sources')` list connectors |
| **Action** | Operator nhấn "Create" → POST `/api/v1/system/connectors` (CMS proxy, KHÔNG để FE đụng Kafka Connect direct — lesson L-1154 "FE không bypass auth bằng kafka-connect:8083") |
| **Backend** | `cdc-cms-service/internal/api/system_connectors_handler.go:149–200` Create handler |
| **Bus** | `cdc-cms-service/internal/app/commands/system_connector.go:44` command `system-connector.create` (registered ở `internal/server/server.go:288`) |
| **Outbound** | `cdc-cms-service/internal/infra/http/kafka_connect.go:123` `KafkaConnectClient.Create()` → POST tới Kafka Connect `:18083/connectors` |
| **Output** | Connector active ở Kafka Connect, topic pattern `cdc.<prefix>.<db>.<collection_or_table>` (e.g. `cdc.goopay.payment-bill-service.payment-bills`) |

**Verify**: GET `/api/v1/system/connectors/:name` → `state == "RUNNING"`. Manual eyeball trên UI hoặc `curl :18083/connectors/<name>/status`.

---

## 2. Step 2 — Check Source (connection test) — ⚠️ MANUAL

| Field | Evidence |
|---|---|
| **Status** | **KHÔNG có endpoint test_connection trong codebase**. Grep `TestConnection`/`test-connection`/`ping` trong cả `cdc-cms-service` và `centralized-data-service` — 0 hit business-level. |
| **CMS handler** | `cdc-cms-service/internal/api/sources_handler.go:32–54` chỉ có `List` + `Get`, không có POST test |
| **Implication** | Operator phải tự verify source DB reachable từ network của connector. Connector RUNNING không bằng schema introspection thành công — Debezium sẽ retry âm thầm nếu source DB sai cred / firewall block. |
| **Indirect signal** | GET `/api/v1/system/connectors/:name` trả `tasks[].state == "FAILED"` → tức source unreachable. |

**Recommendation cho Boss**: nếu muốn step này thành "automated", cần thêm endpoint `POST /api/v1/sources/:id/test` chạy `pg_isready` / `mongosh ping` / `mysqladmin ping`. Hiện tại pure manual.

---

## 3. Step 3 — Register Shadow (Wizard step 2)

| Field | Evidence |
|---|---|
| **UI step** | `SourceToMasterWizard.tsx:53` — `'2. Register Shadow'`, `verify: 'Row mới trong /registry, is_table_created=true'` |
| **Endpoint** | POST `/v2/sources/register` (admin-api của `centralized-data-service`) |
| **Handler** | `centralized-data-service/internal/admin/source_register.go:18–105` `handleRegisterSource` — comment khai báo **5 sub-step**: registry insert → debezium extend → schema registry preempt → NATS signal → mark active |
| **Sub-step 1** | `source_register.go:40` `step1InsertRegistry` — Tx INSERT vào 2 bảng: `cdc_system.source_object_registry` (idempotent ON CONFLICT object_code) + `cdc_system.shadow_binding` (`ddl_status='pending'`, `is_active=true`) |
| **Sub-step 2** | `source_register.go:52` `extendDebeziumInclude` — PATCH `database.include.list` + `collection.include.list` qua Kafka Connect `:18083/connectors/<name>/config`. Hai-tier filter (lesson L-1688 Cascade Liability — phải PATCH cả tier database lẫn tier collection). |
| **Sub-step 3** | `source_register.go:73` `preemptSchemaRegistry` — set per-subject `compat=NONE` trước khi Debezium register Avro schema mới (lesson L-1629 "decimal.handling.mode change → Schema Registry block"). |
| **Sub-step 4** | `source_register.go:87` `nats.Publish("cdc.cmd.kafka.refresh-topics", "{}")` — non-fatal signal cho worker reload subscription. |
| **Sub-step 5** | `source_register.go:93–97` UPDATE `provisioning_state='active'` trong `source_object_registry`. |
| **Output** | Row mới `source_object_registry.id=N`, `provisioning_state='active'`. Row `shadow_binding` với `ddl_status='pending'` (chưa có table thực!). |

**Naming pattern**:
- `shadowSchemaFor(req)` → `helpers.go:21–34` → `naming.ShadowSchemaName(db) = "shadow_" + db` (`naming/naming.go:34–36`).
- E.g. source DB `goopay-mongodb` → shadow schema `shadow_goopay-mongodb`.

---

## 4. Step 4 — Shadow DDL: tạo schema + table thực (Wizard step 3)

| Field | Evidence |
|---|---|
| **UI step** | `SourceToMasterWizard.tsx:54` — `'3. Shadow DDL'`, `verify: '\\d shadow_<source_db>.<target> có 8 cols + trigger'` |
| **Trigger** | NATS message `cdc.cmd.shadow.bind` → `HandleShadowBind` |
| **Handler** | `centralized-data-service/internal/handler/provisioning_step_handlers.go:98–175` |
| **Adapter** | `provisioning_step_handlers.go:157` gọi `schemaAdapter.PrepareForCDCInsertWithBusinessCols(schema, table, pk, businessCols)` |
| **Auto-create** | `centralized-data-service/internal/service/schema_adapter.go:132–154` — nếu schema nil → `createShadowTableV1WithCols` (Track D Hardening P2 / Bug #6 — idempotent CREATE TABLE IF NOT EXISTS, plan đã ship per `~/.claude/plans/curried-waddling-spindle.md`). |
| **DDL emitted** | `schema_adapter.go:286` `CREATE SCHEMA IF NOT EXISTS shadow_<db>` + `:314–324` `CREATE TABLE IF NOT EXISTS` với CDC system cols inline |
| **CDC cols thực tế (V1 path)** | `_raw_data JSONB, _source VARCHAR(20) DEFAULT 'airbyte', _synced_at TIMESTAMP, _version BIGINT, _hash VARCHAR(64), _deleted BOOLEAN, _created_at TIMESTAMP, _updated_at TIMESTAMP` |
| **Mongo preflight gate** | `provisioning_step_handlers.go:137–142` — Cascade Liability gate: refuse nếu Mongo source collection empty/missing (tránh phantom `running` source). |
| **Business cols inference** | `provisioning_step_handlers.go:155` `inferSourceColumns(...)` — best-effort clone source schema (Phase Auto Provisioning Feature A); fail → fallback PK-only. |
| **Output** | Schema `shadow_<db>` + table `shadow_<db>.<obj>` tồn tại. Row `shadow_binding.ddl_status='created'` (`source_register.go:302–304` per Explore evidence). |

### ⚠️ Drift cần ghi chú

- **`project_context.md`** spec V2 anchor cols là `_gpay_source_id, _raw_data, _source_ts, _synced_at, _version, _hash, _gpay_deleted` (`Domain Knowledge › Shadow Layer`).
- **Code thực tế V1 path (`schema_adapter.go:314–324`)** dùng tên ngắn `_raw_data, _source, _synced_at, _version, _hash, _deleted` (KHÔNG có `_gpay_source_id` anchor).
- **Phán đoán**: V1 path (legacy) vẫn được wired qua `schema_adapter.go`. V2 path nằm ở `centralized-data-service/internal/sinkworker/schema_manager.go` (per `tech_stack.md › Critical Files`) — tách riêng, dùng cho route khác.
- **Cần verify với Boss**: flow này dùng V1 hay V2? Nếu code spec rõ ràng V2 nhưng V1 path bị gọi → drift cần fix. Nếu intentional split (Mongo→V2, PG→V1) → nên doc rõ hơn.

---

## 5. Step 5 — Snapshot Now + Wait for Ingest (Wizard step 4–5)

### 5a. Snapshot trigger
| Field | Evidence |
|---|---|
| **UI step** | `SourceToMasterWizard.tsx:55` — `'4. Snapshot Now'`, `verify: 'SinkWorker log "shadow upsert"'` |
| **Mechanism** | Debezium incremental snapshot — operator gửi signal qua `connector/<name>/incremental-snapshot` API hoặc INSERT row vào `dbz_signal` table source DB. **Code path không có wrapper rõ ràng — manual SQL/HTTP.** |

### 5b. Sync data → shadow (auto)
| Field | Evidence |
|---|---|
| **UI step** | `SourceToMasterWizard.tsx:56` — `'5. Wait for Ingest'`, `verify: 'COUNT(shadow) > 0'` |
| **Consumer** | `centralized-data-service/internal/handler/kafka_consumer.go:205,338` `KafkaConsumer.Start()` consume loop |
| **Process** | `kafka_consumer.go:391,531` `processMessage` → `eventHandler.HandleRaw(ctx, subject, cdcJSON)` |
| **Event handler** | `centralized-data-service/internal/handler/event_handler.go:51–61` `HandleRaw → processEvent → batchBuffer.Add(record)` |
| **Buffer** | `centralized-data-service/internal/handler/batch_buffer.go:67–96` `BatchBuffer.Add` — flush khi đầy hoặc timeout |
| **Upsert** | `batch_buffer.go:131–143` `batchUpsert` → `schemaAdapter.PrepareForCDCInsertInSchema` (auto-ALTER ADD COLUMN nếu source schema drift) → upsert shadow table |
| **Topic discovery** | `kafka_consumer.go:582–626` `discoverTopics` — filter theo `TopicPrefix` config |
| **Alternative path** | SinkWorker (snapshot mode): `centralized-data-service/internal/sinkworker/sinkworker.go:174,179` `EnsureShadowTableInSchema → upsertWithFencing` |
| **Output** | Rows trong `shadow_<db>.<obj>` với CDC cols populated + business cols populated |

---

## 6. Output Verification (Wizard step 5 verify)

| Check | Evidence |
|---|---|
| **Health endpoint** | `cdc-cms-service/internal/router/router.go:90` GET `/api/system/health` (Redis cache `system_health:snapshot`) |
| **SyncHealth aggregate** | `cdc-cms-service/internal/router/router.go:303` GET `/sync/health` → `RegistryHandler.SyncHealth` (`internal/api/registry_handler.go:432–436`) |
| **Table created flag** | `is_table_created=true` trong `cdc_table_registry` (`cdc-cms-service/internal/infra/persistence/sync_health_read_repo_gorm.go:39`) |
| **Binding status** | `ddl_status='created'` trong `cdc_system.shadow_binding` (`cdc-cms-service/internal/api/source_objects_handler.go:148`) |
| **Row count** | ⚠️ KHÔNG có endpoint trả shadow row count — operator phải `psql` query manual `SELECT count(*) FROM shadow_<db>.<table>` |

---

## 7. Sơ đồ tổng quan

```
┌─────────┐  POST /api/v1/wizard/sessions
│Operator │─────────────────────────────────────┐
│ Browser │                                     │
└────┬────┘                                     ▼
     │ Step 1: Click Create Connector   ┌──────────────────┐
     ├──────────────────────────────────│ cdc-cms-service  │
     │ POST /api/v1/system/connectors   │   (Fiber 8083)   │
     │                                  │                  │
     │                                  │ KafkaConnectClient
     │                                  └────────┬─────────┘
     │                                           │ POST /connectors
     │                                           ▼
     │                                  ┌──────────────────┐
     │                                  │  Kafka Connect   │
     │                                  │     :18083       │
     │                                  └──────────────────┘
     │
     │ Step 2: Check (MANUAL — no API)  
     │
     │ Step 3: POST /v2/sources/register
     │ ┌───────────────────────────────────────────────────┐
     └▶│  centralized-data-service/admin-api               │
       │  handleRegisterSource (5 sub-step):               │
       │   1. INSERT registry + binding (ddl=pending)      │
       │   2. PATCH debezium include.list (2-tier)         │
       │   3. Schema Registry compat=NONE preempt          │
       │   4. NATS publish cdc.cmd.kafka.refresh-topics    │
       │   5. UPDATE provisioning_state=active             │
       └────────────────────┬──────────────────────────────┘
                            │
                            │ NATS cdc.cmd.shadow.bind (Step 4 trigger)
                            ▼
       ┌────────────────────────────────────────────────────┐
       │  ProvisioningStepHandler.HandleShadowBind          │
       │   → preflight Mongo gate (Cascade Liability)       │
       │   → inferSourceColumns (best-effort)               │
       │   → schemaAdapter.PrepareForCDCInsertWithBusinessCols
       │      → CREATE SCHEMA IF NOT EXISTS shadow_<db>     │
       │      → CREATE TABLE IF NOT EXISTS w/ CDC + business cols
       │   → upsert shadow_binding ddl_status=created       │
       │   → emit cdc.evt.step.completed                    │
       └─────────────────────┬──────────────────────────────┘
                             │
                             │ (sau Step 4: Snapshot Now manual)
                             ▼
       ┌────────────────────────────────────────────────────┐
       │  Source DB (PG/Mongo/MariaDB)                       │
       │     ↓ Debezium                                      │
       │  Kafka topic cdc.<prefix>.<db>.<obj>                │
       │     ↓                                               │
       │  KafkaConsumer → processMessage → HandleRaw         │
       │     → BatchBuffer.Add → flush → batchUpsert         │
       │     → schemaAdapter.PrepareForCDCInsertInSchema     │
       │        (auto-ALTER if source schema drifts)         │
       │     → INSERT/UPDATE shadow_<db>.<obj>               │
       └────────────────────────────────────────────────────┘
                             │
                             ▼
       ┌─────────────────────────────────┐
       │  OUTPUT: shadow_<db>.<obj>       │
       │  populated + ddl_status=created  │
       │  + is_table_created=true         │
       └─────────────────────────────────┘
```

---

## 8. Gap / Risk / TODO list (cho Boss decision)

| # | Issue | Severity | Suggestion |
|---|---|---|---|
| G-1 | **Step 2 Check Source pure manual** — không có endpoint test_connection, operator phải tự ping bên ngoài | MEDIUM | Add `POST /api/v1/sources/test` wrapping `pg_isready`/`mongosh ping`/`mysqladmin ping` |
| G-2 | **CDC col naming drift** giữa `project_context.md` (V2 `_gpay_*`) vs `schema_adapter.go:314` (V1 ngắn) | HIGH (doc) | Verify V1/V2 split intent; nếu unified V2 → fix schema_adapter; nếu split → doc rõ |
| G-3 | **Snapshot Now không wrap** — operator phải tự gọi Debezium signal API hoặc INSERT `dbz_signal` | LOW | OK manual (ít dùng); nếu auto cần endpoint `POST /v1/sources/:id/snapshot` |
| G-4 | **Output row-count check pure manual** — không có API trả shadow count | LOW | Optional `GET /api/sources/:id/stats` |
| G-5 | **Sub-step 4 NATS publish non-fatal** — nếu fail, worker không refresh topics → ingest stuck nhưng API trả 200 | MEDIUM | Promote to fatal nếu sau 3 retry vẫn fail, hoặc add health check downstream |
| G-6 | **Cascade Liability gate chỉ Mongo** — PG/MariaDB không có preflight schema check, có thể tạo phantom shadow | MEDIUM | Mở rộng `preflight*Source` cho PG (`pg_class.reltuples`) + MariaDB |

---

## 9. Lessons applicable (đã quét trong `agent/memory/global/lessons.md`)

| Lesson | Applies to | Bước |
|---|---|---|
| L-1154 (FE không bypass Kafka Connect direct) | Architecture | Step 1 |
| L-1629 (Schema Registry compat=NONE preempt) | Sub-step 3 of Step 3 | Step 3 |
| L-1688 (Cascade Liability — 2-tier filter Debezium) | Sub-step 2 of Step 3 | Step 3 |
| L-1179 (Wizard non-destructive vs destructive endpoint split) | Wizard session create | Step 1 wizard bootstrap |
| L-1436 (Track E premise sai — không bịa scope khi brief 1 dòng) | Process | meta — đã tránh trong report này bằng cách read code thực tế |

---

## 10. DoD report

- ✅ Mọi bước có evidence file:line
- ✅ Drift CDC col đã note (V1 vs V2)
- ✅ Manual gap đã note (G-1, G-3, G-4)
- ✅ Sơ đồ ASCII overview
- ✅ Lessons cross-reference
- ❌ Smoke test live chưa chạy (Brain role không restart service — pause Q3 chờ Boss confirm)

---

## 11. Skill / Tool đã dùng (per CLAUDE.md §0)

- **Agent (Explore, very thorough)** — multi-step codebase trace cho 7 step
- **Read** — verify file:line cụ thể (Wizard, source_register, provisioning_step_handlers, schema_adapter)
- **Bash (grep)** — quét lessons.md cho relevant patterns
- **Write** — output này

---

— max (Brain)
