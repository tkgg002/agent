# 05_progress — Fix CDC Testing Residual (APPEND-ONLY §11)

## Entry 1 — 2026-06-01 — Workspace bootstrap
- Tạo 00_context, 01_requirements, 02_plan, 08_tasks, 05_progress.
- Carry-over từ audit-rerun: G5/G8/G3/G1 đã PASS Wave 1.
- Bắt đầu Wave 2 với G4-RES.


## Entry 2 — 2026-06-01 — G4-RES DONE
- File: `data-hub/centralized-data-service/scripts/load_test_cdc.js` (+93 LOC, NEW)
- Pattern: xk6-sql driver postgres, 5 VUs × 30s, poll shadow 200ms until 5s.
- Custom metrics: cdc_e2e_latency_ms (Trend), cdc_e2e_success (Rate), cdc_e2e_timeouts (Counter).
- Thresholds: success ≥99%, p95 < 3s.
- Build steps documented inline in header comment (xk6 install + build).
- Verify: `node --check load_test_cdc.js` → SYNTAX_OK.
- Minimal impact: 1 file, no Makefile, no extra README. §6.


## Entry 3 — 2026-06-01 — G2-RES DONE
- File NEW: `internal/service/wal_monitor.go` (+201 LOC)
- File NEW: `internal/service/wal_monitor_test.go` (+93 LOC)
- File EDIT: `pkgs/metrics/prometheus.go` (+10 LOC metric WALSnapshotResumeTotal)
- Pattern: pure `evaluate()` step + dedupe map + inactive grace map.
- NATS subject: `cdc.snapshot.resume`, payload {slot_name, reason, lag_bytes}.
- Reasons: inactive (after 5min grace) | lag_exceeded (≥1GiB default).
- Verify:
  - `go build ./internal/... ./pkgs/...` EXIT=0
  - `go vet ./internal/service/ ./pkgs/metrics/` EXIT=0 (pre-existing sonyflake noise only)
  - `go test ./internal/service/ -run "TestEvaluate_|TestPublish_"` → 4/4 PASS in 0.746s
- Minimal: 1 service + 1 test file + 1 metric, không thay đổi orchestrator (wiring để follow-up).


## Entry 4 — 2026-06-01 — G7-RES DONE
- File EDIT: `internal/handler/kafka_consumer.go` (+24 LOC)
  - adaptiveBatcher: thêm `destHealth func() bool`
  - adjust(): early-return clamp to baseBatchSize khi destHealth=false (bypass time gate)
  - KafkaConsumer: thêm `SetDestHealthCheck(f)`
- File EDIT: `pkgs/metrics/prometheus.go` (+11 LOC metric DestThrottledTotal)
- File NEW: `internal/handler/adaptive_batcher_test.go` (+82 LOC, 4 cases)
- Verify:
  - `go build ./internal/handler/ ./pkgs/metrics/` EXIT=0
  - `go test ./internal/handler/ -run TestAdaptiveBatcher` → 4/4 PASS in 0.667s
- Minimal: thêm 1 field + 1 if-block + 1 setter, không refactor.

## Entry 5 — 2026-06-01 — G6-RES DONE
- File REWRITE: `scripts/chaos_network.sh` (26 → 78 LOC)
- Replace `sudo iptables` (root required, không CI-reproducible)
  bằng `docker run gaiaadm/pumba netem`.
- Modes: loss (20%), delay (200ms), rate (1mbit).
- Tự verify: snapshot lag trước/sau + acceptance gate (after < 2× before).
- Pre-flight: kiểm tra docker + target container tồn tại.
- Verify: `bash -n` SYNTAX_OK; chmod +x done.


## Entry 6 — 2026-06-01 — Suite re-run + report
- Full re-test sau khi đóng 8 gap:
  - centralized-data-service: handler/service/database PASS (0.762/0.536/1.214s)
  - cdc-cms-service: api/commands PASS (1.005/0.477s)
- go vet: chỉ sonyflake noise pre-existing, zero warning từ patches.
- File report: `report_fix_residual_2026-06-01.md` đã tạo.
- TaskList: 8/8 completed.
- Áp §6 Simplicity: không thêm Makefile, không thêm README phụ trợ.

