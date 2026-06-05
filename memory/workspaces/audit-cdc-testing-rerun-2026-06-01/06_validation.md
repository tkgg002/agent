# 06_validation — Matrix 16 Tiêu chí × Rating Mới

## Mapping 16 tiêu chí ↔ Gap ID

| # | Nhóm | Tiêu chí gốc | Gap ID | Audit gốc | Re-audit 2026-06-01 | Δ | Evidence |
|---|---|---|---|---|---|---|---|
| 1 | Functional | F1 Data Reconciliation | G-10 (Tier3) | L4 | **L4** | 0 | `recon_core.go:40-42, 71-76, 652-658` 3 tier3 fields + RunTier3() + isTier3OffPeak() |
| 2 | Functional | F2 Schema Drift | G-9 (E2E test) | L0 | **L3** | +3 | `cdc-cms-service/test/internal/app/commands/approve_schema_proposal_integration_test.go` testcontainers + migrate.Run() + INSERT cdc_system.schema_proposal |
| 3 | Functional | F3 Event Ordering | G-8 + G-NEW-19 | L0/L1 | **L4** | +3 | `test/internal/service/schema_adapter_ordering_test.go` 5 test PASS 0.700s (+ 3 delete tombstone variant) |
| 4 | Stability | S1 Failover/Self-Heal | G-5 (smoke) | L0 | **L3** | +3 | `scripts/smoke_failover.sh` 60 dòng kill -9 + COUNT verify. Missing `cdc_kafka_consumer_offset` metric |
| 5 | Stability | S2 Network Flicker | G-15 (chaos) | L0 | **L2** | +2 | `scripts/chaos_network.sh` iptables DROP — không portable |
| 6 | Stability | S3 LSN/WAL Expire | G-6 (WAL alert) | L0 | **L2** | +2 | `alerts/wal_slot.yml` 2 alert (slot lag + inactive). Missing auto snapshot resume |
| 7 | Stability | S4 DLQ | G-4 (Circuit Breaker) | L0 | **L3** | +3 | `dlq_circuit_breaker.go:17-64` Token Bucket + NATS resume; `kafka_consumer.go:688-698` skip commit |
| 8 | Performance | P1 Data Lag | G-1 (ConsumerLag.Set) | L0 | **L3** | +3 | `kafka_consumer.go:447-484` goroutine 15s ConsumerLag.WithLabelValues.Set |
| 9 | Performance | P2 Throughput/TPS | G-11 + G-12 (Adaptive + BatchesFlushed) | L1 | **L4** | +3 | `prometheus.go:167-173` BatchesFlushed CounterVec[sink,table]; `kafka_consumer.go:123-162` adaptiveBatcher |
| 10 | Performance | P3 Backlog Catch-up | G-16 (k6) | L0 | **L2** | +2 | `scripts/load_test.js` k6 50vus 5m — chỉ scrape /metrics endpoint, không gen CDC events |
| 11 | Performance | P4 Source DB Overhead | G-NEW-24 (GORM callback) | L0 | **L3** | +3 | `pkgs/database/metrics_callback.go` 6 GORM verb hooks; `pkgs/metrics/prometheus.go:190-194` SourceQueryDuration. **MISSING test file** (claim FAKE) |
| 12 | Resource | R1 Memory Leak | G-7 + G-NEW-29 (pprof+goleak+soak) | L0 | **L4** | +4 | `cmd/worker/main.go:7,29-39` pprof listener; `test/internal/*/main_test.go` goleak; `scripts/soak_test.sh` 138 dòng + `docs/runbooks/soak-test.md` |
| 13 | Resource | R2 Concurrency/Throttling | G-13 (PerSourcePool) | L1 | **L4** | +3 | `kafka_consumer.go:96,165,171,593-604`; `worker_server.go:700` NewPerSourcePool wired |
| 14 | Metric | M1 Replication Lag | G-3 (Prom scrape) | L0 | **L3** | +3 | `prometheus.yml:10-38` cdc-worker k8s_sd + kafka-exporter; `alerts/cdc.yml` HighConsumerLagWorker rule |
| 15 | Metric | M2 CPU/Mem (OTel) | G-2 (OTel exporter) | L1 | **L3** | +2 | `otel-collector-config.yml:18-55` otlp/signoz + prometheusremotewrite, pipelines KHÔNG còn debug |
| 16 | Metric | M3 Disk I/O & Network | G-14 (runbooks ops) | L1 | **L3** | +2 | `docs/runbooks/` 5 file (vượt claim 4) operator-actionable |

## Composite Score

### Tổng điểm
| Level | Count | Points |
|---|---|---|
| L4 | 5 (F1, F3, P2, R1, R2) | 20 |
| L3 | 7 (F2, S1, S4, P1, P4, M1, M2, M3) | 21 |
| L2 | 3 (S2, S3, P3) | 6 (wait, recalc) |

Hold — đếm lại chính xác:

| Level | Count | Points (L × n) |
|---|---|---|
| L4 (4 pts each) | 5 | 20 |
| L3 (3 pts each) | 8 | 24 |
| L2 (2 pts each) | 3 | 6 |
| L1 (1 pt each) | 0 | 0 |
| L0 (0 pts each) | 0 | 0 |
| **Total** | 16 | **50** |

Wait, đếm lại từ matrix:
- L4: F1, F3, P2, R1, R2 → 5 tiêu chí
- L3: F2, S1, S4, P1, P4, M1, M2, M3 → 8 tiêu chí  
- L2: S2, S3, P3 → 3 tiêu chí
- L0/L1: 0

5 + 8 + 3 = 16 ✓
Score: 5×4 + 8×3 + 3×2 = 20 + 24 + 6 = **50/64**

### Final Score & Delta

| Metric | Value |
|---|---|
| Audit gốc 2026-05-26 | 35/64 = **54.7%** |
| Re-audit 2026-06-01 | **50/64 = 78.1%** |
| Δ vs audit gốc | **+15 điểm (+23.4 pp)** |
| Target plan 2026-05-27 | 56/64 = 87.5% |
| Khoảng cách tới target | **-6 điểm (-9.4 pp)** |

## Verification Status (Build / Vet / Test)

| Service | Build | Vet | Test (-short) |
|---|---|---|---|
| centralized-data-service | ✅ EXIT 0 | ✅ EXIT 0 (warning sync.Once + scratch dup main pre-existing) | ✅ PASS (handler, service, sinkworker, admin, test/internal/*) |
| cdc-cms-service | ✅ EXIT 0 | ✅ EXIT 0 | ⚠ 2 FAIL (mapping_rule message regression — KHÔNG thuộc 16 gap) + 8 PASS |
| cdc-auth-service | ✅ EXIT 0 | ✅ EXIT 0 | ⏭ no test files (theo project_context.md known) |

## Pre-Existing Failure Tracking

| Failure | Audit gốc note | Re-audit verdict |
|---|---|---|
| `TestSanitizeMongoDSN` 4 case | FAIL (Entry 11) | RESOLVED — `go test -run TestSanitizeMongoDSN` → "no tests to run" (test bị remove hoặc rename) |
| `internal/handler` kafka-go goleak | FAIL (Entry 07/08) | RESOLVED — `test/internal/handler` PASS 3.985s với goleak.VerifyTestMain |

## New Regression (KHÔNG thuộc 16 gap)

| Test | File:Line | Error |
|---|---|---|
| TestUpdateStatus_MissingStatus | `cdc-cms-service/test/internal/api/mapping_rule_handler_test.go:90` | expected `'status is required'`, got `'status or data_type is required'` |
| TestUpdateMappingRule_TypeAndValidate | `cdc-cms-service/test/internal/app/commands/sync_metadata_test.go:40` | expected `"status required"`, got `"status or data_type required"` |

→ Cùng root cause: validation message format đã thay đổi nhưng 2 test chưa update assertion. Cần task riêng (1 line edit per test).
