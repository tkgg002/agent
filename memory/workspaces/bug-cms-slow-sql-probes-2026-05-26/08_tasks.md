# Danh sách nhiệm vụ cho Muscle (08_tasks.md)

## Task: Disable PrepareStmt for healthcheck probes
- **Phase**: GĐ4 (Observability)
- **Service Group**: Utilities
- **Service(s)**: cdc-cms-service
- **Mô tả**: Tắt `PrepareStmt` (bằng cách dùng Session GORM với `PrepareStmt: false`) cho các câu truy vấn healthcheck để tránh tranh chấp mutex trên connection pool.
- **Trạng thái**: [x] DONE (đã thực hiện)

### [Context]
- File ảnh hưởng: `internal/infra/observability/probes/postgres.go`
- Lỗi/Cảnh báo: Cảnh báo SLOW SQL >= 200ms trên `SELECT count(*) FROM "cdc_table_registry"` và `SELECT COALESCE(SUM(n_live_tup),0) FROM pg_stat_user_tables WHERE schemaname='public'`.

### [Definition of Done]
- [x] GORM session được thiết lập với `PrepareStmt: false` trước khi thực thi truy vấn.
- [x] Hàm `probes.Postgres` chạy không crash, kết quả trả về đúng.
- [x] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.

---

## Task: Optimize healthcheck queries and disable PrepareStmt
- **Phase**: GĐ4 (Observability)
- **Service Group**: Utilities
- **Service(s)**: cdc-cms-service
- **Mô tả**: Tắt `PrepareStmt` và sử dụng placeholder tham số mốc thời gian tĩnh được tính toán bằng Go (`time.Now()`) cho các câu truy vấn trong `system_health_queries.go`.
- **Trạng thái**: [x] DONE (đã thực hiện)

### [Context]
- File ảnh hưởng: `internal/infra/observability/system_health_queries.go`
- Lỗi/Cảnh báo: Cảnh báo SLOW SQL >= 200ms trên `cdc_activity_log` và `failed_sync_logs` do dùng `NOW()` cản trở partition pruning và tranh chấp mutex `PrepareStmt`.

### [Definition of Done]
- [x] GORM session được thiết lập với `PrepareStmt: false` cho tất cả các truy vấn trong file.
- [x] Hàm `NOW()` trong các truy vấn SQL được thay thế bằng placeholders `?` và truyền thời gian Go (`time.Now()`).
- [x] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.
