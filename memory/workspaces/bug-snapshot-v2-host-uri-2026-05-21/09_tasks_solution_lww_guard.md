# 09_tasks_solution_lww_guard — Hồ sơ giải pháp kỹ thuật (Code Demo)

> **Phase**: `lww_guard`
> **Đối tượng**: Muscle (CC CLI) execute sau khi User approve.
> **Tree CORRECT**: `/Users/trainguyen/Documents/work/data-hub/` (KHÔNG đụng `cdc-system/`).

---

## Solution overview

```
┌────────────────────────────────────────────────────────────────────┐
│   Race: Snapshot v2 ↔ Realtime ghi cùng row trong shadow V1        │
├────────────────────────────────────────────────────────────────────┤
│  Fix Strategy (Phương án D):                                       │
│   1. Backport `_source_ts BIGINT` xuống V1 cdcCols + DDL inline    │
│   2. Migration ADD COLUMN cho bảng shadow đang tồn tại + backfill  │
│   3. Snapshot envelope dùng Mongo clusterTime thay time.Now()      │
│   4. Snapshot force `_source='snapshot:v2'` ở record map           │
│   5. OCC guard mở rộng tiebreaker khi ts bằng nhau                 │
└────────────────────────────────────────────────────────────────────┘
```

---

## Edit #1 — `cdcCols` map: thêm `_source_ts`

**File**: `data-hub/centralized-data-service/internal/service/schema_adapter.go`
**Hiện trạng (line 195-204)**:

```go
cdcCols := map[string]string{
    "_raw_data":   "JSONB",
    "_source":     "VARCHAR(20) DEFAULT 'airbyte'",
    "_synced_at":  "TIMESTAMP DEFAULT NOW()",
    "_version":    "BIGINT DEFAULT 1",
    "_hash":       "VARCHAR(64)",
    "_deleted":    "BOOLEAN DEFAULT FALSE",
    "_created_at": "TIMESTAMP DEFAULT NOW()",
    "_updated_at": "TIMESTAMP DEFAULT NOW()",
}
```

**Sau khi sửa**:

```go
cdcCols := map[string]string{
    "_raw_data":   "JSONB",
    "_source":     "VARCHAR(20) DEFAULT 'airbyte'",
    "_source_ts":  "BIGINT",                          // ← THÊM: oplog ts (ms epoch). NULL ok cho row legacy.
    "_synced_at":  "TIMESTAMP DEFAULT NOW()",
    "_version":    "BIGINT DEFAULT 1",
    "_hash":       "VARCHAR(64)",
    "_deleted":    "BOOLEAN DEFAULT FALSE",
    "_created_at": "TIMESTAMP DEFAULT NOW()",
    "_updated_at": "TIMESTAMP DEFAULT NOW()",
}
```

**Lý do KHÔNG default NOW()**: `_source_ts` là oplog ts của source store, KHÔNG phải clock của PG. Default sẽ gây sai semantic. NULL = "row legacy không có ts" → guard `IS NULL OR <=` handle đúng.

---

## Edit #2 — `createShadowTableV1WithCols` DDL inline: thêm `_source_ts`

**File**: `data-hub/centralized-data-service/internal/service/schema_adapter.go`
**Hiện trạng (line 314-325)** — DDL tạo bảng mới:

```go
ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    "_gpay_id" BIGINT,
    %s TEXT,%s
    "_raw_data" JSONB,
    "_source" VARCHAR(20) DEFAULT 'airbyte',
    "_synced_at" TIMESTAMP DEFAULT NOW(),
    ...
)`, qualified, pkIdent, bizDDL.String())
```

**Sau khi sửa** — thêm dòng `_source_ts` SAU `_source`:

```go
ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    "_gpay_id" BIGINT,
    %s TEXT,%s
    "_raw_data" JSONB,
    "_source" VARCHAR(20) DEFAULT 'airbyte',
    "_source_ts" BIGINT,                         -- ← THÊM
    "_synced_at" TIMESTAMP DEFAULT NOW(),
    ...
)`, qualified, pkIdent, bizDDL.String())
```

(Muscle: đọc nguyên đoạn DDL hiện tại ở line 314-325 và thêm dòng `_source_ts` BIGINT ở vị trí tương ứng.)

---

## Edit #3 — `BuildUpsertSQLInSchema` mở rộng OCC guard với tiebreaker

**File**: `data-hub/centralized-data-service/internal/service/schema_adapter.go`
**Hiện trạng (line 513-518)**:

```go
var whereClause string
if hasSourceTs && sourceTsMs > 0 {
    whereClause = fmt.Sprintf(
        `WHERE %s."_source_ts" IS NULL OR %s."_source_ts" <= EXCLUDED."_source_ts"`,
        qualifiedTable, qualifiedTable,
    )
} else {
    // fallback hash-based dedup
    ...
}
```

**Sau khi sửa** — đổi `<=` thành `<` và thêm tiebreaker với `_source` discriminator:

```go
var whereClause string
if hasSourceTs && sourceTsMs > 0 {
    // OCC guard với tiebreaker:
    //   1. _source_ts NULL → allow (row legacy chưa có ts).
    //   2. _source_ts < EXCLUDED → realtime/snapshot mới hơn → write.
    //   3. _source_ts == EXCLUDED → tie: realtime thắng snapshot.
    //      (`snapshot:v2` thua mọi nguồn khác; nếu cả 2 cùng snapshot:v2
    //       hoặc cùng realtime → giữ row hiện tại, KHÔNG ghi đè.)
    hasSourceCol := false
    if _, ok := schema.Columns["_source"]; ok {
        hasSourceCol = true
    }
    if hasSourceCol {
        whereClause = fmt.Sprintf(
            `WHERE %s."_source_ts" IS NULL `+
                `OR %s."_source_ts" < EXCLUDED."_source_ts" `+
                `OR (%s."_source_ts" = EXCLUDED."_source_ts" `+
                `    AND %s."_source" = 'snapshot:v2' `+
                `    AND EXCLUDED."_source" <> 'snapshot:v2')`,
            qualifiedTable, qualifiedTable, qualifiedTable, qualifiedTable,
        )
    } else {
        // _source column thiếu → giữ behavior cũ (chỉ guard theo ts).
        whereClause = fmt.Sprintf(
            `WHERE %s."_source_ts" IS NULL OR %s."_source_ts" < EXCLUDED."_source_ts"`,
            qualifiedTable, qualifiedTable,
        )
    }
} else {
    // fallback hash-based dedup (giữ nguyên block hiện tại)
    ...
}
```

**Note quan trọng**: đổi `<=` thành `<` → ts bằng nhau KHÔNG ghi đè mặc định. Chỉ allow ghi đè khi tiebreaker thoả (snapshot → realtime). Đây là behavior change cần document ở report.

---

## Edit #4 — Snapshot envelope dùng Mongo clusterTime

**File**: `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go`
**Hiện trạng (line 498-511)**:

```go
func buildSnapshotEnvelope(afterJSON []byte, now time.Time) []byte {
    var sb strings.Builder
    sb.WriteString(`{"specversion":"1.0","source":"snapshot:v2","type":"cdc.snapshot","time":"`)
    sb.WriteString(now.Format(time.RFC3339))
    sb.WriteString(`","data":{"op":"c","source_ts_ms":`)
    sb.WriteString(fmt.Sprintf("%d", now.UnixMilli()))     // ← wall clock
    sb.WriteString(`,"after":`)
    sb.Write(afterJSON)
    sb.WriteString(`}}`)
    return []byte(sb.String())
}
```

**Sau khi sửa** — accept `clusterTimeMs` parameter:

```go
// buildSnapshotEnvelope shapes a Debezium-compatible CDCEvent.
// clusterTimeMs: Mongo logical clock at snapshot start (ms epoch).
// Passed in by caller from db.hello().$clusterTime — KHÔNG dùng time.Now()
// để tránh snapshot clobber realtime do clock skew.
func buildSnapshotEnvelope(afterJSON []byte, now time.Time, clusterTimeMs int64) []byte {
    var sb strings.Builder
    sb.WriteString(`{"specversion":"1.0","source":"snapshot:v2","type":"cdc.snapshot","time":"`)
    sb.WriteString(now.Format(time.RFC3339))
    sb.WriteString(`","data":{"op":"c","source_ts_ms":`)
    sb.WriteString(fmt.Sprintf("%d", clusterTimeMs))       // ← Mongo clusterTime
    sb.WriteString(`,"after":`)
    sb.Write(afterJSON)
    sb.WriteString(`}}`)
    return []byte(sb.String())
}
```

**Caller update** (Muscle: grep `buildSnapshotEnvelope` để tìm caller, update truyền `clusterTimeMs`):

```bash
grep -n "buildSnapshotEnvelope" data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go
```

---

## Edit #5 — Snapshot runner đọc `clusterTime` từ `db.hello()` ở snapshot start

**File**: `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go`
**Vị trí**: ngay sau khi mở Mongo connection, TRƯỚC khi vào loop chunk read.

**Code mẫu** (Muscle: đặt trong function tương ứng — nhiều khả năng là `runSnapshot` hoặc `executeSnapshot`):

```go
// captureClusterTime đọc Mongo logical clock từ db.hello().
// Trả ms epoch + i ordinal. Fallback chain:
//   1. db.hello() → $clusterTime.clusterTime (BSON Timestamp).
//   2. replSetGetStatus → optimes.appliedOpTime.ts.
//   3. time.Now() + log WARN.
func captureClusterTime(ctx context.Context, db *mongo.Database, log *zap.Logger) int64 {
    var helloResult bson.M
    err := db.RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&helloResult)
    if err == nil {
        if ct, ok := helloResult["$clusterTime"].(bson.M); ok {
            if ts, ok := ct["clusterTime"].(primitive.Timestamp); ok {
                return int64(ts.T) * 1000  // T = Unix seconds
            }
        }
    }

    // Fallback 1: replSetGetStatus
    var rsStatus bson.M
    err = db.RunCommand(ctx, bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&rsStatus)
    if err == nil {
        if optimes, ok := rsStatus["optimes"].(bson.M); ok {
            if applied, ok := optimes["appliedOpTime"].(bson.M); ok {
                if ts, ok := applied["ts"].(primitive.Timestamp); ok {
                    return int64(ts.T) * 1000
                }
            }
        }
    }

    // Fallback 2: wall clock + WARN
    log.Warn("snapshot.v2 clusterTime capture fallback to wall clock",
        zap.Error(err))
    return time.Now().UnixMilli()
}
```

**Tích hợp** (Muscle: pseudo-code, adapt theo struct hiện tại):

```go
// snapshot_runner_handler.go - trong runSnapshot/executeSnapshot
clusterTimeMs := captureClusterTime(ctx, mongoDb, r.logger)
r.logger.Info("snapshot.v2 cluster time captured",
    zap.Int64("cluster_time_ms", clusterTimeMs),
    zap.String("connection_code", connCode))

// Lưu vào snapshot_progress để recovery (xem Migration #2)
r.progressRepo.UpdateClusterTime(ctx, jobID, clusterTimeMs)

// Sau đó loop chunk:
for chunk := range chunks {
    for _, doc := range chunk {
        envelope := buildSnapshotEnvelope(docJSON, time.Now(), clusterTimeMs)
        // ... HandleRaw(ctx, subject, envelope)
    }
}
```

---

## Edit #6 — Force `_source = 'snapshot:v2'` ở downstream record map

**File**: `data-hub/centralized-data-service/internal/handler/event_handler.go` HOẶC `batch_buffer.go` HOẶC `kafka_consumer.go`.

**Strategy**: thêm field `source` trong `Record` struct (hoặc tương đương) populated từ envelope `source`. Snapshot envelope đã có `"source":"snapshot:v2"`; realtime có `source.connector` hoặc default `'debezium-v125'`.

**Audit step BẮT BUỘC trước khi edit** (Muscle):

```bash
# Tìm chỗ extract _source value từ envelope
grep -n "\"_source\"\|_source.*EXCLUDED\|record.Source\|envelope.*source" \
    data-hub/centralized-data-service/internal/handler/*.go \
    data-hub/centralized-data-service/internal/service/batch_buffer.go
```

**Hypothesis về fix**:
- Trong event_handler/HandleRaw, parse envelope `source` field → lưu vào `record.Source`.
- Trong BatchBuffer.batchUpsert call `BuildUpsertSQLInSchema(..., source=record.Source, ...)` — nếu envelope `source="snapshot:v2"` → `_source` column = `'snapshot:v2'`.

**KHÔNG hardcode `'snapshot:v2'`** — phải parse từ envelope để pipeline 1 hướng (snapshot/realtime cùng chung code path, khác ở envelope).

---

## Migration #1 — `060_v1_add_source_ts_to_shadow.sql`

**Path**: `data-hub/cdc-cms-service/migrations/schema/core/060_v1_add_source_ts_to_shadow.sql`

```sql
-- Migration 060: Backport _source_ts column xuống V1 shadow tables.
-- Required cho LWW guard phase `lww_guard` (workspace bug-snapshot-v2-host-uri-2026-05-21).
-- Reference: ADR-004 trong 04_decisions_lww_guard.md.
--
-- Pre-conditions:
--   - PG ≥ 11 (ADD COLUMN không default = metadata-only instant).
--   - Application binary đã include cdcCols updated (Edit #1).
--
-- Post-conditions:
--   - Mọi bảng cdc_internal.* + shadow_*.* có column `_source_ts BIGINT NULL`.
--   - Row legacy có `_source_ts IS NULL` → OCC guard fall vào `IS NULL OR <`.

DO $$
DECLARE
    schema_rec RECORD;
    table_rec RECORD;
    sql_text TEXT;
BEGIN
    -- Loop qua các schema chứa shadow tables.
    FOR schema_rec IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name = 'cdc_internal'
           OR schema_name LIKE 'shadow_%'
    LOOP
        FOR table_rec IN
            SELECT table_name
            FROM information_schema.tables
            WHERE table_schema = schema_rec.schema_name
              AND table_type = 'BASE TABLE'
        LOOP
            sql_text := format(
                'ALTER TABLE %I.%I ADD COLUMN IF NOT EXISTS "_source_ts" BIGINT NULL',
                schema_rec.schema_name, table_rec.table_name
            );
            RAISE NOTICE 'Executing: %', sql_text;
            EXECUTE sql_text;
        END LOOP;
    END LOOP;
END $$;

-- Optional backfill (best-effort, không required cho correctness):
-- Comment ra block dưới nếu data lớn (>1M row/table), chạy tay batch sau.
-- DO $$
-- DECLARE
--     schema_rec RECORD;
--     table_rec RECORD;
-- BEGIN
--     FOR schema_rec IN SELECT schema_name FROM information_schema.schemata
--         WHERE schema_name = 'cdc_internal' OR schema_name LIKE 'shadow_%'
--     LOOP
--         FOR table_rec IN SELECT table_name FROM information_schema.tables
--             WHERE table_schema = schema_rec.schema_name AND table_type = 'BASE TABLE'
--         LOOP
--             EXECUTE format(
--                 'UPDATE %I.%I SET "_source_ts" = (EXTRACT(EPOCH FROM "_synced_at") * 1000)::BIGINT
--                  WHERE "_source_ts" IS NULL AND "_synced_at" IS NOT NULL',
--                 schema_rec.schema_name, table_rec.table_name
--             );
--         END LOOP;
--     END LOOP;
-- END $$;

-- Verify gate: count column tồn tại
SELECT
    n.nspname AS schema_name,
    c.relname AS table_name,
    COUNT(a.attname) FILTER (WHERE a.attname = '_source_ts') AS has_source_ts
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0
WHERE c.relkind = 'r'
  AND (n.nspname = 'cdc_internal' OR n.nspname LIKE 'shadow_%')
GROUP BY n.nspname, c.relname
ORDER BY n.nspname, c.relname;
-- Expect: has_source_ts = 1 cho mọi row.
```

---

## Migration #2 — `061_v1_snapshot_progress_cluster_time.sql`

**Path**: `data-hub/cdc-cms-service/migrations/schema/core/061_v1_snapshot_progress_cluster_time.sql`

```sql
-- Migration 061: Lưu Mongo clusterTime ở snapshot_progress để audit + recovery.
-- Reference: ADR-002 trong 04_decisions_lww_guard.md.

ALTER TABLE cdc_system.snapshot_progress
    ADD COLUMN IF NOT EXISTS "mongo_cluster_time_start_ms" BIGINT NULL,
    ADD COLUMN IF NOT EXISTS "mongo_cluster_time_capture_method" VARCHAR(32) NULL;
    -- method: 'hello' | 'replSetGetStatus' | 'walltime-fallback'

COMMENT ON COLUMN cdc_system.snapshot_progress."mongo_cluster_time_start_ms" IS
    'Mongo logical clock (ms epoch) tại snapshot start, lấy từ db.hello().$clusterTime. NULL = legacy run trước phase lww_guard.';
COMMENT ON COLUMN cdc_system.snapshot_progress."mongo_cluster_time_capture_method" IS
    'Method dùng để capture clusterTime. NULL = legacy run.';

-- Verify
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_schema = 'cdc_system'
  AND table_name = 'snapshot_progress'
  AND column_name IN ('mongo_cluster_time_start_ms', 'mongo_cluster_time_capture_method');
-- Expect: 2 row.
```

---

## Unit test — append `schema_adapter_test.go`

**File**: `data-hub/centralized-data-service/internal/service/schema_adapter_test.go`
(Muscle: nếu file chưa có, tạo mới; nếu có, append test func.)

```go
func TestBuildUpsertSQLInSchema_OCCGuardWithTiebreaker(t *testing.T) {
    sa := &SchemaAdapter{}
    schema := &TableSchema{
        Columns: map[string]ColumnInfo{
            "id":         {DataType: "TEXT"},
            "_source":    {DataType: "VARCHAR"},
            "_source_ts": {DataType: "BIGINT"},
            "_raw_data":  {DataType: "JSONB"},
        },
    }
    tests := []struct {
        name        string
        sourceTsMs  int64
        wantWHERE   string  // substring match
    }{
        {
            name:       "ts > 0 + _source col exists → tiebreaker active",
            sourceTsMs: 1234567890000,
            wantWHERE:  `_source" = 'snapshot:v2' AND EXCLUDED."_source" <> 'snapshot:v2'`,
        },
        {
            name:       "ts = 0 → fallback hash dedup",
            sourceTsMs: 0,
            wantWHERE:  `_hash" IS DISTINCT FROM`,
        },
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            sql, _ := sa.BuildUpsertSQLInSchema(
                schema, "cdc_internal", "test_t", "id", "abc",
                map[string]interface{}{"foo": "bar"},
                `{"foo":"bar"}`, "debezium-v125", "hash123", tc.sourceTsMs,
            )
            if !strings.Contains(sql, tc.wantWHERE) {
                t.Errorf("WHERE clause missing %q in SQL:\n%s", tc.wantWHERE, sql)
            }
        })
    }
}

func TestBuildUpsertSQLInSchema_NoSourceColumn_FallsBackToTsOnly(t *testing.T) {
    sa := &SchemaAdapter{}
    schema := &TableSchema{
        Columns: map[string]ColumnInfo{
            "id":         {DataType: "TEXT"},
            "_source_ts": {DataType: "BIGINT"},
            // KHÔNG có _source
        },
    }
    sql, _ := sa.BuildUpsertSQLInSchema(schema, "s", "t", "id", "x",
        map[string]interface{}{}, "{}", "debezium", "h", 100)
    if strings.Contains(sql, "snapshot:v2") {
        t.Errorf("không có _source col mà SQL vẫn ref 'snapshot:v2':\n%s", sql)
    }
    if !strings.Contains(sql, `_source_ts" < EXCLUDED."_source_ts"`) {
        t.Errorf("expected ts-only guard:\n%s", sql)
    }
}
```

---

## Verify gates checklist (Muscle BẮT BUỘC chạy trước báo Done)

```bash
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service

# Gate 1: Build
go build ./... 2>&1 | tee /tmp/lww_guard_build.log
test ${PIPESTATUS[0]} -eq 0 || { echo "BUILD FAIL"; exit 1; }

# Gate 2: Vet
go vet ./... 2>&1 | tee /tmp/lww_guard_vet.log
test ${PIPESTATUS[0]} -eq 0 || { echo "VET FAIL"; exit 1; }

# Gate 3: Unit test
go test -run 'TestBuildUpsertSQLInSchema' ./internal/service/... -v 2>&1 | tee /tmp/lww_guard_test.log
test ${PIPESTATUS[0]} -eq 0 || { echo "TEST FAIL"; exit 1; }

# Gate 4: Full service suite no regression
go test ./internal/service/... ./internal/handler/... 2>&1 | tee /tmp/lww_guard_full_test.log
# Tolerate pre-existing failures listed in 05_progress.md Followup #5

# Gate 5: Migration apply
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service
# Apply qua migration runner đã có (KHÔNG psql tay)
make migrate-up 2>&1 | tee /tmp/lww_guard_migrate.log
test ${PIPESTATUS[0]} -eq 0 || { echo "MIGRATE FAIL"; exit 1; }

# Gate 6: Verify column tồn tại
psql -h 127.0.0.1 -p 5433 -U postgres -d cdc_dw -c "
SELECT n.nspname, c.relname, COUNT(a.attname) FILTER (WHERE a.attname='_source_ts') AS has_ts
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
LEFT JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum > 0
WHERE c.relkind='r' AND (n.nspname='cdc_internal' OR n.nspname LIKE 'shadow_%')
GROUP BY n.nspname, c.relname
ORDER BY 1,2;"
# Expect: has_ts = 1 cho mọi row.

# Gate 7: Race smoke (coordination với user)
# - User trigger snapshot.v2 cho source_object_id=18 qua FE.
# - Trong lúc snapshot chạy, manual insert/update 1 record Mongo source.
# - SQL verify shadow:
#   SELECT _source, _source_ts, _id, _synced_at
#   FROM cdc_internal.<shadow_table>
#   WHERE _id = '<test_record_id>';
# - Expect: realtime data (_source='debezium-v125', _source_ts=oplog_ts).

# Gate 8: Security review
/security-agent /Users/trainguyen/Documents/work/data-hub/centralized-data-service
```

---

## Rollback (chỉ dùng nếu gate fail không sửa được trong 1 phiên)

```bash
# Code rollback
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service
git revert <commit_lww_guard_hash>

# Migration: forward-only — KHÔNG drop column. Để column NULL, behavior fall về cũ.
# Nếu thực sự cần xoá:
# (chỉ chạy với approval user, KHÔNG copy vào report)
```

---

## File changes summary (cho `report_lww_guard_2026-05-21.md` reference)

| File | Lines changed | Type |
|---|---|---|
| `data-hub/centralized-data-service/internal/service/schema_adapter.go` | ~5 lines (cdcCols map + DDL inline + WHERE clause) | Edit |
| `data-hub/centralized-data-service/internal/handler/snapshot_runner_handler.go` | ~10 lines (buildSnapshotEnvelope signature + captureClusterTime helper + caller update) | Edit |
| `data-hub/centralized-data-service/internal/handler/event_handler.go` OR `batch_buffer.go` | ~5 lines (parse envelope source → record.Source) | Edit (audit trước) |
| `data-hub/cdc-cms-service/migrations/schema/core/060_v1_add_source_ts_to_shadow.sql` | ~50 lines | New |
| `data-hub/cdc-cms-service/migrations/schema/core/061_v1_snapshot_progress_cluster_time.sql` | ~15 lines | New |
| `data-hub/centralized-data-service/internal/service/schema_adapter_test.go` | ~50 lines | Append |
