# Implementation Plan: Phase 1.10 — Schema Scan, Mapping Refactor & FE Bug Fixes

> Created: 2026-04-07T13:54 | Agent: Brain:claude-sonnet-4-6-thinking

## Tóm tắt

4 nhóm thay đổi theo yêu cầu User ngày 2026-04-07:

| # | Nhóm | Scope | Priority |
|---|------|-------|----------|
| 1 | FE Registry — title align-left, full columns | cdc-cms-web | P0 |
| 2 | Backend — Scan Source DB → cdc_table_registry (is_active=false) | cdc-cms-service | P0 |
| 3 | Backend — Scan Fields → cdc_mapping_rules (status + rule_type) | cdc-cms-service | P0 |
| 4 | FE Bugs — QueueMonitoring crash + SchemaChanges list field | cdc-cms-web | P0 |

---

## User Review Required

**Quyết định thiết kế `cdc_mapping_rules`**: Thêm 2 cột mới:
- `status VARCHAR(20) DEFAULT 'approved'` — giá trị: `pending` | `approved` | `rejected`
- `rule_type VARCHAR(20) DEFAULT 'mapping'` — giá trị: `system` | `discovered` | `mapping`

Logic `pending_fields` table sẽ được **deprecated** — duyệt field sẽ xảy ra trực tiếp trên `cdc_mapping_rules`.

---

## Proposed Changes

### A. FE TableRegistry UI Fix

**File**: `cdc-cms-web/src/pages/TableRegistry.tsx`

- Đảm bảo `<Title level={4}>` dùng `style={{ textAlign: 'left' }}`
- Bổ sung columns: `Source DB`, `Airbyte Connection ID`, `Created At`

---

### B. Backend: Migration + Model + Handlers

#### Migration SQL (003_add_mapping_rule_status.sql)

- Đã khởi tạo: thêm `status` (pending|approved) và `rule_type` (system|discovered|mapping) vào `cdc_mapping_rules`.

#### mapping_rule.go — thêm 2 field

```go
Status   string `gorm:"column:status;default:approved" json:"status"`
RuleType string `gorm:"column:rule_type;default:mapping" json:"rule_type"`
```

#### registry_handler.go — ScanSource handler

```
POST /api/registry/scan-source?source_id=xxx
```

Logic:
1. `DiscoverSchema(ctx, sourceID)` → lấy catalog từ Airbyte Source.
2. Với mỗi stream: check `(source_db, source_table)` unique trong Registry.
3. **INSERT** bảng mới nếu chưa có: `is_active=false`, `sync_engine='airbyte'`.
4. Trả về trạng thái quét: `{ added: N, skipped: M, total: K }`.

#### registry_handler.go — ScanFields handler (Source-First)

```
POST /api/registry/:id/scan-fields
```

Logic:
1. Lấy registry entry by ID.
2. `DiscoverSchema(ctx, entry.AirbyteSourceID)` → quét schema từ Nguồn (Airbyte).
3. Duyệt field trong JSON Schema của stream:
   - System fields (`_raw_data`, `_source`, v.v.) → `rule_type='system'`, `status='approved'`.
   - Business fields mới → `rule_type='discovered'`, `status='pending'`.
4. Skip nếu field đã tồn tại trong mapping rules.
5. Gán mapping type mặc định dựa trên JSON type (string -> text, number -> bigint/numeric...).
6. Trả về: `{ added: N, total: K }`.

#### router.go — đăng ký routes mới

```go
registry.Post("/scan-source", registryHandler.ScanSource)
registry.Post("/:id/scan-fields", registryHandler.ScanFields)
```

---

### C. FE Bug Fixes

#### QueueMonitoring.tsx — Fix crash (black screen)

**Root Cause**: `stats.queue.pool_size = 0` → division by zero.

- Fix logic tính toán `workerActivePercent` và `bufferFullPercent` với kiểm tra mẫu số khác không.

#### SchemaChanges.tsx — Refactor Mapping

- Thay đổi endpoint fetch dữ liệu sang `/api/mapping-rules?status=pending`.
- Mapping lại cấu trúc dữ liệu từ `MappingRule` model.

---

## Verification Plan

```bash
# Backend build
cd cdc-cms-service && go build ./...

# Test endpoints
curl -X POST http://localhost:8083/api/registry/scan-source?source_id=... -H "Authorization: Bearer $TOKEN"
curl -X POST http://localhost:8083/api/registry/1/scan-fields -H "Authorization: Bearer $TOKEN"
```

### FE Manual
1. `/registry` → tiêu đề căn lề trái, hiển thị đủ cột mới.
2. `/queue` → truy cập bình thường ngay cả khi stats rỗng.
3. `/schema-changes` → hiển thị đúng các fields `pending` từ `cdc_mapping_rules`.
