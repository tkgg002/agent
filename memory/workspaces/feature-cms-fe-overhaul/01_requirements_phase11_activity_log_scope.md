# Requirements — Phase 11 Activity Log Scope

## Bối cảnh

- `activity-log` là page monitoring quan trọng của `cms-fe operator-flow`.
- Sau Phase 10, `mapping-rules` đã sang V2 nhưng monitoring layer vẫn nhìn log qua `target_table`.
- `useAsyncDispatch` cũng đang assume `target_table` là filter trung tâm cho polling.

## Yêu cầu

1. `activity-log` API phải trả enriched source/shadow context từ metadata V2.
2. API phải hỗ trợ filter theo:
   - `source_database`
   - `source_table`
   - `shadow_schema`
   - `shadow_table`
   - vẫn giữ `target_table` fallback
3. `useAsyncDispatch` phải chấp nhận status params giàu hơn mà không làm gãy caller cũ.
4. Cập nhật swagger/comment cùng phase.
