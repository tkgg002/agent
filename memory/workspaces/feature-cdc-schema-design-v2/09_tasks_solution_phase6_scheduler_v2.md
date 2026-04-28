# Solution — Phase 6 Scheduler V2

## Delivered

`TransmuteScheduler` không còn phụ thuộc vào `cdc_internal.transmute_schedule`.
Từ giờ schedule nằm ở `cdc_system` và được neo trực tiếp vào `master_binding_id`, nên:

- metadata scheduler ở đúng control plane
- không cần match bằng `master_table` string rời rạc
- scheduler tự lọc master binding chưa approved/inactive trước khi dispatch

## Remaining Gaps

- CMS/API chưa quản trị bảng schedule V2 mới.
- Recon/command vẫn còn nhiều điểm lookup từ `TableRegistry`.
