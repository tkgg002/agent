# Flow Testing: Observability

> Date: 2026-04-16
> Phase: observability

---

## Flow 1: System Health — All Components UP

### Steps
1. Tất cả services chạy (Worker, CMS, Kafka, Debezium, NATS, Redis, Postgres, Airbyte)
2. Truy cập `/system-health`
3. Verify 7 cards đều hiện ✅ UP

### Expected
- Worker: UP, pool_size 10
- Kafka: UP, topics 3, lag 0
- Debezium: RUNNING, 1 connector
- NATS: UP, 3 streams
- Postgres: UP, 8 tables
- Redis: UP
- Airbyte: UP, 1 connection

### Verify
```
GET /api/system/health → overall = "healthy"
```

---

## Flow 2: System Health — Component DOWN

### Steps
1. Stop Kafka: `docker stop gpay-kafka`
2. Refresh `/system-health`
3. Kafka card hiện ❌ DOWN
4. overall = "degraded" hoặc "critical"
5. Alert banner hiện warning

### Expected
- Kafka: DOWN
- Debezium: có thể FAILED (lost Kafka connection)
- Overall: critical

### Cleanup
```
docker start gpay-kafka
```

---

## Flow 3: Kafka Consumer Events → Activity Log

### Steps
1. Insert 5 records vào MongoDB
2. Worker consume → process
3. Chờ 30 giây (flush interval)
4. Check Activity Log

### Expected
- Activity Log có entry operation = "kafka-consume-batch"
- Details chứa: `{"export_jobs": 5}`

### Verify
```sql
SELECT operation, target_table, rows_affected, details
FROM cdc_activity_log WHERE operation = 'kafka-consume-batch'
ORDER BY started_at DESC LIMIT 5;
```

---

## Flow 4: Command Handler → Activity Log

### Steps
1. CMS: click "Đồng bộ" cho 1 table
2. Worker nhận NATS command → execute → publishResult
3. Check Activity Log

### Expected
- Activity Log có entry operation = "cmd-bridge-airbyte"
- Status: success/error
- RowsAffected: N

### Verify
```sql
SELECT operation, target_table, status, rows_affected
FROM cdc_activity_log WHERE operation LIKE 'cmd-%'
ORDER BY started_at DESC LIMIT 10;
```

---

## Flow 5: E2E Latency

### Steps
1. Insert record MongoDB
2. Worker consume từ Kafka → upsert Postgres
3. Check Prometheus metric

### Expected
- `cdc_e2e_latency_seconds` histogram có observation
- Latency < 5 giây (normal)

### Verify
```
curl http://localhost:8082/metrics | grep cdc_e2e_latency
```

---

## Flow 6: Auto-refresh

### Steps
1. Mở `/system-health`
2. Chờ 30 giây
3. Page tự update data mới

### Expected
- Recent events section update
- Pipeline metrics update
- Không cần manual refresh

---

## Flow 7: Debezium Connector FAILED

### Steps
1. Stop MongoDB: `docker stop gpay-mongo`
2. Chờ Debezium detect → connector may fail
3. Check `/system-health` → Debezium status

### Expected
- Debezium: FAILED hoặc task state error
- Alert banner critical

### Cleanup
```
docker start gpay-mongo
docker restart gpay-debezium
```

---

## Flow 8: Recon Drift % trên System Health

### Steps
1. Mở `/system-health`
2. Section "Reconciliation" hiện table list
3. Mỗi row: Table, Source Count, Dest Count, **Drift %**, Status

### Expected
- export_jobs: drift = (115-113)/115*100 = 1.7%
- refund_requests: drift = (1714-1713)/1714*100 = 0.06%
- Badge: ⚠ Drift (yellow)

---

## Flow 9: E2E Latency Percentiles

### Steps
1. Insert 10 records MongoDB (spread over 1 min)
2. Worker consume + upsert
3. Mở `/system-health` → section "E2E Latency"

### Expected
- P50: ~800ms (normal)
- P95: ~2000ms
- P99: ~5000ms
- Line chart hiện 30 min history

### Verify
```
curl http://localhost:8082/metrics | grep cdc_e2e_latency_seconds
```

---

## Flow 10: OTel Trace Context (khi SigNoz chạy)

### Steps
1. Insert record MongoDB
2. Worker consume → span created → log kèm trace_id
3. Mở SigNoz → search trace_id

### Expected
- SigNoz hiện trace: Kafka consume → EventHandler → DynamicMapper → BatchBuffer → Postgres
- Mỗi span có duration
- Logs linked to trace

### Verify
- SigNoz Traces tab → search service "cdc-worker"
- Click trace → xem spans + logs

---

## Flow 11: Debezium FAILED — Trace hiện trên UI

### Steps
1. Gây lỗi Debezium (VD: sai MongoDB connection string)
2. Debezium connector FAILED
3. `/system-health` → CDC Pipeline section

### Expected
- Debezium: ❌ FAILED
- Task trace hiện: "org.apache.kafka.connect.errors..."
- Button "Restart Connector"
