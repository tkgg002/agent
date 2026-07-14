# Implementation Plan - Fix Reconciliation History Endpoint

## Problem Statement
Accessing `GET /api/reconciliation/report/schedule_histories` yields a 500 Internal Server Error.

### Root Cause
An integration test on the real database revealed that the SQL query executed in `GetTableHistory` attempts to select columns `healed_mismatched_at`, `healed_missing_src_at`, and `healed_missing_dest_at` from the `cdc_system.cdc_reconciliation_report` table. However, these columns are not present in the current database schema. They were added to the Go struct and queries in the latest commit on branch `recon-heal` but the corresponding migration file was omitted.

## Proposed Changes
We will create a new migration script to add the missing columns to the database.

### Database Migration
#### [NEW] [093_recon_heal_timestamps.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/093_recon_heal_timestamps.sql)
Add columns `healed_mismatched_at`, `healed_missing_src_at`, and `healed_missing_dest_at` to the table `cdc_system.cdc_reconciliation_report`.

```sql
BEGIN;

ALTER TABLE cdc_system.cdc_reconciliation_report
  ADD COLUMN IF NOT EXISTS healed_mismatched_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS healed_missing_src_at TIMESTAMP,
  ADD COLUMN IF NOT EXISTS healed_missing_dest_at TIMESTAMP;

COMMIT;
```

## Verification Plan
### Integration Test
- Execute the real database test:
  ```bash
  CFG_PATH=/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/config/config-local.yml go test -v -run TestGetTableHistory_RealDB ./internal/infra/persistence/recon/...
  ```
- Ensure it runs successfully and does not throw the missing column SQL error.
