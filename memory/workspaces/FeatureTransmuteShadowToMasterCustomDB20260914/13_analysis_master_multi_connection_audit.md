# 13_analysis_master_multi_connection_audit.md — Báo Cáo Audit Toàn Diện Kiến Trúc Master Multi-Connection

> **Thời điểm**: 2026-09-14 17:00:00 +07:00  
> **Người thực hiện**: Role Brain (System Architect)  
> **Bối cảnh**: Người dùng tạo một bảng Master mới trỏ sang kết nối `master_2` (cùng schema `master_centrallized_export_service` và table `export_jobs`), phát hiện:  
> 1. Lỗi hiển thị: Hàng mới tạo (đang `pending_review`, Inactive) bị hiển thị nhầm `Synced (509 / 509)`.  
> 2. Lỗi nghiệp vụ: Bấm Approve gọi `POST /api/v1/masters/master_centrallized_export_service.export_jobs/approve` trả về lỗi HTTP 409 `{"error":"ambiguous_master_name"}`.

---

## I. Phân Tích Gốc Rễ (Root Cause Analysis)

### 1. Bản chất cốt lõi của sự cố
Trong thiết kế cơ sở dữ liệu gốc (Migration `032_v2_master_binding.sql`), bảng `cdc_system.master_binding` có ràng buộc duy nhất:
```sql
UNIQUE (master_connection_id, master_schema, master_table)
```
Kèm khóa chính tự tăng `id BIGSERIAL PRIMARY KEY` và `binding_code VARCHAR(150) NOT NULL UNIQUE`.
Điều này khẳng định: **Hệ thống vốn dĩ cho phép 1 bảng Master tồn tại trên nhiều Database Connection khác nhau (1 Source/Shadow fan-out ra N Master Connection).**

Tuy nhiên, trong quá trình phát triển mã nguồn ở các tầng ứng dụng (CMS Backend, Worker Engine, Frontend UI), các lập trình viên trước đây đã **giả định ngầm** rằng chỉ có 1 Master Database (`default_master`) duy nhất, dẫn đến việc dùng chuỗi `master_name` hoặc `master_schema.master_table` làm định danh toàn cục thay vì sử dụng `master_binding_id` (`id`) hoặc bộ ba `(master_connection_id, master_schema, master_table)`.

---

### 2. Gốc rễ Lỗi 1: `Synced (509 / 509)` hiển thị nhầm trên hàng Master mới

#### A. Backend SQL Join rò rỉ dữ liệu (`cdc-cms-service/internal/infra/persistence/master/master_read_repo_gorm.go`)
Tại hàm `ListEnriched()`, câu query đọc danh sách Master nối với bảng `cdc_system.transmute_jobs` bằng `LEFT JOIN LATERAL` chỉ so sánh tên bảng dạng text thuần túy:
```sql
LEFT JOIN LATERAL (
    SELECT status, rows_affected, total_rows, trace_id, job_id, error_message
    FROM cdc_system.transmute_jobs tj
    WHERE tj.master_table = (COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table)
       OR (mb.master_schema IS NOT NULL AND mb.master_schema <> '' AND tj.master_table = (mb.master_schema || '.' || mb.master_table))
       OR (mb.physical_table_fqn IS NOT NULL AND tj.master_table = mb.physical_table_fqn)
       OR tj.master_table = mb.master_table
    ORDER BY tj.created_at DESC
    LIMIT 1
) tj ON true
```
**Hậu quả**:
- Khi hàng Master 1 (`default_master`) đã chạy Transmute xong 509 dòng, bảng `transmute_jobs` có bản ghi `status = 'COMPLETED'`, `rows_affected = 509`, `master_table = 'master_centrallized_export_service.export_jobs'`.
- Khi tạo hàng Master 2 (`master_2`), hàng này có cùng schema và table name. Câu `LEFT JOIN LATERAL` tìm thấy ngay bản ghi của Master 1 và gán toàn bộ `last_transmute_status = 'COMPLETED'`, `last_transmute_rows = 509` vào Master 2!
- Hơn nữa, câu query hoàn toàn **không kiểm tra** `mb.schema_status = 'approved'`, dẫn đến việc một bảng vừa mới tạo (`pending_review`, chưa có DDL vật lý trên `master_2`) cũng bị gán chỉ số sync đã hoàn tất.

#### B. Thiếu định danh `master_binding_id` trong bảng `cdc_system.transmute_jobs`
Bảng `cdc_system.transmute_jobs` chỉ có cột `master_table VARCHAR(128)`, hoàn toàn thiếu cột `master_binding_id`. Do đó, nếu 2 binding cùng trỏ vào một tên bảng ở 2 DB khác nhau, hệ thống không thể phân biệt job này là của DB nào nếu chỉ dựa vào bảng này.

#### C. Frontend render badge thiếu điều kiện duyệt (`cdc-cms-web/src/pages/MasterRegistry.tsx`)
Tại component `TransmuteStatusBadge`:
```tsx
const effectiveStatus = job?.status || initialStatus;
if (effectiveStatus === 'COMPLETED') {
  return <Badge status="success" text={`Synced (${rowsLabel})`} />
}
```
Component chỉ nhìn vào `status === 'COMPLETED'` mà không kiểm tra xem hàng đó đã `approved` hay chưa.

---

### 3. Gốc rễ Lỗi 2: `ambiguous_master_name` khi Approve

#### A. Frontend gọi API bằng Name thay vì ID (`cdc-cms-web/src/pages/MasterRegistry.tsx`)
Trên giao diện Master Registry, mỗi hàng tương ứng với 1 `MasterRow` đã có sẵn trường `record.id`. Tuy nhiên, khi bấm Approve/Reject, frontend lại kích hoạt mutation:
```tsx
ht.mutate({ name: `${actionTarget.row.master_schema}.${actionTarget.row.master_name}`, op: actionTarget.op, reason: ... })
```
Đường dẫn gửi đi: `POST /api/v1/masters/master_centrallized_export_service.export_jobs/approve`.

#### B. Backend tra cứu với `LIMIT 2` và ném lỗi (`cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go`)
Tại hàm `ApproveSchemaTx()`:
```go
if schema != "" {
    err = r.db.WithContext(ctx).Raw(
        `SELECT id, shadow_binding_id FROM `+tableName+`
          WHERE master_schema = ? AND master_table = ?
          ORDER BY updated_at DESC, id DESC
          LIMIT 2`,
        schema, table,
    ).Scan(&rows).Error
}
...
if len(rows) > 1 {
    return 0, 0, "", errors.New("ambiguous_master_name")
}
```
Vì trong DB có 2 bản ghi (Master 1 và Master 2) có cùng `master_schema` và `master_table`, `len(rows)` = 2 -> Ném lỗi `ambiguous_master_name` và Fiber handler trả về HTTP 409 Conflict.

---

## II. Bảng Kiểm Toán Toàn Diện Các Điểm Chạm (Full System Audit)

Khi một hệ thống cho phép tạo nhiều Master Connection, tất cả các tác vụ vòng đời và xử lý dữ liệu liên quan tới Master đều bị ảnh hưởng nếu không được quy hoạch lại:

| STT | Vị trí / File | Hàm / Endpoint | Cơ chế hiện tại (Rủi ro) | Hậu quả khi có 2 Connection cùng tên Table |
| :--- | :--- | :--- | :--- | :--- |
| **1** | `cdc-cms-service/.../master_repo_gorm.go:199` | `ApproveSchemaTx` | `SELECT ... WHERE master_schema = ? AND master_table = ? LIMIT 2` | **Chặn đứng Approve**: Báo lỗi `ambiguous_master_name`. |
| **2** | `cdc-cms-service/.../master_repo_gorm.go:330` | `RejectSchemaTx` | `SELECT ... WHERE master_schema = ? AND master_table = ? LIMIT 2` | **Chặn đứng Reject**: Báo lỗi `ambiguous_master_name`. |
| **3** | `cdc-cms-service/.../master_repo_gorm.go:405` | `RevertSchemaTx` | `SELECT ... WHERE master_schema = ? AND master_table = ? LIMIT 2` | Saga Revert thất bại khi approve gặp sự cố. |
| **4** | `cdc-cms-service/.../master_repo_gorm.go:861` | `ResolveMasterBindingByName` | `SELECT ... WHERE master_schema = ? AND master_table = ? LIMIT 2` | Hàm này phục vụ `UpdateSpec` và `ToggleActive` -> **Gãy tính năng Bật/Tắt và Sửa Spec**. |
| **5** | `cdc-cms-service/.../master_repo_gorm.go:96` | `GetByName` | `SELECT ... WHERE master_schema = ? AND master_table = ? ORDER BY id DESC LIMIT 1` | Âm thầm lấy bản ghi mới nhất, bỏ qua bản ghi cũ. |
| **6** | `cdc-cms-service/.../master_read_repo_gorm.go:27` | `ListEnriched` | `LEFT JOIN LATERAL ... tj.master_table = ... LIMIT 1` | **Lỗi hiển thị chéo**: Lấy nhầm trạng thái Sync của connection khác gán sang hàng mới. |
| **7** | `cdc-cms-service/.../transmute_schedule_repository_gorm.go:62` | `Save` (Transmute Schedule) | `SELECT id FROM master_binding WHERE master_table = ? ORDER BY id DESC LIMIT 1` | **Gán nhầm lịch Sync**: Gán schedule của table này sang binding ID khác. |
| **8** | `cdc-cms-service/.../run_now.go:81` | `RunNowTransmuteScheduleHandler` | `TransmuteRunCommand` chỉ gửi `masterTableFQN`, không gửi `master_binding_id` | Worker không biết bảng cần sync nằm trên DB đích nào. |
| **9** | `cdc-cms-service/.../approve_master.go:118` | `ApproveMasterHandler` | Bắn NATS `cdc.cmd.master-create` chỉ có `master_table: physicalTableFQN` | Worker không biết bảng DDL cần tạo nằm trên DB đích nào. |
| **10** | `centralized-data-service/.../master_ddl_generator.go:464` | `loadBinding` | `SELECT ... WHERE mb.master_table = ? LIMIT 1` | **Sai lệch DDL**: DDL của `master_2` sẽ bị tạo nhầm trên `default_master` (Master 1)! |
| **11** | `centralized-data-service/.../transmuter.go:522` | `loadMaster` | `SELECT ... WHERE mb.master_table = ? LIMIT 1` | **Sai lệch Dữ liệu**: Dữ liệu transmute của `master_2` sẽ bị ghi đè nhầm vào `default_master`! |
| **12** | `centralized-data-service/.../master_binding_repo.go:86` | `ListMasterTablesByShadowIdentity` | `SELECT ... master_fqn` trả về chuỗi text thuần túy | Realtime CDC fanout không phân biệt được 2 connection cùng schema.table. |
| **13** | `centralized-data-service/.../transmute_handler.go:136` | `HandleTransmuteShadow` | Lặp qua `masterTables` bắn `cdc.cmd.transmute` bằng text table | Cả 2 message đều gọi vào `loadMaster` và bị `LIMIT 1` trỏ về cùng 1 DB. |
| **14** | `cdc-cms-web/.../MasterRegistry.tsx:430` | `TransmuteStatusBadge` | Render `Synced` chỉ dựa trên `effectiveStatus === 'COMPLETED'` | Render sai khi bảng chưa được duyệt. |
| **15** | `cdc-cms-web/.../MasterRegistry.tsx:320` | `handleApproveReject / toggle / spec` | Gọi API với URL chuỗi `name` thay vì truyền `id` / `binding_id` | Kích hoạt lỗi 409 `ambiguous_master_name`. |

---

## III. Giải Pháp Kiến Trúc Chuẩn Tắc (Unified Master Resolution)

Thay vì để các hàm query rải rác mỗi nơi tự viết câu `SELECT ... WHERE master_table = ? LIMIT 1` hoặc `LIMIT 2`, toàn bộ hệ thống phải quy về **Một Bộ Tiêu Chuẩn Định Danh Chuẩn Tắc (Canonical Identity)**:

### 1. Chuẩn hoá Định danh Master (Master Identity Hierarchy)
Mọi thao tác với Master Binding phải ưu tiên nhận dạng theo thứ tự nghiêm ngặt:
1. **Ưu tiên 1 (Tuyệt đối)**: `master_binding_id` (`id` int64).
2. **Ưu tiên 2 (Business Unique Key)**: `(master_connection_id, master_schema, master_table)`.
3. **Ưu tiên 3 (Compatibility Fallback)**: `(master_schema, master_table)`. Nếu tìm thấy duy nhất 1 bản ghi -> Chấp nhận. Nếu tìm thấy > 1 bản ghi và không có `binding_id` -> Báo lỗi yêu cầu chỉ định `binding_id`.

### 2. CMS Backend (`cdc-cms-service`)
- **Tạo hàm Resolver Duy Nhất**: `ResolveMasterBinding(ctx, id int64, name string, connectionID int64) (*master.Binding, error)`.
- **Hỗ trợ `binding_id` trên toàn bộ Master Endpoints**:
  * Cho phép query param `?binding_id=123` hoặc param `:name` nhận luôn số nguyên ID.
  * `Approve`, `Reject`, `ToggleActive`, `UpdateSpec`, `Swap`: Nếu có `binding_id` (từ query hoặc body), query trực tiếp theo `id = ?`, triệt tiêu hoàn toàn lỗi `ambiguous_master_name`.
- **NATS `cdc.cmd.master-create`**: Bổ sung trường `"master_binding_id": masterBindingID` vào payload gửi sang CDS worker.
- **Transmute Schedule `Save`**: Nhận `master_binding_id` từ client, insert chính xác cho binding ID đó thay vì query mò theo tên bảng.
- **NATS `cdc.cmd.transmute`**: Bổ sung `"master_binding_id": schedule.MasterBindingID` vào payload gửi sang CDS worker.
- **Sửa `ListEnriched`**:
  * Điều kiện join `transmute_jobs`: Ưu tiên `tj.master_binding_id = mb.id`.
  * Điều kiện an toàn: Bảng chỉ hiển thị transmute status nếu `mb.schema_status = 'approved'`. Nếu `mb.schema_status != 'approved'`, các trường `last_transmute_*` luôn trả về `NULL`.

### 3. Worker Engine (`centralized-data-service`)
- **`MasterDDLGenerator.loadBinding`**:
  * Nhận `master_binding_id` (nếu có) hoặc parse `masterName` (nếu là số nguyên) -> Query `WHERE mb.id = ?`.
  * Đọc chính xác `master_connection_key` (`master_2`), gọi `connMgr.GetMasterDB(ctx, "master_2")` -> Tạo DDL đúng trên target DB `master_2`.
- **`TransmuterModule.loadMaster`**:
  * Nhận `master_binding_id` từ `TransmuteRequest` hoặc parse `masterName` -> Query `WHERE mb.id = ?`.
  * Đọc chính xác `master_connection_key` (`master_2`) -> Transmute ghi đúng vào target DB `master_2`.
- **`cdc_system.transmute_jobs`**: Bổ sung cột `master_binding_id BIGINT` để lưu lại chính xác binding ID khi job chạy.
- **`ListMasterTablesByShadowIdentity` & `HandleTransmuteShadow`**: Trả về danh sách object `{ ID: mb.id, MasterTable: mb.master_table, MasterFQN: ... }` để realtime CDC fan-out bắn NATS mang theo `master_binding_id` riêng biệt cho từng target connection.

### 4. CMS Web Frontend (`cdc-cms-web`)
- **`MasterRegistry.tsx`**:
  * Sửa các hàm gọi API: Truyền kèm `?binding_id=${record.id}` (hoặc truyền ID vào URL) cho các action: Approve, Reject, Toggle Active, Edit Spec.
  * Sửa `handleRunSync` (`action: 'run_now'`): Truyền `master_binding_id: row.id` khi tạo schedule và kích hoạt sync.
  * Sửa `TransmuteStatusBadge`: Nếu `record.schema_status !== 'approved'`, hiển thị Tag "Chưa sync" (hoặc "Chờ duyệt schema"), không bao giờ render "Synced" khi chưa approved.
