# Gap Analysis: Observability — Mọi tính năng phải có nơi check

> Date: 2026-04-16
> Role: Brain
> Inventory: 105 items (54 API, 13 NATS, 5 Schedule, 4 Background, 9 FE pages)

---

## 1. Tổng quan hệ thống kiểm tra hiện tại

### Đã có
| Nơi check | Covers | Status |
|:----------|:-------|:-------|
| **Activity Log** (`cdc_activity_log`) | APIs, scheduled jobs, manual operations | ✅ Central checkpoint |
| **Worker Log** (stdout/zap) | Kafka consumer, NATS events, errors | ✅ Nhưng chỉ stdout, không persist |
| **Prometheus Metrics** (`/metrics`) | Events processed, sync success/failed, schema drift | ✅ Nhưng chưa có dashboard |
| **Redpanda Console** (`localhost:18088`) | Kafka topics, consumer lag, messages | ✅ |
| **CMS UI Pages** | 9 pages cover registry, mappings, activity, schedule, data integrity | ✅ |
| **NATS Monitoring** (`localhost:18222`) | Streams, connections, subscriptions | ✅ Nhưng CLI only |
| **cdc_reconciliation_report** | Recon tier 1/2/3 results | ✅ Mới tạo |
| **failed_sync_logs** | DLQ — records lỗi | ✅ Mới tạo |

### Thiếu / Gaps

| Gap ID | Mô tả | Impact | Priority |
|:-------|:-------|:-------|:---------|
| G1 | **Worker log không persist** — chỉ stdout, mất khi restart | Không trace lỗi quá khứ | High |
| G2 | **Kafka consumer events không log vào Activity Log** — chỉ log stdout | Không biết bao nhiêu events processed qua CMS | High |
| G3 | **NATS command results không log đầy đủ** — một số commands chỉ log result qua `publishResult`, không ghi Activity Log | Không track bridge/transform result cụ thể (rows affected, duration) | Medium |
| G4 | **Prometheus metrics không có dashboard** — metrics expose nhưng không ai xem | Metrics vô dụng nếu không visualize | Medium |
| G5 | **Debezium health không monitor** — container có thể die mà không biết | Data stop flowing, phát hiện muộn | High |
| G6 | **Kafka Connect health không monitor** — connector có thể FAILED | Debezium stop capture, không alert | High |
| G7 | **MongoDB Oplog size không monitor** — có thể overflow khi bulk update | Debezium mất offset, buộc re-snapshot | Medium |
| G8 | **End-to-end latency không đo** — không biết từ MongoDB insert đến Postgres mất bao lâu | Không biết system performance | Medium |
| G9 | **Schema change events không track** — field mới appear/disappear không có timeline | Khó debug schema drift | Low |
| G10 | **CMS Health page chưa có** — không có 1 page tổng quan health toàn hệ thống | User phải check từng page riêng | High |

---

## 2. Phân tích theo luồng (Workflow)

### Luồng 1: MongoDB → Debezium → Kafka → Worker → Postgres

| Bước | Nơi check hiện tại | Gap |
|:-----|:-------------------|:----|
| MongoDB insert/update | Không | **G11**: Không track source changes |
| Debezium capture | Debezium container log | **G5**: Không monitor health |
| Kafka topic | Redpanda Console | ✅ |
| Kafka consumer lag | Redpanda Console | ✅ Nhưng không alert |
| Worker consume | Worker stdout log | **G2**: Không log Activity |
| Worker parse Avro | Worker stdout log | **G2** |
| Worker DynamicMapper | Worker stdout log | **G2** |
| Worker BatchBuffer upsert | Worker stdout + failed_sync_logs | ✅ DLQ có |
| Postgres data | CMS Data Integrity page | ✅ |

### Luồng 2: Airbyte sync → Postgres

| Bước | Nơi check | Gap |
|:-----|:----------|:----|
| Airbyte cron trigger | Airbyte UI | ✅ |
| Airbyte sync result | Airbyte UI + CMS sync status | ✅ |
| Data ở Postgres | CMS Registry transform status | ✅ |

### Luồng 3: Reconciliation

| Bước | Nơi check | Gap |
|:-----|:----------|:----|
| Schedule trigger | Activity Log | ✅ |
| Source Agent query | Activity Log | Cần verify |
| Dest Agent query | Activity Log | Cần verify |
| Compare result | cdc_reconciliation_report | ✅ |
| Heal action | Activity Log + report | ✅ |

### Luồng 4: Schema Change → Approve → ALTER TABLE

| Bước | Nơi check | Gap |
|:-----|:----------|:----|
| Schema drift detect | SchemaInspector log | ✅ Nhưng chỉ stdout |
| Pending field created | CMS Mapping Approval page | ✅ |
| User approve | CMS UI + Activity Log | ✅ |
| ALTER TABLE executed | schema_changes_log | ✅ |
| Worker reload cache | Worker stdout + NATS | ✅ |

### Luồng 5: Failed record → Retry

| Bước | Nơi check | Gap |
|:-----|:----------|:----|
| Record fail | failed_sync_logs | ✅ |
| User see in CMS | Data Integrity page tab 2 | ✅ |
| User click Retry | API + Activity Log | ✅ |
| Retry result | failed_sync_logs status update | ✅ |

---

## 3. Giải pháp cho từng Gap

| Gap | Giải pháp | Effort |
|:----|:----------|:-------|
| G1 | Worker log → file rotation hoặc gửi qua OTel Logs exporter → SigNoz | Medium (OTel đã init) |
| G2 | Kafka consumer: log processed events vào Activity Log (batch, mỗi 100 events 1 entry) | Small |
| G3 | Command handler: mọi publishResult cũng ghi Activity Log | Small |
| G4 | Grafana dashboard import + Prometheus datasource (hoặc SigNoz metrics) | Medium |
| G5 | Health check endpoint cho Debezium + CMS poll | Small |
| G6 | Kafka Connect status API → CMS poll + alert | Small |
| G7 | MongoDB Oplog monitor script (cron) → alert khi > 80% capacity | Small |
| G8 | OTel traces: span from Kafka consume → Postgres insert → measure latency | Small (OTel đã có) |
| G9 | Schema change timeline → CMS page (đã có schema_changes_log) | Already done |
| G10 | **System Health page** — 1 page tổng quan: Worker, Debezium, Kafka, NATS, Postgres, Airbyte status | Medium |

---

## 4. Đề xuất: System Health Page (G10)

### CMS FE: `/system-health`

```
┌─────────────────────────────────────────────────────────┐
│  System Health Dashboard                                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐     │
│  │ Worker  │ │ Kafka   │ │Debezium │ │ Postgres│     │
│  │ ✅ UP   │ │ ✅ UP   │ │ ✅ UP   │ │ ✅ UP   │     │
│  │ 10 pool │ │ 3 topic │ │ 1 conn  │ │ 8 table │     │
│  │ lag: 0  │ │ lag: 0  │ │ running │ │ 115 rows│     │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘     │
│                                                         │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐                  │
│  │ NATS    │ │ Redis   │ │ Airbyte │                  │
│  │ ✅ UP   │ │ ✅ UP   │ │ ✅ UP   │                  │
│  │ 3 stream│ │ connected│ │ 1 conn  │                  │
│  └─────────┘ └─────────┘ └─────────┘                  │
│                                                         │
│  Pipeline Throughput: 5,640 events/sec (last 5 min)     │
│  E2E Latency: ~2s (MongoDB → Postgres)                  │
│  Failed Sync Logs: 0 (last 24h)                         │
│  Reconciliation: All tables matched ✅                   │
├─────────────────────────────────────────────────────────┤
│  Recent Events (last 10):                                │
│  12:00 bridge export_jobs → 3 rows                      │
│  12:01 kafka CDC event op=c export_jobs                 │
│  12:05 recon-check tier1 → ok                           │
└─────────────────────────────────────────────────────────┘
```

### API needed: `GET /api/system/health`
```json
{
  "worker": {"status": "up", "pool_size": 10, "kafka_lag": 0},
  "kafka": {"status": "up", "topics": 3, "total_lag": 0},
  "debezium": {"status": "running", "connectors": 1},
  "postgres": {"status": "up", "tables": 8, "total_rows": 1828},
  "nats": {"status": "up", "streams": 3},
  "redis": {"status": "up"},
  "airbyte": {"status": "up", "connections": 1},
  "pipeline": {
    "throughput_5m": 0,
    "e2e_latency_ms": 2000,
    "failed_24h": 0,
    "recon_status": "ok"
  }
}
```

---

## 5. Tasks

- [ ] G2: Kafka consumer log → Activity Log (batch per 100 events)
- [ ] G3: Command handler publishResult → Activity Log
- [ ] G5: Debezium health check (poll Kafka Connect API)
- [ ] G6: Kafka Connect status → CMS poll
- [ ] G8: OTel trace span for E2E latency measurement
- [ ] G10: System Health API + FE page
