tôi cần chay các flow tiêu chuẩn. 

tôi sẽ chuẩn bị
 - xoá hết table/stream hiện tại. 
 - xoá hết các maping field
 - chỉ còn connector
Mong muốn
luồng 1 : căn bản chưa có update field từ table gốc
    - mong muốn 1 : hệ thống tự quét full stream (cả active và non-active) từ airbyte
    - mong muốn 2 : hệ thống tự tạo những record cdc_table_registry cho những stream đang active (với config lấy từ airbyte)
    - mong muốn 3 : hệ thống tự lấy những field của stream đang active (từ airbyte), tạo record cdc_mapping_rules + trạng thái là đã approved
    - mong muốn 4 : khi click tạo field default, sẽ tạo các field default vào table đích. 

Luồng 2 : airbyte chạy, kiểm tra xem có field thay đổi không, có thì ghi json thông tin các field thay đổi vào field _raw_data.
    - Note : cơ chế hiện tại là luôn update toàn bộ vào field _raw_data => xem xét 2 trường hợp, 1 là ghi toàn bộ vào, 2 là chỉ những record nào có sự thay đổi mới ghi vào field này

Luồng 3 : có update field mới từ table gốc
    - mong muốn 1 : hệ thống tự lấy những field đc thêm mới của stream đang active (từ airbyte), tạo record cdc_mapping_rules
    - mong muốn 2 : khi click duyêt field, sẽ tạo các field này vào table đích. 
    - mong muốn 3 : khi click bổ sung, sẽ chạy job đi tìm các record có _raw_data chứa dữ liệu các field mới này, bổ sung vào data đích

--- 


# Plan v1.13: Recheck Flow — Xác nhận Phase 1 đáp ứng yêu cầu

> **Date**: 2026-04-13
> **Version**: 1.13
> **Mục tiêu**: Kiểm tra 3 luồng tiêu chuẩn trên hệ thống CDC, xác nhận Phase 1 đã hoạt động đúng hay cần bổ sung
> **Prerequisite**: User đã xoá hết table/stream/mapping, chỉ còn connector trên Airbyte

---

## Điều kiện khởi đầu

- Airbyte: Chỉ còn connector (source + destination), **không còn** stream/table/mapping nào trong CMS
- PostgreSQL (DW): Các bảng `cdc_table_registry`, `cdc_mapping_rules` đã được **xoá sạch records**
- CDC tables: Đã DROP hết
- Mong muốn: Hệ thống tự động làm lại từ đầu thông qua 3 luồng

---

## Luồng 1: Căn bản — Chưa có update field từ table gốc

> Mục tiêu: Hệ thống tự quét Airbyte, tạo registry, tạo mapping rules, tạo CDC tables — **tối thiểu manual steps**.

### Mong muốn 1.1: Hệ thống tự quét full stream từ Airbyte

**Kỳ vọng**: Gọi 1 API → hệ thống quét TẤT CẢ streams (cả active và non-active) từ mọi connection.

**API hiện tại**: `POST /api/registry/sync-from-airbyte`

**Cần verify**:
- [ ] API quét tất cả connections
- [ ] Lấy đủ streams (cả selected=true lẫn selected=false)
- [ ] Phân biệt active / non-active trong kết quả
- [ ] Response: `{ added: N, skipped: N, total: N }`

**Gap tiềm năng**:
- Hiện tại `SyncFromAirbyte` có thể chỉ import streams `selected=true` → cần check code
- Nếu chỉ import active → cần bổ sung option import cả non-active (is_active=false)

### Mong muốn 1.2: Tự tạo cdc_table_registry cho streams đang active

**Kỳ vọng**: Sau sync, mỗi stream active trên Airbyte có 1 record trong `cdc_table_registry` với config lấy từ Airbyte (sync_mode, cursor_field, namespace, connection_id, raw_table name...).

**Cần verify**:
- [ ] Registry entry tạo đúng với: source_db, source_table, target_table, sync_engine
- [ ] Airbyte metadata populated: airbyte_connection_id, airbyte_sync_mode, airbyte_cursor_field, airbyte_namespace, airbyte_raw_table
- [ ] primary_key_field lấy từ Airbyte catalog `sourceDefinedPrimaryKey` (không hardcode "id")
- [ ] is_active = true cho streams selected

### Mong muốn 1.3: Tự lấy fields + tạo cdc_mapping_rules (status = approved)

**Kỳ vọng**: Hệ thống tự parse Airbyte JSONSchema → tạo mapping rules cho TẤT CẢ fields → status = **approved** (không cần duyệt thủ công lần đầu).

**API hiện tại**: `POST /api/airbyte/import/execute` (import stream + discover fields)

**Cần verify**:
- [ ] Parse Airbyte JSONSchema properties → tạo cdc_mapping_rules
- [ ] Mỗi field: source_field, target_column, data_type (inferred), status
- [ ] **Status mặc định = "approved"** (lần import đầu tiên, không cần pending)
- [ ] Type inference: string→TEXT, integer→BIGINT, number→NUMERIC, boolean→BOOLEAN, object→JSONB

**Gap tiềm năng**:
- Hiện tại status mặc định có thể là "pending" → cần đổi sang "approved" cho lần import đầu
- Hoặc thêm param `?auto_approve=true` trên API

### Mong muốn 1.4: Click tạo field default → tạo columns vào table đích

**Kỳ vọng**: 1 action trên CMS → hệ thống:
1. Tạo CDC table (nếu chưa có) với schema v1.12 (id BIGINT + source_id)
2. ALTER TABLE ADD COLUMN cho tất cả approved mapping rules

**API hiện tại**:
- `POST /api/registry/:id/standardize` → tạo CDC table
- `POST /api/schema-changes/:id/approve` → ALTER TABLE add column

**Cần verify**:
- [ ] Standardize tạo table đúng v1.12 schema
- [ ] Có flow "tạo tất cả approved columns" 1 lần (batch standardize)
- [ ] Hoặc: standardize tự động add columns từ approved rules

**Gap tiềm năng**:
- Có thể cần 1 API mới: `POST /api/registry/:id/create-default-columns` — tạo table + add tất cả columns từ approved rules trong 1 step

---

## Luồng 2: Airbyte chạy — Kiểm tra _raw_data cập nhật

> Mục tiêu: Khi Airbyte sync data, hệ thống bridge vào CDC tables và cập nhật `_raw_data`.

### Cơ chế hiện tại

Bridge (`HandleAirbyteBridge`) đọc từ Airbyte table → `INSERT ... ON CONFLICT DO UPDATE SET _raw_data = EXCLUDED._raw_data WHERE _hash IS DISTINCT FROM EXCLUDED._hash`.

→ **Chỉ update _raw_data khi hash thay đổi** (tức data có sự thay đổi).

### 2 trường hợp cần xem xét

**Trường hợp A: Ghi toàn bộ vào _raw_data**
- Hiện tại: Bridge ghi `to_jsonb(src.*)` → tất cả columns → _raw_data
- Ưu: Đơn giản, _raw_data luôn chứa snapshot mới nhất
- Nhược: Storage lớn hơn

**Trường hợp B: Chỉ ghi record có thay đổi**
- Hiện tại: Bridge đã có `WHERE _hash IS DISTINCT FROM EXCLUDED._hash` → chỉ UPDATE khi data thay đổi
- Bridge cũng hỗ trợ incremental: `WHERE _airbyte_extracted_at > last_bridge_at`
- → **Đã đáp ứng case B** ở mức row level

**Cần verify**:
- [ ] Bridge periodic scheduler chạy đúng interval
- [ ] `_raw_data` chứa đúng data từ source (không chứa Airbyte internal columns)
- [ ] Hash comparison hoạt động: row không đổi → _version không tăng
- [ ] Incremental bridge: chỉ đọc rows mới từ Airbyte (last_bridge_at)

---

## Luồng 3: Có update field mới từ table gốc

> Mục tiêu: Khi source thêm field mới → hệ thống detect → user duyệt → tạo column → backfill data.

### Mong muốn 3.1: Tự phát hiện fields mới

**Kỳ vọng**: Hệ thống tự detect field mới trong `_raw_data` mà chưa có mapping rule → tạo `cdc_mapping_rules` với status = "pending".

**Cơ chế hiện tại**:
- Periodic field scan (`HandlePeriodicScan`): quét `jsonb_object_keys(_raw_data)` mỗi 1h
- So sánh với existing mapping rules → tạo rules mới cho fields chưa có
- Status mặc định: "pending" (chờ duyệt)

**Cần verify**:
- [ ] Periodic scan chạy đúng
- [ ] Fields mới xuất hiện trong CMS UI (pending status)
- [ ] Type inference cho field mới (dựa trên giá trị trong _raw_data)

### Mong muốn 3.2: Click duyệt field → tạo column vào table đích

**Kỳ vọng**: User approve pending field → hệ thống tự ALTER TABLE ADD COLUMN.

**API hiện tại**: `PATCH /api/mapping-rules/:id` (status → approved) hoặc `PATCH /api/mapping-rules/batch`

**Cần verify**:
- [ ] Approve mapping rule → trigger ALTER TABLE ADD COLUMN trên CDC table
- [ ] Column type đúng với data_type trong mapping rule
- [ ] NATS `schema.config.reload` published → Worker reload cache

**Gap tiềm năng**:
- Approve mapping rule hiện tại có thể chỉ đổi status, KHÔNG tự động ALTER TABLE
- Cần verify: sau approve, column có được tạo không?
- Nếu chưa → cần flow: approve → auto ALTER TABLE → reload cache

### Mong muốn 3.3: Click bổ sung → backfill data từ _raw_data

**Kỳ vọng**: Sau khi column đã tạo, user click 1 button → hệ thống:
1. Quét tất cả rows có `_raw_data` chứa field mới
2. Extract giá trị → UPDATE vào column mới
3. Report: N rows backfilled

**API hiện tại**: `POST /api/mapping-rules/:id/backfill`

**Cần verify**:
- [ ] Backfill đọc từ `_raw_data->>'field_name'` → cast → UPDATE column
- [ ] Chỉ update rows chưa có giá trị (WHERE column IS NULL)
- [ ] Report số rows affected

**Gap tiềm năng**:
- Backfill hiện tại gọi `cdc.cmd.backfill` → Worker handler
- Cần verify Worker handler thực sự chạy UPDATE trên đúng table/column

---

## Checklist tổng hợp — Verify trước khi chạy

### Luồng 1 Pre-check (Code Review)
| # | Item | File | Check |
|:--|:-----|:-----|:------|
| 1.1 | SyncFromAirbyte quét cả non-active streams | `registry_handler.go:SyncFromAirbyte` | [ ] |
| 1.2 | Import populate đủ Airbyte metadata | `registry_handler.go:SyncFromAirbyte` | [ ] |
| 1.3 | PK auto-detect từ Airbyte catalog | `registry_handler.go:SyncFromAirbyte` | [ ] |
| 1.4 | Mapping rules status = approved (lần đầu) | `registry_handler.go` hoặc `airbyte_handler.go` | [ ] |
| 1.5 | Standardize tạo table v1.12 + add columns | `command_handler.go:HandleStandardize` | [ ] |

### Luồng 2 Pre-check
| # | Item | File | Check |
|:--|:-----|:-----|:------|
| 2.1 | Bridge periodic scheduler | `worker_server.go:177-214` | [ ] |
| 2.2 | _raw_data stripped Airbyte columns | `command_handler.go:HandleAirbyteBridge` | [ ] |
| 2.3 | Hash comparison (skip unchanged rows) | Bridge SQL WHERE clause | [ ] |
| 2.4 | Incremental via last_bridge_at | Bridge SQL WHERE clause | [ ] |

### Luồng 3 Pre-check
| # | Item | File | Check |
|:--|:-----|:-----|:------|
| 3.1 | Periodic field scan detect new fields | `command_handler.go:HandlePeriodicScan` | [ ] |
| 3.2 | Approve → auto ALTER TABLE | `approval_service.go` hoặc `mapping_rule_handler.go` | [ ] |
| 3.3 | Backfill handler works | `command_handler.go:HandleBackfill` | [ ] |
| 3.4 | schema.config.reload after approve | `approval_service.go` | [ ] |

---

---

## CODE REVIEW RESULTS (2026-04-13)

### Gap #1: SyncFromAirbyte chỉ import active streams
- **File**: `registry_handler.go:996-998`
- **Code**: `if !sc.Config.Selected { continue }` → skip non-active
- **Fix**: Bỏ check `Selected`, import tất cả streams. Đặt `is_active` theo `sc.Config.Selected`

### Gap #2: ExecuteImport không parse JSONSchema cho all fields
- **File**: `airbyte_handler.go:256-266`
- **Code**: Chỉ tạo 1 rule hardcoded `source_field: "id", data_type: "text"`
- **Fix**: Parse `sc.Stream.JSONSchema.Properties` → tạo mapping rule cho mỗi field, status="approved", type inference

### Gap #3: Không có flow "tạo default columns"
- **File**: `command_handler.go:HandleStandardize` chỉ gọi `standardize_cdc_table()`
- **Code**: Tạo bare table, không add columns từ mapping rules
- **Fix**: Tạo handler mới `HandleCreateDefaultColumns`: create table + ALTER TABLE ADD COLUMN cho tất cả approved rules

### Gap #4: Approve mapping rule không auto ALTER TABLE
- **File**: `mapping_rule_handler.go:UpdateStatus` chỉ update status + publish reload
- **Note**: `approval_service.go` CÓ ALTER TABLE nhưng flow đó dùng cho `pending_fields` (schema changes), KHÔNG phải `mapping_rules`
- **Fix cho Luồng 3**: Khi approve mapping rule → trigger ALTER TABLE ADD COLUMN + backfill option

---

## FIX PLAN

### Fix #1: SyncFromAirbyte import cả non-active
```go
// Bỏ: if !sc.Config.Selected { continue }
// Thay: is_active = sc.Config.Selected
entry.IsActive = sc.Config.Selected
```

### Fix #2: Parse JSONSchema → tạo all mapping rules (approved)
```go
// Sau khi tạo registry entry:
if sc.Stream.JSONSchema != nil {
    for fieldName, fieldSchema := range sc.Stream.JSONSchema.Properties {
        dataType := inferDataType(fieldSchema)
        rule := model.MappingRule{
            SourceTable:  sc.Stream.Name,
            SourceField:  fieldName,
            TargetColumn: fieldName,
            DataType:     dataType,
            IsActive:     true,
            Status:       "approved",
        }
        mappingRepo.CreateIfNotExists(ctx, &rule)
    }
}
```

### Fix #3: HandleCreateDefaultColumns (Worker)
```go
// 1. Create CDC table (create_cdc_table)
// 2. Get all approved mapping rules for this table
// 3. For each rule: ALTER TABLE ADD COLUMN IF NOT EXISTS
```
CMS endpoint: `POST /api/registry/:id/create-default-columns`

### Fix #4: Approve mapping rule → auto ALTER TABLE
```go
// Trong UpdateStatus hoặc BatchUpdate:
// Nếu status chuyển sang "approved":
//   1. ALTER TABLE ADD COLUMN IF NOT EXISTS
//   2. Optional: auto trigger backfill
```

---

## Execution Plan

### Step 1: Code Review ✅ DONE (2026-04-13)
### Step 2: Fix 4 Gaps ✅ DONE (2026-04-13)
### Step 2.5: Worker Activity Log (monitoring)
- Migration: `cdc_activity_log` table
- Worker: log mọi auto operation (bridge, transform, scan, reload, partition)
- CMS API: `GET /api/activity-log` (paginated, filterable)
- FE: Monitoring page hiển thị chi tiết
### Step 3: User xoá data + chạy Luồng 1
### Step 4: Chạy Luồng 2
### Step 5: Chạy Luồng 3
### Step 6: Sign-off
