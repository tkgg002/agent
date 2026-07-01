# TODO - FixReconColumnNotExist

- [x] Rename `listActiveMasterBindings` to `ListActiveMasterBindings` and make it public.
- [x] Fix `healSegmentB` in `recon_heal_v4.go` to resolve table thô to master FQN.
- [x] Fix `GetLatestMissingReport` call in `recon_handler_run.go` to use qualified target table name.
- [x] Implement `ColumnExists` in `ReconDestAgent`.
- [x] Implement synchronized `resolveSourceTSField` filtering check in `recon_tier_a.go`.
- [x] Run existing tests and add new test cases for column fallback and sync.
- [x] Run all reconciliation tests to verify compile and correctness.


