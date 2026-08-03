# Hồ Sơ Giải Pháp Kỹ Thuật (Technical Solution Profile)

## 1. Migration SQL File Mới
Tạo file `migrations/schema/partitioning/012_optimize_activity_log_indexes.sql`:

```sql
-- ------------------------------------------------------------
-- Composite Indexes for cdc_system.cdc_activity_log optimization
-- ------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_act_created_started_op ON cdc_system.cdc_activity_log (created_at DESC, started_at DESC, operation, status);
CREATE INDEX IF NOT EXISTS idx_act_status_started ON cdc_system.cdc_activity_log (status, started_at DESC, created_at DESC);
```

## 2. Refactor Code Go: `internal/infra/persistence/system/activity_log_read_repo_gorm.go`

### A. Tối ưu `Stats24h` Aggregation:
Thêm cờ prune `created_at > NOW() - INTERVAL '24 hours'`:
```sql
SELECT
    operation,
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'success') as success,
    COUNT(*) FILTER (WHERE status = 'error') as error,
    COUNT(*) FILTER (WHERE status = 'skipped') as skipped
FROM cdc_activity_log
WHERE created_at > NOW() - INTERVAL '24 hours'
  AND started_at > NOW() - INTERVAL '24 hours'
GROUP BY operation
ORDER BY total DESC
```

### B. Tối ưu `Stats24h` Recent Errors (10 lỗi gần nhất):
Sử dụng Derived Subquery `page_al` phân trang trước, rồi mới `LEFT JOIN LATERAL`:
```sql
SELECT
    al.id, al.operation, al.target_table, ...
FROM (
    SELECT al.*
    FROM cdc_activity_log al
    WHERE al.created_at > NOW() - INTERVAL '30 days'
      AND al.status = 'error'
    ORDER BY al.started_at DESC
    LIMIT 10
) al
LEFT JOIN LATERAL ( ... ) sb ON TRUE
...
ORDER BY al.started_at DESC
```

### C. Tối ưu `ListActivity`:
Tách biệt query lọc chính trên `cdc_activity_log` với subquery enrichment:
- Query chính lấy tập dữ liệu `al` đã filter + sort + offset + limit từ inner table `cdc_activity_log`.
- `LEFT JOIN LATERAL` enrichment chỉ thi hành trên N bản ghi trang hiện tại.
- Với Count Query khi không có filter: Thêm `created_at > NOW() - INTERVAL '30 days'` để prune các partition lịch sử cũ.
