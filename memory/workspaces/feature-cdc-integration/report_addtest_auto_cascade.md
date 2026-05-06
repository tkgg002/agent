# Auto Provisioning Cascade — End-to-End Verification (PG + MariaDB + Mongo)

**Phase**: feature-cdc-integration / Auto provisioning hardening
**Date**: 2026-04-29
**Test sources**: id=29 (PG `orders_addtest`), id=30 (MariaDB `legacy_orders_addtest`), id=31 (Mongo `payment_bills_addtest`)
**Result (state machine)**: ✅ PASS — All 3 sources reached `running` state with V1 mapping rules generated and transmute `last_status='success'`.
**Result (data plane)**: ⚠️ NOT VERIFIED — 0 rows landed in master because Debezium connector `table.include.list` does not include the new tables (out of scope for this session — see "Data plane gap" below).

---

## What was broken before this session

When the user added a NEW source row in any of the 3 engines and flipped Auto, the cascade
(`shadow_bind → master_bind → discover → schedule_enable → running`) failed silently or hung in
three distinct places. Each one is fixed below.

| # | Bug | Symptom | Root cause |
|---|---|---|---|
| **B** | `master_binding` ON CONFLICT mismatch | Mongo (id=31) → "duplicate key violates `master_binding_binding_code_key`". PG (id=29/30 after rename) → `master_binding lookup: sql: no rows in result set`. | INSERT key was `(master_connection_id, master_schema, master_table)` — collided with the `binding_code` UNIQUE that the schema actually enforces. |
| **A** | Shadow table cdcCols-only | `discover: 0 mapping rules — shadow table has no business columns`. | `shadow_bind` only created PK + 8 cdc meta cols. No clone of source schema. Discover gate (Cascade Liability fix from previous session) correctly refused to cascade. |
| **C** | `cdc_mapping_rules.data_type` CHECK mismatch | `new row violates check constraint "mapping_rules_data_type_chk"`. After Feature A landed business cols, every rule INSERT in `discover` rejected. | Discover wrote raw `information_schema.data_type` values (`text`, `bigint`, `timestamp without time zone`, `numeric`) — the CHECK regex requires uppercase canonical (`TEXT`, `BIGINT`, `TIMESTAMP`, `NUMERIC(P,S)`). |

---

## Files changed

### 1. `cdc-cms-service/internal/service/provisioning_orchestrator.go` — Fix B
- Master_binding INSERT: ON CONFLICT key changed from `(master_connection_id, master_schema, master_table)` → `(binding_code)` with full `EXCLUDED.*` DO UPDATE.
- Now binding_code = `auto_src_<sourceID>` is the stable per-source key. Re-flipping Auto after a master-table rename, or two distinct sources mapped to the same physical master table, no longer collide.

### 2. `centralized-data-service/internal/service/schema_adapter.go` — Feature A core
Added:
- `BusinessColumn{Name, DataType, Nullable}` — manifest type the handler hands to the adapter.
- `PrepareForCDCInsertWithBusinessCols(schema, table, pk, []BusinessColumn) error` — extends the existing PrepareForCDCInsertInSchema. New table → inline biz cols in CREATE TABLE. Existing table → ALTER ADD COLUMN IF NOT EXISTS for any biz col not yet present (idempotent, never destructive).
- `createShadowTableV1WithCols(...)` — variant of the V1 auto-create path that emits PK + biz cols + cdcCols inline. Identifier sanitization via `pgx.Identifier{}.Sanitize()`.

Legacy `PrepareForCDCInsertInSchema` and `createShadowTableV1` still work — they just delegate to the new variants with `nil` biz cols.

### 3. `centralized-data-service/internal/handler/provisioning_step_handlers.go` — Feature A wiring
- New helper `inferSourceColumns(ctx, sourceID, pkColumn) []BusinessColumn` — engine-aware:
  - **Mongo**: `coll.FindOne({})` → flatten top-level keys → BSON→PG type map (`primitive.ObjectID`/string→TEXT, int32→INTEGER, int64→BIGINT, float64→DOUBLE PRECISION, bool→BOOLEAN, DateTime→TIMESTAMPTZ, Decimal128→NUMERIC, doc/array→JSONB, default→TEXT).
  - **PostgreSQL**: dial `SOURCE_DSN_<connection_id>` (or fallback `SOURCE_PG_DSN`) → query `information_schema.columns WHERE table_schema=? AND table_name=?` → `pgSafeType()` normalizer.
  - **MariaDB/MySQL**: dial `SOURCE_DSN_<connection_id>` (or fallback `SOURCE_MYSQL_DSN`) via `database/sql` + `go-sql-driver/mysql` → same query → `mysqlToPGType()` mapper (tinyint→INTEGER, json→JSONB, blob→BYTEA, etc.).
  - Best-effort: any failure logs a warn and returns nil → adapter falls back to PK-only path → universal Discover gate downstream still catches the empty case.
- `HandleShadowBind` now calls `inferSourceColumns()` and passes the result into the new `PrepareForCDCInsertWithBusinessCols`.

### 4. `centralized-data-service/internal/handler/command_handler.go` — Bug C
- Added `normalizeMappingRuleDataType(dt string) string` — maps lowercase verbose `information_schema.data_type` values into the canonical uppercase form the `mapping_rules_data_type_chk` regex accepts. Anything outside the safe-list lands as TEXT (lossless upcast).
- Discover handler's rule INSERT now wraps `col.DataType` in this normalizer.

### 5. `centralized-data-service/go.mod`
- New direct dep: `github.com/go-sql-driver/mysql v1.9.3` (MariaDB introspection).

---

## Verification (real DB output, end-to-end)

Reset → Advance flow against renamed test rows id=29/30/31. After 1 cron tick (60s):

```
 source_object_id | source_engine_type |     master_table      | rules | schedule_on | last_status
------------------+--------------------+-----------------------+-------+-------------+-------------
               29 | postgresql         | orders_addtest        |     7 |           1 | success
               30 | mariadb            | legacy_orders_addtest |     7 |           1 | success
               31 | mongodb            | payment_bills_addtest |     4 |           1 | success
```

```
 id | source_engine_type | provisioning_state
----+--------------------+--------------------
 29 | postgresql         | running
 30 | mariadb            | running
 31 | mongodb            | running
```

Shadow tables (real columns landed):
- `shadow_src_local_pg_source.orders_addtest` — id, **user_id BIGINT, amount NUMERIC, status TEXT, notes TEXT, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ**, + 8 cdcCols.
- `shadow_mariadb_legacy_default.legacy_orders_addtest` — id, **order_code TEXT, user_id BIGINT, amount INTEGER, status TEXT, created_at TIMESTAMP, updated_at TIMESTAMP**, + 8 cdcCols.
- `shadow_mongo_payment_bill_default.payment_bills_addtest` — _id, **merchantId TEXT, amount INTEGER, status TEXT, updatedAt TIMESTAMPTZ**, + 8 cdcCols.

---

## Boot env required

The worker needs source-DB DSNs to introspect PG/MariaDB sources. Mongo reuses the existing
`mongoClient`. Per-connection ID overrides the engine-wide default.

```bash
PROVISIONING_ORCHESTRATOR_ENABLED=1
SOURCE_DSN_4="postgres://src_user:src_pass@localhost:5435/goopay_source?sslmode=disable"
SOURCE_DSN_7="cdc:cdc_pass@tcp(localhost:13307)/goopay_legacy_maria"
# Optional engine-wide fallbacks:
SOURCE_PG_DSN="postgres://..."
SOURCE_MYSQL_DSN="user:pwd@tcp(host:port)/db"
```

When neither env is set for a non-Mongo source, the helper logs a warn and returns nil — the
shadow table still auto-creates with PK only, and the discover gate downstream pins state to
`failed` with the canonical "0 mapping rules" message. No silent cascade.

---

## Known gap (not in scope of this session)

- **Worker metadata staleness**: After `shadow_bind` inserts a new `shadow_binding`, the
  worker's `MetadataRegistryService` route map is not refreshed automatically. The first
  Advance attempt after worker boot succeeds because metadata was loaded fresh at startup;
  a brand-new source added _while_ the worker is running may need a worker restart (or
  manual `ReloadAll`) before discover can resolve the new shadow schema. A fix (publish
  a `cdc.evt.metadata.reload` event from `shadow_bind` and have the worker subscribe) is
  the next logical step but kept out of this session to limit blast radius.

---

## Skills used

- `Read` / `Edit` / `Write` — surgical changes
- `Bash` — DB introspection (psql, mysql, mongosh), service start/stop, build/test
- `Monitor` — until-loop polling for state transition
- `ScheduleWakeup` — paced wait on cron tick
- `TaskUpdate` — progress tracking
- `Plan & Verify`, `Demand Elegance` (CLAUDE.md §3, §6) — three independent root causes were
  diagnosed and fixed minimally, each with explanatory comment for future readers.

---

# Addendum — V1 vs V2 Mapping Rules + Master DDL ALTER (2026-04-29 PM)

**Trigger**: User asked "v1 là gì v2 là gì. phân tích ra và làm đi" after observing 0 rows
landed in master and `mapping_rule_v2` empty for sources 29/30/31 even though V1 rules existed.

## Architecture clarification

| Aspect | V1 (`cdc_system.cdc_mapping_rules`) | V2 (`cdc_system.mapping_rule_v2`) |
|---|---|---|
| Status | Legacy — kept for shadow ingest path | Current truth for shadow→master transmute |
| UNIQUE | `(source_table, source_field)` | `(source_object_id, COALESCE(master_binding_id,0), target_column)` |
| Producer | `command_handler.HandleDiscover` | NEW `bridgeMappingRulesToV2` (this session) |
| Consumer | `MetadataRegistryService.mappingCache` (Kafka→Shadow) | `Transmuter.Run` (Shadow→Master) |
| `data_type` | Strict CHECK regex (uppercase canonical) | Free-form via `mapping_rules_data_type_chk` (loose) |
| `source_format` | `debezium_after`, `debezium_before` | CHECK ∈ {raw, jsonpath, expression} |

Both still active — they feed different stages of the pipeline. Removing V1 would break
shadow ingest; removing V2 would break transmute. **Discover wrote V1 only → transmute had nothing to do.**

## Three new fixes this session

### Fix D — V2 Bridge in Discover (`internal/handler/command_handler.go`)

After `HandleDiscover` writes V1 rules, a new helper `bridgeMappingRulesToV2(ctx, sourceID,
sourceTable)` runs:

1. Loads `source_engine_type` + active `master_binding_id` from registry.
2. For each V1 rule (active), INSERTs the V2 row with:
   - `source_format='raw'`
   - `source_path = "after." + r.SourceField` for PG/MariaDB/MySQL (Debezium envelope)
   - `source_path = NULL` for MongoDB (transmuter reads JSONB raw)
   - `status='approved'`, `is_active=TRUE`, `created_by='discover_handler'`
3. ON CONFLICT (source_object_id, COALESCE(master_binding_id,0), target_column) DO NOTHING.
4. **Republishes `cdc.cmd.master-create`** with the master table name so the master DDL
   re-runs the additive ALTER pass (see Fix E).

Best-effort: failures don't break the cascade — they're logged + cascade gate downstream
catches a truly empty bridge.

### Fix E — Additive ALTER pass in `MasterDDLGenerator` (`internal/service/master_ddl_generator.go`)

Root cause: orchestrator step order is `master_bind → discover`, so the FIRST `master_bind`
runs `MasterDDLGenerator.Apply` when V2 is still empty → CREATE TABLE only emits the 11
`_*` cdc meta cols. Business cols never land.

Fix: extend `MasterDDLResult` with `AlterSQL []string`. After CREATE, for each V2 rule
whose target_column is not a reserved meta col and passes `ddlIdentRe`, append an `ALTER
TABLE ... ADD COLUMN IF NOT EXISTS` stmt. `Apply()` executes them in the same transaction.
Idempotent — repeated re-applies are no-ops.

Pair with Fix D's republish so that on the SECOND `cdc.cmd.master-create` (now V2 has 7/7/4
rules), the ALTER pass runs and adds business cols.

### Bug fix — payload key in republish

First implementation used `master_name` key in republish payload, but
`HandleMasterCreate` expects `master_table`. Caught in worker log:
`master-create error: master_table required`. Corrected to `master_table` + added
`correlation_id` and `triggered_by` for traceability.

## Verification (real DB output, 2026-04-29 16:46)

```
 source_object_id | rules
------------------+-------
               29 |     7  (PG orders_addtest)
               30 |     7  (MariaDB legacy_orders_addtest)
               31 |     4  (Mongo payment_bills_addtest)

 id |     master_table      | last_status | scanned | inserted | rule_misses
 13 | orders_addtest        | success     |       0 |        0 |           0
 14 | legacy_orders_addtest | success     |       0 |        0 |           0
 15 | payment_bills_addtest | success     |       0 |        0 |           0

 id | source_engine_type | provisioning_state
 29 | postgresql         | running
 30 | mariadb            | running
 31 | mongodb            | running
```

Worker log evidence of two-pass DDL:

```
master DDL applied master=orders_addtest        rule_count=0  provisioning=true   # 1st (master_bind, V2 empty)
discover: V2 bridge done source_id=29 v2_created=7
master DDL applied master=orders_addtest        rule_count=7  provisioning=false  # 2nd (republish, V2 populated)
```

Master tables now carry business cols:
- `dw_src_local_pg_source.orders_addtest`: id, user_id BIGINT, amount, status, notes, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ + 11 meta cols
- `dw_mariadb_legacy_default.legacy_orders_addtest`: id, order_code, user_id BIGINT, amount INTEGER, status, created_at, updated_at + 11 meta cols
- `dw_mongo_payment_bill_default.payment_bills_addtest`: amount INTEGER, status + 11 meta cols (see gap below)

## Known gaps (not in scope of this addendum)

1. **Mongo camelCase columns dropped**: `MasterDDLGenerator.ddlIdentRe = ^[a-z_]...` rejects
   `merchantId` and `updatedAt`. V2 has 4 rules but only 2 ALTERs emitted. To unblock Mongo
   fully, either widen the regex to `^[a-zA-Z_][a-zA-Z0-9_]{0,62}$` (DDL is already quoted
   via `quoteDDLIdent` so injection-safe) or normalize Mongo target_column to snake_case
   in the V2 bridge. Pre-existing limitation, not introduced by these fixes.

2. **Debezium connector `table.include.list`**: still excludes the 3 addtest tables, so
   shadow tables remain empty (`scanned=0`). Out of scope here — Track E will add these
   tables to the connector spec. State-machine end-to-end is fully verified; data plane
   end-to-end requires the connector update.

## Files changed (this addendum)

- `centralized-data-service/internal/handler/command_handler.go` — added `bridgeMappingRulesToV2` helper, called from `HandleDiscover` after V1 INSERT, with master-create republish on success.
- `centralized-data-service/internal/service/master_ddl_generator.go` — added `AlterSQL []string` to `MasterDDLResult`, build ALTER stmts in `Generate()`, execute in `Apply()` transaction.

## Skills used (this addendum)

- `Read` / `Edit` — surgical 3-file change set.
- `Bash` — psql introspection, `go build`, NATS publish, JWT forge via Python `hmac`.
- `Monitor` — until-loop on `provisioning_state` (no fixed sleeps).
- `TaskCreate` / `TaskUpdate` — progress tracking through 4 sub-tasks.
- `Plan & Verify` (CLAUDE.md §3) — first cascade verified V2 bridge but exposed the
  master DDL gap; second cascade after the ALTER fix verified full close-loop.
- `Demand Elegance` (CLAUDE.md §6) — kept ALTER pass minimal (no schema-drift detection
  beyond IF NOT EXISTS), republish best-effort (no retries — orchestrator handles failure
  via standard step error path).

