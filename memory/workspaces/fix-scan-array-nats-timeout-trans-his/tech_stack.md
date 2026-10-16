# Tech Stack

- **Go**: 1.22+
- **Framework API**: Fiber v2 (`cdc-cms-service`)
- **Message Broker / RPC**: NATS (Subject: `cdc.cmd.scan-array`, queue group: `cdc-worker-group`)
- **Database**: PostgreSQL (JSONB columns: `_raw_data`)
- **ORM / SQL**: GORM, raw SQL
- **Observability**: OpenTelemetry, SigNoz, Zap Logger
