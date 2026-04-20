# Plan: Observability — FINAL (Merged)

> Date: 2026-04-16
> Phase: observability
> Sources:
>   - `01_requirements_observability.md` (base)
>   - `01_requirements_observability_solution.md` (user deep requirements)

---

## 1. Mô hình Observability (Core/Agent — 4 Layers)

```
┌────────────────────────────────────────────────────────────┐
│  Layer 4: PRESENTATION (React FE)                          │
│  /system-health — Single Source of Truth                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Infrastructure Heartbeat (Kafka/NATS/PG/Mongo/Redis) │  │
│  │ CDC Pipeline (Debezium connector + Worker metrics)    │  │
│  │ Recon Status (table, source, dest, drift%, last recon)│  │
│  │ E2E Latency (line chart realtime, P95/P99)            │  │
│  │ Alerts (critical/warning banners + trace lỗi)         │  │
│  │ Recent Events (Activity Log last 10)                   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────┬──────────────────────────────────────┘
                      │ GET /api/system/health
                      ▼
┌────────────────────────────────────────────────────────────┐
│  Layer 3: STORAGE                                          │
│  SigNoz (ClickHouse) ← OTel traces + logs + metrics       │
│  Prometheus ← Worker /metrics endpoint                     │
│  Postgres ← cdc_activity_log, cdc_reconciliation_report,  │
│              failed_sync_logs                               │
└─────────────────────┬──────────────────────────────────────┘
                      │
┌────────────────────────────────────────────────────────────┐
│  Layer 2: CORE (CMS Service — Aggregator)                  │
│  - Poll Worker health API                                   │
│  - Poll Kafka Connect API (connector status + trace lỗi)   │
│  - Poll NATS monitoring (/jsz)                              │
│  - Query Postgres (registry, activity, recon, failed)       │
│  - Tổng hợp → /api/system/health response                  │
│  - Compute alerts (lag threshold, connector failed, drift)  │
└─────────────────────┬──────────────────────────────────────┘
                      │
┌────────────────────────────────────────────────────────────┐
│  Layer 1: AGENT (Worker — Collector)                       │
│  - Zap logger → OTel zap core → gRPC → SigNoz             │
│  - Prometheus metrics (events, latency, sync success/fail)  │
│  - Batch Activity Log (per 100 msgs or 5 giây)             │
│  - E2E Latency: T_postgres - T_kafka → Histogram P95/P99  │
│  - trace_id trong mỗi log → trace ngược Kafka → Postgres  │
└────────────────────────────────────────────────────────────┘
```

---

## 2. R1: System Health Page — Readiness & Performance

### Layout chi tiết

**Section 1: Infrastructure Heartbeat**
| Component | Check | Hiển thị |
|:----------|:------|:---------|
| Kafka | Broker connect + topic list | Status + topic count + total lag |
| NATS | HTTP /jsz | Status + stream count + consumer count |
| Postgres | DB ping + table count | Status + registered tables + total rows |
| MongoDB | Client ping | Status + databases |
| Redis | PING | Status |

**Section 2: CDC Pipeline Status**
| Metric | Source | Hiển thị |
|:-------|:-------|:---------|
| Debezium Connector | Kafka Connect API `/connectors/status` | Running/Paused/Failed + task count + **trace lỗi nếu failed** |
| Consumer Lag | Kafka consumer groups describe | Số msg đang chờ per topic |
| Throughput | Prometheus `cdc_events_processed_total` rate | msg/sec (5 min avg) |

**Section 3: Reconciliation Status (Core View)**
| Column | Source |
|:-------|:-------|
| Table Name | cdc_table_registry |
| Source Count | Latest recon report |
| Dest Count | Latest recon report |
| **Drift (%)** | `(source - dest) / source * 100` |
| Last Recon | checked_at |
| Status | ok/drift badge |

**Section 4: E2E Latency**
- **Line chart realtime** (last 30 min, mỗi điểm = 1 phút avg)
- **P50, P95, P99** hiện bên cạnh chart
- Source: Prometheus histogram `cdc_e2e_latency_seconds`

**Section 5: Alerts**
- Critical (red): Debezium FAILED, Kafka DOWN, Consumer lag > 10000
- Warning (yellow): Debezium PAUSED, Consumer lag > 1000, Drift > 0
- Info (blue): SigNoz not connected, Worker restart detected

**Section 6: Recent Events**
- Activity Log last 10 entries (compact table)
- Auto-refresh 30 giây

---

## 3. R2+R3: Activity Log Enhancement

### Kafka Consumer Batch Log
**Format mỗi entry**:
```json
{
  "operation": "kafka-consume-batch",
  "details": {
    "topic": "cdc.goopay.centralized-export-service.export-jobs",
    "processed": 100,
    "success": 98,
    "failed": 2,
    "failed_link": "/data-integrity?tab=failed&table=export_jobs",
    "duration_ms": 450
  }
}
```
- Ghi mỗi 100 messages HOẶC mỗi 5 giây (whichever first)
- Per topic riêng (không gộp nhiều topics)

### NATS Command Log
**Format**: Mọi `publishResult` ghi Activity Log kèm:
- operation: `cmd-{command_name}`
- target_table
- status + rows_affected
- error message nếu fail
- triggered_by: "nats-command"

---

## 4. R4: Debezium + Kafka Connect Health (Deep)

### Poll logic
```go
// 1. GET /connectors/goopay-mongodb-cdc/status
// 2. Parse response:
status := response.connector.state   // RUNNING | PAUSED | FAILED
tasks := response.tasks               // [{id, state, trace}]

// 3. Nếu FAILED → bóc trace lỗi:
for _, task := range tasks {
    if task.State == "FAILED" {
        trace := task.Trace  // Full Java stack trace
        // Truncate + display trên UI
        alert("critical", "Debezium task FAILED: " + firstLine(trace))
    }
}
```

### UI hiển thị khi FAILED
```
┌─────────────────────────────────────────────┐
│ ❌ Debezium: FAILED                          │
│ Task 0: FAILED                               │
│ Error: org.apache.kafka.connect.errors...    │
│ [Xem chi tiết trace]  [Restart Connector]    │
└─────────────────────────────────────────────┘
```

---

## 5. R5: E2E Latency — Percentiles

### Prometheus Histogram
```go
E2ELatency = promauto.NewHistogram(prometheus.HistogramOpts{
    Name:    "cdc_e2e_latency_seconds",
    Help:    "End-to-end latency: Kafka message → Postgres insert",
    Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60},
})
```

### Tính P95, P99
- Prometheus query: `histogram_quantile(0.95, rate(cdc_e2e_latency_seconds_bucket[5m]))`
- FE: fetch từ Prometheus API hoặc từ system health API (pre-computed)

### Cách đo
```
T1 = msg.Time             (Kafka message timestamp — set by Debezium)
T2 = time.Now()           (sau khi UPSERT thành công)
Latency = T2 - T1
metrics.E2ELatency.Observe(latency.Seconds())
```

---

## 6. R6: Worker Log Persistence — OTel + SigNoz

### Stack
```
Zap Logger (Go Worker)
    ↓ OTel zap core bridge
    ↓ gRPC
SigNoz OTEL Collector (localhost:4317 gRPC / 4318 HTTP)
    ↓
ClickHouse (SigNoz storage)
```

### Trace context
- Mỗi Kafka event → tạo OTel span → span context propagate qua EventHandler → BatchBuffer
- Log entries kèm `trace_id` + `span_id`
- SigNoz: search log by trace_id → xem toàn bộ journey: Kafka consume → parse → map → upsert

### Implementation
```go
// OTel zap core bridge
import "go.opentelemetry.io/contrib/bridges/otelzap"

// In main.go after OTel init:
otelCore := otelzap.NewCore("cdc-worker", otelzap.WithLoggerProvider(loggerProvider))
logger = zap.New(zapcore.NewTee(consoleCore, otelCore))
```

### Config
```yaml
otel:
  enabled: true
  serviceName: cdc-worker
  endpoint: http://localhost:4318    # HTTP (traces + metrics)
  grpcEndpoint: http://localhost:4317  # gRPC (logs)
  sampleRatio: 1.0
```

---

## 7. Tasks (FINAL — 15 tasks)

### Phase 1: System Health API + FE
- [x] T1: CMS `system_health_handler.go` — aggregate all components + pipeline + recon + alerts
- [x] T2: CMS config (workerUrl, kafkaConnectUrl, natsMonitorUrl)
- [x] T3: CMS route + server init
- [x] T4: FE `SystemHealth.tsx` — 6 sections + Debezium trace + Restart button (fixed events_processed_24h gap)
- [x] T5: FE route + menu + auto-refresh 30s

### Phase 2: Activity Log Enhancement
- [x] T6: Kafka consumer batch Activity Log (per 100 msgs or 5s, per topic) ✅ runtime verified
- [x] T7: Command handler publishResult → Activity Log (kèm metadata)

### Phase 3: E2E Latency + Metrics
- [x] T8: Prometheus histogram `cdc_e2e_latency_seconds` (custom buckets)
- [x] T9: Kafka consumer measure T2-T1 → histogram observe ✅ runtime: 7.14s E2E
- [x] T10: System Health API compute P50/P95/P99 from activity_log ✅ runtime: P50=152ms

### Phase 4: Debezium Deep Health
- [x] T11: Poll Kafka Connect API → bóc trace lỗi khi FAILED
- [x] T12: FE: hiện trace lỗi + Restart Connector button + CMS endpoint ✅ runtime: restart 204

### Phase 5: OTel Log Persistence
- [ ] T13: OTel zap core bridge (Go) → logs gửi SigNoz gRPC (deferred: needs SigNoz running stable)
- [ ] T14: trace_id trong mỗi Kafka event span → log context (deferred: depends on T13)

### Phase 6: Verify
- [x] T15: Flow testing — T6/T9/T10/T12 runtime verified

---

## 8. Files (FINAL)

### CMS
| File | New/Edit | Purpose |
|:-----|:---------|:--------|
| `internal/api/system_health_handler.go` | New | Aggregate + alerts + Debezium trace |
| `config/config.go` | Edit | +SystemConfig |
| `config/config-local.yml` | Edit | +system section |
| `internal/router/router.go` | Edit | Route |
| `internal/server/server.go` | Edit | Init handler |

### FE
| File | New/Edit | Purpose |
|:-----|:---------|:--------|
| `src/pages/SystemHealth.tsx` | New | 6-section dashboard + chart + auto-refresh |
| `src/App.tsx` | Edit | Route + menu |

### Worker
| File | Edit | Purpose |
|:-----|:-----|:--------|
| `internal/handler/kafka_consumer.go` | Edit | Batch log + E2E latency measure |
| `internal/handler/command_handler.go` | Edit | publishResult → Activity Log |
| `pkgs/metrics/prometheus.go` | Edit | +E2ELatency histogram |
| `pkgs/observability/otel.go` | Edit | +Log exporter (gRPC to SigNoz) |
| `cmd/worker/main.go` | Edit | OTel zap bridge |
| `config/config.go` | Edit | +grpcEndpoint |

---

## 9. Definition of Done

- [x] `/system-health` hiện 7 components với Readiness + Performance metrics
- [x] Debezium FAILED → trace lỗi hiện trên UI + Restart button, không cần ssh
- [x] Kafka consumer events → Activity Log (batch format: processed/success/failed/duration)
- [x] Command handler results → Activity Log (operation, table, rows, error)
- [x] E2E Latency histogram P50/P95/P99 hiện trên dashboard (runtime: P50=152ms)
- [x] Recon status hiện drift % per table
- [x] Alerts banner: critical (component down, source unreachable), warning (lag/drift)
- [ ] Worker logs persist qua OTel → SigNoz (deferred: T13/T14)
- [x] Auto-refresh 30 giây
- [x] Mọi tính năng có nơi check — không blind spot
