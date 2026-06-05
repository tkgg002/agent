# 09_tasks_solution_backend — Technical Solutions (Backend)

> **Mục đích**: Hồ sơ giải pháp kỹ thuật cụ thể cho từng task T-BE-*. Khác với `03_implementation_backend.md` (design blueprint), file này focus vào **bản patch cuối cùng** + **commands** + **expected output** để Muscle apply.
> **Brain code prohibition**: file này LÀ artifact plan; code chỉ là demo — Muscle responsable cho actual edits.

---

## T-BE-01 (Solution) — Extract classifier

**Patch**:
1. New file `centralized-data-service/pkgs/kafka/classifier.go` — content như Section 1 của `03_implementation_backend.md`.
2. New file `pkgs/kafka/classifier_test.go`:
```go
package kafka

import (
	"errors"
	"testing"

	kafkago "github.com/segmentio/kafka-go"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want TransientClass
	}{
		{"nil", nil, ClassNotTransient},
		{"not leader typed", kafkago.NotLeaderForPartition, ClassNotLeader},
		{"broker unavail string", errors.New("[8] Broker Not Available"), ClassBrokerUnavail},
		{"timeout string", errors.New("request timed out after 10s"), ClassTimeout},
		{"conn reset string", errors.New("connection reset by peer"), ClassConnReset},
		{"fatal", errors.New("schema validation failed"), ClassNotTransient},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Classify(c.err); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}
```
3. Edit `internal/handler/kafka_consumer.go` — locate function `isKafkaTransientError(err error) bool` (≈ line 1116). Remove function body, replace by:
```go
import kafkapkg "centralized-data-service/pkgs/kafka"

func isKafkaTransientError(err error) bool { return kafkapkg.IsTransient(err) }
```
Or directly call `kafkapkg.IsTransient(err)` at all call-sites and remove the local function. Verify by `grep -n isKafkaTransientError internal/`.

**Commands**:
```
go build ./...
go vet ./...
go test ./pkgs/kafka/...
go test ./internal/handler/...
```

**Expected**: all PASS, no new warnings.

---

## T-BE-02 (Solution) — Append 4 metric

**Patch**: insert vào `pkgs/metrics/prometheus.go` ngay TRƯỚC dòng đóng `)` cuối cùng (sau dòng 152). 4 block metric như Section 2 của impl doc.

**Commands**: `go build ./... && curl -s localhost:9090/metrics | grep -E 'cdc_(ingest_rate|consume_rate|kafka_transient)' | head`

**Expected**: 3 line `# HELP` + `# TYPE` cho mỗi metric.

---

## T-BE-03 + T-BE-04 (Solution) — Wire ConsumerLag + RateMeter

**Patch**:

A. Append to bottom of `internal/handler/kafka_consumer.go`:
```go
// ----- Dashboard V2 RateMeter -----
type RateMeter struct {
    mu       sync.Mutex
    samples  []rateSample
}
type rateSample struct {
    ts    time.Time
    count int64
}
func NewRateMeter(window time.Duration) *RateMeter { return &RateMeter{} }
func (r *RateMeter) Add(n int64) {
    r.mu.Lock(); defer r.mu.Unlock()
    now := time.Now()
    r.samples = append(r.samples, rateSample{ts: now, count: n})
    cutoff := now.Add(-10 * time.Second)
    i := 0
    for i < len(r.samples) && r.samples[i].ts.Before(cutoff) { i++ }
    r.samples = r.samples[i:]
}
func (r *RateMeter) Rate() float64 {
    r.mu.Lock(); defer r.mu.Unlock()
    if len(r.samples) == 0 { return 0 }
    var sum int64
    for _, s := range r.samples { sum += s.count }
    dur := time.Since(r.samples[0].ts).Seconds()
    if dur <= 0 { return 0 }
    return float64(sum) / dur
}
```

B. Add to `KafkaConsumer` struct: 
```go
ingestMeters map[string]*RateMeter
meterMu      sync.Mutex
```

C. Add helper method:
```go
func (c *KafkaConsumer) ingestRate(topic string) *RateMeter {
    c.meterMu.Lock(); defer c.meterMu.Unlock()
    if c.ingestMeters == nil {
        c.ingestMeters = make(map[string]*RateMeter)
    }
    m, ok := c.ingestMeters[topic]
    if !ok {
        m = NewRateMeter(10 * time.Second)
        c.ingestMeters[topic] = m
    }
    return m
}
```

D. In `fetchAndProcess` (or main loop), AFTER `FetchMessage` returns OK:
```go
c.ingestRate(msg.Topic).Add(1)
metrics.IngestRateMsgsPerSec.WithLabelValues(msg.Topic).Set(c.ingestRate(msg.Topic).Rate())
```

E. After consumer constructor, kick goroutine:
```go
go func() {
    t := time.NewTicker(5 * time.Second)
    defer t.Stop()
    for {
        select {
        case <-ctx.Done(): return
        case <-t.C:
            stats := c.reader.Stats()
            metrics.ConsumerLag.WithLabelValues(stats.Topic, fmt.Sprintf("%d", stats.Partition)).Set(float64(stats.Lag))
        }
    }
}()
```

F. Apply parallel patch in `batch_buffer.go` for consume rate:
```go
// after successful UPSERT:
for table, n := range writtenPerTable {
    bb.consumeRate(table).Add(int64(n))
    metrics.ConsumeRateMsgsPerSec.WithLabelValues(table).Set(bb.consumeRate(table).Rate())
}
```
Helper `consumeRate(table)` identical to `ingestRate`.

**Expected smoke (30s of traffic)**:
```
$ curl -s :9090/metrics | grep -E 'cdc_(ingest_rate|consume_rate|kafka_consumer_lag)' | head
cdc_ingest_rate_msgs_per_sec{topic="cdc.goopay.auth.user_auths"} 152.3
cdc_consume_rate_msgs_per_sec{target_table="user_auths"} 148.9
cdc_kafka_consumer_lag{topic="cdc.goopay.auth.user_auths",partition="0"} 1024
```

---

## T-BE-05 (Solution) — Inc transient counter

**Patch** in fetch error branch:
```go
cls := kafkapkg.Classify(err)
if cls != kafkapkg.ClassNotTransient {
    metrics.KafkaTransientErrors.WithLabelValues("kafka_consumer", string(cls)).Inc()
    c.logger.Warn("kafka fetch transient error, retrying",
        zap.String("error_class", string(cls)),
        zap.Error(err))
    time.Sleep(200 * time.Millisecond)
    continue
}
```
Apply also to sinkworker + DLQ publisher with component label respectively.

**Smoke test**: kill broker mid-stream → `cdc_kafka_transient_errors_total{component="kafka_consumer",error_class="broker_unavailable"}` increase.

---

## T-BE-06 (Solution) — sinkworker classifier

**Patch** `cmd/sinkworker/main.go:153` (line approx):
```go
msg, err := r.FetchMessage(ctx)
if err != nil {
    cls := kafkapkg.Classify(err)
    if cls != kafkapkg.ClassNotTransient {
        metrics.KafkaTransientErrors.WithLabelValues("sinkworker", string(cls)).Inc()
        logger.Warn("kafka fetch transient error, retrying",
            zap.String("error_class", string(cls)), zap.Error(err))
        time.Sleep(200 * time.Millisecond)
        continue
    }
    logger.Error("kafka fetch fatal", zap.Error(err))
    return err
}
```

**Verify**: compare log before/after — Error count khi broker restart giảm xuống Warn.

---

## T-BE-07 + T-BE-08 (Solution) — Snapshot metrics

**Patch** `snapshot_runner_handler.go`. Locate function `RunSnapshotV2(ctx, signal)` (Muscle xác định name chính xác). Wrap với:
```go
metrics.SnapshotActiveSlots.Inc()
defer metrics.SnapshotActiveSlots.Dec()

snapshotID := signal.SnapshotID
table := signal.TargetTable

totalDocs, _ := collection.EstimatedDocumentCount(ctx, nil)
processed := int64(0)
tpMeter := NewRateMeter(10 * time.Second)
defer func() {
    metrics.SnapshotProgressPercent.DeleteLabelValues(snapshotID, table)
    metrics.SnapshotThroughputMBps.DeleteLabelValues(snapshotID, table)
    metrics.SnapshotETASeconds.DeleteLabelValues(snapshotID, table)
}()

for cursor.Next(ctx) {
    var doc bson.M
    if err := cursor.Decode(&doc); err != nil { ... }
    bs, _ := bson.Marshal(doc)
    tpMeter.Add(int64(len(bs)))
    processed++

    if processed % 100 == 0 {   // emit every 100 docs (avoid hot-loop label churn)
        metrics.SnapshotProgressPercent.WithLabelValues(snapshotID, table).
            Set(100.0 * float64(processed) / float64(totalDocs))
        bps := tpMeter.Rate()
        metrics.SnapshotThroughputMBps.WithLabelValues(snapshotID, table).
            Set(bps / (1024 * 1024))
        if bps > 0 {
            remainBytes := float64(totalDocs - processed) * float64(len(bs))
            metrics.SnapshotETASeconds.WithLabelValues(snapshotID, table).
                Set(remainBytes / bps)
        }
    }
    // ... existing apply logic
}
```

**Verify**: kick snapshot, tail `:9090/metrics`:
```
cdc_snapshot_active_slots 1
cdc_snapshot_progress_percent{snapshot_id="snap-...",table="user_auths"} 42.5
cdc_snapshot_throughput_mb_per_sec{snapshot_id="snap-...",table="user_auths"} 14.2
cdc_snapshot_eta_seconds{snapshot_id="snap-...",table="user_auths"} 348
```

---

## T-BE-09 (Solution) — Debezium queue probe

**Patch**: tạo file mới `cdc-cms-service/internal/infra/observability/probes/debezium_queue.go` với content như Section 6 impl doc.

Test file `debezium_queue_test.go`:
```go
package probes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"context"
	"time"
)

func TestDebeziumQueue_NoURL(t *testing.T) {
	out := DebeziumQueue(context.Background(), HTTPDeps{ProbeTimeout: time.Second}, "")
	if out["status"] != StatusUnknown {
		t.Fatalf("expect unknown, got %v", out["status"])
	}
}

func TestDebeziumQueue_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/connectors":
			w.Write([]byte(`["conn-1"]`))
		case "/connectors/conn-1/metrics":
			w.Write([]byte(`{"metrics":{"queue-size":100,"queue-total-capacity":8192,"source-record-write-rate":12.5}}`))
		}
	}))
	defer srv.Close()
	deps := HTTPDeps{Client: srv.Client(), ProbeTimeout: 2 * time.Second}
	out := DebeziumQueue(context.Background(), deps, srv.URL)
	if out["status"] != StatusOK {
		t.Fatalf("expect ok, got %v", out["status"])
	}
}
```

---

## T-BE-11..17 (Solution) — Dashboard handler

**Patch**: tạo file `cdc-cms-service/internal/api/dashboard_handler.go` (toàn bộ Section 7 impl). Wire trong `internal/router/router.go`:
```go
import "centralized-data-service/cdc-cms-service/internal/api"   // adjust import path
// inside Register or Setup:
dashboardH := api.NewDashboardHandler(cfg.PromURL, natsConn, dlqRepo, driftRepo, snapshotSvc)
dashboardH.Register(app)
```

**Repository implementations** (Muscle viết riêng):

```go
// dlq_repo.go
type GormDLQRepo struct { DB *gorm.DB; signozBase string }

func (r *GormDLQRepo) Recent(ctx context.Context, limit int, since time.Duration) ([]DLQItem, error) {
    var rows []DLQItem
    cutoff := time.Now().Add(-since)
    if err := r.DB.WithContext(ctx).
        Table("failed_sync_logs").
        Where("occurred_at >= ?", cutoff).
        Order("occurred_at DESC").
        Limit(limit).
        Scan(&rows).Error; err != nil {
        return nil, err
    }
    for i := range rows {
        if rows[i].TraceID != "" {
            rows[i].SignozURL = strings.TrimRight(r.signozBase, "/") + "/trace/" + rows[i].TraceID
        }
    }
    return rows, nil
}
```

**Verify**:
```
curl -s 'http://localhost:8080/api/v1/dashboard/timeline?range=5m&step=15s' | jq
curl -s 'http://localhost:8080/api/v1/dashboard/snapshot/active' | jq
curl -X POST 'http://localhost:8080/api/v1/dashboard/snapshot/snap-xyz/prioritize' \
     -H 'content-type: application/json' \
     -d '{"priority":100}'
curl -s 'http://localhost:8080/api/v1/dashboard/dlq/recent?limit=5' | jq
curl -s 'http://localhost:8080/api/v1/dashboard/drift/recent?limit=5' | jq
```

---

## T-BE-18..20 (Solution) — Trace correlation

**Migration SQL**: như Section 8 impl doc.

**DLQ producer patch** (Muscle locate file in `internal/service/dlq_*.go`):
```go
import "go.opentelemetry.io/otel/trace"

// inside Write/Persist function:
sc := trace.SpanContextFromContext(ctx)
if sc.IsValid() {
    record.OtelTraceID = sc.TraceID().String()
    record.OtelSpanID  = sc.SpanID().String()
}
```

**Model struct** (`internal/model/failed_sync_log.go`):
```go
type FailedSyncLog struct {
    // ... existing fields
    OtelTraceID string `gorm:"column:_otel_trace_id" json:"trace_id,omitempty"`
    OtelSpanID  string `gorm:"column:_otel_span_id"  json:"span_id,omitempty"`
}
```

**Verify**: 
```
make migrate-up
psql -d cdc_metadata -c "\d failed_sync_logs"   # check 2 cột mới
# trigger 1 DLQ
psql -d cdc_metadata -c "SELECT _otel_trace_id, _otel_span_id FROM failed_sync_logs ORDER BY occurred_at DESC LIMIT 1"
```

---

## T-BE-21..22 (Solution) — Smoke gate

**Patch**: tạo `cmd/metrics_smoke/main.go` (Section 10).

**Makefile**:
```makefile
.PHONY: smoke-metrics
smoke-metrics:
	@echo ">> running metrics smoke (assumes worker live on :9090)"
	@METRICS_URL=$${METRICS_URL:-http://localhost:9090/metrics} \
		go run ./cmd/metrics_smoke
```

**CI step** (`.github/workflows/ci.yml`):
```yaml
- name: Smoke metrics
  run: |
    docker compose up -d worker
    sleep 30
    make smoke-metrics
```

---

## Cross-cutting verification

```
go build ./...                       # PASS
go vet ./...                         # PASS
go test ./pkgs/kafka/...             # PASS
go test ./internal/handler/...       # PASS  (no regression)
go test ./internal/service/...       # PASS
go test ./cdc-cms-service/...        # PASS

# manual
make smoke-metrics                   # PASS
curl 5 endpoint                      # 200
```

Sau khi tất cả task done, Muscle PHẢI:
1. APPEND `05_progress.md` per task (Format: `| ts | Muscle | model | T-BE-NN completed: <short> |`).
2. Chạy `/security-agent` review (§8).
3. Báo Brain (Antigravity) status để Brain duyệt + cập nhật `active_plans.md`.
