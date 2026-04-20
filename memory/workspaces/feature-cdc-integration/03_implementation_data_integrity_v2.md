# Implementation: Data Integrity (v2)

> Date: 2026-04-16
> Source: User's `01_requirements_data_integrity_solution.md`

## Migration SQL

```sql
-- 008_reconciliation.sql

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
    check_type VARCHAR(20) NOT NULL,  -- count, id_set, hash
    status VARCHAR(20) NOT NULL,       -- ok, drift, error
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
    record_id VARCHAR(200),
    operation VARCHAR(10),             -- c, u, d
    raw_json JSONB,
    error_message TEXT NOT NULL,
    kafka_topic VARCHAR(200),
    kafka_partition INT,
    kafka_offset BIGINT,
    retry_count INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'failed', -- failed, retried, resolved
    created_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP
);

CREATE INDEX idx_failed_table ON failed_sync_logs(target_table);
CREATE INDEX idx_failed_status ON failed_sync_logs(status);
CREATE INDEX idx_failed_created ON failed_sync_logs(created_at DESC);
```

## Source Agent (MongoDB)

```go
// recon_source_agent.go
type SourceAgent struct {
    mongoClient *mongo.Client
    logger      *zap.Logger
}

type SourceReport struct {
    Database   string
    Collection string
    Count      int64
    IDs        []string   // for Tier 2
    SampleHash string     // for Tier 3
}

func (sa *SourceAgent) CountCheck(ctx, db, collection) int64
func (sa *SourceAgent) IDSetCheck(ctx, db, collection, batchSize, offset) []string
func (sa *SourceAgent) HashCheck(ctx, db, collection, sampleSize) string
```

## Dest Agent (Postgres)

```go
// recon_dest_agent.go  
type DestAgent struct {
    db     *gorm.DB
    logger *zap.Logger
}

type DestReport struct {
    Table      string
    Count      int64
    IDs        []string
    SampleHash string
}

func (da *DestAgent) CountCheck(ctx, tableName) int64
func (da *DestAgent) IDSetCheck(ctx, tableName, pkColumn, batchSize, offset) []string
func (da *DestAgent) HashCheck(ctx, tableName, pkColumn, sampleSize) string
```

## Recon Core

```go
// recon_core.go
type ReconCore struct {
    sourceAgent *SourceAgent
    destAgent   *DestAgent
    db          *gorm.DB           // for reports + heal
    schema      *SchemaAdapter     // for upsert
    logger      *zap.Logger
}

func (rc *ReconCore) RunTier1(ctx, entry TableRegistry) *ReconciliationReport
func (rc *ReconCore) RunTier2(ctx, entry TableRegistry) *ReconciliationReport
func (rc *ReconCore) RunTier3(ctx, entry TableRegistry) *ReconciliationReport
func (rc *ReconCore) Heal(ctx, table string, missingIDs []string) (int, error)
func (rc *ReconCore) CheckAll(ctx) []ReconciliationReport
```

### Heal logic
```go
func (rc *ReconCore) Heal(ctx, entry TableRegistry, missingIDs []string) {
    for _, id := range missingIDs {
        // 1. Fetch from MongoDB
        doc := mongoClient.FindOne(ctx, bson.M{"_id": id})
        
        // 2. Convert to map + raw JSON
        rawJSON, _ := bson.MarshalExtJSON(doc, true, false)
        
        // 3. Upsert Postgres via SchemaAdapter (bypass Kafka)
        schema := schemaAdapter.GetSchema(entry.TargetTable)
        schemaAdapter.PrepareForCDCInsert(entry.TargetTable, entry.PrimaryKeyField)
        query, values := schemaAdapter.BuildUpsertSQL(schema, ...)
        db.Exec(query, values...)
    }
}
```

## BatchBuffer → failed_sync_logs

```go
// In batchUpsert, when error:
if err := bb.db.Exec(query, values...).Error; err != nil {
    // Log to failed_sync_logs instead of just zap.Error
    bb.db.Create(&model.FailedSyncLog{
        TargetTable:  tableName,
        RecordID:     r.PrimaryKeyValue,
        Operation:    "upsert",
        RawJSON:      r.RawData,
        ErrorMessage: err.Error(),
        Status:       "failed",
    })
}
```

## Prometheus Counters

```go
// pkgs/metrics/prometheus.go
var (
    SyncSuccess = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "cdc_sync_success_total"},
        []string{"table", "operation", "source"},
    )
    SyncFailed = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "cdc_sync_failed_total"},
        []string{"table", "operation", "source"},
    )
)
```

## CMS API

| Method | Path | Purpose |
|:-------|:-----|:--------|
| GET | /api/reconciliation/report | Latest report per table |
| POST | /api/reconciliation/check | Trigger Tier 1 check now |
| POST | /api/reconciliation/check/:table | Trigger Tier 2 for specific table |
| POST | /api/reconciliation/heal/:table | Trigger heal for specific table |
| GET | /api/failed-sync-logs | List failed records (paginated) |
| POST | /api/failed-sync-logs/:id/retry | Retry a failed record |

## MongoDB Config

```yaml
mongodb:
  url: mongodb://localhost:17017/?replicaSet=rs0
```
