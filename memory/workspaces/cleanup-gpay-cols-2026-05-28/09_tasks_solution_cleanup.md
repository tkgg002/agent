# 09_tasks_solution_cleanup — RENAME `_gpay_source_id` → `source_id` + `_gpay_deleted` → `_deleted`

> **REVISED 2026-05-28 sau user re-direct**: Cleanup intent = RENAME (unify naming), KHÔNG remove logic. Option A/B/C cũ bị retire.

> Code demo cho UNIFIED RENAME plan. PHASE BRAIN: KHÔNG apply. Chờ user verb.

---

## Scope tổng
- **16 file code** (3 service) + **1 migration SQL**.
- Mechanical rename: `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted`.
- **Out of scope**: `_gpay_id` (PK column V2 master) — user không nêu.
- DoD destination: `grep -rn "_gpay_source_id\|_gpay_deleted" {centralized-data-service,cdc-cms-service,cdc-cms-web}` = **0 hits** (trừ migration SQL).

## Strategy
- **Code**: 16 file đổi text mechanical (rename identifier + SQL string + GORM tag + index name).
- **DB**: ALTER TABLE RENAME COLUMN + DROP/CREATE partial UNIQUE INDEX (giữ ngữ nghĩa tombstone-aware).
- **Conflict handling Path B**: shadow tables tạo bởi `command_handler.go` đang có **CẢ `_gpay_deleted` + `_deleted`** (rác Bug #2 yesterday). Migration DROP `_gpay_deleted` (giữ `_deleted` đã có), RENAME `_gpay_source_id` → `source_id`.
- **Path A no-op DB**: shadow_automator tables đã có `source_id`/`_deleted` → migration skip.

---

## CODE CHANGES (16 file)

### Path A — cdc-cms-service (FE shadow)

#### A.1 — `cdc-cms-service/internal/api/mapping_preview_handler.go:63-69`
**Before**:
```go
var rows []struct {
    GpayID   int64  `gorm:"column:_gpay_id"`
    SourceID string `gorm:"column:_gpay_source_id"`
    RawData  []byte `gorm:"column:_raw_data"`
}
q := `SELECT _gpay_id, _gpay_source_id, _raw_data FROM "` + shadowSchema + `"."` + req.ShadowTable + `" ORDER BY _synced_at DESC LIMIT ?`
```
**After**:
```go
var rows []struct {
    ID       int64  `gorm:"column:id"`
    SourceID string `gorm:"column:source_id"`
    RawData  []byte `gorm:"column:_raw_data"`
}
q := `SELECT id, source_id, _raw_data FROM "` + shadowSchema + `"."` + req.ShadowTable + `" ORDER BY _synced_at DESC LIMIT ?`
```
**Note**: `_gpay_id` → `id` vì Path A shadow_automator dùng `id BIGINT PK`. Đồng nhất.

#### A.2 — `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go:100`
**Before**: test setup expect column `_gpay_source_id`.
**After**: đổi expect column `source_id`.

#### A.3 — `cdc-cms-service/internal/infra/persistence/shadow_automator.go`
**Status**: NO CHANGE (đã dùng `source_id`/`_deleted`).

---

### Path B — centralized-data-service handler (FE shadow)

#### B.1 — `internal/handler/command_handler.go:163-180` cdcColumns ALTER list
**Before**:
```go
cdcColumns := []struct{ name, def string }{
    {"_gpay_source_id", "TEXT"},
    {"_raw_data", "JSONB NOT NULL DEFAULT '{}'::jsonb"},
    {"_source", "TEXT NOT NULL DEFAULT 'debezium'"},
    {"_source_ts", "BIGINT"},
    {"_synced_at", "TIMESTAMP NOT NULL DEFAULT NOW()"},
    {"_version", "BIGINT NOT NULL DEFAULT 1"},
    {"_hash", "TEXT"},
    {"_gpay_deleted", "BOOLEAN DEFAULT FALSE"},
    {"_deleted", "BOOLEAN DEFAULT FALSE"},
    {"_created_at", "TIMESTAMP DEFAULT NOW()"},
    {"_updated_at", "TIMESTAMP DEFAULT NOW()"},
}
```
**After**:
```go
cdcColumns := []struct{ name, def string }{
    {"source_id", "TEXT"},
    {"_raw_data", "JSONB NOT NULL DEFAULT '{}'::jsonb"},
    {"_source", "TEXT NOT NULL DEFAULT 'debezium'"},
    {"_source_ts", "BIGINT"},
    {"_synced_at", "TIMESTAMP NOT NULL DEFAULT NOW()"},
    {"_version", "BIGINT NOT NULL DEFAULT 1"},
    {"_hash", "TEXT"},
    {"_deleted", "BOOLEAN DEFAULT FALSE"},
    {"_created_at", "TIMESTAMP DEFAULT NOW()"},
    {"_updated_at", "TIMESTAMP DEFAULT NOW()"},
}
```
- `_gpay_source_id` → `source_id`.
- `_gpay_deleted` removed (đã có `_deleted`).

#### B.2 — `internal/handler/command_handler.go:192-208` DO block ADD CONSTRAINT
**Before**:
```sql
... constraint_name = 'uq_<t>_gpay_source_id' ...
ALTER TABLE ... ADD CONSTRAINT uq_<t>_gpay_source_id UNIQUE (_gpay_source_id);
```
**After**:
```sql
... constraint_name = 'uq_<t>_source_id' ...
ALTER TABLE ... ADD CONSTRAINT uq_<t>_source_id UNIQUE (source_id);
```

#### B.3 — `internal/handler/command_handler.go:617-636` CREATE TABLE inline
**Before**:
```go
`CREATE TABLE IF NOT EXISTS %s.%s (
    %s %s PRIMARY KEY,
    _gpay_source_id TEXT UNIQUE,
    _raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    ...
    _gpay_deleted BOOLEAN DEFAULT FALSE,
    _deleted BOOLEAN DEFAULT FALSE,
    ...
)`
```
**After**:
```go
`CREATE TABLE IF NOT EXISTS %s.%s (
    %s %s PRIMARY KEY,
    source_id TEXT UNIQUE,
    _raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    ...
    _deleted BOOLEAN DEFAULT FALSE,
    ...
)`
```

#### B.4 — `internal/handler/event_handler.go:233-244` tombstone INSERT
**Before**:
```sql
INSERT INTO %s (%s, _gpay_source_id, _deleted, _created_at, _updated_at, _source)
VALUES (?, ?::text, TRUE, NOW(), NOW(), 'debezium')
ON CONFLICT (%s) DO UPDATE SET _deleted = TRUE, _updated_at = NOW()
```
**After**:
```sql
INSERT INTO %s (%s, source_id, _deleted, _created_at, _updated_at, _source)
VALUES (?, ?::text, TRUE, NOW(), NOW(), 'debezium')
ON CONFLICT (%s) DO UPDATE SET _deleted = TRUE, _updated_at = NOW()
```

---

### Path C — centralized-data-service Master + Sinkworker V2

#### C.1 — `internal/sinkworker/upsert.go`
**Before**:
```go
var immutableOnUpdate = map[string]struct{}{
    "_gpay_id":        {},
    "_gpay_source_id": {},
    "_created_at":     {},
}
// SQL line 30, 67, 118:
... ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted DO UPDATE SET ...
```
**After**:
```go
var immutableOnUpdate = map[string]struct{}{
    "_gpay_id":        {},
    "source_id":       {},
    "_created_at":     {},
}
... ON CONFLICT (source_id) WHERE NOT _deleted DO UPDATE SET ...
```

#### C.2 — `internal/sinkworker/schema_manager.go:225-272`
**Before**:
```go
cols := []string{
    `"_gpay_id" BIGINT PRIMARY KEY`,
    `"_gpay_source_id" TEXT NOT NULL`,
    `"_raw_data" JSONB NOT NULL`,
    ...
    `"_gpay_deleted" BOOLEAN NOT NULL DEFAULT FALSE`,
}
// Partial UNIQUE INDEX:
CREATE UNIQUE INDEX IF NOT EXISTS ux_<t>_source_id_active ON "..."."..."  (_gpay_source_id) WHERE NOT _gpay_deleted;

systemFieldsSet := map[string]struct{}{
    "_gpay_id": {}, "_gpay_source_id": {}, ..., "_gpay_deleted": {},
}
```
**After**:
```go
cols := []string{
    `"_gpay_id" BIGINT PRIMARY KEY`,
    `"source_id" TEXT NOT NULL`,
    `"_raw_data" JSONB NOT NULL`,
    ...
    `"_deleted" BOOLEAN NOT NULL DEFAULT FALSE`,
}
CREATE UNIQUE INDEX IF NOT EXISTS ux_<t>_source_id_active ON "..."."..." (source_id) WHERE NOT _deleted;

systemFieldsSet := map[string]struct{}{
    "_gpay_id": {}, "source_id": {}, ..., "_deleted": {},
}
```

#### C.3 — `internal/sinkworker/sinkworker.go` (7 site: lines 40, 84, 117, 147, 154, 160, 256)
- Đổi map key + struct field reference `_gpay_source_id` → `source_id`.
- Đổi tham chiếu cột build record `_gpay_deleted` → `_deleted`.

#### C.4 — `internal/sinkworker/envelope.go:222`
- Đổi comment `extractSourceID returns _gpay_source_id` → `... returns source_id`.

#### C.5 — `internal/sinkworker/sinkworker_test.go` (9 site: lines 33, 80, 86, 93, 110, 114, 256, 263, 273)
- Test fixture map key + assertion đổi `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted`.

#### C.6 — `internal/service/transmuter.go:85-91`
**Before**:
```go
type shadowBatchRow struct {
    GpayID      int64  `gorm:"column:_gpay_id"`
    SourceID    string `gorm:"column:_gpay_source_id"`
    RawData     []byte `gorm:"column:_raw_data"`
    SourceTs    int64  `gorm:"column:_source_ts"`
    GpayDeleted bool   `gorm:"column:_gpay_deleted"`
}
```
**After**:
```go
type shadowBatchRow struct {
    GpayID    int64  `gorm:"column:_gpay_id"`
    SourceID  string `gorm:"column:source_id"`
    RawData   []byte `gorm:"column:_raw_data"`
    SourceTs  int64  `gorm:"column:_source_ts"`
    Deleted   bool   `gorm:"column:_deleted"`
}
```
- SQL SELECT FROM shadow + ON CONFLICT (master) đổi `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted` (lines 87, 90, 328, 335, 362, 367, 449, 456).

#### C.7 — `internal/service/master_ddl_generator.go:87-156`
**Before**:
```go
cols := []string{
    `"_gpay_id" BIGINT PRIMARY KEY`,
    `"_gpay_source_id" TEXT NOT NULL`,
    ...
    `"_gpay_deleted" BOOLEAN NOT NULL DEFAULT FALSE`,
}
// UNIQUE INDEX:
CREATE UNIQUE INDEX IF NOT EXISTS ux_<t>_source_id ON "..."."..." (_gpay_source_id);
```
**After**:
```go
cols := []string{
    `"_gpay_id" BIGINT PRIMARY KEY`,
    `"source_id" TEXT NOT NULL`,
    ...
    `"_deleted" BOOLEAN NOT NULL DEFAULT FALSE`,
}
CREATE UNIQUE INDEX IF NOT EXISTS ux_<t>_source_id ON "..."."..." (source_id);
```

#### C.8 — `internal/service/schema_adapter.go:497-540`
**Before**:
```go
if _, ok := schema.Columns["_gpay_source_id"]; ok {
    cols = append(cols, `"_gpay_source_id"`)
}
...
if _, ok := schema.Columns["_gpay_source_id"]; ok {
    placeholders = append(placeholders, "?")
    vals = append(vals, fmt.Sprintf("%v", gpayId))
}
```
**After**:
```go
if _, ok := schema.Columns["source_id"]; ok {
    cols = append(cols, `"source_id"`)
}
...
if _, ok := schema.Columns["source_id"]; ok {
    placeholders = append(placeholders, "?")
    vals = append(vals, fmt.Sprintf("%v", gpayId))
}
```

---

### Path D — Test fixtures + UI

#### D.1 — `centralized-data-service/test/internal/service/schema_adapter_ordering_test.go`
- Đổi PK fixture + tombstone column `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted` (lines 23, 30, 46, 53, 58, 62, 92-220).

#### D.2 — `centralized-data-service/test/internal/service/schema_adapter_test.go`
- Đổi V2 schema branch test (lines 11-83).

#### D.3 — `cdc-cms-web/src/pages/MasterRegistry.tsx:68, 425`
**Before**:
```tsx
spec: '{"pk":"_gpay_source_id"}',
```
**After**:
```tsx
spec: '{"pk":"source_id"}',
```

---

## DB MIGRATION SQL

> Apply SAU code deploy. Coordinate với prod DBA. Idempotent guards.

```sql
-- Migration: rename _gpay_source_id → source_id, _gpay_deleted → _deleted
-- Affected: shadow.* + master.* tables (loop từng table có _gpay_*)

BEGIN;

DO $$
DECLARE
    r RECORD;
BEGIN
    -- 1) RENAME _gpay_source_id → source_id (skip nếu source_id đã tồn tại)
    FOR r IN
        SELECT table_schema, table_name
        FROM information_schema.columns
        WHERE column_name = '_gpay_source_id'
          AND table_schema IN ('shadow', 'master')
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.table_schema
              AND table_name = r.table_name
              AND column_name = 'source_id'
        ) THEN
            EXECUTE format('ALTER TABLE %I.%I RENAME COLUMN _gpay_source_id TO source_id',
                           r.table_schema, r.table_name);
        ELSE
            -- Trùng (Path B trường hợp đặc biệt nếu có) → DROP _gpay_*
            EXECUTE format('ALTER TABLE %I.%I DROP COLUMN _gpay_source_id',
                           r.table_schema, r.table_name);
        END IF;
    END LOOP;

    -- 2) RENAME _gpay_deleted → _deleted (skip nếu _deleted đã tồn tại → DROP _gpay_deleted)
    FOR r IN
        SELECT table_schema, table_name
        FROM information_schema.columns
        WHERE column_name = '_gpay_deleted'
          AND table_schema IN ('shadow', 'master')
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.table_schema
              AND table_name = r.table_name
              AND column_name = '_deleted'
        ) THEN
            EXECUTE format('ALTER TABLE %I.%I RENAME COLUMN _gpay_deleted TO _deleted',
                           r.table_schema, r.table_name);
        ELSE
            -- Path B: tables yesterday có cả 2 cột → drop _gpay_deleted
            EXECUTE format('ALTER TABLE %I.%I DROP COLUMN _gpay_deleted',
                           r.table_schema, r.table_name);
        END IF;
    END LOOP;

    -- 3) Rebuild partial UNIQUE INDEX (sinkworker shadow tables)
    FOR r IN
        SELECT schemaname, tablename, indexname
        FROM pg_indexes
        WHERE indexdef ILIKE '%_gpay_source_id%' OR indexdef ILIKE '%_gpay_deleted%'
    LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I.%I', r.schemaname, r.indexname);
    END LOOP;

    -- Recreate index per table (PostgreSQL không inline được — phải explicit)
    -- NOTE: code Go schema_manager.go / master_ddl_generator.go sẽ self-heal khi service restart
    --       gọi `CREATE UNIQUE INDEX IF NOT EXISTS`. Phần này chỉ DROP để code rebuild với tên cột mới.

    -- 4) Rebuild named UNIQUE constraint (Path B handler-created tables)
    FOR r IN
        SELECT table_schema, table_name, constraint_name
        FROM information_schema.table_constraints
        WHERE constraint_name LIKE 'uq_%_gpay_source_id'
    LOOP
        EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT %I',
                       r.table_schema, r.table_name, r.constraint_name);
        -- Code command_handler.go sẽ ADD CONSTRAINT mới với tên uq_<t>_source_id ở lần create-default-columns kế tiếp.
    END LOOP;
END $$;

COMMIT;
```

**Idempotent**: chạy nhiều lần an toàn — guards `IF NOT EXISTS` cho rename target, `DROP CONSTRAINT IF EXISTS` cho constraint.

---

## Verify plan

### Build (3 service)
```bash
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go build ./...
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go build ./...
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-web && npm run build
```

### Vet (Go only, scoped)
```bash
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go vet ./internal/... ./test/...
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go vet ./internal/...
```
- Pre-existing sonyflake.go vet error (line 77, 82) tolerable — không phải do rename.

### Test
```bash
cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go test ./internal/... ./test/... -count=1 -timeout 120s
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go test ./internal/... -count=1 -timeout 60s
```

### Grep zero-residue (DoD destination)
```bash
cd /Users/trainguyen/Documents/work/data-hub
grep -rn "_gpay_source_id\|_gpay_deleted" \
  centralized-data-service/ cdc-cms-service/ cdc-cms-web/ \
  --include="*.go" --include="*.ts" --include="*.tsx" --include="*.sql"
```
- Expect: **0 hit** (trừ migration script nếu commit cùng repo).

### Destination verify PG
```sql
-- Sau migration:
\d shadow.export_jobs_2
-- Expect: source_id TEXT, _deleted BOOLEAN. KHÔNG _gpay_*.

\d master.export_jobs_2
-- Expect: _gpay_id BIGINT PK (giữ), source_id TEXT NOT NULL, _deleted BOOLEAN.

SELECT indexname, indexdef FROM pg_indexes WHERE schemaname IN ('shadow','master') AND indexdef ILIKE '%source_id%';
-- Expect: ux_<t>_source_id_active ON shadow.<t> (source_id) WHERE NOT _deleted;
--         ux_<t>_source_id ON master.<t> (source_id);
```

### Smoke test runtime
1. Tạo shadow table mới qua `/shadow` flow → expect cột `source_id`/`_deleted`, không có `_gpay_*`.
2. Trigger snapshot 1 record → expect master table có row, `master.<t>.source_id` = source row PK.
3. Trigger DELETE event → expect `master.<t>._deleted = TRUE`, partial UNIQUE INDEX không violate khi re-INSERT cùng source_id.
4. Preview API `/api/mapping/preview` → expect trả về field `source_id`, KHÔNG `_gpay_source_id`.

---

## Risk profile (unified plan)
| Aspect | Level | Mitigation |
|---|---|---|
| Files touched | 16 | Mechanical rename — typesafe Go + TS catch ngay |
| LOC delta NET | ~0 | Rename, không thêm logic |
| Code reversibility | HIGH | Single git revert |
| DB reversibility | MED | Reverse migration: RENAME COLUMN ngược + ADD back column |
| Production data risk | MED | ALTER COLUMN RENAME giữ data; DROP `_gpay_deleted` Path B mất data soft-delete (nhưng `_deleted` đã có giá trị tương đương) |
| Cross-service breaking change | MED | Code + DB phải deploy đồng bộ. Đề xuất deploy order: code (compatible với cả 2 schema bằng `IF EXISTS`)? — KHÔNG, code mới expect `source_id` → phải migration TRƯỚC hoặc CÙNG deploy |
| Partial UNIQUE INDEX rebuild | MED | DROP + code self-heal qua `CREATE UNIQUE INDEX IF NOT EXISTS` ở schema_manager.go khi service restart |

## Deploy order recommended
1. Tắt sinkworker + snapshot runner (drain in-flight).
2. Apply migration SQL.
3. Deploy code 3 service (đồng thời).
4. Bật lại sinkworker + snapshot runner.
5. Smoke test + destination verify.

## Lesson cross-check
- **2026-05-26 "Define DoD at the destination"**: DoD = grep `_gpay_*` = 0 + `\d shadow.<t>` không có `_gpay_*` + smoke test pass.
- **2026-05-20 "Verify ở destination"**: verify ở PG sau migration + smoke test runtime.
- **2026-05-28 lesson mới (sẽ promote)**: "Cleanup ≠ Remove. Khi user nói 2 field 'trùng' thì semantic là RENAME/MERGE chứ không phải DELETE. Verify intent NGỮ NGHĨA trước khi build option scope."
