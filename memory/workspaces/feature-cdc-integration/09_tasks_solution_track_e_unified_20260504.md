# Task Solution: Track E Unified Plan (5 quyết định 2026-05-04)

**Date**: 2026-05-04
**Status**: User approved 5 decisions → Brain document → Muscle execute (subset)
**Scope**: B3 + G3 + G4 + G1 + D1 (đã list trong `report_pending_options_and_unified_plan_20260504.md`)
**Brain Boundary** (CLAUDE.md §12): Brain document + runtime ops (psql ad-hoc, docker exec, curl). Code (.go) + IaC (.yml repo files) → Muscle.

---

## 1. B3 — Logical-clone fan-out (Worker dispatch 1 event → N shadows)

### Decision
Giữ nguyên Debezium `table.include.list` (KHÔNG thêm `*_addtest`). Worker khi nhận 1 event của master object (e.g., `cdc.gpay.public.orders`) sẽ FAN-OUT sang **nhiều** shadow tables — cả master shadow (`shadow_*.orders`) và clone shadow (`shadow_*.orders_addtest`).

### Why this option
- Source DB không phải maintain duplicate rows (1 INSERT vào `orders` → fan-out logical, không phải physical replication).
- Add/remove logical-clone không cần touch Debezium config (zero downtime).
- Phù hợp `source_locator_json` đã có sẵn trong `source_object_registry` để khai báo "logical clone of <master_id>".

### Critical files (Muscle to modify)

#### File 1: `internal/service/registry_service.go`
**Change**: `ResolveSourceRoute(sourceDB, sourceTable string) *ResolvedSourceRoute` (single) → `ResolveSourceRoutes(sourceDB, sourceTable string) []*ResolvedSourceRoute` (multiple).
- Cache index: `sourceToTargets map[string][]string` (1:N) thay vì `sourceToTarget map[string]string` (1:1) ở line 82.
- Khi build cache (line 80-82), join `source_object_registry` để lấy logical-clones của 1 source object: `WHERE source_locator_json->>'logical_clone_of' = '<master_source_object_id>'` (cần thêm convention key `logical_clone_of` trong `source_locator_json`).

#### File 2: `internal/handler/event_handler.go`
**Change** at `processEvent` line 70:
```go
// BEFORE:
route := h.registrySvc.ResolveSourceRoute(sourceDB, sourceTable)
if route == nil { return nil }

// AFTER:
routes := h.registrySvc.ResolveSourceRoutes(sourceDB, sourceTable)
if len(routes) == 0 { return nil }

for _, route := range routes {
    // ... existing logic from line 75–134, but indexed per-route
    record := &model.UpsertRecord{...} // unchanged shape
    h.batchBuffer.Add(record)
}
```

#### File 3: `internal/handler/event_handler.go::handleDelete` (line 139)
**Change**: similar fan-out — DELETE event propagates `_deleted=TRUE` cho mỗi shadow table trong fan-out group.

#### File 4: `migrations/cdc/050_logical_clone_locator_keys.sql` (NEW)
**Migration**: backfill `source_locator_json` cho 3 addtest sources:
```sql
-- id=29 orders_addtest → logical-clone of id=11 (orders)
UPDATE cdc_system.source_object_registry
   SET source_locator_json = source_locator_json ||
                             jsonb_build_object('logical_clone_of', 11,
                                                'fan_out_role', 'clone')
 WHERE id = 29;

-- id=30 legacy_orders_addtest → logical-clone of id=27 (legacy_orders) hoặc 11 (nếu chưa active)
UPDATE cdc_system.source_object_registry
   SET source_locator_json = source_locator_json ||
                             jsonb_build_object('logical_clone_of', 27,
                                                'fan_out_role', 'clone')
 WHERE id = 30;

-- id=31 payment_bills_addtest → logical-clone of id=28 (payment_bills)
UPDATE cdc_system.source_object_registry
   SET source_locator_json = source_locator_json ||
                             jsonb_build_object('logical_clone_of', 28,
                                                'fan_out_role', 'clone')
 WHERE id = 31;
```

### Tests (Muscle to add)

`internal/service/registry_service_test.go::TestResolveSourceRoutes_FanOut`:
- Setup 2 source_object_registry rows: master (id=A, source_table=orders, target=shadow_*.orders) + clone (id=B, source_table=orders, target=shadow_*.orders_addtest, locator.logical_clone_of=A)
- Assert `ResolveSourceRoutes("public", "orders")` returns 2 routes.

`internal/handler/event_handler_test.go::TestProcessEvent_FanOut`:
- Mock registry returns 2 routes, assert batchBuffer.Add called 2 times with different schema/table.

### Verify (post-deploy)

```bash
# Insert 1 row vào source orders
docker exec -i gpay-postgres-source psql -U src_user -d goopay_source <<SQL
INSERT INTO public.orders (user_id, amount, status, notes)
  VALUES (8001, 999, 'pending', 'b3-fanout-test');
SQL

# Wait 30s
sleep 30

# Verify 2 shadow tables BOTH có row mới
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "
SELECT 'main shadow' AS t, count(*) FROM shadow_src_local_pg_source.orders WHERE notes='b3-fanout-test'
UNION ALL
SELECT 'addtest clone', count(*) FROM shadow_src_local_pg_source.orders_addtest WHERE notes='b3-fanout-test';"
# Expect: 1, 1
```

### Risks
- **Backwards compat**: existing single-route consumers (Mongo, MariaDB main path) phải work với new API. Solution: `ResolveSourceRoutes` trả về `[]*ResolvedSourceRoute` length 1 cho non-fan-out case → caller code path đồng nhất.
- **Schema drift between master + clone**: nếu master shadow có column X mà clone không có → fan-out write sang clone fail. Mitigation: provisioning đảm bảo clone shadow inherit cùng column set như master shadow (đã đúng với current `MasterDDLGenerator` pattern).
- **DLQ duplication**: 1 source event fail → 2 DLQ rows? Solution: DLQ dedupe theo (kafka_offset, target_table). Acceptable as-is.

---

## 2. G3 — Deploy otel-collector

### Decision
Add `otel-collector` container vào `deployments/docker-compose.yml`, expose 4318 (OTLP HTTP). Worker config `endpoint: http://otel-collector:4318` (hiện đang `localhost:4318` → connection refused mỗi 5s).

### Critical files (Muscle to modify)

#### File 1: `deployments/docker-compose.yml`
Add service block:
```yaml
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.96.0
    container_name: gpay-otel-collector
    restart: unless-stopped
    command: ["--config=/etc/otel-collector-config.yml"]
    volumes:
      - ./otel-collector-config.yml:/etc/otel-collector-config.yml:ro
    ports:
      - "14318:4318"  # OTLP HTTP
      - "14317:4317"  # OTLP gRPC
    networks:
      - cdc-network
```

#### File 2: `deployments/otel-collector-config.yml` (NEW)
```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  batch:
    timeout: 5s
exporters:
  debug:
    verbosity: basic
  file/logs:
    path: /tmp/cdc-logs.json
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug, file/logs]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
```

#### File 3: `centralized-data-service/config/config-local.yml:81`
Update endpoint:
```yaml
otel:
  enabled: true
  serviceName: cdc-worker
  endpoint: http://otel-collector:4318  # was: http://localhost:4318
```

#### File 4: env override at `deployments/docker-compose.yml::cdc-worker`
Add:
```yaml
    environment:
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```

### Verify

```bash
docker compose -f deployments/docker-compose.yml up -d otel-collector
docker restart gpay-cdc-worker
sleep 15
docker logs gpay-cdc-worker --tail 50 | grep -c 'connection refused'
# Expect: 0

docker logs gpay-otel-collector --tail 30 | grep -i 'received'
# Expect: log entries showing OTLP receive
```

### Risks
- Container thêm consume RAM (~100MB). Acceptable.
- File exporter `/tmp/cdc-logs.json` không bounded → cần rotate. Mitigation later: switch sang Loki/Elasticsearch khi production.

---

## 3. G4 — Mongo recon connection fix

### Decision
Fix env `MONGODB_URL` cho cdc-worker container để ReconCore khởi tạo thành công, kích hoạt full recon engine T1/T2/T3.

### Critical files

#### File 1: `deployments/docker-compose.yml::cdc-worker.environment`
Add:
```yaml
    environment:
      - MONGODB_URL=mongodb://gpay-mongo:27017/?replicaSet=rs0
      # (dùng service name `gpay-mongo` trong docker network, không phải localhost)
```

#### File 2: Verify `centralized-data-service/cmd/worker/main.go`
Đảm bảo recon code path dùng env `MONGODB_URL` thay vì hardcoded `localhost`. Grep:
```bash
grep -n 'MONGODB_URL\|MongoDB URL\|mongo.*config' centralized-data-service/cmd/worker/main.go
```

### Verify

```bash
docker restart gpay-cdc-worker
sleep 30
docker logs gpay-cdc-worker --since 5m | grep -E 'reconCore|MongoDB' | head -10
# Expect: "MongoDB connection established", recon ticks RUNNING (not SKIPPED)
docker logs gpay-cdc-worker --since 5m | grep -c 'reconCore is nil'
# Expect: 0
```

### Risks
- Recon T3 (full hash diff) tốn CPU/IO. Acceptable on dev, monitor on prod.
- Recon có thể phát hiện drift sẵn có giữa shadow và master (do B6 fix mới deploy 2 ngày → có thể có row chưa sync). Dự kiến: 0 drift cho `orders_fact` (5 rows), drift mạnh cho `orders_addtest` (vì shadow=0). → đợi B3 fan-out xong rồi recon mới ý nghĩa.

---

## 4. G1 — Archive 6 failed sources (Brain runtime ops — SQL ad-hoc)

### Decision
Archive 6 sources `provisioning_state='failed'` (id 19/23/24/25/27/28). Drop 3 orphan shadow tables (`orders_e2e_d_v2/v3/v4`).

### Brain executes (CLAUDE.md §12 — runtime SQL OK, không touch repo files)

```sql
-- 1. Archive 6 failed sources
UPDATE cdc_system.source_object_registry
   SET provisioning_state = 'archived',
       updated_at = NOW(),
       last_step_error = COALESCE(last_step_error, '') ||
                         E'\n[archived 2026-05-04 by Brain — G1 cleanup]'
 WHERE id IN (19, 23, 24, 25, 27, 28)
   AND provisioning_state = 'failed';

-- 2. Deactivate associated bindings (defensive)
UPDATE cdc_system.shadow_binding
   SET is_active = false,
       updated_at = NOW()
 WHERE source_object_id IN (19, 23, 24, 25, 27, 28);

UPDATE cdc_system.master_binding
   SET is_active = false,
       updated_at = NOW()
 WHERE source_object_id IN (19, 23, 24, 25, 27, 28);

-- 3. Drop orphan shadow tables (chỉ 3 cái stale, KHÔNG drop v5 vì id=26 đang running)
DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v2 CASCADE;
DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v3 CASCADE;
DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v4 CASCADE;

-- 4. Verify
SELECT id, source_object_name, provisioning_state, is_active
  FROM cdc_system.source_object_registry
 WHERE id IN (19, 23, 24, 25, 27, 28)
 ORDER BY id;
```

### Risks
- `legacy_orders` (id=27) và `payment_bills` (id=28) là master object — nếu archive thì addtest clone của chúng (id=30, id=31) bị orphan locator (B3 fan-out plan reference id=27/28).
  - Solution: trước khi archive id=27/28, retry chúng (chuyển từ failed → draft → re-trigger provisioning). Sau khi running, B3 fan-out sẽ resolve.
  - **Quyết định execute**: KHÔNG archive id=27/28 ngay; chỉ archive 4 sources thực sự stale (19/23/24/25). Document để user biết.

### Updated execute (revised)

```sql
-- Archive CHỈ 4 sources stale (giữ id=27, 28 cho B3 fan-out plan resolve sau)
UPDATE cdc_system.source_object_registry
   SET provisioning_state = 'archived',
       updated_at = NOW(),
       last_step_error = COALESCE(last_step_error, '') ||
                         E'\n[archived 2026-05-04 by Brain — G1 cleanup]'
 WHERE id IN (19, 23, 24, 25)
   AND provisioning_state = 'failed';

UPDATE cdc_system.shadow_binding
   SET is_active = false, updated_at = NOW()
 WHERE source_object_id IN (19, 23, 24, 25);

UPDATE cdc_system.master_binding
   SET is_active = false, updated_at = NOW()
 WHERE source_object_id IN (19, 23, 24, 25);

DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v2 CASCADE;
DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v3 CASCADE;
DROP TABLE IF EXISTS shadow_src_local_pg_source.orders_e2e_d_v4 CASCADE;
```

---

## 5. D1 — Schema Schism coexist (Brain document — `agent/memory/global/conventions.md`)

### Decision
Coexist V1 (PK = `id`, TEXT) + V2 (PK = `_gpay_id`, BIGINT). Transmuter dynamic detect.

### Brain APPENDs `agent/memory/global/conventions.md`

```markdown
## 2026-05-04 — Schema Schism Coexistence (CDC Shadow Convention)

### Context
CDC pipeline có 2 generation shadow table convention:

| Generation | PK Column | PK Type | Generator | Note |
|-----------|-----------|---------|-----------|------|
| V1 (legacy) | `id` | TEXT | `internal/service/schema_adapter.go` (Provisioning path) | Source PK as-is, no Sonyflake |
| V2 (new) | `_gpay_id` | BIGINT | `internal/sinkworker/schema_manager.go` (Direct sink path) | Sonyflake-generated, allows cross-source dedup |

### Decision
**Coexist** — KHÔNG migrate V1 → V2 forcibly. Transmuter `internal/service/transmuter.go`
auto-detects via `information_schema.columns`:

```go
// transmuter.go:222
EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_schema = sb.shadow_schema
          AND table_name = sb.shadow_table
          AND column_name = '_gpay_id')
THEN '_gpay_id' ELSE 'id' END AS shadow_pk
```

### Rule for new shadows
- Mới: V2 default (`_gpay_id` BIGINT). Sonyflake gen ở write path.
- Migrate V1 → V2 chỉ khi có lý do nghiệp vụ rõ (e.g., cross-source dedup).
- KHÔNG ALTER existing V1 shadow để add `_gpay_id` — risk lock + complicates transmute logic.
```

---

## 6. Execution Order (proposed)

| Step | Task | Owner | Estimated time |
|------|------|-------|----------------|
| 1 | Document this file (09_tasks_solution_track_e_unified_20260504.md) | Brain | DONE |
| 2 | Append D1 to conventions.md | Brain | 5m |
| 3 | Execute G1 archive 4 sources + drop 3 orphan shadows | Brain (SQL ad-hoc) | 5m |
| 4 | Execute B8 install MariaDB plugin + create connector (still needed for legacy_orders main ingest) | Brain (docker exec + curl) | 15m |
| 5 | Delegate Muscle: B3 logical-clone fan-out code (4 files: registry_service.go, event_handler.go, migration 050) | Muscle | 1-2 days |
| 6 | Delegate Muscle: G3 otel-collector deployment (4 files: docker-compose.yml, otel-collector-config.yml, config-local.yml, env) | Muscle | 1 day |
| 7 | Delegate Muscle: G4 Mongo recon URL fix (1 file: docker-compose.yml env block) | Muscle | 30m |
| 8 | E2E smoke test sau Muscle deploy | Brain (verify) | 30m |
| 9 | Update progress log + this report | Brain | 10m |

---

## 7. Definition of Done (cả Track E)

- ✅ B3 fan-out: insert 1 row vào `public.orders` source → cả `shadow_*.orders` VÀ `shadow_*.orders_addtest` đều +1 row
- ✅ B8: `/connector-plugins` lists `MySqlConnector`, `cdc-mariadb-source` connector RUNNING
- ✅ Track E E2E: 3 master DW addtest tables (`dw_*.orders_addtest`, `legacy_orders_addtest`, `payment_bills_addtest`) tất cả có >0 rows
- ✅ G3: worker logs SẠCH (no connection refused), otel-collector UP & receiving
- ✅ G4: recon scheduler tick RUNNING (not SKIPPED), reconCore non-nil
- ✅ G1: 0 stale "failed" sources (chỉ id=27/28 còn cho fan-out resolution)
- ✅ D1: conventions.md updated

---

## 8. Skills used (CLAUDE.md §0)

- `Bash` — psql/docker/curl audit
- `Read` / `grep` — code archaeology cho B3 plan
- `Write` — sinh solution doc
- `TaskCreate` / `TaskUpdate` — track 8 tasks tiếp theo
- Governance: §3 (Plan & Verify), §11 (APPEND-only), §12 (Brain prohibition — code → Muscle), §14 (pre-flight)
- Lessons applied: cascade-liability, three-layer-trust-failure (B8 compose ≠ runtime plugin), schema-from-real-DB

---

## Section 7. NEW BLOCKER B9 — Avro Union Decode (Discovered 2026-05-04 03:44 UTC)

### Symptom
Sau khi B8 deploy thành công (MariaDB plugin v2.5.4.Final loaded + cdc-mariadb-source connector RUNNING), Kafka topic `cdc.mariadb.goopay_legacy_maria.legacy_orders_addtest` đã consume hết 5 records (lag=0). Worker batch upsert fail với:

```
upsert failed pk=4
error: failed to encode args[2]: unable to encode 
  map[string]interface {}{"string":"2026-04-29T07:44:47Z"} 
  into binary format for timestamp (OID 1114): cannot find encode plan
```

SQL tail confirms raw map dump injected as literal:
```
INSERT INTO shadow_mariadb_legacy_default.legacy_orders_addtest 
  (..., 'created_at' = 'map[string:2026-04-29T07:44:47Z]', ...)
```

### Root cause
Avro schema cho cột nullable type emit union `{"type": "null", "string": "..."}`. Khi schema_registry deserialize, value trả về Go map dạng `map[string]interface{}{"string": "..."}`. Worker tại `internal/handler/batch_buffer.go:163` truyền thẳng map vào pgx encoder → encoder không có plan cho map → fail.

V2 main path (Postgres orders → orders_fact) KHÔNG bị vì các cột nullable đều có default value/non-null trong source schema, nên Avro emit raw value (không union envelope).

### Scope của fix (Muscle)
- **File**: `internal/handler/kafka_consumer.go` (hoặc helper utility shared) — thêm bước unwrap union cho mọi field type sau Avro decode:
  ```go
  // Unwrap Avro union: {"string": "x"} → "x"; {"long": 123} → 123; {"null": nil} → nil
  func unwrapAvroUnion(v any) any {
      m, ok := v.(map[string]interface{})
      if !ok || len(m) != 1 { return v }
      for _, val := range m { return val }  // single-key map
      return v
  }
  ```
- **Apply point**: trước khi build `MappedData` map (xem `kafka_consumer.go::processMessage` hoặc `event_handler.go::processEvent`). Apply recursively cho mọi field value.
- **Test**: Re-consume cdc.mariadb.* events → verify shadow_mariadb_legacy_default.legacy_orders_addtest có 5 rows + valid timestamp.

### DoD
- [ ] Unit test: `unwrapAvroUnion({"string":"x"}) == "x"`, `unwrapAvroUnion({"null":nil}) == nil`, `unwrapAvroUnion("plain") == "plain"`.
- [ ] Integration test: post 1 fake Avro union event → assert `MappedData["created_at"]` is `time.Time` not `map`.
- [ ] Smoke test: docker restart cdc-worker → consume cdc.mariadb topic → 5 rows in shadow_mariadb_legacy_default.legacy_orders_addtest.

### Blast radius
- **Affected**: MariaDB và bất kỳ source MySQL nào emit Avro union → tất cả pipeline cdc.mariadb.* hiện đang fail.
- **Side-effect**: Sau fix, Postgres path sẽ qua hàm unwrap nhưng vô hại vì input không phải map → return as-is.
- **Risk**: Thấp. Hàm unwrap thuần, idempotent, không alter non-union values.

### Status
- 2026-05-04 03:44 UTC: Brain discovered B9 sau khi B8 plugin install thành công.
- 2026-05-04 03:46 UTC: Documented + delegate Muscle.
