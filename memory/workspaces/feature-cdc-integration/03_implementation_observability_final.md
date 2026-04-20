# Implementation: Observability (FINAL)

> Date: 2026-04-16
> Merged from user's deep requirements

---

## 1. SystemHealthHandler (CMS API)

### Response Schema `/api/system/health`
```json
{
  "timestamp": "2026-04-16T16:00:00Z",
  "overall": "healthy | degraded | critical",

  "infrastructure": {
    "kafka":    {"status": "up|down", "broker": "...", "topics": N, "total_lag": N},
    "nats":     {"status": "up|down", "streams": N, "consumers": N},
    "postgres": {"status": "up|down", "tables_registered": N, "tables_created": N, "total_rows": N},
    "mongodb":  {"status": "up|down", "databases": N},
    "redis":    {"status": "up|down"}
  },

  "cdc_pipeline": {
    "debezium": {
      "status": "running|paused|failed|unknown",
      "connector": "goopay-mongodb-cdc",
      "tasks": [{"id": 0, "state": "RUNNING|FAILED", "trace": "...java stack trace if FAILED"}]
    },
    "consumer_lag": [
      {"topic": "cdc.goopay.centralized-export-service.export-jobs", "lag": 0},
      {"topic": "cdc.goopay.payment-bill-service.refund-requests", "lag": 0}
    ],
    "throughput_msg_per_sec": 0.5
  },

  "reconciliation": [
    {"table": "export_jobs", "source_count": 115, "dest_count": 113, "drift_pct": 1.7, "status": "drift", "last_check": "..."},
    {"table": "refund_requests", "source_count": 1714, "dest_count": 1713, "drift_pct": 0.06, "status": "drift", "last_check": "..."}
  ],

  "latency": {
    "p50_ms": 800,
    "p95_ms": 2000,
    "p99_ms": 5000
  },

  "failed_sync": {
    "count_24h": 2,
    "count_1h": 0
  },

  "alerts": [
    {"level": "warning", "component": "reconciliation", "message": "2 tables have data drift (export_jobs: 1.7%, refund_requests: 0.06%)"},
    {"level": "info", "component": "otel", "message": "SigNoz not connected"}
  ],

  "recent_events": [
    {"time": "...", "operation": "kafka-consume-batch", "table": "export_jobs", "details": "processed: 5, success: 5"},
    {"time": "...", "operation": "cmd-bridge-airbyte", "table": "refund_requests", "details": "rows: 1712"}
  ]
}
```

### Implementation Logic
```go
func (h *SystemHealthHandler) Health(c *fiber.Ctx) error {
    ctx := c.Context()
    result := make(map[string]interface{})
    
    // 1. Infrastructure (parallel HTTP calls, timeout 3s each)
    var wg sync.WaitGroup
    infra := make(map[string]interface{})
    
    // Kafka: topic list + consumer lag
    wg.Add(1); go func() { defer wg.Done(); infra["kafka"] = h.checkKafka(ctx) }()
    // NATS: GET natsMonitorUrl/jsz
    wg.Add(1); go func() { defer wg.Done(); infra["nats"] = h.checkNATS(ctx) }()
    // Postgres: ping + counts
    wg.Add(1); go func() { defer wg.Done(); infra["postgres"] = h.checkPostgres(ctx) }()
    // MongoDB: ping
    wg.Add(1); go func() { defer wg.Done(); infra["mongodb"] = h.checkMongoDB(ctx) }()
    // Redis: ping
    wg.Add(1); go func() { defer wg.Done(); infra["redis"] = h.checkRedis(ctx) }()
    wg.Wait()
    
    // 2. CDC Pipeline: Debezium + consumer lag + throughput
    // GET kafkaConnectUrl/connectors/goopay-mongodb-cdc/status
    // Parse: connector.state, tasks[].state, tasks[].trace
    pipeline := h.checkCDCPipeline(ctx)
    
    // 3. Reconciliation: latest report per table + drift %
    recon := h.getReconStatus(ctx)
    
    // 4. Latency: query Prometheus histogram or compute from activity_log
    latency := h.getLatencyPercentiles(ctx)
    
    // 5. Failed sync: count last 24h + 1h
    failed := h.getFailedSyncCount(ctx)
    
    // 6. Alerts: compute from all data
    alerts := h.computeAlerts(infra, pipeline, recon, failed)
    
    // 7. Recent events: last 10 activity log
    events := h.getRecentEvents(ctx)
    
    // 8. Overall status
    overall := "healthy"
    for _, a := range alerts { if a.Level == "critical" { overall = "critical"; break } else if a.Level == "warning" { overall = "degraded" } }
    
    result["timestamp"] = time.Now()
    result["overall"] = overall
    result["infrastructure"] = infra
    result["cdc_pipeline"] = pipeline
    result["reconciliation"] = recon
    result["latency"] = latency
    result["failed_sync"] = failed
    result["alerts"] = alerts
    result["recent_events"] = events
    
    return c.JSON(result)
}
```

### Debezium Trace Extraction
```go
func (h *SystemHealthHandler) checkDebezium(ctx context.Context) map[string]interface{} {
    resp, err := httpGetJSON(h.kafkaConnectURL+"/connectors/goopay-mongodb-cdc/status", 3*time.Second)
    if err != nil { return map[string]interface{}{"status": "unknown", "error": err.Error()} }
    
    connector := resp["connector"].(map[string]interface{})
    tasks := resp["tasks"].([]interface{})
    
    var taskDetails []map[string]interface{}
    for _, t := range tasks {
        task := t.(map[string]interface{})
        detail := map[string]interface{}{
            "id": task["id"], "state": task["state"],
        }
        if task["state"] == "FAILED" {
            // Bóc trace lỗi — truncate 500 chars cho UI
            if trace, ok := task["trace"].(string); ok {
                if len(trace) > 500 { trace = trace[:500] + "..." }
                detail["trace"] = trace
            }
        }
        taskDetails = append(taskDetails, detail)
    }
    
    return map[string]interface{}{
        "status":    connector["state"],
        "connector": "goopay-mongodb-cdc",
        "tasks":     taskDetails,
    }
}
```

---

## 2. Kafka Consumer Batch Activity Log

```go
type eventBatchLogger struct {
    counters map[string]*batchCounter  // topic → counter
    mu       sync.Mutex
    db       *gorm.DB
    interval time.Duration
}

type batchCounter struct {
    processed int64
    success   int64
    failed    int64
    startTime time.Time
}

func (ebl *eventBatchLogger) Record(topic string, success bool) {
    ebl.mu.Lock()
    defer ebl.mu.Unlock()
    if ebl.counters[topic] == nil {
        ebl.counters[topic] = &batchCounter{startTime: time.Now()}
    }
    c := ebl.counters[topic]
    c.processed++
    if success { c.success++ } else { c.failed++ }
}

func (ebl *eventBatchLogger) Flush() {
    ebl.mu.Lock()
    defer ebl.mu.Unlock()
    
    for topic, c := range ebl.counters {
        if c.processed == 0 { continue }
        duration := int(time.Since(c.startTime).Milliseconds())
        details, _ := json.Marshal(map[string]interface{}{
            "topic":     topic,
            "processed": c.processed,
            "success":   c.success,
            "failed":    c.failed,
            "duration_ms": duration,
        })
        now := time.Now()
        ebl.db.Create(&model.ActivityLog{
            Operation: "kafka-consume-batch", TargetTable: extractTable(topic),
            Status: "success", RowsAffected: c.processed,
            Details: details, TriggeredBy: "kafka",
            StartedAt: c.startTime, CompletedAt: &now,
        })
        // Reset counter
        ebl.counters[topic] = nil
    }
}
```

---

## 3. E2E Latency — Percentiles

### Prometheus
```go
E2ELatency = promauto.NewHistogram(prometheus.HistogramOpts{
    Name:    "cdc_e2e_latency_seconds",
    Help:    "E2E latency: Kafka message timestamp → Postgres insert",
    Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60},
})
```

### System Health API — compute percentiles from histogram
```go
func (h *SystemHealthHandler) getLatencyPercentiles(ctx context.Context) map[string]interface{} {
    // Option A: query Prometheus API
    // histogram_quantile(0.5, rate(cdc_e2e_latency_seconds_bucket[5m]))
    
    // Option B: from activity_log (simpler, less accurate)
    var p50, p95, p99 float64
    h.db.Raw(`SELECT 
        percentile_cont(0.5) WITHIN GROUP (ORDER BY duration_ms) as p50,
        percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95,
        percentile_cont(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99
    FROM cdc_activity_log 
    WHERE operation = 'kafka-consume-batch' AND started_at > NOW() - INTERVAL '30 minutes'`).
    Scan(&struct{ P50, P95, P99 *float64 }{&p50, &p95, &p99})
    
    return map[string]interface{}{"p50_ms": p50, "p95_ms": p95, "p99_ms": p99}
}
```

---

## 4. OTel Zap Bridge (Log Persistence)

### Dependencies
```
go.opentelemetry.io/contrib/bridges/otelzap
go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc
go.opentelemetry.io/otel/sdk/log
```

### Init in main.go
```go
// After OTel trace/metric init:
logExporter, _ := otlploggrpc.New(ctx, otlploggrpc.WithEndpoint(cfg.Otel.GrpcEndpoint))
logProvider := log.NewLoggerProvider(log.WithProcessor(log.NewBatchProcessor(logExporter)))

// Bridge zap → OTel
otelCore := otelzap.NewCore("cdc-worker", otelzap.WithLoggerProvider(logProvider))
logger = zap.New(zapcore.NewTee(
    zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.InfoLevel),  // stdout
    otelCore,  // → SigNoz
))
```

### Trace context in Kafka consumer
```go
func (kc *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) error {
    // Create span for this message
    ctx, span := observability.StartSpan(ctx, "kafka-consume",
        attribute.String("topic", msg.Topic),
        attribute.Int64("offset", msg.Offset),
    )
    defer span.End()
    
    // All logs within this context will have trace_id
    kc.logger.Info("kafka CDC event", zap.String("trace_id", span.SpanContext().TraceID().String()), ...)
}
```

---

## 5. FE SystemHealth Layout

```
┌─ System Health Dashboard ──────────────────────────────────┐
│                                                            │
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐   │
│ │Kafka │ │NATS  │ │PG    │ │Mongo │ │Redis │ │Airbyte│   │
│ │✅ UP │ │✅ UP │ │✅ UP │ │✅ UP │ │✅ UP │ │✅ UP │   │
│ │3 top │ │3 str │ │8 tbl │ │2 db  │ │conn  │ │1 conn│   │
│ │lag:0 │ │1 con │ │1828r │ │      │ │      │ │6 str │   │
│ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘   │
│                                                            │
│ ┌── CDC Pipeline ──────────────────────────────────────┐   │
│ │ Debezium: ✅ RUNNING  │  Consumer Lag: 0  │  5 msg/s │   │
│ └──────────────────────────────────────────────────────┘   │
│                                                            │
│ ┌── Reconciliation ───────────────────────────────────┐    │
│ │ Table           │ Source │ Dest │ Drift% │ Status   │    │
│ │ export_jobs     │  115   │ 113  │ 1.7%   │ ⚠ Drift │    │
│ │ refund_requests │ 1714   │ 1713 │ 0.06%  │ ⚠ Drift │    │
│ └─────────────────────────────────────────────────────┘    │
│                                                            │
│ ┌── E2E Latency ──────┐  ┌── Pipeline Stats ──────────┐   │
│ │ [Line chart P50/P95] │  │ Events 24h: 150            │   │
│ │ P50: 800ms           │  │ Failed 24h: 2              │   │
│ │ P95: 2000ms          │  │ Throughput:  0.5/sec        │   │
│ │ P99: 5000ms          │  │                             │   │
│ └──────────────────────┘  └─────────────────────────────┘  │
│                                                            │
│ ⚠ Warning: 2 tables have data drift                       │
│                                                            │
│ ┌── Recent Events ────────────────────────────────────┐    │
│ │ 12:00  kafka-consume-batch  export_jobs  5 msgs     │    │
│ │ 12:01  cmd-bridge-airbyte   refund_req   1712 rows  │    │
│ │ 12:05  recon-check tier1    *            ok          │    │
│ └─────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────┘
```
