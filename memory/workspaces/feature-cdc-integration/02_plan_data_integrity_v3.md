# Plan: Data Integrity v3 — Scale-ready, Fintech-grade

> **Date**: 2026-04-17
> **Author**: Brain (claude-opus-4-7)
> **Supersedes**: `02_plan_data_integrity_final.md` (v2, claude-sonnet-4-6)
> **Based on**: `10_gap_analysis_data_integrity_review.md` + user answers + archaeology (Explore agent)
> **Target scale**: 50M records hiện tại, tăng theo cấp số (design cho **500M+**), > 2000 events/sec, 200+ bảng, 30+ DB.

---

## 0. Scale Budget (MANDATORY — new convention)

| Metric | Current | Design target | Hard limit |
|:-------|:--------|:--------------|:-----------|
| Bảng lớn nhất | 50M records | 500M records | 1B records |
| Throughput | ~500 events/s | 2000 events/s | 5000 events/s |
| Recon RAM budget per table run | unbounded (BUG) | 200 MB | 500 MB |
| Recon network per Tier 2 run | ~1.2 GB (BUG) | < 20 MB | 100 MB |
| Recon Mongo secondary CPU | unbounded | < 40% | 60% |
| PG replica CPU during Recon | unbounded | < 40% | 60% |
| DLQ rows/day (ước tính @ 0.01% fail) | < 500K | < 2M | 10M |
| Activity log rows/day | ~50K | < 500K | 2M |
| `failed_sync_logs` retention | forever (BUG) | 90 days | 180 days |
| Kafka retention | current = TBD | 14 days | 30 days |

**Pattern bắt buộc cho mọi operation**:
- **Streaming, không load full set**: dùng cursor + batch, XOR-hash aggregate, window-based compare.
- **Time-partitioned**: data cũ không đổi → cache hash vĩnh viễn, chỉ recompute recent.
- **Rate-limited**: token bucket + circuit breaker trên mọi Agent DB call.
- **Replica-first**: Mongo `readPreference=secondary`, PG read-replica.
- **Partitioned tables**: mọi bảng log/report PARTITION BY RANGE (created_at), TTL tự động.

---

## 1. Verified Assumptions

| ID | Assumption | Status | Source | Implication |
|:---|:-----------|:-------|:-------|:-----------|
| V1 | MongoDB secondary replica tồn tại | ✅ Confirmed (user) | — | Agent MUST use `readPreference=secondary` |
| V2 | PostgreSQL read-replica tồn tại | ✅ Confirmed (user) | — | Agent MUST dùng replica DSN cho Tier 1-3 read |
| V3 | Prometheus EXISTS (cạnh SigNoz) | ⚠️ Conflict | user: có; Explore: code define metrics nhưng **không expose `/metrics`**, không có Prom service trong `docker-compose.yml` | Task: expose `/metrics` + add Prom scrape target. Confirm prod infra có Prom server riêng chưa. |
| V4 | Debezium converter | ⚠️ Conflict | user: Avro; Explore: `mongodb-connector.json` = JSON converter, `schemas.enable=false` | Plan v3 design **2 phase**: Phase A — current JSON (schema validation qua `cdc_table_registry`). Phase B — migrate to Avro với Schema Registry (project đã có Schema Registry service nhưng chưa dùng cho CDC data) |
| V5 | NATS JetStream (streams thật) | ✅ Confirmed | Explore: `nats_client.go:57-87` — 3 streams `CDC_EVENTS`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`, FileStorage, retention 7d | OK, dùng `/jsz` endpoint cho monitoring |
| V6 | Index `updated_at` cả Mongo + PG | ✅ Confirmed (user) | — | Window-based Recon khả thi |
| V7 | Cột `_source_ts` | ❌ Missing | Explore: chỉ có `_synced_at` (wall clock Worker), KHÔNG có cột Debezium ts_ms | **T0: Migration 009 thêm `_source_ts BIGINT` (ts_ms) + index** |
| V8 | OTel Kafka instrumentation | ❌ Missing | Explore: `segmentio/kafka-go` bare, không có `otelkafka`, không có span wrap | Worker phải tạo span thủ công từ message header (W3C traceparent) hoặc bắt đầu root span mới |
| V9 | FE React Query | ❌ Not used | user + Explore: React 19 + Ant Design 6 + Axios thuần, no state lib | Plan thêm React Query tối thiểu cho 2 page (System Health + Data Integrity) |
| V10 | Redis available | ✅ Yes, underutilized | Explore: `pkgs/rediscache/`, docker-compose :16379, chỉ dùng cho schema cache | Free để dùng cho background health collector cache, recon state, idempotency key |

---

## 2. Kiến trúc v3 — Core/Agent mở rộng

```
┌───────────────────────────────────────────────────────────────────┐
│  Recon Core (CMS + Worker sidecar)                                │
│  - Orchestrate Tier 1 → 2 → 3 (budget-gated)                      │
│  - Heal via Debezium Signal (preferred) hoặc direct UPSERT       │
│  - Version-aware OCC trên _source_ts                              │
│  - Advisory lock (leader election)                                │
│  - Idempotency-Key middleware (Redis TTL 1h)                      │
│  - Audit Log batched (100 actions/insert)                         │
└────────────────────┬──────────────────────────────────────────────┘
                     │ (rate-limited, budgeted)
          ┌──────────┴──────────┐
          ▼                     ▼
┌──────────────────┐   ┌──────────────────┐
│  Source Agent    │   │  Dest Agent      │
│  Mongo SECONDARY │   │  PG REPLICA      │
│  readPref=sec    │   │  SET TX READ ONLY│
│                  │   │                  │
│  Tier 1: count   │   │  Tier 1: pg_class│
│   by window      │   │   .reltuples     │
│   (updated_at    │   │   (estimate)     │
│    watermark)    │   │  or exact count  │
│                  │   │   by window      │
│                  │   │                  │
│  Tier 2: XOR-    │   │  Tier 2: XOR-    │
│   hash per       │   │   hash per       │
│   window         │   │   window         │
│   (no ID list)   │   │   (no ID list)   │
│                  │   │                  │
│  Tier 3: 256-    │   │  Tier 3: 256-    │
│   bucket hash    │   │   bucket hash    │
│   (recent + 1%   │   │   (recent + 1%   │
│    sampled hist) │   │    sampled hist) │
│                  │   │                  │
│  Rate: token     │   │  Rate: token     │
│   bucket 5K/s    │   │   bucket 5K/s    │
│  Breaker: sony/  │   │  Breaker: sony/  │
│   gobreaker      │   │   gobreaker      │
└──────────────────┘   └──────────────────┘
```

**Changes so với v2**:
- Agent **streaming-only**: không bao giờ `GetAllIDs()` — chỉ hash_agg (KHÁC v2 `recon_source_agent.go:130`).
- Time window là primitive, không phải ID range.
- Bucketed hash 256 bucket cố định (thay Merkle "chunk 10K sort-by-_id" của v2).
- Rate limiter + circuit breaker (v2 không có).

---

## 3. Tiered Approach v3

| Tier | Method | Frequency | Scale Cost | Action khi lệch |
|:-----|:-------|:----------|:-----------|:-----------------|
| **Tier 1** | PG: `pg_class.reltuples` estimate HOẶC exact count per window. Mongo: `countDocuments(updated_at ∈ window)` | 5 phút staggered | O(1) per table (estimate) hoặc O(log N) (indexed range count) | Alert + trigger Tier 2 |
| **Tier 2** | XOR-hash aggregate per time window (15-phút windows, freeze watermark now-5m) | Hourly per table | O(N) đọc nhưng O(1) mem/network | Drill down vào windows lệch → list IDs → heal |
| **Tier 3** | 256-bucket XOR-hash full table (recent 7d 100% + 1% sampled historical) | Daily off-peak (2-5 AM), budget 10M docs/run | Budget-gated, skip nếu vượt | Detect stale content (same count, different hash) |

**Tier 1 staggered**:
- 200 bảng chia 300 giây (5 phút) → 1 bảng / 1.5s → Mongo secondary không spike.
- Budget: `max_concurrent_checks=5`.

**Tier 2 window freeze**:
- Scan windows `[t_lo, t_hi]` với `t_hi < now - 5 phút` để tránh phantom read (CDC chưa kịp land).

**Tier 3 budget gate**:
```go
if estimated_docs > config.Recon.Tier3MaxDocsPerRun {
    // skip historical, only recent 7d
    log.Warn("Tier 3 budget exceeded, sampling only")
}
```

---

## 4. XOR-hash Aggregate — chi tiết triển khai

### Algorithm
```go
// Source Agent
func (a *SourceAgent) HashWindow(ctx context.Context, table string, tLo, tHi time.Time) (*WindowResult, error) {
    limiter := rate.NewLimiter(a.cfg.MaxDocsPerSec, int(a.cfg.MaxDocsPerSec))
    filter := bson.M{"updated_at": bson.M{"$gte": tLo, "$lt": tHi}}
    opts := options.Find().
        SetProjection(bson.M{"_id": 1, "updated_at": 1}).
        SetBatchSize(1000)
    
    cursor, err := a.coll(table).Find(ctx, filter, opts)
    if err != nil { return nil, err }
    defer cursor.Close(ctx)
    
    var (
        xor   uint64
        count int64
    )
    for cursor.Next(ctx) {
        if err := limiter.Wait(ctx); err != nil { return nil, err }
        var doc struct {
            ID        primitive.ObjectID `bson:"_id"`
            UpdatedAt time.Time          `bson:"updated_at"`
        }
        if err := cursor.Decode(&doc); err != nil { continue }
        h := xxhash.Sum64(append(doc.ID[:], []byte(doc.UpdatedAt.Format(time.RFC3339Nano))...))
        xor ^= h
        count++
    }
    return &WindowResult{Count: count, XorHash: xor}, cursor.Err()
}
```

- **Network per window**: 1 request ra (filter) + N docs (chỉ `_id` + `updated_at` = ~30 bytes × count). 
  - 1M records/window × 30 bytes = 30 MB / window → chấp nhận.
  - Nhưng với 15 phút windows và fintech traffic, 1 window ~1.8M events = 54 MB — OK.
- **Memory per window**: 2 uint64 + cursor buffer — O(1).
- **Result per window**: 16 bytes (count + hash).
- **Diff cost**: Core compare 2 × 16 bytes per window → O(W windows).

### Symmetric dest Agent
```sql
-- Postgres Dest Agent
SELECT
  COUNT(*) AS count,
  (BIT_XOR(('x' || substr(md5(id::text || COALESCE(_source_ts::text, '')), 1, 16))::bit(64)::bigint::bit(64))::bigint) AS xor_hash
FROM {table}
WHERE updated_at >= $1 AND updated_at < $2;
```

### Compare logic
```
FOR window in windows:
  if src.count != dst.count OR src.xor != dst.xor:
    mark window as "drifted"
    
FOR drifted_window:
  list_ids(src) ∪ list_ids(dst)  -- each < few K typical
  diff → missing_from_dest + missing_from_src
  → heal
```

---

## 5. 256-bucket Hash (Tier 3)

### Thay vì "sort by _id, chunk 10K"
- Bucket = first byte của xxhash(_id) → 256 bucket cố định.
- Mỗi bucket: XOR hash của (doc_id + key fields hash).
- Insert/update 1 record → ảnh hưởng **1 bucket** (16/256 bytes kết quả đổi).
- Compare: 256 × 2 × 8 bytes = 4 KB metadata → drill vào bucket lệch.

### Recent + sampled historical
```
recent_partition = [now-7d, now]  # scan 100% (hourly windows)
historical_partitions = [now-N, now-7d]  # N = 1 year
  sample_rate = 0.01  # 1% partitions/day
  run_id % 100 == 0 → scan full bucket of random 1% partitions
```

### Budget enforcement
```go
type Tier3Config struct {
    MaxDocsPerRun      int64  // 10M
    OffPeakWindow      string // "02:00-05:00"
    HistoricalSampleRate float64 // 0.01
}
```

---

## 6. Version-aware Heal (fixed)

### Migration 009 — `_source_ts`
```sql
-- Dynamic SQL generated from cdc_table_registry:
DO $$
DECLARE
  tbl RECORD;
BEGIN
  FOR tbl IN SELECT table_name FROM cdc_table_registry WHERE is_active = true
  LOOP
    EXECUTE format('
      ALTER TABLE %I ADD COLUMN IF NOT EXISTS _source_ts BIGINT;
      CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_%I_source_ts ON %I (_source_ts);
      CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_%I_updated_at ON %I (updated_at);
    ', tbl.table_name, tbl.table_name, tbl.table_name, tbl.table_name, tbl.table_name);
  END LOOP;
END $$;
```

### Worker upsert
```go
// kafka_consumer.go:150 area
sourceTS := msg.Payload.Source.TsMs  // Debezium ts_ms from message
record["_source_ts"] = sourceTS
```

### Heal UPSERT (OCC)
```sql
INSERT INTO {table} (_id, ..., _source_ts)
VALUES ($1, ..., $N)
ON CONFLICT (_id) DO UPDATE
  SET ... = EXCLUDED.*
  WHERE {table}._source_ts IS NULL 
     OR {table}._source_ts < EXCLUDED._source_ts;
```

- Nếu PG đã có row với `_source_ts` mới hơn → WHERE FALSE → không UPDATE.
- Nếu `_source_ts IS NULL` (row cũ chưa migrated) → overwrite.

### Batch heal via `$in`
```go
const healBatch = 500
for chunk := range chunks(missingIDs, healBatch) {
    docs, err := a.sourceColl.Find(ctx, bson.M{"_id": bson.M{"$in": chunk}})
    // pipeline → PG BatchUpsert với OCC
    auditLogBatched(healActions)  // flush mỗi 100
}
```

---

## 7. Heal via Debezium Signal (preferred path)

### Ưu tiên 1: Signal với `additional-conditions`
```json
{
  "type": "execute-snapshot",
  "data": {
    "data-collections": ["goopay.export_jobs"],
    "type": "incremental",
    "additional-conditions": [
      {
        "data-collection": "goopay.export_jobs",
        "filter": "updated_at >= ISODate('2026-04-15T00:00:00Z') AND updated_at < ISODate('2026-04-15T01:00:00Z')"
      }
    ]
  }
}
```

- Debezium re-snapshot **chỉ range cần**, qua Kafka → Worker xử lý bình thường → tận dụng logic CDC đã có.
- Chunk size giới hạn:
  ```properties
  # connector config
  incremental.snapshot.chunk.size=5000
  incremental.snapshot.watermarking.strategy=INSERT_INSERT
  ```

### Ưu tiên 2 (fallback): Direct fetch + UPSERT
- Khi Debezium signal không available (ví dụ Mongo source collection không có signal collection, hoặc Debezium connector down).
- Agent trực tiếp `$in` batch + PG upsert với OCC.

### Flow decision
```
mismatch_ids found
    │
    ├─► Debezium healthy? → Signal with range filter → done
    │
    └─► Debezium down? → Direct fetch + upsert (OCC) → mark audit
```

---

## 8. Kafka Hardening — BỎ compact blanket

### Đổi policy
| Topic type | Policy | Retention |
|:-----------|:-------|:----------|
| CDC data (`cdc.goopay.*`) | `delete` | `retention.ms=1209600000` (14d), `retention.bytes=107374182400` (100GB) |
| Schema History (`_schemas`, Debezium history) | `compact` | Unlimited |
| `__consumer_offsets` | `compact` (native) | Default |

### Apply script
```bash
# Inventory topic từ cdc_table_registry
for topic in $(get_cdc_topics); do
  kafka-configs --bootstrap-server kafka:9092 \
    --alter --entity-type topics --entity-name $topic \
    --add-config "cleanup.policy=delete,retention.ms=1209600000,retention.bytes=107374182400"
done
```

### Alert rule (Prometheus)
```yaml
- alert: CDCConsumerLagApproachingRetentionWindow
  expr: kafka_consumergroup_lag_seconds > 14 * 86400 * 0.7  # 70% retention
  for: 5m
  labels: { severity: warning }
- alert: CDCConsumerLagCriticalRetentionRisk
  expr: kafka_consumergroup_lag_seconds > 14 * 86400 * 0.9  # 90%
  for: 5m
  labels: { severity: critical }
```

---

## 9. DLQ — ghi TRƯỚC khi ACK Kafka

### Vấn đề v2
- ACK offset trước/sau khi ghi DLQ không rõ → có thể leak event.

### v3 flow
```go
for message := range reader.FetchMessage(ctx) {
    err := a.processMessage(ctx, message)
    if err != nil {
        // Ghi DLQ trong CÙNG TX với tracking offset
        if dlqErr := a.writeDLQ(ctx, message, err); dlqErr != nil {
            // DLQ fail → retry với backoff, KHÔNG commit offset
            metrics.DLQWriteFail.Inc()
            continue  // không CommitMessages
        }
    }
    // Chỉ commit khi process thành công HOẶC DLQ đã lưu
    reader.CommitMessages(ctx, message)
}
```

### Bảng DLQ partitioned + TTL
```sql
CREATE TABLE failed_sync_logs (
  id BIGSERIAL,
  table_name TEXT NOT NULL,
  record_id TEXT NOT NULL,
  operation TEXT,
  raw_json JSONB,
  error_message TEXT,
  retry_count INT DEFAULT 0,
  status TEXT DEFAULT 'pending',  -- pending | retrying | resolved | dead_letter
  created_at TIMESTAMPTZ DEFAULT NOW(),
  resolved_at TIMESTAMPTZ
) PARTITION BY RANGE (created_at);

-- Monthly partitions via pg_partman OR manual
-- Drop partition > 90 days (scheduled job)
```

### State machine retry
- `pending` → worker pick up → `retrying`
- Retry OK → `resolved`
- Retry fail 5 lần → `dead_letter` → alert ops
- Auto-retry: exponential backoff (1m, 5m, 30m, 2h, 6h).

---

## 10. Recon Concurrency & Leader Election

### Advisory lock (per table)
```sql
SELECT pg_try_advisory_lock(hashtext('recon_' || $table));
-- if returns false → skip run, log "previous run ongoing"
```

### Leader election (Worker multi-instance)
```go
// Redis SETNX với TTL 60s + heartbeat
lockKey := "recon:leader"
acquired, _ := redis.SetNX(ctx, lockKey, instanceID, 60*time.Second).Result()
if !acquired { return /* not leader, skip */ }
// Extend lock mỗi 20s (heartbeat)
go leaderHeartbeat(ctx, lockKey, instanceID)
```

### Run state table
```sql
CREATE TABLE recon_runs (
  id UUID PRIMARY KEY,
  table_name TEXT NOT NULL,
  tier INT NOT NULL,
  status TEXT NOT NULL,  -- running | success | failed | cancelled
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  docs_scanned BIGINT,
  windows_checked INT,
  mismatches_found INT,
  heal_actions INT,
  error_message TEXT,
  instance_id TEXT
);
CREATE UNIQUE INDEX recon_runs_one_running 
  ON recon_runs (table_name) 
  WHERE status = 'running';
```

### Jitter + staggered start
```go
jitter := time.Duration(rand.Intn(30)) * time.Second
time.Sleep(jitter)
// Cron chia đều 200 bảng trên 5 phút cửa sổ
```

---

## 11. Schema Validation — Phase A (JSON) + Phase B (Avro)

### Phase A (current JSON converter)
- Tận dụng `cdc_table_registry.expected_fields JSONB` (hoặc tương đương).
- Worker parse message → validate fields:
  ```go
  func validatePayload(payload map[string]any, registry *TableRegistry) error {
      for reqField := range registry.RequiredFields {
          if _, ok := payload[reqField]; !ok {
              return fmt.Errorf("missing required field: %s", reqField)
          }
      }
      for payloadField := range payload {
          if !registry.KnownFields[payloadField] {
              // Unknown field → DLQ + alert for schema drift
              return ErrSchemaDrift
          }
      }
      return nil
  }
  ```
- Schema drift → DLQ với `error_message="schema_drift: unknown_field=X"` → CMS UI tool "Add Field + Apply ALTER TABLE".

### Phase B (migrate to Avro)
- Switch Debezium connector config:
  ```json
  "value.converter": "io.confluent.connect.avro.AvroConverter",
  "value.converter.schema.registry.url": "http://schema-registry:8081"
  ```
- Worker dùng `go-avro` hoặc `hamba/avro` để decode.
- Schema Registry enforce compatibility (BACKWARD).
- Planned **sau Phase A stable** (ước tính 2-3 tháng).

---

## 12. Recon Observability (NEW)

### Prometheus metrics (expose qua Worker `/metrics`)
```go
var (
    ReconRunDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "cdc_recon_run_duration_seconds",
        Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800},
    }, []string{"table", "tier"})
    
    ReconMismatchCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "cdc_recon_mismatch_count",
    }, []string{"table", "tier"})
    
    ReconHealActions = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "cdc_recon_heal_actions_total",
    }, []string{"table", "action"})  // action: upsert | skip | error
    
    ReconLastSuccessTs = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "cdc_recon_last_success_timestamp",
    }, []string{"table", "tier"})
)
```

### Alerts
```yaml
- alert: ReconStale
  expr: (time() - cdc_recon_last_success_timestamp) > 7200  # 2h
  labels: { severity: warning }
  
- alert: ReconPersistentMismatch
  expr: cdc_recon_mismatch_count > 0
  for: 1h
  labels: { severity: warning }
```

---

## 13. RBAC + Audit + Idempotency

### RBAC middleware
```go
func RequireRole(role string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        user := c.Locals("user").(*User)
        if !user.HasRole(role) {
            return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
        }
        return c.Next()
    }
}

// Routes
r.Post("/api/recon/heal", RequireRole("ops-admin"), IdempotencyMiddleware, handler.Heal)
r.Post("/api/debezium/signal", RequireRole("ops-admin"), IdempotencyMiddleware, handler.Signal)
r.Post("/api/kafka/reset-offset", RequireRole("ops-admin"), IdempotencyMiddleware, handler.ResetOffset)
```

### Idempotency-Key
```go
func IdempotencyMiddleware(c *fiber.Ctx) error {
    key := c.Get("Idempotency-Key")
    if key == "" {
        return c.Status(400).JSON(fiber.Map{"error": "missing Idempotency-Key header"})
    }
    cacheKey := fmt.Sprintf("idem:%s", key)
    if cached, _ := redis.Get(ctx, cacheKey).Bytes(); cached != nil {
        return c.Status(200).Send(cached)  // replay response
    }
    // Lock để tránh double exec
    ok, _ := redis.SetNX(ctx, cacheKey+":lock", 1, 30*time.Second).Result()
    if !ok { return c.Status(409).JSON(fiber.Map{"error": "in progress"}) }
    
    err := c.Next()
    if c.Response().StatusCode() < 400 {
        redis.Set(ctx, cacheKey, c.Response().Body(), 1*time.Hour)
    }
    redis.Del(ctx, cacheKey+":lock")
    return err
}
```

### Audit log
```sql
CREATE TABLE admin_actions (
  id BIGSERIAL PRIMARY KEY,
  user_id TEXT NOT NULL,
  action TEXT NOT NULL,  -- restart_connector | reset_offset | heal | ...
  target TEXT,
  payload JSONB,
  reason TEXT,
  result TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (created_at);
```

### FE confirm modal
- Required: textarea "Lý do" min 10 chars.
- Show summary: action + target + estimated impact.

---

## 14. Tasks v3 (FINAL)

### Phase 0 — Foundation (Week 1, Quick Wins)
- [ ] **T0-1**: Migration 009 — `_source_ts` column + index cho mọi bảng trong `cdc_table_registry` (CONCURRENTLY).
- [ ] **T0-2**: Worker `kafka_consumer.go` cập nhật set `_source_ts = msg.Payload.Source.TsMs`.
- [ ] **T0-3**: Migration 010 — partition `failed_sync_logs` + `admin_actions` + `recon_runs`.
- [ ] **T0-4**: Kafka config script — set `cleanup.policy=delete, retention.ms=14d` cho CDC topics.
- [ ] **T0-5**: Worker expose `/metrics` (promauto + `promhttp.Handler()`).

### Phase 1 — Recon Agent Rewrite (Week 2)
- [ ] **T1-1**: Rewrite `recon_source_agent.go` — bỏ `GetAllIDs()`, thêm `HashWindow()`, rate limiter, readPreference=secondary.
- [ ] **T1-2**: Rewrite `recon_dest_agent.go` — thêm `HashWindow()` qua PG SQL, read-replica DSN.
- [ ] **T1-3**: Thêm `BucketHash()` (256-bucket) cho Tier 3.
- [ ] **T1-4**: Circuit breaker (sony/gobreaker) + timeout 30s/query.

### Phase 2 — Recon Core Rewrite (Week 2-3)
- [ ] **T2-1**: Rewrite `recon_core.go` — Tier 1 per-window, Tier 2 XOR-hash compare, Tier 3 budget-gated.
- [ ] **T2-2**: Advisory lock + Redis leader election + heartbeat.
- [ ] **T2-3**: `recon_runs` state table + run_id tracking.
- [ ] **T2-4**: Jitter + staggered scheduling (200 tables / 5 phút).

### Phase 3 — Heal v3 (Week 3)
- [ ] **T3-1**: OCC UPSERT dùng `_source_ts` (WHERE clause).
- [ ] **T3-2**: Heal batch `$in` size 500 + pipeline.
- [ ] **T3-3**: Heal via Debezium Signal với `additional-conditions` (preferred path).
- [ ] **T3-4**: Fallback direct fetch + upsert khi Debezium down.
- [ ] **T3-5**: Batched audit log (100 actions/insert).

### Phase 4 — DLQ + Schema Validation (Week 3)
- [ ] **T4-1**: DLQ write-before-ACK flow trong Kafka consumer.
- [ ] **T4-2**: DLQ state machine (pending → retrying → resolved | dead_letter).
- [ ] **T4-3**: Retry worker với exponential backoff.
- [ ] **T4-4**: Schema validation Phase A (JSON vs `cdc_table_registry.known_fields`).
- [ ] **T4-5**: DLQ retention drop partition > 90d (scheduled job).

### Phase 5 — Security & Idempotency (Week 4)
- [ ] **T5-1**: RBAC middleware + role `ops-admin`.
- [ ] **T5-2**: Idempotency-Key middleware (Redis TTL 1h).
- [ ] **T5-3**: Audit log table + middleware.
- [ ] **T5-4**: Rate limit destructive actions (3/h/user cho restart connector).
- [ ] **T5-5**: FE confirm modal + reason field.

### Phase 6 — Recon Observability (Week 4)
- [ ] **T6-1**: Prometheus metrics (duration, mismatch, heal, last_success).
- [ ] **T6-2**: Expose qua Worker `/metrics`.
- [ ] **T6-3**: Alert rules (stale, persistent mismatch).
- [ ] **T6-4**: Expose Recon metrics trong System Health page.

### Phase 7 — Load Test + Verify (Week 5)
- [ ] **T7-1**: Mirror prod dataset → staging Mongo + PG.
- [ ] **T7-2**: Run Recon full cycle → verify metrics không vượt Scale Budget.
- [ ] **T7-3**: Chaos test — Mongo secondary lag, PG replica down, Kafka rebalance giữa Recon.
- [ ] **T7-4**: Heal current drift dữ liệu production.

### Phase 8 — Avro Migration (Future, Month 2-3)
- [ ] **T8-1**: Kafka Connect reconfigure Avro + Schema Registry.
- [ ] **T8-2**: Worker migrate decoder (hamba/avro).
- [ ] **T8-3**: Schema compatibility testing.

---

## 15. Files Impact

### Worker (`centralized-data-service/`)
| File | Action | Change |
|:-----|:-------|:-------|
| `migrations/009_source_ts.sql` | **NEW** | `_source_ts` column + indexes |
| `migrations/010_partitioning.sql` | **NEW** | Partition failed_sync_logs, admin_actions, recon_runs |
| `internal/handler/kafka_consumer.go` | **EDIT** | Set `_source_ts`, DLQ write-before-ACK |
| `internal/service/recon_source_agent.go` | **REWRITE** | HashWindow, rate limiter, secondary read |
| `internal/service/recon_dest_agent.go` | **REWRITE** | HashWindow (SQL), replica DSN |
| `internal/service/recon_core.go` | **REWRITE** | Tier 1-3 window/budget logic, lock, run state |
| `internal/service/recon_heal.go` | **NEW** | OCC upsert, batch $in, signal preferred |
| `internal/service/schema_validator.go` | **NEW** | Phase A JSON validation |
| `internal/service/dlq_worker.go` | **NEW** | Retry worker with backoff |
| `pkgs/metrics/prometheus.go` | **EDIT** | Add Recon metrics |
| `pkgs/metrics/http.go` | **NEW** | `/metrics` HTTP endpoint |
| `pkgs/mongodb/client.go` | **EDIT** | readPreference config |
| `pkgs/postgres/replica.go` | **NEW** | Replica DSN client |
| `config/config.go` | **EDIT** | Recon budget, replica DSN, rate limits |
| `cmd/worker/main.go` | **EDIT** | Register metrics HTTP, DLQ worker, leader election |

### CMS (`cdc-cms-service/`)
| File | Action | Change |
|:-----|:-------|:-------|
| `migrations/011_admin_actions.sql` | **NEW** | Audit log table |
| `internal/middleware/rbac.go` | **NEW** | Role check |
| `internal/middleware/idempotency.go` | **NEW** | Idempotency-Key |
| `internal/middleware/audit.go` | **NEW** | Audit logging |
| `internal/api/reconciliation_handler.go` | **EDIT** | Apply RBAC + Idempotency + Audit |
| `internal/api/debezium_handler.go` | **EDIT** | Signal with range filter + RBAC |
| `internal/api/admin_actions_handler.go` | **NEW** | Query audit log |

### FE (`cdc-cms-web/`)
| File | Action | Change |
|:-----|:-------|:-------|
| `package.json` | **EDIT** | Thêm `@tanstack/react-query` |
| `src/api/client.ts` | **EDIT** | Axios interceptor thêm Idempotency-Key |
| `src/pages/DataIntegrity.tsx` | **EDIT** | Confirm modal + reason field cho heal/reset |
| `src/pages/AdminAudit.tsx` | **NEW** | Audit log viewer |
| `src/hooks/useReconStatus.ts` | **NEW** | React Query hook |

---

## 16. Definition of Done

- [ ] Recon 50M-record bảng: RAM peak < 200 MB, network < 20 MB/run (verified load test).
- [ ] Recon 500M-record extrapolation: budget-gated Tier 3 chạy được trong 2-5 AM window.
- [ ] Mongo primary CPU không spike khi Recon full run (verified metrics).
- [ ] PG replica CPU < 40% during Recon.
- [ ] Heal OCC: unit test pass — heal với ts cũ SKIP, ts mới UPSERT.
- [ ] Heal 10K IDs < 30 giây (batch $in + pipeline).
- [ ] DLQ: Kafka ACK chỉ sau khi DLQ ghi thành công (chaos test).
- [ ] Kafka retention 14d + lag alert < 70% retention.
- [ ] Destructive action require `ops-admin` + Idempotency-Key + audit (E2E test).
- [ ] Recon có metrics expose qua Prom, alert rules fired đúng scenario.
- [ ] FE confirm modal, React Query polling 30s staleWhileRevalidate.
- [ ] Load test pass với dataset mirror prod.
- [ ] Schema validation Phase A (JSON vs registry) chặn schema drift.

---

## 17. Changelog vs v2

| Aspect | v2 | v3 |
|:-------|:---|:---|
| Scale Budget | Không có | Mục 0 mandatory |
| Tier 2 | "ID Set batch 10K" mơ hồ | XOR-hash aggregate per time window (streaming, no full set) |
| Tier 3 | Flat chunk 10K MD5 (miscalled Merkle) | 256-bucket XOR-hash + time-partitioned + budget-gated |
| Heal version | `timestamp > _synced_at` (sai field) | OCC trên `_source_ts` + migration thêm cột |
| Heal fetch | Per-ID | Batch `$in` 500 pipeline |
| Heal path | Direct | Debezium Signal preferred + direct fallback |
| Kafka policy | `cleanup.policy=compact` blanket | `delete` + 14d retention + lag alert |
| Agent DB load | No throttle | Rate limiter + breaker + replica-first |
| Schema validation | Schema Registry vague | Phase A JSON vs registry, Phase B Avro roadmap |
| Signal chunk | Default | `chunk.size=5000` + `additional-conditions` filter range |
| Concurrency | None | Advisory lock + Redis leader election |
| RBAC | None | `ops-admin` role cho destructive |
| Idempotency | None | Idempotency-Key + Redis TTL 1h |
| DLQ ACK | Ambiguous | Write-before-ACK, state machine retry |
| DLQ retention | Unbounded | Partition + 90d drop |
| Recon observability | None | Prometheus metrics + alert rules |
| Audit | None | `admin_actions` partitioned table |
| FE data fetching | Axios thuần | + React Query |

---

## 18. Open Questions (non-blocking)

1. **Prometheus production server**: User nói "có Prometheus cạnh SigNoz". Explore thấy code push OTLP tới SigNoz (port 4318) nhưng không có Prom scrape config. → Cần DevOps confirm: Prom server URL + scrape target format + Prom retention config.
2. **Avro migration timeline**: Plan v3 design Phase A = JSON (current reality). Phase B = Avro. Timeline phụ thuộc business priority. → Brain/User decide.
3. **Read-replica lag tolerance**: Nếu PG replica lag > 30s, Recon compare source (Mongo secondary gần realtime) vs dest (PG replica stale) → false positive drift. → Cần policy: skip Recon nếu `pg_replication_lag > 60s`.
4. **Bảng nào là "top 10 largest"**: User confirm 50M hiện tại. → Cần list cụ thể top 10 bảng để prioritize load test + migration effort.
