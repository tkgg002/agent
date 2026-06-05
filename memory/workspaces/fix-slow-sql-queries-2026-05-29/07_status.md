# Status — DONE

| Item | Status |
|---|---|
| Migration `053_perf_idx_cdc_jobs_running.sql` | DONE |
| `system_health_queries.go` Q2 ORDER BY fix | DONE |
| `system_health_queries.go` Q3 merged COUNT | DONE |
| `go build ./...` | PASS |
| `go vet` affected packages | PASS |
| Workspace docs | DONE |
| Report | DONE |

## Deployment
- **Auto**: Next service boot → migrate runner applies `053_*` qua advisory lock.
- **Manual (recommended cho prod cluster)**: Ops chạy `CREATE INDEX CONCURRENTLY idx_cdc_jobs_running_started ON cdc_system.cdc_jobs (started_at) WHERE status='running';` trước deploy → runner sẽ no-op qua `IF NOT EXISTS`.

## Rollback
- Code: revert commit (2 file `system_health_queries.go` + 1 SQL).
- Index: `DROP INDEX IF EXISTS cdc_system.idx_cdc_jobs_running_started;` (rollback chỉ cần khi index causes write amplification — không xảy ra với partial running set nhỏ).
