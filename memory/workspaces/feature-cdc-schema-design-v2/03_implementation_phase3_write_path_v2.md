# Implementation Phase 3 — Write Path V2 Foundation

## Code changes

### 1. Extended `UpsertRecord`

- Updated `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/model/cdc_event.go`
- Added fields:
  - `SchemaName`
  - `ConnectionRole`
  - `ConnectionKey`
  - `PhysicalTableFQN`

### 2. Metadata route now exposes connection identity

- Updated `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/metadata_registry_service.go`
- Added:
  - `connectionRepo`
  - `ResolvedSourceRoute.ShadowConnectionKey`
- `ReloadAll()` now loads `connection_registry` and maps `shadow_connection_id -> connection_code`

### 3. Worker wiring now includes `ConnectionManager`

- Updated `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/server/worker_server.go`
- Added:
  - `connectionRepo`
  - `connectionManager`
- Injected `connectionManager` into:
  - `BatchBuffer`
  - `EventHandler`

### 4. Event handler emits V2-aware ingest records

- Updated `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/handler/event_handler.go`
- Shadow ingest records now include:
  - shadow schema
  - shadow connection key
  - physical table FQN
- Delete path now uses:
  - route-derived schema/table
  - connection manager when a shadow connection key is available

### 5. Batch buffer now groups by connection + schema + table

- Updated `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/handler/batch_buffer.go`
- Added:
  - `SetConnectionManager`
  - connection-aware grouping
  - schema-aware schema adapter resolution per destination DB
- Writes now resolve DB by:
  - `shadow` role + `ConnectionKey`
  - `master` role + `ConnectionKey`
  - fallback to legacy shared DB

### 6. Schema adapter upgraded for schema-qualified tables

- Updated `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/schema_adapter.go`
- Added:
  - `GetSchemaInSchema`
  - `InvalidateCacheInSchema`
  - `PrepareForCDCInsertInSchema`
  - `BuildUpsertSQLInSchema`
- Preserved old methods as wrappers to `public`
- Added backward-compatible cache lookup/store using both:
  - `schema.table`
  - legacy `table`

## Result

Ingest write path is now materially closer to V2:

- metadata lookup: V2
- route carries connection key: yes
- batch grouping: connection + schema + table
- SQL generation: schema-aware
- actual DB selection: connection-manager aware

Remaining gap:
- full master/transmuter runtime is not yet moved to the same V2 transport pattern
