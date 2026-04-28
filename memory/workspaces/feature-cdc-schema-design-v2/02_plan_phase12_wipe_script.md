# Plan — Phase 12 Wipe Script

## Kế hoạch

1. Đọc lại schema V2 và các bảng control-plane liên quan.
2. Thiết kế thứ tự wipe an toàn:
   - drop master physical tables
   - drop shadow schemas
   - cleanup legacy public tables
   - truncate `cdc_system`
   - drop `cdc_internal`
3. Cập nhật runbook dùng script này.
4. Tự rà lại SQL và query verify.
