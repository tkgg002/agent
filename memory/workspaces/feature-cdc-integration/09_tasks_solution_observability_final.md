# Solution: Observability (FINAL)

> Date: 2026-04-16

## Files mới

### CMS
| File | Lines est. | Purpose |
|:-----|:-----------|:--------|
| `internal/api/system_health_handler.go` | 300 | Parallel polls + DB queries + alerts + response aggregate |

### FE
| File | Lines est. | Purpose |
|:-----|:-----------|:--------|
| `src/pages/SystemHealth.tsx` | 350 | 6-section dashboard + latency chart + auto-refresh |

## Files sửa

### CMS
| File | Thay đổi |
|:-----|:---------|
| `config/config.go` | +SystemConfig |
| `config/config-local.yml` | +system section |
| `internal/router/router.go` | Route |
| `internal/server/server.go` | Init handler |

### Worker
| File | Thay đổi |
|:-----|:---------|
| `internal/handler/kafka_consumer.go` | +eventBatchLogger + E2E latency measure + span per message |
| `internal/handler/command_handler.go` | publishResult → Activity Log |
| `pkgs/metrics/prometheus.go` | +E2ELatency histogram |
| `pkgs/observability/otel.go` | +Log exporter gRPC |
| `cmd/worker/main.go` | OTel zap bridge |
| `config/config.go` | +grpcEndpoint in OtelConfig |
| `config/config-local.yml` | +otel.grpcEndpoint |

## Dependencies mới
```
go.opentelemetry.io/contrib/bridges/otelzap
go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc
go.opentelemetry.io/otel/sdk/log
```

## Execution order
```
T1-T3 (API) → T4-T5 (FE) → T6-T7 (Activity Log) → T8-T10 (Latency) → T11-T12 (Debezium) → T13-T14 (OTel) → T15 (verify)
```
