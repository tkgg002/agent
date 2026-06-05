# 06_validation — Acceptance & Verify Commands

## Verify pattern chung cho mọi gap
1. **Build**: `go build ./...` (centralized-data-service + cdc-cms-service)
2. **Vet**: `go vet ./...`
3. **Unit test**: `go test ./...`
4. **Lint FE**: `pnpm lint && pnpm typecheck` (cdc-cms-web)
5. **E2E (nếu có testcontainers)**: `go test -tags=integration ./...`

---

## Phase P0

| Gap | Acceptance | Verify command |
|---|---|---|
| G-1 | `cdc_kafka_consumer_lag` xuất hiện trên /metrics với non-zero value | `curl -s :9090/metrics \| grep cdc_kafka_consumer_lag` |
| G-2 | OTel Collector receive trace + push SigNoz/Prom | `docker logs otel-collector \| grep "TracesExporter"` + check SigNoz UI |
| G-3 | Prom server scrape 5 job thành công | `curl -s :9091/api/v1/targets \| jq '.data.activeTargets[] \| .health'` (5x "up") |
| G-4 | DLQ spike → pipeline paused (`cdc_pipeline_paused_total > 0`) | Inject 500 DLQ msg/s test → assert metric increment + NATS event `cdc.alert.pipeline-paused` published |

---

## Phase P1

| Gap | Acceptance | Verify command |
|---|---|---|
| G-5 | Insert 10k Mongo doc, kill -9 worker, restart → shadow count == 10k, không dup | `bash scripts/smoke_failover.sh` returns exit 0 |
| G-6 | Stop replication consumer 10 phút → alert `ReplicationSlotLagHigh` fire | `promtool test rules deployments/prometheus/alerts/wal_slot.yml` |
| G-7 | Test `go test -race ./internal/handler/...` không leak goroutine | `goleak.VerifyTestMain` không fail; pprof goroutine count stable < 200 sau 1h soak |
| G-8 | TestEventOrdering_OlderTsIgnored PASS | `go test ./internal/service/ -run TestEventOrdering -v` |
| G-9 | E2E flow approve drift → ALTER TABLE + mapping_rule + NATS event | `go test -tags=integration ./internal/app/commands/ -run TestApproveSchemaProposalE2E -v` |

---

## Phase P2

| Gap | Acceptance | Verify command |
|---|---|---|
| G-10 | Config Tier3OffPeakStart=22 → tier-3 chạy 22:00-05:00 | `go test ./internal/service -run TestTier3OffPeak` |
| G-11 | `cdc_batches_flushed_total` increment sau mỗi flush | `curl :9090/metrics \| grep cdc_batches_flushed_total` |
| G-12 | Lag > 50k → batchSize x2 trong 30s | Synthetic load test, assert `cdc_burst_mode_active == 1` |
| G-13 | Per-source pool saturation visible | `curl :9090/metrics \| grep cdc_per_source_pool_in_use` |
| G-14 | 4 runbook tồn tại + link trong alert annotation | `ls docs/runbooks/` (4 file MD) |
| G-15 | Chaos network 10 phút → recon drift = 0 sau 30 phút | `bash scripts/chaos_network.sh` + check Grafana `cdc_recon_drift_count` |
| G-16 | k6 P99 e2e latency < 5s @ 1000 TPS | `k6 run scripts/load_test.js` (threshold pass) |

---

## Phase UI

| Item | Acceptance | Verify command |
|---|---|---|
| Migration | Bảng `cdc_system.qa_gap_state` + `qa_criterion_rating` tồn tại với 16 + 16 row seed | `psql -c "SELECT COUNT(*) FROM cdc_system.qa_gap_state"` returns 16 |
| API /audit/qa-summary | Return composite score + 16 criterion + gap counts | `curl -s :8080/api/v1/admin/audit/qa-summary -H "Authorization: Bearer $ADMIN"` |
| API /audit/gaps | Filter `?priority=P0&status=open` works | `curl -s ".../gaps?priority=P0"` returns 4 row |
| API /audit/metric-health | Return 4 metric với status | `curl -s ".../metric-health"` returns consumer_lag/e2e/dlq/recon |
| AuditPage UI | Render 4 section (Composite + Metric Cards + Rating Matrix + Gap Tabs) không lỗi console | Browser DevTools, navigate `/audit`, check Network tab 3 API call 200 |
| Refresh interval | qa-summary refetch mỗi 30s, metric-health 15s | Network tab observe pattern |

---

## Composite score recalculate (chứng minh delta)

| Phase | Trước | Delta | Sau | % |
|---|---|---|---|---|
| Baseline (audit cũ) | — | — | 35/64 | 54.7% |
| P0 done | 35 | +9 | 44/64 | 68.75% |
| P0+P1 done | 44 | +7 | 51/64 | 79.7% |
| P0+P1+P2 done | 51 | +5 | 56/64 | 87.5% |
| + UI (no score impact, visibility only) | 56 | 0 | 56/64 | 87.5% |

---

## Definition of Done global
- [ ] Tất cả phase được approve có verify command PASS evidence (log/screenshot/curl output).
- [ ] Service `centralized-data-service` build + vet + test PASS.
- [ ] Service `cdc-cms-service` build + vet + test PASS.
- [ ] Service `cdc-cms-web` lint + typecheck + build PASS.
- [ ] Composite score new ≥ target phase.
- [ ] /security-agent scan PASS (§8).
