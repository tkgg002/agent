# Context: ReconHealReportDetails

## Problem Description
- When performing reconciliation healing (`recon-heal-a` or `recon-heal-b`), the user needs to know:
  1. The execution time range/duration of the heal process.
  2. The number of missing records (`missing_count`).
  3. The number of mismatched records (`mismatched_count`).
  4. The number of orphan records (`orphan_count`).
- These metrics must be recorded in the `cdc_reconciliation_report` database table and exposed in the NATS response payload of the heal commands.
- Currently, the database table `cdc_reconciliation_report` has:
  - `duration_ms` (which represents check duration, not heal duration).
  - `missing_count` (missing from destination).
  - `stale_count` (mismatched).
  - No explicit columns for:
    - `orphan_count` (missing from source, stored as part of the JSON `stale_ids`).
    - `healed_duration_ms` (time it took to run the heal dispatch process).
- The Go struct `ReconciliationReport` in both `centralized-data-service` and `cdc-cms-service` needs to be updated.
- Database migrations need to be created/applied to add the missing columns (`orphan_count`, `healed_duration_ms`) to the `cdc_reconciliation_report` table.
