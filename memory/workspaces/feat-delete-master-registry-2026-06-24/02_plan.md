# Plan: Delete Master Registry Functionality

## Goal
Implement a secure and reliable mechanism to delete a Master Registry configuration (binding) from the CDC control plane (CMS DB) via a `DELETE /api/v1/masters/:name` endpoint.

## Proposed Architecture
1. **API Layer**:
   - Register route `DELETE /v1/masters/:name` using the OpsAdmin role requirement and standard destructive middleware (idempotency, auditing).
   - Implement `Delete` method in `MasterRegistryHandler` (new file `internal/api/master/master_registry_handler_delete.go`).
   - The handler will execute a synchronous command `master.delete` through the command bus.

2. **Application Layer (CQRS Commands)**:
   - Create `DeleteMasterCommand` and its handler `DeleteMasterHandler` under `internal/app/commands/master/delete_master.go`.
   - The handler will validate if the master registry exists.
   - For safety, if the registry is active (`is_active = true`), prevent deletion and return a specific validation error (e.g. `ErrMasterIsActive` -> HTTP 409 Conflict).
   - If not active, delete the master binding.

3. **Repository Layer**:
   - The existing `ports.MasterRepo` has `DeleteMasterBinding(ctx, id)` and `DeleteClonedRules(ctx, id)`.
   - In PostgreSQL schema, both `transmute_schedule` and `mapping_rule_master` tables reference `master_binding(id)` with `ON DELETE CASCADE`.
   - Therefore, deleting the master binding row directly will automatically cascade delete all mapping rules and schedules.
   - However, to ensure perfect cleanup and safety, we will wrap the deletion in a database transaction or leverage the cascade constraint. We can add a repository method or reuse the existing `DeleteMasterBinding` method.

4. **Router & Server Registration**:
   - Register `DeleteMasterCommand` sync handler in `internal/server/server.go`.
   - Register the route in `internal/router/router.go`.

## Verification Steps
1. Create unit tests for `DeleteMasterHandler` verifying:
   - Success path (registry exists, is not active, deleted successfully).
   - Not found error path.
   - Active constraint error path (cannot delete active registry).
2. Execute compile validation.
3. Manually test via HTTP requests (or simulate with curl/test mocks).
