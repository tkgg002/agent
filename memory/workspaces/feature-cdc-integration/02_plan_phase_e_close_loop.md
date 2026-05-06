# 02 — Plan Phase E: Close-loop fix for residual gaps

**Date**: 2026-05-04
**Sequencing rationale**: blast radius nhỏ → lớn (zero-risk SQL trước, code change sau, security gate cuối).

---

## Strategic order

1. **E3 (G5 prune V1 legacy)** — pure SQL, zero blast radius, idempotent. ETA 5 phút.
2. **E4 (G4 schedule audit)** — diagnostic only, output report. ETA 10 phút.
3. **E1 (G7 multi-tier filter code)** — code change `internal/admin/helpers.go` + unit test + smoke. ETA 30-45 phút.
4. **E2 (G2 Mongo addtest smoke)** — chỉ valid khi E1 land. ETA 15 phút.
5. **E5 (G8 /security-agent gate)** — chạy review, không block phase nếu chỉ MED. ETA 15-20 phút.

**Total ETA**: ~80-100 phút end-to-end (Brain plan + Muscle execute).

---

## E3 — Prune V1 legacy (G5)

### Critical files (read-only Brain check)
- `deployments/sql/cdc/prune_legacy_v1_bindings.sql` (existing, idempotent)
- `migrations/cdc/035_v2_backfill_legacy_registry.sql` (gốc 10 row legacy)

### Approach
1. Brain: verify file SQL tồn tại + nội dung idempotent (`WHERE is_active=true`).
2. Muscle (foreground Bash): chạy script qua `docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw < ...`.
3. Muscle: re-run script lần 2 → expect 0 row updated (idempotent verify).
4. Muscle: query verify count `legacy_*` active = 0.

### Verification
- `SELECT count(*) FROM cdc_system.source_object_registry WHERE object_code LIKE 'legacy\_%' ESCAPE '\' AND is_active=true;` = 0.
- Lần 2 re-run script: output `pruned_sources=10` lần 1, `0` lần 2.

### Rollback
SQL không drop row, chỉ `is_active=false` + stamp `notes`. Rollback: `UPDATE … SET is_active=true WHERE object_code LIKE 'legacy\_%' AND notes LIKE '%pruned by deployments%';`

---

## E4 — Schedule audit cho `orders_addtest` (G4)

### Critical queries
```sql
-- 1. binding state
SELECT mb.id, mb.binding_code, mb.is_active, sor.object_code
  FROM cdc_system.master_binding mb
  JOIN cdc_system.source_object_registry sor ON sor.id = mb.source_object_id
 WHERE sor.object_code = 'addtest_pg_orders' OR mb.binding_code LIKE '%orders_addtest%';

-- 2. schedule state
SELECT ts.* FROM cdc_system.transmute_schedule ts
  JOIN cdc_system.master_binding mb ON mb.id = ts.master_binding_id
 WHERE mb.binding_code LIKE '%orders_addtest%';

-- 3. master DDL exists?
\dt dw_src_local_pg_source.*

-- 4. shadow has data?
SELECT count(*) FROM shadow_src_local_pg_source.orders_addtest;

-- 5. cdc_activity_log recent for this binding
SELECT timestamp, action_type, payload FROM cdc_system.cdc_activity_log
 WHERE payload::text LIKE '%orders_addtest%' ORDER BY timestamp DESC LIMIT 20;
```

### Output
File `report_g4_diag_<ts>.md` với:
- Snapshot 5 query trên
- Root cause classification: (a) `is_enabled=false` schedule, (b) binding `is_active=false`, (c) master DDL chưa tạo, (d) transmute fail loop, (e) bug khác
- Recommendation: enable schedule / re-trigger provisioning / archive / fix code

### Rollback
N/A (chỉ đọc).

---

## E1 — Multi-tier filter close-loop (G7) — **CHÍNH**

### Critical files (Muscle EDIT)
- `internal/admin/helpers.go::extendDebeziumInclude` — extend logic.
- `internal/admin/helpers.go` (new helper `extendDatabaseList`).
- `internal/admin/server_test.go` — thêm 3 test case.
- `internal/admin/types.go::RegisterSourceResponse` — thêm field `Warnings []string`.

### Approach (per architecture per L-multi-tier-filter-mirror lesson)
1. Tách `extendDebeziumInclude` thành 2 step: tier-cao (`extendDatabaseList`) trước, tier-thấp (`extendCollectionList` / `extendTableList`) sau.
2. Per-engine adapter:
   ```go
   switch sourceType {
   case "mongodb":
       extendDatabaseList(config, "database.include.list", databaseName)
       extendCollectionList(config, "collection.include.list", databaseName+"."+collectionName)
   case "mysql", "mariadb":
       extendDatabaseList(config, "database.include.list", databaseName)
       extendTableList(config, "table.include.list", databaseName+"."+tableName)
   case "postgres":
       if config["database.dbname"] != databaseName {
           return nil, fmt.Errorf("pg connector locked to db=%s, requested db=%s", config["database.dbname"], databaseName)
       }
       extendTableList(config, "table.include.list", schemaName+"."+tableName)
   }
   ```
3. Helper `extendDatabaseList(config, key, value)`:
   - Parse comma-separated list
   - Dedup
   - Append nếu chưa có
   - Return tuple `(newValue, wasAdded bool)` để emit warning
4. Response `RegisterSourceResponse.Warnings`: nếu `wasAdded=true` ở tier-cao → push warning "database 'X' was just added to debezium include — first event may be delayed; connector task may need a moment to snapshot".
5. Helper update payload PUT `/connectors/{name}/config` nguyên cấu trúc (Debezium yêu cầu PUT toàn bộ config, không PATCH).

### Unit tests
```go
func TestExtendDatabaseList_NewValue(t *testing.T) { /* expect appended, wasAdded=true */ }
func TestExtendDatabaseList_AlreadyPresent(t *testing.T) { /* expect no-op, wasAdded=false */ }
func TestExtendDebeziumInclude_Mongo_BothTiers(t *testing.T) { /* config trước có db1 + col1; sau khi extend db2.col2 → both list updated */ }
func TestExtendDebeziumInclude_PG_DBLockMismatch(t *testing.T) { /* expect error */ }
```

### Live verification (sau code land)
```bash
# 1. Snapshot connector config trước
curl -s localhost:18083/connectors/goopay-mongodb-cdc/config | tee /tmp/before.json

# 2. PUT register collection mới ở namespace mới (e.g. "phase_e_smoke")
TS=$(date +%s)
curl -X POST http://127.0.0.1:8090/v1/sources/register \
  -H "Authorization: Bearer $CDC_ADMIN_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_code": "phase_e_smoke_'"$TS"'",
    "source_type": "mongodb",
    "database_name": "phase_e_ns_'"$TS"'",
    "collection_name": "items",
    ...
  }'

# 3. Verify config update
curl -s localhost:18083/connectors/goopay-mongodb-cdc/config | jq '.["database.include.list"], .["collection.include.list"]'
# Expect: database includes "phase_e_ns_<ts>", collection includes "phase_e_ns_<ts>.items"

# 4. Verify response có warnings
# Expect: response.warnings contains "database 'phase_e_ns_<ts>' was just added..."
```

### Rollback
- Code: revert commit (Muscle).
- Connector config: PUT lại `/tmp/before.json`.

---

## E2 — Mongo `payment_bills_addtest` smoke (G2)

### Pre-condition
E1 đã land + admin-api restart picked up code mới.

### Steps
```bash
# 1. PUT register payment_bills_addtest qua admin-api
curl -X POST http://127.0.0.1:8090/v1/sources/register \
  -H "Authorization: Bearer $CDC_ADMIN_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_code": "addtest_mongo_bills_v2",
    "source_type": "mongodb",
    "database_name": "payment-bill-service",
    "collection_name": "payment_bills_addtest",
    ...
  }'

# 2. INSERT 1 doc vào source
docker exec gpay-mongo mongosh "mongodb://localhost:27017/payment-bill-service" \
  --eval "db.payment_bills_addtest.insertOne({_id: 'phase-e-smoke-1', amount: 100})"

# 3. Wait 30s → query shadow
sleep 30
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM shadow_mongo_payment_bill_default.payment_bills_addtest;"
# Expect: >= 1

# 4. Wait 60s cron tick → query master
sleep 60
docker exec gpay-postgres-dest psql -U gpay_admin -d goopay_dest -c \
  "SELECT count(*) FROM dw_mongo_payment_bill_default.payment_bills_addtest;"
# Expect: >= 1
```

### Rollback
- DELETE source doc; admin-api không có endpoint un-register, để row registry là `is_active=false` và Debezium stop streaming khi DELETE happens.

---

## E5 — `/security-agent` gate (G8)

### Approach
1. Brain delegate `/security-agent` skill với scope: review `cmd/admin-api/main.go`, `internal/admin/server.go`, `internal/admin/source_register.go`, `internal/admin/helpers.go`.
2. Output: severity-tagged list (HIGH/MED/LOW).
3. Phase E phạm vi: chỉ chạy gate + record output. Fix HIGH severity → Phase F1 (tách riêng).

### Verification
- File `report_security_agent_admin_api_<ts>.md` (do `/security-agent` skill output).
- Append summary vào `05_progress.md`.

### Rollback
N/A.

---

## End-to-end verification (sau toàn phase)

```bash
# 1. G5 prune state
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM cdc_system.source_object_registry WHERE object_code LIKE 'legacy\_%' ESCAPE '\' AND is_active=true;"
# Expect: 0

# 2. G4 audit report exists
ls agent/memory/workspaces/feature-cdc-integration/report_g4_diag_*.md

# 3. G7 fix verify both tiers updated
curl -s localhost:18083/connectors/goopay-mongodb-cdc/config | \
  jq '{db: .["database.include.list"], col: .["collection.include.list"]}'
# Expect: both lists include phase_e namespace + collection

# 4. G2 Mongo addtest E2E
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM shadow_mongo_payment_bill_default.payment_bills_addtest;"
# Expect: >= 1

# 5. G8 security gate output exists
ls agent/memory/workspaces/feature-cdc-integration/report_security_agent_*.md

# 6. Worker stability check
docker inspect gpay-cdc-worker --format '{{.State.Status}} restartCount={{.RestartCount}}'
# Expect: status=running, restartCount unchanged from start of phase
```

---

## Files được tạo / sửa

| Path | Action | Owner |
|---|---|---|
| `01_requirements_phase_e_close_loop.md` | NEW | Brain |
| `02_plan_phase_e_close_loop.md` (this file) | NEW | Brain |
| `08_tasks_phase_e_close_loop.md` | NEW | Brain |
| `09_tasks_solution_phase_e_close_loop.md` | NEW | Brain |
| `internal/admin/helpers.go` | EDIT | Muscle |
| `internal/admin/types.go` | EDIT | Muscle |
| `internal/admin/server_test.go` | EDIT | Muscle (3 test mới) |
| `internal/admin/source_register.go` | EDIT (use new return tuple) | Muscle |
| `report_g4_diag_<ts>.md` | NEW | Muscle (output diagnostic) |
| `report_security_agent_<ts>.md` | NEW | `/security-agent` |
| `report_phase_e_close_loop_<ts>.md` | NEW | Brain (cuối phase) |
| `05_progress.md` | APPEND ×5 | Brain |
| `lessons.md` | APPEND ≥1 nếu phase phát sinh lesson | Brain |
