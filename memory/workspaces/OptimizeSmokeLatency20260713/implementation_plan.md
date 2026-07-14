# Optimizing Smoke Drift Latency

Provide a brief description of the problem, any background context, and what the change accomplishes:
Currently, the smoke reconciliation check encounters an 11-second delay when drift is detected because both Segment A and Segment B default to a full 7-day time-window scan. Additionally, the `BucketCounts` database calls are executed sequentially. This plan updates the lookback resolution logic to correctly respect the configured Mode (returning 2 hours for Hot mode), updates Segment B lookback checks to use the dynamic `effectiveLookback`, and runs `BucketCounts` queries concurrently.

## User Review Required

> [!NOTE]
> Setting `RunMode` to `"hot"` (default) will reduce the default smoke lookback scan window when drift is found from 7 days (168 hours) to 2 hours. If a historical drift occurred more than 2 hours ago and hasn't been corrected, a "hot" smoke check might not scan far enough back during the detailed check to find it. However, the initial count scan (`scanExact`) checks the entire table count, so any drift will still be flagged as drift, but the detailed lookback analysis will only pinpoint the exact drift times within the last 2 hours. This is the expected behavior for "hot" mode reconciliation.

## Proposed Changes

### Recon Engine & Smoke

---

#### [MODIFY] [recon_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go)
Correct `effectiveLookback` to respect `RunMode` configuration rather than returning the default 7 days even when in "hot" mode.

#### [MODIFY] [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)
- Refactor `runLookbackCheckB` to use `effectiveLookback` instead of hardcoded 7 days.
- Resolve custom timestamp columns dynamically from registry in `runLookbackCheckB`.
- Parallelize `BucketCounts` calls in `runLookbackCheckA` and `runLookbackCheckB` using `sync.WaitGroup`.

## Verification Plan

### Automated Tests
- Run the full package tests to ensure no regressions:
  `go test -v ./internal/service/recon/...`

### Manual Verification
- Verify code compliance using the governance verification check script.
