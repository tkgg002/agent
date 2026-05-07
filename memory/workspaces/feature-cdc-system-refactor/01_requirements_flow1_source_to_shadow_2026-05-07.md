# 01 — Requirements: Flow 1 — Input Source → Shadow DB

> **Author**: x2 (Muscle, cms-lane) | **Date**: 2026-05-07 ICT
> **Boss directive**: "x2 chuẩn bị làm trước cho tao flow 1: input connect source → (manual) các bước check source, kết nối Debezium, tạo db shadow, tạo table shadow, sync dữ liệu qua shadow. output là có shadow db."
> **Scope**: cms-lane (read worker code as-is, không sửa worker).
> **Workspace**: `feature-cdc-system-refactor` (sau Task #19 closed cms `b453d36`).

## 1. Mục tiêu

Cho phép operator (CMS UI hoặc API client) chạy Flow 1 **manual step-by-step** để tạo 1 shadow DB cho 1 source mới. Output: 1 shadow schema + ≥1 shadow table có data đã ingest từ source.

**KHÔNG bao gồm** (các flow sau này):
- Master DDL (`master_bind`, `discover`, `schedule_enable`).
- Reconciliation hash window.
- Master swap / RLS / Transmute pipeline.

## 2. Input từ operator

| # | Trường | Bắt buộc? | Ví dụ Mongo | Ví dụ Postgres |
|---|---|---|---|---|
| 1 | Source kind | ✅ | `mongodb` | `postgres` |
| 2 | Connection name (= `connection_code`) | ✅ | `goopay-mongodb` | `gpay-pg` |
| 3 | Server address | ✅ | `mongodb://user:pwd@host:27017/?replicaSet=rs0` | `host=db port=5432 user=… dbname=…` |
| 4 | DB include list | ✅ | `goopay,payment-bill-service` | `gpay_main` |
| 5 | Collection/Table include list | ✅ | `goopay.users` | `public.users` |
| 6 | Connector class | ✅ | `io.debezium.connector.mongodb.MongoDbConnector` | `io.debezium.connector.postgresql.PostgresConnector` |
| 7 | Topic prefix | ✅ | `cdc.goopay-mongodb` | `cdc.gpay-pg` |
| 8 | Source object name | ✅ | `users` | `users` |
| 9 | Primary key field | optional (default `id`) | `_id` | `id` |
| 10 | Reason action ≥10 chars | ✅ destructive endpoints | "register source X for warehouse Y" | same |

## 3. Steps Flow 1 (operator manual)

### Step 0 — Pre-flight (out-of-band, optional)

Operator tự kiểm tra source reachable từ network worker (ping mongo/psql). Hiện **chưa có cms endpoint cho ping** — sẽ là gap để max plan.

### Step 1 — Tạo Debezium connector

```http
POST /api/v1/system/connectors
Authorization: Bearer <admin JWT>
Idempotency-Key: connect-<ts>
X-Action-Reason: "register source <name> Debezium connector"
Content-Type: application/json

{
  "name": "<connector_name>",
  "config": {
    "connector.class": "io.debezium.connector.mongodb.MongoDbConnector",
    "mongodb.connection.string": "mongodb://...",
    "topic.prefix": "cdc.<conn>",
    "database.include.list": "<db>",
    "collection.include.list": "<db>.<coll>",
    "snapshot.mode": "initial",
    ...
  }
}
```

Side-effect:
- Forward Kafka Connect REST `POST /connectors`.
- Persist `cdc_system.system_connector_registry` row (Source fingerprint, status='created').
- Audit `admin_actions` row (qua destructive chain).

Verify: response 201 + `{name, status}` body.

### Step 2 — Check connector status

```http
GET /api/v1/system/connectors/<name>
Authorization: Bearer <token>
```

Verify:
- `status.connector.state == "RUNNING"`.
- `status.tasks[].state == "RUNNING"` (tất cả task).
- Nếu FAILED → operator đọc `trace`, sửa config, restart hoặc delete + recreate.

**Đây là step "check source"** — connector RUNNING ⇒ Debezium đã connect tới source thành công, nắm replication slot / oplog / binlog.

### Step 3 — Đăng ký source object

```http
POST /api/v1/source-objects/register
Authorization: Bearer <admin JWT>
Idempotency-Key: register-<ts>
X-Action-Reason: "register source object <coll> for shadow"
Content-Type: application/json

{
  "source_database": "<db>",
  "source_table": "<coll>",
  "target_table": "<coll>",
  "source_engine_type": "mongodb",
  "sync_engine": "debezium",
  "source_connection_id": <id-from-step-1-fingerprint>,
  "primary_key_field": "_id",
  ...
}
```

Side-effect:
- INSERT `cdc_system.source_object_registry` (V2) + `cdc_table_registry` (legacy bridge).
- **Inline call `ShadowAutomator.EnsureShadowTable`**: CREATE shadow DDL tại schema = caller-resolved (likely `shadow_<source_db>`) + sonyflake trigger + flip `is_table_created=true`.
- Dispatch CDC commands (scan-fields, etc.) qua NATS.

Verify: 201 + entry body.

**LƯU Ý**: Register-time đã tạo shadow table luôn (đường thứ nhất). Nhưng provisioning state machine ở Step 4 sẽ tạo lần thứ hai theo convention khác.

### Step 4 — Provisioning shadow_bind (đường chính thức V2)

```http
POST /api/v1/cms/sources/:source_object_id/provisioning/mode
body: {"mode": "manual"}
```

Set **manual** mode để KHÔNG cascade tự động qua master_bind.

```http
POST /api/v1/cms/sources/:source_object_id/provisioning/advance
Authorization: Bearer <admin JWT>
Idempotency-Key: advance-<ts>
X-Action-Reason: "advance source X to shadow_active"
```

Side-effect:
- CMS: `provisioning_state` flip `draft → shadow_pending` (CAS guard).
- CMS: publish NATS `cdc.cmd.shadow.bind` payload `{source_id, correlation_id, ...}`.
- Worker `ProvisioningStepHandler.HandleShadowBind`:
  1. Mongo pre-flight: `EstimatedDocumentCount > 0` (gate per L-cascade-liability).
  2. `inferSourceColumns(source_id)` — quét sample row từ source.
  3. `SchemaAdapter.PrepareForCDCInsertWithBusinessCols(schema=shadow_<conn>, table, pk, businessCols)` — CREATE TABLE IF NOT EXISTS.
  4. `upsertShadowBinding` INSERT/UPSERT `cdc_system.shadow_binding` (binding_code, shadow_schema, shadow_table, ddl_status='created', is_active=true).
  5. Emit NATS `cdc.evt.provisioning.step_completed` → CMS RecoveryLoop CAS-flip `shadow_pending → shadow_active`.

### Step 5 — Verify shadow active

```http
GET /api/v1/cms/sources/:source_object_id/provisioning
```
Expect: `state == "shadow_active"`, latest step_log entry `step=shadow_bind, success=true`.

```http
GET /api/v1/shadow-bindings?source_db=<db>
```
Expect: ≥1 row với `ddl_status='created'`, `is_active=true`, `physical_table_fqn = shadow_<conn>.<table>`.

### Step 6 — Sync dữ liệu (Debezium snapshot tự chạy)

Connector tạo ở Step 1 với `snapshot.mode=initial` → Debezium tự ingest snapshot rồi switch sang streaming. **KHÔNG có endpoint cms thủ công cần gọi.**

Nếu cần re-snapshot (operator-decided):
```http
POST /api/v1/system/connectors/<name>/restart
```

### Step 7 — Verify data

```http
GET /api/v1/source-objects/:source_object_id/transform-status
```
Expect `total_rows > 0` (số dòng đã land vào shadow table).

Hoặc raw psql:
```sql
SELECT COUNT(*) FROM shadow_<conn>.<table>;
SELECT _synced_at, _source FROM shadow_<conn>.<table> LIMIT 5;
```

→ **Output**: shadow DB ready, có data.

## 4. Definition of Done (Boss-facing)

- ✅ 1 connector RUNNING ở Kafka Connect.
- ✅ 1 row `cdc_system.system_connector_registry`.
- ✅ 1 row `cdc_system.source_object_registry` (sync_engine='debezium', is_active=true).
- ✅ 1 row `cdc_system.shadow_binding` (ddl_status='created', is_active=true).
- ✅ `provisioning_state = 'shadow_active'` (V2 path) hoặc registry `is_table_created=true` (legacy path).
- ✅ Shadow table có ≥ 1 dòng data từ Debezium snapshot.
- ✅ `admin_actions` audit row cho từng destructive call (Step 1, 3, 4).
- ✅ Báo Boss `report_flow1_*.md` với HTTP code + DB row evidence.

## 5. Stakeholders & Lane

- **Boss** — approve final plan + verify output.
- **max-Brain** — ratify x2's plan (per §1 Brain plans, Muscle executes).
- **x2 (Muscle)** — implement cms-side gaps (nếu có), run smoke E2E.
- **Worker (centralized-data-service)** — read-only, lane phân cho max nếu cần sửa.
- **FE (cdc-cms-web)** — read-only, scope sau Flow 1.

## 6. Constraints

- Lane lock: x2 chỉ stage `cdc-cms-service/`. Worker code chỉ đọc.
- Manual mode: không cascade, dừng tại `shadow_active` trừ khi Boss yêu cầu next phase.
- Per L-pre-flight-check: build/vet/test/runtime smoke trước khi báo done.
- Per L-route-tier: Step 1+3+4 destructive, Step 2+5+7 shared GET.
- Per L-multi-tier-filter: verify `connector.status==RUNNING` + first event arrives, không tin write success.
- Per L-cascade-liability: Mongo source bắt buộc có data trước Step 4 (worker pre-flight đã enforce).

— x2
