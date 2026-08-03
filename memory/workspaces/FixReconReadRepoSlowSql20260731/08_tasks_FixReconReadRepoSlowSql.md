# Danh sách Task - Fix Recon Read Repo Slow SQL

- [x] **Task 1: Tạo Migration File SQL tối ưu indexes cho Recon Tables**
  - Tạo `migrations/schema/recon_dlq/101_optimize_recon_read_indexes.sql`:
    - `CREATE INDEX IF NOT EXISTS idx_recon_runs_tier_started ON cdc_system.recon_runs (tier, started_at DESC);`
    - `CREATE INDEX IF NOT EXISTS idx_smoke_result_checked_at ON cdc_system.cdc_recon_smoke_result (checked_at DESC);`

- [x] **Task 2: Refactor `listLatestPrimary` & Time Window Pruning**
  - Bổ sung điều kiện khoanh vùng thời gian `WHERE checked_at >= NOW() - INTERVAL '7 days'` trong CTE `smoke_latest` của `listLatestPrimary` để chỉ scan các kết quả smoke check 7 ngày gần đây.

- [x] **Task 3: Refactor `GetBackfillStatus` Query**
  - Đảm bảo query `GetBackfillStatus` sử dụng index `(tier, started_at DESC)` bằng cách bổ sung `started_at >= NOW() - INTERVAL '7 days'`.

- [x] **Task 4: Build & Verify Test**
  - Chạy `go build ./cmd/server` thành công 100%.
