# Requirements Phase 1 Implementation

## Scope

- Scaffold V2 metadata foundation directly inside `cdc-system/centralized-data-service`.
- Add new SQL migrations for V2 control-plane tables.
- Add Go models for the new tables.
- Add repository scaffolding for the new models.
- Keep runtime behavior unchanged for now; no risky rewiring of ingest/transmute paths in this phase.

## Deliverables

1. New migrations under `migrations/` for:
   - `cdc_system` schema bootstrap
   - `connection_registry`
   - `source_object_registry`
   - `shadow_binding`
   - `master_binding`
   - `mapping_rule_v2`
   - `sync_runtime_state`
   - legacy backfill bootstrap
2. New models under `internal/model/`.
3. New repositories under `internal/repository/`.
4. Formatting + compile-level verification for the new Go code.

## Non-Goals

- Refactor `RegistryService`, `EventHandler`, `TransmuterModule`, `MasterDDLGenerator` in this phase.
- Replace existing V1 tables.
- Build the full connection manager yet.
