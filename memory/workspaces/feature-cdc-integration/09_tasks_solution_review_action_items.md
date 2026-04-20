# Action Items — Consolidated Solution cho 2 Plan Review

> **Date**: 2026-04-17
> **Tác giả**: Brain (claude-opus-4-7)
> **Nguồn**:
>   - `10_gap_analysis_data_integrity_review.md`
>   - `10_gap_analysis_observability_review.md`
>   - `10_gap_analysis_master_summary.md`
> **Mục đích**: Action item cụ thể để Muscle thực thi sau khi Brain-User thống nhất plan v3. Mỗi task có: ID, file đụng, acceptance criteria, effort estimate.

---

## Phase A — Verification (Muscle, 0.5-1 ngày)

Trước khi rewrite plan, Muscle verify 10 assumption trong code thực tế.

| ID | Verify cái gì | Cách verify | Output |
|:---|:--------------|:------------|:-------|
| V1 | Mongo connection string có `readPreference` không, có secondary không | Grep `mongodb://` trong config files, check `recon_source_agent` (nếu có) | Ghi vào `10_gap_analysis_assumptions_verified.md` |
| V2 | PG có read-replica không, DSN thứ 2 config ở đâu | Grep `postgres://`, `DATABASE_URL`, check config-local.yml | Same |
| V3 | Prom có sẵn (port 9090?), SigNoz-only? | Check docker-compose, infra repo | Same |
| V4 | Kafka Connect converter — JSON hay Avro? | Check Debezium connector config JSON `value.converter=` | Same |
| V5 | NATS mode — Core hay JetStream? | Grep `js.Publish`, `nc.Publish`, `JetStream()` trong worker/cms code | Same |
| V6 | Mongo collection có index `updated_at`? | `db.coll.getIndexes()` cho các bảng trong `cdc_table_registry` | Same |
| V7 | PG bảng CDC có cột `_source_ts`? | `\d+ tbl_name` hoặc check migration files 001-007 | Same |
| V8 | Worker dùng OTel Kafka instrumentation? | Check `cmd/worker/main.go`, `pkgs/observability/otel.go` | Same |
| V9 | FE có React Query/SWR? | `cat cdc-cms-service-fe/package.json \| jq .dependencies` | Same |
| V10 | Infra có Redis sẵn cho CMS cache? | Grep `redis` trong config + `docker-compose.yml` | Same |

**Deliverable**: `10_gap_analysis_assumptions_verified.md` với bảng kết quả + ảnh hưởng tới plan.

---

## Phase B — Plan v3 rewrite (Brain, 1 ngày)

Sau khi Phase A xong, Brain viết:

### B.1 — `02_plan_data_integrity_v3.md`

Phải có:
1. **Mục 0 — Scale Budget**:
   - Bảng lớn nhất bao nhiêu records.
   - Expected events/sec/topic.
   - Memory/network budget per Recon run.
   - DB CPU/IO budget.
2. **Mục 1 — Assumption verified** (reference V1-V10).
3. **Rewrite Tier 2/3** theo window + XOR-hash + bucketed hash.
4. **Rewrite Heal** với `_source_ts` OCC + batch `$in`.
5. **Rewrite Kafka config** bỏ compact, dùng long retention + lag alert.
6. **Thêm** migration `_source_ts`, partition `failed_sync_logs`, Recon metrics.
7. **Thêm** throttling config, read-replica config.
8. **Changelog vs v2**: bảng rõ task nào keep/rewrite/add/remove.

### B.2 — `02_plan_observability_v3.md`

Phải có:
1. **Mục 0 — Scale Budget** (user QPS, cardinality budget, Prom/SigNoz storage).
2. **Mục 1 — Assumption verified**.
3. **Rewrite T1**: background collector + Redis cache.
4. **Rewrite T10**: percentile via Prom `histogram_quantile` hoặc in-memory T-Digest.
5. **Rewrite T13**: OTel sample theo severity + memory_limiter + fallback.
6. **Thêm** alert state machine, SLO definition.
7. **Thêm** partition activity_log.
8. **Changelog vs v2**.

**Approval gate**: User review v3 → approve/amend → Muscle mới thực thi.

---

## Phase C — Implementation Action Items

### C.1 — Quick wins (CRITICAL, tuần 1)

#### C.1.1. Obs-T10 fix — Percentile silent bug
- **File**: `cdc-cms-service/internal/api/system_health_handler.go`
- **Thay đổi**:
  - Bỏ SQL `percentile_cont` từ activity_log.
  - Thêm http client gọi Prometheus `/api/v1/query?query=histogram_quantile(0.95, ...)`.
  - Fallback (nếu Prom down): gọi Worker `/metrics` endpoint, parse histogram buckets, compute percentile in-process.
- **AC**:
  - [ ] P50/P95/P99 match giá trị Prom UI khi load test.
  - [ ] Nếu Prom down, API response có `source=fallback` flag.
- **Effort**: 4h.

#### C.1.2. Obs-T1 fix — Background collector + cache
- **File**:
  - New: `cdc-cms-service/internal/service/system_health_collector.go`
  - Edit: `cdc-cms-service/internal/api/system_health_handler.go`
- **Thay đổi**:
  - Goroutine ticker 15s, poll từng component với timeout 2s (`errgroup`).
  - Write snapshot JSON vào Redis key `system_health:snapshot` TTL 60s.
  - Handler chỉ `GET redis` → return. Thêm field `cache_age_seconds`.
- **AC**:
  - [ ] API response < 50ms p99 (không phụ thuộc external).
  - [ ] Nếu 1 component timeout, section đó `status=unknown`, không block response.
  - [ ] 100 concurrent requests không làm spike Kafka Connect REST API.
- **Effort**: 6h.

#### C.1.3. Obs-T13 fix — OTel backpressure
- **File**: `centralized-data-service/pkgs/observability/otel.go`
- **Thay đổi**:
  - Sample ratio:
    - Info: 0.1
    - Warn: 1.0
    - Error: 1.0
  - `batch` processor với `send_batch_size=512`, `timeout=5s`.
  - `memory_limiter`: `limit_mib=256`, `spike_limit_mib=64`.
  - Fallback: nếu OTel exporter return error > 10% rate / 1 phút → Zap log về console-only mode, cờ ghi Prom metric `otel_exporter_degraded`.
- **AC**:
  - [ ] Load test: SigNoz down 10 phút, Worker RAM không tăng > 50 MB.
  - [ ] Khi SigNoz up lại, log lại flow vào SigNoz bình thường.
- **Effort**: 6h.

#### C.1.4. DI-Migration `_source_ts`
- **File**:
  - New: `centralized-data-service/migrations/009_add_source_ts.sql`
- **Thay đổi**:
  ```sql
  -- For each CDC table (loop via cdc_table_registry):
  ALTER TABLE {table} ADD COLUMN IF NOT EXISTS _source_ts TIMESTAMPTZ;
  CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_{table}_source_ts ON {table}(_source_ts);
  ```
  - Generate dynamic SQL per table.
  - Worker BatchBuffer cập nhật để set `_source_ts = message.source.ts_ms`.
- **AC**:
  - [ ] Migration idempotent (IF NOT EXISTS).
  - [ ] Worker upsert có set `_source_ts` (verified với SELECT sample).
  - [ ] Index không lock table (CONCURRENTLY).
- **Effort**: 4h.

#### C.1.5. DI-Failed_sync_logs partition
- **File**: `centralized-data-service/migrations/008_reconciliation.sql` (nếu chưa apply) hoặc `009_*`.
- **Thay đổi**:
  - `failed_sync_logs` PARTITION BY RANGE (created_at), monthly.
  - TTL job: pg_cron or background goroutine drop partition > 90 days.
- **AC**:
  - [ ] INSERT phân bố vào partition đúng.
  - [ ] Drop partition cũ xóa được.
- **Effort**: 4h.

---

### C.2 — Core rewrite (HIGH, tuần 2-3)

#### C.2.1. DI-Recon rewrite Tier 2/3
- **File**:
  - Rewrite: `centralized-data-service/internal/service/recon_source_agent.go`
  - Rewrite: `centralized-data-service/internal/service/recon_dest_agent.go`
  - Edit: `centralized-data-service/internal/service/recon_core.go`
- **Thay đổi**:
  - Bỏ `GetAllIDs()` (nếu có).
  - Thêm `HashWindow(table, t_lo, t_hi) (count, xorHash, error)` — XOR-hash streaming per window.
  - Thêm `BucketHash(table, prefix)` — 256-bucket XOR hash.
  - Core orchestrate:
    - Tier 1: count per window.
    - Tier 2: if count mismatch → hash window → diff-locate → list IDs trong windows lệch.
    - Tier 3: bucketed-hash full table (off-peak, budget gated).
- **AC**:
  - [ ] Recon 50M records bảng: RAM peak < 200 MB.
  - [ ] Total network transfer < 10 MB per tier-2 run.
  - [ ] Mongo query dùng `readPreference=secondary` (verified qua log).
- **Effort**: 3 ngày.

#### C.2.2. DI-Heal OCC + batch $in
- **File**: `centralized-data-service/internal/service/recon_core.go`
- **Thay đổi**:
  - `FetchDocs(ids []string)` dùng `$in` batch 500.
  - Upsert SQL có `WHERE tbl._source_ts < EXCLUDED._source_ts`.
  - Batch audit log 100 actions/insert.
- **AC**:
  - [ ] Heal 10K IDs < 30 giây (thay vì ~1 giờ per-ID).
  - [ ] OCC chứng minh qua unit test: heal với ts cũ → skip; heal với ts mới → upsert.
- **Effort**: 1 ngày.

#### C.2.3. DI-Agent rate limit + replica
- **File**:
  - Edit: `centralized-data-service/pkgs/mongodb/client.go`
  - Edit: `centralized-data-service/config/config.go` + `config-local.yml`
  - Edit: agents.
- **Thay đổi**:
  - Config `recon.source_mongo.uri` với `readPreference=secondary`.
  - Config `recon.dest_postgres.dsn` riêng cho replica (nếu có).
  - Token bucket `golang.org/x/time/rate.NewLimiter(5000, 500)` (5K docs/s).
  - Circuit breaker: `github.com/sony/gobreaker`.
- **AC**:
  - [ ] Recon throughput bị cap đúng 5K docs/s (metric).
  - [ ] Mongo primary CPU không spike khi Recon chạy.
  - [ ] Breaker open → alert fired → Recon pause 60s → auto-resume.
- **Effort**: 1 ngày.

#### C.2.4. Obs-Activity_log single flusher multi-topic + partition
- **File**:
  - Edit: `centralized-data-service/internal/handler/kafka_consumer.go` (hoặc nơi ghi activity log).
  - New: `migrations/010_activity_log_partition.sql`.
- **Thay đổi**:
  - Single goroutine nhận events từ `chan ActivityEntry` — dùng `map[topic]*Batch` trong RAM.
  - Flush mỗi 5s hoặc total >= 1000 rows → multi-row INSERT 1 TX.
  - Migration: partition daily, TTL 30 ngày.
- **AC**:
  - [ ] Insert rate < 1/s trung bình (chứng minh).
  - [ ] Activity log query "last 10" < 50ms.
  - [ ] Drop partition cũ tự động.
- **Effort**: 1 ngày.

#### C.2.5. Obs-Alert state machine + SLO
- **File**:
  - New: `cdc-cms-service/internal/service/alert_manager.go`
  - New: `migrations/011_alerts.sql` (bảng `cdc_alerts`).
  - Doc: `07_slo_definition.md` trong workspace.
- **Thay đổi**:
  - Alert state: firing → resolved → acknowledged → silenced.
  - Fingerprint = hash(alert_name + labels) → dedup.
  - SLO thresholds: P99 latency ≤ 5s, drift ≤ 0.01%, Worker availability ≥ 99.95%.
  - Derive alert rules từ SLO.
- **AC**:
  - [ ] Alert flap (fire/resolve/fire trong 5 phút) → chỉ hiển thị 1 banner (dedup).
  - [ ] User bấm "ack" → hide 5 phút, reset khi fire lại.
  - [ ] Maintenance silence window hoạt động.
- **Effort**: 2 ngày.

---

### C.3 — Hardening (MEDIUM, tuần 4)

#### C.3.1. RBAC + audit cho destructive actions
- **File**:
  - Edit: `cdc-cms-service/internal/middleware/rbac.go` (nếu chưa có thì new).
  - New: `migrations/012_admin_actions_audit.sql`.
  - Edit: FE modals.
- **Thay đổi**:
  - Middleware require role `ops-admin` cho endpoints:
    - `POST /api/connectors/:name/restart`
    - `POST /api/debezium/signal`
    - `POST /api/kafka/reset-offset`
    - `POST /api/recon/heal`
  - FE confirm modal với "Lý do" field required.
  - Audit: INSERT `admin_actions(user_id, action, target, payload, reason, ts)`.
  - Rate limit: max 3/hour/user cho restart.
- **AC**:
  - [ ] Role không đúng → 403.
  - [ ] Audit trail query được theo user + action + date range.
  - [ ] Rate limit 3/h enforce (test bấm 4 lần → 4th = 429).
- **Effort**: 1.5 ngày.

#### C.3.2. Idempotency-Key middleware
- **File**:
  - New: `cdc-cms-service/internal/middleware/idempotency.go`
  - Edit: FE heal/retry/snapshot actions.
- **Thay đổi**:
  - Middleware check header `Idempotency-Key`.
  - Cache Redis key `idem:{key}` TTL 1h → chứa response 200 cached.
  - Nếu trùng key đang chạy → 409 Conflict + `Retry-After`.
- **AC**:
  - [ ] Click button 2 lần nhanh → chỉ 1 action thực hiện.
  - [ ] Response second click = cached 200 hoặc 409.
- **Effort**: 1 ngày.

#### C.3.3. Kafka retention config + lag alert
- **File**:
  - Edit: infra script/Helm (outside main repo?) — coordinate với DevOps.
  - New: Prom alert rule file.
- **Thay đổi**:
  - `kafka-configs --alter` cho CDC topics:
    - `retention.ms=1209600000` (14 ngày)
    - `retention.bytes=107374182400` (100 GB)
    - `cleanup.policy=delete` (rollback từ compact nếu đã apply).
  - Prom rule: `kafka_consumergroup_lag > retention_bytes * 0.7` → warning, `> 0.9` → critical.
- **AC**:
  - [ ] Topic config applied (kafka-configs --describe verify).
  - [ ] Alert fires in test khi inject lag giả.
- **Effort**: 0.5 ngày + coord DevOps.

#### C.3.4. Load test Recon với dataset mirror prod
- **Setup**: mongo-restore prod dump vào staging → chạy Recon → đo metrics.
- **Checklist**:
  - Tier 1 full run < 2 phút (200 tables).
  - Tier 2 full run < 10 phút khi no drift.
  - Tier 3 bảng 50M < 15 phút.
  - Mongo secondary CPU < 50% suốt run.
  - PG replica CPU < 50%.
  - RAM Worker < 500 MB.
- **Effort**: 2 ngày (bao gồm fix nếu fail).

---

## 4. Tổng effort

| Phase | Effort |
|:------|:-------|
| A — Verification | 0.5-1 ngày (Muscle) |
| B — Plan v3 | 1 ngày (Brain) |
| C.1 — Quick wins | 3 ngày (Muscle) |
| C.2 — Core rewrite | 8 ngày (Muscle) |
| C.3 — Hardening | 5 ngày (Muscle) + coord DevOps |
| **Total** | **17-18 ngày** |

---

## 5. Dependency graph

```
V1-V10 (Verify)
    │
    ▼
Plan v3 (Brain rewrite) ──── User Approve ────► GO
    │
    ├─► C.1.1 Obs-T10 fix ─┐
    ├─► C.1.2 Obs-T1 ──────┤
    ├─► C.1.3 Obs-T13 ─────┼── parallel OK
    ├─► C.1.4 DI-source_ts ┤
    └─► C.1.5 DI-partition ┘
    │
    ├─► C.2.1 Recon rewrite ── depends on C.1.4 (source_ts)
    ├─► C.2.2 Heal OCC ─────── depends on C.1.4
    ├─► C.2.3 Rate limit ────── parallel với C.2.1
    ├─► C.2.4 Activity log ──── independent
    └─► C.2.5 Alert SM ──────── independent
    │
    └─► C.3.1-4 Hardening ───── sau khi core xong, parallel
```

---

## 6. Definition of Done cho toàn bộ review + rewrite cycle

- [x] 2 file `10_gap_analysis_*_review.md` hoàn tất.
- [x] `10_gap_analysis_master_summary.md` hoàn tất.
- [x] `09_tasks_solution_review_action_items.md` hoàn tất (file này).
- [ ] User đọc + approve/amend gap analysis.
- [ ] Muscle chạy Phase A verify.
- [ ] Brain rewrite plan v3.
- [ ] User approve v3.
- [ ] Muscle implement theo Phase C.
- [ ] Load test pass acceptance criteria.
- [ ] 2 lesson mới ghi vào `agent/memory/global/lessons.md` (done).
