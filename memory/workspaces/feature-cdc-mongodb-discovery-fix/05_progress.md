# Progress Log - CDC MongoDB Discovery Fix

## 2026-05-12 11:00 [Brain:Antigravity] Started task stabilization.

### Analysis & Root Cause
- **Field Scanning**: `scanFieldsMongoSource` used `TableRegistry` model and legacy ID resolution, failing to find V2 `SourceObjectRegistry` records. Fallback introspection was missing when shadow tables were absent.
- **Snapshot Now**: `ReconHandler` used raw MongoDB insertion into a hardcoded signal collection, bypassing the robust `DebeziumSignalClient`.
- **Auto-Sync**: CMS service lacked NATS dispatch to notify worker/debezium of registry changes.

### Actions Taken
1. **[Worker] Fix Scan Fields ID Resolution**:
   - Updated `scanFieldsMongoSource` in `command_handler.go` to query `SourceObjectRegistry` first.
   - Added heuristic fallback for legacy numeric IDs.
   - Implemented direct MongoDB introspection fallback when shadow tables are empty.
   - Fixed pointer dereference bug for `SourceDatabase`.
2. **[Worker] Standardize Snapshot Signals**:
   - Integrated `DebeziumSignalClient` into `ReconHandler`.
   - Wired `DebeziumSignalClient` in `worker_server.go`.
   - Refactored `HandleDebeziumSignal` to use the client for consistent signaling (qualified names, incremental snapshots).
3. **[CMS] Implement Auto-Sync Trigger**:
   - Added `RestartDebeziumCommand` to `source_async.go`.
   - Updated `RegistryHandler.Update` and `RegistryHandler.Register` to dispatch `cdc.cmd.restart-debezium` when Debezium registry changes.
4. **[Hardening] Introspection, Port Cleanup & Build Fixes**:
   - Refactored `MongoIntrospectionService` to support full URIs and enforced `directConnection=true`.
   - Fixed `cdc-cms-service` redeclaration error (`RestartDebeziumCommand`).
   - Fixed `centralized-data-service` unused import (`fmt`).
   - Added missing `debezium` section to worker's `config-local.yml`.
   - Freed ports 8081, 8082, 8083.

### Verification Status
- [x] Code audited for ID resolution logic.
- [x] Code audited for Debezium signal path.
- [x] All services build successfully (`go build`).
- [x] Ports 8081-8083 verified available.
- [x] Detailed report generated: [report_stabilization_20260512.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-mongodb-discovery-fix/report_stabilization_20260512.md)
- [ ] E2E Verification (requires real NATS/Mongo/Kafka).
