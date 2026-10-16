# 12 Implementation Plan — Sửa lỗi Unique Constraint `(operation, target_table)` trên `cdc_worker_schedule`

## 1. Mục tiêu
Loại bỏ lỗi `SQLSTATE 23505` (`"Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)"`) khi tạo nhiều lịch trình `source_count` (hoặc `confluence`) cho cùng tên bảng (ví dụ `payment_bills`) ở các tầng khác nhau (`source`, `shadow`, `master`), kết nối khác nhau (`connection_code`) hoặc schema khác nhau (`container_namespace`).

## 2. Các bước thực thi (Sau khi có lệnh `APPROVE` từ User)
1. **Bước 1 — Database Migration (`cdc-cms-service`):**
   - Tạo `migrations/schema/worker/026_fix_worker_schedule_unique_scope.sql` để drop constraint phẳng `cdc_worker_schedule_operation_target_table_key`, chuẩn hóa dữ liệu `target_table` của `source_count` và tạo 2 Partial Unique Index (`uq_worker_schedule_source_count_scope` theo 5 thành phần và `uq_worker_schedule_legacy_op_target` cho các operation cũ).
   - Cập nhật `migrations/seed/worker_schedules.sql` khớp với Partial Unique Index.
2. **Bước 2 — Backend (`cdc-cms-service`):**
   - Cập nhật `create_worker_schedule.go` và `update_schedule.go` bổ sung `CanonicalScheduleTargetTable` để tự động chuẩn hóa `target_table` dạng `<layer_type>:<connection_code>:<container_namespace>.<object_name>` cho `source_count`.
   - Cập nhật unit test trong `cdc-cms-service/internal/app/commands/scheduler/`.
3. **Bước 3 — Worker (`centralized-data-service`):**
   - Cập nhật `source_count_runner.go` để đảm bảo `params.ObjectName` luôn lấy tên bảng thuần cho query SQL/upsert `daily_source_counts`, đồng thời ghi `targetName` đầy đủ `[<layer>:<conn>] <namespace>.<object>` vào `cdc_activity_log`.
4. **Bước 4 — Frontend (`cdc-cms-web`):**
   - Cập nhật `ActivityManager.tsx` (`buildSchedulePayload`, `handleSaveEdit`, và cột hiển thị `Phạm vi / Đối tượng`) + `apiError.ts` (`CONSTRAINT_HUMAN`).
5. **Bước 5 — Kiểm thử Thực tế (Gate G1–G8):**
   - Chạy unit tests, restart `cdc-cms-service` & `centralized-data-service`, bắn HTTP tạo 2 lịch trình `source_count` cho `payment_bills` (1 `shadow`, 1 `master`) để chứng minh cả 2 cùng được tạo thành công (HTTP 201) và chỉ chặn (HTTP 409/23505) khi trùng khít cả 4 tọa độ `(layer_type, connection_code, container_namespace, object_name)`.
