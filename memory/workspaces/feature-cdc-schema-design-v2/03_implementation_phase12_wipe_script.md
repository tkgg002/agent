# Implementation — Phase 12 Wipe Script

## File mới

- [wipe_cdc_runtime_v2.sql](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/deployments/sql/wipe_cdc_runtime_v2.sql)

## Logic wipe

1. đọc `cdc_system.master_binding` để drop master physical tables
2. drop các schema `shadow_%`
3. cleanup legacy system tables ở `public` nếu còn sót
4. truncate toàn bộ bảng trong `cdc_system` với `RESTART IDENTITY CASCADE`
5. drop `cdc_internal`

## Runbook update

- [wipe_bootstrap_v2.md](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/deployments/runbooks/wipe_bootstrap_v2.md)

Đã thêm:

- tham chiếu tới wipe script
- cách chạy script
- fix query verify `sync_runtime_state` để dùng cột thật:
  - `runtime_scope`
  - `ddl_status`
  - `last_success_at`
  - `last_error_message`
