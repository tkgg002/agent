# Implementation: Debezium + NATS JetStream (Phase 2)

> Date: 2026-04-14
> Phase: phase_2_debezium

## Architecture hiện tại

```
MongoDB (rs0, port 17017)
    ↓ Change Stream (realtime < 1s)
Debezium Server 2.5 (container gpay-debezium, port 18083)
    ↓ NATS JetStream Sink (auth: debezium/debezium_secret_2026)
NATS JetStream (container gpay-nats, port 14222)
    ↓ Stream: DebeziumStream, subjects: cdc.goopay.>
CDC Worker (10 goroutines, pull subscribe)
    ↓ EventHandler → DynamicMapper → BatchBuffer
PostgreSQL (port 5432)
```

## Config files

| File | Nội dung |
|:-----|:---------|
| `deployments/debezium/application.properties` | Debezium Server config: MongoDB source + NATS sink + auth |
| `deployments/nats/nats-server.conf` | NATS ACL: 4 users (cdc_worker, cms_service, auth_service, debezium) |
| `docker-compose.yml` | Container gpay-debezium (image quay.io/debezium/server:2.5) |

## NATS JetStream config

- Stream name: `DebeziumStream` (auto-created by Debezium)
- Subjects: `cdc.goopay.>`
- Storage: memory (default Debezium config)
- Retention: limits
- Consumer: `cdc-worker-group` (durable, pull-based, 30s ack wait, max 5 delivery)

## Debezium config

- Source: MongoDB (`change_streams_update_full`)
- Database filter: `payment-bill-service`
- Snapshot mode: `initial` (first run snapshots all data, then streams changes)
- Offset storage: file-based (`/debezium/data/offsets.dat`)
- Format: JSON schemaless (no Avro, no schema registry)

## NATS ACL

| User | Publish | Subscribe | Purpose |
|:-----|:--------|:----------|:--------|
| cdc_worker | `cdc.>`, `schema.config.reload` | `cdc.>`, `schema.config.reload` | Worker process events |
| cms_service | `cdc.cmd.>`, `schema.config.reload` | `cdc.cmd.>`, `schema.config.reload`, `_INBOX.>` | CMS dispatch commands |
| debezium | `cdc.>`, `$JS.API.>` | `cdc.>`, `$JS.API.>`, `_INBOX.>` | Debezium publish CDC events + manage streams |
| auth_service | (none) | (none) | Minimal |

## Verified E2E (2026-04-14)

1. Insert document vào MongoDB `payment-bill-service.payment-bills`
2. Debezium capture change event → log "2 records sent"
3. NATS JetStream nhận message trên subject `cdc.goopay.payment-bill-service.payment-bills`
4. CDC Worker consumer pool pull message → EventHandler → DynamicMapper → PostgreSQL
