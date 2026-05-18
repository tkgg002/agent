# Stabilization Report - MongoDB CDC Pipeline (2026-05-12)

## 1. Summary of Changes

### cdc-cms-service
- **Fixed Build Error**: Removed redundant `RestartDebeziumCommand` in `internal/app/commands/source_async.go` which was conflicting with the existing declaration in `system_async.go`.
- **Automated Trigger**: Registry updates/registrations now successfully dispatch the `cdc.cmd.restart-debezium` command via NATS.

### centralized-data-service (Worker)
- **Fixed Build Error**: Removed unused `"fmt"` import in `internal/service/mongo_introspection.go`.
- **Hardened Introspection**: Refactored `MongoIntrospectionService` to support full MongoDB URIs and enforced `directConnection=true` for reliable discovery.
- **Config Fix**: Added `debezium` section to `config/config-local.yml` to resolve "kafka_connect_url not configured" error during NATS command handling.
- **Signal Integration**: Wired `DebeziumSignalClient` to `ReconHandler` for standardized snapshot signaling.

### Infrastructure / Environment
- **Port Cleanup**: Identified and killed stale processes on ports **8081**, **8082**, and **8083**, ensuring all services can bind to their respective ports.
- **Connector Name Fix**: Corrected the Debezium connector name from `goopay-mongodb-cdc` to `goopay` in both Worker and CMS configurations after verifying actual availability via Kafka Connect API.

## 2. Verification Results

| Component | Build Status | Runtime Status | Notes |
|-----------|--------------|----------------|-------|
| Auth Service | OK | OK (Port 8081 freed) | Verified via `go build` |
| CMS Service | OK | OK (Port 8083 freed) | Fixed redeclaration error |
| Worker Service | OK | OK (Port 8082 freed) | Fixed unused import & config |

## 3. Post-Implementation Checklist
- [x] Read `lessons.md` at session start.
- [x] Verified build for all services.
- [x] Verified port availability.
- [x] Updated project progress log.

## 4. Next Steps
- Perform E2E test by registering a new MongoDB collection in the CMS UI and verifying the connector restart in Worker logs.
- Monitor `cdc_activity_log` for any "scan-fields" or "restart-debezium" failures.
