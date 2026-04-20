# Plan: BatchBuffer Redesign — Dynamic Schema Adapter

> Date: 2026-04-15
> Role: Brain
> Priority: P0 — chặn toàn bộ Debezium → Postgres flow
> Status: PLAN READY — session mới tự chủ thực hiện theo Rule 2

## Root Cause

BatchBuffer hardcode column names, types, constraints. Mỗi table schema khác → lỗi khác → fix mì ăn liền 8-9 lần. Không systematic.

## Giải pháp: Schema-Aware Upsert

### Nguyên tắc
1. **KHÔNG hardcode bất kỳ column name nào** — đọc từ DB
2. **Đọc target table schema 1 lần, cache** — không query mỗi INSERT
3. **Adapter tự xử lý**: NOT NULL drop, UNIQUE constraint, JSONB cast, column quoting
4. **Hoạt động cho BẤT KỲ table nào** — không chỉ export_jobs

### Implementation

#### File mới: `internal/service/schema_adapter.go`

```go
type TableSchema struct {
    Columns     map[string]ColumnInfo  // column_name → info
    PKColumn    string
    HasUnique   bool
    JSONBCols   map[string]bool
}

type ColumnInfo struct {
    Name       string
    DataType   string
    IsNullable bool
    HasDefault bool
}

// LoadTableSchema reads target table schema from information_schema (cached)
func LoadTableSchema(db *gorm.DB, tableName string) *TableSchema

// PrepareForCDCInsert ensures table ready for CDC upserts:
// 1. Add CDC columns if missing (_raw_data, _source, _synced_at, _version, _hash, _deleted, _created_at, _updated_at)
// 2. Drop NOT NULL on _airbyte_* columns
// 3. Add UNIQUE on PK if missing
// All dynamic — reads actual schema, no hardcoded names
func PrepareForCDCInsert(db *gorm.DB, tableName, pkColumn string) error

// BuildUpsertSQL builds INSERT ON CONFLICT with proper quoting + JSONB casting
// Uses TableSchema to know which columns are JSONB, which exist
func BuildUpsertSQL(schema *TableSchema, tableName string, record *UpsertRecord) (string, []interface{})
```

#### Fix BatchBuffer
- Remove ALL hardcoded column names
- Use `schema_adapter.LoadTableSchema()` (cached per table)
- Use `schema_adapter.PrepareForCDCInsert()` once per table
- Use `schema_adapter.BuildUpsertSQL()` per record

### Cache strategy
- `sync.Map` keyed by table name
- Load once on first INSERT for each table
- Invalidate on `schema.config.reload` NATS event

### JSONB handling
- Read `data_type` from `information_schema.columns`
- If JSONB: check value is valid JSON, base64 decode if needed
- If TEXT/VARCHAR: send as string
- NEVER hardcode which columns are JSONB

### NOT NULL handling
- Read `is_nullable` from `information_schema.columns`
- Columns starting with `_airbyte_` that are NOT NULL → DROP NOT NULL once
- Do NOT hardcode column names — pattern match `_airbyte_%`

### UNIQUE handling
- Query `pg_constraint` for UNIQUE on PK column
- If missing → ADD UNIQUE once
- Cache result

## Execution
Session mới:
1. Đọc file này
2. Revert tất cả hardcode fixes trong batch_buffer.go
3. Implement schema_adapter.go
4. Integrate vào BatchBuffer
5. Test với export_jobs + refund_requests
6. Verify: MongoDB insert → Kafka → Worker → Postgres (data ở đích)
7. Ghi progress + update tasks

## Definition of Done
- [ ] MongoDB insert → Postgres trong < 5 giây (bất kỳ table nào)
- [ ] KHÔNG hardcode bất kỳ column name nào
- [ ] Hoạt động cho export_jobs + refund_requests + bất kỳ table mới
- [ ] Redpanda Console: consumer lag = 0
- [ ] No errors trong Worker log
