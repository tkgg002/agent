# Implementation: Data Integrity

> Date: 2026-04-16
> Phase: data_integrity

## Files mới

| File | Purpose |
|:-----|:--------|
| `migrations/008_reconciliation_report.sql` | Table cdc_reconciliation_report |
| `internal/model/reconciliation_report.go` | Model |
| `internal/service/reconciliation_worker.go` | MongoDB vs Postgres comparison + auto-heal |
| `pkgs/mongodb/client.go` | MongoDB connection (Go driver) |

## Files sửa

| File | Thay đổi |
|:-----|:---------|
| `config/config.go` | Thêm MongoDBConfig |
| `config/config-local.yml` | Thêm mongodb section |
| `internal/server/worker_server.go` | Init MongoDB client + reconciliation schedule |
| `internal/model/worker_schedule.go` | (giữ nguyên, thêm default schedule "reconcile") |

## CMS files mới/sửa

| File | Purpose |
|:-----|:--------|
| `internal/api/reconciliation_handler.go` (CMS) | Report + Check + Heal endpoints |
| `internal/router/router.go` (CMS) | Register routes |
| `src/pages/DataIntegrity.tsx` (FE) | Dashboard page |
| `src/App.tsx` (FE) | Add route |

## Reconciliation Worker logic

```go
func (rw *ReconciliationWorker) CheckTable(ctx context.Context, entry TableRegistry) ReconciliationReport {
    // 1. Count source
    sourceCount = mongoClient.Database(entry.SourceDB).Collection(entry.SourceTable).CountDocuments(ctx, bson.M{})
    
    // 2. Count dest
    destCount = db.Raw("SELECT COUNT(*) FROM " + entry.TargetTable).Scan(&count)
    
    // 3. If diff > 0: find missing IDs
    sourceIDs = mongoClient.Collection.Distinct(ctx, "_id", bson.M{})
    destIDs = db.Raw("SELECT _id FROM " + entry.TargetTable)
    missingIDs = sourceIDs - destIDs
    
    // 4. Return report
    return ReconciliationReport{
        SourceCount: sourceCount,
        DestCount:   destCount,
        Diff:        sourceCount - destCount,
        MissingIDs:  missingIDs,
        Status:      "drift" or "ok",
    }
}

func (rw *ReconciliationWorker) Heal(ctx context.Context, table string, missingIDs []string) {
    // Fetch full documents from MongoDB
    // Insert into Postgres via SchemaAdapter
    for _, id := range missingIDs {
        doc = mongoClient.FindOne(ctx, bson.M{"_id": id})
        schemaAdapter.PrepareForCDCInsert(table, "_id")
        schemaAdapter.BuildUpsertSQL(...)
        db.Exec(sql, values...)
    }
}
```

## MongoDB config

```yaml
mongodb:
  url: mongodb://localhost:17017/?replicaSet=rs0
  databases:
    - name: payment-bill-service
    - name: centralized-export-service
```
