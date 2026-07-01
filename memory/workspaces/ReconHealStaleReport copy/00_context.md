# Context - ReconHealStaleReport

## Goal
Fix Reconciliation Heal Segment A logic to correctly fetch the latest check report (including healthy "ok" ones) instead of persistently retrieving stale drifted reports, which blocks the heal process and prevents recovery.

## Scope
1. Update `healSegmentA` in `internal/handler/recon/recon_heal_v4.go` to use `GetLatestByTable` instead of `GetLatestMissingReportWithSegment`.
2. Add comprehensive unit tests in `internal/handler/recon/recon_heal_v4_test.go` or similar to verify:
   - When a table is healthy (last check is "ok" with 0 drifts), heal returns "noop" directly.
   - When a table is drifted, heal proceeds with snapshot signal.
3. Validate overall code compile and unit tests pass.
