# 09_tasks_solution — FixSourceObjectListingDedupe

## Diagnosed Root Cause
`source_object_read_repo_gorm.go:46-52` — `LEFT JOIN cdc_system.cdc_table_registry tr` chỉ predicate `(source_db, source_table, target_table)`, KHÔNG có `source_connection_id`. Sau khi migration 054/055/056 introduced `source_connection_id` thì cùng `(source_db, source_table, target_table)` có thể tồn tại cho NHIỀU connection → JOIN cartesian-amplify mỗi source_object lên N rows (N = số tr rows trùng db+table+target nhưng khác connection).

## Solution SQL — listBaseFromWhere mới

```sql
FROM cdc_system.source_object_registry so
LEFT JOIN cdc_system.shadow_binding sb
  ON sb.source_object_id = so.id
LEFT JOIN cdc_system.connection_registry cn
  ON cn.id = so.source_connection_id
LEFT JOIN LATERAL (
    SELECT
        tr.id,
        tr.sync_interval,
        tr.priority,
        tr.timestamp_field,
        tr.notes,
        tr.is_table_created,
        tr.updated_at
    FROM cdc_system.cdc_table_registry tr
    WHERE tr.source_db = so.source_database
      AND tr.source_table = so.source_object_name
      AND (
            (sb.shadow_table IS NOT NULL AND tr.target_table = sb.shadow_table)
         OR (sb.shadow_table IS NULL     AND tr.target_table = so.source_object_name)
      )
      AND (
            tr.source_connection_id = so.source_connection_id
         OR tr.source_connection_id IS NULL
      )
    ORDER BY
      (tr.source_connection_id IS NULL) ASC,  -- prefer exact match over legacy NULL
      tr.id ASC                                -- deterministic tiebreaker
    LIMIT 1
) tr ON TRUE
LEFT JOIN LATERAL (
    SELECT rr.target_table, rr.diff, rr.status, rr.checked_at
    FROM cdc_system.cdc_reconciliation_report rr
    WHERE rr.target_table = COALESCE(sb.shadow_table, tr.target_table)
    ORDER BY rr.checked_at DESC
    LIMIT 1
) rr ON TRUE
WHERE so.sync_engine = 'debezium'
```

## Note
- LATERAL subquery cần SELECT lại `tr.target_table` không? Không — caller chỉ dùng `tr.id, tr.sync_interval, tr.priority, tr.timestamp_field, tr.notes, tr.is_table_created, tr.updated_at`. Còn `tr.target_table` chỉ dùng trong predicate JOIN (đã đặt trong WHERE) + downstream `LEFT JOIN LATERAL rr ON ... = COALESCE(sb.shadow_table, tr.target_table)` → cần expose `tr.target_table`.
- Thêm `tr.target_table` vào SELECT của subquery.

## Expected Outcome
- Listing trả 4 rows (so id 1, 36, 18, 5) — 1 row/source_object.
- `total = 4`.
- Mỗi row có `registry_id` đúng connection scope (so id=1 → registry thuộc conn_id=2; so id=36 → registry thuộc conn_id=42).
