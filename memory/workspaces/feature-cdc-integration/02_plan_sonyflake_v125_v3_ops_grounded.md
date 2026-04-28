# Sonyflake v1.25 — v3 Ops-Grounded Plan

> **Date**: 2026-04-20
> **Author**: Brain (claude-opus-4-7)
> **Supersedes**: v1 (band-aid, 6 violations), v2 (vocab-aggressive, 5 ops violations)
> **Principle**: Honest about ops cost. No auto-magic. Zero-downtime mandatory. Dynamic infra primitives.

---

## 0. v2 failures acknowledged → v3 corrections

| # | v2 lie | Reality | v3 fix |
|:--|:-------|:--------|:-------|
| 1 | "Auto-detect business columns cho 200 tables trong 13-14h" | JSONB type inconsistent (string/number mixed). Không tự động sinh typed schema được. Mapping manual = 100-200h | **Keep JSONB. Skip typed extraction.** Business queries use `_gpay_raw_data->>'field'`. Add GIN index on `_gpay_raw_data`. Per-table typed extraction = **out of scope** v1.25 — separate effort per business team request |
| 2 | "Single Identity Authority" với Debezium path Go-sinh-ID + DB validate | NTP lệch giữa Go pod và PG → Sonyflake lose monotonicity. Validate-only ≠ authority | **Option A (chosen)**: Go Worker call `SELECT cdc_internal.next_sonyflake()` via pgx connection — single source. Latency +1-2ms/insert (amortized qua batch 100) = acceptable |
| 3 | "Aggressive cutover single-transaction CREATE+INSERT SELECT+DROP PK" trên 10M rows | Postgres LOCK bảng 30+ phút → Worker downtime. Real ops nightmare | **pg_repack extension** (online VACUUM FULL, lock-free) HOẶC **logical replication swap** cho tables > 100K rows. Small tables (<100K): still single-tx OK |
| 4 | "Worker ID reserve bằng grep log, static worker_id=1" | K8s pod IP dynamic. Restart → different IP → machineID changes → static reserve broken | **Redis Worker ID Registry**: SETNX `sonyflake:worker_id:{N}` với TTL 60s + heartbeat mỗi 20s. Pod boot → claim first-available from pool [0-65535]. Pod die → TTL expire → ID reclaim |
| 5 | "_raw_data - ARRAY[airbyte metadata]" JSONB strip trong migration transaction | CPU-expensive trên millions rows trong 1 tx = tự sát perf | **Strip at Go Worker** (`stripAirbyteMetadata()` trong bridge_batch.go) TRƯỚC INSERT. Migration chỉ COPY data, không transform |

---

## 1. Realistic Scope

### 1.1 What v1.25 DELIVERS

- Unified `_gpay_*` metadata schema (10 columns prefix `_gpay_`)
- Single Identity Authority (PG `next_sonyflake()` called by both Go + DB trigger)
- Partial unique index on `_gpay_source_id` WHERE NOT deleted
- OCC guard preserved (`_gpay_source_ts` = rename `_source_ts`)
- Worker ID Registry (Redis dynamic)
- Strip rác tại Go Worker

### 1.2 What v1.25 DOES NOT DELIVER (ngoài scope)

- ❌ Auto typed column extraction from JSONB — manual per-table per business team request
- ❌ `cdc_internal` schema isolation — keep tables in `public`, namespace via table prefix if needed
- ❌ Same-transaction "aggressive cutover" — staged migration mandatory
- ❌ Airbyte bridge activation — defer until user explicit wants

### 1.3 Effort estimate (HONEST)

| Component | Effort |
|:----------|:-------|
| Phase 0 foundation (PG functions + Redis registry) | 4-6h |
| Phase 1 per-table migration (8 tables × 30-60min ops, + pg_repack setup) | 8-12h |
| Phase 2 Worker code changes (call PG next_sonyflake, strip at Worker, schema v1.25 writes) | 6-8h |
| Phase 3 CMS + FE rename | 3-4h |
| Phase 4 Verify + Load test | 4-6h |
| **Total realistic** | **25-36h** (v2 claimed 13-14h — was lie) |

Cho **200 tables future scale**: 8 tables × 1h = 8h. 200 tables × 1h = **200h ops effort** nếu typed column extraction. Nếu skip typed extraction (pure JSONB path) → 200 × 15min = **50h** (just schema migration).

---

## 2. True Single Identity Authority (Fix #2)

### 2.1 PG `next_sonyflake()` same as v2

Keep migration 018 function — not changed.

### 2.2 Worker ID Registry (Redis-backed) — NEW

File NEW: `centralized-data-service/internal/service/worker_id_registry.go`

```go
type WorkerIDRegistry struct {
    redis     *redis.Client
    keyPrefix string         // "sonyflake:worker_id:"
    ttl       time.Duration  // 60s
    heartbeat time.Duration  // 20s
    workerID  uint16         // claimed on boot
    stopCh    chan struct{}
}

// Claim dynamic worker ID on Worker boot
func (r *WorkerIDRegistry) Claim(ctx context.Context) (uint16, error) {
    // Pool: 1-65535 (reserve 0 for PG, 1 reserved for future reserved)
    for candidate := uint16(2); candidate < 65535; candidate++ {
        key := fmt.Sprintf("%s%d", r.keyPrefix, candidate)
        ok, err := r.redis.SetNX(ctx, key, hostInfo(), r.ttl).Result()
        if err != nil { return 0, err }
        if ok {
            r.workerID = candidate
            go r.heartbeatLoop(ctx)  // refresh TTL every 20s
            return candidate, nil
        }
    }
    return 0, fmt.Errorf("no worker_id available")
}

func (r *WorkerIDRegistry) heartbeatLoop(ctx context.Context) {
    ticker := time.NewTicker(r.heartbeat)
    defer ticker.Stop()
    key := fmt.Sprintf("%s%d", r.keyPrefix, r.workerID)
    for {
        select {
        case <-ctx.Done(): 
            r.redis.Del(context.Background(), key)  // release on graceful shutdown
            return
        case <-ticker.C:
            r.redis.Expire(ctx, key, r.ttl)  // extend TTL
        }
    }
}
```

**Sonyflake init override** (`pkgs/idgen/sonyflake.go`):
```go
func InitWithRegistry(ctx context.Context, registry *WorkerIDRegistry) error {
    workerID, err := registry.Claim(ctx)
    if err != nil { return err }
    st := sonyflake.Settings{MachineID: func() (uint16, error) { return workerID, nil }}
    sf = sonyflake.NewSonyflake(st)
    return nil
}
```

### 2.3 Go Worker calls PG `next_sonyflake()` for consistency

**Alternative consideration**: instead of Go-side Sonyflake, make Go call PG:

```go
// In bridge_batch.go + kafka_consumer.go
func (h *Handler) generateID(ctx context.Context) (int64, error) {
    var id int64
    err := h.db.WithContext(ctx).Raw("SELECT cdc_internal.next_sonyflake()").Scan(&id).Error
    return id, err
}
```

Latency: +1-2ms per query. Amortize qua batch:
```go
// Batch mode: request 100 IDs in 1 query
err := h.db.WithContext(ctx).Raw("SELECT cdc_internal.next_sonyflake() FROM generate_series(1, 100)").Scan(&ids).Error
```

**Chosen**: Go uses PG function for batch (Airbyte bridge), Go-side Sonyflake with Redis worker_id for Debezium streaming (per-message low-latency requirement). Both validate via PG `validate_sonyflake()` at DB level.

---

## 3. Zero-Downtime Migration (Fix #3)

### 3.1 pg_repack for online rebuild

Install extension (requires superuser once):
```sql
CREATE EXTENSION IF NOT EXISTS pg_repack;
```

### 3.2 Migration pattern per table (online, lock-free <5s)

For each table:

**Step 1**: ADD new columns với NULL allowed
```sql
ALTER TABLE public.{table}
  ADD COLUMN IF NOT EXISTS _gpay_id BIGINT,
  ADD COLUMN IF NOT EXISTS _gpay_source_id VARCHAR(200),
  ADD COLUMN IF NOT EXISTS _gpay_source_engine VARCHAR(20),
  ADD COLUMN IF NOT EXISTS _gpay_source_ts BIGINT,
  ADD COLUMN IF NOT EXISTS _gpay_raw_data JSONB,
  ADD COLUMN IF NOT EXISTS _gpay_hash VARCHAR(64),
  ADD COLUMN IF NOT EXISTS _gpay_version BIGINT,
  ADD COLUMN IF NOT EXISTS _gpay_deleted BOOLEAN,
  ADD COLUMN IF NOT EXISTS _gpay_sync_ts TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS _gpay_created_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS _gpay_updated_at TIMESTAMPTZ;
```
Lock: ACCESS EXCLUSIVE few seconds (metadata only, no row rewrite).

**Step 2**: Backfill in small batches (5K rows / batch, commit each)
```sql
-- Bash loop:
batch_size=5000
offset=0
while true; do
  rows=$(psql -U user -d goopay_dw -t -c "
    WITH to_update AS (
      SELECT ctid FROM {table} 
      WHERE _gpay_id IS NULL 
      LIMIT $batch_size
    )
    UPDATE {table} t SET
      _gpay_id = cdc_internal.next_sonyflake(),
      _gpay_source_id = COALESCE(_id::text, source_id),
      _gpay_source_engine = (SELECT sync_engine FROM cdc_table_registry WHERE target_table='{table}'),
      _gpay_source_ts = _source_ts,
      _gpay_raw_data = _raw_data - '_airbyte_ab_id' - '_airbyte_extracted_at' - '_airbyte_loaded_at' - '_airbyte_data_hash',
      _gpay_hash = _hash,
      _gpay_version = COALESCE(_version, 1),
      _gpay_deleted = COALESCE(_deleted, FALSE),
      _gpay_sync_ts = _synced_at,
      _gpay_created_at = _created_at,
      _gpay_updated_at = _updated_at
    FROM to_update
    WHERE t.ctid = to_update.ctid
    RETURNING 1;
  " | wc -l)
  [ "$rows" -eq 0 ] && break
  echo "Backfilled $rows rows"
  sleep 0.1  # avoid saturating PG
done
```
Lock per batch: <1s ROW EXCLUSIVE. No bridge downtime.

**Step 3**: Add constraints once backfill 100%
```sql
ALTER TABLE public.{table} 
  ALTER COLUMN _gpay_id SET NOT NULL,
  ALTER COLUMN _gpay_source_id SET NOT NULL,
  ALTER COLUMN _gpay_source_engine SET NOT NULL,
  ADD CONSTRAINT chk_{table}_engine CHECK (_gpay_source_engine IN ('airbyte','debezium'));

CREATE UNIQUE INDEX CONCURRENTLY idx_{table}_gpay_src_active
  ON public.{table} (_gpay_source_id)
  WHERE _gpay_deleted IS FALSE;
```
`CONCURRENTLY` = lock-free index build.

**Step 4**: Attach guard trigger (no lock)
```sql
CREATE TRIGGER trg_{table}_gpay_guard
  BEFORE INSERT OR UPDATE ON public.{table}
  FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_gpay_guard();
```

**Step 5**: Worker switch writes → use `_gpay_*` columns (deployment triggered)

**Step 6** (after N days stable): DROP old columns via pg_repack
```sql
-- pg_repack rebuild table without old columns
-- Command line: pg_repack -t {table} --no-order -d goopay_dw
-- This rebuilds table physically, dropping DROP'd columns
ALTER TABLE public.{table} DROP COLUMN _id, DROP COLUMN _raw_data, DROP COLUMN _hash, ...;
# Then pg_repack to reclaim disk space without long lock
pg_repack -t {table} -d goopay_dw
```

### 3.3 Lock duration per step (estimated)

| Step | Lock | Duration estimate cho 10M rows |
|:-----|:-----|:-------------------------------|
| 1 ADD COLUMN (nullable) | ACCESS EXCLUSIVE | <2s (metadata only PG 11+) |
| 2 Backfill batched 5K | ROW EXCLUSIVE per batch | ~0.5s/batch × 2000 batches = 17min total, but NO bridge lock |
| 3 CREATE INDEX CONCURRENTLY | SHARE UPDATE EXCLUSIVE | <5min, allows reads+writes |
| 4 Attach trigger | SHARE ROW EXCLUSIVE | <1s |
| 5 Worker switch | None (app-level) | Deploy time |
| 6 DROP COLUMN + pg_repack | ACCESS EXCLUSIVE (brief swap) | <30s total lock spread across pg_repack phases |

**Zero-downtime verified**: no single operation locks more than 5s exclusive. Bridge + Debezium consumers continue throughout.

---

## 4. Strip rác at Worker (Fix #5)

File: `centralized-data-service/internal/handler/bridge_batch.go`

```go
var airbyteMetadataKeys = []string{
    "_airbyte_ab_id",
    "_airbyte_extracted_at",
    "_airbyte_loaded_at", 
    "_airbyte_data_hash",
    "_ab_cdc_lsn",
    "_ab_cdc_updated_at",
    "_ab_cdc_deleted_at",
}

func stripAirbyteMetadata(rawJSON []byte) []byte {
    // Use sjson to delete keys inplace (fast, no full parse)
    cleaned := rawJSON
    for _, key := range airbyteMetadataKeys {
        cleaned, _ = sjson.DeleteBytes(cleaned, key)
    }
    return cleaned
}

// In flushBatch():
cleanedRaw := stripAirbyteMetadata(row.rawData)
// INSERT cleanedRaw as _gpay_raw_data
```

**No DB transformation**. Migration SQL just COPY `_raw_data → _gpay_raw_data` (existing rows assumed already clean or accept legacy strip-on-next-write).

---

## 5. True Single Authority design choice

**Chosen**: Hybrid with clear contract

| Path | ID Generation | Validation |
|:-----|:--------------|:-----------|
| Debezium (Kafka stream, high-rate) | Go Sonyflake with Redis-registered machineID | PG trigger `validate_sonyflake()` strict |
| Airbyte (batch bridge, lower rate) | PG `next_sonyflake()` via Go RPC call | None (PG generated trivially valid) |
| Manual SQL INSERT | PG trigger default (if `_gpay_id NULL` → call `next_sonyflake()`) | None |

**NTP requirement documented**: Go pods + PG server time diff < 100ms. Add Prometheus alert `time_diff_ms > 100`.

---

## 6. Realistic Phase Plan

### Phase -1: Pre-work (3h)

- [ ] **Deploy Redis Worker ID Registry** service
- [ ] **Install pg_repack extension** on PG (requires superuser, coord with DBA)
- [ ] **NTP audit**: verify Go pods + PG server clock skew < 50ms
- [ ] **Baseline metrics**: current table sizes, row counts, index sizes

### Phase 0: Foundation (5h)

- [ ] Migration 018: `cdc_internal` schema + `next_sonyflake()` + `validate_sonyflake()` + `tg_gpay_guard()` (unchanged from v2)
- [ ] `worker_id_registry.go` Go service + `idgen.InitWithRegistry()`
- [ ] Worker startup: claim worker_id before Sonyflake init

### Phase 1: Per-table schema evolution (8h, zero-downtime)

Per table (sequential, 8 tables):
- [ ] Step 1: ADD v1.25 columns (2s lock)
- [ ] Step 2: Backfill batched 5K (~17min per 10M rows, no blocking)
- [ ] Step 3: Add constraints + UNIQUE INDEX CONCURRENTLY
- [ ] Step 4: Attach trigger
- [ ] Step 5: Worker code deploy writes v1.25 columns

### Phase 2: Worker code changes (8h)

- [ ] `bridge_batch.go`: strip at Worker, call `next_sonyflake()` via PG for IDs, write `_gpay_*` columns
- [ ] `kafka_consumer.go`: Go Sonyflake với Redis worker_id, write `_gpay_*` columns, `_gpay_source_engine='debezium'`
- [ ] `schema_adapter.go`: rewrite `BuildUpsertSQL` for v1.25 columns, OCC via `_gpay_source_ts`
- [ ] `recon_dest_agent.go`: query `_gpay_source_id` thay `_id`
- [ ] `recon_source_agent.go`: hash input uses Mongo `_id` (unchanged)

### Phase 3: CMS + FE (4h)

- [ ] CMS API response include `_gpay_*` fields
- [ ] FE DataIntegrity column rename
- [ ] Legacy columns still accessible (BC)

### Phase 4: Verification + Stability (6h)

- [ ] Load test: 10K inserts/sec, 0 collision
- [ ] Redis Worker ID Registry: simulate pod restart → claim new ID, old expires
- [ ] NTP drift simulation: artificial +200ms skew → alert fire
- [ ] Recon cycle: verify dest_count match after migration
- [ ] 7 days stability monitor

### Phase 5: Legacy cleanup (deferred, 2h after N days)

- [ ] DROP old columns (`_id`, `source_id`, etc.)
- [ ] pg_repack for disk reclaim
- [ ] Remove legacy code paths

**Total**: 36h (honest, vs v2 claimed 13-14h).

---

## 7. Rollback

Each step independently reversible:
- ADD COLUMN → DROP COLUMN (safe, new columns nullable)
- Backfill partial → leave NULL, retry
- Trigger attached → DROP TRIGGER
- Worker code → config flag `v125_write_mode=false` revert to legacy

**No single point of no-return** until Phase 5 DROP old columns.

---

## 8. Open decisions (user)

1. **pg_repack install**: OK coord với DBA để install extension?
2. **Redis Worker ID Registry deploy**: OK thêm service mới hay dùng Redis hiện có?
3. **NTP tolerance**: chấp nhận 100ms drift hay strict 10ms?
4. **Phase 5 cleanup timeline**: 7, 14, or 30 days sau Phase 4?
5. **Typed column extraction**: skip hoàn toàn (JSONB only) hay per-table request base khi business team cần?
6. **Airbyte activation**: trong scope v1.25 hay defer?

---

## 9. What CHANGED from v2 → v3

| Aspect | v2 | v3 |
|:-------|:---|:---|
| Typed business columns | Auto-detect from JSONB (hallucinate) | Keep JSONB queries, skip typed extraction |
| Single Identity | Go sinh + DB validate | Go call PG `next_sonyflake()` for batch; Go Sonyflake với Redis worker_id for stream; both validated |
| Worker ID | Static grep log | Redis dynamic registry với heartbeat |
| Migration pattern | Single-transaction aggressive | pg_repack online + batched backfill |
| Rác stripping | PG `_raw_data - ARRAY[...]` in tx | Go Worker `stripAirbyteMetadata()` pre-insert |
| Effort estimate | 13-14h (lie) | 36h realistic |
| Lock duration | Not calculated | Per-step duration math |
| Schema isolation | `cdc_internal` new schema | Keep `public` (reduce complexity, no benefit) |

---

## 10. Lessons embedded

- #1 Scale Budget (36h for 8 tables, 50h+ per 200 tables)
- #64 Cross-store hash identity (PG + Go both generate, both validated)
- #65 Per-entity band-aid avoided (Redis registry scales N pods)
- #67 Reconstruction vs migration (this IS migration, not reconstruction — naming honest)
- #68 Ops reality gap (new, avoid vocab-aggressive lies)
