# SLO Definition — CDC Integration

> **Date**: 2026-04-17
> **Author**: Brain (claude-opus-4-7)
> **Purpose**: Định nghĩa Service Level Objectives cho hệ thống CDC, dùng làm gốc derive alert thresholds (thay vì đặt số tùy tiện như v2).
> **Reference**: `02_plan_observability_v3.md` §11.

---

## 1. Service definition

**Service**: CDC Data Integration Pipeline — MongoDB source → Debezium → Kafka → Worker → Postgres dest + sync monitoring.

**User**: Downstream services consume Postgres data (BI, analytics, fintech apps). SLA commit với downstream là "near-realtime + no data loss".

**Critical path**:
1. Mongo write → Debezium capture → Kafka event → Worker upsert → PG ready for read.
2. Reconciliation detect + heal drift.
3. Observability + alerting cho ops team.

---

## 2. SLO Catalog

| ID | SLO | Indicator (SLI) | Target | Window | Error Budget (30d) |
|:---|:----|:----------------|:-------|:-------|:-------------------|
| **SLO-1** | E2E Latency P99 | `histogram_quantile(0.99, rate(cdc_e2e_latency_seconds_bucket[5m]))` | ≤ 5s trong ≥ 99% 5-min windows | 30 ngày rolling | 86 windows (7.2h) |
| **SLO-2** | Reconciliation Drift | `cdc_recon_mismatch_count == 0` ở tier 1 | ≥ 99.9% daily checks | 30 ngày | 0.2 ngày |
| **SLO-3** | Worker Availability | `up{job="cdc-worker"} == 1` | ≥ 99.95% | 30 ngày | 21.6 phút |
| **SLO-4** | Zero Data Loss | Rows `failed_sync_logs.status='dead_letter'` AND `created_at > now()-24h` == 0 | = 0 records stuck > 24h | Rolling | 0 records |
| **SLO-5** | Kafka Retention Safety | `kafka_consumergroup_lag_seconds < 14d × 0.9` | 100% luôn luôn | Rolling | 0 events lost |
| **SLO-6** | DLQ Write Success Rate | `1 - (rate(cdc_dlq_write_failures_total[5m]) / rate(cdc_events_failed_total[5m]))` | ≥ 99.99% | 30 ngày | 4.3 events / 30d |
| **SLO-7** | System Health API Availability | `probe_success{target="/api/system/health"}` | ≥ 99.9% | 30 ngày | 43 phút |

---

## 3. Rationale — tại sao chọn targets này

### SLO-1: P99 5s
- Downstream commits near-realtime (sub-minute). 5s P99 cho phép spike ngắn mà user không phát hiện.
- P99 thay vì P95 vì fintech cần protect edge case (outlier = risk).
- Phase 0 baseline P50 = 152ms → có room cho P99 lên 5s.

### SLO-2: Recon drift 99.9% ~ 0 daily
- Fintech = **zero tolerance** với lệch số dư / transaction. Drift = incident, không phải metric để optimize.
- 0.2 day budget = ~1 ngày/5 tháng — cho phép 1 incident "detection lag" mỗi năm.

### SLO-3: Worker 99.95%
- Worker single-instance hiện tại. HA multi-instance = Phase 2+ future.
- 21.6 min/month = 2 maintenance windows 10 phút.

### SLO-4: Zero dead-letter > 24h
- Data loss = unacceptable. Dead-letter stuck = ops failure.
- 24h window cho ops response.

### SLO-5: Kafka retention 14d × 90%
- Retention = 14d. Lag > 12.6d = critical (event sắp bị xoá).
- 90% cap cho buffer restart/maintenance.

### SLO-6: DLQ write 99.99%
- DLQ là safety net. Nếu DLQ ghi fail → ACK bị block → Kafka redeliver → loop.
- 99.99% = strict vì đây là gate bảo vệ data loss.

### SLO-7: Health API 99.9%
- Ops dashboard cần uptime cao để debug production issues.
- 99.9% tiêu chuẩn cho internal tooling.

---

## 4. Derived Alert Rules (Prometheus)

### SLO-1: E2E Latency
```yaml
- alert: E2ELatencyP99Warning
  expr: |
    histogram_quantile(0.99, 
      sum by (le) (rate(cdc_e2e_latency_seconds_bucket[5m]))
    ) > 5
  for: 5m
  labels: { severity: warning, slo: "1" }
  annotations:
    summary: "E2E P99 latency > 5s"
    runbook: "https://wiki.internal/cdc/runbook#slo-1"

- alert: E2ELatencyP99Critical
  expr: |
    histogram_quantile(0.99, 
      sum by (le) (rate(cdc_e2e_latency_seconds_bucket[5m]))
    ) > 5
  for: 15m
  labels: { severity: critical, slo: "1" }
```

### SLO-2: Reconciliation Drift
```yaml
- alert: ReconDriftDetected
  expr: cdc_recon_mismatch_count > 0
  for: 1h
  labels: { severity: warning, slo: "2" }

- alert: ReconDriftPersistent
  expr: cdc_recon_mismatch_count > 0
  for: 6h
  labels: { severity: critical, slo: "2" }

- alert: ReconStale
  expr: (time() - cdc_recon_last_success_timestamp) > 7200
  for: 5m
  labels: { severity: warning, slo: "2" }
  annotations:
    summary: "Recon không chạy thành công > 2h"
```

### SLO-3: Worker Availability
```yaml
- alert: WorkerDown
  expr: up{job="cdc-worker"} == 0
  for: 1m
  labels: { severity: critical, slo: "3" }
```

### SLO-4: Dead Letter
```yaml
- alert: DLQDeadLetterStuck
  expr: cdc_dlq_stuck_records_total{status="dead_letter"} > 0
  for: 24h
  labels: { severity: critical, slo: "4" }
  annotations:
    summary: "Có record DLQ dead_letter > 24h — ops cần xử lý manual"
```

### SLO-5: Kafka Retention
```yaml
- alert: KafkaConsumerLagWarning
  expr: kafka_consumergroup_lag_seconds > 14 * 86400 * 0.7
  for: 5m
  labels: { severity: warning, slo: "5" }

- alert: KafkaConsumerLagCritical
  expr: kafka_consumergroup_lag_seconds > 14 * 86400 * 0.9
  for: 5m
  labels: { severity: critical, slo: "5" }
  annotations:
    summary: "Consumer lag sắp hết retention window — event sắp bị xoá"
```

### SLO-6: DLQ Write
```yaml
- alert: DLQWriteFailureHigh
  expr: |
    rate(cdc_dlq_write_failures_total[5m]) 
    / 
    rate(cdc_events_failed_total[5m]) > 0.0001
  for: 10m
  labels: { severity: critical, slo: "6" }
```

### SLO-7: Health API
```yaml
- alert: HealthAPIDown
  expr: probe_success{target="/api/system/health"} == 0
  for: 5m
  labels: { severity: warning, slo: "7" }
```

---

## 5. Error Budget Policy

### Khi nào cut feature / focus reliability
- **Burn rate 14× normal** (1h budget burn = 14h normal) → **freeze non-critical deploys** cho đến khi recover.
- **Burn rate 6× normal** → notify team, investigate.
- **Burn rate 1× normal** → normal ops.

### Calculation example
- SLO-1 budget 30d: 86 windows × 5 min = 430 min allowed.
- 1h burn 14× = 60 min × 14 = 840 min → exceeds 1-month budget → STOP.

### Process
1. SRE dashboard hiển thị burn rate real-time.
2. Alert burn-rate > 14× → page oncall + post-mortem required.
3. Weekly review SLO compliance.

---

## 6. SLO Dashboard Panels

### Panel 1: SLO compliance (30d rolling)
```promql
# Compliance % SLO-1
1 - (sum(rate(slo_latency_violation_total[30d])) / sum(rate(slo_latency_total[30d])))
```

### Panel 2: Error budget remaining
```promql
(1 - (0.99 - current_compliance)) × 100
# Nếu < 20% → alert
```

### Panel 3: Burn rate
```promql
# Burn rate 1h
(1 - compliance_1h) / (1 - 0.99) 
# > 14 = cut
```

---

## 7. Review cadence

- **Weekly**: SRE review burn rates, adjust alert thresholds nếu false positive > 10%.
- **Monthly**: product + SRE review SLO met → negotiate targets nếu quá strict hoặc quá loose.
- **Quarterly**: audit process incidents, update runbooks.

---

## 8. Non-SLO metrics (informational only)

KHÔNG dùng làm alert nhưng track để context:
- Events/sec per topic (capacity planning)
- Mongo secondary lag (Recon accuracy)
- PG replica lag (Recon accuracy)
- FE p95 latency (UX)
- Memory/CPU Worker (capacity)

---

## 9. TODO — wire vào code

- [ ] Prom scrape target `cdc-worker:9090/metrics` (DevOps).
- [ ] Alert rules YAML apply qua Prometheus reload.
- [ ] SigNoz dashboard import rules (SigNoz hỗ trợ PromQL).
- [ ] Blackbox exporter cho SLO-7 (`/api/system/health` probe).
- [ ] Recording rules precompute compliance % để dashboard nhanh.
- [ ] Runbook page mỗi SLO (internal wiki).
