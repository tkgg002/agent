# 13 Analysis — Phân tích Gốc rễ Lỗi Vi phạm Ràng buộc Duy nhất (`payment_bills` trên Shadow và Master)

## 1. Dấu vết Lỗi (Error Trace)
- **Thông báo UI:** `Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất).`
- **Nguồn phát sinh thông báo UI:** [apiError.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/apiError.ts#L48-L50)
  ```ts
  '23505': (raw) => {
    const cn = raw.match(/unique constraint "([^"]+)"/)?.[1];
    return (cn && CONSTRAINT_HUMAN[cn]) || 'Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất).';
  },
  ```
- **Nguồn phát sinh SQLSTATE 23505 tại PostgreSQL (`cdc_dw`):**
  Bảng `cdc_system.cdc_worker_schedule` được khởi tạo tại [007_worker_schedule.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/007_worker_schedule.sql#L34) với ràng buộc:
  `UNIQUE(operation, target_table)` (tên constraint: `cdc_worker_schedule_operation_target_table_key`).

## 2. Sự bất nhất kiến trúc giữa `024_enhance_worker_schedule.sql` và `025_create_daily_source_counts.sql`
- Ở bảng lưu kết quả đếm [025_create_daily_source_counts.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/025_create_daily_source_counts.sql#L28), khóa duy nhất đã được thiết kế theo 5 thành phần:
  `UNIQUE (layer_type, connection_code, container_namespace, object_name, report_date)`
- Nhưng ở bảng cấu hình lịch chạy [024_enhance_worker_schedule.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/024_enhance_worker_schedule.sql#L10-L12), constraint cũ `UNIQUE(operation, target_table)` vẫn giữ nguyên, cộng với việc `ActivityManager.tsx` (dòng 1186) gán `payload.target_table = values.object_name` (`"payment_bills"`).
- Do đó, bất kỳ cặp lịch trình nào cùng `operation = 'source_count'` và cùng tên bảng `object_name = 'payment_bills'` — dù khác `layer_type` (`shadow` vs `master` vs `source`), khác `connection_code` hay khác `container_namespace` — đều có chung `(operation='source_count', target_table='payment_bills')` và bị PostgreSQL chặn cứng với `SQLSTATE 23505`.
