# Phase 39 — Plan: Schema Consolidation

## Step 0 — Pre-flight (Brain)

- Tạo workspace docs đủ bộ: 01/02/03/06/08/09 prefix Phase 39
- Append `05_progress.md` với mốc Phase 39 START
- Soạn migration drafts trong `09_tasks_solution_phase39_*.md`
- Soạn code patches trong `09_tasks_solution_phase39_*.md`
- User approve trước khi Muscle exec

## Step 1 — Stop services

```bash
# Tìm pid theo port, kill graceful
for port in 8081 8083 8084 8085; do
  pid=$(lsof -ti :$port 2>/dev/null || true)
  [ -n "$pid" ] && kill "$pid" && echo "killed $port pid=$pid"
done
sleep 2
```

## Step 2 — Backup

```bash
ts=$(date +%Y%m%d_%H%M%S)
docker exec gpay-postgres pg_dump -U user -d goopay_dw \
  --schema-only > /tmp/phase39_schema_${ts}.sql
docker exec gpay-postgres pg_dump -U user -d goopay_dw \
  -n cdc_system > /tmp/phase39_cdc_system_${ts}.sql
docker exec gpay-postgres pg_dump -U user -d goopay_dw \
  -t 'public.auth_users' > /tmp/phase39_auth_users_${ts}.sql
echo "Backup: /tmp/phase39_*_${ts}.sql"
```

## Step 3 — Wipe

Dùng script `wipe_cdc_runtime_v2.sql` (bản cập nhật):

1. `DROP SCHEMA IF EXISTS cdc_internal CASCADE;`
2. Drop dynamic schemas: `DROP SCHEMA shadow_*` + `DROP SCHEMA dw_*` (loop qua `pg_namespace`)
3. `DROP SCHEMA public CASCADE; CREATE SCHEMA public;` (giữ schema rỗng cho extension)
4. `TRUNCATE` toàn bộ table trong `cdc_system` `RESTART IDENTITY CASCADE`

Áp dụng:
```bash
docker exec -i gpay-postgres psql -U user -d goopay_dw \
  -v ON_ERROR_STOP=1 < cdc-system/centralized-data-service/deployments/sql/wipe_cdc_runtime_v2.sql
```

## Step 4 — Migrate centralized-data-service

```bash
cd cdc-system/centralized-data-service
make migrate   # 001..042 (gồm 040/041/042 mới)
```

Trong đó:
- `040_admin_actions_in_cdc_system.sql` — partitioned table `cdc_system.admin_actions` + 4 partitions
- `041_cdc_alerts_in_cdc_system.sql` — table `cdc_system.cdc_alerts` + 3 indexes
- `042_search_path_with_auth.sql` — `ALTER ROLE "user" SET search_path = cdc_system, cdc_auth_service, public;`

## Step 5 — Migrate cdc-auth-service

```bash
docker exec -i gpay-postgres psql -U user -d goopay_dw -v ON_ERROR_STOP=1 \
  < cdc-system/cdc-auth-service/migrations/001_auth_users.sql
```

File `001_auth_users.sql` đã rewrite:
- `CREATE SCHEMA IF NOT EXISTS cdc_auth_service;`
- `CREATE TABLE cdc_auth_service.auth_users (...)`
- Indexes + seed admin/admin123

## Step 6 — Bootstrap seed

```bash
cd cdc-system/centralized-data-service
make migrate-bootstrap-local
```

(File `bootstrap_cdc_system_v2_local.sql` không đụng `auth_users` → giữ nguyên.)

## Step 7 — Apply code patches

3 file Go đổi schema qualify:
- `cdc-system/cdc-auth-service/internal/model/user.go:17` — `TableName() = "cdc_auth_service.auth_users"`
- `cdc-system/cdc-cms-service/internal/model/alert.go:37` — `TableName() = "cdc_system.cdc_alerts"`
- `cdc-system/cdc-cms-service/internal/middleware/audit.go:166` — Raw SQL `INSERT INTO cdc_system.admin_actions`

Build verify:
```bash
cd cdc-system/cdc-auth-service && go build ./...
cd cdc-system/cdc-cms-service && go build ./...
cd cdc-system/centralized-data-service && go build ./...
```

## Step 8 — Restart services

Theo thứ tự dependency:
1. `cdc-auth-service` (port 8081) → cung cấp JWT cho cms
2. `cdc-cms-service` (port 8083)
3. `centralized-data-service` worker (port 8084)
4. `centralized-data-service` transmute (port 8085)

```bash
cd cdc-system/cdc-auth-service && CONFIG_PATH=./config/config-local.yml \
  nohup go run ./cmd/server > /tmp/cdc-auth.log 2>&1 &
cd cdc-system/cdc-cms-service && CONFIG_PATH=./config/config-local.yml \
  nohup go run ./cmd/server > /tmp/cdc-cms.log 2>&1 &
# … worker / transmute tương tự
```

## Step 9 — Verify (Definition of Done)

```bash
# 1. public rỗng
docker exec gpay-postgres psql -U user -d goopay_dw -c \
  "SELECT count(*) FROM pg_tables WHERE schemaname='public';"
# Expected: 0

# 2. schema map
docker exec gpay-postgres psql -U user -d goopay_dw -c "\dn"
# Expected: cdc_system, cdc_auth_service, public, pg_* defaults

# 3. cdc_system tables
docker exec gpay-postgres psql -U user -d goopay_dw -c \
  "SELECT count(*) FROM pg_tables WHERE schemaname='cdc_system';"
# Expected: ≥25 (23 cũ + admin_actions + cdc_alerts)

# 4. auth_users moved
docker exec gpay-postgres psql -U user -d goopay_dw -c \
  "SELECT username, role FROM cdc_auth_service.auth_users;"
# Expected: admin row

# 5. cdc_internal gone
docker exec gpay-postgres psql -U user -d goopay_dw -c \
  "SELECT count(*) FROM pg_namespace WHERE nspname='cdc_internal';"
# Expected: 0

# 6. search_path
docker exec gpay-postgres psql -U user -d goopay_dw -c "SHOW search_path;"
# Expected: cdc_system, cdc_auth_service, public

# 7. Login
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")

# 8. 11 endpoints (theo Phase 38)
URLS=(/api/schema-changes/pending?status=pending\&page_size=1
      /api/v1/source-objects/stats
      /api/v1/source-objects?page=1\&page_size=100
      /api/v1/shadow-bindings?page=1\&page_size=100
      /api/v1/schema-proposals?status=pending
      /api/v1/schedules
      /api/activity-log?page=1\&page_size=30
      /api/activity-log/stats
      /api/failed-sync-logs?page_size=50
      /api/worker-schedule
      /api/v1/source-objects?page_size=500)
for u in "${URLS[@]}"; do
  CODE=$(curl -s -o /tmp/_ -w "%{http_code}" \
    -H "Authorization: Bearer $TOKEN" "http://localhost:8083${u}")
  printf "[%s] %s\n" "$CODE" "$u"
done

# 9. Auto-flow
curl -s http://localhost:18083/connectors/goopay-mongodb-cdc/status
docker exec gpay-kafka kafka-topics --bootstrap-server localhost:9092 --list | grep '^cdc\.goopay\.'
tail -20 /tmp/cdc-worker.log
```

## Step 10 — Document & Lesson

- Fill `03_implementation_phase39_*.md` với danh sách file thực sự đã thay đổi (Muscle log)
- Fill `06_validation_phase39_*.md` với output verify thật (HTTP codes, schema dump …)
- Append `05_progress.md` mốc Phase 39 DONE
- Append lesson nếu có pattern mới (vd: GORM TableName qualify schema cross-service)
- Update task list: T-39.* tất cả về ✅

## Rollback strategy

Nếu wipe + bootstrap fail nửa chừng:
```bash
docker exec -i gpay-postgres psql -U user -d goopay_dw < /tmp/phase39_schema_${ts}.sql
docker exec -i gpay-postgres psql -U user -d goopay_dw < /tmp/phase39_cdc_system_${ts}.sql
```

Không rollback partial — luôn restore từ backup tổng.
