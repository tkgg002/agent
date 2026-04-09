# Solution: Phase 1.11 — Fix CDC Tồn Đọng

> **Date**: 2026-04-07
> **Agent**: Muscle:claude-opus-4-6
> **Scope**: 5 bug fixes (P0-P2)

---

## Bug 1 (P0 CRITICAL): SourceTable vs TargetTable Mismatch

### Root Cause
`extractSourceAndTable()` trả về SourceTable (e.g., `merchants`) từ NATS subject `cdc.goopay.{db}.{table}`.
Nhưng `GetTableConfig()` và `GetMappingRules()` lookup cache theo TargetTable (e.g., `cdc_merchants`).
→ Events bị drop silently.

### Fix
**File 1**: `centralized-data-service/internal/service/registry_service.go`
- Thêm `sourceCache map[string]*model.TableRegistry` vào struct
- Populate trong `ReloadAll()` cùng loop hiện tại
- Thêm `GetTableConfigBySource(sourceTable) *model.TableRegistry`

**File 2**: `centralized-data-service/internal/handler/event_handler.go`
- Đổi `GetTableConfig(tableName)` → `GetTableConfigBySource(tableName)`
- Dùng `tableConfig.TargetTable` cho: SQL table name, mapping rules lookup, batch buffer, handleDelete

---

## Bug 2 (P1): Missing PATCH /api/mapping-rules/:id

### Root Cause
FE `SchemaChanges.tsx` gọi `PATCH /api/mapping-rules/:id` để approve/reject nhưng endpoint không tồn tại → 404.

### Fix
**File 3**: `cdc-cms-service/internal/api/mapping_rule_handler.go` — thêm `UpdateStatus()` method
**File 4**: `cdc-cms-service/internal/router/router.go` — đăng ký `admin.Patch("/mapping-rules/:id", ...)`

---

## Bug 3 (P1): List API bỏ qua filter status

### Root Cause
`MappingRuleHandler.List()` chỉ đọc query `table`, bỏ qua `status` và `rule_type` → FE filter không hoạt động.

### Fix
**File 3** (cùng file Bug 2): Update `List()` để đọc `status`, `rule_type` query params, gọi `GetAllFiltered()`

---

## Bug 4 (P1): NATS reload payload không nhất quán

### Root Cause
5 call sites publish `schema.config.reload`:
- 1 chỗ gửi JSON (approval_service.go) ✓
- 4 chỗ gửi plain string ✗

### Fix
**File 5**: `cdc-cms-service/pkgs/natsconn/nats_client.go` — thêm `PublishReload()` helper
**File 6**: `cdc-cms-service/internal/api/registry_handler.go` — 3 chỗ
**File 7**: `cdc-cms-service/internal/api/mapping_rule_handler.go` — 1 chỗ
**File 8**: `cdc-cms-service/internal/api/airbyte_handler.go` — 1 chỗ
**File 9**: `cdc-cms-service/internal/service/approval_service.go` — refactor dùng helper

---

## Bug 5 (P2): SchemaChanges page đặt tên sai

### Fix
**File 10**: `cdc-cms-web/src/App.tsx` — đổi menu label "Schema Changes" → "Mapping Approval"
