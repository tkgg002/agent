# Plan v2 — Airbyte ➔ Debezium Native Refactor (Command Center + Transmuter + Bridge surgery)

> **Date**: 2026-04-21
> **Author**: Muscle (claude-opus-4-7[1m]) — SOP Stage 2 PLAN, supersedes v1
> **Trigger**: User feedback trên v1 — KHÔNG delete Source panel, mà REFIT thành Command Center; thêm JsonPath mapping rule; phẫu thuật triệt để `HandleAirbyteBridge`
> **Workspace context đã nạp**: `01_requirements_bridge_fix.md`, `01_requirements_data_flow_core.md`, `01_requirements_multi_destination.md`, `01_requirements_raw_data_strategy.md`, `03_implementation_bridge_fix.md`, `09_tasks_solution_bridge_fix.md`, `04_decisions.md`

---

## 0. Executive summary — thay đổi từ v1

| v1 plan | v2 plan |
|---|---|
| DELETE `SourceConnectors.tsx` | **REFIT** → "Debezium Command Center" (Kafka Connect proxy) |
| Mapping rule giữ nguyên | **REFACTOR** schema + Worker: JsonPath support để bóc Shadow→Master |
| Shadow→Master **gap** | **NEW** `TransmuterModule` service — đóng gap |
| `HandleAirbyteBridge` "refactor comments" | **SURGICAL REMOVAL** 180+ LOC + NATS subject + DI wiring |
| 5 phases R1-R5 | 7 phases R0-R6 (R0 schema prep; R6 Transmuter) |

**Bức tranh tổng thể** (v2 target):

```
Mongo ──Debezium──► Kafka ──SinkWorker──► cdc_internal.<shadow>  (JSONB _raw_data, full envelope)
                                                │
                                                │ TransmuterModule reads mapping_rule
                                                │ evaluates JsonPath $.after.<field>
                                                │ type-casts + upserts idempotent
                                                ▼
                                     public.<master>  (typed, queryable by analysts/BI)
                                                ▲
                                                │
                              Admin plane: CMS API (Command Center routes + v1/tables + mapping-rules)
                                                ▲
                                                │
                              FE: /cdc-internal (registry), /sources (Command Center),
                                  /registry/:id/mappings (JsonPath editor)
```

---

## 1. Bối cảnh đã verify (từ scan kỹ hơn v1)

### 1.1 `cdc_mapping_rule` schema hiện tại
File: `cdc-cms-service/internal/model/mapping_rule.go` + `centralized-data-service/internal/model/mapping_rule.go` (mirror).

| Column | Type | Note |
|---|---|---|
| `id` | uint PK | |
| `source_table` | text NOT NULL | e.g. "payment-bills" (hyphen) |
| `source_field` | text NOT NULL | flat field name (e.g. "fee") — **KHÔNG hỗ trợ dot/JsonPath** |
| `target_column` | text NOT NULL | PG col name |
| `data_type` | text NOT NULL | BIGINT / TEXT / JSONB / TIMESTAMP |
| `is_active` | bool default true | soft delete |
| `is_enriched` | bool default false | enrichment flag |
| `is_nullable` | bool default true | |
| `default_value` | *text | |
| `enrichment_function` | *text | Phase 2+ custom transform (deferred) |
| `status` | text default 'approved' | pending\|approved\|rejected (Phase 1.10) |
| `rule_type` | text default 'mapping' | system\|discovered\|mapping |
| `created_at/updated_at/by/notes` | | audit |

**Gap cho Shadow→Master**:
- ❌ Không JsonPath (chỉ flat `source_field`)
- ❌ Không format-aware (Airbyte flat col vs Debezium `$.after.field`)
- ❌ Không versioning — rule mutable không history

### 1.2 Mapping rules USED today

- **Worker `HandleAirbyteBridge`** (`command_handler.go:495-672`, 180 LOC): **KHÔNG** dùng mapping rule. Bridge chỉ `to_jsonb(src) - '_airbyte_raw_id' - '_airbyte_extracted_at' - '_airbyte_meta' - '_airbyte_generation_id'` → pack full JSONB vào `_raw_data`. Target = CDC table, source = Airbyte typed table. NATS subject `cdc.cmd.bridge-airbyte`.
- **Worker `HandleBackfill`** (`command_handler.go:322-378`): **DÙNG** mapping rule — read `_raw_data` JSONB, `buildCastExpr(data_type, source_field)` → UPDATE target column. Đây là **prototype đủ tốt** cho TransmuterModule.
- **Worker `DynamicMapper`** (`internal/service/dynamic_mapper.go:50-100`): `MapData(ctx, targetTable, rawData) → MappedData` hỗ trợ **nested field** (`getNestedField("info.fee")`) + `unwrapMongoTypes()` — **extensible** sang JsonPath qua `tidwall/gjson` (đã imported).
- **CMS CRUD** (`cdc-cms-service/internal/api/mapping_rule_handler.go`): List/Create/Update/Delete + batch PATCH `status` approve/reject.
- **FE** (`cdc-cms-web/src/pages/MappingFieldsPage.tsx:347 LOC`): Table rules + Add/Edit modal + batch approve. Không có JsonPath editor.

### 1.3 `HandleAirbyteBridge` chi tiết (phẫu thuật target)

```
File:    centralized-data-service/internal/handler/command_handler.go
Lines:   495-672 (180 LOC main) + 676-700 bridgeInPlace helper (25 LOC)
NATS:    cdc.cmd.bridge-airbyte (subscribed at wiring time)
Input:   {target_table, airbyte_raw_table, primary_key_field, source_type}
Reads:   public.<airbyte_typed_table>  (Airbyte full_refresh+append/overwrite)
Writes:  public.<cdc_table> (_raw_data JSONB, _source='airbyte', _hash MD5, _synced_at, _version)
SQL:     INSERT ... SELECT ... to_jsonb(src) - '_airbyte_*' AS _raw_data
         ON CONFLICT (source_id|pk) DO UPDATE WHERE target._hash <> EXCLUDED._hash
Side:    registry.last_bridge_at = NOW(); publishResult qua NATS reply subject
```

Helpers phụ thuộc: `ensureCDCColumns()` (ALTER ADD 8 cols `_raw_data/_source/_synced_at/_hash/_version/_created_at/_updated_at/_deleted` vào Airbyte table), `bridgeInPlace()` (same-schema case).

### 1.4 Shadow→Master — **HIỆN KHÔNG CÓ CODE**

Scan 3 repos: không file nào đọc `cdc_internal.<shadow>` rồi ghi `public.<master>`. Đây là GAP sẽ đóng bằng Phase R6 TransmuterModule.

### 1.5 Kafka Connect REST proxy — primitives có sẵn

- `connectCall(ctx, method, url, body)` helper ở `command_handler.go:2034-2066` — 10s timeout + 1 retry 5xx.
- Đã gọi: `GET /connectors`, `PUT /connectors/:name/{pause,resume}`, `POST /connectors/:name/restart`.
- Cần thêm cho Command Center: `GET /connectors/:name/status`, `POST /connectors/:name/tasks/:id/restart`, `GET /connector-plugins`.
- `kafkaConnectURL` đã inject ở `worker_server.go:215` + CMS `config.System.KafkaConnectURL`.

### 1.6 FE SourceConnectors hiện tại (94 LOC, minimal)

`SourceConnectors.tsx` call `axios.get('http://localhost:8083/api/airbyte/sources')` hard-coded (không qua `cmsApi`), render table 5 cols (name/type/database/id/actions → "View in Airbyte"). Tái sử dụng được structure, thay data source + action buttons.

### 1.7 Workspace decisions

- `01_requirements_raw_data_strategy.md`: P0 blocking issue — `_raw_data` miss field mới. **Option D (Debezium Change Stream direct)** được ưu tiên, đã deliver qua Phase 0-2. Giờ `cdc_internal.<shadow>._raw_data` có full envelope 100% → GAP R1 đóng.
- `01_requirements_multi_destination.md`: R3 — registry hỗ trợ nhiều destination/connection. Dùng được với Debezium qua `cdc_internal.table_registry` + target_table = multi row per source (future).
- `01_requirements_bridge_fix.md`: R4 gap "source_table hyphen vs target_table underscore" — còn open, giải trong R6 Transmuter (hardcoded normalization).

---

## 2. Bảng thống kê Tái cấu trúc (sync với user table)

| Thành phần | Hiện trạng | Target Debezium Native | Action | Phase |
|---|---|---|---|---|
| **FE: Source Connectors** | `/api/airbyte/sources` axios hardcoded | `/api/v1/system/connectors` (Kafka Connect proxy) | REFIT | R2 |
| **FE: Mapping UI** | Flat source_field input | JsonPath `$.after.<field>` input + preview | REFIT | R6 |
| **CMS: Source Proxy** | `airbyte_handler.go` 8 routes | `system_connectors_handler.go` NEW | REPLACE | R2 |
| **CMS: Registry Handler** | Airbyte connection ID CRUD | Keep `cdc_internal_registry_handler.go` (từ Phase 2 S4) | PRUNE | R3 |
| **CMS: Approval/Recon service** | airbyteClient poll | Drop polling, Debezium connector status probe | REFIT | R3 |
| **Worker: Bridge** | `HandleAirbyteBridge` 180 LOC + bridgeInPlace | **REMOVED** — thay bằng TransmuterModule | REPLACE | R4 |
| **Worker: Source Router** | `ShouldUseAirbyte`/`ShouldUseDebezium` | Single-branch (Debezium) | PRUNE | R4 |
| **Worker: Kafka Consumer** | 715 LOC Airbyte refs trong comment | Clean comments + drop Airbyte subjects | REFIT | R4 |
| **Worker: Transmuter (NEW)** | None | `internal/service/transmuter.go` + tests + NATS handler | CREATE | R6 |
| **DB: mapping_rule schema** | flat source_field | +jsonpath +source_format +version | MIGRATE | R0 |
| **DB: Registry legacy cols** | airbyte_source_id/connection_id nullable | `COMMENT ON COLUMN ... DEPRECATED` | DEPRECATE | R5 |
| **Activity log ops** | cmd-bridge-airbyte, scan-airbyte-streams | cmd-transmute, cmd-connector-restart | REFIT | R4+R6 |

---

## 3. Phased execution — R0 → R6

### Phase R0 — DB migration: mapping_rule JsonPath support (prerequisite)

**File**: `cdc-cms-service/migrations/020_mapping_rule_jsonpath.sql` (NEW)

```sql
-- Phase R0: JsonPath + source-format awareness cho cdc_mapping_rule
ALTER TABLE cdc_mapping_rule
  ADD COLUMN IF NOT EXISTS source_format TEXT NOT NULL DEFAULT 'debezium_after'
    CHECK (source_format IN ('debezium_after', 'debezium_full_envelope', 'airbyte_flat', 'raw_jsonb')),
  ADD COLUMN IF NOT EXISTS jsonpath TEXT NULL,
  ADD COLUMN IF NOT EXISTS transform_fn TEXT NULL, -- 'mongo_date_ms', 'oid_to_hex', 'lowercase', ...
  ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS previous_version_id INTEGER NULL
    REFERENCES cdc_mapping_rule(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS approved_by_admin BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ NULL;

-- Index cho TransmuterModule lookup nhanh
CREATE INDEX IF NOT EXISTS idx_mapping_rule_active_sorted
  ON cdc_mapping_rule (source_table, status, is_active)
  WHERE is_active = true AND status = 'approved';

COMMENT ON COLUMN cdc_mapping_rule.jsonpath IS
  'gjson-compatible path to extract value from _raw_data. E.g. "after.fee" for Debezium, "." for Airbyte flat. Takes precedence over source_field when non-null.';
COMMENT ON COLUMN cdc_mapping_rule.source_format IS
  'Anchor format: debezium_after reads _raw_data.after.<path>; debezium_full_envelope reads _raw_data.<path>; airbyte_flat reads column directly; raw_jsonb reads _raw_data.<path>';
COMMENT ON COLUMN cdc_mapping_rule.transform_fn IS
  'Optional transform applied after extraction. Whitelist only — see transmuter.go transformRegistry.';
```

**DoD**: migration apply xong, existing rules có `source_format='debezium_after'`, `jsonpath=NULL` — backward compat (fall back to `source_field`). `\d+ cdc_mapping_rule` show 5 new cols + check constraint.

**Rollback**: `ALTER TABLE DROP COLUMN IF EXISTS ...` — an toàn vì default values.

**Effort**: 30 min.

---

### Phase R1 — Worker core: Transmuter skeleton + JsonPath eval (foundation for R6)

**Files NEW**:
- `centralized-data-service/internal/service/transmuter.go` (~300 LOC)
- `centralized-data-service/internal/service/transmuter_test.go` (~200 LOC unit tests)
- `centralized-data-service/internal/service/transform_registry.go` (~100 LOC whitelist transform fn)

**Logic TransmuterModule**:
1. Input: `shadow_table` + batch of `_raw_data` rows (JSONB) + primary key col
2. Lookup `cdc_mapping_rule WHERE source_table = <shadow> AND status='approved' AND is_active=true`
3. For mỗi row:
   - For mỗi rule:
     - Nếu `jsonpath NOT NULL`: `gjson.Get(rawDataStr, rule.jsonpath)` (tidwall/gjson)
     - Else (legacy flat): `gjson.Get(rawDataStr, rule.source_format == 'debezium_after' ? "after." + rule.source_field : rule.source_field)`
     - Nếu `transform_fn`: `transformRegistry[transform_fn](value)` (mongo_date_ms → timestamp; oid_to_hex → string)
     - Type-cast theo `data_type`
   - Build `record[target_column] = value`
4. UPSERT `public.<master_table>` với `ON CONFLICT (pk) DO UPDATE WHERE target._hash <> EXCLUDED._hash` — **idempotent**
5. Emit metric `cdc_transmute_rows_total{table, outcome}`, `cdc_transmute_rule_miss_total{table, field}`

**Whitelist `transformRegistry`**:
| Name | Input | Output | Use case |
|---|---|---|---|
| `mongo_date_ms` | `{"$date": 1776...}` | `time.Time` | Mongo Extended JSON date |
| `oid_to_hex` | `{"$oid": "abc..."}` | `string` | Mongo ObjectID |
| `lowercase` | string | string | normalization |
| `bigint_str` | number or string | int64 | safe cast |
| `jsonb_passthrough` | any | string (JSON) | store as JSONB |

**DoD**:
- `go build ./...` PASS
- `go test ./internal/service/transmuter_test.go -count=1` PASS — covers: nested path, missing key, transform chain, idempotent upsert (run twice same hash → 0 update)
- No TCP/DB side effect (tests use sqlmock + in-memory rule fixtures)

**Effort**: 6h.

---

### Phase R2 — CMS Debezium Command Center + FE refit

#### R2.1 CMS new proxy handler

**File NEW**: `cdc-cms-service/internal/api/system_connectors_handler.go` (~220 LOC)

Routes (mount tại `router.go`):
- `GET  /api/v1/system/connectors` — list all connectors + per-task status + consumer lag snapshot
- `GET  /api/v1/system/connectors/:name` — detail: config + tasks + 10 recent log events
- `POST /api/v1/system/connectors/:name/restart` — full connector restart (destructive chain)
- `POST /api/v1/system/connectors/:name/tasks/:taskId/restart` — per-task restart (destructive chain)
- `POST /api/v1/system/connectors/:name/pause` — pause (destructive)
- `POST /api/v1/system/connectors/:name/resume` — resume (destructive)
- `GET  /api/v1/system/connector-plugins` — available plugin types (read-only)

Implementation:
- Reuse HTTP helper pattern `connectCall` — copy từ worker hoặc tạo `pkgs/kafkaconnect/client.go` shared nếu team OK.
- Injected `kafkaConnectURL string` vào handler constructor.
- Consumer lag: query `kafkaExporterURL` (đã có từ Phase 1 kafka-exporter sidecar) + aggregate per-topic.
- Response shape: `{connectors: [{name, state, type, tasks: [{id, state, worker_id, trace}], lag_total, last_event}], ...}`.

#### R2.2 CMS router wiring

`internal/router/router.go`:
- Add handler param `systemConnectorsHandler *api.SystemConnectorsHandler`
- Mount read routes trên `shared` group; write routes qua `registerDestructive` chain
- Delete Airbyte route group (L79-88)

#### R2.3 CMS server.go DI

`internal/server/server.go`:
- Delete `airbyteClient :=` line + handler injections (5 lines)
- Add `systemConnectorsHandler := api.NewSystemConnectorsHandler(cfg.System.KafkaConnectURL, cfg.System.KafkaExporterURL, logger)`
- Pass to `router.SetupRoutes`

#### R2.4 CMS cleanup
- DELETE `pkgs/airbyte/` directory
- DELETE `internal/api/airbyte_handler.go`
- DELETE `internal/api/registry_handler.go::SyncFromAirbyte` + `RefreshCatalog` (L869, L549)
- Refactor `internal/service/reconciliation_service.go` — drop `airbyteClient` field, replace interval polling với Debezium connector status check (call TransmuterModule metrics)
- Refactor `internal/service/approval_service.go` — drop airbyteClient
- Refactor `internal/service/system_health_collector.go` — drop Airbyte probe, add task-level Debezium probe

#### R2.5 FE SourceConnectors REFIT → Command Center

`cdc-cms-web/src/pages/SourceConnectors.tsx` (rewrite từ 94 LOC → ~260 LOC):
- Rename page title "Debezium Command Center"
- Fetch `/api/v1/system/connectors` via `cmsApi` (thay axios hardcoded)
- Table columns: Name | Type | Status (Tag running/failed/paused) | Tasks count | Consumer lag | Actions
- Expandable row: task list với per-task status + restart button
- Top bar: overall health summary card + "Restart all failed tasks" button (destructive confirm modal)
- Use `useAsyncDispatch` cho restart mutations (202+poll pattern đã có)
- Reuse `ReDetectButton` component pattern for reason-required modal

Keep route `/sources` + menu `Menu.Item key="sources"` — rename label "Source Connectors" → "Debezium Command Center".

**DoD R2**:
- `go build ./... && go vet ./...` cms-api PASS
- `tsc --noEmit` cms-fe PASS
- `curl localhost:8083/api/v1/system/connectors` → 200 với 1 connector `goopay-mongodb-cdc` running
- `curl -X POST localhost:8083/api/v1/system/connectors/goopay-mongodb-cdc/restart` (với JWT ops-admin) → 202 + audit row
- Kill task 0 in Kafka Connect → FE shows red Tag → click restart → returns green
- `curl localhost:8083/api/airbyte/sources` → **404** (removed)
- `/sources` page render Command Center UI end-to-end
- System health `/api/system/health` payload không còn `airbyte` section

**Effort**: 8h (CMS backend ~4h + FE refit ~3h + testing ~1h).

---

### Phase R3 — CMS Registry + Approval service DI prune

**Files**:
- `internal/api/registry_handler.go` — drop Airbyte-specific methods
- `internal/service/reconciliation_service.go` — refactor
- `internal/service/approval_service.go` — refactor
- `internal/model/table_registry.go` — mark `@deprecated` on `AirbyteSourceID/ConnectionID/DestinationID/RawTable/DestinationName`
- `config/config.go` — remove `AirbyteConfig` struct

**DoD R3**:
- `go build ./... && go vet ./... && go test ./...` PASS
- CMS startup log không còn "Airbyte client initialized"
- `/api/registry` return legacy rows with airbyte_* fields as `null` (not missing — schema intact)

**Effort**: 4h.

---

### Phase R4 — Worker bridge surgery + source router prune

#### R4.1 Delete `HandleAirbyteBridge` + helpers

`centralized-data-service/internal/handler/command_handler.go`:
- DELETE `HandleAirbyteBridge` method L495-672 (180 LOC)
- DELETE `bridgeInPlace` helper L676-700 (25 LOC)
- DELETE `ensureCDCColumns` method (nếu chỉ dùng bởi bridge) — check callers first
- DELETE `airbyteClient` field + constructor param
- DROP NATS subject subscription `cdc.cmd.bridge-airbyte` — remove from `worker_server.go` subscribe wiring

#### R4.2 Delete `source_router.go::ShouldUseAirbyte`

`internal/service/source_router.go`:
- DELETE `ShouldUseAirbyte(string) bool` function
- Rename `ShouldUseDebezium` → `IsCDCManaged` hoặc inline — review single-caller
- Remove file entirely nếu Phase 2 `cdc_internal.table_registry` đã canonical

#### R4.3 Delete `bridge_service.go`

Full file DELETE nếu chỉ dùng bởi HandleAirbyteBridge.

#### R4.4 Kafka consumer cleanup

`internal/handler/kafka_consumer.go` (715 LOC): Scan comment references "airbyte" → reword "CDC event processing"; check nếu còn dispatch subject airbyte-* → drop.

#### R4.5 Worker bootstrap

`internal/server/worker_server.go`:
- DELETE `import pkgs/airbyte`
- DELETE `airbyteClient := airbyte.NewClient(...)` 
- DELETE injections vào `NewCommandHandler`/`NewEventHandler`/`NewRegistryService` (5 sites)
- REMOVE `pkgs/airbyte/client.go` file

#### R4.6 Config

`config/config.go`: DELETE `AirbyteConfig` struct + env parsing.

**DoD R4**:
- `go build ./... && go vet ./... && go test ./...` worker PASS
- `nats pub cdc.cmd.bridge-airbyte <payload>` → no consumer (test: `nats sub cdc.cmd.bridge-airbyte` times out)
- Worker startup log: N-3 subjects subscribed (lost airbyte-bridge + introspect + scan-airbyte-streams)
- SinkWorker unaffected — streams events như trước
- Test: `rg -i airbyte internal/ cmd/worker/ --type go` → chỉ còn NAMING comments (no functional references)

**Effort**: 6h (command_handler surgery là concentration risk — senior eng preferred).

---

### Phase R5 — DB deprecation markers + legacy log retention

**Files**:
- NEW `cdc-cms-service/migrations/021_airbyte_deprecation_comments.sql`:

```sql
COMMENT ON COLUMN cdc_table_registry.airbyte_source_id IS 'DEPRECATED 2026-04-21 — Airbyte pipeline removed per v7.2 parallel system. Retained for back-compat of legacy public.* schema queries.';
COMMENT ON COLUMN cdc_table_registry.airbyte_connection_id IS 'DEPRECATED 2026-04-21';
COMMENT ON COLUMN cdc_table_registry.airbyte_destination_id IS 'DEPRECATED 2026-04-21';
COMMENT ON COLUMN cdc_table_registry.airbyte_destination_name IS 'DEPRECATED 2026-04-21';
COMMENT ON COLUMN cdc_table_registry.airbyte_raw_table IS 'DEPRECATED 2026-04-21';
```

- OPTIONAL `migrations/022_airbyte_activity_log_archive.sql`: Move old `cdc_activity_log_*` partitions với operations `airbyte-*` sang archive schema (nếu DBA OK) — defer R5.2 subject.

**DoD**:
- `\d+ cdc_table_registry` show DEPRECATED comments
- Analyst BI query trên `public.*` không affected (read path untouched)

**Effort**: 1h.

---

### Phase R6 — Transmuter wiring (Shadow → Master) + FE JsonPath editor

#### R6.1 Worker NATS handler + mapping_rule consumer

**Files NEW**:
- `centralized-data-service/internal/handler/transmute_handler.go` (~150 LOC)
  - Subscribe `cdc.cmd.transmute` — input: `{shadow_table, master_table, batch_size, since_id}`
  - Subscribe `cdc.cmd.transmute-all` — iterate all `cdc_internal.table_registry WHERE profile_status='active'`
  - Inject TransmuterModule from R1
  - Publish result `cdc.result.transmute` with stats

**Files MODIFY**:
- `internal/server/worker_server.go` — wire transmuteHandler + subject subscriptions

#### R6.2 Automatic trigger from SinkWorker (optional, Phase R6.2)

Post-ingest hook: after SinkWorker upsert to `cdc_internal.<shadow>`, enqueue NATS message `cdc.cmd.transmute` với `{shadow_table, source_ids: [<just inserted>]}` cho incremental refresh. Implementation trong `sinkworker/sinkworker.go::HandleMessage` — append NATS publish ở cuối (không block upsert path).

#### R6.3 CMS handler — admin trigger + preview

**File NEW**: `cdc-cms-service/internal/api/transmute_handler.go` (~180 LOC):
- `POST /api/v1/tables/:name/transmute` (destructive) — publish NATS `cdc.cmd.transmute`
- `POST /api/v1/mapping-rules/:id/preview` (shared) — evaluate rule against sample shadow row, return extracted value without write
- `GET  /api/v1/tables/:name/transmute-status` — aggregate metrics from Prometheus

#### R6.4 FE — MappingFieldsPage extend

`cdc-cms-web/src/pages/MappingFieldsPage.tsx`:
- Add column "Source Format" (Select: debezium_after | debezium_full_envelope | airbyte_flat | raw_jsonb)
- Add column "JsonPath" (Input — lazy-validated on save via preview endpoint)
- Add "Preview" button per rule → modal shows sample extracted value from 3 random shadow rows
- Add "Transmute Now" button (destructive confirm) at page top → call R6.3 POST endpoint

Components NEW:
- `cdc-cms-web/src/components/JsonPathInput.tsx` — Input + live validation via debounced preview API call
- `cdc-cms-web/src/components/TransmutePreviewModal.tsx` — shows 3 sample rows + extracted values table

#### R6.5 Unit + integration tests

- `internal/service/transmuter_test.go` (R1) — extend với 10+ JsonPath cases: nested, array index, mongo_date transform, missing path, invalid JSON
- `internal/handler/transmute_handler_test.go` NEW — NATS mock + DB sqlmock
- E2E: insert 1 doc into Mongo → Debezium captures → SinkWorker writes cdc_internal → Transmuter writes public.master → verify row exists with correct typed columns

**DoD R6**:
- All tests PASS
- Manual E2E: `mongosh insertOne refund-requests {amount: 99999, fee: 200}` → in 3s, `SELECT * FROM public.refund_requests_master WHERE amount=99999` returns row with fee=200 correctly typed BIGINT
- FE preview modal shows correct extraction
- `/api/v1/tables/refund_requests/transmute` returns 202 + audit row
- Metric `cdc_transmute_rows_total{outcome=success}` > 0

**Effort**: 10h (worker 4h + CMS 2h + FE 3h + e2e testing 1h).

---

## 4. Sequencing + critical path

```
R0 (DB migration)
 ├─► R1 (Transmuter core)
 │    └─► R6 (wiring + FE editor)
 └─► R2 (CMS Command Center) ──┐
                               ├─► R5 (DB deprecation)
R3 (CMS DI prune) ─────────────┤
R4 (Worker bridge surgery) ────┘
```

**Order recommendation**:
1. R0 first — unblocks everything.
2. R1 parallel với R2 (independent).
3. R3 + R4 có thể parallel (khác service, khác scope).
4. R6 sau R1 (depends on Transmuter core).
5. R5 cuối cùng (DB comment once code không reference).

**Total effort**: ~36h sequential. **Wall time**: ~22-26h với 2 engineer parallel (FE+BE).

---

## 5. Rủi ro + mitigation

| # | Risk | Severity | Mitigation |
|---|---|---|---|
| 1 | TransmuterModule hash mismatch với legacy bridge hash → duplicate rows trong master | HIGH | TransmuterModule use canonical JSON hash (same alg `sha256hex(canonicalJSON(envelope))` as SinkWorker từ Phase 1). Cut-over contract: truncate master + fresh transmute per table |
| 2 | JsonPath typo trong rule ghi nullvalue vào master column NOT NULL | MED | R6.3 preview endpoint required before Apply; FE disable save button until preview passes |
| 3 | `HandleAirbyteBridge` removal break in-flight NATS messages | LOW | `nats sub cdc.cmd.bridge-airbyte` verify empty trong 10 phút trước delete |
| 4 | Kafka Connect REST proxy miss auth (exposed admin ops) | HIGH | Destructive chain (JWT + ops-admin + Idempotency + Audit) trên mọi POST route |
| 5 | Transmuter rate — 10M+ rows batch → PG lock contention | HIGH | Default batch_size=1000; per-table rate limit `cdc_transmute_rate_limit` table; can be tuned per-table |
| 6 | Idempotent key collision nếu 2 SinkWorker instances emit same source_id | NONE | Phase 0 fencing token + cdc_internal unique constraint trên `_gpay_source_id` prevents double-write at shadow layer; master derives from shadow |
| 7 | Dynamic JsonPath injection attack via mapping rule field | MED | `gjson.Valid` check + path whitelist regex `^[a-zA-Z_][a-zA-Z0-9_.\[\]]*$` tại R6.3 save endpoint |
| 8 | Legacy analyst query on `public.<cdc_table>._source='airbyte'` breaks after bridge removal | LOW | Migration R5 không drop cột `_source` → existing rows giữ `_source='airbyte'`; new rows (qua Transmuter) `_source='debezium-transmute'` |

---

## 6. DoD tổng quan Phase 3 (release gate)

- [ ] R0 migration applied; 5 new cols trên cdc_mapping_rule
- [ ] R1 TransmuterModule unit tests 100% pass
- [ ] R2 CMS endpoints `/api/v1/system/connectors*` live; FE Command Center render + restart-task button work
- [ ] R3 CMS startup không còn Airbyte client; `/api/airbyte/*` → 404
- [ ] R4 Worker bridge removed; `HandleAirbyteBridge` grep 0 results
- [ ] R5 DB comments DEPRECATED trên 5 cols
- [ ] R6 E2E mongosh → Mongo → Debezium → SinkWorker → cdc_internal → Transmuter → public.<master> working
- [ ] Security self-review Rule 8 PASS (JWT/RBAC/idempotency/audit/path validation)
- [ ] OTel trace coverage cho Transmute flow (optional polish)
- [ ] Workspace docs: NEW `03_implementation_airbyte_removal_v2.md` + APPEND `05_progress.md`

---

## 7. Out-of-scope (confirmed defer)

1. **Shutdown Airbyte instance** — Airbyte still runs on :18000 cho user sync legacy public.* manually. Sẽ decommission sau khi consumers migrate sang transmuted master.
2. **Hard DROP columns** `airbyte_*` trong cdc_table_registry — chờ 1 quarter observation.
3. **Rename `cdc_table_registry` → `cdc_legacy_registry`** — confusing với `cdc_internal.table_registry` nhưng destructive; defer.
4. **OTel fiber middleware** — separate initiative.
5. **Multi-destination support** (per `01_requirements_multi_destination.md`) — Transmuter R6 có thể extend với `master_table` field trong rule để 1 shadow → N masters. Scope v2.1 next.

---

## 8. User's JsonPath sample offer

User đề nghị: "Bạn có muốn tôi liệt kê chi tiết các JsonPath mẫu cho bảng refund_requests?"

**Muscle trả lời**: **YES — xin ngay tại Stage 3 EXECUTE**. Muscle sẽ seed migration R0 với sample rules. Format mong muốn:

```yaml
table: refund_requests
master_table: refund_requests_master
rules:
  - target_column: amount
    jsonpath: after.amount
    data_type: BIGINT
    transform_fn: bigint_str
  - target_column: refund_reason
    jsonpath: after.refundReason
    data_type: TEXT
  - target_column: created_at
    jsonpath: after.createdAt.$date
    data_type: TIMESTAMPTZ
    transform_fn: mongo_date_ms
  - ...
```

---

## 9. Approval gate (SOP Stage 2 exit)

Chờ user + Architect review. Option:

- **(A)** OK phase sequencing R0→R6; Muscle start R0 (migration 020) + R1 (transmuter core) parallel. Sample JsonPath list sau khi R0 apply.
- **(B)** OK plan nhưng chia nhỏ: chỉ R0+R1 round này; R2-R6 review sau.
- **(C)** Chỉnh phase ordering hoặc cắt bớt scope (ví dụ skip R6.2 auto-trigger, giữ manual only).
- **(D)** Hỏi thêm clarification.

Muscle **KHÔNG execute** đến khi user duyệt. SOP Stage 2 vẫn mở.

---

## 10. SOP Stage coverage

| Stage | Status |
|---|---|
| 1 INTAKE | ✅ v2 feedback absorbed: Command Center (keep+refit), JsonPath mapping, bridge surgery triệt để |
| 2 PLAN | ✅ This doc (supersedes v1) |
| 3-7 | ⏳ Gated on user A/B/C/D |

---

## 11. Insights từ sample record `payment_bills` — APPENDED 2026-04-21

> User đưa 1 record thật (30 top-level keys) → xem `01_requirements_mapping_rule_payment_bills_sample.md`. Rút 15 insights dưới đây — update hard-decision cho các phase R0/R1/R4/R6.

### 11.1 Bức tranh lớn — tại sao Transmuter **NOT optional**

Record chứa **3 loại shape** không thể cùng lý tưởng trong 1 container:
- **Flat scalar** (21 fields): ideal cho typed column (BIGINT, TEXT, NUMERIC, BOOLEAN, TIMESTAMPTZ)
- **Nested object** (merchant, extraInfo): schema-on-read tạo JSONB column → analyst query `WHERE merchant_email='x'` phải `->>` + full scan → slow
- **Mongo Extended JSON** (`$date`, `$oid`, `$numberLong`, `$numberDecimal`): **CẠM BẪY TYPE PRESERVATION**

→ Shadow (`cdc_internal._raw_data`) = **audit source-of-truth**.
→ Master (typed cols via Transmuter) = **OLAP queryability layer**.
→ Chúng KHÔNG thay thế nhau — cần **CẢ HAI**. Transmuter là cầu elegant duy nhất giữa 2 layer. Plan v2 đã đúng hướng.

### 11.2 Cạm bẫy #1 — Schema-on-read mù quáng tạo cột JSONB cho date

Phase 1 SinkWorker `schema_manager.go::inferSQLType` quyết định type từ Go value. Với `createdAt = {"$date": "2024-07-10T..."}`:
- Go parse → `map[string]any{"$date": "..."}` → inferSQLType returns **`JSONB`** (map case, line 276-277 schema_manager.go).
- User query `WHERE createdAt > '2024-07-01'` → **FAIL** (JSONB không so sánh được date).

**Hậu quả**: SinkWorker tạo `cdc_internal.payment_bills.createdAt JSONB`. Analyst không dùng được. Forces Transmuter để tạo `public.payment_bills_master.created_at TIMESTAMPTZ` với `mongo_date_ms` transform.

**Action cho R1 Transmuter**:
- `transform_fn=mongo_date_ms` phải handle cả 2 shape: ISO string + int ms.
- Unit test (đã có trong sample doc Section 6 #2, #3) — add concrete with real field names.

### 11.3 Cạm bẫy #2 — `_id: 1` integer KHÔNG phải ObjectID

Record có `_id: 1` (Long integer, chắc từ legacy migration) thay vì `ObjectID("abc...")`. Phase 1 `extractSourceID` (`envelope.go::206-228`) hiện cover:
1. `after._id.$oid` (Mongo ObjectID wrapper)
2. `after._id` scalar (string or number)
3. `msgKey` fallback

**Fork check**: case 2 (`after._id` scalar) via `coerceID` line 230-256 — `case float64, int64` returns `strconv.FormatInt`. ✅ **Đã handle**. Nhưng chưa có unit test explicit cho integer `_id` shape.

**Action cho R1 test suite**: add `Test_integer_id` case — record `{"_id": 1, ...}` → `_gpay_source_id="1"`.

### 11.4 Cạm bẫy #3 — Empty object ≠ null ≠ missing

Record có 3 empty objects `reason: {}`, `instrument: {}`, `ewalletInfo: {}`. Phase 1 SinkWorker sẽ:
- First message creates `cdc_internal.payment_bills` với cột `reason JSONB`, `instrument JSONB`, `ewalletInfo JSONB` (inferSQLType map → JSONB).
- Forever store `{}` — 100% rows x 3 cols = **dead storage**.

**Action cho R1 Transmuter + R6 FE**:
- Mapping rule engine **không** thêm cột master cho empty-object-only fields.
- Add `transform_fn: null_if_empty` (đã có whitelist Section 4 của sample doc).
- FE preview modal (R6.4): nếu 3 sample rows đều return `{}` → show warning "Field chỉ chứa empty object — khuyến nghị skip native column, giữ trong _raw_data".

### 11.5 Cạm bẫy #4 — CamelCase Mongo vs snake_case PG

Mongo: `channelID`, `merchantTransId`, `partnerCode`, `lastUpdatedAt`.
Target Postgres convention: `channel_id`, `merchant_trans_id`, `partner_code`, `last_updated_at`.

**Không thể auto-derive** (`channelID` → `channel_id` hay `channel_i_d`?). Rule phải explicit `target_column`. User (Admin) quyết định naming.

**Action cho R6.4 FE**:
- Modal NEW mapping rule: input `target_column` **required** — không auto-suggest; admin type ra để accountability.
- Optional: dropdown "camel → snake converter preview" as hint, không auto-apply.

### 11.6 Cạm bẫy #5 — Currency-dependent precision

Record: `amount: 10000, currency: "VND"`. VND không có decimal → BIGINT đủ. Nhưng nếu row khác `currency: "USD", amount: 123.45` → cần `NUMERIC(20,4)`.

**Action cho R0 migration + R6 UX**:
- Default `data_type` cho money fields = `NUMERIC(20,4)` (covers both). BIGINT chỉ khi admin confident 100% currency duy nhất.
- FE modal: warning tooltip khi select `data_type: BIGINT` cho field có tên regex `amount|fee|balance|price|refund` — gợi ý `NUMERIC(20,4)`.

### 11.7 Cạm bẫy #6 — `fxRate: 1` ambiguous

Integer `1` có thể serialize là Mongo `Int32`, `Int64`, `Double`, hoặc `$numberDecimal`. 5 shapes khả thi trong `_raw_data`.

**Action cho R1 transform_fn `numeric_cast`**:
- Input handling matrix: plain int (`1`) / plain float (`1.0`) / string (`"1"`) / `{"$numberLong":"1"}` / `{"$numberDecimal":"1"}` / `{"$numberInt":"1"}`.
- Try order: `$numberDecimal` → string/float → `$numberLong` → `$numberInt` → plain number → error.
- Return canonical `decimal.Decimal` (nếu `shopspring/decimal` imported; else `*big.Rat`).

### 11.8 Cạm bẫy #7 — Nested naming ambiguity

`merchant.platformClient = null` và `merchant.platformMerchantId = null` — 2 nullable FK fields với naming redundant `platform*` inside `merchant`.

**Action cho R6 rule design**:
- Target column naming convention: `<parent_prefix>_<field>` — e.g. `merchant_platform_client`, `merchant_platform_merchant_id`.
- Rule skip: business query không touch → không cần native col. Giữ trong `_raw_data`.
- Enforce trong R6.4 FE validation: reject `target_column` trùng giữa 2 rules.

### 11.9 Cạm bẫy #8 — Array length variance = JSONB required

`extraInfo.ruleLogs = []` hôm nay, ngày mai `[{...}, {...}, ...]`. Không thể flatten thành native cols.

**Action cho R1 Transmuter**:
- `transform_fn: jsonb_passthrough` + `data_type: JSONB` là lời giải chính thức cho array.
- Query analyst: `jsonb_array_length(extra_rule_logs) > 0`, `jsonb_path_query_array(extra_rule_logs, '$[*].code')`.
- Document pattern trong `03_implementation_airbyte_removal_v2.md` sau khi R6 land.

### 11.10 Cạm bẫy #9 — Triplet timestamps cần index cho BI

Record có `expireTime`, `createdAt`, `lastUpdatedAt` — 3 TIMESTAMPTZ cho payment lifecycle. BI queries:
- SLA breach: `WHERE expire_time < NOW() AND state != 'SUCCESS'`
- Payment age: `NOW() - created_at > INTERVAL '30 days'`
- Stale payment: `last_updated_at < NOW() - INTERVAL '7 days'`

**Action cho R0 migration master table**:
- Create index `CREATE INDEX idx_payment_bills_master_created_at ON public.payment_bills_master(created_at)` + `expire_time` + `last_updated_at`.
- Add partial index `CREATE INDEX idx_active_unsettled ON payment_bills_master(expire_time) WHERE is_delete=false AND state NOT IN ('SUCCESS','REFUNDED')`.

### 11.11 Financial field classification — SAMPLE XÁC THỰC Phase 2 S4

Record có `amount, fee, refundedAmount, fxRate` + `currency: "VND"` → **100% financial table**. Phase 2 S4 registry-driven `is_financial` flag đã đúng approach — admin review trước auto-ALTER.

Nếu có cột mới xuất hiện sau (vd `settlementFee`):
- Với `is_financial=true` (default) → SinkWorker block ALTER, giữ trong `_raw_data.after.settlementFee`.
- Admin review → flip `is_financial=false` qua CMS PATCH → SinkWorker auto-ALTER cột mới.
- Transmuter rule sẽ **không** tự động pick up — admin phải thêm mapping rule explicit (R6.4 FE).

**Validation**: Sample `payment_bills` xác nhận Phase 2 S4 Controller pattern **đúng ở scale production**.

### 11.12 "30 cột" không phải tùy tiện — match production shape

Phase 2 DoD cho `refund_requests` fail vì Mongo local dev chỉ có 11 keys; user expectation "~30 cột" dựa trên model **production**. Sample `payment_bills` có **chính xác 30 top-level keys** → user đã design với master table shape chuẩn production trong đầu.

**Action cho documentation**:
- Update `03_implementation_v3_backfill_and_kafka_config.md` Section 3 (hoặc NEW addendum): note rằng "~30 cột" target **áp dụng per-table production shape, không universal**. `refund_requests` prod có bao nhiêu keys chưa rõ vì Mongo local = seed/test data.
- Recommend: chạy Transmuter test trên `payment_bills` production-mirror data để validate 30-col claim.

### 11.13 Mongoose `__v` — Phase 1 đã cover

Field `__v: 0` (Mongoose version counter) — `shouldSkipBusinessKey` (`sinkworker.go::203-216`) prefix `__v` → skipped. ✅ Không cần action.

### 11.14 UUID fields như FK discovery target

`partnerCode: "d246b80a-6e88-4b9c-b594-015dcb5a5af9"` — UUID v4. Likely FK to `partners` collection in Mongo. Future Transmuter extension:
- `transform_fn: join_lookup` (v2.1, defer) — lookup partner name/type từ partners master.
- For v1 R6: rule chỉ cast TEXT.

**Action cho R6.1+ (out-of-scope note trong plan)**: add `join_lookup` whitelist transform cho multi-collection enrichment. Defer to Phase R7.

### 11.15 Security concern — row-level multi-tenancy

`partnerCode` + `merchant.reference` expose multi-tenant identifier. Nếu BI dashboard shared giữa các partner → row-level security (RLS) cần thiết:
- `CREATE POLICY partner_isolation ON payment_bills_master USING (partner_code = current_setting('app.current_partner')::text)`.
- Shadow đã fenced bởi `tg_fencing_guard` (insert/update guarded). Master **cần tương tự**.

**Action cho R6.1 Transmuter + R5 migration**:
- Migration 021 (R5 deprecation) extend: `ENABLE ROW LEVEL SECURITY` + default policy `USING (true)` (dev-mode), production tighten.
- Document trong Phase R5 DoD: RLS enabled, default permissive. Real tightening defer Phase R8.

---

## 12. Update cho decision matrix (Section 2) dựa insights

| Thành phần | Update từ Section 11 |
|---|---|
| `transform_fn` whitelist | Expand: `mongo_date_ms` handle both shapes (§11.2); `numeric_cast` 6-shape matrix (§11.7); `null_if_empty` primary use (§11.4); `join_lookup` deferred Phase R7 (§11.14) |
| Rule validation R6.4 | camelCase→snake hint only, not auto-apply (§11.5); BIGINT warning cho money fields (§11.6); reject duplicate target_column (§11.8) |
| Master table schema | 3 timestamp indexes + 1 partial active-unsettled index (§11.10); RLS enabled default permissive (§11.15) |
| Empty object behavior | Skip native column, preview modal warn (§11.4) |
| Integer `_id` support | Add unit test `Test_integer_id` (§11.3) |
| Schema-on-read date | Document cạm bẫy JSONB-not-TIMESTAMPTZ, Transmuter bridge required (§11.2) |
| `is_financial` flag | Validate production sample (§11.11) — registry-driven Controller đã right approach |
| "~30 cột" DoD | Reframe "per-table production shape" in progress log (§11.12) |

---

## 13. SOP Stage coverage — updated

| Stage | Status |
|---|---|
| 1 INTAKE | ✅ v2 feedback + sample record insights absorbed |
| 2 PLAN | ✅ This doc + `01_requirements_mapping_rule_payment_bills_sample.md` addendum |
| 3-7 | ⏳ Gated on user A/B/C/D |

---

## 14. Data Type Catalog — precision siết chặt (APPENDED)

> User: "data_type chưa siết hoàn toàn, update thêm để nó chính xác hơn ví dụ `varchar(10)`..."

### 14.1 Whitelist đầy đủ (thay cho TEXT/NUMERIC/BIGINT generic)

| Category | Type spec (ex) | Regex validation | Use case |
|---|---|---|---|
| **Integer** | `SMALLINT` \| `INTEGER` \| `BIGINT` | `^(SMALLINT\|INTEGER\|BIGINT)$` | ID, count, flag int |
| **Decimal** | `NUMERIC(p,s)` (p=1..38, s=0..p) \| `DECIMAL(p,s)` \| `REAL` \| `DOUBLE PRECISION` | `^(NUMERIC\|DECIMAL)\(\d{1,2},\d{1,2}\)$` | Money (20,4), fx rate (10,6) |
| **Character** | `CHAR(n)` \| `VARCHAR(n)` \| `TEXT` (n=1..10485760) | `^(CHAR\|VARCHAR)\(\d{1,8}\)$\|^TEXT$` | Fixed codes CHAR(3); phone VARCHAR(15); unbounded TEXT |
| **Binary** | `BYTEA` | `^BYTEA$` | hash, payload |
| **Boolean** | `BOOLEAN` | `^BOOLEAN$` | flag |
| **Date/Time** | `DATE` \| `TIME` \| `TIMESTAMP` \| `TIMESTAMPTZ` \| `INTERVAL` | `^(DATE\|TIME\|TIMESTAMP\|TIMESTAMPTZ\|INTERVAL)$` | TIMESTAMPTZ mặc định cho event time |
| **JSON** | `JSON` \| `JSONB` | `^JSONB?$` | Array, nested object passthrough |
| **UUID** | `UUID` | `^UUID$` | `partnerCode` UUID v4 từ record sample |
| **Network** | `INET` \| `CIDR` \| `MACADDR` | `^(INET\|CIDR\|MACADDR)$` | IP source audit |
| **Array** | `<BASE>[]` (1-d only) | `^(SMALLINT\|INTEGER\|BIGINT\|TEXT\|UUID)\[\]$` | tag list, role list |
| **Enum** | `ENUM:<enum_name>` | `^ENUM:[a-z_][a-z0-9_]{0,62}$` | reference registered enum (R0 migration table `cdc_internal.enum_types`) |
| **Constraint shorthand** | `<BASE> NOT NULL DEFAULT <expr>` | parsed via sqlparser | Set NOT NULL + default in rule |

### 14.2 Migration R0 update — type validator + enum registry

Thêm vào `migrations/020_mapping_rule_jsonpath.sql`:

```sql
-- 14.2.a: tighten data_type column với CHECK constraint
ALTER TABLE cdc_mapping_rule
  ADD CONSTRAINT mapping_rule_data_type_chk
  CHECK (data_type ~ '^(SMALLINT|INTEGER|BIGINT|REAL|DOUBLE PRECISION|BOOLEAN|DATE|TIME|TIMESTAMP|TIMESTAMPTZ|INTERVAL|JSON|JSONB|UUID|INET|CIDR|MACADDR|BYTEA|TEXT|CHAR\([1-9][0-9]{0,7}\)|VARCHAR\([1-9][0-9]{0,7}\)|NUMERIC\([1-9][0-9]?,[0-9][0-9]?\)|DECIMAL\([1-9][0-9]?,[0-9][0-9]?\)|(SMALLINT|INTEGER|BIGINT|TEXT|UUID)\[\]|ENUM:[a-z_][a-z0-9_]{0,62})$');

-- 14.2.b: NEW table cho named enum types (reference-able từ data_type='ENUM:payment_state')
CREATE TABLE IF NOT EXISTS cdc_internal.enum_types (
  name        TEXT PRIMARY KEY CHECK (name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  values      TEXT[] NOT NULL CHECK (array_length(values, 1) BETWEEN 1 AND 100),
  is_active   BOOLEAN NOT NULL DEFAULT true,
  created_by  TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  description TEXT
);

INSERT INTO cdc_internal.enum_types (name, values, description) VALUES
  ('payment_state', ARRAY['PENDING','SUCCESS','FAILED','CANCELLED','REFUNDED'], 'Payment lifecycle states (derived từ sample record payment_bills.state)'),
  ('api_type', ARRAY['REDIRECT','QR','DIRECT','WEBHOOK'], 'Payment API surface types'),
  ('currency_iso', ARRAY['VND','USD','EUR','JPY','SGD','THB','KRW','CNY'], 'Supported currency codes');
```

### 14.3 Worker-side type resolver

**File NEW**: `centralized-data-service/internal/service/type_resolver.go` (~200 LOC):
- `ResolveSQLType(spec string) (pgType string, goType reflect.Type, err error)` — parse `NUMERIC(20,4)` → return `"NUMERIC(20,4)"`, `decimal.Decimal`.
- `ResolveEnum(name string) ([]string, error)` — lookup `cdc_internal.enum_types` (TTL cache 60s).
- `ValidateValue(value any, spec string) (cast any, violation string)` — e.g. VARCHAR(10) trả `"too long: 15 > 10"` nếu violate; NUMERIC(5,2) trả `"overflow: abs > 999.99"`.

### 14.4 FE modal update

`src/pages/MappingFieldsPage.tsx` (R6.4):
- Replace `data_type` Input với **Cascader**:
  - Level 1: Category (Integer / Decimal / Character / Date / ...)
  - Level 2: Specific type (within category)
  - Level 3 (conditional): Precision params — 2 inputs (p, s) cho NUMERIC; 1 input (n) cho VARCHAR/CHAR
- Preview endpoint returns **type violation report** kèm sample extracted values — shows "3/3 fit VARCHAR(10)" or "1/3 overflow NUMERIC(5,2) (value 12345.67 needs NUMERIC(7,2))".

---

## 15. Execution Modes — Cron + Immediate (APPENDED)

> User: "đảm bảo cơ chế chạy có thể chọn: 1 theo cron, 2 tức thời"

### 15.1 Mode matrix

| Mode | Trigger | Use case | Infra needed |
|---|---|---|---|
| **A. Immediate** | `POST /api/v1/tables/:name/transmute` | Admin one-shot rebuild, debug | Existing R6.3 endpoint + NATS |
| **B. Cron schedule** | Scheduler poll mỗi 60s, match `cron_expr` | Hourly summary, daily rebuild | NEW scheduler (R7) |
| **C. Post-ingest trigger** | SinkWorker emit NATS sau UPSERT | Real-time sync Shadow→Master | R6.2 (đã plan) |
| **D. On-schema-change** | SinkWorker detect new field → proposal pending → admin approve → re-transmute batch | New column backfill | R8 (schema approval) |

### 15.2 NEW migration: `cdc_internal.transmute_schedule`

`migrations/023_transmute_schedule.sql`:

```sql
CREATE TABLE IF NOT EXISTS cdc_internal.transmute_schedule (
  id              BIGSERIAL PRIMARY KEY,
  master_table    TEXT NOT NULL,         -- reference cdc_internal.master_table_registry.name
  mode            TEXT NOT NULL CHECK (mode IN ('immediate','cron','post_ingest')),
  cron_expr       TEXT NULL,             -- 5-field crontab (validated nếu mode='cron')
  last_run_at     TIMESTAMPTZ NULL,
  next_run_at     TIMESTAMPTZ NULL,
  last_status     TEXT NULL CHECK (last_status IS NULL OR last_status IN ('success','failed','running','skipped')),
  last_error      TEXT NULL,
  last_stats      JSONB NULL,            -- {rows_scanned, rows_upserted, rows_skipped, duration_ms}
  is_enabled      BOOLEAN NOT NULL DEFAULT false,  -- default OFF — admin must flip
  created_by      TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (master_table, mode)
);

CREATE INDEX idx_schedule_due
  ON cdc_internal.transmute_schedule(next_run_at)
  WHERE is_enabled = true AND mode = 'cron';
```

### 15.3 Phase R7 — Scheduler service (NEW phase)

**File NEW**: `centralized-data-service/internal/service/transmute_scheduler.go` (~300 LOC):

```
Loop (60s):
  rows = SELECT * FROM cdc_internal.transmute_schedule
         WHERE is_enabled=true AND mode='cron' AND next_run_at <= NOW()
         FOR UPDATE SKIP LOCKED LIMIT 10
  for row in rows:
    UPDATE ... SET last_status='running', last_run_at=NOW()
    publish NATS cdc.cmd.transmute {master_table: row.master_table, triggered_by: 'scheduler'}
    compute next_run_at via cron-parser (robfig/cron)
    UPDATE ... SET next_run_at=..., updated_at=NOW()
```

**Dependencies**: `github.com/robfig/cron/v3` (add to go.mod).

**Fencing**: Scheduler goroutine claims distinct machine_id từ `cdc_internal.worker_registry` → fencing guard prevents duplicate schedule trong multi-instance deploy.

### 15.4 CMS endpoints cho schedule

- `GET /api/v1/schedules` (shared) — list
- `POST /api/v1/schedules` (destructive) — create `{master_table, mode, cron_expr, is_enabled}`
- `PATCH /api/v1/schedules/:id` (destructive) — update (toggle enabled, change cron)
- `POST /api/v1/schedules/:id/run-now` (destructive) — immediate trigger ngoài lịch
- `DELETE /api/v1/schedules/:id` (destructive)

### 15.5 FE — Schedule Management page NEW

`cdc-cms-web/src/pages/TransmuteSchedules.tsx`:
- Table: master_table | mode tag | cron_expr | next_run_at | last_run_at | last_status | is_enabled Switch | Actions (Run now / Edit / Delete)
- Cron input with validator: shows "Next 3 runs: 2026-04-22 00:00, 2026-04-23 00:00, 2026-04-24 00:00"
- "Run now" button = destructive modal reason required
- Live poll every 10s (react-query)

Menu item NEW: `Menu.Item key="schedules"` → `/schedules`.

---

## 16. Data Warehouse: Master Registry (new master from existing source) APPENDED

> User: "có thể tạo 1 table đích mới từ table nguồn có sẵn (kiểu data warehouse)"

### 16.1 Decoupling shadow vs master

Hiện tại plan v2 assume 1 shadow → 1 master. User muốn 1 shadow → **N masters** (warehouse pattern):
- `cdc_internal.payment_bills` (shadow)
  → `public.payment_bills_master` (full row 1:1 per sample record)
  → `public.payment_bills_daily_summary` (GROUP BY date(created_at), sum(amount))
  → `public.payment_bills_merchant_stats` (GROUP BY merchant_reference)
  → `public.payment_bills_bank_analytics` (WHERE channel_id='BANK_TRANSFER')

### 16.2 NEW registry table

`migrations/024_master_table_registry.sql`:

```sql
CREATE TABLE IF NOT EXISTS cdc_internal.master_table_registry (
  id              BIGSERIAL PRIMARY KEY,
  master_name     TEXT NOT NULL UNIQUE CHECK (master_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  source_shadow   TEXT NOT NULL,   -- FK cdc_internal.table_registry.target_table
  transform_type  TEXT NOT NULL CHECK (transform_type IN ('copy_1_to_1','filter','aggregate','group_by','join')),
  spec            JSONB NOT NULL,  -- transform spec (see §16.3)
  is_active       BOOLEAN NOT NULL DEFAULT false,
  schema_status   TEXT NOT NULL DEFAULT 'pending_review'
                    CHECK (schema_status IN ('pending_review','approved','rejected','failed')),
  schema_reviewed_by TEXT,
  schema_reviewed_at TIMESTAMPTZ,
  rejection_reason   TEXT,
  created_by      TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  FOREIGN KEY (source_shadow) REFERENCES cdc_internal.table_registry(target_table)
);

CREATE INDEX idx_master_active ON cdc_internal.master_table_registry(source_shadow, is_active);
```

### 16.3 `spec` JSONB shape — 5 transform types

| `transform_type` | `spec` example | What Transmuter does |
|---|---|---|
| `copy_1_to_1` | `{"pk":"_gpay_source_id"}` | INSERT per shadow row, use mapping_rule to select cols |
| `filter` | `{"where":"after.state='SUCCESS'","pk":"_gpay_source_id"}` | Same as copy but filters rows |
| `aggregate` | `{"agg":[{"fn":"sum","src":"after.amount","target":"total_amount"},{"fn":"count","target":"row_count"}], "group_by":["created_at::date"], "window":"daily"}` | SELECT + GROUP BY into master |
| `group_by` | `{"group_by":["after.merchant.reference"], "select":["merchant_email","merchant_type"]}` | DISTINCT rows per group key |
| `join` | `{"join":[{"master":"partners_master","on":"partner_code"}], "select":[...]}` | Enrich from other master (deferred to Phase R9) |

### 16.4 Phase R8 — Master Registry handlers

**CMS** (`cdc-cms-service/internal/api/master_registry_handler.go` NEW ~250 LOC):
- `GET /api/v1/masters` — list
- `POST /api/v1/masters` — create (auto sets `schema_status='pending_review'`, `is_active=false`)
- `PATCH /api/v1/masters/:name` — update transform spec (triggers re-review)
- `POST /api/v1/masters/:name/approve` — flip schema_status=approved + auto-generate DDL + apply
- `POST /api/v1/masters/:name/reject` — schema_status=rejected, rejection_reason required
- `POST /api/v1/masters/:name/toggle-active` — flip `is_active` (only if schema_status=approved)

**Worker** (`internal/service/master_registry_loader.go` NEW ~150 LOC):
- Cache `master_table_registry` with 60s TTL
- Transmuter check: `WHERE source_shadow=<shadow> AND is_active=true AND schema_status='approved'` → list masters to materialize
- DDL generator: from spec + approved mapping rules → `CREATE TABLE public.<master_name> (...) WITH (FILLFACTOR=80)` + indexes

### 16.5 FE — Master Registry page NEW

`cdc-cms-web/src/pages/MasterRegistry.tsx`:
- Top bar: filter by source_shadow
- Table: name | shadow | transform_type Tag | schema_status badge | is_active Switch | schedule count | last_run_at | Actions
- Expand row: spec preview (JSON), approved rule count, last 10 run stats
- "+ Create master" modal: multi-step wizard
  1. Select source shadow (dropdown from `/api/v1/tables` = active shadows)
  2. Select transform_type (radio)
  3. Fill spec (conditional forms per type)
  4. Select mapping rules (checkboxes from source shadow rules, approved only)
  5. Preview generated DDL
  6. Submit → create row in `pending_review`

Menu NEW: `Menu.Item key="masters"` → `/masters`.

### 16.6 Active gate semantics

- `schema_status != 'approved'` → **cannot toggle is_active=true** (CMS reject 409)
- `is_active=false` → Transmuter SKIP silently, scheduler không enqueue
- `is_active=true` + `schema_status='approved'` → Transmuter runs per mode (immediate/cron/post-ingest)

---

## 17. Active / Inactive Gates — layered kiểm soát (APPENDED)

> User: "đảm bảo có cơ chế active / inactive chạy cho shadowtable, mastertable, phải active thì mới chạy"

### 17.1 Layer matrix — cả 2 tầng phải PASS

| Layer | Table | Column | Default | Who flips |
|---|---|---|---|---|
| **L1 Shadow** | `cdc_internal.table_registry` | `is_active BOOLEAN` (NEW) + `profile_status='active'` (exist) | `is_active=false` + `profile_status='pending_data'` | Admin via `/cdc-internal` page |
| **L2 Master** | `cdc_internal.master_table_registry` | `is_active BOOLEAN` (đã có trong §16.2) + `schema_status='approved'` | false + `pending_review` | Admin via `/masters` page |
| **L3 Schedule** | `cdc_internal.transmute_schedule` | `is_enabled BOOLEAN` | false | Admin via `/schedules` page |

**Invariant**: 1 Transmute run chỉ fire khi cả **L1 pass + L2 pass + (L3 pass HOẶC manual trigger)**.

### 17.2 R0 migration update — add `is_active` cho shadow registry

`migrations/019_system_registry.sql` (existing từ Phase 0) không có `is_active`. Add migration 025:

```sql
ALTER TABLE cdc_internal.table_registry
  ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT false;

-- Back-fill existing 2 rows (export_jobs + refund_requests) thành active=true
-- vì user đã approve Phase 2 chúng
UPDATE cdc_internal.table_registry
   SET is_active = true
 WHERE target_table IN ('export_jobs','refund_requests')
   AND profile_status = 'active';
```

### 17.3 Worker-side enforcement

**SinkWorker** (`internal/sinkworker/sinkworker.go::HandleMessage`):
- Thêm check đầu function: lookup `cdc_internal.table_registry WHERE target_table=<from topic>`
- Nếu `is_active=false` OR `profile_status != 'active'` → skip message, emit metric `cdc_sinkworker_skip_inactive_total{table}`, commit Kafka offset (không stuck retry loop)
- Cache 60s TTL tương tự `isFinancial`

**Transmuter** (`internal/service/transmuter.go`):
- Lookup chain: shadow L1 → master L2 → rule L3
- Fail fast với log INFO (không ERROR — inactive là admin intent, không phải bug)

### 17.4 FE UX — consistent Switch + warning banner

`CDCInternalRegistry.tsx` (shadow): add column `is_active` Switch (copy pattern từ `is_financial` Switch).
`MasterRegistry.tsx` (master): `is_active` Switch disabled khi `schema_status != 'approved'` — tooltip giải thích tại sao.

Trên pages sử dụng table list: nếu `is_active=false` → row gray-out + tag "Inactive".

### 17.5 Audit + activity log

Mọi flip `is_active` → destructive chain audit. `cdc_activity_log` operation mới: `cdc-toggle-active` với details `{layer: "shadow|master|schedule", table, old_value, new_value}`.

---

## 18. Schema Approval Workflow (APPENDED)

> User: "có cơ chế duyệt schema, update schema theo data_type"

### 18.1 Lifecycle

```
NEW FIELD DISCOVERED          ADMIN REVIEW            APPLIED
──────────────────          ──────────────         ──────────
SinkWorker/Transmuter ─► proposal         ─► approve ─► ALTER TABLE ADD COLUMN
  encounters field        (pending)             │       + add mapping rule active
  not in schema                                 │       + schema_status='approved'
                                                │
                                                └─► reject ─► proposal.status=rejected
                                                            field permanently stays in
                                                            _raw_data only
```

### 18.2 NEW migration: `cdc_internal.schema_proposal`

`migrations/026_schema_proposal.sql`:

```sql
CREATE TABLE IF NOT EXISTS cdc_internal.schema_proposal (
  id                  BIGSERIAL PRIMARY KEY,
  table_name          TEXT NOT NULL,   -- shadow OR master
  table_layer         TEXT NOT NULL CHECK (table_layer IN ('shadow','master')),
  column_name         TEXT NOT NULL CHECK (column_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  proposed_data_type  TEXT NOT NULL,   -- validated via §14.2 regex
  proposed_jsonpath   TEXT NULL,       -- NULL cho shadow auto-ALTER
  proposed_transform_fn TEXT NULL,
  proposed_is_nullable BOOLEAN NOT NULL DEFAULT true,
  proposed_default_value TEXT NULL,
  sample_values       JSONB NULL,      -- 3-5 sample values discovered
  status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','approved','rejected','auto_applied','failed')),
  submitted_by        TEXT NOT NULL,   -- 'sinkworker-auto' | 'admin-manual' | admin email
  submitted_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_by         TEXT,
  reviewed_at         TIMESTAMPTZ,
  applied_at          TIMESTAMPTZ,
  rejection_reason    TEXT,
  override_data_type  TEXT,            -- admin can override proposed_data_type at approve
  override_jsonpath   TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (table_name, table_layer, column_name)
);

CREATE INDEX idx_proposal_pending
  ON cdc_internal.schema_proposal(submitted_at DESC)
  WHERE status='pending';
```

### 18.3 Proposal-gen chain

**SinkWorker** (`internal/sinkworker/schema_manager.go` — extend):
- Hiện auto-ALTER khi `is_financial=false`. Thay thành:
  - `is_financial=false` AND `auto_approve_schema=true` (registry flag NEW) → immediate ALTER + log `schema_proposal.status='auto_applied'`
  - `is_financial=true` OR `auto_approve_schema=false` (default) → INSERT `schema_proposal.status='pending'` with sample_values (3 values from recent messages), **DO NOT ALTER**. Field stays in `_raw_data` until admin approves.

**Transmuter** (`internal/service/transmuter.go`):
- Gặp field mới trong `_raw_data` không có mapping rule → INSERT proposal (`table_layer='master'`, sample_values pulled from current rows).

### 18.4 CMS endpoints

- `GET /api/v1/schema-proposals` (shared) — filter by status, table, layer
- `GET /api/v1/schema-proposals/:id` — detail + sample values inspection
- `POST /api/v1/schema-proposals/:id/approve` (destructive) — payload `{override_data_type?, override_jsonpath?, override_transform_fn?, reason}` → apply ALTER + create mapping_rule row
- `POST /api/v1/schema-proposals/:id/reject` (destructive) — `{rejection_reason}` required
- `POST /api/v1/schema-proposals/batch-approve` (destructive) — bulk approve with per-item overrides

### 18.5 Apply-approve logic

Server `schema_proposal_handler.go::Approve`:
```
1. BEGIN TX
2. Resolve final data_type = override_data_type ?? proposed_data_type
3. Validate final data_type via §14.2 regex
4. If table_layer='shadow': ALTER TABLE cdc_internal.<table> ADD COLUMN <col> <type> WITH (default, nullable)
5. If table_layer='master': ALTER TABLE public.<table> ADD COLUMN ... + INSERT cdc_mapping_rule(source_field, target_column, data_type, jsonpath, transform_fn, status='approved')
6. UPDATE cdc_internal.schema_proposal SET status='approved', applied_at=NOW()
7. COMMIT
8. Async: if master layer, publish NATS cdc.cmd.transmute-backfill for the new col (batch fill historical rows)
```

Rollback path: catch SQLSTATE 42701 (duplicate column) → if pre-existing with same type → status='auto_applied' (idempotent); if type mismatch → `status='failed'` + error.

### 18.6 FE — Schema Proposals page NEW

`cdc-cms-web/src/pages/SchemaProposals.tsx`:
- Top badge: pending count (red if >0)
- Table: table | layer Tag | column | proposed_type | sample values preview | status | submitted_by | Actions
- Expand row: sample_values full list + DDL preview
- Modal Approve:
  - Show proposed data_type + **editable override** (Cascader từ §14.4)
  - Override jsonpath input + preview
  - Required reason ≥10 chars
- Modal Reject: rejection_reason ≥10 chars
- Batch approve: checkbox select + single reason
- Live 5s poll for pending count badge

Menu NEW: `Menu.Item key="proposals"` → `/schema-proposals`.

### 18.7 Auto-proposal rate limit

SinkWorker rate limit (từ plan §R1) 100 ALTER/day → áp dụng cho auto-approved path only. Proposals path không capped (chỉ INSERT row rẻ).

---

## 19. Phase plan update — thêm R7/R8/R9

### 19.1 Updated phase table (supersedes §3)

| Phase | Name | Scope | Depends on | Effort |
|---|---|---|---|---|
| R0 | DB migrations | 020 mapping_rule jsonpath + type constraint; 021 airbyte deprecation comments; 022 activity_log archive (opt); 023 transmute_schedule; 024 master_table_registry; 025 shadow is_active; 026 schema_proposal; 027 enum_types seed | None | 2h |
| R1 | Transmuter core + type resolver | `transmuter.go`, `type_resolver.go`, `transform_registry.go` + 16+ unit tests | R0 | 8h |
| R2 | CMS Command Center + FE refit | Kafka Connect proxy + SourceConnectors rewrite | Phase 2 S4 | 8h |
| R3 | CMS Airbyte DI prune | server.go + registry_handler.go + services | R2 | 4h |
| R4 | Worker bridge surgery | Delete HandleAirbyteBridge + source_router + bridge_service | R3 (no-conflict) | 6h |
| R5 | DB deprecation + RLS | migration 021 comments + RLS default permissive | R3+R4 | 1.5h |
| R6 | Transmuter wiring + FE JsonPath editor + mapping preview | transmute_handler + MappingFieldsPage extend + JsonPathInput + TransmutePreviewModal | R1 | 10h |
| **R7 NEW** | **Scheduler** | `transmute_scheduler.go` + CMS schedules routes + FE `TransmuteSchedules.tsx` | R6 | 6h |
| **R8 NEW** | **Master Registry + DDL generator** | `master_registry_loader.go` + master_registry_handler + FE `MasterRegistry.tsx` wizard | R6 | 10h |
| **R9 NEW** | **Schema Approval workflow** | `schema_proposal_handler.go` + proposal-gen in sinkworker + FE `SchemaProposals.tsx` | R6+R8 | 8h |

**Total updated effort**: ~63h sequential (v1 was 36h). Wall time với 2 engineer parallel: ~36-40h.

### 19.2 Critical path

```
R0 (migrations) ──► R1 (Transmuter) ──► R6 (wiring + FE) ──► R7 Scheduler
                                                          └► R8 Master Registry ──► R9 Schema Approval
                    R2 (Command Center) ──► R3 (CMS prune) ──► R5 (DB+RLS)
                    R4 (Worker prune) ──────────────────────┘
```

### 19.3 Recommended delivery order

1. **Sprint 1 (15h)**: R0 + R1 + R2 (foundation + Command Center visible)
2. **Sprint 2 (18h)**: R3 + R4 + R5 (prune Airbyte toàn bộ)
3. **Sprint 3 (20h)**: R6 + R7 (Transmuter + Scheduler — usable warehouse)
4. **Sprint 4 (18h)**: R8 + R9 (full admin control — approval workflow + master DDL generator)

---

## 20. Updated SOP Stage coverage

| Stage | Status |
|---|---|
| 1 INTAKE | ✅ absorb: data_type precision + execution modes + warehouse + active gate + schema approval |
| 2 PLAN | ✅ This doc full (Sections 0-20) + sample record doc |
| 3-7 | ⏳ Gated on user approval of updated 4-sprint delivery |

---

## 21. Approval gate v3

Chờ user + Architect duyệt bundle:

- **(A)** OK 4-sprint delivery — Muscle start Sprint 1 (R0+R1+R2) đồng thời.
- **(B)** Chia nhỏ hơn: chỉ approve Sprint 1 round này; review sau.
- **(C)** Điều chỉnh phase (ví dụ gộp R7 Scheduler vào R6; hoặc defer R9 Schema Approval).
- **(D)** Thêm/bớt feature (ví dụ thêm `join_lookup` transform sớm hơn Phase R7, hoặc cắt RLS khỏi R5).

Muscle **KHÔNG execute** đến khi approve. Workspace docs đã đủ:
- `02_plan_airbyte_removal_v2_command_center.md` (doc này — 21 sections)
- `01_requirements_mapping_rule_payment_bills_sample.md` (seed data)

SOP Stage 2 vẫn mở — không code.
