# Bối cảnh & Phạm vi (Scope & Context)

- **Vấn đề**: Cảnh báo SLOW SQL (>= 200ms) xuất hiện ở cdc-cms-service tại `system_health_queries.go` và `probes/postgres.go` định kỳ mỗi 15 giây (khi background collector chạy). Cụ thể là các câu truy vấn:
  1. `SELECT count(*) FROM "cdc_table_registry"` (probes/postgres.go)
  2. `SELECT * FROM "cdc_activity_log" WHERE created_at > NOW() - INTERVAL '1 day' AND created_at <= NOW() ORDER BY started_at DESC LIMIT 10` (system_health_queries.go)
  3. `SELECT count(*) FROM "cdc_system"."failed_sync_logs" WHERE created_at > NOW() - INTERVAL '24 hours' AND created_at <= NOW()` (system_health_queries.go)
  4. `SELECT COALESCE(SUM(n_live_tup),0) FROM pg_stat_user_tables WHERE schemaname='public'` (probes/postgres.go)
- **Phạm vi xử lý**:
  1. Xác định nguyên nhân thực thi/lập kế hoạch (planning/execution) chậm trong PostgreSQL, bao gồm ảnh hưởng của hàm `NOW()` động trên bảng phân hoạch (`cdc_activity_log`, `failed_sync_logs`) và ảnh hưởng của `PrepareStmt: true` trong GORM.
  2. Đưa ra giải pháp tối ưu hóa truy vấn:
     - Tính toán trước mốc thời gian bằng Go (`time.Now()`) và truyền làm tham số thay vì dùng hàm `NOW()` trực tiếp trong SQL, giúp PostgreSQL tối ưu hóa tĩnh phân hoạch (partition pruning).
     - Thiết lập session GORM với `PrepareStmt: false` cho luồng background healthcheck/probes nhằm giảm thiểu mutex contention của prepared statement cache và các SQL roundtrips `PREPARE`.
  3. Kiểm chứng hiệu năng (verify) sau khi sửa đổi.
