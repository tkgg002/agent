# Action Plan — Sonyflake v1.25 Reconstruction Execution

> **Date**: 2026-04-20
> **Author**: Brain (claude-opus-4-7)
> **Reference**: `02_plan_sonyflake_v125_reconstruction.md`
> **Mode**: Task-level execution — Muscle thực thi từng Phase. Brain set DEFAULT decisions, user flag để adjust.
> **Principle**: Aggressive reconstruction, zero band-aid, single identity authority, forced cutover.

---

## DEFAULTS (user override nếu cần)

| # | Decision | DEFAULT | Rationale |
|:--|:---------|:--------|:----------|
| 1 | PG worker_id | **1** | Go observed 2622 ∈ [2560-2815]; PG=1 safely out of range |
| 2 | Business columns extraction | **Auto-detect per table** (Phase -1 script) | Scalable 200+ tables, không manual mapping |
| 3 | Legacy retention | **24h** | Aggressive reconstruction mode; all-or-nothing |
| 4 | Registry mislabel fix | **Phase 1** (source of truth first) | Correct label BEFORE migration, avoid downstream confusion |
| 5 | Airbyte activation | **Defer Phase 4** | Stabilize Debezium-only first, Airbyte bridge empty hiện |
| 6 | Rollout | **Sequential 1-by-1** với verify giữa | Lower risk, catch issue trước khi impact nhiều tables |

User **stop** Brain/Muscle nếu default nào không ok.

---

## Phase -1: Environment Verification

### Task -1.1: Go Worker machineID inventory

```bash
# Inspect all recent Worker startup logs
for log in /tmp/worker*.log; do
  grep "sonyflake initialized" $log | head -5
done

# Cluster subnet (if Kubernetes, otherwise local)
# If local dev: machineID derived from laptop IP
# Observed from session: machineID=2622, IP=192.168.10.62
```

**Expected output**:
```
machineID:2622, ip:192.168.10.62  → octet2=10, octet3=62 → 10*256+62=2622
```

**Range allocation**:
- Go: machineID 2622 (single dev instance). Production K8s: depends on cluster subnet. 
- **Reserve for PG**: worker_id=1 (NOT in Go range for this deployment).

**Verify contract**:
```sql
-- Add validation function (will reject any ID with machineID outside known ranges)
-- See Phase 0 validate_sonyflake() function
```

### Task -1.2: Business columns per table inventory

Auto-scan script:
```bash
for table in identitycounters payment_bill_codes payment_bill_events payment_bill_histories payment_bill_holdings payment_bills refund_requests export_jobs; do
  echo "=== $table ==="
  docker exec gpay-postgres psql -U user -d goopay_dw -c "
    SELECT jsonb_object_keys(_raw_data) AS field, COUNT(*) AS coverage
    FROM $table 
    WHERE _raw_data IS NOT NULL 
    GROUP BY 1 
    ORDER BY 2 DESC LIMIT 30
  " 2>&1 | head -35
done > /tmp/table_columns_inventory.txt
```

**Output**: Per-table business field list + coverage % → Brain produce extraction SQL per table.

### Task -1.3: Registry current state snapshot

```sql
SELECT target_table, source_db, source_table, sync_engine, primary_key_field, timestamp_field
FROM cdc_table_registry WHERE is_active=true
ORDER BY target_table;
```

Save output to `10_gap_analysis_sonyflake_environment.md` for reference.

### Task -1.4: Existing row counts

```sql
SELECT 'identitycounters' AS t, COUNT(*) FROM identitycounters
UNION ALL SELECT 'payment_bill_codes', COUNT(*) FROM payment_bill_codes
UNION ALL SELECT 'payment_bill_events', COUNT(*) FROM payment_bill_events
UNION ALL SELECT 'payment_bill_histories', COUNT(*) FROM payment_bill_histories
UNION ALL SELECT 'payment_bill_holdings', COUNT(*) FROM payment_bill_holdings
UNION ALL SELECT 'payment_bills', COUNT(*) FROM payment_bills
UNION ALL SELECT 'refund_requests', COUNT(*) FROM refund_requests
UNION ALL SELECT 'export_jobs', COUNT(*) FROM export_jobs;
```

### Task -1.5: Registry mislabel fix (moved to Phase 1 per decision #4)

**Defer** — apply trong Phase 1 migration per-table.

**Phase -1 output**: `10_gap_analysis_sonyflake_environment.md` với:
- Go machineID observed + reserved range
- PG worker_id chosen
- Business columns per 8 tables
- Row counts pre-migration
- Registry state snapshot

---

## Phase 0: Foundation Migration

### Task 0.1: Migration 018 — schema + functions

File: `centralized-data-service/migrations/018_sonyflake_v125_foundation.sql`

```sql
-- Migration 018: Sonyflake v1.25 Foundation
-- Creates cdc_internal schema + identity provider functions + guard trigger template
-- Idempotent: safe re-run

BEGIN;

-- 1. Schema
CREATE SCHEMA IF NOT EXISTS cdc_internal;
COMMENT ON SCHEMA cdc_internal IS 'CDC v1.25 canonical tables, Sonyflake-native identity';

-- 2. Sequence for PG-side Sonyflake
CREATE SEQUENCE IF NOT EXISTS cdc_internal.sonyflake_seq 
  MINVALUE 0 MAXVALUE 255 CYCLE START 0;

-- 3. next_sonyflake() — PG-native ID generator, worker_id=1
CREATE OR REPLACE FUNCTION cdc_internal.next_sonyflake() RETURNS bigint AS $func$
DECLARE
  our_epoch  bigint := 1477267200000;   -- 2016-10-21 UTC (Sony default, match Go)
  elapsed    bigint;
  seq_id     bigint;
  worker_id  int    := 1;               -- PG allocated; Go uses 2560-2815 range
  result     bigint;
BEGIN
  elapsed := ((extract(epoch FROM clock_timestamp()) * 1000)::bigint - our_epoch) / 10;
  
  IF elapsed < 0 THEN
    RAISE EXCEPTION 'next_sonyflake: clock before epoch 2016-10-21 — NTP broken';
  END IF;
  
  seq_id := nextval('cdc_internal.sonyflake_seq') % 256;
  result := (elapsed << 24) | ((seq_id & 255) << 16) | (worker_id & 65535);
  RETURN result;
END;
$func$ LANGUAGE plpgsql VOLATILE;

COMMENT ON FUNCTION cdc_internal.next_sonyflake IS 
  'Generate Sonyflake 64-bit ID. worker_id=1 reserved for PG. Go Worker uses IP-derived machineID ∈ [2560-2815] for this deployment.';

-- 4. validate_sonyflake() — strict validation for Go-provided IDs
CREATE OR REPLACE FUNCTION cdc_internal.validate_sonyflake(sf bigint) RETURNS void AS $func$
DECLARE
  -- Configurable: Go allocated range (Phase -1 verified)
  go_min_machine int := 2560;
  go_max_machine int := 2815;
  pg_machine     int := 1;
  extracted_machine int;
  extracted_time    bigint;
  max_future_time   bigint;
BEGIN
  IF sf IS NULL OR sf <= 0 THEN
    RAISE EXCEPTION 'validate_sonyflake: NULL or non-positive';
  END IF;
  
  extracted_machine := (sf & 65535);
  extracted_time    := sf >> 24;
  
  IF extracted_machine != pg_machine 
     AND (extracted_machine < go_min_machine OR extracted_machine > go_max_machine) THEN
    RAISE EXCEPTION 'validate_sonyflake: machineID % not in [PG=%] or Go[%..%]',
      extracted_machine, pg_machine, go_min_machine, go_max_machine;
  END IF;
  
  max_future_time := ((extract(epoch FROM clock_timestamp()) * 1000 - 1477267200000) / 10) + 8640000;
  IF extracted_time < 0 OR extracted_time > max_future_time THEN
    RAISE EXCEPTION 'validate_sonyflake: timestamp out of range';
  END IF;
END;
$func$ LANGUAGE plpgsql IMMUTABLE;

-- 5. Guard trigger function (attached per-table in Phase 1)
CREATE OR REPLACE FUNCTION cdc_internal.tg_gpay_guard()
RETURNS TRIGGER AS $func$
BEGIN
  -- Identity Authority (strict)
  IF NEW._gpay_source_engine IS NULL THEN
    RAISE EXCEPTION 'tg_gpay_guard: _gpay_source_engine required';
  END IF;
  
  CASE NEW._gpay_source_engine
    WHEN 'airbyte' THEN
      IF TG_OP = 'INSERT' THEN
        IF NEW._gpay_id IS NOT NULL THEN
          RAISE EXCEPTION 'tg_gpay_guard: Airbyte INSERT must not provide _gpay_id (DB-generated)';
        END IF;
        NEW._gpay_id := cdc_internal.next_sonyflake();
      END IF;
    WHEN 'debezium' THEN
      IF TG_OP = 'INSERT' THEN
        IF NEW._gpay_id IS NULL THEN
          RAISE EXCEPTION 'tg_gpay_guard: Debezium INSERT must provide _gpay_id (Go-generated)';
        END IF;
        PERFORM cdc_internal.validate_sonyflake(NEW._gpay_id);
      END IF;
    ELSE
      RAISE EXCEPTION 'tg_gpay_guard: _gpay_source_engine must be airbyte|debezium, got %', NEW._gpay_source_engine;
  END CASE;
  
  -- Source ID anchor
  IF NEW._gpay_source_id IS NULL OR NEW._gpay_source_id = '' THEN
    NEW._gpay_source_id := COALESCE(
      (NEW._gpay_raw_data->>'_id'),
      (NEW._gpay_raw_data->'_id'->>'$oid'),
      (NEW._gpay_raw_data->>'id')
    );
    IF NEW._gpay_source_id IS NULL OR NEW._gpay_source_id = '' THEN
      RAISE EXCEPTION 'tg_gpay_guard: cannot derive _gpay_source_id from _gpay_raw_data';
    END IF;
  END IF;
  
  -- Timing
  NEW._gpay_updated_at := CURRENT_TIMESTAMP;
  IF TG_OP = 'INSERT' THEN
    NEW._gpay_sync_ts := COALESCE(NEW._gpay_sync_ts, CURRENT_TIMESTAMP);
    NEW._gpay_created_at := COALESCE(NEW._gpay_created_at, CURRENT_TIMESTAMP);
    NEW._gpay_version := COALESCE(NEW._gpay_version, 1);
  ELSIF TG_OP = 'UPDATE' THEN
    NEW._gpay_version := COALESCE(OLD._gpay_version, 0) + 1;
    NEW._gpay_created_at := OLD._gpay_created_at;
    NEW._gpay_id := OLD._gpay_id;
    NEW._gpay_source_id := OLD._gpay_source_id;
  END IF;
  
  -- Integrity hash
  IF NEW._gpay_hash IS NULL THEN
    NEW._gpay_hash := encode(digest(NEW._gpay_raw_data::text, 'sha256'), 'hex');
  END IF;
  
  RETURN NEW;
END;
$func$ LANGUAGE plpgsql;

-- 6. Ensure pgcrypto for digest() (hash function)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

COMMIT;

-- Self-test (run separately, verify no errors)
-- SELECT cdc_internal.next_sonyflake();
-- SELECT cdc_internal.validate_sonyflake(cdc_internal.next_sonyflake());  -- should succeed
-- SELECT cdc_internal.validate_sonyflake(1);  -- should fail (machineID=1 is OK but timestamp=0)
```

### Task 0.2: Post-migration verify

```bash
# Apply migration
docker exec -i gpay-postgres psql -U user -d goopay_dw < migrations/018_sonyflake_v125_foundation.sql

# Test function
docker exec gpay-postgres psql -U user -d goopay_dw -c "
SELECT cdc_internal.next_sonyflake() AS id1, cdc_internal.next_sonyflake() AS id2;
SELECT cdc_internal.next_sonyflake() > cdc_internal.next_sonyflake();  -- later should be greater (but race here, may fail)
"

# Test validation: positive case
docker exec gpay-postgres psql -U user -d goopay_dw -c "
SELECT cdc_internal.validate_sonyflake(cdc_internal.next_sonyflake());  -- void return, no error
"

# Test validation: negative case (wrong machineID)
docker exec gpay-postgres psql -U user -d goopay_dw -c "
-- Build fake ID with machineID=9999 (out of range)
SELECT cdc_internal.validate_sonyflake((1000000::bigint << 24) | 9999);
-- Expect: EXCEPTION
"
```

---

## Phase 1: Per-Table Reconstruction (Sequential)

### Task 1.X: Migration script template

File: `centralized-data-service/migrations/019_{table}_v125_reconstruct.sql` (per-table)

Template (parameterized by table name + business columns):

```sql
-- Migration 019_{table}_v125_reconstruct — Reconstruct {table} to v1.25 unified schema
-- ALL-OR-NOTHING transaction

BEGIN;

-- 1. Create v1.25 table
CREATE TABLE cdc_internal.{table} (
  _gpay_id            BIGINT PRIMARY KEY,
  _gpay_source_id     VARCHAR(200) NOT NULL,
  _gpay_source_engine VARCHAR(20) NOT NULL,
  _gpay_source_ts     BIGINT,
  _gpay_sync_ts       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  _gpay_raw_data      JSONB NOT NULL DEFAULT '{}'::jsonb,
  _gpay_hash          VARCHAR(64),
  _gpay_version       BIGINT NOT NULL DEFAULT 1,
  _gpay_deleted       BOOLEAN NOT NULL DEFAULT FALSE,
  
  -- Business columns (Phase -1 auto-detected — example for payment_bills)
  -- These are extracted from _raw_data at migration time
  -- {business_col_1}   {type},
  -- {business_col_2}   {type},
  -- ...
  
  CONSTRAINT chk_{table}_source_engine 
    CHECK (_gpay_source_engine IN ('airbyte','debezium'))
);

-- 2. Copy data with strip + extract
INSERT INTO cdc_internal.{table} (
  _gpay_id,
  _gpay_source_id,
  _gpay_source_engine,
  _gpay_source_ts,
  _gpay_sync_ts,
  _gpay_created_at,
  _gpay_updated_at,
  _gpay_raw_data,
  _gpay_hash,
  _gpay_version,
  _gpay_deleted
  -- , {business_col_1}, {business_col_2}  -- extracted from _raw_data
)
SELECT
  -- ID: reuse Go-generated BIGINT if exists (from bridge_batch path), else sinh mới
  COALESCE(
    CASE 
      WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='{table}' AND column_name='id' AND data_type='bigint')
        THEN NULLIF(id::text, '')::bigint
      ELSE NULL
    END,
    cdc_internal.next_sonyflake()
  ),
  -- Source ID anchor
  COALESCE(_id::text, NULLIF(source_id, ''), id::text),
  -- Registry lookup for actual engine (even if mislabeled)
  (SELECT CASE 
    WHEN r.sync_engine = 'debezium' THEN 'debezium'
    WHEN r.sync_engine = 'airbyte' AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='_airbyte_raw_{table}') THEN 'airbyte'
    ELSE 'debezium'  -- DEFAULT: registry mislabel fix Phase 1 (decision #4)
   END FROM cdc_table_registry r WHERE r.target_table='{table}' LIMIT 1),
  _source_ts,
  _synced_at,
  _created_at,
  _updated_at,
  -- Strip Airbyte/CDC internal metadata from raw
  _raw_data 
    - '_airbyte_ab_id' 
    - '_airbyte_extracted_at' 
    - '_airbyte_loaded_at' 
    - '_airbyte_data_hash'
    - '_ab_cdc_lsn'
    - '_ab_cdc_updated_at'
    - '_ab_cdc_deleted_at',
  _hash,
  _version,
  _deleted
FROM public.{table};

-- 3. Indexes
CREATE UNIQUE INDEX idx_{table}_gpay_src_active
  ON cdc_internal.{table} (_gpay_source_id)
  WHERE _gpay_deleted IS FALSE;

CREATE INDEX idx_{table}_gpay_src_ts
  ON cdc_internal.{table} (_gpay_source_ts)
  WHERE _gpay_source_ts IS NOT NULL;

CREATE INDEX idx_{table}_gpay_updated_at
  ON cdc_internal.{table} (_gpay_updated_at DESC);

-- 4. Attach guard trigger
CREATE TRIGGER trg_{table}_gpay_guard
  BEFORE INSERT OR UPDATE ON cdc_internal.{table}
  FOR EACH ROW EXECUTE FUNCTION cdc_internal.tg_gpay_guard();

-- 5. Row count verify (abort tx nếu mismatch)
DO $body$
DECLARE
  old_cnt bigint;
  new_cnt bigint;
BEGIN
  SELECT COUNT(*) INTO old_cnt FROM public.{table};
  SELECT COUNT(*) INTO new_cnt FROM cdc_internal.{table};
  IF old_cnt != new_cnt THEN
    RAISE EXCEPTION 'Migration count mismatch {table}: old=%, new=%', old_cnt, new_cnt;
  END IF;
END;
$body$;

-- 6. Rename: legacy keep 24h
ALTER TABLE public.{table} RENAME TO {table}_legacy_v125;

-- 7. Registry update: fix mislabel (decision #4 — Phase 1)
UPDATE cdc_table_registry 
SET sync_engine = (SELECT _gpay_source_engine FROM cdc_internal.{table} LIMIT 1)
WHERE target_table = '{table}';

COMMIT;

-- Post-migration action: 
-- (Run separately trong Phase 2 code update)
-- Worker + CMS code switch reads to cdc_internal.{table}
```

### Task 1.Y: Apply per table (sequential)

```bash
# Sequential migration 1-by-1 với verify giữa
for table in export_jobs refund_requests identitycounters payment_bill_codes payment_bill_events payment_bill_histories payment_bill_holdings payment_bills; do
  echo "=== Migrating $table ==="
  
  # Generate per-table migration from template (Brain pre-compute với business cols)
  # sed replace {table} và insert business columns
  
  # Apply
  docker exec -i gpay-postgres psql -U user -d goopay_dw < migrations/019_${table}_v125_reconstruct.sql
  
  # Verify
  docker exec gpay-postgres psql -U user -d goopay_dw -c "
    SELECT 
      (SELECT COUNT(*) FROM public.${table}_legacy_v125) AS legacy_cnt,
      (SELECT COUNT(*) FROM cdc_internal.${table}) AS new_cnt,
      (SELECT COUNT(*) FROM cdc_internal.${table} WHERE _gpay_id IS NOT NULL) AS with_id,
      (SELECT COUNT(*) FROM cdc_internal.${table} WHERE _gpay_source_id IS NOT NULL) AS with_srcid,
      (SELECT COUNT(DISTINCT _gpay_source_engine) FROM cdc_internal.${table}) AS distinct_engines
  "
  
  # Pause for manual inspection
  read -p "Continue next table? (y/n): " ok
  [ "$ok" != "y" ] && break
done
```

---

## Phase 2: Worker + CMS + FE Code Updates

### Task 2.1: Worker — kafka_consumer.go (Debezium path)

File: `centralized-data-service/internal/handler/kafka_consumer.go`

**Changes**:
```go
// Change 1: Target table schema → cdc_internal
targetTable := fmt.Sprintf("cdc_internal.%s", entry.TargetTable)  // was: entry.TargetTable

// Change 2: Build upsert với _gpay_* columns
type gpayRecord struct {
    GpayID           int64           // Go Sonyflake generated
    GpaySourceID     string          // Mongo _id
    GpaySourceEngine string          // "debezium"
    GpaySourceTs     *int64          // Debezium source.ts_ms
    GpayRawData      json.RawMessage // After strip _airbyte_*
    GpayDeleted      bool
    // Business columns mapped from Debezium payload
}

sfID, _ := idgen.NextID()  // Go Sonyflake (machineID from IP)
record := gpayRecord{
    GpayID:           int64(sfID),
    GpaySourceID:     extractMongoID(payload),
    GpaySourceEngine: "debezium",
    GpaySourceTs:     &sourceTsMs,
    GpayRawData:      stripAirbyte(rawJSON),
    GpayDeleted:      op == "d",
}
```

### Task 2.2: Worker — bridge_batch.go (Airbyte path)

**Changes**:
```go
// Change 1: Target cdc_internal
// Change 2: DO NOT set _gpay_id — DB trigger sinh
// Change 3: Engine = 'airbyte'

// BEFORE:
sfID, _ := idgen.NextID()
batch.Queue(upsertSQL, sfID, sourceID, rawData, ...)

// AFTER:
// Airbyte path: _gpay_id=NULL, trigger fills
batch.Queue(upsertSQL_v125,
    nil,                         // _gpay_id (NULL → trigger sinh)
    sourceID,                    // _gpay_source_id
    "airbyte",                   // _gpay_source_engine
    nil,                         // _gpay_source_ts (Airbyte không có)
    stripAirbyte(rawData),       // _gpay_raw_data
    false,                       // _gpay_deleted
    // business cols...
)
```

### Task 2.3: Worker — schema_adapter.go BuildUpsertSQL v1.25

**Rewrite** để generate SQL với `_gpay_*` column names + OCC guard on `_gpay_source_ts`:

```go
func (sa *SchemaAdapter) BuildUpsertSQL_v125(table string, sourceEngine string, hasSourceTs bool) string {
    baseCols := `_gpay_id, _gpay_source_id, _gpay_source_engine, _gpay_source_ts, 
                 _gpay_raw_data, _gpay_deleted`
    
    insertSQL := fmt.Sprintf(`
        INSERT INTO cdc_internal.%s (%s, {business_cols})
        VALUES ($1, $2, $3, $4, $5, $6, {business_placeholders})
        ON CONFLICT (_gpay_source_id) WHERE _gpay_deleted IS FALSE
        DO UPDATE SET
          _gpay_raw_data = EXCLUDED._gpay_raw_data,
          _gpay_source_ts = EXCLUDED._gpay_source_ts,
          _gpay_deleted = EXCLUDED._gpay_deleted,
          {business_updates}
    `, table, baseCols)
    
    // OCC guard — only for Debezium path with _source_ts
    if sourceEngine == "debezium" && hasSourceTs {
        insertSQL += `
          WHERE cdc_internal.` + table + `._gpay_source_ts IS NULL 
             OR cdc_internal.` + table + `._gpay_source_ts < EXCLUDED._gpay_source_ts
        `
    }
    
    return insertSQL
}
```

### Task 2.4: Recon agents

File: `centralized-data-service/internal/service/recon_{source,dest}_agent.go`

**Changes**:
- Source agent (Mongo): unchanged — queries Mongo, không touch PG schema
- Dest agent: query `cdc_internal.{table}._gpay_source_id` thay `_id`
- Hash field: `hashIDPlusTsMs(_gpay_source_id, _gpay_source_ts)` (unchanged function, different column source)

### Task 2.5: CMS — reconciliation_handler.go

```go
// Change: JOIN cdc_table_registry + cdc_reconciliation_report (already done)
// Add: include _gpay_* columns in response

// Response field rename:
{
  "target_table": "payment_bills",
  "_gpay_id_example": 15245781234567890,    // NEW - show sample ID
  "source_id_key": "_gpay_source_id",        // NEW - explicit anchor column name
  "sync_engine": "debezium",
  ...
}
```

### Task 2.6: CMS — registry_handler.go (Register new table flow)

```go
// On POST /api/registry — after insert registry row, publish NATS:
h.natsClient.Conn.Publish("cdc.cmd.migrate-to-v125", payload)
// Worker handler: run Phase 1 per-table migration script
```

### Task 2.7: FE — DataIntegrity.tsx

```tsx
// Rename columns reference
{ title: 'Sonyflake ID (sample)', dataIndex: '_gpay_id_example' },
{ title: 'Source Anchor', dataIndex: '_gpay_source_id' },
// Hide legacy _id / source_id references
```

---

## Phase 3: Legacy Cleanup (After 24h stability)

### Task 3.1: DROP legacy tables

```bash
for table in export_jobs refund_requests identitycounters payment_bill_codes payment_bill_events payment_bill_histories payment_bill_holdings payment_bills; do
  docker exec gpay-postgres psql -U user -d goopay_dw -c "
    DROP TABLE IF EXISTS public.${table}_legacy_v125 CASCADE;
  "
done
```

### Task 3.2: Remove v125_mode config flag + legacy code paths

File: `centralized-data-service/internal/handler/kafka_consumer.go` + `bridge_batch.go`
- Remove `if cfg.V125Mode { ... } else { // legacy }` branches
- Remove references to `_id`, `source_id`, `_synced_at`, etc.

### Task 3.3: Deprecate old migration functions

```sql
DROP FUNCTION IF EXISTS public.create_cdc_table(text, text);
DROP FUNCTION IF EXISTS public.standardize_cdc_table(text);
```

---

## Verification Checklist (per Phase)

### Phase -1 Success
- [ ] Go machineID range documented
- [ ] PG worker_id chosen, not in Go range
- [ ] Business columns per 8 tables listed với coverage %
- [ ] Registry snapshot saved

### Phase 0 Success
- [ ] `cdc_internal` schema exists
- [ ] `next_sonyflake()` returns monotonic IDs (test 100 calls)
- [ ] `validate_sonyflake()` rejects wrong machineID
- [ ] `tg_gpay_guard()` compiles + test trigger attach on throwaway table

### Phase 1 Success (per table)
- [ ] `cdc_internal.{table}` exists với v1.25 schema
- [ ] Row count match legacy (DO block assertion passed)
- [ ] `_gpay_id` 100% populated
- [ ] `_gpay_source_id` 100% populated
- [ ] `_gpay_source_engine` ∈ {airbyte, debezium}
- [ ] Partial unique index active (no dup active source_id)
- [ ] Trigger attached, test INSERT/UPDATE passes
- [ ] `public.{table}_legacy_v125` exists (rollback ready)

### Phase 2 Success
- [ ] Worker startup log clean, writes to `cdc_internal.{table}`
- [ ] Kafka consumer + bridge_batch use new schema
- [ ] Recon still works (Tier 1/2 counts match expected)
- [ ] CMS API returns `_gpay_*` fields
- [ ] FE DataIntegrity renders correctly

### Phase 3 Success
- [ ] No `_id` / `source_id` / `_synced_at` references in code (grep audit)
- [ ] Legacy tables dropped
- [ ] Old functions dropped
- [ ] 7 days runtime stable (0 trigger failures, 0 recon anomalies)

---

## Rollback Procedures

### Phase 0 Rollback
```sql
DROP SCHEMA cdc_internal CASCADE;
```

### Phase 1 Rollback (per table, within 24h)
```sql
BEGIN;
DROP TABLE cdc_internal.{table} CASCADE;
ALTER TABLE public.{table}_legacy_v125 RENAME TO {table};
-- Revert registry label if changed
COMMIT;
```

### Phase 2 Rollback
```yaml
# Worker config
v125_mode: false  # fall back to legacy path

# Worker code: keep legacy branches until Phase 3
```

### Phase 3: No rollback (point of no return)

---

## Task-Muscle Mapping

| Phase | Task | Est Effort | Owner |
|:------|:-----|:-----------|:------|
| -1.1 | Go machineID inventory | 15m | Muscle |
| -1.2 | Business columns scan | 30m | Muscle |
| -1.3 | Registry snapshot | 5m | Muscle |
| -1.4 | Row counts | 5m | Muscle |
| 0.1 | Migration 018 apply | 30m | Muscle |
| 0.2 | Foundation verify | 15m | Muscle |
| 1.X | Per-table migration (×8) | 30m × 8 = 4h | Muscle |
| 2.1-2.3 | Worker code update | 3h | Muscle |
| 2.4 | Recon agents update | 1h | Muscle |
| 2.5-2.6 | CMS update | 2h | Muscle |
| 2.7 | FE update | 1h | Muscle |
| Verify | All phases | 1h | Brain + Muscle |
| **Total** | | **~13-14h** | |

---

## Pre-flight before Muscle start

Brain confirm:
- [x] Lesson #67 applied (reconstruction mode)
- [x] 6 defaults chosen với rationale
- [x] Per-phase rollback defined
- [x] Verification checklist embedded
- [x] Rule 7 workspace doc prefix `09_tasks_solution_*`
- [x] Rule 12 Brain-no-code — plan only
- [ ] User confirm 6 defaults (or flag override)
- [ ] User approve Phase -1 start

---

## Files persisted

- `02_plan_sonyflake_v125_reconstruction.md` — ADR design
- `09_tasks_solution_sonyflake_v125.md` — this file (task-level execution)
- `05_progress.md` APPEND trigger-ready

## Lessons referenced

- #1 Scale Budget
- #60 ADR passive
- #64 Cross-store hash identity
- #65 Per-entity band-aid
- #66 Hallucination state (verified registry counts now)
- #67 Reconstruction vs migration mode
