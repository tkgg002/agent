# Implementation Plan: Fix `resolveSourceIDsForSegmentB` with Dynamic `idType` PK Resolution

## User Review Required
- Trình User duyệt việc áp dụng 100% chuẩn `idType` & Dynamic PK Resolution vào hàm `resolveSourceIDsForSegmentB` trong [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go):
  - Xóa bỏ câu SQL cũ `WHERE _source_id IN (?) OR _gpay_id::text IN (?)` trên Shadow DB.
  - Phân loại theo `PrimaryKeyField` và `PrimaryKeyType` từ `TableRegistry`:
    1. Tìm theo `_gpay_id` (`int8` Sonyflake) bằng mảng `[]int64` ➔ Dùng B-Tree Index trên Shadow DB.
    2. Tìm theo `PrimaryKeyField` của bảng gốc (`pkCol`) bằng mảng số `[]int64` hoặc mảng chuỗi `[]string` tùy theo `PrimaryKeyType` ➔ Dùng B-Tree Index trên Shadow DB.

## Proposed Changes

### Backend: `centralized-data-service`
#### [MODIFY] [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
- Refactor `resolveSourceIDsForSegmentB`:
  - Thêm tham số `targetTable string`.
  - Áp dụng logic `idType` đồng bộ 100% với Prune Master.

## Verification Plan
### Automated Tests
- Run `go test ./internal/handler/recon/...`
