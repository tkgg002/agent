# Gap Analysis — Review `02_plan_observability_final.md`

> **Date**: 2026-04-17
> **Reviewer**: Brain (claude-opus-4-7)
> **Reviewee**: Muscle/Brain cũ (claude-sonnet-4-6) — tác giả plan gốc
> **Scope**: Review logic, performance, optimization cho luồng Observability tại scale 50M records, ~200 tables, 20+ topics, ~500 events/s throughput ước tính.
> **Verdict**: Plan đã **deliver được v1 chạy được** (T1-T12 runtime verified theo progress). Nhưng có **flaws về performance, reliability, cardinality, và security** cần fix trước khi push lên prod full scale.

---

## 0. Tổng quan severity

| # | Vấn đề | Severity | Task liên quan |
|:--|:-------|:---------|:---------------|
| 1 | `/api/system/health` aggregate 5+ external calls đồng bộ, không timeout/circuit breaker → cascade fail | **CRITICAL** | T1, T4 |
| 2 | P95/P99 "compute từ activity_log" — SAI semantic, không phải histogram | **CRITICAL** | T10 |
| 3 | Activity Log batch per-topic × 20+ topics → 20 flusher goroutine + thundering insert | **HIGH** | T6 |
| 4 | Activity Log không partition/TTL → bloat | **HIGH** | T6 |
| 5 | OTel Logs `sampleRatio=1.0` + no backpressure → OOM khi SigNoz down | **HIGH** | T13 |
| 6 | Prometheus cardinality không kiểm soát (table × op × topic) | **HIGH** | T8, metrics nói chung |
| 7 | FE auto-refresh 30s nhưng API response 2-5s → UX giật + stale banner | **MEDIUM** | T4, T5 |
| 8 | Restart Connector button không RBAC/audit/confirm | **MEDIUM** | T12 |
| 9 | Consumer lag source không nêu → ai implement sao cũng được | **MEDIUM** | T11 |
| 10 | Alert rule không có dedup/silencing → alert fatigue | **MEDIUM** | §1 Section 5 |
| 11 | Trace propagation qua Kafka không dùng OTel Kafka instrumentation | **MEDIUM** | T14 |
| 12 | NATS `/jsz` check — có thể sai endpoint (JetStream vs core NATS) | **LOW** | T1 |
| 13 | E2E latency buckets không tối ưu cho distribution thực | **LOW** | T8 |
| 14 | Không có SLO/error budget định nghĩa → alert threshold tùy tiện | **LOW** | §5 |
| 15 | FE không handle partial data (1 component down thì cả page lỗi?) | **LOW** | T4 |

---

## 1. `/api/system/health` — blocking aggregate là bom nổ chậm

### Plan hiện tại (§2, T1)
CMS aggregate:
- Worker health API
- Kafka Connect API (`/connectors/status`)
- NATS `/jsz`
- Postgres query (registry, activity, recon, failed_logs)
- Compute alerts

### Vấn đề
- **Sequential call 5+ services**. Mỗi call timeout mặc định (Go http client default = 0 = infinite). Nếu 1 service hang → cả endpoint hang.
- Với auto-refresh FE mỗi 30s, **mỗi user tab = 1 request × 5 component call** = amplification. 10 người xem FE = 50 đồng thời request tới Worker/Kafka Connect.
- Không có cache → **mỗi request = full aggregate**. Kafka Connect REST API không thiết kế cho high QPS → có thể bị rate-limit hoặc chậm.

### Đề xuất

**A. Background collector + cache**
```
┌─────────────────────────┐
│ Background worker       │
│ (every 15s)             │
│  ├─► Poll Worker        │
│  ├─► Poll Kafka Connect │
│  ├─► Poll NATS          │
│  └─► Query Postgres     │
│       → write Redis key │
│         system_health:* │
└────────┬────────────────┘
         │
┌────────▼────────────────┐
│ /api/system/health      │
│ Read Redis (≤ 1ms)      │
│ Return cached snapshot  │
│ + cache_age field       │
└─────────────────────────┘
```
- 100 user tab đồng thời = 100 Redis GET = không đau gì.
- Nếu background worker die → `cache_age` lớn → FE hiển thị "stale — last update 5 min ago".

**B. Per-component timeout + fallback**
```go
type HealthResult struct {
    Status string  `json:"status"` // ok | degraded | down | unknown
    Latency int    `json:"latency_ms"`
    Error  string  `json:"error,omitempty"`
}

func (s *Service) probeKafkaConnect(ctx context.Context) HealthResult {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    // ... call API
    // on timeout → Status="unknown", not blocking other probes
}
```

**C. Parallel call (minimum)**
- Nếu chưa kịp chuyển sang cached: dùng `errgroup` + per-probe timeout 2s. Tổng tối đa 2s thay vì sequential 10s.

### Đề xuất task
- **T1 rewrite**: Background collector (goroutine ticker 15s) + Redis cache + handler chỉ read Redis.
- **Task thêm** T1a: Per-component timeout 2s, fallback "unknown" với last-known status.

---

## 2. P95/P99 "compute từ activity_log" — SAI CƠ BẢN

### Plan hiện tại (T10, progress note)
> T10: System Health API compute P50/P95/P99 from activity_log ✅ runtime: P50=152ms

### Vấn đề chí mạng
- **Prometheus histogram** (T8 `cdc_e2e_latency_seconds`) là semantic đúng cho P95/P99.
- Nhưng T10 lại "compute từ activity_log" — tức từ DB rows → query SQL như:
  ```sql
  SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms) FROM cdc_activity_log ...
  ```
- **Đây KHÔNG phải histogram-based percentile**. Đây là **sample-based**:
  - activity_log batch mỗi 100 msg hoặc 5s → 1 row có `duration_ms` TRUNG BÌNH CỦA BATCH.
  - Percentile của trung bình batch ≠ percentile của individual events.
  - **Mất fidelity**: outlier 30s trong batch 100 msg (99 msg 100ms) → avg ~400ms → khuất.
- Nếu thực sự dùng Prometheus histogram ở T8 + T9, thì T10 phải dùng `histogram_quantile(0.95, rate(cdc_e2e_latency_seconds_bucket[5m]))` qua Prom HTTP API.

### Đề xuất
- **T10 rewrite**:
  - System Health API gọi Prometheus HTTP API `/api/v1/query`:
    ```
    GET /api/v1/query?query=histogram_quantile(0.95, sum by (le)(rate(cdc_e2e_latency_seconds_bucket[5m])))
    ```
  - Parse result → return.
- **Hoặc**: Nếu không có Prometheus ở prod → Worker tự expose `/metrics` + một process background scrape + compute percentile trong memory (T-Digest, HdrHistogram) — **không** đọc activity_log.
- Activity_log là event log, KHÔNG phải metric store.

### Note về "runtime verified P50=152ms"
- Nếu progress nói P50=152ms là **đã verified**, thì T10 đang chạy code sai semantics nhưng ra số trông hợp lý. Đây là **silent bug** — dễ gây misleading khi có spike.

---

## 3. Activity Log batch per-topic × 20+ topics

### Plan hiện tại (T6)
> Ghi mỗi 100 messages HOẶC mỗi 5 giây (whichever first), per topic riêng.

### Vấn đề ở scale
- 20 topics × 1 flusher goroutine/topic = **20 goroutines** chỉ để flush log.
- Nếu 5 topic đang bursty → 5 goroutine đồng thời INSERT activity_log → connection pool contention.
- Mỗi topic flush **riêng** → PG nhận 20 × 12 = **240 insert/phút** với batch nhỏ → tốn TX overhead.

### Đề xuất
- **Single flusher, multi-topic buffer**:
  ```
  buffer: map[topic]*BatchEntry
  Flush (5s ticker HOẶC khi tổng size > 1000 records):
    tx := db.Begin()
    INSERT ... VALUES (...), (...), (...)  -- multi-row insert tất cả topics
    tx.Commit()
  ```
- Gộp multi-topic vào 1 TX → **1 insert/5s** thay vì 20.
- Thread-safe queue với `chan` hoặc `sync.Mutex + map`.

### Performance so sánh
- Cũ: 20 goroutine × (1 insert / 5s) = 4 insert/s × 1 row avg = 4 rows/s với overhead TX 20 lần.
- Mới: 1 goroutine × (1 insert / 5s) = 0.2 insert/s × N rows batch.
- DB load giảm ~20×.

---

## 4. Activity Log table bloat

### Plan hiện tại
- Không nêu retention/partition cho `cdc_activity_log`.

### Scale
- 20 topics × batch/5s × 24h × 365d = **126M rows/năm** cho mỗi topic, tổng **2.5B rows/năm**.
- Kafka consumer batch log + NATS command log + admin action log → càng nhiều.

### Đề xuất
- **Partition theo thời gian** (daily hoặc weekly): `PARTITION BY RANGE (created_at)`.
- **TTL**: drop partition > 30 ngày (hoặc archive S3).
- Index: `(operation, created_at DESC)` cho "Recent Events" query.
- Nếu muốn long-term audit → dump định kỳ sang ClickHouse/S3.

### Task đề xuất
- **T6a**: Migration partition + retention policy cho activity_log.

---

## 5. OTel Logs — không có backpressure

### Plan hiện tại (T13)
```yaml
otel:
  sampleRatio: 1.0
```
- Bridge Zap → OTel → gRPC → SigNoz (localhost:4317).

### Vấn đề
- `sampleRatio=1.0` = **100% log đi qua** → Worker log 1000/s thì 1000/s gửi SigNoz.
- Nếu SigNoz collector chậm/down: default OTel exporter buffer **in-memory queue**. Queue full → **block hoặc drop silently**.
- OOM Worker nếu block + log tiếp tục sinh.

### Đề xuất
- **Sample ratio**:
  - Info log: 0.1 (10%) → giảm volume.
  - Error log: 1.0 (giữ tất cả).
  - Tail sampling trong OTel Collector (nếu cần full trace cho error-related span).
- **Batch processor config** với bounded queue + drop-oldest policy:
  ```yaml
  batch:
    send_batch_size: 512
    timeout: 5s
  memory_limiter:
    check_interval: 1s
    limit_mib: 256
    spike_limit_mib: 64
  ```
- **Fallback**: khi SigNoz down → log vẫn ghi console (file rotation local), KHÔNG leak ra OOM.

### Task đề xuất
- **T13 rewrite**: OTel config với sample ratio theo severity + memory_limiter + fallback console-only.

---

## 6. Prometheus cardinality — nguy hiểm

### Plan hiện tại
- `cdc_events_processed_total{table, op, topic}`
- `cdc_sync_success_total{table, op}`, `cdc_sync_failed_total{table, op}`
- `cdc_e2e_latency_seconds{?}` histogram với 9 buckets.

### Tính cardinality
- 200 tables × 4 ops (insert/update/delete/tombstone) × 20 topics × 9 buckets (histogram) = **144K time series** chỉ riêng e2e_latency.
- `sync_success/failed`: 200 × 4 = 800 series × 2 metrics = 1600.
- Tổng ~150K series. Prometheus khuyến nghị < 10M total instance → vẫn fit nhưng **eat budget nhanh**.

### Vấn đề thực tế
- Nếu label `table` động (plan có notion "registry tự động đăng ký bảng mới") → series tăng theo bảng → **không bounded**.
- Label `topic` = CDC topic name → có thể chứa tên DB/collection dynamic → cardinality bùng.

### Đề xuất
- **Whitelist label values**: chỉ các bảng "critical" có label riêng, còn lại gộp `table="other"`.
- **Drop `topic` label** (redundant với table, suy được từ registry) → giảm 20×.
- **Histogram buckets tối ưu cho CDC**:
  - Hiện tại: `[0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60]`.
  - P50 runtime = 152ms → bucket dưới 1s quá rộng. Nên thêm `0.05, 0.1, 0.15, 0.25, 0.5, 1, 2, 5, 30`.
  - Hoặc dùng **exponential buckets** (`prometheus.ExponentialBuckets(0.05, 2, 10)`).
- **Recording rules** (Prometheus): precompute percentile theo table_group, giảm query-time compute.

### Task đề xuất
- **T8 bổ sung**: Cardinality budget analysis + label whitelist.
- **T8b**: Prometheus recording rules cho system_health query.

---

## 7. FE auto-refresh 30s — UX cân nhắc

### Plan hiện tại (T5)
- Auto-refresh 30s.

### Vấn đề
- Mỗi refresh = full reload 6 sections, aggregate API 2-5s response → **UI giật** trong 2-5s.
- Chart realtime (E2E latency line chart) reload full → mất animation, cảm giác lag.
- Không handle partial data: nếu Kafka Connect down → toàn bộ Section 2 biến mất? Hay hiển thị "unknown" state?

### Đề xuất
- **React Query / SWR** với `staleWhileRevalidate` + `refetchInterval: 30000`. UI giữ data cũ, fetch background.
- **Individual section polling**: mỗi section tự fetch endpoint riêng (nếu latency khác nhau). Ví dụ: `/api/system/health/infrastructure` (nhanh, cache dài), `/api/system/health/pipeline` (chậm hơn).
- **Partial data**: mỗi component có `status` field (`ok|degraded|unknown`), FE render "skeleton" với unknown state.

### Task đề xuất
- **T5 bổ sung**: React Query + per-section endpoint + unknown-state rendering.

---

## 8. Restart Connector button — security miss

### Plan hiện tại (T12)
> Restart Connector button + CMS endpoint ✅ runtime: restart 204

### Vấn đề
- Không nêu: ai được bấm? Có confirm dialog? Có ghi audit?
- Restart connector = **production impact**. Misclick → dừng pipeline.

### Đề xuất (tương tự Data Integrity review §13)
- Role check: chỉ `ops-admin` bấm được.
- Confirm modal: "Bạn có chắc restart connector X? Lý do:" [textbox] → required.
- Audit log: `admin_actions(user, action='restart_connector', target=X, reason=..., ts)`.

---

## 9. Consumer lag source

### Plan hiện tại (§2 Section 2)
> Consumer Lag | Kafka consumer groups describe | Số msg đang chờ per topic

### Vấn đề
- "Kafka consumer groups describe" = CLI (`kafka-consumer-groups.sh`). Gọi CLI từ Go = subprocess → **chậm, fragile**.
- Có nhiều cách khác nhau (Kafka Admin API, JMX, kafka_exporter, Redpanda Console API) — chưa chọn.

### Đề xuất
- Dùng **Sarama admin client** trong Go Worker hoặc CMS:
  ```go
  admin, _ := sarama.NewClusterAdmin(brokers, config)
  offsets, _ := admin.ListConsumerGroupOffsets(groupID, nil)
  ```
- Hoặc dùng `kafka_exporter` + Prometheus scrape → metric `kafka_consumergroup_lag`.
- Chọn 1 cách, document rõ.

### Task đề xuất
- **T1 spec**: chỉ định rõ method + library cho consumer lag.

---

## 10. Alert rule — không dedup/silencing

### Plan hiện tại (§2 Section 5)
- Critical/Warning/Info banner.

### Vấn đề
- Debezium flap (FAILED → RUNNING → FAILED ...) → banner liên tục đổi → user làm ngơ.
- Không có notion silence (maintenance window).

### Đề xuất
- **Alert state machine**:
  - `firing` → hiển thị.
  - `resolved` → auto-hide sau 60s.
  - `acknowledged` → user bấm "ack" → hide cho đến khi fire lại (5 phút cool down).
  - `silenced` → admin silence 1h / 1 ngày / maintenance.
- Store alert state trong `cdc_alerts(id, fingerprint, status, fired_at, resolved_at, ack_by, silence_until)`.
- Fingerprint = hash(alert_name + labels) → dedup.

### Task đề xuất
- **Task thêm T_alert**: Alert state management + silencing.

---

## 11. Trace context qua Kafka

### Plan hiện tại (§6)
> Mỗi Kafka event → tạo OTel span → span context propagate qua EventHandler → BatchBuffer

### Vấn đề
- **Kafka message headers** là nơi inject trace context (W3C Trace Context). Plan không nêu: producer side (Debezium) có inject không?
- Debezium CDC connector **không tự inject** OTel trace context (feature này chỉ có khi dùng OTel Kafka Connect interceptor).
- Worker tự tạo span mới → **không có parent** → không trace ngược về nguồn.

### Đề xuất
- Option A: **Install OTel Kafka Connect interceptor** cho Debezium → producer inject W3C headers.
- Option B: **Worker bắt đầu span mới per event** (không parent) + attach `attributes.source.ts_ms`, `attributes.kafka.topic`, `attributes.kafka.partition`, `attributes.kafka.offset`. Trace được từ Worker consume → batch → UPSERT → PG, KHÔNG trace được Mongo → Debezium → Kafka (acceptable v1).
- Document rõ chọn option nào.

### Task đề xuất
- **T14 clarification**: Chọn option + config instrumentation.

---

## 12. NATS `/jsz` endpoint

### Plan hiện tại (§2 Section 1)
> NATS | HTTP /jsz | Status + stream count + consumer count

### Vấn đề
- `/jsz` = JetStream info. CDC dự án đang dùng **NATS Core** (pub/sub thuần) hay **JetStream** (stream persistent)?
- Nếu là NATS Core (command pattern từ CMS → Worker) → `/jsz` trả rỗng → UI hiển thị "0 streams" gây hiểu nhầm.

### Đề xuất
- Check code hiện tại: dùng `js, _ := nc.JetStream()` hay `nc.Publish()`?
- Nếu NATS Core → dùng `/varz` (server info), `/connz` (connections), `/subsz` (subscriptions).
- Nếu JetStream → dùng `/jsz`.
- Nếu cả hai → query cả hai, hiển thị riêng.

### Task đề xuất
- **T1 verify**: Verify NATS mode trước khi implement probe.

---

## 13. E2E latency buckets

### Plan hiện tại
```go
Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60}
```

### Vấn đề
- Runtime progress note: P50 = 152ms → nằm giữa bucket 0.1 và 0.25 → **độ phân giải thấp**. P95/P99 sẽ không chính xác trong dải 100-500ms.
- Bucket 30s và 60s có ý nghĩa gì? CDC "healthy" latency thường < 2s → bucket > 5s là "anomaly" → chỉ cần 1 bucket "long tail".

### Đề xuất
```go
Buckets: []float64{
    0.025, 0.05, 0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8,
}
// Exponential, factor 2, base 25ms
```
- Resolution cao ở dải 25ms-1.6s (typical CDC).
- 6.4-12.8s bucket đủ để catch anomaly.
- Tổng 10 bucket thay vì 9 → cardinality gần như không đổi.

---

## 14. Không có SLO/error budget

### Plan hiện tại
- Alert threshold hardcoded: "lag > 10000 = critical, lag > 1000 = warning, drift > 0 = warning".
- Con số này dựa vào đâu?

### Đề xuất định nghĩa SLO
- **SLO 1**: P99 E2E latency ≤ 5s trong 99% 5-min window / 30 ngày.
- **SLO 2**: Recon drift = 0 trong 99.9% daily check / 30 ngày.
- **SLO 3**: Worker availability ≥ 99.95% (max 22 phút downtime/tháng).
- **SLO 4**: Zero data loss — failed_sync_logs không có record "dead" quá 24h.

### Threshold derive từ SLO
- P99 > 5s × 5 phút → alert warning.
- P99 > 5s × 15 phút → alert critical.
- Lag threshold = retention_ms × 0.5 (ví dụ retention 14d × 50% = 7d lag tương đương N msg).

### Task đề xuất
- **Task thêm T_SLO**: Define SLO + derive alert rules từ SLO.

---

## 15. FE không handle partial data

### Plan hiện tại (T4)
- Section 1-6 hiển thị đầy đủ.

### Case fail
- Kafka Connect API timeout → section 2 không có data → FE hiển thị `undefined` / `N/A` / crash?

### Đề xuất
- Mỗi section có `status` enum: `ok | stale | error | unknown`.
- API response:
  ```json
  {
    "sections": {
      "infrastructure": { "status": "ok", "data": {...} },
      "pipeline": { "status": "error", "error": "kafka connect timeout", "data": null },
      ...
    },
    "cache_age_seconds": 12
  }
  ```
- FE render per-section: error → red banner trong section đó + "retry" button, không ảnh hưởng section khác.

---

## 16. Flow review — từng API/luồng

### 16.1. `GET /api/system/health` — tổng hợp
Đã phân tích ở §1, §7, §15. Tóm lại cần:
- Background collector + Redis cache.
- Per-component timeout 2s.
- Per-section status enum.

### 16.2. Kafka consumer batch log → Activity Log
Đã phân tích ở §3, §4. Tóm lại:
- Single flusher multi-topic.
- Partitioned table + TTL.

### 16.3. NATS command publishResult → Activity Log
- Plan OK về nội dung. Bổ sung:
  - Batch insert nếu command volume cao (> 100/s).
  - TTL cùng bảng activity_log.

### 16.4. E2E latency measurement
Đã phân tích ở §2. Tóm lại:
- Prometheus histogram (T8/T9) là source of truth.
- Bỏ T10 "compute từ activity_log" HOẶC chuyển sang gọi Prom API.

### 16.5. Debezium status poll + Restart
- Poll: xử lý ở §1 (cache).
- Restart: xử lý ở §8 (RBAC + audit).
- **Thêm**: rate limit — không cho restart > 3 lần / giờ (auto-block).

### 16.6. OTel pipeline (log/trace)
Đã phân tích ở §5, §11. Tóm lại:
- Sample ratio theo severity.
- Memory limiter.
- Fallback console.
- Trace context: chọn instrumentation mode + document.

---

## 17. Tasks ĐỀ XUẤT BỔ SUNG

| ID | Task | Lý do |
|:---|:-----|:------|
| T1a | Per-component timeout + fallback status | Reliability |
| T1b | Background collector + Redis cache | Performance |
| T5a | React Query + per-section status | UX |
| T6a | Activity_log partition + TTL | Bloat control |
| T8c | Prometheus cardinality budget + label whitelist | Prom health |
| T8d | Prometheus recording rules | Query performance |
| T11a | Chỉ định rõ lag source (Sarama admin vs exporter) | Implementation clarity |
| T12a | Restart connector RBAC + audit + rate limit | Security |
| T_alert | Alert state machine + silencing | Alert fatigue |
| T_SLO | Define SLO + derive alert thresholds | Engineering rigor |

---

## 18. Tasks ĐỀ XUẤT REWRITE

| ID | Plan cũ | Plan mới |
|:---|:--------|:---------|
| T1 | Sync aggregate 5+ external calls | Background collector → Redis → handler read cache |
| T10 | Compute P95/P99 từ activity_log | Gọi Prom `histogram_quantile` hoặc compute từ `/metrics` histogram |
| T13 | `sampleRatio=1.0` đơn giản | Severity-aware sample + memory_limiter + fallback |

---

## 19. Check list "Staff Engineer có duyệt PR này?"

- [ ] API aggregate có timeout + fallback? → CHƯA (T1).
- [ ] Percentile đúng semantics Prometheus? → CHƯA (T10).
- [ ] Activity log có retention? → CHƯA.
- [ ] OTel logs không gây OOM khi SigNoz down? → CHƯA (T13).
- [ ] Cardinality bounded? → CHƯA.
- [ ] Destructive button có RBAC + audit? → CHƯA.
- [ ] Alert có dedup? → CHƯA.
- [ ] SLO defined? → CHƯA.
- [ ] FE handle partial data? → CHƯA.

**Kết luận**: Plan đã chạy được v1 (progress note runtime verify) nhưng **nhiều khía cạnh "silent bug" ở scale prod**. Cần cập nhật v3 trước khi claim Observability xong.

---

## 20. Action Items

1. **Brain (tôi)**: KHÔNG chạm code. Tài liệu này là review.
2. **User review** các đề xuất → Brain cập nhật `02_plan_observability_final.md` v3 (có changelog).
3. **Muscle**:
   - T10 silent bug phải fix trước tiên (percentile đang sai semantics nhưng ra số thật).
   - T13 trước khi enable (chưa enable = chưa OOM).
   - T1 rewrite trước khi prod scale up user count.
4. **Pre-flight verify**:
   - NATS dùng Core hay JetStream?
   - Dự án có Prometheus không (SigNoz đã có, Prom riêng?)?
   - FE có đang dùng React Query chưa?
   - Traffic thực tế events/sec/topic (ước tính để tính cardinality)?

---

## 21. Ghi chú

- Các runtime-verified items (T6, T9, T10, T12) **không có nghĩa là đúng** — chỉ nghĩa là "chạy không crash + có data". Semantic đúng sai cần review riêng.
- Silent bug T10 là **ví dụ kinh điển**: AI write code passing smoke test nhưng concept sai → gây sai lệch decision khi dùng metric này trigger alert/capacity planning.

**Lesson cho `agent/memory/global/lessons.md`**:
> **Global Pattern [Muscle runtime verify A = số X hợp lý → kết luận A đúng] → Pitfall Y**: Runtime verify chỉ chứng minh A không crash + trả số. KHÔNG chứng minh A đúng semantics. **Đúng**: Mỗi metric/aggregation phải đi kèm "semantic validation" — so sánh với source-of-truth độc lập (Prom histogram_quantile vs sample percentile). Nếu chênh → stop, analyze.
