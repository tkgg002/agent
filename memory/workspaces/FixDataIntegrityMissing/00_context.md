# Context - FixDataIntegrityMissing

## Problem Statement
The user reported that the `/data-integrity` page on `http://localhost:5173` is displaying incomplete data.
Specifically:
1. Shadow tables are not showing up.
2. Master tables are missing `master_centrallized_export_service.export_jobs` and `master_centrallized_export_service_2.export_jobs`.
3. We need to audit and fix the logic in both `cdc-cms-service` and `cdc-cms-web`.

## Investigation Strategy
1. Search `cdc-cms-service` backend APIs related to `/data-integrity` or data integrity stats to see how it gathers shadow/master bindings/tables.
2. Search `cdc-cms-web` frontend to see where it requests the data integrity endpoint and how it renders it.
3. Investigate why `master_centrallized_export_service.export_jobs` and `master_centrallized_export_service_2.export_jobs` are missing (e.g., regex matching, source/destination classification, or database filtering).
