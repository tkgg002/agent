# 03_implementation_backend — Technical Design (Backend)

> **Reader contract**: code demo dưới đây là **bản thiết kế** Brain đề xuất cho Muscle (theo §12). KHÔNG paste-and-go — Muscle PHẢI verify imports, types, build/vet/test trước khi commit.

---

## Section 1: Extract Kafka transient error classifier

**File mới**: `centralized-data-service/pkgs/kafka/classifier.go`

```go
package kafka

import (
	"errors"
	"net"
	"strings"

	kafkago "github.com/segmentio/kafka-go"
)

// TransientClass phân loại error để Prom counter và log strategy.
type TransientClass string

const (
	ClassNotLeader        TransientClass = "not_leader"
	ClassBrokerUnavail    TransientClass = "broker_unavailable"
	ClassTimeout          TransientClass = "timeout"
	ClassConnReset        TransientClass = "conn_reset"
	ClassOther            TransientClass = "other"
	ClassNotTransient     TransientClass = ""
)

// Classify trả về class transient nếu err thuộc danh sách known transient,
// hoặc ClassNotTransient nếu fatal.
func Classify(err error) TransientClass {
	if err == nil {
		return ClassNotTransient
	}
	msg := strings.ToLower(err.Error())

	// kafka-go typed errors
	var kerr kafkago.Error
	if errors.As(err, &kerr) {
		switch kerr {
		case kafkago.NotLeaderForPartition:
			return ClassNotLeader
		case kafkago.BrokerNotAvailable, kafkago.LeaderNotAvailable:
			return ClassBrokerUnavail
		case kafkago.RequestTimedOut, kafkago.NetworkException:
			return ClassTimeout
		}
	}

	// net.Error timeout
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return ClassTimeout
	}

	switch {
	case strings.Contains(msg, "not leader for partition"):
		return ClassNotLeader
	case strings.Contains(msg, "broker not available"), strings.Contains(msg, "leader not available"):
		return ClassBrokerUnavail
	case strings.Contains(msg, "request timed out"), strings.Contains(msg, "i/o timeout"):
		return ClassTimeout
	case strings.Contains(msg, "connection reset by peer"), strings.Contains(msg, "broken pipe"):
		return ClassConnReset
	case strings.Contains(msg, "network resume"):
		return ClassConnReset
	}
	return ClassNotTransient
}

// IsTransient = convenience wrapper.
func IsTransient(err error) bool {
	return Classify(err) != ClassNotTransient
}
```

**File test**: `pkgs/kafka/classifier_test.go` (Muscle PHẢI viết — 5+ test case).

---

## Section 2: Bổ sung 6 metric mới

**File**: `centralized-data-service/pkgs/metrics/prometheus.go`

```go
// APPEND vào file hiện có, GIỮ NGUYÊN block var (...) hiện tại
// ----- NEW METRICS (Dashboard V2) -----

IngestRateMsgsPerSec = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_ingest_rate_msgs_per_sec",
        Help: "Rolling 10s ingest rate (Kafka consumer fetch) per topic",
    },
    []string{"topic"},
)

ConsumeRateMsgsPerSec = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_consume_rate_msgs_per_sec",
        Help: "Rolling 10s consume rate (post-batch-flush) per target table",
    },
    []string{"target_table"},
)

KafkaTransientErrors = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "cdc_kafka_transient_errors_total",
        Help: "Transient Kafka errors classified by component and error class",
    },
    []string{"component", "error_class"},
)

SnapshotActiveSlots = promauto.NewGauge(
    prometheus.GaugeOpts{
        Name: "cdc_snapshot_active_slots",
        Help: "Current number of in-flight Snapshot V2 runners",
    },
)

SnapshotProgressPercent = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_snapshot_progress_percent",
        Help: "Progress (0..100) of a Snapshot V2 runner",
    },
    []string{"snapshot_id", "table"},
)

SnapshotThroughputMBps = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_snapshot_throughput_mb_per_sec",
        Help: "Rolling 10s throughput (MB/s) of a Snapshot V2 runner",
    },
    []string{"snapshot_id", "table"},
)

SnapshotETASeconds = promauto.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdc_snapshot_eta_seconds",
        Help: "Estimated remaining seconds for a Snapshot V2 runner",
    },
    []string{"snapshot_id", "table"},
)
```

---

## Section 3: Wire `ConsumerLag.Set()` + ingest/consume rate

**File**: `centralized-data-service/internal/handler/kafka_consumer.go`

Khu vực target: vòng lặp `fetchAndProcess` (audit cho thấy ở khoảng line 358-365 + line 416-423).

**Helper mới (cùng file)**:
```go
// RateMeter — rolling rate calculator (10s window).
type RateMeter struct {
    mu       sync.Mutex
    samples  []rateSample   // ring buffer
    capacity int
}
type rateSample struct {
    ts    time.Time
    count int64
}

func NewRateMeter(window time.Duration) *RateMeter {
    cap := int(window/time.Second) + 1
    return &RateMeter{samples: make([]rateSample, 0, cap), capacity: cap}
}

// Add records n events at time t.
func (r *RateMeter) Add(n int64) {
    r.mu.Lock()
    defer r.mu.Unlock()
    now := time.Now()
    r.samples = append(r.samples, rateSample{ts: now, count: n})
    cutoff := now.Add(-10 * time.Second)
    i := 0
    for i < len(r.samples) && r.samples[i].ts.Before(cutoff) {
        i++
    }
    r.samples = r.samples[i:]
}

// Rate returns msgs/sec averaged over the window.
func (r *RateMeter) Rate() float64 {
    r.mu.Lock()
    defer r.mu.Unlock()
    if len(r.samples) == 0 {
        return 0
    }
    var sum int64
    for _, s := range r.samples {
        sum += s.count
    }
    dur := time.Since(r.samples[0].ts).Seconds()
    if dur <= 0 {
        return 0
    }
    return float64(sum) / dur
}
```

**Wire ở consumer main loop**:
```go
// Trong KafkaConsumer struct, thêm field:
//   ingestMeters map[string]*RateMeter  // per topic
//   meterMu      sync.Mutex

// Sau khi fetch message OK:
msg, err := c.reader.FetchMessage(ctx)
if err != nil {
    // R-BE-4: classify + count
    cls := kafkapkg.Classify(err)
    if cls != kafkapkg.ClassNotTransient {
        metrics.KafkaTransientErrors.WithLabelValues("kafka_consumer", string(cls)).Inc()
        c.logger.Warn("kafka fetch transient error, retrying",
            zap.String("error_class", string(cls)),
            zap.Error(err))
        time.Sleep(200 * time.Millisecond)
        continue
    }
    // fatal — escalate
    c.logger.Error("kafka fetch fatal", zap.Error(err))
    return err
}

// R-BE-1 + R-BE-3
c.ingestRate(msg.Topic).Add(1)
metrics.IngestRateMsgsPerSec.
    WithLabelValues(msg.Topic).
    Set(c.ingestRate(msg.Topic).Rate())

// kafka-go: HighWaterMark exposed via reader.Stats() — ticker mỗi 5s
// (set up trong go routine khi NewKafkaConsumer):
go func() {
    t := time.NewTicker(5 * time.Second)
    defer t.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-t.C:
            stats := c.reader.Stats()
            metrics.ConsumerLag.
                WithLabelValues(stats.Topic, fmt.Sprintf("%d", stats.Partition)).
                Set(float64(stats.Lag))
        }
    }
}()
```

**Helper `ingestRate(topic)`**:
```go
func (c *KafkaConsumer) ingestRate(topic string) *RateMeter {
    c.meterMu.Lock()
    defer c.meterMu.Unlock()
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

---

## Section 4: Wire consume rate trong BatchBuffer.Flush()

**File**: `centralized-data-service/internal/handler/batch_buffer.go`

```go
// Sau khi UPSERT thành công, đếm theo target_table:
for table, written := range writtenPerTable {
    bb.consumeRate(table).Add(int64(written))
    metrics.ConsumeRateMsgsPerSec.
        WithLabelValues(table).
        Set(bb.consumeRate(table).Rate())
}
```

Helper `consumeRate` cùng pattern với `ingestRate`.

---

## Section 5: Snapshot metric emit

**File**: `centralized-data-service/internal/handler/snapshot_runner_handler.go`

```go
// Tại điểm bắt đầu run:
metrics.SnapshotActiveSlots.Inc()
defer metrics.SnapshotActiveSlots.Dec()

// Trong loop process N rows (audit cho thấy chunks ~1000 docs):
processed := int64(0)
total := totalCount // có sẵn từ db.collection.estimatedDocumentCount()
throughputMeter := NewRateMeter(10 * time.Second)

for batch := range cursor {
    bytesIn := estimateBytes(batch)
    throughputMeter.Add(bytesIn)
    processed += int64(len(batch))

    progressPct := 100.0 * float64(processed) / float64(total)
    metrics.SnapshotProgressPercent.
        WithLabelValues(snapshotID, tableName).
        Set(progressPct)

    bytesPerSec := throughputMeter.Rate()
    metrics.SnapshotThroughputMBps.
        WithLabelValues(snapshotID, tableName).
        Set(bytesPerSec / (1024 * 1024))

    // ETA
    if bytesPerSec > 0 {
        remaining := total - processed
        avgBytesPerRow := float64(estimateBytesFor1Row())  // heuristic
        remainingBytes := float64(remaining) * avgBytesPerRow
        metrics.SnapshotETASeconds.
            WithLabelValues(snapshotID, tableName).
            Set(remainingBytes / bytesPerSec)
    }
}

// Cleanup khi done:
metrics.SnapshotProgressPercent.DeleteLabelValues(snapshotID, tableName)
metrics.SnapshotThroughputMBps.DeleteLabelValues(snapshotID, tableName)
metrics.SnapshotETASeconds.DeleteLabelValues(snapshotID, tableName)
```

---

## Section 6: Probe `debezium_queue.go` (CMS)

**File mới**: `cdc-cms-service/internal/infra/observability/probes/debezium_queue.go`

```go
package probes

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// DebeziumQueueResult — output cached as snap.CDCPipeline["debezium_queue"].
type DebeziumQueueResult struct {
	Status     string                  `json:"status"`
	Connectors []DebeziumConnectorQueue `json:"connectors"`
	LatencyMs  int64                   `json:"latency_ms"`
	Error      string                  `json:"error,omitempty"`
}

type DebeziumConnectorQueue struct {
	Name                  string  `json:"name"`
	QueueSize             int64   `json:"queue_size"`
	QueueMax              int64   `json:"queue_max"`
	QueuePct              float64 `json:"queue_pct"`
	SourceRecordWriteRate float64 `json:"source_record_write_rate"`
}

// DebeziumQueue scrapes Kafka Connect REST API per-connector metrics.
// Endpoint: GET {connectURL}/connectors?expand=status&expand=info
// then per connector: GET {connectURL}/connectors/{name}/tasks
// Production: rely on Prometheus JMX exporter `debezium_metrics_*`.
func DebeziumQueue(ctx context.Context, deps HTTPDeps, kafkaConnectURL string) map[string]any {
	start := time.Now()
	out := map[string]any{"latency_ms": int64(0), "source": "kafka_connect_rest"}

	if strings.TrimSpace(kafkaConnectURL) == "" {
		out["status"] = StatusUnknown
		out["error"] = "kafka_connect_url not configured"
		out["latency_ms"] = time.Since(start).Milliseconds()
		return out
	}

	// 1. List connectors
	conns, err := listConnectors(ctx, deps, kafkaConnectURL)
	if err != nil {
		out["status"] = StatusUnknown
		out["error"] = SanitizeErr(err)
		out["latency_ms"] = time.Since(start).Milliseconds()
		return out
	}

	connectors := make([]DebeziumConnectorQueue, 0, len(conns))
	degraded := false
	down := false

	for _, name := range conns {
		// 2. Pull metrics endpoint (Connect 3.x+: /connectors/{n}/metrics)
		q, err := fetchConnectorQueue(ctx, deps, kafkaConnectURL, name)
		if err != nil {
			// graceful: skip connector
			continue
		}
		connectors = append(connectors, q)
		if q.QueuePct > 95 {
			down = true
		} else if q.QueuePct > 80 {
			degraded = true
		}
	}

	status := StatusOK
	if down {
		status = StatusDown
	} else if degraded {
		status = StatusDegraded
	}
	out["status"] = status
	out["connectors"] = connectors
	out["latency_ms"] = time.Since(start).Milliseconds()
	return out
}

func listConnectors(ctx context.Context, deps HTTPDeps, base string) ([]string, error) {
	ctxQ, cancel := context.WithTimeout(ctx, deps.ProbeTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctxQ, http.MethodGet, strings.TrimRight(base, "/")+"/connectors", nil)
	resp, err := deps.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var names []string
	if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
		return nil, err
	}
	return names, nil
}

func fetchConnectorQueue(ctx context.Context, deps HTTPDeps, base, name string) (DebeziumConnectorQueue, error) {
	// Path 1: /connectors/{name}/metrics (3.7+)
	// Path 2: JMX exporter labels — operator-dependent. For now, Path 1 only.
	ctxQ, cancel := context.WithTimeout(ctx, deps.ProbeTimeout)
	defer cancel()
	url := strings.TrimRight(base, "/") + "/connectors/" + name + "/metrics"
	req, _ := http.NewRequestWithContext(ctxQ, http.MethodGet, url, nil)
	resp, err := deps.Client.Do(req)
	if err != nil {
		return DebeziumConnectorQueue{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return DebeziumConnectorQueue{Name: name}, nil // graceful empty
	}
	var raw struct {
		Metrics map[string]float64 `json:"metrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return DebeziumConnectorQueue{Name: name}, nil
	}
	qs := raw.Metrics["queue-size"]
	qm := raw.Metrics["queue-total-capacity"]
	wr := raw.Metrics["source-record-write-rate"]
	pct := 0.0
	if qm > 0 {
		pct = (qs / qm) * 100
	}
	return DebeziumConnectorQueue{
		Name:                  name,
		QueueSize:             int64(qs),
		QueueMax:              int64(qm),
		QueuePct:              pct,
		SourceRecordWriteRate: wr,
	}, nil
}
```

**Wire vào aggregator**: trong `internal/infra/observability/system_health.go` thêm 1 dòng:
```go
pipeline["debezium_queue"] = probes.DebeziumQueue(ctx, h.deps, h.cfg.KafkaConnectURL)
```

---

## Section 7: Dashboard aggregator handler

**File mới**: `cdc-cms-service/internal/api/dashboard_handler.go`

```go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct {
	promURL     string
	httpClient  *http.Client
	natsPub     NATSPublisher
	dlqRepo     DLQRepo
	driftRepo   DriftRepo
	snapshotSvc SnapshotQuery

	cacheMu        sync.Mutex
	timelineCache  map[string]cachedTimeline
}

type cachedTimeline struct {
	body      []byte
	expiresAt time.Time
}

func (h *DashboardHandler) Register(app fiber.Router) {
	g := app.Group("/api/v1/dashboard")
	g.Get("/timeline", h.Timeline)
	g.Get("/snapshot/active", h.SnapshotActive)
	g.Post("/snapshot/:id/prioritize", h.SnapshotPrioritize)
	g.Get("/dlq/recent", h.DlqRecent)
	g.Get("/drift/recent", h.DriftRecent)
}

// Timeline — R-BE-8
func (h *DashboardHandler) Timeline(c *fiber.Ctx) error {
	rangeStr := c.Query("range", "15m")
	step := c.Query("step", "15s")
	topicPrefix := c.Query("topic_prefix", "cdc.")
	cacheKey := rangeStr + "|" + step + "|" + topicPrefix

	h.cacheMu.Lock()
	if ent, ok := h.timelineCache[cacheKey]; ok && time.Now().Before(ent.expiresAt) {
		body := ent.body
		h.cacheMu.Unlock()
		return c.Type("json").Send(body)
	}
	h.cacheMu.Unlock()

	end := time.Now()
	dur, _ := time.ParseDuration(rangeStr)
	start := end.Add(-dur)
	stepDur, _ := time.ParseDuration(step)

	ingest, err := h.promRange(c.Context(),
		fmt.Sprintf(`sum by () (rate(cdc_events_processed_total{status="success"}[%s]))`, step),
		start, end, stepDur)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	consume, _ := h.promRange(c.Context(),
		`sum by () (cdc_consume_rate_msgs_per_sec)`, start, end, stepDur)
	lag, _ := h.promRange(c.Context(),
		`sum by () (cdc_kafka_consumer_lag)`, start, end, stepDur)

	resp := fiber.Map{
		"range_start":   start.UTC().Format(time.RFC3339),
		"range_end":     end.UTC().Format(time.RFC3339),
		"step_seconds":  int(stepDur.Seconds()),
		"series": fiber.Map{
			"ingest_rate":  ingest,
			"consume_rate": consume,
			"consumer_lag": lag,
		},
	}
	body, _ := json.Marshal(resp)
	h.cacheMu.Lock()
	h.timelineCache[cacheKey] = cachedTimeline{body: body, expiresAt: time.Now().Add(10 * time.Second)}
	h.cacheMu.Unlock()
	return c.Type("json").Send(body)
}

// promRange — minimal Prom range query client.
func (h *DashboardHandler) promRange(ctx context.Context, q string, start, end time.Time, step time.Duration) ([]map[string]any, error) {
	if h.promURL == "" {
		return []map[string]any{}, nil
	}
	v := url.Values{}
	v.Set("query", q)
	v.Set("start", strconv.FormatFloat(float64(start.Unix()), 'f', 0, 64))
	v.Set("end", strconv.FormatFloat(float64(end.Unix()), 'f', 0, 64))
	v.Set("step", strconv.FormatFloat(step.Seconds(), 'f', 0, 64))
	url := strings.TrimRight(h.promURL, "/") + "/api/v1/query_range?" + v.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var raw struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Values [][]any `json:"values"` // [ts, value-string]
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := []map[string]any{}
	if len(raw.Data.Result) == 0 {
		return out, nil
	}
	for _, kv := range raw.Data.Result[0].Values {
		if len(kv) < 2 {
			continue
		}
		tsFloat, _ := kv[0].(float64)
		valStr, _ := kv[1].(string)
		val, _ := strconv.ParseFloat(valStr, 64)
		out = append(out, map[string]any{
			"t": time.Unix(int64(tsFloat), 0).UTC().Format(time.RFC3339),
			"v": val,
		})
	}
	return out, nil
}

// SnapshotPrioritize — R-BE-10
func (h *DashboardHandler) SnapshotPrioritize(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Priority int `json:"priority"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	payload, _ := json.Marshal(map[string]any{
		"snapshot_id": id,
		"priority":    body.Priority,
	})
	if err := h.natsPub.Publish("cdc.cmd.snapshot.priority", payload); err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "queued", "snapshot_id": id, "priority": body.Priority})
}

// DlqRecent / DriftRecent / SnapshotActive — implementation reads from existing
// repositories. Muscle to fill from DLQRepo / DriftRepo / snapshot query.
```

**Interfaces** (file `dashboard_handler.go` cùng package):
```go
type NATSPublisher interface { Publish(subject string, data []byte) error }
type DLQRepo interface { Recent(ctx context.Context, limit int, since time.Duration) ([]DLQItem, error) }
type DriftRepo interface { Recent(ctx context.Context, limit int) ([]DriftItem, error) }
type SnapshotQuery interface { ListActive(ctx context.Context) (active []SnapshotItem, pending []SnapshotItem, slots int, err error) }
```

Wire concrete impl trong `cmd/cms-server/main.go`.

---

## Section 8: Migration trace_id

**File mới**: `centralized-data-service/migrations/0XXX_add_otel_trace_id_to_failed_sync_logs.up.sql`

```sql
-- +goose Up
ALTER TABLE failed_sync_logs
    ADD COLUMN IF NOT EXISTS _otel_trace_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS _otel_span_id  VARCHAR(32);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_fsl_otel_trace_id
    ON failed_sync_logs(_otel_trace_id)
    WHERE _otel_trace_id IS NOT NULL;
```

**Companion `.down.sql`**:
```sql
-- +goose Down
DROP INDEX IF EXISTS idx_fsl_otel_trace_id;
ALTER TABLE failed_sync_logs
    DROP COLUMN IF EXISTS _otel_span_id,
    DROP COLUMN IF EXISTS _otel_trace_id;
```

---

## Section 9: Capture trace_id ở DLQ producer

**File**: `centralized-data-service/internal/service/dlq_*.go` (Muscle xác định file đúng — file producer DLQ).

```go
import "go.opentelemetry.io/otel/trace"

func (s *DLQService) Write(ctx context.Context, item DLQItem) error {
    sc := trace.SpanContextFromContext(ctx)
    if sc.IsValid() {
        item.OtelTraceID = sc.TraceID().String()
        item.OtelSpanID  = sc.SpanID().String()
    }
    // ... existing INSERT logic, persist 2 cột mới
}
```

---

## Section 10: Smoke gate

**File mới**: `centralized-data-service/cmd/metrics_smoke/main.go`

```go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Smoke check: boot worker với fixture, đợi N giây, scrape /metrics,
// assert 6 metric Dashboard V2 có sample ≠ 0.
//
// Usage: go run ./cmd/metrics_smoke -metrics-url http://localhost:9090/metrics -wait 30s
func main() {
	url := os.Getenv("METRICS_URL")
	if url == "" {
		url = "http://localhost:9090/metrics"
	}
	wait := 30 * time.Second
	log.Printf("[smoke] waiting %s for worker to emit metrics …", wait)
	time.Sleep(wait)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("[smoke] FAIL fetch metrics: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)

	required := []string{
		"cdc_ingest_rate_msgs_per_sec",
		"cdc_consume_rate_msgs_per_sec",
		"cdc_kafka_consumer_lag",
		"cdc_kafka_transient_errors_total",
		"cdc_snapshot_active_slots",
		// snapshot progress / throughput / eta may be absent if no snapshot running
	}
	for _, name := range required {
		if !hasNonZeroSample(text, name) {
			log.Fatalf("[smoke] FAIL: metric %s missing or all-zero", name)
		}
		fmt.Printf("[smoke] OK: %s\n", name)
	}
	fmt.Println("[smoke] ALL PASS")
}

func hasNonZeroSample(text, metric string) bool {
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, metric) {
			continue
		}
		if strings.Contains(line, "#") {
			continue
		}
		// "metric_name{labels} 0.5" — extract last token
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		v := parts[len(parts)-1]
		if v != "0" && v != "0.0" && v != "0.000000" {
			return true
		}
	}
	return false
}
```

**Makefile target**:
```makefile
smoke-metrics:
	@echo ">> running metrics smoke (assumes worker live on :9090)"
	@METRICS_URL=$${METRICS_URL:-http://localhost:9090/metrics} \
		go run ./cmd/metrics_smoke
```

---

## Verification checklist (theo §3 Plan & Verify)

- [ ] `go build ./...` PASS
- [ ] `go vet ./...` PASS
- [ ] `go test ./pkgs/kafka/...` PASS — classifier 5 case
- [ ] `go test ./internal/handler/...` PASS không regression
- [ ] `go test ./cdc-cms-service/internal/...` PASS
- [ ] Migration apply local PASS + rollback PASS
- [ ] `make smoke-metrics` PASS
- [ ] curl 5 endpoint dashboard trả 200 với schema đúng (kiểm bằng `jq`)
- [ ] `/security-agent` review PASS trước commit (§8)
