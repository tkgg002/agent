# Plan - ReconHealStaleReport

## Execution Phases
1. **Research & Design**:
   - Trace how `healSegmentA` retrieves reports.
   - Verify if `GetLatestByTable` is already available on `reportRepo`.
   - Verify how the return values and safety checks change when `GetLatestByTable` returns a report with 0 drifts.
2. **Implementation (Delegate to Muscle)**:
   - Edit `internal/handler/recon/recon_heal_v4.go` to change query function call.
   - Adjust `healSegmentA` logic to handle healthy reports (status="ok" or missing_count=0 & stale_count=0) by returning `noop`.
3. **Verification**:
   - Write unit tests in `recon_heal_v4_test.go` or equivalent.
   - Run tests.
