# 05_progress — Re-Audit Progress Log (APPEND ONLY)

> Quy luật §7 / §11: TUYỆT ĐỐI không sửa entry cũ. Sai thì ghi entry mới "Sai - Revert".

## Entry 01 — 2026-06-01 [Muscle:claude-opus-4-7] Context loaded
- Đọc CLAUDE.md, GEMINI.md, project_context.md (CDC system overview), active_plans.md (audit-cdc-qa-process-2026-05-26 và plan-cdc-qa-gap-fix-2026-05-27 — status DONE), tech_stack.md (3 service Go + 1 FE TS).
- Identify code root: `/Users/trainguyen/Documents/work/data-hub/{centralized-data-service, cdc-cms-service, cdc-auth-service, cdc-cms-web}`.
- Đọc 3 execution report tại workspace plan-cdc-qa-gap-fix-2026-05-27.

## Entry 02 — 2026-06-01 [Muscle:claude-opus-4-7] Workspace + TaskList created
- Tạo workspace `audit-cdc-testing-rerun-2026-06-01`.
- TaskList 8 task: 6 gap-verify (G1-4 P0, G5-9 P1, G10-16 P2+NEW) + 1 build/test verify + 1 report.
- Doc 00_context, 01_requirements, 02_plan, 05_progress khởi tạo.

## Entry 03 — 2026-06-01 [Muscle:claude-opus-4-7] Parallel verification kicked off
- Spawn 3 Explore subagent parallel:
  - Agent C1: G-1..G-4 (P0 critical).
  - Agent C2: G-5..G-9 (P1).
  - Agent C3: G-10..G-16 + G-NEW-19/24/29 (P2 + remaining).
- Mỗi agent có prompt yêu cầu evidence file:line, KHÔNG TIN claim, mở file thật.

## Entry 04 — 2026-06-01 [Muscle:claude-opus-4-7] Agent C1 (P0) results
- G-1 ConsumerLag.Set: **L3 FIXED**. Evidence: `kafka_consumer.go:447-484` goroutine 15s + `.WithLabelValues(stats.Topic, stats.Partition).Set(float64(stats.Lag))`. Risk: chỉ scrape `readers[0]`.
- G-2 OTel exporter: **L3 FIXED**. Evidence: `otel-collector-config.yml:18-55` đủ 3 exporter (debug + otlp/signoz + prometheusremotewrite); pipelines KHÔNG còn ref debug. Env fallback đúng.
- G-3 Prometheus scrape: **L3 FIXED**. Evidence: `prometheus.yml:10-38` có 3 job (cdc-worker k8s sd, kafka-exporter, cdc-cms-service); `alerts/cdc.yml:4-25` 4 rule với expr/for hợp lý.
- G-4 DLQ Circuit Breaker: **L3 FIXED**. Evidence: `dlq_circuit_breaker.go:17-64` Token Bucket rate.Limiter + NATS `cdc.pipeline.resume`; `kafka_consumer.go:688-698` check IsPaused → skip CommitMessages. Risk: chỉ paused/resumed, không có Open/HalfOpen/Closed; resume thủ công qua NATS.

## Entry 05 — 2026-06-01 [Muscle:claude-opus-4-7] Agent C2 (P1) results
- G-5 Failover smoke: **L3 PARTIAL**. Evidence: `scripts/smoke_failover.sh` 60 dòng kill -9 + docker stop + COUNT verify. Gap: metric `cdc_kafka_consumer_offset` KHÔNG tồn tại → không verify resume position; script dùng `./bin/worker` local thay container.
- G-6 WAL: **L2 PARTIAL**. Evidence: `alerts/wal_slot.yml` có `ReplicationSlotLagHigh` + `ReplicationSlotInactive`. Gap: snapshot auto-resume khi WAL expire **HOÀN TOÀN VẮNG MẶT** — alert chỉ thông báo, không có automation.
- G-7 pprof+goleak: **L4 FIXED**. Evidence: `cmd/worker/main.go:7` `_ "net/http/pprof"`; conditional listener tại `:29-39`; config `config-local.yml:103-105` `debug.pprofEnabled: true`; `goleak.VerifyTestMain` trong 4 package test/internal/{handler,service,sinkworker,admin}/main_test.go.
- G-8 Event Ordering: **L4 FIXED**. Evidence: `test/internal/service/schema_adapter_ordering_test.go` 227 dòng, 5 test thực: PASS 0.700s. Tests dùng in-memory DB, gọi BuildUpsertSQL thực, assert OCC reject qua RowsAffected==0.
- G-9 Schema Drift E2E: **L3 PARTIAL**. Evidence: file thực ở `cdc-cms-service/test/internal/app/commands/approve_schema_proposal_integration_test.go` (KHÔNG phải path claim `internal/app/commands/_e2e_test.go`). Import testcontainers, gọi migrate.Run(), seed FK rows. Build tag `//go:build integration`. Path discrepancy là documentation bug.

## Entry 06 — 2026-06-01 [Muscle:claude-opus-4-7] Agent C3 (P2 + NEW) results
- G-10 Tier3 config: **L4 FIXED**. Evidence: `recon_core.go:40-42, 71-76, 652-658` có Tier3MaxDocsPerRun + OffPeakStart/End + RunTier3() + isTier3OffPeak() guard.
- G-11 BatchesFlushed CounterVec: **L4 FIXED**. Evidence: `prometheus.go:167-173` NewCounterVec[sink,table]; `batch_buffer.go:206` `.WithLabelValues("postgres", records[0].TableName).Inc()`. Audit gốc G-11 SAI claim — đã re-verify ĐÚNG.
- G-12 Adaptive batch: **L4 FIXED**. Evidence: `kafka_consumer.go:123-162, 208-228` adaptiveBatcher struct + adjust() tăng size khi lag>threshold + BurstModeActive metric; wired qua `worker_server.go:717-733`. Note: logic là burst-up không phải throttle-down.
- G-13 PerSourcePool: **L4 FIXED**. Evidence: `kafka_consumer.go:96,165,171,593-604`; `worker_server.go:700` `NewPerSourcePool(s.db, 0)`. Audit gốc G-13 SAI — wired thực tế.
- G-14 Runbooks: **L3 FIXED**. Evidence: `docs/runbooks/` có 5 file (vượt claim 4): pipeline-pause-resume, recon-drift-response, schema-drift-approve-sla, soak-test, wal-slot-expire. Có commands thực không phải template.
- G-15 Chaos: **L2 PARTIAL**. Evidence: `scripts/chaos_network.sh` dùng `iptables -A OUTPUT DROP`. Gap: cần sudo + platform-specific Linux + không chạy được trong container CI tiêu chuẩn.
- G-16 k6: **L2 PARTIAL**. Evidence: `scripts/load_test.js` k6 với 50vus 5m. Gap: target chỉ `http://localhost:8080/metrics` (Prometheus scrape endpoint), KHÔNG generate CDC data, KHÔNG đo throughput pipeline thật.
- G-NEW-19 Delete ordering: **L4 FIXED**. Evidence: `schema_adapter_ordering_test.go:156,180,206` 3 test mới có body logic đầy đủ. PASS chung trong run.
- G-NEW-24 Source DB metric: **L3 PARTIAL FAKE-PARTIAL**. Evidence:
  - `pkgs/database/metrics_callback.go` TỒN TẠI — RegisterQueryMetrics 6 GORM verbs.
  - `pkgs/metrics/prometheus.go:190-194` SourceQueryDuration histogram TỒN TẠI.
  - `pkgs/database/multi.go:178-204` openGorm(dsn, role) call site đúng.
  - **`pkgs/database/metrics_callback_test.go` KHÔNG TỒN TẠI** (claim PASS 2 test → FAKE).
- G-NEW-29 Soak script + runbook: **L4 FIXED**. Evidence: `scripts/soak_test.sh` 138 dòng executable + `docs/runbooks/soak-test.md` acceptance criteria table.

## Entry 07 — 2026-06-01 [Muscle:claude-opus-4-7] Build/Vet/Test verification
- centralized-data-service: `go build ./...` EXIT 0 ✅; `go vet` EXIT 0 ✅ (warning sync.Once copy ở pkgs/idgen/sonyflake.go:77,82 pre-existing + scratch/ dup main pre-existing).
- cdc-cms-service: `go build` EXIT 0 ✅; `go vet` EXIT 0 ✅.
- cdc-auth-service: `go build` EXIT 0 ✅; `go vet` EXIT 0 ✅.
- Test PASS:
  - centralized-data-service/test/internal/{service,handler,sinkworker,admin}: PASS với goleak. G-7 verified.
  - centralized-data-service/internal/{handler,service}: PASS.
  - cdc-cms-service test/internal/{app/queries,infra/http,infra/messaging,infra/observability,middleware,infra/persistence}: PASS.
- Test FAIL (regression mới — KHÔNG thuộc 16 gap audit):
  - `cdc-cms-service/test/internal/api/mapping_rule_handler_test.go:90` TestUpdateStatus_MissingStatus → expected `'status is required'` got `'status or data_type is required'`.
  - `cdc-cms-service/test/internal/app/commands/sync_metadata_test.go:40` TestUpdateMappingRule_TypeAndValidate → expected `"status required"` got `"status or data_type required"`.
- Pre-existing failure check:
  - `TestSanitizeMongoDSN` trong centralized-data-service/internal/service: KHÔNG tồn tại test khi grep `-run "TestSanitizeMongoDSN"` → "no tests to run". Đã được resolve hoặc move/rename.
  - kafka-go goleak: PASS — không còn leak trong test/internal/handler.

## Entry 08 — 2026-06-01 [Muscle:claude-opus-4-7] Synthesis
- Score audit gốc: 35/64 = 54.7%.
- Score re-audit hôm nay (16 tiêu chí gốc): 49/64 = 76.6%.
- Delta: +14 điểm. Vượt 70% nhưng CHƯA đạt target plan 87.5%.
- Verdict: **8 FIXED + 5 PARTIAL + 0 NOT IMPLEMENTED + 1 FAKE-PARTIAL (G-NEW-24 test)** + 2 audit gốc sai (G-11/G-13 đã correction).

## Entry 09 — 2026-06-01 [Muscle:claude-opus-4-7] Files created
- 00_context.md, 01_requirements.md, 02_plan.md, 05_progress.md (this), 06_validation.md, 07_status_report.md, 10_gap_analysis.md, report_audit_testing_rerun_2026-06-01.md.
