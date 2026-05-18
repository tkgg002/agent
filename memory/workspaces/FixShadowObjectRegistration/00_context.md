# Workspace Context: FixShadowObjectRegistration

## Objective
Resolve the regression in Source Object Registration where newly registered objects do not appear in the shadow registry list. Address port conflicts and migration errors preventing the server from running correctly.

## Root Cause Analysis (Initial)
- **Port Conflict**: Server failed to start with `bind: address already in use` on port 8083.
- **Migration Error**: Syntax error in `007_worker_schedule.sql` near `ON CONFLICT` due to possible missing delimiters or incorrect placement of the new `airbyte-sync` row.
- **Registration Visibility**: If the server is not running the latest code (due to the above), the V2 sync logic I added won't be active.

## Current State
- `cdc-cms-service` is failing to start.
- `007_worker_schedule.sql` has a syntax error.
- User reported registration is not working (objects not showing in `/shadow`).
