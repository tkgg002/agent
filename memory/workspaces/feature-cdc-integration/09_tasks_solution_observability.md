# Solution: Observability

> Date: 2026-04-16
> Phase: observability

## Files mới

### CMS
| File | Purpose |
|:-----|:--------|
| `internal/api/system_health_handler.go` | Aggregate health từ 7 components + pipeline + recon |

### FE
| File | Purpose |
|:-----|:--------|
| `src/pages/SystemHealth.tsx` | 7 component cards + pipeline metrics + alerts + recent events |

## Files sửa

### CMS
| File | Thay đổi |
|:-----|:---------|
| `config/config.go` | +SystemConfig (workerUrl, kafkaConnectUrl, natsMonitorUrl) |
| `config/config-local.yml` | +system section |
| `internal/router/router.go` | Register /api/system/health |
| `internal/server/server.go` | Init SystemHealthHandler |

### Worker
| File | Thay đổi |
|:-----|:---------|
| `internal/handler/kafka_consumer.go` | Batch Activity Log (eventBatchLogger) |
| `internal/handler/command_handler.go` | publishResult → Activity Log |
| `pkgs/metrics/prometheus.go` | +E2ELatency histogram |

### FE
| File | Thay đổi |
|:-----|:---------|
| `src/App.tsx` | Route + menu /system-health |

## Dependencies
- Không cần thêm Go dependencies
- HTTP client calls to: Worker health, Kafka Connect API, NATS monitoring

## Execution order
```
T1-T3 (API) → T4-T5 (FE) → T6-T8 (Worker enhancement) → T9-T12 (verify)
```
