# 05_progress.md - Reconciliation Extended Metrics & Stale IDs

## Progress Audit Log

[2026-07-21 17:09:00] [Agent:Deepmind] Complete implementation of granular reconciliation diagnostics for `payment_bills` pipeline:
- Created official SQL migration file `095_add_recon_jobs_drift_metrics.sql` in `cdc-cms-service/migrations/schema/recon_dlq/`.
- Removed ad-hoc inline DDL query from `NewReconJobRepo` constructor in `recon_job_repo.go`.
- Updated `ReconJob` struct and `cdc_system.recon_jobs` schema with `total_record_diff_count`, `source_count`, `dest_count`, and `stale_ids`.
- Extended `ChunkStreamBucketEngine` to extract `StaleIDs` (missing_from_shadow, missing_from_master, mismatched) and wrapped `diffIDTs` in OpenTelemetry span `cdc.recon.diff_idts`.
- Corrected `total_diff_count` assignment in `ReconJobWorker` to preserve actual record diff count (`totalDiff`) instead of sub-window count.
- Extended `ReconJobRepository` interface with `UpdateStatusExtended`.
- Verified all unit tests (`go test ./internal/service/recon/...`) pass successfully.
[2026-07-21 17:55:00] [Agent:Deepmind] Aligned total_diff_count and cdc_reconciliation_report field mapping:
- Assigned `totalDiff = res.DriftWindowCount` (count of 15-min drift windows) to `total_diff_count` DB parameter.
- Preserved `totalRecordDiff = res.TotalRecordDiffCount` for aggregate record-level drift count.
- Structured `cdc_reconciliation_report` field population: mapped `MissingFromShadow` to `missing_ids` and `missing_count`, `Mismatched` to `stale_count` and `stale_ids`, and `MissingFromMaster` to `orphan_count`.

[2026-07-21 18:00:00] [Agent:Deepmind] Enhanced observability trace hierarchy and span titles:
- Connected `drift_drill_down` child spans directly to parent trace context `ctx` to resolve disconnected/orphaned trace trees.
- Grouped 15-minute sub-window hash comparisons under `cdc.recon.hash_window` parent span.
- Enriched span title formatting across `ChunkStreamBucketEngine`, Tier-A, and Tier-B with explicit time bounds `[start -> end]`.


