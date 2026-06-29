# Plan: Active V2 & Abandon V1 for export-jobs (COMPLETED)

## Phase 1: Chuẩn bị hạ tầng & Đồng bộ hóa V2 schema
1. [x] Tạo script scratch để publish NATS command `cdc.cmd.create-default-columns` cho `export_jobs_1`.
2. [x] Khởi động worker server bằng terminal (đã chạy sẵn trên máy host).
3. [x] Chạy script publish để worker thực thi DDL tạo các cột business cho `export_jobs_1`.
4. [x] Verify bảng `shadow_cls_testing.export_jobs_1` đã có đầy đủ các cột nghiệp vụ (như `__v`).

## Phase 2: Chuyển đổi trạng thái DB (Abandon V1 -> Active V2)
1. [x] Cập nhật `cdc_system.shadow_binding` qua SQL:
   - Set `is_active = false` cho `id = 1` (V1, `export_jobs`).
   - Set `ddl_status = 'created'` cho `id = 3` (V2, `export_jobs_1`).
2. [x] Cập nhật `cdc_system.cdc_table_registry` qua SQL:
   - Set `is_active = false` cho `id = 1` (V1, `export_jobs`).
   - Set `is_active = true` cho `id = 2` (V2, `export_jobs_1`).

## Phase 3: Verify & Monitor
1. [x] Trigger command NATS `cdc.cmd.batch-transform` với payload `"export_jobs_1"`.
2. [x] Kiểm tra log worker xem transform có thành công hay không (Thành công: 457 rows affected).
3. [x] Chạy automated tests để verify code.

