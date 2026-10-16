# 11 Change Report — bug-worker-schedule-unique-constraint-2026-10-15

## 1. Tổng quan Thay đổi (Overview)
Giải quyết triệt để lỗi `SQLSTATE 23505` (`duplicate key value violates unique constraint "cdc_worker_schedule_operation_target_table_key"`) khi tạo lịch trình kiểm đếm (`source_count`) cho cùng tên bảng (ví dụ `payment_bills`) trên các tầng khác nhau (`source`, `shadow`, `master`) hoặc các kết nối/schema khác nhau:
1. **Database Migration & Seed (`cdc-cms-service`):**
   - Gỡ bỏ constraint phẳng `cdc_worker_schedule_operation_target_table_key` (`UNIQUE(operation, target_table)`).
   - Tạo Partial Functional Unique Index 5 thành phần `uq_worker_schedule_source_count_scope` theo `(operation, layer_type, connection_code, container_namespace, object_name)` cho `source_count`.
   - Tạo Partial Unique Index `uq_worker_schedule_legacy_op_target` trên `(operation, target_table) WHERE operation NOT IN ('source_count', 'confluence')` cho các tác vụ vận hành đơn tầng truyền thống.
   - Chuẩn hóa `target_table` của toàn bộ bản ghi `source_count` hiện hữu sang định dạng `<layer_type>:<connection_code>:<container_namespace>.<object_name>`.
2. **Backend (`cdc-cms-service`):**
   - Bổ sung hàm `CanonicalScheduleTargetTable` tự động chuẩn hóa `target_table` theo `<layer_type>:<connection_code>:<container_namespace>.<object_name>` khi tạo mới (`create_worker_schedule.go`) và cập nhật (`update_schedule.go`) lịch trình `source_count`.
   - Chuyển mã phản hồi HTTP khi vi phạm unique constraint (`23505`) từ `500` sang `409 Conflict` trong `schedule_handler.go`.
   - Bổ sung unit test `canonical_target_table_test.go`.
3. **Worker (`centralized-data-service`):**
   - Cập nhật `source_count_runner.go`: giữ nguyên `params.ObjectName` là tên bảng thuần để chạy câu lệnh SQL `SELECT COUNT(*)`, bóc tách tên bảng thuần nếu fallback từ `sched.TargetTable`, và ghi `cdc_activity_log` với định danh `[<layer_type>:<connection_code>] <container_namespace>.<object_name>`.
4. **Frontend (`cdc-cms-web`):**
   - Cập nhật `ActivityManager.tsx` (`buildSchedulePayload`, `handleOpenEdit`, `handleSaveEdit`, và cột hiển thị `"Phạm vi / Đối tượng"`).
   - Bổ sung thông báo tiếng Việt thân thiện cho `uq_worker_schedule_source_count_scope` trong `apiError.ts`.

---

## 2. Danh sách File Thay đổi & Thống kê Dòng code

| STT | File thay đổi | Loại | Số dòng thêm/sửa | Mô tả chi tiết |
|---|---|---|---|---|
| 1 | `cdc-cms-service/migrations/schema/worker/026_fix_worker_schedule_unique_scope.sql` | Tạo mới | +45 dòng | Gỡ constraint phẳng, chuẩn hóa dữ liệu `target_table` hiện có, tạo `uq_worker_schedule_source_count_scope` và `uq_worker_schedule_legacy_op_target`. |
| 2 | `cdc-cms-service/migrations/seed/worker_schedules.sql` | Cập nhật | +1 / -1 dòng | Cập nhật mệnh đề `ON CONFLICT` khớp với Partial Unique Index. |
| 3 | `cdc-cms-service/internal/app/commands/scheduler/create_worker_schedule.go` | Cập nhật | +36 / -3 dòng | Thêm `CanonicalScheduleTargetTable` và áp dụng khi tạo lịch trình + ghi audit log. |
| 4 | `cdc-cms-service/internal/app/commands/scheduler/update_schedule.go` | Cập nhật | +8 / -1 dòng | Đồng bộ `target_table` qua `CanonicalScheduleTargetTable` khi cập nhật `source_count`. |
| 5 | `cdc-cms-service/internal/api/scheduler/schedule_handler.go` | Cập nhật | +5 dòng | Trả về HTTP `409 Conflict` khi gặp lỗi PostgreSQL `23505`. |
| 6 | `cdc-cms-service/internal/app/commands/scheduler/canonical_target_table_test.go` | Tạo mới | +43 dòng | Unit test kiểm chứng `CanonicalScheduleTargetTable` cho `shadow` vs `master` cùng tên bảng `payment_bills`. |
| 7 | `centralized-data-service/internal/service/sourcecount/source_count_runner.go` | Cập nhật | +17 / -2 dòng | Tách tên bảng thuần ở nhánh fallback và format `targetName` có `[layer:conn] ns.obj` cho `cdc_activity_log`. |
| 8 | `cdc-cms-web/src/pages/ActivityManager.tsx` | Cập nhật | +14 / -4 dòng | Chuẩn hóa `payload.target_table` khi tạo/sửa `source_count` và hiển thị `[connection_code]` trên bảng danh sách. |
| 9 | `cdc-cms-web/src/utils/apiError.ts` | Cập nhật | +6 dòng | Thêm mapping thông báo tiếng Việt cho `uq_worker_schedule_source_count_scope`. |
