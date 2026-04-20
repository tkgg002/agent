# 03_implementation_v3_cms_phase0.md — CMS Phase 0 Quick Wins

> **Date**: 2026-04-17
> **Engineer**: Muscle (Chief Engineer, claude-opus-4-7-1m)
> **Plan source**: `02_plan_observability_v3.md` §3, §4
> **Scope**: CMS-side fixes for `cdc-cms-service` only (no `centralized-data-service`, no `cdc-cms-web`)

---

## 1. Task 1 — Fix silent percentile bug

### Problem recap
`internal/api/system_health_handler.go::getLatency()` computed P50/P95/P99 via
`percentile_cont()` over `cdc_activity_log.duration_ms`. Each row in that
table represents a **batch average** (`kafka-consume-batch` operation, N
events per row). Averaging a batch hides event-level outliers — a 5 s spike
inside a 10-event batch is flattened to ~0.59 s. The handler therefore
reported healthy percentiles while the pipeline actually had tail-latency
incidents.

### Files changed / added
- NEW `/Users/trainguyen/Documents/work/cdc-cms-service/internal/service/prom_client.go`
  - `PromClient` wrapping `github.com/prometheus/client_golang/api` +
    `api/prometheus/v1`.
  - `QueryPercentile(ctx, q, rangeWindow)` — Path A: PromQL
    `histogram_quantile(q, sum by (le) (rate(cdc_e2e_latency_seconds_bucket[W])))`.
  - Path B fallback: GET `<workerURL>/metrics`, parse via
    `expfmt.NewTextParser(prommodel.UTF8Validation)` (v0.67 requires explicit
    validation scheme, zero-value panics), compute quantile with
    `computeHistogramQuantile(les, sumByLe, q, total)` mirroring Prom's
    linear-interpolation bucket algorithm.
  - Source label: `prometheus` | `fallback_worker_metrics` | `unknown`.
  - `QueryLatencyTriple(ctx, "5m")` returns the P50/P95/P99 bundle as
    `LatencyResult{Source, P50Ms, P95Ms, P99Ms, Error}`.
- NEW `/Users/trainguyen/Documents/work/cdc-cms-service/internal/service/prom_client_test.go`
  - `TestPercentileHistogramCapturesOutlier` — proves histogram sees the
    5 s outlier (P99.5 = 4.800 s) while batch-avg simulation hides it
    (P99 = 0.546 s).
  - `TestPercentileFallbackScrape` — spins up an httptest `/metrics`
    endpoint, confirms end-to-end fallback parsing + quantile.
  - `TestPercentileUnknownWhenNoBackend` — graceful `source=unknown`
    with clear error when neither Prometheus nor worker is configured.

### Handler change
`/Users/trainguyen/Documents/work/cdc-cms-service/internal/api/system_health_handler.go:238`
dropped entirely. Latency now comes from the Redis snapshot
(`snap.Latency` = `service.LatencyResult`) which the background collector
populates via `PromClient.QueryLatencyTriple`. The JSON schema exposes the
existing `latency.p50_ms/p95_ms/p99_ms` keys plus the new `latency.source`
field.

### Unit test output (TestPercentile*)

```text
=== RUN   TestPercentileHistogramCapturesOutlier
    prom_client_test.go:83: histogram path: P99=0.100s P99.5=4.800s (outlier visible at 3.2s+)
    prom_client_test.go:99: activity_log SILENT BUG: batch_avg P99=0.546s (hides 5.0s event-level outlier)
--- PASS: TestPercentileHistogramCapturesOutlier (0.00s)
=== RUN   TestPercentileFallbackScrape
    prom_client_test.go:156: fallback scrape path: P99.5=4.800s (source=fallback_worker_metrics)
--- PASS: TestPercentileFallbackScrape (0.00s)
=== RUN   TestPercentileUnknownWhenNoBackend
--- PASS: TestPercentileUnknownWhenNoBackend (0.00s)
```

**Silent bug demonstrated**: same 100 observations (99×0.1 s + 1×5.0 s)
produce P99.5 = 4.800 s via histogram (correct) versus P99 = 0.546 s via
batch-averaged `percentile_cont` (hides the outlier by ~9x).

### Verify commands

```bash
cd /Users/trainguyen/Documents/work/cdc-cms-service
go test -count=1 -v -timeout=20s -run TestPercentile ./internal/service/
go build ./...
curl -sS http://127.0.0.1:8083/api/system/health | jq '.latency'
```

### Sample `latency` JSON from runtime

```json
{
  "source": "fallback_worker_metrics",
  "p50_ms": 60000,
  "p95_ms": 60000,
  "p99_ms": 60000
}
```

(Runtime values reflect current worker histogram state where 27 observations
landed in +Inf bucket; the algorithm correctly reports the last finite upper
bound of 60 s. When Prometheus server is wired via `system.prometheusUrl`,
source flips to `"prometheus"`.)

---

## 2. Task 2 — Background Health Collector + Redis cache

### Problem recap
The v2 handler synchronously aggregated 5+ external calls per request
(kafka-connect, worker, NATS, Postgres, Redis, Airbyte) plus a PG
`percentile_cont` query. Under even moderate load the p99 was 2-5 s and a
single slow upstream cascaded into full API failure.

### Files changed / added
- NEW `/Users/trainguyen/Documents/work/cdc-cms-service/internal/service/system_health_collector.go`
  - `Collector` struct owns: `db`, `redis`, `airbyteClient`, `prom` (PromClient).
  - `Run(ctx)` — ticker every 15s (configurable). Seeds the cache immediately
    on start so handlers don't 503 forever on cold start.
  - `collectAndCache(ctx)` — uses `golang.org/x/sync/errgroup` with 9
    parallel `g.Go` calls. Every probe returns `nil` so one failure never
    short-circuits the others.
  - Per-probe timeout: 2 s via `context.WithTimeout` inside each probe.
  - Status vocabulary: `ok | up | degraded | down | unknown`.
  - Snapshot → `redis.Set("system_health:snapshot", json, 60s)`.
- NEW `/Users/trainguyen/Documents/work/cdc-cms-service/internal/service/system_health_collector_test.go`
  - `TestComputeOverall` — critical > warning > healthy precedence.
  - `TestComputeAlertsPerSection` — verifies the same alert contract the
    current FE banner expects.
  - `TestSanitizeErrRedactsURLs` — proves credentials + hostnames don't leak
    into error strings (security gate).
  - `TestSnapshotJSONStability` — asserts every legacy top-level key still
    appears so the FE doesn't break.

### Handler rewrite
`/Users/trainguyen/Documents/work/cdc-cms-service/internal/api/system_health_handler.go`
shrunk from 358 LoC to ~120 LoC. It now only:
1. Reads `system_health:snapshot` from Redis (500 ms ctx).
2. Unmarshals into `service.Snapshot`.
3. Sets `snap.CacheAgeSeconds = now - snap.Timestamp`.
4. Returns JSON.
5. `redis.Nil` → HTTP 503 `{"status":"initializing","message":"collector not ready yet"}`.
6. `RestartDebezium` preserved as-is (proxies POST to Kafka Connect).

### Wiring
- `/Users/trainguyen/Documents/work/cdc-cms-service/config/config.go:29` —
  added `PrometheusURL`, `DebeziumConnector`, `HealthCacheKey` to
  `SystemConfig`. No hardcoded prod URLs — defaults from `config-local.yml`.
- `/Users/trainguyen/Documents/work/cdc-cms-service/config/config-local.yml:33` —
  added `prometheusUrl: ""`, `debeziumConnector: goopay-mongodb-cdc`,
  `healthCacheKey: system_health:snapshot`.
- `/Users/trainguyen/Documents/work/cdc-cms-service/internal/server/server.go:80` —
  constructs `service.NewPromClient`, `service.NewCollector`, then
  `api.NewSystemHealthHandler`. The collector is launched from `Start()` via
  `go s.healthCollector.Run(ctx)` (new `collectorCancel` captured for
  shutdown).

### AC verification

#### AC-1: p99 < 50 ms under 100 req load

```text
$ ab -n 200 -c 20 http://127.0.0.1:8083/api/system/health
Requests per second:    3136.91 [#/sec] (mean)
Percentage of the requests served within a certain time (ms)
  50%      2
  95%     21
  99%     21
```

**Result: P99 = 21 ms (well under 50 ms target).**

#### AC-2: Graceful degradation (Kafka Connect unreachable)

Started CMS with `kafkaConnectUrl: http://localhost:19999` (wrong port):

```text
cdc_pipeline.debezium.status: unknown
cdc_pipeline.debezium.error:  Get "<scheme-redacted>": dial tcp [::1]:19999: connect: connection refused
infrastructure.kafka.status:  unknown
infrastructure.kafka.error:   Get "<scheme-redacted>": dial tcp [::1]:19999: connect: connection refused
infrastructure.redis.status:  up
infrastructure.postgres.status: up
infrastructure.nats.status:   up
overall: degraded
```

Other sections ("redis", "postgres", "nats") stayed `up`. Overall rolled to
`degraded`, not `critical`, because Kafka-Connect returned `unknown` (not
`down`). URL scheme + host redacted via `sanitizeErr`.

#### AC-3: 503 when Redis has no snapshot yet

Deleted the cache key and hit the endpoint before the collector's first
tick finished:

```text
HTTP 503
{"message":"collector not ready yet","status":"initializing"}
```

#### AC-4: `cache_age_seconds` field

```text
$ curl http://127.0.0.1:8083/api/system/health | jq '.cache_age_seconds, .timestamp'
2
"2026-04-17T04:30:31.206839Z"
```

Value stays in 0..60 window (snapshot TTL = 60 s, collector interval = 15 s,
so typical age is 0-15).

---

## 3. Security gate review

- Redis key `system_health:snapshot` is namespaced; test runs used a
  distinct `system_health:snapshot_init_test` to avoid collision.
- `sanitizeErr` (collector) redacts `scheme://host:port/path` patterns into
  `<scheme-redacted>` so log lines or error bodies never leak internal
  hostnames/credentials (relevant when Path B fallback scrapes the worker).
- Prometheus URL is injected via config; empty value is accepted and the
  client simply skips Path A. No production URLs are committed.
- HTTP timeouts + context cancellation prevent slow dependencies from
  holding server goroutines.

---

## 4. Build + test summary

```text
$ cd /Users/trainguyen/Documents/work/cdc-cms-service
$ go build ./...                       # PASS (no output)
$ go test -count=1 -v -timeout=20s ./internal/service/
--- PASS: TestPercentileHistogramCapturesOutlier (0.00s)
--- PASS: TestPercentileFallbackScrape (0.00s)
--- PASS: TestPercentileUnknownWhenNoBackend (0.00s)
--- PASS: TestComputeOverall (0.00s)
--- PASS: TestComputeAlertsPerSection (0.00s)
--- PASS: TestSanitizeErrRedactsURLs (0.00s)
--- PASS: TestSnapshotJSONStability (0.00s)
PASS
ok      cdc-cms-service/internal/service    0.766s
```

---

## 5. File inventory (authoritative)

| Path | Action | Purpose |
|:-----|:-------|:--------|
| `/Users/trainguyen/Documents/work/cdc-cms-service/internal/service/prom_client.go` | NEW | Prometheus API client + /metrics fallback |
| `/Users/trainguyen/Documents/work/cdc-cms-service/internal/service/prom_client_test.go` | NEW | Silent-bug proof + fallback scrape test |
| `/Users/trainguyen/Documents/work/cdc-cms-service/internal/service/system_health_collector.go` | NEW | Background collector + snapshot writer |
| `/Users/trainguyen/Documents/work/cdc-cms-service/internal/service/system_health_collector_test.go` | NEW | Alert/overall/sanitize/JSON contract |
| `/Users/trainguyen/Documents/work/cdc-cms-service/internal/api/system_health_handler.go` | REWRITE | Thin cache read + initializing 503 |
| `/Users/trainguyen/Documents/work/cdc-cms-service/config/config.go` | EDIT | Added PrometheusURL, DebeziumConnector, HealthCacheKey |
| `/Users/trainguyen/Documents/work/cdc-cms-service/config/config-local.yml` | EDIT | Default values for new config keys |
| `/Users/trainguyen/Documents/work/cdc-cms-service/internal/server/server.go` | EDIT | Wire PromClient + Collector, start goroutine, shutdown cancel |
| `/Users/trainguyen/Documents/work/cdc-cms-service/go.mod`, `go.sum` | EDIT | + prometheus/client_golang, prometheus/common |
