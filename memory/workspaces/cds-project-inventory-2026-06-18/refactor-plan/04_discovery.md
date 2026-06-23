# Phase 4: Domain `discovery`

## Mục tiêu
`internal/discovery/` — Introspect source, sample fields, snapshot MongoDB.

---

## Bước 4.1: `internal/discovery/model.go`

| Struct | Table |
|---|---|
| `SnapshotDLQ` | `cdc_system.snapshot_dlq` |

> `SourceObjectRegistry` dùng chung — import từ `internal/source`

---

## Bước 4.2: `internal/discovery/repository.go` (Port — MỚI)

```go
package discovery

type SnapshotDLQRepository interface {
    Create(ctx, item *SnapshotDLQ) error
    GetPending(ctx) ([]SnapshotDLQ, error)
    MarkDone(ctx, id int64) error
    MarkError(ctx, id int64, errMsg string) error
}
```

> **Hiện tại**: `SnapshotDLQ` write inline, chưa có repo → cần tạo mới `internal/discovery/repository/gorm_snapshot_dlq_repo.go`

---

## Bước 4.3: Move Services → `internal/discovery/service/`

| File cũ | File mới |
|---|---|
| `internal/service/mongo_introspection.go` | `internal/discovery/service/mongo_introspection.go` |
| `internal/service/timestamp_detector.go` | `internal/discovery/service/timestamp_detector.go` |
| `internal/service/backfill_source_ts.go` | `internal/discovery/service/backfill_source_ts.go` |
| `internal/service/scan_service.go` | `internal/discovery/service/scan_service.go` |

**Key functions trong `mongo_introspection.go`** (6 funcs):

| Func | Hành động |
|---|---|
| `SanitizeMongoDSN(uri)` | Move |
| `NewMongoIntrospectionService()` | Move |
| `DiscoverDatabases(uri)` | Move |
| `DiscoverCollections(uri, dbName)` | Move |
| `IntrospectCollection(uri, dbName, collectionName, sampleSize)` | Move |
| `IntrospectCollectionDiagnose(uri, dbName, collectionName, sampleSize)` | Move |

**Key functions trong `backfill_source_ts.go`** (14 funcs):

| Func | Hành động |
|---|---|
| `NewBackfillSourceTsService(...)` | Move |
| `SetMetadataRegistry(metadata)` | Move |
| `BackfillAll(ctx, ...)` | Move |
| `BackfillOne(ctx, ...)` | Move |
| `lookupRegistry(...)` | Move (private) |
| `listActiveTableConfigs(ctx)` | Move (private) |
| `fetchNullBatch(...)` | Move (private) |
| `fetchSourceTsMap(...)` | Move (private) |
| `applyBatch(...)` | Move (private) |
| `beginRun(...)` | Move (private) |
| `touchRunProgress(...)` | Move (private) |
| `finishRun(...)` | Move (private) |
| `WriteActivity(...)` | Move |
| `applyDefaults()` | Move (private) |

---

## Bước 4.4: Move Handlers → `internal/discovery/handler/`

**Tách từ `command_handler.go`** → 3 files:

### `internal/discovery/handler/discover_handler.go`

| Func | Từ dòng |
|---|---|
| `HandleDiscover(msg)` | L.962 |
| `scanFieldsMongoSource(ctx, registryID, shadowBindingID, sourceTable, autoApprove)` | L.407 |
| `scanFieldsDebezium(ctx, registryID, shadowBindingID, targetTable, sourceTable, sourceType, autoApprove)` | L.2693 |
| `HandleScanFields(msg)` | L.2745 |
| `findSimilarCollections(target, all)` | L.551 |
| `inferSQLTypeFromLegacyCatalogProp(prop)` | L.2659 |

### `internal/discovery/handler/scan_handler.go`

| Func | Từ dòng |
|---|---|
| `HandleScanRawData(msg)` | L.1817 |
| `HandleScanArrayFields(msg)` | L.2018 |
| `HandlePeriodicScan(msg)` | L.2375 |
| `HandleBackfill(msg)` | L.1238 |
| `validScanIdent(s)` | L.1956 |
| `explodePathToPGPath(path)` | L.1976 |
| `flattenJSONWithTypes(prefix, value, result)` | L.2326 |
| `buildCastExpr(field, dataType)` | L.2492 |
| `detectPrimaryKey(execDB, schemaName, tableName)` | L.1794 |

### `internal/discovery/handler/mongo_discover_handler.go`

| Func | Từ dòng |
|---|---|
| `HandleDiscoverMongoDatabases(msg)` | L.1378 |
| `HandleDiscoverMongoCollections(msg)` | L.1440 |
| `replyMongoDiscovery(msg, replyTo, payload)` | L.1553 |

**Move toàn bộ** `internal/handler/snapshot_runner_handler.go` → `internal/discovery/handler/snapshot_runner.go`:

| Func | Hành động |
|---|---|
| `NewSnapshotRunner(...)` | Move |
| `Handle(msg)` | Move |
| `runSnapshot(ctx, p, jobID)` | Move (private) |
| `claimProgress(ctx, p, jobID)` | Move (private) |
| `checkpoint(ctx, progressID, lastSeen, rowsTotal)` | Move (private) |
| `markProgressError(ctx, progressID, msg)` | Move (private) |
| `markProgressDone(ctx, progressID, rowsTotal, totalRows)` | Move (private) |
| `updateClusterTime(ctx, progressID, clusterTimeMs, method)` | Move (private) |
| `captureClusterTime(ctx, db)` | Move (private) |
| `buildResumeFilter(lastSeen)` | Move (private) |
| `extractDocID(doc)` | Move (private) |
| `buildSnapshotEnvelope(afterJSON, now, clusterTimeMs)` | Move (private) |
| `nullableString(s)` | Move (private) |
| `writeActivity(...)` | Move (private) |

---

## Bước 4.5: Compile Check

```bash
go build ./internal/discovery/...
go test ./internal/discovery/...
```
