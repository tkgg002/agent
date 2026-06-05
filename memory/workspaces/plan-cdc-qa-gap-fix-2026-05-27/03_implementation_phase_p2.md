# 03_implementation_phase_p2 — Chi tiết kỹ thuật

## G-10 — Tier 3 off-peak config

### File: `internal/service/recon_core.go`
```go
// Current: hardcoded 02:00-05:00 trong selectTier3
// Plan: read từ ReconCoreConfig
type ReconCoreConfig struct {
    ...
    Tier3OffPeakStart int `mapstructure:"tier3OffPeakStart"` // hour 0-23, default 2
    Tier3OffPeakEnd   int `mapstructure:"tier3OffPeakEnd"`   // hour 0-23, default 5
}

func (rc *ReconCore) isOffPeak(now time.Time) bool {
    h := now.Hour()
    return h >= rc.config.Tier3OffPeakStart && h < rc.config.Tier3OffPeakEnd
}
```

### Verify
- `recon_core_test.go` thêm test off-peak window configurable.

---

## G-11 — `cdc_batches_flushed_total` counter

### File: `pkgs/metrics/prometheus.go`
```go
BatchesFlushed = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "cdc_batches_flushed_total",
    Help: "Số batch đã flush vào shadow DB.",
}, []string{"shadow_db", "table", "status"})
```

### File: `internal/handler/batch_buffer.go`
**Vị trí**: trong `flush()` method sau `batchUpsert`:
```go
status := "success"
if err != nil { status = "fail" }
metrics.BatchesFlushed.WithLabelValues(shadowDB, table, status).Inc()
```

### Verify
- `curl localhost:9090/metrics | grep cdc_batches_flushed_total` → counter visible.
- Grafana panel `rate(cdc_batches_flushed_total[1m])` show batches/sec.

---

## G-12 — Burst mode adaptive batch

### File: `internal/handler/kafka_consumer.go`
**Concept**: monitor `ConsumerLag` gauge mỗi 30s. Khi lag > threshold, tăng `batchSize` lên 2x; khi lag bình thường, revert.

```go
type adaptiveBatcher struct {
    baseBatchSize int
    currentSize   atomic.Int64
    lastAdjust    time.Time
    lagThreshold  int64 // events
}

func (ab *adaptiveBatcher) adjust(currentLag int64) {
    if time.Since(ab.lastAdjust) < 30*time.Second { return }
    if currentLag > ab.lagThreshold {
        ab.currentSize.Store(int64(ab.baseBatchSize * 2))
        metrics.BurstModeActive.Set(1)
    } else {
        ab.currentSize.Store(int64(ab.baseBatchSize))
        metrics.BurstModeActive.Set(0)
    }
    ab.lastAdjust = time.Now()
}
```

### Config knob
```yaml
worker:
  adaptiveBatchEnabled: true
  adaptiveBatchLagThreshold: 50000
  adaptiveBatchMaxMultiplier: 4
```

### Verify
- Synthetic test: produce 100k messages → assert `BurstModeActive == 1` trong 30s đầu.

---

## G-13 — Per-source connection pool semaphore

### File: `pkgs/database/postgres.go` + NEW `pkgs/database/per_source_pool.go`
```go
// Per-source semaphore wrapper. Cap số concurrent connection per source.
type PerSourcePool struct {
    pool      *gorm.DB
    semaphore map[string]chan struct{} // source_code → semaphore
    maxPerSrc int
    mu        sync.RWMutex
}

func (p *PerSourcePool) Acquire(ctx context.Context, source string) (release func(), err error) {
    p.mu.RLock()
    sem, ok := p.semaphore[source]
    p.mu.RUnlock()
    if !ok {
        p.mu.Lock()
        sem = make(chan struct{}, p.maxPerSrc)
        p.semaphore[source] = sem
        p.mu.Unlock()
    }
    select {
    case sem <- struct{}{}:
        return func() { <-sem }, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### Metric
```go
PerSourcePoolSaturation = promauto.NewGaugeVec(prometheus.GaugeOpts{
    Name: "cdc_per_source_pool_in_use",
}, []string{"source_code"})
```

---

## G-14 — Runbook

### Files NEW:
- `docs/runbooks/recon-drift-response.md` — escalation khi `cdc_recon_mismatch_count > 0 for > 1h`.
- `docs/runbooks/wal-slot-expire.md` (đã làm ở G-6).
- `docs/runbooks/pipeline-pause-resume.md` — flow operator-driven resume sau khi DLQ CB pause.
- `docs/runbooks/schema-drift-approve-sla.md` — SLA propose→approve.

### Content template (recon-drift-response.md)
```markdown
# Runbook: Recon Drift Response

## Trigger
Alert `ReconDriftPersistent` (`cdc_recon_drift_count > 0 for 1h`).

## Severity Triage
| Drift count | Severity |
|---|---|
| 1-10 | Info — wait next tier-2 cycle |
| 10-100 | Warning — investigate batch +-1h |
| >100 | Critical — pause writes, manual heal |

## Diagnose
1. Identify drift table: query `cdc_system.recon_runs WHERE status='drift_detected' ORDER BY started_at DESC LIMIT 10`.
2. Compare source count vs shadow count.

## Resolve
- Auto-heal: NATS `cdc.cmd.recon-heal {table: '...', window: '...'}`.
- Manual: re-snapshot specific PK range.
```

---

## G-15 — Chaos test network flicker

### File NEW: `scripts/chaos_network.sh`
```bash
#!/usr/bin/env bash
# Simulate network partition between worker and Mongo source (5-15 min)
set -euo pipefail

DURATION_SEC=${DURATION_SEC:-600} # 10 min default
TARGET_HOST=${TARGET_HOST:-mongodb-source}
TARGET_PORT=${TARGET_PORT:-27017}

echo "Adding iptables DROP rule for $TARGET_HOST:$TARGET_PORT for $DURATION_SEC seconds..."
sudo iptables -A OUTPUT -p tcp -d $TARGET_HOST --dport $TARGET_PORT -j DROP

# Capture metric snapshot
BEFORE_LAG=$(curl -s localhost:9090/metrics | grep cdc_kafka_consumer_lag | awk '{print $2}')

sleep $DURATION_SEC

echo "Removing iptables DROP rule..."
sudo iptables -D OUTPUT -p tcp -d $TARGET_HOST --dport $TARGET_PORT -j DROP

# Wait for catch-up
sleep 60
AFTER_LAG=$(curl -s localhost:9090/metrics | grep cdc_kafka_consumer_lag | awk '{print $2}')

echo "Before: $BEFORE_LAG, After (1min catch-up): $AFTER_LAG"
# Acceptance: AFTER_LAG should be < 2x BEFORE_LAG
```

### Verify
- Run in staging, observe metric: `cdc_recon_drift_count` should remain 0 sau chaos + 30 phút.

---

## G-16 — Load test k6

### File NEW: `scripts/load_test.js`
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';

const e2eLatency = new Trend('e2e_latency_ms');

export const options = {
  stages: [
    { duration: '1m',  target: 100 },  // ramp up
    { duration: '5m',  target: 1000 }, // sustained
    { duration: '2m',  target: 5000 }, // burst
    { duration: '1m',  target: 0 },    // ramp down
  ],
  thresholds: {
    e2e_latency_ms: ['p(99) < 5000'], // P99 < 5s
  },
};

export default function () {
  // Insert via Mongo direct
  const id = `load_${__VU}_${__ITER}_${Date.now()}`;
  http.post('http://mongo-proxy/insert', JSON.stringify({ _id: id }));

  // Poll shadow for arrival
  const start = Date.now();
  let arrived = false;
  for (let i = 0; i < 30; i++) {
    const r = http.get(`http://cms-api/api/v1/shadow/users/${id}`);
    if (r.status === 200) { arrived = true; break; }
    sleep(0.5);
  }
  if (arrived) e2eLatency.add(Date.now() - start);
  check({ arrived }, { 'event arrived': (r) => r.arrived });
}
```

### CI integration
- Run weekly trong staging, compare với baseline.
- Acceptance: P99 latency < 5s @ 1000 TPS sustained.

---

## Composite score change (P2 done)
- G-10 → 1.1 Data Reconciliation L4 stay L4 (refinement only).
- G-11 → 3.2 TPS L3 → L4 (+1).
- G-12 → 3.3 Backlog L3 → L4 (+1).
- G-13 → 4.2 Concurrency L3 → L4 (+1).
- G-14 → 1.1 Data Reconciliation L4 stay (runbook part) + L4 cho 2.3 LSN.
- G-15 → 2.2 Network Flicker L2 → L3 (+1).
- G-16 → 3.1 Data Lag L3 → L4 (+1) + 3.2 TPS validation.

**Sau P0+P1+P2**: 51 + 5 = 56/64 ≈ 87.5%.
