# Yêu cầu Tối ưu hóa SLOW SQL Activity Log (FixActivityLogSlowSql)

## 1. Bối cảnh & Hiện trạng
Hệ thống ghi nhận 3 câu truy vấn SQL bị cảnh báo SLOW SQL (>= 200ms) trên bảng `cdc_system.cdc_activity_log` tại file `internal/infra/persistence/system/activity_log_read_repo_gorm.go`:

1. **Slow Query 1 (Line 264 - Stats24h Aggregation):** 205.035ms
   ```sql
   SELECT operation, COUNT(*) as total, ...
   FROM cdc_activity_log
   WHERE started_at > NOW() - INTERVAL '24 hours'
   GROUP BY operation ORDER BY total DESC
   ```
2. **Slow Query 2 (Line 235 - ListActivity Count):** 379.618ms
   ```sql
   SELECT COUNT(*) FROM cdc_activity_log al WHERE 1=1
   ```
3. **Slow Query 3 (Line 274 - Stats24h Recent Errors / List Query):** 333.650ms
   ```sql
   SELECT al.id, ... FROM cdc_activity_log al
   LEFT JOIN LATERAL (...) sb ON TRUE
   LEFT JOIN LATERAL (...) tm ON TRUE
   ...
   WHERE 1=1 AND al.status = 'error'
   ORDER BY al.started_at DESC LIMIT 10
   ```

## 2. Mục tiêu (Definition of Done)
- [ ] Tối ưu cả 3 câu truy vấn SQL xuống dưới **50ms** (mục tiêu < 20ms).
- [ ] Bổ sung cờ Partition Pruning (`created_at`) cho các query trên bảng partitioned `cdc_activity_log`.
- [ ] Tái cấu trúc câu SQL Enriched List (`projectionColumns` + `baseFromClause`) theo chuẩn **Subquery Pagination First**: Phân trang / Filter trên bảng `cdc_activity_log` TRƯỚC, rồi mới `LEFT JOIN LATERAL` các bảng thông tin liên quan (`shadow_binding`, `master_binding`, `source_object_registry`).
- [ ] Bổ sung file SQL migration tạo composite indexes tối ưu trên `cdc_system.cdc_activity_log`:
  - `idx_act_created_started` trên `(created_at DESC, started_at DESC, status)`
  - `idx_act_status_started` trên `(status, started_at DESC, created_at DESC)`
- [ ] Không làm phá vỡ wire contract của API `/api/activity-log` và `/api/activity-log/stats`.
