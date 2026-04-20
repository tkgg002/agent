# Plan: Observability v3 — Scale-ready, Silent-bug-free

> **Date**: 2026-04-17
> **Author**: Brain (claude-opus-4-7)
> **Supersedes**: `02_plan_observability_final.md` (v2, claude-sonnet-4-6)
> **Based on**: `10_gap_analysis_observability_review.md` + user answers + archaeology
> **Target scale**: > 2000 events/sec, 200+ bảng, 20+ Kafka topics, 3 JetStream streams, 500M+ records projected

---

## 0. Scale Budget

| Metric | Current (runtime verified v1) | v3 target | Hard limit |
|:-------|:------------------------------|:----------|:-----------|
| `/api/system/health` response p99 | 2-5s (sync aggregate) | < 50ms (cached) | 100ms |
| System Health collector interval | 30s FE poll × 10 users = 10 req/min external | 1 background run/15s = 4 req/min | 8 req/min |
| Activity log rows/day (events batch) | ~50K | < 500K | 2M |
| Activity log retention | ∞ (BUG) | 30 days | 60 days |
| OTel log sample rate Info | 1.0 (OOM risk) | 0.1 | 0.2 |
| OTel log sample rate Error | 1.0 | 1.0 | 1.0 |
| OTel batch queue memory | unbounded | 256 MB limit | 512 MB |
| Prometheus active series budget | ~150K (current design) | < 100K | 500K |
| FE auto-refresh per tab | 30s full page | 30s cached + per-section stale-while-revalidate | — |
| Alert fingerprint cache | — | dedup 5-min window | 15 min |

---

## 1. Verified Assumptions

| ID | Assumption | Status | Notes |
|:---|:-----------|:-------|:------|
| V1 | NATS JetStream thật (3 streams) | ✅ | Dùng `/jsz` cho monitoring OK |
| V2 | Redis có, underutilized | ✅ | Free dùng cho health cache, alert state, idempotency |
| V3 | Prometheus "có" nhưng chưa wire | ⚠️ | Metrics define qua `promauto` ở `pkgs/metrics/prometheus.go` — nhưng **không expose `/metrics`** ở Worker. Phải bổ sung (T0-1). OTLP metric push SigNoz đang có ở `otel.go`. User confirm có Prom server riêng cạnh SigNoz — cần DevOps confirm URL + scrape target. |
| V4 | OTel Kafka instrumentation | ❌ | Không có. Trace đứt. Worker phải tạo span thủ công từ W3C header hoặc root span mới. |
| V5 | FE stack | React 19 + Ant Design 6 + Axios, no state lib | Thêm `@tanstack/react-query` |
| V6 | FE reconciliation/health page | Đã có v1 runtime verified | v3 refactor thêm React Query + per-section status |
| V7 | Kafka library | `segmentio/kafka-go` | Consumer lag phải dùng `kafka.NewConn()` Admin API thủ công (segmentio không có ClusterAdmin như Sarama) |
| V8 | SigNoz OTLP endpoint | HTTP :4318 (traces+metrics), gRPC :4317 (logs) | OK, thêm memory_limiter |

---

## 2. Kiến trúc v3

```
┌────────────────────────────────────────────────────────────────────┐
│  Layer 4: PRESENTATION (React + Ant Design + React Query)          │
│  /system-health — per-section fetch + staleWhileRevalidate        │
│  Per-section status badge: ok | degraded | down | unknown          │
│  Alert banner với dedup + ack + silence                           │
└─────────────────────┬──────────────────────────────────────────────┘
                      │ GET /api/system/health (cached, < 50ms)
                      ▼
┌────────────────────────────────────────────────────────────────────┐
│  Layer 3: CMS SERVICE                                              │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ Background Collector (ticker 15s)                           │ │
│  │  ├─► Probe Worker /health (timeout 2s)                       │ │
│  │  ├─► Probe Kafka Connect API (timeout 2s)                    │ │
│  │  ├─► Probe NATS /jsz (timeout 2s)                            │ │
│  │  ├─► Query Postgres (registry, recon, dlq, activity)         │ │
│  │  └─► Prometheus query (percentiles, consumer lag)            │ │
│  │     WRITE snapshot → Redis key "system_health:snapshot"      │ │
│  │     TTL 60s, with cache_age_seconds field                   │ │
│  └──────────────────────────────────────────────────────────────┘ │
│  Handler `/api/system/health` → Redis GET → return (O(1))          │
└─────────────────────┬──────────────────────────────────────────────┘
                      │
┌────────────────────────────────────────────────────────────────────┐
│  Layer 2: STORAGE                                                  │
│  Prometheus: metrics scraping, histogram_quantile compute         │
│  SigNoz (ClickHouse): traces + logs (severity-sampled)            │
│  Postgres: activity_log (partitioned), alerts_state (dedup)       │
│  Redis: health cache, alert fingerprint dedup, idempotency keys   │
└─────────────────────┬──────────────────────────────────────────────┘
                      │
┌────────────────────────────────────────────────────────────────────┐
│  Layer 1: AGENT (Worker)                                           │
│  - Zap logger → OTel zap core (severity sample)                    │
│  - OTel batch processor (memory_limiter 256MB, fallback console)   │
│  - Prometheus HTTP /metrics (expose on :9090)                      │
│  - E2E latency histogram with buckets optimized cho CDC            │
│  - Single flusher multi-topic activity log (1 TX / 5s)             │
│  - Consumer lag polling (segmentio kafka.Conn admin API)           │
│  - W3C traceparent parse/inject per Kafka message                  │
└────────────────────────────────────────────────────────────────────┘
```

---

## 3. System Health API v3 — Background Collector Pattern

### Thay vì synchronous aggregate (v2 BUG)

```go
// cdc-cms-service/internal/service/system_health_collector.go
type Collector struct {
    redis    *redis.Client
    workerURL, kafkaConnectURL, natsURL string
    pgPool   *pgxpool.Pool
    promAPI  v1.API  // prometheus client
    interval time.Duration
}

func (c *Collector) Run(ctx context.Context) {
    ticker := time.NewTicker(c.interval)  // 15s
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C:
            c.collectAndCache(ctx)
        }
    }
}

func (c *Collector) collectAndCache(ctx context.Context) {
    snapshot := &HealthSnapshot{Timestamp: time.Now()}
    g, gCtx := errgroup.WithContext(ctx)
    
    g.Go(func() error { return c.probeWorker(gCtx, snapshot) })
    g.Go(func() error { return c.probeKafkaConnect(gCtx, snapshot) })
    g.Go(func() error { return c.probeNATS(gCtx, snapshot) })
    g.Go(func() error { return c.probeQueries(gCtx, snapshot) })
    g.Go(func() error { return c.queryPrometheus(gCtx, snapshot) })
    
    _ = g.Wait()  // ignore error — per-section status records it
    
    data, _ := json.Marshal(snapshot)
    c.redis.Set(ctx, "system_health:snapshot", data, 60*time.Second)
}

// Per-probe with individual timeout
func (c *Collector) probeKafkaConnect(ctx context.Context, snap *HealthSnapshot) error {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    
    resp, err := http.DefaultClient.Do(req)
    section := &SectionResult{}
    if err != nil {
        section.Status = "unknown"
        section.Error = err.Error()
    } else if resp.StatusCode != 200 {
        section.Status = "down"
    } else {
        section.Status = "ok"
        section.Data = parseConnectorStatus(resp.Body)
    }
    snap.Pipeline.Debezium = section
    return nil  // never block other probes
}
```

### Handler

```go
func (h *Handler) SystemHealth(c *fiber.Ctx) error {
    data, err := h.redis.Get(ctx, "system_health:snapshot").Bytes()
    if err == redis.Nil {
        return c.Status(503).JSON(fiber.Map{
            "status": "initializing", 
            "message": "collector not ready yet",
        })
    }
    var snap HealthSnapshot
    json.Unmarshal(data, &snap)
    snap.CacheAgeSeconds = int(time.Since(snap.Timestamp).Seconds())
    return c.JSON(snap)
}
```

### Per-section schema
```json
{
  "timestamp": "2026-04-17T10:00:00Z",
  "cache_age_seconds": 3,
  "sections": {
    "infrastructure": {
      "status": "ok",
      "components": {
        "kafka":    { "status": "ok", "latency_ms": 12, "topics": 42 },
        "nats":     { "status": "ok", "latency_ms": 5, "streams": 3 },
        "postgres": { "status": "ok", "latency_ms": 8 },
        "mongodb":  { "status": "ok", "latency_ms": 22 },
        "redis":    { "status": "ok", "latency_ms": 1 }
      }
    },
    "pipeline": {
      "status": "degraded",
      "components": {
        "debezium": {
          "status": "degraded",
          "connector_state": "RUNNING",
          "tasks": [{"id": 0, "state": "FAILED", "trace": "..."}]
        },
        "consumer_lag": { "status": "ok", "total_lag": 1250 }
      }
    },
    "reconciliation": { "status": "ok", "data": [...] },
    "latency": {
      "status": "ok",
      "source": "prometheus",  // or "fallback"
      "p50_ms": 152,
      "p95_ms": 890,
      "p99_ms": 2100
    },
    "alerts": { "status": "warning", "active": [...], "silenced": [...] },
    "recent_events": { "status": "ok", "events": [...] }
  }
}
```

---

## 4. E2E Latency Percentile — FIX SILENT BUG

### v2 BUG (T10)
- "Compute P50/P95/P99 from activity_log" → sample avg batch ≠ true percentile.

### v3 Correct — 2 paths

**Path A (preferred): Prometheus `histogram_quantile`**

```go
func (c *Collector) queryPrometheus(ctx context.Context, snap *HealthSnapshot) error {
    query := `histogram_quantile(0.95, sum by (le) (rate(cdc_e2e_latency_seconds_bucket[5m])))`
    result, _, err := c.promAPI.Query(ctx, query, time.Now())
    if err != nil {
        // Fallback to path B
        return c.computeFromMetricsEndpoint(ctx, snap)
    }
    // ... parse result
    snap.Latency.Source = "prometheus"
    snap.Latency.P95Ms = ...
}
```

**Path B (fallback): scrape Worker `/metrics` + compute in-process**

```go
import "github.com/prometheus/common/expfmt"

func (c *Collector) computeFromMetricsEndpoint(ctx context.Context, snap *HealthSnapshot) error {
    resp, err := http.Get(c.workerURL + "/metrics")
    defer resp.Body.Close()
    
    var parser expfmt.TextParser
    families, _ := parser.TextToMetricFamilies(resp.Body)
    
    hist := families["cdc_e2e_latency_seconds"]
    // Compute percentile from histogram buckets (manual math)
    p50, p95, p99 := histogramPercentiles(hist, 0.5, 0.95, 0.99)
    snap.Latency.Source = "fallback_worker_metrics"
    snap.Latency.P50Ms = int(p50 * 1000)
    // ...
}
```

### Histogram buckets optimized
```go
// pkgs/metrics/prometheus.go
E2ELatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
    Name: "cdc_e2e_latency_seconds",
    Buckets: []float64{
        0.025, 0.05, 0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8,
    },  // exponential factor 2, tuned cho CDC
}, []string{"table_group"})  // NOT {table, op, topic} — cardinality control
```

### Label whitelist
```go
// Gom vào label "table_group" thay vì label {table, op, topic}
// Group theo cdc_table_registry.group field (vd: "payments", "loyalty", "users", "other")
groupLabels := []string{"payments", "loyalty", "users", "merchants", "other"}
// 5 group × 10 buckets = 50 series per histogram → tổng ~200 series
```

### Test semantic correctness
```go
// unit test
func TestHistogramPercentileMatches(t *testing.T) {
    // Inject 100 observations với pattern known
    for i := 0; i < 99; i++ {
        hist.Observe(0.1)  // 99 × 100ms
    }
    hist.Observe(5.0)  // 1 × 5s outlier
    
    // Prom histogram_quantile
    p99 := computeP99FromBuckets(hist)
    assert.GreaterOrEqual(t, p99, 3.2)  // outlier visible
    
    // So với SQL percentile_cont trên activity_log (batch avg)
    sampleAvg := sqlPercentile99(activityLogRows)
    assert.Less(t, sampleAvg, 1.0)  // SILENT BUG - outlier bị hide
    
    t.Log("✅ Histogram correctly captures outlier; activity_log sample_avg misses it.")
}
```

---

## 5. Activity Log — Single Flusher Multi-Topic + Partition

### Current v2 (problem)
- Per-topic goroutine flush → 20+ goroutines → PG contention.
- Không partition → bloat.

### v3 — Consolidated flusher
```go
type ActivityLogger struct {
    buffer chan Entry  // unbounded? No — bounded 10K với drop-oldest
    db     *pgxpool.Pool
    ticker *time.Ticker  // 5s
}

type Entry struct {
    Topic     string
    Operation string
    Details   map[string]any
    Timestamp time.Time
}

func (l *ActivityLogger) Run(ctx context.Context) {
    batch := make([]Entry, 0, 1000)
    for {
        select {
        case e := <-l.buffer:
            batch = append(batch, e)
            if len(batch) >= 1000 {
                l.flush(ctx, batch)
                batch = batch[:0]
            }
        case <-l.ticker.C:
            if len(batch) > 0 {
                l.flush(ctx, batch)
                batch = batch[:0]
            }
        case <-ctx.Done():
            if len(batch) > 0 { l.flush(ctx, batch) }
            return
        }
    }
}

func (l *ActivityLogger) flush(ctx context.Context, batch []Entry) {
    // Multi-row INSERT trong 1 TX
    tx, _ := l.db.Begin(ctx)
    defer tx.Rollback(ctx)
    
    rows := make([][]any, len(batch))
    for i, e := range batch {
        rows[i] = []any{e.Topic, e.Operation, mustJSON(e.Details), e.Timestamp}
    }
    copyCount, err := tx.CopyFrom(ctx, 
        pgx.Identifier{"cdc_activity_log"}, 
        []string{"topic", "operation", "details", "created_at"},
        pgx.CopyFromRows(rows))
    if err == nil {
        tx.Commit(ctx)
    }
    metrics.ActivityLogFlushed.Add(float64(copyCount))
}
```

### Migration partition
```sql
-- migrations/010_activity_log_partition.sql
-- Tạo parent table partitioned
CREATE TABLE cdc_activity_log_new (
    id BIGSERIAL,
    topic TEXT,
    operation TEXT NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (created_at, id)
) PARTITION BY RANGE (created_at);

-- Daily partitions (pg_partman hoặc manual)
SELECT partman.create_parent(
    p_parent_table := 'public.cdc_activity_log_new',
    p_control := 'created_at',
    p_type := 'native',
    p_interval := 'daily',
    p_premake := 7
);

-- Migration: copy old → new (online), swap table names
-- Retention: drop partition > 30 days
```

### Index cho query "last 10 events"
```sql
CREATE INDEX idx_activity_log_created_at_desc ON cdc_activity_log_new (created_at DESC);
CREATE INDEX idx_activity_log_operation ON cdc_activity_log_new (operation, created_at DESC);
```

---

## 6. OTel — Severity Sample + Memory Limiter + Fallback

### Config v3
```yaml
# config-local.yml
otel:
  enabled: true
  service_name: cdc-worker
  traces:
    endpoint: http://localhost:4318
    sample_ratio: 0.1  # 10% traces
  metrics:
    endpoint: http://localhost:4318
    interval: 30s
  logs:
    grpc_endpoint: localhost:4317
    sample_by_severity:
      debug: 0.0  # drop debug trong prod
      info:  0.1
      warn:  1.0
      error: 1.0
      fatal: 1.0
    memory_limiter:
      limit_mib: 256
      spike_limit_mib: 64
    batch:
      max_queue_size: 8192
      scheduled_delay: 5s
      max_export_batch_size: 512
    fallback:
      enabled: true
      degraded_after_errors: 10  # after 10 consecutive export errors
      console_only_until: 5m     # fallback to console for 5 min
```

### Zap core wrapper
```go
// pkgs/observability/otel.go
func NewSeverityAwareCore(otelCore zapcore.Core, cfg *Config) zapcore.Core {
    return &severityAwareCore{
        Core: otelCore,
        sampler: map[zapcore.Level]float64{
            zapcore.DebugLevel: cfg.SampleBySeverity.Debug,
            zapcore.InfoLevel:  cfg.SampleBySeverity.Info,
            zapcore.WarnLevel:  cfg.SampleBySeverity.Warn,
            zapcore.ErrorLevel: cfg.SampleBySeverity.Error,
            zapcore.FatalLevel: cfg.SampleBySeverity.Fatal,
        },
    }
}

func (c *severityAwareCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
    ratio := c.sampler[entry.Level]
    if rand.Float64() > ratio { return ce }  // drop
    return c.Core.Check(entry, ce)
}
```

### Fallback to console khi SigNoz down
```go
type fallbackCore struct {
    primary   zapcore.Core  // OTel
    secondary zapcore.Core  // console
    errorRate *ratemeter.Meter  // track export errors
    degraded  atomic.Bool
    degradedUntil time.Time
}

func (c *fallbackCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
    if c.degraded.Load() && time.Now().Before(c.degradedUntil) {
        return c.secondary.Write(entry, fields)  // console only
    }
    err := c.primary.Write(entry, fields)
    if err != nil {
        c.errorRate.Hit()
        if c.errorRate.Rate(1*time.Minute) > 10 {
            c.degraded.Store(true)
            c.degradedUntil = time.Now().Add(5 * time.Minute)
            metrics.OTelDegraded.Inc()
        }
    }
    return c.secondary.Write(entry, fields)  // always write console too
}
```

---

## 7. Prometheus `/metrics` + Cardinality Budget

### Expose Worker /metrics (GAP từ archaeology)
```go
// cmd/worker/main.go
import "github.com/prometheus/client_golang/prometheus/promhttp"

go func() {
    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.Handler())
    mux.HandleFunc("/health", healthHandler)
    log.Fatal(http.ListenAndServe(":9090", mux))
}()
```

### Cardinality audit
| Metric | Labels | Cardinality |
|:-------|:-------|:------------|
| `cdc_events_processed_total` | `table_group, op` | 5 × 4 = 20 |
| `cdc_sync_success_total` | `table_group` | 5 |
| `cdc_sync_failed_total` | `table_group, reason` | 5 × 10 = 50 |
| `cdc_e2e_latency_seconds` | `table_group` × 10 buckets | 50 |
| `cdc_recon_run_duration_seconds` | `table, tier` × 9 buckets | 200 × 3 × 9 = 5400 ⚠️ |
| `cdc_recon_mismatch_count` | `table, tier` | 200 × 3 = 600 |
| `cdc_kafka_consumer_lag_seconds` | `topic, partition` | 20 × 5 = 100 |
| **TOTAL** | | **~6500 series** |

**Optimization**: Recon metrics cũng group by `table_group` + `top_10_tables` để giảm:
- `cdc_recon_run_duration_seconds{table_group, tier}` = 5 × 3 × 9 = 135
- Top 10 bảng lớn nhất: label `table` riêng → 10 × 3 × 9 = 270
- Others gộp → 135 + 270 = 405
- Total series xuống < 1500.

### Prometheus scrape config (cho DevOps)
```yaml
# prometheus.yml
scrape_configs:
  - job_name: cdc-worker
    static_configs:
      - targets: ['cdc-worker:9090']
    scrape_interval: 15s
  - job_name: cdc-cms
    static_configs:
      - targets: ['cdc-cms:9091']
```

---

## 8. Kafka Consumer Lag (segmentio/kafka-go specific)

### Vấn đề v2
- T11 mơ hồ. Segmentio **không có** ClusterAdmin API như Sarama.

### v3 approach
```go
// pkgs/kafka/lag.go
import "github.com/segmentio/kafka-go"

func FetchConsumerLag(ctx context.Context, brokers []string, groupID, topic string) (map[int]int64, error) {
    conn, err := kafka.Dial("tcp", brokers[0])
    if err != nil { return nil, err }
    defer conn.Close()
    
    // 1. Fetch high watermark (latest offset) per partition
    partitions, err := conn.ReadPartitions(topic)
    if err != nil { return nil, err }
    
    lagByPartition := make(map[int]int64)
    for _, p := range partitions {
        leaderConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", p.Leader.Host, p.Leader.Port))
        if err != nil { continue }
        defer leaderConn.Close()
        
        latest, err := leaderConn.ReadLastOffset()
        if err != nil { continue }
        
        // 2. Fetch committed offset for group via OffsetFetch protocol
        req := &kafka.OffsetFetchRequest{
            GroupID: groupID,
            Topics: map[string][]int{topic: {p.ID}},
        }
        // ... call kafka protocol
        committed, _ := fetchCommittedOffset(ctx, brokers, groupID, topic, p.ID)
        
        lagByPartition[p.ID] = latest - committed
    }
    return lagByPartition, nil
}
```

### Alternative: Prometheus `kafka_exporter`
- Deploy `danielqsj/kafka_exporter` as sidecar → Prom scrape → `kafka_consumergroup_lag` metric readily available.
- **Đề xuất**: Dùng kafka_exporter thay vì self-implement. Ít code, standardized.

### Prometheus metric (cả 2 option đều expose)
```go
ConsumerLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
    Name: "cdc_kafka_consumer_lag_messages",
}, []string{"topic", "partition", "group"})
```

---

## 9. Trace Context — W3C traceparent qua Kafka

### Gap từ archaeology
- Không có `otelkafka` wrapper → trace đứt.

### v3 — Manual propagation
```go
// kafka_consumer.go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

func (h *Handler) handleMessage(ctx context.Context, msg kafka.Message) {
    // Extract trace context từ Kafka headers (W3C)
    headers := kafkaHeadersToMap(msg.Headers)
    carrier := propagation.MapCarrier(headers)
    ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
    
    // Start span
    ctx, span := tracer.Start(ctx, "kafka.consume",
        trace.WithAttributes(
            attribute.String("messaging.system", "kafka"),
            attribute.String("messaging.destination", msg.Topic),
            attribute.Int64("messaging.kafka.partition", int64(msg.Partition)),
            attribute.Int64("messaging.kafka.offset", msg.Offset),
            attribute.Int64("source.ts_ms", extractSourceTs(msg.Value)),
        ))
    defer span.End()
    
    // ... process
}
```

### Producer side (Debezium)
- Debezium **không tự inject** W3C headers (không có OTel interceptor nativelly).
- **Accept v1 limitation**: Worker start root span mới, attribute Kafka metadata. Trace từ Kafka consume → downstream đầy đủ. Không có parent → source Mongo.
- **Future**: Deploy OTel Kafka Connect interceptor (Debezium 2.5+ có support limited).

---

## 10. Alert State Machine

### Bảng
```sql
CREATE TABLE cdc_alerts (
    id UUID PRIMARY KEY,
    fingerprint TEXT NOT NULL,  -- hash(name + labels)
    name TEXT NOT NULL,
    severity TEXT NOT NULL,
    labels JSONB,
    description TEXT,
    status TEXT NOT NULL,  -- firing | resolved | acknowledged | silenced
    fired_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    ack_by TEXT,
    ack_at TIMESTAMPTZ,
    silenced_until TIMESTAMPTZ,
    UNIQUE (fingerprint)
);
CREATE INDEX idx_alerts_status ON cdc_alerts (status, fired_at DESC);
```

### Logic
```go
func (am *AlertManager) Fire(alert Alert) {
    fp := alert.Fingerprint()
    existing, _ := am.db.GetByFingerprint(fp)
    
    if existing != nil {
        if existing.Status == "silenced" && time.Now().Before(existing.SilencedUntil) {
            return  // skip
        }
        if existing.Status == "firing" {
            // dedup — update fired_at only
            am.db.UpdateFiredAt(fp, time.Now())
            return
        }
    }
    
    // New or resolved → now firing
    am.db.Upsert(Alert{
        Fingerprint: fp,
        Status: "firing",
        FiredAt: time.Now(),
        // ...
    })
    am.notify(alert)
}

func (am *AlertManager) Ack(fp string, user string) error {
    return am.db.UpdateStatus(fp, "acknowledged", map[string]any{
        "ack_by": user,
        "ack_at": time.Now(),
    })
}

func (am *AlertManager) Silence(fp string, until time.Time) error {
    return am.db.UpdateStatus(fp, "silenced", map[string]any{
        "silenced_until": until,
    })
}
```

### FE rendering
- Firing (unack) → red banner, count, list.
- Acknowledged → grey footer "3 acknowledged (click to view)".
- Silenced → hidden unless admin toggle "show silenced".

---

## 11. SLO Definition

### SLOs
| SLO | Target | Budget (30 days) |
|:----|:-------|:-----------------|
| SLO-1: E2E latency P99 ≤ 5s | 99% 5-min windows | 86 windows violation |
| SLO-2: Reconciliation drift = 0 | 99.9% daily checks | 0.2 days violation |
| SLO-3: Worker availability | ≥ 99.95% | 21.6 min downtime |
| SLO-4: DLQ "dead" records | 0 records > 24h stuck | 0 |
| SLO-5: Kafka retention safety | 0 events lost due to retention expiry | 0 |

### Derived alert rules
```yaml
# prometheus alert rules
groups:
  - name: cdc_slo
    rules:
      - alert: E2ELatencyP99High
        expr: histogram_quantile(0.99, sum by (le) (rate(cdc_e2e_latency_seconds_bucket[5m]))) > 5
        for: 5m
        labels: { severity: warning, slo: "1" }
      
      - alert: E2ELatencyP99Critical
        expr: histogram_quantile(0.99, sum by (le) (rate(cdc_e2e_latency_seconds_bucket[5m]))) > 5
        for: 15m
        labels: { severity: critical, slo: "1" }
      
      - alert: ReconDrift
        expr: cdc_recon_mismatch_count > 0
        for: 1h
        labels: { severity: warning, slo: "2" }
      
      - alert: WorkerDown
        expr: up{job="cdc-worker"} == 0
        for: 1m
        labels: { severity: critical, slo: "3" }
      
      - alert: DLQDeadRecords
        expr: cdc_dlq_stuck_records > 0  # rows with status=dead_letter
        for: 5m
        labels: { severity: critical, slo: "4" }
      
      - alert: KafkaRetentionRisk
        expr: kafka_consumergroup_lag_seconds > 14 * 86400 * 0.7
        for: 5m
        labels: { severity: warning, slo: "5" }
```

---

## 12. FE v3 — React Query + Per-section

### Add React Query
```json
// cdc-cms-web/package.json
{
  "dependencies": {
    "@tanstack/react-query": "^5.59.0",
    "@tanstack/react-query-devtools": "^5.59.0"
  }
}
```

### Hook
```typescript
// src/hooks/useSystemHealth.ts
import { useQuery } from '@tanstack/react-query';

export function useSystemHealth() {
  return useQuery({
    queryKey: ['system-health'],
    queryFn: () => axios.get('/api/system/health').then(r => r.data),
    refetchInterval: 30_000,
    staleTime: 25_000,
    retry: 2,
  });
}
```

### Per-section render
```tsx
// src/pages/SystemHealth.tsx
function SystemHealth() {
  const { data, isLoading, error } = useSystemHealth();
  if (isLoading) return <Skeleton />;
  if (error) return <GlobalErrorBanner />;
  
  return (
    <>
      {data.cache_age_seconds > 60 && (
        <Alert type="warning" message={`Data ${data.cache_age_seconds}s old`} />
      )}
      <Section name="Infrastructure" section={data.sections.infrastructure} />
      <Section name="Pipeline" section={data.sections.pipeline} />
      <Section name="Reconciliation" section={data.sections.reconciliation} />
      <Section name="Latency" section={data.sections.latency} />
      <AlertsBanner alerts={data.sections.alerts} onAck={handleAck} />
      <RecentEvents events={data.sections.recent_events.events} />
    </>
  );
}

function Section({ name, section }) {
  if (section.status === 'unknown') {
    return <Card title={name}><Empty description={`Unknown — ${section.error}`} /></Card>;
  }
  if (section.status === 'down') {
    return <Card title={name} type="inner" bordered={false}><Alert type="error" /></Card>;
  }
  // ...
}
```

### Confirm modal cho destructive
```tsx
function RestartConnectorButton({ name }) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const mutation = useMutation({
    mutationFn: () => axios.post(`/api/connectors/${name}/restart`, 
      { reason }, 
      { headers: { 'Idempotency-Key': crypto.randomUUID() } }),
  });
  
  return (
    <>
      <Button onClick={() => setOpen(true)} danger>Restart</Button>
      <Modal open={open} onOk={() => mutation.mutate()}>
        <p>Restart connector <b>{name}</b>?</p>
        <Input.TextArea 
          placeholder="Lý do (min 10 chars)" 
          value={reason} 
          onChange={e => setReason(e.target.value)} 
          minLength={10} 
          required 
        />
      </Modal>
    </>
  );
}
```

---

## 13. Tasks v3 (FINAL)

### Phase 0 — Foundation (Week 1, Quick Wins)
- [ ] **O0-1**: Worker expose `/metrics` endpoint (bổ sung `promhttp.Handler` vào `cmd/worker/main.go` + cổng 9090)
- [ ] **O0-2**: Migration 010 — partition `cdc_activity_log` daily + drop > 30 days
- [ ] **O0-3**: DevOps coordinate — Prometheus scrape target config (nếu Prom production khác SigNoz OTLP)

### Phase 1 — Fix Silent Bug (Week 1, CRITICAL)
- [ ] **O1-1**: Rewrite `system_health_handler.go` T10 — bỏ SQL percentile_cont trên activity_log
- [ ] **O1-2**: Implement Prometheus HTTP client (`prometheus/client_golang/api/prometheus/v1`)
- [ ] **O1-3**: Path A `histogram_quantile` query; Path B fallback parse `/metrics` histogram
- [ ] **O1-4**: Unit test semantic correctness (outlier visibility)

### Phase 2 — Background Collector (Week 1)
- [ ] **O2-1**: New `system_health_collector.go` với errgroup + per-probe timeout 2s
- [ ] **O2-2**: Redis cache key `system_health:snapshot` TTL 60s + `cache_age_seconds`
- [ ] **O2-3**: Handler đơn giản hóa — chỉ GET Redis + return
- [ ] **O2-4**: Graceful degradation — probe fail → section status "unknown", không block khác

### Phase 3 — OTel Hardening (Week 2)
- [ ] **O3-1**: Severity-aware sample (`severityAwareCore`)
- [ ] **O3-2**: Memory limiter 256 MB
- [ ] **O3-3**: Fallback console-only khi export error rate > 10/min
- [ ] **O3-4**: Metric `otel_exporter_degraded` expose
- [ ] **O3-5**: Load test SigNoz down 10 phút → Worker RAM stable

### Phase 4 — Activity Log + Cardinality (Week 2)
- [ ] **O4-1**: Single flusher multi-topic buffered channel (bounded 10K)
- [ ] **O4-2**: Multi-row INSERT qua `CopyFrom` 1 TX / 5s
- [ ] **O4-3**: Migration cardinality — labels `table_group` thay `table, op, topic`
- [ ] **O4-4**: Histogram buckets tối ưu (exponential factor 2, 10 buckets)
- [ ] **O4-5**: Top 10 tables có label riêng, others gộp

### Phase 5 — Kafka Consumer Lag (Week 2)
- [ ] **O5-1**: Option A: kafka_exporter sidecar + Prom scrape (ưu tiên)
- [ ] **O5-2**: Option B: self-implement lag polling với segmentio `kafka.Dial` + OffsetFetch (fallback)
- [ ] **O5-3**: Alert rule lag > 70%/90% retention window

### Phase 6 — Trace Context Kafka (Week 3)
- [ ] **O6-1**: W3C traceparent extract từ Kafka headers
- [ ] **O6-2**: Span per message với Kafka attributes + source.ts_ms
- [ ] **O6-3**: Span propagate xuống BatchBuffer → PG upsert

### Phase 7 — Alert State Machine (Week 3)
- [ ] **O7-1**: Migration 011 — `cdc_alerts` table
- [ ] **O7-2**: AlertManager với fingerprint dedup
- [ ] **O7-3**: Ack / Silence / Resolve API
- [ ] **O7-4**: FE hiển thị per-state

### Phase 8 — SLO + Alert Rules (Week 3)
- [ ] **O8-1**: Tài liệu `07_slo_definition.md` trong workspace
- [ ] **O8-2**: Alert rules YAML file (apply qua Prom config OR SigNoz)
- [ ] **O8-3**: Dashboards (SigNoz / Grafana) cho từng SLO

### Phase 9 — FE React Query + Hardening (Week 4)
- [ ] **O9-1**: Thêm `@tanstack/react-query` + QueryClient provider
- [ ] **O9-2**: Refactor SystemHealth + DataIntegrity → useQuery hooks
- [ ] **O9-3**: Per-section status rendering (unknown/degraded/down)
- [ ] **O9-4**: Confirm modal + Idempotency-Key cho destructive (Restart, Reset, Heal)
- [ ] **O9-5**: Stale-while-revalidate 25s + refetch 30s

---

## 14. Files Impact

### Worker (`centralized-data-service/`)
| File | Action | Change |
|:-----|:-------|:-------|
| `cmd/worker/main.go` | **EDIT** | Expose /metrics, fallback core, severity sampler |
| `pkgs/observability/otel.go` | **REWRITE** | Severity sample, memory limiter, fallback |
| `pkgs/observability/trace.go` | **NEW** | W3C traceparent helper |
| `pkgs/metrics/http.go` | **NEW** | /metrics endpoint |
| `pkgs/metrics/prometheus.go` | **EDIT** | Labels `table_group`, optimized buckets |
| `pkgs/kafka/lag.go` | **NEW** | Consumer lag polling (or skip if kafka_exporter used) |
| `internal/handler/kafka_consumer.go` | **EDIT** | W3C extract, span creation, single flusher activity log |
| `internal/handler/activity_logger.go` | **NEW** | Consolidated buffered flusher |

### CMS (`cdc-cms-service/`)
| File | Action | Change |
|:-----|:-------|:-------|
| `internal/service/system_health_collector.go` | **NEW** | Background collector |
| `internal/api/system_health_handler.go` | **REWRITE** | Cache read only, no aggregate |
| `internal/service/prom_client.go` | **NEW** | Prometheus API client + fallback /metrics scrape |
| `internal/service/alert_manager.go` | **NEW** | State machine |
| `internal/api/alerts_handler.go` | **NEW** | Ack/Silence/List |
| `migrations/010_activity_log_partition.sql` | **NEW** | |
| `migrations/011_alerts.sql` | **NEW** | |
| `internal/middleware/idempotency.go` | **NEW** | Shared với data integrity v3 |

### Infra
| File | Action | Change |
|:-----|:-------|:-------|
| `docker-compose.yml` | **EDIT** | Add `prometheus` + `kafka_exporter` service (nếu DevOps confirm prod dùng) |
| `infra/prometheus.yml` | **NEW** | Scrape config |
| `infra/alert_rules.yml` | **NEW** | SLO-derived rules |

### FE (`cdc-cms-web/`)
| File | Action | Change |
|:-----|:-------|:-------|
| `package.json` | **EDIT** | + `@tanstack/react-query` |
| `src/main.tsx` | **EDIT** | QueryClientProvider |
| `src/hooks/useSystemHealth.ts` | **NEW** | |
| `src/hooks/useReconStatus.ts` | **NEW** | |
| `src/pages/SystemHealth.tsx` | **REWRITE** | Per-section + React Query |
| `src/pages/DataIntegrity.tsx` | **EDIT** | React Query + confirm modal |
| `src/components/ConfirmDestructiveModal.tsx` | **NEW** | Reason field required |
| `src/api/client.ts` | **EDIT** | Idempotency-Key interceptor |

---

## 15. Definition of Done

- [ ] `/api/system/health` p99 < 50ms load test 100 QPS.
- [ ] Kafka Connect down → section "pipeline.debezium" status="unknown", other sections "ok".
- [ ] T10 percentile semantic validation pass (outlier visible).
- [ ] SigNoz down 10 phút → Worker RAM không tăng > 50 MB; log tiếp tục ghi console.
- [ ] Activity log insert rate < 0.5 TX/s trung bình (single flusher).
- [ ] Activity log drop partition > 30d tự động.
- [ ] Prometheus total series < 10K sau deploy v3.
- [ ] Consumer lag metric có data liên tục.
- [ ] Trace từ Kafka consume → PG upsert trace được trên SigNoz (1 trace_id xuyên suốt).
- [ ] Alert flap dedup: fire/resolve/fire trong 5 phút → 1 banner.
- [ ] Ack/Silence hoạt động.
- [ ] SLO dashboard + alert rules deployed.
- [ ] FE confirm modal + reason + Idempotency-Key cho mọi destructive button.
- [ ] FE React Query stale-while-revalidate, không giật khi refetch.

---

## 16. Changelog vs v2

| Aspect | v2 | v3 |
|:-------|:---|:---|
| System Health aggregate | Sync 5+ external calls | Background collector + Redis cache |
| API p99 | 2-5s | < 50ms |
| Percentile compute | From activity_log (SILENT BUG) | Prometheus histogram_quantile + fallback |
| Activity log flush | Per-topic goroutine × 20 | Single flusher multi-topic 1 TX/5s |
| Activity log retention | Unbounded | Partition daily, drop > 30d |
| OTel sample | 1.0 all | 0.1 info / 1.0 error + memory limiter |
| OTel fallback | None | Console-only 5m when SigNoz degraded |
| Prometheus /metrics | Defined but not exposed | Expose on :9090 |
| Prometheus cardinality | ~150K series | < 10K (label_group + top10) |
| Kafka consumer lag | Mơ hồ | kafka_exporter OR self-implement |
| Trace context Kafka | None | W3C traceparent extract/create span |
| Alert state | Banner "show raw" | State machine: fire/resolve/ack/silence |
| Alert dedup | None | Fingerprint-based |
| SLO | None | 5 SLO + derived alert rules |
| FE data fetching | Axios thuần | React Query staleWhileRevalidate |
| FE partial data | Không | Per-section status |
| Destructive RBAC | None | ops-admin + confirm modal + Idempotency + audit |

---

## 17. Open Questions

1. **Prom production server location**: cần DevOps xác nhận URL (nếu không có → plan v3 vẫn chạy được qua SigNoz metric query API — SigNoz cũng hỗ trợ PromQL trên ClickHouse).
2. **Kafka exporter deploy**: DevOps approve sidecar container?
3. **Alert routing**: banner + log, hay thêm Slack/Telegram webhook? (v3 plan dừng ở banner + log, webhook = future).
4. **SigNoz OTLP metric** vs **Prometheus scrape**: dùng cái nào chính? v3 recommend dùng cả (OTLP push cho low-cardinality, Prom scrape cho high-cardinality histogram).
