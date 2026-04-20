# Tasks: Observability

> Date: 2026-04-16
> Phase: observability

## Phase 1: System Health API + FE
- [ ] T1: CMS `system_health_handler.go` — aggregate health from Worker/Kafka/Debezium/NATS/Postgres/Redis/Airbyte
- [ ] T2: CMS config (workerUrl, kafkaConnectUrl, natsMonitorUrl)
- [ ] T3: CMS route `/api/system/health`
- [ ] T4: FE `SystemHealth.tsx` — 7 component cards + pipeline metrics + alerts + recent events
- [ ] T5: FE route + menu `/system-health`

## Phase 2: Activity Log Enhancement
- [ ] T6: Kafka consumer batch Activity Log (per 30s flush)
- [ ] T7: Command handler `publishResult` → Activity Log
- [ ] T8: E2E latency Prometheus histogram + measurement in Kafka consumer

## Phase 3: Verify
- [ ] T9: System Health page hiện TẤT CẢ components UP/DOWN
- [ ] T10: Kafka consumer events visible trong Activity Log
- [ ] T11: Command handler results visible trong Activity Log
- [ ] T12: Auto-refresh 30 giây hoạt động
