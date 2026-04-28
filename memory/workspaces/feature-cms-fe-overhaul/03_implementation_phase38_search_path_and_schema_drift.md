# Phase 38 — Implementation Log

## Files changed (Muscle commits)

### 1) Migration mới
`cdc-system/centralized-data-service/migrations/039_set_search_path.sql`
```sql
ALTER ROLE "user" SET search_path = cdc_system, public;
```
- Áp dụng trực tiếp qua `docker exec gpay-postgres psql -U user -c '...'` để
  reload search_path cho session.

### 2) `cdc-cms-service/internal/api/schema_proposal_handler.go`
- 7 lần thay `cdc_internal.schema_proposal` → `cdc_system.schema_proposal`
  (List, Get, Approve.load, Approve.update, Approve.failed-mark, Reject.update).

### 3) `cdc-cms-service/internal/api/transmute_schedule_handler.go`
- 4 lần thay `cdc_internal.transmute_schedule` → `cdc_system.transmute_schedule`.
- `List` rewrite thành Raw SQL JOIN `cdc_system.master_binding` để map
  `mb.master_table` về `ScheduleRow.MasterTable`.
- `RunNow` cũng rewrite Raw SQL với JOIN tương đương.

### 4) `cdc-cms-service/internal/api/source_objects_handler.go`
- Xóa `tr.sync_status` khỏi 2 COALESCE (List + GetMappingContext); chỉ còn
  CASE fallback.
- Replace 3 `COALESCE(rr.diff, tr.recon_drift, 0)` → `COALESCE(rr.diff, 0)`.

### 5) `cdc-cms-service/internal/api/schedule_handler.go`
- Thêm struct `workerScheduleScanRow` (22 field) với gorm column tag.
- `listResponses` đổi sang `Scan(&[]workerScheduleScanRow)` rồi transpose ra
  `[]WorkerScheduleResponse` (gắn sub-struct `Scope`).
- Qualify `FROM cdc_worker_schedule` → `FROM cdc_system.cdc_worker_schedule`.

## Build matrix
- `cdc-cms-service`: `go build ./...` → exit 0.
- `centralized-data-service`: `go build ./...` → exit 0 (giữ từ phase 37,
  phase 38 không sửa worker code).

## Restart steps
```bash
# Stop CMS (pid found via `lsof -i :8083`)
kill <pid>
# Restart
cd cdc-system/cdc-cms-service
CONFIG_PATH=./config/config-local.yml nohup go run ./cmd/server > /tmp/cdc-cms.log 2>&1 &
```

## Auth note (gotcha)
`cdc-auth-service` yêu cầu field `username` (không phải `email`) trong body
login. Bảng đích là `public.auth_users` (không phải `public.users` — bảng
sau là CDC shadow). Credentials test: `admin / admin123`.
```bash
curl -s -X POST http://localhost:8081/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])"
```
