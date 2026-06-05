# 08_tasks_phase_p2 — Checklist Muscle Phase P2

> Reference: `03_implementation_phase_p2.md`. Parallel sprints OK.

## G-10 — Tier3 off-peak config (1h)
- [ ] Sửa `internal/service/recon_core.go`:
  - Thêm field `Tier3OffPeakStart int` + `Tier3OffPeakEnd int` vào `ReconCoreConfig`.
  - Default 2, 5.
  - Hàm `isOffPeak(now time.Time) bool` đọc từ config.
- [ ] Test `recon_core_test.go` thêm case `Tier3OffPeakStart=22` → 23:00 = offpeak.
- [ ] Verify: `go test ./internal/service -run TestTier3OffPeak` PASS.

## G-11 — Batches flushed counter (0.5h)
- [ ] Sửa `pkgs/metrics/prometheus.go`: thêm `BatchesFlushed = promauto.NewCounterVec(prometheus.CounterOpts{Name: "cdc_batches_flushed_total"}, []string{"shadow_db", "table", "status"})`.
- [ ] Sửa `internal/handler/batch_buffer.go`: trong `flush()` method sau `batchUpsert`, tăng counter với label status=success/fail.
- [ ] Verify: `curl :9090/metrics | grep cdc_batches_flushed_total` thấy counter.

## G-12 — Adaptive batch (2h)
- [ ] Sửa `internal/handler/kafka_consumer.go`: thêm struct `adaptiveBatcher{baseBatchSize, currentSize atomic.Int64, lastAdjust time.Time, lagThreshold int64}`.
- [ ] Method `adjust(currentLag int64)`: nếu lag > threshold → `currentSize.Store(base*2)` + `metrics.BurstModeActive.Set(1)`; ngược lại revert.
- [ ] Config knob: `worker.adaptiveBatchEnabled`, `adaptiveBatchLagThreshold` (default 50000), `adaptiveBatchMaxMultiplier` (default 4).
- [ ] Verify: produce 100k message → `cdc_burst_mode_active == 1` trong 30s đầu.

## G-13 — Per-source pool semaphore (4h)
- [ ] Tạo NEW `pkgs/database/per_source_pool.go`:
  - Struct `PerSourcePool{pool *gorm.DB; semaphore map[string]chan struct{}; maxPerSrc int; mu sync.RWMutex}`.
  - Method `Acquire(ctx, source) (release func(), err error)`.
- [ ] Sửa `pkgs/database/postgres.go` wire PerSourcePool vào DI.
- [ ] Metric `PerSourcePoolSaturation = promauto.NewGaugeVec(..., []string{"source_code"})`.
- [ ] Verify: `curl :9090/metrics | grep cdc_per_source_pool_in_use` show per-source gauge.

## G-14 — Runbooks (2h)
- [ ] Tạo NEW `docs/runbooks/recon-drift-response.md` (template trong 03_impl_p2.md §G-14).
- [ ] Tạo NEW `docs/runbooks/pipeline-pause-resume.md` (flow operator resume sau DLQ CB pause).
- [ ] Tạo NEW `docs/runbooks/schema-drift-approve-sla.md`.
- [ ] (`wal-slot-expire.md` đã làm ở G-6.)
- [ ] Link annotation runbook_url trong alert rule.
- [ ] Verify: `ls docs/runbooks/` show 4 file.

## G-15 — Chaos network (4h)
- [ ] Tạo NEW `scripts/chaos_network.sh`:
  - Param `DURATION_SEC`, `TARGET_HOST`, `TARGET_PORT`.
  - `iptables -A OUTPUT -p tcp -d $HOST --dport $PORT -j DROP`.
  - Sleep duration.
  - Remove rule.
  - Capture metric before/after, assert `AFTER_LAG < 2x BEFORE_LAG`.
- [ ] Run trên staging, observe `cdc_recon_drift_count` remain 0 sau chaos + 30 phút.
- [ ] Verify: script exit 0 + drift count check PASS.

## G-16 — k6 load test (2.5h)
- [ ] Tạo NEW `scripts/load_test.js`:
  - Stage: 1m→100, 5m→1000, 2m→5000, 1m→0.
  - Threshold `e2e_latency_ms p(99) < 5000`.
  - Inject Mongo + poll shadow API.
- [ ] Tạo CI workflow weekly chạy k6 trong staging, compare baseline.
- [ ] Verify: `k6 run scripts/load_test.js` threshold PASS.

## Post-phase
- [ ] Build/vet/test all services PASS.
- [ ] /security-agent scan PASS.
- [ ] APPEND `05_progress.md`.
- [ ] Composite score → kỳ vọng 56/64.
