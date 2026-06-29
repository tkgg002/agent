# Context: Bug Batch Transform & Abandon V1

## Problem Description
- Lỗi `cmd-batch-transform` bị skipped hoặc báo lỗi `"ERROR: column \"__v\" does not exist (SQLSTATE 42703)"` trên worker.
- Luồng V1 (`export_jobs`) cũ không được bỏ qua hoàn toàn, trong khi luồng V2 (`export_jobs_1`) đã được tạo trong shadow DB nhưng chưa được kích hoạt và cập nhật trạng thái đúng đắn.
- Do bước tạo schema/table của V2 trước đó bị lỗi `"schema shadow_cls_testing does not exist"`, quá trình kích hoạt V2 bị đứt gãy, dẫn đến `shadow_binding` của `export_jobs_1` vẫn ở trạng thái `pending` và chưa được migrate các cột business từ mapping rules.

## Core Objective
- Loại bỏ hoàn toàn luồng V1 (`export_jobs`) khỏi quá trình transform.
- Kích hoạt và cấu hình đầy đủ cột cho luồng V2 (`export_jobs_1`), đưa nó vào hoạt động.
- Cập nhật trạng thái `is_active` và `ddl_status` trong database (`cdc_table_registry` và `shadow_binding`) để Scheduler và Worker nhận dạng và xử lý đúng luồng V2.
