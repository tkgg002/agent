# 00_context.md — CDS Project Inventory 2026-06-18

> **Workspace**: `cds-project-inventory-2026-06-18`
> **Agent**: Brain (Antigravity)
> **Mục tiêu**: Đọc và lưu trữ toàn bộ thông tin dự án `centralized-data-service` vào workspace — chi tiết từng thư mục, file, function.
> **Source**: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`

---

## 1. Tổng quan dự án

**centralized-data-service** là CDC (Change Data Capture) Worker Service — trung tâm xử lý dữ liệu cho hệ thống GooPay CDC pipeline. Service này:

- Lắng nghe events từ Kafka (Debezium CDC topics)
- Transform & replicate data từ nguồn (MongoDB, PostgreSQL, MySQL) sang shadow DB và master DB
- Quản lý schema, masking, reconciliation, provisioning
- Expose Admin API và metrics endpoint

## 2. Thông tin Module

| Field | Giá trị |
|---|---|
| Go Module | `centralized-data-service` |
| Go Version | `1.26.1` |
| Tổng file `.go` | **191 files** |
| Config path default | `./config/config-local.yml` |
| Service name | `cdc-worker` (worker), `cdc-admin-api`, `cdc-sinkworker` |

## 3. Ba Entrypoint (cmd/)

| Binary | Path | Mô tả |
|---|---|---|
| `worker` | `cmd/worker/main.go` | Worker chính: Kafka consumer + NATS handler + cron jobs |
| `sinkworker` | `cmd/sinkworker/main.go` | Kafka sink: consume `cdc.goopay.*` topics → upsert vào shadow DB |
| `admin-api` | `cmd/admin-api/main.go` | REST API để operator quản lý CDC pipeline |

## 4. Cấu trúc thư mục

```
centralized-data-service/
├── cmd/
│   ├── worker/           — Worker entrypoint
│   ├── sinkworker/       — Sink worker entrypoint
│   └── admin-api/        — Admin API entrypoint
├── config/               — Config loading (viper + YAML)
├── internal/
│   ├── activity/         — Taxonomy/event types
│   ├── admin/            — Admin HTTP server (gin/fiber)
│   ├── handler/          — NATS command handlers + Kafka consumer
│   ├── model/            — GORM models (DB entities)
│   ├── naming/           — Naming conventions
│   ├── repository/       — Data access layer (GORM repos)
│   ├── server/           — WorkerServer wiring + lifecycle
│   ├── service/          — Business logic layer
│   └── sinkworker/       — Kafka sink processing
├── migrations/
│   └── dest/             — Destination DB migrations (1 file)
├── pkgs/                 — Shared packages
│   ├── crypto/
│   ├── database/         — GORM + pgx pool helpers
│   ├── idgen/            — Sonyflake ID generator
│   ├── kafka/            — Avro encoder
│   ├── metrics/          — Prometheus metrics
│   ├── mongodb/          — MongoDB client
│   ├── natsconn/         — NATS connection factory
│   ├── observability/    — OTel (traces + logs + metrics)
│   ├── rediscache/       — Redis client
│   └── utils/
├── scripts/
├── test/
├── docs/
├── deployments/
├── Makefile
├── docker-compose.yml
└── go.mod
```

## 5. Key Dependencies

| Dependency | Version | Dùng cho |
|---|---|---|
| `gorm.io/gorm` | v1.31.1 | ORM cho PostgreSQL |
| `go.mongodb.org/mongo-driver` | v1.17.9 | MongoDB client |
| `github.com/segmentio/kafka-go` | v0.4.50 | Kafka consumer/producer |
| `github.com/nats-io/nats.go` | v1.51.0 | NATS messaging |
| `github.com/gin-gonic/gin` | v1.12.0 | HTTP framework |
| `github.com/gofiber/fiber/v2` | v2.52.12 | HTTP framework (sinkworker) |
| `go.opentelemetry.io/otel` | v1.43.0 | Tracing & metrics |
| `go.uber.org/zap` | v1.27.1 | Structured logging |
| `github.com/sony/sonyflake` | v1.3.0 | Distributed ID generation |
| `github.com/redis/go-redis/v9` | v9.18.0 | Redis cache |
| `github.com/prometheus/client_golang` | v1.23.2 | Metrics |
| `github.com/spf13/viper` | v1.21.0 | Config |
| `github.com/robfig/cron/v3` | v3.0.1 | Scheduled jobs |
| `github.com/sony/gobreaker` | v1.0.0 | Circuit breaker |
