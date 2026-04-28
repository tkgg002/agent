# Solution — Phase 12 Reconciliation Scope V2

## Vấn đề

`DataIntegrity` đã hiển thị source/shadow semantics khá hơn ở UI, nhưng các action check/heal và phần report/failed logs vẫn buộc vào `target_table`. Điều đó làm operator-flow của CMS dễ mơ hồ khi namespace trùng tên.

## Giải pháp

1. Dùng `cdc_system.shadow_binding` + `cdc_system.source_object_registry` để resolve scope reconciliation.
2. Reuse `POST /api/reconciliation/check` cho cả:
   - check tất cả
   - check một scope cụ thể khi body có metadata V2
3. Thêm generic `POST /api/reconciliation/heal` nhận body scope-aware, nhưng vẫn giữ route path legacy.
4. Enrich `report` và `failed-sync-logs` bằng:
   - `source_table`
   - `shadow_schema`
   - `shadow_table`
   - `scope_ambiguous`
5. Refactor FE `DataIntegrity` và `useReconStatus` để gửi/hiển thị scope thật thay vì chỉ string table.

## Outcome

- Operator nhìn đúng `source/shadow` scope khi check/heal/review failed logs.
- API bớt target-table-centric nhưng không mở thêm surface thừa.
- Swagger/comment đã được cập nhật cùng phase.
