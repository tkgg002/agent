# Plan: ReconHealReportDetails

## Goal
Add detailed stats (lookback/heal time range, missing count, mismatched count, and orphan count) to the reconciliation reports table and the NATS response payload of the heal commands.

## Proposed Changes

### Component: GORM Models
Update GORM models for `ReconciliationReport` in both:
- `centralized-data-service`: `internal/model/recon/reconciliation_report.go`
- `cdc-cms-service`: `internal/model/recon/reconciliation_report.go` and `internal/domain/recon/report.go`

Add fields:
- `OrphanCount` (`orphan_count`)
- `HealedDurationMs` (`healed_duration_ms`)

### Component: Recon Engine & Check
Update `RunTier2` (in `recon_tier_a.go`) and `RunSegmentB` (in `recon_tier_b.go`) to compute and set `OrphanCount` during the reconciliation check.
- In `RunTier2` (Segment A): `OrphanCount` is `len(missingFromSrc)`.
- In `RunSegmentB` (Segment B): `OrphanCount` is `len(staleObj.OrphanInMaster)`.

### Component: Recon Healer & Handler
Update `healSegmentA` and `healSegmentB` in `internal/handler/recon/recon_heal_v4.go`:
- Track the duration of the heal process (`healed_duration_ms`).
- Retrieve counts of missing, mismatched, and orphan IDs from the latest report.
- Save these counts and the `healed_duration_ms` back to the database using `reportRepo.UpdateByID`.
- Return detailed JSON structure in NATS responses containing:
  - `status`, `segment`, `healed_count`, `healed_at`, `healed_duration_ms`
  - `checked_at`
  - `missing_count`
  - `mismatched_count`
  - `orphan_count`

## Verification Plan
1. Compile both services.
2. Run tests to ensure everything is correct.
