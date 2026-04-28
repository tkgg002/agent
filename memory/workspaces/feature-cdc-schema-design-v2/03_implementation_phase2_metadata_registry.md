# Implementation Phase 2 — Metadata Registry

## Code changes

### 1. Added V2-aware registry service

- Added `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/metadata_registry_service.go`
- Added:
  - `MetadataRegistry` interface
  - `ResolvedSourceRoute`
  - `MetadataRegistryService`

### 2. Routing source switched to V2 tables

`MetadataRegistryService` now loads:
- `cdc_system.source_object_registry`
- `cdc_system.shadow_binding`

and synthesizes temporary `model.TableRegistry` structs for compatibility with legacy downstream code.

### 3. Mapping compatibility preserved

- `MetadataRegistryService` still reads legacy `cdc_mapping_rules` via `MappingRuleRepo`
- Mapping rules are projected onto shadow target tables using V2 source/shadow route information
- This is intentional because current ingest path still maps into shadow-like tables and V2 master-bound rules are not yet fully consumed by the runtime

### 4. Consumers switched to interface

- Updated `internal/service/dynamic_mapper.go`
  - now depends on `MetadataRegistry`
- Updated `internal/handler/event_handler.go`
  - now depends on `MetadataRegistry`
  - now resolves route through `ResolveSourceRoute(sourceDB, sourceTable)`
- Updated `internal/server/worker_server.go`
  - now constructs `MetadataRegistryService`
  - now injects it into `DynamicMapper` and `EventHandler`

### 5. Legacy registry kept as compatibility implementation

- Updated `internal/service/registry_service.go`
  - added `ResolveSourceRoute` so the old service still satisfies the new interface

### 6. Tests added

- Added `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/metadata_registry_service_test.go`
  - verifies route resolution by `sourceDB + sourceTable`

## Important limitation

This phase only upgrades metadata lookup, not the final write transport.

Current runtime still has these legacy constraints:
- `BatchBuffer` groups by `TableName`
- `UpsertRecord` only carries `TableName`, not `schema` or `connection key`
- writes still go through the current shared DB handle

Therefore this phase is **hybrid**:
- metadata lookup = V2-aware
- write path = legacy-compatible
