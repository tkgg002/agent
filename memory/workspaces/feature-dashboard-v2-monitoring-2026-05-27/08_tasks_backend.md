# 08_tasks_backend — Checklist Muscle thực thi (Backend)

> Mỗi task = 1 PR nhỏ (atomic). DoD bắt buộc trước khi tick.
> File reference dạng `path/to/file.go:LINE` để Muscle nhảy nhanh.

---

## Phase B1 — Streaming metric foundation (~5h)

### T-BE-01 — Extract Kafka transient classifier
- [ ] Tạo `centralized-data-service/pkgs/kafka/classifier.go` (xem `03_implementation_backend.md` Section 1).
- [ ] Tạo `pkgs/kafka/classifier_test.go` — 6+ test case (NotLeader, BrokerUnavail, Timeout, ConnReset, Other, Nil).
- [ ] Cập nhật `internal/handler/kafka_consumer.go:1116-1126` — xóa local `isKafkaTransientError`, import `pkgs/kafka.Classify`.
- **DoD**: `go test ./pkgs/kafka/...` PASS, `go build ./...` PASS, `go vet ./...` PASS.

### T-BE-02 — Thêm 4 metric mới
- [ ] Append 4 metric (`IngestRateMsgsPerSec`, `ConsumeRateMsgsPerSec`, `KafkaTransientErrors`, + ConsumerLag GIỮ NGUYÊN) vào `pkgs/metrics/prometheus.go` (sau dòng 152, trước `)`).
- **DoD**: `go build` PASS; `curl :9090/metrics | grep cdc_ingest_rate_msgs_per_sec` → có line (giá trị 0 ban đầu OK).

### T-BE-03 — Wire `ConsumerLag.Set()` + RateMeter
- [ ] Thêm struct `RateMeter` (xem Section 3 of impl) vào `internal/handler/kafka_consumer.go` (cuối file).
- [ ] Thêm field `ingestMeters map[string]*RateMeter` + `meterMu sync.Mutex` vào `KafkaConsumer` struct.
- [ ] Wire `metrics.IngestRateMsgsPerSec.WithLabelValues(topic).Set(...)` sau mỗi fetch OK.
- [ ] Thêm ticker 5s đọc `reader.Stats()` → `metrics.ConsumerLag.Set(stats.Lag)`.
- **DoD**: chạy worker 30s với traffic giả → `curl :9090/metrics | grep cdc_kafka_consumer_lag` thấy value > 0 khi backlog.

### T-BE-04 — Wire consume rate (BatchBuffer.Flush)
- [ ] Mở `internal/handler/batch_buffer.go` — sau khi UPSERT thành công, đếm `writtenPerTable map[string]int`.
- [ ] Thêm helper `consumeRate(table)` (cùng pattern RateMeter).
- [ ] Set `metrics.ConsumeRateMsgsPerSec.WithLabelValues(table).Set(rate.Rate())`.
- **DoD**: smoke 30s → `cdc_consume_rate_msgs_per_sec{target_table="..."}` ≠ 0 sau khi worker UPSERT thật.

### T-BE-05 — Inc transient errors (3 call-site)
- [ ] `kafka_consumer.go` FetchMessage error branch — `metrics.KafkaTransientErrors.WithLabelValues("kafka_consumer", string(cls)).Inc()`.
- [ ] `cmd/sinkworker/main.go:153` — sau khi classify, inc với component=`sinkworker`.
- [ ] DLQ publisher service (file Muscle xác định) — component=`dlq_publisher`.
- **DoD**: simulate broker restart → counter tăng theo error_class.

### T-BE-06 — sinkworker refactor
- [ ] `cmd/sinkworker/main.go:143-180` — replace `logger.Error("kafka fetch error", ...)` bằng classifier pattern + Warn + 200ms sleep + Inc metric.
- **DoD**: chạy sinkworker, sample 100 message dummy, log noise giảm; smoke compare before/after.

---

## Phase B2 — Snapshot + Debezium queue (~4h)

### T-BE-07 — 4 snapshot metric
- [ ] Append `SnapshotActiveSlots`, `SnapshotProgressPercent`, `SnapshotThroughputMBps`, `SnapshotETASeconds` vào `pkgs/metrics/prometheus.go`.
- **DoD**: `go build` PASS.

### T-BE-08 — Emit snapshot metric
- [ ] `internal/handler/snapshot_runner_handler.go` — đầu run: `metrics.SnapshotActiveSlots.Inc()`, defer `Dec()`.
- [ ] Trong loop process: tính `processed/total` → set progress.
- [ ] Throughput: `RateMeter` cho byte volume.
- [ ] ETA: `remaining / rate`.
- [ ] Cleanup: `Delete` labels khi done.
- **DoD**: kick 1 snapshot Mongo→shadow → tail `/metrics` thấy progress 0→100; cleanup OK.

### T-BE-09 — Probe debezium_queue
- [ ] Tạo `cdc-cms-service/internal/infra/observability/probes/debezium_queue.go` (Section 6 impl).
- [ ] Tạo `debezium_queue_test.go` — 2 test (URL empty → unknown; mock 200 với metrics body → trả connectors).
- **DoD**: `go test ./internal/infra/observability/probes/...` PASS.

### T-BE-10 — Wire probe vào health aggregator
- [ ] Mở `cdc-cms-service/internal/infra/observability/system_health.go` (Muscle locate file aggregator).
- [ ] Thêm 1 dòng `pipeline["debezium_queue"] = probes.DebeziumQueue(ctx, h.deps, h.cfg.KafkaConnectURL)`.
- **DoD**: GET `/api/system/health` → response chứa `cdc_pipeline.debezium_queue.status`.

---

## Phase B3 — Aggregator API (5 endpoint) (~5h)

### T-BE-11 — File handler + router wire
- [ ] Tạo `cdc-cms-service/internal/api/dashboard_handler.go` (Section 7 impl skeleton).
- [ ] Tạo `dashboard_handler_test.go` skeleton.
- [ ] Wire vào `internal/router/router.go` (hoặc tương đương) — `dashboardH.Register(app)`.
- **DoD**: 5 route hiện qua `app.Stack()` debug.

### T-BE-12 — `/timeline` impl
- [ ] Implement `promRange` (Section 7 impl).
- [ ] Cache TTL 10s.
- [ ] Test: mock Prom server trả range_data → assert 3 series có cùng length.
- **DoD**: `curl '/api/v1/dashboard/timeline?range=5m&step=15s'` trả 200 + 3 series.

### T-BE-13 — `/snapshot/active` impl
- [ ] Implement bằng query Prom (ADR-003) hoặc gọi worker (fallback).
- [ ] Group by snapshot_id, table.
- **DoD**: smoke với 1 active snapshot → list trả.

### T-BE-14 — `/snapshot/:id/prioritize` impl
- [ ] Implement publish NATS subject `cdc.cmd.snapshot.priority`.
- [ ] Worker side (`snapshot_runner_handler.go` hoặc dispatcher) subscribe + re-order queue → **scope worker task riêng T-BE-14b**.
- [ ] Test: assert NATS publish call.
- **DoD**: curl POST → 200; logs worker thấy "priority bump".

### T-BE-15 — `/dlq/recent` impl
- [ ] Query `failed_sync_logs` ORDER BY occurred_at DESC LIMIT N.
- [ ] Bao gồm `_otel_trace_id` (Phase B4 sẽ có column).
- [ ] Build `signoz_url` từ env `SIGNOZ_BASE_URL`.
- **DoD**: insert 1 DLQ row test → endpoint trả.

### T-BE-16 — `/drift/recent` impl
- [ ] Query `cdc_internal.schema_proposal` JOIN `pending_field`.
- **DoD**: drift detect 1 field mới → endpoint trả trong < 30s.

### T-BE-17 — Cache layer
- [ ] In-memory `sync.Map` hoặc map+mutex với TTL 10s cho timeline.
- **DoD**: gọi 2 lần liên tiếp → cache hit (verify qua log `prom_query=skipped`).

---

## Phase B4 — Trace correlation (~2h)

### T-BE-18 — Migration
- [ ] Tạo `centralized-data-service/migrations/0XXX_add_otel_trace_id_to_failed_sync_logs.up.sql` + `.down.sql` (Section 8 impl).
- **DoD**: `make migrate-up` PASS; `\d failed_sync_logs` thấy 2 cột mới; `make migrate-down` PASS.

### T-BE-19 — DLQ producer capture trace
- [ ] Mở DLQ producer service (Muscle locate: `internal/service/dlq_*.go`).
- [ ] Import `go.opentelemetry.io/otel/trace`.
- [ ] Extract `trace.SpanContextFromContext(ctx)` → persist trace_id/span_id.
- **DoD**: insert 1 DLQ → row có trace_id (`SELECT _otel_trace_id FROM failed_sync_logs LIMIT 1`).

### T-BE-20 — Surface trace_id qua API
- [ ] Đảm bảo T-BE-15 response chứa `trace_id`, `span_id`, `signoz_url`.
- **DoD**: curl endpoint → trace_id 32-char hex; signoz_url valid format.

---

## Phase B5 — Smoke gate CI (~1.5h)

### T-BE-21 — Tạo smoke binary
- [ ] Tạo `centralized-data-service/cmd/metrics_smoke/main.go` (Section 10 impl).
- **DoD**: `go run ./cmd/metrics_smoke` chạy local PASS.

### T-BE-22 — Makefile + CI step
- [ ] Thêm target `smoke-metrics` vào Makefile.
- [ ] Wire vào CI (GitHub Action `.github/workflows/ci.yml` hoặc file CI hiện có) — step `make smoke-metrics` sau khi build + worker up trong container.
- **DoD**: PR cố tình bỏ `.Set()` cho 1 metric → CI fail.

---

## Cross-cutting / final

### T-BE-99 — Security review + verify gate
- [ ] Chạy `/security-agent` review toàn bộ patch (theo §8).
- [ ] `go build ./...` PASS.
- [ ] `go vet ./...` PASS.
- [ ] `go test ./...` PASS.
- [ ] Manual smoke 5 endpoint dashboard.
- [ ] Append `05_progress.md` cho từng task completion.
