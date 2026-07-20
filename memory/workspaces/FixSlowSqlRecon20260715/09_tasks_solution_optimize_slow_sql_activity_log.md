# Hồ sơ Giải pháp: Tối ưu hóa SQL cdc_activity_log bằng CTE

Hồ sơ này mô tả chi tiết phương án kỹ thuật loại bỏ hoàn toàn các phép `LEFT JOIN LATERAL` đắt đỏ trong các câu truy vấn nhật ký hoạt động (`cdc_activity_log`) bằng cách sử dụng Common Table Expressions (CTE).

## 1. Thiết kế SQL tối ưu bằng CTE

Chúng ta sẽ chuyển đổi các phép `LEFT JOIN LATERAL` hiện tại thành hai CTE:
1. `active_bindings`: Lọc các shadow binding đang hoạt động và gán số thứ tự (`ROW_NUMBER()`) phân nhóm theo `shadow_table` nhằm mô phỏng chính xác logic lấy bản ghi mới nhất (`LIMIT 1`).
2. `binding_counts`: Tính toán số lượng shadow binding đang hoạt động cho từng bảng thông qua hàm `COUNT(*)` và `GROUP BY` đơn giản.

### Câu SQL CTE & JOIN đề xuất:

```sql
WITH active_bindings AS (
    SELECT 
        id,
        source_object_id,
        shadow_schema,
        shadow_table,
        ROW_NUMBER() OVER (PARTITION BY shadow_table ORDER BY updated_at DESC, id DESC) as rn
    FROM cdc_system.shadow_binding
    WHERE is_active = TRUE
),
binding_counts AS (
    SELECT 
        shadow_table,
        COUNT(*) AS binding_count
    FROM cdc_system.shadow_binding
    WHERE is_active = TRUE
    GROUP BY shadow_table
)
SELECT
    al.id,
    al.operation,
    al.target_table,
    so.source_database,
    so.source_schema,
    so.source_namespace,
    so.source_object_name AS source_table,
    sb.shadow_schema,
    sb.shadow_table,
    COALESCE(bc.binding_count, 0) > 1 AS scope_ambiguous,
    al.status,
    al.rows_affected,
    al.duration_ms,
    COALESCE(al.details::text, '{}') as details,
    al.error_message,
    al.triggered_by,
    TO_CHAR(al.started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS started_at,
    CASE
        WHEN al.completed_at IS NULL THEN NULL
        ELSE TO_CHAR(al.completed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
    END AS completed_at
FROM cdc_activity_log al
LEFT JOIN active_bindings sb ON 
    (al.target_table IS NOT NULL AND al.target_table <> '*')
    AND sb.shadow_table = al.target_table
    AND (
        (
            NULLIF(al.details->>'shadow_binding_id', '') IS NOT NULL 
            AND sb.id = (al.details->>'shadow_binding_id')::bigint
        )
        OR
        (
            NULLIF(al.details->>'shadow_binding_id', '') IS NULL
            AND sb.rn = 1
        )
    )
LEFT JOIN binding_counts bc ON 
    (al.target_table IS NOT NULL AND al.target_table <> '*')
    AND bc.shadow_table = al.target_table
LEFT JOIN cdc_system.source_object_registry so
  ON so.id = sb.source_object_id
WHERE 1=1
```

## 2. Kế hoạch triển khai mã nguồn

### Các thay đổi trong `activity_log_read_repo_gorm.go`:
- Thêm phương thức `cteBase() string` để định nghĩa phần CTE.
- Đổi tên `baseFromClause() string` thành `fromClause() string` và chuyển đổi sang các phép LEFT JOIN với CTE.
- Cập nhật `projectionColumns() string` để dùng `bc.binding_count` thay cho `scope_counts.binding_count`.
- Cập nhật hàm `ListActivity`:
  - Ghép `cteBase()`, `projectionColumns()`, và `fromClause()` cho câu truy vấn chính.
  - Khi cần join để đếm (`needJoinsForCount = true`), ghép `cteBase()` và `fromClause()` cho câu truy vấn đếm.
- Cập nhật hàm `Stats24h`: Ghép `cteBase()`, `projectionColumns()`, và `fromClause()` cho câu truy vấn lấy 10 lỗi gần nhất.
