# Implementation Plan v3 — Reconcile ALL Hardening + Scale + Metrics
> **Service**: centralized-data-service
> **Scale**: 200 tables × 5–50M records/table
> **Updated**: 2026-06-24T17:36 +07:00
> **Status**: 🔴 Awaiting Execution

---

## 📍 Key Files (với line numbers)

| File | Role |
|------|------|
| `internal/service/recon/recon_tier_a.go` | withTableLock (L26), adaptiveFreeze (L138), RunTier1 (L370), pickScanRangeWithLag (L184) |
| `internal/service/recon/recon_engine_run.go` | CheckAll (L146), finishRun (L110), globalSem/connSem (L169) |
| `internal/service/recon/recon_engine.go` | ReconCore struct (L78), ReconCoreConfig (L25), NewReconCoreWithConfig (L110) |
| `internal/service/recon/recon_dest_query.go` | CountRows-COUNT(*) BUG (L22), BucketCounts (L67) |
| `internal/service/recon/recon_query.go` | EstimatedCount-O(1) (L73), BucketCounts MongoDB (L94) |
| `internal/service/recon/recon_dest_models.go` | ReconDestAgentConfig, QueryTimeout=30s (L24) |
| `pkgs/metrics/prometheus.go` | Tất cả metric defs (L1-286); dead code L249-285 |

---

## 🚨 Danh sách Bug (7 vấn đề)

### BUG-1 — Advisory Lock Leak (🔴 Critical)
**File**: `recon_tier_a.go` L26-43
**Root cause**: `rc.db.WithContext(ctx).Raw(...)` lấy connection ngẫu nhiên từ GORM pool.
`pg_try_advisory_lock` là session-level lock. `unlock()` có thể chạy trên connection khác → lock kẹt vĩnh viễn.
```go
// HIỆN TẠI — BUG
unlock := func() {
    rc.db.Exec("SELECT pg_advisory_unlock(?)", key) // ← Có thể khác connection!
}
```

### BUG-2 — False Drift khi Lag > 60m (🔴 Critical)
**File**: `recon_tier_a.go` L138-207
**Root cause**: `adaptiveFreeze` clamp max = 60m. Khi lag > 60m, shadow chưa kịp catch up, nhưng `RunTier1` vẫn gọi `EstimatedCount` → count mismatch → False Drift alert.

### BUG-3 — Thundering Herd BucketCounts (🟡 High)
**File**: `recon_engine_run.go` L169
**Root cause**: `globalSem=8` chỉ limit goroutines. Khi 8 bảng cùng drift → 8 MongoDB heavy aggregate đồng thời. Không có rate-limit riêng cho BucketCounts.

### BUG-4 — Cycle Metrics luôn = 0 (🔴 Critical Dead Code)
**File**: `prometheus.go` L249-285 định nghĩa 4 metrics nhưng `recon_engine_run.go` KHÔNG có dòng nào emit chúng.
`ReconCycleTotal`, `ReconCycleDuration`, `ReconCycleTablesChecked`, `ReconCycleDriftDetected` → luôn = 0 trên SigNoz.
**Proof**: `grep -n "ReconCycleTablesChecked\|ReconCycleDriftDetected\|ReconCycleTotal" internal/service/recon/*.go` → 0 kết quả.

### BUG-5 — `COUNT(*)` Full Scan Shadow DB 50M rows (🔴 Critical @ Scale)
**File**: `recon_dest_query.go` L22
```go
sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteRelation(tableName)) // ← Full scan!
```
MongoDB dùng O(1) `EstimatedDocumentCount()` (`recon_query.go:82`), nhưng PG Shadow dùng `COUNT(*)`. Với 8 goroutines × 50M rows → Shadow DB CPU đỏ lửa, block luồng CDC ghi.

### BUG-6 — Timeout 45s không đủ cho Drill-down (🔴 Critical @ Scale)
**File**: `recon_engine_run.go` L221: `context.WithTimeout(ctx, 45*time.Second)`
Inner timeout: `ReconDestAgentConfig.QueryTimeout = 30s` (`recon_dest_models.go:24`).
BucketCounts ở 50M rows @ 7d lookback cần 2-10 phút. Kết quả: Inner 30s timeout → error thay vì drift. SRE thấy errors, không thấy drift.

### BUG-7 — WindowLookback 7 ngày scan lại data đã đối soát (🟡 High)
**File**: `recon_engine.go` L45: `c.WindowLookback = 7 * 24 * time.Hour`
200 tables × 50M records × 168 buckets (7d/1h) = massive scan mỗi 15 phút.

---

## 📋 7 Phases — Execution Order

### Thứ tự thực hiện: 4a→4b→4c → 2 → 5 → 6 → 1 → 7 → 3

---

### PHASE 4a — Fix Dead Code Metrics (🟢 Zero Risk)
**File**: `recon_engine_run.go`
**Thêm sau `wg.Wait()` trong `CheckAll` (sau L246):**

```go
driftCount, errorCount, skipCount := 0, 0, 0
for _, r := range reports {
    switch r.Status {
    case "drift":   driftCount++
    case "error":   errorCount++
    case "skipped": skipCount++
    }
}
boolStr := func(b bool) string {
    if b { return "true" }
    return "false"
}
metrics.ReconCycleTablesChecked.Set(float64(len(reports)))
metrics.ReconCycleDriftDetected.Set(float64(driftCount))
metrics.ReconCycleTotal.WithLabelValues(
    boolStr(driftCount > 0),
    boolStr(errorCount > 0),
).Inc()
```

---

### PHASE 4b — Emit IngestLag Prometheus (🟢 Zero Risk)
**File**: `recon_tier_a.go`
**Trong `pickScanRangeWithLag`, sau L195 `rc.upsertReconLag(...)`:**
```go
metrics.ReconIngestLagMs.WithLabelValues(entry.TargetTable).Set(float64(ingestLagMs))
```

---

### PHASE 4c — Thêm 5 Metrics Mới (🟢 Zero Risk)
**File**: `pkgs/metrics/prometheus.go` — Append vào cuối block `var (`:

```go
ReconIngestLagMs = promauto.NewGaugeVec(prometheus.GaugeOpts{
    Name: "cdc_recon_ingest_lag_ms",
    Help: "Ingest lag ms (MongoDB → Shadow). Alert when > 3600000.",
}, []string{"table"})

ReconLockContention = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "cdc_recon_lock_contention_total",
    Help: "Times advisory lock was busy (previous run ongoing or lock leak).",
}, []string{"table"})

ReconCircuitBreakerTrips = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "cdc_recon_circuit_breaker_trips_total",
    Help: "Times Tier1 skipped due to ingest lag > MaxTolerableLagMs.",
}, []string{"table"})

ReconDrillDownWaitMs = promauto.NewHistogramVec(prometheus.HistogramOpts{
    Name:    "cdc_recon_drill_down_wait_ms",
    Help:    "Semaphore wait before BucketCounts. Tune DrillDownConcurrency if p95 > 2s.",
    Buckets: []float64{0, 10, 50, 100, 250, 500, 1000, 2000, 5000},
}, []string{"table"})

ReconFastPathHit = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "cdc_recon_fast_path_total",
    Help: "path=pg_class (O1 estimate) | path=count_star (full scan fallback).",
}, []string{"table", "path"})
```

---

### PHASE 2 — Lag Circuit Breaker (🟢 Low Risk)
**File 1**: `recon_engine.go` — Thêm field vào `ReconCoreConfig` struct (L25-38):
```go
MaxTolerableLagMs int64 // default: 3_600_000 (60 phút)
```
**Thêm vào `applyDefaults()` (L40):**
```go
if c.MaxTolerableLagMs <= 0 {
    c.MaxTolerableLagMs = 60 * 60 * 1000
}
```

**File 2**: `recon_tier_a.go` — Thêm sau `pickScanRangeWithLag` trong `RunTier1` (sau L394):
```go
if ingestLagMs > rc.cfg.MaxTolerableLagMs {
    status = "skipped"
    metrics.ReconCircuitBreakerTrips.WithLabelValues(entry.TargetTable).Inc()
    observability.Ctx(ctx, rc.logger).Warn(
        "tier1 circuit-open — lag exceeds threshold, skip to avoid false drift",
        zap.String("table", entry.TargetTable),
        zap.Int64("lag_ms", ingestLagMs),
        zap.Int64("threshold_ms", rc.cfg.MaxTolerableLagMs),
    )
    return &recon.ReconciliationReport{
        TargetTable: entry.TargetTable,
        Status:      "skipped",
        CheckType:   "count_total",
        Tier:        1,
        CheckedAt:   time.Now().UTC(),
    }
}
```

---

### PHASE 5 — Shadow DB O(1) Estimate Count (🟢 Low Risk)
**File**: `recon_dest_query.go` — Thêm function mới (sau `CountRows`):

```go
// splitSchemaTable tách "schema.table" → (schema, table). Dùng cho pg_class query.
func splitSchemaTable(qualified string) (schema, table string) {
    parts := strings.SplitN(strings.ReplaceAll(qualified, `"`, ""), ".", 2)
    if len(parts) == 2 {
        return parts[0], parts[1]
    }
    return "public", parts[0]
}

// EstimatedCountRows đọc pg_class.reltuples — O(1), không block, không lock.
// Sai số ~0.1-1% sau VACUUM chưa chạy. Fallback sang COUNT(*) khi reltuples=0.
func (da *ReconDestAgent) EstimatedCountRows(ctx context.Context, qualifiedTable string) (int64, error) {
    schema, table := splitSchemaTable(qualifiedTable)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    result, err := da.breaker.Execute(func() (interface{}, error) {
        tx := da.readOnlyDB(ctx)
        defer tx.Rollback()
        var est int64
        return est, tx.Raw(`
            SELECT COALESCE(c.reltuples::bigint, 0)
            FROM   pg_class c
            JOIN   pg_namespace n ON n.oid = c.relnamespace
            WHERE  n.nspname = ? AND c.relname = ?
        `, schema, table).Scan(&est).Error
    })
    if err != nil {
        return 0, err
    }
    return result.(int64), nil
}
```

**File**: `recon_tier_a.go` — Thay `CountRows` trong `RunTier1` (L401-405):
```go
// TRƯỚC:
dstTotal, errD := rc.destAgent.CountRows(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField)

// SAU:
dstTotal, errD := rc.destAgent.EstimatedCountRows(ctx, entry.QualifiedTarget())
if errD != nil || dstTotal == 0 {
    // Fallback: bảng mới hoặc pg_class chưa ANALYZE
    dstTotal, errD = rc.destAgent.CountRows(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField)
    metrics.ReconFastPathHit.WithLabelValues(entry.TargetTable, "count_star").Inc()
} else {
    metrics.ReconFastPathHit.WithLabelValues(entry.TargetTable, "pg_class").Inc()
}
if errD != nil {
    status = "failed"
    return rc.errorReport(entry, "count", 1, fmt.Errorf("dst count: %w", errD))
}
```

---

### PHASE 6 — Adaptive Timeout (🟡 Medium Risk)
**File**: `recon_dest_models.go` — Split `QueryTimeout` thành 2:
```go
type ReconDestAgentConfig struct {
    MaxRowsPerSec    int
    FastPathTimeout  time.Duration // default: 5s  — EstimatedCountRows, MaxWindowTs
    DrillDownTimeout time.Duration // default: 8m  — BucketCounts, CountRows fallback
    BreakerTimeout   time.Duration
    BreakerThreshold uint32
    ReadReplicaDSN   string
}

func (c *ReconDestAgentConfig) applyDefaults() {
    if c.MaxRowsPerSec <= 0 { c.MaxRowsPerSec = 5000 }
    if c.FastPathTimeout <= 0 { c.FastPathTimeout = 5 * time.Second }
    if c.DrillDownTimeout <= 0 { c.DrillDownTimeout = 8 * time.Minute }
    if c.BreakerTimeout <= 0 { c.BreakerTimeout = 60 * time.Second }
    if c.BreakerThreshold == 0 { c.BreakerThreshold = 5 }
}
```

**File**: `recon_engine_run.go` — Tách table timeout (thay L221):
```go
// TRƯỚC (L221):
tableCtx, cancelTable := context.WithTimeout(ctx, 45*time.Second)

// SAU — tách 2 phase:
fastCtx, cancelFast := context.WithTimeout(ctx, 10*time.Second)
defer cancelFast()
// RunTier1 tự tạo drillDown context bên trong nếu cần
report := rc.RunTier1(fastCtx, e) // fastCtx cho fast path; drill-down dùng parent ctx
```

**File**: `recon_tier_a.go` — Trong `RunTier1`, trước BucketCounts (sau L444), tạo dedicated drill-down ctx:
```go
// Khi đã xác nhận mismatch (abs(srcEst-dstTotal) > estTolerance), tạo drill ctx riêng
drillCtx, cancelDrill := context.WithTimeout(ctx, 8*time.Minute)
defer cancelDrill()
// dùng drillCtx cho BucketCounts thay vì ctx gốc
srcBuckets, err = rc.sourceAgent.BucketCounts(drillCtx, ...)
dstBuckets, err := rc.destAgent.BucketCounts(drillCtx, ...)
```

---

### PHASE 1 — Advisory Lock Connection Pinning (🟡 Medium Risk)
**File**: `recon_tier_a.go` — Replace toàn bộ `withTableLock` (L26-43):

```go
func (rc *ReconCore) withTableLock(ctx context.Context, table string) (bool, func()) {
    key := advisoryLockKey("recon_" + table)
    sqlDB, err := rc.db.DB()
    if err != nil {
        observability.Ctx(ctx, rc.logger).Warn("advisory lock: cannot get sql.DB",
            zap.String("table", table), zap.Error(err))
        return false, func() {}
    }
    conn, err := sqlDB.Conn(ctx)
    if err != nil {
        observability.Ctx(ctx, rc.logger).Warn("advisory lock: cannot acquire pinned conn",
            zap.String("table", table), zap.Error(err))
        return false, func() {}
    }
    var acquired bool
    if err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
        _ = conn.Close()
        observability.Ctx(ctx, rc.logger).Warn("advisory lock: query failed",
            zap.String("table", table), zap.Error(err))
        return false, func() {}
    }
    if !acquired {
        _ = conn.Close()
        metrics.ReconLockContention.WithLabelValues(table).Inc()
        return false, func() {}
    }
    return true, func() {
        _, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", key)
        _ = conn.Close()
    }
}
```

**Imports cần thêm**: `"database/sql"` (nếu chưa có — verify với `go build`).

---

### PHASE 7 — Hot/Cold Schedule (🟡 Medium Risk)
**File**: `recon_engine.go` — Thêm vào `ReconCoreConfig`:
```go
RunMode           string        // "hot" | "cold" | "" (default: hot behavior)
HotWindowLookback time.Duration // default: 2h
ColdConcurrency   int           // default: 2 (giảm I/O pressure cho cold run)
```

**Thêm vào `applyDefaults()`:**
```go
if c.HotWindowLookback <= 0 { c.HotWindowLookback = 2 * time.Hour }
if c.ColdConcurrency <= 0 { c.ColdConcurrency = 2 }
```

**Thêm helper vào `recon_engine.go`:**
```go
func (rc *ReconCore) effectiveLookback() time.Duration {
    if rc.cfg.RunMode == "cold" && rc.cfg.WindowLookback > 0 {
        return rc.cfg.WindowLookback // 7d cho cold
    }
    return rc.cfg.HotWindowLookback // 2h cho hot
}
```

**Cron schedule đề xuất (env vars):**
```
# Hot: mỗi 15 phút, lookback 2h
RECON_MODE=hot RECON_HOT_LOOKBACK=2h

# Cold: Chủ Nhật 2am, lookback 7d, concurrency thấp
RECON_MODE=cold RECON_CONCURRENCY=2
```

---

### PHASE 3 — Drill-down Semaphore (🟡 Medium Risk)
**File**: `recon_engine.go` — Thêm field vào `ReconCore` struct (L78):
```go
type ReconCore struct {
    // ... existing fields ...
    drillDownSem chan struct{} // Rate-limit concurrent BucketCounts
}
```

**Trong `ReconCoreConfig`:**
```go
DrillDownConcurrency int // default: 3
```

**Trong `applyDefaults()`:**
```go
if c.DrillDownConcurrency <= 0 { c.DrillDownConcurrency = 3 }
```

**Trong `NewReconCoreWithConfig` — khởi tạo semaphore:**
```go
return &ReconCore{
    // ... existing fields ...
    drillDownSem: make(chan struct{}, cfg.DrillDownConcurrency),
}
```

**File**: `recon_tier_a.go` — Acquire semaphore trong `RunTier1` TRƯỚC BucketCounts:
```go
// Thêm vào đầu khối BucketCounts (sau khi confirmed mismatch, trước L436):
if rc.drillDownSem != nil {
    t := time.Now()
    rc.drillDownSem <- struct{}{}
    defer func() { <-rc.drillDownSem }()
    metrics.ReconDrillDownWaitMs.WithLabelValues(entry.TargetTable).Observe(
        float64(time.Since(t).Milliseconds()),
    )
}
```

---

## 📊 Metrics Dashboard — Sau khi hoàn thành

| Metric | Type | Alert |
|--------|------|-------|
| `cdc_recon_ingest_lag_ms` | Gauge | `> 3_600_000` |
| `cdc_recon_lock_contention_total` | Counter | `rate[5m] > 0.5/s` |
| `cdc_recon_circuit_breaker_trips_total` | Counter | `rate[5m] > 0` sustained 10m |
| `cdc_recon_drill_down_wait_ms p95` | Histogram | `> 2000ms` |
| `cdc_recon_fast_path_total{path="count_star"}` | Counter | `rate > 1/m` |
| `cdc_recon_cycle_tables_checked` | Gauge | `< 0.8 * expected` |
| `cdc_recon_cycle_drift_detected` | Gauge | `> 0` sustained 30m |
| `cdc_recon_last_success_timestamp` | Gauge | `time()-value > 3600` |

---

## ✅ Verification Commands

```bash
# Sau từng phase:
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service
go build ./...

# Race detector (bắt buộc sau Phase 1 & 3):
go test -race ./internal/service/recon/... -v -timeout 180s

# Verify metrics được emit (không còn dead code):
grep -rn "ReconCycleTablesChecked\|ReconCycleDriftDetected\|ReconCycleTotal\|ReconIngestLagMs" \
  internal/service/recon/ pkgs/metrics/

# Verify lock pinning (sau Phase 1 — manual):
# Query PostgreSQL: SELECT locktype, classid, objid, granted
#                  FROM pg_locks WHERE locktype = 'advisory';
# Expected: 0 stale locks sau khi cycle hoàn thành
```
