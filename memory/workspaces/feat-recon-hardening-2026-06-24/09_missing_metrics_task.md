# Task — Missing Metrics cho SigNoz Dashboard

> **Date**: 2026-06-25T11:11 +07:00
> **Context**: Dashboard cần vẽ đúng 3 node (Source→Shadow→Master) với các mũi tên throughput

---

## Audit Result — Metrics hiện có vs Dashboard cần

### ✅ Đã có — dùng được ngay

| Metric | Prometheus Name | Dashboard Panel |
|--------|----------------|----------------|
| Events Capture (Kafka→Shadow) | `cdc_events_processed_total{operation,source_db,table,status}` | Arrow Capture/Sink rate |
| Sync Shadow→Master success | `cdc_sync_success_total{table,operation,source}` | Arrow Transmute rate |
| Sync Shadow→Master failed | `cdc_sync_failed_total{table,operation,source}` | Arrow fail rate |
| E2E Latency | `cdc_e2e_latency_seconds` | Latency panel |
| Kafka consumer lag | `cdc_kafka_consumer_lag{topic,partition}` | Capture lag |
| Recon ingest lag | `cdc_recon_ingest_lag_ms{table}` | Shadow lag gauge |
| Recon drift count | `cdc_recon_drift_count{table,tier}` | Drift badge |
| Recon mismatch | `cdc_recon_mismatch_count{table,tier}` | Drift detail |
| Recon last success ts | `cdc_recon_last_success_timestamp{table,tier}` | Staleness alert |
| Cycle tables checked | `cdc_recon_cycle_tables_checked` | Cycle summary |
| Cycle drift detected | `cdc_recon_cycle_drift_detected` | Cycle summary |
| Cycle total counter | `cdc_recon_cycle_total{has_drift,has_error}` | Cycle counter |
| Cycle duration | `cdc_recon_cycle_duration_seconds` | Cycle duration hist |
| Fast path hit | `cdc_recon_fast_path_total{table,path}` | pg_class ratio |
| Circuit breaker trips | `cdc_recon_circuit_breaker_trips_total{table}` | CB panel |
| Lock contention | `cdc_recon_lock_contention_total{table}` | Lock health |
| Drill-down wait | `cdc_recon_drill_down_wait_ms{table}` | Semaphore wait |
| Batches flushed | `cdc_batches_flushed_total{sink,table}` | Throughput |
| DLQ write fail | `cdc_dlq_write_failures_total` | DLQ counter |
| Pipeline paused | `cdc_pipeline_paused_total` | Circuit breaker |
| Burst mode | `cdc_burst_mode_active` | Batcher status |
| WAL resume | `cdc_wal_snapshot_resume_total{slot,reason}` | WAL health |

---

## ❌ MISSING — Cần thêm để vẽ đầy đủ

### Gap 1 — Source DB Row Count (per table, per source_db) 🔴
**Dashboard cần**: Node "Source DB" hiển thị tổng rows hiện tại.
**Hiện tại**: Không có metric nào track số row của Source MongoDB per table.
**Fix**: Thêm metric được emit từ `EstimatedCount` call trong `RunTier1`:

```go
// prometheus.go
SourceTableRowCount = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_source_table_row_count",
        Help: "Estimated row count of the source MongoDB collection (O(1) EstimatedDocumentCount)",
    },
    []string{"table", "source_db"},
)

// recon_tier_a.go — sau khi có srcEst
metrics.SourceTableRowCount.WithLabelValues(entry.TargetTable, entry.SourceDB).Set(float64(srcEst))
```

---

### Gap 2 — Shadow DB Row Count (per table) 🔴
**Dashboard cần**: Node "Shadow DB" hiển thị tổng rows.
**Hiện tại**: Không có metric nào. `EstimatedCountRows` chạy nhưng chỉ dùng nội bộ để compare.
**Fix**: Emit sau khi có `dstTotal` trong `RunTier1`:

```go
// prometheus.go
ShadowTableRowCount = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_shadow_table_row_count",
        Help: "Estimated row count of the shadow PostgreSQL table (GREATEST(reltuples, 0))",
    },
    []string{"table"},
)

// recon_tier_a.go
metrics.ShadowTableRowCount.WithLabelValues(entry.TargetTable).Set(float64(dstTotal))
```

---

### Gap 3 — Master DB Row Count (per table) 🟡
**Dashboard cần**: Node "Master DB" hiển thị tổng rows.
**Hiện tại**: Không có metric nào track số row Master.
**Fix**: Thêm query trong `RunTier1` hoặc dedicated master count check:

```go
// prometheus.go
MasterTableRowCount = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_master_table_row_count",
        Help: "Estimated row count of the master PostgreSQL table",
    },
    []string{"table"},
)
```

> **Lưu ý**: Master DB count cần query riêng — không có trong flow hiện tại của RunTier1.
> Option đơn giản nhất: emit từ `finishRun` hoặc thêm 1 query pg_class vào master DB.

---

### Gap 4 — DLQ Depth Gauge (current queue size) 🟡
**Dashboard cần**: Số message đang nằm trong DLQ hiện tại.
**Hiện tại**: `cdc_dlq_write_failures_total` là counter (tổng lũy kế), không phản ánh depth hiện tại.
**Fix**:

```go
// prometheus.go
DLQDepth = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_dlq_depth",
        Help: "Current number of messages sitting in DLQ per table",
    },
    []string{"table"},
)
```

---

### Gap 5 — Pipeline Status per Table (composite health) 🟡
**Dashboard cần**: Badge healthy/drift/error per table cho table selector.
**Hiện tại**: Cần join nhiều metrics để suy ra — không có 1 metric status tổng hợp.
**Fix**:

```go
// prometheus.go — 0=healthy, 1=drift, 2=error, 3=skipped
PipelineTableStatus = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_pipeline_table_status",
        Help: "Composite pipeline health per table: 0=healthy 1=drift 2=error 3=skipped",
    },
    []string{"table"},
)

// recon_engine_run.go — emit sau wg.Wait()
for _, r := range reports {
    statusCode := map[string]float64{"ok":0,"drift":1,"error":2,"skipped":3}
    metrics.PipelineTableStatus.WithLabelValues(r.TargetTable).Set(statusCode[r.Status])
}
```

---

### Gap 6 — Transmute Throughput rate (events/s) 🟡
**Dashboard cần**: Arrow "Transmute → Master" hiển thị events/s.
**Hiện tại**: `cdc_sync_success_total` là counter — có thể `rate()` để tính events/s trong SigNoz, nhưng không có label `target_table` rõ ràng.
**Action**: Không cần metric mới, chỉ cần dùng `rate(cdc_sync_success_total[1m])` trong SigNoz query. ✅ OK

---

## Execution Plan

| Priority | Gap | Files | Effort |
|----------|-----|-------|--------|
| 🔴 P0 | Gap 1 — Source row count | `prometheus.go` + `recon_tier_a.go` | 10 min |
| 🔴 P0 | Gap 2 — Shadow row count | `prometheus.go` + `recon_tier_a.go` | 5 min |
| 🟡 P1 | Gap 5 — Pipeline status | `prometheus.go` + `recon_engine_run.go` | 10 min |
| 🟡 P1 | Gap 4 — DLQ depth | `prometheus.go` + DLQ handler | 15 min |
| 🟡 P2 | Gap 3 — Master row count | `prometheus.go` + new query | 20 min |

**Tổng**: ~60 phút. Sau khi xong, SigNoz có đủ data để vẽ toàn bộ 3 DB nodes + 2 arrows.

---

## [x] Checklist — DONE 2026-06-25T11:18 +07:00

- [x] Gap 1: `cdc_source_table_row_count{table,source_db}` — defined + emit trong `recon_tier_a.go:490`
- [x] Gap 2: `cdc_shadow_table_row_count{table}` — defined + emit trong `recon_tier_a.go:491`
- [x] Gap 5: `cdc_pipeline_table_status{table}` — defined + emit trong `recon_engine_run.go` sau wg.Wait()
- [x] Gap 4: `cdc_dlq_depth{table}` — defined + emit trong `dlq_handler.go:245` sau PublishMsg success
- [x] Gap 3: `cdc_master_table_row_count{table}` — defined + emit async goroutine trong `recon_engine_run.go` (O(1) pg_class via masterAgent)
- [x] Build `./internal/... ./pkgs/... ./cmd/...` — PASS
- [x] `go test -race ./internal/service/recon/... ./internal/handler/recon/...` — PASS (1.977s + 1.803s)
