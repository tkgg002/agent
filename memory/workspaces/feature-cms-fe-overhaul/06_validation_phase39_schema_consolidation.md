# Phase 39 — Validation (template — Muscle fill sau exec)

## Status: ⏳ pending exec

## DoD checklist

| # | Criteria | Verify command | Expected | Result |
|---|---|---|---|---|
| 1 | public rỗng | `SELECT count(*) FROM pg_tables WHERE schemaname='public';` | `0` | ⏳ |
| 2 | schema map clean | `\dn` | `cdc_system, cdc_auth_service, public, pg_*` | ⏳ |
| 3 | admin_actions in cdc_system | `\d+ cdc_system.admin_actions` | partitioned + 4 partitions | ⏳ |
| 4 | cdc_alerts in cdc_system | `\dt cdc_system.cdc_alerts` | exist | ⏳ |
| 5 | auth_users in cdc_auth_service | `SELECT username,role FROM cdc_auth_service.auth_users;` | `admin\|admin` row | ⏳ |
| 6 | cdc_internal gone | `SELECT count(*) FROM pg_namespace WHERE nspname='cdc_internal';` | `0` | ⏳ |
| 7 | search_path | `SHOW search_path;` | `cdc_system, cdc_auth_service, public` | ⏳ |
| 8 | Login admin | `curl -X POST :8081/api/auth/login {username,password}` | 200 + JWT | ⏳ |
| 9 | 11 CMS endpoints | bash loop verify | 11/11 = 200 | ⏳ |
| 10 | Debezium connector RUNNING | `curl :18083/connectors/.../status` | `RUNNING` | ⏳ |
| 11 | Worker discoverTopics healthy | `tail /tmp/cdc-worker.log` | loop, no panic | ⏳ |
| 12 | Build all 4 services | `go build ./...` × 4 | exit 0 | ⏳ |

## Operator-flow (11 endpoints)

| HTTP | Endpoint | Body head 200B |
|---:|---|---|
| ⏳ | `/api/schema-changes/pending?status=pending&page_size=1` | |
| ⏳ | `/api/v1/source-objects/stats` | |
| ⏳ | `/api/v1/source-objects?page=1&page_size=100` | |
| ⏳ | `/api/v1/shadow-bindings?page=1&page_size=100` | |
| ⏳ | `/api/v1/schema-proposals?status=pending` | |
| ⏳ | `/api/v1/schedules` | |
| ⏳ | `/api/activity-log?page=1&page_size=30` | |
| ⏳ | `/api/activity-log/stats` | |
| ⏳ | `/api/failed-sync-logs?page_size=50` | |
| ⏳ | `/api/worker-schedule` | |
| ⏳ | `/api/v1/source-objects?page_size=500` | |

## Auto-flow

- Debezium `goopay-mongodb-cdc` status: ⏳
- Kafka topics `cdc.goopay.*`: ⏳
- Worker log tail: ⏳

## Schema-level final state

```
\dn output:
⏳

\dt cdc_system.* count:
⏳ (expected ≥25)

\dt cdc_auth_service.* count:
⏳ (expected = 1)

\dt public.* count:
⏳ (expected = 0)
```

## Re-verify command pack

```bash
# 1) Schema map
docker exec gpay-postgres psql -U user -d goopay_dw -c "\
  SELECT schemaname, count(*) FROM pg_tables \
  WHERE schemaname NOT LIKE 'pg_%' AND schemaname <> 'information_schema' \
  GROUP BY schemaname ORDER BY schemaname;"

# 2) Search_path
docker exec gpay-postgres psql -U user -d goopay_dw -c "SHOW search_path;"

# 3) Build × 4
for d in cdc-auth-service cdc-cms-service centralized-data-service; do
  (cd cdc-system/$d && go build ./...) || echo "FAIL: $d"
done

# 4) Login + 11 endpoints (xem 02_plan_phase39 Step 9)
```

## API Impact Matrix (post-exec MUST verify hết)

Quét code path 3 keyword bị move schema (`auth_users` → `cdc_auth_service`, `admin_actions`+`cdc_alerts` → `cdc_system`). Không endpoint nào được skip — Muscle phải probe đủ trước khi báo PASS.

### Group 1 — Auth endpoints (đụng `cdc_auth_service.auth_users`)

| Endpoint | Method | Service | Code path | Read/Write |
|---|---|---|---|---|
| `/api/auth/login` | POST | cdc-auth-service:8081 | `userRepo.GetByUsername` | READ |
| `/api/auth/register` | POST | cdc-auth-service:8081 | `userRepo.ExistsByUsername` + `ExistsByEmail` + `Create` | READ + WRITE |
| `/api/auth/refresh` | POST | cdc-auth-service:8081 | `userRepo.GetByID` | READ |

**Note**: cdc-cms-service JWT middleware (`internal/middleware/rbac.go`) chỉ verify chữ ký JWT — KHÔNG hit DB → không bị ảnh hưởng schema move.

### Group 2 — Alert endpoints (đụng `cdc_system.cdc_alerts`)

| Endpoint | Method | Service | Code path | Read/Write |
|---|---|---|---|---|
| `/api/alerts/active` | GET | cdc-cms:8083 | `alerts_handler.Active` → GORM `Alert` model | READ |
| `/api/alerts/silenced` | GET | cdc-cms:8083 | `alerts_handler.Silenced` | READ |
| `/api/alerts/history` | GET | cdc-cms:8083 | `alerts_handler.History` | READ |
| `/api/alerts/:fingerprint/ack` | POST | cdc-cms:8083 | `alerts_handler.Ack` → `alert_manager.Updates` | WRITE + AUDIT |
| `/api/alerts/:fingerprint/silence` | POST | cdc-cms:8083 | `alerts_handler.Silence` → `alert_manager.Updates` | WRITE + AUDIT |

**Background writer**: `service/alert_manager.go` qua `system_health_collector` tick (mỗi 30-60s) — `tx.Create(&Alert)` + `Updates(...)`. Nếu `TableName()` sai schema → log spam, panic loop. Tail `/tmp/cdc-cms.log` ≥ 60s sau restart để confirm.

### Group 3 — Audit-wrapped mutation endpoints (đụng `cdc_system.admin_actions` write)

Audit middleware `destructive.Audit` raw INSERT vào `admin_actions` cho TẤT CẢ destructive endpoint. Total ~30 endpoints — Muscle smoke-test ≥1 endpoint MỖI nhóm:

| Group | Sample endpoint smoke test | Verify |
|---|---|---|
| Reconciliation | `POST /api/reconciliation/check` (no body) | 200 + audit row INSERT |
| Failed-sync retry | `POST /api/failed-sync-logs/<existing-id>/retry` (404 nếu không có id, OK) | log INSERT attempt |
| Tools | `POST /api/tools/reset-debezium-offset` | 200/202 + audit row |
| Schema-changes | `POST /api/schema-changes/<id>/reject {"reason":"phase39 smoke"}` | 200/404 + audit row |
| Source-objects | `PATCH /api/v1/source-objects/registry/12 {"priority":"critical"}` | 200 + audit row |
| Mapping-rules | `POST /api/mapping-rules/reload` | 200 + audit row |
| Worker-schedule | `PATCH /api/worker-schedule/5 {"is_enabled":true}` | 200 + audit row |
| Wizard | `POST /api/v1/wizard/sessions {"step":"connect"}` | 201 + audit row |
| Connectors | `DELETE /api/v1/system/connectors/nonexistent` | 404 vẫn ghi audit ATTEMPT |
| Alerts (đã list trên) | covered ở Group 2 | |

Verify `admin_actions` row sau test:
```bash
docker exec gpay-postgres psql -U user -d goopay_dw -c \
  "SELECT count(*), max(created_at) FROM cdc_system.admin_actions;"
# Expected: count > 0, max(created_at) trong vòng 5 phút
```

### Group 4 — 11 operator read endpoints (giữ từ Phase 38)

Đã list ở Step 9 plan + bảng "Operator-flow (11 endpoints)" phía trên. Phải re-verify để đảm bảo schema move không gãy thứ khác.

### Group 5 — Search_path silent dependencies (KHÔNG có code rõ ràng)

Bất kỳ Raw SQL nào không qualify schema sẽ resolve qua `search_path = cdc_system, cdc_auth_service, public`. Nếu có handler nào ngầm dùng `auth_users` (không qualify), nó sẽ tìm ở `cdc_system` trước → miss → `cdc_auth_service` → hit. Đây là defense-in-depth do migration 042 cung cấp. Test: nếu `cdc-auth-service` GORM `TableName()` quên qualify, search_path vẫn cứu — nhưng em đã đặt qualify để không phụ thuộc.

**Audit**: `grep -rnE "FROM\s+(auth_users|admin_actions|cdc_alerts)\b|INTO\s+(auth_users|admin_actions|cdc_alerts)\b" cdc-system/` sau exec — phải = 0 hit (không có raw SQL chưa qualify).

## Extended Verify Pack (post-exec, dán vào shell)

```bash
set -e
TS=$(date +%H%M%S)
echo "=== Phase 39 verify @ $TS ==="

# A. Schema-level
docker exec gpay-postgres psql -U user -d goopay_dw -tA -c "
  SELECT schemaname || '|' || count(*)
  FROM pg_tables
  WHERE schemaname NOT LIKE 'pg_%' AND schemaname <> 'information_schema'
  GROUP BY schemaname ORDER BY schemaname;"
# Expected exact:
#   cdc_auth_service|1
#   cdc_system|≥25
#   public|0

# B. cdc_internal gone
docker exec gpay-postgres psql -U user -d goopay_dw -tA -c \
  "SELECT count(*) FROM pg_namespace WHERE nspname='cdc_internal';"
# Expected: 0

# C. search_path
docker exec gpay-postgres psql -U user -d goopay_dw -tA -c "SHOW search_path;"
# Expected: cdc_system, cdc_auth_service, public

# D. Group 1 — Auth (3 endpoints)
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")
echo "TOKEN=${TOKEN:0:40}..."

REFRESH_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  http://localhost:8081/api/auth/refresh \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{}')
echo "[$REFRESH_CODE] /api/auth/refresh"
# Register: skip (admin đã có), nhưng probe để confirm READ path:
REG_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  http://localhost:8081/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","email":"x@y.z","password":"xxxxxx"}')
echo "[$REG_CODE] /api/auth/register (expect 409 conflict — chứng minh ExistsByUsername hit auth_users)"

# E. Group 2 — Alerts (5 endpoints)
for u in /api/alerts/active /api/alerts/silenced /api/alerts/history; do
  CODE=$(curl -s -o /tmp/_ -w "%{http_code}" \
    -H "Authorization: Bearer $TOKEN" "http://localhost:8083${u}")
  echo "[$CODE] $u"
done
# Ack/Silence: chỉ test khi có alert thật. Skip nếu count = 0.

# F. Group 3 — Audit smoke (1 mutation rep nhóm)
RECON_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  http://localhost:8083/api/reconciliation/check \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: phase39-smoke-recon' -d '{}')
echo "[$RECON_CODE] /api/reconciliation/check"

RELOAD_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  http://localhost:8083/api/mapping-rules/reload \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: phase39-smoke-reload' -d '{}')
echo "[$RELOAD_CODE] /api/mapping-rules/reload"

sleep 3   # đợi async audit writer flush
docker exec gpay-postgres psql -U user -d goopay_dw -c \
  "SELECT count(*) AS audit_rows, max(created_at) AS latest \
   FROM cdc_system.admin_actions;"
# Expected: audit_rows ≥ 2, latest trong vòng 1 phút

# G. Group 4 — 11 operator endpoints (Phase 38 list)
URLS=(
  "/api/schema-changes/pending?status=pending&page_size=1"
  "/api/v1/source-objects/stats"
  "/api/v1/source-objects?page=1&page_size=100"
  "/api/v1/shadow-bindings?page=1&page_size=100"
  "/api/v1/schema-proposals?status=pending"
  "/api/v1/schedules"
  "/api/activity-log?page=1&page_size=30"
  "/api/activity-log/stats"
  "/api/failed-sync-logs?page_size=50"
  "/api/worker-schedule"
  "/api/v1/source-objects?page_size=500"
)
for u in "${URLS[@]}"; do
  CODE=$(curl -s -o /tmp/_ -w "%{http_code}" \
    -H "Authorization: Bearer $TOKEN" "http://localhost:8083${u}")
  printf "[%s] %s\n" "$CODE" "$u"
done

# H. Background alert writer no panic
sleep 60   # 1 collector tick
grep -i 'panic\|cdc_alerts.*does not exist\|relation.*alerts' /tmp/cdc-cms.log || echo "no alert errors"

# I. Auto-flow probe (Phase 38 baseline)
curl -s http://localhost:18083/connectors/goopay-mongodb-cdc/status | python3 -m json.tool | head -10
docker exec gpay-kafka kafka-topics --bootstrap-server localhost:9092 --list | grep '^cdc\.goopay\.' | wc -l

# J. No raw SQL leftover (3 keyword chưa qualify)
grep -rnE "FROM[[:space:]]+(auth_users|admin_actions|cdc_alerts)\b|INTO[[:space:]]+(auth_users|admin_actions|cdc_alerts)\b" \
  /Users/trainguyen/Documents/work/cdc-system/cdc-auth-service \
  /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service \
  /Users/trainguyen/Documents/work/cdc-system/centralized-data-service 2>/dev/null \
  | grep -v _test.go | grep -v '\.md' | grep -v migrations/ \
  | grep -v 'cdc_system\.' | grep -v 'cdc_auth_service\.'
# Expected: 0 lines (nếu có lines = chưa qualify, fail DoD)
```

## Pass criteria final

PASS = đồng thời thoả:
- A: 3 schemas đúng count ✓
- B: cdc_internal = 0 ✓
- C: search_path đúng ✓
- D: 3 auth endpoints behave correctly (login 200, register 409 conflict, refresh 200) ✓
- E: 3 alert read endpoints = 200 ✓
- F: 2 audit smoke endpoints + admin_actions count ≥ 2 ✓
- G: 11/11 operator endpoints = 200 ✓
- H: Alert background writer no error in log ≥ 60s ✓
- I: Connector RUNNING + ≥4 topics ✓
- J: 0 raw SQL chưa qualify ✓

Bất kỳ tiêu chí nào fail → KHÔNG báo PASS, dừng lại re-plan.

## Lessons triggered (sẽ append nếu có)

⏳ Pending — Brain decide sau khi exec xong:
- Nếu phát sinh issue: append vào `agent/memory/global/lessons.md` theo Global Pattern format
- Candidate lessons:
  - GORM TableName() qualify schema cross-service vs search_path (nếu phát sinh xung đột)
  - Wipe public CASCADE side-effects với pgcrypto extension (nếu xảy ra)
  - Pre-flight impact matrix là bắt buộc trước mọi schema-move task (≥1 endpoint mỗi consumer group)
