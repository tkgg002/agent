# Plan — Phase 9 Namespace Finalization

## Kế hoạch thực thi

1. Audit toàn bộ runtime/test để tìm SQL còn chạm `public` / `cdc_internal` cho system tables.
2. Refactor runtime calls còn lại:
   - `claim_machine_id`
   - `heartbeat_machine_id`
   - `tg_fencing_guard`
   - `enable_master_rls`
3. Chuẩn hóa comment và test theo namespace mới để tránh drift tài liệu nội bộ.
4. Thêm migration finalization:
   - move sequence sang `cdc_system`
   - recreate function ở `cdc_system`
   - drop function cũ
   - drop `cdc_internal`
5. Cập nhật grant để role runtime có `USAGE/EXECUTE` trên `cdc_system`.
6. Verify bằng:
   - search audit
   - `gofmt`
   - `go test` cho các package chính

## Risk cần theo dõi

- `public` là schema mặc định của Postgres; không nên cưỡng ép drop schema này ở DB level.
- Migration lịch sử vẫn có thể tạo object ở schema cũ rồi được move ở cuối; end-state mới là điều cần bảo đảm cho đợt bootstrap.
