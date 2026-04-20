# Tasks: Phase 2 Execution

> Date: 2026-04-14
> Reference: 03_implementation_phase_2.md
> Execution order: Step 1 → Step 7

## Step 1: Dynamic Mapper Full ✅
- [x] T1.1: `LoadRules()` — delegates to RegistryService
- [x] T1.2: `MapData()` — type conversion + nested fields + enriched routing + raw JSON
- [x] T1.3: `BuildUpsertQuery()` — dynamic INSERT...ON CONFLICT
- [x] T1.4: `convertType()` — INT, FLOAT, BOOL, TIMESTAMP, JSONB, TEXT + MongoDB $date
- [x] T1.5: Config reload via RegistryService (NATS subscription)
- [x] T1.6: EventHandler uses DynamicMapper.MapData() instead of static loop
- [x] T1.7: Unit tests — 8 tests pass (convertType, getNestedField)
- [x] T1.8: Build OK, all tests pass

## Step 2: Enrichment + DLQ ✅
- [x] T2.1: `enrichment_service.go` — Enrich() with function routing
- [x] T2.2: `dlq_handler.go` — HandleWithRetry (3x exponential backoff) + sendToDLQ + ReplayDLQ
- [x] T2.3: Build OK

## Step 3: Debezium Config ✅
- [x] T3.1: `deployments/debezium/mongodb-connector.json` — MongoDB CDC config
- [ ] T3.2: MySQL connector (khi cần)
- [ ] T3.3: Verify CDC events → NATS (cần Debezium server chạy)
- [ ] T3.4: E2E verify (cần Debezium server chạy)

## Step 4: Event Bridge ✅
- [x] T4.1+T4.2: `event_bridge.go` — StartTriggerListener (LISTEN/NOTIFY)
- [x] T4.3: StartPoller (non-critical tables, interval-based)
- [x] T4.4: MoleculerEvent CloudEvents format + NATS publish
- [x] Build OK

## Step 5: Data Reconciliation ✅
- [x] T5.1: reconciliation_service.go (CMS) — auto-heal mismatches
- [x] T5.2: Schedule via cdc_worker_schedule + Activity Manager

## Step 6: Production Scaling ✅
- [x] T6.1: `deployments/k8s/cdc-worker-deployment.yaml` — replicas=5, HPA cpu 70%, resource limits
- [x] T6.2: Load test PASS — 50K rows at 5,640 rows/sec (target 5K)

## Step 7: Integration Testing ✅
- [x] T7.1: E2E bridge+transform PASS trên DB thực (100 rows, idempotency verified)
- [x] T7.2: DLQ tests PASS (serialize, retry success/fail)

## Remaining (cần Debezium server)
- [ ] T3.3: Verify Debezium CDC events → NATS (cần deploy Debezium)
- [ ] T3.4: Worker processes Debezium events E2E
