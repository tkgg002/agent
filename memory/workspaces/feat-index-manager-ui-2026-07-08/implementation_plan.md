# Kế hoạch Khắc phục Lock Contention — Quản lý Index qua CMS UI

## 1. Bối cảnh & Gốc rễ vấn đề

### Hiện trạng từ Audit
Từ logs runtime thực tế, các slow query vẫn đang xảy ra **SAU KHI** deploy code tạo index:

| Slow Query | Elapsed | Root Cause |
|:---|:---|:---|
| `COUNT(*) FROM shadow WHERE _deleted = true` | **6.7s** | Partial index `_deleted` chưa tồn tại trên bảng cũ |
| `BucketCounts` (GROUP BY _source_ts) | **6.4s** | Thiếu index `_source_ts` trên shadow |
| `ListIDTsInWindow` | **7.6s** | Thiếu index timestamp nghiệp vụ |
| `COUNT(*) FROM master...` | **2.2s** | Full table scan, thiếu index |

### Nguyên nhân gốc
Code tạo index chỉ chạy khi **setup bảng mới** (`EnsureCDCColumnsInSchema` / `EnsureMaster`). Bảng **đã tồn tại** trước khi deploy code mới **KHÔNG** được retroactively tạo index.

### Giải pháp: CMS UI quản lý Index
Thêm **Index Manager** vào CMS UI. Operator có thể:
1. **Xem** danh sách index hiện có trên bất kỳ shadow/master table
2. **Tạo** index thủ công (CONCURRENTLY) cho bảng cần tối ưu
3. **Xóa** index không cần thiết

---

## 2. Kiến trúc hệ thống hiện tại (quan trọng — phân quyền service)

> [!IMPORTANT]
> **Phân tách DB connections:**
> - `cdc-cms-service` (control plane): có `shadowReader` port → **query trực tiếp Shadow DB** (dùng cho `ShadowColumns`). **KHÔNG có** connection tới Master DB (port 5434).
> - `centralized-data-service` (worker): có connection tới **cả Shadow DB và Master DB** — nơi duy nhất thực thi DDL (`EnsureMaster`, `EnsureCDCColumnsInSchema`, `ALTER TABLE`, `CREATE INDEX`).

> [!CAUTION]
> **Quy tắc bất di bất dịch**: Mọi DDL mutation (CREATE INDEX, DROP INDEX) **PHẢI** đi qua worker (`centralized-data-service`) thông qua NATS RPC — giống pattern hiện có của `cdc.cmd.create-default-columns`, `cdc.cmd.scan-array`, v.v. CMS Service chỉ là proxy dispatcher.

### Luồng xử lý

```
[CMS Web]                [CMS Service]              [Worker (centralized-data-service)]
    │                         │                              │
    │ GET /introspection/     │                              │
    │   indexes/:table        │                              │
    │ (plane=shadow)          │──shadowReader.GetIndexes()──→│ (query trực tiếp Shadow DB)
    │ (plane=master)          │──NATS RPC ──────────────────→│ cdc.cmd.introspect-indexes
    │                         │                              │   → query Master DB pg_indexes
    │                         │                              │
    │ POST /introspection/    │                              │
    │   indexes               │──NATS RPC ──────────────────→│ cdc.cmd.create-index
    │                         │                              │   → CREATE INDEX CONCURRENTLY
    │                         │                              │     (shadow hoặc master)
    │                         │                              │
    │ DELETE /introspection/  │                              │
    │   indexes/:name         │──NATS RPC ──────────────────→│ cdc.cmd.drop-index
    │                         │                              │   → DROP INDEX CONCURRENTLY
```

---

## 3. Proposed Changes

### 3.1. Worker — `centralized-data-service` (nơi thực thi DDL)

#### [NEW] NATS Handler: `cdc.cmd.introspect-indexes`
- Đăng ký trong `server_setup.go` (pattern giống `cdc.cmd.scan-raw-data`)
- Nhận payload: `{ "schema": "...", "table": "...", "plane": "shadow"|"master", "reply_to": "..." }`
- Logic: Query `pg_indexes` + `pg_stat_user_indexes` trên đúng DB connection (shadow hoặc master via `connMgr`)
- Trả về danh sách index: name, columns, definition, size, scan count, is_valid

#### [NEW] NATS Handler: `cdc.cmd.create-index`
- Nhận payload:
```json
{
  "schema": "shadow_test_pmb",
  "table": "payment_bills",
  "columns": ["_deleted"],
  "plane": "shadow",
  "is_partial": true,
  "where_clause": "_deleted = true",
  "is_unique": false,
  "reply_to": "..."
}
```
- Logic:
  - Validate identifier injection (whitelist `[a-zA-Z0-9_]`)
  - Generate `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_{table}_{col1}[_partial] ON {schema}.{table} ({columns}) [WHERE clause]`
  - Execute trên đúng DB connection (shadow hoặc master)
  - `CREATE INDEX CONCURRENTLY` chạy ngoài transaction (dùng raw `sql.DB.ExecContext`)

#### [NEW] NATS Handler: `cdc.cmd.drop-index`
- Nhận payload: `{ "schema": "...", "index_name": "...", "plane": "shadow"|"master", "reply_to": "..." }`
- Logic: `DROP INDEX CONCURRENTLY IF EXISTS {schema}.{index_name}`
- Guard: Cấm drop index bắt đầu bằng `pk_` hoặc `ux_` (primary/unique constraint)

---

## 3.2. CMS Service — `cdc-cms-service` (proxy dispatcher)

#### [MODIFY] `internal/api/system/introspection_handler.go`

Thêm 3 method mới vào `IntrospectionHandler`, tuân theo pattern RPC hiện có:

**`ListIndexes(c *fiber.Ctx) error`** — GET `/introspection/indexes/:table`
- Query param: `schema`, `plane` = `shadow` | `master`
- Nếu `plane=shadow` VÀ `shadowReader` có khả năng query `pg_indexes` → query trực tiếp (performance tốt hơn)
- Nếu `plane=master` → bắt buộc RPC qua NATS `cdc.cmd.introspect-indexes` (CMS không có master DB connection)

**`CreateIndex(c *fiber.Ctx) error`** — POST `/introspection/indexes`
- Proxy body tới worker qua NATS RPC `cdc.cmd.create-index`
- Timeout 30s (CONCURRENTLY có thể lâu trên bảng lớn)

**`DropIndex(c *fiber.Ctx) error`** — DELETE `/introspection/indexes/:name`
- Proxy qua NATS RPC `cdc.cmd.drop-index`

#### [MODIFY] `internal/router/router.go`
```go
dual("GET",    shared, "/introspection/indexes/:table", h.System.Introspection.ListIndexes)
dual("POST",   shared, "/introspection/indexes",        h.System.Introspection.CreateIndex)
dual("DELETE", shared, "/introspection/indexes/:name",   h.System.Introspection.DropIndex)
```

#### [MODIFY] `internal/app/ports/shadow_reader.go` (hoặc file port tương ứng)
- Thêm method `GetIndexes(ctx, schema, table string) ([]IndexInfo, error)` vào interface `ShadowSchemaReader` nếu chọn query trực tiếp cho shadow plane.

---

## 3.3. CMS Web — `cdc-cms-web` (Frontend)

#### [NEW] `src/components/TableIndexManager.tsx`

Component tái sử dụng cho cả Shadow và Master page.

**Props:**
```typescript
interface TableIndexManagerProps {
  schema: string;
  table: string;
  plane: 'shadow' | 'master';
}
```

**UI:**
- **Card** với title "📊 Table Indexes"
- **Table** hiển thị index hiện có: Name, Columns, Type (Unique/Partial/BTree), Size, Scans, Valid
  - Index invalid → Tag đỏ "INVALID"
  - Index size format humanize (KB/MB)
- **Button "Add Index"** → Modal form:
  - Input: Columns (multi-select từ `shadow-columns` API đã có)
  - Checkbox: Partial? → Input: WHERE clause
  - Checkbox: Unique?
- **Button "Drop"** trên mỗi row → Confirm modal → gọi DELETE
- **Button "Refresh"** reload danh sách
- **Suggested Indexes:** Auto-detect index thiếu vs danh sách khuyến nghị:
  - `_deleted` partial index → `WHERE _deleted = true`
  - `_source_ts` index
  - `_source_id` index
  - `_updated_at` index
  - Nếu thiếu → Alert warning + Button "Tạo tất cả"

#### [MODIFY] `src/pages/MappingFieldsPage.tsx`
```tsx
{registry && registry.shadow_schema && (
  <TableIndexManager schema={registry.shadow_schema} table={registry.target_table} plane="shadow" />
)}
```

#### [MODIFY] `src/pages/MasterMappingFieldsPage.tsx`
```tsx
{binding && (
  <TableIndexManager schema={binding.master_schema || 'public'} table={binding.master_name} plane="master" />
)}
```

---

## 4. Kế hoạch xác minh

### Kiểm thử tự động
- `go build ./cmd/... ./internal/...` trên cả `centralized-data-service` và `cdc-cms-service`
- `go test ./internal/...` trên cả 2 service
- `npm run build` cho FE

### Kiểm thử thủ công (Browser)
1. Mở `/shadow/4/mappings?binding_id=10` → Card "Table Indexes" hiển thị index hiện có
2. Bấm "Add Index" → tạo partial index `_deleted WHERE _deleted = true` → verify xuất hiện trong list
3. Mở `/masters/schedule_histories/mappings?binding_id=4` → tương tự cho master
4. Chạy recon cycle, check logs không còn `elapsed > 2s`
