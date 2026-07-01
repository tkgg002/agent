# Context - ReconSelfHealing

## Goal
Implement Self-healing RE-TRIGGER (Recon V4 P2) - Soft-delete orphan master records.

## Scope
1. Update `TransmuterModule.Run` logic in `internal/service/master/transmuter.go` to soft-delete physical and logical orphan records when run with a specific list of `onlySourceIDs`.
2. Ensure updated `_source_ts` timestamp to prevent race conditions during updates.
3. Write comprehensive unit test validation verifying orphan detection and deletion rules.
