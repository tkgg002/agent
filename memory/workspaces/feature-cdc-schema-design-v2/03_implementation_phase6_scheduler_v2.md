# Implementation — Phase 6 Scheduler V2

## Code Changes

- thêm [036_v2_transmute_schedule.sql](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/migrations/036_v2_transmute_schedule.sql)
  - tạo `cdc_system.transmute_schedule`
  - schedule gắn với `master_binding_id`
  - backfill từ `cdc_internal.transmute_schedule`

- thêm model/repo:
  - [transmute_schedule.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/model/transmute_schedule.go)
  - [transmute_schedule_repo.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/repository/transmute_schedule_repo.go)

- sửa [transmute_scheduler.go](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/transmute_scheduler.go)
  - poll từ `cdc_system.transmute_schedule`
  - join `cdc_system.master_binding`
  - chỉ dispatch binding `is_active=true` và `schema_status='approved'`

## Result

Scheduler metadata đã được dời khỏi data-plane legacy `cdc_internal` sang control-plane `cdc_system`, giảm thêm một vùng legacy trước đợt wipe/bootstrap.
