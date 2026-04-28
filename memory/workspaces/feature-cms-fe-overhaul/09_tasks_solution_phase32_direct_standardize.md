# Phase 32 Solution — Direct Standardize

## Giải pháp chốt
- `standardize` được direct-V2 hóa vì worker hiện chỉ cần `target_table`
- FE không còn chặn row V2-only ở action `Tạo Field MĐ`

## Lợi ích
- operator-flow thực chiến hơn
- giảm thêm bridge dependence mà không cần đụng worker sâu

## Debt còn lại
- `create-default-columns`
- các action mapping/sync còn phụ thuộc `registry_id`
