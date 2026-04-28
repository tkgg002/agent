# Sonyflake v1.25 — v4 FULL RECONSTRUCTION (zero compromise)

> **Date**: 2026-04-20
> **Author**: Brain (claude-opus-4-7)
> **Supersedes**: v1 band-aid, v2 vocab-lie, v3 scope-cut
> **Principle**: No compromise. Full cost accepted. Typed schema mandatory. PG sole authority. Data cũ transformed, không copy.
> **Rejected anti-patterns**: "out of scope", "hybrid", "auto-detect", "pragmatic compromise"

---

## 0. v3 failures acknowledged (4th attempt)

| # | v3 hèn nhát | Real reconstruction |
|:--|:------------|:--------------------|
| 1 | Skip typed columns = rename cột only | **Typed schema mandatory per table**. Manual mapping, accept 200+h effort for 200 tables |
| 2 | Hybrid identity (Go local + PG batch) | **PG sole authority**. Go fetches batch 100 IDs via RPC, caches in memory, never generates local |
| 3 | Redis Worker ID Registry (SPOF) | **PG-native registry** table với `FOR UPDATE SKIP LOCKED`. No Redis dependency |
| 4 | pg_repack không check disk/IO | **Disk precheck required + I/O throttle + off-peak schedule**. Mitigations explicit |
| 5 | Strip rác tại Worker only, data cũ bẩn | **TRANSFORM trong migration** — parse JSONB, extract typed, strip rác per-row batched |

---

## 1. Target schema — TYPED + clean

Per table, migration tạo **typed columns** extracted from JSONB + strip rác:

```sql
-- Example: payment_bills (Brain will produce exact per-table from Phase -1 business analysis)
CREATE TABLE cdc_internal.payment_bills (
  -- Identity (v1.25)
  _gpay_id            BIGINT PRIMARY KEY,
  _gpay_source_id     VARCHAR(200) NOT NULL,
  _gpay_source_engine VARCHAR(20) NOT NULL CHECK (_gpay_source_engine IN ('airbyte','debezium')),
  _gpay_source_ts     BIGINT,
  _gpay_sync_ts       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_version       BIGINT NOT NULL DEFAULT 1,
  _gpay_deleted       BOOLEAN NOT NULL DEFAULT FALSE,
  _gpay_hash          VARCHAR(64),
  
  -- Typed business columns (manual mapping Phase -1)
  bill_no             VARCHAR(100),
  merchant_id         VARCHAR(50),
  amount              NUMERIC(18,2),
  currency            CHAR(3),
  status              VARCHAR(30),
  user_id             VARCHAR(50),
  due_date            TIMESTAMPTZ,
  paid_at             TIMESTAMPTZ,
  
  -- Raw payload preserved for audit, BUT stripped + minified
  _gpay_raw_data      JSONB NOT NULL
);

-- Type mismatch handling: CHECK constraint for critical fields
ALTER TABLE cdc_internal.payment_bills
  ADD CONSTRAINT chk_amount_positive CHECK (amount IS NULL OR amount >= 0);
```

**Type mismatch strategy** (user critique Q1):
- Field `amount` lúc string lúc number → migration tạo cột `NUMERIC` + **DLQ logging** khi source value không parse được
- `failed_sync_logs` capture row + reason `TYPE_MISMATCH_amount: got string, expected numeric`
- Admin UI surface DLQ rows → manual fix hoặc schema override

**Effort per table** (honest):
- Business analysis (which fields, types, constraints): **2-4h**
- Migration script write: **1-2h**
- Backfill + verify: **1-2h**
- Go Worker schema adapter update: **0.5-1h**
- FE column display: **0.5h**
- **Per-table total: 5-10h**
- **8 current tables: 40-80h**
- **200 tables future: 1000-2000h** (months of work, multiple engineers)

---

## 2. PG Sole Identity Authority (Fix #2)

### 2.1 Both paths fetch from PG

**Airbyte bridge**:
```go
// bridge_batch.go
ids := make([]int64, 0, batchSize)
db.Raw("SELECT cdc_internal.next_sonyflake() FROM generate_series(1, ?)", batchSize).Scan(&ids)
// Use ids[i] cho row i
```

**Debezium streaming** (previously Go-local):
```go
// kafka_consumer.go — ID cache with refill
type IDCache struct {
    ids    []int64
    mu     sync.Mutex
    refillAt int  // refill when len < refillAt
    batchSize int // fetch batchSize when refill
    db     *gorm.DB
}

func (c *IDCache) Next(ctx context.Context) (int64, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if len(c.ids) < c.refillAt {
        newIDs := make([]int64, 0, c.batchSize)
        if err := c.db.WithContext(ctx).Raw(
            "SELECT cdc_internal.next_sonyflake() FROM generate_series(1, ?)", 
            c.batchSize,
        ).Scan(&newIDs).Error; err != nil {
            return 0, err
        }
        c.ids = append(c.ids, newIDs...)
    }
    id := c.ids[0]
    c.ids = c.ids[1:]
    return id, nil
}
```

**Config**: `batchSize=1000`, `refillAt=200`. 
- Per-message overhead: ~0 (fetched from cache)
- Refill trigger: every ~800 messages, 1 PG query = ~2ms
- Effective per-message cost: 2ms / 800 = **2.5μs** — negligible

**No NTP dependency**: PG is sole clock source. Go pods consuming IDs don't need synced clock.

### 2.2 Remove Go-local Sonyflake library entirely

Delete references to `idgen.NextID()` based on local generation. All ID fetches go through PG.

---

## 3. PG-Native Worker Registry (Fix #3)

**No Redis**. Use PG table + `FOR UPDATE SKIP LOCKED`.

```sql
-- Migration 018 thêm
CREATE TABLE cdc_internal.worker_registry (
  worker_id     INTEGER PRIMARY KEY,
  hostname      TEXT,
  claimed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  heartbeat_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  pid           INTEGER,
  status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','released','zombie'))
);

-- Seed: worker_id 0 = PG sentinel, 1-65534 available
INSERT INTO cdc_internal.worker_registry (worker_id, hostname, status)
  VALUES (0, 'pg-internal', 'active')
  ON CONFLICT DO NOTHING;

-- Function: claim a worker_id atomically
CREATE OR REPLACE FUNCTION cdc_internal.claim_worker_id(p_hostname TEXT, p_pid INT)
RETURNS INTEGER AS $$
DECLARE
  claimed INTEGER;
BEGIN
  -- First: try to reclaim expired (heartbeat > 90s ago)
  UPDATE cdc_internal.worker_registry
    SET hostname = p_hostname, pid = p_pid, claimed_at = NOW(), 
        heartbeat_at = NOW(), status = 'active'
    WHERE worker_id = (
      SELECT worker_id FROM cdc_internal.worker_registry
      WHERE status IN ('released', 'zombie')
         OR heartbeat_at < NOW() - INTERVAL '90 seconds'
      ORDER BY worker_id
      FOR UPDATE SKIP LOCKED
      LIMIT 1
    )
    RETURNING worker_id INTO claimed;
  
  IF claimed IS NOT NULL THEN RETURN claimed; END IF;
  
  -- Else: allocate new worker_id (sequential fill)
  INSERT INTO cdc_internal.worker_registry (worker_id, hostname, pid, status)
  VALUES (
    (SELECT COALESCE(MAX(worker_id), 0) + 1 FROM cdc_internal.worker_registry),
    p_hostname, p_pid, 'active'
  )
  RETURNING worker_id INTO claimed;
  
  RETURN claimed;
END;
$$ LANGUAGE plpgsql;

-- Heartbeat function
CREATE OR REPLACE FUNCTION cdc_internal.heartbeat_worker(p_worker_id INT) RETURNS void AS $$
BEGIN
  UPDATE cdc_internal.worker_registry
    SET heartbeat_at = NOW()
    WHERE worker_id = p_worker_id AND status = 'active';
END;
$$ LANGUAGE plpgsql;

-- Release on shutdown
CREATE OR REPLACE FUNCTION cdc_internal.release_worker(p_worker_id INT) RETURNS void AS $$
BEGIN
  UPDATE cdc_internal.worker_registry
    SET status = 'released', heartbeat_at = NOW()
    WHERE worker_id = p_worker_id;
END;
$$ LANGUAGE plpgsql;
```

**Go code**:
```go
// pkgs/workerid/registry.go
func Claim(ctx context.Context, db *gorm.DB) (int, error) {
    hostname, _ := os.Hostname()
    var workerID int
    err := db.WithContext(ctx).Raw(
        "SELECT cdc_internal.claim_worker_id(?, ?)", 
        hostname, os.Getpid(),
    ).Scan(&workerID).Error
    return workerID, err
}

// Heartbeat loop mỗi 30s
go func() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        db.Exec("SELECT cdc_internal.heartbeat_worker(?)", workerID)
    }
}()

// Shutdown hook
defer db.Exec("SELECT cdc_internal.release_worker(?)", workerID)
```

**Trade-off**: PG đã là dependency bắt buộc. Adding Redis = new SPOF, new ops complexity. PG metadata table = zero new infra.

---

## 4. Zero-Downtime Migration with Disk/IO Safety (Fix #4)

### 4.1 Pre-migration checklist

**Disk precheck**:
```sql
-- Query free space
SELECT 
  pg_size_pretty(pg_database_size('goopay_dw')) AS db_size,
  pg_size_pretty(available_bytes) AS available
FROM (SELECT 1) x
-- Cần query qua pg_ls_dir hoặc external monitoring
-- Rule: available_disk >= 2 × largest_table_size_total
```

**I/O baseline**:
- Measure pg_stat_io before migration
- Set `maintenance_work_mem = 1GB` tạm thời
- `checkpoint_timeout = 30min` để reduce WAL pressure

### 4.2 Migration pattern (TRANSFORM + zero-downtime)

Per table:

**Step 1**: Create new TYPED table in cdc_internal schema
```sql
CREATE TABLE cdc_internal.{table}_new (...v1.25 typed schema...);
```

**Step 2**: Dual-write trigger on OLD table forward to NEW
```sql
CREATE OR REPLACE FUNCTION public.dual_write_{table}() RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    INSERT INTO cdc_internal.{table}_new (
      _gpay_id, _gpay_source_id, _gpay_source_engine, _gpay_source_ts,
      _gpay_raw_data, _gpay_hash, _gpay_version, _gpay_deleted,
      -- Typed columns extracted from _raw_data JSONB
      bill_no, merchant_id, amount, currency, status, user_id, due_date, paid_at
    ) VALUES (
      COALESCE(NEW._id::bigint, cdc_internal.next_sonyflake()),
      COALESCE(NEW._id::text, NEW.source_id),
      NEW._source,
      NEW._source_ts,
      -- Strip rác in-trigger
      NEW._raw_data - '_airbyte_ab_id' - '_airbyte_extracted_at' - '_airbyte_loaded_at' - '_airbyte_data_hash',
      NEW._hash,
      COALESCE(NEW._version, 1),
      COALESCE(NEW._deleted, FALSE),
      -- Typed extraction
      (NEW._raw_data->>'bill_no'),
      (NEW._raw_data->>'merchant_id'),
      CASE 
        WHEN jsonb_typeof(NEW._raw_data->'amount') = 'number' 
          THEN (NEW._raw_data->>'amount')::numeric 
        WHEN jsonb_typeof(NEW._raw_data->'amount') = 'string' 
          THEN NULLIF(NEW._raw_data->>'amount','')::numeric
        ELSE NULL 
      END,
      (NEW._raw_data->>'currency'),
      (NEW._raw_data->>'status'),
      (NEW._raw_data->>'user_id'),
      NULLIF(NEW._raw_data->>'due_date','')::timestamptz,
      NULLIF(NEW._raw_data->>'paid_at','')::timestamptz
    )
    ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted 
    DO UPDATE SET ... -- full update semantics
    ;
  END IF;
  RETURN NEW;
EXCEPTION WHEN OTHERS THEN
  -- Type mismatch logged DLQ, but don't fail old write
  INSERT INTO failed_sync_logs (target_table, record_id, error_code, error_message, raw_data)
  VALUES ('{table}', COALESCE(NEW._id::text, NEW.source_id), 'TYPE_MISMATCH', SQLERRM, NEW._raw_data);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_{table}_dual_write
  AFTER INSERT OR UPDATE ON public.{table}
  FOR EACH ROW EXECUTE FUNCTION public.dual_write_{table}();
```

**Step 3**: Backfill historical data (batched, transform included, throttled)
```bash
# Application-layer script (Go / shell)
batch_size=1000
sleep_between_ms=100  # throttle IO

while true; do
  rows=$(psql -c "
    WITH batch AS (
      SELECT ctid FROM public.{table}
      WHERE NOT EXISTS (
        SELECT 1 FROM cdc_internal.{table}_new n 
        WHERE n._gpay_source_id = public.{table}._id::text
      )
      LIMIT $batch_size
      FOR UPDATE SKIP LOCKED
    )
    INSERT INTO cdc_internal.{table}_new (...)  -- same transform as trigger
    SELECT ... FROM public.{table} t JOIN batch b ON t.ctid = b.ctid
    ON CONFLICT DO NOTHING
    RETURNING 1
  " | wc -l)
  [ "$rows" -eq 0 ] && break
  sleep 0.${sleep_between_ms}
done
```

**Lock math**: 1000 rows/batch × 100ms sleep. For 10M rows: 10000 batches × 100ms = ~17min wall, but NO blocking of concurrent reads/writes. I/O spread out.

**Step 4**: Verify new table matches expected
```sql
-- Row count match
SELECT 
  (SELECT COUNT(*) FROM public.{table} WHERE NOT _deleted) AS old_active,
  (SELECT COUNT(*) FROM cdc_internal.{table}_new WHERE NOT _gpay_deleted) AS new_active,
  (SELECT COUNT(*) FROM failed_sync_logs WHERE target_table = '{table}' AND error_code = 'TYPE_MISMATCH') AS type_errors;
-- If new_active < old_active - type_errors: backfill incomplete, investigate
```

**Step 5**: Cutover (application-level, no lock)
- Worker + CMS config flag `table.{table}.mode = v125`
- Workers now write to `cdc_internal.{table}_new` directly
- Trigger on OLD table still dual-writes (safety net for rogue writers)

**Step 6**: After N days stable, swap
```sql
BEGIN;
ALTER TABLE public.{table} RENAME TO {table}_legacy_v125;
ALTER TABLE cdc_internal.{table}_new SET SCHEMA public;
ALTER TABLE public.{table}_new RENAME TO {table};
-- Drop dual-write trigger (no longer needed)
DROP TRIGGER trg_{table}_dual_write ON public.{table}_legacy_v125;
COMMIT;
```

**Step 7**: Legacy table cleanup
- Keep `{table}_legacy_v125` 7-30 days for rollback
- Eventually DROP + pg_repack any fragmented storage

### 4.3 Risk mitigations explicit

| Risk | Mitigation |
|:-----|:-----------|
| Disk 2x during dual-table | Precheck `available_bytes >= 2 × sum(table_sizes)`. Fail fast if insufficient |
| I/O spike backfill | Batch 1000 + sleep 100ms. Schedule off-peak. Monitor `pg_stat_io` |
| Dual-write trigger CPU | JSONB extraction per row = ~100μs. At 10K msg/sec = 1s CPU/sec = 1 core load. Add PG replica capacity nếu cần |
| Type mismatch data loss | DLQ capture + admin UI. User reviews before DROP legacy |
| Trigger infinite loop | Trigger on old table only writes to new, no reverse. Verify. |
| pg_repack disk | Run **only after** dual-write phase completes + legacy dropped. At that point = 1x table, not 2x |

---

## 5. Data transform included (Fix #5)

Migration **TRANSFORM**, not just COPY:
- Strip `_airbyte_*` keys from JSONB
- Extract typed columns (amount → NUMERIC, timestamps → TIMESTAMPTZ)
- Normalize values (trim whitespace, lowercase enums)
- Validate against CHECK constraints (amount >= 0)
- Log type mismatches to DLQ

All 10M+ existing rows pass through transform during backfill. Post-migration: `_gpay_raw_data` stripped clean, typed columns populated.

No "strip at Worker only" — Worker strips NEW writes, migration strips OLD writes. Full coverage.

---

## 6. Effort — TRUE COST

### 6.1 Current 8 tables

| Phase | Effort |
|:------|:-------|
| Phase -1 business analysis (typed schema per table) | 8 × 3h = **24h** |
| Phase 0 PG foundation (functions, worker_registry, pg_repack install) | **6h** |
| Phase 1 per-table migration (per-table trigger, backfill, verify) | 8 × 5h = **40h** |
| Phase 2 Worker code (ID cache, typed column mapping, schema adapter rewrite) | **16h** |
| Phase 3 CMS + FE (typed column display) | **8h** |
| Phase 4 verify + load test + stability monitor | **12h** |
| **Total (8 tables)** | **~106h** |

### 6.2 Future 200 tables scale

- Per table: ~5h (without learning curve). With experience curve + tooling: maybe 2-3h.
- **200 tables: 400-600h = 10-15 weeks full-time single engineer OR 3-4 weeks team of 4**

### 6.3 What v4 commits

- Typed schema MANDATORY per table — no JSONB-only shortcut
- PG sole identity — no hybrid, no Go-local
- PG worker registry — no Redis
- Data transform during migration — no scope cut
- Disk/IO risk explicit — no tool hand-waving

**This IS reconstruction cost. User decides: accept 106h (8 tables) + 600h (200 tables) investment, or defer.**

---

## 7. Open decisions

1. **Accept 106h for 8 tables**: proceed or scope down to 3-4 tables initial?
2. **Typed schema ownership**: Brain produce per-table mapping (user review) or business team provides?
3. **Dual-write trigger CPU budget**: OK with ~1 core load during backfill?
4. **Legacy retention**: 7/14/30 days after swap?
5. **DLQ type-mismatch handling policy**: auto-retry after manual fix, or require admin intervention?
6. **Start tables order**: smallest first (learning curve) or most critical first (max impact)?

---

## 8. What v4 DOES NOT compromise

- ❌ No JSONB-only path for business queries (typed columns mandatory)
- ❌ No Go-local ID generation (PG sole source)
- ❌ No Redis dependency (PG worker registry suffices)
- ❌ No "copy without transform" (migration transforms data)
- ❌ No "13-14h" or "36h" lies (106h honest for 8 tables)
- ❌ No scope-cut calling itself "pragmatic"

---

## 9. Self-critique

This is Brain's 4th attempt. Pattern:
- v1 passive → v2 vocab-aggressive → v3 ops-hèn → v4 full-cost-commit

If v4 still has flaws, they're likely:
- Typed extraction CASE logic may be naive for complex types (nested objects, arrays) — requires user review per-table
- Dual-write trigger CPU estimate 100μs/row may be optimistic — load test required
- Worker ID `claim_worker_id` function SQL may have edge cases (concurrent claim racing for same worker_id=max+1)

User please flag specific gaps. Brain will patch concrete, not shift layers.

---

## 10. Lessons applied

- #1 Scale Budget (honest 106h cho 8 tables, 600h cho 200)
- #64 Cross-store hash identity (now mooted — PG sole authority)
- #65 Per-entity band-aid (typed schema per-table, accept manual cost)
- #67 Reconstruction vs migration (this IS full reconstruction)
- #68 Ops reality gap (disk/IO math explicit, tool caveats)
- #69 Scope-cut = hèn nhát (v4 accepts full cost, no "out of scope")
