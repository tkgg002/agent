# 09 Technical Solution — Giải quyết triệt để lỗi Unique Constraint `(operation, target_table)` khi tạo lịch trình cho `Shadow` và `Master` cùng tên bảng (`payment_bills`)

> **Trạng thái:** 🟢 **APPROVED & EXECUTED** (Đã được User `APPROVE` và thực thi hoàn tất 100% qua Gate G1–G8).

---

## 1. Phân tích Gốc rễ (Root Cause Analysis — Đối soát 100% từ Code & Schema)

Khi anh tạo 1 lịch trình cho tầng `shadow` và 1 lịch trình cho tầng `master` cùng chọn bảng `payment_bills` trên `/activity-manager`, hệ thống báo lỗi:
> **`Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)`**

Nguyên nhân đến từ **3 điểm làm việc hời hợt nối tiếp nhau** giữa Database Migration, Frontend và Backend:

### 1.1. Tầng Database Migration: Bỏ quên ràng buộc phẳng `UNIQUE(operation, target_table)` từ schema cũ
- Tại file [007_worker_schedule.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/007_worker_schedule.sql#L21-L35), bảng `cdc_worker_schedule` ban đầu chỉ phục vụ các job đơn tầng (`bridge`, `transform`, `field-scan`, `partition-check`) nên đặt khóa duy nhất phẳng 2 cột:
  ```sql
  UNIQUE(operation, target_table)
  -- PostgreSQL sinh constraint: cdc_worker_schedule_operation_target_table_key
  ```
- Đến khi nâng cấp bảng lập lịch tại [024_enhance_worker_schedule.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/024_enhance_worker_schedule.sql#L10-L12) để hỗ trợ `source_count` đa tầng (`layer_type`: `source` / `shadow` / `master`), đa kết nối (`connection_code`) và đa schema (`container_namespace`) lưu trong cột `params JSONB`:
  - Trong khi bảng kết quả [025_create_daily_source_counts.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/025_create_daily_source_counts.sql#L28) được định nghĩa khóa duy nhất 5 thành phần:
    `UNIQUE (layer_type, connection_code, container_namespace, object_name, report_date)`
  - Thì trên bảng lập lịch `cdc_system.cdc_worker_schedule`, chỉ `ADD COLUMN params JSONB` mà **không gỡ bỏ `UNIQUE(operation, target_table)`**!

### 1.2. Tầng Frontend (`ActivityManager.tsx`) & Backend (`create_worker_schedule.go`): Gán tên bảng trần trụi vào `target_table`
- Tại [ActivityManager.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L1185-L1196) (`buildSchedulePayload`) và dòng 1325 (`handleSaveEdit`):
  ```ts
  if (values.operation === 'source_count') {
    payload.target_table = values.object_name || null; // <-- Gán trần trụi "payment_bills" cho cả source, shadow, master!
  ```
- Tại [create_worker_schedule.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/scheduler/create_worker_schedule.go#L62-L74), Backend nhận nguyên `target_table = "payment_bills"` và `INSERT` vào `cdc_system.cdc_worker_schedule`.
- Khi tạo dòng thứ 2 có cùng `operation = 'source_count'` và `target_table = 'payment_bills'` (dù `params.layer_type = 'master'` khác hẳn `params.layer_type = 'shadow'`), PostgreSQL lập tức ném lỗi:
  ```text
  ERROR: duplicate key value violates unique constraint "cdc_worker_schedule_operation_target_table_key" (SQLSTATE 23505)
  ```
- Tại [apiError.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/apiError.ts#L48-L50), mã `SQLSTATE 23505` không tìm thấy tên constraint trong `CONSTRAINT_HUMAN` nên rơi vào thông báo mặc định:
  `'Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất).'`

---

## 2. Phương Án Giải Quyết Tối Ưu Duy Nhất (The Single Best Approach)

Giải quyết đồng bộ từ gốc CSDL lên Backend, Worker và Frontend để **100% nhất quán với mô hình đa tầng 1-N-M (`layer_type` + `connection_code` + `container_namespace` + `object_name`)**:

### 2.1. Tầng Database Migration (`cdc-cms-service/migrations`)
1. Tạo migration mới `cdc-cms-service/migrations/schema/worker/026_fix_worker_schedule_unique_scope.sql`:
   - Gỡ bỏ ràng buộc phẳng `cdc_worker_schedule_operation_target_table_key`:
     ```sql
     ALTER TABLE cdc_system.cdc_worker_schedule
         DROP CONSTRAINT IF EXISTS cdc_worker_schedule_operation_target_table_key;
     ```
   - Tạo **Partial Functional Unique Index** chuẩn xác 5 thành phần cho `source_count` (khớp 1-1 với khóa của `daily_source_counts`):
     ```sql
     CREATE UNIQUE INDEX IF NOT EXISTS uq_worker_schedule_source_count_scope
     ON cdc_system.cdc_worker_schedule (
         operation,
         COALESCE(params->>'layer_type', 'source'),
         COALESCE(params->>'connection_code', ''),
         COALESCE(params->>'container_namespace', ''),
         COALESCE(params->>'object_name', COALESCE(target_table, ''))
     )
     WHERE operation = 'source_count';
     ```
   - Tạo **Partial Unique Index** cho các tác vụ vận hành đơn tầng truyền thống (`bridge`, `transform`, `reconcile`, `field-scan`, `partition-check`) để giữ nguyên tính toàn vẹn:
     ```sql
     CREATE UNIQUE INDEX IF NOT EXISTS uq_worker_schedule_legacy_op_target
     ON cdc_system.cdc_worker_schedule (operation, target_table)
     WHERE operation NOT IN ('source_count', 'confluence');
     ```
   - Chuẩn hóa lại cột `target_table` của các bản ghi `source_count` đang có trong DB thành định danh đầy đủ `<layer_type>:<connection_code>:<container_namespace>.<object_name>` để dễ tra cứu và log:
     ```sql
     UPDATE cdc_system.cdc_worker_schedule
     SET target_table = CONCAT(
         COALESCE(NULLIF(params->>'layer_type', ''), 'source'), ':',
         COALESCE(NULLIF(params->>'connection_code', ''), 'default'), ':',
         COALESCE(NULLIF(params->>'container_namespace', ''), 'public'), '.',
         COALESCE(NULLIF(params->>'object_name', ''), COALESCE(target_table, ''))
     )
     WHERE operation = 'source_count'
       AND COALESCE(params->>'object_name', '') <> '';
     ```
2. Cập nhật `cdc-cms-service/migrations/seed/worker_schedules.sql`:
   - Đổi `ON CONFLICT (operation, target_table) DO NOTHING` thành `ON CONFLICT (operation, target_table) WHERE operation NOT IN ('source_count', 'confluence') DO NOTHING` để tương thích với Partial Unique Index khi khởi tạo DB mới.

### 2.2. Tầng Backend (`cdc-cms-service`)
1. Tại `cdc-cms-service/internal/app/commands/scheduler/create_worker_schedule.go` và `update_schedule.go`:
   - Bổ sung hàm `CanonicalScheduleTargetTable(operation string, rawTarget *string, params json.RawMessage) *string`:
     - Với `operation == "source_count"`: Tự động trích xuất `layer_type`, `connection_code`, `container_namespace`, `object_name` từ `params` để chuẩn hóa `target_table = "<layer_type>:<connection_code>:<container_namespace>.<object_name>"` (ví dụ: `shadow:default_shadow:shadow_payment.payment_bills` và `master:default_master:master_payment.payment_bills`).
   - Nhờ vậy, kể cả trong `cdc_worker_schedule` lẫn `cdc_activity_log` (`schedule-create`, `schedule-update`), mỗi lịch trình đều mang định danh trạm rõ ràng, không bao giờ nhập nhằng giữa Shadow và Master.
2. Tại `cdc-cms-service/internal/api/scheduler/schedule_handler.go`:
   - Nếu `Create` hoặc `Update` gặp lỗi PostgreSQL `23505` (khi người dùng thực sự tạo trùng cả 4 thông số `layer_type + connection_code + container_namespace + object_name`), trả về HTTP `409 Conflict` kèm thông điệp lỗi gốc để Frontend `humanizeApiError` hiển thị rõ ràng.

### 2.3. Tầng Worker (`centralized-data-service`)
- Tại `centralized-data-service/internal/service/sourcecount/source_count_runner.go`:
  - Giữ `params.ObjectName` là tên bảng thuần (`payment_bills`) để câu lệnh `countSQL` (`SELECT COUNT(*) FROM "<container_namespace>"."<object_name>"`) và bảng `daily_source_counts.object_name` hoạt động chính xác 100%.
  - Nếu `params.ObjectName == ""` (fallback từ `sched.TargetTable`), tách phần sau dấu `.` cuối cùng để lấy tên bảng thuần.
  - Chuẩn hóa `targetName` ghi vào `cdc_activity_log` dạng `[<layer_type>:<connection_code>] <container_namespace>.<object_name>` để nhật ký thực thi phân biệt rõ đếm ở Shadow hay Master.

### 2.4. Tầng Frontend (`cdc-cms-web`)
1. Tại `cdc-cms-web/src/pages/ActivityManager.tsx`:
   - Trong `buildSchedulePayload` và `handleSaveEdit` (nhánh `source_count`):
     Gán `payload.target_table = `${values.layer_type || 'source'}:${values.connection_code || 'default'}:${values.container_namespace || 'public'}.${values.object_name}`` (trong khi `payload.params.object_name` giữ nguyên `values.object_name`).
   - Tại cột **"Phạm vi / Đối tượng"** (dòng 1406–1417): Hiển thị đầy đủ badge `[connection_code]` đứng trước `{container_namespace}.{object_name}` để nhìn trên bảng biết ngay dòng nào của Shadow, dòng nào của Master và thuộc Connection nào.
2. Tại `cdc-cms-web/src/utils/apiError.ts`:
   - Bổ sung mapping cho `uq_worker_schedule_source_count_scope` và `cdc_worker_schedule_operation_target_table_key` trong `CONSTRAINT_HUMAN`:
     `'Lịch trình đếm (Source Count) cho đúng Trạm (Layer) + Kết nối + Schema + Bảng này đã tồn tại.'`

---

## 3. Code Demo Chi Tiết Cho Từng File

### 3.1. File Migration: `cdc-cms-service/migrations/schema/worker/026_fix_worker_schedule_unique_scope.sql`
```sql
BEGIN;

-- 1. Gỡ bỏ ràng buộc phẳng UNIQUE(operation, target_table) từ 007_worker_schedule.sql
ALTER TABLE cdc_system.cdc_worker_schedule
    DROP CONSTRAINT IF EXISTS cdc_worker_schedule_operation_target_table_key;

DROP INDEX IF EXISTS cdc_system.cdc_worker_schedule_operation_target_table_key;

-- 2. Chuẩn hóa target_table cho các dòng source_count hiện có
UPDATE cdc_system.cdc_worker_schedule
SET target_table = CONCAT(
    COALESCE(NULLIF(params->>'layer_type', ''), 'source'), ':',
    COALESCE(NULLIF(params->>'connection_code', ''), 'default'), ':',
    COALESCE(NULLIF(params->>'container_namespace', ''), 'public'), '.',
    COALESCE(NULLIF(params->>'object_name', ''), COALESCE(target_table, ''))
)
WHERE operation = 'source_count'
  AND COALESCE(params->>'object_name', '') <> '';

-- 3. Tạo Unique Index 5 thành phần cho source_count: (operation, layer_type, connection_code, container_namespace, object_name)
CREATE UNIQUE INDEX IF NOT EXISTS uq_worker_schedule_source_count_scope
ON cdc_system.cdc_worker_schedule (
    operation,
    COALESCE(params->>'layer_type', 'source'),
    COALESCE(params->>'connection_code', ''),
    COALESCE(params->>'container_namespace', ''),
    COALESCE(params->>'object_name', COALESCE(target_table, ''))
)
WHERE operation = 'source_count';

-- 4. Giữ Unique Index (operation, target_table) cho các tác vụ đơn tầng truyền thống
CREATE UNIQUE INDEX IF NOT EXISTS uq_worker_schedule_legacy_op_target
ON cdc_system.cdc_worker_schedule (operation, target_table)
WHERE operation NOT IN ('source_count', 'confluence');

COMMIT;
```

### 3.2. File Backend Command: `cdc-cms-service/internal/app/commands/scheduler/create_worker_schedule.go`
```go
// CanonicalScheduleTargetTable chuẩn hóa target_table theo đầy đủ tọa độ trạm đối với source_count
func CanonicalScheduleTargetTable(operation string, rawTarget *string, params json.RawMessage) *string {
	if operation == "source_count" && len(params) > 0 {
		var p struct {
			LayerType          string `json:"layer_type"`
			ConnectionCode     string `json:"connection_code"`
			ContainerNamespace string `json:"container_namespace"`
			ObjectName         string `json:"object_name"`
		}
		if err := json.Unmarshal(params, &p); err == nil && strings.TrimSpace(p.ObjectName) != "" {
			layer := strings.TrimSpace(p.LayerType)
			if layer == "" {
				layer = "source"
			}
			conn := strings.TrimSpace(p.ConnectionCode)
			if conn == "" {
				conn = "default"
			}
			ns := strings.TrimSpace(p.ContainerNamespace)
			if ns == "" {
				ns = "public"
			}
			fqn := fmt.Sprintf("%s:%s:%s.%s", layer, conn, ns, strings.TrimSpace(p.ObjectName))
			return &fqn
		}
	}
	return rawTarget
}
```
*(Áp dụng tại cả `CreateWorkerScheduleHandler.Handle` và `UpdateScheduleHandler.Handle`)*.
