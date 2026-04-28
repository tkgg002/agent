# Requirements — Phase 10 Mapping Rules V2

## Bối cảnh

- `Mapping Rules` là page quan trọng của cả auto-flow lẫn cms-fe operator-flow:
  - auto-flow dùng rule để transform/backfill
  - cms-fe dùng rule để review, approve, reload, trigger backfill
- Sau Phase 9, `masters` đã chuyển sang `cdc_system.master_binding`.
- `mapping-rules` vẫn còn bám `cdc_mapping_rules.source_table`.

## Yêu cầu

1. API `mapping-rules` phải chuyển sang `cdc_system.mapping_rule_v2`.
2. Contract phải hỗ trợ identity V2:
   - `source_object_id`
   - `source_database`
   - `source_table`
   - `shadow_schema`
   - `shadow_table`
3. Vẫn giữ compatibility tối thiểu với `source_table` / `table` query cũ.
4. Cập nhật swagger/comment cùng lúc khi đổi API.
