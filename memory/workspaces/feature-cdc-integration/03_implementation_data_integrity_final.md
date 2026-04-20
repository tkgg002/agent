# Implementation: Data Integrity (FINAL)

> Date: 2026-04-16
> Phase: data_integrity
> Ref: 02_plan_data_integrity_final.md

---

## 1. Migration SQL

```sql
-- 008_reconciliation.sql

BEGIN;

CREATE TABLE IF NOT EXISTS cdc_reconciliation_report (
    id BIGSERIAL PRIMARY KEY,
    target_table VARCHAR(200) NOT NULL,
    source_db VARCHAR(200),
    source_count BIGINT DEFAULT 0,
    dest_count BIGINT DEFAULT 0,
    diff BIGINT DEFAULT 0,
    missing_count INT DEFAULT 0,
    missing_ids JSONB,
    stale_count INT DEFAULT 0,
    stale_ids JSONB,
    check_type VARCHAR(20) NOT NULL,     -- count, id_set, merkle_hash
    status VARCHAR(20) NOT NULL,          -- ok, drift, error
    tier INT DEFAULT 1,                   -- 1, 2, 3
    duration_ms INT,
    error_message TEXT,
    checked_at TIMESTAMP DEFAULT NOW(),
    healed_at TIMESTAMP,
    healed_count INT DEFAULT 0
);

CREATE INDEX idx_recon_table ON cdc_reconciliation_report(target_table);
CREATE INDEX idx_recon_status ON cdc_reconciliation_report(status);
CREATE INDEX idx_recon_checked ON cdc_reconciliation_report(checked_at DESC);

CREATE TABLE IF NOT EXISTS failed_sync_logs (
    id BIGSERIAL PRIMARY KEY,
    target_table VARCHAR(200) NOT NULL,
    source_table VARCHAR(200),
    source_db VARCHAR(200),
    record_id VARCHAR(200),
    operation VARCHAR(10),                -- c, u, d
    raw_json JSONB,
    error_message TEXT NOT NULL,
    error_type VARCHAR(50),               -- schema_mismatch, type_error, timeout, unknown
    kafka_topic VARCHAR(200),
    kafka_partition INT,
    kafka_offset BIGINT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    status VARCHAR(20) DEFAULT 'failed',  -- failed, retrying, resolved, abandoned
    created_at TIMESTAMP DEFAULT NOW(),
    last_retry_at TIMESTAMP,
    resolved_at TIMESTAMP,
    resolved_by VARCHAR(100)
);

CREATE INDEX idx_failed_table ON failed_sync_logs(target_table);
CREATE INDEX idx_failed_status ON failed_sync_logs(status);
CREATE INDEX idx_failed_created ON failed_sync_logs(created_at DESC);

COMMIT;
```

---

## 2. Source Agent (MongoDB)

File: `internal/service/recon_source_agent.go`

```go
type SourceAgent struct {
    client *mongo.Client
    logger *zap.Logger
}

type SourceReport struct {
    Database   string
    Collection string
    Count      int64
    IDs        []string      // Tier 2
    ChunkHashes []ChunkHash  // Tier 3 (Merkle)
}

type ChunkHash struct {
    StartID string
    EndID   string
    Count   int
    Hash    string // MD5 of sorted concat(record hashes)
}

// Tier 1: Count
func (sa *SourceAgent) CountDocuments(ctx, database, collection) (int64, error)

// Tier 2: ID Set (batch)
func (sa *SourceAgent) GetIDs(ctx, database, collection, batchSize, skip) ([]string, error)
// Query: db.collection.find({}, {_id: 1}).sort({_id: 1}).skip(skip).limit(batchSize)

// Tier 3: Merkle Tree Hash
func (sa *SourceAgent) GetChunkHashes(ctx, database, collection, chunkSize) ([]ChunkHash, error)
// Logic:
//   1. Sort by _id, batch chunkSize records
//   2. Per chunk: hash = MD5(concat(BSONhash(doc) for doc in chunk))
//   3. Return array of ChunkHash
```

---

## 3. Dest Agent (Postgres)

File: `internal/service/recon_dest_agent.go`

```go
type DestAgent struct {
    db     *gorm.DB
    logger *zap.Logger
}

type DestReport struct {
    Table       string
    Count       int64
    IDs         []string
    ChunkHashes []ChunkHash
}

// Tier 1: Count
func (da *DestAgent) CountRows(ctx, tableName, pkColumn) (int64, error)
// Query: SELECT COUNT(*) FROM "table"

// Tier 2: ID Set (batch)
func (da *DestAgent) GetIDs(ctx, tableName, pkColumn, batchSize, offset) ([]string, error)
// Query: SELECT "pk" FROM "table" ORDER BY "pk" LIMIT batchSize OFFSET offset

// Tier 3: Merkle Tree Hash
func (da *DestAgent) GetChunkHashes(ctx, tableName, pkColumn, chunkSize) ([]ChunkHash, error)
// Query per chunk:
//   SELECT md5(string_agg("pk" || COALESCE("_hash",''), '' ORDER BY "pk"))
//   FROM (SELECT "pk", "_hash" FROM "table" ORDER BY "pk" LIMIT chunkSize OFFSET N) sub
```

---

## 4. Recon Core

File: `internal/service/recon_core.go`

```go
type ReconCore struct {
    sourceAgent   *SourceAgent
    destAgent     *DestAgent
    db            *gorm.DB
    schemaAdapter *SchemaAdapter
    mongoClient   *mongo.Client
    logger        *zap.Logger
}

// RunTier1: Quick count comparison
func (rc *ReconCore) RunTier1(ctx, entry TableRegistry) *ReconciliationReport {
    sourceCount := rc.sourceAgent.CountDocuments(ctx, entry.SourceDB, entry.SourceTable)
    destCount := rc.destAgent.CountRows(ctx, entry.TargetTable, entry.PrimaryKeyField)
    diff := sourceCount - destCount
    status := "ok"
    if diff != 0 { status = "drift" }
    // Save report + if drift → trigger Tier 2
}

// RunTier2: ID set comparison (find missing IDs)
func (rc *ReconCore) RunTier2(ctx, entry TableRegistry) *ReconciliationReport {
    // Batch 10K: get IDs from both sides
    // Set diff: sourceIDs - destIDs = missingIDs
    // Save report with missingIDs
}

// RunTier3: Merkle Tree hash comparison (find stale records)
func (rc *ReconCore) RunTier3(ctx, entry TableRegistry) *ReconciliationReport {
    sourceChunks := rc.sourceAgent.GetChunkHashes(ctx, ...)
    destChunks := rc.destAgent.GetChunkHashes(ctx, ...)
    // Compare chunk by chunk
    // Mismatched chunks → drill down to find exact stale records
}

// Version-aware Heal
func (rc *ReconCore) Heal(ctx, entry TableRegistry, missingIDs []string) (healed int, err error) {
    for _, id := range missingIDs {
        // 1. Fetch from MongoDB
        doc := mongoClient.FindOne(bson.M{"_id": id})
        mongoTimestamp := doc["updatedAt"] or doc["_id"].Timestamp()
        
        // 2. Check Postgres version
        var pgSyncedAt time.Time
        db.Raw("SELECT _synced_at FROM table WHERE pk = ?", id).Scan(&pgSyncedAt)
        
        // 3. Compare: only UPSERT if source newer
        if pgSyncedAt.IsZero() || mongoTimestamp.After(pgSyncedAt) {
            // UPSERT via SchemaAdapter
            healed++
        }
        
        // 4. Audit log
        db.Create(&AuditLog{Action: "heal", Table: entry.TargetTable, RecordID: id, Reason: "missing"})
    }
}

// CheckAll: run Tier 1 for all active tables
func (rc *ReconCore) CheckAll(ctx) []ReconciliationReport
```

---

## 5. Worker Hardening

### 5.1 BatchBuffer → failed_sync_logs

```go
// In batchUpsert, when error:
if err := bb.db.Exec(query, values...).Error; err != nil {
    bb.db.Create(&model.FailedSyncLog{
        TargetTable:  tableName,
        RecordID:     r.PrimaryKeyValue,
        Operation:    "upsert",
        RawJSON:      datatypes.JSON(r.RawData),
        ErrorMessage: err.Error(),
        ErrorType:    classifyError(err),  // schema_mismatch, type_error, timeout
        Status:       "failed",
    })
    metrics.SyncFailed.WithLabelValues(tableName, "upsert", r.Source).Inc()
} else {
    metrics.SyncSuccess.WithLabelValues(tableName, "upsert", r.Source).Inc()
}
```

### 5.2 Prometheus Counters

```go
var (
    SyncSuccess = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cdc_sync_success_total",
            Help: "Total successful CDC syncs",
        },
        []string{"table", "operation", "source"},
    )
    SyncFailed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cdc_sync_failed_total",
            Help: "Total failed CDC syncs",
        },
        []string{"table", "operation", "source"},
    )
    ConsumerLag = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "cdc_kafka_consumer_lag",
            Help: "Kafka consumer group lag per topic partition",
        },
        []string{"topic", "partition"},
    )
)
```

---

## 6. CMS API

| Method | Path | Purpose |
|:-------|:-----|:--------|
| GET | /api/reconciliation/report | Latest report per table |
| GET | /api/reconciliation/report/:table | History for specific table |
| POST | /api/reconciliation/check | Trigger Tier 1 all tables |
| POST | /api/reconciliation/check/:table | Trigger Tier 2 specific table |
| POST | /api/reconciliation/deep-check/:table | Trigger Tier 3 |
| POST | /api/reconciliation/heal/:table | Version-aware heal |
| GET | /api/failed-sync-logs | List (paginated, filterable) |
| POST | /api/failed-sync-logs/:id/retry | Retry single record |
| POST | /api/failed-sync-logs/retry-all | Retry all failed for table |
| POST | /api/tools/reset-debezium-offset | Signal → MongoDB debezium_signal |
| POST | /api/tools/trigger-snapshot/:table | Signal → incremental snapshot |
| POST | /api/tools/reset-kafka-offset/:topic | Pause consumer + reset + resume |

---

## 7. FE Data Integrity Dashboard

### Page: `/data-integrity`

**Tab 1: Tổng quan**
- Table per row: source count, dest count, diff, status badge (Matched/Drifted), last check time
- Action buttons per row: Check Now (Tier 1), Deep Check (Tier 3), Heal

**Tab 2: Failed Sync Logs**
- Filterable table: target_table, error_type, status, date range
- Per row: record_id, error message, raw JSON preview, retry button
- Bulk: Retry All, Abandon All

**Tab 3: Tools**
- Reset Debezium Offset (select table)
- Trigger Snapshot (select table)
- Reset Kafka Consumer Offset (select topic)

---

## 8. Kafka + Debezium Config

### Kafka cleanup.policy=compact
```bash
# Per CDC topic
kafka-configs --alter --entity-type topics \
  --entity-name cdc.goopay.centralized-export-service.export-jobs \
  --add-config cleanup.policy=compact \
  --bootstrap-server localhost:9092
```

### Schema History Topic retention
```bash
# Unlimited retention for schema history
kafka-configs --alter --entity-type topics \
  --entity-name __debezium-schema-history \
  --add-config retention.ms=-1 \
  --bootstrap-server localhost:9092
```

### Debezium Signal Collection
```javascript
// MongoDB: create signal collection
db.createCollection("debezium_signal")

// Trigger incremental snapshot:
db.debezium_signal.insertOne({
    "type": "execute-snapshot",
    "data": {
        "data-collections": ["payment-bill-service.export-jobs"],
        "type": "incremental"
    }
})
```

---

## 9. MongoDB Go Driver Config

```yaml
# config-local.yml
mongodb:
  url: mongodb://localhost:17017/?replicaSet=rs0
```

```go
// config.go
type MongoDBConfig struct {
    URL string `mapstructure:"url"`
}
```
