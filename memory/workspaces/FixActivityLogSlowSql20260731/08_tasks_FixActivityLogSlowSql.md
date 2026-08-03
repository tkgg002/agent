# Danh sách Task - Fix Activity Log Slow SQL

- [x] **Task 1: Tạo Migration File SQL tối ưu indexes**
  - Tạo `migrations/schema/partitioning/012_optimize_activity_log_indexes.sql` bổ sung composite index:
    - `idx_act_created_started_op` ON `cdc_system.cdc_activity_log (created_at DESC, started_at DESC, operation, status)`
    - `idx_act_status_started` ON `cdc_system.cdc_activity_log (status, started_at DESC, created_at DESC)`

- [x] **Task 2: Refactor Query Stats24h (Aggregation + Recent Errors)**
  - Bổ sung `created_at > NOW() - INTERVAL '24 hours'` vào câu SQL Stats aggregation để PostgreSQL thực hiện **Partition Pruning**.
  - Refactor câu query Recent Errors (10 lỗi gần nhất) theo pattern **CTE / Subquery Pagination First**: Lọc và lấy 10 bản ghi `al` từ `cdc_activity_log` trước, sau đó mới `LEFT JOIN LATERAL` enrichment.

- [x] **Task 3: Refactor Query ListActivity & Count Query**
  - Refactor `ListActivity` dùng **Derived Subquery / CTE Pagination First**: Lọc `al` với offset + limit trước, sau đó JOIN các bảng enrichment `shadow_binding`, `master_binding`, `source_object_registry`.
  - Với Count Query: Khi không có filter `created_at`/`started_at`, bổ sung `created_at >= NOW() - INTERVAL '30 days'` (tận dụng Partition Pruning) để tránh scan toàn bộ các partition lịch sử.

- [x] **Task 4: Build & Verify Test**
  - Chạy `go build ./cmd/server` thành công 100%.
