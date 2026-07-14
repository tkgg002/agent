# Requirements: Lookback Parity & Validation Audit

## Goal
To audit lookback window propagation and ensure strict functional parity with legacy `heal_v4` behavior. Solve the discrepancy where lookback scans (Tier 2 and full-diff checks) query MongoDB using Postgres column names, causing zero records to be found. Enforce mutual exclusivity of parameters in `recon check`.

## Requirements
1. **Timestamp Field Discrepancy Resolution**:
   - Differentiate between source timestamp field (`srcTS`) and destination timestamp field (`dstTS`).
   - For MongoDB source: `srcTS` must be the MongoDB field name (configured `TimestampField` on `TableRegistry`), while `dstTS` is the Postgres shadow column name (camelToSnake or probed candidate).
   - For Postgres source: both `srcTS` and `dstTS` are identical.
   - Update `pickScanRangeWithLag`, `RunTier2`, `RunTier3`, and `TimeBoundedDiffMissingFromShadow` to use their respective fields (`srcTS` for source queries, `dstTS` for Postgres shadow queries).

2. **Recon Check Options Validation**:
   - Enforce mutual exclusivity for three modes in `recon check`:
     - **Lookback Mode (hot/cold)**: `lookback` is set (`hot`/`cold`), time range and `deep` are not set.
     - **Full Search (full_diff)**: time range is set, `lookback` is not set, `deep` is false.
     - **Deep Check**: time range is set, `deep` is true.
   - Validate that if time range is specified, the range must not exceed 30 days.

3. **Recon Check Full Search Implementation**:
   - When a full search (`start_time`/`end_time` provided) is requested on Segment A, invoke `TimeBoundedDiffMissingFromShadow` instead of `RunTier2` to align with `heal_v4`'s direct diff check.
   - Return a correctly structured `ReconciliationReport` representation.

4. **Verify and Decommission**:
   - Verify by compiling and running tests.
   - Check smoke test (`recon_smoke.go`).
