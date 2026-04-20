# Tasks: Observability (FINAL)

> Date: 2026-04-16
> Merged: base + user deep requirements

## Phase 1: System Health API + FE
- [ ] T1: CMS `system_health_handler.go` (parallel HTTP polls + DB queries + alerts compute)
- [ ] T2: CMS SystemConfig (workerUrl, kafkaConnectUrl, natsMonitorUrl)
- [ ] T3: CMS route `/api/system/health` + server init
- [ ] T4: FE `SystemHealth.tsx` (6 sections: heartbeat, pipeline, recon table with drift%, latency chart P50/P95/P99, alerts, recent events)
- [ ] T5: FE route + menu + auto-refresh 30s

## Phase 2: Activity Log Enhancement
- [ ] T6: Kafka consumer `eventBatchLogger` (per topic, per 100 msgs or 5s, format: processed/success/failed/duration)
- [ ] T7: Command handler `publishResult` → Activity Log (cmd-{name}, table, rows, error)

## Phase 3: E2E Latency + Metrics
- [ ] T8: Prometheus histogram `cdc_e2e_latency_seconds` (custom buckets 0.1→60s)
- [ ] T9: Kafka consumer: observe T2-T1 latency after successful upsert
- [ ] T10: System Health API: compute P50/P95/P99 from histogram or activity_log

## Phase 4: Debezium Deep Health
- [ ] T11: Poll Kafka Connect API → bóc `trace` lỗi khi task FAILED (truncate 500 chars)
- [ ] T12: FE: hiện trace lỗi + Restart Connector button

## Phase 5: OTel Log Persistence
- [ ] T13: OTel zap core bridge → logs gửi SigNoz gRPC (with trace_id)
- [ ] T14: Kafka consumer: create span per message → trace context propagate

## Phase 6: Verify
- [ ] T15: Flow testing theo `11_flow_testing_observability.md` (7 flows)
