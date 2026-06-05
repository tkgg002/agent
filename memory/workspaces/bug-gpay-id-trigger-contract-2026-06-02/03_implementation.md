# 03_implementation.md — Patch Spec (for Muscle)

> Spec chi tiết: file path tuyệt đối + before/after + verify command.
> Brain KHÔNG thực thi. Muscle thực thi sau User approve.

---

## §1. Migration mới `019_sonyflake_default_fill.sql`

### File path
```
/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/ids/019_sonyflake_default_fill.sql
```

### Content (Brain spec)
```sql
-- 019_sonyflake_default_fill.sql
-- Fix Contract Drift `_gpay_id NULL` — workspace bug-gpay-id-trigger-contract-2026-06-02
-- Single source of truth: DB-side DEFAULT cdc_internal.sf_nextval()
-- Idempotent: chạy nhiều lần OK.

BEGIN;

-- ============================================================================
-- 1. Sonyflake nextval function (server-side ID gen)
-- ============================================================================
-- Reads `app.fencing_machine_id` session var (set by Go sink connection
-- bootstrap, xem migration 018 fencing system).
-- ID layout (Sonyflake): 39bit time | 8bit seq | 16bit machine_id

CREATE OR REPLACE FUNCTION cdc_internal.sf_nextval()
RETURNS BIGINT
LANGUAGE plpgsql
AS $$
DECLARE
  v_machine_id  BIGINT;
  v_time_ms     BIGINT;
  v_seq         BIGINT;
  v_epoch_ms    BIGINT := 1577836800000;  -- 2020-01-01 UTC (sonyflake default)
  v_id          BIGINT;
BEGIN
  -- 1. machine_id từ session var (fencing system đã set)
  BEGIN
    v_machine_id := current_setting('app.fencing_machine_id', true)::BIGINT;
  EXCEPTION WHEN OTHERS THEN
    v_machine_id := NULL;
  END;

  IF v_machine_id IS NULL THEN
    RAISE EXCEPTION 'sf_nextval: app.fencing_machine_id session var not set '
                    '(connection bootstrap missing? table=%, user=%)',
                    TG_TABLE_NAME, current_user
      USING ERRCODE = 'P0001';
  END IF;

  IF v_machine_id < 0 OR v_machine_id > 65535 THEN
    RAISE EXCEPTION 'sf_nextval: machine_id % out of range [0, 65535]', v_machine_id
      USING ERRCODE = 'P0001';
  END IF;

  -- 2. time component (ms since sonyflake epoch, /10 = 10ms granularity)
  v_time_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT - v_epoch_ms;
  v_time_ms := v_time_ms / 10;  -- sonyflake 10ms unit

  IF v_time_ms < 0 OR v_time_ms >= (1::BIGINT << 39) THEN
    RAISE EXCEPTION 'sf_nextval: time overflow (epoch drift?)' USING ERRCODE = 'P0001';
  END IF;

  -- 3. sequence per-machine-per-time-unit (dùng advisory lock + per-second seq)
  --    Đơn giản: dùng random 8-bit (collision rate ~ 1/256 trong cùng 10ms cùng machine).
  --    Acceptable vì sink Go gọi qua DEFAULT thưa, không phải hot loop.
  v_seq := (random() * 255)::BIGINT;

  -- 4. compose: [time:39 | seq:8 | machine:16]
  v_id := (v_time_ms << 24) | (v_seq << 16) | v_machine_id;

  RETURN v_id;
END;
$$;

COMMENT ON FUNCTION cdc_internal.sf_nextval() IS
  'Server-side Sonyflake ID generator for V2 shadow tables. Reads machine_id '
  'from app.fencing_machine_id session var. See workspace '
  'bug-gpay-id-trigger-contract-2026-06-02.';

-- ============================================================================
-- 2. Heal existing V2 shadow tables — ALTER SET DEFAULT (metadata-only)
-- ============================================================================
DO $$
DECLARE
  r RECORD;
  v_sql TEXT;
BEGIN
  FOR r IN
    SELECT n.nspname AS schema_name, c.relname AS table_name
    FROM pg_attribute a
    JOIN pg_class c ON c.oid = a.attrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE a.attname = '_gpay_id'
      AND a.attnum > 0
      AND NOT a.attisdropped
      AND a.atthasdef = FALSE            -- chưa có DEFAULT
      AND c.relkind = 'r'                -- regular table
      AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'cdc_internal', 'cdc_system')
  LOOP
    v_sql := format(
      'ALTER TABLE %I.%I ALTER COLUMN %I SET DEFAULT cdc_internal.sf_nextval()',
      r.schema_name, r.table_name, '_gpay_id'
    );
    RAISE NOTICE 'Healing table %.%: %', r.schema_name, r.table_name, v_sql;
    EXECUTE v_sql;
  END LOOP;
END;
$$;

COMMIT;
```

### Verify
```bash
# Apply
make migrate
# Re-apply (idempotent check)
psql $DB_URL -f migrations/schema/ids/019_sonyflake_default_fill.sql
# Inspect
psql $DB_URL -c "\df cdc_internal.sf_nextval"
psql $DB_URL -c "\d+ data_hub.tokens" | grep _gpay_id
```

---

## §2. Go DDL fix `schema_manager.go`

### File path
```
/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/sinkworker/schema_manager.go
```

### Edit (line ~226)

**BEFORE:**
```go
cols := []string{
    `"_gpay_id" BIGINT PRIMARY KEY`,
    `"_source_id" TEXT NOT NULL`,
    // ... các cột khác
}
```

**AFTER:**
```go
cols := []string{
    `"_gpay_id" BIGINT PRIMARY KEY DEFAULT cdc_internal.sf_nextval()`,
    `"_source_id" TEXT NOT NULL`,
    // ... các cột khác
}
```

### Verify
```bash
cd data-hub/centralized-data-service
go build ./...
go vet ./internal/sinkworker/...
# Unit test mới (xem §4)
go test ./internal/sinkworker/ -run TestCreateShadowTable_HasDefault -v
```

---

## §3. Comment fix `batch_buffer.go`

### File path
```
/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/batch_buffer.go
```

### Edit (line 246-250)

**BEFORE:**
```go
// V2 shadow contract: bootstrap (ShadowAutomator) emits `_gpay_id BIGINT
// PK` (sonyflake trigger fills) + `_source_id TEXT NOT NULL` partial
// UNIQUE WHERE NOT _deleted (ON CONFLICT anchor).
effectivePK := first.PrimaryKeyField
```

**AFTER:**
```go
// V2 shadow contract: `_gpay_id BIGINT PK DEFAULT cdc_internal.sf_nextval()`
// (xem migrations/schema/ids/019_sonyflake_default_fill.sql) + `_source_id
// TEXT NOT NULL` partial UNIQUE WHERE NOT _deleted (ON CONFLICT anchor).
effectivePK := first.PrimaryKeyField
```

### Verify
```bash
grep -n "cdc_internal.sf_nextval" internal/handler/batch_buffer.go
# expect: 1 match (line 246-249)
grep -c "sonyflake trigger fills" internal/handler/batch_buffer.go
# expect: 0 match
```

---

## §4. Integration test `batch_buffer_v2shadow_test.go`

### File path
```
/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/batch_buffer_v2shadow_test.go
```

### Content (Brain spec, simplified)
```go
package handler_test

import (
    "context"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/sony/sonyflake"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestBatchUpsert_V2Shadow_NoExplicitGpayID verifies AC-1 + AC-2:
// INSERT vào V2 shadow table không chỉ định _gpay_id phải thành công sau fix.
func TestBatchUpsert_V2Shadow_NoExplicitGpayID(t *testing.T) {
    ctx := context.Background()

    // 1. Spawn Postgres container
    pg, err := postgres.RunContainer(ctx, /* image, init scripts apply migrations */)
    require.NoError(t, err)
    defer pg.Terminate(ctx)

    dsn, _ := pg.ConnectionString(ctx, "sslmode=disable")
    pool, err := pgxpool.New(ctx, dsn)
    require.NoError(t, err)
    defer pool.Close()

    // 2. Set fencing session vars (giả lập sink bootstrap)
    _, err = pool.Exec(ctx, `SET LOCAL app.fencing_machine_id = 42`)
    require.NoError(t, err)
    _, err = pool.Exec(ctx, `SET LOCAL app.fencing_token = 1`)
    require.NoError(t, err)

    // 3. Create V2 shadow table giống production
    _, err = pool.Exec(ctx, `
        CREATE TABLE test_v2.tokens (
            _gpay_id   BIGINT PRIMARY KEY DEFAULT cdc_internal.sf_nextval(),
            _source_id TEXT NOT NULL,
            _raw_data  JSONB,
            _deleted   BOOLEAN DEFAULT FALSE
        );
        CREATE UNIQUE INDEX ON test_v2.tokens (_source_id) WHERE NOT _deleted;
    `)
    require.NoError(t, err)

    // 4. INSERT không chỉ định _gpay_id
    _, err = pool.Exec(ctx, `
        INSERT INTO test_v2.tokens (_source_id, _raw_data)
        VALUES ('src-1', '{}'::jsonb)
    `)
    require.NoError(t, err, "INSERT phải success sau fix")

    // 5. Verify _gpay_id non-null + decode sonyflake
    var gpayID uint64
    err = pool.QueryRow(ctx, `SELECT _gpay_id FROM test_v2.tokens WHERE _source_id='src-1'`).Scan(&gpayID)
    require.NoError(t, err)
    require.NotZero(t, gpayID)

    decoded := sonyflake.Decompose(gpayID)
    require.Equal(t, uint64(42), decoded["machine-id"], "machine_id phải match session var")
}

// TestBatchUpsert_V2Shadow_5000Rows_Perf (AC-8): benchmark batch 5000 rows.
func TestBatchUpsert_V2Shadow_5000Rows_Perf(t *testing.T) {
    // ... setup giống trên ...
    // ... INSERT batch 5000 rows không chỉ định _gpay_id ...
    // ... assert duration ≤ baseline + 5% ...
}

// TestSonyflakeIDDecode (AC-7)
func TestSonyflakeIDDecode(t *testing.T) {
    // ... fetch generated _gpay_id, decode, assert thành phần ...
}
```

### Verify
```bash
cd data-hub/centralized-data-service
go test -v -race -count=3 ./internal/handler/ -run TestBatchUpsert_V2Shadow_NoExplicitGpayID
go test -v -race ./internal/handler/ -run TestSonyflakeIDDecode
go test -bench BenchmarkBatchUpsert_5000 ./internal/handler/
```

---

## §5. Deploy script

### Pre-deploy
```bash
# 1. Backup
pg_dump $PROD_DB_URL --schema-only > backup_$(date +%Y%m%d_%H%M).sql

# 2. Verify migration trên staging
psql $STAGING_DB_URL -f migrations/schema/ids/019_sonyflake_default_fill.sql
psql $STAGING_DB_URL -c "\d+ data_hub.tokens" | grep DEFAULT

# 3. Apply lên prod
psql $PROD_DB_URL -f migrations/schema/ids/019_sonyflake_default_fill.sql
psql $PROD_DB_URL -c "\d+ data_hub.tokens" | grep DEFAULT  # confirm
```

### Deploy Go binary
```bash
# Build với P2 + P3 patches
cd data-hub/centralized-data-service
go build -o bin/sink ./cmd/sink
docker build -t centralized-data-service:bugfix-gpayid .
kubectl set image deploy/centralized-data-service sink=centralized-data-service:bugfix-gpayid
kubectl rollout status deploy/centralized-data-service
```

### Post-deploy verify
```bash
kubectl logs -f deploy/centralized-data-service --tail=200 | grep -E "(batch upsert|_gpay_id)"
# Expect: 0 lỗi "null value in column \"_gpay_id\""
```

---

## §6. Rollback procedure (nếu fail)

```sql
-- 1. Remove DEFAULT (table-by-table)
DO $$
DECLARE r RECORD;
BEGIN
  FOR r IN
    SELECT n.nspname, c.relname
    FROM pg_attribute a
    JOIN pg_class c ON c.oid = a.attrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE a.attname = '_gpay_id' AND a.atthasdef = TRUE
      AND n.nspname NOT IN ('pg_catalog','information_schema','cdc_internal','cdc_system')
  LOOP
    EXECUTE format('ALTER TABLE %I.%I ALTER COLUMN _gpay_id DROP DEFAULT',
                   r.nspname, r.relname);
  END LOOP;
END $$;

-- 2. Drop function
DROP FUNCTION IF EXISTS cdc_internal.sf_nextval();
```

```bash
# 3. Redeploy Go binary cũ
kubectl rollout undo deploy/centralized-data-service
```

---

## §7. Summary patch files

| # | File | Action | LOC delta |
|---|---|---|---|
| 1 | `migrations/schema/ids/019_sonyflake_default_fill.sql` | CREATE | +~90 |
| 2 | `internal/sinkworker/schema_manager.go` | EDIT line 226 | +0 / -0 (replace) |
| 3 | `internal/handler/batch_buffer.go` | EDIT comment | +0 / -0 (replace) |
| 4 | `internal/handler/batch_buffer_v2shadow_test.go` | CREATE | +~100 |

**Total source delta:** ~190 LOC (mostly tests + new migration).
**Code logic delta:** 1 dòng Go (DDL string).
