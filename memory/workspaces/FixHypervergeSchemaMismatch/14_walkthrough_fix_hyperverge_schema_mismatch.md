# Walkthrough: Fix Schema Mismatch Primary Key Resolution

## Changes Made
### `centralized-data-service`
1. [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L350-L390)
   - Removed `if pgPKField == "_id" { pgPKField = "id" }` and `if !mappedPK && pkField == "_id" { pgPKField = "id" }`.
2. [bridge_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/bridge_handler.go#L281-L284)
   - Replaced forced fallback `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }` with `if resolved.pgPKField == "" { resolved.pgPKField = "_id" }`.

## Validation Results
- Governance Linter validation PASSED 🟢.
- Preserved exact MongoDB `_id` Primary Key contract for shadow table CDC upserts.
