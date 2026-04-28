# Requirements — Phase 6 Scheduler V2

## Scope

- Dời metadata của transmute scheduler khỏi `cdc_internal` sang `cdc_system`.
- Gắn schedule với `master_binding_id` thay vì tên bảng rời rạc.
- Đảm bảo scheduler chỉ dispatch các `master_binding` đang active/approved.

## Required Outcomes

1. Tạo `cdc_system.transmute_schedule`.
2. Backfill schedule cũ từ `cdc_internal.transmute_schedule`.
3. Refactor `TransmuteScheduler` đọc/ghi bảng V2 mới.
4. Verify compile/test cho `internal/service`, `internal/server`, `internal/repository`, `internal/model`.

## Non-Goals

- Chưa xoá bảng `cdc_internal.transmute_schedule` cũ trong phase này.
- Chưa thêm API/CMS layer quản trị schedule V2.
