# Validation — Phase 12 Wipe Script

## Đã kiểm tra

- đọc lại `wipe_cdc_runtime_v2.sql`
- đọc lại `wipe_bootstrap_v2.md`
- diff lại phần thay đổi

## Lỗi đã bắt và sửa

- runbook ban đầu query `transmute_status` trong `cdc_system.sync_runtime_state`
- schema thật không có cột này
- đã sửa về query đúng với schema hiện tại

## Lưu ý

- Chưa chạy wipe script thật trong turn này vì đây là thao tác destructive trên DB.
