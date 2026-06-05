# 08_tasks_phase_p1 — Checklist Muscle Phase P1

> Reference: `03_implementation_phase_p1.md`. Dependency: P0 done.

## G-5 — Failover smoke test (4h)
- [ ] Tạo NEW `centralized-data-service/scripts/smoke_failover.sh`:
  - Bước 1: Insert 10k document Mongo (script seed).
  - Bước 2: Đợi worker consume ~50% (check `cdc_events_consumed_total`).
  - Bước 3: `kill -9 <worker_pid>`.
  - Bước 4: Restart worker.
  - Bước 5: Đợi consume hết → query shadow PG `SELECT COUNT(*) FROM shadow.users` == 10000.
  - Bước 6: Query `SELECT COUNT(DISTINCT _id)` == 10000 (no dup).
- [ ] Tạo `.github/workflows/smoke-failover.yml` chạy script trên runner (PG + Mongo + Kafka container).
- [ ] Verify: `bash scripts/smoke_failover.sh` exit 0.

## G-6 — WAL slot alert (4h)
- [ ] Tạo `deployments/prometheus/alerts/wal_slot.yml`:
  - `ReplicationSlotLagHigh`: `pg_replication_slot_current_wal_lsn_bytes - pg_replication_slot_confirmed_flush_lsn_bytes > 1e9 for 10m`.
  - `ReplicationSlotInactive`: `pg_replication_slot_active == 0 for 5m`.
- [ ] Deploy `postgres-exporter` K8s manifest với env `DATA_SOURCE_NAME`.
- [ ] Tạo `docs/runbooks/wal-slot-expire.md` với escalation flow + `pg_drop_replication_slot` command.
- [ ] Verify: `promtool test rules wal_slot.yml` PASS.

## G-7 — pprof + goleak (2h)
- [ ] Sửa `centralized-data-service/cmd/worker/main.go`: thêm `_ "net/http/pprof"` import + `go http.ListenAndServe(cfg.Debug.PprofAddr, nil)`.
- [ ] Config knob `Debug.PprofAddr` default `:6060` (chỉ enable khi `Debug.PprofEnabled=true`).
- [ ] 3 test package thêm `TestMain` với `goleak.VerifyTestMain(m, goleak.IgnoreTopFunction("github.com/segmentio/kafka-go.(*Reader).run"), goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"))`:
  - `internal/handler/`
  - `internal/service/`
  - `internal/sinkworker/`
- [ ] Verify: `go test -race ./internal/handler/...` PASS no leak; `curl :6060/debug/pprof/goroutine?debug=1` returns.

## G-8 — Ordering test (2h)
- [ ] Tạo NEW `internal/service/schema_adapter_ordering_test.go`:
  - `TestEventOrdering_OlderTsIgnored`: Insert ts=1000 → Update ts=3000 → Update ts=2000 (assert REJECTED) → Delete ts=4000.
  - `TestEventOrdering_HashTiebreaker`: 2 event cùng ts khác hash → assert ưu tiên theo hash deterministic.
- [ ] Verify: `go test ./internal/service -run TestEventOrdering -v` PASS.

## G-9 — Drift E2E test (8h)
- [ ] Tạo NEW `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` với build tag `//go:build integration`.
- [ ] testcontainers spin: postgres + nats.
- [ ] Step flow:
  1. Spin container.
  2. Apply migrations.
  3. Insert `schema_drift_proposal` row status=`pending`.
  4. Subscribe NATS subject `cdc.schema.applied.*`.
  5. Call `ApproveSchemaProposalHandler.Handle(ctx, cmd)`.
  6. Assert ALTER TABLE applied (query `information_schema.columns`).
  7. Assert mapping_rule row inserted.
  8. Assert NATS event received within 5s.
- [ ] Verify: `go test -tags=integration ./internal/app/commands/ -run TestApproveSchemaProposalE2E -v` PASS.

## Post-phase
- [ ] Build/vet/test all services PASS.
- [ ] /security-agent scan PASS.
- [ ] APPEND `05_progress.md`.
- [ ] Composite score → kỳ vọng 51/64.
