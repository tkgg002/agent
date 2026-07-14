# Tasks: Lookback Parity & Validation Audit

- [ ] Task 1: Create Implementation Plan and get User Approval
- [ ] Task 2: Implement source vs destination timestamp field resolution in `ReconCore`
- [ ] Task 3: Refactor `pickScanRangeWithLag`, `RunTier2`, `RunTier3`, and `TimeBoundedDiffMissingFromShadow` to use split `srcTS` and `dstTS` fields
- [ ] Task 4: Add parameter validation and mutual exclusivity in `HandleReconCheck`
- [ ] Task 5: Implement direct `TimeBoundedDiffMissingFromShadow` call in `HandleReconCheck` for Segment A full search mode
- [ ] Task 6: Compile and verify correctness of logic via tests
- [ ] Task 7: Decommission old `recon_bk` package
