# Implementation Plan - Fix Ambiguous Transform Scope (V2 Hybrid)

This document details the exact files and lines of code that will be changed to address target table name collisions in the batch transform pipeline.

## Proposed Code Changes

### Phase 1: Service Layer & Helpers

#### 1. `metadata_registry_service.go`
- **Location**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/metadata_registry_service.go`
- **Changes**: Cache qualified target names (`ShadowSchema.TargetTable`) in both `rs.targetCache` and `rs.targetRouteMap` if `ShadowSchema` is present.
- **Lines to edit**: ~220-221

#### 2. `helpers.go`
- **Location**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/metadata/helpers.go`
- **Changes**:
  - Update `ResolveTargetSchema` to extract schema prefix from a qualified name (like `schema.table`).
  - Update `ResolveTargetTableConfig` to parse target table names with a schema prefix. Use type assertion on `RegistryResolver` to call `GetByTargetTableAndSchema` if supported.
- **Lines to edit**: ~24-50

### Phase 2: Repository Layer

#### 3. `table_registry_repo.go`
- **Location**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/source/table_registry_repo.go`
- **Changes**:
  - In `GetAllActive`, query shadow schemas for active bindings and assign them to `ShadowSchema` of the synthesized table registry objects.
  - In `GetByID` and `GetByTargetTable`, dynamically fetch and populate the `ShadowSchema` property.
  - Implement `GetByTargetTableAndSchema` method to retrieve specific table configurations when schema namespace conflicts exist.
- **Lines to edit**: ~18-35

### Phase 3: Handler Layer & Job Dispatcher

#### 4. `batch_transform_handler.go`
- **Location**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_transform_handler.go`
- **Changes**:
  - In `runTransformJob`, separate the target table into schema prefix and `pureTable` name.
  - Use `pureTable` for checking table exists, checking `_raw_data` column exists, checking target column exists, and detecting primary keys to prevent doubling the schema namespace (e.g. `schema.schema.table`).
  - Query mapping rules uniquely using `ListActiveBySourceObjectAndBinding` or `ListActiveBySourceObject` if the route has a resolved `SourceObject.ID` / `ShadowBinding.ID`, avoiding flat `sourceTable` collisions.
- **Lines to edit**: ~93-137, ~162, ~182, ~186

#### 5. `server_jobs.go`
- **Location**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_jobs.go`
- **Changes**: Modify dispatcher logic to dispatch qualified table names if `QualifiedTarget()` exists, and allow targeting by either flat or qualified table names.
- **Lines to edit**: ~60-70

### Phase 4: CMS Actions Handler & Tests

#### 6. `source_object_actions_handler.go`
- **Location**: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/source_object_actions_handler.go`
- **Changes**: Dispatch qualified table names (`schema.table`) in the NATS message payload if `ShadowSchema` is populated.
- **Lines to edit**: ~748-757

#### 7. `source_object_actions_handler_test.go`
- **Location**: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/test/internal/api/source_object_actions_handler_test.go`
- **Changes**: Update unit assertions to verify that the NATS payload now correctly carries the qualified target table name.
- **Lines to edit**: ~109-120

## Verification Plan

### Test Suites to Run
1. `centralized-data-service`:
   - `go test -v ./internal/handler/shadow/...`
   - `go test -v ./internal/service/metadata/...`
2. `cdc-cms-service`:
   - `go test -v ./internal/api/source/...`

All tests must pass successfully (Green).
