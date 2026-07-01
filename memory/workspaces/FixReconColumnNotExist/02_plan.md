# Plan - FixReconColumnNotExist

## Execution Phases

1. **Research & Design**:
   - Analyze mapping from raw table names to Master FQN via `ListActiveMasterBindings`.
   - Design column check API for `ReconDestAgent` to verify column existence on Postgres shadow table.
   - Design centralized timestamp field resolution logic in `resolveSourceTSField`.
2. **Implementation**:
   - Rename `listActiveMasterBindings` to `ListActiveMasterBindings` and make it public.
   - Fix report lookup in `healSegmentB` and `recon_handler_run.go` to use FQN.
   - Add `ColumnExists` in `ReconDestAgent`.
   - Update `resolveSourceTSField` to use `ColumnExists` for source timestamp verification and fallback synchronization.
3. **Verification**:
   - Execute package unit tests: `go test -v ./internal/service/recon/...`.
   - Add new unit tests validating the synchronized fallback logic.

