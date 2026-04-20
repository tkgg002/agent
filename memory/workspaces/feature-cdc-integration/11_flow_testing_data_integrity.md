# Flow Testing: Data Integrity System

> Date: 2026-04-16
> Phase: data_integrity
> Dùng để test sau khi deploy

---

## Flow 1: Tier 1 — Count Check (tự động mỗi 5 phút)

### Precondition
- Worker chạy, MongoDB + Postgres có data
- cdc_worker_schedule có entry "reconcile" enabled

### Steps
1. Reconciliation schedule trigger (hoặc manual API POST /api/reconciliation/check)
2. Source Agent query MongoDB: `db.collection.countDocuments()`
3. Dest Agent query Postgres: `SELECT COUNT(*) FROM table`
4. Core compare: diff = source - dest

### Expected
- `cdc_reconciliation_report` có record mới
- status = "ok" nếu count match, "drift" nếu lệch
- CMS Dashboard hiện status per table

### Verify
```sql
SELECT target_table, source_count, dest_count, diff, status, checked_at
FROM cdc_reconciliation_report ORDER BY checked_at DESC LIMIT 10;
```

---

## Flow 2: Tier 2 — ID Set Check (on demand)

### Precondition
- Tier 1 detect drift

### Steps
1. API POST /api/reconciliation/check/:table (trigger Tier 2)
2. Source Agent: get ALL IDs từ MongoDB (batch 10K)
3. Dest Agent: get ALL IDs từ Postgres
4. Set diff: source - dest = missing IDs

### Expected
- Report với missing_ids JSONB chứa danh sách IDs thiếu
- missing_count > 0

### Verify
```sql
SELECT target_table, missing_count, missing_ids, status
FROM cdc_reconciliation_report WHERE tier = 2 ORDER BY checked_at DESC LIMIT 5;
```

---

## Flow 3: Tier 3 — Merkle Tree Hash (daily)

### Precondition
- Count match nhưng nghi ngờ data stale

### Steps
1. API POST /api/reconciliation/deep-check/:table
2. Source Agent: chunk 10K records, hash per chunk
3. Dest Agent: same
4. Core compare chunk hashes → find mismatched chunks

### Expected
- stale_count = số chunks lệch
- stale_ids chứa chunk ranges

### Verify
```sql
SELECT target_table, stale_count, stale_ids, status
FROM cdc_reconciliation_report WHERE tier = 3 ORDER BY checked_at DESC LIMIT 5;
```

---

## Flow 4: Heal — Version-aware

### Precondition
- Tier 2 report có missing_ids

### Steps
1. API POST /api/reconciliation/heal/:table
2. Core đọc missing_ids từ latest report
3. Per missing ID:
   a. Fetch full document từ MongoDB
   b. Check Postgres _synced_at (version compare)
   c. MongoDB newer → UPSERT
   d. Postgres newer → SKIP (race condition guard)
4. Audit Log ghi mỗi heal action

### Expected
- healed_count > 0 trong report
- Activity Log có "recon-heal" entries
- Count sau heal: source == dest

### Verify
```sql
-- Check heal result
SELECT target_table, healed_count, healed_at
FROM cdc_reconciliation_report WHERE healed_count > 0 ORDER BY checked_at DESC;

-- Check activity log
SELECT operation, target_table, details FROM cdc_activity_log
WHERE operation = 'recon-heal' ORDER BY started_at DESC LIMIT 10;

-- Re-run Tier 1 → verify match
```

---

## Flow 5: DLQ — Failed Sync Logs

### Precondition
- Worker nhận Kafka message mà INSERT fail

### Steps
1. Message fail (schema mismatch, type error...)
2. BatchBuffer catch error → ghi failed_sync_logs
3. Prometheus counter cdc_sync_failed_total increment

### Expected
- failed_sync_logs có record với error_message + raw_json
- Worker KHÔNG crash
- CMS UI hiện failed records

### Verify
```sql
SELECT target_table, record_id, error_type, error_message, status
FROM failed_sync_logs ORDER BY created_at DESC LIMIT 10;
```

---

## Flow 6: DLQ Retry

### Steps
1. CMS FE: click Retry trên failed record
2. API POST /api/failed-sync-logs/:id/retry
3. Re-attempt upsert
4. Success → status = "resolved"
5. Fail → retry_count++, status stays "failed"

### Verify
```sql
SELECT id, retry_count, status, resolved_at FROM failed_sync_logs WHERE id = ?;
```

---

## Flow 7: Worker Die + Recovery

### Steps
1. Stop Worker (Ctrl+C)
2. Insert records vào MongoDB (5-10 records)
3. Wait 1 phút
4. Start Worker
5. Check: Kafka consumer resume → process messages → Postgres has new records

### Expected
- Consumer lag = 0 sau process
- Tất cả records insert during downtime present in Postgres
- No errors in Worker log

### Verify
```bash
# Check consumer lag
docker exec gpay-kafka kafka-consumer-groups --bootstrap-server localhost:9092 --group cdc-worker-group --describe

# Check Postgres
SELECT COUNT(*) FROM export_jobs WHERE _source = 'debezium';
```

---

## Flow 8: Debezium Signal Snapshot

### Steps
1. CMS: click "Trigger Snapshot" cho table
2. API ghi signal vào MongoDB debezium_signal collection
3. Debezium đọc signal → re-snapshot table
4. Messages re-published vào Kafka
5. Worker consume → upsert Postgres

### Verify
```javascript
// Check signal was written
db.getSiblingDB("payment-bill-service").debezium_signal.find()
```

---

## Flow 9: Kafka Compact — Data Persistence

### Steps
1. Verify cleanup.policy=compact cho CDC topics
2. Stop Worker 1 giờ
3. Insert + update records trong MongoDB
4. Start Worker
5. Verify: TẤT CẢ records (cả insert lẫn update) present

### Verify
```bash
# Check topic config
docker exec gpay-kafka kafka-configs --describe --entity-type topics --entity-name cdc.goopay.centralized-export-service.export-jobs --bootstrap-server localhost:9092
```
