# Plan — Phase 6 Scheduler V2

## Execution Plan

1. Tạo migration mới cho `cdc_system.transmute_schedule`.
2. Backfill dữ liệu cũ bằng cách join `master_table -> master_binding`.
3. Thêm model/repository tối thiểu cho schedule V2.
4. Đổi `TransmuteScheduler` sang truy cập bảng mới.
5. Verify targeted packages.
