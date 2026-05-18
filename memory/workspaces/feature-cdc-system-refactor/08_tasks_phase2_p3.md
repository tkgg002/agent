# Tasks: Phase 2 Decoupling - Pillar P3 & P5

**Workspace**: `feature-cdc-system-refactor`
**Date**: 2026-05-07
**Context**: User approved skipping P2 (Read Queries) to focus on P3 (Commands/Writes) and P5 (Frontend Async Wiring).

## Checklist:

### 1. Backend (cdc-cms-service) - P3
- [ ] T3.1: Migrate Master Swap inline logic to `app/commands/master_swap.go`.
- [ ] T3.2: Migrate V2 Sync inline logic to `app/commands/v2_sync.go`.
- [ ] T3.3: Refactor `master_registry_handler.go` - remove inline `ALTER TABLE` and call `CommandBus`.
- [ ] T3.4: Refactor `registry_handler.go` - remove inline `V2Sync` and call `CommandBus`.
- [ ] T3.5: Refactor `reconciliation_handler.go` to use `CommandBus` for heavy writes.
- [ ] T3.6: Verify API handlers return `202 Accepted` with `{ "job_id": "..." }`.

### 2. Worker (centralized-data-service) - P3
- [ ] T3.7: Implement `MasterSwap` handler responding to `cdc.cmd.master-swap`.
- [ ] T3.8: Implement `V2Sync` handler responding to `cdc.cmd.v2-sync`.
- [ ] T3.9: Update `job_monitor` or handlers to emit companion events `cdc.evt.*.completed`.
- [ ] T3.10: Update `cdc_system.cdc_jobs` status based on completion events.

### 3. Frontend (cdc-cms-web) - P5
- [ ] T5.1: Create `src/hooks/useAsyncJob.ts` for polling `/api/jobs/:id`.
- [ ] T5.2: Update `MasterRegistry.tsx` (or related swap components) to handle 202 responses and poll using `useAsyncJob`.
- [ ] T5.3: Ensure UI displays loading/processing state properly without breaking existing flows.
