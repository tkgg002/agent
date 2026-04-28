# Implementation Phase 4 — Master Runtime V2

## Code changes

### 1. Repository support extended

- Updated `internal/repository/master_binding_repo.go`
  - added `GetByMasterTable`
  - added `ListActiveByShadowBinding`
- Updated `internal/repository/mapping_rule_v2_repo.go`
  - added `ListActiveBySourceObject`

### 2. Transmuter moved to V2 bindings

- Replaced `internal/service/transmuter.go`
- New behavior:
  - loads master destination from `cdc_system.master_binding`
  - resolves:
    - `master_connection_key`
    - `master_schema`
    - `master_table`
    - `shadow_connection_key`
    - `shadow_schema`
    - `shadow_table`
  - loads rules from `cdc_system.mapping_rule_v2`
  - reads shadow rows from the correct shadow DB through `ConnectionManager`
  - writes master rows to the correct master DB/schema through `ConnectionManager`

### 3. Master DDL generator moved to V2 metadata

- Replaced `internal/service/master_ddl_generator.go`
- New behavior:
  - resolves master destination from `cdc_system.master_binding`
  - builds schema-qualified DDL
  - reads business columns from `mapping_rule_v2`
  - applies DDL on the resolved master DB from `ConnectionManager`
  - only applies legacy RLS helper when target schema is `public`

### 4. NATS fanout made V2-aware

- Updated `internal/handler/transmute_handler.go`
- `HandleTransmuteShadow` now fans out via:
  - `cdc_system.master_binding`
  - `cdc_system.shadow_binding`

### 5. Worker wiring updated

- Updated `internal/server/worker_server.go`
- Transmuter now receives:
  - system DB
  - `ConnectionManager`
  - `TypeResolver`
- Master DDL generator now receives:
  - system DB
  - `ConnectionManager`
  - `MappingRuleV2Repo`

## Important note

This phase upgrades runtime master execution to V2, but does **not** yet redesign the schedule storage itself:
- `transmute_schedule` still lives in legacy `cdc_internal`
- however the schedule dispatch target (`master_table`) now resolves into V2 runtime bindings
