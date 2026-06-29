# Progress — bug-schema-shadow-cls-testing-not-exist-2026-06-23

## Root Cause Analysis (Completed)

### [2026-06-23T09:18 Agent:Opus-4.6] RCA Hoàn tất

**Root Cause xác nhận**: Refactor `command_handler.go` → `schema_ddl_handler.go` bỏ sót 6 logic paths.

## Execution Log

### [2026-06-23T09:21 Agent:Opus-4.6] Component 1: CreateEmptyTable
- ✅ Added `CREATE SCHEMA IF NOT EXISTS` before `CREATE TABLE`
- ✅ Verified: no SQLite hack in codebase

### [2026-06-23T09:22 Agent:Opus-4.6] Component 2: SchemaAdapter.ListColumnsWithType
- ✅ Added new method, restored from pre-refactor `listShadowColumnsWithType`

### [2026-06-23T09:23 Agent:Opus-4.6] Component 2: HandleCreateDefaultColumns
- ✅ Step 5: Auto-discovery via `fieldScanner.ScanFieldsDebezium`
- ✅ Step 6: Sync approved mapping rules → ALTER TABLE ADD COLUMN + type drift
- ✅ Step 8: Update registry states (`is_table_created`, `ddl_status`)
- ✅ Step 7: Bridge to mapping v2 (was already there, renumbered)

### [2026-06-23T09:24 Agent:Opus-4.6] Verification
- ✅ `go build ./...` — pass
- ✅ `go test ./test/internal/service/...` — pass (0.793s)
- ✅ `go test ./test/internal/handler/...` — pass (4.362s)


### [2026-06-23T09:40 Agent:Antigravity] Database Connection Alignment
- ✅ Modify `worker_server_init.go` to inject `shadowDB` into `schemaAdapter` instead of `db`.
- ✅ Verified: `go build ./...` passed
- ✅ Verified: `go test -count=1 ./test/internal/service/... ./test/internal/handler/...` passed (service: 0.854s, handler: 4.580s)

### Pending
- ⏳ User restart worker service (`centralized-data-service`) & verify runtime
