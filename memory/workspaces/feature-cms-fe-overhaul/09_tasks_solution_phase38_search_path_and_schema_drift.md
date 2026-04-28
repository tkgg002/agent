# Phase 38 — Solution Reference (per task)

## T-38.1 — search_path migration
```sql
-- migrations/039_set_search_path.sql
ALTER ROLE "user" SET search_path = cdc_system, public;
```
Run inline trong docker:
```bash
docker exec gpay-postgres psql -U user -d goopay_dw \
  -c "ALTER ROLE \"user\" SET search_path = cdc_system, public;"
```

## T-38.2 — Replace cdc_internal → cdc_system
- `internal/api/schema_proposal_handler.go`: replace all
  `cdc_internal.schema_proposal` → `cdc_system.schema_proposal` (7 chỗ).
- `internal/api/transmute_schedule_handler.go`: replace all
  `cdc_internal.transmute_schedule` → `cdc_system.transmute_schedule` (4 chỗ).

## T-38.3 — transmute_schedule List + RunNow Raw JOIN
```go
err := h.db.WithContext(ctx).Raw(`
  SELECT
    ts.id, mb.master_table AS master_table, ts.mode, ts.cron_expr,
    ts.last_run_at, ts.next_run_at, ts.last_status, ts.last_error,
    ts.last_stats, ts.is_enabled, ts.created_by, ts.created_at, ts.updated_at
  FROM cdc_system.transmute_schedule ts
  LEFT JOIN cdc_system.master_binding mb ON mb.id = ts.master_binding_id
  ORDER BY mb.master_table NULLS LAST, ts.mode
`).Scan(&rows).Error
```
RunNow tương tự nhưng `WHERE ts.id = ? LIMIT 1`.

## T-38.4 — Drop missing columns
- Xoá `tr.sync_status,` trong COALESCE → CASE WHEN/THEN/ELSE đứng độc lập.
- `COALESCE(rr.diff, tr.recon_drift, 0)` → `COALESCE(rr.diff, 0)`.

## T-38.5 — WorkerScheduleResponse flat scan
```go
type workerScheduleScanRow struct {
  ID uint `gorm:"column:id"`
  // ... 22 fields với gorm:"column:..."
}

var scan []workerScheduleScanRow
db.Raw(SQL).Scan(&scan)
out := make([]WorkerScheduleResponse, 0, len(scan))
for _, s := range scan {
  out = append(out, WorkerScheduleResponse{
    ID: s.ID, /* … */,
    Scope: WorkerScheduleScope{ /* … */ },
  })
}
```

## T-38.6 — Verify pack
```bash
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")

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
  CODE=$(curl -s -o /tmp/_resp -w "%{http_code}" \
    -H "Authorization: Bearer $TOKEN" "http://localhost:8083${u}")
  printf "[%s] %s\n" "$CODE" "$u"
done
```
Expected: tất cả `[200]`.

## T-38.7 — Auto-flow probes
```bash
curl -s http://localhost:18083/connectors/goopay-mongodb-cdc/status \
  | python3 -m json.tool
docker exec gpay-kafka kafka-topics --bootstrap-server localhost:9092 --list \
  | grep '^cdc\.goopay\.'
tail -20 /tmp/cdc-worker.log
```
Expected: connector `RUNNING`, ≥4 topics, worker không panic.

## T-38.9 — Lessons template (append vào agent/memory/global/lessons.md)
1) **Schema rename ↔ search_path coupling**.
   `Global Pattern: Khi A move tables sang schema B, mọi code path không
   qualify schema (GORM TableName, raw SQL không prefix) sẽ break do
   search_path mặc định không bao gồm B. Đúng: kèm
   ALTER ROLE … SET search_path = B, public; trong cùng PR migration,
   hoặc audit qualify toàn bộ.`
2) **GORM Raw().Scan không deep map nested struct**.
   `Global Pattern: Khi query SELECT projects flat columns, struct đích
   chứa sub-struct X sẽ bị "invalid field". Đúng: dùng flat scan struct
   với gorm:"column:" tags rồi transpose tay sang struct chứa sub-struct.`
