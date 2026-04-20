# Solution: Data Integrity (FINAL)

> Date: 2026-04-16

## Files tạo mới

### Worker (centralized-data-service)
| File | Lines est. | Purpose |
|:-----|:-----------|:--------|
| `migrations/008_reconciliation.sql` | 50 | cdc_reconciliation_report + failed_sync_logs |
| `internal/model/reconciliation_report.go` | 30 | Model |
| `internal/model/failed_sync_log.go` | 30 | Model |
| `internal/service/recon_source_agent.go` | 150 | MongoDB: count + IDs batch + Merkle hash |
| `internal/service/recon_dest_agent.go` | 120 | Postgres: count + IDs batch + Merkle hash |
| `internal/service/recon_core.go` | 250 | Orchestrator: tiered check + version-aware heal + audit |
| `pkgs/mongodb/client.go` | 40 | MongoDB Go driver init |

### Worker files sửa
| File | Thay đổi |
|:-----|:---------|
| `config/config.go` | +MongoDBConfig |
| `config/config-local.yml` | +mongodb section |
| `internal/handler/batch_buffer.go` | Error → failed_sync_logs + metrics |
| `internal/server/worker_server.go` | Init MongoDB + ReconCore + schedule |
| `pkgs/metrics/prometheus.go` | +SyncSuccess, SyncFailed, ConsumerLag |

### CMS (cdc-cms-service)
| File | Purpose |
|:-----|:--------|
| `internal/api/reconciliation_handler.go` | 12 endpoints |
| `internal/model/reconciliation_report.go` | Model |
| `internal/model/failed_sync_log.go` | Model |
| `internal/router/router.go` | Register routes |

### FE (cdc-cms-web)
| File | Purpose |
|:-----|:--------|
| `src/pages/DataIntegrity.tsx` | 3-tab dashboard |
| `src/App.tsx` | Route + menu |

## Execution order

```
T1-T4 (infra) → T5-T7 (agents) → T8-T10 (core) → T11-T13 (hardening) → T14-T16 (CMS) → T17-T22 (verify + extras)
```

## Dependencies
- `go.mongodb.org/mongo-driver/mongo` — MongoDB Go driver
- `go.mongodb.org/mongo-driver/bson` — BSON encoding
