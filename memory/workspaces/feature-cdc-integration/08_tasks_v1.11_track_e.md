# v1.11 Track E: Airbyte Bridge + Worker Transform

## Status: COMPLETED

## Task Checklist

### E0: Airbyte → CDC Bridge
- [x] **E0.1**: Migration `004_bridge_columns.sql` + model update (cả 2 services)
- [x] **E0.2**: `HandleAirbyteBridge` trong Worker command_handler.go
- [x] **E0.3**: Subscribe `cdc.cmd.bridge-airbyte` trong worker_server.go
- [x] **E0.4**: CMS API `POST /api/registry/:id/bridge` + route
- [x] **Build verify**: OK

### E1: Batch Transform
- [x] **E1.1**: `HandleBatchTransform` trong Worker command_handler.go
- [x] **E1.2**: CMS API `POST /api/registry/:id/transform` + route
- [x] **Build verify**: OK

### E2: Periodic Scheduler
- [x] **E2**: Ticker goroutine trong worker_server.go (bridge + transform mỗi 5m)
- [x] **Config**: `transformInterval: 5m` trong config.go + config-local.yml

### E3: Transform Status
- [x] **E3**: CMS API `GET /api/registry/:id/transform-status`

## Files Changed

### Worker (centralized-data-service)
- `internal/handler/command_handler.go` — +HandleAirbyteBridge, +HandleBatchTransform
- `internal/server/worker_server.go` — +Subscribe 2 commands, +Periodic ticker, +registryRepo field
- `internal/model/table_registry.go` — +AirbyteRawTable, +LastBridgeAt
- `config/config.go` — +TransformInterval in WorkerConfig
- `config/config-local.yml` — +transformInterval: 5m

### CMS (cdc-cms-service)
- `internal/model/table_registry.go` — +AirbyteRawTable, +LastBridgeAt
- `internal/api/registry_handler.go` — +Bridge(), +Transform(), +TransformStatus()
- `internal/router/router.go` — +3 routes
- `migrations/004_bridge_columns.sql` — NEW

## New APIs
| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/registry/:id/bridge` | Copy Airbyte raw → CDC _raw_data |
| `POST` | `/api/registry/:id/transform` | Apply mapping rules → typed columns |
| `GET` | `/api/registry/:id/transform-status` | Progress: total/bridged/pending rows |

## New NATS Commands
| Subject | Pattern | Purpose |
|---------|---------|---------|
| `cdc.cmd.bridge-airbyte` | Pub/Sub | Bridge Airbyte raw → CDC table |
| `cdc.cmd.batch-transform` | Pub/Sub | Transform _raw_data → typed columns |
