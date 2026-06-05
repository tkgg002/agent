# Implementation

## Files changed

### 1. NEW — `cdc-cms-service/migrations/schema/ops/053_perf_idx_cdc_jobs_running.sql`
- Partial idx `(started_at) WHERE status='running'` trong schema `cdc_system`.
- `IF NOT EXISTS` re-runnable.
- COMMENT giải thích purpose + ngày hotfix.
- Verification SELECT trả `new_index_exists=1`.

### 2. EDIT — `cdc-cms-service/internal/infra/observability/system_health_queries.go`

#### Q2 (`queryRecentEvents`)
```diff
-Order("started_at DESC").Limit(10).Find(&logs)
+Order("created_at DESC").Limit(10).Find(&logs)
```
+ Cập nhật doc comment giải thích lý do đổi.

#### Q3 (`queryFailedCount`)
Thay 2 `db.Table(...).Count(...)` riêng → 1 `db.Raw(...).Scan(&row)` với struct anonymous + FILTER:
```go
var row struct {
    Count24h int64 `gorm:"column:count_24h"`
    Count1h  int64 `gorm:"column:count_1h"`
}
db.Raw(
    `SELECT
        count(*) AS count_24h,
        count(*) FILTER (WHERE created_at > ?) AS count_1h
     FROM cdc_system.failed_sync_logs
     WHERE created_at > ? AND created_at <= ?`,
    oneHourAgo, twentyFourHoursAgo, now,
).Scan(&row)
```
+ Cập nhật doc comment.

## Why no test added
- Package `internal/infra/observability` không có test file existing.
- Add test mới = scope creep (user yêu cầu fix, không yêu cầu test infra).
- Functional verify: `go build ./...` PASS + comment in-source documented intent.

## Migration apply path
- Runner đọc embed.FS `migrations/schema/*/*.sql` theo thứ tự lexical → `053_*` chạy sau `052_*` tự động on next boot.
- Advisory lock + tx wrap đảm bảo single-instance apply.
- Khi prod cluster cần online apply không lock, ops có thể chạy thủ công `CREATE INDEX CONCURRENTLY` trước rồi runner sẽ no-op qua `IF NOT EXISTS`.
