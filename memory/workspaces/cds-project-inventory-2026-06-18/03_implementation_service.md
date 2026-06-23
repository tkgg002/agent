# 03_implementation_service.md — internal/service/

## Tổng quan
Business logic layer. 48 files, ~300KB code.

---

## activity_logger.go (3,660 bytes)
- `ActivityLogger` — ghi activity log vào DB (`cdc_activity_log`)

## backfill_source_ts.go (14,458 bytes)
- `BackfillSourceTsService` — backfill `source_ts` (timestamp field) cho các records trong shadow table

## bridge_service.go (1,398 bytes)
- `BridgeService` — bridge schema changes giữa CDC streams

## child_explode.go (10,921 bytes)
- `childExplode(...)` — "explode" nested JSON object thành flat columns trong shadow table
- Tạo ALTER TABLE thêm columns cho nested paths

## child_explode_master.go (2,182 bytes)
- Variant của child_explode cho master table

## connection_manager.go (1,401 bytes)
- `ConnectionManager` — quản lý DB connections theo source code

## connection_overrides.go (1,577 bytes)
- Xử lý `CONNECTION_OVERRIDE_*` env vars

## connector_resolver.go (3,278 bytes)
- `ConnectorResolver` — resolve Debezium connector name từ registry entry

## debezium_signal.go (17,082 bytes)
- `DebeziumSignalClient` — gửi incremental snapshot signals tới Debezium
- Execute snapshot, pause, resume qua Kafka signal topic

## dlq_worker.go (12,305 bytes)
- `DLQWorker` — xử lý DLQ retry loop

## dynamic_mapper.go (10,497 bytes)
- `DynamicMapper` — map CDC event fields → shadow table columns
- Apply masking rules, type coercion

## enrichment_service.go (1,604 bytes)
- `EnrichmentService` — enrich event data với metadata

## full_count_aggregator.go (8,534 bytes)
- `FullCountAggregator` — đếm records trong source collection (cho reconciliation)

## job_monitor.go (6,254 bytes)
- `JobMonitor` — monitor trạng thái các sync jobs

## masking_service.go (18,228 bytes)
**Functions:**
- `NewMaskingService(db, logger)` — khởi tạo
- `SetAESKey(key)` — inject AES key override
- `GetMaskedValue(fieldName, value, bindingID)` — mask 1 field value
- `LoadSensitiveFields(ctx)` — load sensitive fields config từ DB
- `ApplyMasking(ctx, record, shadowBindingID)` — apply masking cho cả record
- **4 strategies**: NONE, DROP, HASH_HMAC, PARTIAL (phone/email)

## masking_service_test.go (24,749 bytes)
- Unit tests cho masking service

## master_ddl_generator.go (28,550 bytes)
- `MasterDDLGenerator` — sinh DDL SQL cho master table
- `GenerateCreateTable(...)` — CREATE TABLE statement
- `GenerateAlterColumn(...)` — ALTER COLUMN statement
- Validate type whitelist, quote identifiers safely

## metadata_registry_service.go (31,707 bytes)
- `MetadataRegistryService` — registry in-memory cho tất cả configs
- `ReloadAll(ctx)` — reload toàn bộ từ DB vào cache
- `GetRoute(sourceID)` — get routing config cho source
- `GetSourceDSN(code)` — resolve DSN từ connection registry (multi-scheme: literal/env-pointer/build-from-fields/AES)
- `GetMappingRules(bindingID)` — get mapping rules cho shadow binding
- `GetSensitiveFields()` — get sensitive fields config
- `MetadataRegistry` interface — contract cho service

## mongo_introspection.go (6,473 bytes)
- `MongoIntrospectionService` — introspect MongoDB schema
- `DiscoverDatabases(ctx, uri)` — list databases
- `DiscoverCollections(ctx, uri, db)` — list collections
- `SampleSchema(ctx, uri, db, collection)` — sample documents để infer schema

## partition_dropper.go (18,882 bytes)
- `PartitionDropper` — drop old shadow table partitions theo retention policy

## provisioning_orchestrator.go (30,666 bytes)
- `ProvisioningOrchestrator` — orchestrate multi-step provisioning wizard
- State machine: `pending` → `shadow_bind` → `discover` → `schedule_enable` → `done`

## provisioning_state_machine.go (4,300 bytes)
- State transitions cho provisioning

## recon_alert.go (3,322 bytes)
- `ReconAlert` — publish alerts khi phát hiện data drift

## recon_core.go (70,405 bytes — 1900 dòng) ⚠️ FILE LỚN
- `ReconCore` — core reconciliation engine
- 3-tier reconciliation: Tier 1 (fast hash), Tier 2 (deep compare), Tier 3 (full count)
- `RunReconcile(ctx, table)` — entry point

## recon_dest_agent.go (20,844 bytes)
- `ReconDestAgent` — đọc data từ destination (shadow DB) để so sánh

## recon_heal.go (27,664 bytes)
- `ReconHealer` — heal mismatches sau reconciliation

## recon_source_agent.go (39,766 bytes)
- `ReconSourceAgent` — đọc data từ source (MongoDB/PG/MySQL) để so sánh
- `getClient(sourceURL)` — resolve MongoDB client

## registry_service.go (9,850 bytes)
- `RegistryService` — CRUD operations cho table registry

## scan_service.go (2,819 bytes)
- `ScanService` — trigger schema scan cho source tables

## schema_adapter.go (28,116 bytes)
- `SchemaAdapter` — adapt events theo shadow schema definition
- Coerce types, handle nulls, validate payload

## schema_inspector.go (9,820 bytes)
- `SchemaInspector` — inspect live schema changes từ CDC events
- `resolveTargetSchema(tableName)` — find target schema
- `publishDriftAlert(...)` — publish alert khi phát hiện schema drift

## schema_validator.go (10,095 bytes)
- `SchemaValidator` — validate event payload trước khi insert
- `ValidatePayload(tableName, payload)` — validate
- `loadOrBuild(table)` — build expectations từ DB schema
- `introspectColumns(table)` — query DB để lấy column list

## source_router.go (2,403 bytes)
- `ShouldUseDebezium(entry)` — quyết định dùng Debezium hay direct scan
- `InferTypeFromRawData(jsonValue)` — infer SQL type từ JSON value

## text_sanitizer.go (2,248 bytes)
- `SanitizeFreeformText(input, max)` — sanitize free-form text
- `SanitizeNestedStrings(value, max)` — sanitize nested JSON strings

## timestamp_detector.go (9,126 bytes)
- `TimestampDetector` — detect timestamp fields trong MongoDB collection
- `DetectForCollection(ctx, uri, db, collection)` — sample documents để detect

## transform_registry.go (7,020 bytes)
- Registry các transform functions (applied per field)
- `ApplyTransform(name, raw)` — apply 1 transform
- **Transforms**: `mongo_date_ms`, `oid_to_hex`, `bigint_str`, `numeric_cast`, `lowercase`, `jsonb_passthrough`, `null_if_empty`

## transmute_scheduler.go (5,739 bytes)
- `TransmuteScheduler` — cron scheduler cho transmute jobs
- Chạy transmute theo interval từ config

## transmuter.go (31,592 bytes)
- `TransmuterModule` — transform data từ shadow → master table
- `Run(ctx, masterName, onlySourceIDs)` — core run loop
- `loadMaster(...)` — load master binding config
- `loadRules(...)` — load mapping rules (với cache)
- `fetchShadowBatch(...)` — fetch batch từ shadow table
- `processBatch(...)` — apply rules, upsert master
- `InvalidateRuleCache(bindingID, masterTable)` — invalidate cache
- `upsertMaster(...)` — UPSERT vào master table
- Supports: deterministic gpay_id generation, BSON Extended JSON unwrapping

## type_resolver.go (6,854 bytes)
- `TypeResolver` — resolve và validate SQL types
- `IsTypeWhitelisted(spec)` — whitelist check (phòng SQL injection)
- `ValidateValue(ctx, spec, value)` — validate value against type
- `ResolveEnum(ctx, name)` — resolve ENUM values từ DB

## wal_monitor.go (5,791 bytes)
- `WALMonitor` — monitor PostgreSQL WAL replication slots
- `Run(ctx)` — run monitor loop
- `evaluate(rows, now)` — detect inactive/lagging slots
- `publish(ev)` — publish resume event qua NATS

## transmute/ (subdirectory)
- Pure functions cho transmute rules

---

## internal/service/transmute/ subdirectory
- `MappingRule` struct
- `ColumnExtractor` type
- Pure transform functions
