# Requirements — Phase 13 Legacy Artifact Purge

## Mục tiêu

- Xóa vật lý các artifact FE/BE đã hết vai trò sau khi route và navigation legacy bị loại bỏ.
- Giảm dư thừa feature/API đúng theo target:
  - Debezium-only
  - auto-flow là luồng chính
  - cms-fe giữ operator-flow monitoring / backup / retry / reconcile

## Yêu cầu chức năng

1. Chỉ xóa artifact khi đã audit usage và xác nhận không còn được wire vào runtime.
2. Purge phải bao gồm cả FE page lẫn backend handler nếu chúng là cặp legacy đã chết.
3. Sau purge, FE build và backend test vẫn phải pass.

## Đối tượng purge của phase này

- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/QueueMonitoring.tsx`
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/CDCInternalRegistry.tsx`
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/cdc_internal_registry_handler.go`
