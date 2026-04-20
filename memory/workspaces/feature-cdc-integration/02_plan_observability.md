# Plan: Observability — Full System Monitoring

> Date: 2026-04-16
> Phase: observability
> Ref: 01_requirements_observability.md + 10_gap_analysis_observability.md

---

## 1. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  CMS FE: /system-health                                     │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          │
│  │ Worker  │ │ Kafka   │ │Debezium │ │ Postgres│ ...       │
│  │ status  │ │ topics  │ │ connectors│ │ tables │           │
│  │ lag     │ │ lag     │ │ status  │ │ rows   │           │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘          │
│  Pipeline: throughput | latency | failed | recon            │
│  Recent events (Activity Log last 10)                       │
└─────────────────────┬───────────────────────────────────────┘
                      │ GET /api/system/health
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  CMS API: SystemHealthHandler                                │
│  Aggregates from:                                            │
│  - Worker internal API (/api/v1/internal/stats)              │
│  - Kafka Connect API (localhost:18083/connectors/status)     │
│  - Postgres direct query (table counts)                      │
│  - NATS monitoring (localhost:18222/jsz)                     │
│  - Redis ping                                                │
│  - Airbyte API (connections list)                            │
│  - cdc_activity_log (recent events)                          │
│  - cdc_reconciliation_report (latest status)                 │
│  - failed_sync_logs (count last 24h)                         │
│  - Kafka consumer groups (lag via Kafka admin)               │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Tasks

### Phase 1: System Health API + FE
- [ ] T1: `GET /api/system/health` — aggregate all component status
- [ ] T2: FE `/system-health` page — cards + pipeline metrics + recent events

### Phase 2: Activity Log Enhancement
- [ ] T3: Kafka consumer → batch Activity Log (per 100 events hoặc per 30 giây)
- [ ] T4: Command handler publishResult → Activity Log
- [ ] T5: E2E latency tracking (Kafka timestamp vs Postgres insert time)

### Phase 3: External Health Checks
- [ ] T6: Poll Kafka Connect API → connector status
- [ ] T7: Poll Debezium health (via Kafka Connect connector status)
- [ ] T8: NATS stream/consumer info (via HTTP /jsz)

### Phase 4: Alerting Foundation
- [ ] T9: Health check thresholds: Kafka lag > 1000 = warning, > 10000 = critical
- [ ] T10: Debezium connector FAILED = critical alert
- [ ] T11: failed_sync_logs count > 0 last hour = warning

---

## 3. System Health API Design

### `GET /api/system/health`

```json
{
  "timestamp": "2026-04-16T16:00:00Z",
  "overall": "healthy",  // healthy | degraded | critical

  "components": {
    "worker": {
      "status": "up",
      "pool_size": 10,
      "uptime_seconds": 3600
    },
    "kafka": {
      "status": "up",
      "broker": "localhost:19092",
      "topics": [
        {"name": "cdc.goopay.centralized-export-service.export-jobs", "partitions": 3, "lag": 0},
        {"name": "cdc.goopay.payment-bill-service.refund-requests", "partitions": 3, "lag": 0}
      ],
      "total_lag": 0
    },
    "debezium": {
      "status": "running",  // running | failed | paused | unknown
      "connector": "goopay-mongodb-cdc",
      "tasks": [{"id": 0, "state": "RUNNING"}]
    },
    "nats": {
      "status": "up",
      "streams": 3,
      "consumers": 1,
      "messages": 0
    },
    "postgres": {
      "status": "up",
      "tables_registered": 8,
      "tables_created": 6
    },
    "redis": {
      "status": "up"
    },
    "airbyte": {
      "status": "up",
      "connections": 1,
      "active_streams": 6
    }
  },

  "pipeline": {
    "events_processed_24h": 150,
    "events_failed_24h": 2,
    "avg_latency_ms": 2000,
    "throughput_per_sec": 0.5
  },

  "reconciliation": {
    "tables_matched": 6,
    "tables_drifted": 2,
    "last_check": "2026-04-16T15:55:00Z"
  },

  "alerts": [
    {"level": "warning", "message": "2 tables have data drift", "component": "reconciliation"},
    {"level": "info", "message": "SigNoz not connected (OTel metrics fail)", "component": "otel"}
  ]
}
```

### Implementation Sources

| Data | Source | Method |
|:-----|:-------|:-------|
| Worker status | Worker `/health` endpoint | HTTP GET |
| Kafka topics + lag | Kafka admin API (via Go kafka client) | Direct query |
| Debezium status | Kafka Connect REST (`localhost:18083/connectors/goopay-mongodb-cdc/status`) | HTTP GET |
| NATS info | NATS monitoring (`localhost:18222/jsz`) | HTTP GET |
| Postgres tables | `cdc_table_registry` + `information_schema` | DB query |
| Redis | Redis PING | Direct |
| Airbyte | Airbyte API (ListConnections) | HTTP GET |
| Pipeline events | `cdc_activity_log` last 24h aggregation | DB query |
| Pipeline failed | `failed_sync_logs` count last 24h | DB query |
| Recon status | `cdc_reconciliation_report` latest per table | DB query |
| E2E latency | `cdc_activity_log` avg duration for kafka CDC events | DB query |

---

## 4. FE System Health Page Design

### Route: `/system-health`

**Section 1: Component Cards (top)**
- 7 cards: Worker, Kafka, Debezium, NATS, Postgres, Redis, Airbyte
- Each: icon + status badge (UP/DOWN/DEGRADED) + key metric
- Color: green=up, red=down, yellow=degraded

**Section 2: Pipeline Metrics (middle)**
- 4 stat cards: Events 24h, Failed 24h, Avg Latency, Throughput/sec
- Recon summary: N matched / N drifted

**Section 3: Alerts (if any)**
- Warning/Critical banners
- Component + message

**Section 4: Recent Activity (bottom)**
- Last 10 Activity Log entries (auto-refresh)
- Compact table: time, operation, table, status

**Auto-refresh**: Poll `/api/system/health` every 30 seconds

---

## 5. Kafka Consumer → Activity Log

### Current: chỉ log stdout
```go
kc.logger.Info("kafka CDC event", ...)
```

### After: batch log Activity Log
```go
// Accumulate counter per table
var eventCounter sync.Map  // table → count

// Every 30s or 100 events: flush to Activity Log
func (kc *KafkaConsumer) flushEventLog() {
    kc.eventCounter.Range(func(key, val interface{}) bool {
        table := key.(string)
        count := val.(*int64)
        activityLogger.Quick("kafka-consume", table, "kafka", "success", atomic.LoadInt64(count), nil, "")
        atomic.StoreInt64(count, 0)
        return true
    })
}
```

---

## 6. Command Handler → Activity Log

### Current: `publishResult` logs via NATS reply + zap
### After: ALSO ghi Activity Log

```go
func (h *CommandHandler) publishResult(msg *nats.Msg, result CommandResult) {
    // Existing: NATS reply + zap log
    // NEW: Activity Log
    h.activityLogger.Quick(result.Command, result.TargetTable, "nats-command",
        result.Status, int64(result.RowsAffected), nil, result.Error)
}
```
Cần inject `activityLogger` vào CommandHandler.

---

## 7. E2E Latency

### Measure
```go
// In Kafka consumer processMessage:
kafkaTimestamp := msg.Time  // Kafka message timestamp
// After successful upsert:
e2eLatency := time.Since(kafkaTimestamp)
metrics.E2ELatency.Observe(e2eLatency.Seconds())
```

### Prometheus histogram
```go
E2ELatency = promauto.NewHistogram(prometheus.HistogramOpts{
    Name:    "cdc_e2e_latency_seconds",
    Help:    "End-to-end latency from Kafka message to Postgres insert",
    Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
})
```

---

## 8. Files

### CMS
| File | New/Edit | Purpose |
|:-----|:---------|:--------|
| `internal/api/system_health_handler.go` | New | Aggregate health from all components |
| `internal/router/router.go` | Edit | Register route |
| `internal/server/server.go` | Edit | Init handler |

### FE
| File | New/Edit | Purpose |
|:-----|:---------|:--------|
| `src/pages/SystemHealth.tsx` | New | Health dashboard |
| `src/App.tsx` | Edit | Route + menu |

### Worker
| File | Edit | Purpose |
|:-----|:-----|:--------|
| `internal/handler/kafka_consumer.go` | Edit | Batch Activity Log |
| `internal/handler/command_handler.go` | Edit | publishResult → Activity Log |
| `pkgs/metrics/prometheus.go` | Edit | E2E latency histogram |

---

## 9. Definition of Done

- [ ] `/system-health` page hiện status TẤT CẢ 7 components
- [ ] Debezium connector status hiện trên health page
- [ ] Kafka consumer lag hiện per topic
- [ ] Pipeline metrics: events 24h, failed 24h, latency, throughput
- [ ] Recon status: matched/drifted per table
- [ ] Alerts: warning/critical banners khi component down
- [ ] Kafka consumer events log Activity Log (batch)
- [ ] Command handler results log Activity Log
- [ ] Auto-refresh 30 giây
