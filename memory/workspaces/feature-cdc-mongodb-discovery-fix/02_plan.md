# Implementation Plan - CDC MongoDB Discovery Fix

## Phase 1: Stabilization (Completed)
- [x] Fix ScanFields ID resolution in Worker.
- [x] Integrate DebeziumSignalClient in Worker.
- [x] Add auto-sync restart trigger in CMS.

## Phase 2: Verification & Hardening
- [ ] Verify "Snapshot Now" button in UI triggers correct MongoDB signal entry.
- [ ] Verify Field Scanning works for new collections without shadow tables.
- [ ] Monitor logs for `restart-debezium` success after registry update.

## Phase 3: Cleanup
- [ ] Remove legacy `HandleDebeziumSignal` raw BSON logic (Done).
- [ ] Remove unused `TableRegistry` lookups in scan fields (Done).
