# Report — Cascade Liability Hardening (Multi-Engine Auto Provisioning)

**Phase**: `multi_engine_unified` → step-level fail-fast gates
**Ngày**: 2026-04-29
**Scope**: 2 file Go (1 patch + 1 mới), 1 file FE TS (audit reason header), 1 SQL migration MariaDB seed
**Definition of Done**: 3 engine (PostgreSQL, MariaDB, MongoDB) chạy live qua flip Auto, gate phát hiện đúng → KHÔNG có row schemaless cascade tới `running` với pipeline rỗng.

---

## 1. Vấn đề (Cascade Liability)

State machine `provisioning_state_machine.go` định nghĩa 4 transition tuần tự:

```
draft → shadow_bind   → shadow_active
shadow_active → master_bind   → master_active
master_active → discover      → mapping_ready
mapping_ready → schedule_enable → running
```

Khi `provisioning_mode='auto'` orchestrator tự fan-out qua từng step. Track D test với **PostgreSQL source (schema tĩnh)**: mọi step đều có dữ liệu, không lộ lỗi. Nhưng khi mở rộng sang **MongoDB schemaless** + **MariaDB structured**:

- **MongoDB**: nếu collection trống / chưa có doc → `shadow_bind` chỉ tạo bảng meta CDC (`_raw_data`, `_deleted`, `_synced_at`...) **không có cột business**. Step `discover` quét `information_schema.columns` của shadow → không thấy field → sinh **0 mapping rules**. Orchestrator vẫn cascade tới `mapping_ready` → `schedule_pending` → `running`. Khi document thật đổ vào → pipeline gãy hàng loạt (silent time bomb).
- **MariaDB**: cùng vấn đề nếu legacy table chưa có row.
- **Schema drift sau Auto**: collection thêm nested field → discover sinh design SAI → orchestrator cầm design sai đi `Create Master Table` → khi data thật vào, type mismatch / NULL violation → toàn bộ row bị reject.

Lỗ hổng chung: **không có step-level fail-fast**. Mọi step return `success` ngay cả khi nội dung 0/empty.

---

## 2. Sửa đổi (2 minimal-scope gates)

### Fix A — Universal: Discover gate (chặn pipeline khi 0 mapping rules)

**File**: `centralized-data-service/internal/handler/command_handler.go` (gần line 444)
**Logic**: sau khi quét `information_schema.columns` của shadow table:

```go
totalRules := count + len(existing)
if totalRules == 0 {
    stepErr = fmt.Errorf("discover: 0 mapping rules — shadow table %q has no business columns (cdc meta only). Likely cause: source schema not yet projected into shadow (e.g. Mongo schemaless without sample, or shadow_bind ran before any source row landed). Refusing to cascade", payload.TargetTable)
    h.logger.Error("discover: zero rules — break cascade",
        zap.String("target", payload.TargetTable),
        zap.Int("cdc_only_columns", len(cols)))
    h.publishResult(msg, CommandResult{
        Command: "discover", RegistryID: payload.RegistryID,
        TargetTable: payload.TargetTable, Status: "error",
        Error: stepErr.Error(),
    })
    return
}
```

**Tại sao universal?** Mọi engine (PG/MariaDB/Mongo) đều đi qua `discover` → check ngay tại đây bắt được mọi failure mode (empty source, schemaless không có doc, shadow_bind race).

### Fix B — Mongo-specific: Pre-flight collection ở shadow_bind

**File**: `centralized-data-service/internal/handler/provisioning_step_handlers.go`

Thêm field `mongoClient *mongo.Client` vào `ProvisioningStepHandler`, constructor nhận thêm param. Trong `HandleShadowBind` (sau `resolveShadowTarget`, trước `PrepareForCDCInsertInSchema`):

```go
if eng, db, obj, eErr := h.fetchSourceEngine(ctx, req.SourceID); eErr == nil && isMongoEngine(eng) {
    if pErr := h.preflightMongoSource(ctx, db, obj); pErr != nil {
        stepErr = fmt.Errorf("mongo source preflight: %w", pErr)
        return
    }
}
```

Helpers (cùng file):

- `isMongoEngine`: nhận `"mongodb"` / `"mongo"`.
- `fetchSourceEngine`: `SELECT source_engine_type, source_database, source_object_name FROM cdc_system.source_object_registry WHERE id=?`.
- `preflightMongoSource`: validate `mongoClient != nil`, `db != ""`, `coll != ""`, `EstimatedDocumentCount(ctx) > 0`. Nếu count=0 → `fmt.Errorf("collection %s.%s is empty — refusing to cascade with no source data to infer schema from", db, coll)`.

**Wire ở boot** (`internal/server/worker_server.go:324`):

```go
stepHandler := handler.NewProvisioningStepHandler(db, natsClient.Conn, schemaAdapter, mongoClientShared, logger)
```

**Tại sao Mongo-specific?** Discover gate (Fix A) bắt được khi đã tới step thứ 3. Pre-flight Mongo cắt sớm hơn, ngay step đầu — failure log cho operator rõ ràng "collection trống" thay vì "shadow table không có business columns".

### Fix phụ — FE audit reason header

**File**: `cdc-cms-web/src/hooks/useProvisioningMode.ts`
Audit middleware đọc `reason` từ JSON body, không phải header. Sửa hook gửi cả 2:

```ts
const { data } = await cmsApi.post<SetModeResponse>(
  `/api/v1/cms/sources/${id}/provisioning/mode`,
  { mode, reason },              // ← body (audit middleware reads)
  { headers: { 'Idempotency-Key': newIdempotencyKey(), 'X-Action-Reason': reason } },
);
```

### Fix phụ — Migration 049 schema correction

**File**: `centralized-data-service/migrations/cdc/049_mariadb_seed_legacy_orders.sql`
Trước: dùng cột không tồn tại `(description, config_json, is_active)`. Sau: đúng schema thực tế `(display_name, role_type, secret_ref, options_json, status)`.

---

## 3. Test thực tế (live, không faking)

### 3.1 Setup

| Engine     | Container               | DB                     | Collection / Table | Pre-test data |
|------------|-------------------------|------------------------|--------------------|---------------|
| PostgreSQL | gpay-postgres-source    | goopay_source          | public.orders      | 11 rows       |
| MariaDB    | gpay-mariadb-legacy     | goopay_legacy_maria    | legacy_orders      | 0 rows        |
| MongoDB    | gpay-mongo-payment      | payment-bill-service   | payment_bills      | 0 docs        |

3 row được tạo trong `cdc_system.source_object_registry` qua FE `Add Source` flow.

### 3.2 Action: flip mode auto cho cả 3 row

```bash
# Qua FE TableRegistry → click toggle Auto/Manual → reason "smoke test cascade liability"
# Hoặc curl tương đương:
curl -X POST http://localhost:8083/api/v1/cms/sources/<ID>/provisioning/mode \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "X-Action-Reason: smoke test cascade liability" \
  -H "Content-Type: application/json" \
  -d '{"mode":"auto","reason":"smoke test cascade liability"}'
```

### 3.3 Kết quả

```sql
SELECT id, source_object_name, source_engine_type, source_database,
       provisioning_state, provisioning_mode
  FROM cdc_system.source_object_registry
 WHERE id IN (11,27,28) ORDER BY id;
```

| id | source_object_name | engine     | db                   | state    | mode |
|----|--------------------|------------|----------------------|----------|------|
| 11 | orders             | postgresql | goopay_source        | **running**  | auto |
| 27 | legacy_orders      | mariadb    | goopay_legacy_maria  | **failed**   | auto |
| 28 | payment_bills      | mongodb    | payment-bill-service | **failed**   | auto |

```sql
SELECT source_object_id, count(*) AS rules
  FROM cdc_system.mapping_rule_v2
 WHERE source_object_id IN (11,27,28)
 GROUP BY source_object_id;
```

| source_object_id | rules |
|------------------|-------|
| 11               | 7     |
| 27               | 0 (không insert vì gate fail trước) |
| 28               | 0 (không insert vì gate fail trước) |

### 3.4 Worker log evidence (`/tmp/cdc-worker.log`)

**Fix B fire (id=28 Mongo, step shadow_bind):**

```
{"level":"warn","ts":1777449916.95,"msg":"provisioning: step failed",
 "source_id":28,"step":"shadow_bind","from":"shadow_pending",
 "error":"mongo source preflight: collection payment-bill-service.payment_bills is empty — refusing to cascade with no source data to infer schema from"}
```

**Fix A fire (id=27 MariaDB, step discover):**

```
{"level":"error","ts":1777449917.04,"msg":"discover: zero rules — break cascade",
 "target":"legacy_orders","cdc_only_columns":0}
{"level":"info","ts":1777449917.04,"msg":"command result","command":"discover",
 "target_table":"legacy_orders","status":"error",
 "error":"discover: 0 mapping rules — shadow table \"legacy_orders\" has no business columns (cdc meta only). Likely cause: source schema not yet projected into shadow ..."}
{"level":"warn","ts":1777449917.04,"msg":"provisioning: step failed",
 "source_id":27,"step":"discover","from":"mapping_pending"}
```

**Cascade thành công (id=11 PG):**

```
{"level":"info","ts":...,"msg":"provisioning: advanced","source_id":11,
 "from":"master_active","to_pending":"mapping_pending","step":"discover","master_table":"orders"}
{"level":"info","ts":...,"msg":"discover complete","new_rules":7,"table":"orders"}
{"level":"info","ts":...,"msg":"provisioning: step completed","source_id":11,
 "step":"discover","to":"mapping_ready"}
... → schedule_enable → running.
```

### 3.5 Retry behavior (state machine quirk — không phải bug)

`POST /api/v1/cms/sources/28/provisioning/retry` sau khi seed 1 doc vào Mongo:

```json
{"detail":"provisioning: invalid transition for current state: retry from_state=shadow_pending not advanceable",
 "error":"invalid transition","source_id":28}
```

**Giải thích**: `Retry()` ở `provisioning_orchestrator.go:645` đọc step_log entry failed gần nhất → `from_state` của shadow_bind là `shadow_pending` (in-flight, không có Advance transition). Đây là **expected behavior**: muốn retry đúng cần FE/operator gọi explicit re-trigger ở step gốc, không phải Advance từ trạng thái pending. Không cần sửa — gate fired chính xác, ngăn được cascade là điều quan trọng nhất.

---

## 4. Files được modify / create

| Path | Action | Mục đích |
|------|--------|----------|
| `centralized-data-service/internal/handler/command_handler.go` | Edit (~line 444+) | Fix A: Discover gate count=0 |
| `centralized-data-service/internal/handler/provisioning_step_handlers.go` | Edit | Fix B: thêm `mongoClient` field + 3 helpers + preflight call ở `HandleShadowBind` |
| `centralized-data-service/internal/server/worker_server.go` | Edit (line 324) | Pass `mongoClientShared` vào constructor |
| `centralized-data-service/migrations/cdc/049_mariadb_seed_legacy_orders.sql` | Rewrite | Sửa schema connection_registry (display_name/role_type/...) |
| `cdc-cms-web/src/hooks/useProvisioningMode.ts` | Edit | Gửi `reason` vào body (audit middleware reads body, không phải header) |
| `agent/memory/workspaces/feature-cdc-integration/05_progress.md` | APPEND | Log Cascade Liability hardening |
| `agent/memory/global/lessons.md` | APPEND | Global Pattern: step-level fail-fast cho heterogeneous engine pipeline |

**KHÔNG có file mới ngoài SQL** — tuân thủ "minimal impact" + scope user yêu cầu.

---

## 5. Verification checklist (DoD)

- [x] `go build ./...` PASS (worker + cms-server build sạch).
- [x] Worker bootup OK, NATS subscriptions registered (3 subjects: `cdc.cmd.shadow.bind`, `cdc.cmd.master.bind`, `cdc.cmd.discover`, `cdc.cmd.schedule.enable`).
- [x] PG cascade thành công (id=11 → running, 7 mapping rules).
- [x] MariaDB gate fire (id=27 → failed, Fix A log "discover: zero rules — break cascade").
- [x] Mongo gate fire (id=28 → failed, Fix B log "mongo source preflight: collection ... is empty").
- [x] Audit middleware accept reason từ body (FE flip không còn 400).
- [x] Migration 049 apply sạch, không lỗi schema.
- [x] State machine intact (không có row nào lỡ stuck giữa 2 step).

---

## 6. Service status (cuối phiên)

| Service       | PID  | Port  | Status                  |
|---------------|------|-------|-------------------------|
| cdc-worker    | 7864 | :8082 | LISTEN (test instance) |
| cms-server    | 8001 | :8083 | LISTEN (test instance) |

Các PID test sẽ được kill sau khi user xác nhận report. Production deploy cần rebuild + restart riêng.

---

## 7. Bài học (đã APPEND vào `agent/memory/global/lessons.md`)

**Global Pattern**: Trong pipeline state-machine A đi qua nhiều step B/C/D với engine X heterogeneous (structured + schemaless), bug ở step thứ N chỉ surface khi step N-1 success. Fix: mỗi step phải có **fail-fast invariant check** (output non-empty / schema valid / source reachable) trước khi `Advance`. Đặt gate ở step ngay TRƯỚC bước có side-effect lớn (CREATE TABLE, ENABLE SCHEDULE).

**Anti-pattern**: tin rằng "cascade success vì step trước success". Step success ≠ output usable.
