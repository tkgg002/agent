# Requirements: Data Integrity — Worker Downtime + Reconciliation

> Date: 2026-04-16
> Phase: data_integrity
> Priority: P0

## Câu hỏi 1: CDC Worker die → miss data như thế nào?

### Phân tích luồng khi Worker die

```
MongoDB insert/update → Debezium capture → Kafka topic (persistent)
                                                ↓
                                    Worker die ← messages tích lũy trong Kafka
                                                ↓
                                    Worker restart → consumer group resume từ last committed offset
                                                ↓
                                    Messages SAU offset cuối = được consume
                                    Messages TRƯỚC offset cuối đã committed = KHÔNG re-consume (đã ack)
```

### Case 1: Worker die GIỮA batch
- Worker đang process batch 5 messages, commit offset cho message 1-3
- Worker crash trước khi commit message 4-5
- Restart → resume từ offset 4 → message 4-5 được re-consume ✅
- Message 1-3 nếu INSERT fail (trước fix SchemaAdapter) → data MISS ❌

### Case 2: Worker die SAU commit
- Worker commit offset → die
- Kafka giữ messages, offset đã committed
- Restart → resume từ committed offset → messages mới consume ✅
- Messages cũ (đã committed nhưng INSERT fail) → MISS ❌

### Case 3: Worker die, data nguồn tiếp tục thay đổi
- MongoDB insert/update vẫn chạy
- Debezium capture → Kafka topic (persistent, retention 7 ngày)
- Worker restart → consume TẤT CẢ messages tích lũy → OK ✅
- **KHÔNG MISS** nếu downtime < Kafka retention (7 ngày)

### Kết luận
- **Kafka đảm bảo không mất messages** (persistent, retention 7 ngày)
- **Nhưng**: messages đã committed offset mà INSERT fail → data MISS
- **Cần**: Reconciliation để detect + heal gaps

## Câu hỏi 2: Reconciliation — toàn vẹn dữ liệu

### Yêu cầu

#### R1: Dashboard báo cáo tổng quan data integrity
- Per table: source count vs destination count
- Per table: last sync time, lag
- Highlight tables có chênh lệch

#### R2: Reconciliation check tự động
- So sánh row count: MongoDB source vs Postgres destination
- So sánh sample records: random 1% records, compare hash
- Detect: missing records, stale records, corrupted records

#### R3: Auto-heal mechanism
- Missing records: trigger Debezium re-snapshot cho table cụ thể
- Stale records: re-consume từ Kafka (reset offset per topic)
- Corrupted records: flag + alert, manual review

#### R4: CMS UI page
- Data Integrity Dashboard
- Per-table comparison report
- Action buttons: Check Now, Re-sync, Reset Offset

### Giải pháp Reconciliation

#### Approach 1: Count-based (fast, approximate)
```
Source (MongoDB):  db.collection.countDocuments()
Dest (Postgres):   SELECT COUNT(*) FROM table WHERE _source = 'debezium'
Diff > 0 → ALERT
```

#### Approach 2: Hash-based (accurate, slower)
```
Source: db.collection.aggregate([{$group: {_id: null, hash: {$sum: "$_id"}}}])
Dest:   SELECT COUNT(*), md5(string_agg(_id, '')) FROM table
Compare hashes → mismatch = drift
```

#### Approach 3: ID-based (precise, find exact missing records)
```
Source IDs: db.collection.distinct("_id")
Dest IDs:   SELECT DISTINCT _id FROM table
Diff = missing records → re-fetch from source
```

### Recommended: Tiered approach
| Tier | Method | Frequency | Tables |
|:-----|:-------|:----------|:-------|
| Quick | Count comparison | 5 min | All active |
| Standard | ID set comparison | 1 hour | Active + debezium |
| Deep | Sample hash comparison | 24 hour | All |
