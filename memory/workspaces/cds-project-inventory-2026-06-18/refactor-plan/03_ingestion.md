# Phase 3: Domain `ingestion`

## Mục tiêu
`internal/ingestion/` — Kafka/NATS → Shadow DB write pipeline.

---

## Bước 3.1: `internal/ingestion/model.go`

| Struct | Nguồn |
|---|---|
| `CDCEvent` | `internal/model/cdc_event.go` |
| `CDCEventData` | `internal/model/cdc_event.go` |
| `UpsertRecord` | `internal/model/cdc_event.go` |
| `ShadowBinding` | `internal/model/shadow_binding.go` |

---

## Bước 3.2: `internal/ingestion/repository.go` (Port — MỚI)

```go
package ingestion

type ShadowBindingRepository interface {
    GetByID(ctx, id int64) (*ShadowBinding, error)
    GetByCode(ctx, code string) (*ShadowBinding, error)
    GetActiveBySourceObject(ctx, sourceObjectID int64) (*ShadowBinding, error)
    ListBySourceObject(ctx, sourceObjectID int64) ([]ShadowBinding, error)
    Create(ctx, item *ShadowBinding) error
    Update(ctx, item *ShadowBinding) error
}
```

---

## Bước 3.3: Move GORM Repo

| File cũ | File mới |
|---|---|
| `internal/repository/shadow_binding_repo.go` | `internal/ingestion/repository/gorm_shadow_binding_repo.go` |

---

## Bước 3.4: Move Services → `internal/ingestion/service/`

| File cũ | File mới |
|---|---|
| `internal/service/schema_adapter.go` | `internal/ingestion/service/schema_adapter.go` |
| `internal/service/dynamic_mapper.go` | `internal/ingestion/service/dynamic_mapper.go` |
| `internal/service/enrichment_service.go` | `internal/ingestion/service/enrichment_service.go` |
| `internal/service/source_router.go` | `internal/ingestion/service/source_router.go` |
| `internal/service/child_explode.go` | `internal/ingestion/service/child_explode.go` |
| `internal/service/schema_validator.go` | `internal/ingestion/service/schema_validator.go` |

**Key functions trong `schema_adapter.go`** (33 funcs — file 900L):

| Func | Hành động |
|---|---|
| `NewSchemaAdapter(db, logger)` | Move |
| `GetSchema(tableName)` | Move |
| `GetSchemaInSchema(schemaName, tableName)` | Move |
| `InvalidateCache(tableName)` | Move |
| `InvalidateCacheInSchema(schemaName, tableName)` | Move |
| `FlushAll()` | Move |
| `loadSchemaInSchema(schemaName, tableName)` | Move (private) |
| `PrepareForCDCInsert(tableName, pkColumn)` | Move |
| `PrepareForCDCInsertInSchema(schemaName, tableName, pkColumn)` | Move |
| `PrepareForCDCInsertWithBusinessCols(...)` | Move |
| `createShadowTableV1(schemaName, tableName, pkColumn)` | Move (private) |
| `createShadowTableV1WithCols(...)` | Move (private) |
| `BuildUpsertSQL(schema, tableName, pkField, ...)` | Move |
| `BuildUpsertSQLInSchema(schema, schemaName, tableName, pkField, ...)` | Move |
| `BuildBatchUpsertSQLInSchema(...)` | Move |
| `BuildBatchUpsertSQLsInSchema(...)` | Move |
| `IsJSONB(schema, colName)` | Move |
| `CoerceValue(schema, colName, val)` | Move |
| `buildConflictTarget(schema, pkField, pkIdent)` | Move (private) |
| `getMetadataInsertCols(schema, pkField)` | Move (private) |
| `getMetadataInsertPlaceholdersAndValues(...)` | Move (private) |
| `buildMetadataUpdateSets(schema, qualifiedTable)` | Move (private) |
| `buildOCCWhereClause(schema, qualifiedTable, hasPositiveTs)` | Move (private) |
| `coerceToIntOrNull(logger, colName, val)` | Move (private) |
| `coerceToFloatOrNull(logger, colName, val)` | Move (private) |
| `coerceToBoolOrNull(logger, colName, val)` | Move (private) |
| `decodeBase64JSON(v)` | Move (private) |
| `normalizeMongoExtendedJSON(val)` | Move (private) |
| `asOIDValue(v)` | Move (private) |
| `asDateValue(v)` | Move (private) |
| `schemaCacheKey(schemaName, tableName)` | Move (private) |
| `quoteQualifiedTable(schemaName, tableName)` | Move (private) |

**Key functions trong `dynamic_mapper.go`** (19 funcs):

| Func | Hành động |
|---|---|
| `NewDynamicMapper(registry, logger, adapters...)` | Move |
| `SetSchemaAdapter(schemaAdapter)` | Move |
| `SetMaskingService(masking)` | Move |
| `Masking()` | Move |
| `LoadRules(ctx)` | Move |
| `GetRulesForBinding(bindingID)` | Move |
| `MapData(ctx, bindingID, rawData)` | Move |
| `maskRawData(bindingID, rawData)` | Move (private) |
| `maybeMaskColumn(bindingID, rule, value)` | Move (private) |
| `BuildUpsertQuery(targetTable, pkField, mappedData)` | Move |
| `convertType(val, dataType)` | Move (private) |
| `toInt64(val)`, `toFloat64(val)`, `toBool(val)` | Move (private) |
| `toTimestamp(val)` | Move (private) |
| `unwrapMongoTypes(val)` | Move (private) |
| `getNestedField(data, path)` | Move (private) |

---

## Bước 3.5: Move Handlers → `internal/ingestion/handler/`

| File cũ | File mới |
|---|---|
| `internal/handler/kafka_consumer.go` | `internal/ingestion/handler/kafka_consumer.go` |
| `internal/handler/event_handler.go` | `internal/ingestion/handler/event_handler.go` |
| `internal/handler/event_bridge.go` | `internal/ingestion/handler/event_bridge.go` |
| `internal/handler/batch_buffer.go` | `internal/ingestion/handler/batch_buffer.go` |
| `internal/handler/consumer_pool.go` | `internal/ingestion/handler/consumer_pool.go` |

**Key functions trong `kafka_consumer.go`** (41 funcs — file 1520L):

| Func | Hành động |
|---|---|
| `NewKafkaConsumer(...)` | Move |
| `SetBatchFlushSize(n)` | Move |
| `SetFlushInterval(seconds)` | Move |
| `SetPostConsumeAction(name, action)` | Move |
| `SetDLQCircuitBreaker(cb)` | Move |
| `SetDestHealthCheck(f)` | Move |
| `SetAdaptiveBatchConfig(enabled, lagThreshold, maxMultiplier)` | Move |
| `buildReader(topics)` | Move (private) |
| `RefreshTopics(ctx)` | Move |
| `Start(ctx)` | Move |
| `processMessage(ctx, msg)` | Move (private) |
| `extractSourceTsMs(source)` | Move (private) |
| `discoverTopics(ctx)` | Move (private) |
| `filterMatchingTopics(topicNames, configuredPrefixes, debeziumTables)` | Move (private) |
| `getAvroCodec(schemaID)` | Move (private) |
| `unwrapAvroUnion(v)`, `unwrapAvroUnionMap(m)` | Move (private) |
| `sanitizeAvroSchemaNames(schema)` | Move (private) |
| `fixNames(v)` | Move (private) |
| `getOrCreateBatch(topic)` | Move (private) |
| `flushBatch(ctx, topic)` | Move (private) |
| `flushAllBatches(ctx)` | Move (private) |
| `runPostConsumeAction(ctx, b, entry, completedAt)` | Move (private) |
| `writeDLQ(ctx, msg, procErr)` | Move (private) |
| `extractDLQMetadata(msg)` | Move (private) |
| `sanitizeDLQRawJSON(table, raw)` | Move (private) |
| `Stop()` | Move |
| `isKafkaTransientError(err)`, `classifyKafkaErr(err)` | Move (private) |
| `topicSetEqual(a, b)` | Move (private) |

**Key functions trong `event_handler.go`** (15 funcs):

| Func | Hành động |
|---|---|
| `NewEventHandler(...)` | Move |
| `SetChildExplodeService(svc)` | Move |
| `FlushDrift(ctx)` | Move |
| `FlushBatchBuffer()` | Move |
| `HandleRaw(ctx, subject, data)` | Move |
| `Handle(ctx, msg)` | Move |
| `processEvent(ctx, start, event, subject, sourceDB, sourceTable)` | Move (private) |
| `handleDelete(ctx, event, routes)` | Move (private) |
| `extractSourceAndTable(subject, source)` | Move (private) |
| `extractPrimaryKey(data, pkField, sourceType)` | Move (private) |
| `shadowSchemaName(route)`, `shadowPhysicalTable(route)`, `qualifiedShadowTable(route)` | Move (private) |

---

## Bước 3.6: Move sinkworker (giữ nguyên path)

`internal/sinkworker/` — **không move**, chỉ update imports để dùng `ingestion.ShadowBinding` thay vì `model.ShadowBinding`.

| File | Thay đổi |
|---|---|
| `sinkworker.go` | Update import path |
| `envelope.go` | Update import path |
| `upsert.go` | Update import path |
| `schema_manager.go` | Update import path |

---

## Bước 3.7: Compile Check

```bash
go build ./internal/ingestion/...
go build ./internal/sinkworker/...
```
