# Phân Tích Sâu Nguyên Nhân Slow SQL Activity Log & Giải Pháp Tối Ưu

## I. Phân tích Chi tiết 3 Câu Query Chậm

### 1. Query `Stats24h` Aggregation (205.035ms)
```sql
SELECT
    operation,
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'success') as success,
    COUNT(*) FILTER (WHERE status = 'error') as error,
    COUNT(*) FILTER (WHERE status = 'skipped') as skipped
FROM cdc_activity_log
WHERE started_at > NOW() - INTERVAL '24 hours'
GROUP BY operation
ORDER BY total DESC
```
- **Vấn đề:** Bảng `cdc_system.cdc_activity_log` được chia partition theo `created_at` (`PARTITION BY RANGE (created_at)`).
- Vì câu query chỉ lọc theo `started_at > NOW() - INTERVAL '24 hours'`, PostgreSQL Query Planner **KHÔNG THỂ THỰC HIỆN PARTITION PRUNING**. Nó buộc phải quét qua tất cả các partition daily trong quá khứ và default partition.
- **Giải pháp:**
  1. Thêm `created_at > NOW() - INTERVAL '24 hours'` song song với `started_at > NOW() - INTERVAL '24 hours'`. Việc này kích hoạt Partition Pruning lập tức, giúp Postgres chỉ truy vấn trên 1-2 partition ngày gần nhất.
  2. Tạo composite index `(created_at DESC, started_at DESC, operation, status)` trên bảng `cdc_activity_log`.

---

### 2. Query `ListActivity` Total Count (379.618ms)
```sql
SELECT COUNT(*) FROM cdc_activity_log al WHERE 1=1
```
- **Vấn đề:** `SELECT COUNT(*)` không có filter điều kiện thời gian buộc Postgres phải quét (Seq Scan / Index Scan) toàn bộ bảng `cdc_activity_log` trên tất cả partition để kiểm tra MVCC visibility cho từng row.
- Khi người dùng xem trang Activity Log ở góc nhìn tổng quan (mặc định không chọn filter), hệ thống đếm toàn bộ dòng lịch sử log.
- **Giải pháp:**
  1. Khi filter rỗng (không filter theo `SourceDatabase`, `SourceTable`, `ShadowSchema`, `ShadowTable`, `Operation`, `Status`, `TriggeredBy`), ta bổ sung điều kiện thời gian mặc định `created_at >= NOW() - INTERVAL '30 days'` (hoặc 14/30 ngày) để đếm trên cửa sổ log gần đây và kích hoạt Partition Pruning.
  2. Nếu có filter `Status` / `Operation` / `TriggeredBy`..., bổ sung `created_at` pruning tương ứng nếu không có range time cụ thể.

---

### 3. Query Enriched Activity List / Recent Errors (333.650ms)
```sql
SELECT
    al.id, al.operation, al.target_table, ...
FROM cdc_activity_log al
LEFT JOIN LATERAL ( SELECT ... FROM cdc_system.shadow_binding ... LIMIT 1 ) sb ON TRUE
LEFT JOIN LATERAL ( SELECT ... FROM cdc_system.master_binding ... LIMIT 1 ) tm ON TRUE
LEFT JOIN cdc_system.shadow_binding tm_sb ON tm_sb.id = tm.tm_sb_id
LEFT JOIN cdc_system.master_binding mb ON mb.shadow_binding_id = sb.shadow_binding_id AND mb.is_active = TRUE
LEFT JOIN LATERAL ( SELECT COUNT(*)::int AS binding_count FROM cdc_system.shadow_binding ... ) scope_counts ON TRUE
LEFT JOIN cdc_system.source_object_registry so ON so.id = sb.source_object_id
LEFT JOIN cdc_system.source_object_registry tm_so ON tm_so.id = tm_sb.source_object_id
WHERE 1=1 AND al.status = 'error'
ORDER BY al.started_at DESC LIMIT 10
```
- **Vấn đề:** 
  Cấu trúc cũ bọc 3 cái `LEFT JOIN LATERAL` + `LEFT JOIN` cho MỖI DÒNG trong bảng `cdc_activity_log` **TRƯỚC** khi filter `status = 'error'` và `ORDER BY al.started_at DESC LIMIT 10`.
  Nếu bảng có 10,000 dòng log `error`, Postgres phải thi hành 3 subquery `LATERAL` 10,000 lần rồi mới sort lấy 10 dòng!
- **Giải pháp bứt phá (Subquery / CTE Pagination First):**
  Lọc (`WHERE ...`), Sắp xếp (`ORDER BY ...`), Phân trang (`OFFSET ... LIMIT ...`) trên bảng gốc `cdc_activity_log al` TRƯỚC trong một Subquery / CTE `filtered_al`.
  Sau đó mới thực hiện `LEFT JOIN LATERAL` các thông tin enrichment (`shadow_binding`, `master_binding`, `source_object_registry`) trên tập kết quả ĐÃ PHÂN TRANG (đúng 10 dòng!).
  
  Mô hình SQL tối ưu:
  ```sql
  SELECT
      al.id, al.operation, al.target_table,
      COALESCE(tm_so.source_database, so.source_database) AS source_database,
      ...
  FROM (
      SELECT al.*
      FROM cdc_activity_log al
      WHERE 1=1 [filters...]
      ORDER BY al.started_at DESC
      OFFSET ? LIMIT ?
  ) al
  LEFT JOIN LATERAL ( ... ) sb ON TRUE
  LEFT JOIN LATERAL ( ... ) tm ON TRUE
  LEFT JOIN cdc_system.shadow_binding tm_sb ON tm_sb.id = tm.tm_sb_id
  LEFT JOIN cdc_system.master_binding mb ON mb.shadow_binding_id = sb.shadow_binding_id AND mb.is_active = TRUE
  LEFT JOIN LATERAL ( ... ) scope_counts ON TRUE
  LEFT JOIN cdc_system.source_object_registry so ON so.id = sb.source_object_id
  LEFT JOIN cdc_system.source_object_registry tm_so ON tm_so.id = tm_sb.source_object_id
  ORDER BY al.started_at DESC
  ```
  Nhờ đó, 3 subquery `LATERAL` chỉ phải chạy ĐÚNG `pageSize` lần (ví dụ 10 lần), giúp tốc độ phản hồi giảm từ 333ms xuống **< 5ms**!
