# Gap Analysis — Review `02_plan_data_integrity_final.md`

> **Date**: 2026-04-17
> **Reviewer**: Brain (claude-opus-4-7)
> **Reviewee**: Muscle/Brain cũ (claude-sonnet-4-6) — tác giả plan gốc
> **Scope**: Review logic, performance, optimization tại scale thực tế (ước tính 50 triệu records / bảng lớn, ~200 bảng, ~30 DB)
> **Verdict**: Plan có **khung đúng (Core/Agent, Merkle, DLQ, Version-aware Heal)** nhưng **nhiều chi tiết fatal ở scale thực tế**. Muscle đang viết plan ở mindset "1 triệu records" — không áp dụng cho production 50M.

---

## 0. Tổng quan severity

| # | Vấn đề | Severity | File/Task liên quan |
|:--|:-------|:---------|:--------------------|
| 1 | Tier 2 ID-set "batch 10K" — chưa định nghĩa chiến lược, dễ hiểu nhầm fetch full | **CRITICAL** | T6, T7 |
| 2 | Merkle Tree sai khái niệm (concat hash ≠ Merkle), chunk không khôi phục được | **CRITICAL** | §2, T6, T7 |
| 3 | Hash on-the-fly mỗi 24h → full scan 50M records 2 lần | **CRITICAL** | Tier 3 |
| 4 | `cleanup.policy=compact` blanket cho CDC topic — mất ordering semantics | **HIGH** | T3 |
| 5 | Version-aware Heal so sánh sai field (`_synced_at` không phải event timestamp) | **HIGH** | §4, T9 |
| 6 | Không có throttling/rate-limit trên Agent → đấm production DB | **HIGH** | T6, T7 |
| 7 | Heal fetch-per-ID thay vì `$in` batch | **HIGH** | §4 |
| 8 | `failed_sync_logs` không partition/TTL → bloat vô hạn | **MEDIUM** | T1, T11 |
| 9 | Không có concurrency lock cho Recon → race giữa 2 run | **MEDIUM** | T8 |
| 10 | Schema Registry validation nói mơ hồ, không nêu converter (JSON vs Avro) | **MEDIUM** | T13 |
| 11 | Debezium signal không throttle chunk-size → OOM broker khi snapshot 50M | **MEDIUM** | T4 |
| 12 | Không có Recon observability (metrics cho chính Recon) | **MEDIUM** | Thiếu task |
| 13 | Không có RBAC/audit cho destructive actions (Restart connector, Reset offset) | **MEDIUM** | T22 |
| 14 | Heal API không idempotent → click 2 lần = double heal | **LOW** | T14 |
| 15 | Không nêu rõ đọc từ replica (Mongo secondary, PG read-replica) | **LOW** | T5, T7 |

---

## 1. Tier 2 — "ID Set Batch 10K": cần định nghĩa lại

### Plan hiện tại nói gì
> Tier 2 | ID Set/Boundary (batch 10K) | On demand / 1h | Tìm dải ID missing → report

### Vấn đề
- "Batch 10K" là câu nói chung chung — **không định nghĩa**:
  - Fetch cái gì? Nếu `find({}, {_id:1}).batch(10K)` thì vẫn phải đi qua **toàn bộ 50M `_id`** qua network.
  - 50M × 12 bytes ObjectId = **600 MB** chỉ ID thô/chiều. × 2 (source + dest) = **1.2 GB transfer**.
  - Cost memory giữ Set trong RAM để diff: 50M × ~40 bytes (string key + overhead) ≈ **2 GB RAM** cho riêng 1 bảng.
- Nếu bảng có insert đang chạy (streaming) → set lệch do **phantom read** trong lúc scan (source thêm 1000 records khi dest scan xong trước).

### Cách làm đúng (chuẩn production)

**A. Window-based comparison (watermark-aware)**

```
FOR EACH window [t_lo, t_hi] by updated_at, width = 15 phút:
  source.count(updated_at ∈ [t_lo, t_hi])  ─┐
  dest.count(updated_at ∈ [t_lo, t_hi])    ─┴─► so sánh count
  IF mismatch:
    source.hash_agg(updated_at ∈ window)  ─┐
    dest.hash_agg(updated_at ∈ window)    ─┴─► so sánh XOR-hash
    IF mismatch:
      drill-down → list ID trong window đó (max vài nghìn IDs/window)
```

- **Freeze watermark**: chỉ scan windows có `t_hi < now - lag_tolerance` (ví dụ now−5 phút) để tránh phantom read.
- Dùng **XOR hash aggregate** (XOR-combining hash mỗi record) — associative, commutative → KHÔNG cần sort, cộng dồn streaming.

**B. Xor-hash aggregate thay vì lấy full ID set**

```go
// Source Agent
cursor := coll.Find(ctx, bson.M{"updated_at": bson.M{"$gte": t_lo, "$lt": t_hi}}, opts.SetProjection(bson.M{"_id":1, "_ts":1}))
var xorHash uint64
for cursor.Next() {
    h := xxhash.Sum64(concat(_id, _ts))
    xorHash ^= h
}
return {count, xorHash}

// Dest Agent: tương tự, read-only replica
```

- **Network**: O(1) per window (chỉ gửi về count + hash 8 bytes), thay vì O(N) IDs.
- **Diff chỉ 1 lần đi qua DB** mỗi side; không cần giữ Set trong RAM.

**C. Chỉ liệt kê IDs khi window đã mismatch**

- Source window mismatch thường < 1% → liệt kê IDs trong windows đó, chỉ vài nghìn — xử lý được.

### Tác động lên plan
- T6/T7 phải rewrite: **bỏ "lấy full ID set"**, thay bằng `hash_agg(window)`.
- Thêm field vào requirement: cần **updated_at index** trên Mongo + Postgres (cả 2 đều phải có index → kiểm tra trước).
- Thêm task: T6a — đảm bảo index `updated_at` tồn tại trên cả 2 side, nếu thiếu thì phải cảnh báo không chạy Recon (tránh full-collection-scan).

---

## 2. Merkle Tree — Hiện plan KHÔNG phải Merkle Tree

### Plan hiện tại nói gì
> Merkle Tree Hash: Chia records thành chunks (10K records mỗi chunk, sort by _id). Mỗi chunk: hash = MD5(concat(all record hashes in chunk)). So sánh chunk hashes giữa source + dest.

### Đây KHÔNG phải Merkle Tree — đây là **flat chunk hashing**

Merkle Tree thực sự là **cây băm hierarchical** cho phép **log N bisection**:

```
              root
            /      \
         h12        h34
         /  \      /  \
       h1   h2   h3   h4
       |    |    |    |
     chunk chunk chunk chunk
```

- 50M records, chunk 10K → **5000 leaves**.
- Flat chunk (như plan): **5000 hash comparisons** mỗi lần so sánh.
- True Merkle: **log₂(5000) ≈ 13 levels** → bisect root → tìm branch sai → xuống leaf. **O(log N)** khi chỉ có vài chunks sai.

### Vấn đề khác

- **Chunk boundary**: "sort by _id" — Mongo ObjectId order có thể **không giống** thứ tự ID trong Postgres nếu đã convert type (string vs bytes vs hex).
- **Chunk instability**: Insert 1 record ở giữa → toàn bộ chunks sau đó **shift** → hash đổi toàn bộ. Không có Recon "incremental", phải hash lại 50M mỗi lần.
- **MD5**: chậm (~500 MB/s single thread). 50M × avg 2 KB doc = 100 GB → 200s chỉ tính hash trên 1 side, chưa kể I/O.

### Cách làm đúng

**Option A: Bucketed hash by range of `_id` prefix (stable)**
- Chia theo **prefix của `_id`** (ví dụ first 2 bytes hex → 256 buckets cố định). Insert record không làm lệch bucket boundary.
- Mỗi bucket có XOR-hash. 256 buckets × 8 bytes = 2 KB meta-data.
- Compare 256 XOR-hashes → bucket nào lệch → drill xuống bucket đó.

**Option B: Time-partitioned Merkle (thực tế hơn cho CDC)**
- Thay "by `_id` range" bằng "by `updated_at` hourly/daily partition".
- Mỗi partition có 1 hash. Recon so sánh partition-hash.
- Partition cũ không đổi → cache hash vĩnh viễn. Chỉ recompute partition hiện tại.
- Với 50M records trải đều 1 năm → ~140K records/ngày → hash 1 ngày trong 1-2s.

**Option C: Incremental Merkle (nâng cao)**
- Maintain bảng phụ `record_hash_index(_id, hash, bucket_id)`.
- Trigger trên Postgres (ON INSERT/UPDATE → update hash).
- Worker ghi record mới → đồng thời ghi hash vào `record_hash_index`.
- Recon chỉ đọc `record_hash_index` — **không chạm** bảng chính.

### Đề xuất cho plan
- **Đổi tên** "Merkle Tree Hash" → "Bucketed XOR-hash aggregate" để tránh hiểu nhầm.
- **Chọn Option B** (time-partitioned) cho Tier 3 — realistic với data CDC.
- Tier 3 scan chỉ các partition gần (ví dụ 7 ngày gần nhất) + 1 sample random 1% partition cũ — **phát hiện drift lịch sử không nhất thiết 100%** nhưng đủ.

---

## 3. Tier 3 "Merkle 24h" — Full scan 50M là không chấp nhận

### Plan hiện tại
> Tier 3 | Merkle Tree Hash (per chunk) | 24h | Detect stale data

### Calculation ở 50M
- MongoDB scan 50M docs, avg 2 KB/doc = 100 GB read.
- Mongo secondary node I/O: giả sử 200 MB/s → **500 giây = 8 phút** chỉ đọc.
- Kèm hash compute → ~10-12 phút (1 bảng).
- Nếu có 20 bảng cỡ này × 12 phút = **4 tiếng** mỗi chu kỳ 24h → chiếm 16% CPU/IO thường xuyên của secondary.
- Nếu có bảng 500M (không phải là không thể ở fintech/payment) → 2 tiếng/bảng → vỡ trận.

### Đề xuất
- **Tier 3 không còn là "full hash 24h"** mà là **"recent-window + sampled historical"**:
  - Recent window (7 ngày): hash 100% partitions.
  - Historical: sample 1 random partition/ngày cũ mỗi run → rotation, sau 365 ngày phủ hết.
- **Schedule**: chạy off-peak (2-5 AM).
- **Budget-based**: `recon_config.max_docs_per_run = 10M` → nếu plan vượt, skip historical sampling.

---

## 4. `cleanup.policy=compact` — SAI nếu áp dụng cho toàn bộ CDC topics

### Plan hiện tại
> cleanup.policy=compact cho CDC topics (giữ latest per key)
> Worker chậm bao lâu cũng không mất data (compact giữ latest)

### Vấn đề chí mạng

**A. Mất ORDERING semantics**
- CDC event: INSERT → UPDATE(v1) → UPDATE(v2) → DELETE.
- Với `cleanup.policy=compact`: log cleaner có thể **xóa UPDATE(v1)** (vì có v2 mới hơn cùng key) trước khi Worker đọc tới. Worker đọc UPDATE(v2) thẳng — ổn.
- Nhưng nếu Worker **đang chạy** và đọc offset cũ → có thể miss event trung gian. Nếu downstream logic cần biết "field X đổi từ A sang B sang C" (audit) → mất dữ liệu intermediate.
- **Worse**: DELETE + subsequent INSERT cùng `_id` → compact có thể giữ chỉ INSERT (tombstone bị clean trước TTL) → Worker không thấy DELETE → **data ở PG bị kẹt** (bản cũ chưa xóa, bản mới UPSERT → 2 version lẫn nhau).

**B. Tombstone retention**
- `delete.retention.ms` mặc định 24h — sau đó tombstone xóa → KHÔNG thể replay delete.
- Với Worker downtime > 24h → DELETE biến mất.

**C. Không bảo vệ khỏi "retention spike"**
- Plan nói "compact = không lo retention.ms". Sai: compact log cleaner chỉ chạy sau khi segment roll + dirty ratio → nếu Worker chậm, segment hiện tại vẫn full data, compact không giúp.

### Cách làm đúng

**Chính sách đúng cho CDC**:
- `cleanup.policy=delete` (default, giữ ordering).
- `retention.ms` = **long enough** (14 ngày khuyến nghị cho CDC — bằng SLO max downtime).
- `retention.bytes` = đủ cho traffic × 14 ngày.
- Monitoring consumer lag: nếu lag sắp chạm ngưỡng "hết retention" → **page oncall** (critical).

**Khi nào dùng compact**:
- Topic dạng "state store" (Kafka Streams KTable), không phải CDC firehose.
- Debezium Schema History Topic → compact hợp lý.
- `__consumer_offsets` → compact (native).

### Đề xuất cho plan
- **T3 đổi thành**: "Configure `retention.ms=1209600000` (14 ngày) + `retention.bytes=100GB` per-topic + alert khi lag > 70% retention window".
- **Task thêm**: `T3a — Monitoring consumer lag vs retention window, alert P0 khi approaching limit`.
- **Schema History topic**: vẫn dùng compact (OK).

---

## 5. Version-aware Heal — So sánh sai field

### Plan hiện tại
```
Query Postgres: SELECT _synced_at FROM table WHERE _id = ?
Compare: MongoDB timestamp > Postgres _synced_at?
```

### Vấn đề
- `_synced_at` = thời điểm **Worker ghi PG** (wall clock của Worker). KHÔNG phải timestamp của event gốc.
- Scenario lỗi:
  1. MongoDB có event A lúc `T_mongo=10:00`, Worker chậm ghi PG lúc `T_synced=10:05`.
  2. Recon chạy lúc 11:00, fetch Mongo doc → thấy `updated_at=10:00`.
  3. So `10:00 > 10:05`? FALSE → Recon skip.
  4. Nhưng PG có data **mới hơn** từ event B (`T_mongo=10:30, T_synced=10:35`) — Recon tưởng data bình thường.
  5. Thực ra nếu data lệch, ta KHÔNG biết version nào thực sự mới hơn vì dùng sai field.

### Cách làm đúng
- **Store source event timestamp** trong PG: thêm cột `_source_ts TIMESTAMPTZ` = Debezium `source.ts_ms` (MongoDB oplog timestamp — monotonic per source).
- Heal SQL:
  ```sql
  INSERT INTO tbl (...) VALUES (...)
  ON CONFLICT (_id) DO UPDATE
    SET ... 
    WHERE tbl._source_ts < EXCLUDED._source_ts;  -- OCC
  ```
- So sánh **source_ts vs source_ts**, không phải source vs wall-clock.

### Đề xuất cho plan
- **Task thêm** T9a: Migration bổ sung cột `_source_ts` vào mọi bảng CDC (nếu chưa có).
- **T9 rewrite**: Heal UPSERT sử dụng OCC trên `_source_ts`, không phải `_synced_at`.
- **Worker hiện tại** có ghi `_source_ts` không? → cần verify code trước.

---

## 6. Agent không có throttle — đấm production DB

### Plan hiện tại
- Source Agent scan Mongo trực tiếp. Dest Agent scan Postgres trực tiếp.
- Không nêu: read preference, rate limit, query budget.

### Vấn đề
- Recon chạy đồng thời 10 bảng × 50M records scan → Mongo primary CPU spike → **affecting production writes**.
- Postgres connection pool exhaust nếu Agent mở 10 connections × 10 tables.

### Đề xuất bổ sung

**A. Read preference**
- MongoDB Agent: **bắt buộc** `readPreference=secondary` hoặc `secondaryPreferred`.
- Postgres Agent: **bắt buộc** đọc từ read-replica (nếu có), hoặc `SET TRANSACTION READ ONLY`.

**B. Rate limit / Query budget**
- Agent có config:
  ```yaml
  recon:
    max_concurrent_tables: 2
    max_docs_per_second: 5000
    batch_size: 1000  # cursor batchSize
    query_timeout: 30s
  ```
- Dùng `golang.org/x/time/rate` (token bucket) trong Agent.

**C. Circuit breaker**
- Nếu query p99 > 5s → pause 60s → retry. Tránh cascade failure.

### Đề xuất cho plan
- **Task thêm**: `T5a — Configure Mongo read preference + rate limiter cho Agent`.
- **Config**: thêm section `recon:` trong `config-local.yml` với các field trên.

---

## 7. Heal — fetch per-ID thay vì batch `$in`

### Plan hiện tại (§4)
> Fetch full document từ MongoDB (với timestamp)

### Ngầm định
- Loop qua missing IDs → `coll.FindOne({_id: id})` mỗi lần = **N round-trips**.
- 10K missing IDs × 5ms RTT = **50 giây** chỉ network.

### Cách làm đúng
```go
const batchSize = 500
for chunk := range chunks(missingIDs, batchSize) {
    cursor := coll.Find(ctx, bson.M{"_id": bson.M{"$in": chunk}})
    for cursor.Next() { ... upsert ... }
}
```
- 10K IDs / 500 = 20 round-trips → **100ms thay vì 50s**.

### Đề xuất cho plan
- §4 (Version-aware Heal) thêm dòng: "Fetch by `$in` batch 500, pipelined với PG upsert".

---

## 8. `failed_sync_logs` — table bloat

### Plan hiện tại (T1, T11)
- CREATE TABLE + INSERT on error. Không partition, không TTL, không archive.

### Scale thực tế
- Failure rate giả sử 0.01% × 50M × 200 bảng × ghi lại mỗi lần = không tưởng tượng nổi nếu có schema drift nghiêm trọng.
- 1 day × 1% failure in throughput 500 events/sec = 432K rows/day → 158M rows/năm / bảng `failed_sync_logs`.

### Đề xuất
- **Partition theo thời gian** (monthly): `PARTITION BY RANGE (created_at)`.
- **TTL job**: drop partition > 90 ngày (hoặc archive ra S3/cold storage).
- **Index**: `(table_name, status, created_at)` — query pattern retry screen.
- **Status state machine**: `pending → retrying → resolved | dead_letter`. Plan hiện không nêu state transition rõ.

### Đề xuất cho plan
- T1 migration: partition table + retention policy (pg_partman hoặc native declarative partitioning).
- Thêm task: `T11a — Failed log retention job (drop partition > 90 days)`.

---

## 9. Recon concurrency — không có lock

### Plan hiện tại
- T8 `recon_core.go` orchestrate + schedule. Không mention: nếu schedule run 1 đang chạy, run 2 tới thì sao?

### Vấn đề
- Cron chạy mỗi 5 phút, một run Tier 2 mất 10 phút → 2 run overlap → double scan → double heal → Audit Log lộn xộn.

### Đề xuất
- **Advisory lock** Postgres: `pg_try_advisory_lock(hashtext('recon_'||table_name))` trước khi chạy.
- **Run state table**: `recon_run(id, table_name, status, started_at, finished_at)` với unique constraint `(table_name, status='running')`.
- Nếu lock fail → skip run với log "previous run still ongoing".

### Đề xuất cho plan
- Task thêm: `T8a — Recon run state table + advisory lock`.

---

## 10. Schema Registry validation — mơ hồ

### Plan hiện tại (T13)
> Worker check schema version từ Kafka message header

### Vấn đề
- **Debezium converter** mặc định cho MongoDB source = **JSON** (không có Schema Registry).
- Schema Registry chỉ có khi dùng **Avro/Protobuf converter** (Confluent). Plan không nêu.
- Header `schema.id` chỉ có với Avro.

### Câu hỏi cần trả lời
- Dự án hiện dùng JSON converter hay Avro?
- Nếu JSON → không có Schema Registry → phải dùng cách khác (ví dụ: snapshot schema trong `cdc_table_registry.current_schema_version`, so sánh với payload).

### Đề xuất cho plan
- **T13 rewrite**: "Schema validation — compare payload fields against `cdc_table_registry.expected_fields`. Nếu field mới xuất hiện → DLQ + alert; nếu field thiếu mà NOT NULL → DLQ".
- Không nhất thiết phải Schema Registry nếu đã có registry internal.

---

## 11. Debezium Signal — không throttle

### Plan hiện tại (T4)
```json
{"type": "execute-snapshot", "data": {"data-collections": [...], "type": "incremental"}}
```

### Vấn đề
- Trigger snapshot bảng 50M → Debezium đẩy 50M events → Kafka topic full → Worker lag.
- Debezium `incremental.snapshot.chunk.size` mặc định = 1024. Với 50M → 48K chunks → 6 giờ snapshot straight.
- Không có khái niệm "resumable" giữa chừng.

### Đề xuất
- Config Debezium:
  ```
  incremental.snapshot.chunk.size=5000
  incremental.snapshot.watermarking.strategy=INSERT_INSERT  # hoặc INSERT_DELETE
  ```
- Signal với `additional-condition` để chỉ snapshot phần cần:
  ```json
  {
    "type": "execute-snapshot",
    "data": {
      "data-collections": ["db.coll"],
      "type": "incremental",
      "additional-conditions": [{"data-collection": "db.coll", "filter": "updated_at > ISODate('2026-04-10')"}]
    }
  }
  ```
- Kết hợp với Recon: chỉ snapshot range `updated_at` mà Recon detect drift, không phải toàn bộ.

### Đề xuất cho plan
- **T4 rewrite**: Signal với `additional-conditions` để filter range từ Recon output.
- **Task thêm** `T18a — Heal orchestrator: Recon báo lệch range → ghi signal có filter → Debezium re-snapshot chỉ range đó`.

---

## 12. Recon observability — thiếu metrics

### Plan hiện tại
- Không có task riêng cho Recon metrics.

### Cần có
- `cdc_recon_run_duration_seconds{table, tier}` histogram.
- `cdc_recon_mismatch_count{table, tier}` gauge.
- `cdc_recon_heal_actions_total{table, action="upsert|skip"}` counter.
- `cdc_recon_last_success_timestamp{table}` gauge → alert nếu quá cũ.

### Đề xuất
- **Task thêm** `T8b — Recon Prometheus metrics (expose qua Worker /metrics)`.
- Bổ sung vào System Health page (plan observability).

---

## 13. RBAC + Audit cho destructive action

### Plan hiện tại (T22)
- FE buttons: Reset Debezium offset, Trigger snapshot, Reset Kafka offset.

### Vấn đề
- Không nêu auth check. Nếu JWT hiện tại bất kỳ user nào có thể bấm → **ai reset cũng được**.
- Không có audit log → không biết ai reset, lúc nào.

### Đề xuất
- **Role "ops-admin"** required cho các action này.
- Audit: mỗi bấm → ghi `cdc_admin_actions(user_id, action, target, payload, timestamp)`.
- Frontend: confirm modal với lý do (ghi vào audit).

---

## 14. Heal API idempotency

### Plan hiện tại (T14)
> API (report, check tiers, heal, failed logs, retry)

### Vấn đề
- `POST /heal?table=X` không có request ID. Click 2 lần → heal chạy song song → race.

### Đề xuất
- Endpoint nhận `Idempotency-Key` header (UUID từ FE).
- Server dedup dựa key (cache Redis 1h).
- Response include `run_id` để FE track.

---

## 15. Đọc từ replica

### Plan hiện tại
- Không nêu connection string riêng cho Recon.

### Đề xuất
- Config:
  ```yaml
  recon:
    source_mongo:
      uri: "mongodb://secondary.mongo:27017/?readPreference=secondary"
    dest_postgres:
      dsn: "host=pg-replica.internal user=readonly..."
  ```
- Worker CDC dùng primary write; Recon/Agent dùng replica → isolation.

---

## 16. Flow review — từng API/luồng

### 16.1. Recon Tier 1 (Count) — flow
```
┌─────────────────────────────────────────────────────────┐
│ Core (cron 5m)                                          │
│   │                                                     │
│   ├─► Source Agent: db.coll.estimatedDocumentCount()    │  ⚠️ NOT accurate (caches)
│   │                                                     │
│   └─► Dest Agent: SELECT COUNT(*) FROM tbl              │  ⚠️ Full scan, slow on 50M
│                                                         │
│   Compare → if |Δ| > threshold → trigger Tier 2         │
└─────────────────────────────────────────────────────────┘
```

**Vấn đề**:
- `estimatedDocumentCount()` không accurate (có thể lệch với `countDocuments()` — metadata cache).
- `SELECT COUNT(*)` trên PG 50M: **full index scan** ~5-30s (tùy tablescan vs index-only). Không rẻ.
- Với 200 bảng × mỗi 5 phút → 200 × 12 × 60 = 2400 count queries/giờ. PG replica chịu không nổi.

**Đề xuất**:
- Tier 1 dùng `countDocuments({})` cho Mongo (accurate) HOẶC dùng `estimatedDocumentCount` với **threshold cao hơn** (chấp nhận 0.1% drift).
- PG: dùng `pg_class.reltuples` (statistics estimate, ~1ms) cho Tier 1 — chỉ estimate, KHÔNG exact. Nếu lệch → Tier 2 mới exact count.
- Schedule staggered: 200 bảng không chạy cùng lúc, spread qua 5 phút.

### 16.2. Recon Tier 2 (ID Set) — flow đã phân tích ở §1

### 16.3. Heal flow
```
Core: mismatch_ids = diff(source.ids, dest.ids)
  │
  ├─► FOR id in mismatch_ids:
  │     Mongo.FindOne({_id})  ⚠️ per-ID network roundtrip
  │     Compare ts
  │     PG UPSERT
  │     Audit log
  │
  └─► Done
```

**Đã nêu vấn đề** ở §5, §7. Gom lại:
1. Batch `$in` 500.
2. Pipeline: fetch song song upsert.
3. OCC trên `_source_ts`.
4. Batch audit log (gom mỗi 100 heal action → 1 insert).

### 16.4. DLQ flow
```
Worker consume Kafka
  ├─► Try map + UPSERT
  ├─► Error? → INSERT failed_sync_logs + ACK offset
  └─► Success → ACK offset
```

**Vấn đề ngầm định**:
- **ACK offset sau khi ghi DLQ** → nếu DLQ insert fail (PG down) → mất event vì đã ACK.
- **Đúng**: ghi DLQ phải ở TX chung với UPSERT chính (2PC hoặc outbox pattern), hoặc retry-forever-until-DLQ-succeed trước khi ACK.

**Đề xuất**:
- Outbox pattern: DLQ là bảng Postgres trong **cùng DB** với data table → 1 transaction:
  ```sql
  BEGIN;
    INSERT INTO failed_sync_logs (...);
    -- (không có UPSERT data, vì đã fail)
  COMMIT;
  -- Nếu COMMIT fail → retry trước khi ACK Kafka.
  ```
- Nếu DLQ và data khác DB → dùng retry + fallback local disk buffer.

### 16.5. Schedule / Cron
- Plan: "Tier 1 — 5 min, Tier 2 — 1h, Tier 3 — 24h".
- Không nêu: **jitter**, **staggered start**, **leader election** (nếu multi-instance Worker/Core).

**Đề xuất**:
- Leader election qua Redis SETNX hoặc Postgres advisory lock → chỉ 1 instance chạy Recon.
- Jitter ±30s để tránh thundering herd.
- Rolling schedule: bảng 1 chạy 00:00, bảng 2 chạy 00:05 Tier 1, ... — spread.

---

## 17. Tasks ĐỀ XUẤT BỔ SUNG (ngoài plan hiện tại)

| ID | Task | Lý do |
|:---|:-----|:------|
| T3a | Monitor consumer lag vs retention window, P0 alert | Thay thế compact |
| T5a | Mongo secondary read preference + rate limiter | Bảo vệ prod |
| T6a | Verify `updated_at` index trên cả 2 side | Điều kiện cho window-based Recon |
| T8a | Recon run state + advisory lock | Concurrency |
| T8b | Recon Prometheus metrics | Observability |
| T9a | Migration thêm `_source_ts` cột vào bảng CDC | OCC đúng |
| T11a | Failed log retention/partition | Chống bloat |
| T14a | Heal API idempotency key | UX + safety |
| T18a | Heal via Debezium signal với range filter | Scalable re-snapshot |
| T22a | RBAC + audit cho destructive actions | Security |

---

## 18. Tasks ĐỀ XUẤT REWRITE

| ID | Plan cũ | Plan mới |
|:---|:--------|:---------|
| T3 | `cleanup.policy=compact` toàn bộ CDC topics | `retention.ms=14d, retention.bytes=sized` + alert lag > 70% retention |
| T6 | Agent trả count + **full ID set** + Merkle | Agent trả count + **XOR-hash per window** + bucketed-hash (drill only on mismatch) |
| T7 | Tương tự T6 (dest) | Tương tự T6 |
| T9 | `timestamp > _synced_at` | OCC trên `_source_ts` + `$in` batch 500 |
| T13 | Schema Registry check | Schema check vs `cdc_table_registry.expected_fields` (project có registry internal) |

---

## 19. Check list "Staff Engineer có duyệt PR này?"

- [ ] Không có full-scan 50M trong luồng normal-path → chưa đạt (Tier 2/3 hiện tại).
- [ ] Mọi destructive action có RBAC + audit → chưa đạt.
- [ ] Mọi external query có timeout + rate limit → chưa đạt.
- [ ] DLQ ghi trước khi ACK Kafka, không leak event → chưa đạt (plan không rõ).
- [ ] Heal OCC đúng semantics (source_ts vs source_ts) → chưa đạt.
- [ ] Recon có observability của chính nó → chưa đạt.
- [ ] Table bloat có retention plan → chưa đạt.
- [ ] Concurrency/lock cho scheduled job → chưa đạt.
- [ ] Idempotent API → chưa đạt.

**Kết luận**: plan hiện tại **không pass Staff Review**. Phải rewrite các task T3, T6, T7, T9, T13 và bổ sung 10 task mới trước khi implement.

---

## 20. Action Items

1. **Brain (tôi)**: tài liệu này là review, KHÔNG chạm code.
2. **User**: review thống nhất các thay đổi → Brain cập nhật `02_plan_data_integrity_final.md` v3 (có changelog).
3. **Muscle**: chỉ implement sau khi plan v3 được user approve.
4. **Pre-flight**: trước khi code, verify:
   - Có index `updated_at` trên Mongo + PG?
   - Dùng Debezium JSON hay Avro converter?
   - Có Postgres read-replica không?
   - Có MongoDB secondary không?
   - Traffic thực tế (events/sec) per topic?
   - Kích thước bảng thực tế (50M là ước tính — có bảng nào lớn hơn?)

---

## 21. Ghi chú cuối

Plan gốc cho thấy tác giả hiểu concept (Merkle, DLQ, Version-aware, Signal) nhưng **chưa calibrate cho scale** và **chưa check implementation detail**. Lỗi thường gặp khi LLM plan ở mức "book-example" mà không chạm prod numbers.

**Lesson (để ghi vào `agent/memory/global/lessons.md`)**:
> **Global Pattern [A lập plan B áp dụng X records] → Result Y**: Khi A (AI) lập plan cho hệ thống data B với quy mô X (> 10M records), phải luôn tính toán Y = [memory footprint, network transfer, DB load, latency] cho từng operation. Nếu Y > ngưỡng production chấp nhận → plan cần rewrite theo hướng: window-based, sampled, incremental, hash-aggregate. **Đúng**: Plan phải bao gồm bảng "scale calculation" trước khi task list.
