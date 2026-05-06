# Tech Stack & Guidelines

> **Last Updated**: 2026-05-04
> **Maintained by**: Brain (Antigravity) qua workspace `feature-system-refactor-2026-05`

## Tech Stack

| Layer | Technology | Notes |
|---|---|---|
| **Language (BE)** | Go 1.26.1 | 3 service: cdc-auth, cdc-cms, centralized-data-service |
| **Framework (BE)** | Gin (worker/admin-api), Fiber v2 (cms + auth) | |
| **Language (FE)** | TypeScript | |
| **Framework (FE)** | React + Vite | `cdc-cms-web` 22 .tsx, ~7600 LOC |
| **Database** | PostgreSQL 4 instance (5432/5433/5434/5435) | metadata + cdc + dest DW + source |
| **Source Stores** | MongoDB (17017), MariaDB (13307), PostgreSQL (5435) | multi-engine source |
| **Messaging** | Apache Kafka (19092/19093) + NATS (14222) | Kafka cho CDC events từ Debezium; NATS cho command/event bus internal |
| **CDC Capture** | Debezium qua Kafka Connect (18083) + Schema Registry (18081) | Đã gỡ Airbyte (commit 8ef7d71). |
| **Cache** | Redis (16379) | mapping cache, leader election lite |
| **ORM** | GORM (Go) | |
| **Auth** | golang-jwt v5 + bcrypt | trong cdc-auth-service |
| **Rate limit** | golang.org/x/time/rate | per-token bucket trong admin-api Phase F1 |
| **Infra runtime** | Docker (compose-style local) | 13 container live |
| **Monitoring** | OpenTelemetry collector (14317/14318) → SigNoz | OTel hiện đang lookup `otel-collector` DNS sai trong worker container, log spam (cosmetic). |
| **AI (Brain)** | claude-opus-4-7 (Antigravity) | Planning + Reasoning |
| **AI (Muscle)** | Claude Code CLI | Execution + CLI Tools |

---

## AI Models & Roles Mapping

| Role | Strategy | Primary Model | Fallback |
|---|---|---|---|
| **Brain** (Chairman) | Reasoning & quality | `claude-opus-4-7` | `claude-sonnet-4-6` |
| **Muscle** (Engineer) | Speed & cost | `claude-sonnet-4-6` | `claude-haiku-4-5` |

> Cấu hình model có thể được ghi đè qua `agent/models.env` cho từng phiên đặc thù.

---

## Coding Guidelines

### General

- **Go**: gofmt + go vet. Error wrap qua `fmt.Errorf("... : %w", err)`. Logger zap.
- **TypeScript**: tsc strict. Lint qua eslint.config.js. Không tự edit `node_modules`.
- **SQL**: tên dynamic table/schema MUST quote (lesson L13). Không trust string concat.
- **Security**: token compare qua `crypto/subtle.ConstantTimeCompare`. Body size cap. Sanitize freeform error trước khi response/persist.
- **Memory file**: APPEND only (CLAUDE.md §11).

### Service / Module Structure (Go services)

```
<service>/
├── cmd/<binary>/main.go     # entrypoint
├── internal/
│   ├── handler/             # HTTP/NATS handler
│   ├── service/             # business logic
│   ├── repository/          # GORM queries
│   ├── model/               # data structs
│   └── config/              # viper-based loader
├── migrations/              # SQL migration files
├── deployments/             # docker-compose, k8s manifests
├── config/                  # YAML/env loader files
└── Makefile
```

### Patterns đang sử dụng

- ✅ **CQRS** (light, read/write tách qua handler) — cdc-cms-service.
- ✅ **Worker pool + Kafka consumer** (BatchBuffer + flush) — centralized-data-service.
- ✅ **Event-driven close-loop** (NATS evt → JobMonitor) — Phase D-39.A.
- ✅ **Outbox/saga light** — DLQ state machine.
- ✅ **Fencing token** — TransmuteScheduler chống double-tick.
- ✅ **Schema-driven mapping** — gjson + transform_fn từ `cdc_mapping_rules`.
- 🚧 **Master cascade từ admin-api** — pending B3.
- 📋 **Property-based test mapping** — planned B4.

---

## Architecture

### Communication Patterns

```
Browser → cdc-cms-web → cdc-auth-service (login)
                     ↘ cdc-cms-service (control plane CRUD)
                                       → PG cdc-metadata (5433)
                                       → NATS (cdc.cmd.*, cdc.evt.*)
                                       → Redis cache
                                       → centralized-data-service worker (signal)

Mongo / PG / MariaDB source
   → Debezium (Kafka Connect 18083)
   → Kafka topics cdc.<conn>.<db>.<table>
   → centralized-data-service worker
       → SchemaInspector (drift detect)
       → DynamicMapper + Mask
       → SchemaAdapter (auto ALTER + UNIQUE)
       → Shadow tables PG cdc-metadata (5433)
       → Post-ingest NATS cdc.cmd.transmute-shadow
   → TransmuteModule (subscribe NATS)
       → gjson eval + transform_fn
       → Master tables PG dest-DW (5434)
       → Publish cdc.evt.transmute.completed
   → JobMonitor (subscribe completed)
       → UPDATE transmute_schedule.last_status
```

### Critical Paths

1. **Operator login → Source Register → Master Approval → Schedule Run** (B2 smoke target):
   `cms-web → auth-service → cms-service → admin-api → worker → shadow → master`.
2. **Real-time CDC ingest** (verified live 2026-05-04 09:51:13 UTC):
   `mongo/pg → debezium → kafka → worker → shadow`.
3. **Cron-driven transmute close-loop**:
   `scheduler → NATS cmd → handler → svc.Run → NATS evt → JobMonitor → UPDATE schedule`.
4. **Recon + Heal**:
   `cms-service → NATS cmd recon-check → ReconCore → ReconHealer → OCC upsert`.

### Critical Files

- `centralized-data-service/internal/admin/helpers.go` — sinh shadow_schema, topic name, debezium include list. (3 vị trí pattern bug đã fix Phase F3.)
- `centralized-data-service/internal/admin/server.go` — auth + rate limit + body limit middleware (Phase F1).
- `centralized-data-service/cmd/admin-api/main.go` — boot fail-fast (Phase F1).
- `centralized-data-service/internal/service/transmuter.go` — gate chain + OCC upsert.
- `centralized-data-service/internal/service/job_monitor.go` — close-loop NATS subscriber (Phase D-39.A).
- `centralized-data-service/internal/sinkworker/schema_manager.go` — auto-ALTER shadow + schema_proposal emit.
- `cdc-cms-service/internal/api/master_registry_handler.go` — master CRUD + cdc.cmd.master-create publish.
- `cdc-cms-web/src/pages/SourceToMasterWizard.tsx` — 11-step operator wizard (FE).
