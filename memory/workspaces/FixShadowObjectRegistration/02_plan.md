# Implementation Plan: Fix Registration and Server Startup

## Proposed Changes

### 1. Fix Migration Syntax
- **File**: `cdc-cms-service/migrations/007_worker_schedule.sql`
- **Action**: Ensure the `INSERT` statement is correctly formatted. The `ON CONFLICT` clause must follow the `VALUES` list correctly.

### 2. Environment Cleanup
- **Action**: Kill any process occupying port 8083.
- **Action**: Restart the server and verify migrations complete successfully.

### 3. Verify Registration Logic
- **Action**: Test the registration flow again.
- **Action**: Check server logs for any synchronization errors between legacy registry and V2 registry.

## Tasks
- [ ] Fix `007_worker_schedule.sql` syntax.
- [ ] Kill process on port 8083.
- [ ] Run server and confirm successful startup.
- [ ] Verify object registration visibility in `/shadow`.
