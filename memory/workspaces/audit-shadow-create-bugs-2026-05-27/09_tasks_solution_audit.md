# 09_tasks_solution_audit — Code Demo cho Bug 1 + Bug 2

> Phase audit-only. Code dưới đây chưa apply, chờ user verb "ok / triển khai" → Muscle thực thi.

## SOL-1: Fix Bug 1 — Stop cross-entity field bleed

### File: `centralized-data-service/internal/handler/command_handler.go`
### Line: 649

**Before:**
```go
// 2. Add approved business fields (works for both new and existing tables)
// V2 Schema Migration: We read from mapping_rule_v2 via source_table join.
rules, err := h.mappingV2Repo.GetActiveRulesBySourceTable(context.Background(), payload.SourceTable)
```

**After:**
```go
// 2. Add approved business fields (works for both new and existing tables).
// FIX bug auto-bind cross-entity (2026-05-27): query by source_object_id
// thay vì source_object_name. Hai registry rows cùng source_table_name
// (vd. `export_jobs`) là use-case hợp lệ (chia thành 2 master khác nhau)
// — query theo NAME sẽ kéo mapping rules của registry kia → ALTER ADD
// COLUMN nhầm shadow. `effectiveID` đã resolve ở line 620-623 = V2
// source_object_id (ưu tiên) hoặc legacy registry_id (fallback).
rules, err := h.mappingV2Repo.ListActiveBySourceObject(context.Background(), effectiveID)
```

### Note signature
- `ListActiveBySourceObject(ctx, sourceObjectID int64)` đã tồn tại tại `mapping_rule_v2_repo.go:37-44`.
- `effectiveID` ở line 620-623 đã là `int64` — pass thẳng.

### Side effect
- Log line 656 hiện ghi `zap.String("source_table", payload.SourceTable)` — đổi thành `zap.Int64("source_object_id", effectiveID)` để trace_id signal đúng.
- Warn line 667 `"no active+approved mapping rules joined to source_object_name"` — update message thành `"no active+approved mapping rules for source_object_id"`.

### Verify
```go
// Test scenario:
// 1. Insert source_object_registry (id=42, source_object_name="export_jobs", target="master_exports_v1")
// 2. Insert 12 mapping_rule_v2 (source_object_id=42, status=approved, is_active=true)
// 3. Insert source_object_registry (id=99, source_object_name="export_jobs", target="sd_export_jobs_1")
// 4. Dispatch create-default-columns cho registry id=99
// Expected: shadow sd_export_jobs_1 CHỈ có cột system (10 cột), KHÔNG có 12 business cols của id=42.
// Before fix: shadow sd_export_jobs_1 có 10 + 12 = 22 cột (BUG).
```

---

## SOL-2: Fix Bug 2 — Add `_source_ts`, `_gpay_source_id`, `_gpay_deleted` vào shadow CREATE

### File: `centralized-data-service/internal/handler/command_handler.go`
### Patch A — Line 586-602 (CREATE TABLE)

**Before:**
```go
createSQL := fmt.Sprintf(
    `CREATE TABLE IF NOT EXISTS %s.%s (
        %s %s PRIMARY KEY,
        _raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
        _source VARCHAR(20) NOT NULL DEFAULT 'debezium',
        _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
        _version BIGINT NOT NULL DEFAULT 1,
        _hash VARCHAR(64),
        _deleted BOOLEAN DEFAULT FALSE,
        _created_at TIMESTAMP DEFAULT NOW(),
        _updated_at TIMESTAMP DEFAULT NOW()
    )`,
    quoteCommandIdent(schemaName),
    quoteCommandIdent(payload.TargetTable),
    quoteCommandIdent(pkField),
    pkType,
)
```

**After:**
```go
createSQL := fmt.Sprintf(
    `CREATE TABLE IF NOT EXISTS %s.%s (
        %s %s PRIMARY KEY,
        _gpay_source_id TEXT UNIQUE,
        _raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
        _source VARCHAR(20) NOT NULL DEFAULT 'debezium',
        _source_ts BIGINT,
        _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
        _version BIGINT NOT NULL DEFAULT 1,
        _hash VARCHAR(64),
        _gpay_deleted BOOLEAN DEFAULT FALSE,
        _deleted BOOLEAN DEFAULT FALSE,
        _created_at TIMESTAMP DEFAULT NOW(),
        _updated_at TIMESTAMP DEFAULT NOW()
    )`,
    quoteCommandIdent(schemaName),
    quoteCommandIdent(payload.TargetTable),
    quoteCommandIdent(pkField),
    pkType,
)
```

### Patch B — Line 163-172 (`ensureCDCColumnsInSchema` cdcColumns slice)

**Before:**
```go
cdcColumns := []struct{ name, def string }{
    {"_raw_data", "JSONB"},
    {"_source", "VARCHAR(20) DEFAULT 'debezium'"},
    {"_synced_at", "TIMESTAMP DEFAULT NOW()"},
    {"_version", "BIGINT DEFAULT 1"},
    {"_hash", "VARCHAR(64)"},
    {"_deleted", "BOOLEAN DEFAULT FALSE"},
    {"_created_at", "TIMESTAMP DEFAULT NOW()"},
    {"_updated_at", "TIMESTAMP DEFAULT NOW()"},
}
```

**After:**
```go
// FIX bug missing system columns (2026-05-27): _source_ts là OCC anchor
// (sinkworker/upsert.go:69-122 dùng EXCLUDED._source_ts > shadow._source_ts);
// _gpay_source_id là V2 UNIQUE anchor cho master ON CONFLICT;
// _gpay_deleted là tombstone soft-delete. Cả 3 đều mandatory per
// project_context.md §Shadow Layer required cols.
cdcColumns := []struct{ name, def string }{
    {"_gpay_source_id", "TEXT"},
    {"_raw_data", "JSONB"},
    {"_source", "VARCHAR(20) DEFAULT 'debezium'"},
    {"_source_ts", "BIGINT"},
    {"_synced_at", "TIMESTAMP DEFAULT NOW()"},
    {"_version", "BIGINT DEFAULT 1"},
    {"_hash", "VARCHAR(64)"},
    {"_gpay_deleted", "BOOLEAN DEFAULT FALSE"},
    {"_deleted", "BOOLEAN DEFAULT FALSE"},
    {"_created_at", "TIMESTAMP DEFAULT NOW()"},
    {"_updated_at", "TIMESTAMP DEFAULT NOW()"},
}
```

### Patch C — Line 176-177 (thêm index + unique constraint cho idempotency)

**Before:**
```go
indexName := fmt.Sprintf("idx_%s_raw", tableName)
h.shadowDB.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s.%s USING GIN(_raw_data)`, quoteCommandIdent(indexName), quoteCommandIdent(schemaName), quoteCommandIdent(tableName)))
return nil
```

**After:**
```go
indexName := fmt.Sprintf("idx_%s_raw", tableName)
h.shadowDB.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s.%s USING GIN(_raw_data)`, quoteCommandIdent(indexName), quoteCommandIdent(schemaName), quoteCommandIdent(tableName)))

// OCC sort index — sinkworker.upsert dùng _source_ts older-wins guard,
// shadow load by anchor cần lookup nhanh. Match sinkworker/schema_manager.go:231.
sourceTsIdx := fmt.Sprintf("idx_%s_source_ts", tableName)
h.shadowDB.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s.%s(_source_ts)`, quoteCommandIdent(sourceTsIdx), quoteCommandIdent(schemaName), quoteCommandIdent(tableName)))

// UNIQUE constraint cho ON CONFLICT key — ALTER ADD COLUMN không tự thêm
// UNIQUE nếu cột tồn tại sẵn. Dùng DO-block để idempotent (skip nếu đã có).
h.shadowDB.Exec(fmt.Sprintf(`
    DO $$ BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'uq_%s_gpay_source_id'
        ) THEN
            ALTER TABLE %s.%s ADD CONSTRAINT %s UNIQUE (_gpay_source_id);
        END IF;
    END $$`,
    tableName,
    quoteCommandIdent(schemaName), quoteCommandIdent(tableName),
    quoteCommandIdent(fmt.Sprintf("uq_%s_gpay_source_id", tableName)),
))

return nil
```

### Verify
```bash
# After applying:
go build ./...                              # PASS
go vet ./...                                # PASS
go test ./internal/handler/... -run TestCreateDefaultColumns   # nếu có test, phải PASS

# Live verify:
# 1. FE /shadow → tạo sd_test_2026
# 2. psql -h localhost -p 5436 -U postgres -d shadow -c "\\d+ shadow_xxx.sd_test_2026"
# Expect: thấy đủ 11 cột system (kể cả _source_ts, _gpay_source_id, _gpay_deleted).
# 3. Insert _raw_data với _source_ts cũ → sinkworker OCC guard hoạt động:
#    WHERE EXCLUDED._source_ts > shadow._source_ts phải reference được cột thật.
```

---

## Estimated touched LOC
| File | Lines changed | Reason |
|---|---|---|
| `centralized-data-service/internal/handler/command_handler.go` | ~24 (3 patch sites) | SOL-1 swap caller (line 649 + 2 log fields) + SOL-2 add 3 cols × 2 vị trí + index + UNIQUE |
| **Total source code** | **~24 lines** | (chưa kể test) |

## Out-of-scope ở task này
- Migration shadow đã tồn tại lỗi → `08_tasks_audit.md` MIGR-1.
- Refactor `HandleCreateDefaultColumns` → `sinkworker/schema_manager` → `10_gap_analysis.md` GAP-1.
- Sửa `HandleScanFields` line 1389 (cũng dùng `GetActiveRulesBySourceTable`) → `10_gap_analysis.md` GAP-2.
