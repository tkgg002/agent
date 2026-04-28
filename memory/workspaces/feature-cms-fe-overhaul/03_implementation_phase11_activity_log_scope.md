# Implementation — Phase 11 Activity Log Scope

- Backend:
  - add `ActivityLogRow` response with source/shadow metadata
  - list query joins `cdc_system.shadow_binding` + `source_object_registry`
  - support source/shadow filters while keeping `target_table`
  - enrich `recent_errors` in stats
  - update swagger/comment
- Frontend:
  - `useAsyncDispatch` now supports `statusParams`
  - `ActivityLog.tsx` renders `source_database.source_table` and `shadow_schema.shadow_table`
  - add source DB filter on page
