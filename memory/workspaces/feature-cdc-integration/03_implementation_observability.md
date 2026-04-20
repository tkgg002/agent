# Implementation: Observability

> Date: 2026-04-16
> Phase: observability
> Ref: 02_plan_observability.md

---

## 1. SystemHealthHandler (CMS API)

File: `cdc-cms-service/internal/api/system_health_handler.go`

### Dependencies
- `gorm.DB` — Postgres queries (registry, activity, recon, failed logs)
- `http.Client` — poll Worker health, Kafka Connect, NATS monitoring
- `AirbyteClient` — connection status
- `RedisCache` — ping

### Endpoints
| Method | Path | Purpose |
|:-------|:-----|:--------|
| GET | /api/system/health | Full system health aggregate |

### Implementation
```go
type SystemHealthHandler struct {
    db            *gorm.DB
    airbyteClient *airbyte.Client
    redisCache    *rediscache.RedisCache
    workerURL     string    // http://localhost:8082
    kafkaConnectURL string  // http://localhost:18083
    natsMonitorURL  string  // http://localhost:18222
    logger        *zap.Logger
}

func (h *SystemHealthHandler) Health(c *fiber.Ctx) error {
    // 1. Worker: GET workerURL/health
    // 2. Kafka Connect: GET kafkaConnectURL/connectors/goopay-mongodb-cdc/status
    // 3. NATS: GET natsMonitorURL/jsz
    // 4. Postgres: query cdc_table_registry count
    // 5. Redis: ping
    // 6. Airbyte: ListConnections
    // 7. Pipeline: query cdc_activity_log + failed_sync_logs last 24h
    // 8. Recon: query cdc_reconciliation_report latest per table
    // 9. Alerts: compute from statuses
}
```

### HTTP polling logic (non-blocking, timeout 3s)
```go
func httpGetJSON(url string, timeout time.Duration) (map[string]interface{}, error) {
    client := &http.Client{Timeout: timeout}
    resp, err := client.Get(url)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}
```

---

## 2. SystemHealth FE Page

File: `cdc-cms-web/src/pages/SystemHealth.tsx`

### Layout
```
Row 1: 7 component cards (Worker, Kafka, Debezium, NATS, Postgres, Redis, Airbyte)
        Each: icon, status badge, key metric

Row 2: 4 pipeline stat cards (Events 24h, Failed 24h, Latency, Throughput)
        + Recon summary (N matched / N drifted)

Row 3: Alerts banner (warning/critical)

Row 4: Recent Activity (last 10 entries, compact table)
```

### Auto-refresh
```typescript
useEffect(() => {
    const interval = setInterval(fetchHealth, 30000);
    return () => clearInterval(interval);
}, []);
```

---

## 3. Kafka Consumer → Activity Log (batch)

File: `centralized-data-service/internal/handler/kafka_consumer.go`

### Logic
- Counter per table (atomic int64)
- Flush timer: every 30 seconds
- On flush: for each table with count > 0 → write 1 Activity Log entry
- Details: `{"events": N, "tables": {"export_jobs": 5, "refund_requests": 2}}`

```go
type eventBatchLogger struct {
    counts  sync.Map  // table name → *int64
    db      *gorm.DB
    ticker  *time.Ticker
}

func (ebl *eventBatchLogger) Increment(table string) {
    val, _ := ebl.counts.LoadOrStore(table, new(int64))
    atomic.AddInt64(val.(*int64), 1)
}

func (ebl *eventBatchLogger) Flush() {
    details := make(map[string]int64)
    total := int64(0)
    ebl.counts.Range(func(key, val interface{}) bool {
        table := key.(string)
        count := atomic.SwapInt64(val.(*int64), 0)
        if count > 0 {
            details[table] = count
            total += count
        }
        return true
    })
    if total > 0 {
        detailsJSON, _ := json.Marshal(details)
        now := time.Now()
        ebl.db.Create(&model.ActivityLog{
            Operation: "kafka-consume-batch", TargetTable: "*",
            Status: "success", RowsAffected: total,
            Details: detailsJSON, TriggeredBy: "kafka",
            StartedAt: now, CompletedAt: &now,
        })
    }
}
```

---

## 4. Command Handler → Activity Log

File: `centralized-data-service/internal/handler/command_handler.go`

### Current publishResult
```go
func (h *CommandHandler) publishResult(msg *nats.Msg, result CommandResult) {
    data, _ := json.Marshal(result)
    if msg.Reply != "" { msg.Respond(data) }
    h.logger.Info("command result", zap.String("payload", string(data)))
}
```

### After: + Activity Log
```go
func (h *CommandHandler) publishResult(msg *nats.Msg, result CommandResult) {
    data, _ := json.Marshal(result)
    if msg.Reply != "" { msg.Respond(data) }
    h.logger.Info("command result", zap.String("payload", string(data)))
    
    // Activity Log
    now := time.Now()
    var errPtr *string
    if result.Error != "" { errPtr = &result.Error }
    h.db.Create(&model.ActivityLog{
        Operation: "cmd-" + result.Command, TargetTable: result.TargetTable,
        Status: result.Status, RowsAffected: int64(result.RowsAffected),
        ErrorMessage: errPtr, TriggeredBy: "nats-command",
        StartedAt: now, CompletedAt: &now,
    })
}
```
Cần: `h.db` access — CommandHandler đã có `h.db`.

---

## 5. E2E Latency Tracking

File: `centralized-data-service/pkgs/metrics/prometheus.go`

```go
E2ELatency = promauto.NewHistogram(prometheus.HistogramOpts{
    Name:    "cdc_e2e_latency_seconds",
    Help:    "End-to-end latency from Kafka message to Postgres insert",
    Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s → 51.2s
})
```

File: `centralized-data-service/internal/handler/kafka_consumer.go`
```go
// After successful upsert in processMessage:
e2eLatency := time.Since(msg.Time)
metrics.E2ELatency.Observe(e2eLatency.Seconds())
```

---

## 6. Config additions

### CMS config-local.yml
```yaml
system:
  workerUrl: http://localhost:8082
  kafkaConnectUrl: http://localhost:18083
  natsMonitorUrl: http://localhost:18222
```

### CMS config.go
```go
type SystemConfig struct {
    WorkerURL       string `mapstructure:"workerUrl"`
    KafkaConnectURL string `mapstructure:"kafkaConnectUrl"`
    NatsMonitorURL  string `mapstructure:"natsMonitorUrl"`
}
```
