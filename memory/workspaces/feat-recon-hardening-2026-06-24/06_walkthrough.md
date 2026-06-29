# Walkthrough — feat-recon-hardening-2026-06-24

> **Completed**: 2026-06-25T09:08 +07:00 | **Bug-fixed**: 2026-06-25T09:25 +07:00
> **Service**: `centralized-data-service`
> **Branch**: working branch (chưa push — user quyết định push)
> **Build**: ✅ `go build ./internal/... ./pkgs/... ./cmd/...` — PASS
> **Race test**: ✅ `go test -race ./internal/service/recon/...` — PASS (2.064s)

---

## Tóm tắt những gì đã làm

Hardening pipeline `Reconcile ALL` để chạy ổn định ở scale **200 tables × 50M records/table**. Tổng cộng **7 phases**, sửa **5 files**, thêm **60 dòng metrics**, fix **3 critical bugs + 4 scale issues**.

---

## 🚨 Bug Fixes sau Code Review (2026-06-25T09:25)

### Bug 1 — Context leak + đứt TraceID
**Files**: `recon_engine_run.go` + `recon_tier_a.go`
- **Trước**: `CheckAll` wrap `RunTier1` bằng `45s` context; trong `RunTier1` dùng `context.Background()` cho `drillCtx` → OTel TraceID bị đứt, orphan goroutine khi 45s hết hạn, `finishRun` ném `context deadline exceeded`.
- **Fix**: Xóa `tableCtx 45s` ở `CheckAll`. Trong `RunTier1`: `fastCtx = WithTimeout(ctx, 10s)` cho fast-path; `drillCtx = WithTimeout(ctx, 8m)` kế thừa từ `ctx` gốc.

### Bug 2 — `reltuples = -1` trên PostgreSQL 14+
**Files**: `recon_dest_query.go` + `recon_tier_a.go`
- **Trước**: `COALESCE(reltuples::bigint, 0)` trả `-1` cho bảng chưa ANALYZE. Điều kiện `dstTotal == 0` không bắt được → so sánh `-1` vs Source → False Drift.
- **Fix**: SQL dùng `GREATEST(COALESCE(..., 0), 0)`; Go condition đổi `== 0` → `<= 0`.

### Bug 3 — Hot/Cold Lookback không được wire
**Files**: `recon_engine.go` + `recon_tier_a.go`
- **Trước**: `HotWindowLookback`, `RunMode` khai báo trong config nhưng `pickScanRangeWithLag` hardcode `rc.cfg.WindowLookback` → hot recon 15m/lần vẫn quét 7d.
- **Fix**: Thêm method `effectiveLookback()` → `lower := upper.Add(-rc.effectiveLookback())`.

---


### Phase 4c — Thêm 5 Prometheus Metrics mới
**File**: `pkgs/metrics/prometheus.go`
**Thêm vào cuối `var (` block:**

| Metric | Type | Mô tả |
|--------|------|--------|
| `cdc_recon_ingest_lag_ms` | GaugeVec | Lag ms MongoDB→Shadow. Alert khi > 3.6M |
| `cdc_recon_lock_contention_total` | CounterVec | Số lần advisory lock bị busy |
| `cdc_recon_circuit_breaker_trips_total` | CounterVec | Số lần bỏ qua recon vì lag quá cao |
| `cdc_recon_drill_down_wait_ms` | HistogramVec | Thời gian chờ semaphore BucketCounts |
| `cdc_recon_fast_path_total` | CounterVec | pg_class vs count_star path ratio |

---

### Phase 4a — Fix Dead-Code: Cycle Metrics
**File**: `recon_engine_run.go` — sau `wg.Wait()` (~L247)

**Vấn đề**: `ReconCycleTotal`, `ReconCycleTablesChecked`, `ReconCycleDriftDetected` được define trong `prometheus.go` nhưng **không có dòng nào emit** → luôn = 0 trên SigNoz.

**Fix**: Thêm emit block sau `wg.Wait()`:
```go
driftCount, errorCount := 0, 0
for _, r := range reports { ... }
metrics.ReconCycleTablesChecked.Set(float64(len(reports)))
metrics.ReconCycleDriftDetected.Set(float64(driftCount))
metrics.ReconCycleTotal.WithLabelValues(boolStr(driftCount > 0), boolStr(errorCount > 0)).Inc()
```

---

### Phase 4b — Emit IngestLag Prometheus
**File**: `recon_tier_a.go` — `pickScanRangeWithLag()` (~L195)

**Vấn đề**: Lag chỉ được `upsertReconLag` lưu vào DB, không có Prometheus gauge → không alert được realtime.

**Fix**:
```go
rc.upsertReconLag(ctx, entry.TargetTable, "ingest_lag_ms", ingestLagMs)
metrics.ReconIngestLagMs.WithLabelValues(entry.TargetTable).Set(float64(ingestLagMs)) // ← MỚI
```

---

### Phase 2 — Lag Circuit Breaker
**Files**: `recon_engine.go` + `recon_tier_a.go`

**Vấn đề**: Khi lag > 60m, shadow chưa catch up nhưng recon vẫn chạy → count mismatch → False Drift alert.

**Config thêm vào `ReconCoreConfig`**:
```go
MaxTolerableLagMs int64 // default: 3_600_000 (60m)
```

**Fix trong `RunTier1`** — ngay sau `pickScanRangeWithLag`:
```go
if ingestLagMs > rc.cfg.MaxTolerableLagMs {
    status = "skipped"
    metrics.ReconCircuitBreakerTrips.WithLabelValues(entry.TargetTable).Inc()
    // log warning + return skipped report
}
```

---

### Phase 5 — Shadow DB O(1) Estimate Count
**File**: `recon_dest_query.go`

**Vấn đề**: `CountRows` dùng `SELECT COUNT(*) FROM table` — full scan trên 50M rows. MongoDB side dùng `EstimatedDocumentCount()` O(1) — **bất đối xứng chết người**.

**Fix — Thêm 2 functions mới**:
```go
func splitSchemaTable(qualified string) (schema, table string) { ... }

func (da *ReconDestAgent) EstimatedCountRows(ctx context.Context, qualifiedTable string) (int64, error) {
    // SELECT COALESCE(c.reltuples::bigint, 0)
    // FROM pg_class c JOIN pg_namespace n ...
    // WHERE n.nspname = ? AND c.relname = ?
}
```

**Cập nhật `RunTier1`** — 2-tier strategy:
```go
dstTotal, errD := rc.destAgent.EstimatedCountRows(ctx, ...) // O(1) first
if errD != nil || dstTotal == 0 {
    dstTotal, errD = rc.destAgent.CountRows(...)  // COUNT(*) fallback
    metrics.ReconFastPathHit.WithLabelValues(table, "count_star").Inc()
} else {
    metrics.ReconFastPathHit.WithLabelValues(table, "pg_class").Inc()
}
```

---

### Phase 1 — Advisory Lock Connection Pinning
**File**: `recon_tier_a.go` — `withTableLock()`

**Vấn đề**: `pg_try_advisory_lock` là session-level lock. Code cũ dùng `rc.db.WithContext(ctx).Raw(...)` → GORM pool dispatch → lock và unlock có thể chạy trên **2 connections khác nhau** → lock kẹt vĩnh viễn.

**Fix — Pin `*sql.Conn`**:
```go
func (rc *ReconCore) withTableLock(ctx context.Context, table string) (bool, func()) {
    sqlDB, _ := rc.db.DB()
    conn, _ := sqlDB.Conn(ctx)           // ← Pin 1 connection
    conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired)
    return true, func() {
        conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", key)
        conn.Close()                     // ← Cùng connection giải phóng
    }
}
```

---

### Phase 3 — Drill-down Semaphore (Thundering Herd)
**Files**: `recon_engine.go` + `recon_tier_a.go`

**Vấn đề**: `globalSem=8` limit goroutines, nhưng khi 8 bảng cùng drift → 8 MongoDB BucketCounts heavy aggregate đồng thời → MongoDB CPU/IO bão hòa.

**Fix**:
```go
// recon_engine.go — thêm field
drillDownSem chan struct{} // default capacity = 3

// recon_tier_a.go — acquire trước BucketCounts
if rc.drillDownSem != nil {
    t0 := time.Now()
    rc.drillDownSem <- struct{}{}
    defer func() { <-rc.drillDownSem }()
    metrics.ReconDrillDownWaitMs.WithLabelValues(table).Observe(time.Since(t0).Milliseconds())
}
```

---

### Phase 6 — Adaptive Timeout
**File**: `recon_tier_a.go`

**Vấn đề**: `CheckAll` set context = 45s cho cả fast-path lẫn drill-down. BucketCounts ở 50M rows cần 2-10 phút → inner 30s timeout → report `error` thay vì `drift`.

**Fix — Dedicated `drillCtx`** ngay trước BucketCounts:
```go
drillCtx, cancelDrill := context.WithTimeout(context.Background(), 8*time.Minute)
defer cancelDrill()
// BucketCounts dùng drillCtx thay vì ctx gốc
srcBuckets, err = rc.sourceAgent.BucketCounts(drillCtx, ...)
dstBuckets, err := rc.destAgent.BucketCounts(drillCtx, ...)
```

---

### Phase 7 — Hot/Cold Config Fields
**File**: `recon_engine.go`

**Vấn đề**: `WindowLookback = 7d` scan toàn bộ history mỗi 15 phút → 70 tỷ row-scans/cycle ở 200 tables.

**Fix — Thêm config fields** (implementation sẵn sàng, cron schedule do ops team):
```go
HotWindowLookback   time.Duration // default: 2h (hot recon, frequent)
RunMode             string        // "hot" | "cold" | ""
DrillDownConcurrency int          // default: 3
ColdConcurrency      int          // default: 2
```

---

## Files đã thay đổi

| File | Loại thay đổi |
|------|---------------|
| `pkgs/metrics/prometheus.go` | Thêm 5 metrics hardening |
| `internal/service/recon/recon_engine_run.go` | Emit cycle metrics (Phase 4a) |
| `internal/service/recon/recon_tier_a.go` | 4b + 2 + 5 + 1 + 3 + 6 |
| `internal/service/recon/recon_engine.go` | Config fields + drillDownSem struct/init |
| `internal/service/recon/recon_dest_query.go` | EstimatedCountRows + splitSchemaTable |

---

## Metrics Dashboard — Sau hardening

| Metric | Alert Rule | Ý nghĩa |
|--------|-----------|---------|
| `cdc_recon_ingest_lag_ms` | `> 3_600_000` | Pipeline lag realtime |
| `cdc_recon_lock_contention_total` | `rate[5m] > 0.5/s` | Lock leak early warning |
| `cdc_recon_circuit_breaker_trips_total` | `rate[5m] > 0` sustained 10m | Pipeline stuck |
| `cdc_recon_drill_down_wait_ms{p95}` | `> 2000ms` | Tăng DrillDownConcurrency |
| `cdc_recon_fast_path_total{path="count_star"}` | `rate > 1/m` | pg_class không có stats |
| `cdc_recon_cycle_tables_checked` | `< 0.8 * expected` | Registry miss |
| `cdc_recon_cycle_drift_detected` | `> 0` sustained 30m | Drift thật cần xử lý |

---

## Lưu ý cho lần tiếp theo

1. **Phase 7 chưa wire vào cron**: Config fields đã có, cần ops team set `RECON_MODE=hot/cold` theo schedule.
2. **pg_class estimate (Phase 5)**: Cần bảng đã chạy `ANALYZE` hoặc `autovacuum`. Verify: `SELECT reltuples FROM pg_class WHERE relname = 'table_name'` — nếu = -1 thì chưa analyze.
3. **drillCtx (Phase 6)**: Dùng `context.Background()` thay vì parent ctx để tránh bị cancel khi parent 10s hết. Traceability cần thêm span propagation nếu cần SigNoz trace liên tục.
