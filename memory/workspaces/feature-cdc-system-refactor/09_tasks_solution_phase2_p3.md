# Task Solutions: Phase 2 Decoupling - Pillar P3 & P5

**Workspace**: `feature-cdc-system-refactor`
**Date**: 2026-05-07

## 1. Backend Solutions (cdc-cms-service)

### A. Extract Master Swap Logic
**File**: `internal/app/commands/master_swap.go` (NEW)
**Logic**: 
- Create `MasterSwapCommand` struct.
- In `Handle()`, build the NATS payload, create a `cdc_jobs` entry (status: pending).
- Publish to `cdc.cmd.master-swap`.
- Return 202 and `job_id`.

### B. Extract V2 Sync Logic
**File**: `internal/app/commands/v2_sync.go` (NEW)
**Logic**:
- Create `V2SyncCommand` struct.
- Handle creation of `cdc_jobs` entry, publish to `cdc.cmd.v2-sync`.
- Return 202 and `job_id`.

### C. Refactor Handlers
**Files**: `internal/api/master_registry_handler.go` & `internal/api/registry_handler.go`
**Logic**:
- Remove inline `ALTER TABLE RENAME TO` and `SyncFromLegacy` functions.
- Inject `CommandBus`.
- Call `cmdBus.Dispatch(c.Context(), MasterSwapCommand{...})` and return `c.Status(202).JSON(...)`.

## 2. Worker Solutions (centralized-data-service)

### A. Subscriptions & Handlers
**File**: `internal/handler/command_handler.go` & `internal/handler/master_ddl_handler.go`
**Logic**:
- Subscribe to `cdc.cmd.master-swap`. Create handler to execute the `ALTER TABLE RENAME TO` query safely on the master DB.
- Subscribe to `cdc.cmd.v2-sync`. Create handler to execute the DB sync logic.

### B. Event Emitters & Job Tracking
**File**: `internal/service/job_monitor.go` (or individual handlers)
**Logic**:
- Upon completion of swap/sync, emit `cdc.evt.master-swap.completed` / `cdc.evt.v2-sync.completed`.
- Update `cdc_system.cdc_jobs` via `jobRepo.UpdateStatus(jobID, "success"/"failed")`.

## 3. Frontend Solutions (cdc-cms-web)

### A. Polling Hook
**File**: `src/hooks/useAsyncJob.ts` (NEW)
**Logic**:
- Uses `setInterval` to poll `/api/jobs/:id` every 2 seconds until status is `success` or `failed`.
- Returns `{ status, loading, result, error }`.

### B. Component Integration
**File**: `src/pages/MasterRegistry/MasterRegistry.tsx` (and related)
**Logic**:
- On Action (Swap): Dispatch API call. If response is 202 with `job_id`, trigger `useAsyncJob`.
- Display a modal/spinner overlay showing "Processing Swap..." while loading.
- On success, reload the table list and show toast message.

---
> **MUSCLE DELEGATION**: 
> Execute these changes strictly per file. Follow the `[Brain:Unverified]` status by validating endpoints after code insertion. RUN tests!
