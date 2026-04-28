# Requirements — Phase 9 Namespace Finalization

## Mục tiêu

- Dọn nốt toàn bộ system table về `cdc_system`.
- Xóa dependency runtime còn sống vào schema `cdc_internal`.
- Chốt naming shadow physical table theo `shadow_<source_db>.<collection_or_table>`.
- Đảm bảo sau đợt `wipe & bootstrap`, runtime chính không còn đọc/ghi system table ở `public` hoặc `cdc_internal`.

## Rule chốt với User

1. Chỉ `shadow tables` và `master tables` được phép nằm ngoài `cdc_system`.
2. Tất cả bảng hệ thống/control/ops phải nằm trong `cdc_system`.
3. `cdc_internal` không còn là nơi chứa object runtime của hệ thống.
4. Shadow namespace ở phase hiện tại dùng convention dễ nhìn:
   - `shadow_<source_db>`

## Scope

- `centralized-data-service/internal/**`
- `centralized-data-service/cmd/sinkworker/**`
- `centralized-data-service/migrations/**`
- integration tests liên quan tới system tables

## Definition of Done

1. Runtime code không còn gọi `cdc_internal.*` cho các table/function hệ thống.
2. Integration tests không còn truy vấn system table ở schema cũ.
3. Có migration kết thúc để dời sequence/function còn lại sang `cdc_system` và drop `cdc_internal`.
4. Test package trọng yếu pass.
5. Có tài liệu cutover rõ ràng cho đợt `wipe & bootstrap`.
