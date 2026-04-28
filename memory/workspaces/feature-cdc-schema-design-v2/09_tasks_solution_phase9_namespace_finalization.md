# Solution — Phase 9 Namespace Finalization

## Tóm tắt giải pháp

Phase này chốt nốt bài toán namespace theo đúng rule của user:

- `cdc_system` là control plane duy nhất cho bảng hệ thống.
- `shadow_<source_db>` là namespace vật lý cho shadow tables.
- `master` tiếp tục nằm ở schema đích theo binding.
- `cdc_internal` bị loại khỏi runtime path.

## Những gì đã hoàn tất

1. Runtime đã gọi function hệ thống từ `cdc_system`.
2. Test integration đã đọc/ghi system table từ `cdc_system`.
3. Đã có migration `038` để dời sequence/function cuối cùng và drop `cdc_internal`.
4. Đã mở rộng quyền role để schema `cdc_system` dùng được ngay sau bootstrap.

## Ý nghĩa cho đợt wipe & bootstrap

Sau khi chạy migration tới `038` và seed metadata V2:

1. System tables của app sẽ nằm ở `cdc_system`.
2. Shadow tables mới sẽ được tạo trong `shadow_<source_db>`.
3. Master tables sẽ được tạo theo `master_binding`.
4. Runtime chính không còn lý do phải chạm `cdc_internal`.

## Khuyến nghị vận hành

Thứ tự cutover:

1. wipe dữ liệu cũ
2. chạy migration full tới `038`
3. seed metadata V2
4. start service
5. verify:
   - source ingest vào `shadow_<source_db>`
   - transmute tạo đúng master schema/table
   - system logs/state nằm ở `cdc_system`
