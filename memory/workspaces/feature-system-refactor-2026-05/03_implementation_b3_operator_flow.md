# B3 — Operator Add-Source-DB Flow Inventory

**Phase**: B3.3 (FR-B3-7)
**Author**: Brain (Antigravity, claude-opus-4-7)
**Date**: 2026-05-05
**Source-of-truth**: code dưới đây inspected live tại commit hiện hành.

---

## 1. Service map (4 service + 1 admin-api + 1 worker)

| Service | Port | Role |
|---|---|---|
| cdc-cms-web (Vite) | 5173 | FE wizard + dashboard |
| cdc-auth-service | 8081 | JWT issue (admin@goopay.vn / admin123) |
| cdc-cms-service | 8083 | Operator API (`/api/v1/*`) — JWT-gated |
| centralized-data-service admin-api | 8090 | Privileged source register (`/v2/sources/register`, Bearer token) |
| centralized-data-service worker (NATS sub) | n/a | Schema discover, ingest, transmute |
| Kafka Connect (Debezium) | 18083 | Per-engine connector |

## 2. NATS subjects (worker_server.go inventory)

| Subject | Direction | Purpose |
|---|---|---|
| `cdc.cmd.discover` | cms → worker | Run schema discover on shadow row |
| `cdc.cmd.standardize` | cms → worker | Apply standardize rules |
| `cdc.cmd.batch-transform` | scheduler → worker | Periodic transform tick |
| `cdc.cmd.transmute` | scheduler → worker | Materialize 1 master |
| `cdc.cmd.transmute-shadow` | post-ingest hook | Fan-out per shadow row |
| `cdc.cmd.master-create` | cms → worker | DDL master table from manifest |
| `cdc.cmd.shadow.bind` | provisioning → worker | Bind shadow_binding row → CREATE TABLE |
| `cdc.cmd.schedule.enable` | provisioning → worker | Enable cron after master ready |
| `cdc.cmd.scan-fields` | cms → worker | Sample source schema |
| `cdc.cmd.sync-register` | cms → worker | Register Airbyte/connector metadata |
| `cdc.evt.transmute.completed` | worker → JobMonitor | Close-loop UPDATE last_status |
| `cdc.evt.provisioning.step.completed` | step handler → orchestrator | Provisioning state advance |

## 3. 11-Step Operator Flow (engine-agnostic)

```
1. (FE) Login admin/admin123 → JWT 245 char
2. (FE) Navigate /wizard → SourceToMasterWizard.tsx
3. (FE) Step 1: Choose engine (mongo|mariadb|postgres)
        Step 2: Enter connection (host/port/db/user/password)
        Step 3: Pick namespace + objects
4. (BE) FE → cdc-cms-service POST /api/v1/wizard/sessions  (create draft)
        FE → PATCH /api/v1/wizard/sessions/:id            (save fields)
5. (FE) Click "Execute" → POST /api/v1/wizard/sessions/:id/execute
6. (BE) cdc-cms-service forwards to admin-api:
        POST :8090/v2/sources/register  (Bearer ADMIN_API_TOKEN)
        → INSERT cdc_system.source_object_registry + shadow_binding
        → publish cdc.cmd.shadow.bind {schema, table, pk, business_cols}
7. (worker) HandleShadowBind → schema_adapter.go::PrepareForCDCInsertWithBusinessCols
        → IF NOT EXISTS CREATE TABLE shadow.<schema>.<table> with V1 CDC cols
        (idempotent — safe re-run; B3.P2 fix)
8. (worker) HandleDiscover samples shadow rows → emits schema_proposal row
9. (FE)  /schema-proposals page lists proposal → operator approve
        → POST /api/v1/schema-proposals/:id/approve
        → ALTER shadow + INSERT mapping_rule_v2 atomic
10. (FE) /masters page → operator clicks "Create master"
        → POST /api/v1/masters {master_name, source_table, manifest}
        → INSERT cdc_system.master_binding + publish cdc.cmd.master-create
        → worker masterDDLHandler runs DDL on goopay_dest
        → publish cdc.cmd.schedule.enable → INSERT cdc_system.transmute_schedule (cron */1)
11. (cron tick 60s) scheduler picks schedule, publishes cdc.cmd.transmute
        → handler runs Transmuter.Run → upsert into goopay_dest master
        → publish cdc.evt.transmute.completed
        → JobMonitor UPDATE transmute_schedule.last_status='success'
```

## 4. Per-engine quirk

### 4.1 PostgreSQL source
- **Debezium connector**: `io.debezium.connector.postgresql.PostgresConnector`
- **Pre-req**: `wal_level=logical`, `max_replication_slots>=4`, `REPLICATION` privilege.
- **Slot name**: `gpay_<source_db>_slot` (auto-created by connector).
- **PK**: native column (`id BIGINT` etc) — `pkColumn` in source_object_registry stores actual PK.
- **Engine type literal in registry**: `postgres`.

### 4.2 MariaDB source
- **Debezium connector**: `io.debezium.connector.mysql.MySqlConnector` (compat with MariaDB binlog).
- **Pre-req**: `server_id` unique, `binlog_format=ROW`, `binlog_row_image=FULL`, user with `REPLICATION SLAVE, REPLICATION CLIENT`.
- **PK**: auto-increment `BIGINT` column.
- **Engine type literal**: `mariadb`.
- **Snapshot mode**: `initial` (full + tail binlog).

### 4.3 MongoDB source
- **Debezium connector**: `io.debezium.connector.mongodb.MongoDbConnector`
- **Pre-req**: ReplicaSet (1 node OK for dev: `rs0`), `local.oplog.rs` readable, role `clusterMonitor` + `read` on target DB.
- **PK**: `_id` ObjectID — schema_adapter normalizes via `normalizeMongoExtendedJSON` (`{"$oid":...}` → string).
- **Engine type literal**: `mongo`.
- **Schemaless quirk**: business_cols inferred from sample doc; missing fields stored as NULL in shadow.

## 5. CMS routes inventory (cdc-cms-service router.go)

### Public (no auth)
- `GET /health` → 200

### JWT-gated `/api/v1/*`
- Wizard: `POST /v1/wizard/sessions/:id/execute`
- Sources: `GET /v1/source-objects`
- Masters: `POST /v1/masters`, `POST /v1/masters/:name/approve|reject|toggle-active|swap`
- Schema: `POST /v1/schema-proposals/:id/approve|reject`
- Schedules: `POST /v1/schedules`, `PATCH /v1/schedules/:id`, `POST /v1/schedules/:id/run-now`
- Recon: `POST /v1/reconciliation/check[/:table]`, `POST /v1/reconciliation/heal`
- Provisioning: `GET /v1/cms/sources/:id/provisioning`, `POST .../advance|pause|resume|retry|archive|mode`
- Connectors: `POST /v1/system/connectors`, `POST .../:name/restart|pause|resume`, `DELETE /v1/system/connectors/:name`

### Admin-api (Bearer token)
- `GET /healthz` (no auth)
- `POST /v2/sources/register` (auth + idempotency + 64KiB body cap + 10 req/min/token)

## 6. Verify checklist per engine (smoke template)

```bash
# Set ENG = mongo|mariadb|postgres
ENG=postgres
SRC_NAME=smoke_b3_${ENG}_$(date +%s)

# 1. Login → JWT
JWT=$(curl -s -X POST localhost:8081/api/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@goopay.vn","password":"admin123"}' | jq -r .token)

# 2. Register source via wizard execute (or direct admin-api)
curl -s -X POST localhost:8090/v2/sources/register \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'content-type: application/json' \
  -d "{\"engine\":\"$ENG\",\"name\":\"$SRC_NAME\",\"connection\":{...},\"objects\":[...]}"

# 3. Wait 30s for schema_proposal
sleep 30
PROPOSAL_ID=$(curl -s -H "Authorization: Bearer $JWT" \
  localhost:8083/api/v1/schema-changes/pending | jq -r '.[0].id')

# 4. Approve
curl -s -X POST localhost:8083/api/v1/schema-proposals/$PROPOSAL_ID/approve \
  -H "Authorization: Bearer $JWT" \
  -H 'idempotency-key: '$(uuidgen) \
  -d '{"reason":"smoke b3"}'

# 5. Create master
curl -s -X POST localhost:8083/api/v1/masters \
  -H "Authorization: Bearer $JWT" \
  -H 'idempotency-key: '$(uuidgen) \
  -d "{\"master_name\":\"${SRC_NAME}_master\",\"source\":\"$SRC_NAME\",\"reason\":\"smoke b3\"}"

# 6. Wait cron tick (60s)
sleep 65

# 7. Verify
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM shadow.${SRC_NAME};"
docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest -c \
  "SELECT count(*) FROM dw_${SRC_NAME}.${SRC_NAME}_master;"
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT id, last_status, last_run_at FROM cdc_system.transmute_schedule \
   WHERE master_binding_id IN (SELECT id FROM cdc_system.master_binding \
                               WHERE name='${SRC_NAME}_master');"
```

## 7. Failure modes + rollback

| Failure | Symptom | Rollback |
|---|---|---|
| Connector boot panic | Kafka Connect log `failed to connect` | `DELETE /v1/system/connectors/:name` then re-create |
| Shadow CREATE permission denied | worker log `permission denied for schema` | grant `gpay_admin` on schema, retry shadow.bind |
| Schema discover empty | proposal not appearing | check shadow row count `> 0`, re-fire `cdc.cmd.discover` |
| Master DDL drift | master table missing column | re-approve schema_proposal → ALTER cascades |
| Schedule stuck `running` | last_status='running' >120s | JobMonitor missed event; manual `UPDATE last_status='failed'` then `POST /v1/schedules/:id/run-now` |

## 8. Manual fallback (operator clicks bypass — when wizard breaks)

1. SQL: `INSERT cdc_system.source_object_registry (object_code, engine, ...) VALUES (...)`
2. SQL: `INSERT cdc_system.shadow_binding (...)`
3. NATS `cdc.cmd.shadow.bind` direct publish
4. SQL: `INSERT cdc_system.master_binding (...)`
5. NATS `cdc.cmd.master-create` direct publish
6. SQL: `INSERT cdc_system.transmute_schedule (...) VALUES (..., is_enabled=true)`

Document chỉ; routine không khuyến nghị.
