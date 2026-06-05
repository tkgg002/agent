# 02_plan.md — Kế Hoạch Triển Khai: Masters Page Toggle-Active, UI Schema/Mapping & Schedules API Fix

> **Workspace**: `agent/memory/workspaces/feature-masters-page-audit-2026-06-02/`
> **Ngày**: 2026-06-03

---

## PHẦN 1: Masters Page — Toggle-Active Fix & UI Enrichment (Đã Hoàn Thành)

### 1. Bối cảnh & Root Cause
- **Bug 1**: `POST /api/v1/masters/exj/toggle-active` trả về `500 internal_error`.
  * **Root Cause**: Trong `master_registry_handler_resolve.go`, query lookup master trỏ vào bảng legacy `cdc_system.master_table_registry` đã deprecated.
  * **Giải pháp**: Sửa câu SELECT trỏ vào bảng đúng `cdc_system.master_binding`.
- **Gap 2**: UI `/masters` không hiển thị schema của master DB, field mapping, flatten mode.
  * **Root Cause**: API `GET /api/v1/masters` thiếu các trường `ddl_status`, `mapping_count` và `is_flatten`.
  * **Giải pháp**: Enrich query trong `master_read_repo_gorm.go` để lấy đầy đủ metadata và cập nhật UI.

### 2. Các Thay Đổi Đã Thực Hiện
- Sửa `master_registry_handler_resolve.go` trỏ sang `cdc_system.master_binding`.
- Cập nhật `list_masters.go` (queries) và `master_read_repo_gorm.go` để enrich data (DDL status, mapping count, flatten flag).
- Cập nhật UI `MasterRegistry.tsx`: Thêm các cột DDL Status, Mapping Rules Count, Flatten Badge và hiển thị `MasterMappingsPanel` trong expandable row.
- Cấu hình NATS Gate post_ingest trong `sinkworker.go` để tối ưu hóa việc publish trigger.

---

## PHẦN 2: Bổ Sung Kế Hoạch — Sửa Lỗi API Schedules & Tách Biệt Master Mapping Rules (Triển Khai Mới)

### 1. Phân Tích Lỗi & Nguyên Nhân Gốc Rễ

#### Bug A: API `POST /api/v1/schedules` bị `500 Internal Server Error`
- **Root Cause**: Bảng `cdc_system.transmute_schedule` trong DB không còn cột `master_table` (đã chuẩn hóa sang `master_binding_id` trỏ sang `master_binding(id)`). Nhưng `CreateTransmuteScheduleHandler` trong `cdc-cms-service` vẫn cố gắng INSERT trực tiếp cột `master_table` vào DB, gây ra lỗi SQL.
- **Giải pháp**: Resolve `master_binding_id` từ `cmd.MasterTable` trước, sau đó thực hiện UPSERT theo `master_binding_id` và `mode`.

#### Bug B: Tách biệt Mapping Rules của Master bằng bảng riêng `mapping_rule_master`
Để hỗ trợ transform type `flatten` (nơi một shadow column map sang nhiều master columns qua JSON path lồng nhau), đồng thời bảo vệ Master table khỏi việc bị bê nguyên các system columns của Shadow, ta sẽ tạo một bảng riêng biệt hoàn toàn.
- **Giải pháp**:
  1. **Tạo bảng `cdc_system.mapping_rule_master`**:
     * Lưu trữ mapping từ Shadow columns -> Master columns.
     * Có trường `source_path` để lưu JSON path (ví dụ: `items[*].item_id`) cho flatten mode.
     * Có các trường bảo mật: `is_sensitive`, `mask_strategy`.
  2. **Backend REST API cho Master Rules**:
     * `GET /api/v1/master-mapping-rules?master_binding_id=X` (List rules)
     * `POST /api/v1/master-mapping-rules` (Create/Update rule)
     * `PATCH /api/v1/master-mapping-rules/:id` (Update fields/status)
     * `PATCH /api/v1/master-mapping-rules/batch` (Batch approve/reject)
     * `DELETE /api/v1/master-mapping-rules/:id` (Delete rule)
     * `POST /api/v1/master-mapping-rules/flatten` (Auto-flatten JSON fields)
  3. **Auto-clone default rules khi tạo Master**:
     * Khi tạo Master Binding (`CreateMasterHandler`), tự động clone default rules từ `mapping_rule_v2` của shadow sang `mapping_rule_master` của master.
     * Sử dụng blacklist để loại bỏ hoàn toàn các cột hệ thống: `_gpay_id`, `_raw_data`, `_synced_at`, `_deleted`, `_change_type`, `_source_timestamp`, `_processed_at`, `_id`, `progress`, `status`, `params`, `error`, `_source_ts`, `_source`, `_hash`, `_version`.
  4. **Cập nhật Worker (`transmuter.go`)**:
     * Đổi hàm `loadRules` sang query từ bảng `cdc_system.mapping_rule_master` dựa theo `master_binding_id = ?` (không có fallback `master_binding_id IS NULL`).

#### Bug C: Nâng cấp UI MasterMappingsPanel trên Master Registry
- **Root Cause**: UI expandable row hiện tại chỉ hiển thị list mappings read-only đơn giản, thiếu các cột quan trọng và các thao tác quản lý.
- **Giải pháp**: Nâng cấp `MasterMappingsPanel` thành một trình quản lý mapping đầy đủ (tương tự như `MappingFieldsPage` của Shadow) với các cột:
  1. **Source Field** (Cột shadow nguồn)
  2. **Target Column** (Cột master đích)
  3. **Source Data Type** (Kiểu dữ liệu shadow)
  4. **Data Type Target** (Kiểu dữ liệu master)
  5. **Rule Type** (Cách thức map: raw/jsonpath/expression)
  6. **Status** (Trạng thái duyệt: pending/approved/rejected)
  7. **Active** (Switch toggle active)
  8. **Sensitive** (Switch toggle sensitive)
  9. **Mask Strategy** (Chọn mask strategy)
  10. **Actions** (Approve, Reject, Edit, Delete)
- **Hành động hàng loạt (Batch Actions)** phía trên bảng:
  * Nút **Add Rule**: Thêm mapping rule thủ công.
  * Nút **Batch Approve / Batch Reject**: Duyệt/Từ chối hàng loạt các rules được check chọn.
  * Nút **Sync default rules**: Đồng bộ lại các default rules từ shadow.

---

## PHẦN 3: Thiết Kế Logic Flatten Tự Động Cho JSON/JSONB

### 1. Phía UI (Frontend)
- Cạnh các mapping rule có kiểu dữ liệu nguồn (Source Data Type) là `JSON`, `JSONB`, `RECORD` hoặc `ARRAY` trong bảng `MasterMappingsPanel`:
  - Hiển thị thêm nút hành động **Flatten**.
- Khi nhấn nút **Flatten**:
  - Hiển thị Modal nhập `Explode Path` (gợi ý mặc định là `[source_field]` hoặc `[source_field][*]`).
  - Bấm "Confirm" -> Gọi API `POST /api/v1/master-mapping-rules/flatten`.
  - API thành công -> Refresh danh sách rules để hiển thị các mapping mới được phân tách ở trạng thái `pending`.

### 2. Phía Backend API & Database
- Thêm API `POST /api/v1/master-mapping-rules/flatten`:
  - Payload: `{ "master_binding_id": X, "source_field": "items", "explode_path": "items[*]" }`
  - Các bước xử lý trong Handler:
    1. Lấy thông tin `master_binding` bằng `master_binding_id`. Lấy thông tin `shadow_schema` và `shadow_table` của shadow tương ứng.
    2. Query dữ liệu mẫu (sample data) từ shadow table:
       `SELECT [source_field] FROM [shadow_schema].[shadow_table] WHERE [source_field] IS NOT NULL LIMIT 5`
    3. Phân tích (parse) mẫu JSON thu được:
       - Nếu `explode_path` dạng array (ví dụ `items[*]`), duyệt qua các item bên trong array của tất cả bản ghi mẫu.
       - Trích xuất các thuộc tính (keys) lồng nhau và kiểu dữ liệu thực tế của chúng (string, number, boolean...).
    4. Với mỗi thuộc tính lồng nhau tìm thấy (ví dụ: `item_id`, `price`):
       - Kiểm tra tính trùng lặp: Nếu đã tồn tại rule trong `mapping_rule_master` với cùng `master_binding_id = X` và `target_column = [key]` -> Bỏ qua.
       - Ngược lại, tự động tạo mới một mapping rule với:
         * `master_binding_id = X`
         * `source_field = [source_field]`
         * `source_path = [explode_path].[key]` (ví dụ: `items[*].item_id`)
         * `target_column = [key]`
         * `data_type = [PG type tương ứng]` (ví dụ: VARCHAR, NUMERIC, BOOLEAN...)
         * `source_data_type = [kiểu trích xuất]`
         * `source_format = 'jsonpath'`
         * `status = 'pending'` (chờ duyệt)
         * `is_active = true`
    5. Trả về danh sách các key mới được sinh ra và số lượng mapping rules được tạo mới thành công.

---

## 4. Chi Tiết Các Thay Đổi Mã Nguồn Bổ Sung

### A. Database Migrations
#### [NEW] `migrations/schema/cdc_system_model/074_v2_mapping_rule_master.sql`
Tạo bảng `cdc_system.mapping_rule_master` và các indexes:
```sql
CREATE TABLE IF NOT EXISTS cdc_system.mapping_rule_master (
  id                BIGSERIAL PRIMARY KEY,
  master_binding_id BIGINT NOT NULL REFERENCES cdc_system.master_binding(id) ON DELETE CASCADE,
  source_field      VARCHAR(255) NOT NULL,
  source_path       VARCHAR(500),
  target_column     VARCHAR(255) NOT NULL,
  data_type         VARCHAR(100) NOT NULL,
  source_data_type  VARCHAR(100),
  source_format     VARCHAR(32) NOT NULL DEFAULT 'raw' CHECK (source_format IN ('raw','jsonpath','expression')),
  transform_fn      VARCHAR(100),
  is_nullable       BOOLEAN NOT NULL DEFAULT TRUE,
  default_value     TEXT,
  is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  status            VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
  is_sensitive      BOOLEAN NOT NULL DEFAULT FALSE,
  mask_strategy     VARCHAR(100),
  notes             TEXT,
  created_by        VARCHAR(100),
  updated_by        VARCHAR(100),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_mapping_rule_master_identity
  ON cdc_system.mapping_rule_master (master_binding_id, target_column);

CREATE INDEX IF NOT EXISTS idx_mapping_rule_master_binding
  ON cdc_system.mapping_rule_master(master_binding_id);
```

### B. Backend CMS Service (`cdc-cms-service`)
#### 1. `internal/app/commands/create_schedule.go`
- Sửa hàm `Handle` của `CreateTransmuteScheduleHandler`:
  - Lấy `master_binding_id` dựa trên `cmd.MasterTable`.
  - Thực hiện UPSERT sử dụng `master_binding_id` thay cho `master_table`.

#### 2. `internal/app/commands/create_master.go`
- Cập nhật hàm `Handle` của `CreateMasterHandler`:
  - Thêm `RETURNING id` vào câu lệnh INSERT `master_binding` để lấy `masterBindingID`.
  - Truy vấn active shadow rules từ `mapping_rule_v2`.
  - Clone các business rules (bỏ qua blacklist system columns) sang bảng `cdc_system.mapping_rule_master`.

#### 3. Domain Model & Repository cho Master Rules
- Tạo `internal/domain/mapping/master_rule.go`: Định nghĩa struct `MasterMappingRule` và interface `MasterMappingRuleRepo`.
- Tạo `internal/infra/persistence/master_mapping_rule_repo_gorm.go`: Triển khai các query CRUD, list, batch update cho bảng `mapping_rule_master`.

#### 4. API Handlers cho Master Rules
- Tạo `internal/api/master_mapping_rule_handler.go`: Expose các REST API endpoints cho master mapping rules, bao gồm handler `/api/v1/master-mapping-rules/flatten`.
- Đăng ký routes mới trong `internal/api/router.go`.

### C. Frontend Web (`cdc-cms-web`)
#### 5. `src/pages/MasterRegistry.tsx` (Component `MasterMappingsPanel`)
- Chuyển đổi panel từ read-only sang full management (table 10 cột, CRUD mutations, Modal Add/Edit mapping rule, Batch actions).
- Thêm button **Flatten** cho các JSON columns và tích hợp Modal nhập Explode Path để kích hoạt scan-flatten.

### D. Worker Service (`centralized-data-service`)
#### 6. `internal/service/transmuter.go`
- Cập nhật hàm `loadRules`: Đổi query sang bảng `cdc_system.mapping_rule_master` và chỉ lọc theo `master_binding_id = ?`.

---

## 5. Kế Hoạch Xác Minh (Verification Plan)

### Build Verification
```bash
# 1. Build check CMS Service
cd cdc-cms-service && go build ./...

# 2. Build check Worker Service
cd centralized-data-service && go build ./...

# 3. Build check Frontend
cd cdc-cms-web && npm run build
```

### Manual Verification
1. **Thiết lập sync schedule**:
   - Mở Sync Modal trên UI `/masters` -> Bấm OK.
   - Xác nhận API trả về `201 Created` thành công, không bị 500.
2. **Tạo Master Registry mới**:
   - Thực hiện tạo một Master Registry mới.
   - Mở rộng detail row trên `/masters` -> Xác nhận danh sách mapping rules được sinh mặc định và không chứa bất kỳ system columns nào.
3. **Quản lý Mapping trên Master**:
   - Thử thêm một mapping rule mới (Add Rule).
   - Thử sửa kiểu dữ liệu (Data Type Target), toggle Active/Sensitive, chọn Mask Strategy.
   - Thử Approve/Reject một rule và Batch Approve nhiều rules.
   - Xác nhận mọi thay đổi được cập nhật chính xác vào bảng `mapping_rule_master` trong PostgreSQL.
4. **Kiểm tra Auto-Flatten JSON**:
   - Chọn cột JSON (ví dụ `metadata` hoặc `items`).
   - Nhấn **Flatten** -> Nhập path `items[*]` -> Nhấn Confirm.
   - Xác nhận danh sách mapping rules xuất hiện các record mới dạng `items[*].id`, `items[*].price` ở trạng thái `pending`.
   - Tiến hành Approve các rules này -> Xác nhận DDL được chạy tạo cột tương ứng trên Master Table.
5. **Đồng bộ dữ liệu**:
   - Chạy trigger sync -> check worker log để đảm bảo transmuter load đúng rules từ `mapping_rule_master` và sync thành công sang Master DB.
